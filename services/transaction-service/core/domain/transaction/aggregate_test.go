package transaction

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccountAggregate(t *testing.T) {
	agg := NewAccountAggregate("acc-1")
	assert.Equal(t, "acc-1", agg.AccountID)
	assert.True(t, agg.Balance.Equal(decimal.Zero))
	assert.Equal(t, int64(0), agg.SequenceNumber)
}

func TestAggregate_Rehydrate(t *testing.T) {
	t.Run("nil snapshot and no events keeps zero state", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		agg.Rehydrate(nil, nil)
		assert.True(t, agg.Balance.Equal(decimal.Zero))
		assert.Equal(t, int64(0), agg.SequenceNumber)
	})

	t.Run("snapshot only loads balance and sequence", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		snap := &AccountSnapshot{Balance: dec("500"), LastSequenceNumber: 5}
		agg.Rehydrate(snap, nil)
		assert.True(t, agg.Balance.Equal(dec("500")))
		assert.Equal(t, int64(5), agg.SequenceNumber)
	})

	t.Run("events only applied from zero", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		events := []TransactionEvent{
			{EventType: EventDeposited, Amount: dec("100"), SequenceNumber: 1},
			{EventType: EventWithdrawn, Amount: dec("30"), SequenceNumber: 2},
		}
		agg.Rehydrate(nil, events)
		assert.True(t, agg.Balance.Equal(dec("70")))
		assert.Equal(t, int64(2), agg.SequenceNumber)
	})

	t.Run("snapshot + events nets correctly", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		snap := &AccountSnapshot{Balance: dec("200"), LastSequenceNumber: 3}
		events := []TransactionEvent{
			{EventType: EventTransferIn, Amount: dec("50"), SequenceNumber: 4},
			{EventType: EventTransferOut, Amount: dec("80"), SequenceNumber: 5},
		}
		agg.Rehydrate(snap, events)
		assert.True(t, agg.Balance.Equal(dec("170")))
		assert.Equal(t, int64(5), agg.SequenceNumber)
	})
}

func TestAggregate_Deposit(t *testing.T) {
	t.Run("positive amount increases balance and sequence", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		agg.Balance = dec("100")
		agg.SequenceNumber = 3

		event, err := agg.Deposit(dec("50"), "key-1")
		require.NoError(t, err)
		assert.Equal(t, EventDeposited, event.EventType)
		assert.True(t, event.Amount.Equal(dec("50")))
		assert.Equal(t, int64(4), event.SequenceNumber)
		assert.True(t, agg.Balance.Equal(dec("150")))
		assert.Equal(t, int64(4), agg.SequenceNumber)
	})

	t.Run("zero amount returns error", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		_, err := agg.Deposit(decimal.Zero, "key-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		_, err := agg.Deposit(dec("-10"), "key-1")
		require.Error(t, err)
	})
}

func TestAggregate_Withdraw(t *testing.T) {
	t.Run("sufficient balance decrements correctly", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		agg.Balance = dec("200")
		agg.SequenceNumber = 1

		event, err := agg.Withdraw(dec("80"), "key-1")
		require.NoError(t, err)
		assert.Equal(t, EventWithdrawn, event.EventType)
		assert.True(t, event.Amount.Equal(dec("80")))
		assert.Equal(t, int64(2), event.SequenceNumber)
		assert.True(t, agg.Balance.Equal(dec("120")))
	})

	t.Run("exact balance withdrawal succeeds with zero remaining", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		agg.Balance = dec("100")

		_, err := agg.Withdraw(dec("100"), "key-1")
		require.NoError(t, err)
		assert.True(t, agg.Balance.Equal(decimal.Zero))
	})

	t.Run("overdraft returns insufficient balance error", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		agg.Balance = dec("50")

		_, err := agg.Withdraw(dec("100"), "key-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient balance")
		// Balance must not have changed
		assert.True(t, agg.Balance.Equal(dec("50")))
	})

	t.Run("zero amount returns error", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		_, err := agg.Withdraw(decimal.Zero, "key-1")
		require.Error(t, err)
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		agg := NewAccountAggregate("acc-1")
		_, err := agg.Withdraw(dec("-5"), "key-1")
		require.Error(t, err)
	})
}

func TestAggregate_TransferOut(t *testing.T) {
	t.Run("sufficient balance sets ReferenceID and out key suffix", func(t *testing.T) {
		agg := NewAccountAggregate("src")
		agg.Balance = dec("300")

		event, err := agg.TransferOut(dec("100"), "dst", "idem-1")
		require.NoError(t, err)
		assert.Equal(t, EventTransferOut, event.EventType)
		assert.Equal(t, "dst", event.ReferenceID)
		assert.Equal(t, "idem-1:out", event.IdempotencyKey)
		assert.True(t, agg.Balance.Equal(dec("200")))
	})

	t.Run("insufficient balance returns error without mutating", func(t *testing.T) {
		agg := NewAccountAggregate("src")
		agg.Balance = dec("50")

		_, err := agg.TransferOut(dec("100"), "dst", "idem-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient balance")
		assert.True(t, agg.Balance.Equal(dec("50")))
	})

	t.Run("zero amount returns error", func(t *testing.T) {
		agg := NewAccountAggregate("src")
		agg.Balance = dec("100")
		_, err := agg.TransferOut(decimal.Zero, "dst", "idem-1")
		require.Error(t, err)
	})
}

func TestAggregate_TransferIn(t *testing.T) {
	t.Run("sets ReferenceID and in key suffix", func(t *testing.T) {
		agg := NewAccountAggregate("dst")
		agg.Balance = dec("100")

		event, err := agg.TransferIn(dec("50"), "src", "idem-1")
		require.NoError(t, err)
		assert.Equal(t, EventTransferIn, event.EventType)
		assert.Equal(t, "src", event.ReferenceID)
		assert.Equal(t, "idem-1:in", event.IdempotencyKey)
		assert.True(t, agg.Balance.Equal(dec("150")))
	})

	t.Run("zero amount returns error", func(t *testing.T) {
		agg := NewAccountAggregate("dst")
		_, err := agg.TransferIn(decimal.Zero, "src", "idem-1")
		require.Error(t, err)
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		agg := NewAccountAggregate("dst")
		_, err := agg.TransferIn(dec("-1"), "src", "idem-1")
		require.Error(t, err)
	})
}

func TestAggregate_ShouldSnapshot(t *testing.T) {
	tests := []struct {
		name            string
		currentSeq      int64
		lastSnapshotSeq int64
		want            bool
	}{
		{"diff less than interval", 999, 0, false},
		{"diff exactly interval", 1000, 0, true},
		{"diff greater than interval", 1500, 0, true},
		{"same sequence as snapshot", 5, 5, false},
		{"small diff", 10, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := NewAccountAggregate("acc-1")
			agg.SequenceNumber = tt.currentSeq
			assert.Equal(t, tt.want, agg.ShouldSnapshot(tt.lastSnapshotSeq))
		})
	}
}

// dec is a helper to create decimal.Decimal from a string.
func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}
