package governance_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/governance"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	ucgovernance "github.com/sacramento-finance/backend/internal/usecase/governance"
	ucpayment "github.com/sacramento-finance/backend/internal/usecase/payment"
	"github.com/shopspring/decimal"
)

// ─── mocks ────────────────────────────────────────────────────────────────────

type mockProposalRepo struct {
	stored    *governance.Proposal
	createErr error
	updated   *governance.Proposal
	updateErr error
	incErr    error
	incCount  int
}

func (m *mockProposalRepo) Create(_ context.Context, p *governance.Proposal) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.stored = p
	return nil
}
func (m *mockProposalRepo) GetByID(_ context.Context, _ uuid.UUID) (*governance.Proposal, error) {
	return m.stored, nil
}
func (m *mockProposalRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*governance.Proposal, error) {
	if m.stored == nil {
		return nil, nil
	}
	return []*governance.Proposal{m.stored}, nil
}
func (m *mockProposalRepo) UpdateStatus(_ context.Context, p *governance.Proposal) error {
	m.updated = p
	return m.updateErr
}
func (m *mockProposalRepo) IncrementVotes(_ context.Context, _ uuid.UUID, choice governance.VoteChoice) error {
	if m.incErr != nil {
		return m.incErr
	}
	m.incCount++
	if m.stored != nil {
		if choice == governance.VoteYes {
			m.stored.VotesFor++
		} else {
			m.stored.VotesAgainst++
		}
	}
	return nil
}

type mockVoteRepo struct {
	existing  *governance.Vote
	created   *governance.Vote
	createErr error
}

func (m *mockVoteRepo) Create(_ context.Context, v *governance.Vote) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = v
	return nil
}
func (m *mockVoteRepo) GetByProposalAndMember(_ context.Context, _, _ uuid.UUID) (*governance.Vote, error) {
	return m.existing, nil
}
func (m *mockVoteRepo) ListByProposal(_ context.Context, _ uuid.UUID) ([]*governance.Vote, error) {
	return nil, nil
}

type mockFundRepo struct {
	f       *fund.Fund
	updated *fund.Fund
}

func (m *mockFundRepo) Create(_ context.Context, _ *fund.Fund) error { return nil }
func (m *mockFundRepo) GetByID(_ context.Context, _ uuid.UUID) (*fund.Fund, error) {
	return m.f, nil
}
func (m *mockFundRepo) ListByMember(_ context.Context, _ uuid.UUID) ([]*fund.Fund, error) {
	return nil, nil
}
func (m *mockFundRepo) Update(_ context.Context, f *fund.Fund) error {
	m.updated = f
	return nil
}

type mockMemberRepo struct {
	members     []*fund.FundMember
	byID        *fund.FundMember
	activeCount int
	updated     *fund.FundMember
}

func (m *mockMemberRepo) Add(_ context.Context, _ *fund.FundMember) error { return nil }
func (m *mockMemberRepo) GetByFundAndUser(_ context.Context, _, _ uuid.UUID) (*fund.FundMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) GetByID(_ context.Context, _ uuid.UUID) (*fund.FundMember, error) {
	return m.byID, nil
}
func (m *mockMemberRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*fund.FundMember, error) {
	return m.members, nil
}
func (m *mockMemberRepo) Update(_ context.Context, mem *fund.FundMember) error {
	m.updated = mem
	return nil
}
func (m *mockMemberRepo) CountActive(_ context.Context, _ uuid.UUID) (int, error) {
	return m.activeCount, nil
}

type mockPaymentWriteRepo struct {
	payment           *ledger.Payment
	updated           *ledger.Payment
	getErr            error
	createBatchCalled bool
}

