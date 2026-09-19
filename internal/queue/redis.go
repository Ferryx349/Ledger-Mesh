package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
	key    string
}

func NewRedis(ctx context.Context, redisURL, queueKey string) (*Redis, error) {
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
	return &Redis{client: client, key: queueKey}, nil
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
	result, err := q.client.BRPop(ctx, 0, q.key).Result()
	if err != nil {
		return "", err
	}
	if len(result) != 2 {
		return "", fmt.Errorf("queue: unexpected brpop result")
	}
	return result[1], nil
}
