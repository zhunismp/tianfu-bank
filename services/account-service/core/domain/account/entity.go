package account

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	AccountId   string          `json:"account_id"`
	UserId      string          `json:"user_id"`
	BranchId    string          `json:"branch_id"`
	AccountType string          `json:"account_type"`
	Balance     decimal.Decimal `json:"balance"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
