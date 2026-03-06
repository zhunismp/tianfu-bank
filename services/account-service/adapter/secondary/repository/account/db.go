package account

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	domain "github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
	"gorm.io/gorm"
)

type accountRepository struct {
	*gorm.DB
}

func NewAccountRepository(db *gorm.DB) *accountRepository {
	return &accountRepository{DB: db}
}

func (r *accountRepository) CreateAccount(ctx context.Context, userId, branchId, accountType string) (*domain.Account, error) {
	var am AccountModel

	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw(`SELECT nextval('account_seq')`).Scan(&seq).Error; err != nil {
			return fmt.Errorf("fetch sequence: %w", err)
		}

		am = AccountModel{
			AccountId:   fmt.Sprintf("%s%07d", branchId[:3], seq),
			UserId:      userId,
			AccountType: accountType,
			BranchId:    branchId,
			Balance:     decimal.Zero,
		}
		return tx.Create(&am).Error
	})

	if err != nil {
		return nil, fmt.Errorf("CreateAccount: %w", err)
	}

	return am.ToEntity(), nil
}

func (r *accountRepository) GetAccountById(ctx context.Context, accountId string) (*domain.Account, error) {
	var am AccountModel
	if err := r.DB.WithContext(ctx).Where("account_id = ?", accountId).First(&am).Error; err != nil {
		return nil, err
	}
	return am.ToEntity(), nil
}

func (r *accountRepository) UpdateBalance(ctx context.Context, accountId string, balance decimal.Decimal) error {
	if err := r.DB.WithContext(ctx).Model(&AccountModel{}).Where("account_id = ?", accountId).Update("balance", balance).Error; err != nil {
		return err
	}
	return nil
}
