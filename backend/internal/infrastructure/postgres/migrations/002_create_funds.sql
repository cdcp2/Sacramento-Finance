-- +goose Up
CREATE TYPE fund_type AS ENUM ('circulo', 'vaca', 'fondo_ahorro');
CREATE TYPE fund_status AS ENUM ('draft', 'active', 'paused', 'completed', 'cancelled');
CREATE TYPE period_frequency AS ENUM ('weekly', 'biweekly', 'monthly');
CREATE TYPE penalty_type AS ENUM ('fixed', 'percentage');
CREATE TYPE member_role AS ENUM ('admin', 'member');
CREATE TYPE member_status AS ENUM ('active', 'suspended', 'left');

CREATE TABLE funds (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    type                fund_type NOT NULL,
    status              fund_status NOT NULL DEFAULT 'draft',
    creator_id          UUID NOT NULL REFERENCES users(id),
    currency            VARCHAR(3) NOT NULL DEFAULT 'COP',
    -- Rules (embedded for simplicity, separate table if rules become more complex)
    contribution_amount NUMERIC(19,4) NOT NULL,
    frequency           period_frequency NOT NULL,
    total_periods       INT NOT NULL CHECK (total_periods > 0),
    start_date          DATE NOT NULL,
    penalty_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    penalty_type        penalty_type,
    penalty_amount      NUMERIC(19,4) NOT NULL DEFAULT 0,
    grace_period_days   INT NOT NULL DEFAULT 0,
    min_members         INT NOT NULL DEFAULT 2,
    max_members         INT NOT NULL DEFAULT 50,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE fund_members (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fund_id      UUID NOT NULL REFERENCES funds(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id),
    role         member_role NOT NULL DEFAULT 'member',
    status       member_status NOT NULL DEFAULT 'active',
    payout_order INT,         -- Only for circulo
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(fund_id, user_id)
);

CREATE INDEX idx_fund_members_fund   ON fund_members(fund_id);
CREATE INDEX idx_fund_members_user   ON fund_members(user_id);
CREATE INDEX idx_funds_creator       ON funds(creator_id);
CREATE INDEX idx_funds_status        ON funds(status);

-- Circulo-specific config
CREATE TABLE circulo_configs (
    fund_id           UUID PRIMARY KEY REFERENCES funds(id) ON DELETE CASCADE,
    payout_order_type VARCHAR(20) NOT NULL DEFAULT 'fixed', -- 'fixed' | 'randomized'
    current_round     INT NOT NULL DEFAULT 1,
    rounds_completed  INT NOT NULL DEFAULT 0
);

-- Vaca-specific config
CREATE TABLE vaca_configs (
    fund_id           UUID PRIMARY KEY REFERENCES funds(id) ON DELETE CASCADE,
    goal_amount       NUMERIC(19,4) NOT NULL,
    goal_description  TEXT NOT NULL DEFAULT '',
    distribution_type VARCHAR(30) NOT NULL DEFAULT 'goal_reached' -- 'goal_reached' | 'unanimous_vote'
);

-- Fondo de ahorro specific config
CREATE TABLE fondo_configs (
    fund_id                  UUID PRIMARY KEY REFERENCES funds(id) ON DELETE CASCADE,
    interest_rate            NUMERIC(8,4) NOT NULL DEFAULT 0,
    early_withdrawal_penalty NUMERIC(8,4) NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS fondo_configs;
DROP TABLE IF EXISTS vaca_configs;
DROP TABLE IF EXISTS circulo_configs;
DROP TABLE IF EXISTS fund_members;
DROP TABLE IF EXISTS funds;
DROP TYPE IF EXISTS member_status;
DROP TYPE IF EXISTS member_role;
DROP TYPE IF EXISTS penalty_type;
DROP TYPE IF EXISTS period_frequency;
DROP TYPE IF EXISTS fund_status;
DROP TYPE IF EXISTS fund_type;
