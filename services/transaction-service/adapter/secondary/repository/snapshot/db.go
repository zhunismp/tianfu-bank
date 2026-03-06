package snapshot

import (
	"context"
	"fmt"

	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
	"gorm.io/gorm"
)

type snapshotRepository struct {
	*gorm.DB
}

func NewSnapshotRepository(db *gorm.DB) domain.SnapshotRepository {
	return &snapshotRepository{DB: db}
}

func (r *snapshotRepository) GetLatestSnapshot(ctx context.Context, accountID string) (*domain.AccountSnapshot, error) {
	var model AccountSnapshotModel
	err := r.DB.WithContext(ctx).
		Where("account_id = ?", accountID).
		Order("last_sequence_number DESC").
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	return model.ToEntity(), nil
}

func (r *snapshotRepository) CreateSnapshot(ctx context.Context, snapshot *domain.AccountSnapshot) error {
	model := FromSnapshotEntity(snapshot)
	if err := r.DB.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	return nil
}
