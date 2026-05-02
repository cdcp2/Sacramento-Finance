package notification

import (
	"time"

	"github.com/google/uuid"
)

// Type classifies what event triggered the notification.
type Type string

const (
	TypePaymentConfirmed Type = "payment_confirmed"
	TypePaymentWaived    Type = "payment_waived"
	TypePaymentOverdue   Type = "payment_overdue"
	TypePayoutReceived   Type = "payout_received"
	TypeRoundClosed      Type = "round_closed"
	TypeWithdrawal       Type = "withdrawal"
	TypeInterestAccrued  Type = "interest_accrued"
	TypeGoalReached      Type = "goal_reached"
	TypeMemberJoined     Type = "member_joined"
	TypeMemberLeft       Type = "member_left"
	TypeProposalCreated  Type = "proposal_created"
	TypeProposalResolved Type = "proposal_resolved"
)

// Notification is an in-app message delivered to a specific user.
type Notification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`           // recipient (users.id)
	FundID    *uuid.UUID `json:"fund_id,omitempty"` // nil for account-level notifications
	Type      Type       `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
}
