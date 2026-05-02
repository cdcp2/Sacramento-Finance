package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/shopspring/decimal"
)

type mockReadPaymentRepo struct {
	payment *ledger.Payment
	err     error
}

func (m *mockReadPaymentRepo) GetByID(_ context.Context, _ uuid.UUID) (*ledger.Payment, error) {
	return m.payment, m.err
}

type mockPaymentLedger struct {
	payment *ledger.Payment
	entry   *ledger.LedgerEntry
	err     error
	called  bool
}

func (m *mockPaymentLedger) RecordPaymentTx(_ context.Context, p *ledger.Payment, entry *ledger.LedgerEntry) error {
	m.called = true
	m.payment = p
	m.entry = entry
	return m.err
}

func basePayment(memberID uuid.UUID) *ledger.Payment {
	return &ledger.Payment{
		ID:           uuid.New(),
		FundID:       uuid.New(),
		MemberID:     memberID,
		PeriodNumber: 12,
		DueDate:      time.Now().UTC().Add(24 * time.Hour),
		AmountDue:    decimal.NewFromInt(100000),
		AmountPaid:   decimal.Zero,
		Status:       ledger.PaymentStatusPending,
	}
}

func basePaymentFund() *fund.Fund {
	return &fund.Fund{
		ID: uuid.New(),
		Rules: fund.Rules{
			PenaltyEnabled: false,
		},
	}
}

