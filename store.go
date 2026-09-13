package main

import (
	"context"
	"errors"
	"sync"
)

var errNotFound = errors.New("job not found")

type Store struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

func newStore() *Store {
	return &Store{jobs: make(map[string]Job)}
}

func (s *Store) create(_ context.Context, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	return nil
}

func (s *Store) get(_ context.Context, id string) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, errNotFound
	}
	return job, nil
}

func (s *Store) update(_ context.Context, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[job.ID]; !ok {
		return errNotFound
	}
	s.jobs[job.ID] = job
	return nil
}
