package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/shopspring/decimal"
)

type FundRepo struct {
	db *pgxpool.Pool
}

func NewFundRepo(db *pgxpool.Pool) *FundRepo {
	return &FundRepo{db: db}
}

func (r *FundRepo) Create(ctx context.Context, f *fund.Fund) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO funds (
			id, name, description, type, status, creator_id, currency,
			contribution_amount, frequency, total_periods, start_date,
			penalty_enabled, penalty_type, penalty_amount, grace_period_days,
			min_members, max_members, governance_type, voting_deadline_hours,
			created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21
		)`,
		f.ID, f.Name, f.Description, f.Type, f.Status, f.CreatorID, f.Currency,
		f.Rules.ContributionAmount, f.Rules.Frequency, f.Rules.TotalPeriods,
		f.Rules.StartDate,
		f.Rules.PenaltyEnabled, nullablePenaltyType(f.Rules.PenaltyType),
		f.Rules.PenaltyAmount, f.Rules.GracePeriodDays,
		f.Rules.MinMembers, f.Rules.MaxMembers,
		f.Rules.GovernanceType, f.Rules.VotingDeadlineHours,
		f.CreatedAt, f.UpdatedAt,
	)
	return err
}

func (r *FundRepo) GetByID(ctx context.Context, id uuid.UUID) (*fund.Fund, error) {
	return r.scanOne(ctx, `
		SELECT id, name, description, type, status, creator_id, currency,
		       contribution_amount, frequency, total_periods, start_date,
		       penalty_enabled, penalty_type, penalty_amount, grace_period_days,
		       min_members, max_members, governance_type, voting_deadline_hours,
		       created_at, updated_at
		FROM funds WHERE id = $1
	`, id)
}

func (r *FundRepo) ListByMember(ctx context.Context, userID uuid.UUID) ([]*fund.Fund, error) {
	rows, err := r.db.Query(ctx, `
		SELECT f.id, f.name, f.description, f.type, f.status, f.creator_id, f.currency,
		       f.contribution_amount, f.frequency, f.total_periods, f.start_date,
		       f.penalty_enabled, f.penalty_type, f.penalty_amount, f.grace_period_days,
		       f.min_members, f.max_members, f.governance_type, f.voting_deadline_hours,
		       f.created_at, f.updated_at
		FROM funds f
		JOIN fund_members fm ON fm.fund_id = f.id
		WHERE fm.user_id = $1 AND fm.status = 'active'
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funds []*fund.Fund
	for rows.Next() {
		f, err := scanFund(rows)
		if err != nil {
			return nil, err
		}
		funds = append(funds, f)
	}
	return funds, rows.Err()
}

func (r *FundRepo) Update(ctx context.Context, f *fund.Fund) error {
	f.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE funds SET
			name=$1, description=$2, status=$3,
			contribution_amount=$4, frequency=$5, total_periods=$6,
			penalty_enabled=$7, penalty_type=$8, penalty_amount=$9,
			grace_period_days=$10, governance_type=$11, voting_deadline_hours=$12,
			updated_at=$13
		WHERE id=$14
	`,
		f.Name, f.Description, f.Status,
		f.Rules.ContributionAmount, f.Rules.Frequency, f.Rules.TotalPeriods,
		f.Rules.PenaltyEnabled, nullablePenaltyType(f.Rules.PenaltyType), f.Rules.PenaltyAmount,
		f.Rules.GracePeriodDays, f.Rules.GovernanceType, f.Rules.VotingDeadlineHours,
		f.UpdatedAt, f.ID,
	)
	return err
}

func (r *FundRepo) scanOne(ctx context.Context, query string, args ...any) (*fund.Fund, error) {
	row := r.db.QueryRow(ctx, query, args...)
	f, err := scanFund(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return f, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanFund(row scanner) (*fund.Fund, error) {
	f := &fund.Fund{}
	var penaltyType *string
	var contribAmount, penaltyAmount decimal.Decimal

	err := row.Scan(
		&f.ID, &f.Name, &f.Description, &f.Type, &f.Status, &f.CreatorID, &f.Currency,
		&contribAmount, &f.Rules.Frequency, &f.Rules.TotalPeriods, &f.Rules.StartDate,
		&f.Rules.PenaltyEnabled, &penaltyType, &penaltyAmount, &f.Rules.GracePeriodDays,
		&f.Rules.MinMembers, &f.Rules.MaxMembers,
		&f.Rules.GovernanceType, &f.Rules.VotingDeadlineHours,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	f.Rules.ContributionAmount = contribAmount
	f.Rules.PenaltyAmount = penaltyAmount
	if penaltyType != nil {
		f.Rules.PenaltyType = fund.PenaltyType(*penaltyType)
	}
	return f, nil
}

func nullablePenaltyType(pt fund.PenaltyType) *string {
	if pt == "" {
		return nil
	}
	s := string(pt)
	return &s
}

// --- MemberRepo ---

type MemberRepo struct {
	db *pgxpool.Pool
}

func NewMemberRepo(db *pgxpool.Pool) *MemberRepo {
	return &MemberRepo{db: db}
}

func (r *MemberRepo) Add(ctx context.Context, m *fund.FundMember) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO fund_members (id, fund_id, user_id, role, status, payout_order, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, m.ID, m.FundID, m.UserID, m.Role, m.Status, m.PayoutOrder, m.JoinedAt)
	return err
}

func (r *MemberRepo) GetByFundAndUser(ctx context.Context, fundID, userID uuid.UUID) (*fund.FundMember, error) {
	return r.scanMember(ctx, `
		SELECT id, fund_id, user_id, role, status, payout_order, joined_at
		FROM fund_members WHERE fund_id=$1 AND user_id=$2
	`, fundID, userID)
}

func (r *MemberRepo) GetByID(ctx context.Context, id uuid.UUID) (*fund.FundMember, error) {
	return r.scanMember(ctx, `
		SELECT id, fund_id, user_id, role, status, payout_order, joined_at
		FROM fund_members WHERE id=$1
	`, id)
}

func (r *MemberRepo) ListByFund(ctx context.Context, fundID uuid.UUID) ([]*fund.FundMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, fund_id, user_id, role, status, payout_order, joined_at
		FROM fund_members WHERE fund_id=$1
		ORDER BY joined_at ASC
	`, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*fund.FundMember
	for rows.Next() {
		m := &fund.FundMember{}
		if err := rows.Scan(&m.ID, &m.FundID, &m.UserID, &m.Role, &m.Status, &m.PayoutOrder, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *MemberRepo) Update(ctx context.Context, m *fund.FundMember) error {
	_, err := r.db.Exec(ctx, `
		UPDATE fund_members SET role=$1, status=$2, payout_order=$3 WHERE id=$4
	`, m.Role, m.Status, m.PayoutOrder, m.ID)
	return err
}

func (r *MemberRepo) CountActive(ctx context.Context, fundID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM fund_members WHERE fund_id=$1 AND status='active'
	`, fundID).Scan(&count)
	return count, err
}

func (r *MemberRepo) scanMember(ctx context.Context, query string, args ...any) (*fund.FundMember, error) {
	m := &fund.FundMember{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&m.ID, &m.FundID, &m.UserID, &m.Role, &m.Status, &m.PayoutOrder, &m.JoinedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return m, nil
}
