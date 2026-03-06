package account

import (
	"context"
	"log/slog"
)

type accountService struct {
	accountRepo AccountRepository
	publisher   EventPublisher
}

func NewAccountService(accountRepo AccountRepository, publisher EventPublisher) AccountService {
	return &accountService{accountRepo: accountRepo, publisher: publisher}
}

func (s *accountService) CreateAccount(ctx context.Context, cmd *CreateAccountCmd) (*Account, error) {
	if cmd == nil {
		return nil, nil
	}

	acc, err := s.accountRepo.CreateAccount(ctx, cmd.UserId, cmd.BranchId, cmd.AccountType)
	if err != nil {
		return nil, err
	}

	// Publish event to RabbitMQ for transaction-service
	if err := s.publisher.PublishAccountCreated(ctx, acc); err != nil {
		slog.Error("Failed to publish AccountCreated event", "accountId", acc.AccountId, "error", err)
		// Do not fail the request; the event can be retried
	}

	return acc, nil
}

func (s *accountService) GetAccountById(ctx context.Context, query *GetAccountQuery) (*Account, error) {
	if query == nil {
		return nil, nil
	}

	return s.accountRepo.GetAccountById(ctx, query.AccountId)
}
