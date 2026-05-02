-- +goose Up
CREATE TYPE entry_type AS ENUM ('deposit', 'withdrawal', 'penalty', 'payout', 'interest', 'refund');
CREATE TYPE entry_direction AS ENUM ('credit', 'debit');
CREATE TYPE payment_status AS ENUM ('pending', 'partial', 'paid', 'missed', 'waived');

-- Simulated fund balances (since we're not moving real money)
CREATE TABLE fund_balances (
    fund_id     UUID PRIMARY KEY REFERENCES funds(id) ON DELETE CASCADE,
    balance     NUMERIC(19,4) NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Immutable financial ledger — never UPDATE or DELETE rows here
CREATE TABLE ledger_entries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fund_id       UUID NOT NULL REFERENCES funds(id),
    user_id       UUID REFERENCES users(id), -- NULL for system entries
    type          entry_type NOT NULL,
    direction     entry_direction NOT NULL,
    amount        NUMERIC(19,4) NOT NULL CHECK (amount > 0),
    balance_after NUMERIC(19,4) NOT NULL, -- fund balance snapshot after this entry
    reference_id  UUID,                   -- links to payments or payouts
    description   TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- No updated_at: ledger entries are immutable
);

CREATE INDEX idx_ledger_fund      ON ledger_entries(fund_id, created_at DESC);
CREATE INDEX idx_ledger_user      ON ledger_entries(user_id, created_at DESC);
CREATE INDEX idx_ledger_reference ON ledger_entries(reference_id) WHERE reference_id IS NOT NULL;

-- Payment schedule
CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fund_id         UUID NOT NULL REFERENCES funds(id),
    member_id       UUID NOT NULL REFERENCES fund_members(id),
    period_number   INT NOT NULL,
    due_date        DATE NOT NULL,
    amount_due      NUMERIC(19,4) NOT NULL,
    amount_paid     NUMERIC(19,4) NOT NULL DEFAULT 0,
    status          payment_status NOT NULL DEFAULT 'pending',
    paid_at         TIMESTAMPTZ,
    penalty_applied BOOLEAN NOT NULL DEFAULT FALSE,
    penalty_amount  NUMERIC(19,4) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(fund_id, member_id, period_number)
);

CREATE INDEX idx_payments_fund     ON payments(fund_id);
CREATE INDEX idx_payments_member   ON payments(member_id);
CREATE INDEX idx_payments_due_date ON payments(due_date, status);

-- Payouts (for Circulo rounds)
CREATE TABLE payouts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fund_id        UUID NOT NULL REFERENCES funds(id),
    recipient_id   UUID NOT NULL REFERENCES fund_members(id),
    round_number   INT NOT NULL,
    amount         NUMERIC(19,4) NOT NULL,
    status         VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    scheduled_date DATE NOT NULL,
    completed_at   TIMESTAMPTZ,
    UNIQUE(fund_id, round_number)
);

-- +goose Down
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS fund_balances;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS entry_direction;
DROP TYPE IF EXISTS entry_type;
