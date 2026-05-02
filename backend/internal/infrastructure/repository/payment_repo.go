package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/shopspring/decimal"
)

type PaymentRepo struct {
	db *pgxpool.Pool
}

func NewPaymentRepo(db *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) Create(ctx context.Context, p *ledger.Payment) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO payments (
			id, fund_id, member_id, period_number, due_date,
			amount_due, amount_paid, status, paid_at,
			penalty_applied, penalty_amount, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`,
		p.ID, p.FundID, p.MemberID, p.PeriodNumber, p.DueDate,
		p.AmountDue, p.AmountPaid, p.Status, p.PaidAt,
		p.PenaltyApplied, p.PenaltyAmount, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *PaymentRepo) CreateBatch(ctx context.Context, payments []*ledger.Payment) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, p := range payments {
		_, err := tx.Exec(ctx, `
			INSERT INTO payments (
				id, fund_id, member_id, period_number, due_date,
				amount_due, amount_paid, status, paid_at,
				penalty_applied, penalty_amount, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		`,
			p.ID, p.FundID, p.MemberID, p.PeriodNumber, p.DueDate,
			p.AmountDue, p.AmountPaid, p.Status, p.PaidAt,
			p.PenaltyApplied, p.PenaltyAmount, p.CreatedAt, p.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PaymentRepo) GetByID(ctx context.Context, id uuid.UUID) (*ledger.Payment, error) {
	return r.scanOne(ctx, `
		SELECT id, fund_id, member_id, period_number, due_date,
		       amount_due, amount_paid, status, paid_at,
		       penalty_applied, penalty_amount, created_at, updated_at
		FROM payments WHERE id = $1
	`, id)
}

func (r *PaymentRepo) ListByMember(ctx context.Context, memberID uuid.UUID) ([]*ledger.Payment, error) {
	return r.scanMany(ctx, `
		SELECT id, fund_id, member_id, period_number, due_date,
		       amount_due, amount_paid, status, paid_at,
		       penalty_applied, penalty_amount, created_at, updated_at
		FROM payments WHERE member_id = $1 ORDER BY period_number ASC
	`, memberID)
}

func (r *PaymentRepo) ListByFund(ctx context.Context, fundID uuid.UUID) ([]*ledger.Payment, error) {
	return r.scanMany(ctx, `
		SELECT id, fund_id, member_id, period_number, due_date,
		       amount_due, amount_paid, status, paid_at,
		       penalty_applied, penalty_amount, created_at, updated_at
		FROM payments WHERE fund_id = $1 ORDER BY period_number ASC, member_id ASC
	`, fundID)
}

func (r *PaymentRepo) ListOverdue(ctx context.Context, before time.Time) ([]*ledger.Payment, error) {
	return r.scanMany(ctx, `
		SELECT id, fund_id, member_id, period_number, due_date,
		       amount_due, amount_paid, status, paid_at,
		       penalty_applied, penalty_amount, created_at, updated_at
		FROM payments WHERE due_date < $1 AND status = 'pending'
	`, before)
}

func (r *PaymentRepo) Update(ctx context.Context, p *ledger.Payment) error {
	p.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE payments SET
			amount_paid=$1, status=$2, paid_at=$3,
			penalty_applied=$4, penalty_amount=$5, updated_at=$6
		WHERE id=$7
	`, p.AmountPaid, p.Status, p.PaidAt,
		p.PenaltyApplied, p.PenaltyAmount, p.UpdatedAt,
		p.ID)
	return err
}

func (r *PaymentRepo) scanOne(ctx context.Context, query string, args ...any) (*ledger.Payment, error) {
	row := r.db.QueryRow(ctx, query, args...)
	p := &ledger.Payment{}
	var amountDue, amountPaid, penaltyAmount decimal.Decimal
	err := row.Scan(
		&p.ID, &p.FundID, &p.MemberID, &p.PeriodNumber, &p.DueDate,
		&amountDue, &amountPaid, &p.Status, &p.PaidAt,
		&p.PenaltyApplied, &penaltyAmount, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	p.AmountDue = amountDue
	p.AmountPaid = amountPaid
	p.PenaltyAmount = penaltyAmount
	return p, nil
}

func (r *PaymentRepo) scanMany(ctx context.Context, query string, args ...any) ([]*ledger.Payment, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*ledger.Payment
	for rows.Next() {
		p := &ledger.Payment{}
		var amountDue, amountPaid, penaltyAmount decimal.Decimal
		if err := rows.Scan(
			&p.ID, &p.FundID, &p.MemberID, &p.PeriodNumber, &p.DueDate,
			&amountDue, &amountPaid, &p.Status, &p.PaidAt,
			&p.PenaltyApplied, &penaltyAmount, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.AmountDue = amountDue
		p.AmountPaid = amountPaid
		p.PenaltyAmount = penaltyAmount
		payments = append(payments, p)
	}
	return payments, rows.Err()
}
