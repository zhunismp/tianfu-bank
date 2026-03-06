package transaction

import (
	"context"

	"github.com/shopspring/decimal"
)

// EventStoreRepository persists and retrieves events from the event store.
type EventStoreRepository interface {
	AppendEvent(ctx context.Context, event *TransactionEvent) error
	GetEventsSince(ctx context.Context, accountID string, sequenceNumber int64) ([]TransactionEvent, error)
}

// SnapshotRepository manages account snapshots.
type SnapshotRepository interface {
	GetLatestSnapshot(ctx context.Context, accountID string) (*AccountSnapshot, error)
	CreateSnapshot(ctx context.Context, snapshot *AccountSnapshot) error
}

// IdempotencyRepository manages idempotency key tracking.
type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)
	Save(ctx context.Context, record *IdempotencyRecord) error
}

// TxAdapters groups the full repository instances bound to a single DB transaction.
type TxAdapters struct {
	EventStore   EventStoreRepository
	SnapshotRepo SnapshotRepository
	Idempotency  IdempotencyRepository
}

// TransactionProvider executes fn inside a database transaction, providing
// fresh repo instances already bound to that transaction.
type TransactionProvider interface {
	Transact(fn func(tx TxAdapters) error) error
}

// EventPublisher publishes domain events to the message broker.
type EventPublisher interface {
	PublishBalanceUpdated(ctx context.Context, accountID string, newBalance decimal.Decimal, eventType string, eventID string) error
}

// --- Commands & Queries ---

type DepositCmd struct {
	AccountID      string
	Amount         decimal.Decimal
	IdempotencyKey string
}

type WithdrawCmd struct {
	AccountID      string
	Amount         decimal.Decimal
	IdempotencyKey string
}

type TransferCmd struct {
	SourceAccountID      string
	DestinationAccountID string
	Amount               decimal.Decimal
	IdempotencyKey       string
}

type GetHistoryQuery struct {
	AccountID string
	Limit     int
	Offset    int
}

// TransactionService defines the primary port for transaction operations.
type TransactionService interface {
	Deposit(ctx context.Context, cmd *DepositCmd) (*TransactionEvent, error)
	Withdraw(ctx context.Context, cmd *WithdrawCmd) (*TransactionEvent, error)
	Transfer(ctx context.Context, cmd *TransferCmd) ([]*TransactionEvent, error)
	GetHistory(ctx context.Context, query *GetHistoryQuery) ([]TransactionHistoryItem, error)
}
