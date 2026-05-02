package handler

import (
	"net/http"
	"strings"

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

type updateMeRequest struct {
	Phone    *string `json:"phone"`
	FullName *string `json:"full_name"`
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

func (h *UserHandler) UpdateMe(c *gin.Context) {
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

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	if req.Phone == nil && req.FullName == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "provide phone or full_name",
		}})
		return
	}

	u, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrNotFound})
		return
	}

	if req.FullName != nil {
		fullName := strings.TrimSpace(*req.FullName)
		if len(fullName) < 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "full_name must be at least 3 characters",
			}})
			return
		}
		u.FullName = fullName
	}

	if req.Phone != nil {
		phone := strings.TrimSpace(*req.Phone)
		if len(phone) < 7 || len(phone) > 15 || !reOnlyDigits.MatchString(phone) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "phone must contain 7 to 15 digits",
			}})
			return
		}
		u.Phone = phone
	}

	if err := h.users.Update(c.Request.Context(), u); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": u.Sanitized()})
}
