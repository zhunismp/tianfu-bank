package transaction_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	. "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/primary/http/transaction"
	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
	"github.com/zhunismp/tianfu-bank/shared/apperror"
)

// ---- inline mock ----

type mockTransactionService struct{ mock.Mock }

func (m *mockTransactionService) Deposit(ctx context.Context, cmd *domain.DepositCmd) (*domain.TransactionEvent, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TransactionEvent), args.Error(1)
}

func (m *mockTransactionService) Withdraw(ctx context.Context, cmd *domain.WithdrawCmd) (*domain.TransactionEvent, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TransactionEvent), args.Error(1)
}

func (m *mockTransactionService) Transfer(ctx context.Context, cmd *domain.TransferCmd) ([]*domain.TransactionEvent, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TransactionEvent), args.Error(1)
}

func (m *mockTransactionService) GetHistory(ctx context.Context, query *domain.GetHistoryQuery) ([]domain.TransactionHistoryItem, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.TransactionHistoryItem), args.Error(1)
}

// ---- helpers ----

func newTxTestApp(svc domain.TransactionService) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	h := NewTransactionHttpHandler(svc)
	app.Post("/transactions/deposit", h.Deposit)
	app.Post("/transactions/withdraw", h.Withdraw)
	app.Post("/transactions/transfer", h.Transfer)
	app.Get("/transactions/history/:accountId", h.GetHistory)
	return app
}

func txRequest(app *fiber.App, method, path string, body any, headers map[string]string) *http.Response {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, _ := app.Test(req)
	return resp
}

func withKey(key string) map[string]string {
	return map[string]string{"X-Idempotency-Key": key}
}

func sampleEvent(eventType string) *domain.TransactionEvent {
	return &domain.TransactionEvent{
		ID:             "evt-1",
		AccountID:      "acc-1",
		EventType:      eventType,
		Amount:         decimal.NewFromInt(100),
		SequenceNumber: 1,
		CreatedAt:      time.Now(),
	}
}

// ---- Deposit handler tests ----

