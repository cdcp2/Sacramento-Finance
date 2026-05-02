package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/handler"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	ucpayment "github.com/sacramento-finance/backend/internal/usecase/payment"
	"github.com/shopspring/decimal"
)

type mockPaymentHandlerPaymentRepo struct {
	payment       *ledger.Payment
	updated       *ledger.Payment
	byMember      []*ledger.Payment
	byFund        []*ledger.Payment
	getErr        error
	updateErr     error
	listMemberErr error
	listFundErr   error
}

func (m *mockPaymentHandlerPaymentRepo) GetByID(_ context.Context, _ uuid.UUID) (*ledger.Payment, error) {
	return m.payment, m.getErr
}

func (m *mockPaymentHandlerPaymentRepo) Update(_ context.Context, p *ledger.Payment) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = p
	return nil
}

func (m *mockPaymentHandlerPaymentRepo) ListByMember(_ context.Context, _ uuid.UUID) ([]*ledger.Payment, error) {
	return m.byMember, m.listMemberErr
}

func (m *mockPaymentHandlerPaymentRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*ledger.Payment, error) {
	return m.byFund, m.listFundErr
}

type mockPaymentHandlerLedgerRepo struct {
	recordedPayment *ledger.Payment
	recordedEntry   *ledger.LedgerEntry
	recordErr       error
	entries         []*ledger.LedgerEntry
	balance         decimal.Decimal
	listErr         error
	balanceErr      error
}

func (m *mockPaymentHandlerLedgerRepo) RecordPaymentTx(_ context.Context, p *ledger.Payment, entry *ledger.LedgerEntry) error {
	if m.recordErr != nil {
		return m.recordErr
	}
	m.recordedPayment = p
	m.recordedEntry = entry
	return nil
}

func (m *mockPaymentHandlerLedgerRepo) GetFundBalance(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
	return m.balance, m.balanceErr
}

func (m *mockPaymentHandlerLedgerRepo) ListByFund(_ context.Context, _ uuid.UUID, _, _ int) ([]*ledger.LedgerEntry, error) {
	return m.entries, m.listErr
}

type paymentHandlerDeps struct {
	funds     *mockFundHandlerFundRepo
	members   *mockFundHandlerMemberRepo
	payments  *mockPaymentHandlerPaymentRepo
	ledger    *mockPaymentHandlerLedgerRepo
	notifRepo *mockNotifRepo
}

func newPaymentHandlerRouter(userID uuid.UUID, deps *paymentHandlerDeps) *gin.Engine {
	if deps.funds == nil {
		deps.funds = &mockFundHandlerFundRepo{}
	}
	if deps.members == nil {
		deps.members = &mockFundHandlerMemberRepo{}
	}
	if deps.payments == nil {
		deps.payments = &mockPaymentHandlerPaymentRepo{}
	}
	if deps.ledger == nil {
		deps.ledger = &mockPaymentHandlerLedgerRepo{}
	}
	if deps.notifRepo == nil {
		deps.notifRepo = &mockNotifRepo{}
	}

	recordPayment := ucpayment.NewRecordPaymentUseCase(deps.payments, deps.ledger)
	waivePayment := ucpayment.NewWaivePaymentUseCase(deps.payments)
	notifSvc := ucnotif.NewService(deps.notifRepo)
	h := handler.NewPaymentHandler(recordPayment, waivePayment, deps.payments, deps.ledger, deps.funds, deps.members, notifSvc)

	injectUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID.String())
		c.Next()
	}

	r := gin.New()
	r.GET("/funds/:fund_id/payments", injectUser, h.ListMine)
	r.GET("/funds/:fund_id/payments/all", injectUser, h.ListAll)
	r.POST("/funds/:fund_id/payments/:payment_id/pay", injectUser, h.Pay)
	r.POST("/funds/:fund_id/payments/:payment_id/waive", injectUser, h.Waive)
	r.GET("/funds/:fund_id/ledger", injectUser, h.Ledger)
	return r
}

