package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sacramento-finance/backend/internal/domain/user"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"golang.org/x/crypto/bcrypt"
)

type LoginInput struct {
	DocumentType   user.DocumentType // optional if Email provided
	DocumentNumber string            // optional if Email provided
	Email          string            // optional if Document provided
	Password       string
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Claims struct {
	UserID    string `json:"user_id"`
	FullName  string `json:"full_name"`
	TokenKind string `json:"kind"` // "access" | "refresh"
	jwt.RegisteredClaims
}

type LoginUseCase struct {
	users     user.Repository
	jwtSecret string
	accessTTL time.Duration
}

func NewLoginUseCase(users user.Repository, jwtSecret string, accessTTL time.Duration) *LoginUseCase {
	return &LoginUseCase{users: users, jwtSecret: jwtSecret, accessTTL: accessTTL}
}

func (uc *LoginUseCase) Execute(ctx context.Context, in LoginInput) (*TokenPair, error) {
	var u *user.User

	if in.Email != "" {
		u, _ = uc.users.GetByEmail(ctx, in.Email)
	}
	if u == nil && in.DocumentType != "" && in.DocumentNumber != "" {
		u, _ = uc.users.GetByDocument(ctx, in.DocumentType, in.DocumentNumber)
	}
	if u == nil {
		// Same error regardless of whether user exists — prevents enumeration
		return nil, apperror.ErrInvalidPassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)); err != nil {
		return nil, apperror.ErrInvalidPassword
	}

	now := time.Now().UTC()
	accessExpiry := now.Add(uc.accessTTL)

	claims := Claims{
		UserID:    u.ID.String(),
		FullName:  u.FullName,
		TokenKind: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(uc.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("signing token: %w", err)
	}

	refreshExpiry := now.Add(7 * 24 * time.Hour)
	refreshClaims := Claims{
		UserID:    u.ID.String(),
		TokenKind: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefresh, err := refreshToken.SignedString([]byte(uc.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  signed,
		RefreshToken: signedRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(uc.accessTTL.Seconds()),
	}, nil
}
