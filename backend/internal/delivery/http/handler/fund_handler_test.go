package handler_test

import (
	"bytes"
	"context"
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
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/internal/domain/user"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	ucpayment "github.com/sacramento-finance/backend/internal/usecase/payment"
	"github.com/shopspring/decimal"
)

type mockFundHandlerFundRepo struct {
	created      *fund.Fund
	updated      *fund.Fund
	byID         *fund.Fund
	listByMember []*fund.Fund
	createErr    error
	updateErr    error
	getErr       error
	listErr      error
}

func (m *mockFundHandlerFundRepo) Create(_ context.Context, f *fund.Fund) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = f
	m.byID = f
	return nil
}

func (m *mockFundHandlerFundRepo) GetByID(_ context.Context, _ uuid.UUID) (*fund.Fund, error) {
	return m.byID, m.getErr
}

func (m *mockFundHandlerFundRepo) Update(_ context.Context, f *fund.Fund) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = f
	m.byID = f
	return nil
}

func (m *mockFundHandlerFundRepo) ListByMember(_ context.Context, _ uuid.UUID) ([]*fund.Fund, error) {
	return m.listByMember, m.listErr
}

type mockFundHandlerMemberRepo struct {
	added     []*fund.FundMember
	byUser    map[uuid.UUID]*fund.FundMember
	members   []*fund.FundMember
	addErr    error
	updateErr error
	listErr   error
	count     int
	countErr  error
}

func (m *mockFundHandlerMemberRepo) Add(_ context.Context, member *fund.FundMember) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.added = append(m.added, member)
	if m.byUser == nil {
		m.byUser = map[uuid.UUID]*fund.FundMember{}
	}
	m.byUser[member.UserID] = member
	m.members = append(m.members, member)
	return nil
}

func (m *mockFundHandlerMemberRepo) GetByFundAndUser(_ context.Context, fundID uuid.UUID, userID uuid.UUID) (*fund.FundMember, error) {
	if m.byUser == nil {
		return nil, nil
	}
	member := m.byUser[userID]
	if member == nil || member.FundID != fundID {
		return nil, nil
	}
	return member, nil
}

func (m *mockFundHandlerMemberRepo) GetByID(_ context.Context, id uuid.UUID) (*fund.FundMember, error) {
	for _, member := range m.members {
		if member.ID == id {
			return member, nil
		}
	}
	return nil, nil
}

func (m *mockFundHandlerMemberRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*fund.FundMember, error) {
	return m.members, m.listErr
}

func (m *mockFundHandlerMemberRepo) Update(_ context.Context, _ *fund.FundMember) error {
	return m.updateErr
}

func (m *mockFundHandlerMemberRepo) CountActive(_ context.Context, _ uuid.UUID) (int, error) {
	return m.count, m.countErr
}

type mockFundHandlerCirculoRepo struct {
	created *fund.CirculoConfig
}

func (m *mockFundHandlerCirculoRepo) Create(_ context.Context, c *fund.CirculoConfig) error {
	m.created = c
	return nil
}
func (m *mockFundHandlerCirculoRepo) GetByFundID(_ context.Context, _ uuid.UUID) (*fund.CirculoConfig, error) {
	return m.created, nil
}
func (m *mockFundHandlerCirculoRepo) Update(_ context.Context, c *fund.CirculoConfig) error {
	m.created = c
	return nil
}

type mockFundHandlerVacaRepo struct {
	created *fund.VacaConfig
}

func (m *mockFundHandlerVacaRepo) Create(_ context.Context, v *fund.VacaConfig) error {
	m.created = v
	return nil
}
func (m *mockFundHandlerVacaRepo) GetByFundID(_ context.Context, _ uuid.UUID) (*fund.VacaConfig, error) {
	return m.created, nil
}
func (m *mockFundHandlerVacaRepo) Update(_ context.Context, v *fund.VacaConfig) error {
	m.created = v
	return nil
}

type mockFundHandlerFondoRepo struct {
	created *fund.FondoConfig
}

func (m *mockFundHandlerFondoRepo) Create(_ context.Context, f *fund.FondoConfig) error {
	m.created = f
	return nil
}
func (m *mockFundHandlerFondoRepo) GetByFundID(_ context.Context, _ uuid.UUID) (*fund.FondoConfig, error) {
	return m.created, nil
}
func (m *mockFundHandlerFondoRepo) Update(_ context.Context, f *fund.FondoConfig) error {
	m.created = f
	return nil
}

type mockFundHandlerPaymentRepo struct {
	created []*ledger.Payment
	err     error
}

func (m *mockFundHandlerPaymentRepo) CreateBatch(_ context.Context, payments []*ledger.Payment) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, payments...)
	return nil
}

