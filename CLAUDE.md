# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Structure

This is a **Go workspace** (`go.work`) containing four modules:

| Module | Path | Purpose |
|--------|------|---------|
| `api` | `./apps/api` | HTTP API executable (Gin, service + handler layers implemented) |
| `utils` | `./apps/utils` | CLI tool for DB migrations |
| `relay` | `./apps/relay` | Standalone daemon draining the transactional outbox to the event broker (ADR-018). Producer-only — see `apps/relay/CLAUDE.md`. |
| `internal/shared` | `./internal/shared` | Shared library: GORM models, repositories, DB connection |

**Layout rule (ADR-022): `apps/` holds every deployable module; `internal/shared` is the library.** New deployables (`notifier`, `janitor`, `shipping`) go under `apps/` too.

**Module paths stay flat and do *not* mirror the directory.** `apps/api` is `module commerce/api`, `apps/utils` is `module commerce/utils`, `apps/relay` is `module commerce/relay` — there is no `commerce/apps/...` import path anywhere. This is deliberate: it keeps imports short and makes future directory moves free of Go source changes. When editing docs, a `commerce/api/...` string is an **import path and must not be rewritten**; only filesystem paths gained the `apps/` prefix.

`api`, `utils`, and `relay` all depend on `internal/shared`. No module `require`s a sibling — they are wired solely by `go.work`, which is why CI must run `go work init` before anything compiles. All external dependencies (GORM, PostgreSQL driver, bcrypt) live only in `internal/shared`; `relay` additionally pulls in `aws-sdk-go-v2` directly for SQS.

## Commands

All commands must be run from the specific module directory, not the workspace root.

**Build:**
```bash
(cd apps/api && go build -o ../../bin/api .)
(cd apps/utils && go build -o ../../bin/utils .)
(cd apps/relay && go build -o ../../bin/relay .)
```

**Run:**
```bash
(cd apps/utils && go run .)   # loads config and runs DB migrations
(cd apps/api && go run .)     # starts Gin HTTP server on SERVER_ADDRESS
(cd apps/relay && go run .)   # starts the outbox-draining daemon
```

**Test:**
```bash
(cd apps/api && go test ./...)
(cd apps/utils && go test ./...)
(cd internal/shared && go test ./...)
(cd apps/relay && go test ./...)
```

**Lint** (golangci-lint required):
```bash
(cd apps/api && golangci-lint run ./...)
(cd apps/utils && golangci-lint run ./...)
(cd internal/shared && golangci-lint run ./...)
(cd apps/relay && golangci-lint run ./...)
```

**Module tidy:**
```bash
(cd apps/api && go mod tidy)
(cd apps/utils && go mod tidy)
(cd internal/shared && go mod tidy)
(cd apps/relay && go mod tidy)
go work sync
```

**Docker (local dev):**
```bash
docker compose up           # builds + runs api and utils; Postgres must be externally reachable via .env
docker compose down         # tears down containers
```
Requires a root `.env` (see `.env.example`). Postgres is not run by compose — it's expected to be externally provisioned (ADR-016 amendment).

## Continuous Integration

`.github/workflows/go.yml` runs on every **pull request**: a single `build` job that builds and tests modules per-module (the per-module rule above applies — there is no root `go.mod`). The Go version comes from `go-version-file: apps/api/go.mod`. A failing run **blocks merge to `main`** via the required status check named `build` (branch protection, configured server-side via the GitHub API — not in the repo).

All four modules are now built and tested by this job, `apps/relay` included (added 2026-07-27 alongside the `apps/` move — it previously ran in neither workflow).

**⚠️ Remaining relay gap: there is still no `docker/relay/Dockerfile`, and `relay` is absent from `publish-images.yml`'s image matrix.** CI compiles and tests it, but never builds or ships an image for it, so relay cannot be deployed by the existing pipeline. Tracked in `docs/project-notes/issues.md` #130. Note also that `relay` currently has **no test files** — `go test ./...` passes vacuously, so the CI gate proves it compiles, not that it works.

