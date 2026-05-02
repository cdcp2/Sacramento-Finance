package handler_test

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/shopspring/decimal"
)

type mockDashboardPaymentRepo struct {
	byMember map[uuid.UUID][]*ledger.Payment
	err      error
}

func (m *mockDashboardPaymentRepo) Create(_ context.Context, _ *ledger.Payment) error {
	return nil
}

func (m *mockDashboardPaymentRepo) GetByID(_ context.Context, _ uuid.UUID) (*ledger.Payment, error) {
	return nil, nil
}

func (m *mockDashboardPaymentRepo) ListByMember(_ context.Context, memberID uuid.UUID) ([]*ledger.Payment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byMember[memberID], nil
}

func (m *mockDashboardPaymentRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*ledger.Payment, error) {
	return nil, nil
}

func (m *mockDashboardPaymentRepo) ListOverdue(_ context.Context, _ time.Time) ([]*ledger.Payment, error) {
	return nil, nil
}

func (m *mockDashboardPaymentRepo) Update(_ context.Context, _ *ledger.Payment) error {
	return nil
}

type mockDashboardProposalRepo struct {
	byFund map[uuid.UUID][]*governance.Proposal
	err    error
}

func (m *mockDashboardProposalRepo) Create(_ context.Context, _ *governance.Proposal) error {
	return nil
}

func (m *mockDashboardProposalRepo) GetByID(_ context.Context, _ uuid.UUID) (*governance.Proposal, error) {
	return nil, nil
}

func (m *mockDashboardProposalRepo) ListByFund(_ context.Context, fundID uuid.UUID) ([]*governance.Proposal, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byFund[fundID], nil
}

func (m *mockDashboardProposalRepo) UpdateStatus(_ context.Context, _ *governance.Proposal) error {
	return nil
}

func (m *mockDashboardProposalRepo) IncrementVotes(_ context.Context, _ uuid.UUID, _ governance.VoteChoice) error {
	return nil
}

func newDashboardRouter(userID uuid.UUID, deps *dashboardDeps) *gin.Engine {
	if deps.funds == nil {
		deps.funds = &mockFundHandlerFundRepo{}
	}
	if deps.members == nil {
		deps.members = &mockFundHandlerMemberRepo{}
	}
	if deps.payments == nil {
		deps.payments = &mockDashboardPaymentRepo{byMember: map[uuid.UUID][]*ledger.Payment{}}
	}
	if deps.proposals == nil {
		deps.proposals = &mockDashboardProposalRepo{byFund: map[uuid.UUID][]*governance.Proposal{}}
	}
	if deps.notifs == nil {
		deps.notifs = &mockNotifRepo{}
	}

	h := handler.NewDashboardHandler(deps.funds, deps.members, deps.payments, deps.proposals, deps.notifs)
	injectUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID.String())
		c.Next()
	}

	r := gin.New()
	r.GET("/dashboard", injectUser, h.Get)
	return r
}

type dashboardDeps struct {
	funds     *mockFundHandlerFundRepo
	members   *mockFundHandlerMemberRepo
	payments  *mockDashboardPaymentRepo
	proposals *mockDashboardProposalRepo
	notifs    *mockNotifRepo
}

