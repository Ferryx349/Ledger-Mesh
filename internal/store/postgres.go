package store

import (
	"context"
	"errors"

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
		INSERT INTO jobs (id, type, status, payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		j.ID, j.Type, j.Status, j.Payload, j.CreatedAt, j.UpdatedAt,
	)
	return err
}

func (s *Postgres) Get(ctx context.Context, id string) (job.Job, error) {
	var j job.Job
	err := s.pool.QueryRow(ctx, `
		SELECT id, type, status, payload, created_at, updated_at
		FROM jobs WHERE id = $1`, id,
	).Scan(&j.ID, &j.Type, &j.Status, &j.Payload, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.Job{}, ErrNotFound
	}
	return j, err
}

func (s *Postgres) Update(ctx context.Context, j job.Job) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE jobs SET type = $2, status = $3, payload = $4, updated_at = $5
		WHERE id = $1`,
		j.ID, j.Type, j.Status, j.Payload, j.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
