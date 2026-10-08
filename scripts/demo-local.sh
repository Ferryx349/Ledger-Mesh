#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

export WEBHOOK_SECRET="${WEBHOOK_SECRET:-dev-webhook-secret}"
export DATABASE_URL="${DATABASE_URL:-postgres://ledger:ledger@localhost:5432/ledgerqueue?sslmode=disable}"
export REDIS_URL="${REDIS_URL:-redis://localhost:6379/0}"

echo "==> migrate"
make migrate >/dev/null 2>&1 || true
psql "$DATABASE_URL" -f migrations/001_jobs.sql >/dev/null 2>&1 || true
psql "$DATABASE_URL" -f migrations/002_fintech.sql >/dev/null 2>&1 || true

echo "==> start api + worker in background (if not running)"
if ! curl -sf http://localhost:8080/health >/dev/null; then
  go run ./cmd/api &
  API_PID=$!
  go run ./cmd/worker &
  WORKER_PID=$!
  trap 'kill $API_PID $WORKER_PID 2>/dev/null || true' EXIT
  sleep 2
fi

echo "==> mock provider (webhooks + provider rows + 1 orphan)"
MOCK_ORPHAN_TXN=1 go run ./cmd/mockprovider

echo "==> wait for jobs"
sleep 5

echo "==> reconcile"
go run ./cmd/reconcile

echo "==> latest reconciliation"
curl -s http://localhost:8080/reconciliation/latest | python3 -m json.tool || curl -s http://localhost:8080/reconciliation/latest

echo "==> metrics sample"
curl -s http://localhost:8080/metrics | head -20
