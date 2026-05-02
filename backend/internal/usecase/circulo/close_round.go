package circulo

import (
	"context"
	"strconv"
	"time"

	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
	"github.com/shopspring/decimal"
)

// LedgerWriter is the subset of LedgerRepo needed by this use case.
type LedgerWriter interface {
	RecordEntriesTx(ctx context.Context, entries []*ledger.LedgerEntry) error
}

type CloseRoundUseCase struct {
	funds   fund.Repository
	members fund.MemberRepository
	circulo fund.CirculoRepository
	payouts ledger.PayoutRepository
	ledger  LedgerWriter
}

func NewCloseRoundUseCase(
	funds fund.Repository,
	members fund.MemberRepository,
	circulo fund.CirculoRepository,
	payouts ledger.PayoutRepository,
	ledger LedgerWriter,
) *CloseRoundUseCase {
	return &CloseRoundUseCase{
		funds:   funds,
		members: members,
		circulo: circulo,
		payouts: payouts,
		ledger:  ledger,
	}
}

// CloseRoundResult is returned on success.
type CloseRoundResult struct {
	Payout    *ledger.Payout
	Config    *fund.CirculoConfig
	Completed bool // true when the last round was just closed
}

// Execute closes the current round of a circulo fund:
//  1. Finds the member whose payout_order matches current_round.
//  2. Calculates payout = active_members × contribution_amount.
//  3. Records an immutable ledger entry (debit) and a payout row.
//  4. Advances the round counter; marks fund completed when all rounds finish.
func (uc *CloseRoundUseCase) Execute(ctx context.Context, f *fund.Fund) (*CloseRoundResult, error) {
	if f.Type != fund.FundTypeCirculo {
		return nil, apperror.New(400, "WRONG_FUND_TYPE", "this fund is not a circulo")
	}
	if f.Status != fund.FundStatusActive {
		return nil, apperror.ErrFundNotActive
	}

	cfg, err := uc.circulo.GetByFundID(ctx, f.ID)
	if err != nil || cfg == nil {
		return nil, apperror.New(404, "CIRCULO_CONFIG_NOT_FOUND", "circulo config not found")
	}
	if cfg.RoundsCompleted >= f.Rules.TotalPeriods {
		return nil, apperror.New(400, "ALL_ROUNDS_COMPLETED", "all rounds have already been completed")
	}

	members, err := uc.members.ListByFund(ctx, f.ID)
	if err != nil {
		return nil, err
	}

	// Find beneficiary for this round
	var beneficiary *fund.FundMember
	activeCount := 0
	for _, m := range members {
		if m.Status != fund.MemberStatusActive {
			continue
		}
		activeCount++
		if m.PayoutOrder != nil && *m.PayoutOrder == cfg.CurrentRound {
			beneficiary = m
		}
	}
	if beneficiary == nil {
		return nil, apperror.New(400, "NO_BENEFICIARY",
			"no member is assigned to round "+strconv.Itoa(cfg.CurrentRound)+"; call assign-payout-order first")
	}

	// Payout amount = all active members × contribution (theoretical pot)
	payoutAmount := f.Rules.ContributionAmount.Mul(decimal.NewFromInt(int64(activeCount)))

	now := time.Now().UTC()
	payoutID := idgen.New()

	payout := &ledger.Payout{
		ID:            payoutID,
		FundID:        f.ID,
		RecipientID:   beneficiary.ID,
		RoundNumber:   cfg.CurrentRound,
		Amount:        payoutAmount,
		Status:        "completed",
		ScheduledDate: now,
		CompletedAt:   &now,
	}
	if err := uc.payouts.Create(ctx, payout); err != nil {
		return nil, err
	}

	// Record immutable debit ledger entry
	entry := &ledger.LedgerEntry{
		ID:          idgen.New(),
		FundID:      f.ID,
		UserID:      &beneficiary.UserID,
		Type:        ledger.EntryTypePayout,
		Direction:   ledger.DirectionDebit,
		Amount:      payoutAmount,
		ReferenceID: &payoutID,
		Description: "circulo round " + strconv.Itoa(cfg.CurrentRound) + " payout to member",
		CreatedAt:   now,
	}
	if err := uc.ledger.RecordEntriesTx(ctx, []*ledger.LedgerEntry{entry}); err != nil {
		return nil, err
	}

	// Advance counters
	cfg.RoundsCompleted++
	cfg.CurrentRound++
	if err := uc.circulo.Update(ctx, cfg); err != nil {
		return nil, err
	}

	completed := cfg.RoundsCompleted >= f.Rules.TotalPeriods
	if completed {
		f.Status = fund.FundStatusCompleted
		f.UpdatedAt = now
		_ = uc.funds.Update(ctx, f)
	}

	return &CloseRoundResult{
		Payout:    payout,
		Config:    cfg,
		Completed: completed,
	}, nil
}
