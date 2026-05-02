package vaca_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/internal/usecase/vaca"
	"github.com/shopspring/decimal"
)

// ─── mocks (extend those already declared in get_progress_test.go) ────────────
// get_progress_test.go already declares mockVacaRepo and a fund builder.
// We declare what we need here without duplicates.

type mockDistributeLedger struct {
	balance   decimal.Decimal
	balanceErr error
	recorded  []*ledger.LedgerEntry
	recordErr error
}

func (m *mockDistributeLedger) GetFundBalance(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
	return m.balance, m.balanceErr
}
func (m *mockDistributeLedger) RecordEntriesTx(_ context.Context, entries []*ledger.LedgerEntry) error {
	if m.recordErr != nil {
		return m.recordErr
	}
	m.recorded = append(m.recorded, entries...)
	return nil
}

type mockDistributeFundRepo struct{ updated *fund.Fund }

func (m *mockDistributeFundRepo) Create(_ context.Context, _ *fund.Fund) error { return nil }
func (m *mockDistributeFundRepo) GetByID(_ context.Context, _ uuid.UUID) (*fund.Fund, error) {
	return nil, nil
}
func (m *mockDistributeFundRepo) ListByMember(_ context.Context, _ uuid.UUID) ([]*fund.Fund, error) {
	return nil, nil
}
func (m *mockDistributeFundRepo) Update(_ context.Context, f *fund.Fund) error {
	m.updated = f
	return nil
}

type mockDistributeMemberRepo struct{ members []*fund.FundMember }