func (m *mockPaymentWriteRepo) GetByID(_ context.Context, _ uuid.UUID) (*ledger.Payment, error) {
	return m.payment, m.getErr
}
func (m *mockPaymentWriteRepo) Update(_ context.Context, p *ledger.Payment) error {
	m.updated = p
	return nil
}
func (m *mockPaymentWriteRepo) CreateBatch(_ context.Context, _ []*ledger.Payment) error {
	m.createBatchCalled = true
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func majorityFund() *fund.Fund {
	return &fund.Fund{
		ID:     uuid.New(),
		Type:   fund.FundTypeCirculo,
		Status: fund.FundStatusActive,
		Rules: fund.Rules{
			ContributionAmount:  decimal.NewFromInt(100000),
			TotalPeriods:        3,
			Frequency:           fund.FrequencyMonthly,
			StartDate:           time.Now().UTC(),
			GovernanceType:      fund.GovernanceMajority,
			VotingDeadlineHours: 72,
		},
	}
}

func unanimousFund() *fund.Fund {
	f := majorityFund()
	f.Rules.GovernanceType = fund.GovernanceUnanimous
	return f
}

func openProposal(fundID uuid.UUID, pType governance.ProposalType, totalMembers int) *governance.Proposal {
	payload, _ := json.Marshal(map[string]any{})
	return &governance.Proposal{
		ID:           uuid.New(),
		FundID:       fundID,
		ProposerID:   uuid.New(),
		Type:         pType,
		Status:       governance.ProposalStatusOpen,
		Payload:      payload,
		VotesFor:     0,
		VotesAgainst: 0,
		QuorumNeeded: (totalMembers + 1) / 2,
		TotalMembers: totalMembers,
		DeadlineAt:   time.Now().UTC().Add(72 * time.Hour),
		CreatedAt:    time.Now().UTC(),
	}
}

func newCastVoteUC(proposals *mockProposalRepo, votes *mockVoteRepo, funds *mockFundRepo, members *mockMemberRepo, payments *mockPaymentWriteRepo) *ucgovernance.CastVoteUseCase {
	genSchedule := ucpayment.NewGenerateScheduleUseCase(payments)
	return ucgovernance.NewCastVoteUseCase(proposals, votes, funds, members, payments, genSchedule)
}

// ─── CreateProposal tests ─────────────────────────────────────────────────────

func TestCreateProposal_Majority_CreatesWithCorrectQuorum(t *testing.T) {
	f := majorityFund()
	proposalRepo := &mockProposalRepo{}
	memberRepo := &mockMemberRepo{activeCount: 4}
	uc := ucgovernance.NewCreateProposalUseCase(proposalRepo, memberRepo)

	proposerID := uuid.New()
	p, err := uc.Execute(context.Background(), f, proposerID, governance.ProposalTypeActivateFund, map[string]any{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// ceil(4/2) = 2
	if p.QuorumNeeded != 2 {
		t.Errorf("QuorumNeeded = %d, want 2", p.QuorumNeeded)
	}
	if p.TotalMembers != 4 {
		t.Errorf("TotalMembers = %d, want 4", p.TotalMembers)
	}
	if p.Status != governance.ProposalStatusOpen {
		t.Errorf("Status = %s, want open", p.Status)
	}
	if p.DeadlineAt.IsZero() {
		t.Error("DeadlineAt should be set")
	}
}

func TestCreateProposal_Unanimous_QuorumEqualsTotal(t *testing.T) {
	f := unanimousFund()
	memberRepo := &mockMemberRepo{activeCount: 5}
	uc := ucgovernance.NewCreateProposalUseCase(&mockProposalRepo{}, memberRepo)

	p, err := uc.Execute(context.Background(), f, uuid.New(), governance.ProposalTypeCancelFund, map[string]any{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if p.QuorumNeeded != 5 {
		t.Errorf("QuorumNeeded = %d, want 5 (unanimous)", p.QuorumNeeded)
	}
}

func TestCreateProposal_AdminOnly_ReturnsError(t *testing.T) {
	f := majorityFund()
	f.Rules.GovernanceType = fund.GovernanceAdminOnly
	uc := ucgovernance.NewCreateProposalUseCase(&mockProposalRepo{}, &mockMemberRepo{activeCount: 3})

	_, err := uc.Execute(context.Background(), f, uuid.New(), governance.ProposalTypeActivateFund, map[string]any{})
	if err == nil {
		t.Error("expected error for admin_only fund")
	}
}

func TestCreateProposal_TooFewMembers_ReturnsError(t *testing.T) {
	f := majorityFund()
	uc := ucgovernance.NewCreateProposalUseCase(&mockProposalRepo{}, &mockMemberRepo{activeCount: 1})

	_, err := uc.Execute(context.Background(), f, uuid.New(), governance.ProposalTypeActivateFund, map[string]any{})
	if err == nil {
		t.Error("expected error with only 1 active member")
	}
}

func TestCreateProposal_DefaultsDeadlineWhenNotConfigured(t *testing.T) {
	f := majorityFund()
	f.Rules.VotingDeadlineHours = 0
	uc := ucgovernance.NewCreateProposalUseCase(&mockProposalRepo{}, &mockMemberRepo{activeCount: 3})

	before := time.Now().UTC()
	p, err := uc.Execute(context.Background(), f, uuid.New(), governance.ProposalTypeActivateFund, map[string]any{})
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	minDeadline := before.Add(72 * time.Hour)
	maxDeadline := after.Add(72 * time.Hour)
	if p.DeadlineAt.Before(minDeadline) || p.DeadlineAt.After(maxDeadline) {
		t.Errorf("DeadlineAt = %s, want within default 72h window [%s, %s]",
			p.DeadlineAt, minDeadline, maxDeadline)
	}
}

func TestCreateProposal_RepositoryCreateError_ReturnsError(t *testing.T) {
	f := majorityFund()
	wantErr := errors.New("create proposal failed")
	uc := ucgovernance.NewCreateProposalUseCase(
		&mockProposalRepo{createErr: wantErr},
		&mockMemberRepo{activeCount: 3},
	)

	_, err := uc.Execute(context.Background(), f, uuid.New(), governance.ProposalTypeActivateFund, map[string]any{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

// ─── CastVote — basic flow ────────────────────────────────────────────────────

func TestCastVote_FirstVote_DoesNotResolveYet(t *testing.T) {
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 4)
	proposalRepo := &mockProposalRepo{stored: p}
	payments := &mockPaymentWriteRepo{}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, &mockFundRepo{f: f}, &mockMemberRepo{}, payments)

	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Resolved {
		t.Error("should not be resolved after first vote of 4 members (majority)")
	}
	if result.Vote == nil {
		t.Error("Vote should be set in result")
	}
}

func TestCastVote_ProposalNotFound_ReturnsError(t *testing.T) {
	f := majorityFund()
	uc := newCastVoteUC(
		&mockProposalRepo{},
		&mockVoteRepo{},
		&mockFundRepo{f: f},
		&mockMemberRepo{},
		&mockPaymentWriteRepo{},
	)

	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), governance.VoteYes, f)
	if err == nil {
		t.Fatal("expected error for missing proposal")
	}
}

func TestCastVote_ProposalFromOtherFund_ReturnsError(t *testing.T) {
	f := majorityFund()
	p := openProposal(uuid.New(), governance.ProposalTypeActivateFund, 3)
	uc := newCastVoteUC(
		&mockProposalRepo{stored: p},
		&mockVoteRepo{},
		&mockFundRepo{f: f},
		&mockMemberRepo{},
		&mockPaymentWriteRepo{},
	)

	_, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err == nil {
		t.Fatal("expected error for proposal from another fund")
	}
}

func TestCastVote_CreateVoteError_DoesNotIncrementVoteCounters(t *testing.T) {
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 3)
	proposalRepo := &mockProposalRepo{stored: p}
	wantErr := errors.New("insert vote failed")
	voteRepo := &mockVoteRepo{createErr: wantErr}

	uc := newCastVoteUC(proposalRepo, voteRepo, &mockFundRepo{f: f}, &mockMemberRepo{}, &mockPaymentWriteRepo{})

	_, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if proposalRepo.incCount != 0 {
		t.Errorf("IncrementVotes called %d times, want 0", proposalRepo.incCount)
	}
}

func TestCastVote_DuplicateVote_ReturnsError(t *testing.T) {
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 3)
	memberID := uuid.New()

	uc := newCastVoteUC(
		&mockProposalRepo{stored: p},
		&mockVoteRepo{existing: &governance.Vote{ID: uuid.New(), MemberID: memberID}},
		&mockFundRepo{f: f},
		&mockMemberRepo{},
		&mockPaymentWriteRepo{},
	)

	_, err := uc.Execute(context.Background(), p.ID, memberID, governance.VoteYes, f)
	if err == nil {
		t.Error("expected error for duplicate vote")
	}
}

func TestCastVote_ExpiredProposal_ReturnsError(t *testing.T) {
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 3)
	p.DeadlineAt = time.Now().UTC().Add(-1 * time.Hour) // in the past

	uc := newCastVoteUC(
		&mockProposalRepo{stored: p},
		&mockVoteRepo{},
		&mockFundRepo{f: f},
		&mockMemberRepo{},
		&mockPaymentWriteRepo{},
	)

	_, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err == nil {
		t.Error("expected error for expired proposal")
	}
}

func TestCastVote_ClosedProposal_ReturnsError(t *testing.T) {
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 3)
	p.Status = governance.ProposalStatusApproved

	uc := newCastVoteUC(
		&mockProposalRepo{stored: p},
		&mockVoteRepo{},
		&mockFundRepo{f: f},
		&mockMemberRepo{},
		&mockPaymentWriteRepo{},
	)

	_, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err == nil {
		t.Error("expected error for already-closed proposal")
	}
}

