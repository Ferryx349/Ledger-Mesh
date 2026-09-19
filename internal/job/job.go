package job

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Job struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Status    Status          `json:"status"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func New(id string, req CreateRequest) Job {
	now := time.Now().UTC()
	return Job{
		ID:        id,
		Type:      req.Type,
		Status:    StatusPending,
		Payload:   req.Payload,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
