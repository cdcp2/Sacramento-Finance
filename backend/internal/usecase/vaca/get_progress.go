package vaca

import (
	"context"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/shopspring/decimal"
)

// BalanceReader is the subset of LedgerRepo needed here.
type BalanceReader interface {
	GetFundBalance(ctx context.Context, fundID uuid.UUID) (decimal.Decimal, error)
}

// ProgressResult carries the vaca goal progress data.
type ProgressResult struct {
	GoalAmount      decimal.Decimal `json:"goal_amount"`
	GoalDescription string          `json:"goal_description"`
	DistributionType string         `json:"distribution_type"`
	CurrentBalance  decimal.Decimal `json:"current_balance"`
	Percentage      decimal.Decimal `json:"percentage"`
	GoalReached     bool            `json:"goal_reached"`
}

type GetProgressUseCase struct {
	vaca    fund.VacaRepository
	balance BalanceReader
}

func NewGetProgressUseCase(vaca fund.VacaRepository, balance BalanceReader) *GetProgressUseCase {
	return &GetProgressUseCase{vaca: vaca, balance: balance}
}

func (uc *GetProgressUseCase) Execute(ctx context.Context, f *fund.Fund) (*ProgressResult, error) {
	if f.Type != fund.FundTypeVaca {
		return nil, apperror.New(400, "WRONG_FUND_TYPE", "this fund is not a vaca")
	}

	cfg, err := uc.vaca.GetByFundID(ctx, f.ID)
	if err != nil || cfg == nil {
		return nil, apperror.New(404, "VACA_CONFIG_NOT_FOUND", "vaca config not found")
	}

	balance, err := uc.balance.GetFundBalance(ctx, f.ID)
	if err != nil {
		return nil, err
	}

	var percentage decimal.Decimal
	if !cfg.GoalAmount.IsZero() {
		percentage = balance.Div(cfg.GoalAmount).Mul(decimal.NewFromInt(100)).Round(2)
	}

	return &ProgressResult{
		GoalAmount:       cfg.GoalAmount,
		GoalDescription:  cfg.GoalDescription,
		DistributionType: cfg.DistributionType,
		CurrentBalance:   balance,
		Percentage:       percentage,
		GoalReached:      balance.GreaterThanOrEqual(cfg.GoalAmount),
	}, nil
}
