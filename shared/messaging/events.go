package messaging

import "github.com/shopspring/decimal"

// AccountCreatedEvent is published by account-service when a new account is created.
type AccountCreatedEvent struct {
	AccountID   string `json:"account_id"`
	UserID      string `json:"user_id"`
	BranchID    string `json:"branch_id"`
	AccountType string `json:"account_type"`
}

// BalanceUpdatedEvent is published by transaction-service after a successful transaction.
type BalanceUpdatedEvent struct {
	AccountID  string          `json:"account_id"`
	NewBalance decimal.Decimal `json:"new_balance"`
	EventType  string          `json:"event_type"`
	EventID    string          `json:"event_id"`
}
