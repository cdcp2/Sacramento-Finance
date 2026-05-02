package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/shopspring/decimal"
)

type mockWritePaymentRepo struct {
	payment *ledger.Payment
	updated *ledger.Payment
	getErr  error
	updErr  error
}

func (m *mockWritePaymentRepo) GetByID(_ context.Context, _ uuid.UUID) (*ledger.Payment, error) {
	return m.payment, m.getErr
}

func (m *mockWritePaymentRepo) Update(_ context.Context, p *ledger.Payment) error {
	if m.updErr != nil {
		return m.updErr
	}
	m.updated = p
	return nil
}

func waiveablePayment(fundID uuid.UUID) *ledger.Payment {
	return &ledger.Payment{
		ID:        uuid.New(),
		FundID:    fundID,
		MemberID:  uuid.New(),
		AmountDue: decimal.NewFromInt(100000),
		Status:    ledger.PaymentStatusPending,
	}
}

func TestWaivePayment_Success_SetsWaivedAndUpdates(t *testing.T) {
	fundID := uuid.New()
	p := waiveablePayment(fundID)
	repo := &mockWritePaymentRepo{payment: p}
	uc := NewWaivePaymentUseCase(repo)

	result, err := uc.Execute(context.Background(), p.ID, fundID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Status != ledger.PaymentStatusWaived {
		t.Errorf("status = %s, want waived", result.Status)
	}
	if result.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
	if repo.updated == nil || repo.updated.ID != p.ID {
		t.Errorf("Update was not called with payment")
	}
}

func TestWaivePayment_NotFound_ReturnsPaymentNotFound(t *testing.T) {
	uc := NewWaivePaymentUseCase(&mockWritePaymentRepo{})

	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, apperror.ErrPaymentNotFound) {
		t.Fatalf("Execute() error = %v, want ErrPaymentNotFound", err)
	}
}

func TestWaivePayment_FundMismatch_ReturnsPaymentNotFoundAndDoesNotUpdate(t *testing.T) {
	p := waiveablePayment(uuid.New())
	repo := &mockWritePaymentRepo{payment: p}
	uc := NewWaivePaymentUseCase(repo)

	_, err := uc.Execute(context.Background(), p.ID, uuid.New())
	if !errors.Is(err, apperror.ErrPaymentNotFound) {
		t.Fatalf("Execute() error = %v, want ErrPaymentNotFound", err)
	}
	if repo.updated != nil {
		t.Error("Update should not be called for fund mismatch")
	}
}

func TestWaivePayment_InvalidStatuses_ReturnErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     ledger.PaymentStatus
		wantCode   string
		wantCommon error
	}{
		{"paid", ledger.PaymentStatusPaid, "", apperror.ErrPaymentAlreadyPaid},
		{"already waived", ledger.PaymentStatusWaived, "ALREADY_WAIVED", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fundID := uuid.New()
			p := waiveablePayment(fundID)
			p.Status = tc.status
			repo := &mockWritePaymentRepo{payment: p}
			uc := NewWaivePaymentUseCase(repo)

			_, err := uc.Execute(context.Background(), p.ID, fundID)
			if tc.wantCommon != nil && !errors.Is(err, tc.wantCommon) {
				t.Fatalf("Execute() error = %v, want %v", err, tc.wantCommon)
			}
			if tc.wantCode != "" {
				appErr, ok := apperror.As(err)
				if !ok || appErr.Code != tc.wantCode {
					t.Fatalf("Execute() error = %v, want code %s", err, tc.wantCode)
				}
			}
			if repo.updated != nil {
				t.Error("Update should not be called for invalid status")
			}
		})
	}
}

func TestWaivePayment_UpdateError_ReturnsError(t *testing.T) {
	fundID := uuid.New()
	p := waiveablePayment(fundID)
	wantErr := errors.New("update failed")
	uc := NewWaivePaymentUseCase(&mockWritePaymentRepo{payment: p, updErr: wantErr})

	_, err := uc.Execute(context.Background(), p.ID, fundID)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
