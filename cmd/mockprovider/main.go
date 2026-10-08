package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/abhaypandey/ledgerqueue/internal/config"
	"github.com/abhaypandey/ledgerqueue/internal/payment"
	"github.com/abhaypandey/ledgerqueue/internal/store"
	"github.com/google/uuid"
)

func main() {
	cfg := config.LoadMockProvider()
	ctx := context.Background()

	st, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer st.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	log.Printf("mock provider sending %d payments to %s", cfg.Count, cfg.APIURL)

	for i := 0; i < cfg.Count; i++ {
		ev := payment.WebhookEvent{
			EventID:     fmt.Sprintf("evt_%s", uuid.NewString()),
			Type:        "payment.succeeded",
			UserID:      fmt.Sprintf("user_%d", i+1),
			AmountCents: int64((i + 1) * 1000),
			Currency:    "USD",
		}
		if err := sendWebhook(ctx, client, cfg, ev); err != nil {
			log.Printf("webhook %s failed: %v", ev.EventID, err)
		} else {
			log.Printf("webhook sent event_id=%s user=%s amount=%d", ev.EventID, ev.UserID, ev.AmountCents)
		}

		providerTxnID := fmt.Sprintf("ptxn_%s", ev.EventID)
		settledAt := time.Now().UTC()
		if err := st.InsertProviderTransaction(ctx, providerTxnID, ev.EventID, ev.AmountCents, ev.Currency, settledAt); err != nil {
			log.Printf("provider row %s: %v", providerTxnID, err)
		}

		if cfg.Interval > 0 {
			time.Sleep(cfg.Interval)
		}
	}

	if cfg.OrphanTxn {
		orphanID := fmt.Sprintf("evt_orphan_%s", uuid.NewString())
		providerTxnID := fmt.Sprintf("ptxn_%s", orphanID)
		log.Printf("inserting orphan provider txn (no ledger) event_id=%s", orphanID)
		_ = st.InsertProviderTransaction(ctx, providerTxnID, orphanID, 5000, "USD", time.Now().UTC())
	}

	log.Println("mock provider finished")
}

func sendWebhook(ctx context.Context, client *http.Client, cfg config.MockProvider, ev payment.WebhookEvent) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.APIURL+"/webhooks/payment", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", payment.Sign(body, cfg.WebhookSecret))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
