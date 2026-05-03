package payment

import (
	"context"
	"time"

	"github.com/sacramento-finance/backend/internal/domain/ledger"
)

type OverduePaymentRepository interface {
	ListOverdue(ctx context.Context, before time.Time) ([]*ledger.Payment, error)
	Update(ctx context.Context, p *ledger.Payment) error
}

type MarkOverdueUseCase struct {
	payments OverduePaymentRepository
}

func NewMarkOverdueUseCase(payments OverduePaymentRepository) *MarkOverdueUseCase {
	return &MarkOverdueUseCase{payments: payments}
}

// Execute marks all pending payments whose due_date is before `now` as missed.
// Returns the number of payments updated.
func (uc *MarkOverdueUseCase) Execute(ctx context.Context, now time.Time) (int, error) {
	overdue, err := uc.payments.ListOverdue(ctx, now)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, p := range overdue {
		p.Status = ledger.PaymentStatusMissed
		if err := uc.payments.Update(ctx, p); err != nil {
			continue // best-effort: log and keep going
		}
		count++
	}
	return count, nil
}
