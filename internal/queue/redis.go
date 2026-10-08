package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client         *redis.Client
	key            string
	processingKey  string
	dlqKey         string
}

func NewRedis(ctx context.Context, redisURL, queueKey, processingKey, dlqKey string) (*Redis, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	if queueKey == "" {
		queueKey = "jobs:queue"
	}
	if processingKey == "" {
		processingKey = "jobs:processing"
	}
	if dlqKey == "" {
		dlqKey = "jobs:dlq"
	}
	return &Redis{
		client:        client,
		key:           queueKey,
		processingKey: processingKey,
		dlqKey:        dlqKey,
	}, nil
}

func (q *Redis) Close() error {
	return q.client.Close()
}

func (q *Redis) Ping(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

func (q *Redis) Enqueue(ctx context.Context, jobID string) error {
	return q.client.LPush(ctx, q.key, jobID).Err()
}

func (q *Redis) Dequeue(ctx context.Context) (string, error) {
	result, err := q.client.BRPopLPush(ctx, q.key, q.processingKey, 0).Result()
	if err != nil {
		return "", err
	}
	if result == "" {
		return "", fmt.Errorf("queue: empty brpoplpush result")
	}
	return result, nil
}

func (q *Redis) Ack(ctx context.Context, jobID string) error {
	return q.client.LRem(ctx, q.processingKey, 1, jobID).Err()
}

func (q *Redis) Requeue(ctx context.Context, jobID string) error {
	pipe := q.client.Pipeline()
	pipe.LRem(ctx, q.processingKey, 1, jobID)
	pipe.LPush(ctx, q.key, jobID)
	_, err := pipe.Exec(ctx)
	return err
}

func (q *Redis) EnqueueDLQ(ctx context.Context, jobID string) error {
	pipe := q.client.Pipeline()
	pipe.LRem(ctx, q.processingKey, 1, jobID)
	pipe.LPush(ctx, q.dlqKey, jobID)
	_, err := pipe.Exec(ctx)
	return err
}

func (q *Redis) RequeueProcessing(ctx context.Context) (int, error) {
	n := 0
	for {
		jobID, err := q.client.RPopLPush(ctx, q.processingKey, q.key).Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return n, err
		}
		if jobID == "" {
			break
		}
		n++
	}
	return n, nil
}

func (q *Redis) Depth(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.key).Result()
}

func (q *Redis) DLQDepth(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.dlqKey).Result()
}