func TestRecordPayment_Success_NoPenalty(t *testing.T) {
	memberID := uuid.New()
	userID := uuid.New()
	p := basePayment(memberID)
	f := basePaymentFund()
	p.FundID = f.ID

	ledgerMock := &mockPaymentLedger{}
	uc := NewRecordPaymentUseCase(&mockReadPaymentRepo{payment: p}, ledgerMock)

	result, err := uc.Execute(context.Background(), p.ID, memberID, userID, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !ledgerMock.called {
		t.Fatal("expected ledger transaction to be called")
	}
	if result.Payment.Status != ledger.PaymentStatusPaid {
		t.Errorf("payment status = %s, want paid", result.Payment.Status)
	}
	if !result.Payment.AmountPaid.Equal(p.AmountDue) {
		t.Errorf("AmountPaid = %s, want %s", result.Payment.AmountPaid, p.AmountDue)
	}
	if result.Payment.PaidAt == nil {
		t.Error("PaidAt should be set")
	}
	if result.Payment.PenaltyApplied {
		t.Error("PenaltyApplied should be false")
	}
	if result.Entry.Type != ledger.EntryTypeDeposit {
		t.Errorf("entry type = %s, want deposit", result.Entry.Type)
	}
	if result.Entry.Direction != ledger.DirectionCredit {
		t.Errorf("entry direction = %s, want credit", result.Entry.Direction)
	}
	if !result.Entry.Amount.Equal(p.AmountDue) {
		t.Errorf("entry amount = %s, want %s", result.Entry.Amount, p.AmountDue)
	}
	if result.Entry.UserID == nil || *result.Entry.UserID != userID {
		t.Errorf("entry UserID = %v, want %s", result.Entry.UserID, userID)
	}
	if result.Entry.ReferenceID == nil || *result.Entry.ReferenceID != p.ID {
		t.Errorf("entry ReferenceID = %v, want %s", result.Entry.ReferenceID, p.ID)
	}
	if result.Entry.Description != "Payment for period 12" {
		t.Errorf("description = %q, want period description", result.Entry.Description)
	}
}

func TestRecordPayment_OverduePastGrace_AppliesFixedPenalty(t *testing.T) {
	memberID := uuid.New()
	p := basePayment(memberID)
	p.DueDate = time.Now().UTC().AddDate(0, 0, -3)
	f := basePaymentFund()
	f.ID = p.FundID
	f.Rules.PenaltyEnabled = true
	f.Rules.PenaltyType = fund.PenaltyTypeFixed
	f.Rules.PenaltyAmount = decimal.NewFromInt(5000)
	f.Rules.GracePeriodDays = 1

	uc := NewRecordPaymentUseCase(&mockReadPaymentRepo{payment: p}, &mockPaymentLedger{})

	result, err := uc.Execute(context.Background(), p.ID, memberID, uuid.New(), f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantTotal := decimal.NewFromInt(105000)
	if !result.Payment.AmountPaid.Equal(wantTotal) {
		t.Errorf("AmountPaid = %s, want %s", result.Payment.AmountPaid, wantTotal)
	}
	if !result.Payment.PenaltyApplied {
		t.Error("PenaltyApplied should be true")
	}
	if !result.Payment.PenaltyAmount.Equal(decimal.NewFromInt(5000)) {
		t.Errorf("PenaltyAmount = %s, want 5000", result.Payment.PenaltyAmount)
	}
	if !result.Entry.Amount.Equal(wantTotal) {
		t.Errorf("entry amount = %s, want %s", result.Entry.Amount, wantTotal)
	}
}

func TestRecordPayment_OverduePastGrace_AppliesPercentagePenalty(t *testing.T) {
	memberID := uuid.New()
	p := basePayment(memberID)
	p.DueDate = time.Now().UTC().AddDate(0, 0, -5)
	f := basePaymentFund()
	f.ID = p.FundID
	f.Rules.PenaltyEnabled = true
	f.Rules.PenaltyType = fund.PenaltyTypePercentage
	f.Rules.PenaltyAmount = decimal.NewFromInt(10)
	f.Rules.GracePeriodDays = 2

	uc := NewRecordPaymentUseCase(&mockReadPaymentRepo{payment: p}, &mockPaymentLedger{})

	result, err := uc.Execute(context.Background(), p.ID, memberID, uuid.New(), f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantPenalty := decimal.NewFromInt(10000)
	wantTotal := decimal.NewFromInt(110000)
	if !result.Payment.PenaltyAmount.Equal(wantPenalty) {
		t.Errorf("PenaltyAmount = %s, want %s", result.Payment.PenaltyAmount, wantPenalty)
	}
	if !result.Payment.AmountPaid.Equal(wantTotal) {
		t.Errorf("AmountPaid = %s, want %s", result.Payment.AmountPaid, wantTotal)
	}
}

func TestRecordPayment_OverdueInsideGrace_DoesNotApplyPenalty(t *testing.T) {
	memberID := uuid.New()
	p := basePayment(memberID)
	p.DueDate = time.Now().UTC().AddDate(0, 0, -1)
	f := basePaymentFund()
	f.ID = p.FundID
	f.Rules.PenaltyEnabled = true
	f.Rules.PenaltyType = fund.PenaltyTypeFixed
	f.Rules.PenaltyAmount = decimal.NewFromInt(5000)
	f.Rules.GracePeriodDays = 3

	uc := NewRecordPaymentUseCase(&mockReadPaymentRepo{payment: p}, &mockPaymentLedger{})

	result, err := uc.Execute(context.Background(), p.ID, memberID, uuid.New(), f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Payment.PenaltyApplied {
		t.Error("PenaltyApplied should be false inside grace period")
	}
	if !result.Payment.AmountPaid.Equal(decimal.NewFromInt(100000)) {
		t.Errorf("AmountPaid = %s, want 100000", result.Payment.AmountPaid)
	}
}

func TestRecordPayment_WrongMember_ReturnsForbiddenAndDoesNotWriteLedger(t *testing.T) {
	p := basePayment(uuid.New())
	ledgerMock := &mockPaymentLedger{}
	uc := NewRecordPaymentUseCase(&mockReadPaymentRepo{payment: p}, ledgerMock)

	_, err := uc.Execute(context.Background(), p.ID, uuid.New(), uuid.New(), basePaymentFund())
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("Execute() error = %v, want ErrForbidden", err)
	}
	if ledgerMock.called {
		t.Error("ledger should not be called for wrong member")
	}
}

func TestRecordPayment_NotFound_ReturnsPaymentNotFound(t *testing.T) {
	uc := NewRecordPaymentUseCase(&mockReadPaymentRepo{}, &mockPaymentLedger{})

	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), uuid.New(), basePaymentFund())
	if !errors.Is(err, apperror.ErrPaymentNotFound) {
		t.Fatalf("Execute() error = %v, want ErrPaymentNotFound", err)
	}
}

func TestRecordPayment_InvalidStatuses_ReturnErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     ledger.PaymentStatus
		wantCode   string
		wantCommon error
	}{
		{"paid", ledger.PaymentStatusPaid, "", apperror.ErrPaymentAlreadyPaid},
		{"waived", ledger.PaymentStatusWaived, "PAYMENT_WAIVED", nil},
		{"missed", ledger.PaymentStatusMissed, "PAYMENT_MISSED", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			memberID := uuid.New()
			p := basePayment(memberID)
			p.Status = tc.status
			ledgerMock := &mockPaymentLedger{}
			uc := NewRecordPaymentUseCase(&mockReadPaymentRepo{payment: p}, ledgerMock)

			_, err := uc.Execute(context.Background(), p.ID, memberID, uuid.New(), basePaymentFund())
			if tc.wantCommon != nil && !errors.Is(err, tc.wantCommon) {
				t.Fatalf("Execute() error = %v, want %v", err, tc.wantCommon)
			}
			if tc.wantCode != "" {
				appErr, ok := apperror.As(err)
				if !ok || appErr.Code != tc.wantCode {
					t.Fatalf("Execute() error = %v, want code %s", err, tc.wantCode)
				}
			}
			if ledgerMock.called {
				t.Error("ledger should not be called for invalid status")
			}
		})
	}
}

func TestRecordPayment_LedgerError_ReturnsError(t *testing.T) {
	memberID := uuid.New()
	p := basePayment(memberID)
	wantErr := errors.New("ledger failed")
	uc := NewRecordPaymentUseCase(&mockReadPaymentRepo{payment: p}, &mockPaymentLedger{err: wantErr})

	_, err := uc.Execute(context.Background(), p.ID, memberID, uuid.New(), basePaymentFund())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
