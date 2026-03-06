package account

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	domain "github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
	"github.com/zhunismp/tianfu-bank/shared/messaging"
)

// BalanceUpdatedConsumer listens for balance.updated events from transaction-service
// and updates the account balance in the local read model.
type BalanceUpdatedConsumer struct {
	ch          AMQPChannel
	accountRepo domain.AccountRepository
}

func NewBalanceUpdatedConsumer(ch AMQPChannel, accountRepo domain.AccountRepository) *BalanceUpdatedConsumer {
	return &BalanceUpdatedConsumer{
		ch:          ch,
		accountRepo: accountRepo,
	}
}

func (c *BalanceUpdatedConsumer) Start(ctx context.Context) error {
	q, err := c.ch.QueueDeclare(
		"account-service.balance.updated", // queue name
		true,                              // durable
		false,                             // auto-delete
		false,                             // exclusive
		false,                             // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	if err := c.ch.QueueBind(
		q.Name,
		messaging.RoutingKeyBalanceUpdated,
		messaging.ExchangeName,
		false,
		nil,
	); err != nil {
		return err
	}

	msgs, err := c.ch.Consume(
		q.Name,
		"",    // consumer tag
		false, // auto-ack - manual for reliability
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	slog.Info("BalanceUpdatedConsumer started listening", "queue", q.Name)

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("BalanceUpdatedConsumer shutting down")
				return
			case msg, ok := <-msgs:
				if !ok {
					slog.Info("BalanceUpdatedConsumer channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *BalanceUpdatedConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var event messaging.BalanceUpdatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("Failed to unmarshal BalanceUpdatedEvent", "error", err)
		msg.Nack(false, false) // discard
		return
	}

	slog.Info("Received balance.updated event", "accountId", event.AccountID, "newBalance", event.NewBalance)

	if err := c.accountRepo.UpdateBalance(ctx, event.AccountID, event.NewBalance); err != nil {
		slog.Error("Failed to update balance in read model", "accountId", event.AccountID, "error", err)
		msg.Nack(false, true) // requeue
		return
	}

	msg.Ack(false)
	slog.Info("Successfully processed balance.updated event", "accountId", event.AccountID)
}
