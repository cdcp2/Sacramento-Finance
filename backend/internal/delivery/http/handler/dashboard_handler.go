package handler

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/governance"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/shopspring/decimal"
)

type DashboardHandler struct {
	funds         fund.Repository
	members       fund.MemberRepository
	payments      ledger.PaymentRepository
	proposals     governance.ProposalRepository
	notifications notification.Repository
}

func NewDashboardHandler(
	funds fund.Repository,
	members fund.MemberRepository,
	payments ledger.PaymentRepository,
	proposals governance.ProposalRepository,
	notifications notification.Repository,
) *DashboardHandler {
	return &DashboardHandler{
		funds:         funds,
		members:       members,
		payments:      payments,
		proposals:     proposals,
		notifications: notifications,
	}
}

type dashboardSummary struct {
	TotalFunds          int             `json:"total_funds"`
	ActiveFunds         int             `json:"active_funds"`
	DraftFunds          int             `json:"draft_funds"`
	CompletedFunds      int             `json:"completed_funds"`
	AdminFunds          int             `json:"admin_funds"`
	PendingPayments     int             `json:"pending_payments"`
	OverduePayments     int             `json:"overdue_payments"`
	OpenProposals       int             `json:"open_proposals"`
	UnreadNotifications int             `json:"unread_notifications"`
	TotalPendingAmount  decimal.Decimal `json:"total_pending_amount"`
}

type dashboardFund struct {
	ID              uuid.UUID           `json:"id"`
	Name            string              `json:"name"`
	Type            fund.FundType       `json:"type"`
	Status          fund.FundStatus     `json:"status"`
	Role            fund.MemberRole     `json:"role"`
	GovernanceType  fund.GovernanceType `json:"governance_type"`
	PendingPayments int                 `json:"pending_payments"`
	OverduePayments int                 `json:"overdue_payments"`
	OpenProposals   int                 `json:"open_proposals"`
	NextPayment     *dashboardPayment   `json:"next_payment,omitempty"`
}

type dashboardPayment struct {
	ID           uuid.UUID            `json:"id"`
	FundID       uuid.UUID            `json:"fund_id"`
	FundName     string               `json:"fund_name"`
	PeriodNumber int                  `json:"period_number"`
	DueDate      time.Time            `json:"due_date"`
	AmountDue    decimal.Decimal      `json:"amount_due"`
	AmountPaid   decimal.Decimal      `json:"amount_paid"`
	Status       ledger.PaymentStatus `json:"status"`
	IsOverdue    bool                 `json:"is_overdue"`
}

type dashboardProposal struct {
	ID           uuid.UUID                 `json:"id"`
	FundID       uuid.UUID                 `json:"fund_id"`
	FundName     string                    `json:"fund_name"`
	Type         governance.ProposalType   `json:"type"`
	Status       governance.ProposalStatus `json:"status"`
	VotesFor     int                       `json:"votes_for"`
	VotesAgainst int                       `json:"votes_against"`
	QuorumNeeded int                       `json:"quorum_needed"`
	DeadlineAt   time.Time                 `json:"deadline_at"`
}

type dashboardResponse struct {
	Summary          dashboardSummary    `json:"summary"`
	Funds            []dashboardFund     `json:"funds"`
	UpcomingPayments []dashboardPayment  `json:"upcoming_payments"`
	OpenProposals    []dashboardProposal `json:"open_proposals"`
}

