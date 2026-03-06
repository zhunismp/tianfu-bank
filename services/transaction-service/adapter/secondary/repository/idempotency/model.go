package idempotency

import (
	"encoding/json"
	"time"

	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
)

type IdempotencyKeyModel struct {
	IdempotencyKey string    `gorm:"primaryKey;column:idempotency_key;size:100"`
	StatusCode     int       `gorm:"column:status_code;not null"`
	ResponseBody   []byte    `gorm:"column:response_body;type:jsonb"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	ExpiresAt      time.Time `gorm:"column:expires_at"`
}

func (m *IdempotencyKeyModel) TableName() string {
	return "idempotency_keys"
}

func (m *IdempotencyKeyModel) ToEntity() *domain.IdempotencyRecord {
	var bodyMap map[string]any
	if m.ResponseBody != nil {
		_ = json.Unmarshal(m.ResponseBody, &bodyMap)
	}
	return &domain.IdempotencyRecord{
		IdempotencyKey: m.IdempotencyKey,
		StatusCode:     m.StatusCode,
		ResponseBody:   bodyMap,
		CreatedAt:      m.CreatedAt,
		ExpiresAt:      m.ExpiresAt,
	}
}
