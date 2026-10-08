package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/abhaypandey/ledgerqueue/internal/config"
	"github.com/abhaypandey/ledgerqueue/internal/job"
	"github.com/abhaypandey/ledgerqueue/internal/queue"
	"github.com/abhaypandey/ledgerqueue/internal/reconcile"
	"github.com/abhaypandey/ledgerqueue/internal/store"
	"github.com/google/uuid"
)

func main() {
	mode := os.Getenv("RECONCILE_MODE")
	if mode == "" {
		mode = "inline"
	}

	cfg := config.LoadReconcile()
	ctx := context.Background()

	st, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer st.Close()

	start, end := reconcile.DefaultPeriod(time.Now().UTC())
	if mode == "enqueue" {
		common := config.LoadWorker()
		q, err := queue.NewRedis(ctx, common.RedisURL, common.QueueKey, common.ProcessingKey, common.DLQKey)
		if err != nil {
			log.Fatalf("redis: %v", err)
		}
		defer func() { _ = q.Close() }()

		payload, _ := json.Marshal(map[string]string{
			"period_start": start.Format(time.RFC3339),
			"period_end":   end.Format(time.RFC3339),
		})
		j := job.New(uuid.NewString(), job.CreateRequest{
			Type:    job.TypeReconcileDaily,
			Payload: payload,
		})
		if err := st.Create(ctx, j); err != nil {
			log.Fatalf("create job: %v", err)
		}
		if err := q.Enqueue(ctx, j.ID); err != nil {
			log.Fatalf("enqueue: %v", err)
		}
		log.Printf("enqueued reconcile job %s for %s — %s", j.ID, start, end)
		return
	}

	res, err := reconcile.RunDaily(ctx, st, start, end)
	if err != nil {
		log.Fatalf("reconcile: %v", err)
	}
	log.Printf("reconciliation complete run_id=%d matched=%d exceptions=%d period=%s to %s",
		res.RunID, res.MatchedCount, res.ExceptionCount, start, end)
}
