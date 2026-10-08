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
	StatusDead      Status = "dead"
)

const (
	TypeProcessWebhook = "process_webhook"
	TypeReconcileDaily = "reconcile_daily"
)

type Job struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Status         Status          `json:"status"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	Attempts       int             `json:"attempts"`
	MaxAttempts    int             `json:"max_attempts"`
	LastError      string          `json:"last_error,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateRequest struct {
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	MaxAttempts    int             `json:"max_attempts,omitempty"`
}

func New(id string, req CreateRequest) Job {
	now := time.Now().UTC()
	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return Job{
		ID:             id,
		Type:           req.Type,
		Status:         StatusPending,
		Payload:        req.Payload,
		Attempts:       0,
		MaxAttempts:    maxAttempts,
		IdempotencyKey: req.IdempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
