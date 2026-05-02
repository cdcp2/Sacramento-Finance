package ledger_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/shopspring/decimal"
)

func TestPayment_IsOverdue(t *testing.T) {
	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	tests := []struct {
		name     string
		dueDate  time.Time
		status   ledger.PaymentStatus
		overdue  bool
	}{
		{"past due + pending → overdue", yesterday, ledger.PaymentStatusPending, true},
		{"future due + pending → not overdue", tomorrow, ledger.PaymentStatusPending, false},
		{"past due + paid → not overdue", yesterday, ledger.PaymentStatusPaid, false},
		{"past due + waived → not overdue", yesterday, ledger.PaymentStatusWaived, false},
		{"past due + missed → not overdue", yesterday, ledger.PaymentStatusMissed, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &ledger.Payment{
				ID:      uuid.New(),
				DueDate: tc.dueDate,
				Status:  tc.status,
				AmountDue: decimal.NewFromInt(100000),
			}
			got := p.IsOverdue(now)
			if got != tc.overdue {
				t.Errorf("IsOverdue() = %v, want %v", got, tc.overdue)
			}
		})
	}
}
