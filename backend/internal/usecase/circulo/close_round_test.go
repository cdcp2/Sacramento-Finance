package circulo_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/internal/usecase/circulo"
	"github.com/shopspring/decimal"
)

// ─── mocks ────────────────────────────────────────────────────────────────────

type mockFundRepo struct{ updated *fund.Fund }

func (m *mockFundRepo) Create(_ context.Context, f *fund.Fund) error { return nil }
func (m *mockFundRepo) GetByID(_ context.Context, _ uuid.UUID) (*fund.Fund, error) {
	return nil, nil
}
func (m *mockFundRepo) ListByMember(_ context.Context, _ uuid.UUID) ([]*fund.Fund, error) {
	return nil, nil
}
func (m *mockFundRepo) Update(_ context.Context, f *fund.Fund) error {
	m.updated = f
	return nil
}

type mockMemberRepo struct {
	members   []*fund.FundMember
	updated   []*fund.FundMember
	listErr   error
	updateErr error
}

func (m *mockMemberRepo) Add(_ context.Context, _ *fund.FundMember) error { return nil }
func (m *mockMemberRepo) GetByFundAndUser(_ context.Context, _, _ uuid.UUID) (*fund.FundMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) GetByID(_ context.Context, _ uuid.UUID) (*fund.FundMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*fund.FundMember, error) {
	return m.members, m.listErr
}
func (m *mockMemberRepo) Update(_ context.Context, member *fund.FundMember) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, member)
	return nil
}
func (m *mockMemberRepo) CountActive(_ context.Context, _ uuid.UUID) (int, error) {
	return len(m.members), nil
}

type mockCirculoRepo struct{ cfg *fund.CirculoConfig }

func (m *mockCirculoRepo) Create(_ context.Context, c *fund.CirculoConfig) error { return nil }
func (m *mockCirculoRepo) GetByFundID(_ context.Context, _ uuid.UUID) (*fund.CirculoConfig, error) {
	return m.cfg, nil
}
func (m *mockCirculoRepo) Update(_ context.Context, c *fund.CirculoConfig) error {
	m.cfg = c
	return nil
}

type mockPayoutRepo struct{ created *ledger.Payout }

func (m *mockPayoutRepo) Create(_ context.Context, p *ledger.Payout) error {
	m.created = p
	return nil
}
func (m *mockPayoutRepo) GetByID(_ context.Context, _ uuid.UUID) (*ledger.Payout, error) {
	return nil, nil
}
func (m *mockPayoutRepo) GetByFundAndRound(_ context.Context, _ uuid.UUID, _ int) (*ledger.Payout, error) {
	return nil, nil
}
func (m *mockPayoutRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*ledger.Payout, error) {
	return nil, nil
}
func (m *mockPayoutRepo) Update(_ context.Context, _ *ledger.Payout) error { return nil }

type mockLedgerWriter struct{ entries []*ledger.LedgerEntry }

