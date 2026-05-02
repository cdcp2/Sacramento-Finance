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

// ─── shared mocks ─────────────────────────────────────────────────────────────

type mockFondoRepo struct{ cfg *fund.FondoConfig }

func (m *mockFondoRepo) Create(_ context.Context, f *fund.FondoConfig) error { return nil }
func (m *mockFondoRepo) GetByFundID(_ context.Context, _ uuid.UUID) (*fund.FondoConfig, error) {
	return m.cfg, nil
}
func (m *mockFondoRepo) Update(_ context.Context, _ *fund.FondoConfig) error { return nil }

type mockFondoLedger struct {
	fundBalance   decimal.Decimal
	memberBalance decimal.Decimal
	recorded      []*ledger.LedgerEntry
}

func (m *mockFondoLedger) RecordEntriesTx(_ context.Context, entries []*ledger.LedgerEntry) error {
	m.recorded = append(m.recorded, entries...)
	return nil
}
func (m *mockFondoLedger) GetFundBalance(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
	return m.fundBalance, nil
}
func (m *mockFondoLedger) GetMemberBalance(_ context.Context, _, _ uuid.UUID) (decimal.Decimal, error) {
	return m.memberBalance, nil
}

func fondoFund() *fund.Fund {
	return &fund.Fund{ID: uuid.New(), Type: fund.FundTypeFondoAhorro, Status: fund.FundStatusActive}
}

// ─── AccrueInterest tests ─────────────────────────────────────────────────────

func TestAccrueInterest_CalculatesCorrectly(t *testing.T) {
	tests := []struct {
		name           string
		balance        string
		rate           string // percentage, e.g. "2" means 2%
		wantInterest   string
	}{
		{"2% on 1,000,000", "1000000", "2", "20000"},
		{"1.5% on 500,000", "500000", "1.5", "7500"},
		{"0.5% on 200,000", "200000", "0.5", "1000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			balance, _ := decimal.NewFromString(tc.balance)
			rate, _    := decimal.NewFromString(tc.rate)
			wantAmt, _ := decimal.NewFromString(tc.wantInterest)

			ledgerMock := &mockFondoLedger{fundBalance: balance}
			uc := fondo.NewAccrueInterestUseCase(
				&mockFondoRepo{cfg: &fund.FondoConfig{
					FundID:       uuid.New(),
					InterestRate: rate,
				}},
				ledgerMock,
			)

			entry, err := uc.Execute(context.Background(), fondoFund())
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !entry.Amount.Equal(wantAmt) {
				t.Errorf("interest = %s, want %s", entry.Amount, wantAmt)
			}
			if entry.Direction != ledger.DirectionCredit {
				t.Errorf("entry direction = %s, want credit", entry.Direction)
			}
			if entry.Type != ledger.EntryTypeInterest {
				t.Errorf("entry type = %s, want interest", entry.Type)
			}
		})
	}
}

func TestAccrueInterest_ZeroBalance_ReturnsError(t *testing.T) {
	ledgerMock := &mockFondoLedger{fundBalance: decimal.Zero}
	uc := fondo.NewAccrueInterestUseCase(
		&mockFondoRepo{cfg: &fund.FondoConfig{InterestRate: decimal.NewFromInt(2)}},
		ledgerMock,
	)
	_, err := uc.Execute(context.Background(), fondoFund())
	if err == nil {
		t.Error("expected error when fund balance is zero")
	}
}

func TestAccrueInterest_ZeroRate_ReturnsError(t *testing.T) {
	ledgerMock := &mockFondoLedger{fundBalance: decimal.NewFromInt(1000000)}
	uc := fondo.NewAccrueInterestUseCase(
		&mockFondoRepo{cfg: &fund.FondoConfig{InterestRate: decimal.Zero}},
		ledgerMock,
	)
	_, err := uc.Execute(context.Background(), fondoFund())
	if err == nil {
		t.Error("expected error when interest rate is zero")
	}
}

func TestAccrueInterest_WrongFundType_ReturnsError(t *testing.T) {
	uc := fondo.NewAccrueInterestUseCase(&mockFondoRepo{}, &mockFondoLedger{})
	f := &fund.Fund{ID: uuid.New(), Type: fund.FundTypeVaca, Status: fund.FundStatusActive}
	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error for non-fondo fund type")
	}
}
