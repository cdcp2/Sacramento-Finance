package payment

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

type WritePaymentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ledger.Payment, error)
	Update(ctx context.Context, p *ledger.Payment) error
}

type WaivePaymentUseCase struct {
	payments WritePaymentRepository
}

func NewWaivePaymentUseCase(payments WritePaymentRepository) *WaivePaymentUseCase {
	return &WaivePaymentUseCase{payments: payments}
}

// Execute marks a payment as waived. Only callable by a fund admin.
// Authorization (admin check) must be done in the handler before calling this.
func (uc *WaivePaymentUseCase) Execute(ctx context.Context, paymentID uuid.UUID, fundID uuid.UUID) (*ledger.Payment, error) {
	p, err := uc.payments.GetByID(ctx, paymentID)
	if err != nil || p == nil {
		return nil, apperror.ErrPaymentNotFound
	}

	// Ensure payment belongs to the correct fund
	if p.FundID != fundID {
		return nil, apperror.ErrPaymentNotFound
	}

	switch p.Status {
	case ledger.PaymentStatusPaid:
		return nil, apperror.ErrPaymentAlreadyPaid
	case ledger.PaymentStatusWaived:
		return nil, apperror.New(409, "ALREADY_WAIVED", "payment is already waived")
	}

	now := time.Now().UTC()
	p.Status = ledger.PaymentStatusWaived
	p.UpdatedAt = now

	if err := uc.payments.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
