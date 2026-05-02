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
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
	"github.com/sacramento-finance/backend/pkg/money"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	ucpayment "github.com/sacramento-finance/backend/internal/usecase/payment"
)

type FundHandler struct {
	funds            fund.Repository
	members          fund.MemberRepository
	generateSchedule *ucpayment.GenerateScheduleUseCase
	circuloRepo      fund.CirculoRepository
	vacaRepo         fund.VacaRepository
	fondoRepo        fund.FondoRepository
	notifSvc         *ucnotif.Service
}

func NewFundHandler(
	funds fund.Repository,
	members fund.MemberRepository,
	generateSchedule *ucpayment.GenerateScheduleUseCase,
	circuloRepo fund.CirculoRepository,
	vacaRepo fund.VacaRepository,
	fondoRepo fund.FondoRepository,
	notifSvc *ucnotif.Service,
) *FundHandler {
	return &FundHandler{
		funds:            funds,
		members:          members,
		generateSchedule: generateSchedule,
		circuloRepo:      circuloRepo,
		vacaRepo:         vacaRepo,
		fondoRepo:        fondoRepo,
		notifSvc:         notifSvc,
	}
}

type createFundRequest struct {
	Name               string               `json:"name" binding:"required,min=3,max=100"`
	Description        string               `json:"description"`
	Type               fund.FundType        `json:"type" binding:"required,oneof=circulo vaca fondo_ahorro"`
	ContributionAmount string               `json:"contribution_amount" binding:"required"`
	Frequency          fund.PeriodFrequency `json:"frequency" binding:"required,oneof=weekly biweekly monthly"`
	TotalPeriods       int                  `json:"total_periods" binding:"required,min=1"`
	StartDate          string               `json:"start_date" binding:"required"`
	PenaltyEnabled     bool                 `json:"penalty_enabled"`
	PenaltyType        fund.PenaltyType     `json:"penalty_type"`
	PenaltyAmount      string               `json:"penalty_amount"`
	GracePeriodDays    int                  `json:"grace_period_days"`
	MinMembers         int                  `json:"min_members" binding:"min=1"`
	MaxMembers         int                  `json:"max_members"`
	GovernanceType     fund.GovernanceType  `json:"governance_type" binding:"omitempty,oneof=admin_only majority unanimous"`
	VotingDeadlineHours int                 `json:"voting_deadline_hours"`

	// Circulo-specific
	PayoutOrderType string `json:"payout_order_type" binding:"omitempty,oneof=fixed randomized"`

	// Vaca-specific
	GoalAmount       string `json:"goal_amount"`
	GoalDescription  string `json:"goal_description"`
	DistributionType string `json:"distribution_type" binding:"omitempty,oneof=goal_reached unanimous_vote"`

	// Fondo de ahorro-specific
	InterestRate            string `json:"interest_rate"`
	EarlyWithdrawalPenalty  string `json:"early_withdrawal_penalty"`
}

func (h *FundHandler) Create(c *gin.Context) {
	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	var req createFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	amount, err := money.FromString(req.ContributionAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_AMOUNT", "message": "contribution_amount must be a valid decimal"}})
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		startDate, err = time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "start_date must be YYYY-MM-DD"}})
			return
		}
	}

	penaltyAmount := money.Zero
	if req.PenaltyAmount != "" {
		penaltyAmount, _ = money.FromString(req.PenaltyAmount)
	}

	minMembers := req.MinMembers
	if minMembers == 0 {
		minMembers = 2
	}
	maxMembers := req.MaxMembers
	if maxMembers == 0 {
		maxMembers = 50
	}
	governanceType := req.GovernanceType
	if governanceType == "" {
		governanceType = fund.GovernanceAdminOnly
	}
	votingDeadlineHours := req.VotingDeadlineHours
	if votingDeadlineHours == 0 {
		votingDeadlineHours = 48
	}

	f := &fund.Fund{
		ID:          idgen.New(),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Status:      fund.FundStatusDraft,
		CreatorID:   userID,
		Currency:    "COP",
		Rules: fund.Rules{
			ContributionAmount: amount,
			Frequency:          req.Frequency,
			TotalPeriods:       req.TotalPeriods,
			StartDate:          startDate,
			PenaltyEnabled:     req.PenaltyEnabled,
			PenaltyType:        req.PenaltyType,
			PenaltyAmount:      penaltyAmount,
			GracePeriodDays:     req.GracePeriodDays,
			MinMembers:          minMembers,
			MaxMembers:          maxMembers,
			GovernanceType:      governanceType,
			VotingDeadlineHours: votingDeadlineHours,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := h.funds.Create(c.Request.Context(), f); err != nil {
		respondError(c, err)
		return
	}

	// Auto-add creator as admin member
	member := &fund.FundMember{
		ID:       idgen.New(),
		FundID:   f.ID,
		UserID:   userID,
		Role:     fund.RoleAdmin,
		Status:   fund.MemberStatusActive,
		JoinedAt: time.Now().UTC(),
	}
	_ = h.members.Add(c.Request.Context(), member)

	// Save type-specific config
	ctx := c.Request.Context()
	switch f.Type {
	case fund.FundTypeCirculo:
		payoutOrderType := req.PayoutOrderType
		if payoutOrderType == "" {
			payoutOrderType = "fixed"
		}
		_ = h.circuloRepo.Create(ctx, &fund.CirculoConfig{
			FundID:          f.ID,
			PayoutOrderType: payoutOrderType,
			CurrentRound:    1,
			RoundsCompleted: 0,
		})

	case fund.FundTypeVaca:
		goalAmount := money.Zero
		if req.GoalAmount != "" {
			goalAmount, _ = money.FromString(req.GoalAmount)
		}
		distType := req.DistributionType
		if distType == "" {
			distType = "goal_reached"
		}
		_ = h.vacaRepo.Create(ctx, &fund.VacaConfig{
			FundID:           f.ID,
			GoalAmount:       goalAmount,
			GoalDescription:  req.GoalDescription,
			DistributionType: distType,
		})

	case fund.FundTypeFondoAhorro:
		interestRate := money.Zero
		if req.InterestRate != "" {
			interestRate, _ = money.FromString(req.InterestRate)
		}
		earlyPenalty := money.Zero
		if req.EarlyWithdrawalPenalty != "" {
			earlyPenalty, _ = money.FromString(req.EarlyWithdrawalPenalty)
		}
		_ = h.fondoRepo.Create(ctx, &fund.FondoConfig{
			FundID:                 f.ID,
			InterestRate:           interestRate,
			EarlyWithdrawalPenalty: earlyPenalty,
		})
	}

	c.JSON(http.StatusCreated, gin.H{"data": f})
}

func (h *FundHandler) Get(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": f})
}

func (h *FundHandler) ListMine(c *gin.Context) {
	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	funds, err := h.funds.ListByMember(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": funds})
}

func (h *FundHandler) UpdateRules(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	// Check admin role
	member, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundMember})
		return
	}
	if !member.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundAdmin})
		return
	}

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return
	}

	if f.Status != fund.FundStatusDraft {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "FUND_NOT_DRAFT",
			"message": "rules can only be modified while fund is in draft status",
		}})
		return
	}
	if f.Rules.IsGoverned() {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code":    "USE_PROPOSAL",
			"message": "this fund uses governance voting; create a change_rules proposal instead",
		}})
		return
	}

	var req createFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	if req.ContributionAmount != "" {
		if amount, err := money.FromString(req.ContributionAmount); err == nil {
			f.Rules.ContributionAmount = amount
		}
	}
	if req.TotalPeriods > 0 {
		f.Rules.TotalPeriods = req.TotalPeriods
	}
	if req.Frequency != "" {
		f.Rules.Frequency = req.Frequency
	}
	f.Rules.PenaltyEnabled = req.PenaltyEnabled
	if req.PenaltyAmount != "" {
		if pa, err := money.FromString(req.PenaltyAmount); err == nil {
			f.Rules.PenaltyAmount = pa
		}
	}
	f.Rules.GracePeriodDays = req.GracePeriodDays
	f.UpdatedAt = time.Now().UTC()

	if err := h.funds.Update(c.Request.Context(), f); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": f})
}

