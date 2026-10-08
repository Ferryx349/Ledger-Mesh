package reconcile

import (
	"context"
	"encoding/json"
	"time"

	"github.com/abhaypandey/ledgerqueue/internal/store"
)

const (
	KindMissingInLedger   = "missing_in_ledger"
	KindMissingInProvider = "missing_in_provider"
	KindAmountMismatch    = "amount_mismatch"
)

type Result struct {
	RunID          int64
	MatchedCount   int
	ExceptionCount int
}

// RunDaily compares ledger credits to provider settlement rows for [start, end).
func RunDaily(ctx context.Context, st store.Store, start, end time.Time) (Result, error) {
	ledgerRows, err := st.ListLedgerInPeriod(ctx, start, end)
	if err != nil {
		return Result{}, err
	}
	providerRows, err := st.ListProviderInPeriod(ctx, start, end)
	if err != nil {
		return Result{}, err
	}

	ledgerByEvent := make(map[string]store.LedgerEntry, len(ledgerRows))
	for _, e := range ledgerRows {
		ledgerByEvent[e.ExternalEventID] = e
	}
	providerByEvent := make(map[string]store.ProviderTransaction, len(providerRows))
	for _, t := range providerRows {
		providerByEvent[t.ExternalEventID] = t
	}

	matched := 0
	var exceptions []struct {
		kind    string
		details map[string]any
	}

	for eventID, le := range ledgerByEvent {
		pt, ok := providerByEvent[eventID]
		if !ok {
			exceptions = append(exceptions, struct {
				kind    string
				details map[string]any
			}{KindMissingInProvider, map[string]any{"external_event_id": eventID, "ledger_amount_cents": le.AmountCents}})
			continue
		}
		if pt.AmountCents != le.AmountCents {
			exceptions = append(exceptions, struct {
				kind    string
				details map[string]any
			}{KindAmountMismatch, map[string]any{
				"external_event_id":   eventID,
				"ledger_amount_cents": le.AmountCents,
				"provider_amount_cents": pt.AmountCents,
			}})
			continue
		}
		matched++
	}

	for eventID, pt := range providerByEvent {
		if _, ok := ledgerByEvent[eventID]; !ok {
			exceptions = append(exceptions, struct {
				kind    string
				details map[string]any
			}{KindMissingInLedger, map[string]any{
				"external_event_id":     eventID,
				"provider_txn_id":       pt.ProviderTxnID,
				"provider_amount_cents": pt.AmountCents,
			}})
		}
	}

	runID, err := st.CreateReconciliationRun(ctx, start, end, matched, len(exceptions))
	if err != nil {
		return Result{}, err
	}
	for _, ex := range exceptions {
		raw, _ := json.Marshal(ex.details)
		if err := st.InsertReconciliationException(ctx, runID, ex.kind, raw); err != nil {
			return Result{}, err
		}
	}

	return Result{RunID: runID, MatchedCount: matched, ExceptionCount: len(exceptions)}, nil
}

// DefaultPeriod is the last 24 hours in UTC (demo-friendly; production often uses calendar days).
func DefaultPeriod(now time.Time) (start, end time.Time) {
	end = now.UTC()
	start = end.Add(-24 * time.Hour)
	return start, end
}
