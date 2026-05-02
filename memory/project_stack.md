---
name: Sacramento Finance - Backend Stack
description: Tech stack and framework decisions for the Go backend MVP
type: project
---

Backend is being built in Go with Gin framework (not Echo/Fiber/Chi). Module path: `github.com/sacramento-finance/backend`. Located at `/home/carleto/Documents/Sacramento-Finance/backend/`.

Stack: Gin v1.10, pgx v5, sqlc (planned), zerolog, shopspring/decimal, golang-jwt/jwt v5, argon2id (planned, currently bcrypt), spf13/viper for config.

Database: PostgreSQL 16 via Docker Compose (user: sacramento, pass: sacramento123, db: sacramento_finance, port: 5432).

Migration tool: goose (SQL files in `internal/infrastructure/postgres/migrations/`).

Money: always shopspring/decimal, stored as NUMERIC(19,4) in Postgres. Never float64.

Auth: JWT HS256 for MVP (RS256 upgrade planned before production). Access token 15m, refresh 7 days (currently simple JWT, DB-backed revocation planned for Phase 2).

**Why:** Solo developer, needs speed. Gin chosen for community size (most StackOverflow/GitHub examples). Money is simulated — no real payment APIs yet (MVP demo phase).

**How to apply:** Recommend Gin-compatible solutions. No real financial API integrations needed yet. Keep changes simple and fast to ship.