// ─── CastVote — majority resolution ──────────────────────────────────────────

func TestCastVote_Majority_ApprovesWhenMoreThanHalf(t *testing.T) {
	// 3 members, majority needs >1 yes (2 yes wins)
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeCancelFund, 3)
	p.VotesFor = 1 // one yes already cast
	proposalRepo := &mockProposalRepo{stored: p}
	fundRepo := &mockFundRepo{f: f}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, fundRepo, &mockMemberRepo{}, &mockPaymentWriteRepo{})

	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Resolved {
		t.Error("should be resolved after 2 yes out of 3")
	}
	if !result.Approved {
		t.Error("should be approved")
	}
	// cancel_fund execution should update fund status
	if fundRepo.updated == nil || fundRepo.updated.Status != fund.FundStatusCancelled {
		t.Errorf("expected fund status = cancelled, got %v", fundRepo.updated)
	}
}

func TestCastVote_ApprovedProposal_StatusSetToApproved(t *testing.T) {
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeCancelFund, 3)
	p.VotesFor = 1
	proposalRepo := &mockProposalRepo{stored: p}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, &mockFundRepo{f: f}, &mockMemberRepo{}, &mockPaymentWriteRepo{})

	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved result")
	}
	if proposalRepo.updated == nil || proposalRepo.updated.Status != governance.ProposalStatusApproved {
		t.Errorf("expected proposal status = approved, got %v", proposalRepo.updated)
	}
}

