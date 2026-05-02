package fund

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, f *Fund) error
	GetByID(ctx context.Context, id uuid.UUID) (*Fund, error)
	Update(ctx context.Context, f *Fund) error
	ListByMember(ctx context.Context, userID uuid.UUID) ([]*Fund, error)
}

type MemberRepository interface {
	Add(ctx context.Context, m *FundMember) error
	GetByFundAndUser(ctx context.Context, fundID, userID uuid.UUID) (*FundMember, error)
	GetByID(ctx context.Context, id uuid.UUID) (*FundMember, error)
	ListByFund(ctx context.Context, fundID uuid.UUID) ([]*FundMember, error)
	Update(ctx context.Context, m *FundMember) error
	CountActive(ctx context.Context, fundID uuid.UUID) (int, error)
}

type CirculoRepository interface {
	Create(ctx context.Context, c *CirculoConfig) error
	GetByFundID(ctx context.Context, fundID uuid.UUID) (*CirculoConfig, error)
	Update(ctx context.Context, c *CirculoConfig) error
}

type VacaRepository interface {
	Create(ctx context.Context, v *VacaConfig) error
	GetByFundID(ctx context.Context, fundID uuid.UUID) (*VacaConfig, error)
	Update(ctx context.Context, v *VacaConfig) error
}

type FondoRepository interface {
	Create(ctx context.Context, f *FondoConfig) error
	GetByFundID(ctx context.Context, fundID uuid.UUID) (*FondoConfig, error)
	Update(ctx context.Context, f *FondoConfig) error
}