**Gitignored files the build needs are regenerated on the runner, never committed** (deliberate, to keep `go.work`/config out of git):

| File | Why it's needed | CI step that recreates it |
|------|-----------------|---------------------------|
| `go.work` (+ `go.work.sum`) | Only thing wiring the 4 modules together; no module `require`s a sibling, so without it cross-module imports fail to compile (`package ... is not in std`) | `go work init ./apps/api ./internal/shared ./apps/utils ./apps/relay` |
| `apps/utils/configs/config.json` | `apps/utils/main.go` `//go:embed configs/config.json` requires the file at compile time | `cp apps/utils/configs/config.example apps/utils/configs/config.json` |

**When adding a module**, update the `go work init` step too — the workspace `use` list is duplicated in *three* places: the (gitignored) local `go.work`, this CI step, and the `go.work` heredoc in `publish-images.yml`. All three must agree. If the `build` job is renamed, update the required-status-check context or every PR blocks forever.

**When moving a module**, the `use` lists above are only half of it. Also check: both Dockerfiles' `COPY` paths (including the sibling-`go.mod` lines), `.dockerignore`, and any **path-anchored `.gitignore` rule** — a rule like `utils/configs/config.json` silently stops matching after a move, quietly un-ignoring a credentials file. That exact trap fired during the `apps/` move (caught pre-push).

### Image publishing — `.github/workflows/publish-images.yml`

Triggered by pushing a **`v*` tag** (e.g. `v1.0.0`). Builds, tags, and pushes the `api` + `utils` images to ECR (`commerce-api-registry`, `commerce-utils-registry`). Key facts:

- **Image tag is the commit SHA**, not the version tag — the `v*` tag is only the trigger (`${{ github.sha }}` of the tagged commit). ECR repos are `IMMUTABLE`; re-tagging the same commit fails the push.
- Same gitignored-file caveat applies: the job reconstructs `go.work` before `docker build` because both Dockerfiles `COPY go.work`.
- **AWS auth is OIDC under the `aws` GitHub environment.** `AWS_CI_ROLE_ARN` / `AWS_REGION` are **environment-scoped** variables (in the `aws` environment, not repo-level), so the job *must* declare `environment: aws` or `vars.*` resolve to empty.
- **The OIDC token `sub` is therefore `repo:akhakpouri/commerce:environment:aws`** (the `environment:` form, never a branch/tag ref). The `commerce-ci` role's trust policy in `iac-matrix` (`aws/commerce/iam.tf`) must allow exactly that subject. It originally allowed `:ref:refs/heads/main`, which can never match an environment-scoped token — that was the first-run failure (`Not authorized to perform sts:AssumeRoleWithWebIdentity`). (The repo was renamed `commerce-api` → `commerce` on 2026-06-26; the subject changed with it and the trust policy was updated out-of-band — see the TFC gotcha below.)
- **TFC gotcha:** the `iac-matrix` `commerce` workspace runs Terraform remotely as an IAM user (`terraform`) that lacks `iam:UpdateAssumeRolePolicy`, so the trust-policy change could not be applied through the pipeline — it was applied out-of-band with an admin principal. Keep `iam.tf` in sync with live or a future TFC apply reverts it.

## Architecture

### internal/shared

The core library. Three packages:

