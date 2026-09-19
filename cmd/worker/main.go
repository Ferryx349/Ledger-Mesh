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

	q, err := queue.NewRedis(ctx, cfg.RedisURL, cfg.QueueKey)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer func() { _ = q.Close() }()

	log.Println("worker started")
	worker.Run(ctx, st, q)
	log.Println("worker stopped")
}
