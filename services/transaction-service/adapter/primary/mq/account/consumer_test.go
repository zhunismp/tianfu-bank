package account

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
)

// ---- inline mocks ----

type mockAMQPChannel struct{ mock.Mock }

func (m *mockAMQPChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	a := m.Called(name, durable, autoDelete, exclusive, noWait, args)
	return a.Get(0).(amqp.Queue), a.Error(1)
}

func (m *mockAMQPChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return m.Called(name, key, exchange, noWait, args).Error(0)
}

func (m *mockAMQPChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	a := m.Called(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	if a.Get(0) == nil {
		return nil, a.Error(1)
	}
	return a.Get(0).(<-chan amqp.Delivery), a.Error(1)
}

type mockSnapshotRepo struct{ mock.Mock }

func (m *mockSnapshotRepo) GetLatestSnapshot(ctx context.Context, accountID string) (*domain.AccountSnapshot, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AccountSnapshot), args.Error(1)
}

func (m *mockSnapshotRepo) CreateSnapshot(ctx context.Context, snapshot *domain.AccountSnapshot) error {
	return m.Called(ctx, snapshot).Error(0)
}

type mockEventStore struct{ mock.Mock }

func (m *mockEventStore) AppendEvent(ctx context.Context, event *domain.TransactionEvent) error {
	return m.Called(ctx, event).Error(0)
}

func (m *mockEventStore) GetEventsSince(ctx context.Context, accountID string, seq int64) ([]domain.TransactionEvent, error) {
	args := m.Called(ctx, accountID, seq)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.TransactionEvent), args.Error(1)
}

// stubAcknowledger records which ack/nack was called.
type stubAcknowledger struct {
	acked       bool
	nacked      bool
	nackRequeue bool
}

func (s *stubAcknowledger) Ack(_ uint64, _ bool) error {
	s.acked = true
	return nil
}

func (s *stubAcknowledger) Nack(_ uint64, _ bool, requeue bool) error {
	s.nacked = true
	s.nackRequeue = requeue
	return nil
}

func (s *stubAcknowledger) Reject(_ uint64, _ bool) error { return nil }

func delivery(body []byte, ack *stubAcknowledger) amqp.Delivery {
	return amqp.Delivery{Body: body, Acknowledger: ack}
}

// ---- handleMessage tests ----

func TestAccountCreatedConsumer_HandleMessage_MalformedJSONIsDiscarded(t *testing.T) {
	sr := new(mockSnapshotRepo)
	es := new(mockEventStore)
	ack := &stubAcknowledger{}
	c := NewAccountCreatedConsumer(new(mockAMQPChannel), sr, es)

	c.handleMessage(context.Background(), delivery([]byte("{bad}"), ack))

	assert.False(t, ack.acked)
	assert.True(t, ack.nacked)
	assert.False(t, ack.nackRequeue, "malformed message must be discarded, not requeued")
	sr.AssertNotCalled(t, "CreateSnapshot")
}

func TestAccountCreatedConsumer_HandleMessage_CreatesInitialSnapshotWithZeroBalance(t *testing.T) {
	sr := new(mockSnapshotRepo)
	es := new(mockEventStore)
	ack := &stubAcknowledger{}
	c := NewAccountCreatedConsumer(new(mockAMQPChannel), sr, es)

	body := []byte(`{"account_id":"acc-1","user_id":"u1","branch_id":"b1","account_type":"savings"}`)
	sr.On("CreateSnapshot", mock.Anything, mock.MatchedBy(func(s *domain.AccountSnapshot) bool {
		return s.AccountID == "acc-1" &&
			s.Balance.Equal(decimal.Zero) &&
			s.LastSequenceNumber == 0
	})).Return(nil)

	c.handleMessage(context.Background(), delivery(body, ack))

	assert.True(t, ack.acked)
	assert.False(t, ack.nacked)
	sr.AssertExpectations(t)
}

func TestAccountCreatedConsumer_HandleMessage_SnapshotFailureStillAcks(t *testing.T) {
	// Snapshot creation is non-critical — failure is logged and message is still acked.
	sr := new(mockSnapshotRepo)
	es := new(mockEventStore)
	ack := &stubAcknowledger{}
	c := NewAccountCreatedConsumer(new(mockAMQPChannel), sr, es)

	body := []byte(`{"account_id":"acc-1","user_id":"u1","branch_id":"b1","account_type":"savings"}`)
	sr.On("CreateSnapshot", mock.Anything, mock.Anything).Return(errors.New("db error"))

	c.handleMessage(context.Background(), delivery(body, ack))

	// Must still ack — snapshot failure is not fatal
	assert.True(t, ack.acked, "message must be acked even when snapshot creation fails")
	assert.False(t, ack.nacked)
}

// ---- Start tests ----

func TestAccountCreatedConsumer_Start_QueueDeclareFailureReturnsError(t *testing.T) {
	ch := new(mockAMQPChannel)
	c := NewAccountCreatedConsumer(ch, new(mockSnapshotRepo), new(mockEventStore))

	ch.On("QueueDeclare", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(amqp.Queue{}, errors.New("declare error"))

	err := c.Start(context.Background())
	require.Error(t, err)
}

func TestAccountCreatedConsumer_Start_QueueBindFailureReturnsError(t *testing.T) {
	ch := new(mockAMQPChannel)
	c := NewAccountCreatedConsumer(ch, new(mockSnapshotRepo), new(mockEventStore))

	ch.On("QueueDeclare", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(amqp.Queue{Name: "q"}, nil)
	ch.On("QueueBind", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("bind error"))

	err := c.Start(context.Background())
	require.Error(t, err)
}

func TestAccountCreatedConsumer_Start_ConsumeFailureReturnsError(t *testing.T) {
	ch := new(mockAMQPChannel)
	c := NewAccountCreatedConsumer(ch, new(mockSnapshotRepo), new(mockEventStore))

	ch.On("QueueDeclare", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(amqp.Queue{Name: "q"}, nil)
	ch.On("QueueBind", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ch.On("Consume", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("consume error"))

	err := c.Start(context.Background())
	require.Error(t, err)
}

func TestAccountCreatedConsumer_Start_StartsListeningWithoutError(t *testing.T) {
	ch := new(mockAMQPChannel)
	c := NewAccountCreatedConsumer(ch, new(mockSnapshotRepo), new(mockEventStore))

	msgs := make(chan amqp.Delivery)
	ch.On("QueueDeclare", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(amqp.Queue{Name: "q"}, nil)
	ch.On("QueueBind", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ch.On("Consume", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return((<-chan amqp.Delivery)(msgs), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := c.Start(ctx)
	require.NoError(t, err)
}
