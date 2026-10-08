package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/abhaypandey/ledgerqueue/internal/config"
	"github.com/abhaypandey/ledgerqueue/internal/handler"
	"github.com/abhaypandey/ledgerqueue/internal/queue"
	"github.com/abhaypandey/ledgerqueue/internal/store"
)

func main() {
	cfg := config.LoadAPI()

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

	api := &handler.API{Store: st, Queue: q, WebhookSecret: cfg.WebhookSecret}
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	log.Println("api stopped")
}
