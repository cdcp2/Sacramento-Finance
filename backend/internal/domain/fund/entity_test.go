package fund_test

import (
	"testing"

	"github.com/sacramento-finance/backend/internal/domain/fund"
)

func TestCanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		from    fund.FundStatus
		to      fund.FundStatus
		allowed bool
	}{
		{"draft→active", fund.FundStatusDraft, fund.FundStatusActive, true},
		{"draft→cancelled", fund.FundStatusDraft, fund.FundStatusCancelled, true},
		{"draft→completed", fund.FundStatusDraft, fund.FundStatusCompleted, false},
		{"active→paused", fund.FundStatusActive, fund.FundStatusPaused, true},
		{"active→completed", fund.FundStatusActive, fund.FundStatusCompleted, true},
		{"active→cancelled", fund.FundStatusActive, fund.FundStatusCancelled, true},
		{"active→draft", fund.FundStatusActive, fund.FundStatusDraft, false},
		{"paused→active", fund.FundStatusPaused, fund.FundStatusActive, true},
		{"paused→cancelled", fund.FundStatusPaused, fund.FundStatusCancelled, true},
		{"paused→completed", fund.FundStatusPaused, fund.FundStatusCompleted, false},
		{"completed→anything", fund.FundStatusCompleted, fund.FundStatusActive, false},
		{"cancelled→anything", fund.FundStatusCancelled, fund.FundStatusActive, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &fund.Fund{Status: tc.from}
			got := f.CanTransitionTo(tc.to)
			if got != tc.allowed {
				t.Errorf("CanTransitionTo(%s→%s) = %v, want %v", tc.from, tc.to, got, tc.allowed)
			}
		})
	}
}

func TestIsGoverned(t *testing.T) {
	tests := []struct {
		governance fund.GovernanceType
		governed   bool
	}{
		{fund.GovernanceAdminOnly, false},
		{fund.GovernanceMajority, true},
		{fund.GovernanceUnanimous, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.governance), func(t *testing.T) {
			r := &fund.Rules{GovernanceType: tc.governance}
			if r.IsGoverned() != tc.governed {
				t.Errorf("IsGoverned() with %s = %v, want %v", tc.governance, r.IsGoverned(), tc.governed)
			}
		})
	}
}
