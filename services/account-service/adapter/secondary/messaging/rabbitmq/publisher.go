package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	domain "github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
	"github.com/zhunismp/tianfu-bank/shared/messaging"
)

type accountPublisher struct {
	ch *amqp.Channel
}

func NewAccountPublisher(ch *amqp.Channel) domain.EventPublisher {
	return &accountPublisher{ch: ch}
}

func (p *accountPublisher) PublishAccountCreated(ctx context.Context, account *domain.Account) error {
	event := messaging.AccountCreatedEvent{
		AccountID:   account.AccountId,
		UserID:      account.UserId,
		BranchID:    account.BranchId,
		AccountType: account.AccountType,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal account created event: %w", err)
	}

	if err := p.ch.PublishWithContext(ctx,
		messaging.ExchangeName,
		messaging.RoutingKeyAccountCreated,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		return fmt.Errorf("publish account created: %w", err)
	}

	slog.Info("Published account.created event", "accountId", account.AccountId)
	return nil
}
