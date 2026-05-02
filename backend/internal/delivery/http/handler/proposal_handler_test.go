package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/handler"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/governance"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	ucgovernance "github.com/sacramento-finance/backend/internal/usecase/governance"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	ucpayment "github.com/sacramento-finance/backend/internal/usecase/payment"
	ucvaca "github.com/sacramento-finance/backend/internal/usecase/vaca"
	"github.com/shopspring/decimal"
)

type mockProposalHandlerProposalRepo struct {
	stored    *governance.Proposal
	listed    []*governance.Proposal
	created   *governance.Proposal
	updated   []*governance.Proposal
	createErr error
	getErr    error
	listErr   error
	updateErr error
	incErr    error
}

func (m *mockProposalHandlerProposalRepo) Create(_ context.Context, p *governance.Proposal) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = p
	m.stored = p
	return nil
}

func (m *mockProposalHandlerProposalRepo) GetByID(_ context.Context, _ uuid.UUID) (*governance.Proposal, error) {
	return m.stored, m.getErr
}

func (m *mockProposalHandlerProposalRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*governance.Proposal, error) {
	if m.listed != nil {
		return m.listed, m.listErr
	}
	if m.stored != nil {
		return []*governance.Proposal{m.stored}, m.listErr
	}
	return nil, m.listErr
}

func (m *mockProposalHandlerProposalRepo) UpdateStatus(_ context.Context, p *governance.Proposal) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	now := time.Now().UTC()
	p.ResolvedAt = &now
	m.updated = append(m.updated, p)
	return nil
}

func (m *mockProposalHandlerProposalRepo) IncrementVotes(_ context.Context, _ uuid.UUID, choice governance.VoteChoice) error {
	if m.incErr != nil {
		return m.incErr
	}
	if m.stored != nil {
		if choice == governance.VoteYes {
			m.stored.VotesFor++
		} else {
			m.stored.VotesAgainst++
		}
	}
	return nil
}

type mockProposalHandlerVoteRepo struct {
	existing *governance.Vote
	listed   []*governance.Vote
	created  *governance.Vote
}

func (m *mockProposalHandlerVoteRepo) Create(_ context.Context, v *governance.Vote) error {
	m.created = v
	m.existing = v
	m.listed = append(m.listed, v)
	return nil
}

func (m *mockProposalHandlerVoteRepo) GetByProposalAndMember(_ context.Context, _, _ uuid.UUID) (*governance.Vote, error) {
	return m.existing, nil
}

func (m *mockProposalHandlerVoteRepo) ListByProposal(_ context.Context, _ uuid.UUID) ([]*governance.Vote, error) {
	return m.listed, nil
}

type mockProposalHandlerPaymentRepo struct {
	payment *ledger.Payment
	updated *ledger.Payment
	batch   []*ledger.Payment
}

func (m *mockProposalHandlerPaymentRepo) GetByID(_ context.Context, _ uuid.UUID) (*ledger.Payment, error) {
	return m.payment, nil
}
func (m *mockProposalHandlerPaymentRepo) Update(_ context.Context, p *ledger.Payment) error {
	m.updated = p
	return nil
}
func (m *mockProposalHandlerPaymentRepo) CreateBatch(_ context.Context, payments []*ledger.Payment) error {
	m.batch = append(m.batch, payments...)
	return nil
}

type mockProposalHandlerVacaRepo struct{}

func (m *mockProposalHandlerVacaRepo) Create(_ context.Context, _ *fund.VacaConfig) error { return nil }
func (m *mockProposalHandlerVacaRepo) GetByFundID(_ context.Context, fundID uuid.UUID) (*fund.VacaConfig, error) {
	return &fund.VacaConfig{FundID: fundID, GoalAmount: decimal.NewFromInt(1000000)}, nil
}
func (m *mockProposalHandlerVacaRepo) Update(_ context.Context, _ *fund.VacaConfig) error { return nil }

type mockProposalHandlerVacaLedger struct{}

func (m *mockProposalHandlerVacaLedger) GetFundBalance(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
	return decimal.NewFromInt(1000000), nil
}
func (m *mockProposalHandlerVacaLedger) RecordEntriesTx(_ context.Context, _ []*ledger.LedgerEntry) error {
	return nil
}

type proposalHandlerDeps struct {
	funds     *mockFundHandlerFundRepo
	members   *mockFundHandlerMemberRepo
	proposals *mockProposalHandlerProposalRepo
	votes     *mockProposalHandlerVoteRepo
	payments  *mockProposalHandlerPaymentRepo
	notifRepo *mockNotifRepo
}