type mockFundHandlerUserRepo struct {
	byID       map[uuid.UUID]*user.User
	byEmail    map[string]*user.User
	byDocument map[string]*user.User
	updated    *user.User
	err        error
}

func (m *mockFundHandlerUserRepo) Create(_ context.Context, _ *user.User) error {
	return nil
}

func (m *mockFundHandlerUserRepo) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byID[id], nil
}

func (m *mockFundHandlerUserRepo) GetByDocument(_ context.Context, docType user.DocumentType, documentNumber string) (*user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byDocument[string(docType)+"|"+documentNumber], nil
}

func (m *mockFundHandlerUserRepo) GetByEmail(_ context.Context, email string) (*user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byEmail[email], nil
}

func (m *mockFundHandlerUserRepo) Update(_ context.Context, u *user.User) error {
	if m.err != nil {
		return m.err
	}
	m.updated = u
	return nil
}

type fundHandlerDeps struct {
	funds     *mockFundHandlerFundRepo
	members   *mockFundHandlerMemberRepo
	users     *mockFundHandlerUserRepo
	circulo   *mockFundHandlerCirculoRepo
	vaca      *mockFundHandlerVacaRepo
	fondo     *mockFundHandlerFondoRepo
	payments  *mockFundHandlerPaymentRepo
	notifRepo *mockNotifRepo
}

func newFundHandlerRouter(userID uuid.UUID, deps *fundHandlerDeps) *gin.Engine {
	if deps.funds == nil {
		deps.funds = &mockFundHandlerFundRepo{}
	}
	if deps.members == nil {
		deps.members = &mockFundHandlerMemberRepo{}
	}
	if deps.users == nil {
		deps.users = &mockFundHandlerUserRepo{
			byID:       map[uuid.UUID]*user.User{},
			byEmail:    map[string]*user.User{},
			byDocument: map[string]*user.User{},
		}
	}
	if deps.circulo == nil {
		deps.circulo = &mockFundHandlerCirculoRepo{}
	}
	if deps.vaca == nil {
		deps.vaca = &mockFundHandlerVacaRepo{}
	}
	if deps.fondo == nil {
		deps.fondo = &mockFundHandlerFondoRepo{}
	}
	if deps.payments == nil {
		deps.payments = &mockFundHandlerPaymentRepo{}
	}
	if deps.notifRepo == nil {
		deps.notifRepo = &mockNotifRepo{}
	}

	generateSchedule := ucpayment.NewGenerateScheduleUseCase(deps.payments)
	notifSvc := ucnotif.NewService(deps.notifRepo)
	h := handler.NewFundHandler(deps.funds, deps.members, deps.users, generateSchedule, deps.circulo, deps.vaca, deps.fondo, notifSvc)

	injectUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID.String())
		c.Next()
	}

	r := gin.New()
	r.POST("/funds", injectUser, h.Create)
	r.GET("/funds", injectUser, h.ListMine)
	r.GET("/funds/:fund_id", injectUser, h.Get)
	r.PATCH("/funds/:fund_id", injectUser, h.UpdateRules)
	r.POST("/funds/:fund_id/activate", injectUser, h.Activate)
	r.GET("/funds/:fund_id/members", injectUser, h.ListMembers)
	r.POST("/funds/:fund_id/members", injectUser, h.AddMember)
	return r
}

func fundRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func assertJSONContains(t *testing.T, raw []byte, want string) {
	t.Helper()
	if !bytes.Contains(raw, []byte(want)) {
		t.Fatalf("response body %s does not contain %q", raw, want)
	}
}

func validVacaFundJSON() string {
	return `{
		"name": "Vaca viaje",
		"description": "Ahorro para viaje",
		"type": "vaca",
		"contribution_amount": "100000",
		"frequency": "monthly",
		"total_periods": 4,
		"start_date": "2026-01-01",
		"min_members": 2,
		"max_members": 10,
		"governance_type": "majority",
		"voting_deadline_hours": 24,
		"goal_amount": "1000000",
		"goal_description": "Viaje grupal",
		"distribution_type": "goal_reached"
	}`
}