func (h *FundHandler) Activate(c *gin.Context) {
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

	if f.Rules.IsGoverned() {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code":    "USE_PROPOSAL",
			"message": "this fund uses governance voting; create an activate_fund proposal instead",
		}})
		return
	}
	if !f.CanTransitionTo(fund.FundStatusActive) {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrInvalidFundState})
		return
	}

	count, _ := h.members.CountActive(c.Request.Context(), fundID)
	if count < f.Rules.MinMembers {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "NOT_ENOUGH_MEMBERS",
			"message": "fund does not have the minimum required members",
		}})
		return
	}

	f.Status = fund.FundStatusActive
	f.UpdatedAt = time.Now().UTC()
	if err := h.funds.Update(c.Request.Context(), f); err != nil {
		respondError(c, err)
		return
	}

	// Generate payment schedule for all active members
	members, err := h.members.ListByFund(c.Request.Context(), fundID)
	if err == nil {
		_ = h.generateSchedule.Execute(c.Request.Context(), f, members)
	}

	c.JSON(http.StatusOK, gin.H{"data": f})
}

func (h *FundHandler) ListMembers(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	members, err := h.members.ListByFund(c.Request.Context(), fundID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": members})
}

type addMemberRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

func (h *FundHandler) AddMember(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	requesterID, _ := uuid.Parse(userIDStr)

	// Only admins can add members
	requester, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, requesterID)
	if err != nil || requester == nil || !requester.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundAdmin})
		return
	}

	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	targetID, _ := uuid.Parse(req.UserID)

	// Check not already a member
	existing, _ := h.members.GetByFundAndUser(c.Request.Context(), fundID, targetID)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": apperror.ErrAlreadyMember})
		return
	}

	member := &fund.FundMember{
		ID:       idgen.New(),
		FundID:   fundID,
		UserID:   targetID,
		Role:     fund.RoleMember,
		Status:   fund.MemberStatusActive,
		JoinedAt: time.Now().UTC(),
	}

	if err := h.members.Add(c.Request.Context(), member); err != nil {
		respondError(c, err)
		return
	}

	go func() {
		ctx := context.Background()
		f, err := h.funds.GetByID(ctx, fundID)
		if err != nil || f == nil {
			return
		}
		// Notify new member
		h.notifSvc.NotifyOne(ctx, targetID, &fundID,
			notification.TypeMemberJoined,
			"Bienvenido al fondo",
			fmt.Sprintf("Fuiste añadido al fondo '%s'.", f.Name),
		)
		// Notify existing members
		existing, err := h.members.ListByFund(ctx, fundID)
		if err != nil {
			return
		}
		h.notifSvc.NotifyAll(ctx, existing, &targetID, &fundID,
			notification.TypeMemberJoined,
			"Nuevo miembro",
			fmt.Sprintf("Un nuevo miembro se unió al fondo '%s'.", f.Name),
		)
	}()

	c.JSON(http.StatusCreated, gin.H{"data": member})
}
