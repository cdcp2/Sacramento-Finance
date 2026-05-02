package governance_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/governance"
)

func proposal(totalMembers, votesFor, votesAgainst int) *governance.Proposal {
	return &governance.Proposal{
		ID:           uuid.New(),
		FundID:       uuid.New(),
		Status:       governance.ProposalStatusOpen,
		TotalMembers: totalMembers,
		QuorumNeeded: (totalMembers + 1) / 2,
		VotesFor:     votesFor,
		VotesAgainst: votesAgainst,
		DeadlineAt:   time.Now().Add(48 * time.Hour),
	}
}

func TestResolveResult_Majority(t *testing.T) {
	tests := []struct {
		name         string
		total, for_, against int
		wantApproved bool
		wantResolved bool
	}{
		{"3 members, 2 yes → approved", 3, 2, 0, true, true},
		{"3 members, 2 no → rejected", 3, 0, 2, false, true},
		{"3 members, 1 yes, 1 no → not resolved", 3, 1, 1, false, false},
		{"4 members, 3 yes → approved", 4, 3, 0, true, true},
		{"4 members, 2 yes, 2 no, all voted → approved by count", 4, 2, 2, false, true},
		{"5 members, 3 yes → approved (irresolvable)", 5, 3, 0, true, true},
		{"5 members, 3 no → rejected", 5, 0, 3, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := proposal(tc.total, tc.for_, tc.against)
			approved, resolved := p.ResolveResult(fund.GovernanceMajority)
			if approved != tc.wantApproved || resolved != tc.wantResolved {
				t.Errorf("ResolveResult() = (%v, %v), want (%v, %v)",
					approved, resolved, tc.wantApproved, tc.wantResolved)
			}
		})
	}
}

func TestResolveResult_Unanimous(t *testing.T) {
	tests := []struct {
		name         string
		total, for_, against int
		wantApproved bool
		wantResolved bool
	}{
		{"all yes → approved", 4, 4, 0, true, true},
		{"one no → rejected immediately", 4, 3, 1, false, true},
		{"partial yes, no no → not resolved", 4, 2, 0, false, false},
		{"no votes yet → not resolved", 4, 0, 0, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := proposal(tc.total, tc.for_, tc.against)
			approved, resolved := p.ResolveResult(fund.GovernanceUnanimous)
			if approved != tc.wantApproved || resolved != tc.wantResolved {
				t.Errorf("ResolveResult() = (%v, %v), want (%v, %v)",
					approved, resolved, tc.wantApproved, tc.wantResolved)
			}
		})
	}
}

func TestProposal_IsExpired(t *testing.T) {
	now := time.Now().UTC()

	p := &governance.Proposal{
		Status:     governance.ProposalStatusOpen,
		DeadlineAt: now.Add(-1 * time.Hour), // 1 hour ago
	}
	if !p.IsExpired(now) {
		t.Error("expected expired proposal to be detected")
	}

	p.DeadlineAt = now.Add(1 * time.Hour) // 1 hour from now
	if p.IsExpired(now) {
		t.Error("expected non-expired proposal to not be expired")
	}

	p.DeadlineAt = now.Add(-1 * time.Hour)
	p.Status = governance.ProposalStatusApproved
	if p.IsExpired(now) {
		t.Error("approved proposal should never be considered expired")
	}
}

func TestProposal_QuorumMet(t *testing.T) {
	p := &governance.Proposal{QuorumNeeded: 3, VotesFor: 2, VotesAgainst: 0}
	if p.QuorumMet() {
		t.Error("quorum should not be met with 2 of 3 required")
	}
	p.VotesFor = 3
	if !p.QuorumMet() {
		t.Error("quorum should be met with 3 of 3")
	}
}
