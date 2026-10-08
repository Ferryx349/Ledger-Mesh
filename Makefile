.PHONY: deps-up deps-down migrate run-api run-worker run-mock run-reconcile demo test build docker-build

deps-up:
	docker compose up -d postgres redis migrate

deps-down:
	docker compose down

migrate:
	docker compose run --rm migrate

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

run-mock:
	go run ./cmd/mockprovider

run-reconcile:
	go run ./cmd/reconcile

demo:
	docker compose --profile demo up --build

test:
	go test ./...

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	go build -o bin/mockprovider ./cmd/mockprovider
	go build -o bin/reconcile ./cmd/reconcile

docker-build:
	docker build -t ledgerqueue:latest .
