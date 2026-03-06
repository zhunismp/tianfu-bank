package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/zhunismp/tianfu-bank/shared/apperror"
)

type transactionService struct {
	txProvider TransactionProvider
	publisher  EventPublisher
}

func NewTransactionService(
	txProvider TransactionProvider,
	publisher EventPublisher,
) TransactionService {
	return &transactionService{
		txProvider: txProvider,
		publisher:  publisher,
	}
}

// rehydrate loads the aggregate from snapshot + events inside the active transaction.
// Returns an error if no snapshot and no events exist for the account.
func (s *transactionService) rehydrate(ctx context.Context, tx TxAdapters, accountID string) (*AccountAggregate, int64, error) {
	agg := NewAccountAggregate(accountID)

	snapshot, err := tx.SnapshotRepo.GetLatestSnapshot(ctx, accountID)
	if err != nil {
		return nil, 0, fmt.Errorf("get snapshot: %w", err)
	}

	var sinceSeq int64
	if snapshot != nil {
		sinceSeq = snapshot.LastSequenceNumber
	}

	events, err := tx.EventStore.GetEventsSince(ctx, accountID, sinceSeq)
	if err != nil {
		return nil, 0, fmt.Errorf("get events: %w", err)
	}

	if snapshot == nil && len(events) == 0 {
		return nil, 0, apperror.New(apperror.ErrCodeAccountNotFound, fmt.Sprintf("account not found: %s", accountID), nil)
	}

	agg.Rehydrate(snapshot, events)

	return agg, sinceSeq, nil
}

// maybeSnapshot creates a snapshot if enough events have accumulated.
func (s *transactionService) maybeSnapshot(ctx context.Context, repo SnapshotRepository, agg *AccountAggregate, lastSnapshotSeq int64) {
	if agg.ShouldSnapshot(lastSnapshotSeq) {
		snap := &AccountSnapshot{
			AccountID:          agg.AccountID,
			Balance:            agg.Balance,
			LastSequenceNumber: agg.SequenceNumber,
		}
		if err := repo.CreateSnapshot(ctx, snap); err != nil {
			slog.Error("Failed to create snapshot", "accountId", agg.AccountID, "error", err)
		} else {
			slog.Info("Snapshot created", "accountId", agg.AccountID, "sequence", agg.SequenceNumber)
		}
	}
}

