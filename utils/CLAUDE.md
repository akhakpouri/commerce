# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in `utils/` — the commerce-api **one-shot migration / backfill** binary.

Sibling to `api/` (the long-running HTTP server — see `api/CLAUDE.md`). This file documents how `utils` binds to the infrastructure managed in the separate **`matrix`** repo, not the Go internals.

## What this binary is

A **one-shot CLI** that runs database migrations and backfills (GORM AutoMigrate — idempotent) and exits. It does **not** serve HTTP, does **not** validate JWTs, and has no health endpoint. In production it runs as a one-shot ECS task, not a long-running service.

It exists so schema changes are a discrete, gated deploy step rather than something the API does at startup.

## Runtime config contract

The ECS task definition in `matrix` (`aws/commerce/`, `ecs-utils.tf`) injects **only the database config** — no `SERVER_ADDRESS`, `AUTH_*`, or `CORS_*` vars. `../api/configs/dev.env.example` is the canonical key list; the DB subset must match.

| Var | Bound to |
|-----|----------|
| `ENV` | plain config from the task def |
| `DB_HOST` / `DB_PORT` | shared RDS, from the `platform-shared` workspace outputs |
| `DB_NAME` / `DB_USER` / `DB_SCHEMA` | `commerce` / `commerce` / `commerce` |
| `DB_SSLMODE` | `require` in prod |
| `DB_PASSWORD` | **secret** — injected from AWS Secrets Manager (`/commerce-api/rds/psql`) at task start; never logged or hardcoded |

> If this binary ever reuses the API's config loader (which also validates `AUTH_*` / `SERVER_ADDRESS`), it will crash before migrating because those vars aren't set on the utils task def. Keep its config surface DB-only, or have the infra add the missing keys.

## Database contract

- Connects to the **logical DB `commerce`** (user `commerce`, schema `commerce` + `public`) on the shared RDS instance — provisioned by `matrix` (ADR-005). Authenticates with `DB_USER`/`DB_PASSWORD`, **not** the RDS master.
- **Owns the schema lifecycle.** This is the only place migrations run. Migrations must be **idempotent** — the task may run on every deploy.

## No HTTP / auth / health

Out of scope here (those belong to `api/`): no routes, no JWT/Auth0, no `/health/*` endpoint, no listener. This binary connects, migrates, logs, and exits with a meaningful status code.

## Image & deploy contract

- Image → ECR repo **`commerce-utils-registry`** (created in `matrix`).
- **sha-only tags** (`:${git-sha}`) — the repo is `IMMUTABLE`; never push `latest`. Built at the same sha as the `api` image in the same CI run.
- Build **`linux/amd64`** — Fargate's platform; an ARM image fails with `exec format error`.
- CI authenticates by assuming the **`commerce-ci`** AWS role via **OIDC** (no stored keys).
- **Runs before the API rolls:** CI does `aws ecs run-task` on the utils task def, waits, and **fails the workflow on a non-zero exit** — migrations gate the API deploy. A non-zero exit must mean "migration failed."

## Where things live (cross-repo map)

| Concern | Repo / path |
|---------|-------------|
| ECR repo, utils task def, IAM, logical DB + secret | `matrix` → `aws/commerce/` |
| Shared VPC + RDS instance | `matrix` → `aws/` (`platform-shared`) |
| The long-running API + its scopes | sibling `api/` in this repo |
| Migration / backfill code, CI workflow YAML | **here** |
