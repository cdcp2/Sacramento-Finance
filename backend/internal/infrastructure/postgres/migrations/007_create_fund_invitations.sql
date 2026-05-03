-- +goose Up
CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'rejected', 'cancelled', 'expired');

CREATE TABLE fund_invitations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fund_id      UUID NOT NULL REFERENCES funds(id) ON DELETE CASCADE,
    inviter_id   UUID NOT NULL REFERENCES users(id),
    invitee_id   UUID NOT NULL REFERENCES users(id),
    status       invitation_status NOT NULL DEFAULT 'pending',
    message      TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ NOT NULL,
    responded_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (inviter_id <> invitee_id)
);

CREATE UNIQUE INDEX idx_fund_invitations_one_pending
    ON fund_invitations(fund_id, invitee_id)
    WHERE status = 'pending';

CREATE INDEX idx_fund_invitations_invitee
    ON fund_invitations(invitee_id, status, created_at DESC);

CREATE INDEX idx_fund_invitations_fund
    ON fund_invitations(fund_id, status, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS fund_invitations;
DROP TYPE IF EXISTS invitation_status;
