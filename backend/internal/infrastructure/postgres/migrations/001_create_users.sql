-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cedula      VARCHAR(12) NOT NULL UNIQUE,
    email       VARCHAR(255) NOT NULL UNIQUE,
    phone       VARCHAR(20) NOT NULL,
    full_name   VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_cedula ON users(cedula);
CREATE INDEX idx_users_email  ON users(email);

-- +goose Down
DROP TABLE IF EXISTS users;
