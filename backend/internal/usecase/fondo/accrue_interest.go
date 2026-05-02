package fondo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
	"github.com/sacramento-finance/backend/pkg/money"
	"github.com/shopspring/decimal"
)

// FondoLedger is the subset of LedgerRepo needed by fondo use cases.
type FondoLedger interface {
	RecordEntriesTx(ctx context.Context, entries []*ledger.LedgerEntry) error
	GetFundBalance(ctx context.Context, fundID uuid.UUID) (decimal.Decimal, error)
	GetMemberBalance(ctx context.Context, fundID, userID uuid.UUID) (decimal.Decimal, error)
}

type AccrueInterestUseCase struct {
	fondo  fund.FondoRepository
	ledger FondoLedger
}

func NewAccrueInterestUseCase(fondo fund.FondoRepository, ledger FondoLedger) *AccrueInterestUseCase {
	return &AccrueInterestUseCase{fondo: fondo, ledger: ledger}
}

// Execute calculates interest on the current fund balance and records a credit entry.
// Interest = balance × (rate / 100).
func (uc *AccrueInterestUseCase) Execute(ctx context.Context, f *fund.Fund) (*ledger.LedgerEntry, error) {
	if f.Type != fund.FundTypeFondoAhorro {
		return nil, apperror.New(400, "WRONG_FUND_TYPE", "this fund is not a fondo de ahorro")
	}
	if f.Status != fund.FundStatusActive {
		return nil, apperror.ErrFundNotActive
	}

	cfg, err := uc.fondo.GetByFundID(ctx, f.ID)
	if err != nil || cfg == nil {
		return nil, apperror.New(404, "FONDO_CONFIG_NOT_FOUND", "fondo config not found")
	}
	if cfg.InterestRate.IsZero() {
		return nil, apperror.New(400, "NO_INTEREST_RATE", "this fund has no interest rate configured")
	}

	balance, err := uc.ledger.GetFundBalance(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	if balance.IsZero() {
		return nil, apperror.New(400, "ZERO_BALANCE", "no balance to apply interest to")
	}

	interestAmount := money.Percentage(balance, cfg.InterestRate)
	if interestAmount.IsZero() {
		return nil, apperror.New(400, "ZERO_INTEREST", "calculated interest rounds to zero")
	}

	now := time.Now().UTC()
	entry := &ledger.LedgerEntry{
		ID:          idgen.New(),
		FundID:      f.ID,
		Type:        ledger.EntryTypeInterest,
		Direction:   ledger.DirectionCredit,
		Amount:      interestAmount,
		Description: "interest accrual at " + cfg.InterestRate.String() + "%",
		CreatedAt:   now,
	}

	if err := uc.ledger.RecordEntriesTx(ctx, []*ledger.LedgerEntry{entry}); err != nil {
		return nil, err
	}
	return entry, nil
}
