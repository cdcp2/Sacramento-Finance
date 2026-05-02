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
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPartial  PaymentStatus = "partial"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusMissed   PaymentStatus = "missed"
	PaymentStatusWaived   PaymentStatus = "waived"
)

// LedgerEntry is an immutable record of a financial movement in a fund.
// Never update or delete ledger entries — use compensating entries instead.
type LedgerEntry struct {
	ID           uuid.UUID
	FundID       uuid.UUID
	UserID       *uuid.UUID // nil for fund-level system entries
	Type         EntryType
	Direction    Direction
	Amount       decimal.Decimal // always positive
	BalanceAfter decimal.Decimal // fund balance snapshot after this entry
	ReferenceID  *uuid.UUID      // links to Payment or Payout
	Description  string
	CreatedAt    time.Time // immutable — no UpdatedAt
}

// Payment represents one scheduled contribution in a fund.
type Payment struct {
	ID             uuid.UUID
	FundID         uuid.UUID
	MemberID       uuid.UUID
	PeriodNumber   int
	DueDate        time.Time
	AmountDue      decimal.Decimal
	AmountPaid     decimal.Decimal
	Status         PaymentStatus
	PaidAt         *time.Time
	PenaltyApplied bool
	PenaltyAmount  decimal.Decimal
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsOverdue returns true if the payment is past its due date and not paid.
func (p *Payment) IsOverdue(now time.Time) bool {
	return now.After(p.DueDate) && p.Status == PaymentStatusPending
}

// Payout represents a distribution of funds to a member (used in Circulo).
type Payout struct {
	ID              uuid.UUID
	FundID          uuid.UUID
	RecipientID     uuid.UUID // FundMember ID
	RoundNumber     int
	Amount          decimal.Decimal
	Status          string // "scheduled" | "completed" | "failed"
	ScheduledDate   time.Time
	CompletedAt     *time.Time
}
