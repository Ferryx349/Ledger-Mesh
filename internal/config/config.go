package config

import (
	"os"
	"strconv"
	"time"
)

type Common struct {
	DatabaseURL   string
	RedisURL      string
	QueueKey      string
	ProcessingKey string
	DLQKey        string
}

type API struct {
	Common
	Addr          string
	WebhookSecret string
}

type Worker struct {
	Common
	WorkerID      string
	RetryBase     time.Duration
}

type MockProvider struct {
	Common
	APIURL        string
	WebhookSecret string
	Count         int
	Interval      time.Duration
	OrphanTxn     bool
}

type Reconcile struct {
	DatabaseURL string
}

func LoadAPI() API {
	c := loadCommon()
	return API{
		Common:        c,
		Addr:          env("ADDR", ":8080"),
		WebhookSecret: env("WEBHOOK_SECRET", "dev-webhook-secret"),
	}
}

func LoadWorker() Worker {
	c := loadCommon()
	retrySec, _ := strconv.Atoi(env("RETRY_BASE_SECONDS", "2"))
	if retrySec < 1 {
		retrySec = 2
	}
	return Worker{
		Common:    c,
		WorkerID:  env("WORKER_ID", "worker-1"),
		RetryBase: time.Duration(retrySec) * time.Second,
	}
}

func LoadMockProvider() MockProvider {
	c := loadCommon()
	count, _ := strconv.Atoi(env("MOCK_PAYMENT_COUNT", "3"))
	intervalSec, _ := strconv.Atoi(env("MOCK_INTERVAL_SECONDS", "1"))
	return MockProvider{
		Common:        c,
		APIURL:        env("API_URL", "http://localhost:8080"),
		WebhookSecret: env("WEBHOOK_SECRET", "dev-webhook-secret"),
		Count:         count,
		Interval:      time.Duration(intervalSec) * time.Second,
		OrphanTxn:     env("MOCK_ORPHAN_TXN", "") == "1",
	}
}

func LoadReconcile() Reconcile {
	return Reconcile{
		DatabaseURL: env("DATABASE_URL", "postgres://ledger:ledger@localhost:5432/ledgerqueue?sslmode=disable"),
	}
}

func loadCommon() Common {
	return Common{
		DatabaseURL:   env("DATABASE_URL", "postgres://ledger:ledger@localhost:5432/ledgerqueue?sslmode=disable"),
		RedisURL:      env("REDIS_URL", "redis://localhost:6379/0"),
		QueueKey:      env("QUEUE_KEY", "jobs:queue"),
		ProcessingKey: env("PROCESSING_KEY", "jobs:processing"),
		DLQKey:        env("DLQ_KEY", "jobs:dlq"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
