package fondo_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/internal/usecase/fondo"
	"github.com/shopspring/decimal"
)

func TestWithdraw_NoEarlyPenalty_WhenCompleted(t *testing.T) {
	userID := uuid.New()
	balance := decimal.NewFromInt(500000)
	amount  := decimal.NewFromInt(200000)

	ledgerMock := &mockFondoLedger{memberBalance: balance}
	uc := fondo.NewWithdrawUseCase(
		&mockFondoRepo{cfg: &fund.FondoConfig{
			FundID:                 uuid.New(),
			EarlyWithdrawalPenalty: decimal.NewFromInt(5), // 5%
		}},
		ledgerMock,
	)

	// Fund is completed — no early-withdrawal penalty applies
	f := &fund.Fund{ID: uuid.New(), Type: fund.FundTypeFondoAhorro, Status: fund.FundStatusCompleted}
	result, err := uc.Execute(context.Background(), f, userID, amount)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.PenaltyEntry != nil {
		t.Error("expected no penalty entry when fund is completed")
	}
	if len(ledgerMock.recorded) != 1 {
		t.Errorf("expected 1 ledger entry, got %d", len(ledgerMock.recorded))
	}
}

func TestWithdraw_EarlyPenaltyApplied_WhenActive(t *testing.T) {
	userID  := uuid.New()
	balance := decimal.NewFromInt(1000000)
	amount  := decimal.NewFromInt(200000)
	// 5% of 200,000 = 10,000
	wantPenalty := decimal.NewFromInt(10000)

	ledgerMock := &mockFondoLedger{memberBalance: balance}
	uc := fondo.NewWithdrawUseCase(
		&mockFondoRepo{cfg: &fund.FondoConfig{
			FundID:                 uuid.New(),
			EarlyWithdrawalPenalty: decimal.NewFromInt(5),
		}},
		ledgerMock,
	)

	f := &fund.Fund{ID: uuid.New(), Type: fund.FundTypeFondoAhorro, Status: fund.FundStatusActive}
	result, err := uc.Execute(context.Background(), f, userID, amount)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.PenaltyEntry == nil {
		t.Fatal("expected penalty entry when fund is active")
	}
	if !result.PenaltyEntry.Amount.Equal(wantPenalty) {
		t.Errorf("penalty = %s, want %s", result.PenaltyEntry.Amount, wantPenalty)
	}
	if result.PenaltyEntry.Type != ledger.EntryTypePenalty {
		t.Errorf("penalty entry type = %s, want penalty", result.PenaltyEntry.Type)
	}
	// Two entries: withdrawal + penalty
	if len(ledgerMock.recorded) != 2 {
		t.Errorf("expected 2 ledger entries, got %d", len(ledgerMock.recorded))
	}
}

func TestWithdraw_InsufficientBalance_ReturnsError(t *testing.T) {
	userID  := uuid.New()
	balance := decimal.NewFromInt(100000)
	amount  := decimal.NewFromInt(200000) // more than balance

	ledgerMock := &mockFondoLedger{memberBalance: balance}
	uc := fondo.NewWithdrawUseCase(
		&mockFondoRepo{cfg: &fund.FondoConfig{FundID: uuid.New()}},
		ledgerMock,
	)

	f := &fund.Fund{ID: uuid.New(), Type: fund.FundTypeFondoAhorro, Status: fund.FundStatusActive}
	_, err := uc.Execute(context.Background(), f, userID, amount)
	if err == nil {
		t.Error("expected error when withdrawal exceeds member balance")
	}
}

func TestWithdraw_Preview_CalculatesPenalty(t *testing.T) {
	userID  := uuid.New()
	balance := decimal.NewFromInt(1000000)
	amount  := decimal.NewFromInt(500000)
	// 3% of 500,000 = 15,000
	wantPenalty := decimal.NewFromInt(15000)
	wantNet     := decimal.NewFromInt(485000)

	ledgerMock := &mockFondoLedger{memberBalance: balance}
	uc := fondo.NewWithdrawUseCase(
		&mockFondoRepo{cfg: &fund.FondoConfig{
			FundID:                 uuid.New(),
			EarlyWithdrawalPenalty: decimal.NewFromInt(3),
		}},
		ledgerMock,
	)

	f := &fund.Fund{ID: uuid.New(), Type: fund.FundTypeFondoAhorro, Status: fund.FundStatusActive}
	preview, err := uc.Preview(context.Background(), f, userID, amount)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if !preview.PenaltyAmount.Equal(wantPenalty) {
		t.Errorf("penalty = %s, want %s", preview.PenaltyAmount, wantPenalty)
	}
	if !preview.NetAmount.Equal(wantNet) {
		t.Errorf("net = %s, want %s", preview.NetAmount, wantNet)
	}
	if !preview.IsEarly {
		t.Error("expected IsEarly=true for active fund")
	}
	// Preview must not write to the ledger
	if len(ledgerMock.recorded) != 0 {
		t.Error("Preview() must not write ledger entries")
	}
}

func TestWithdraw_WrongFundType_ReturnsError(t *testing.T) {
	uc := fondo.NewWithdrawUseCase(&mockFondoRepo{}, &mockFondoLedger{})
	f := &fund.Fund{ID: uuid.New(), Type: fund.FundTypeCirculo, Status: fund.FundStatusActive}
	_, err := uc.Execute(context.Background(), f, uuid.New(), decimal.NewFromInt(1000))
	if err == nil {
		t.Error("expected error for non-fondo fund type")
	}
}
