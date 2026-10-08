package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/abhaypandey/ledgerqueue/internal/config"
	"github.com/abhaypandey/ledgerqueue/internal/queue"
	"github.com/abhaypandey/ledgerqueue/internal/store"
	"github.com/abhaypandey/ledgerqueue/internal/worker"
)

func main() {
	cfg := config.LoadWorker()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer st.Close()

	q, err := queue.NewRedis(ctx, cfg.RedisURL, cfg.QueueKey, cfg.ProcessingKey, cfg.DLQKey)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer func() { _ = q.Close() }()

	log.Printf("worker %s started", cfg.WorkerID)
	worker.Run(ctx, st, q, worker.Config{
		WorkerID:  cfg.WorkerID,
		RetryBase: cfg.RetryBase,
	})
	log.Println("worker stopped")
}
