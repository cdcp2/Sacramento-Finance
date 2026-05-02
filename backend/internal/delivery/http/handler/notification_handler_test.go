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
	domainnotif "github.com/sacramento-finance/backend/internal/domain/notification"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockNotifRepo struct {
	listed      []*domainnotif.Notification
	listErr     error
	unread      int
	markReadErr error
	markAllErr  error
}

func (m *mockNotifRepo) Create(_ context.Context, _ *domainnotif.Notification) error { return nil }
func (m *mockNotifRepo) ListByUser(_ context.Context, _ uuid.UUID, _, _ int) ([]*domainnotif.Notification, error) {
	return m.listed, m.listErr
}
func (m *mockNotifRepo) CountUnread(_ context.Context, _ uuid.UUID) (int, error) {
	return m.unread, nil
}
func (m *mockNotifRepo) MarkRead(_ context.Context, _, _ uuid.UUID) error { return m.markReadErr }
func (m *mockNotifRepo) MarkAllRead(_ context.Context, _ uuid.UUID) error { return m.markAllErr }

// ─── test helpers ─────────────────────────────────────────────────────────────

// newNotifRouter wires the handler into a Gin engine with the user_id key pre-set.
func newNotifRouter(repo domainnotif.Repository, userID uuid.UUID) *gin.Engine {
	r := gin.New()
	h := handler.NewNotificationHandler(repo)

	// Inject user_id without running the real JWT middleware
	inject := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID.String())
		c.Next()
	}

	r.GET("/notifications", inject, h.List)
	r.PATCH("/notifications/:notif_id/read", inject, h.MarkRead)
	r.POST("/notifications/read-all", inject, h.MarkAllRead)
	return r
}

func makeNotif(userID uuid.UUID, read bool) *domainnotif.Notification {
	return &domainnotif.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      domainnotif.TypePaymentConfirmed,
		Title:     "Aporte confirmado",
		Body:      "Tu aporte fue registrado.",
		IsRead:    read,
		CreatedAt: time.Now().UTC(),
	}
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestNotificationHandler_List_ReturnsNotificationsAndUnreadCount(t *testing.T) {
	userID := uuid.New()
	notifs := []*domainnotif.Notification{
		makeNotif(userID, false),
		makeNotif(userID, false),
		makeNotif(userID, true),
	}
	repo := &mockNotifRepo{listed: notifs, unread: 2}
	r := newNotifRouter(repo, userID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body struct {
		Data struct {
			Notifications []any `json:"notifications"`
			UnreadCount   int   `json:"unread_count"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(body.Data.Notifications) != 3 {
		t.Errorf("notifications count = %d, want 3", len(body.Data.Notifications))
	}
	if body.Data.UnreadCount != 2 {
		t.Errorf("unread_count = %d, want 2", body.Data.UnreadCount)
	}
}

func TestNotificationHandler_List_EmptyList_ReturnsNullNotifications(t *testing.T) {
	repo := &mockNotifRepo{listed: nil, unread: 0}
	r := newNotifRouter(repo, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestNotificationHandler_List_RepoError_Returns500(t *testing.T) {
	repo := &mockNotifRepo{listErr: errors.New("db error")}
	r := newNotifRouter(repo, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestNotificationHandler_List_LimitQueryParam_Accepted(t *testing.T) {
	repo := &mockNotifRepo{}
	r := newNotifRouter(repo, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=5&offset=10", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// ─── MarkRead ─────────────────────────────────────────────────────────────────

func TestNotificationHandler_MarkRead_ValidID_Returns200(t *testing.T) {
	repo := &mockNotifRepo{}
	r := newNotifRouter(repo, uuid.New())

	notifID := uuid.New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/"+notifID.String()+"/read", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestNotificationHandler_MarkRead_InvalidUUID_Returns400(t *testing.T) {
	repo := &mockNotifRepo{}
	r := newNotifRouter(repo, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/not-a-uuid/read", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestNotificationHandler_MarkRead_NotOwner_Returns404(t *testing.T) {
	// MarkRead returns ErrNoRows (from pgx) when user_id doesn't match.
	// Our mock simulates that with a non-nil error.
	repo := &mockNotifRepo{markReadErr: errors.New("not found")}
	r := newNotifRouter(repo, uuid.New())

	notifID := uuid.New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/"+notifID.String()+"/read", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ─── MarkAllRead ──────────────────────────────────────────────────────────────

func TestNotificationHandler_MarkAllRead_Returns200(t *testing.T) {
	repo := &mockNotifRepo{}
	r := newNotifRouter(repo, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body struct {
		Data struct {
			Ok bool `json:"ok"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Data.Ok {
		t.Error("expected ok=true")
	}
}

func TestNotificationHandler_MarkAllRead_RepoError_Returns500(t *testing.T) {
	repo := &mockNotifRepo{markAllErr: errors.New("db error")}
	r := newNotifRouter(repo, uuid.New())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}