func newProposalHandlerRouter(userID uuid.UUID, deps *proposalHandlerDeps) *gin.Engine {
	if deps.funds == nil {
		deps.funds = &mockFundHandlerFundRepo{}
	}
	if deps.members == nil {
		deps.members = &mockFundHandlerMemberRepo{}
	}
	if deps.proposals == nil {
		deps.proposals = &mockProposalHandlerProposalRepo{}
	}
	if deps.votes == nil {
		deps.votes = &mockProposalHandlerVoteRepo{}
	}
	if deps.payments == nil {
		deps.payments = &mockProposalHandlerPaymentRepo{}
	}
	if deps.notifRepo == nil {
		deps.notifRepo = &mockNotifRepo{}
	}

	createProposal := ucgovernance.NewCreateProposalUseCase(deps.proposals, deps.members)
	generateSchedule := ucpayment.NewGenerateScheduleUseCase(deps.payments)
	distributeVaca := ucvaca.NewDistributeUseCase(&mockProposalHandlerVacaRepo{}, deps.funds, deps.members, &mockProposalHandlerVacaLedger{})
	castVote := ucgovernance.NewCastVoteUseCase(
		deps.proposals, deps.votes,
		deps.funds, deps.members,
		deps.payments, generateSchedule, distributeVaca,
	)
	notifSvc := ucnotif.NewService(deps.notifRepo)
	h := handler.NewProposalHandler(createProposal, castVote, deps.proposals, deps.votes, deps.funds, deps.members, notifSvc)

	injectUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID.String())
		c.Next()
	}

	r := gin.New()
	r.POST("/funds/:fund_id/proposals", injectUser, h.Create)
	r.GET("/funds/:fund_id/proposals", injectUser, h.List)
	r.GET("/funds/:fund_id/proposals/:proposal_id", injectUser, h.Get)
	r.POST("/funds/:fund_id/proposals/:proposal_id/vote", injectUser, h.Vote)
	return r
}

func governedFundForHandler(userID uuid.UUID) *fund.Fund {
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceMajority)
	f.Type = fund.FundTypeCirculo
	f.Name = "Circulo gobernado"
	f.Rules.MinMembers = 2
	return f
}

func proposalForHandler(fundID uuid.UUID, proposalType governance.ProposalType, totalMembers int) *governance.Proposal {
	payload, _ := json.Marshal(map[string]any{})
	return &governance.Proposal{
		ID:           uuid.New(),
		FundID:       fundID,
		ProposerID:   uuid.New(),
		Type:         proposalType,
		Status:       governance.ProposalStatusOpen,
		Payload:      payload,
		VotesFor:     0,
		VotesAgainst: 0,
		QuorumNeeded: (totalMembers + 1) / 2,
		TotalMembers: totalMembers,
		DeadlineAt:   time.Now().UTC().Add(48 * time.Hour),
		CreatedAt:    time.Now().UTC(),
	}
}

func proposalRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestProposalHandler_Create_CreatesProposalForActiveMember(t *testing.T) {
	userID := uuid.New()
	f := governedFundForHandler(userID)
	member := memberForHandler(f.ID, userID)
	other := memberForHandler(f.ID, uuid.New())
	deps := &proposalHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser:  map[uuid.UUID]*fund.FundMember{userID: member, other.UserID: other},
			members: []*fund.FundMember{member, other},
			count:   2,
		},
	}
	r := newProposalHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, proposalRequest(http.MethodPost, "/funds/"+f.ID.String()+"/proposals", `{"type":"activate_fund"}`))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if deps.proposals.created == nil {
		t.Fatal("expected proposal to be created")
	}
	if deps.proposals.created.Type != governance.ProposalTypeActivateFund {
		t.Errorf("proposal type = %s, want activate_fund", deps.proposals.created.Type)
	}
	if deps.proposals.created.ProposerID != member.ID {
		t.Errorf("ProposerID = %s, want %s", deps.proposals.created.ProposerID, member.ID)
	}
	if deps.proposals.created.QuorumNeeded != 1 {
		t.Errorf("QuorumNeeded = %d, want 1 for 2 majority members", deps.proposals.created.QuorumNeeded)
	}
}

func TestProposalHandler_Create_InvalidPayload_Returns400(t *testing.T) {
	userID := uuid.New()
	f := governedFundForHandler(userID)
	member := memberForHandler(f.ID, userID)
	deps := &proposalHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: member},
			count:  2,
		},
	}
	r := newProposalHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, proposalRequest(http.MethodPost, "/funds/"+f.ID.String()+"/proposals", `{
		"type": "waive_payment",
		"payload": {}
	}`))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "INVALID_PAYLOAD")
}

