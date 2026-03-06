package transaction

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---- inline mocks ----

type mockEventStore struct{ mock.Mock }

func (m *mockEventStore) AppendEvent(ctx context.Context, event *TransactionEvent) error {
	return m.Called(ctx, event).Error(0)
}
func (m *mockEventStore) GetEventsSince(ctx context.Context, accountID string, seq int64) ([]TransactionEvent, error) {
	args := m.Called(ctx, accountID, seq)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TransactionEvent), args.Error(1)
}

type mockSnapshotRepo struct{ mock.Mock }

func (m *mockSnapshotRepo) GetLatestSnapshot(ctx context.Context, accountID string) (*AccountSnapshot, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountSnapshot), args.Error(1)
}
func (m *mockSnapshotRepo) CreateSnapshot(ctx context.Context, snapshot *AccountSnapshot) error {
	return m.Called(ctx, snapshot).Error(0)
}

type mockIdempotency struct{ mock.Mock }

func (m *mockIdempotency) Get(ctx context.Context, key string) (*IdempotencyRecord, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*IdempotencyRecord), args.Error(1)
}
func (m *mockIdempotency) Save(ctx context.Context, record *IdempotencyRecord) error {
	return m.Called(ctx, record).Error(0)
}

// mockTxProvider calls through to the callback with the configured adapters,
// propagating any error returned by the callback.
type mockTxProvider struct {
	es *mockEventStore
	sr *mockSnapshotRepo
	ir *mockIdempotency
}

func (m *mockTxProvider) Transact(fn func(TxAdapters) error) error {
	return fn(TxAdapters{EventStore: m.es, SnapshotRepo: m.sr, Idempotency: m.ir})
}

type mockPublisher struct{ mock.Mock }

func (m *mockPublisher) PublishBalanceUpdated(ctx context.Context, accountID string, newBalance decimal.Decimal, eventType, eventID string) error {
	return m.Called(ctx, accountID, newBalance, eventType, eventID).Error(0)
}

// ---- helpers ----

func newProvider() (*mockTxProvider, *mockEventStore, *mockSnapshotRepo, *mockIdempotency) {
	es := new(mockEventStore)
	sr := new(mockSnapshotRepo)
	ir := new(mockIdempotency)
	return &mockTxProvider{es: es, sr: sr, ir: ir}, es, sr, ir
}

// setupRehydrate configures snapshot + empty events so the aggregate loads the given balance.
func setupRehydrate(sr *mockSnapshotRepo, es *mockEventStore, accountID string, balance decimal.Decimal, seq int64) {
	snap := &AccountSnapshot{AccountID: accountID, Balance: balance, LastSequenceNumber: seq}
	sr.On("GetLatestSnapshot", mock.Anything, accountID).Return(snap, nil)
	es.On("GetEventsSince", mock.Anything, accountID, seq).Return([]TransactionEvent{}, nil)
}

// ---- Deposit tests ----

