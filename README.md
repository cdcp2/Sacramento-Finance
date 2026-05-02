# Sacramento Finance

Backend MVP for a Colombian collective savings fintech prototype.

The app supports three simulated savings products:

- `circulo`: rotating savings circle.
- `vaca`: shared savings goal.
- `fondo_ahorro`: traditional savings fund.

## Backend

Stack:

- Go 1.22
- Gin
- PostgreSQL 16
- pgx
- goose migrations
- JWT authentication

Run locally:

```bash
cd backend
make setup
make run
```

Run tests:

```bash
cd backend
go test ./...
```

Health check:

```bash
curl http://localhost:8080/health
```

API contract for frontend work:

- [backend/API.md](backend/API.md)
