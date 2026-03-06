package transaction

import (
	"time"

	"github.com/shopspring/decimal"
)

// Event type constants
const (
	EventDeposited   = "DEPOSITED"
	EventWithdrawn   = "WITHDRAWN"
	EventTransferIn  = "TRANSFER_IN"
	EventTransferOut = "TRANSFER_OUT"
)

// TransactionEvent represents a single immutable event in the event store.
type TransactionEvent struct {
	ID              string          `json:"id"`
	AccountID       string          `json:"account_id"`
	EventType       string          `json:"event_type"`
	Amount          decimal.Decimal `json:"amount"`
	ReferenceID     string          `json:"reference_id,omitempty"`
	IdempotencyKey  string          `json:"idempotency_key,omitempty"`
	SequenceNumber  int64           `json:"sequence_number"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// AccountSnapshot represents a materialized account state at a point in time.
type AccountSnapshot struct {
	ID                 string          `json:"id"`
	AccountID          string          `json:"account_id"`
	Balance            decimal.Decimal `json:"balance"`
	LastSequenceNumber int64           `json:"last_sequence_number"`
	CreatedAt          time.Time       `json:"created_at"`
}


// IdempotencyRecord tracks processed idempotency keys.
type IdempotencyRecord struct {
	IdempotencyKey string         `json:"idempotency_key"`
	StatusCode     int            `json:"status_code"`
	ResponseBody   map[string]any `json:"response_body,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	ExpiresAt      time.Time      `json:"expires_at"`
}

// TransactionHistoryItem represents a single item in the transaction history response.
type TransactionHistoryItem struct {
	ID             string          `json:"id"`
	EventType      string          `json:"event_type"`
	Amount         decimal.Decimal `json:"amount"`
	BalanceAfter   decimal.Decimal `json:"balance_after"` // Calculated dynamically from replayed events
	ReferenceID    string          `json:"reference_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}