func TestCastVote_Majority_RejectsWhenMoreThanHalfNo(t *testing.T) {
	// 3 members: 0 yes, 2 no → rejected
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 3)
	p.VotesAgainst = 1
	proposalRepo := &mockProposalRepo{stored: p}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, &mockFundRepo{f: f}, &mockMemberRepo{}, &mockPaymentWriteRepo{})

	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteNo, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Resolved {
		t.Error("should be resolved")
	}
	if result.Approved {
		t.Error("should be rejected, not approved")
	}
}

func TestCastVote_Majority_TieResolvedByCount(t *testing.T) {
	// 2 members: 1 yes, 1 no → all voted, yes !> no → rejected
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 2)
	p.VotesFor = 1
	proposalRepo := &mockProposalRepo{stored: p}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, &mockFundRepo{f: f}, &mockMemberRepo{}, &mockPaymentWriteRepo{})

	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteNo, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Resolved {
		t.Error("should be resolved when all members have voted")
	}
	if result.Approved {
		t.Error("1 yes vs 1 no should not be approved (no majority)")
	}
}

// ─── CastVote — unanimous resolution ─────────────────────────────────────────

func TestCastVote_Unanimous_ApprovesWhenAllYes(t *testing.T) {
	f := unanimousFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 2)
	p.VotesFor = 1
	proposalRepo := &mockProposalRepo{stored: p}
	fundRepo := &mockFundRepo{f: f}
	members := &mockMemberRepo{members: []*fund.FundMember{}}
	payments := &mockPaymentWriteRepo{}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, fundRepo, members, payments)

	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Resolved || !result.Approved {
		t.Errorf("resolved=%v approved=%v; want both true", result.Resolved, result.Approved)
	}
}

func TestCastVote_Unanimous_RejectsOnFirstNo(t *testing.T) {
	f := unanimousFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 5)
	p.VotesFor = 3 // 3 yes already, one no kills it
	proposalRepo := &mockProposalRepo{stored: p}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, &mockFundRepo{f: f}, &mockMemberRepo{}, &mockPaymentWriteRepo{})

	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteNo, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Resolved {
		t.Error("should be resolved immediately on first no (unanimous)")
	}
	if result.Approved {
		t.Error("should be rejected")
	}
}

