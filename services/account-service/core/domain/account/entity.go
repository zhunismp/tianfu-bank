package account

import "time"

type Account struct {
	AccountId string    `json:"account_id"`
	UserId    string    `json:"user_id"`
	BranchId  string    `json:"branch_id"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
