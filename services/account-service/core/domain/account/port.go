package account

import "context"

type AccountRepository interface {
	CreateAccount(ctx context.Context, userId string, branchId string, accountType string) (*Account, error)
	GetAccountById(ctx context.Context, accountId string) (*Account, error)
}

type AccountService interface {
	CreateAccount(ctx context.Context, cmd *CreateAccountCmd) (*Account, error)
	GetAccountById(ctx context.Context, query *GetAccountQuery) (*Account, error)
}

type CreateAccountCmd struct {
	UserId      string
	BranchId	string
	AccountType string
}

type GetAccountQuery struct {
	AccountId string
}
