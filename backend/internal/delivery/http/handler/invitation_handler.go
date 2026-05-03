package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/internal/domain/user"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
)

type InvitationHandler struct {
	funds       fund.Repository
	members     fund.MemberRepository
	invitations fund.InvitationRepository
	users       user.Repository
	notifSvc    *ucnotif.Service
}

func NewInvitationHandler(
	funds fund.Repository,
	members fund.MemberRepository,
	invitations fund.InvitationRepository,
	users user.Repository,
	notifSvc *ucnotif.Service,
) *InvitationHandler {
	return &InvitationHandler{
		funds:       funds,
		members:     members,
		invitations: invitations,
		users:       users,
		notifSvc:    notifSvc,
	}
}

type createInvitationRequest struct {
	UserID         string `json:"user_id"`
	Email          string `json:"email"`
	DocumentType   string `json:"document_type"`
	DocumentNumber string `json:"document_number"`
	Message        string `json:"message"`
	ExpiresInDays  int    `json:"expires_in_days"`
}

func (h *InvitationHandler) Create(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	inviterID, _ := uuid.Parse(userIDStr)

	requester, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, inviterID)
	if err != nil || requester == nil || !requester.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundAdmin})
		return
	}

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return
	}

	var req createInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	inviteeID, err := h.resolveInvitee(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	if inviteeID == inviterID {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "SELF_INVITE", "message": "you cannot invite yourself"}})
		return
	}

	existingMember, _ := h.members.GetByFundAndUser(c.Request.Context(), fundID, inviteeID)
	if existingMember != nil && existingMember.Status == fund.MemberStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": apperror.ErrAlreadyMember})
		return
	}

	existingInvitation, err := h.invitations.FindPending(c.Request.Context(), fundID, inviteeID)
	if err != nil {
		respondError(c, err)
		return
	}
	if existingInvitation != nil {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "INVITATION_ALREADY_PENDING", "message": "there is already a pending invitation for this user"}})
		return
	}

	expiresInDays := req.ExpiresInDays
	if expiresInDays <= 0 {
		expiresInDays = 7
	}
	now := time.Now().UTC()
	invitation := &fund.FundInvitation{
		ID:        idgen.New(),
		FundID:    fundID,
		InviterID: inviterID,
		InviteeID: inviteeID,
		Status:    fund.InvitationStatusPending,
		Message:   req.Message,
		ExpiresAt: now.Add(time.Duration(expiresInDays) * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.invitations.Create(c.Request.Context(), invitation); err != nil {
		respondError(c, err)
		return
	}

	h.notifSvc.NotifyOne(
		c.Request.Context(),
		inviteeID,
		&fundID,
		notification.TypeFundInvitation,
		"Nueva invitación a un fondo",
		fmt.Sprintf("Fuiste invitado al fondo '%s'.", f.Name),
	)

	c.JSON(http.StatusCreated, gin.H{"data": invitation})
}

func (h *InvitationHandler) ListMine(c *gin.Context) {
	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	invitations, err := h.invitations.ListByInvitee(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	if invitations == nil {
		invitations = []*fund.FundInvitation{}
	}

	c.JSON(http.StatusOK, gin.H{"data": invitations})
}

func (h *InvitationHandler) ListByFund(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	member, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, userID)
	if err != nil || member == nil || !member.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundAdmin})
		return
	}

	invitations, err := h.invitations.ListByFund(c.Request.Context(), fundID)
	if err != nil {
		respondError(c, err)
		return
	}
	if invitations == nil {
		invitations = []*fund.FundInvitation{}
	}

	c.JSON(http.StatusOK, gin.H{"data": invitations})
}

func (h *InvitationHandler) Accept(c *gin.Context) {
	h.respond(c, fund.InvitationStatusAccepted)
}

func (h *InvitationHandler) Reject(c *gin.Context) {
	h.respond(c, fund.InvitationStatusRejected)
}

func (h *InvitationHandler) respond(c *gin.Context, status fund.InvitationStatus) {
	invitationID, err := uuid.Parse(c.Param("invitation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	invitation, err := h.invitations.GetByID(c.Request.Context(), invitationID)
	if err != nil {
		respondError(c, err)
		return
	}
	if invitation == nil || invitation.InviteeID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrNotFound})
		return
	}
	if invitation.Status != fund.InvitationStatusPending {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "INVITATION_NOT_PENDING", "message": "invitation is not pending"}})
		return
	}

	now := time.Now().UTC()
	if now.After(invitation.ExpiresAt) {
		invitation.Status = fund.InvitationStatusExpired
		invitation.RespondedAt = &now
		_ = h.invitations.Update(c.Request.Context(), invitation)
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "INVITATION_EXPIRED", "message": "invitation has expired"}})
		return
	}

	invitation.Status = status
	invitation.RespondedAt = &now
	if err := h.invitations.Update(c.Request.Context(), invitation); err != nil {
		respondError(c, err)
		return
	}

	if status == fund.InvitationStatusAccepted {
		existing, _ := h.members.GetByFundAndUser(c.Request.Context(), invitation.FundID, userID)
		if existing == nil {
			member := &fund.FundMember{
				ID:       idgen.New(),
				FundID:   invitation.FundID,
				UserID:   userID,
				Role:     fund.RoleMember,
				Status:   fund.MemberStatusActive,
				JoinedAt: now,
			}
			if err := h.members.Add(c.Request.Context(), member); err != nil {
				respondError(c, err)
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": invitation})
}

func (h *InvitationHandler) resolveInvitee(ctx context.Context, req createInvitationRequest) (uuid.UUID, error) {
	if req.UserID != "" {
		targetID, err := uuid.Parse(req.UserID)
		if err != nil {
			return uuid.Nil, apperror.ErrBadRequest
		}
		return targetID, nil
	}

	if req.Email != "" {
		u, err := h.users.GetByEmail(ctx, req.Email)
		if err != nil {
			return uuid.Nil, err
		}
		if u == nil {
			return uuid.Nil, apperror.ErrNotFound
		}
		return u.ID, nil
	}

	if req.DocumentType != "" || req.DocumentNumber != "" {
		if req.DocumentType == "" || req.DocumentNumber == "" {
			return uuid.Nil, apperror.ErrBadRequest
		}
		if msg := validateDocumentNumber(req.DocumentType, req.DocumentNumber); msg != "" {
			return uuid.Nil, apperror.ErrBadRequest
		}
		u, err := h.users.GetByDocument(ctx, user.DocumentType(req.DocumentType), req.DocumentNumber)
		if err != nil {
			return uuid.Nil, err
		}
		if u == nil {
			return uuid.Nil, apperror.ErrNotFound
		}
		return u.ID, nil
	}

	return uuid.Nil, apperror.ErrBadRequest
}
