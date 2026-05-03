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
	"github.com/sacramento-finance/backend/internal/domain/user"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
)

type mockInvitationHandlerRepo struct {
	created       *fund.FundInvitation
	byID          *fund.FundInvitation
	listByInvitee []*fund.FundInvitation
	listByFund    []*fund.FundInvitation
	pending       *fund.FundInvitation
	createErr     error
	getErr        error
	listErr       error
	updateErr     error
	findErr       error
}

func (m *mockInvitationHandlerRepo) Create(_ context.Context, invitation *fund.FundInvitation) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = invitation
	m.byID = invitation
	return nil
}

func (m *mockInvitationHandlerRepo) GetByID(_ context.Context, _ uuid.UUID) (*fund.FundInvitation, error) {
	return m.byID, m.getErr
}

func (m *mockInvitationHandlerRepo) ListByInvitee(_ context.Context, _ uuid.UUID) ([]*fund.FundInvitation, error) {
	return m.listByInvitee, m.listErr
}

func (m *mockInvitationHandlerRepo) ListByFund(_ context.Context, _ uuid.UUID) ([]*fund.FundInvitation, error) {
	return m.listByFund, m.listErr
}

func (m *mockInvitationHandlerRepo) Update(_ context.Context, invitation *fund.FundInvitation) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.byID = invitation
	return nil
}

func (m *mockInvitationHandlerRepo) FindPending(_ context.Context, _, _ uuid.UUID) (*fund.FundInvitation, error) {
	return m.pending, m.findErr
}

type invitationHandlerDeps struct {
	funds       *mockFundHandlerFundRepo
	members     *mockFundHandlerMemberRepo
	invitations *mockInvitationHandlerRepo
	users       *mockFundHandlerUserRepo
	notifRepo   *mockNotifRepo
}

func newInvitationHandlerRouter(userID uuid.UUID, deps *invitationHandlerDeps) *gin.Engine {
	if deps.funds == nil {
		deps.funds = &mockFundHandlerFundRepo{}
	}
	if deps.members == nil {
		deps.members = &mockFundHandlerMemberRepo{}
	}
	if deps.invitations == nil {
		deps.invitations = &mockInvitationHandlerRepo{}
	}
	if deps.users == nil {
		deps.users = &mockFundHandlerUserRepo{
			byID:       map[uuid.UUID]*user.User{},
			byEmail:    map[string]*user.User{},
			byDocument: map[string]*user.User{},
		}
	}
	if deps.notifRepo == nil {
		deps.notifRepo = &mockNotifRepo{}
	}

	notifSvc := ucnotif.NewService(deps.notifRepo)
	h := handler.NewInvitationHandler(deps.funds, deps.members, deps.invitations, deps.users, notifSvc)
	injectUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID.String())
		c.Next()
	}

	r := gin.New()
	r.POST("/funds/:fund_id/invitations", injectUser, h.Create)
	r.GET("/funds/:fund_id/invitations", injectUser, h.ListByFund)
	r.GET("/invitations", injectUser, h.ListMine)
	r.POST("/invitations/:invitation_id/accept", injectUser, h.Accept)
	r.POST("/invitations/:invitation_id/reject", injectUser, h.Reject)
	return r
}

func invitationForHandler(fundID, inviterID, inviteeID uuid.UUID, status fund.InvitationStatus) *fund.FundInvitation {
	now := time.Now().UTC()
	return &fund.FundInvitation{
		ID:        uuid.New(),
		FundID:    fundID,
		InviterID: inviterID,
		InviteeID: inviteeID,
		Status:    status,
		Message:   "Únete al fondo",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestInvitationHandler_CreateByEmail_AdminCreatesPendingInvitation(t *testing.T) {
	adminID := uuid.New()
	inviteeID := uuid.New()
	f := fundForHandler(adminID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, adminID)
	target := testUser(inviteeID)
	target.Email = "nuevo@example.com"
	deps := &invitationHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{adminID: admin},
		},
		users: &mockFundHandlerUserRepo{
			byEmail: map[string]*user.User{target.Email: target},
		},
	}
	r := newInvitationHandlerRouter(adminID, deps)

	w := httptest.NewRecorder()
	body := `{"email":"nuevo@example.com","message":"Únete al fondo"}`
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/invitations", body))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if deps.invitations.created == nil {
		t.Fatal("expected invitation to be created")
	}
	if deps.invitations.created.FundID != f.ID || deps.invitations.created.InviterID != adminID || deps.invitations.created.InviteeID != inviteeID {
		t.Fatalf("created invitation = %+v, want fund/admin/invitee ids", deps.invitations.created)
	}
	if deps.invitations.created.Status != fund.InvitationStatusPending {
		t.Fatalf("status = %s, want pending", deps.invitations.created.Status)
	}
}

func TestInvitationHandler_Create_NonAdminReturns403(t *testing.T) {
	userID := uuid.New()
	inviteeID := uuid.New()
	f := fundForHandler(uuid.New(), fund.FundStatusDraft, fund.GovernanceAdminOnly)
	member := memberForHandler(f.ID, userID)
	deps := &invitationHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{userID: member},
		},
		invitations: &mockInvitationHandlerRepo{},
	}
	r := newInvitationHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/invitations", `{"user_id":"`+inviteeID.String()+`"}`))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "NOT_FUND_ADMIN")
	if deps.invitations.created != nil {
		t.Fatal("non-admin should not create invitations")
	}
}

