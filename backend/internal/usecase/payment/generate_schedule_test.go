package payment

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/shopspring/decimal"
)

// mockBatchRepo captures created payments for assertion.
type mockBatchRepo struct {
	created []*ledger.Payment
}

func (m *mockBatchRepo) CreateBatch(_ context.Context, payments []*ledger.Payment) error {
	m.created = append(m.created, payments...)
	return nil
}

func TestDueDate(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		freq     fund.PeriodFrequency
		period   int
		wantDate time.Time
	}{
		{fund.FrequencyWeekly, 1, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{fund.FrequencyWeekly, 2, time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC)},
		{fund.FrequencyWeekly, 3, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)},
		{fund.FrequencyBiweekly, 1, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{fund.FrequencyBiweekly, 2, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)},
		{fund.FrequencyBiweekly, 3, time.Date(2025, 1, 29, 0, 0, 0, 0, time.UTC)},
		{fund.FrequencyMonthly, 1, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{fund.FrequencyMonthly, 2, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
		{fund.FrequencyMonthly, 12, time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		got := dueDate(start, tc.freq, tc.period)
		if !got.Equal(tc.wantDate) {
			t.Errorf("dueDate(%s, period=%d) = %v, want %v", tc.freq, tc.period, got, tc.wantDate)
		}
	}
}

func TestGenerateSchedule_PaymentCount(t *testing.T) {
	repo := &mockBatchRepo{}
	uc := NewGenerateScheduleUseCase(repo)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	f := &fund.Fund{
		ID: uuid.New(),
		Rules: fund.Rules{
			ContributionAmount: decimal.NewFromInt(100000),
			Frequency:          fund.FrequencyMonthly,
			TotalPeriods:       6,
			StartDate:          start,
		},
	}

	memberA   := &fund.FundMember{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusActive}
	memberB   := &fund.FundMember{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusActive}
	suspended := &fund.FundMember{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusSuspended}

	err := uc.Execute(context.Background(), f, []*fund.FundMember{memberA, memberB, suspended})
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	// 2 active members × 6 periods = 12; suspended member is skipped
	if len(repo.created) != 12 {
		t.Errorf("expected 12 payments, got %d", len(repo.created))
	}
}

func TestGenerateSchedule_DueDates(t *testing.T) {
	repo := &mockBatchRepo{}
	uc := NewGenerateScheduleUseCase(repo)

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	f := &fund.Fund{
		ID: uuid.New(),
		Rules: fund.Rules{
			ContributionAmount: decimal.NewFromInt(50000),
			Frequency:          fund.FrequencyMonthly,
			TotalPeriods:       3,
			StartDate:          start,
		},
	}
	member := &fund.FundMember{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusActive}

	_ = uc.Execute(context.Background(), f, []*fund.FundMember{member})

	wantDates := []time.Time{
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	for i, p := range repo.created {
		if !p.DueDate.Equal(wantDates[i]) {
			t.Errorf("payment[%d].DueDate = %v, want %v", i, p.DueDate, wantDates[i])
		}
		if p.PeriodNumber != i+1 {
			t.Errorf("payment[%d].PeriodNumber = %d, want %d", i, p.PeriodNumber, i+1)
		}
	}
}

func TestGenerateSchedule_EmptyWhenNoActiveMembers(t *testing.T) {
	repo := &mockBatchRepo{}
	uc := NewGenerateScheduleUseCase(repo)

	f    := &fund.Fund{ID: uuid.New(), Rules: fund.Rules{TotalPeriods: 5}}
	left := &fund.FundMember{ID: uuid.New(), Status: fund.MemberStatusLeft}

	err := uc.Execute(context.Background(), f, []*fund.FundMember{left})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.created) != 0 {
		t.Errorf("expected 0 payments for non-active members, got %d", len(repo.created))
	}
}