func TestDepositHandler_MissingIdempotencyKey(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	resp := txRequest(app, http.MethodPost, "/transactions/deposit",
		map[string]any{"account_id": "acc-1", "amount": "100"}, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDepositHandler_MalformedJSON(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	req := httptest.NewRequest(http.MethodPost, "/transactions/deposit", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", "key-1")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDepositHandler_EmptyAccountID(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	resp := txRequest(app, http.MethodPost, "/transactions/deposit",
		map[string]any{"account_id": "", "amount": "100"}, withKey("key-1"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDepositHandler_ZeroAmount(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	resp := txRequest(app, http.MethodPost, "/transactions/deposit",
		map[string]any{"account_id": "acc-1", "amount": "0"}, withKey("key-1"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDepositHandler_Success(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	evt := sampleEvent(domain.EventDeposited)
	svc.On("Deposit", mock.Anything, mock.MatchedBy(func(cmd *domain.DepositCmd) bool {
		return cmd.AccountID == "acc-1" && cmd.Amount.Equal(decimal.NewFromInt(100)) && cmd.IdempotencyKey == "key-1"
	})).Return(evt, nil)

	resp := txRequest(app, http.MethodPost, "/transactions/deposit",
		map[string]any{"account_id": "acc-1", "amount": "100"}, withKey("key-1"))

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "evt-1", body["id"])
	assert.Equal(t, domain.EventDeposited, body["event_type"])
}

func TestDepositHandler_ServiceError(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	svc.On("Deposit", mock.Anything, mock.Anything).Return(nil, apperror.New(apperror.ErrCodeInsufficientFunds, "insufficient balance", nil))

	resp := txRequest(app, http.MethodPost, "/transactions/deposit",
		map[string]any{"account_id": "acc-1", "amount": "100"}, withKey("key-1"))
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

// ---- Withdraw handler tests ----

func TestWithdrawHandler_MissingIdempotencyKey(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	resp := txRequest(app, http.MethodPost, "/transactions/withdraw",
		map[string]any{"account_id": "acc-1", "amount": "50"}, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWithdrawHandler_EmptyAccountID(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	resp := txRequest(app, http.MethodPost, "/transactions/withdraw",
		map[string]any{"account_id": "", "amount": "50"}, withKey("key-1"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWithdrawHandler_Success(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	evt := sampleEvent(domain.EventWithdrawn)
	svc.On("Withdraw", mock.Anything, mock.Anything).Return(evt, nil)

	resp := txRequest(app, http.MethodPost, "/transactions/withdraw",
		map[string]any{"account_id": "acc-1", "amount": "50"}, withKey("key-1"))

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestWithdrawHandler_ServiceError(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	svc.On("Withdraw", mock.Anything, mock.Anything).Return(nil, apperror.New(apperror.ErrCodeInsufficientFunds, "insufficient balance", nil))

	resp := txRequest(app, http.MethodPost, "/transactions/withdraw",
		map[string]any{"account_id": "acc-1", "amount": "50"}, withKey("key-1"))
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

// ---- Transfer handler tests ----

func TestTransferHandler_MissingIdempotencyKey(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	resp := txRequest(app, http.MethodPost, "/transactions/transfer",
		map[string]any{"source_account_id": "src", "destination_account_id": "dst", "amount": "100"}, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTransferHandler_EmptySourceAccountID(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	resp := txRequest(app, http.MethodPost, "/transactions/transfer",
		map[string]any{"source_account_id": "", "destination_account_id": "dst", "amount": "100"}, withKey("key-1"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTransferHandler_EmptyDestinationAccountID(t *testing.T) {
	app := newTxTestApp(new(mockTransactionService))
	resp := txRequest(app, http.MethodPost, "/transactions/transfer",
		map[string]any{"source_account_id": "src", "destination_account_id": "", "amount": "100"}, withKey("key-1"))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTransferHandler_Success(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	outEvt := sampleEvent(domain.EventTransferOut)
	outEvt.AccountID = "src"
	inEvt := sampleEvent(domain.EventTransferIn)
	inEvt.ID = "evt-2"
	inEvt.AccountID = "dst"

	svc.On("Transfer", mock.Anything, mock.Anything).Return([]*domain.TransactionEvent{outEvt, inEvt}, nil)

	resp := txRequest(app, http.MethodPost, "/transactions/transfer",
		map[string]any{"source_account_id": "src", "destination_account_id": "dst", "amount": "100"}, withKey("key-1"))

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.NotNil(t, body["source_event"])
	assert.NotNil(t, body["destination_event"])
}

func TestTransferHandler_IdempotentReplay(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	// idempotent transfer: service returns nil, nil
	svc.On("Transfer", mock.Anything, mock.Anything).Return(nil, nil)

	resp := txRequest(app, http.MethodPost, "/transactions/transfer",
		map[string]any{"source_account_id": "src", "destination_account_id": "dst", "amount": "100"}, withKey("key-1"))

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "transfer already processed", body["message"])
}

func TestTransferHandler_ServiceError(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	svc.On("Transfer", mock.Anything, mock.Anything).Return(nil, apperror.New(apperror.ErrCodeAccountNotFound, "account not found: src", nil))

	resp := txRequest(app, http.MethodPost, "/transactions/transfer",
		map[string]any{"source_account_id": "src", "destination_account_id": "dst", "amount": "100"}, withKey("key-1"))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---- GetHistory handler tests ----

func TestGetHistoryHandler_Success(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	items := []domain.TransactionHistoryItem{
		{ID: "e1", EventType: domain.EventDeposited, Amount: decimal.NewFromInt(100), BalanceAfter: decimal.NewFromInt(100), CreatedAt: time.Now()},
	}
	svc.On("GetHistory", mock.Anything, mock.MatchedBy(func(q *domain.GetHistoryQuery) bool {
		return q.AccountID == "acc-1" && q.Limit == 50 && q.Offset == 0
	})).Return(items, nil)

	resp := txRequest(app, http.MethodGet, "/transactions/history/acc-1", nil, nil)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "acc-1", body["account_id"])
	txns := body["transactions"].([]any)
	assert.Len(t, txns, 1)
}

func TestGetHistoryHandler_CustomPagination(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	svc.On("GetHistory", mock.Anything, mock.MatchedBy(func(q *domain.GetHistoryQuery) bool {
		return q.AccountID == "acc-1" && q.Limit == 10 && q.Offset == 5
	})).Return([]domain.TransactionHistoryItem{}, nil)

	resp := txRequest(app, http.MethodGet, "/transactions/history/acc-1?limit=10&offset=5", nil, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	svc.AssertExpectations(t)
}

func TestGetHistoryHandler_ServiceError(t *testing.T) {
	svc := new(mockTransactionService)
	app := newTxTestApp(svc)

	svc.On("GetHistory", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	resp := txRequest(app, http.MethodGet, "/transactions/history/acc-1", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
