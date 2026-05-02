package ledger

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type EntryType string

const (
	EntryTypeDeposit    EntryType = "deposit"
	EntryTypeWithdrawal EntryType = "withdrawal"
	EntryTypePenalty    EntryType = "penalty"
	EntryTypePayout     EntryType = "payout"
	EntryTypeInterest   EntryType = "interest"
	EntryTypeRefund     EntryType = "refund"
)

type Direction string

const (
	DirectionCredit Direction = "credit"
	DirectionDebit  Direction = "debit"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPartial PaymentStatus = "partial"
	PaymentStatusPaid    PaymentStatus = "paid"
	PaymentStatusMissed  PaymentStatus = "missed"
	PaymentStatusWaived  PaymentStatus = "waived"
)

// LedgerEntry is an immutable record of a financial movement in a fund.
// Never update or delete ledger entries — use compensating entries instead.
type LedgerEntry struct {
	ID           uuid.UUID       `json:"id"`
	FundID       uuid.UUID       `json:"fund_id"`
	UserID       *uuid.UUID      `json:"user_id,omitempty"` // nil for fund-level system entries
	Type         EntryType       `json:"type"`
	Direction    Direction       `json:"direction"`
	Amount       decimal.Decimal `json:"amount"`                 // always positive
	BalanceAfter decimal.Decimal `json:"balance_after"`          // fund balance snapshot after this entry
	ReferenceID  *uuid.UUID      `json:"reference_id,omitempty"` // links to Payment or Payout
	Description  string          `json:"description"`
	CreatedAt    time.Time       `json:"created_at"` // immutable — no UpdatedAt
}

// Payment represents one scheduled contribution in a fund.
type Payment struct {
	ID             uuid.UUID       `json:"id"`
	FundID         uuid.UUID       `json:"fund_id"`
	MemberID       uuid.UUID       `json:"member_id"`
	PeriodNumber   int             `json:"period_number"`
	DueDate        time.Time       `json:"due_date"`
	AmountDue      decimal.Decimal `json:"amount_due"`
	AmountPaid     decimal.Decimal `json:"amount_paid"`
	Status         PaymentStatus   `json:"status"`
	PaidAt         *time.Time      `json:"paid_at,omitempty"`
	PenaltyApplied bool            `json:"penalty_applied"`
	PenaltyAmount  decimal.Decimal `json:"penalty_amount"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// IsOverdue returns true if the payment is past its due date and not paid.
func (p *Payment) IsOverdue(now time.Time) bool {
	return now.After(p.DueDate) && p.Status == PaymentStatusPending
}

// Payout represents a distribution of funds to a member (used in Circulo).
type Payout struct {
	ID            uuid.UUID       `json:"id"`
	FundID        uuid.UUID       `json:"fund_id"`
	RecipientID   uuid.UUID       `json:"recipient_id"` // FundMember ID
	RoundNumber   int             `json:"round_number"`
	Amount        decimal.Decimal `json:"amount"`
	Status        string          `json:"status"` // "scheduled" | "completed" | "failed"
	ScheduledDate time.Time       `json:"scheduled_date"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
}
