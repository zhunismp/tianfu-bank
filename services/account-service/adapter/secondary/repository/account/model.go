package account

import (
	"time"

	"github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
)

// Represents the account schema in database
type AccountModel struct {
	AccountId   string    `gorm:"primaryKey;column:account_id;size:36;not null"`
	UserId      string    `gorm:"column:user_id;index:uid_user_id;size:36;not null"`
	AccountType string    `gorm:"column:account_type;size:20;not null"`
	BranchId    string    `gorm:"column:branch_id;index:uid_branch_id;size:36;not null"`
	Balance     float64   `gorm:"column:balance;not null;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (a *AccountModel) TableName() string {
	return "account"
}

func (a *AccountModel) ToEntity() *account.Account {
	return &account.Account{
		AccountId:   a.AccountId,
		UserId:      a.UserId,
		BranchId:    a.BranchId,
		AccountType: a.AccountType,
		Balance:     a.Balance,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
