package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/handler"
	"github.com/sacramento-finance/backend/internal/domain/user"
	"github.com/sacramento-finance/backend/internal/usecase/auth"
	"golang.org/x/crypto/bcrypt"
)

type mockAuthUserRepo struct {
	byID       *user.User
	byDocument *user.User
	byEmail    *user.User
	created    *user.User
	updated    *user.User
	createErr  error
}

func (m *mockAuthUserRepo) Create(_ context.Context, u *user.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = u
	return nil
}

func (m *mockAuthUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*user.User, error) {
	return m.byID, nil
}

func (m *mockAuthUserRepo) GetByDocument(_ context.Context, _ user.DocumentType, _ string) (*user.User, error) {
	return m.byDocument, nil
}

func (m *mockAuthUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return m.byEmail, nil
}

func (m *mockAuthUserRepo) Update(_ context.Context, u *user.User) error {
	m.updated = u
	return nil
}

func newAuthRouter(repo *mockAuthUserRepo) *gin.Engine {
	r := gin.New()
	registerUC := auth.NewRegisterUseCase(repo)
	loginUC := auth.NewLoginUseCase(repo, "test-secret", 15*time.Minute)
	h := handler.NewAuthHandler(registerUC, loginUC)

	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	return r
}

func authRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func existingAuthUser(password string) *user.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return &user.User{
		ID:                 uuid.New(),
		DocumentType:       user.DocumentCedulaCiudadania,
		DocumentNumber:     "1234567890",
		Email:              "ana@example.com",
		Phone:              "3001234567",
		FullName:           "Ana Perez",
		PasswordHash:       string(hash),
		VerificationStatus: user.VerificationNone,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

func validRegisterJSON() string {
	return `{
		"document_type": "cedula_ciudadania",
		"document_number": "1234567890",
		"email": "ana@example.com",
		"phone": "3001234567",
		"full_name": "Ana Perez",
		"password": "super-secret"
	}`
}

func TestAuthHandler_Register_Success_ReturnsCreatedUser(t *testing.T) {
	repo := &mockAuthUserRepo{}
	r := newAuthRouter(repo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/register", validRegisterJSON()))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if repo.created == nil {
		t.Fatal("expected user to be created")
	}
	if repo.created.PasswordHash == "super-secret" {
		t.Fatal("password should be hashed before persistence")
	}

	var body struct {
		Data struct {
			ID                 string `json:"id"`
			DocumentType       string `json:"document_type"`
			DocumentNumber     string `json:"document_number"`
			Email              string `json:"email"`
			FullName           string `json:"full_name"`
			VerificationStatus string `json:"verification_status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Data.ID == "" {
		t.Error("response should include user id")
	}
	if body.Data.DocumentType != string(user.DocumentCedulaCiudadania) {
		t.Errorf("document_type = %s, want cedula_ciudadania", body.Data.DocumentType)
	}
	if body.Data.DocumentNumber != "1234567890" {
		t.Errorf("document_number = %s, want 1234567890", body.Data.DocumentNumber)
	}
	if body.Data.Email != "ana@example.com" || body.Data.FullName != "Ana Perez" {
		t.Errorf("unexpected user response: %+v", body.Data)
	}
	if body.Data.VerificationStatus != string(user.VerificationNone) {
		t.Errorf("verification_status = %s, want none", body.Data.VerificationStatus)
	}
	if strings.Contains(w.Body.String(), "password") {
		t.Errorf("response should not include password fields: %s", w.Body.String())
	}
}

func TestAuthHandler_Register_InvalidJSON_Returns400(t *testing.T) {
	r := newAuthRouter(&mockAuthUserRepo{})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/register", `{"email":"not-valid"}`))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "VALIDATION_ERROR")
}

func TestAuthHandler_Register_InvalidCedula_Returns400(t *testing.T) {
	r := newAuthRouter(&mockAuthUserRepo{})
	body := strings.Replace(validRegisterJSON(), `"1234567890"`, `"0123"`, 1)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/register", body))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "INVALID_DOCUMENT_NUMBER")
}

func TestAuthHandler_Register_DuplicateEmail_Returns409(t *testing.T) {
	repo := &mockAuthUserRepo{byEmail: existingAuthUser("super-secret")}
	r := newAuthRouter(repo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/register", validRegisterJSON()))

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "DUPLICATE_EMAIL")
}

func TestAuthHandler_Register_CreateError_Returns500(t *testing.T) {
	repo := &mockAuthUserRepo{createErr: errors.New("db down")}
	r := newAuthRouter(repo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/register", validRegisterJSON()))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "INTERNAL_ERROR")
}

func TestAuthHandler_Login_WithEmail_ReturnsTokens(t *testing.T) {
	repo := &mockAuthUserRepo{byEmail: existingAuthUser("super-secret")}
	r := newAuthRouter(repo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/login", `{
		"email": "ana@example.com",
		"password": "super-secret"
	}`))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Data.AccessToken == "" || body.Data.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}
	if body.Data.ExpiresIn != int64((15 * time.Minute).Seconds()) {
		t.Errorf("expires_in = %d, want 900", body.Data.ExpiresIn)
	}
}

func TestAuthHandler_Login_WithDocument_ReturnsTokens(t *testing.T) {
	repo := &mockAuthUserRepo{byDocument: existingAuthUser("super-secret")}
	r := newAuthRouter(repo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/login", `{
		"document_type": "cedula_ciudadania",
		"document_number": "1234567890",
		"password": "super-secret"
	}`))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Login_MissingIdentifier_Returns400(t *testing.T) {
	r := newAuthRouter(&mockAuthUserRepo{})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/login", `{"password":"super-secret"}`))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), "VALIDATION_ERROR")
}

func TestAuthHandler_Login_InvalidCredentials_Returns401(t *testing.T) {
	repo := &mockAuthUserRepo{byEmail: existingAuthUser("super-secret")}
	r := newAuthRouter(repo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authRequest(http.MethodPost, "/auth/login", `{
		"email": "ana@example.com",
		"password": "wrong-password"
	}`))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "INVALID_PASSWORD")
}

func assertErrorCode(t *testing.T, raw []byte, want string) {
	t.Helper()

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != want {
		t.Fatalf("error code = %s, want %s; body=%s", body.Error.Code, want, string(raw))
	}
}
