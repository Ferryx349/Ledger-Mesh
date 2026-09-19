package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/abhaypandey/ledgerqueue/internal/job"
	"github.com/abhaypandey/ledgerqueue/internal/queue"
	"github.com/abhaypandey/ledgerqueue/internal/store"
)

func Run(ctx context.Context, st store.Store, q queue.Queue) {
	for {
		jobID, err := q.Dequeue(ctx)
		if err != nil {
			return
		}
		if err := process(ctx, st, jobID); err != nil {
			log.Printf("job %s failed: %v", jobID, err)
		}
	}
}

func process(ctx context.Context, st store.Store, jobID string) error {
	j, err := st.Get(ctx, jobID)
	if err != nil {
		return err
	}

	j.Status = job.StatusRunning
	j.UpdatedAt = time.Now().UTC()
	if err := st.Update(ctx, j); err != nil {
		return err
	}

	if err := execute(j); err != nil {
		j.Status = job.StatusFailed
		j.UpdatedAt = time.Now().UTC()
		return st.Update(ctx, j)
	}

	j.Status = job.StatusCompleted
	j.UpdatedAt = time.Now().UTC()
	return st.Update(ctx, j)
}

func execute(j job.Job) error {
	switch j.Type {
	case "sleep":
		time.Sleep(2 * time.Second)
		return nil
	case "fail":
		return fmt.Errorf("simulated failure")
	default:
		log.Printf("processed job %s type=%s payload=%s", j.ID, j.Type, string(j.Payload))
		return nil
	}
}
