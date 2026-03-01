package account

import "context"

type accountService struct {
	accountRepo AccountRepository
}

func NewAccountService(accountRepo AccountRepository) AccountService {
	return &accountService{accountRepo: accountRepo}
}

func (s *accountService) CreateAccount(ctx context.Context, cmd *CreateAccountCmd) (*Account, error) {
	if cmd == nil {
		return nil, nil
	}

	acc, err := s.accountRepo.CreateAccount(ctx, cmd.UserId, cmd.BranchId, cmd.AccountType)
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func (s *accountService) GetAccountById(ctx context.Context, query *GetAccountQuery) (*Account, error) {
	if query == nil {
		return nil, nil
	}
	
	return s.accountRepo.GetAccountById(ctx, query.AccountId)
}
