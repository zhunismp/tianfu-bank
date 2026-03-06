package transaction

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type DepositRequest struct {
	AccountID string          `json:"account_id"`
	Amount    decimal.Decimal `json:"amount"`
}

func (r *DepositRequest) Validate() error {
	if r.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if !r.Amount.IsPositive() {
		return fmt.Errorf("amount must be greater than 0")
	}
	return nil
}

type WithdrawRequest struct {
	AccountID string          `json:"account_id"`
	Amount    decimal.Decimal `json:"amount"`
}

func (r *WithdrawRequest) Validate() error {
	if r.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if !r.Amount.IsPositive() {
		return fmt.Errorf("amount must be greater than 0")
	}
	return nil
}

type TransferRequest struct {
	SourceAccountID      string          `json:"source_account_id"`
	DestinationAccountID string          `json:"destination_account_id"`
	Amount               decimal.Decimal `json:"amount"`
}

func (r *TransferRequest) Validate() error {
	if r.SourceAccountID == "" {
		return fmt.Errorf("source_account_id is required")
	}
	if r.DestinationAccountID == "" {
		return fmt.Errorf("destination_account_id is required")
	}
	if !r.Amount.IsPositive() {
		return fmt.Errorf("amount must be greater than 0")
	}
	return nil
}

type TransactionResponse struct {
	ID           string `json:"id"`
	AccountID    string `json:"account_id"`
	EventType    string `json:"event_type"`
	Amount       string `json:"amount"`
	BalanceAfter string `json:"balance_after,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type TransferResponse struct {
	SourceEvent      *TransactionResponse `json:"source_event"`
	DestinationEvent *TransactionResponse `json:"destination_event"`
}

type HistoryResponse struct {
	AccountID    string                `json:"account_id"`
	Transactions []TransactionResponse `json:"transactions"`
}
