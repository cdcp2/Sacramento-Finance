-- +goose Up
CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fund_id    UUID REFERENCES funds(id) ON DELETE CASCADE,
    type       VARCHAR(50) NOT NULL,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user   ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_fund   ON notifications(fund_id) WHERE fund_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS notifications;
