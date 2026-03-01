package account

type CreateAccountRequest struct {
	UserId      string `json:"user_id" validate:"required"`
	BranchId    string `json:"branch_id" validate:"required"`
	AccountType string `json:"account_type" validate:"required,oneof=savings checking"`
}

type CreateAccountResponse struct {
	AccountId   string `json:"account_id"`
	AccountType string `json:"account_type"`
	CreatedAt   string `json:"created_at"`
}

type GetAccountResponse struct {
	AccountId   string  `json:"account_id"`
	AccountType string  `json:"account_type"`
	Balance     float64 `json:"balance"`
	UpdatedAt   string  `json:"updated_at"`
}
