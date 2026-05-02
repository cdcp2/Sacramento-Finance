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
	"github.com/sacramento-finance/backend/internal/domain/user"
)

type mockUserHandlerRepo struct {
	user      *user.User
	updated   *user.User
	getErr    error
	updateErr error
}

func (m *mockUserHandlerRepo) Create(_ context.Context, _ *user.User) error {
	return nil
}

func (m *mockUserHandlerRepo) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.user == nil || m.user.ID != id {
		return nil, nil
	}
	return m.user, nil
}

func (m *mockUserHandlerRepo) GetByDocument(_ context.Context, _ user.DocumentType, _ string) (*user.User, error) {
	return nil, nil
}

func (m *mockUserHandlerRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, nil
}

func (m *mockUserHandlerRepo) Update(_ context.Context, u *user.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = u
	return nil
}

func newUserHandlerRouter(userID uuid.UUID, repo *mockUserHandlerRepo) *gin.Engine {
	h := handler.NewUserHandler(repo)
	injectUser := func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID.String())
		c.Next()
	}

	r := gin.New()
	r.GET("/users/me", injectUser, h.Me)
	r.PATCH("/users/me", injectUser, h.UpdateMe)
	return r
}

func testUser(id uuid.UUID) *user.User {
	return &user.User{
		ID:                 id,
		DocumentType:       user.DocumentCedulaCiudadania,
		DocumentNumber:     "123456789",
		Email:              "ana@example.com",
		Phone:              "3001234567",
		FullName:           "Ana Perez",
		PasswordHash:       "secret-hash",
		IsVerified:         false,
		VerificationStatus: user.VerificationNone,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

func TestUserHandler_Me_ReturnsSanitizedUser(t *testing.T) {
	userID := uuid.New()
	repo := &mockUserHandlerRepo{user: testUser(userID)}
	r := newUserHandlerRouter(userID, repo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	assertJSONContains(t, w.Body.Bytes(), `"email":"ana@example.com"`)
	if bytes.Contains(w.Body.Bytes(), []byte("secret-hash")) {
		t.Fatalf("response should not expose password hash: %s", w.Body.String())
	}
}

func TestUserHandler_UpdateMe_UpdatesProfileFields(t *testing.T) {
	userID := uuid.New()
	repo := &mockUserHandlerRepo{user: testUser(userID)}
	r := newUserHandlerRouter(userID, repo)

	w := httptest.NewRecorder()
	body := `{"full_name":"  Ana Maria Perez  ","phone":"3109876543"}`
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if repo.updated == nil {
		t.Fatal("expected user to be updated")
	}
	if repo.updated.FullName != "Ana Maria Perez" {
		t.Fatalf("FullName = %q, want trimmed updated name", repo.updated.FullName)
	}
	if repo.updated.Phone != "3109876543" {
		t.Fatalf("Phone = %q, want updated phone", repo.updated.Phone)
	}
	assertJSONContains(t, w.Body.Bytes(), `"full_name":"Ana Maria Perez"`)
}

func TestUserHandler_UpdateMe_InvalidPhoneReturns400(t *testing.T) {
	userID := uuid.New()
	repo := &mockUserHandlerRepo{user: testUser(userID)}
	r := newUserHandlerRouter(userID, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`{"phone":"abc"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "VALIDATION_ERROR")
	if repo.updated != nil {
		t.Fatal("user should not be updated for invalid phone")
	}
}

func TestUserHandler_UpdateMe_NoFieldsReturns400(t *testing.T) {
	userID := uuid.New()
	repo := &mockUserHandlerRepo{user: testUser(userID)}
	r := newUserHandlerRouter(userID, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "VALIDATION_ERROR")
}

func TestUserHandler_UpdateMe_UpdateErrorReturns500(t *testing.T) {
	userID := uuid.New()
	repo := &mockUserHandlerRepo{user: testUser(userID), updateErr: errors.New("db down")}
	r := newUserHandlerRouter(userID, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`{"full_name":"Ana Maria"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "INTERNAL_ERROR")
}
