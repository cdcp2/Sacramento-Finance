package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sacramento-finance/backend/internal/usecase/auth"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

const UserIDKey = "user_id"

// Auth validates the Bearer JWT and sets user_id in the context.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": apperror.ErrUnauthorized})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims := &auth.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apperror.ErrInvalidToken
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": apperror.ErrInvalidToken})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

// GetUserID retrieves the authenticated user ID from context.
func GetUserID(c *gin.Context) (string, bool) {
	id, ok := c.Get(UserIDKey)
	if !ok {
		return "", false
	}
	return id.(string), true
}
