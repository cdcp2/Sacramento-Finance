package fund

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FundType represents the 3 savings product types.
type FundType string

const (
	FundTypeCirculo     FundType = "circulo"
	FundTypeVaca        FundType = "vaca"
	FundTypeFondoAhorro FundType = "fondo_ahorro"
)

// FundStatus represents the state machine for a fund.
type FundStatus string

const (
	FundStatusDraft     FundStatus = "draft"
	FundStatusActive    FundStatus = "active"
	FundStatusPaused    FundStatus = "paused"
	FundStatusCompleted FundStatus = "completed"
	FundStatusCancelled FundStatus = "cancelled"
)

// PeriodFrequency is how often contributions are expected.
type PeriodFrequency string

const (
	FrequencyWeekly   PeriodFrequency = "weekly"
	FrequencyBiweekly PeriodFrequency = "biweekly"
	FrequencyMonthly  PeriodFrequency = "monthly"
)

// MemberRole within a fund.
type MemberRole string

const (
	RoleAdmin  MemberRole = "admin"
	RoleMember MemberRole = "member"
)

// MemberStatus within a fund.
type MemberStatus string

const (
	MemberStatusActive    MemberStatus = "active"
	MemberStatusSuspended MemberStatus = "suspended"
	MemberStatusLeft      MemberStatus = "left"
)

// PenaltyType defines how a penalty is calculated.
type PenaltyType string

const (
	PenaltyTypeFixed      PenaltyType = "fixed"
	PenaltyTypePercentage PenaltyType = "percentage"
)

// GovernanceType defines how decisions are made in a fund.
type GovernanceType string

const (
	GovernanceAdminOnly GovernanceType = "admin_only"
	GovernanceMajority  GovernanceType = "majority"
	GovernanceUnanimous GovernanceType = "unanimous"
)

// Fund is the core aggregate for any savings product.
type Fund struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        FundType   `json:"type"`
	Status      FundStatus `json:"status"`
	CreatorID   uuid.UUID  `json:"creator_id"`
	Currency    string     `json:"currency"`
	Rules       Rules      `json:"rules"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Rules contains the configurable parameters of a fund.
type Rules struct {
	ContributionAmount  decimal.Decimal `json:"contribution_amount"`
	Frequency           PeriodFrequency `json:"frequency"`
	TotalPeriods        int             `json:"total_periods"`
	StartDate           time.Time       `json:"start_date"`
	PenaltyEnabled      bool            `json:"penalty_enabled"`
	PenaltyType         PenaltyType     `json:"penalty_type"`
	PenaltyAmount       decimal.Decimal `json:"penalty_amount"` // fixed amount or percentage value
	GracePeriodDays     int             `json:"grace_period_days"`
	MinMembers          int             `json:"min_members"`
	MaxMembers          int             `json:"max_members"`
	GovernanceType      GovernanceType  `json:"governance_type"`
	VotingDeadlineHours int             `json:"voting_deadline_hours"` // how many hours a proposal stays open
}

// IsGoverned returns true when actions require a vote instead of direct admin execution.
func (r *Rules) IsGoverned() bool {
	return r.GovernanceType == GovernanceMajority || r.GovernanceType == GovernanceUnanimous
}

// CanTransitionTo validates the fund state machine.
func (f *Fund) CanTransitionTo(next FundStatus) bool {
	allowed := map[FundStatus][]FundStatus{
		FundStatusDraft:     {FundStatusActive, FundStatusCancelled},
		FundStatusActive:    {FundStatusPaused, FundStatusCompleted, FundStatusCancelled},
		FundStatusPaused:    {FundStatusActive, FundStatusCancelled},
		FundStatusCompleted: {},
		FundStatusCancelled: {},
	}
	for _, s := range allowed[f.Status] {
		if s == next {
			return true
		}
	}
	return false
}

// FundMember represents a user's membership in a fund.
type FundMember struct {
	ID          uuid.UUID    `json:"id"`
	FundID      uuid.UUID    `json:"fund_id"`
	UserID      uuid.UUID    `json:"user_id"`
	Role        MemberRole   `json:"role"`
	Status      MemberStatus `json:"status"`
	PayoutOrder *int         `json:"payout_order,omitempty"` // Only used in Circulo
	JoinedAt    time.Time    `json:"joined_at"`
}

// IsAdmin returns true if the member has admin role.
func (m *FundMember) IsAdmin() bool {
	return m.Role == RoleAdmin
}

type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"
	InvitationStatusAccepted  InvitationStatus = "accepted"
	InvitationStatusRejected  InvitationStatus = "rejected"
	InvitationStatusCancelled InvitationStatus = "cancelled"
	InvitationStatusExpired   InvitationStatus = "expired"
)

type FundInvitation struct {
	ID          uuid.UUID        `json:"id"`
	FundID      uuid.UUID        `json:"fund_id"`
	InviterID   uuid.UUID        `json:"inviter_id"`
	InviteeID   uuid.UUID        `json:"invitee_id"`
	Status      InvitationStatus `json:"status"`
	Message     string           `json:"message"`
	ExpiresAt   time.Time        `json:"expires_at"`
	RespondedAt *time.Time       `json:"responded_at,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// CirculoConfig holds Circulo-specific settings.
type CirculoConfig struct {
	FundID          uuid.UUID `json:"fund_id"`
	PayoutOrderType string    `json:"payout_order_type"` // "fixed" | "randomized"
	CurrentRound    int       `json:"current_round"`
	RoundsCompleted int       `json:"rounds_completed"`
}

// VacaConfig holds Vaca-specific settings.
type VacaConfig struct {
	FundID           uuid.UUID       `json:"fund_id"`
	GoalAmount       decimal.Decimal `json:"goal_amount"`
	GoalDescription  string          `json:"goal_description"`
	DistributionType string          `json:"distribution_type"` // "goal_reached" | "unanimous_vote"
}

// FondoConfig holds Fondo de Ahorro-specific settings.
type FondoConfig struct {
	FundID                 uuid.UUID       `json:"fund_id"`
	InterestRate           decimal.Decimal `json:"interest_rate"`
	EarlyWithdrawalPenalty decimal.Decimal `json:"early_withdrawal_penalty"`
}
