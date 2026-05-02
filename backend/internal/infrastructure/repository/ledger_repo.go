package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/shopspring/decimal"
)

type LedgerRepo struct {
	db *pgxpool.Pool
}

func NewLedgerRepo(db *pgxpool.Pool) *LedgerRepo {
	return &LedgerRepo{db: db}
}

// RecordPaymentTx writes the ledger entry, updates the payment, and updates the fund
// balance — all inside a single transaction. This is the only correct way to record money.
func (r *LedgerRepo) RecordPaymentTx(ctx context.Context, p *ledger.Payment, entry *ledger.LedgerEntry) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Lock the fund balance row to prevent concurrent updates
	var currentBalance decimal.Decimal
	err = tx.QueryRow(ctx, `
		SELECT balance FROM fund_balances WHERE fund_id = $1 FOR UPDATE
	`, entry.FundID).Scan(&currentBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		// First movement in this fund — initialize balance row
		_, err = tx.Exec(ctx, `
			INSERT INTO fund_balances (fund_id, balance) VALUES ($1, 0)
		`, entry.FundID)
		if err != nil {
			return err
		}
		currentBalance = decimal.Zero
	} else if err != nil {
		return err
	}

	newBalance := currentBalance.Add(entry.Amount)
	entry.BalanceAfter = newBalance

	// Insert immutable ledger entry
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (
			id, fund_id, user_id, type, direction, amount,
			balance_after, reference_id, description, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`,
		entry.ID, entry.FundID, entry.UserID, entry.Type, entry.Direction,
		entry.Amount, entry.BalanceAfter, entry.ReferenceID, entry.Description, entry.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Update fund balance
	_, err = tx.Exec(ctx, `
		UPDATE fund_balances SET balance = $1, updated_at = NOW() WHERE fund_id = $2
	`, newBalance, entry.FundID)
	if err != nil {
		return err
	}

	// Update payment status
	_, err = tx.Exec(ctx, `
		UPDATE payments SET
			amount_paid=$1, status=$2, paid_at=$3,
			penalty_applied=$4, penalty_amount=$5, updated_at=NOW()
		WHERE id=$6
	`, p.AmountPaid, p.Status, p.PaidAt, p.PenaltyApplied, p.PenaltyAmount, p.ID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *LedgerRepo) GetFundBalance(ctx context.Context, fundID uuid.UUID) (decimal.Decimal, error) {
	var balance decimal.Decimal
	err := r.db.QueryRow(ctx, `
		SELECT balance FROM fund_balances WHERE fund_id = $1
	`, fundID).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, nil
	}
	return balance, err
}

func (r *LedgerRepo) ListByFund(ctx context.Context, fundID uuid.UUID, limit, offset int) ([]*ledger.LedgerEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, fund_id, user_id, type, direction, amount,
		       balance_after, reference_id, description, created_at
		FROM ledger_entries
		WHERE fund_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, fundID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (r *LedgerRepo) ListByMember(ctx context.Context, fundID, userID uuid.UUID, limit, offset int) ([]*ledger.LedgerEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, fund_id, user_id, type, direction, amount,
		       balance_after, reference_id, description, created_at
		FROM ledger_entries
		WHERE fund_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, fundID, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

// RecordEntriesTx atomically inserts one or more ledger entries and updates the fund balance.
// Credits (DirectionCredit) increase the balance; debits (DirectionDebit) decrease it.
// All entries must belong to the same fund (uses entries[0].FundID to lock the balance row).
func (r *LedgerRepo) RecordEntriesTx(ctx context.Context, entries []*ledger.LedgerEntry) error {
	if len(entries) == 0 {
		return nil
	}
	fundID := entries[0].FundID

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var currentBalance decimal.Decimal
	err = tx.QueryRow(ctx, `
		SELECT balance FROM fund_balances WHERE fund_id = $1 FOR UPDATE
	`, fundID).Scan(&currentBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `INSERT INTO fund_balances (fund_id, balance) VALUES ($1, 0)`, fundID)
		if err != nil {
			return err
		}
		currentBalance = decimal.Zero
	} else if err != nil {
		return err
	}

	running := currentBalance
	for _, entry := range entries {
		if entry.Direction == ledger.DirectionCredit {
			running = running.Add(entry.Amount)
		} else {
			running = running.Sub(entry.Amount)
		}
		entry.BalanceAfter = running

		_, err = tx.Exec(ctx, `
			INSERT INTO ledger_entries (
				id, fund_id, user_id, type, direction, amount,
				balance_after, reference_id, description, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`,
			entry.ID, entry.FundID, entry.UserID, entry.Type, entry.Direction,
			entry.Amount, entry.BalanceAfter, entry.ReferenceID, entry.Description, entry.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE fund_balances SET balance = $1, updated_at = NOW() WHERE fund_id = $2
	`, running, fundID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetMemberBalance returns the net balance for a specific user in a fund
// by summing credits minus debits in the ledger.
func (r *LedgerRepo) GetMemberBalance(ctx context.Context, fundID, userID uuid.UUID) (decimal.Decimal, error) {
	var balance decimal.Decimal
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(
			CASE WHEN direction = 'credit' THEN amount ELSE -amount END
		), 0)
		FROM ledger_entries
		WHERE fund_id = $1 AND user_id = $2
	`, fundID, userID).Scan(&balance)
	if err != nil {
		return decimal.Zero, err
	}
	return balance, nil
}

func scanEntries(rows pgx.Rows) ([]*ledger.LedgerEntry, error) {
	var entries []*ledger.LedgerEntry
	for rows.Next() {
		e := &ledger.LedgerEntry{}
		var amount, balanceAfter decimal.Decimal
		if err := rows.Scan(
			&e.ID, &e.FundID, &e.UserID, &e.Type, &e.Direction,
			&amount, &balanceAfter, &e.ReferenceID, &e.Description, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		e.Amount = amount
		e.BalanceAfter = balanceAfter
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