func fundForHandler(ownerID uuid.UUID, status fund.FundStatus, governance fund.GovernanceType) *fund.Fund {
	return &fund.Fund{
		ID:        uuid.New(),
		Name:      "Fondo test",
		Type:      fund.FundTypeCirculo,
		Status:    status,
		CreatorID: ownerID,
		Currency:  "COP",
		Rules: fund.Rules{
			ContributionAmount:  decimal.NewFromInt(100000),
			Frequency:           fund.FrequencyMonthly,
			TotalPeriods:        2,
			StartDate:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			MinMembers:          2,
			MaxMembers:          10,
			GovernanceType:      governance,
			VotingDeadlineHours: 48,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func adminMemberForHandler(fundID, userID uuid.UUID) *fund.FundMember {
	return &fund.FundMember{
		ID:       uuid.New(),
		FundID:   fundID,
		UserID:   userID,
		Role:     fund.RoleAdmin,
		Status:   fund.MemberStatusActive,
		JoinedAt: time.Now().UTC(),
	}
}

func memberForHandler(fundID, userID uuid.UUID) *fund.FundMember {
	return &fund.FundMember{
		ID:       uuid.New(),
		FundID:   fundID,
		UserID:   userID,
		Role:     fund.RoleMember,
		Status:   fund.MemberStatusActive,
		JoinedAt: time.Now().UTC(),
	}
}

func TestFundHandler_CreateVaca_CreatesFundAdminMemberAndConfig(t *testing.T) {
	userID := uuid.New()
	deps := &fundHandlerDeps{}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds", validVacaFundJSON()))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if deps.funds.created == nil {
		t.Fatal("expected fund to be created")
	}
	if deps.funds.created.Type != fund.FundTypeVaca {
		t.Errorf("fund type = %s, want vaca", deps.funds.created.Type)
	}
	if deps.funds.created.CreatorID != userID {
		t.Errorf("CreatorID = %s, want %s", deps.funds.created.CreatorID, userID)
	}
	if deps.funds.created.Rules.GovernanceType != fund.GovernanceMajority {
		t.Errorf("GovernanceType = %s, want majority", deps.funds.created.Rules.GovernanceType)
	}
	if len(deps.members.added) != 1 {
		t.Fatalf("added members = %d, want creator admin", len(deps.members.added))
	}
	if deps.members.added[0].Role != fund.RoleAdmin || deps.members.added[0].UserID != userID {
		t.Errorf("creator member = %+v, want admin for user", deps.members.added[0])
	}
	if deps.vaca.created == nil {
		t.Fatal("expected vaca config to be created")
	}
	if !deps.vaca.created.GoalAmount.Equal(decimal.NewFromInt(1000000)) {
		t.Errorf("GoalAmount = %s, want 1000000", deps.vaca.created.GoalAmount)
	}
	if deps.vaca.created.DistributionType != "goal_reached" {
		t.Errorf("DistributionType = %s, want goal_reached", deps.vaca.created.DistributionType)
	}
}

func TestFundHandler_Create_InvalidAmount_Returns400(t *testing.T) {
	userID := uuid.New()
	deps := &fundHandlerDeps{}
	r := newFundHandlerRouter(userID, deps)
	body := `{
		"name": "Fondo",
		"type": "circulo",
		"contribution_amount": "not-money",
		"frequency": "monthly",
		"total_periods": 3,
		"start_date": "2026-01-01",
		"min_members": 2
	}`

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds", body))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "INVALID_AMOUNT")
	if deps.funds.created != nil {
		t.Error("fund should not be created for invalid amount")
	}
}

func TestFundHandler_Create_RepositoryError_Returns500(t *testing.T) {
	userID := uuid.New()
	deps := &fundHandlerDeps{funds: &mockFundHandlerFundRepo{createErr: errors.New("db down")}}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds", validVacaFundJSON()))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "INTERNAL_ERROR")
}

func TestFundHandler_Get_MemberCanViewFund(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	member := memberForHandler(f.ID, userID)
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: member},
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodGet, "/funds/"+f.ID.String(), ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	assertJSONContains(t, w.Body.Bytes(), `"Name":"Fondo test"`)
}

func TestFundHandler_Get_NonMemberReturns403(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(uuid.New(), fund.FundStatusDraft, fund.GovernanceAdminOnly)
	deps := &fundHandlerDeps{funds: &mockFundHandlerFundRepo{byID: f}}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodGet, "/funds/"+f.ID.String(), ""))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_FUND_MEMBER")
}

func TestFundHandler_ListMembers_MemberCanList(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusActive, fund.GovernanceAdminOnly)
	member := memberForHandler(f.ID, userID)
	other := memberForHandler(f.ID, uuid.New())
	deps := &fundHandlerDeps{
		members: &mockFundHandlerMemberRepo{
			byUser:  map[uuid.UUID]*fund.FundMember{userID: member},
			members: []*fund.FundMember{member, other},
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodGet, "/funds/"+f.ID.String()+"/members", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	assertJSONContains(t, w.Body.Bytes(), member.ID.String())
	assertJSONContains(t, w.Body.Bytes(), other.ID.String())
}

func TestFundHandler_ListMembers_NonMemberReturns403(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(uuid.New(), fund.FundStatusActive, fund.GovernanceAdminOnly)
	deps := &fundHandlerDeps{}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodGet, "/funds/"+f.ID.String()+"/members", ""))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_FUND_MEMBER")
}

