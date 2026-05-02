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
	FrequencyWeekly    PeriodFrequency = "weekly"
	FrequencyBiweekly  PeriodFrequency = "biweekly"
	FrequencyMonthly   PeriodFrequency = "monthly"
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
	ID          uuid.UUID
	Name        string
	Description string
	Type        FundType
	Status      FundStatus
	CreatorID   uuid.UUID
	Currency    string
	Rules       Rules
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Rules contains the configurable parameters of a fund.
type Rules struct {
	ContributionAmount  decimal.Decimal
	Frequency           PeriodFrequency
	TotalPeriods        int
	StartDate           time.Time
	PenaltyEnabled      bool
	PenaltyType         PenaltyType
	PenaltyAmount       decimal.Decimal // fixed amount or percentage value
	GracePeriodDays     int
	MinMembers          int
	MaxMembers          int
	GovernanceType      GovernanceType
	VotingDeadlineHours int // how many hours a proposal stays open
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
	ID          uuid.UUID
	FundID      uuid.UUID
	UserID      uuid.UUID
	Role        MemberRole
	Status      MemberStatus
	PayoutOrder *int // Only used in Circulo
	JoinedAt    time.Time
}

// IsAdmin returns true if the member has admin role.
func (m *FundMember) IsAdmin() bool {
	return m.Role == RoleAdmin
}

// CirculoConfig holds Circulo-specific settings.
type CirculoConfig struct {
	FundID           uuid.UUID
	PayoutOrderType  string // "fixed" | "randomized"
	CurrentRound     int
	RoundsCompleted  int
}

// VacaConfig holds Vaca-specific settings.
type VacaConfig struct {
	FundID           uuid.UUID
	GoalAmount       decimal.Decimal
	GoalDescription  string
	DistributionType string // "goal_reached" | "unanimous_vote"
}

// FondoConfig holds Fondo de Ahorro-specific settings.
type FondoConfig struct {
	FundID                 uuid.UUID
	InterestRate           decimal.Decimal
	EarlyWithdrawalPenalty decimal.Decimal
}