// ─── executeProposal — per type ───────────────────────────────────────────────

func TestCastVote_ActivateFund_GeneratesSchedule(t *testing.T) {
	f := majorityFund()
	f.Status = fund.FundStatusDraft
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 2)
	p.VotesFor = 1
	proposalRepo := &mockProposalRepo{stored: p}
	fundRepo := &mockFundRepo{f: f}
	members := &mockMemberRepo{members: []*fund.FundMember{
		{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusActive},
		{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusActive},
	}}
	payments := &mockPaymentWriteRepo{}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, fundRepo, members, payments)
	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved")
	}
	if fundRepo.updated == nil || fundRepo.updated.Status != fund.FundStatusActive {
		t.Errorf("expected fund status = active, got %v", fundRepo.updated)
	}
	if !payments.createBatchCalled {
		t.Error("expected payment schedule to be generated on fund activation")
	}
}

func TestCastVote_ChangeRules_UpdatesFundRules(t *testing.T) {
	f := majorityFund()
	payload, _ := json.Marshal(governance.ChangeRulesPayload{
		ContributionAmount: "200000",
		TotalPeriods:       6,
	})
	p := openProposal(f.ID, governance.ProposalTypeChangeRules, 2)
	p.VotesFor = 1
	p.Payload = payload
	proposalRepo := &mockProposalRepo{stored: p}
	fundRepo := &mockFundRepo{f: f}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, fundRepo, &mockMemberRepo{}, &mockPaymentWriteRepo{})
	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved")
	}
	if fundRepo.updated == nil {
		t.Fatal("expected fund to be updated")
	}
	want := decimal.NewFromInt(200000)
	if !fundRepo.updated.Rules.ContributionAmount.Equal(want) {
		t.Errorf("ContributionAmount = %s, want %s", fundRepo.updated.Rules.ContributionAmount, want)
	}
	if fundRepo.updated.Rules.TotalPeriods != 6 {
		t.Errorf("TotalPeriods = %d, want 6", fundRepo.updated.Rules.TotalPeriods)
	}
}

func TestCastVote_WaivePayment_SetsPaymentWaived(t *testing.T) {
	f := majorityFund()
	paymentID := uuid.New()
	payment := &ledger.Payment{
		ID:     paymentID,
		FundID: f.ID,
		Status: ledger.PaymentStatusPending,
	}
	payload, _ := json.Marshal(governance.WaivePaymentPayload{PaymentID: paymentID.String()})
	p := openProposal(f.ID, governance.ProposalTypeWaivePayment, 2)
	p.VotesFor = 1
	p.Payload = payload
	proposalRepo := &mockProposalRepo{stored: p}
	payments := &mockPaymentWriteRepo{payment: payment}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, &mockFundRepo{f: f}, &mockMemberRepo{}, payments)
	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved")
	}
	if payments.updated == nil || payments.updated.Status != ledger.PaymentStatusWaived {
		t.Errorf("expected payment status = waived, got %v", payments.updated)
	}
}

func TestCastVote_RemoveMember_SetsMemberLeft(t *testing.T) {
	f := majorityFund()
	targetMember := &fund.FundMember{
		ID:     uuid.New(),
		FundID: f.ID,
		Status: fund.MemberStatusActive,
	}
	payload, _ := json.Marshal(governance.RemoveMemberPayload{MemberID: targetMember.ID.String()})
	p := openProposal(f.ID, governance.ProposalTypeRemoveMember, 2)
	p.VotesFor = 1
	p.Payload = payload
	proposalRepo := &mockProposalRepo{stored: p}
	members := &mockMemberRepo{byID: targetMember}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, &mockFundRepo{f: f}, members, &mockPaymentWriteRepo{})
	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved")
	}
	if members.updated == nil || members.updated.Status != fund.MemberStatusLeft {
		t.Errorf("expected member status = left, got %v", members.updated)
	}
}

