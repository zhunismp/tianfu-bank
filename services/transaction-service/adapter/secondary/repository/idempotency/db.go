package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
	"gorm.io/gorm"
)

type idempotencyRepository struct {
	*gorm.DB
}

func NewIdempotencyRepository(db *gorm.DB) domain.IdempotencyRepository {
	return &idempotencyRepository{DB: db}
}

func (r *idempotencyRepository) Get(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	var model IdempotencyKeyModel
	err := r.DB.WithContext(ctx).Where("idempotency_key = ? AND expires_at > ?", key, time.Now()).First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get idempotency key: %w", err)
	}
	return model.ToEntity(), nil
}

func (r *idempotencyRepository) Save(ctx context.Context, record *domain.IdempotencyRecord) error {
	bodyJSON, err := json.Marshal(record.ResponseBody)
	if err != nil {
		return fmt.Errorf("marshal response body: %w", err)
	}

	model := &IdempotencyKeyModel{
		IdempotencyKey: record.IdempotencyKey,
		StatusCode:     record.StatusCode,
		ResponseBody:   bodyJSON,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	if err := r.DB.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("save idempotency key: %w", err)
	}
	return nil
}
