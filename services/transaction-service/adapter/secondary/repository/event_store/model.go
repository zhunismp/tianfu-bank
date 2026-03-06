package event_store

import (
	"time"

	"github.com/shopspring/decimal"
	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
)

type TransactionEventModel struct {
	ID              string          `gorm:"primaryKey;column:id;type:uuid;default:gen_random_uuid()"`
	AccountID       string          `gorm:"column:account_id;size:36;not null;index:idx_txe_account_seq"`
	EventType       string          `gorm:"column:event_type;size:30;not null"`
	Amount          decimal.Decimal `gorm:"column:amount;type:numeric(20,2);not null;default:0"`
	ReferenceID     *string         `gorm:"column:reference_id;size:36"`
	IdempotencyKey  *string         `gorm:"column:idempotency_key;size:100;uniqueIndex:idx_txe_idemp"`
	SequenceNumber  int64           `gorm:"column:sequence_number;not null"`
	Metadata        *string         `gorm:"column:metadata;type:jsonb"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime"`
}

func (m *TransactionEventModel) TableName() string {
	return "transaction_events"
}

func (m *TransactionEventModel) ToEntity() *domain.TransactionEvent {
	e := &domain.TransactionEvent{
		ID:             m.ID,
		AccountID:      m.AccountID,
		EventType:      m.EventType,
		Amount:         m.Amount,
		SequenceNumber: m.SequenceNumber,
		CreatedAt:      m.CreatedAt,
	}
	if m.ReferenceID != nil {
		e.ReferenceID = *m.ReferenceID
	}
	if m.IdempotencyKey != nil {
		e.IdempotencyKey = *m.IdempotencyKey
	}
	return e
}

func FromEventEntity(e *domain.TransactionEvent) *TransactionEventModel {
	m := &TransactionEventModel{
		AccountID:      e.AccountID,
		EventType:      e.EventType,
		Amount:         e.Amount,
		SequenceNumber: e.SequenceNumber,
	}
	if e.ReferenceID != "" {
		m.ReferenceID = &e.ReferenceID
	}
	if e.IdempotencyKey != "" {
		m.IdempotencyKey = &e.IdempotencyKey
	}
	return m
}
