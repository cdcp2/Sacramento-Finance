package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/shopspring/decimal"
)

// ─── CirculoConfigRepo ────────────────────────────────────────────────────────

type CirculoConfigRepo struct {
	db *pgxpool.Pool
}

func NewCirculoConfigRepo(db *pgxpool.Pool) *CirculoConfigRepo {
	return &CirculoConfigRepo{db: db}
}

func (r *CirculoConfigRepo) Create(ctx context.Context, c *fund.CirculoConfig) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO circulo_configs (fund_id, payout_order_type, current_round, rounds_completed)
		VALUES ($1, $2, $3, $4)
	`, c.FundID, c.PayoutOrderType, c.CurrentRound, c.RoundsCompleted)
	return err
}

func (r *CirculoConfigRepo) GetByFundID(ctx context.Context, fundID uuid.UUID) (*fund.CirculoConfig, error) {
	c := &fund.CirculoConfig{}
	err := r.db.QueryRow(ctx, `
		SELECT fund_id, payout_order_type, current_round, rounds_completed
		FROM circulo_configs WHERE fund_id = $1
	`, fundID).Scan(&c.FundID, &c.PayoutOrderType, &c.CurrentRound, &c.RoundsCompleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *CirculoConfigRepo) Update(ctx context.Context, c *fund.CirculoConfig) error {
	_, err := r.db.Exec(ctx, `
		UPDATE circulo_configs SET current_round=$1, rounds_completed=$2 WHERE fund_id=$3
	`, c.CurrentRound, c.RoundsCompleted, c.FundID)
	return err
}

// ─── VacaConfigRepo ───────────────────────────────────────────────────────────

type VacaConfigRepo struct {
	db *pgxpool.Pool
}

func NewVacaConfigRepo(db *pgxpool.Pool) *VacaConfigRepo {
	return &VacaConfigRepo{db: db}
}

func (r *VacaConfigRepo) Create(ctx context.Context, v *fund.VacaConfig) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO vaca_configs (fund_id, goal_amount, goal_description, distribution_type)
		VALUES ($1, $2, $3, $4)
	`, v.FundID, v.GoalAmount, v.GoalDescription, v.DistributionType)
	return err
}

func (r *VacaConfigRepo) GetByFundID(ctx context.Context, fundID uuid.UUID) (*fund.VacaConfig, error) {
	v := &fund.VacaConfig{}
	var goalAmount decimal.Decimal
	err := r.db.QueryRow(ctx, `
		SELECT fund_id, goal_amount, goal_description, distribution_type
		FROM vaca_configs WHERE fund_id = $1
	`, fundID).Scan(&v.FundID, &goalAmount, &v.GoalDescription, &v.DistributionType)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v.GoalAmount = goalAmount
	return v, nil
}

func (r *VacaConfigRepo) Update(ctx context.Context, v *fund.VacaConfig) error {
	_, err := r.db.Exec(ctx, `
		UPDATE vaca_configs SET goal_amount=$1, goal_description=$2, distribution_type=$3 WHERE fund_id=$4
	`, v.GoalAmount, v.GoalDescription, v.DistributionType, v.FundID)
	return err
}

// ─── FondoConfigRepo ──────────────────────────────────────────────────────────

type FondoConfigRepo struct {
	db *pgxpool.Pool
}

func NewFondoConfigRepo(db *pgxpool.Pool) *FondoConfigRepo {
	return &FondoConfigRepo{db: db}
}

func (r *FondoConfigRepo) Create(ctx context.Context, f *fund.FondoConfig) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO fondo_configs (fund_id, interest_rate, early_withdrawal_penalty)
		VALUES ($1, $2, $3)
	`, f.FundID, f.InterestRate, f.EarlyWithdrawalPenalty)
	return err
}

func (r *FondoConfigRepo) GetByFundID(ctx context.Context, fundID uuid.UUID) (*fund.FondoConfig, error) {
	f := &fund.FondoConfig{}
	var rate, penalty decimal.Decimal
	err := r.db.QueryRow(ctx, `
		SELECT fund_id, interest_rate, early_withdrawal_penalty
		FROM fondo_configs WHERE fund_id = $1
	`, fundID).Scan(&f.FundID, &rate, &penalty)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.InterestRate = rate
	f.EarlyWithdrawalPenalty = penalty
	return f, nil
}

func (r *FondoConfigRepo) Update(ctx context.Context, f *fund.FondoConfig) error {
	_, err := r.db.Exec(ctx, `
		UPDATE fondo_configs SET interest_rate=$1, early_withdrawal_penalty=$2 WHERE fund_id=$3
	`, f.InterestRate, f.EarlyWithdrawalPenalty, f.FundID)
	return err
}

// ─── PayoutRepo ───────────────────────────────────────────────────────────────

type PayoutRepo struct {
	db *pgxpool.Pool
}

func NewPayoutRepo(db *pgxpool.Pool) *PayoutRepo {
	return &PayoutRepo{db: db}
}

func (r *PayoutRepo) Create(ctx context.Context, p *ledger.Payout) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO payouts (id, fund_id, recipient_id, round_number, amount, status, scheduled_date, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, p.ID, p.FundID, p.RecipientID, p.RoundNumber, p.Amount, p.Status, p.ScheduledDate, p.CompletedAt)
	return err
}

func (r *PayoutRepo) GetByID(ctx context.Context, id uuid.UUID) (*ledger.Payout, error) {
	return r.scanOne(ctx, `
		SELECT id, fund_id, recipient_id, round_number, amount, status, scheduled_date, completed_at
		FROM payouts WHERE id = $1
	`, id)
}

func (r *PayoutRepo) GetByFundAndRound(ctx context.Context, fundID uuid.UUID, round int) (*ledger.Payout, error) {
	return r.scanOne(ctx, `
		SELECT id, fund_id, recipient_id, round_number, amount, status, scheduled_date, completed_at
		FROM payouts WHERE fund_id = $1 AND round_number = $2
	`, fundID, round)
}

func (r *PayoutRepo) ListByFund(ctx context.Context, fundID uuid.UUID) ([]*ledger.Payout, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, fund_id, recipient_id, round_number, amount, status, scheduled_date, completed_at
		FROM payouts WHERE fund_id = $1 ORDER BY round_number ASC
	`, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payouts []*ledger.Payout
	for rows.Next() {
		p, err := scanPayout(rows)
		if err != nil {
			return nil, err
		}
		payouts = append(payouts, p)
	}
	return payouts, rows.Err()
}

func (r *PayoutRepo) Update(ctx context.Context, p *ledger.Payout) error {
	_, err := r.db.Exec(ctx, `
		UPDATE payouts SET status=$1, completed_at=$2 WHERE id=$3
	`, p.Status, p.CompletedAt, p.ID)
	return err
}

func (r *PayoutRepo) scanOne(ctx context.Context, query string, args ...any) (*ledger.Payout, error) {
	row := r.db.QueryRow(ctx, query, args...)
	p, err := scanPayout(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

type payoutScanner interface {
	Scan(dest ...any) error
}

func scanPayout(row payoutScanner) (*ledger.Payout, error) {
	p := &ledger.Payout{}
	var amount decimal.Decimal
	var scheduledDate time.Time
	err := row.Scan(
		&p.ID, &p.FundID, &p.RecipientID, &p.RoundNumber,
		&amount, &p.Status, &scheduledDate, &p.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Amount = amount
	p.ScheduledDate = scheduledDate
	return p, nil
}
