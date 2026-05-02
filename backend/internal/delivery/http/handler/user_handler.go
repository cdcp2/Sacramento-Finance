package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/user"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

type UserHandler struct {
	users user.Repository
}

func NewUserHandler(users user.Repository) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) Me(c *gin.Context) {
	userIDStr, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": apperror.ErrUnauthorized})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": apperror.ErrUnauthorized})
		return
	}

	u, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": u.Sanitized()})
}