func (m *mockLedgerWriter) RecordEntriesTx(_ context.Context, entries []*ledger.LedgerEntry) error {
	m.entries = append(m.entries, entries...)
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func activeCirculoFund(totalPeriods int) *fund.Fund {
	return &fund.Fund{
		ID:     uuid.New(),
		Type:   fund.FundTypeCirculo,
		Status: fund.FundStatusActive,
		Rules: fund.Rules{
			ContributionAmount: decimal.NewFromInt(100000),
			TotalPeriods:       totalPeriods,
		},
	}
}

func memberWithOrder(fundID uuid.UUID, order int) *fund.FundMember {
	return &fund.FundMember{
		ID:          uuid.New(),
		FundID:      fundID,
		UserID:      uuid.New(),
		Status:      fund.MemberStatusActive,
		PayoutOrder: &order,
	}
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestCloseRound_Success(t *testing.T) {
	f := activeCirculoFund(3)
	order1 := 1
	members := []*fund.FundMember{
		memberWithOrder(f.ID, 1),
		memberWithOrder(f.ID, 2),
		{ID: uuid.New(), FundID: f.ID, UserID: uuid.New(), Status: fund.MemberStatusActive, PayoutOrder: &order1},
	}
	members[2].PayoutOrder = func() *int { n := 3; return &n }()

	cfg := &fund.CirculoConfig{FundID: f.ID, CurrentRound: 1, RoundsCompleted: 0, PayoutOrderType: "fixed"}

	fundRepo := &mockFundRepo{}
	memberRepo := &mockMemberRepo{members: members}
	circuloRepo := &mockCirculoRepo{cfg: cfg}
	payoutRepo := &mockPayoutRepo{}
	ledgerWriter := &mockLedgerWriter{}

	uc := circulo.NewCloseRoundUseCase(fundRepo, memberRepo, circuloRepo, payoutRepo, ledgerWriter)
	result, err := uc.Execute(context.Background(), f)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Payout == nil {
		t.Fatal("expected payout to be created")
	}

	// Payout = 3 active members × 100000
	wantAmount := decimal.NewFromInt(300000)
	if !result.Payout.Amount.Equal(wantAmount) {
		t.Errorf("payout amount = %s, want %s", result.Payout.Amount, wantAmount)
	}

	// Round counter advanced
	if circuloRepo.cfg.RoundsCompleted != 1 {
		t.Errorf("RoundsCompleted = %d, want 1", circuloRepo.cfg.RoundsCompleted)
	}
	if circuloRepo.cfg.CurrentRound != 2 {
		t.Errorf("CurrentRound = %d, want 2", circuloRepo.cfg.CurrentRound)
	}

	// Ledger entry recorded
	if len(ledgerWriter.entries) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", len(ledgerWriter.entries))
	}
	entry := ledgerWriter.entries[0]
	if entry.Direction != ledger.DirectionDebit {
		t.Errorf("entry direction = %s, want debit", entry.Direction)
	}
	if entry.Type != ledger.EntryTypePayout {
		t.Errorf("entry type = %s, want payout", entry.Type)
	}

	// Fund not completed yet (only 1 of 3 rounds done)
	if result.Completed {
		t.Error("fund should not be marked completed after round 1 of 3")
	}
}

func TestCloseRound_LastRound_MarksCompleted(t *testing.T) {
	f := activeCirculoFund(1)
	cfg := &fund.CirculoConfig{FundID: f.ID, CurrentRound: 1, RoundsCompleted: 0}
	members := []*fund.FundMember{memberWithOrder(f.ID, 1)}

	fundRepo := &mockFundRepo{}
	memberRepo := &mockMemberRepo{members: members}
	circuloRepo := &mockCirculoRepo{cfg: cfg}
	payoutRepo := &mockPayoutRepo{}
	ledgerWriter := &mockLedgerWriter{}

	uc := circulo.NewCloseRoundUseCase(fundRepo, memberRepo, circuloRepo, payoutRepo, ledgerWriter)
	result, err := uc.Execute(context.Background(), f)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Completed {
		t.Error("fund should be marked completed after the last round")
	}
	if fundRepo.updated == nil || fundRepo.updated.Status != fund.FundStatusCompleted {
		t.Error("fund status should be updated to completed in the repo")
	}
}

func TestCloseRound_NoBeneficiary_ReturnsError(t *testing.T) {
	f := activeCirculoFund(3)
	cfg := &fund.CirculoConfig{FundID: f.ID, CurrentRound: 1, RoundsCompleted: 0}
	// Member has order=2, not 1 — no one assigned to round 1
	order2 := 2
	members := []*fund.FundMember{
		{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusActive, PayoutOrder: &order2},
	}

	uc := circulo.NewCloseRoundUseCase(
		&mockFundRepo{}, &mockMemberRepo{members: members},
		&mockCirculoRepo{cfg: cfg}, &mockPayoutRepo{}, &mockLedgerWriter{},
	)
	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error when no member is assigned to current round")
	}
}

func TestCloseRound_AllRoundsDone_ReturnsError(t *testing.T) {
	f := activeCirculoFund(2)
	cfg := &fund.CirculoConfig{FundID: f.ID, CurrentRound: 3, RoundsCompleted: 2}

	uc := circulo.NewCloseRoundUseCase(
		&mockFundRepo{}, &mockMemberRepo{}, &mockCirculoRepo{cfg: cfg},
		&mockPayoutRepo{}, &mockLedgerWriter{},
	)
	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error when all rounds already completed")
	}
}

func TestCloseRound_WrongFundType_ReturnsError(t *testing.T) {
	f := &fund.Fund{
		ID:     uuid.New(),
		Type:   fund.FundTypeVaca,
		Status: fund.FundStatusActive,
	}
	uc := circulo.NewCloseRoundUseCase(
		&mockFundRepo{}, &mockMemberRepo{}, &mockCirculoRepo{},
		&mockPayoutRepo{}, &mockLedgerWriter{},
	)
	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error for non-circulo fund")
	}
}

func TestCloseRound_InactiveFund_ReturnsError(t *testing.T) {
	f := &fund.Fund{
		ID:     uuid.New(),
		Type:   fund.FundTypeCirculo,
		Status: fund.FundStatusDraft,
	}
	uc := circulo.NewCloseRoundUseCase(
		&mockFundRepo{}, &mockMemberRepo{}, &mockCirculoRepo{},
		&mockPayoutRepo{}, &mockLedgerWriter{},
	)
	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error for non-active fund")
	}
}

// suppress unused import warning
var _ = time.Now