func pendingPayment(fundID, memberID uuid.UUID) *ledger.Payment {
	return &ledger.Payment{
		ID:           uuid.New(),
		FundID:       fundID,
		MemberID:     memberID,
		PeriodNumber: 1,
		DueDate:      time.Now().UTC().Add(24 * time.Hour),
		AmountDue:    decimal.NewFromInt(100000),
		AmountPaid:   decimal.Zero,
		Status:       ledger.PaymentStatusPending,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
}

func activePaymentFund(userID uuid.UUID) *fund.Fund {
	f := fundForHandler(userID, fund.FundStatusActive, fund.GovernanceAdminOnly)
	f.Type = fund.FundTypeFondoAhorro
	return f
}

func TestPaymentHandler_ListMine_ReturnsMemberPayments(t *testing.T) {
	userID := uuid.New()
	f := activePaymentFund(userID)
	member := memberForHandler(f.ID, userID)
	p := pendingPayment(f.ID, member.ID)
	deps := &paymentHandlerDeps{
		members: &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
		payments: &mockPaymentHandlerPaymentRepo{
			byMember: []*ledger.Payment{p},
		},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/funds/"+f.ID.String()+"/payments", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !bodyContains(w.Body.String(), p.ID.String()) {
		t.Fatalf("response should include payment id %s: %s", p.ID, w.Body.String())
	}
}

func TestPaymentHandler_ListMine_NotMember_Returns403(t *testing.T) {
	userID := uuid.New()
	fundID := uuid.New()
	r := newPaymentHandlerRouter(userID, &paymentHandlerDeps{})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/funds/"+fundID.String()+"/payments", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_FUND_MEMBER")
}

func TestPaymentHandler_ListAll_AdminReturnsAllPayments(t *testing.T) {
	userID := uuid.New()
	f := activePaymentFund(userID)
	admin := adminMemberForHandler(f.ID, userID)
	p := pendingPayment(f.ID, admin.ID)
	deps := &paymentHandlerDeps{
		members:  &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: admin}},
		payments: &mockPaymentHandlerPaymentRepo{byFund: []*ledger.Payment{p}},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/funds/"+f.ID.String()+"/payments/all", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !bodyContains(w.Body.String(), p.ID.String()) {
		t.Fatalf("response should include payment id %s: %s", p.ID, w.Body.String())
	}
}

func TestPaymentHandler_ListAll_NonAdminReturns403(t *testing.T) {
	userID := uuid.New()
	f := activePaymentFund(userID)
	member := memberForHandler(f.ID, userID)
	deps := &paymentHandlerDeps{
		members: &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/funds/"+f.ID.String()+"/payments/all", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_FUND_ADMIN")
}

func TestPaymentHandler_Pay_ActiveFundRecordsPayment(t *testing.T) {
	userID := uuid.New()
	f := activePaymentFund(userID)
	member := memberForHandler(f.ID, userID)
	p := pendingPayment(f.ID, member.ID)
	deps := &paymentHandlerDeps{
		funds:    &mockFundHandlerFundRepo{byID: f},
		members:  &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
		payments: &mockPaymentHandlerPaymentRepo{payment: p},
		ledger:   &mockPaymentHandlerLedgerRepo{},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/funds/"+f.ID.String()+"/payments/"+p.ID.String()+"/pay", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if deps.ledger.recordedPayment == nil || deps.ledger.recordedPayment.Status != ledger.PaymentStatusPaid {
		t.Fatalf("recorded payment = %+v, want paid", deps.ledger.recordedPayment)
	}
	if deps.ledger.recordedEntry == nil || deps.ledger.recordedEntry.Type != ledger.EntryTypeDeposit {
		t.Fatalf("recorded entry = %+v, want deposit entry", deps.ledger.recordedEntry)
	}
}

