# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in `api/` — the commerce-api **HTTP server** binary.

Sibling to `utils/` (the one-shot migration tool — see `utils/CLAUDE.md`). This file documents how the API binds to the infrastructure and Auth0 tenant managed in the separate **`matrix`** repo, not the Go internals.

## What this binary is

Long-running REST API for the commerce product. JWT-protected via Auth0, talks to PostgreSQL, listens on container port `8080`. Deployed as a long-running ECS Fargate service behind an ALB.

## Runtime config contract

The ECS task definition in `matrix` (`aws/commerce/`, `ecs-api.tf`) injects configuration as environment variables. The app must read **exactly these names**; `configs/dev.env.example` is the canonical list and must stay in lock-step with the task def — adding or removing a key is a change in *both* repos.

| Var | Bound to |
|-----|----------|
| `SERVER_ADDRESS` | `:8080` — must match the container port the ALB/target group expect |
| `ENV` / `CORS_ALLOWED_ORIGIN` | plain config from the task def |
| `DB_HOST` / `DB_PORT` | shared RDS, from the `platform-shared` workspace outputs |
| `DB_NAME` / `DB_USER` / `DB_SCHEMA` | `commerce` / `commerce` / `commerce` |
| `DB_SSLMODE` | `require` in prod |
| `DB_PASSWORD` | **secret** — injected from AWS Secrets Manager (`/commerce-api/rds/psql`) at task start; never logged or hardcoded |
| `AUTH_DOMAIN` / `AUTH_AUDIENCE` | Auth0 tenant + `urn:commerce-api` |

`DB_PASSWORD` is the only secret; everything else is plain config.

## Health-check contract

The ALB target group health-checks **`GET /health/status/live`** on port `8080`, expecting `200`. Keep it cheap and dependency-free (don't gate it on the DB) or the service gets pulled out of rotation. The container **must** listen on `8080`.

## Database contract

- Connects to the **logical DB `commerce`** (user `commerce`, schema `commerce` + `public`) on the shared RDS instance — all provisioned by `matrix` (ADR-005). Authenticates with `DB_USER`/`DB_PASSWORD`, **not** the RDS master.
- **The API must not run migrations at startup.** Schema changes are applied by the `utils` binary as a separate deploy step (see `utils/CLAUDE.md`); the API assumes the schema already exists.

## Auth0 scope coupling

Per-route authorization enforces scopes defined in `internal/auth/scope.go` — the **source of truth on the consumer side**. The matching audience (`urn:commerce-api`) + scope vocabulary is owned by `matrix` (`auth0/`). **Renaming or adding a scope is a two-repo change** (`scope.go` here *and* `auth0/`); if they drift, tokens won't carry the scopes routes expect and requests 403.

## Image & deploy contract

- Image → ECR repo **`commerce-api-registry`** (created in `matrix`).
- **sha-only tags** (`:${git-sha}`) — the repo is `IMMUTABLE`; never push `latest`.
- Build **`linux/amd64`** — Fargate's platform; an ARM image fails with `exec format error`.
- CI authenticates by assuming the **`commerce-ci`** AWS role via **OIDC** (no stored keys); the role ARN is a `matrix` output, stored here as `vars.AWS_CI_ROLE_ARN`.
- **The API service rolls only after `utils` has migrated** (CI runs the utils task first, fails the build on non-zero, then `aws ecs update-service`). The infra ignores TF-vs-CI drift on the service, so CI owns the live revision.

## Where things live (cross-repo map)

| Concern | Repo / path |
|---------|-------------|
| ECR repo, ECS service/task-def, ALB, IAM, logical DB + secret | `matrix` → `aws/commerce/` |
| Shared VPC + RDS instance | `matrix` → `aws/` (`platform-shared`) |
| Auth0 audience + scopes + clients | `matrix` → `auth0/` |
| Migrations / backfills | sibling `utils/` in this repo |
| API application code, CI workflow YAML | **here** |
