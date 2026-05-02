package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sacramento-finance/backend/internal/domain/governance"
)

type ProposalRepo struct {
	db *pgxpool.Pool
}

func NewProposalRepo(db *pgxpool.Pool) *ProposalRepo {
	return &ProposalRepo{db: db}
}

func (r *ProposalRepo) Create(ctx context.Context, p *governance.Proposal) error {
	payload, err := json.Marshal(p.Payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO proposals (
			id, fund_id, proposer_id, type, status, payload,
			votes_for, votes_against, quorum_needed, total_members,
			deadline_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`,
		p.ID, p.FundID, p.ProposerID, p.Type, p.Status, payload,
		p.VotesFor, p.VotesAgainst, p.QuorumNeeded, p.TotalMembers,
		p.DeadlineAt, p.CreatedAt,
	)
	return err
}

func (r *ProposalRepo) GetByID(ctx context.Context, id uuid.UUID) (*governance.Proposal, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, fund_id, proposer_id, type, status, payload,
		       votes_for, votes_against, quorum_needed, total_members,
		       deadline_at, resolved_at, created_at
		FROM proposals WHERE id = $1
	`, id)
	return scanProposal(row)
}

func (r *ProposalRepo) ListByFund(ctx context.Context, fundID uuid.UUID) ([]*governance.Proposal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, fund_id, proposer_id, type, status, payload,
		       votes_for, votes_against, quorum_needed, total_members,
		       deadline_at, resolved_at, created_at
		FROM proposals WHERE fund_id = $1
		ORDER BY created_at DESC
	`, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proposals []*governance.Proposal
	for rows.Next() {
		p, err := scanProposal(rows)
		if err != nil {
			return nil, err
		}
		proposals = append(proposals, p)
	}
	return proposals, rows.Err()
}

func (r *ProposalRepo) UpdateStatus(ctx context.Context, p *governance.Proposal) error {
	now := time.Now().UTC()
	p.ResolvedAt = &now
	_, err := r.db.Exec(ctx, `
		UPDATE proposals SET status=$1, resolved_at=$2 WHERE id=$3
	`, p.Status, p.ResolvedAt, p.ID)
	return err
}

// IncrementVotes atomically increments the vote counter for a proposal.
func (r *ProposalRepo) IncrementVotes(ctx context.Context, proposalID uuid.UUID, choice governance.VoteChoice) error {
	var col string
	if choice == governance.VoteYes {
		col = "votes_for"
	} else {
		col = "votes_against"
	}
	_, err := r.db.Exec(ctx,
		"UPDATE proposals SET "+col+" = "+col+" + 1 WHERE id = $1",
		proposalID,
	)
	return err
}

type VoteRepo struct {
	db *pgxpool.Pool
}

func NewVoteRepo(db *pgxpool.Pool) *VoteRepo {
	return &VoteRepo{db: db}
}

func (r *VoteRepo) Create(ctx context.Context, v *governance.Vote) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO votes (id, proposal_id, member_id, choice, created_at)
		VALUES ($1,$2,$3,$4,$5)
	`, v.ID, v.ProposalID, v.MemberID, v.Choice, v.CreatedAt)
	return err
}

func (r *VoteRepo) GetByProposalAndMember(ctx context.Context, proposalID, memberID uuid.UUID) (*governance.Vote, error) {
	v := &governance.Vote{}
	err := r.db.QueryRow(ctx, `
		SELECT id, proposal_id, member_id, choice, created_at
		FROM votes WHERE proposal_id=$1 AND member_id=$2
	`, proposalID, memberID).Scan(&v.ID, &v.ProposalID, &v.MemberID, &v.Choice, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return v, nil
}

func (r *VoteRepo) ListByProposal(ctx context.Context, proposalID uuid.UUID) ([]*governance.Vote, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, proposal_id, member_id, choice, created_at
		FROM votes WHERE proposal_id=$1 ORDER BY created_at ASC
	`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []*governance.Vote
	for rows.Next() {
		v := &governance.Vote{}
		if err := rows.Scan(&v.ID, &v.ProposalID, &v.MemberID, &v.Choice, &v.CreatedAt); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProposal(row rowScanner) (*governance.Proposal, error) {
	p := &governance.Proposal{}
	var rawPayload []byte
	err := row.Scan(
		&p.ID, &p.FundID, &p.ProposerID, &p.Type, &p.Status, &rawPayload,
		&p.VotesFor, &p.VotesAgainst, &p.QuorumNeeded, &p.TotalMembers,
		&p.DeadlineAt, &p.ResolvedAt, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	p.Payload = json.RawMessage(rawPayload)
	return p, nil
}
