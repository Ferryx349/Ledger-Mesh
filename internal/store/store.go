package store

import (
	"context"
	"errors"

	"github.com/abhaypandey/ledgerqueue/internal/job"
)

var ErrNotFound = errors.New("job not found")

type Store interface {
	Create(ctx context.Context, j job.Job) error
	Get(ctx context.Context, id string) (job.Job, error)
	Update(ctx context.Context, j job.Job) error
	Ping(ctx context.Context) error
}
