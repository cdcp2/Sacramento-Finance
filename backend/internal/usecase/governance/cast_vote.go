package governance

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/governance"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	ucpayment "github.com/sacramento-finance/backend/internal/usecase/payment"
	ucvaca "github.com/sacramento-finance/backend/internal/usecase/vaca"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
	"github.com/sacramento-finance/backend/pkg/money"
)

type CastVoteUseCase struct {
	proposals        governance.ProposalRepository
	votes            governance.VoteRepository
	funds            fund.Repository
	members          fund.MemberRepository
	payments         ucpayment.WritePaymentRepository
	generateSchedule *ucpayment.GenerateScheduleUseCase
	distributeVaca   *ucvaca.DistributeUseCase
}

func NewCastVoteUseCase(
	proposals governance.ProposalRepository,
	votes governance.VoteRepository,
	funds fund.Repository,
	members fund.MemberRepository,
	payments ucpayment.WritePaymentRepository,
	generateSchedule *ucpayment.GenerateScheduleUseCase,
	distributeVaca *ucvaca.DistributeUseCase,
) *CastVoteUseCase {
	return &CastVoteUseCase{
		proposals:        proposals,
		votes:            votes,
		funds:            funds,
		members:          members,
		payments:         payments,
		generateSchedule: generateSchedule,
		distributeVaca:   distributeVaca,
	}
}

type CastVoteResult struct {
	Vote     *governance.Vote     `json:"vote"`
	Proposal *governance.Proposal `json:"proposal"`
	Resolved bool                 `json:"resolved"`
	Approved bool                 `json:"approved"`
}

