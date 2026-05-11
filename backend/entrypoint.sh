#!/bin/sh
set -e

DB_URL="postgres://${DATABASE_USER:-sacramento}:${DATABASE_PASSWORD:-sacramento123}@${DATABASE_HOST:-postgres}:${DATABASE_PORT:-5432}/${DATABASE_NAME:-sacramento_finance}?sslmode=${DATABASE_SSL_MODE:-disable}"

echo "Running database migrations..."
goose -dir ./migrations postgres "$DB_URL" up

echo "Starting Sacramento Finance API..."
exec ./sacramento-api