func TestProposalHandler_Create_NotMember_Returns403(t *testing.T) {
	userID := uuid.New()
	f := governedFundForHandler(userID)
	deps := &proposalHandlerDeps{funds: &mockFundHandlerFundRepo{byID: f}}
	r := newProposalHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, proposalRequest(http.MethodPost, "/funds/"+f.ID.String()+"/proposals", `{"type":"activate_fund"}`))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_FUND_MEMBER")
}

func TestProposalHandler_List_ExpiresOpenProposalsPastDeadline(t *testing.T) {
	userID := uuid.New()
	f := governedFundForHandler(userID)
	member := memberForHandler(f.ID, userID)
	expired := proposalForHandler(f.ID, governance.ProposalTypeCancelFund, 2)
	expired.DeadlineAt = time.Now().UTC().Add(-1 * time.Hour)
	open := proposalForHandler(f.ID, governance.ProposalTypeActivateFund, 2)
	deps := &proposalHandlerDeps{
		members: &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
		proposals: &mockProposalHandlerProposalRepo{
			listed: []*governance.Proposal{expired, open},
		},
	}
	r := newProposalHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/funds/"+f.ID.String()+"/proposals", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if expired.Status != governance.ProposalStatusExpired {
		t.Errorf("expired proposal status = %s, want expired", expired.Status)
	}
	if len(deps.proposals.updated) != 1 || deps.proposals.updated[0].ID != expired.ID {
		t.Fatalf("updated proposals = %+v, want only expired proposal", deps.proposals.updated)
	}
}

func TestProposalHandler_Get_ReturnsProposalVotesAndMyVote(t *testing.T) {
	userID := uuid.New()
	f := governedFundForHandler(userID)
	member := memberForHandler(f.ID, userID)
	p := proposalForHandler(f.ID, governance.ProposalTypeActivateFund, 3)
	p.VotesFor = 1
	myVote := &governance.Vote{ID: uuid.New(), ProposalID: p.ID, MemberID: member.ID, Choice: governance.VoteYes, CreatedAt: time.Now().UTC()}
	deps := &proposalHandlerDeps{
		members:   &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
		proposals: &mockProposalHandlerProposalRepo{stored: p},
		votes:     &mockProposalHandlerVoteRepo{existing: myVote, listed: []*governance.Vote{myVote}},
	}
	r := newProposalHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/funds/"+f.ID.String()+"/proposals/"+p.ID.String(), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !bodyContains(w.Body.String(), p.ID.String()) {
		t.Fatalf("response should include proposal id %s: %s", p.ID, w.Body.String())
	}
	if !bodyContains(w.Body.String(), myVote.ID.String()) {
		t.Fatalf("response should include vote id %s: %s", myVote.ID, w.Body.String())
	}
}

func TestProposalHandler_Vote_ApprovesAndExecutesProposal(t *testing.T) {
	userID := uuid.New()
	f := governedFundForHandler(userID)
	f.Status = fund.FundStatusDraft
	member := memberForHandler(f.ID, userID)
	other := memberForHandler(f.ID, uuid.New())
	p := proposalForHandler(f.ID, governance.ProposalTypeActivateFund, 3)
	p.VotesFor = 1
	deps := &proposalHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser:  map[uuid.UUID]*fund.FundMember{userID: member, other.UserID: other},
			members: []*fund.FundMember{member, other},
			count:   2,
		},
		proposals: &mockProposalHandlerProposalRepo{stored: p},
	}
	r := newProposalHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, proposalRequest(http.MethodPost, "/funds/"+f.ID.String()+"/proposals/"+p.ID.String()+"/vote", `{"choice":"yes"}`))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if deps.proposals.stored.Status != governance.ProposalStatusApproved {
		t.Fatalf("proposal status = %s, want approved", deps.proposals.stored.Status)
	}
	if deps.funds.updated == nil || deps.funds.updated.Status != fund.FundStatusActive {
		t.Fatalf("fund status = %+v, want active", deps.funds.updated)
	}
	if len(deps.payments.batch) != 4 {
		t.Fatalf("generated payments = %d, want 4", len(deps.payments.batch))
	}
}

func TestProposalHandler_Vote_InvalidChoice_Returns400(t *testing.T) {
	userID := uuid.New()
	f := governedFundForHandler(userID)
	member := memberForHandler(f.ID, userID)
	p := proposalForHandler(f.ID, governance.ProposalTypeActivateFund, 2)
	deps := &proposalHandlerDeps{
		funds:     &mockFundHandlerFundRepo{byID: f},
		members:   &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
		proposals: &mockProposalHandlerProposalRepo{stored: p},
	}
	r := newProposalHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, proposalRequest(http.MethodPost, "/funds/"+f.ID.String()+"/proposals/"+p.ID.String()+"/vote", `{"choice":"maybe"}`))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "VALIDATION_ERROR")
}
