# Ledger Mesh (v1)

Distributed-shaped job queue: **API** and **worker** are separate processes; **Postgres** stores jobs; **Redis** holds the work queue.

```
POST /jobs  →  Postgres  +  Redis LPUSH  →  worker BRPOP  →  update Postgres
```

## Prerequisites

- Go 1.22+
- Docker (for Postgres and Redis)

## Run locally

```bash
make deps-up
make migrate
```

Terminal 1:

```bash
make run-worker
```

Terminal 2:

```bash
make run-api
```

## Try it

```bash
curl http://localhost:8080/ready

curl -X POST http://localhost:8080/jobs \
  -H 'Content-Type: application/json' \
  -d '{"type":"sleep"}'

curl http://localhost:8080/jobs/<job_id>
```

Job types for testing: `sleep` (2s), `fail` (marks failed), anything else logs and completes.

## Config (env)

| Variable | Default |
|----------|---------|
| `ADDR` | `:8080` (api only) |
| `DATABASE_URL` | `postgres://ledger:ledger@localhost:5432/ledgerqueue?sslmode=disable` |
| `REDIS_URL` | `redis://localhost:6379/0` |
| `QUEUE_KEY` | `jobs:queue` |

## Endpoints

- `GET /health` — process alive
- `GET /ready` — Postgres + Redis reachable
- `POST /jobs` — create and enqueue
- `GET /jobs/{id}` — job status from Postgres
