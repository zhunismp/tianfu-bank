package event_store

import (
	"context"
	"fmt"

	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
	"gorm.io/gorm"
)

type eventStoreRepository struct {
	*gorm.DB
}

func NewEventStoreRepository(db *gorm.DB) domain.EventStoreRepository {
	return &eventStoreRepository{DB: db}
}

func (r *eventStoreRepository) AppendEvent(ctx context.Context, event *domain.TransactionEvent) error {
	model := FromEventEntity(event)
	if err := r.DB.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("append event: %w", err)
	}
	// Set the generated ID back on the entity
	event.ID = model.ID
	event.CreatedAt = model.CreatedAt
	return nil
}

func (r *eventStoreRepository) GetEventsSince(ctx context.Context, accountID string, sequenceNumber int64) ([]domain.TransactionEvent, error) {
	var models []TransactionEventModel
	err := r.DB.WithContext(ctx).
		Where("account_id = ? AND sequence_number > ?", accountID, sequenceNumber).
		Order("sequence_number ASC").
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("get events since: %w", err)
	}

	events := make([]domain.TransactionEvent, len(models))
	for i, m := range models {
		e := m.ToEntity()
		events[i] = *e
	}
	return events, nil
}
