package queue

import "context"

type Queue interface {
	Enqueue(ctx context.Context, jobID string) error
	Dequeue(ctx context.Context) (jobID string, err error)
	Ping(ctx context.Context) error
}
