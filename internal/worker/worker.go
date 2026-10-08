package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/abhaypandey/ledgerqueue/internal/job"
	"github.com/abhaypandey/ledgerqueue/internal/ledger"
	"github.com/abhaypandey/ledgerqueue/internal/metrics"
	"github.com/abhaypandey/ledgerqueue/internal/payment"
	"github.com/abhaypandey/ledgerqueue/internal/queue"
	"github.com/abhaypandey/ledgerqueue/internal/reconcile"
	"github.com/abhaypandey/ledgerqueue/internal/store"
)

type Config struct {
	WorkerID  string
	RetryBase time.Duration
}

func Run(ctx context.Context, st store.Store, q queue.Queue, cfg Config) {
	if n, err := q.RequeueProcessing(ctx); err != nil {
		log.Printf("requeue processing: %v", err)
	} else if n > 0 {
		log.Printf("requeued %d in-flight jobs after restart", n)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		jobID, err := q.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("dequeue: %v", err)
			continue
		}

		if err := processOne(ctx, st, q, cfg, jobID); err != nil {
			log.Printf("job %s: %v", jobID, err)
		}
		updateQueueGauges(ctx, q)
	}
}

func processOne(ctx context.Context, st store.Store, q queue.Queue, cfg Config, jobID string) error {
	j, err := st.Get(ctx, jobID)
	if err != nil {
		_ = q.Ack(ctx, jobID)
		return err
	}

	j.Status = job.StatusRunning
	j.UpdatedAt = time.Now().UTC()
	if err := st.Update(ctx, j); err != nil {
		return err
	}

	execErr := execute(ctx, st, j)
	if execErr == nil {
		j.Status = job.StatusCompleted
		j.LastError = ""
		j.UpdatedAt = time.Now().UTC()
		if err := st.Update(ctx, j); err != nil {
			return err
		}
		_ = q.Ack(ctx, jobID)
		metrics.JobsProcessed.WithLabelValues(j.Type, "success").Inc()
		return nil
	}

	j.Attempts++
	j.LastError = execErr.Error()
	j.UpdatedAt = time.Now().UTC()

	if j.Attempts >= j.MaxAttempts {
		j.Status = job.StatusDead
		if err := st.Update(ctx, j); err != nil {
			return err
		}
		if err := q.EnqueueDLQ(ctx, jobID); err != nil {
			return err
		}
		metrics.JobsDeadLettered.Inc()
		metrics.JobsProcessed.WithLabelValues(j.Type, "dead").Inc()
		log.Printf("job %s moved to DLQ after %d attempts", jobID, j.Attempts)
		return execErr
	}

	j.Status = job.StatusPending
	if err := st.Update(ctx, j); err != nil {
		return err
	}
	sleep := cfg.RetryBase * time.Duration(j.Attempts)
	time.Sleep(sleep)
	if err := q.Requeue(ctx, jobID); err != nil {
		return err
	}
	metrics.JobsRetried.Inc()
	metrics.JobsProcessed.WithLabelValues(j.Type, "retry").Inc()
	log.Printf("job %s retry %d/%d in %s", jobID, j.Attempts, j.MaxAttempts, sleep)
	return execErr
}

func execute(ctx context.Context, st store.Store, j job.Job) error {
	switch j.Type {
	case job.TypeProcessWebhook:
		return handleProcessWebhook(ctx, st, j)
	case job.TypeReconcileDaily:
		return handleReconcileDaily(ctx, st, j)
	case "sleep":
		time.Sleep(2 * time.Second)
		return nil
	case "fail":
		return fmt.Errorf("simulated failure")
	default:
		log.Printf("worker %s processed job %s type=%s", "?", j.ID, j.Type)
		return nil
	}
}

func handleProcessWebhook(ctx context.Context, st store.Store, j job.Job) error {
	ev, err := payment.ParseWebhookPayload(j.Payload)
	if err != nil {
		return fmt.Errorf("payload: %w", err)
	}
	if !ev.Valid() {
		return fmt.Errorf("invalid webhook event")
	}
	_, err = ledger.CreditFromPayment(ctx, st, ev.EventID, ev.UserID, ev.AmountCents)
	return err
}

func handleReconcileDaily(ctx context.Context, st store.Store, j job.Job) error {
	start, end := reconcile.DefaultPeriod(time.Now().UTC())
	var payload struct {
		PeriodStart string `json:"period_start"`
		PeriodEnd   string `json:"period_end"`
	}
	if len(j.Payload) > 0 {
		_ = json.Unmarshal(j.Payload, &payload)
		if payload.PeriodStart != "" && payload.PeriodEnd != "" {
			var err error
			start, err = time.Parse(time.RFC3339, payload.PeriodStart)
			if err != nil {
				return err
			}
			end, err = time.Parse(time.RFC3339, payload.PeriodEnd)
			if err != nil {
				return err
			}
		}
	}
	res, err := reconcile.RunDaily(ctx, st, start, end)
	if err != nil {
		return err
	}
	metrics.ReconciliationRuns.Inc()
	log.Printf("reconciliation run %d: matched=%d exceptions=%d", res.RunID, res.MatchedCount, res.ExceptionCount)
	return nil
}

func updateQueueGauges(ctx context.Context, q queue.Queue) {
	if d, err := q.Depth(ctx); err == nil {
		metrics.QueueDepth.Set(float64(d))
	}
	if d, err := q.DLQDepth(ctx); err == nil {
		metrics.DLQDepth.Set(float64(d))
	}
}
