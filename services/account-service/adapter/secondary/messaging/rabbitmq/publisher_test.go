package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	domain "github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
	"github.com/shopspring/decimal"
)

// ---- inline mock ----

type mockAMQPChannel struct{ mock.Mock }

func (m *mockAMQPChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	args := m.Called(ctx, exchange, key, mandatory, immediate, msg)
	return args.Error(0)
}

// ---- tests ----

func TestAccountPublisher_PublishAccountCreated_Success(t *testing.T) {
	ch := new(mockAMQPChannel)
	pub := NewAccountPublisher(ch)

	acc := &domain.Account{
		AccountId:   "acc-001",
		UserId:      "user-1",
		BranchId:    "branch-1",
		AccountType: "savings",
		Balance:     decimal.Zero,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ch.On("PublishWithContext",
		mock.Anything,
		"tianfu.events",
		"account.created",
		false,
		false,
		mock.MatchedBy(func(msg amqp.Publishing) bool {
			var event map[string]any
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				return false
			}
			return event["account_id"] == "acc-001" &&
				event["user_id"] == "user-1" &&
				msg.ContentType == "application/json"
		}),
	).Return(nil)

	err := pub.PublishAccountCreated(context.Background(), acc)
	require.NoError(t, err)
	ch.AssertExpectations(t)
}

func TestAccountPublisher_PublishAccountCreated_PublishError(t *testing.T) {
	ch := new(mockAMQPChannel)
	pub := NewAccountPublisher(ch)

	acc := &domain.Account{AccountId: "acc-001", UserId: "u", BranchId: "b", AccountType: "savings"}

	ch.On("PublishWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("broker unreachable"))

	err := pub.PublishAccountCreated(context.Background(), acc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish account created")
}
