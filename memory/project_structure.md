---
name: Sacramento Finance - Project Structure
description: Folder layout and key file locations for the backend
type: project
---

Clean Architecture layout:
- `cmd/api/main.go` — entry point, dependency wiring
- `internal/domain/` — pure domain entities and repository interfaces (no framework imports)
  - `user/`, `fund/`, `ledger/` — each has `entity.go` and `repository.go`
- `internal/usecase/` — application business logic
  - `auth/register.go`, `auth/login.go` — implemented
- `internal/delivery/http/` — Gin handlers, middleware, router
  - `handler/auth_handler.go`, `user_handler.go`, `fund_handler.go`
  - `middleware/auth.go` (JWT), `middleware/logger.go`
  - `router.go` — all routes registered here
- `internal/infrastructure/` — concrete implementations
  - `postgres/db.go` — pgxpool setup
  - `postgres/migrations/` — goose SQL migrations (001 users, 002 funds, 003 ledger)
  - `repository/user_repo.go`, `fund_repo.go` — raw pgx queries
- `internal/config/config.go` — viper config (defaults to localhost postgres)
- `pkg/apperror/`, `pkg/money/`, `pkg/idgen/` — shared utilities
- `docker-compose.yml` — PostgreSQL only
- `Makefile` — targets: run, build, test, docker-up, migrate-up, migrate-down

3 fund types: circulo (rotating savings), vaca (shared goal), fondo_ahorro (traditional savings fund).
Fund has state machine: draft → active → paused/completed/cancelled.

**How to apply:** Follow Clean Architecture layers. New features: add domain entity/interface first, then use case, then handler. Never import framework packages in domain or usecase layers.