func TestCastVote_ChangeGovernance_UpdatesGovernanceType(t *testing.T) {
	f := majorityFund()
	payload, _ := json.Marshal(governance.ChangeGovernancePayload{
		GovernanceType:      fund.GovernanceUnanimous,
		VotingDeadlineHours: 96,
	})
	p := openProposal(f.ID, governance.ProposalTypeChangeGovernance, 2)
	p.VotesFor = 1
	p.Payload = payload
	proposalRepo := &mockProposalRepo{stored: p}
	fundRepo := &mockFundRepo{f: f}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, fundRepo, &mockMemberRepo{}, &mockPaymentWriteRepo{})
	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved")
	}
	if fundRepo.updated.Rules.GovernanceType != fund.GovernanceUnanimous {
		t.Errorf("GovernanceType = %s, want unanimous", fundRepo.updated.Rules.GovernanceType)
	}
	if fundRepo.updated.Rules.VotingDeadlineHours != 96 {
		t.Errorf("VotingDeadlineHours = %d, want 96", fundRepo.updated.Rules.VotingDeadlineHours)
	}
}

func TestCastVote_DistributeVaca_MarksFundCompleted(t *testing.T) {
	f := majorityFund()
	f.Type = fund.FundTypeVaca
	f.Status = fund.FundStatusActive
	p := openProposal(f.ID, governance.ProposalTypeDistributeVaca, 2)
	p.VotesFor = 1
	proposalRepo := &mockProposalRepo{stored: p}
	fundRepo := &mockFundRepo{f: f}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, fundRepo, &mockMemberRepo{}, &mockPaymentWriteRepo{})
	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteYes, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Approved {
		t.Fatal("expected approved")
	}
	if fundRepo.updated == nil || fundRepo.updated.Status != fund.FundStatusCompleted {
		t.Errorf("expected fund status = completed, got %v", fundRepo.updated)
	}
}

func TestCastVote_RejectsProposal_StatusSetToRejected(t *testing.T) {
	f := majorityFund()
	p := openProposal(f.ID, governance.ProposalTypeActivateFund, 2)
	p.VotesAgainst = 1 // one no already
	proposalRepo := &mockProposalRepo{stored: p}

	uc := newCastVoteUC(proposalRepo, &mockVoteRepo{}, &mockFundRepo{f: f}, &mockMemberRepo{}, &mockPaymentWriteRepo{})
	result, err := uc.Execute(context.Background(), p.ID, uuid.New(), governance.VoteNo, f)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if proposalRepo.updated == nil || proposalRepo.updated.Status != governance.ProposalStatusRejected {
		t.Errorf("expected proposal status = rejected, got %v", proposalRepo.updated)
	}
	if result.Approved {
		t.Error("result should not be approved")
	}
}

// ─── Domain — ResolveResult ───────────────────────────────────────────────────

func TestResolveResult_Majority_OddMembers(t *testing.T) {
	tests := []struct {
		name         string
		total        int
		votesFor     int
		votesAgainst int
		wantApproved bool
		wantResolved bool
	}{
		{"3 members, 2 yes → approved", 3, 2, 0, true, true},
		{"3 members, 2 no → rejected", 3, 0, 2, false, true},
		{"3 members, 1 yes, 1 no → pending", 3, 1, 1, false, false},
		{"5 members, 3 yes → approved", 5, 3, 0, true, true},
		{"5 members, 3 no → rejected", 5, 0, 3, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &governance.Proposal{
				TotalMembers: tc.total,
				VotesFor:     tc.votesFor,
				VotesAgainst: tc.votesAgainst,
			}
			approved, resolved := p.ResolveResult(fund.GovernanceMajority)
			if approved != tc.wantApproved || resolved != tc.wantResolved {
				t.Errorf("approved=%v resolved=%v; want approved=%v resolved=%v",
					approved, resolved, tc.wantApproved, tc.wantResolved)
			}
		})
	}
}

func TestResolveResult_Unanimous(t *testing.T) {
	tests := []struct {
		name         string
		total        int
		votesFor     int
		votesAgainst int
		wantApproved bool
		wantResolved bool
	}{
		{"all yes → approved", 3, 3, 0, true, true},
		{"one no → rejected immediately", 5, 4, 1, false, true},
		{"pending", 4, 2, 0, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &governance.Proposal{
				TotalMembers: tc.total,
				VotesFor:     tc.votesFor,
				VotesAgainst: tc.votesAgainst,
			}
			approved, resolved := p.ResolveResult(fund.GovernanceUnanimous)
			if approved != tc.wantApproved || resolved != tc.wantResolved {
				t.Errorf("approved=%v resolved=%v; want approved=%v resolved=%v",
					approved, resolved, tc.wantApproved, tc.wantResolved)
			}
		})
	}
}
