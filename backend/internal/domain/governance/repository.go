package governance

import (
	"context"

	"github.com/google/uuid"
)

type ProposalRepository interface {
	Create(ctx context.Context, p *Proposal) error
	GetByID(ctx context.Context, id uuid.UUID) (*Proposal, error)
	ListByFund(ctx context.Context, fundID uuid.UUID) ([]*Proposal, error)
	UpdateStatus(ctx context.Context, p *Proposal) error
	IncrementVotes(ctx context.Context, proposalID uuid.UUID, choice VoteChoice) error
}

type VoteRepository interface {
	Create(ctx context.Context, v *Vote) error
	GetByProposalAndMember(ctx context.Context, proposalID, memberID uuid.UUID) (*Vote, error)
	ListByProposal(ctx context.Context, proposalID uuid.UUID) ([]*Vote, error)
}
