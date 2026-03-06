package snapshot

import (
	"time"

	"github.com/shopspring/decimal"
	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
)

type AccountSnapshotModel struct {
	ID                 string          `gorm:"primaryKey;column:id;type:uuid;default:gen_random_uuid()"`
	AccountID          string          `gorm:"column:account_id;size:36;not null;index:idx_snap_account"`
	Balance            decimal.Decimal `gorm:"column:balance;type:numeric(20,2);not null;default:0"`
	LastSequenceNumber int64           `gorm:"column:last_sequence_number;not null"`
	CreatedAt          time.Time       `gorm:"column:created_at;autoCreateTime"`
}

func (m *AccountSnapshotModel) TableName() string {
	return "account_snapshots"
}

func (m *AccountSnapshotModel) ToEntity() *domain.AccountSnapshot {
	return &domain.AccountSnapshot{
		ID:                 m.ID,
		AccountID:          m.AccountID,
		Balance:            m.Balance,
		LastSequenceNumber: m.LastSequenceNumber,
		CreatedAt:          m.CreatedAt,
	}
}

func FromSnapshotEntity(s *domain.AccountSnapshot) *AccountSnapshotModel {
	return &AccountSnapshotModel{
		AccountID:          s.AccountID,
		Balance:            s.Balance,
		LastSequenceNumber: s.LastSequenceNumber,
	}
}
