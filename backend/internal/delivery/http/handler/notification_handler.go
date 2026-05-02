package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

type NotificationHandler struct {
	repo notification.Repository
}

func NewNotificationHandler(repo notification.Repository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

// List returns paginated notifications for the authenticated user.
// Unread notifications appear first.
// GET /notifications?limit=20&offset=0
func (h *NotificationHandler) List(c *gin.Context) {
	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	notifications, err := h.repo.ListByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}

	unread, _ := h.repo.CountUnread(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"notifications": notifications,
			"unread_count":  unread,
		},
	})
}

// MarkRead marks a single notification as read (must belong to caller).
// PATCH /notifications/:notif_id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	notifID, err := uuid.Parse(c.Param("notif_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	if err := h.repo.MarkRead(c.Request.Context(), notifID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// MarkAllRead marks every unread notification for the caller as read.
// POST /notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	if err := h.repo.MarkAllRead(c.Request.Context(), userID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}
