package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/pkg/apperror"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	ucvaca "github.com/sacramento-finance/backend/internal/usecase/vaca"
)

type VacaHandler struct {
	getProgress *ucvaca.GetProgressUseCase
	distribute  *ucvaca.DistributeUseCase
	vaca        fund.VacaRepository
	funds       fund.Repository
	members     fund.MemberRepository
	notifSvc    *ucnotif.Service
}

func NewVacaHandler(
	getProgress *ucvaca.GetProgressUseCase,
	distribute *ucvaca.DistributeUseCase,
	vaca fund.VacaRepository,
	funds fund.Repository,
	members fund.MemberRepository,
	notifSvc *ucnotif.Service,
) *VacaHandler {
	return &VacaHandler{
		getProgress: getProgress,
		distribute:  distribute,
		vaca:        vaca,
		funds:       funds,
		members:     members,
		notifSvc:    notifSvc,
	}
}

// GetProgress returns vaca config + goal progress.
// GET /funds/:fund_id/vaca
func (h *VacaHandler) GetProgress(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	member, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundMember})
		return
	}

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return
	}

	if f.Type != fund.FundTypeVaca {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a vaca fund"}})
		return
	}

	result, err := h.getProgress.Execute(c.Request.Context(), f)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// Distribute divides the fund balance equally among active members (admin only).
// POST /funds/:fund_id/vaca/distribute
func (h *VacaHandler) Distribute(c *gin.Context) {
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

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return
	}

	if f.Type != fund.FundTypeVaca {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a vaca fund"}})
		return
	}

	if f.Rules.IsGoverned() {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code":    "USE_PROPOSAL",
			"message": "this fund uses governance voting; create a distribute_vaca proposal instead",
		}})
		return
	}

	result, err := h.distribute.Execute(c.Request.Context(), f)
	if err != nil {
		respondError(c, err)
		return
	}

	go func() {
		ctx := context.Background()
		members, err := h.members.ListByFund(ctx, fundID)
		if err != nil {
			return
		}
		h.notifSvc.NotifyAll(ctx, members, nil, &fundID,
			notification.TypeGoalReached,
			"¡Distribución completada!",
			fmt.Sprintf("El fondo '%s' fue distribuido: %s COP por miembro.", f.Name, result.SharePerMember.StringFixed(0)),
		)
	}()

	c.JSON(http.StatusOK, gin.H{"data": result})
}
