package transaction

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDepositRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       DepositRequest
		wantErr   bool
		errContains string
	}{
		{
			name:        "empty AccountID",
			req:         DepositRequest{AccountID: "", Amount: decimal.NewFromInt(100)},
			wantErr:     true,
			errContains: "account_id",
		},
		{
			name:        "zero amount",
			req:         DepositRequest{AccountID: "acc-1", Amount: decimal.Zero},
			wantErr:     true,
			errContains: "amount",
		},
		{
			name:        "negative amount",
			req:         DepositRequest{AccountID: "acc-1", Amount: decimal.NewFromInt(-5)},
			wantErr:     true,
			errContains: "amount",
		},
		{
			name:    "valid request",
			req:     DepositRequest{AccountID: "acc-1", Amount: decimal.NewFromInt(100)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWithdrawRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		req         WithdrawRequest
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty AccountID",
			req:         WithdrawRequest{AccountID: "", Amount: decimal.NewFromInt(50)},
			wantErr:     true,
			errContains: "account_id",
		},
		{
			name:        "zero amount",
			req:         WithdrawRequest{AccountID: "acc-1", Amount: decimal.Zero},
			wantErr:     true,
			errContains: "amount",
		},
		{
			name:        "negative amount",
			req:         WithdrawRequest{AccountID: "acc-1", Amount: decimal.NewFromInt(-1)},
			wantErr:     true,
			errContains: "amount",
		},
		{
			name:    "valid request",
			req:     WithdrawRequest{AccountID: "acc-1", Amount: decimal.NewFromInt(50)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransferRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		req         TransferRequest
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty SourceAccountID",
			req:         TransferRequest{SourceAccountID: "", DestinationAccountID: "dst", Amount: decimal.NewFromInt(100)},
			wantErr:     true,
			errContains: "source_account_id",
		},
		{
			name:        "empty DestinationAccountID",
			req:         TransferRequest{SourceAccountID: "src", DestinationAccountID: "", Amount: decimal.NewFromInt(100)},
			wantErr:     true,
			errContains: "destination_account_id",
		},
		{
			name:        "zero amount",
			req:         TransferRequest{SourceAccountID: "src", DestinationAccountID: "dst", Amount: decimal.Zero},
			wantErr:     true,
			errContains: "amount",
		},
		{
			name:        "negative amount",
			req:         TransferRequest{SourceAccountID: "src", DestinationAccountID: "dst", Amount: decimal.NewFromInt(-10)},
			wantErr:     true,
			errContains: "amount",
		},
		{
			name:    "valid request",
			req:     TransferRequest{SourceAccountID: "src", DestinationAccountID: "dst", Amount: decimal.NewFromInt(100)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