- **`database`** — a **thin shim** over the external `github.com/akhakpouri/gorm-kit` module (issue #127, ADR-015 amendment). `Migrate(cfg DbConfig)` delegates to `pg.Connect(cfg)` + gorm-kit's driver-agnostic `database.Migrate(db, models...)`; the **model registration list lives in this shim** (`main.go`), not a `setup.go` anymore. `DbConfig` is gorm-kit's type. `api` calls `pg.Connect` directly; DSN construction itself now lives in gorm-kit, not here. `gorm.io/driver/postgres` is no longer a direct dep (pulled via `gorm-kit/pg`); `gorm.io/gorm` stays direct.
- **`models`** — Nine domain models (`User`, `Address`, `Product`, `Category`, `ProductCategory`, `Review`, `Order`, `OrderItem`, `Payment`), all embedding `Base` (auto-increment PK, CreatedAt, UpdatedAt, soft-delete DeletedAt as `time.Time`). All tables live in the `commerce` PostgreSQL schema.
- **`repositories`** — one sub-package per domain. Each defines an interface and a concrete GORM implementation. Constructor takes `*gorm.DB`.

### utils

`main.go` embeds `configs/config.json` via `//go:embed`, passes bytes to `managers.NewDbConfig([]byte)`, then calls `database.Migrate(cfg)`. If JSON parsing fails, falls back to env vars: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `DB_SCHEMA`.

### api

HTTP server using Gin (ADR-004). Entry point is `main.go` — composition root for config, router, and server.

**Structure:**
- `configs/config.go` — `NewConfig()` loads `configs/dev.env` via godotenv; individual DB fields (not a DSN string); `Port` parsed to `int` at startup; `databaseConfig.Connect()` delegates to `pg.Connect()` from `gorm-kit` (ADR-015, amended #127); `CorsNew()` returns Gin CORS middleware.
- `server/server.go` — `Server` struct with `Run()`: starts HTTP server in a goroutine, blocks on `SIGINT`/`SIGTERM`, graceful shutdown with 30s timeout.
- `server/router/routes.go` — `RegisterRoutes(*gin.Engine)`: handler wiring. Will receive a `*container.Container` once the container pattern is implemented.
- `internal/constants/constants.go` — typed structs for env key names and HTTP header names.
- `internal/handlers/` — one sub-package per domain. Each handler struct holds an injected service interface and exposes `RegisterRoutes(*gin.RouterGroup)`.
- `internal/dto/` and `internal/services/` — fully implemented. One sub-package per domain. Services own all business logic and return DTOs; repositories are injected at construction time.
- `docs/` — generated by `swag init -g main.go --output docs`; committed alongside code. Regenerate after changing handler annotations.

Unit tests live alongside each service (`*_test.go`) using gomock-generated mocks (`mock_*_test.go`).

**Known limitation:** `configs/dev.env` uses a relative path — binary must be run from `apps/api/`.

### apps/relay

Standalone daemon (module `commerce/relay`) that drains `commerce.outbox` to the event broker — the "relay" in ADR-018's transactional-outbox backbone. **Producer-only**; downstream consumers (`notifier`, and future `billing`/`shipping`) are separate apps, not part of this module. Full detail: `apps/relay/CLAUDE.md`.

- `main.go` — composition root: connect to Postgres, build via `worker.NewDaemon`, call `Start(ctx)` synchronously (blocks until `signal.NotifyContext` cancels it).
- `worker/daemon.go` — `NewDaemon(ctx, db, awsCfg)` resolves the queue URL (never creates it — Terraform in `matrix` owns queue infra) and wires the publisher/service/manager.
- `internal/managers` — `RelayManagerI.Start(ctx)`: the poll-loop ticker calling `OutboxService.ProcessBatch`.
- `internal/publisher` — `SqsPublisher`, direct-to-SQS today; ADR-018's resolved design is relay→SNS→per-consumer-SQS, so this is expected to become an SNS publisher (see `docs/project-notes/decisions.md`, 2026-07-21 amendment).
- `internal/services/outbox` — `ProcessBatch`: claim (`FOR UPDATE SKIP LOCKED`) → publish → mark, one transaction via the Unit-of-Work manager in `internal/shared/managers/transaction`.

## Database

PostgreSQL 13+, schema `commerce`. Setup SQL (from readme):

```sql
CREATE DATABASE commerce;
CREATE USER commerce WITH ENCRYPTED PASSWORD 'commerce@123';
GRANT ALL PRIVILEGES ON DATABASE commerce TO commerce;
\c commerce
CREATE SCHEMA commerce AUTHORIZATION commerce;
```

- `utils` config: `apps/utils/configs/config.json` (gitignored; use `config.example` as template)
- `api` config: `apps/api/configs/dev.env` — **gitignored** (it holds real credentials). `apps/api/configs/dev.env.example` is the committed template; copy it and fill in locally before running.

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
