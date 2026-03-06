package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---- inline mocks ----

type mockAccountRepo struct{ mock.Mock }

func (m *mockAccountRepo) CreateAccount(ctx context.Context, userId, branchId, accountType string) (*Account, error) {
	args := m.Called(ctx, userId, branchId, accountType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Account), args.Error(1)
}

func (m *mockAccountRepo) GetAccountById(ctx context.Context, accountId string) (*Account, error) {
	args := m.Called(ctx, accountId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Account), args.Error(1)
}

func (m *mockAccountRepo) UpdateBalance(ctx context.Context, accountId string, balance decimal.Decimal) error {
	args := m.Called(ctx, accountId, balance)
	return args.Error(0)
}

type mockEventPublisher struct{ mock.Mock }

func (m *mockEventPublisher) PublishAccountCreated(ctx context.Context, account *Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

// ---- helpers ----

func testAccount() *Account {
	return &Account{
		AccountId:   "acc-001",
		UserId:      "user-1",
		BranchId:    "branch-1",
		AccountType: "savings",
		Balance:     decimal.Zero,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ---- CreateAccount tests ----

func TestAccountService_CreateAccount_NilCmd(t *testing.T) {
	svc := NewAccountService(new(mockAccountRepo), new(mockEventPublisher))
	acc, err := svc.CreateAccount(context.Background(), nil)
	assert.Nil(t, acc)
	assert.Nil(t, err)
}

func TestAccountService_CreateAccount_CreatesAccountAndPublishesEvent(t *testing.T) {
	repo := new(mockAccountRepo)
	pub := new(mockEventPublisher)
	svc := NewAccountService(repo, pub)

	created := testAccount()
	repo.On("CreateAccount", mock.Anything, "user-1", "branch-1", "savings").Return(created, nil)
	pub.On("PublishAccountCreated", mock.Anything, created).Return(nil)

	acc, err := svc.CreateAccount(context.Background(), &CreateAccountCmd{
		UserId: "user-1", BranchId: "branch-1", AccountType: "savings",
	})

	require.NoError(t, err)
	assert.Equal(t, created, acc)
	repo.AssertExpectations(t)
	pub.AssertExpectations(t)
}

func TestAccountService_CreateAccount_RepoError(t *testing.T) {
	repo := new(mockAccountRepo)
	pub := new(mockEventPublisher)
	svc := NewAccountService(repo, pub)

	repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	acc, err := svc.CreateAccount(context.Background(), &CreateAccountCmd{
		UserId: "user-1", BranchId: "branch-1", AccountType: "savings",
	})

	require.Error(t, err)
	assert.Nil(t, acc)
	// publisher must not be called when repo fails
	pub.AssertNotCalled(t, "PublishAccountCreated")
}

func TestAccountService_CreateAccount_PublishErrorSwallowed(t *testing.T) {
	repo := new(mockAccountRepo)
	pub := new(mockEventPublisher)
	svc := NewAccountService(repo, pub)

	created := testAccount()
	repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(created, nil)
	pub.On("PublishAccountCreated", mock.Anything, created).Return(errors.New("broker down"))

	acc, err := svc.CreateAccount(context.Background(), &CreateAccountCmd{
		UserId: "user-1", BranchId: "branch-1", AccountType: "savings",
	})

	// publish failure must NOT propagate — fire-and-forget
	require.NoError(t, err)
	assert.Equal(t, created, acc)
}

// ---- GetAccountById tests ----

func TestAccountService_GetAccountById_NilQuery(t *testing.T) {
	svc := NewAccountService(new(mockAccountRepo), new(mockEventPublisher))
	acc, err := svc.GetAccountById(context.Background(), nil)
	assert.Nil(t, acc)
	assert.Nil(t, err)
}

func TestAccountService_GetAccountById_ReturnsMatchingAccount(t *testing.T) {
	repo := new(mockAccountRepo)
	pub := new(mockEventPublisher)
	svc := NewAccountService(repo, pub)

	expected := testAccount()
	repo.On("GetAccountById", mock.Anything, "acc-001").Return(expected, nil)

	acc, err := svc.GetAccountById(context.Background(), &GetAccountQuery{AccountId: "acc-001"})
	require.NoError(t, err)
	assert.Equal(t, expected, acc)
}

func TestAccountService_GetAccountById_RepoError(t *testing.T) {
	repo := new(mockAccountRepo)
	pub := new(mockEventPublisher)
	svc := NewAccountService(repo, pub)

	repo.On("GetAccountById", mock.Anything, "acc-999").Return(nil, errors.New("not found"))

	acc, err := svc.GetAccountById(context.Background(), &GetAccountQuery{AccountId: "acc-999"})
	require.Error(t, err)
	assert.Nil(t, acc)
}
