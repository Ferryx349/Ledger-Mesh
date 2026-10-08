package queue

import "context"

type Queue interface {
	Enqueue(ctx context.Context, jobID string) error
	Dequeue(ctx context.Context) (jobID string, err error)
	Ack(ctx context.Context, jobID string) error
	Requeue(ctx context.Context, jobID string) error
	EnqueueDLQ(ctx context.Context, jobID string) error
	RequeueProcessing(ctx context.Context) (int, error)
	Ping(ctx context.Context) error
	Depth(ctx context.Context) (int64, error)
	DLQDepth(ctx context.Context) (int64, error)
}
