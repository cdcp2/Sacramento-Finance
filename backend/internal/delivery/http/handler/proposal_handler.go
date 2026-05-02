package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/governance"
	"github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/pkg/apperror"
	ucgovernance "github.com/sacramento-finance/backend/internal/usecase/governance"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	"github.com/sacramento-finance/backend/internal/infrastructure/repository"
)

type ProposalHandler struct {
	createProposal *ucgovernance.CreateProposalUseCase
	castVote       *ucgovernance.CastVoteUseCase
	proposals      *repository.ProposalRepo
	votes          *repository.VoteRepo
	funds          fund.Repository
	members        fund.MemberRepository
	notifSvc       *ucnotif.Service
}

func NewProposalHandler(
	createProposal *ucgovernance.CreateProposalUseCase,
	castVote *ucgovernance.CastVoteUseCase,
	proposals *repository.ProposalRepo,
	votes *repository.VoteRepo,
	funds fund.Repository,
	members fund.MemberRepository,
	notifSvc *ucnotif.Service,
) *ProposalHandler {
	return &ProposalHandler{
		createProposal: createProposal,
		castVote:       castVote,
		proposals:      proposals,
		votes:          votes,
		funds:          funds,
		members:        members,
		notifSvc:       notifSvc,
	}
}

type createProposalRequest struct {
	Type    governance.ProposalType `json:"type" binding:"required"`
	Payload json.RawMessage         `json:"payload"` // validated per type below
}

// Create creates a new governance proposal.
// POST /funds/:fund_id/proposals
func (h *ProposalHandler) Create(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return
	}

	member, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, userID)
	if err != nil || member == nil || member.Status != fund.MemberStatusActive {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundMember})
		return
	}

	var req createProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	// Validate payload shape per proposal type
	if msg := validateProposalPayload(req.Type, req.Payload); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_PAYLOAD", "message": msg}})
		return
	}

	// Decode payload to pass as typed value
	var payload any
	if len(req.Payload) > 0 {
		var raw map[string]any
		_ = json.Unmarshal(req.Payload, &raw)
		payload = raw
	} else {
		payload = map[string]any{}
	}

	proposal, err := h.createProposal.Execute(c.Request.Context(), f, member.ID, req.Type, payload)
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
		h.notifSvc.NotifyAll(ctx, members, &userID, &fundID,
			notification.TypeProposalCreated,
			"Nueva propuesta",
			fmt.Sprintf("Se creó una propuesta de tipo '%s' en '%s'.", string(proposal.Type), f.Name),
		)
	}()

	c.JSON(http.StatusCreated, gin.H{"data": proposal})
}

// List returns all proposals for a fund (open first, then resolved).
// GET /funds/:fund_id/proposals
func (h *ProposalHandler) List(c *gin.Context) {
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

	proposals, err := h.proposals.ListByFund(c.Request.Context(), fundID)
	if err != nil {
		respondError(c, err)
		return
	}

	// Lazy expiry: mark open proposals past their deadline
	now := time.Now().UTC()
	for _, p := range proposals {
		if p.IsExpired(now) {
			p.Status = governance.ProposalStatusExpired
			_ = h.proposals.UpdateStatus(c.Request.Context(), p)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": proposals})
}

// Get returns a single proposal with its votes.
// GET /funds/:fund_id/proposals/:proposal_id
func (h *ProposalHandler) Get(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}
	proposalID, err := uuid.Parse(c.Param("proposal_id"))
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

	proposal, err := h.proposals.GetByID(c.Request.Context(), proposalID)
	if err != nil || proposal == nil || proposal.FundID != fundID {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrNotFound})
		return
	}

	// Lazy expiry
	if proposal.IsExpired(time.Now().UTC()) {
		proposal.Status = governance.ProposalStatusExpired
		_ = h.proposals.UpdateStatus(c.Request.Context(), proposal)
	}

	votes, _ := h.votes.ListByProposal(c.Request.Context(), proposalID)
	myVote, _ := h.votes.GetByProposalAndMember(c.Request.Context(), proposalID, member.ID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"proposal": proposal,
			"votes":    votes,
			"my_vote":  myVote,
		},
	})
}

type castVoteRequest struct {
	Choice governance.VoteChoice `json:"choice" binding:"required,oneof=yes no"`
}

// Vote casts a vote on a proposal.
// POST /funds/:fund_id/proposals/:proposal_id/vote
func (h *ProposalHandler) Vote(c *gin.Context) {
	fundID, err := uuid.Parse(c.Param("fund_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}
	proposalID, err := uuid.Parse(c.Param("proposal_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperror.ErrBadRequest})
		return
	}

	userIDStr, _ := middleware.GetUserID(c)
	userID, _ := uuid.Parse(userIDStr)

	f, err := h.funds.GetByID(c.Request.Context(), fundID)
	if err != nil || f == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": apperror.ErrFundNotFound})
		return
	}

	member, err := h.members.GetByFundAndUser(c.Request.Context(), fundID, userID)
	if err != nil || member == nil || member.Status != fund.MemberStatusActive {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrNotFundMember})
		return
	}

	var req castVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	result, err := h.castVote.Execute(c.Request.Context(), proposalID, member.ID, req.Choice, f)
	if err != nil {
		respondError(c, err)
		return
	}

	if result.Resolved {
		go func() {
			ctx := context.Background()
			members, err := h.members.ListByFund(ctx, fundID)
			if err != nil {
				return
			}
			h.notifSvc.NotifyAll(ctx, members, nil, &fundID,
				notification.TypeProposalResolved,
				"Propuesta resuelta",
				fmt.Sprintf("Una propuesta en '%s' fue resuelta.", f.Name),
			)
		}()
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// validateProposalPayload checks that the payload contains required fields per type.
func validateProposalPayload(pType governance.ProposalType, raw json.RawMessage) string {
	switch pType {
	case governance.ProposalTypeWaivePayment:
		var p governance.WaivePaymentPayload
		if err := json.Unmarshal(raw, &p); err != nil || p.PaymentID == "" {
			return "waive_payment requires payment_id"
		}
	case governance.ProposalTypeRemoveMember:
		var p governance.RemoveMemberPayload
		if err := json.Unmarshal(raw, &p); err != nil || p.MemberID == "" {
			return "remove_member requires member_id"
		}
	case governance.ProposalTypeChangeGovernance:
		var p governance.ChangeGovernancePayload
		if err := json.Unmarshal(raw, &p); err != nil || p.GovernanceType == "" {
			return "change_governance requires governance_type"
		}
		if p.GovernanceType != fund.GovernanceAdminOnly &&
			p.GovernanceType != fund.GovernanceMajority &&
			p.GovernanceType != fund.GovernanceUnanimous {
			return "governance_type must be admin_only, majority, or unanimous"
		}
	// activate_fund, cancel_fund, change_rules, distribute_vaca: payload is optional
	}
	return ""
}
