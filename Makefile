.PHONY: deps-up deps-down migrate run-api run-worker test

deps-up:
	docker compose up -d

deps-down:
	docker compose down

migrate:
	docker compose exec -T postgres psql -U ledger -d ledgerqueue < migrations/001_jobs.sql

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

test:
	go test ./...
