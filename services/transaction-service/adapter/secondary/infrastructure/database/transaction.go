package database

import (
	eventStoreRepo "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/secondary/repository/event_store"
	idempotencyRepo "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/secondary/repository/idempotency"
	snapshotRepo "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/secondary/repository/snapshot"
	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
	"gorm.io/gorm"
)

type transactionProvider struct {
	db *gorm.DB
}

func NewTransactionProvider(db *gorm.DB) domain.TransactionProvider {
	return &transactionProvider{db: db}
}

func (p *transactionProvider) Transact(fn func(tx domain.TxAdapters) error) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		return fn(domain.TxAdapters{
			EventStore:   eventStoreRepo.NewEventStoreRepository(tx),
			SnapshotRepo: snapshotRepo.NewSnapshotRepository(tx),
			Idempotency:  idempotencyRepo.NewIdempotencyRepository(tx),
		})
	})
}
