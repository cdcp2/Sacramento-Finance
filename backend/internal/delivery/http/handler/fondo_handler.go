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
	"github.com/sacramento-finance/backend/internal/infrastructure/repository"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/money"
	ucfondo "github.com/sacramento-finance/backend/internal/usecase/fondo"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
)

type FondoHandler struct {
	accrueInterest *ucfondo.AccrueInterestUseCase
	withdraw       *ucfondo.WithdrawUseCase
	fondo          fund.FondoRepository
	ledger         *repository.LedgerRepo
	funds          fund.Repository
	members        fund.MemberRepository
	notifSvc       *ucnotif.Service
}

func NewFondoHandler(
	accrueInterest *ucfondo.AccrueInterestUseCase,
	withdraw *ucfondo.WithdrawUseCase,
	fondo fund.FondoRepository,
	ledger *repository.LedgerRepo,
	funds fund.Repository,
	members fund.MemberRepository,
	notifSvc *ucnotif.Service,
) *FondoHandler {
	return &FondoHandler{
		accrueInterest: accrueInterest,
		withdraw:       withdraw,
		fondo:          fondo,
		ledger:         ledger,
		funds:          funds,
		members:        members,
		notifSvc:       notifSvc,
	}
}

// GetConfig returns the fondo de ahorro config for a fund.
// GET /funds/:fund_id/fondo
func (h *FondoHandler) GetConfig(c *gin.Context) {
	fundID, f, _, ok := h.resolveMember(c)
	if !ok {
		return
	}
	if f.Type != fund.FundTypeFondoAhorro {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a fondo de ahorro fund"}})
		return
	}
	cfg, err := h.fondo.GetByFundID(c.Request.Context(), fundID)
	if err != nil || cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrNotFound})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cfg})
}

// GetMyBalance returns the calling user's accumulated balance in this fondo.
// GET /funds/:fund_id/fondo/my-balance
func (h *FondoHandler) GetMyBalance(c *gin.Context) {
	fundID, f, _, ok := h.resolveMember(c)
	if !ok {
		return
	}
	if f.Type != fund.FundTypeFondoAhorro {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a fondo de ahorro fund"}})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	balance, err := h.ledger.GetMemberBalance(c.Request.Context(), fundID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"balance": balance}})
}

// WithdrawalPreview returns the penalty calculation for a prospective withdrawal.
// GET /funds/:fund_id/fondo/withdrawal-preview?amount=100000
func (h *FondoHandler) WithdrawalPreview(c *gin.Context) {
	_, f, _, ok := h.resolveMember(c)
	if !ok {
		return
	}
	if f.Type != fund.FundTypeFondoAhorro {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a fondo de ahorro fund"}})
		return
	}

	amountStr := c.Query("amount")
	if amountStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "MISSING_AMOUNT", "message": "amount query param required"}})
		return
	}
	amount, err := money.FromString(amountStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_AMOUNT", "message": "invalid amount"}})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	preview, err := h.withdraw.Preview(c.Request.Context(), f, userID, amount)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": preview})
}

// AccrueInterest calculates and records interest on the fund balance (admin only).
// POST /funds/:fund_id/fondo/accrue-interest
func (h *FondoHandler) AccrueInterest(c *gin.Context) {
	_, f, _, ok := h.resolveAdmin(c)
	if !ok {
		return
	}
	if f.Type != fund.FundTypeFondoAhorro {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a fondo de ahorro fund"}})
		return
	}

	entry, err := h.accrueInterest.Execute(c.Request.Context(), f)
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
		h.notifSvc.NotifyAll(ctx, members, nil, &f.ID,
			notification.TypeInterestAccrued,
			"Interés aplicado",
			fmt.Sprintf("Se aplicó interés de %s COP al fondo '%s'.", entry.Amount.StringFixed(0), f.Name),
		)
	}()

	c.JSON(http.StatusOK, gin.H{"data": entry})
}

// Withdraw records a member withdrawal from their fondo balance.
// POST /funds/:fund_id/fondo/withdraw
// Body: {"amount": "50000"}
func (h *FondoHandler) Withdraw(c *gin.Context) {
	_, f, _, ok := h.resolveMember(c)
	if !ok {
		return
	}
	if f.Type != fund.FundTypeFondoAhorro {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "WRONG_FUND_TYPE", "message": "not a fondo de ahorro fund"}})
		return
	}

	var req struct {
		Amount string `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	amount, err := money.FromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_AMOUNT", "message": "invalid amount"}})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	result, err := h.withdraw.Execute(c.Request.Context(), f, userID, amount)
	if err != nil {
		respondError(c, err)
		return
	}

	go h.notifSvc.NotifyOne(
		context.Background(), userID, &f.ID,
		notification.TypeWithdrawal,
		"Retiro procesado",
		fmt.Sprintf("Tu retiro de %s COP en '%s' fue procesado.", amount.StringFixed(0), f.Name),
	)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"withdrawal": result.WithdrawalEntry,
		"penalty":    result.PenaltyEntry,
	}})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func (h *FondoHandler) resolveMember(c *gin.Context) (uuid.UUID, *fund.Fund, *fund.FundMember, bool) {
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

func (h *FondoHandler) resolveAdmin(c *gin.Context) (uuid.UUID, *fund.Fund, *fund.FundMember, bool) {
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