func TestPaymentHandler_Pay_InactiveFundReturns400(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	member := memberForHandler(f.ID, userID)
	p := pendingPayment(f.ID, member.ID)
	deps := &paymentHandlerDeps{
		funds:    &mockFundHandlerFundRepo{byID: f},
		members:  &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
		payments: &mockPaymentHandlerPaymentRepo{payment: p},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/funds/"+f.ID.String()+"/payments/"+p.ID.String()+"/pay", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "FUND_NOT_ACTIVE")
}

func TestPaymentHandler_Waive_AdminWaivesPayment(t *testing.T) {
	userID := uuid.New()
	targetUserID := uuid.New()
	f := activePaymentFund(userID)
	admin := adminMemberForHandler(f.ID, userID)
	target := memberForHandler(f.ID, targetUserID)
	p := pendingPayment(f.ID, target.ID)
	deps := &paymentHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser:  map[uuid.UUID]*fund.FundMember{userID: admin, targetUserID: target},
			members: []*fund.FundMember{admin, target},
		},
		payments: &mockPaymentHandlerPaymentRepo{payment: p},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/funds/"+f.ID.String()+"/payments/"+p.ID.String()+"/waive", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if deps.payments.updated == nil || deps.payments.updated.Status != ledger.PaymentStatusWaived {
		t.Fatalf("updated payment = %+v, want waived", deps.payments.updated)
	}
}

func TestPaymentHandler_Waive_NonAdminReturns403(t *testing.T) {
	userID := uuid.New()
	f := activePaymentFund(userID)
	member := memberForHandler(f.ID, userID)
	p := pendingPayment(f.ID, member.ID)
	deps := &paymentHandlerDeps{
		members:  &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
		payments: &mockPaymentHandlerPaymentRepo{payment: p},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/funds/"+f.ID.String()+"/payments/"+p.ID.String()+"/waive", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_FUND_ADMIN")
}

func TestPaymentHandler_Waive_GovernedFundReturnsUseProposal(t *testing.T) {
	userID := uuid.New()
	targetUserID := uuid.New()
	f := activePaymentFund(userID)
	f.Rules.GovernanceType = fund.GovernanceMajority
	admin := adminMemberForHandler(f.ID, userID)
	target := memberForHandler(f.ID, targetUserID)
	p := pendingPayment(f.ID, target.ID)
	deps := &paymentHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: admin, targetUserID: target},
		},
		payments: &mockPaymentHandlerPaymentRepo{payment: p},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/funds/"+f.ID.String()+"/payments/"+p.ID.String()+"/waive", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "USE_PROPOSAL")
	if deps.payments.updated != nil {
		t.Fatalf("payment should not be waived directly in governed fund: %+v", deps.payments.updated)
	}
}

func TestPaymentHandler_Ledger_ReturnsEntriesAndBalance(t *testing.T) {
	userID := uuid.New()
	f := activePaymentFund(userID)
	member := memberForHandler(f.ID, userID)
	entry := &ledger.LedgerEntry{
		ID:           uuid.New(),
		FundID:       f.ID,
		Type:         ledger.EntryTypeDeposit,
		Direction:    ledger.DirectionCredit,
		Amount:       decimal.NewFromInt(100000),
		BalanceAfter: decimal.NewFromInt(100000),
		CreatedAt:    time.Now().UTC(),
	}
	deps := &paymentHandlerDeps{
		members: &mockFundHandlerMemberRepo{byUser: map[uuid.UUID]*fund.FundMember{userID: member}},
		ledger: &mockPaymentHandlerLedgerRepo{
			entries: []*ledger.LedgerEntry{entry},
			balance: decimal.NewFromInt(100000),
		},
	}
	r := newPaymentHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/funds/"+f.ID.String()+"/ledger", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !bodyContains(w.Body.String(), entry.ID.String()) {
		t.Fatalf("response should include ledger entry id %s: %s", entry.ID, w.Body.String())
	}
	if !bodyContains(w.Body.String(), "100000") {
		t.Fatalf("response should include balance: %s", w.Body.String())
	}
}

func bodyContains(body, part string) bool {
	return strings.Contains(body, part)
}
