package payment

import "encoding/json"

// WebhookEvent is the payload sent by the mock payment provider (Stripe-shaped).
type WebhookEvent struct {
	EventID     string `json:"event_id"`
	Type        string `json:"type"`
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

func ParseWebhookPayload(raw json.RawMessage) (WebhookEvent, error) {
	var e WebhookEvent
	err := json.Unmarshal(raw, &e)
	return e, err
}

func (e WebhookEvent) Valid() bool {
	return e.EventID != "" && e.UserID != "" && e.AmountCents > 0 && e.Type == "payment.succeeded"
}
