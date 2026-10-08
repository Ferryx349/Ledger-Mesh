package store

import (
	"context"
	"errors"
	"time"

	"github.com/abhaypandey/ledgerqueue/internal/job"
)

var ErrNotFound = errors.New("not found")

type LedgerEntry struct {
	ID              int64
	UserID          string
	AmountCents     int64
	EntryType       string
	ExternalEventID string
	Reference       string
	CreatedAt       time.Time
}

type ProviderTransaction struct {
	ID              int64
	ProviderTxnID   string
	ExternalEventID string
	AmountCents     int64
	Currency        string
	SettledAt       time.Time
}

type ReconciliationRun struct {
	ID             int64
	RunAt          time.Time
	PeriodStart    time.Time
	PeriodEnd      time.Time
	MatchedCount   int
	ExceptionCount int
	Status         string
}

type ReconciliationException struct {
	ID        int64
	RunID     int64
	Kind      string
	Details   []byte
	CreatedAt time.Time
}

type Store interface {
	Ping(ctx context.Context) error

	Create(ctx context.Context, j job.Job) error
	Get(ctx context.Context, id string) (job.Job, error)
	GetByIdempotencyKey(ctx context.Context, key string) (job.Job, error)
	Update(ctx context.Context, j job.Job) error

	InsertLedgerCredit(ctx context.Context, externalEventID, userID string, amountCents int64, reference string) (created bool, err error)
	ListLedgerInPeriod(ctx context.Context, start, end time.Time) ([]LedgerEntry, error)

	InsertProviderTransaction(ctx context.Context, providerTxnID, externalEventID string, amountCents int64, currency string, settledAt time.Time) error
	ListProviderInPeriod(ctx context.Context, start, end time.Time) ([]ProviderTransaction, error)

	CreateReconciliationRun(ctx context.Context, start, end time.Time, matched, exceptions int) (int64, error)
	InsertReconciliationException(ctx context.Context, runID int64, kind string, details []byte) error
	LatestReconciliationRun(ctx context.Context) (ReconciliationRun, error)
}
