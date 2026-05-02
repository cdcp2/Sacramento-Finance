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

type WithdrawUseCase struct {
	fondo  fund.FondoRepository
	ledger FondoLedger
}

func NewWithdrawUseCase(fondo fund.FondoRepository, ledger FondoLedger) *WithdrawUseCase {
	return &WithdrawUseCase{fondo: fondo, ledger: ledger}
}

// WithdrawPreview returns penalty and net amount without writing anything.
type WithdrawPreview struct {
	Amount        decimal.Decimal `json:"amount"`
	PenaltyRate   decimal.Decimal `json:"penalty_rate"`
	PenaltyAmount decimal.Decimal `json:"penalty_amount"`
	NetAmount     decimal.Decimal `json:"net_amount"`
	IsEarly       bool            `json:"is_early"`
}

// WithdrawResult holds the created ledger entries.
type WithdrawResult struct {
	WithdrawalEntry *ledger.LedgerEntry
	PenaltyEntry    *ledger.LedgerEntry // nil when no early-withdrawal penalty
}

// Preview returns the penalty calculation for a prospective withdrawal (no DB writes).
func (uc *WithdrawUseCase) Preview(
	ctx context.Context,
	f *fund.Fund,
	userID uuid.UUID,
	amount decimal.Decimal,
) (*WithdrawPreview, error) {
	if f.Type != fund.FundTypeFondoAhorro {
		return nil, apperror.New(400, "WRONG_FUND_TYPE", "this fund is not a fondo de ahorro")
	}

	cfg, err := uc.fondo.GetByFundID(ctx, f.ID)
	if err != nil || cfg == nil {
		return nil, apperror.New(404, "FONDO_CONFIG_NOT_FOUND", "fondo config not found")
	}

	memberBalance, err := uc.ledger.GetMemberBalance(ctx, f.ID, userID)
	if err != nil {
		return nil, err
	}
	if amount.GreaterThan(memberBalance) {
		return nil, apperror.New(400, "INSUFFICIENT_BALANCE", "withdrawal amount exceeds your balance in this fund")
	}

	isEarly := f.Status == fund.FundStatusActive
	penaltyAmount := decimal.Zero
	if isEarly && !cfg.EarlyWithdrawalPenalty.IsZero() {
		penaltyAmount = money.Percentage(amount, cfg.EarlyWithdrawalPenalty)
	}

	return &WithdrawPreview{
		Amount:        amount,
		PenaltyRate:   cfg.EarlyWithdrawalPenalty,
		PenaltyAmount: penaltyAmount,
		NetAmount:     amount.Sub(penaltyAmount),
		IsEarly:       isEarly,
	}, nil
}

// Execute records the withdrawal (and optional early-withdrawal penalty) in the ledger.
func (uc *WithdrawUseCase) Execute(
	ctx context.Context,
	f *fund.Fund,
	userID uuid.UUID,
	amount decimal.Decimal,
) (*WithdrawResult, error) {
	if f.Type != fund.FundTypeFondoAhorro {
		return nil, apperror.New(400, "WRONG_FUND_TYPE", "this fund is not a fondo de ahorro")
	}
	if f.Status == fund.FundStatusCancelled || f.Status == fund.FundStatusDraft {
		return nil, apperror.New(400, "INVALID_FUND_STATE", "withdrawals not allowed in current fund state")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, apperror.New(400, "INVALID_AMOUNT", "withdrawal amount must be greater than zero")
	}

	cfg, err := uc.fondo.GetByFundID(ctx, f.ID)
	if err != nil || cfg == nil {
		return nil, apperror.New(404, "FONDO_CONFIG_NOT_FOUND", "fondo config not found")
	}

	memberBalance, err := uc.ledger.GetMemberBalance(ctx, f.ID, userID)
	if err != nil {
		return nil, err
	}
	if amount.GreaterThan(memberBalance) {
		return nil, apperror.New(400, "INSUFFICIENT_BALANCE", "withdrawal amount exceeds your balance in this fund")
	}

	isEarly := f.Status == fund.FundStatusActive
	penaltyAmount := decimal.Zero
	if isEarly && !cfg.EarlyWithdrawalPenalty.IsZero() {
		penaltyAmount = money.Percentage(amount, cfg.EarlyWithdrawalPenalty)
	}

	now := time.Now().UTC()
	entries := make([]*ledger.LedgerEntry, 0, 2)

	withdrawalEntry := &ledger.LedgerEntry{
		ID:          idgen.New(),
		FundID:      f.ID,
		UserID:      &userID,
		Type:        ledger.EntryTypeWithdrawal,
		Direction:   ledger.DirectionDebit,
		Amount:      amount,
		Description: "withdrawal by member",
		CreatedAt:   now,
	}
	entries = append(entries, withdrawalEntry)

	var penaltyEntry *ledger.LedgerEntry
	if penaltyAmount.GreaterThan(decimal.Zero) {
		penaltyEntry = &ledger.LedgerEntry{
			ID:          idgen.New(),
			FundID:      f.ID,
			UserID:      &userID,
			Type:        ledger.EntryTypePenalty,
			Direction:   ledger.DirectionDebit,
			Amount:      penaltyAmount,
			Description: "early withdrawal penalty at " + cfg.EarlyWithdrawalPenalty.String() + "%",
			CreatedAt:   now,
		}
		entries = append(entries, penaltyEntry)
	}

	if err := uc.ledger.RecordEntriesTx(ctx, entries); err != nil {
		return nil, err
	}

	return &WithdrawResult{
		WithdrawalEntry: withdrawalEntry,
		PenaltyEntry:    penaltyEntry,
	}, nil
}
