package transaction

import (
	"fmt"

	"github.com/shopspring/decimal"
)

const (
	// SnapshotInterval defines how many events trigger a new snapshot.
	SnapshotInterval = 1000
)

// AccountAggregate is the event-sourced aggregate for an account's balance.
// It is rehydrated from a snapshot + subsequent events before each transaction.
type AccountAggregate struct {
	AccountID      string
	Balance        decimal.Decimal
	SequenceNumber int64
}

// NewAccountAggregate creates a fresh aggregate with zero balance.
func NewAccountAggregate(accountID string) *AccountAggregate {
	return &AccountAggregate{
		AccountID:      accountID,
		Balance:        decimal.Zero,
		SequenceNumber: 0,
	}
}

// Rehydrate rebuilds aggregate state from a snapshot and subsequent events.
func (a *AccountAggregate) Rehydrate(snapshot *AccountSnapshot, events []TransactionEvent) {
	if snapshot != nil {
		a.Balance = snapshot.Balance
		a.SequenceNumber = snapshot.LastSequenceNumber
	}
	for _, e := range events {
		a.apply(e)
	}
}

// apply mutates the aggregate state based on a single event.
func (a *AccountAggregate) apply(event TransactionEvent) {
	switch event.EventType {
	case EventDeposited, EventTransferIn:
		a.Balance = a.Balance.Add(event.Amount)
	case EventWithdrawn, EventTransferOut:
		a.Balance = a.Balance.Sub(event.Amount)
	}
	a.SequenceNumber = event.SequenceNumber
}

// Deposit creates a deposit event. Returns the event to be persisted.
func (a *AccountAggregate) Deposit(amount decimal.Decimal, idempotencyKey string) (*TransactionEvent, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("deposit amount must be positive")
	}

	newBalance := a.Balance.Add(amount)
	nextSeq := a.SequenceNumber + 1

	event := &TransactionEvent{
		AccountID:      a.AccountID,
		EventType:      EventDeposited,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
		SequenceNumber: nextSeq,
	}

	a.Balance = newBalance
	a.SequenceNumber = nextSeq
	return event, nil
}

// Withdraw creates a withdrawal event. Returns error if insufficient balance.
func (a *AccountAggregate) Withdraw(amount decimal.Decimal, idempotencyKey string) (*TransactionEvent, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("withdrawal amount must be positive")
	}

	newBalance := a.Balance.Sub(amount)
	if newBalance.LessThan(decimal.Zero) {
		return nil, fmt.Errorf("insufficient balance: current=%s, requested=%s", a.Balance.String(), amount.String())
	}

	nextSeq := a.SequenceNumber + 1

	event := &TransactionEvent{
		AccountID:      a.AccountID,
		EventType:      EventWithdrawn,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
		SequenceNumber: nextSeq,
	}

	a.Balance = newBalance
	a.SequenceNumber = nextSeq
	return event, nil
}

// TransferOut creates a transfer-out event on the source account.
func (a *AccountAggregate) TransferOut(amount decimal.Decimal, destinationAccountID string, idempotencyKey string) (*TransactionEvent, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("transfer amount must be positive")
	}

	newBalance := a.Balance.Sub(amount)
	if newBalance.LessThan(decimal.Zero) {
		return nil, fmt.Errorf("insufficient balance for transfer: current=%s, requested=%s", a.Balance.String(), amount.String())
	}

	nextSeq := a.SequenceNumber + 1

	event := &TransactionEvent{
		AccountID:      a.AccountID,
		EventType:      EventTransferOut,
		Amount:         amount,
		ReferenceID:    destinationAccountID,
		IdempotencyKey: idempotencyKey + ":out",
		SequenceNumber: nextSeq,
	}

	a.Balance = newBalance
	a.SequenceNumber = nextSeq
	return event, nil
}

// TransferIn creates a transfer-in event on the destination account.
func (a *AccountAggregate) TransferIn(amount decimal.Decimal, sourceAccountID string, idempotencyKey string) (*TransactionEvent, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("transfer amount must be positive")
	}

	newBalance := a.Balance.Add(amount)
	nextSeq := a.SequenceNumber + 1

	event := &TransactionEvent{
		AccountID:      a.AccountID,
		EventType:      EventTransferIn,
		Amount:         amount,
		ReferenceID:    sourceAccountID,
		IdempotencyKey: idempotencyKey + ":in",
		SequenceNumber: nextSeq,
	}

	a.Balance = newBalance
	a.SequenceNumber = nextSeq
	return event, nil
}

// ShouldSnapshot returns true if a new snapshot should be created based on SnapshotInterval.
func (a *AccountAggregate) ShouldSnapshot(lastSnapshotSeq int64) bool {
	return a.SequenceNumber-lastSnapshotSeq >= SnapshotInterval
}