func (h *DashboardHandler) Get(c *gin.Context) {
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

	result, err := h.build(c.Request.Context(), userID, time.Now().UTC())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *DashboardHandler) build(ctx context.Context, userID uuid.UUID, now time.Time) (*dashboardResponse, error) {
	funds, err := h.funds.ListByMember(ctx, userID)
	if err != nil {
		return nil, err
	}

	unread, err := h.notifications.CountUnread(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := &dashboardResponse{
		Summary: dashboardSummary{
			TotalFunds:          len(funds),
			UnreadNotifications: unread,
		},
		Funds:            make([]dashboardFund, 0, len(funds)),
		UpcomingPayments: make([]dashboardPayment, 0),
		OpenProposals:    make([]dashboardProposal, 0),
	}

	for _, f := range funds {
		member, err := h.members.GetByFundAndUser(ctx, f.ID, userID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			continue
		}

		fundItem := dashboardFund{
			ID:             f.ID,
			Name:           f.Name,
			Type:           f.Type,
			Status:         f.Status,
			Role:           member.Role,
			GovernanceType: f.Rules.GovernanceType,
		}
		switch f.Status {
		case fund.FundStatusActive:
			resp.Summary.ActiveFunds++
		case fund.FundStatusDraft:
			resp.Summary.DraftFunds++
		case fund.FundStatusCompleted:
			resp.Summary.CompletedFunds++
		}
		if member.IsAdmin() {
			resp.Summary.AdminFunds++
		}

		payments, err := h.payments.ListByMember(ctx, member.ID)
		if err != nil {
			return nil, err
		}
		for _, p := range payments {
			if !isDashboardPendingPayment(p, now) {
				continue
			}
			paymentItem := dashboardPayment{
				ID:           p.ID,
				FundID:       f.ID,
				FundName:     f.Name,
				PeriodNumber: p.PeriodNumber,
				DueDate:      p.DueDate,
				AmountDue:    p.AmountDue,
				AmountPaid:   p.AmountPaid,
				Status:       p.Status,
				IsOverdue:    isDashboardOverduePayment(p, now),
			}
			resp.Summary.PendingPayments++
			fundItem.PendingPayments++
			resp.Summary.TotalPendingAmount = resp.Summary.TotalPendingAmount.Add(p.AmountDue.Sub(p.AmountPaid))
			if paymentItem.IsOverdue {
				resp.Summary.OverduePayments++
				fundItem.OverduePayments++
			}
			resp.UpcomingPayments = append(resp.UpcomingPayments, paymentItem)
			if fundItem.NextPayment == nil || paymentItem.DueDate.Before(fundItem.NextPayment.DueDate) {
				next := paymentItem
				fundItem.NextPayment = &next
			}
		}

		proposals, err := h.proposals.ListByFund(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		for _, p := range proposals {
			if p.Status != governance.ProposalStatusOpen {
				continue
			}
			proposalItem := dashboardProposal{
				ID:           p.ID,
				FundID:       f.ID,
				FundName:     f.Name,
				Type:         p.Type,
				Status:       p.Status,
				VotesFor:     p.VotesFor,
				VotesAgainst: p.VotesAgainst,
				QuorumNeeded: p.QuorumNeeded,
				DeadlineAt:   p.DeadlineAt,
			}
			resp.Summary.OpenProposals++
			fundItem.OpenProposals++
			resp.OpenProposals = append(resp.OpenProposals, proposalItem)
		}

		resp.Funds = append(resp.Funds, fundItem)
	}

	sort.Slice(resp.UpcomingPayments, func(i, j int) bool {
		return resp.UpcomingPayments[i].DueDate.Before(resp.UpcomingPayments[j].DueDate)
	})
	if len(resp.UpcomingPayments) > 5 {
		resp.UpcomingPayments = resp.UpcomingPayments[:5]
	}

	sort.Slice(resp.OpenProposals, func(i, j int) bool {
		return resp.OpenProposals[i].DeadlineAt.Before(resp.OpenProposals[j].DeadlineAt)
	})
	if len(resp.OpenProposals) > 5 {
		resp.OpenProposals = resp.OpenProposals[:5]
	}

	return resp, nil
}

func isDashboardPendingPayment(p *ledger.Payment, now time.Time) bool {
	return p.Status == ledger.PaymentStatusPending ||
		p.Status == ledger.PaymentStatusPartial ||
		p.IsOverdue(now)
}

func isDashboardOverduePayment(p *ledger.Payment, now time.Time) bool {
	return p.Status == ledger.PaymentStatusMissed || p.IsOverdue(now)
}
