package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func runWorker(ctx context.Context, store *Store, queue *Queue) {
	for {
		jobID, err := queue.dequeue(ctx)
		if err != nil {
			return
		}
		if err := processJob(ctx, store, jobID); err != nil {
			log.Printf("job %s failed: %v", jobID, err)
		}
	}
}

func processJob(ctx context.Context, store *Store, jobID string) error {
	job, err := store.get(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = StatusRunning
	job.UpdatedAt = time.Now().UTC()
	if err := store.update(ctx, job); err != nil {
		return err
	}

	if err := executeJob(job); err != nil {
		job.Status = StatusFailed
		job.UpdatedAt = time.Now().UTC()
		return store.update(ctx, job)
	}

	now := time.Now().UTC()
	job.Status = StatusCompleted
	job.UpdatedAt = now
	return store.update(ctx, job)
}

func executeJob(job Job) error {
	switch job.Type {
	case "sleep":
		time.Sleep(2 * time.Second)
		return nil
	case "fail":
		return fmt.Errorf("simulated failure")
	default:
		log.Printf("processed job %s type=%s payload=%s", job.ID, job.Type, string(job.Payload))
		return nil
	}
}