func (uc *CastVoteUseCase) Execute(
	ctx context.Context,
	proposalID uuid.UUID,
	memberID uuid.UUID,
	choice governance.VoteChoice,
	f *fund.Fund,
) (*CastVoteResult, error) {
	p, err := uc.proposals.GetByID(ctx, proposalID)
	if err != nil || p == nil {
		return nil, apperror.ErrNotFound
	}
	if p.FundID != f.ID {
		return nil, apperror.ErrNotFound
	}

	now := time.Now().UTC()

	// Lazy expiry: check deadline before accepting the vote
	if p.IsExpired(now) {
		p.Status = governance.ProposalStatusExpired
		_ = uc.proposals.UpdateStatus(ctx, p)
		return nil, apperror.New(410, "PROPOSAL_EXPIRED", "this proposal has expired")
	}

	if p.Status != governance.ProposalStatusOpen {
		return nil, apperror.New(409, "PROPOSAL_CLOSED",
			"this proposal is already "+string(p.Status))
	}

	// One vote per member
	existing, _ := uc.votes.GetByProposalAndMember(ctx, proposalID, memberID)
	if existing != nil {
		return nil, apperror.New(409, "ALREADY_VOTED", "you have already voted on this proposal")
	}

	v := &governance.Vote{
		ID:         idgen.New(),
		ProposalID: proposalID,
		MemberID:   memberID,
		Choice:     choice,
		CreatedAt:  now,
	}
	if err := uc.votes.Create(ctx, v); err != nil {
		return nil, err
	}

	// Update vote counter
	if err := uc.proposals.IncrementVotes(ctx, proposalID, choice); err != nil {
		return nil, err
	}

	// Re-fetch to get the counts that include the vote just cast
	p, err = uc.proposals.GetByID(ctx, proposalID)
	if err != nil || p == nil {
		return nil, apperror.ErrInternal
	}

	approved, resolved := p.ResolveResult(f.Rules.GovernanceType)

	result := &CastVoteResult{Vote: v, Proposal: p}

	if resolved {
		result.Resolved = true
		result.Approved = approved

		if approved {
			p.Status = governance.ProposalStatusApproved
			if execErr := uc.executeProposal(ctx, p, f); execErr != nil {
				// Log but don't fail the vote — proposal is approved even if execution has issues
				_ = execErr
			}
		} else {
			p.Status = governance.ProposalStatusRejected
		}

		if err := uc.proposals.UpdateStatus(ctx, p); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// executeProposal carries out the approved action for each proposal type.
func (uc *CastVoteUseCase) executeProposal(ctx context.Context, p *governance.Proposal, f *fund.Fund) error {
	switch p.Type {

	case governance.ProposalTypeActivateFund:
		if !f.CanTransitionTo(fund.FundStatusActive) {
			return nil
		}
		f.Status = fund.FundStatusActive
		f.UpdatedAt = time.Now().UTC()
		if err := uc.funds.Update(ctx, f); err != nil {
			return err
		}
		members, err := uc.members.ListByFund(ctx, f.ID)
		if err != nil {
			return err
		}
		return uc.generateSchedule.Execute(ctx, f, members)

	case governance.ProposalTypeChangeRules:
		var payload governance.ChangeRulesPayload
		if err := json.Unmarshal(p.Payload, &payload); err != nil {
			return err
		}
		if payload.ContributionAmount != "" {
			if amount, err := money.FromString(payload.ContributionAmount); err == nil {
				f.Rules.ContributionAmount = amount
			}
		}
		if payload.Frequency != "" {
			f.Rules.Frequency = payload.Frequency
		}
		if payload.TotalPeriods > 0 {
			f.Rules.TotalPeriods = payload.TotalPeriods
		}
		f.Rules.PenaltyEnabled = payload.PenaltyEnabled
		if payload.PenaltyType != "" {
			f.Rules.PenaltyType = payload.PenaltyType
		}
		if payload.PenaltyAmount != "" {
			if pa, err := money.FromString(payload.PenaltyAmount); err == nil {
				f.Rules.PenaltyAmount = pa
			}
		}
		f.Rules.GracePeriodDays = payload.GracePeriodDays
		f.UpdatedAt = time.Now().UTC()
		return uc.funds.Update(ctx, f)

	case governance.ProposalTypeWaivePayment:
		var payload governance.WaivePaymentPayload
		if err := json.Unmarshal(p.Payload, &payload); err != nil {
			return err
		}
		paymentID, err := uuid.Parse(payload.PaymentID)
		if err != nil {
			return err
		}
		payment, err := uc.payments.GetByID(ctx, paymentID)
		if err != nil || payment == nil {
			return apperror.ErrPaymentNotFound
		}
		payment.Status = ledger.PaymentStatusWaived
		payment.UpdatedAt = time.Now().UTC()
		return uc.payments.Update(ctx, payment)

	case governance.ProposalTypeRemoveMember:
		var payload governance.RemoveMemberPayload
		if err := json.Unmarshal(p.Payload, &payload); err != nil {
			return err
		}
		memberID, err := uuid.Parse(payload.MemberID)
		if err != nil {
			return err
		}
		member, err := uc.members.GetByID(ctx, memberID)
		if err != nil || member == nil {
			return nil
		}
		member.Status = fund.MemberStatusLeft
		return uc.members.Update(ctx, member)

	case governance.ProposalTypeCancelFund:
		if !f.CanTransitionTo(fund.FundStatusCancelled) {
			return nil
		}
		f.Status = fund.FundStatusCancelled
		f.UpdatedAt = time.Now().UTC()
		return uc.funds.Update(ctx, f)

	case governance.ProposalTypeChangeGovernance:
		var payload governance.ChangeGovernancePayload
		if err := json.Unmarshal(p.Payload, &payload); err != nil {
			return err
		}
		f.Rules.GovernanceType = payload.GovernanceType
		if payload.VotingDeadlineHours > 0 {
			f.Rules.VotingDeadlineHours = payload.VotingDeadlineHours
		}
		f.UpdatedAt = time.Now().UTC()
		return uc.funds.Update(ctx, f)

	case governance.ProposalTypeDistributeVaca:
		if uc.distributeVaca == nil {
			return apperror.ErrInternal
		}
		_, err := uc.distributeVaca.Execute(ctx, f)
		return err
	}

	return nil
}
