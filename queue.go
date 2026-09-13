package main

import "context"

type Queue struct {
	ch chan string
}

func newQueue(size int) *Queue {
	return &Queue{ch: make(chan string, size)}
}

func (q *Queue) enqueue(ctx context.Context, jobID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- jobID:
		return nil
	}
}

func (q *Queue) dequeue(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case jobID := <-q.ch:
		return jobID, nil
	}
}
