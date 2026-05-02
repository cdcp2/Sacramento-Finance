BINARY_NAME=sacramento-api
MAIN_PATH=./cmd/api/main.go
MIGRATIONS_DIR=./internal/infrastructure/postgres/migrations
DB_URL=postgres://sacramento:sacramento123@localhost:5432/sacramento_finance?sslmode=disable

.PHONY: run build test migrate-up migrate-down migrate-create sqlc-gen tidy docker-up docker-down

## Run the API server
run:
	go run $(MAIN_PATH)

## Build binary
build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

## Run all tests
test:
	go test ./... -v -race

## Run unit tests only (no DB required)
test-unit:
	go test ./internal/domain/... ./pkg/... -v

## Start Docker services
docker-up:
	docker compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker compose exec postgres pg_isready -U sacramento -d sacramento_finance > /dev/null 2>&1; do sleep 1; done
	@echo "PostgreSQL ready."

## Stop Docker services
docker-down:
	docker compose down

## Apply all pending migrations
migrate-up:
	@which goose > /dev/null || go install github.com/pressly/goose/v3/cmd/goose@latest
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up

## Rollback last migration
migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" down

## Create a new migration: make migrate-create name=add_something
migrate-create:
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

## Show migration status
migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" status

## Generate sqlc code from SQL queries
sqlc-gen:
	@which sqlc > /dev/null || go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	sqlc generate

## Tidy go modules
tidy:
	go mod tidy

## Full local setup from scratch
setup: docker-up migrate-up
	@echo "Setup complete. Run 'make run' to start the server."
