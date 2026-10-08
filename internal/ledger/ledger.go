package ledger

import (
	"context"

	"github.com/abhaypandey/ledgerqueue/internal/store"
)

// CreditFromPayment idempotently credits a user when a payment webhook is processed.
func CreditFromPayment(ctx context.Context, st store.Store, eventID, userID string, amountCents int64) (created bool, err error) {
	return st.InsertLedgerCredit(ctx, eventID, userID, amountCents, "payment_webhook")
}
