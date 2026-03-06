package account

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"github.com/zhunismp/tianfu-bank/shared/messaging"
	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
)

// AccountCreatedConsumer listens for account.created events from account-service
// and saves accounts to the local registry + creates initial snapshots.
type AccountCreatedConsumer struct {
	ch           AMQPChannel
	snapshotRepo domain.SnapshotRepository
	eventStore   domain.EventStoreRepository
}

func NewAccountCreatedConsumer(
	ch AMQPChannel,
	snapshotRepo domain.SnapshotRepository,
	eventStore domain.EventStoreRepository,
) *AccountCreatedConsumer {
	return &AccountCreatedConsumer{
		ch:           ch,
		snapshotRepo: snapshotRepo,
		eventStore:   eventStore,
	}
}

func (c *AccountCreatedConsumer) Start(ctx context.Context) error {
	// Declare queue for this consumer
	q, err := c.ch.QueueDeclare(
		"transaction-service.account.created", // queue name
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	// Bind queue to exchange with routing key
	if err := c.ch.QueueBind(
		q.Name,
		messaging.RoutingKeyAccountCreated,
		messaging.ExchangeName,
		false,
		nil,
	); err != nil {
		return err
	}

	msgs, err := c.ch.Consume(
		q.Name,
		"",    // consumer tag
		false, // auto-ack (manual for reliability)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	slog.Info("AccountCreatedConsumer started listening", "queue", q.Name)

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("AccountCreatedConsumer shutting down")
				return
			case msg, ok := <-msgs:
				if !ok {
					slog.Info("AccountCreatedConsumer channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *AccountCreatedConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var event messaging.AccountCreatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("Failed to unmarshal AccountCreatedEvent", "error", err)
		msg.Nack(false, false) // discard malformed
		return
	}

	slog.Info("Received account.created event", "accountId", event.AccountID)


	// Create initial snapshot with balance 0 (sequence 0)
	snapshot := &domain.AccountSnapshot{
		AccountID:          event.AccountID,
		Balance:            decimal.Zero,
		LastSequenceNumber: 0,
	}
	if err := c.snapshotRepo.CreateSnapshot(ctx, snapshot); err != nil {
		slog.Error("Failed to create initial snapshot", "accountId", event.AccountID, "error", err)
		// Not critical; the aggregate will default to 0 balance if snapshot is missing
	}

	msg.Ack(false)
	slog.Info("Successfully processed account.created event", "accountId", event.AccountID)
}
