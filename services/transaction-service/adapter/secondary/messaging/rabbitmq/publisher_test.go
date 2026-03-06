package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
)

// ---- inline mock ----

type mockAMQPChannel struct{ mock.Mock }

func (m *mockAMQPChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	args := m.Called(ctx, exchange, key, mandatory, immediate, msg)
	return args.Error(0)
}

// ---- tests ----

func TestBalancePublisher_PublishBalanceUpdated_Success(t *testing.T) {
	ch := new(mockAMQPChannel)
	pub := NewBalancePublisher(ch)

	// Ensure it satisfies the domain interface at compile time
	var _ domain.EventPublisher = pub

	ch.On("PublishWithContext",
		mock.Anything,
		"tianfu.events",
		"balance.updated",
		false,
		false,
		mock.MatchedBy(func(msg amqp.Publishing) bool {
			var event map[string]any
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				return false
			}
			return event["account_id"] == "acc-1" &&
				event["event_type"] == domain.EventDeposited &&
				msg.ContentType == "application/json"
		}),
	).Return(nil)

	err := pub.PublishBalanceUpdated(context.Background(), "acc-1", decimal.NewFromInt(200), domain.EventDeposited, "evt-1")
	require.NoError(t, err)
	ch.AssertExpectations(t)
}

func TestBalancePublisher_PublishBalanceUpdated_PublishError(t *testing.T) {
	ch := new(mockAMQPChannel)
	pub := NewBalancePublisher(ch)

	ch.On("PublishWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("broker unreachable"))

	err := pub.PublishBalanceUpdated(context.Background(), "acc-1", decimal.NewFromInt(100), domain.EventDeposited, "evt-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish balance updated")
}
