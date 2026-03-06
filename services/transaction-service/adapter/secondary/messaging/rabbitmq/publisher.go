package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"github.com/zhunismp/tianfu-bank/shared/messaging"
)

type balancePublisher struct {
	ch AMQPChannel
}

func NewBalancePublisher(ch AMQPChannel) *balancePublisher {
	return &balancePublisher{ch: ch}
}

func (p *balancePublisher) PublishBalanceUpdated(ctx context.Context, accountID string, newBalance decimal.Decimal, eventType string, eventID string) error {
	event := messaging.BalanceUpdatedEvent{
		AccountID:  accountID,
		NewBalance: newBalance,
		EventType:  eventType,
		EventID:    eventID,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal balance updated event: %w", err)
	}

	if err := p.ch.PublishWithContext(ctx,
		messaging.ExchangeName,
		messaging.RoutingKeyBalanceUpdated,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		return fmt.Errorf("publish balance updated: %w", err)
	}

	slog.Info("Published balance.updated event", "accountId", accountID, "newBalance", newBalance.String())
	return nil
}
