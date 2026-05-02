package governance

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
)

type ProposalType string

const (
	ProposalTypeActivateFund     ProposalType = "activate_fund"
	ProposalTypeChangeRules      ProposalType = "change_rules"
	ProposalTypeWaivePayment     ProposalType = "waive_payment"
	ProposalTypeRemoveMember     ProposalType = "remove_member"
	ProposalTypeCancelFund       ProposalType = "cancel_fund"
	ProposalTypeChangeGovernance ProposalType = "change_governance"
	ProposalTypeDistributeVaca   ProposalType = "distribute_vaca"
)

type ProposalStatus string

const (
	ProposalStatusOpen     ProposalStatus = "open"
	ProposalStatusApproved ProposalStatus = "approved"
	ProposalStatusRejected ProposalStatus = "rejected"
	ProposalStatusExpired  ProposalStatus = "expired"
)

type VoteChoice string

const (
	VoteYes VoteChoice = "yes"
	VoteNo  VoteChoice = "no"
)

// Proposal represents a governance action pending a vote.
type Proposal struct {
	ID           uuid.UUID       `json:"id"`
	FundID       uuid.UUID       `json:"fund_id"`
	ProposerID   uuid.UUID       `json:"proposer_id"` // fund_member.id
	Type         ProposalType    `json:"type"`
	Status       ProposalStatus  `json:"status"`
	Payload      json.RawMessage `json:"payload"`
	VotesFor     int             `json:"votes_for"`
	VotesAgainst int             `json:"votes_against"`
	QuorumNeeded int             `json:"quorum_needed"` // minimum votes required for result to be valid
	TotalMembers int             `json:"total_members"` // snapshot of active members at creation
	DeadlineAt   time.Time       `json:"deadline_at"`
	ResolvedAt   *time.Time      `json:"resolved_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Vote is an individual member's vote on a proposal.
type Vote struct {
	ID         uuid.UUID  `json:"id"`
	ProposalID uuid.UUID  `json:"proposal_id"`
	MemberID   uuid.UUID  `json:"member_id"`
	Choice     VoteChoice `json:"choice"`
	CreatedAt  time.Time  `json:"created_at"`
}

// IsExpired returns true if the deadline has passed and the proposal is still open.
func (p *Proposal) IsExpired(now time.Time) bool {
	return p.Status == ProposalStatusOpen && now.After(p.DeadlineAt)
}

// TotalVotes returns the number of votes cast so far.
func (p *Proposal) TotalVotes() int {
	return p.VotesFor + p.VotesAgainst
}

// QuorumMet returns true if enough members have voted.
func (p *Proposal) QuorumMet() bool {
	return p.TotalVotes() >= p.QuorumNeeded
}

// ResolveResult computes the outcome after a new vote, given the fund's governance type.
// Returns (approved, resolved) — resolved=true means the proposal can be closed now.
func (p *Proposal) ResolveResult(governance fund.GovernanceType) (approved bool, resolved bool) {
	half := p.TotalMembers / 2

	switch governance {
	case fund.GovernanceUnanimous:
		if p.VotesAgainst > 0 {
			// Any "no" ends it immediately
			return false, true
		}
		if p.VotesFor == p.TotalMembers {
			// Everyone voted yes
			return true, true
		}

	case fund.GovernanceMajority:
		if p.VotesFor > half {
			// Majority secured — remaining votes can't change outcome
			return true, true
		}
		if p.VotesAgainst > half {
			// Rejection secured
			return false, true
		}
		if p.TotalVotes() == p.TotalMembers {
			// Everyone voted, resolve by count
			return p.VotesFor > p.VotesAgainst, true
		}
	}

	return false, false
}

// --- Typed payloads for each proposal type ---

type ChangeRulesPayload struct {
	ContributionAmount string               `json:"contribution_amount,omitempty"`
	Frequency          fund.PeriodFrequency `json:"frequency,omitempty"`
	TotalPeriods       int                  `json:"total_periods,omitempty"`
	PenaltyEnabled     bool                 `json:"penalty_enabled"`
	PenaltyType        fund.PenaltyType     `json:"penalty_type,omitempty"`
	PenaltyAmount      string               `json:"penalty_amount,omitempty"`
	GracePeriodDays    int                  `json:"grace_period_days"`
}

type WaivePaymentPayload struct {
	PaymentID string `json:"payment_id"`
}

type RemoveMemberPayload struct {
	MemberID string `json:"member_id"`
	Reason   string `json:"reason"`
}

type ChangeGovernancePayload struct {
	GovernanceType      fund.GovernanceType `json:"governance_type"`
	VotingDeadlineHours int                 `json:"voting_deadline_hours"`
}

type CancelFundPayload struct {
	Reason string `json:"reason"`
}