func TestDashboardHandler_Get_ReturnsFrontSummary(t *testing.T) {
	userID := uuid.New()
	f := activePaymentFund(userID)
	f.Name = "Fondo principal"
	member := adminMemberForHandler(f.ID, userID)
	now := time.Now().UTC()
	pending := pendingPayment(f.ID, member.ID)
	pending.DueDate = now.Add(24 * time.Hour)
	overdue := pendingPayment(f.ID, member.ID)
	overdue.DueDate = now.Add(-24 * time.Hour)
	paid := pendingPayment(f.ID, member.ID)
	paid.Status = ledger.PaymentStatusPaid
	proposal := &governance.Proposal{
		ID:           uuid.New(),
		FundID:       f.ID,
		ProposerID:   member.ID,
		Type:         governance.ProposalTypeActivateFund,
		Status:       governance.ProposalStatusOpen,
		VotesFor:     1,
		QuorumNeeded: 2,
		DeadlineAt:   now.Add(48 * time.Hour),
		CreatedAt:    now,
	}
	deps := &dashboardDeps{
		funds: &mockFundHandlerFundRepo{listByMember: []*fund.Fund{f}},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: member},
		},
		payments: &mockDashboardPaymentRepo{
			byMember: map[uuid.UUID][]*ledger.Payment{
				member.ID: {pending, overdue, paid},
			},
		},
		proposals: &mockDashboardProposalRepo{
			byFund: map[uuid.UUID][]*governance.Proposal{f.ID: {proposal}},
		},
		notifs: &mockNotifRepo{unread: 3},
	}
	r := newDashboardRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/dashboard", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	raw := w.Body.Bytes()

	var body struct {
		Data struct {
			Summary struct {
				TotalFunds          int `json:"total_funds"`
				ActiveFunds         int `json:"active_funds"`
				AdminFunds          int `json:"admin_funds"`
				PendingPayments     int `json:"pending_payments"`
				OverduePayments     int `json:"overdue_payments"`
				OpenProposals       int `json:"open_proposals"`
				UnreadNotifications int `json:"unread_notifications"`
			} `json:"summary"`
			Funds            []any `json:"funds"`
			UpcomingPayments []any `json:"upcoming_payments"`
			OpenProposals    []any `json:"open_proposals"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Summary.TotalFunds != 1 || body.Data.Summary.ActiveFunds != 1 {
		t.Fatalf("fund summary = %+v, want one active fund", body.Data.Summary)
	}
	if body.Data.Summary.AdminFunds != 1 {
		t.Fatalf("admin_funds = %d, want 1", body.Data.Summary.AdminFunds)
	}
	if body.Data.Summary.PendingPayments != 2 || body.Data.Summary.OverduePayments != 1 {
		t.Fatalf("payment summary = %+v, want 2 pending and 1 overdue", body.Data.Summary)
	}
	if body.Data.Summary.OpenProposals != 1 || body.Data.Summary.UnreadNotifications != 3 {
		t.Fatalf("proposal/notification summary = %+v, want 1 open and 3 unread", body.Data.Summary)
	}
	if len(body.Data.Funds) != 1 || len(body.Data.UpcomingPayments) != 2 || len(body.Data.OpenProposals) != 1 {
		t.Fatalf("dashboard collections = %+v", body.Data)
	}
	assertJSONContains(t, raw, `"fund_name":"Fondo principal"`)
	assertJSONContains(t, raw, `"is_overdue":true`)
}

func TestDashboardHandler_Get_FundRepositoryErrorReturns500(t *testing.T) {
	userID := uuid.New()
	deps := &dashboardDeps{
		funds: &mockFundHandlerFundRepo{listErr: errors.New("db down")},
	}
	r := newDashboardRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/dashboard", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "INTERNAL_ERROR")
}

func TestDashboardHandler_Get_SortsAndLimitsUpcomingPayments(t *testing.T) {
	userID := uuid.New()
	f := activePaymentFund(userID)
	member := memberForHandler(f.ID, userID)
	now := time.Now().UTC()
	payments := make([]*ledger.Payment, 0, 7)
	for i := 7; i >= 1; i-- {
		p := pendingPayment(f.ID, member.ID)
		p.PeriodNumber = i
		p.DueDate = now.Add(time.Duration(i) * 24 * time.Hour)
		p.AmountDue = decimal.NewFromInt(int64(i * 1000))
		payments = append(payments, p)
	}
	deps := &dashboardDeps{
		funds: &mockFundHandlerFundRepo{listByMember: []*fund.Fund{f}},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: member},
		},
		payments: &mockDashboardPaymentRepo{
			byMember: map[uuid.UUID][]*ledger.Payment{member.ID: payments},
		},
	}
	r := newDashboardRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/dashboard", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	raw := w.Body.Bytes()
	var body struct {
		Data struct {
			UpcomingPayments []struct {
				PeriodNumber int `json:"period_number"`
			} `json:"upcoming_payments"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data.UpcomingPayments) != 5 {
		t.Fatalf("upcoming payments = %d, want 5", len(body.Data.UpcomingPayments))
	}
	if body.Data.UpcomingPayments[0].PeriodNumber != 1 {
		t.Fatalf("first upcoming period = %d, want earliest period 1", body.Data.UpcomingPayments[0].PeriodNumber)
	}
}
