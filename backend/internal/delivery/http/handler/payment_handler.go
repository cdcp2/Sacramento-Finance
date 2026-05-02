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
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	ucpayment "github.com/sacramento-finance/backend/internal/usecase/payment"
)

type PaymentHandler struct {
	recordPayment *ucpayment.RecordPaymentUseCase
	waivePayment  *ucpayment.WaivePaymentUseCase
	payments      *repository.PaymentRepo
	ledger        *repository.LedgerRepo
	funds         fund.Repository
	members       fund.MemberRepository
	notifSvc      *ucnotif.Service
}

func NewPaymentHandler(
	recordPayment *ucpayment.RecordPaymentUseCase,
	waivePayment *ucpayment.WaivePaymentUseCase,
	payments *repository.PaymentRepo,
	ledger *repository.LedgerRepo,
	funds fund.Repository,
	members fund.MemberRepository,
	notifSvc *ucnotif.Service,
) *PaymentHandler {
	return &PaymentHandler{
		recordPayment: recordPayment,
		waivePayment:  waivePayment,
		payments:      payments,
		ledger:        ledger,
		funds:         funds,
		members:       members,
		notifSvc:      notifSvc,
	}
}

// ListMine returns all payments for the authenticated user in a fund.
// GET /funds/:fund_id/payments
func (h *PaymentHandler) ListMine(c *gin.Context) {
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

	payments, err := h.payments.ListByMember(c.Request.Context(), member.ID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": payments})
}

// ListAll returns all payments in a fund (admin only).
// GET /funds/:fund_id/payments/all
func (h *PaymentHandler) ListAll(c *gin.Context) {
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

	payments, err := h.payments.ListByFund(c.Request.Context(), fundID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": payments})
}

// Pay records a payment for a specific payment slot.
// POST /funds/:fund_id/payments/:payment_id/pay
func (h *PaymentHandler) Pay(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}
	paymentID, err := uuid.Parse(c.Param("payment_id"))
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

	if f.Status != fund.FundStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrFundNotActive})
		return
	}

	result, err := h.recordPayment.Execute(c.Request.Context(), paymentID, member.ID, userID, f)
	if err != nil {
		respondError(c, err)
		return
	}

	go h.notifSvc.NotifyOne(
		context.Background(), userID, &f.ID,
		notification.TypePaymentConfirmed,
		"Aporte confirmado",
		fmt.Sprintf("Tu aporte del periodo %d en '%s' fue confirmado.", result.Payment.PeriodNumber, f.Name),
	)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"payment": result.Payment,
			"entry":   result.Entry,
		},
	})
}

// Waive marks a payment as waived (admin only).
// POST /funds/:fund_id/payments/:payment_id/waive
func (h *PaymentHandler) Waive(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}
	paymentID, err := uuid.Parse(c.Param("payment_id"))
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

	p, err := h.waivePayment.Execute(c.Request.Context(), paymentID, fundID)
	if err != nil {
		respondError(c, err)
		return
	}

	// Notify the member whose payment was waived
	go func() {
		ctx := context.Background()
		waivedMember, err := h.members.GetByID(ctx, p.MemberID)
		if err != nil || waivedMember == nil {
			return
		}
		fundName := fundID.String()
		if f, err := h.funds.GetByID(ctx, fundID); err == nil && f != nil {
			fundName = f.Name
		}
		h.notifSvc.NotifyOne(
			ctx, waivedMember.UserID, &fundID,
			notification.TypePaymentWaived,
			"Pago condonado",
			fmt.Sprintf("Tu pago del periodo %d en '%s' fue condonado por el administrador.", p.PeriodNumber, fundName),
		)
	}()

	c.JSON(http.StatusOK, gin.H{"data": p})
}

// Ledger returns paginated ledger entries for a fund.
// GET /funds/:fund_id/ledger
func (h *PaymentHandler) Ledger(c *gin.Context) {
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

	entries, err := h.ledger.ListByFund(c.Request.Context(), fundID, 50, 0)
	if err != nil {
		respondError(c, err)
		return
	}

	balance, _ := h.ledger.GetFundBalance(c.Request.Context(), fundID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"balance": balance,
			"entries": entries,
		},
	})
}