func TestTransactionService_Deposit_NilCmd(t *testing.T) {
	provider, _, _, _ := newProvider()
	svc := NewTransactionService(provider, new(mockPublisher))
	_, err := svc.Deposit(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestTransactionService_Deposit_IdempotentReplay(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	existing := &IdempotencyRecord{
		IdempotencyKey: "key-1",
		StatusCode:     200,
		ResponseBody: map[string]any{
			"id":         "evt-cached",
			"account_id": "acc-1",
			"event_type": EventDeposited,
			"amount":     "100",
			"created_at": time.Now().Format(time.RFC3339),
		},
	}
	ir.On("Get", mock.Anything, "key-1").Return(existing, nil)

	svc := NewTransactionService(provider, publisher)
	event, err := svc.Deposit(context.Background(), &DepositCmd{
		AccountID: "acc-1", Amount: dec("100"), IdempotencyKey: "key-1",
	})

	require.NoError(t, err)
	assert.Equal(t, EventDeposited, event.EventType)
	// publisher must NOT be called on idempotent replay (agg is nil)
	publisher.AssertNotCalled(t, "PublishBalanceUpdated")
	es.AssertNotCalled(t, "AppendEvent")
	_ = sr
}

func TestTransactionService_Deposit_CreatesEventAndPublishesBalance(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	setupRehydrate(sr, es, "acc-1", dec("100"), 5)
	es.On("AppendEvent", mock.Anything, mock.MatchedBy(func(e *TransactionEvent) bool {
		return e.EventType == EventDeposited && e.Amount.Equal(dec("50"))
	})).Return(nil)
	sr.On("CreateSnapshot", mock.Anything, mock.Anything).Return(nil).Maybe()
	ir.On("Save", mock.Anything, mock.Anything).Return(nil)
	publisher.On("PublishBalanceUpdated", mock.Anything, "acc-1", dec("150"), EventDeposited, mock.Anything).Return(nil)

	svc := NewTransactionService(provider, publisher)
	event, err := svc.Deposit(context.Background(), &DepositCmd{
		AccountID: "acc-1", Amount: dec("50"), IdempotencyKey: "key-1",
	})

	require.NoError(t, err)
	assert.Equal(t, EventDeposited, event.EventType)
	publisher.AssertCalled(t, "PublishBalanceUpdated", mock.Anything, "acc-1", dec("150"), EventDeposited, mock.Anything)
}

func TestTransactionService_Deposit_PublishFailureIsSwallowed(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	setupRehydrate(sr, es, "acc-1", dec("100"), 0)
	es.On("AppendEvent", mock.Anything, mock.Anything).Return(nil)
	sr.On("CreateSnapshot", mock.Anything, mock.Anything).Return(nil).Maybe()
	ir.On("Save", mock.Anything, mock.Anything).Return(nil)
	publisher.On("PublishBalanceUpdated", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("broker down"))

	svc := NewTransactionService(provider, publisher)
	event, err := svc.Deposit(context.Background(), &DepositCmd{
		AccountID: "acc-1", Amount: dec("50"), IdempotencyKey: "key-1",
	})

	require.NoError(t, err)
	assert.NotNil(t, event)
}

func TestTransactionService_Deposit_AccountNotFoundReturnsError(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	// no snapshot, no events → account not found
	sr.On("GetLatestSnapshot", mock.Anything, "acc-unknown").Return(nil, nil)
	es.On("GetEventsSince", mock.Anything, "acc-unknown", int64(0)).Return([]TransactionEvent{}, nil)

	svc := NewTransactionService(provider, publisher)
	_, err := svc.Deposit(context.Background(), &DepositCmd{
		AccountID: "acc-unknown", Amount: dec("50"), IdempotencyKey: "key-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")
	publisher.AssertNotCalled(t, "PublishBalanceUpdated")
}

func TestTransactionService_Deposit_AppendEventFailureReturnsError(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	setupRehydrate(sr, es, "acc-1", dec("100"), 0)
	es.On("AppendEvent", mock.Anything, mock.Anything).Return(errors.New("db error"))

	svc := NewTransactionService(provider, publisher)
	_, err := svc.Deposit(context.Background(), &DepositCmd{
		AccountID: "acc-1", Amount: dec("50"), IdempotencyKey: "key-1",
	})

	require.Error(t, err)
	publisher.AssertNotCalled(t, "PublishBalanceUpdated")
}

// ---- Withdraw tests ----

func TestTransactionService_Withdraw_NilCmd(t *testing.T) {
	provider, _, _, _ := newProvider()
	svc := NewTransactionService(provider, new(mockPublisher))
	_, err := svc.Withdraw(context.Background(), nil)
	require.Error(t, err)
}

func TestTransactionService_Withdraw_SufficientBalance(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	setupRehydrate(sr, es, "acc-1", dec("200"), 0)
	es.On("AppendEvent", mock.Anything, mock.MatchedBy(func(e *TransactionEvent) bool {
		return e.EventType == EventWithdrawn
	})).Return(nil)
	sr.On("CreateSnapshot", mock.Anything, mock.Anything).Return(nil).Maybe()
	ir.On("Save", mock.Anything, mock.Anything).Return(nil)
	publisher.On("PublishBalanceUpdated", mock.Anything, "acc-1", dec("150"), EventWithdrawn, mock.Anything).Return(nil)

	svc := NewTransactionService(provider, publisher)
	event, err := svc.Withdraw(context.Background(), &WithdrawCmd{
		AccountID: "acc-1", Amount: dec("50"), IdempotencyKey: "key-1",
	})

	require.NoError(t, err)
	assert.Equal(t, EventWithdrawn, event.EventType)
}

func TestTransactionService_Withdraw_InsufficientBalanceReturnsErrorWithoutAppending(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	setupRehydrate(sr, es, "acc-1", dec("30"), 0)

	svc := NewTransactionService(provider, publisher)
	_, err := svc.Withdraw(context.Background(), &WithdrawCmd{
		AccountID: "acc-1", Amount: dec("100"), IdempotencyKey: "key-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient balance")
	es.AssertNotCalled(t, "AppendEvent")
	publisher.AssertNotCalled(t, "PublishBalanceUpdated")
}

func TestTransactionService_Withdraw_IdempotentReplay(t *testing.T) {
	provider, _, _, ir := newProvider()
	publisher := new(mockPublisher)

	existing := &IdempotencyRecord{
		StatusCode: 200,
		ResponseBody: map[string]any{
			"id":         "evt-cached",
			"account_id": "acc-1",
			"event_type": EventWithdrawn,
			"amount":     "50",
			"created_at": time.Now().Format(time.RFC3339),
		},
	}
	ir.On("Get", mock.Anything, "key-1").Return(existing, nil)

	svc := NewTransactionService(provider, publisher)
	event, err := svc.Withdraw(context.Background(), &WithdrawCmd{
		AccountID: "acc-1", Amount: dec("50"), IdempotencyKey: "key-1",
	})

	require.NoError(t, err)
	assert.Equal(t, EventWithdrawn, event.EventType)
	publisher.AssertNotCalled(t, "PublishBalanceUpdated")
}

// ---- Transfer tests ----

func TestTransactionService_Transfer_NilCmd(t *testing.T) {
	provider, _, _, _ := newProvider()
	svc := NewTransactionService(provider, new(mockPublisher))
	_, err := svc.Transfer(context.Background(), nil)
	require.Error(t, err)
}

func TestTransactionService_Transfer_AppendsBothEventsAndPublishesTwice(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	setupRehydrate(sr, es, "src", dec("500"), 0)
	setupRehydrate(sr, es, "dst", dec("100"), 0)
	es.On("AppendEvent", mock.Anything, mock.MatchedBy(func(e *TransactionEvent) bool {
		return e.EventType == EventTransferOut
	})).Return(nil)
	es.On("AppendEvent", mock.Anything, mock.MatchedBy(func(e *TransactionEvent) bool {
		return e.EventType == EventTransferIn
	})).Return(nil)
	sr.On("CreateSnapshot", mock.Anything, mock.Anything).Return(nil).Maybe()
	ir.On("Save", mock.Anything, mock.Anything).Return(nil)
	publisher.On("PublishBalanceUpdated", mock.Anything, "src", mock.Anything, EventTransferOut, mock.Anything).Return(nil)
	publisher.On("PublishBalanceUpdated", mock.Anything, "dst", mock.Anything, EventTransferIn, mock.Anything).Return(nil)

	svc := NewTransactionService(provider, publisher)
	events, err := svc.Transfer(context.Background(), &TransferCmd{
		SourceAccountID: "src", DestinationAccountID: "dst",
		Amount: dec("200"), IdempotencyKey: "key-1",
	})

	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, EventTransferOut, events[0].EventType)
	assert.Equal(t, EventTransferIn, events[1].EventType)
	publisher.AssertNumberOfCalls(t, "PublishBalanceUpdated", 2)
}

func TestTransactionService_Transfer_IdempotentReplayReturnsNilEvents(t *testing.T) {
	provider, _, _, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(&IdempotencyRecord{StatusCode: 200, ResponseBody: map[string]any{}}, nil)

	svc := NewTransactionService(provider, publisher)
	events, err := svc.Transfer(context.Background(), &TransferCmd{
		SourceAccountID: "src", DestinationAccountID: "dst",
		Amount: dec("100"), IdempotencyKey: "key-1",
	})

	require.NoError(t, err)
	assert.Nil(t, events)
	publisher.AssertNotCalled(t, "PublishBalanceUpdated")
}

func TestTransactionService_Transfer_SourceAccountNotFoundReturnsError(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	sr.On("GetLatestSnapshot", mock.Anything, "src").Return(nil, nil)
	es.On("GetEventsSince", mock.Anything, "src", int64(0)).Return([]TransactionEvent{}, nil)

	svc := NewTransactionService(provider, publisher)
	_, err := svc.Transfer(context.Background(), &TransferCmd{
		SourceAccountID: "src", DestinationAccountID: "dst",
		Amount: dec("100"), IdempotencyKey: "key-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")
	sr.AssertNotCalled(t, "GetLatestSnapshot", mock.Anything, "dst")
}

func TestTransactionService_Transfer_SourceInsufficientBalanceReturnsErrorWithoutAppending(t *testing.T) {
	provider, es, sr, ir := newProvider()
	publisher := new(mockPublisher)

	ir.On("Get", mock.Anything, "key-1").Return(nil, nil)
	setupRehydrate(sr, es, "src", dec("10"), 0)
	setupRehydrate(sr, es, "dst", dec("100"), 0)

	svc := NewTransactionService(provider, publisher)
	_, err := svc.Transfer(context.Background(), &TransferCmd{
		SourceAccountID: "src", DestinationAccountID: "dst",
		Amount: dec("200"), IdempotencyKey: "key-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient balance")
	es.AssertNotCalled(t, "AppendEvent")
}

// ---- GetHistory tests ----

func TestTransactionService_GetHistory_NilQuery(t *testing.T) {
	provider, _, _, _ := newProvider()
	svc := NewTransactionService(provider, new(mockPublisher))
	_, err := svc.GetHistory(context.Background(), nil)
	require.Error(t, err)
}

func TestTransactionService_GetHistory_EmptyAccountReturnsNoItems(t *testing.T) {
	provider, es, _, _ := newProvider()
	es.On("GetEventsSince", mock.Anything, "acc-1", int64(0)).Return([]TransactionEvent{}, nil)

	svc := NewTransactionService(provider, new(mockPublisher))
	items, err := svc.GetHistory(context.Background(), &GetHistoryQuery{AccountID: "acc-1", Limit: 50})
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestTransactionService_GetHistory_CalculatesRunningBalance(t *testing.T) {
	provider, es, _, _ := newProvider()
	events := []TransactionEvent{
		{ID: "e1", EventType: EventDeposited, Amount: dec("100"), SequenceNumber: 1},
		{ID: "e2", EventType: EventWithdrawn, Amount: dec("30"), SequenceNumber: 2},
		{ID: "e3", EventType: EventTransferIn, Amount: dec("50"), SequenceNumber: 3},
	}
	es.On("GetEventsSince", mock.Anything, "acc-1", int64(0)).Return(events, nil)

	svc := NewTransactionService(provider, new(mockPublisher))
	items, err := svc.GetHistory(context.Background(), &GetHistoryQuery{AccountID: "acc-1", Limit: 50})
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.True(t, items[0].BalanceAfter.Equal(dec("100")))
	assert.True(t, items[1].BalanceAfter.Equal(dec("70")))
	assert.True(t, items[2].BalanceAfter.Equal(dec("120")))
}

func TestTransactionService_GetHistory_AppliesOffsetAndLimit(t *testing.T) {
	provider, es, _, _ := newProvider()
	events := make([]TransactionEvent, 5)
	for i := range events {
		events[i] = TransactionEvent{
			ID:             string(rune('a' + i)),
			EventType:      EventDeposited,
			Amount:         dec("10"),
			SequenceNumber: int64(i + 1),
		}
	}
	es.On("GetEventsSince", mock.Anything, "acc-1", int64(0)).Return(events, nil)

	svc := NewTransactionService(provider, new(mockPublisher))
	items, err := svc.GetHistory(context.Background(), &GetHistoryQuery{
		AccountID: "acc-1", Offset: 1, Limit: 2,
	})
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, string(rune('b')), items[0].ID)
}
