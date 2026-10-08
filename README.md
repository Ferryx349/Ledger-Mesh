# LedgerQueue

Fintech-shaped **distributed job queue** in Go: webhook ingestion, idempotent ledger credits, retries + dead-letter queue, daily reconciliation, Prometheus metrics, Docker Compose, and Kubernetes manifests.

```
Mock provider ──webhook──▶ API ──▶ Postgres (jobs + ledger)
                              └──▶ Redis queue ──▶ Workers (N)
CronJob / reconcile ──▶ match ledger vs provider statement
```

## Features

| Area | Included |
|------|----------|
| Message broker | Redis (queue + processing list + DLQ) |
| Workers | Multiple processes / K8s replicas |
| Retries | Exponential backoff, max attempts |
| DLQ | Jobs marked `dead`, ID on `jobs:dlq` |
| Persistence | Postgres for jobs, ledger, provider txns, recon |
| Idempotency | Webhook `event_id` → unique job + ledger row |
| Failure recovery | `BRPOPLPUSH` processing list; requeue on worker restart |
| Reconciliation | Compare ledger vs provider transactions |
| Mock payment provider | Signed webhooks + provider settlement rows |
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
make run-mock         # sends 3 signed webhooks
make run-reconcile    # inline reconciliation
curl http://localhost:8080/reconciliation/latest
```

Or one-shot demo script (requires `psql`):

```bash
chmod +x scripts/demo-local.sh
./scripts/demo-local.sh
```

## Full stack demo (Compose)

Runs API, 2 workers, mock provider (with one orphan provider txn), then reconcile:

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
| POST | `/webhooks/payment` | Signed payment webhook → enqueue `process_webhook` |
| POST | `/jobs` | Generic job enqueue |
| GET | `/jobs/{id}` | Job status |
| GET | `/reconciliation/latest` | Last reconciliation run |

Webhook header: `X-Webhook-Signature` = HMAC-SHA256(body, `WEBHOOK_SECRET`).

## Job types

| Type | Behavior |
|------|----------|
| `process_webhook` | Idempotent ledger credit |
| `reconcile_daily` | Run reconciliation for a period |
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
| `MOCK_ORPHAN_TXN` | `1` inserts provider-only row (recon exception demo) |
| `RECONCILE_MODE` | `inline` or `enqueue` (for worker-based recon) |

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

## Architecture notes (for interviews)

- **At-least-once**: jobs may run more than once; ledger uses `ON CONFLICT` on `external_event_id`.  
- **Why Redis + Postgres**: Redis for fast queueing; Postgres as source of truth.  
- **DLQ**: after `max_attempts`, status `dead` and job ID pushed to `jobs:dlq`.  
- **Reconciliation**: finds missing/m mismatched rows between internal ledger and provider export.

## Migrations

```bash
psql "$DATABASE_URL" -f migrations/001_jobs.sql
psql "$DATABASE_URL" -f migrations/002_fintech.sql
```

## Tests

```bash
make test
```
