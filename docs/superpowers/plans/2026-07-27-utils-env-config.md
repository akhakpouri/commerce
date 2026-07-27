# `utils` env-var config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the compile-time `config.json` embed from `utils`, load its database config from environment variables the same way `api` and `relay` do, and retire `install.sh`.

**Architecture:** `utils` gains an `apps/utils/configs` package modeled on `apps/relay/configs` — optional `configs/dev.env` via godotenv for local dev, then `GetEnvOrPanic` against the process environment. `internal/shared/database` splits its connect-and-migrate helper so callers own the connection. All seven `DB_*` keys become mandatory.

**Tech Stack:** Go 1.26.5, `github.com/lpernett/godotenv`, `github.com/akhakpouri/gorm-kit`, GORM.

**Spec:** `docs/superpowers/specs/2026-07-27-utils-env-config-design.md`

## Global Constraints

- Go version floor is **1.26.5** across all modules — do not bump.
- **All commands run from a module directory.** There is no root `go.mod`. `(cd apps/utils && go test ./...)`, never `go test ./...` from the root.
- **Module paths are flat and do not mirror directories** (ADR-022). `apps/utils` is `module commerce/utils`, so the new package imports as `commerce/utils/configs` — *not* `commerce/apps/utils/configs`.
- **Required env keys are all seven:** `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `DB_SCHEMA`. No defaults for any of them.
- **The non-zero exit on failure is a contract, not a detail** (ADR-015). Both the connect failure and the migration failure must terminate the process with a non-zero status — ECS's pre-deploy migration step and the K8s Job's `backoffLimit` both depend on it.
- Commit messages: match the repo's existing informal imperative style (`updated api deployment`, `added the database`). No `feat:`/`fix:` prefixes — the repo doesn't use them.
- Linters enabled at `.golangci.yml`: `errcheck`, `ineffassign`, `unused`, `govet`, `staticcheck`. `errcheck` in particular will reject an ignored error return.

## Pre-flight

- [ ] **Commit or stash the `apps/` move first.** It is currently uncommitted on `feature/issue-147`. Landing it separately keeps this plan's commits reviewable on their own.
- [ ] Decide whether this rides on `feature/issue-147` or gets its own branch. It supersedes two ADRs and closes issue #105, so its own branch/issue is cleaner — but that is your call, and the plan works either way.

## File Structure

| File | Responsibility | Task |
|------|----------------|------|
| `internal/shared/database/main.go` | Owns the model-registration list; exposes `MigrateWith(db)` | 1, 3 |
| `apps/utils/configs/config.go` | **New.** Builds `Config` from env; panics on any missing key | 2 |
| `apps/utils/configs/config_test.go` | **New.** First test in `apps/utils` — proves the required-key policy | 2 |
| `apps/utils/main.go` | Composition root: config → connect → migrate | 3 |
| `apps/utils/internal/managers/config_manager.go` | **Deleted** | 3 |
| `docker/utils/Dockerfile`, `.github/workflows/go.yml`, `.gitignore` | Build scaffolding for the embed — all removed | 4 |
| `apps/utils/install.sh` | **Deleted** | 5 |
| `readme.md`, `docs/project-notes/*`, `CLAUDE.md` files | Documentation | 4, 5, 6 |

---

### Task 1: Split `MigrateWith` out of the shim

`Migrate` currently does connect-then-migrate in one call. Split it so the caller owns the connection — `utils` will build its own `*gorm.DB` from `configs.DatabaseConfig.Connect()` in Task 3.

`Migrate` is kept **transiently** here, delegating to the new function, so the workspace still compiles at the end of this task. It is deleted in Task 3 once nothing calls it. (The spec says not to keep a wrapper — this is a two-step removal, not a permanent wrapper.)

**Files:**
- Modify: `internal/shared/database/main.go:12-40`

**Interfaces:**
- Produces: `func MigrateWith(db *gorm.DB)` — runs `AutoMigrate` over the registered models; calls `log.Fatal` on failure. Task 3 consumes this.

- [ ] **Step 1: Replace the body of `main.go`**

```go
package database

import (
	"commerce/internal/shared/models"
	"log"

	database "github.com/akhakpouri/gorm-kit/database"
	pg "github.com/akhakpouri/gorm-kit/pg"
	"gorm.io/gorm"
)

// Migrate connects using cfg and migrates.
//
// Deprecated: transitional only — removed in Task 3 once utils builds its own
// connection. Use MigrateWith.
func Migrate(cfg database.DbConfig) {
	db, err := pg.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to the database: ", err)
	}

	log.Println("Connected to the database successfully.")
	MigrateWith(db)
}

// MigrateWith runs AutoMigrate for every registered model against db.
// It terminates the process with a non-zero status on failure — that exit
// code is the migrate-gate contract (ADR-015) relied on by the ECS
// pre-deploy task and the Kubernetes Job's backoffLimit.
func MigrateWith(db *gorm.DB) {
	log.Println("Running migration.")

	if err := database.Migrate(
		db,
		&models.Address{},
		&models.User{},
		&models.Product{},
		&models.Category{},
		&models.ProductCategory{},
		&models.Review{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.Outbox{},
		&models.Event{},
	); err != nil {
		log.Fatal("Migration failed: ", err)
	}

	log.Println("Migration completed successfully.")
}
```

Two deliberate changes beyond the split: the `panic(...)` calls that followed each `log.Fatal` are gone (unreachable — `log.Fatal` calls `os.Exit(1)`), and `fmt` is no longer imported since it was only used by the unreachable panic.

- [ ] **Step 2: Verify the shared module builds and its tests pass**

Run: `(cd internal/shared && go build ./... && go test ./...)`
Expected: builds clean; every package reports `[no test files]` — `internal/shared` has no tests today.

- [ ] **Step 3: Verify the workspace still compiles**

Run: `(cd apps/utils && go build ./...)`
Expected: builds clean. `main.go` still calls `Migrate`, which still exists.

- [ ] **Step 4: Lint**

Run: `(cd internal/shared && golangci-lint run ./...)`
Expected: no findings. If `staticcheck` flags the `Deprecated:` marker as unused, ignore it — Task 3 deletes the function.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/database/main.go
git commit -m "split MigrateWith out of the database shim"
```

---

### Task 2: Add the `apps/utils/configs` package

New package, nothing consumes it yet — so this task is independently verifiable by its own tests. It is also the **first test file in `apps/utils`**.

**Files:**
- Create: `apps/utils/configs/config.go`
- Create: `apps/utils/configs/config_test.go`
- Modify: `apps/utils/go.mod` (adds godotenv)

**Interfaces:**
- Consumes: `cfg.DatabaseConfig` and `cfg.GetEnvOrPanic` from `commerce/internal/shared/configs`; `constants.EnvKeys` from `commerce/internal/shared/constants`.
- Produces: `type Config struct { Database cfg.DatabaseConfig }` and `func NewConfig() *Config`. Task 3 consumes both.

- [ ] **Step 1: Write the failing test**

Create `apps/utils/configs/config_test.go`:

```go
package configs

import "testing"

// setAllDBEnv sets every required key to a valid value. Individual tests
// then blank out one key to prove it is mandatory.
func setAllDBEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "commerce")
	t.Setenv("DB_PASSWORD", "commerce@123")
	t.Setenv("DB_NAME", "commerce")
	t.Setenv("DB_SSLMODE", "disable")
	t.Setenv("DB_SCHEMA", "commerce")
}

func TestNewConfigReadsEveryDatabaseKey(t *testing.T) {
	setAllDBEnv(t)

	c := NewConfig()

	if c.Database.Host != "localhost" {
		t.Errorf("Host = %q, want %q", c.Database.Host, "localhost")
	}
	if c.Database.Port != 5432 {
		t.Errorf("Port = %d, want %d", c.Database.Port, 5432)
	}
	if c.Database.User != "commerce" {
		t.Errorf("User = %q, want %q", c.Database.User, "commerce")
	}
	if c.Database.Password != "commerce@123" {
		t.Errorf("Password = %q, want %q", c.Database.Password, "commerce@123")
	}
	if c.Database.DbName != "commerce" {
		t.Errorf("DbName = %q, want %q", c.Database.DbName, "commerce")
	}
	if c.Database.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want %q", c.Database.SSLMode, "disable")
	}
	if c.Database.Schema != "commerce" {
		t.Errorf("Schema = %q, want %q", c.Database.Schema, "commerce")
	}
}

func TestNewConfigPanicsWhenAnyKeyIsMissing(t *testing.T) {
	keys := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD",
		"DB_NAME", "DB_SSLMODE", "DB_SCHEMA",
	}

	for _, missing := range keys {
		t.Run(missing, func(t *testing.T) {
			setAllDBEnv(t)
			t.Setenv(missing, "") // GetEnvOrPanic treats empty as unset

			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic when %s is unset", missing)
				}
			}()

			NewConfig()
		})
	}
}

func TestNewConfigPanicsOnNonNumericPort(t *testing.T) {
	setAllDBEnv(t)
	t.Setenv("DB_PORT", "not-a-number")

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on non-numeric DB_PORT")
		}
	}()

	NewConfig()
}
```

Two things that make these tests reliable, both worth understanding rather than copying blindly:

1. **`t.Setenv` restores the previous value automatically** at test end, so tests don't leak environment state into each other. It is incompatible with `t.Parallel()` — do not add it.
2. **Go runs a test binary with its working directory set to the package directory**, here `apps/utils/configs/`. So `NewConfig`'s `os.Stat("configs/dev.env")` looks for `apps/utils/configs/configs/dev.env`, which does not exist. The tests are therefore hermetic — a real `apps/utils/configs/dev.env` on your machine will *not* leak into them. (Corollary: if anyone ever creates a nested `apps/utils/configs/configs/` directory, these tests start reading it.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `(cd apps/utils && go test ./configs/ -v)`
Expected: FAIL — `undefined: NewConfig`.

- [ ] **Step 3: Write the implementation**

Create `apps/utils/configs/config.go`:

```go
package configs

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"commerce/internal/shared/constants"

	cfg "commerce/internal/shared/configs"

	"github.com/lpernett/godotenv"
)

type Config struct {
	Database cfg.DatabaseConfig
}

// NewConfig builds the utils configuration from the environment.
//
// configs/dev.env is a local-development convenience only: the os.Stat guard
// means a missing file is not an error. In every container (ECS task, K8s
// Job, docker compose) the file is absent and the values come from injected
// environment variables.
//
// All seven database keys are mandatory. A migration runner that defaults its
// own DB_SCHEMA can migrate the wrong schema and report success, so there is
// no safe default to fall back to.
func NewConfig() *Config {
	if _, err := os.Stat("configs/dev.env"); err == nil {
		if err = godotenv.Load("configs/dev.env"); err != nil {
			slog.Error("Error loading configs/dev.env", "error", err)
		}
	}

	portStr := cfg.GetEnvOrPanic(constants.EnvKeys.DBPort)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("invalid DB_PORT value: %s", portStr))
	}

	return &Config{
		Database: cfg.DatabaseConfig{
			Host:     cfg.GetEnvOrPanic(constants.EnvKeys.DBHost),
			Port:     port,
			User:     cfg.GetEnvOrPanic(constants.EnvKeys.DBUser),
			Password: cfg.GetEnvOrPanic(constants.EnvKeys.DBPassword),
			DbName:   cfg.GetEnvOrPanic(constants.EnvKeys.DBName),
			SSLMode:  cfg.GetEnvOrPanic(constants.EnvKeys.DBSSLMode),
			Schema:   cfg.GetEnvOrPanic(constants.EnvKeys.DBSchema),
		},
	}
}
```

- [ ] **Step 4: Add the godotenv dependency**

Run:
```bash
(cd apps/utils && go get github.com/lpernett/godotenv@v0.0.0-20230527005122-0de1d4c5ef5e && go mod tidy)
```

The pinned pseudo-version is the one `apps/api/go.mod:10` and `apps/relay/go.mod:8` already use — keeping all three modules on one version avoids a needless `go.work.sum` entry.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `(cd apps/utils && go test ./configs/ -v)`
Expected: PASS — `TestNewConfigReadsEveryDatabaseKey`, seven `TestNewConfigPanicsWhenAnyKeyIsMissing/*` subtests, and `TestNewConfigPanicsOnNonNumericPort`.

- [ ] **Step 6: Lint**

Run: `(cd apps/utils && golangci-lint run ./...)`
Expected: no findings.

- [ ] **Step 7: Commit**

```bash
git add apps/utils/configs/config.go apps/utils/configs/config_test.go apps/utils/go.mod apps/utils/go.sum
git commit -m "load utils database config from environment variables"
```

---

### Task 3: Rewire `main.go` and delete the old config path

**Files:**
- Modify: `apps/utils/main.go` (full rewrite)
- Delete: `apps/utils/internal/managers/config_manager.go`
- Modify: `internal/shared/database/main.go` (remove the transitional `Migrate`)

**Interfaces:**
- Consumes: `configs.NewConfig()` (Task 2), `MigrateWith(db)` (Task 1), and `cfg.DatabaseConfig.Connect()` from `internal/shared/configs`.

- [ ] **Step 1: Rewrite `apps/utils/main.go`**

```go
package main

import (
	"log"

	"commerce/internal/shared/database"
	"commerce/utils/configs"
)

func main() {
	log.Println("Welcome to the Commerce Utility Application!")

	cfg := configs.NewConfig()

	db, err := cfg.Database.Connect()
	if err != nil {
		// Non-zero exit is the migrate-gate contract (ADR-015): a database we
		// cannot reach must fail the ECS task and the K8s Job, not pass quietly.
		log.Fatal("Failed to connect to the database: ", err)
	}

	log.Println("Connected to the database successfully.")
	database.MigrateWith(db)
}
```

Note the import path: `commerce/utils/configs`, **not** `commerce/apps/utils/configs`. The directory moved under `apps/`; the module path did not (ADR-022).

`fmt.Println` is replaced with `log.Println` so the banner shares a stream and format with everything else the binary emits.

- [ ] **Step 2: Delete the old config manager**

```bash
git rm apps/utils/internal/managers/config_manager.go
```

This removes BUG-023's zero-value check along with it. The bug becomes structurally unreachable — its entry in `bugs.md` gets annotated in Task 6, not deleted.

- [ ] **Step 3: Remove the transitional `Migrate` from the shim**

In `internal/shared/database/main.go`, delete the entire `Migrate` function added in Task 1, and drop the now-unused `pg` and gorm-kit `database` config imports. The import block becomes:

```go
import (
	"commerce/internal/shared/models"
	"log"

	database "github.com/akhakpouri/gorm-kit/database"
	"gorm.io/gorm"
)
```

`database` is still needed — it provides `database.Migrate(db, models...)`, the gorm-kit function `MigrateWith` calls. Only `pg` becomes unused.

- [ ] **Step 4: Verify everything builds and tests pass**

Run:
```bash
(cd internal/shared && go build ./... && go test ./...)
(cd apps/utils && go build ./... && go test ./...)
(cd apps/api && go build ./...)
(cd apps/relay && go build ./...)
```
Expected: all clean. `apps/api` and `apps/relay` are included because Task 3 changed a shared package's exported surface — if either imported `database.Migrate` this is where you would find out. (Neither does; `apps/utils/main.go:34` was the only caller. This step is the proof, not the assumption.)

- [ ] **Step 5: Confirm no dangling references**

Run: `grep -rn 'go:embed\|config_manager\|NewDbConfig\|database\.Migrate(' apps/ internal/ --include='*.go'`
Expected: exactly one hit — `internal/shared/database/main.go`'s internal call to gorm-kit's `database.Migrate(db, ...)` inside `MigrateWith`.

- [ ] **Step 6: Run it against a real database**

Ensure Postgres is reachable and the seven vars are exported (or create `apps/utils/configs/dev.env`), then:

Run: `(cd apps/utils && go run .)`
Expected: `Connected to the database successfully.` → `Running migration.` → `Migration completed successfully.`

Then prove the new failure mode:

Run: `(cd apps/utils && DB_SCHEMA= go run .)`
Expected: panic naming `DB_SCHEMA`, non-zero exit. **This is the whole point of the change** — confirm the message names the key.

- [ ] **Step 7: Lint**

Run: `(cd apps/utils && golangci-lint run ./...) && (cd internal/shared && golangci-lint run ./...)`
Expected: no findings. `unused` would catch a leftover `pg` import here.

- [ ] **Step 8: Commit**

```bash
git add apps/utils/main.go internal/shared/database/main.go
git add -u apps/utils/internal/
git commit -m "wire utils to env config and drop the embedded config manager"
```

---

### Task 4: Remove the build-time scaffolding

Nothing manufactures a `config.json` anymore, so every step that existed to satisfy the embed comes out.

**Files:**
- Modify: `docker/utils/Dockerfile:21-23`
- Modify: `.github/workflows/go.yml:18-19`
- Modify: `.gitignore` (the `apps/utils/configs/config.json` line)
- Delete: `apps/utils/configs/config.json`, `apps/utils/configs/config.example`
- Modify: `docker/CLAUDE.md` (the "Dummy `config.json` for utils" section)
- Modify: `CLAUDE.md` (CI gitignored-files table)

- [ ] **Step 1: Remove the dummy-config line from the Dockerfile**

In `docker/utils/Dockerfile`, delete these three lines:

```dockerfile
# config.json is gitignored — create a dummy so //go:embed succeeds.
# At runtime, JSON parse fails and NewDbConfig falls back to env vars.
RUN echo '{}' > ./apps/utils/configs/config.json
```

Everything else in the file stays, including all the `COPY` paths.

- [ ] **Step 2: Remove the CI staging step**

In `.github/workflows/go.yml`, delete both lines of this step:

```yaml
        - name: Stage utils embed config
          run: cp apps/utils/configs/config.example apps/utils/configs/config.json
```

- [ ] **Step 3: Delete the config files and the dead gitignore rule**

```bash
git rm apps/utils/configs/config.example   # tracked
rm -f apps/utils/configs/config.json       # gitignored, never tracked
```

Then delete the `apps/utils/configs/config.json` line from `.gitignore`. Leave the bare `dev.env` line alone — it already ignores `apps/utils/configs/dev.env` at any depth, which is what you will create locally.

- [ ] **Step 4: Verify both images build with no config.json anywhere**

Run:
```bash
docker build -f docker/utils/Dockerfile .
docker build -f docker/api/Dockerfile .
```
Expected: both succeed. The utils build is the meaningful one — it now compiles without any `configs/config.json` in the context.

- [ ] **Step 5: Verify CI's build/test sequence locally**

Your local `go.work` already lists the same four modules CI reconstructs, so just run what the `build` job runs:

```bash
(cd internal/shared && go build ./... && go test ./...)
(cd apps/api        && go build ./... && go test ./...)
(cd apps/utils      && go build ./... && go test ./...)
(cd apps/relay      && go build ./... && go test ./...)
```
Expected: all four clean, with no config-staging step preceding them.

- [ ] **Step 6: Update the docs that describe the removed scaffolding**

In `docker/CLAUDE.md`, delete the entire `## Dummy config.json for utils` section, and remove step 5 ("Create build-time workarounds (e.g. dummy `config.json` for utils)") from the `## Dockerfile Layering Strategy` list, renumbering the steps that follow.

In `docker/CLAUDE.md`'s `## Conventions`, the `**utils:**` bullet currently ends with "`//go:embed configs/config.json` requires the file to exist at build time (even as `{}`)" — cut that clause. The rest of the bullet (runs migrations and exits, not a long-running service) stays true.

In `CLAUDE.md`, delete the `apps/utils/configs/config.json` row from the "Gitignored files the build needs" table. The `go.work` row stays.

- [ ] **Step 7: Commit**

```bash
git add docker/utils/Dockerfile .github/workflows/go.yml .gitignore docker/CLAUDE.md CLAUDE.md
git add -u apps/utils/configs/
git commit -m "drop the config.json build scaffolding from docker and CI"
```

---

### Task 5: Retire `install.sh` and rewrite the readme's config sections

**Files:**
- Delete: `apps/utils/install.sh`
- Modify: `readme.md` — line 13, line 45, the `## Utils Configuration` section (110-142), the `## Installing commerce-migrate` section (146-182)

- [ ] **Step 1: Delete the script**

```bash
git rm apps/utils/install.sh
```

- [ ] **Step 2: Replace the `## Utils Configuration` section**

The current section documents the JSON shape and the env-var *fallback*. Env vars are no longer a fallback — they are the mechanism. Replace the whole section (from `## Utils Configuration` down to just before `## Installing commerce-migrate`) with:

```markdown
## Utils Configuration

`utils` reads its database configuration from environment variables. All seven
keys are required — there are no defaults, because a migration runner that
guesses its own schema can migrate the wrong one and still report success.

| Variable      | Purpose           |
|---------------|-------------------|
| `DB_HOST`     | Database host     |
| `DB_PORT`     | Database port     |
| `DB_USER`     | Database user     |
| `DB_PASSWORD` | Database password |
| `DB_NAME`     | Database name     |
| `DB_SSLMODE`  | SSL mode          |
| `DB_SCHEMA`   | Schema name       |

A missing key panics at startup naming that key, and exits non-zero.

For local runs, put them in `apps/utils/configs/dev.env` (gitignored) and run
from the module directory:

```bash
(cd apps/utils && go run .)
```

The file is optional — it is loaded only if present, so containers (docker
compose, ECS, Kubernetes) supply the same keys as injected environment
variables and no file is needed.
```

- [ ] **Step 3: Delete the `## Installing commerce-migrate` section**

Remove it in full, from the `## Installing commerce-migrate` heading through the `> The database config is embedded at compile time...` blockquote and its trailing `---`. Stop before `## Running`.

- [ ] **Step 4: Fix the two remaining references**

- Line 13, the status bullet: delete the `apps/utils/install.sh` line entirely.
- Line 12, the bullet above it, currently says `utils` embeds DB config from `config.json` at compile time — replace with:

```markdown
- ✅ `utils` reads DB config from environment variables (all seven keys required), with an optional local `configs/dev.env`
```

- Line 45, in the workspace tree: delete the `│   │   ├── install.sh             # builds & installs binary with custom config` line, and the `config.json` / `config.example` lines under `utils/configs/`. Replace them with a single entry:

```text
│   │   ├── configs/
│   │   │   └── dev.env            # gitignored — local env vars (optional)
```

- [ ] **Step 5: Verify no dangling references remain**

Run: `grep -rn 'install\.sh\|config\.example\|commerce-migrate\|commerce-tools' readme.md CLAUDE.md docker/ apps/ .github/`
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add readme.md
git add -u apps/utils/
git commit -m "retire the utils install script and document env-var config"
```

---

### Task 6: ADRs and project-notes bookkeeping

**Files:**
- Modify: `docs/project-notes/decisions.md` — ADR-003 and ADR-006 statuses, new ADR-023
- Modify: `docs/project-notes/issues.md` — close #105
- Modify: `docs/project-notes/bugs.md` — annotate BUG-023
- Modify: `docs/project-notes/facts.md` — utils config section
- Modify: `CLAUDE.md` — the `### utils` architecture paragraph
- Modify: `apps/utils/CLAUDE.md` — runtime config contract

- [ ] **Step 1: Mark ADR-003 and ADR-006 superseded**

Do **not** rewrite either ADR's body — they are historical records. Change each `**Status:**` line to `Superseded by ADR-023 (2026-07-27)` and add one sentence directly beneath it explaining what replaced it:

- ADR-003: "The compile-time embed and its env-var *fallback* are both gone; env vars are now the only mechanism, and every key is required. See ADR-023."
- ADR-006: "`install.sh` is deleted. The tool's deployment surface is ECS one-shot tasks and Kubernetes Jobs, both env-injected; local runs use `(cd apps/utils && go run .)`. See ADR-023."

- [ ] **Step 2: Append ADR-023**

Add at the end of `decisions.md`, matching the file's existing ADR structure (Date / Status / body / Rationale / Alternatives considered / Consequences). It must record, at minimum:

- The decision: `utils` loads config from env vars via `internal/shared/configs`, identical to `api` and `relay`; no embed, no config file, no `install.sh`.
- The decisive rationale: **the embed was already dead code in every deployed environment** — ECS injected `DB_*` and the `{}` dummy always fell through to the env path, so production behavior did not change.
- The real defect it fixes: `os.LookupEnv` with no required-check produced an empty DSN (`user= database= port=0`) instead of naming the missing key — BUG-023's symptom reachable through a new door, and under a Job's `backoffLimit` that is several opaque retries.
- Why env vars and not a mounted config file: the file contains the password, so the whole file would have to be a Secret, collapsing the ConfigMap/Secret split that env vars give for free. ADR-006 had already rejected a runtime `--config` flag.
- The accepted regression: no more "credentials baked in, run from anywhere." Local runs must happen from `apps/utils/` for `configs/dev.env` to resolve — the same relative-path constraint `api` has.
- The K8s consequence: the Job uses `envFrom` against the existing `commerce-config` and `commerce-secret`, so `deployment/k8s/commerce/utils/` needs no ConfigMap, Secret, or Service of its own.

- [ ] **Step 3: Close issue #105 and annotate BUG-023**

In `issues.md`, mark #105 done, dated 2026-07-27, noting it was resolved by deleting the embed outright rather than by cleaning up the dummy-file pattern.

In `bugs.md`, add one line to BUG-023: the zero-value check was deleted with `config_manager.go` on 2026-07-27 (ADR-023) and the bug is now structurally unreachable — there is no JSON parse left to mistake an empty document for absent config. Leave the rest of the entry intact.

- [ ] **Step 4: Update `facts.md`**

The utils config block near the top currently reads:

```
Config file: `apps/utils/configs/config.json` (gitignored; `config.example` committed as reference)
Embedded via `//go:embed configs/config.json` in `apps/utils/main.go` and passed as `[]byte` to `managers.NewDbConfig`.
```

Replace with a statement that `utils` requires all seven `DB_*` keys from the environment, loaded by `apps/utils/configs.NewConfig()`, with an optional gitignored `apps/utils/configs/dev.env` for local dev. Mirror the wording already used for the `api` env section further down the same file.

- [ ] **Step 5: Update the two CLAUDE.md files**

In root `CLAUDE.md`, the `### utils` paragraph describes the embed and the JSON-parse fallback. Replace it with the env-var mechanism and the required-key policy.

In `apps/utils/CLAUDE.md`, the `## Runtime config contract` section says the ECS task def injects only the database config and that `../api/configs/dev.env.example` is the canonical key list. That is all still accurate — add one sentence noting the binary now *requires* all seven keys and panics naming any that is missing, so a task definition or ConfigMap missing a key fails fast rather than producing an empty DSN.

- [ ] **Step 6: Verify no stale references survive**

Run: `grep -rn 'config\.json\|go:embed\|NewDbConfig\|install\.sh' --include='*.md' . | grep -v '\.git/' | grep -v 'project-notes/bugs.md' | grep -v 'superpowers/'`
Expected: no output. `bugs.md` and the `superpowers/` specs are excluded deliberately — they are historical records that legitimately still name the old paths.

- [ ] **Step 7: Commit**

```bash
git add docs/project-notes/ CLAUDE.md apps/utils/CLAUDE.md
git commit -m "record ADR-023 and update notes for the utils config change"
```

---

## Final verification

Run the spec's full checklist end to end:

```bash
(cd internal/shared && go build ./... && go test ./...)
(cd apps/utils && go build ./... && go test ./...)
(cd apps/api && go build ./... && go test ./...)
(cd apps/relay && go build ./... && go test ./...)
docker build -f docker/utils/Dockerfile .
docker build -f docker/api/Dockerfile .
grep -rn 'config\.json\|install\.sh\|go:embed' apps/utils docker .github
```

The final `grep` must return nothing.

Then the behavioral check that the whole change exists for:

```bash
(cd apps/utils && go run .)              # migrates cleanly
(cd apps/utils && DB_HOST= go run .)     # panics naming DB_HOST, exits non-zero
```

## Out of scope

- `api` and `relay` config loading — already correct.
- ECS task definitions — they already inject the same seven keys; production behavior is unchanged.
- The K8s Job manifest itself (issue #147). This plan unblocks it; writing it is separate work.
- `GetEnvOrDefaultToInt` in `internal/shared/configs/helper.go` is **buggy** (it converts the key instead of the value, so it always returns the default). `utils` does not call it, so fixing it here would be unrelated scope. File it as its own bug — it currently makes every one of relay's `ConsumerConfig` tuning variables inert.
