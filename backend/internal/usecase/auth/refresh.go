package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

type RefreshUseCase struct {
	jwtSecret string
	accessTTL time.Duration
}

func NewRefreshUseCase(jwtSecret string, accessTTL time.Duration) *RefreshUseCase {
	return &RefreshUseCase{jwtSecret: jwtSecret, accessTTL: accessTTL}
}

// Execute validates a refresh token and returns a new access token.
func (uc *RefreshUseCase) Execute(refreshToken string) (*TokenPair, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperror.ErrInvalidToken
		}
		return []byte(uc.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, apperror.ErrInvalidToken
	}
	if claims.TokenKind != "refresh" {
		return nil, apperror.ErrInvalidToken
	}

	now := time.Now().UTC()
	accessExpiry := now.Add(uc.accessTTL)

	newClaims := Claims{
		UserID:    claims.UserID,
		FullName:  claims.FullName,
		TokenKind: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
		},
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	signed, err := newToken.SignedString([]byte(uc.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("signing token: %w", err)
	}

	return &TokenPair{
		AccessToken:  signed,
		RefreshToken: refreshToken, // same refresh token — still valid until its own expiry
		TokenType:    "Bearer",
		ExpiresIn:    int64(uc.accessTTL.Seconds()),
	}, nil
}
