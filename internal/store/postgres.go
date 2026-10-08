package store

import (
	"context"
	"errors"
	"time"

	"github.com/abhaypandey/ledgerqueue/internal/job"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, connString string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (s *Postgres) Close() {
	s.pool.Close()
}

func (s *Postgres) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Postgres) Create(ctx context.Context, j job.Job) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO jobs (id, type, status, payload, attempts, max_attempts, last_error, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		j.ID, j.Type, j.Status, j.Payload, j.Attempts, j.MaxAttempts, nullString(j.LastError), nullString(j.IdempotencyKey),
		j.CreatedAt, j.UpdatedAt,
	)
	return err
}

func (s *Postgres) Get(ctx context.Context, id string) (job.Job, error) {
	var j job.Job
	var lastErr, idemKey *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, type, status, payload, attempts, max_attempts, last_error, idempotency_key, created_at, updated_at
		FROM jobs WHERE id = $1`, id,
	).Scan(&j.ID, &j.Type, &j.Status, &j.Payload, &j.Attempts, &j.MaxAttempts, &lastErr, &idemKey, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.Job{}, ErrNotFound
	}
	if lastErr != nil {
		j.LastError = *lastErr
	}
	if idemKey != nil {
		j.IdempotencyKey = *idemKey
	}
	return j, err
}

func (s *Postgres) GetByIdempotencyKey(ctx context.Context, key string) (job.Job, error) {
	var j job.Job
	var lastErr, idemKey *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, type, status, payload, attempts, max_attempts, last_error, idempotency_key, created_at, updated_at
		FROM jobs WHERE idempotency_key = $1`, key,
	).Scan(&j.ID, &j.Type, &j.Status, &j.Payload, &j.Attempts, &j.MaxAttempts, &lastErr, &idemKey, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.Job{}, ErrNotFound
	}
	if lastErr != nil {
		j.LastError = *lastErr
	}
	if idemKey != nil {
		j.IdempotencyKey = *idemKey
	}
	return j, err
}

func (s *Postgres) Update(ctx context.Context, j job.Job) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE jobs SET type = $2, status = $3, payload = $4, attempts = $5, max_attempts = $6,
			last_error = $7, idempotency_key = $8, updated_at = $9
		WHERE id = $1`,
		j.ID, j.Type, j.Status, j.Payload, j.Attempts, j.MaxAttempts,
		nullString(j.LastError), nullString(j.IdempotencyKey), j.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Postgres) InsertLedgerCredit(ctx context.Context, externalEventID, userID string, amountCents int64, reference string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO ledger_entries (user_id, amount_cents, entry_type, external_event_id, reference, created_at)
		VALUES ($1, $2, 'credit', $3, $4, NOW())
		ON CONFLICT (external_event_id) DO NOTHING`,
		userID, amountCents, externalEventID, reference,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Postgres) ListLedgerInPeriod(ctx context.Context, start, end time.Time) ([]LedgerEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, amount_cents, entry_type, external_event_id, COALESCE(reference, ''), created_at
		FROM ledger_entries
		WHERE created_at >= $1 AND created_at < $2
		ORDER BY created_at`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.AmountCents, &e.EntryType, &e.ExternalEventID, &e.Reference, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Postgres) InsertProviderTransaction(ctx context.Context, providerTxnID, externalEventID string, amountCents int64, currency string, settledAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO provider_transactions (provider_txn_id, external_event_id, amount_cents, currency, settled_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider_txn_id) DO NOTHING`,
		providerTxnID, externalEventID, amountCents, currency, settledAt,
	)
	return err
}

func (s *Postgres) ListProviderInPeriod(ctx context.Context, start, end time.Time) ([]ProviderTransaction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, provider_txn_id, external_event_id, amount_cents, currency, settled_at
		FROM provider_transactions
		WHERE settled_at >= $1 AND settled_at < $2
		ORDER BY settled_at`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProviderTransaction
	for rows.Next() {
		var t ProviderTransaction
		if err := rows.Scan(&t.ID, &t.ProviderTxnID, &t.ExternalEventID, &t.AmountCents, &t.Currency, &t.SettledAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Postgres) CreateReconciliationRun(ctx context.Context, start, end time.Time, matched, exceptions int) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO reconciliation_runs (period_start, period_end, matched_count, exception_count, status)
		VALUES ($1, $2, $3, $4, 'completed')
		RETURNING id`, start, end, matched, exceptions,
	).Scan(&id)
	return id, err
}

func (s *Postgres) InsertReconciliationException(ctx context.Context, runID int64, kind string, details []byte) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO reconciliation_exceptions (run_id, kind, details)
		VALUES ($1, $2, $3)`, runID, kind, details)
	return err
}

func (s *Postgres) LatestReconciliationRun(ctx context.Context) (ReconciliationRun, error) {
	var r ReconciliationRun
	err := s.pool.QueryRow(ctx, `
		SELECT id, run_at, period_start, period_end, matched_count, exception_count, status
		FROM reconciliation_runs ORDER BY id DESC LIMIT 1`,
	).Scan(&r.ID, &r.RunAt, &r.PeriodStart, &r.PeriodEnd, &r.MatchedCount, &r.ExceptionCount, &r.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReconciliationRun{}, ErrNotFound
	}
	return r, err
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

