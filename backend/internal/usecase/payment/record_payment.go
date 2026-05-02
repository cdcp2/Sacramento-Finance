package payment

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
	"github.com/sacramento-finance/backend/pkg/money"
)

type LedgerRepository interface {
	RecordPaymentTx(ctx context.Context, p *ledger.Payment, entry *ledger.LedgerEntry) error
}

type ReadPaymentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ledger.Payment, error)
}

type RecordPaymentUseCase struct {
	payments ReadPaymentRepository
	ledger   LedgerRepository
}

func NewRecordPaymentUseCase(payments ReadPaymentRepository, ledger LedgerRepository) *RecordPaymentUseCase {
	return &RecordPaymentUseCase{payments: payments, ledger: ledger}
}

type RecordPaymentResult struct {
	Payment *ledger.Payment
	Entry   *ledger.LedgerEntry
}

// Execute records a payment for a fund member.
// The payment must belong to the requesting member.
// If overdue past grace period and penalty is enabled, it is automatically applied.
func (uc *RecordPaymentUseCase) Execute(
	ctx context.Context,
	paymentID uuid.UUID,
	memberID uuid.UUID, // the fund_member.id of the requester
	userID uuid.UUID,   // the user.id (for ledger)
	f *fund.Fund,
) (*RecordPaymentResult, error) {
	p, err := uc.payments.GetByID(ctx, paymentID)
	if err != nil || p == nil {
		return nil, apperror.ErrPaymentNotFound
	}

	// Verify this payment belongs to the requesting member
	if p.MemberID != memberID {
		return nil, apperror.ErrForbidden
	}

	switch p.Status {
	case ledger.PaymentStatusPaid:
		return nil, apperror.ErrPaymentAlreadyPaid
	case ledger.PaymentStatusWaived:
		return nil, apperror.New(400, "PAYMENT_WAIVED", "this payment has been waived")
	case ledger.PaymentStatusMissed:
		return nil, apperror.New(400, "PAYMENT_MISSED", "this payment was marked as missed; contact your fund admin")
	}

	now := time.Now().UTC()
	paidAt := now

	// Calculate total amount including penalty if applicable
	totalAmount := p.AmountDue
	penaltyAmount := money.Zero
	penaltyApplied := false

	if f.Rules.PenaltyEnabled && p.IsOverdue(now) {
		graceDeadline := p.DueDate.AddDate(0, 0, f.Rules.GracePeriodDays)
		if now.After(graceDeadline) {
			switch f.Rules.PenaltyType {
			case fund.PenaltyTypeFixed:
				penaltyAmount = f.Rules.PenaltyAmount
			case fund.PenaltyTypePercentage:
				penaltyAmount = money.Percentage(p.AmountDue, f.Rules.PenaltyAmount)
			}
			totalAmount = totalAmount.Add(penaltyAmount)
			penaltyApplied = true
		}
	}

	p.AmountPaid = totalAmount
	p.PaidAt = &paidAt
	p.Status = ledger.PaymentStatusPaid
	p.PenaltyApplied = penaltyApplied
	p.PenaltyAmount = penaltyAmount

	refID := p.ID
	entry := &ledger.LedgerEntry{
		ID:          idgen.New(),
		FundID:      p.FundID,
		UserID:      &userID,
		Type:        ledger.EntryTypeDeposit,
		Direction:   ledger.DirectionCredit,
		Amount:      totalAmount,
		ReferenceID: &refID,
		Description: buildDescription(p, penaltyApplied, penaltyAmount),
		CreatedAt:   now,
	}

	if err := uc.ledger.RecordPaymentTx(ctx, p, entry); err != nil {
		return nil, err
	}

	return &RecordPaymentResult{Payment: p, Entry: entry}, nil
}

func buildDescription(p *ledger.Payment, penaltyApplied bool, penaltyAmount interface{ String() string }) string {
	base := "Payment for period " + itoa(p.PeriodNumber)
	if penaltyApplied {
		return base + " (includes penalty of " + penaltyAmount.String() + ")"
	}
	return base
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
