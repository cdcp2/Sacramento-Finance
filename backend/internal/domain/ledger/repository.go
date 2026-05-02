package ledger

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Repository interface {
	// CreateEntry writes a ledger entry and returns the updated fund balance.
	// Must be called inside a database transaction.
	CreateEntry(ctx context.Context, entry *LedgerEntry) error

	// GetFundBalance returns the current simulated balance for a fund.
	GetFundBalance(ctx context.Context, fundID uuid.UUID) (decimal.Decimal, error)

	// ListByFund returns paginated ledger entries for a fund.
	ListByFund(ctx context.Context, fundID uuid.UUID, limit, offset int) ([]*LedgerEntry, error)

	// ListByMember returns ledger entries for a specific user in a fund.
	ListByMember(ctx context.Context, fundID, userID uuid.UUID, limit, offset int) ([]*LedgerEntry, error)
}

type PaymentRepository interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	ListByMember(ctx context.Context, memberID uuid.UUID) ([]*Payment, error)
	ListByFund(ctx context.Context, fundID uuid.UUID) ([]*Payment, error)
	ListOverdue(ctx context.Context, before time.Time) ([]*Payment, error)
	Update(ctx context.Context, p *Payment) error
}

type PayoutRepository interface {
	Create(ctx context.Context, p *Payout) error
	GetByID(ctx context.Context, id uuid.UUID) (*Payout, error)
	GetByFundAndRound(ctx context.Context, fundID uuid.UUID, round int) (*Payout, error)
	ListByFund(ctx context.Context, fundID uuid.UUID) ([]*Payout, error)
	Update(ctx context.Context, p *Payout) error
}
