package account_test

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
	domain "github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
	. "github.com/zhunismp/tianfu-bank/services/account-service/adapter/primary/http/account"
)

// ---- inline mock ----

type mockAccountService struct{ mock.Mock }

func (m *mockAccountService) CreateAccount(ctx context.Context, cmd *domain.CreateAccountCmd) (*domain.Account, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *mockAccountService) GetAccountById(ctx context.Context, query *domain.GetAccountQuery) (*domain.Account, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

// ---- helpers ----

func newTestApp(svc domain.AccountService) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	h := NewAccountHttpHandler(svc)
	app.Post("/accounts", h.CreateAccount)
	app.Get("/accounts/:accountId", h.GetAccount)
	return app
}

func doRequest(app *fiber.App, method, path string, body any) *http.Response {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, _ := app.Test(req)
	return resp
}

func sampleAccount() *domain.Account {
	return &domain.Account{
		AccountId:   "acc-001",
		UserId:      "user-1",
		BranchId:    "branch-1",
		AccountType: "savings",
		Balance:     decimal.Zero,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ---- CreateAccount handler tests ----

func TestCreateAccountHandler_MalformedJSON(t *testing.T) {
	app := newTestApp(new(mockAccountService))
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateAccountHandler_Success(t *testing.T) {
	svc := new(mockAccountService)
	app := newTestApp(svc)

	acc := sampleAccount()
	svc.On("CreateAccount", mock.Anything, mock.MatchedBy(func(cmd *domain.CreateAccountCmd) bool {
		return cmd.UserId == "user-1" && cmd.BranchId == "branch-1" && cmd.AccountType == "savings"
	})).Return(acc, nil)

	resp := doRequest(app, http.MethodPost, "/accounts", map[string]string{
		"user_id":      "user-1",
		"branch_id":    "branch-1",
		"account_type": "savings",
	})

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "acc-001", body["account_id"])
	svc.AssertExpectations(t)
}

func TestCreateAccountHandler_ServiceError(t *testing.T) {
	svc := new(mockAccountService)
	app := newTestApp(svc)

	svc.On("CreateAccount", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	resp := doRequest(app, http.MethodPost, "/accounts", map[string]string{
		"user_id":      "user-1",
		"branch_id":    "branch-1",
		"account_type": "savings",
	})

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// ---- GetAccount handler tests ----

func TestGetAccountHandler_Success(t *testing.T) {
	svc := new(mockAccountService)
	app := newTestApp(svc)

	acc := sampleAccount()
	svc.On("GetAccountById", mock.Anything, &domain.GetAccountQuery{AccountId: "acc-001"}).Return(acc, nil)

	resp := doRequest(app, http.MethodGet, "/accounts/acc-001", nil)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "acc-001", body["account_id"])
}

func TestGetAccountHandler_ServiceError(t *testing.T) {
	svc := new(mockAccountService)
	app := newTestApp(svc)

	svc.On("GetAccountById", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))

	resp := doRequest(app, http.MethodGet, "/accounts/acc-999", nil)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
