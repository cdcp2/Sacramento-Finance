-- +goose Up

-- Document type enum
CREATE TYPE user_document_type AS ENUM ('cedula_ciudadania', 'cedula_extranjeria', 'pasaporte');

-- Rename cedula → document_number
ALTER TABLE users RENAME COLUMN cedula TO document_number;

-- Add document_type (default existing rows to cedula_ciudadania)
ALTER TABLE users
    ADD COLUMN document_type user_document_type NOT NULL DEFAULT 'cedula_ciudadania';

-- Remove the per-column default now that existing rows are set
ALTER TABLE users ALTER COLUMN document_type DROP DEFAULT;

-- The old UNIQUE was on document_number alone.
-- Now uniqueness must be per (document_type, document_number) — two different people
-- can't share the same document number within the same document type.
ALTER TABLE users DROP CONSTRAINT users_cedula_key;
ALTER TABLE users ADD CONSTRAINT users_document_unique UNIQUE (document_type, document_number);

-- Update indexes
DROP INDEX IF EXISTS idx_users_cedula;
CREATE INDEX idx_users_document ON users(document_type, document_number);

-- Verification fields (for Truora or similar — Phase 2)
ALTER TABLE users
    ADD COLUMN verification_status VARCHAR(20) NOT NULL DEFAULT 'none',
    ADD COLUMN verification_ref    VARCHAR(100),
    ADD COLUMN verified_at         TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS verified_at,
    DROP COLUMN IF EXISTS verification_ref,
    DROP COLUMN IF EXISTS verification_status;

DROP INDEX IF EXISTS idx_users_document;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_document_unique;
ALTER TABLE users DROP COLUMN IF EXISTS document_type;
ALTER TABLE users RENAME COLUMN document_number TO cedula;
ALTER TABLE users ADD CONSTRAINT users_cedula_key UNIQUE (cedula);
CREATE INDEX idx_users_cedula ON users(cedula);

DROP TYPE IF EXISTS user_document_type;
