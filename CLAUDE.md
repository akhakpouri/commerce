# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Structure

This is a **Go workspace** (`go.work`) containing three modules:

| Module | Path | Purpose |
|--------|------|---------|
| `api` | `./api` | HTTP API executable (scaffolding — no routes yet) |
| `utils` | `./utils` | CLI tool for DB migrations |
| `internal/shared` | `./internal/shared` | Shared library: GORM models + DB connection |

`api` and `utils` both depend on `internal/shared`. All external dependencies (GORM, PostgreSQL driver, bcrypt) live only in `internal/shared`.

## Commands

All commands must be run from the specific module directory, not the workspace root.

**Build:**
```bash
(cd api && go build -o ../bin/api .)
(cd utils && go build -o ../bin/utils .)
```

**Run:**
```bash
(cd utils && go run .)   # loads config and runs DB migrations
(cd api && go run .)     # currently prints "hello, world!"
```

**Test:**
```bash
(cd api && go test ./...)
(cd utils && go test ./...)
(cd internal/shared && go test ./...)
```

**Lint** (golangci-lint required):
```bash
(cd api && golangci-lint run ./...)
(cd utils && golangci-lint run ./...)
(cd internal/shared && golangci-lint run ./...)
```

**Module tidy:**
```bash
(cd api && go mod tidy)
(cd utils && go mod tidy)
(cd internal/shared && go mod tidy)
go work sync
```

## Architecture

### internal/shared

The core library. Two packages:

- **`database`** — `Migrate(cfg DbConfig)` opens a GORM+PostgreSQL connection and calls `AutoMigrate` on all registered models. `setup.go` is where models are registered.
- **`models`** — Eight domain models, all embedding `Base` (UUID primary key, CreatedAt, UpdatedAt, soft-delete DeletedAt). All tables live in the `commerce` PostgreSQL schema.

Domain models: `User`, `Address`, `Product`, `Category`, `ProductCategory` (junction), `Review`, `Order`, `OrderItem`.

`User` has bcrypt hooks (`BeforeCreate`, `BeforeUpdate`) that auto-hash the `Password` field, and a `CheckPassword()` method.

`Order` uses string enum types `OrderStatus` and `PaymentStatus` defined in the same file.

### utils

`main.go` calls `managers.NewDbConfig("configs/config.json")` then `database.Migrate(cfg)`.

`managers.NewDbConfig` reads from `utils/configs/config.json` (embedded via `//go:embed`). If JSON parsing fails, it falls back to environment variables: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `DB_SCHEMA`.

### api

`internal/handlers/` is empty — no HTTP framework has been chosen yet (ADR-004 pending).

`internal/dto/` and `internal/services/` are fully implemented. One sub-package per domain under each. Services own all business logic and return DTOs; repositories are injected at construction time. Unit tests live alongside each service (`*_test.go`) using gomock-generated mocks.

## Database

PostgreSQL 13+, schema `commerce`. Setup SQL (from readme):

```sql
CREATE DATABASE commerce;
CREATE USER commerce WITH ENCRYPTED PASSWORD 'commerce@123';
GRANT ALL PRIVILEGES ON DATABASE commerce TO commerce;
\c commerce
CREATE SCHEMA commerce AUTHORIZATION commerce;
```

Local dev config lives in `utils/configs/config.json`.

## Linter Config

`.golangci.yml` at workspace root enables: `errcheck`, `ineffassign`, `unused`, `govet`, `staticcheck`.

## Project Memory System

Notes live in `docs/project-notes/`:

| File           | Purpose                              |
|----------------|--------------------------------------|
| `bugs.md`      | Bug log with root causes and fixes   |
| `decisions.md` | Architectural decision records (ADR) |
| `facts.md`     | Config, constants, connection info   |
| `issues.md`    | Work log with branch/ticket refs     |

### Memory-Aware Protocols

**Before proposing architectural changes:**
- Check `docs/project-notes/decisions.md` for existing decisions.
- Verify the proposed approach doesn't conflict with past choices.

**When encountering errors or bugs:**
- Search `docs/project-notes/bugs.md` for similar issues.
- Apply known fixes if found.
- Document new bugs and solutions when resolved.

**When looking up project configuration:**
- Check `docs/project-notes/facts.md` for credentials, ports, connection strings, and env vars.
- Prefer documented facts over assumptions.
