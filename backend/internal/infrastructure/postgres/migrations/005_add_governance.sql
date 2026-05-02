-- +goose Up
CREATE TYPE governance_type AS ENUM ('admin_only', 'majority', 'unanimous');
CREATE TYPE proposal_type AS ENUM (
    'activate_fund', 'change_rules', 'waive_payment',
    'remove_member', 'cancel_fund', 'change_governance', 'distribute_vaca'
);
CREATE TYPE proposal_status AS ENUM ('open', 'approved', 'rejected', 'expired');
CREATE TYPE vote_choice AS ENUM ('yes', 'no');

-- Add governance fields to funds
ALTER TABLE funds
    ADD COLUMN governance_type       governance_type NOT NULL DEFAULT 'admin_only',
    ADD COLUMN voting_deadline_hours INT             NOT NULL DEFAULT 72;

CREATE TABLE proposals (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fund_id        UUID NOT NULL REFERENCES funds(id) ON DELETE CASCADE,
    proposer_id    UUID NOT NULL REFERENCES fund_members(id),
    type           proposal_type NOT NULL,
    status         proposal_status NOT NULL DEFAULT 'open',
    payload        JSONB NOT NULL DEFAULT '{}',
    votes_for      INT NOT NULL DEFAULT 0,
    votes_against  INT NOT NULL DEFAULT 0,
    quorum_needed  INT NOT NULL,   -- ceil(active_members / 2) for majority; all for unanimous
    total_members  INT NOT NULL,   -- snapshot at creation time
    deadline_at    TIMESTAMPTZ NOT NULL,
    resolved_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE votes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_id UUID NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    member_id   UUID NOT NULL REFERENCES fund_members(id),
    choice      vote_choice NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(proposal_id, member_id)  -- one vote per member per proposal
);

CREATE INDEX idx_proposals_fund   ON proposals(fund_id, status);
CREATE INDEX idx_proposals_status ON proposals(status, deadline_at);
CREATE INDEX idx_votes_proposal   ON votes(proposal_id);

-- +goose Down
DROP INDEX IF EXISTS idx_votes_proposal;
DROP INDEX IF EXISTS idx_proposals_status;
DROP INDEX IF EXISTS idx_proposals_fund;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS proposals;
ALTER TABLE funds DROP COLUMN IF EXISTS voting_deadline_hours;
ALTER TABLE funds DROP COLUMN IF EXISTS governance_type;
DROP TYPE IF EXISTS vote_choice;
DROP TYPE IF EXISTS proposal_status;
DROP TYPE IF EXISTS proposal_type;
DROP TYPE IF EXISTS governance_type;
