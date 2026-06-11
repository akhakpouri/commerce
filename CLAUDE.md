# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Structure

This is a **Go workspace** (`go.work`) containing three modules:

| Module | Path | Purpose |
|--------|------|---------|
| `api` | `./api` | HTTP API executable (Gin, service + handler layers implemented) |
| `utils` | `./utils` | CLI tool for DB migrations |
| `internal/shared` | `./internal/shared` | Shared library: GORM models, repositories, DB connection |

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
(cd api && go run .)     # starts Gin HTTP server on SERVER_ADDRESS
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

**Docker (local dev):**
```bash
docker compose up           # builds + runs api and utils; Postgres must be externally reachable via .env
docker compose down         # tears down containers
```
Requires a root `.env` (see `.env.example`). Postgres is not run by compose — it's expected to be externally provisioned (ADR-016 amendment).

## Continuous Integration

`.github/workflows/go.yml` runs on every **pull request**: a single `build` job that builds and tests all three modules per-module (the per-module rule above applies — there is no root `go.mod`). The Go version comes from `go-version-file: api/go.mod`. A failing run **blocks merge to `main`** via the required status check named `build` (branch protection, configured server-side via the GitHub API — not in the repo).

**Gitignored files the build needs are regenerated on the runner, never committed** (deliberate, to keep `go.work`/config out of git):

| File | Why it's needed | CI step that recreates it |
|------|-----------------|---------------------------|
| `go.work` (+ `go.work.sum`) | Only thing wiring the 3 modules together; without it cross-module imports fail to compile (`package ... is not in std`) | `go work init ./api ./internal/shared ./utils` |
| `utils/configs/config.json` | `utils/main.go` `//go:embed configs/config.json` requires the file at compile time | `cp utils/configs/config.example utils/configs/config.json` |

**When adding a fourth module**, update the `go work init` step too — the workspace `use` list is duplicated between the (gitignored) local `go.work` and this CI step. If the `build` job is renamed, update the required-status-check context or every PR blocks forever.

## Architecture

### internal/shared

The core library. Three packages:

- **`database`** — two functions: `Connect(cfg DbConfig) (*gorm.DB, error)` opens and returns a GORM+PostgreSQL connection; `Migrate(cfg DbConfig)` calls `Connect` then runs `AutoMigrate`. `setup.go` is where models are registered. Both `api` and `utils` use `Connect` or `Migrate` — DSN construction lives here only (ADR-015).
- **`models`** — Nine domain models (`User`, `Address`, `Product`, `Category`, `ProductCategory`, `Review`, `Order`, `OrderItem`, `Payment`), all embedding `Base` (auto-increment PK, CreatedAt, UpdatedAt, soft-delete DeletedAt as `time.Time`). All tables live in the `commerce` PostgreSQL schema.
- **`repositories`** — one sub-package per domain. Each defines an interface and a concrete GORM implementation. Constructor takes `*gorm.DB`.

### utils

`main.go` embeds `configs/config.json` via `//go:embed`, passes bytes to `managers.NewDbConfig([]byte)`, then calls `database.Migrate(cfg)`. If JSON parsing fails, falls back to env vars: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `DB_SCHEMA`.

### api

HTTP server using Gin (ADR-004). Entry point is `main.go` — composition root for config, router, and server.

**Structure:**
- `configs/config.go` — `NewConfig()` loads `configs/dev.env` via godotenv; individual DB fields (not a DSN string); `Port` parsed to `int` at startup; `databaseConfig.Connect()` delegates to `database.Connect()` (ADR-015); `CorsNew()` returns Gin CORS middleware.
- `server/server.go` — `Server` struct with `Run()`: starts HTTP server in a goroutine, blocks on `SIGINT`/`SIGTERM`, graceful shutdown with 30s timeout.
- `server/router/routes.go` — `RegisterRoutes(*gin.Engine)`: handler wiring. Will receive a `*container.Container` once the container pattern is implemented.
- `internal/constants/constants.go` — typed structs for env key names and HTTP header names.
- `internal/handlers/` — one sub-package per domain. Each handler struct holds an injected service interface and exposes `RegisterRoutes(*gin.RouterGroup)`.
- `internal/dto/` and `internal/services/` — fully implemented. One sub-package per domain. Services own all business logic and return DTOs; repositories are injected at construction time.
- `docs/` — generated by `swag init -g main.go --output docs`; committed alongside code. Regenerate after changing handler annotations.

Unit tests live alongside each service (`*_test.go`) using gomock-generated mocks (`mock_*_test.go`).

**Known limitation:** `configs/dev.env` uses a relative path — binary must be run from `api/`.

## Database

PostgreSQL 13+, schema `commerce`. Setup SQL (from readme):

```sql
CREATE DATABASE commerce;
CREATE USER commerce WITH ENCRYPTED PASSWORD 'commerce@123';
GRANT ALL PRIVILEGES ON DATABASE commerce TO commerce;
\c commerce
CREATE SCHEMA commerce AUTHORIZATION commerce;
```

- `utils` config: `utils/configs/config.json` (gitignored; use `config.example` as template)
- `api` config: `api/configs/dev.env` (committed with placeholder credentials — update locally before running)

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
