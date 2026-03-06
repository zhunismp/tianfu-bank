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
	domain "github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
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

type mockAccountRepo struct{ mock.Mock }

func (m *mockAccountRepo) CreateAccount(ctx context.Context, userId, branchId, accountType string) (*domain.Account, error) {
	args := m.Called(ctx, userId, branchId, accountType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *mockAccountRepo) GetAccountById(ctx context.Context, accountId string) (*domain.Account, error) {
	args := m.Called(ctx, accountId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *mockAccountRepo) UpdateBalance(ctx context.Context, accountId string, balance decimal.Decimal) error {
	return m.Called(ctx, accountId, balance).Error(0)
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

func TestBalanceUpdatedConsumer_HandleMessage_MalformedJSONIsDiscarded(t *testing.T) {
	repo := new(mockAccountRepo)
	ack := &stubAcknowledger{}
	c := NewBalanceUpdatedConsumer(new(mockAMQPChannel), repo)

	c.handleMessage(context.Background(), delivery([]byte("{bad json}"), ack))

	assert.False(t, ack.acked)
	assert.True(t, ack.nacked)
	assert.False(t, ack.nackRequeue, "malformed message must be discarded, not requeued")
	repo.AssertNotCalled(t, "UpdateBalance")
}

func TestBalanceUpdatedConsumer_HandleMessage_ValidMessageUpdatesBalance(t *testing.T) {
	repo := new(mockAccountRepo)
	ack := &stubAcknowledger{}
	c := NewBalanceUpdatedConsumer(new(mockAMQPChannel), repo)

	body := []byte(`{"account_id":"acc-1","new_balance":"150.00","event_type":"DEPOSITED","event_id":"e1"}`)
	repo.On("UpdateBalance", mock.Anything, "acc-1", mock.MatchedBy(func(b decimal.Decimal) bool {
		return b.Equal(decimal.RequireFromString("150.00"))
	})).Return(nil)

	c.handleMessage(context.Background(), delivery(body, ack))

	assert.True(t, ack.acked)
	assert.False(t, ack.nacked)
	repo.AssertExpectations(t)
}

func TestBalanceUpdatedConsumer_HandleMessage_UpdateBalanceFailureRequeues(t *testing.T) {
	repo := new(mockAccountRepo)
	ack := &stubAcknowledger{}
	c := NewBalanceUpdatedConsumer(new(mockAMQPChannel), repo)

	body := []byte(`{"account_id":"acc-1","new_balance":"150.00","event_type":"DEPOSITED","event_id":"e1"}`)
	repo.On("UpdateBalance", mock.Anything, "acc-1", mock.Anything).Return(errors.New("db error"))

	c.handleMessage(context.Background(), delivery(body, ack))

	assert.False(t, ack.acked)
	assert.True(t, ack.nacked)
	assert.True(t, ack.nackRequeue, "db failure must be requeued for retry")
}

// ---- Start tests ----

func TestBalanceUpdatedConsumer_Start_QueueDeclareFailureReturnsError(t *testing.T) {
	ch := new(mockAMQPChannel)
	c := NewBalanceUpdatedConsumer(ch, new(mockAccountRepo))

	ch.On("QueueDeclare", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(amqp.Queue{}, errors.New("declare error"))

	err := c.Start(context.Background())
	require.Error(t, err)
}

func TestBalanceUpdatedConsumer_Start_QueueBindFailureReturnsError(t *testing.T) {
	ch := new(mockAMQPChannel)
	c := NewBalanceUpdatedConsumer(ch, new(mockAccountRepo))

	ch.On("QueueDeclare", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(amqp.Queue{Name: "q"}, nil)
	ch.On("QueueBind", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("bind error"))

	err := c.Start(context.Background())
	require.Error(t, err)
}

func TestBalanceUpdatedConsumer_Start_ConsumeFailureReturnsError(t *testing.T) {
	ch := new(mockAMQPChannel)
	c := NewBalanceUpdatedConsumer(ch, new(mockAccountRepo))

	ch.On("QueueDeclare", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(amqp.Queue{Name: "q"}, nil)
	ch.On("QueueBind", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ch.On("Consume", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("consume error"))

	err := c.Start(context.Background())
	require.Error(t, err)
}

func TestBalanceUpdatedConsumer_Start_StartsListeningWithoutError(t *testing.T) {
	ch := new(mockAMQPChannel)
	c := NewBalanceUpdatedConsumer(ch, new(mockAccountRepo))

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
