# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

Tianfu Bank is a Go microservices monorepo implementing a banking system with two services:
- **account-service** — manages account creation and balance reads; runs on port 8080
- **transaction-service** — event-sourced transaction processing (deposit/withdraw/transfer); runs on port 8081

Services share a `shared/` module containing RabbitMQ connection helpers and common event types.

## Build System

The project uses both **Go workspaces** and **Bazel**:

- Go workspace root: `go.work` (links `shared/`, `services/account-service/`, `services/transaction-service/`)
- Each service has its own `go.mod`; `shared` is referenced via `replace` directive pointing to `../../shared`
- Bazel build files (`.bazel`) exist alongside Go modules for hermetic builds

### Running services (Go)

```bash
# From service directory
cd services/account-service && go run .
cd services/transaction-service && go run .
```

### Bazel builds

```bash
bazel build //services/account-service:account-service
bazel build //services/transaction-service:...
```

## Infrastructure (Docker Compose)

Start all dependencies before running services locally:

```bash
docker compose up -d
```

This starts:
- **RabbitMQ** — `localhost:5672` (management UI: `localhost:15672`, user/pass: `guest/guest`)
- **account_db** — PostgreSQL on `localhost:5432`, DB: `account_db`
- **transaction_db** — PostgreSQL on `localhost:5433`, DB: `transaction_db`
- **pgAdmin** — `localhost:5050` (admin@admin.com / admin)
- **LGTM (Grafana/OpenTelemetry)** — Grafana: `localhost:3000`, OTLP gRPC: `localhost:4317`

Database migrations run automatically via Docker init scripts from `services/*/db/migrations/`.

## Configuration

Both services load config from environment variables with sensible defaults (localhost, standard ports). Key env vars:

| Variable | Default |
|---|---|
| `SERVER_PORT` | `8080` |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `localhost/5432/postgres/password/<service>_db` |
| `RABBITMQ_HOST` / `RABBITMQ_PORT` / `RABBITMQ_USER` / `RABBITMQ_PASSWORD` | `localhost/5672/guest/guest` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` |

## Architecture

### Hexagonal (Ports & Adapters)

Each service follows strict hexagonal architecture:

```
core/
  domain/<domain>/
    entity.go      — domain structs and constants
    port.go        — interfaces (repository, service, publisher contracts)
    service.go     — business logic implementation
    aggregate.go   — (transaction-service only) event-sourced aggregate
adapter/
  primary/         — inbound adapters
    http/          — Fiber HTTP handlers + routes
    mq/account/    — RabbitMQ consumers
  secondary/       — outbound adapters
    infrastructure/
      config/      — env config loading
      database/    — GORM/PostgreSQL connection
    messaging/rabbitmq/  — RabbitMQ publishers
    repository/    — GORM repository implementations
```

### Event Sourcing (transaction-service)

The transaction-service uses event sourcing for account balances:
- **EventStore** — append-only log of `TransactionEvent` records (deposited/withdrawn/transfer_in/transfer_out)
- **Snapshots** — created every 1000 events (`SnapshotInterval`) to bound rehydration cost
- **Rehydration** — on each transaction: load latest snapshot + all events since → rebuild `AccountAggregate`
- **Idempotency** — all mutation endpoints require `X-Idempotency-Key` header; results are cached in `idempotency` table
- **Unit of Work** — rehydration, event append, snapshot, and idempotency record saved in a single DB transaction

### Cross-Service Messaging (RabbitMQ)

Two event flows via `shared/messaging`:

1. **account-service** publishes `AccountCreatedEvent` → **transaction-service** consumes it to initialize a snapshot entry (sequence 0) so the aggregate exists for future transactions.

2. **transaction-service** publishes `BalanceUpdatedEvent` → **account-service** consumes it to sync the denormalized `balance` column on the accounts table.

### API Routes

**account-service** (`/api/v1`):
- `POST /accounts` — create account (`userId`, `branchId`, `accountType`)
- `GET /accounts/:accountId` — get account by ID
- `GET /health`

**transaction-service** (`/api/v1`):
- `POST /transactions/deposit` — requires `X-Idempotency-Key` header
- `POST /transactions/withdraw` — requires `X-Idempotency-Key` header
- `POST /transactions/transfer` — requires `X-Idempotency-Key` header
- `GET /transactions/history/:accountId?limit=50&offset=0`
- `GET /health`

## Module Path

The Go module path is `github.com/zhunismp/tianfu-bank/...` (note: `zhunismp`, not `tianfu-bank`).

## Key Libraries

- **HTTP**: [Fiber v2](https://github.com/gofiber/fiber) — handlers return `fiber.Ctx` errors, not standard `http.Handler`
- **ORM**: GORM with PostgreSQL driver
- **Messaging**: `rabbitmq/amqp091-go` via `shared/messaging` helpers
- **Decimal**: `shopspring/decimal` — use this for all monetary amounts, never `float64`
- **Observability**: OpenTelemetry (traces + metrics exported via OTLP gRPC)

## Error Handling

Use `shared/apperror.New(code, message, err)` for domain errors. `AppError` carries a string `Code` (e.g. `"INSUFFICIENT_FUNDS"`), a human-readable `Message`, and wraps the original `Err`.

## Tests

There are currently no test files in this repository.
