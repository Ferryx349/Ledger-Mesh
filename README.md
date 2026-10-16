# LedgerQueue

**Distributed job queue** in Go: separate API and workers, Redis-backed queue, Postgres persistence, retries and dead-letter queue, signed webhook ingestion, scheduled consistency checks, Prometheus metrics, Docker Compose, and Kubernetes manifests.

```
Mock webhook sender ──▶ API ──▶ Postgres (jobs + state)
                              └──▶ Redis queue ──▶ Workers (N)
CronJob / reconcile ──▶ compare internal records vs external export
```

## Features

| Area | Included |
|------|----------|
| Message broker | Redis (queue + processing list + DLQ) |
| Workers | Multiple processes / K8s replicas |
| Retries | Backoff, max attempts |
| DLQ | Jobs marked `dead`, ID on `jobs:dlq` |
| Persistence | Postgres for jobs and domain tables |
| Idempotency | Webhook `event_id` → unique job + deduplicated side effects |
| Failure recovery | `BRPOPLPUSH` processing list; requeue on worker restart |
| Consistency job | Compare internal event log vs external source rows |
| Mock sender | Signed sample webhooks + external rows for demo |
| Metrics | `/metrics` (Prometheus) |
| Deploy | Docker Compose + `k8s/` (API, workers, CronJob) |

## Prerequisites

- Go 1.22+
- Docker (Postgres, Redis, full stack)

## Quick start (local)

```bash
make deps-up          # postgres + redis + migrations
make run-worker       # terminal 1
make run-api          # terminal 2
make run-mock         # sends sample signed webhooks
make run-reconcile    # inline consistency check
curl http://localhost:8080/reconciliation/latest
```

Or one-shot demo script (requires `psql`):

```bash
chmod +x scripts/demo-local.sh
./scripts/demo-local.sh
```

## Full stack demo (Compose)

Runs API, 2 workers, mock webhook sender (with one orphan external row), then reconcile:

```bash
make demo
# or: docker compose --profile demo up --build
```

Then:

```bash
curl http://localhost:8080/reconciliation/latest
curl http://localhost:8080/metrics
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/ready` | Postgres + Redis |
| GET | `/metrics` | Prometheus |
| POST | `/webhooks/payment` | Signed webhook → enqueue `process_webhook` |
| POST | `/jobs` | Generic job enqueue |
| GET | `/jobs/{id}` | Job status |
| GET | `/reconciliation/latest` | Last consistency run |

Webhook header: `X-Webhook-Signature` = HMAC-SHA256(body, `WEBHOOK_SECRET`).

## Job types

| Type | Behavior |
|------|----------|
| `process_webhook` | Idempotent handler for ingested webhook payload |
| `reconcile_daily` | Compare internal vs external records for a period |
| `sleep` / `fail` | Test helpers |

## Config (env)

| Variable | Default |
|----------|---------|
| `DATABASE_URL` | `postgres://ledger:ledger@localhost:5432/ledgerqueue?sslmode=disable` |
| `REDIS_URL` | `redis://localhost:6379/0` |
| `QUEUE_KEY` | `jobs:queue` |
| `PROCESSING_KEY` | `jobs:processing` |
| `DLQ_KEY` | `jobs:dlq` |
| `WEBHOOK_SECRET` | `dev-webhook-secret` |
| `MOCK_ORPHAN_TXN` | `1` inserts external-only row (demo mismatch) |
| `RECONCILE_MODE` | `inline` or `enqueue` (worker-based reconcile job) |

## Kubernetes

Build image and apply manifests (example with kind/minikube):

```bash
make docker-build
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/postgres-redis.yaml
# run migrations manually once against postgres svc
kubectl apply -f k8s/api.yaml -f k8s/worker.yaml -f k8s/cronjob-reconcile.yaml
kubectl apply -f k8s/mockprovider-job.yaml   # optional one-shot demo
```

- **API Deployment**: 2 replicas, `/health` + `/ready` probes  
- **Worker Deployment**: 3 replicas (horizontal scaling)  
- **CronJob** `reconcile-daily`: midnight UTC inline reconcile  

## Architecture notes

- **At-least-once**: jobs may run more than once; handlers use idempotency keys / unique constraints.  
- **Why Redis + Postgres**: Redis for fast queueing; Postgres as source of truth.  
- **DLQ**: after `max_attempts`, status `dead` and job ID pushed to `jobs:dlq`.  
- **Reconcile job**: flags missing or mismatched rows between internal processing log and external export.

## Migrations

```bash
psql "$DATABASE_URL" -f migrations/001_jobs.sql
psql "$DATABASE_URL" -f migrations/002_fintech.sql
```

## Tests

```bash
make test
```