func (s *transactionService) Deposit(ctx context.Context, cmd *DepositCmd) (*TransactionEvent, error) {
	if cmd == nil {
		return nil, fmt.Errorf("deposit command is nil")
	}

	var event *TransactionEvent
	var agg *AccountAggregate

	err := s.txProvider.Transact(func(tx TxAdapters) error {
		// Check idempotency
		existing, err := tx.Idempotency.Get(ctx, cmd.IdempotencyKey)
		if err != nil {
			return fmt.Errorf("idempotency check: %w", err)
		}
		if existing != nil {
			slog.Info("Idempotent request detected (deposit)", "key", cmd.IdempotencyKey)
			event, err = s.reconstructEventFromIdempotency(existing)
			return err
		}

		// Rehydrate aggregate
		var lastSnapSeq int64
		agg, lastSnapSeq, err = s.rehydrate(ctx, tx, cmd.AccountID)
		if err != nil {
			return fmt.Errorf("rehydrate: %w", err)
		}

		// Execute business logic
		event, err = agg.Deposit(cmd.Amount, cmd.IdempotencyKey)
		if err != nil {
			return err
		}

		// Persist event
		if err := tx.EventStore.AppendEvent(ctx, event); err != nil {
			return fmt.Errorf("append event: %w", err)
		}

		s.maybeSnapshot(ctx, tx.SnapshotRepo, agg, lastSnapSeq)
		s.saveIdempotencyRecord(ctx, tx.Idempotency, cmd.IdempotencyKey, event)

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Publish only for fresh transactions (agg is nil on idempotent replay)
	if agg != nil {
		if err := s.publisher.PublishBalanceUpdated(ctx, agg.AccountID, agg.Balance, event.EventType, event.ID); err != nil {
			slog.Error("Failed to publish balance update", "accountId", agg.AccountID, "error", err)
		}
	}

	return event, nil
}

func (s *transactionService) Withdraw(ctx context.Context, cmd *WithdrawCmd) (*TransactionEvent, error) {
	if cmd == nil {
		return nil, fmt.Errorf("withdraw command is nil")
	}

	var event *TransactionEvent
	var agg *AccountAggregate

	err := s.txProvider.Transact(func(tx TxAdapters) error {
		// Check idempotency
		existing, err := tx.Idempotency.Get(ctx, cmd.IdempotencyKey)
		if err != nil {
			return fmt.Errorf("idempotency check: %w", err)
		}
		if existing != nil {
			slog.Info("Idempotent request detected (withdraw)", "key", cmd.IdempotencyKey)
			event, err = s.reconstructEventFromIdempotency(existing)
			return err
		}

		// Rehydrate aggregate
		var lastSnapSeq int64
		agg, lastSnapSeq, err = s.rehydrate(ctx, tx, cmd.AccountID)
		if err != nil {
			return fmt.Errorf("rehydrate: %w", err)
		}

		// Execute business logic
		event, err = agg.Withdraw(cmd.Amount, cmd.IdempotencyKey)
		if err != nil {
			return err
		}

		// Persist event
		if err := tx.EventStore.AppendEvent(ctx, event); err != nil {
			return fmt.Errorf("append event: %w", err)
		}

		s.maybeSnapshot(ctx, tx.SnapshotRepo, agg, lastSnapSeq)
		s.saveIdempotencyRecord(ctx, tx.Idempotency, cmd.IdempotencyKey, event)

		return nil
	})
	if err != nil {
		return nil, err
	}

	if agg != nil {
		if err := s.publisher.PublishBalanceUpdated(ctx, agg.AccountID, agg.Balance, event.EventType, event.ID); err != nil {
			slog.Error("Failed to publish balance update", "accountId", agg.AccountID, "error", err)
		}
	}

	return event, nil
}

func (s *transactionService) Transfer(ctx context.Context, cmd *TransferCmd) ([]*TransactionEvent, error) {
	if cmd == nil {
		return nil, fmt.Errorf("transfer command is nil")
	}

	var outEvent, inEvent *TransactionEvent
	var sourceAgg, destAgg *AccountAggregate

	err := s.txProvider.Transact(func(tx TxAdapters) error {
		// Check idempotency
		existing, err := tx.Idempotency.Get(ctx, cmd.IdempotencyKey)
		if err != nil {
			return fmt.Errorf("idempotency check: %w", err)
		}
		if existing != nil {
			slog.Info("Idempotent request detected (transfer)", "key", cmd.IdempotencyKey)
			return nil // signal already processed; sourceAgg/destAgg stay nil
		}

		// Rehydrate both aggregates
		var sourceSnapSeq, destSnapSeq int64
		sourceAgg, sourceSnapSeq, err = s.rehydrate(ctx, tx, cmd.SourceAccountID)
		if err != nil {
			return fmt.Errorf("rehydrate source: %w", err)
		}

		destAgg, destSnapSeq, err = s.rehydrate(ctx, tx, cmd.DestinationAccountID)
		if err != nil {
			return fmt.Errorf("rehydrate destination: %w", err)
		}

		// Execute business logic
		outEvent, err = sourceAgg.TransferOut(cmd.Amount, cmd.DestinationAccountID, cmd.IdempotencyKey)
		if err != nil {
			return err
		}
		inEvent, err = destAgg.TransferIn(cmd.Amount, cmd.SourceAccountID, cmd.IdempotencyKey)
		if err != nil {
			return err
		}

		// Persist both events
		if err := tx.EventStore.AppendEvent(ctx, outEvent); err != nil {
			return fmt.Errorf("append transfer out event: %w", err)
		}
		if err := tx.EventStore.AppendEvent(ctx, inEvent); err != nil {
			return fmt.Errorf("append transfer in event: %w", err)
		}

		s.maybeSnapshot(ctx, tx.SnapshotRepo, sourceAgg, sourceSnapSeq)
		s.maybeSnapshot(ctx, tx.SnapshotRepo, destAgg, destSnapSeq)
		s.saveIdempotencyRecord(ctx, tx.Idempotency, cmd.IdempotencyKey, outEvent)

		return nil
	})
	if err != nil {
		return nil, err
	}

	if sourceAgg == nil {
		return nil, nil // idempotent replay
	}

	if err := s.publisher.PublishBalanceUpdated(ctx, sourceAgg.AccountID, sourceAgg.Balance, outEvent.EventType, outEvent.ID); err != nil {
		slog.Error("Failed to publish source balance update", "accountId", sourceAgg.AccountID, "error", err)
	}
	if err := s.publisher.PublishBalanceUpdated(ctx, destAgg.AccountID, destAgg.Balance, inEvent.EventType, inEvent.ID); err != nil {
		slog.Error("Failed to publish dest balance update", "accountId", destAgg.AccountID, "error", err)
	}

	return []*TransactionEvent{outEvent, inEvent}, nil
}

func (s *transactionService) GetHistory(ctx context.Context, query *GetHistoryQuery) ([]TransactionHistoryItem, error) {
	if query == nil {
		return nil, fmt.Errorf("get history query is nil")
	}

	var items []TransactionHistoryItem

	err := s.txProvider.Transact(func(tx TxAdapters) error {
		events, err := tx.EventStore.GetEventsSince(ctx, query.AccountID, 0)
		if err != nil {
			return fmt.Errorf("get events: %w", err)
		}

		items = make([]TransactionHistoryItem, 0, len(events))
		runningBalance := decimal.Zero

		for _, e := range events {
			switch e.EventType {
			case EventDeposited, EventTransferIn:
				runningBalance = runningBalance.Add(e.Amount)
			case EventWithdrawn, EventTransferOut:
				runningBalance = runningBalance.Sub(e.Amount)
			}

			items = append(items, TransactionHistoryItem{
				ID:           e.ID,
				EventType:    e.EventType,
				Amount:       e.Amount,
				BalanceAfter: runningBalance,
				ReferenceID:  e.ReferenceID,
				CreatedAt:    e.CreatedAt,
			})
		}

		if query.Offset > 0 && query.Offset < len(items) {
			items = items[query.Offset:]
		}
		if query.Limit > 0 && query.Limit < len(items) {
			items = items[:query.Limit]
		}

		return nil
	})

	return items, err
}

func (s *transactionService) saveIdempotencyRecord(ctx context.Context, repo IdempotencyRepository, key string, event *TransactionEvent) {
	if key == "" {
		return
	}

	body, _ := json.Marshal(event)
	var bodyMap map[string]any
	json.Unmarshal(body, &bodyMap)

	record := &IdempotencyRecord{
		IdempotencyKey: key,
		StatusCode:     200,
		ResponseBody:   bodyMap,
	}
	if err := repo.Save(ctx, record); err != nil {
		slog.Error("Failed to save idempotency record", "key", key, "error", err)
	}
}

func (s *transactionService) reconstructEventFromIdempotency(record *IdempotencyRecord) (*TransactionEvent, error) {
	data, err := json.Marshal(record.ResponseBody)
	if err != nil {
		return nil, fmt.Errorf("marshal idempotency body: %w", err)
	}
	var event TransactionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("unmarshal idempotency body: %w", err)
	}
	return &event, nil
}
