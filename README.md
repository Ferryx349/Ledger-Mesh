# Ledger Mesh

Minimal in-memory job queue in Go — API + worker in one process.

```
POST /jobs  →  in-memory queue  →  worker  →  job status updated
```

## Run

```bash
make run
```

## Try it

```bash
curl -X POST http://localhost:8080/jobs \
  -H 'Content-Type: application/json' \
  -d '{"type":"sleep"}'

curl http://localhost:8080/jobs/<job_id>
```

Job types for testing: `sleep` (waits 2s), `fail` (marks failed), anything else logs and completes.
