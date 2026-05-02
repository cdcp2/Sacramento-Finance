package governance

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/governance"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
)

type CreateProposalUseCase struct {
	proposals governance.ProposalRepository
	members   fund.MemberRepository
}

func NewCreateProposalUseCase(
	proposals governance.ProposalRepository,
	members fund.MemberRepository,
) *CreateProposalUseCase {
	return &CreateProposalUseCase{proposals: proposals, members: members}
}

func (uc *CreateProposalUseCase) Execute(
	ctx context.Context,
	f *fund.Fund,
	proposerMemberID uuid.UUID,
	proposalType governance.ProposalType,
	payload any,
) (*governance.Proposal, error) {
	if !f.Rules.IsGoverned() {
		return nil, apperror.New(400, "NOT_GOVERNED",
			"this fund uses admin_only governance; use direct admin actions instead")
	}

	activeCount, err := uc.members.CountActive(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	if activeCount < 2 {
		return nil, apperror.New(400, "NOT_ENOUGH_MEMBERS",
			"need at least 2 active members to create proposals")
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, apperror.ErrInternal
	}

	deadlineHours := f.Rules.VotingDeadlineHours
	if deadlineHours <= 0 {
		deadlineHours = 72
	}

	now := time.Now().UTC()
	p := &governance.Proposal{
		ID:           idgen.New(),
		FundID:       f.ID,
		ProposerID:   proposerMemberID,
		Type:         proposalType,
		Status:       governance.ProposalStatusOpen,
		Payload:      json.RawMessage(rawPayload),
		VotesFor:     0,
		VotesAgainst: 0,
		QuorumNeeded: quorumNeeded(f.Rules.GovernanceType, activeCount),
		TotalMembers: activeCount,
		DeadlineAt:   now.Add(time.Duration(deadlineHours) * time.Hour),
		CreatedAt:    now,
	}

	if err := uc.proposals.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// quorumNeeded is the minimum number of votes for a result to be valid.
// Unanimous: all members. Majority: ceil(n/2).
func quorumNeeded(g fund.GovernanceType, total int) int {
	if g == fund.GovernanceUnanimous {
		return total
	}
	return (total + 1) / 2
}