func (m *mockDistributeMemberRepo) Add(_ context.Context, _ *fund.FundMember) error { return nil }
func (m *mockDistributeMemberRepo) GetByFundAndUser(_ context.Context, _, _ uuid.UUID) (*fund.FundMember, error) {
	return nil, nil
}
func (m *mockDistributeMemberRepo) GetByID(_ context.Context, _ uuid.UUID) (*fund.FundMember, error) {
	return nil, nil
}
func (m *mockDistributeMemberRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*fund.FundMember, error) {
	return m.members, nil
}
func (m *mockDistributeMemberRepo) Update(_ context.Context, _ *fund.FundMember) error { return nil }
func (m *mockDistributeMemberRepo) CountActive(_ context.Context, _ uuid.UUID) (int, error) {
	return len(m.members), nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func activeMembers(n int, fundID uuid.UUID) []*fund.FundMember {
	out := make([]*fund.FundMember, n)
	for i := range out {
		out[i] = &fund.FundMember{
			ID:     uuid.New(),
			FundID: fundID,
			UserID: uuid.New(),
			Status: fund.MemberStatusActive,
		}
	}
	return out
}

func newDistributeUC(vacaR fund.VacaRepository, fundsR fund.Repository, membersR fund.MemberRepository, l vaca.VacaLedger) *vaca.DistributeUseCase {
	return vaca.NewDistributeUseCase(vacaR, fundsR, membersR, l)
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestDistribute_EqualShareAmongMembers(t *testing.T) {
	f := vacaFund()
	members := activeMembers(3, f.ID)
	balance := decimal.NewFromInt(900000) // 900,000 ÷ 3 = 300,000

	ledgerMock := &mockDistributeLedger{balance: balance}
	fundsMock := &mockDistributeFundRepo{}
	uc := newDistributeUC(&mockVacaRepo{}, fundsMock, &mockDistributeMemberRepo{members: members}, ledgerMock)

	result, err := uc.Execute(context.Background(), f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantShare := decimal.NewFromInt(300000)
	if !result.SharePerMember.Equal(wantShare) {
		t.Errorf("SharePerMember = %s, want %s", result.SharePerMember, wantShare)
	}
	if result.MemberCount != 3 {
		t.Errorf("MemberCount = %d, want 3", result.MemberCount)
	}
	if !result.TotalDistributed.Equal(balance) {
		t.Errorf("TotalDistributed = %s, want %s", result.TotalDistributed, balance)
	}
	if len(ledgerMock.recorded) != 3 {
		t.Errorf("expected 3 ledger entries, got %d", len(ledgerMock.recorded))
	}
}

func TestDistribute_UnEvenBalance_TruncatesToFloor(t *testing.T) {
	// 1,000,000 ÷ 3 = 333,333.33... → floored to 333,333 each
	f := vacaFund()
	members := activeMembers(3, f.ID)
	balance := decimal.NewFromInt(1000000)

	ledgerMock := &mockDistributeLedger{balance: balance}
	uc := newDistributeUC(&mockVacaRepo{}, &mockDistributeFundRepo{}, &mockDistributeMemberRepo{members: members}, ledgerMock)

	result, err := uc.Execute(context.Background(), f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantShare := decimal.NewFromInt(333333)
	if !result.SharePerMember.Equal(wantShare) {
		t.Errorf("SharePerMember = %s, want %s (floor)", result.SharePerMember, wantShare)
	}
	// Total distributed < original balance (remainder stays in fund)
	if result.TotalDistributed.GreaterThan(balance) {
		t.Error("TotalDistributed must not exceed the original balance")
	}
}

func TestDistribute_EachEntryHasCorrectFields(t *testing.T) {
	f := vacaFund()
	members := activeMembers(2, f.ID)
	balance := decimal.NewFromInt(500000)

	ledgerMock := &mockDistributeLedger{balance: balance}
	uc := newDistributeUC(&mockVacaRepo{}, &mockDistributeFundRepo{}, &mockDistributeMemberRepo{members: members}, ledgerMock)

	result, err := uc.Execute(context.Background(), f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for i, entry := range result.Entries {
		if entry.Direction != ledger.DirectionDebit {
			t.Errorf("entry[%d] direction = %s, want debit", i, entry.Direction)
		}
		if entry.Type != ledger.EntryTypePayout {
			t.Errorf("entry[%d] type = %s, want payout", i, entry.Type)
		}
		if entry.FundID != f.ID {
			t.Errorf("entry[%d] FundID mismatch", i)
		}
		if entry.UserID == nil {
			t.Errorf("entry[%d] UserID should be set", i)
		}
		if entry.ID == uuid.Nil {
			t.Errorf("entry[%d] ID should be set", i)
		}
	}
}

func TestDistribute_MarksFundCompleted(t *testing.T) {
	f := vacaFund()
	members := activeMembers(2, f.ID)
	ledgerMock := &mockDistributeLedger{balance: decimal.NewFromInt(200000)}
	fundsMock := &mockDistributeFundRepo{}
	uc := newDistributeUC(&mockVacaRepo{}, fundsMock, &mockDistributeMemberRepo{members: members}, ledgerMock)

	_, err := uc.Execute(context.Background(), f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if fundsMock.updated == nil || fundsMock.updated.Status != fund.FundStatusCompleted {
		t.Errorf("expected fund status = completed, got %v", fundsMock.updated)
	}
}

func TestDistribute_WrongFundType_ReturnsError(t *testing.T) {
	f := vacaFund()
	f.Type = fund.FundTypeCirculo
	uc := newDistributeUC(&mockVacaRepo{}, &mockDistributeFundRepo{}, &mockDistributeMemberRepo{}, &mockDistributeLedger{})

	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error for non-vaca fund type")
	}
}

func TestDistribute_FundNotActive_ReturnsError(t *testing.T) {
	f := vacaFund()
	f.Status = fund.FundStatusDraft
	uc := newDistributeUC(&mockVacaRepo{}, &mockDistributeFundRepo{}, &mockDistributeMemberRepo{}, &mockDistributeLedger{})

	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error for non-active fund")
	}
}

func TestDistribute_ZeroBalance_ReturnsError(t *testing.T) {
	f := vacaFund()
	members := activeMembers(3, f.ID)
	ledgerMock := &mockDistributeLedger{balance: decimal.Zero}
	uc := newDistributeUC(&mockVacaRepo{}, &mockDistributeFundRepo{}, &mockDistributeMemberRepo{members: members}, ledgerMock)

	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error for zero balance")
	}
}

func TestDistribute_NoActiveMembers_ReturnsError(t *testing.T) {
	f := vacaFund()
	// All members are inactive
	inactive := []*fund.FundMember{
		{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusLeft},
	}
	ledgerMock := &mockDistributeLedger{balance: decimal.NewFromInt(500000)}
	uc := newDistributeUC(&mockVacaRepo{}, &mockDistributeFundRepo{}, &mockDistributeMemberRepo{members: inactive}, ledgerMock)

	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error when no active members")
	}
}
