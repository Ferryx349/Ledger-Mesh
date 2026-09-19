package config

import "os"

type API struct {
	Addr         string
	DatabaseURL  string
	RedisURL     string
	QueueKey     string
}

type Worker struct {
	DatabaseURL string
	RedisURL    string
	QueueKey    string
}

func LoadAPI() API {
	return API{
		Addr:        env("ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://ledger:ledger@localhost:5432/ledgerqueue?sslmode=disable"),
		RedisURL:    env("REDIS_URL", "redis://localhost:6379/0"),
		QueueKey:    env("QUEUE_KEY", "jobs:queue"),
	}
}

func LoadWorker() Worker {
	return Worker{
		DatabaseURL: env("DATABASE_URL", "postgres://ledger:ledger@localhost:5432/ledgerqueue?sslmode=disable"),
		RedisURL:    env("REDIS_URL", "redis://localhost:6379/0"),
		QueueKey:    env("QUEUE_KEY", "jobs:queue"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
