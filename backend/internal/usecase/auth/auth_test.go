package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/user"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	byID       *user.User
	byDocument *user.User
	byEmail    *user.User
	created    *user.User
	updated    *user.User
	createErr  error
}

func (m *mockUserRepo) Create(_ context.Context, u *user.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = u
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*user.User, error) {
	return m.byID, nil
}

func (m *mockUserRepo) GetByDocument(_ context.Context, _ user.DocumentType, _ string) (*user.User, error) {
	return m.byDocument, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return m.byEmail, nil
}

func (m *mockUserRepo) Update(_ context.Context, u *user.User) error {
	m.updated = u
	return nil
}

func registerInput() RegisterInput {
	return RegisterInput{
		DocumentType:   user.DocumentCedulaCiudadania,
		DocumentNumber: "1234567890",
		Email:          "ana@example.com",
		Phone:          "+573001234567",
		FullName:       "Ana Perez",
		Password:       "super-secret",
	}
}

func authUser(password string) *user.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return &user.User{
		ID:                 uuid.New(),
		DocumentType:       user.DocumentCedulaCiudadania,
		DocumentNumber:     "1234567890",
		Email:              "ana@example.com",
		Phone:              "+573001234567",
		FullName:           "Ana Perez",
		PasswordHash:       string(hash),
		VerificationStatus: user.VerificationNone,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

func TestRegister_Success_CreatesUserWithHashedPasswordAndSanitizedResponse(t *testing.T) {
	repo := &mockUserRepo{}
	uc := NewRegisterUseCase(repo)
	in := registerInput()

	result, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repo.created == nil {
		t.Fatal("expected user to be created")
	}
	if repo.created.ID == uuid.Nil {
		t.Error("created user ID should be set")
	}
	if repo.created.DocumentType != in.DocumentType || repo.created.DocumentNumber != in.DocumentNumber {
		t.Errorf("created document = (%s, %s), want (%s, %s)",
			repo.created.DocumentType, repo.created.DocumentNumber, in.DocumentType, in.DocumentNumber)
	}
	if repo.created.Email != in.Email {
		t.Errorf("created email = %s, want %s", repo.created.Email, in.Email)
	}
	if repo.created.PasswordHash == in.Password {
		t.Error("password should be stored as a hash, not plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.created.PasswordHash), []byte(in.Password)); err != nil {
		t.Fatalf("stored password hash does not match password: %v", err)
	}
	if repo.created.IsVerified {
		t.Error("new user should not be verified")
	}
	if repo.created.VerificationStatus != user.VerificationNone {
		t.Errorf("VerificationStatus = %s, want none", repo.created.VerificationStatus)
	}
	if repo.created.CreatedAt.IsZero() || repo.created.UpdatedAt.IsZero() {
		t.Error("timestamps should be set")
	}
	if result.ID != repo.created.ID {
		t.Errorf("public ID = %s, want %s", result.ID, repo.created.ID)
	}
	if result.Email != in.Email || result.FullName != in.FullName {
		t.Errorf("unexpected public user: %+v", result)
	}
}

func TestRegister_DuplicateDocument_ReturnsDuplicateCedula(t *testing.T) {
	repo := &mockUserRepo{byDocument: authUser("secret")}
	uc := NewRegisterUseCase(repo)

	_, err := uc.Execute(context.Background(), registerInput())
	if !errors.Is(err, apperror.ErrDuplicateCedula) {
		t.Fatalf("Execute() error = %v, want ErrDuplicateCedula", err)
	}
	if repo.created != nil {
		t.Error("Create should not be called for duplicate document")
	}
}

func TestRegister_DuplicateEmail_ReturnsDuplicateEmail(t *testing.T) {
	repo := &mockUserRepo{byEmail: authUser("secret")}
	uc := NewRegisterUseCase(repo)

	_, err := uc.Execute(context.Background(), registerInput())
	if !errors.Is(err, apperror.ErrDuplicateEmail) {
		t.Fatalf("Execute() error = %v, want ErrDuplicateEmail", err)
	}
	if repo.created != nil {
		t.Error("Create should not be called for duplicate email")
	}
}

func TestRegister_CreateError_ReturnsError(t *testing.T) {
	wantErr := errors.New("insert user failed")
	repo := &mockUserRepo{createErr: wantErr}
	uc := NewRegisterUseCase(repo)

	_, err := uc.Execute(context.Background(), registerInput())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestLogin_WithEmail_Success_ReturnsValidTokens(t *testing.T) {
	secret := "test-secret"
	ttl := 15 * time.Minute
	u := authUser("super-secret")
	repo := &mockUserRepo{byEmail: u}
	uc := NewLoginUseCase(repo, secret, ttl)

	tokens, err := uc.Execute(context.Background(), LoginInput{
		Email:    u.Email,
		Password: "super-secret",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens")
	}
	if tokens.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want Bearer", tokens.TokenType)
	}
	if tokens.ExpiresIn != int64(ttl.Seconds()) {
		t.Errorf("ExpiresIn = %d, want %d", tokens.ExpiresIn, int64(ttl.Seconds()))
	}

	claims := parseTokenClaims(t, tokens.AccessToken, secret)
	if claims.UserID != u.ID.String() {
		t.Errorf("claims.UserID = %s, want %s", claims.UserID, u.ID)
	}
	if claims.Subject != u.ID.String() {
		t.Errorf("claims.Subject = %s, want %s", claims.Subject, u.ID)
	}
	if claims.FullName != u.FullName {
		t.Errorf("claims.FullName = %s, want %s", claims.FullName, u.FullName)
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatal("token should include issued and expiry claims")
	}
	if claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) != ttl {
		t.Errorf("access token TTL = %s, want %s", claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time), ttl)
	}

	refreshClaims := parseTokenClaims(t, tokens.RefreshToken, secret)
	if refreshClaims.Subject != u.ID.String() {
		t.Errorf("refresh subject = %s, want %s", refreshClaims.Subject, u.ID)
	}
	if refreshClaims.ExpiresAt.Time.Sub(refreshClaims.IssuedAt.Time) != 7*24*time.Hour {
		t.Errorf("refresh token TTL = %s, want 168h", refreshClaims.ExpiresAt.Time.Sub(refreshClaims.IssuedAt.Time))
	}
}

func TestLogin_WithDocumentFallback_Success(t *testing.T) {
	secret := "test-secret"
	u := authUser("super-secret")
	repo := &mockUserRepo{byDocument: u}
	uc := NewLoginUseCase(repo, secret, 10*time.Minute)

	tokens, err := uc.Execute(context.Background(), LoginInput{
		DocumentType:   u.DocumentType,
		DocumentNumber: u.DocumentNumber,
		Password:       "super-secret",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestLogin_EmailNotFoundFallsBackToDocument(t *testing.T) {
	secret := "test-secret"
	u := authUser("super-secret")
	repo := &mockUserRepo{byDocument: u}
	uc := NewLoginUseCase(repo, secret, 10*time.Minute)

	tokens, err := uc.Execute(context.Background(), LoginInput{
		Email:          "missing@example.com",
		DocumentType:   u.DocumentType,
		DocumentNumber: u.DocumentNumber,
		Password:       "super-secret",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access token from document fallback")
	}
}

func TestLogin_UserNotFound_ReturnsInvalidPassword(t *testing.T) {
	uc := NewLoginUseCase(&mockUserRepo{}, "secret", 10*time.Minute)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "missing@example.com",
		Password: "super-secret",
	})
	if !errors.Is(err, apperror.ErrInvalidPassword) {
		t.Fatalf("Execute() error = %v, want ErrInvalidPassword", err)
	}
}

func TestLogin_WrongPassword_ReturnsInvalidPassword(t *testing.T) {
	u := authUser("super-secret")
	uc := NewLoginUseCase(&mockUserRepo{byEmail: u}, "secret", 10*time.Minute)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    u.Email,
		Password: "wrong-password",
	})
	if !errors.Is(err, apperror.ErrInvalidPassword) {
		t.Fatalf("Execute() error = %v, want ErrInvalidPassword", err)
	}
}

func TestLogin_NoIdentifier_ReturnsInvalidPassword(t *testing.T) {
	uc := NewLoginUseCase(&mockUserRepo{}, "secret", 10*time.Minute)

	_, err := uc.Execute(context.Background(), LoginInput{Password: "super-secret"})
	if !errors.Is(err, apperror.ErrInvalidPassword) {
		t.Fatalf("Execute() error = %v, want ErrInvalidPassword", err)
	}
}

func parseTokenClaims(t *testing.T, rawToken, secret string) *Claims {
	t.Helper()

	token, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			t.Fatalf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if !token.Valid {
		t.Fatal("token should be valid")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		t.Fatalf("claims type = %T, want *Claims", token.Claims)
	}
	return claims
}
