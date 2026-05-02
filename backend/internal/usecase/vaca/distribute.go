package vaca

import (
	"context"
	"time"

	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
	"github.com/shopspring/decimal"
)

// LedgerDistributor is the subset of LedgerRepo needed for distribution.
type LedgerDistributor interface {
	GetFundBalance(ctx context.Context, fundID interface{ String() string }) (decimal.Decimal, error)
	RecordEntriesTx(ctx context.Context, entries []*ledger.LedgerEntry) error
}

// DistributeLedger is the real interface used (matches LedgerRepo).
type DistributeLedger interface {
	GetFundBalance(ctx context.Context, fundID interface{}) (decimal.Decimal, error)
	RecordEntriesTx(ctx context.Context, entries []*ledger.LedgerEntry) error
}

// VacaLedger is the interface subset used by DistributeUseCase.
type VacaLedger interface {
	BalanceReader
	RecordEntriesTx(ctx context.Context, entries []*ledger.LedgerEntry) error
}

// DistributeResult holds the outcome of a vaca distribution.
type DistributeResult struct {
	TotalDistributed decimal.Decimal     `json:"total_distributed"`
	SharePerMember   decimal.Decimal     `json:"share_per_member"`
	MemberCount      int                 `json:"member_count"`
	Entries          []*ledger.LedgerEntry `json:"entries"`
}

type DistributeUseCase struct {
	vaca    fund.VacaRepository
	funds   fund.Repository
	members fund.MemberRepository
	ledger  VacaLedger
}

func NewDistributeUseCase(
	vaca fund.VacaRepository,
	funds fund.Repository,
	members fund.MemberRepository,
	ledger VacaLedger,
) *DistributeUseCase {
	return &DistributeUseCase{
		vaca:    vaca,
		funds:   funds,
		members: members,
		ledger:  ledger,
	}
}

// Execute distributes the fund balance equally among active members and marks the fund completed.
// Authorization (admin check or proposal approval) must be done by the caller.
func (uc *DistributeUseCase) Execute(ctx context.Context, f *fund.Fund) (*DistributeResult, error) {
	if f.Type != fund.FundTypeVaca {
		return nil, apperror.New(400, "WRONG_FUND_TYPE", "this fund is not a vaca")
	}
	if f.Status != fund.FundStatusActive {
		return nil, apperror.New(409, "FUND_NOT_ACTIVE", "only active funds can be distributed")
	}

	balance, err := uc.ledger.GetFundBalance(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	if balance.IsZero() || balance.IsNegative() {
		return nil, apperror.New(409, "EMPTY_BALANCE", "fund balance is zero, nothing to distribute")
	}

	activeMembers, err := uc.members.ListByFund(ctx, f.ID)
	if err != nil {
		return nil, err
	}

	var recipients []*fund.FundMember
	for _, m := range activeMembers {
		if m.Status == fund.MemberStatusActive {
			recipients = append(recipients, m)
		}
	}
	if len(recipients) == 0 {
		return nil, apperror.New(409, "NO_ACTIVE_MEMBERS", "no active members to distribute to")
	}

	count := decimal.NewFromInt(int64(len(recipients)))
	// Truncate to whole pesos to avoid fractional amounts; remainder stays in fund
	share := balance.Div(count).Floor()

	now := time.Now().UTC()
	entries := make([]*ledger.LedgerEntry, 0, len(recipients))
	for _, m := range recipients {
		userID := m.UserID
		entries = append(entries, &ledger.LedgerEntry{
			ID:          idgen.New(),
			FundID:      f.ID,
			UserID:      &userID,
			Type:        ledger.EntryTypePayout,
			Direction:   ledger.DirectionDebit,
			Amount:      share,
			Description: "Distribución de vaca",
			CreatedAt:   now,
		})
	}

	if err := uc.ledger.RecordEntriesTx(ctx, entries); err != nil {
		return nil, err
	}

	// Mark fund completed
	f.Status = fund.FundStatusCompleted
	f.UpdatedAt = now
	if err := uc.funds.Update(ctx, f); err != nil {
		return nil, err
	}

	return &DistributeResult{
		TotalDistributed: share.Mul(count),
		SharePerMember:   share,
		MemberCount:      len(recipients),
		Entries:          entries,
	}, nil
}
