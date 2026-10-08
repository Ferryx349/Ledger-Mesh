ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS attempts      INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_attempts  INT NOT NULL DEFAULT 3,
    ADD COLUMN IF NOT EXISTS last_error    TEXT,
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs (status);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id                 BIGSERIAL PRIMARY KEY,
    user_id            TEXT NOT NULL,
    amount_cents       BIGINT NOT NULL,
    entry_type         TEXT NOT NULL CHECK (entry_type IN ('credit', 'debit')),
    external_event_id  TEXT NOT NULL UNIQUE,
    reference          TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_user ON ledger_entries (user_id);
CREATE INDEX IF NOT EXISTS idx_ledger_created ON ledger_entries (created_at);

CREATE TABLE IF NOT EXISTS provider_transactions (
    id                 BIGSERIAL PRIMARY KEY,
    provider_txn_id    TEXT NOT NULL UNIQUE,
    external_event_id  TEXT NOT NULL,
    amount_cents       BIGINT NOT NULL,
    currency           TEXT NOT NULL DEFAULT 'USD',
    settled_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provider_settled ON provider_transactions (settled_at);

CREATE TABLE IF NOT EXISTS reconciliation_runs (
    id               BIGSERIAL PRIMARY KEY,
    run_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_start     TIMESTAMPTZ NOT NULL,
    period_end       TIMESTAMPTZ NOT NULL,
    matched_count    INT NOT NULL DEFAULT 0,
    exception_count  INT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'completed'
);

CREATE TABLE IF NOT EXISTS reconciliation_exceptions (
    id          BIGSERIAL PRIMARY KEY,
    run_id      BIGINT NOT NULL REFERENCES reconciliation_runs (id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    details     JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recon_exceptions_run ON reconciliation_exceptions (run_id);
