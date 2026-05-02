package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/pkg/apperror"
	uccirculo "github.com/sacramento-finance/backend/internal/usecase/circulo"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
)

type CirculoHandler struct {
	assignOrder *uccirculo.AssignPayoutOrderUseCase
	closeRound  *uccirculo.CloseRoundUseCase
	circulo     fund.CirculoRepository
	payouts     ledger.PayoutRepository
	funds       fund.Repository
	members     fund.MemberRepository
	notifSvc    *ucnotif.Service
}

func NewCirculoHandler(
	assignOrder *uccirculo.AssignPayoutOrderUseCase,
	closeRound *uccirculo.CloseRoundUseCase,
	circulo fund.CirculoRepository,
	payouts ledger.PayoutRepository,
	funds fund.Repository,
	members fund.MemberRepository,
	notifSvc *ucnotif.Service,
) *CirculoHandler {
	return &CirculoHandler{
		assignOrder: assignOrder,
		closeRound:  closeRound,
		circulo:     circulo,
		payouts:     payouts,
		funds:       funds,
		members:     members,
		notifSvc:    notifSvc,
	}
}

// GetConfig returns the circulo-specific config for a fund.
// GET /funds/:fund_id/circulo
func (h *CirculoHandler) GetConfig(c *gin.Context) {
	fundID, f, _, ok := h.resolveMemberFund(c)
	if !ok {
		return
	}

	if f.Type != fund.FundTypeCirculo {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a circulo fund"}})
		return
	}

	cfg, err := h.circulo.GetByFundID(c.Request.Context(), fundID)
	if err != nil || cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrNotFound})
		return
	}

	members, _ := h.members.ListByFund(c.Request.Context(), fundID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"config": cfg, "members": members}})
}

// AssignPayoutOrder sets payout_order for each member (admin only).
// POST /funds/:fund_id/circulo/payout-order
// Body: {"randomize": true} OR {"assignments": [{"member_id": "...", "order": 1}]}
func (h *CirculoHandler) AssignPayoutOrder(c *gin.Context) {
	_, f, _, ok := h.resolveAdminFund(c)
	if !ok {
		return
	}

	var req struct {
		Randomize   bool                        `json:"randomize"`
		Assignments []uccirculo.MemberOrder     `json:"assignments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	if err := h.assignOrder.Execute(c.Request.Context(), f, req.Assignments, req.Randomize); err != nil {
		respondError(c, err)
		return
	}

	members, _ := h.members.ListByFund(c.Request.Context(), f.ID)
	c.JSON(http.StatusOK, gin.H{"data": members})
}

// ListPayouts returns all payout records for a circulo fund.
// GET /funds/:fund_id/circulo/payouts
func (h *CirculoHandler) ListPayouts(c *gin.Context) {
	fundID, f, _, ok := h.resolveMemberFund(c)
	if !ok {
		return
	}

	if f.Type != fund.FundTypeCirculo {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a circulo fund"}})
		return
	}

	payouts, err := h.payouts.ListByFund(c.Request.Context(), fundID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": payouts})
}

// CloseRound closes the current round and distributes the pot to the beneficiary (admin only).
// POST /funds/:fund_id/circulo/close-round
func (h *CirculoHandler) CloseRound(c *gin.Context) {
	_, f, _, ok := h.resolveAdminFund(c)
	if !ok {
		return
	}

	result, err := h.closeRound.Execute(c.Request.Context(), f)
	if err != nil {
		respondError(c, err)
		return
	}

	go func() {
		ctx := context.Background()
		members, err := h.members.ListByFund(ctx, f.ID)
		if err != nil {
			return
		}
		// Find recipient UserID from payout RecipientID (FundMember.ID)
		var recipientUserID *uuid.UUID
		for _, m := range members {
			if m.ID == result.Payout.RecipientID {
				uid := m.UserID
				recipientUserID = &uid
				break
			}
		}
		roundStr := fmt.Sprintf("%d", result.Payout.RoundNumber)
		if recipientUserID != nil {
			h.notifSvc.NotifyOne(ctx, *recipientUserID, &f.ID,
				notification.TypePayoutReceived,
				"¡Recibiste el pote!",
				fmt.Sprintf("Recibiste el pote de la ronda %s en '%s': %s COP", roundStr, f.Name, result.Payout.Amount.StringFixed(0)),
			)
		}
		// Notify the rest of the members
		h.notifSvc.NotifyAll(ctx, members, recipientUserID, &f.ID,
			notification.TypeRoundClosed,
			"Ronda cerrada",
			fmt.Sprintf("La ronda %s del círculo '%s' fue cerrada.", roundStr, f.Name),
		)
	}()

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"payout":    result.Payout,
			"config":    result.Config,
			"completed": result.Completed,
		},
	})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// resolveAdminFund resolves fund_id and verifies the caller is an admin member.
func (h *CirculoHandler) resolveAdminFund(c *gin.Context) (uuid.UUID, *fund.Fund, *fund.FundMember, bool) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return uuid.Nil, nil, nil, false
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	member, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, userID)
	if err != nil || member == nil || !member.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundAdmin})
		return uuid.Nil, nil, nil, false
	}

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return uuid.Nil, nil, nil, false
	}

	return fundID, f, member, true
}

// resolveMemberFund resolves fund_id and verifies the caller is any active member.
func (h *CirculoHandler) resolveMemberFund(c *gin.Context) (uuid.UUID, *fund.Fund, *fund.FundMember, bool) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return uuid.Nil, nil, nil, false
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	member, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundMember})
		return uuid.Nil, nil, nil, false
	}

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return uuid.Nil, nil, nil, false
	}

	return fundID, f, member, true
}
