# `utils` config — drop the compile-time embed, converge on env vars

**Date:** 2026-07-27
**Status:** Approved by user, ready for implementation planning
**Supersedes:** ADR-003 (embedded config with env var fallback), ADR-006 (shell script install with compile-time config embedding)
**Closes:** issue #105 (dummy `config.json` cleanup)
**Related:** ADR-015 (migrate-gate contract), ADR-016 (Docker structure), ADR-021 (K8s orchestration), ADR-022 (`apps/` layout), BUG-023

## Problem

`apps/utils/main.go` embeds `configs/config.json` at compile time via `//go:embed`. Because the real file is gitignored, every build environment has to manufacture one:

- `docker/utils/Dockerfile` — `RUN echo '{}' > ./apps/utils/configs/config.json`
- `.github/workflows/go.yml` — `cp apps/utils/configs/config.example apps/utils/configs/config.json`

At runtime `{}` parses cleanly into a zero-value struct, which is why `NewDbConfig` needs an explicit zero-value check to reach its env-var fallback at all (BUG-023 — the original code treated "unmarshal failed" as a proxy for "no config", and `{}` never fails).

Two things make this worth removing rather than patching:

**The embed is already dead code in every deployed environment.** The ECS task definition injects `DB_*` and pulls `DB_PASSWORD` from Secrets Manager, so what actually runs in production is `{}` → parse succeeds → zero-value check → `getConfigFromEnv()`. Only `install.sh` still exercises the embed for real. The K8s Job (issue #147) will be the third env-injected environment.

**The fallback has no validation.** `getConfigFromEnv()` uses `os.LookupEnv` with no required-key check, and `getPortFromEnv()` silently defaults to 5432. A missing or misspelled key yields an empty field and a DSN error like ``failed to connect to `user= database= port=0` `` — the exact BUG-023 symptom, reachable through a new door. Under a Job's `backoffLimit` that becomes several opaque retries. Meanwhile `api` and `relay` both use `GetEnvOrPanic`, which names the missing key and exits immediately.

## Decision

Delete `utils`' bespoke config manager and load configuration the same way `api` and `relay` already do: optional `configs/dev.env` via godotenv for local dev, `GetEnvOrPanic` against the process environment everywhere else. `install.sh` is retired rather than adapted.

This reduces the repo from two config mechanisms across three apps to one.

## Target shape

**New — `apps/utils/configs/config.go`**, modeled directly on `apps/relay/configs/config.go`:

```go
package configs

type Config struct {
    Database cfg.DatabaseConfig
}

func NewConfig() *Config {
    if _, err := os.Stat("configs/dev.env"); err == nil {
        if err = godotenv.Load("configs/dev.env"); err != nil {
            slog.Error("Error loading configs/dev.env", "error", err)
        }
    }

    port, err := strconv.Atoi(cfg.GetEnvOrPanic(constants.EnvKeys.DBPort))
    // ... remaining six keys via cfg.GetEnvOrPanic(constants.EnvKeys.*)
}
```

The `os.Stat` guard is load-bearing: in a container the file is absent and `NewConfig` must fall through to the injected environment rather than erroring.

**`apps/utils/main.go`** becomes a composition root — build config, migrate, done. The `embed` import and `migrateDatabase(filePath string)` indirection both go away.

**New dependency:** `github.com/lpernett/godotenv` becomes a direct requirement of `apps/utils/go.mod` (today it only requires `gorm-kit`). Pin the same pseudo-version `api` and `relay` already use — `v0.0.0-20230527005122-0de1d4c5ef5e` — so all three modules stay aligned. `commerce/internal/shared/configs` and `.../constants` need no `require`; they resolve through `go.work`, as `.../database` already does.

### The one wiring decision

`database.Migrate()` takes gorm-kit's `database.DbConfig`, while shared `configs.DatabaseConfig` is a parallel struct with an identical seven fields — `DatabaseConfig.Connect()` already copies between them by hand.

**Chosen: replace `Migrate(cfg DbConfig)` with `MigrateWith(db *gorm.DB)`** in `internal/shared/database`, and have `utils` call `cfg.Database.Connect()` then `MigrateWith(db)`. This removes a duplicated struct copy instead of adding a third one, and `Connect()` already hands back the `*gorm.DB` that gorm-kit's `database.Migrate(db, models...)` wants.

This is a clean signature swap, not a compatibility problem: **`apps/utils/main.go:34` is the only caller of `database.Migrate` in the repo** (verified). Keeping the old signature as a wrapper would leave dead code behind in a change whose whole point is deleting vestigial paths — so it goes.

The connect-failure branch moves with it. `Migrate` currently does `pg.Connect` + `log.Fatal` on failure; after the split, `utils` owns handling the `Connect()` error and must still exit non-zero. `MigrateWith` keeps `log.Fatal` on migration failure. That non-zero exit **is** ADR-015's migrate-gate contract, and both the ECS pre-deploy task and the K8s Job depend on it — losing it on either branch silently breaks the gate.

### Required-key policy

**All seven of `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `DB_SCHEMA` are required.** No defaults.

This matches `api`, but the stronger argument is specific to this binary: a migration runner that silently defaults `DB_SCHEMA` can migrate into the *wrong schema* and report success. For a tool whose entire job is mutating schema, guessing at the target is the worst available failure mode. Failing loudly on a missing key is strictly better.

`GetEnvOrPanic` panicking is the correct behavior here — panic → non-zero exit → Job fails → `backoffLimit` engages, which is exactly ADR-015's contract.

**Behavior change to expect:** `docker compose up` with a partial root `.env` now panics instead of limping to a confusing DSN error. The committed `.env.example` already carries all seven keys, so a correctly-seeded local setup is unaffected.

## Deletions

| Path | Note |
|------|------|
| `apps/utils/internal/managers/config_manager.go` | entire file; `[no test files]`, nothing to port |
| `apps/utils/configs/config.json` | gitignored local file |
| `apps/utils/configs/config.example` | committed template, no longer meaningful |
| `apps/utils/install.sh` | see below |
| `//go:embed` + `embed` import in `main.go` | |
| `RUN echo '{}' > ./apps/utils/configs/config.json` in `docker/utils/Dockerfile` | |
| `Stage utils embed config` step in `.github/workflows/go.yml` | |
| `.gitignore` line for `apps/utils/configs/config.json` | now dead — the bare `dev.env` pattern already ignores `apps/utils/configs/dev.env` at any depth |

BUG-023's zero-value check disappears with `config_manager.go`. The bug becomes structurally unreachable; the entry in `bugs.md` stays as a historical record, annotated rather than deleted.

## Retiring `install.sh`

ADR-006's value proposition was "credentials baked in at compile time, run the binary from anywhere." Without the embed that property is gone, and adapting it would mean reinventing a worse `.env` loader.

Retiring it is acceptable because the tool's real deployment surface is now ECS one-shot tasks and K8s Jobs, both env-injected; local runs are served by `(cd apps/utils && go run .)` with a `configs/dev.env` alongside. Baking credentials into a binary was always a liability — ADR-006 itself lists "rebuild required to target a different database" as its tradeoff, and rotation had the same problem.

**Doc blast radius (verified):** `readme.md` line 13 (status bullet), line 45 (tree entry), and the whole `## Installing commerce-migrate` section, lines 146–182. Nothing in root `CLAUDE.md` or `apps/utils/CLAUDE.md` references it.

## K8s implications (issue #147)

The Job consumes `envFrom` against the **existing** `commerce-config` ConfigMap and `commerce-secret` Secret in the `commerce` namespace — the same DB connection the API already uses. One source of truth for the credentials, no second Secret to rotate in lockstep.

Consequence: `commerce-config` also carries `SERVER_ADDRESS`, `CORS_ALLOWED_ORIGIN`, and `AUTH_*`. Those get injected into the utils container and ignored. The Job's environment is therefore wider than its config surface — accepted deliberately in exchange for not duplicating credentials.

This makes three files under `deployment/k8s/commerce/utils/` unnecessary: `configmap.yaml` and `secret.yaml` (currently empty) and `service.yaml` (meaningless for a Job — nothing fronts a Job with a Service).

## Non-goals

- No change to `api` or `relay` config loading — they already do this.
- No change to the ECS task definitions. They inject the same seven keys today; production behavior is unchanged by this work.
- Not introducing a `--config` flag or a mounted config file. ADR-006 already rejected the flag; a mounted file would force the whole file into a Secret (it contains the password), collapsing the ConfigMap/Secret split that env vars give for free.
- Not touching `internal/shared/configs` itself beyond consuming it.

## Verification

1. `(cd apps/utils && go build ./... && go test ./...)`
2. `(cd internal/shared && go build ./... && go test ./...)` — `MigrateWith` is a shim change
3. `docker build -f docker/utils/Dockerfile .` — must succeed with **no** `config.json` anywhere
4. `docker build -f docker/api/Dockerfile .` — sibling `go.mod` copy unaffected, but confirm
5. `docker compose up` with a full root `.env` — migrations run; then with a key removed, confirm it panics naming that key
6. `grep -rn 'config\.json\|install\.sh\|go:embed' apps/utils docker .github` returns nothing

## Consequences

- One config mechanism across all three apps. A reader who understands `api`'s config understands `utils`'.
- Misconfiguration fails fast and names the key, instead of producing an empty-DSN error several layers downstream.
- CI and Docker both lose a synthetic file-manufacturing step; the Dockerfile's "dummy config" section in `docker/CLAUDE.md` goes with them.
- Local `utils` runs gain the same relative-path constraint `api` has: must run from `apps/utils/` for `configs/dev.env` to resolve. This is the accepted cost of retiring `install.sh`.
- ADR-003 and ADR-006 both move to Superseded; both are already `Closed`, so this is a status change plus a pointer to the superseding ADR, not a rewrite of their bodies.