func TestInvitationHandler_Create_DuplicatePendingReturns409(t *testing.T) {
	adminID := uuid.New()
	inviteeID := uuid.New()
	f := fundForHandler(adminID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, adminID)
	existing := invitationForHandler(f.ID, adminID, inviteeID, fund.InvitationStatusPending)
	deps := &invitationHandlerDeps{
		funds: &mockFundHandlerFundRepo{byID: f},
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{adminID: admin},
		},
		invitations: &mockInvitationHandlerRepo{pending: existing},
	}
	r := newInvitationHandlerRouter(adminID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/funds/"+f.ID.String()+"/invitations", `{"user_id":"`+inviteeID.String()+`"}`))

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "INVITATION_ALREADY_PENDING")
}

func TestInvitationHandler_ListByFund_EmptyListReturnsArray(t *testing.T) {
	adminID := uuid.New()
	f := fundForHandler(adminID, fund.FundStatusDraft, fund.GovernanceAdminOnly)
	admin := adminMemberForHandler(f.ID, adminID)
	deps := &invitationHandlerDeps{
		members: &mockFundHandlerMemberRepo{
			byUser: map[uuid.UUID]*fund.FundMember{adminID: admin},
		},
		invitations: &mockInvitationHandlerRepo{listByFund: nil},
	}
	r := newInvitationHandlerRouter(adminID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodGet, "/funds/"+f.ID.String()+"/invitations", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	assertJSONDataIsEmptyArray(t, w.Body.Bytes())
}

func TestInvitationHandler_ListMine_EmptyListReturnsArray(t *testing.T) {
	userID := uuid.New()
	deps := &invitationHandlerDeps{
		invitations: &mockInvitationHandlerRepo{listByInvitee: nil},
	}
	r := newInvitationHandlerRouter(userID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodGet, "/invitations", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	assertJSONDataIsEmptyArray(t, w.Body.Bytes())
}

func TestInvitationHandler_Accept_AddsInviteeAsMember(t *testing.T) {
	adminID := uuid.New()
	inviteeID := uuid.New()
	fundID := uuid.New()
	invitation := invitationForHandler(fundID, adminID, inviteeID, fund.InvitationStatusPending)
	deps := &invitationHandlerDeps{
		members:     &mockFundHandlerMemberRepo{},
		invitations: &mockInvitationHandlerRepo{byID: invitation},
	}
	r := newInvitationHandlerRouter(inviteeID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/invitations/"+invitation.ID.String()+"/accept", `{}`))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if invitation.Status != fund.InvitationStatusAccepted {
		t.Fatalf("invitation status = %s, want accepted", invitation.Status)
	}
	if len(deps.members.added) != 1 {
		t.Fatalf("added members = %d, want 1", len(deps.members.added))
	}
	added := deps.members.added[0]
	if added.FundID != fundID || added.UserID != inviteeID || added.Role != fund.RoleMember || added.Status != fund.MemberStatusActive {
		t.Fatalf("added member = %+v, want active member for invitee", added)
	}
}

func TestInvitationHandler_Reject_DoesNotAddMember(t *testing.T) {
	adminID := uuid.New()
	inviteeID := uuid.New()
	invitation := invitationForHandler(uuid.New(), adminID, inviteeID, fund.InvitationStatusPending)
	deps := &invitationHandlerDeps{
		members:     &mockFundHandlerMemberRepo{},
		invitations: &mockInvitationHandlerRepo{byID: invitation},
	}
	r := newInvitationHandlerRouter(inviteeID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/invitations/"+invitation.ID.String()+"/reject", `{}`))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if invitation.Status != fund.InvitationStatusRejected {
		t.Fatalf("invitation status = %s, want rejected", invitation.Status)
	}
	if len(deps.members.added) != 0 {
		t.Fatalf("reject should not add members; got %+v", deps.members.added)
	}
}

func TestInvitationHandler_Accept_ExpiredInvitationMarksExpired(t *testing.T) {
	adminID := uuid.New()
	inviteeID := uuid.New()
	invitation := invitationForHandler(uuid.New(), adminID, inviteeID, fund.InvitationStatusPending)
	invitation.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	deps := &invitationHandlerDeps{
		members:     &mockFundHandlerMemberRepo{},
		invitations: &mockInvitationHandlerRepo{byID: invitation},
	}
	r := newInvitationHandlerRouter(inviteeID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/invitations/"+invitation.ID.String()+"/accept", `{}`))

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "INVITATION_EXPIRED")
	if invitation.Status != fund.InvitationStatusExpired {
		t.Fatalf("invitation status = %s, want expired", invitation.Status)
	}
	if len(deps.members.added) != 0 {
		t.Fatalf("expired invitation should not add members; got %+v", deps.members.added)
	}
}

func TestInvitationHandler_Accept_RepoErrorReturns500(t *testing.T) {
	inviteeID := uuid.New()
	deps := &invitationHandlerDeps{
		invitations: &mockInvitationHandlerRepo{getErr: errors.New("db down")},
	}
	r := newInvitationHandlerRouter(inviteeID, deps)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, fundRequest(http.MethodPost, "/invitations/"+uuid.NewString()+"/accept", `{}`))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

func assertJSONDataIsEmptyArray(t *testing.T, raw []byte) {
	t.Helper()

	var body struct {
		Data []any `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode body: %v; body=%s", err, string(raw))
	}
	if body.Data == nil {
		t.Fatalf("data = nil, want empty array; body=%s", string(raw))
	}
	if len(body.Data) != 0 {
		t.Fatalf("data length = %d, want 0; body=%s", len(body.Data), string(raw))
	}
}