func TestFundHandler_Activate_AdminOnly_ActivatesAndGeneratesSchedule(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, userID)
	other := memberForHandler(f.ID, uuid.New())
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser:  map[uuid.UUID]*fund.FundMember{userID: admin, other.UserID: other},
			members: []*fund.FundMember{admin, other},
			count:   2,
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/activate", `{}`))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if deps.funds.updated == nil || deps.funds.updated.Status != fund.FundStatusActive {
		t.Fatalf("fund status = %v, want active", deps.funds.updated)
	}
	if len(deps.payments.created) != 4 {
		t.Fatalf("created payments = %d, want 4 (2 members x 2 periods)", len(deps.payments.created))
	}
}

func TestFundHandler_Activate_GovernedFund_ReturnsUseProposal(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceMajority)
	admin := adminMemberForHandler(f.ID, userID)
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: admin},
			count:  2,
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/activate", `{}`))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "USE_PROPOSAL")
	if deps.funds.updated != nil {
		t.Error("governed fund should not be directly activated")
	}
}

func TestFundHandler_Activate_NotEnoughMembers_Returns400(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, userID)
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: admin},
			count:  1,
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/activate", `{}`))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_ENOUGH_MEMBERS")
}

func TestFundHandler_UpdateRules_GovernedFund_ReturnsUseProposal(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceUnanimous)
	admin := adminMemberForHandler(f.ID, userID)
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: admin},
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPatch, "/funds/"+f.ID.String(), validVacaFundJSON()))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "USE_PROPOSAL")
}

func TestFundHandler_AddMember_AdminAddsNewMember(t *testing.T) {
	userID := uuid.New()
	targetID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, userID)
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser:  map[uuid.UUID]*fund.FundMember{userID: admin},
			members: []*fund.FundMember{admin},
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/members", `{"user_id":"`+targetID.String()+`"}`))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if len(deps.members.added) != 1 {
		t.Fatalf("added members = %d, want 1", len(deps.members.added))
	}
	added := deps.members.added[0]
	if added.UserID != targetID || added.Role != fund.RoleMember || added.Status != fund.MemberStatusActive {
		t.Errorf("added member = %+v, want active member for target", added)
	}
}

func TestFundHandler_AddMember_AdminAddsByEmail(t *testing.T) {
	userID := uuid.New()
	targetID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, userID)
	target := testUser(targetID)
	target.Email = "nuevo@example.com"
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser:  map[uuid.UUID]*fund.FundMember{userID: admin},
			members: []*fund.FundMember{admin},
		},
		users: &mockFundHandlerUserRepo{
			byEmail: map[string]*user.User{target.Email: target},
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/members", `{"email":"nuevo@example.com"}`))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if len(deps.members.added) != 1 || deps.members.added[0].UserID != targetID {
		t.Fatalf("added members = %+v, want target from email", deps.members.added)
	}
}

func TestFundHandler_AddMember_AdminAddsByDocument(t *testing.T) {
	userID := uuid.New()
	targetID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, userID)
	target := testUser(targetID)
	target.DocumentType = user.DocumentCedulaCiudadania
	target.DocumentNumber = "987654321"
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser:  map[uuid.UUID]*fund.FundMember{userID: admin},
			members: []*fund.FundMember{admin},
		},
		users: &mockFundHandlerUserRepo{
			byDocument: map[string]*user.User{"cedula_ciudadania|987654321": target},
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	body := `{"document_type":"cedula_ciudadania","document_number":"987654321"}`
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/members", body))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if len(deps.members.added) != 1 || deps.members.added[0].UserID != targetID {
		t.Fatalf("added members = %+v, want target from document", deps.members.added)
	}
}

func TestFundHandler_AddMember_TargetNotFoundReturns404(t *testing.T) {
	userID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, userID)
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: admin},
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/members", `{"email":"missing@example.com"}`))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_FOUND")
}

func TestFundHandler_AddMember_DuplicateReturns409(t *testing.T) {
	userID := uuid.New()
	targetID := uuid.New()
	f := fundForHandler(userID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, userID)
	existing := memberForHandler(f.ID, targetID)
	deps := &fundHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: admin, targetID: existing},
		},
	}
	r := newFundHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/members", `{"user_id":"`+targetID.String()+`"}`))

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "ALREADY_MEMBER")
}
