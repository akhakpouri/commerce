[![Go](https://github.com/akhakpouri/commerce/actions/workflows/go.yml/badge.svg)](https://github.com/akhakpouri/commerce/actions/workflows/go.yml)

# Commerce

Go workspace for an e-commerce backend, organized into multiple modules with a shared data layer (GORM + PostgreSQL).

## Current Status

- ✅ Go workspace (`go.work`) with 4 modules — `apps/api`, `apps/utils`, `apps/relay`, and `internal/shared`
- ✅ Shared database package with auto-migrations (thin shim over `github.com/akhakpouri/gorm-kit`, ADR-015)
- ✅ Data models: User, Address, Product, Category, ProductCategory, Review, Order, OrderItem, Payment, Outbox
- ✅ `utils` embeds DB config from `apps/utils/configs/config.json` at compile time, with env var fallback
- ✅ `apps/utils/install.sh` — builds and installs the migration binary with custom config to `$GOPATH/bin/commerce-tools/`
- ✅ Service layer fully implemented with DTOs and unit tests (TaxService, OrderService, UserService, PaymentService, and more)
- ✅ Gin HTTP server with config, CORS, graceful shutdown, container pattern, and full handler layer
- ✅ Handlers implemented: Tax, Product, Category, Address, User, Payment, Order (all endpoints wired and Swagger-annotated)
- ✅ Nested routes: `GET /api/users/:user_id/addresses`, `GET /api/orders/:order_id/payments`
- ✅ Swagger UI wired at `/swagger/index.html` — regenerate docs with `swag init`
- ✅ Auth0-backed JWT validation + per-route scope checks (ADR-017)
- ✅ DB connection consolidated in `internal/shared/database` — used by `api`, `utils`, and `apps/relay` (ADR-015)
- 🚧 Event-driven backbone in progress (ADR-018): transactional outbox + `apps/relay` daemon drains it to SQS today; resolved design is SNS fan-out, not yet wired. Now built and tested in CI, but still has **no Dockerfile and no image in the publish pipeline** — see `docs/project-notes/issues.md` #130.

## Workspace Structure

```text
commerce/
├── go.work
├── go.work.sum
├── .golangci.yml
├── readme.md
├── apps/                          # every deployable module (ADR-022)
│   ├── api/                       # API executable — module commerce/api
│   │   ├── go.mod
│   │   ├── main.go
│   │   ├── configs/               # env-based config + CORS setup
│   │   ├── server/                # Server struct + graceful shutdown
│   │   │   └── router/            # RegisterRoutes() — handler wiring
│   │   └── internal/
│   │       ├── handlers/          # HTTP handlers, one sub-package per domain
│   │       ├── dto/               # request/response shapes, one sub-package per domain
│   │       ├── services/          # business logic, one sub-package per domain
│   │       └── constants/         # typed env key + header name constants
│   ├── utils/                     # Migration executable — module commerce/utils
│   │   ├── go.mod
│   │   ├── install.sh             # builds & installs binary with custom config
│   │   ├── configs/
│   │   │   ├── config.json        # gitignored — local credentials
│   │   │   └── config.example     # committed reference template
│   │   ├── internal/
│   │   │   └── managers/
│   │   │       └── config_manager.go
│   │   └── main.go
│   └── relay/                     # Outbox-draining daemon (ADR-018) — module commerce/relay
│       ├── go.mod
│       ├── main.go
│       ├── configs/
│       ├── worker/
│       │   └── daemon.go          # NewDaemon() wiring + graceful shutdown
│       └── internal/
│           ├── managers/          # RelayManagerI (poll loop)
│           ├── publisher/         # SqsPublisher (SNS swap pending)
│           ├── services/outbox/
│           └── dto/outbox/
├── docker/                        # one Dockerfile per service (ADR-016)
├── deployment/k8s/                # minikube manifests (ADR-021)
├── internal/
│   └── shared/                    # Shared library used by every app — module commerce/internal/shared
│       ├── go.mod
│       ├── database/
│       │   └── main.go            # thin shim over gorm-kit; model registration list lives here
│       └── models/
│           ├── address.go
│           ├── base.go
│           ├── category.go
│           ├── event.go
│           ├── order.go
│           ├── order_item.go
│           ├── outbox.go
│           ├── payment.go
│           ├── product.go
│           ├── product_category.go
│           ├── review.go
│           └── user.go
```

## Go / Dependencies

- Go: `1.26`
- ORM: `gorm.io/gorm v1.31.1`
- DB Driver: `gorm.io/driver/postgres v1.6.0`

## Prerequisites

- Go 1.26+
- PostgreSQL 13+
- `golangci-lint` (optional but recommended)

## Database Setup

```sql
CREATE DATABASE commerce;
CREATE USER commerce WITH ENCRYPTED PASSWORD 'commerce@123';
GRANT ALL PRIVILEGES ON DATABASE commerce TO commerce;

\c commerce
CREATE SCHEMA commerce;
GRANT ALL ON SCHEMA commerce TO commerce;
```

## Utils Configuration

`utils` embeds `configs/config.json` into the binary at compile time via `//go:embed`. If the file is missing or fails to parse, it falls back to environment variables and continues without error.

Copy the example to get started locally:

```bash
cp apps/utils/configs/config.example apps/utils/configs/config.json
```

Expected JSON shape:

```json
{
    "host": "localhost",
    "port": 5432,
    "user": "commerce",
    "password": "commerce@123",
    "dbname": "commerce",
    "sslmode": "disable",
    "schema": "commerce"
}
```

### Environment Variable Fallback

| Variable      | Purpose           |
|---------------|-------------------|
| `DB_HOST`     | Database host     |
| `DB_PORT`     | Database port     |
| `DB_USER`     | Database user     |
| `DB_PASSWORD` | Database password |
| `DB_NAME`     | Database name     |
| `DB_SSLMODE`  | SSL mode          |
| `DB_SCHEMA`   | Schema name       |

## Installing `commerce-migrate`

`apps/utils/install.sh` builds the migration binary with your local `config.json` baked in and installs it to `$GOPATH/bin/commerce-tools/`.

**Prerequisites:** `$GOPATH` must be set.

**Steps:**

1. Copy and edit the config template with your target database credentials:

```bash
cp apps/utils/configs/config.example apps/utils/configs/config.json
vim apps/utils/configs/config.json
```

2. Run the install script from the `apps/utils/` directory:

```bash
(cd apps/utils && bash install.sh)
```

This will:
- Create `$GOPATH/bin/commerce-tools/` if it doesn't exist
- Copy `configs/` alongside the binary (for reference)
- Build the binary with `config.json` embedded at compile time
- Install it to `$GOPATH/bin/commerce-tools/utils`
- Execute the binary immediately to run migrations

**To run migrations again after install:**

```bash
$GOPATH/bin/commerce-tools/utils
```

> The database config is embedded at compile time. To target a different database, edit `config.json` and re-run `install.sh`.

---

## Running

Run each executable from its own module directory:

```bash
# Run API executable
(cd apps/api && go run .)

# Run utils executable
(cd apps/utils && go run .)

# Run the relay daemon
(cd apps/relay && go run .)
```

Current behavior:

- `api`: starts Gin HTTP server on `SERVER_ADDRESS`; all handler groups active
- `utils`: loads DB config, then runs GORM auto-migrations
- `apps/relay`: polls `commerce.outbox` for unpublished events and drains them to SQS (SNS swap pending, see Current Status)

## Build

From each module:

```bash
(cd apps/api && go build -o ../../bin/api .)
(cd apps/utils && go build -o ../../bin/utils .)
(cd apps/relay && go build -o ../../bin/relay .)
```

## Linting / Vet / Tests

Run per module:

```bash
(cd apps/api && go test ./...)
(cd apps/utils && go test ./...)
(cd internal/shared && go test ./...)
(cd apps/relay && go test ./...)

(cd apps/api && go vet ./...)
(cd apps/utils && go vet ./...)
(cd internal/shared && go vet ./...)
(cd apps/relay && go vet ./...)

(cd apps/api && golangci-lint run ./...)
(cd apps/utils && golangci-lint run ./...)
(cd internal/shared && golangci-lint run ./...)
(cd apps/relay && golangci-lint run ./...)
```

> Note: `apps/relay` has **no test files**, so its `go test ./...` passes vacuously — CI proves it compiles, not that it works.

## Module Maintenance

Run tidy inside each module:

```bash
(cd apps/api && go mod tidy)
(cd apps/utils && go mod tidy)
(cd internal/shared && go mod tidy)
(cd apps/relay && go mod tidy)
go work sync
```

## Notes

- Modules live under `apps/`, but their **module paths stay flat** — `apps/api` is `module commerce/api`, not `commerce/apps/api`. Imports are unaffected by the directory layout.
- `apps/utils/configs/config.json` is gitignored. Use `config.example` as a template.
- `apps/api/configs/dev.env` is gitignored (it holds real credentials); `apps/api/configs/dev.env.example` is the committed template — copy and fill it in locally before running.
- `api` binary must be run from the `apps/api/` directory (`configs/dev.env` uses a relative path).
- `DeletedDate` on all models uses `time.Time`, not `gorm.DeletedAt` — soft-deleted records are not auto-filtered by GORM.
- Swagger docs in `apps/api/docs/` — regenerate with `(cd apps/api && swag init -g main.go --output docs)` after changing handler annotations.
- `apps/relay` never creates its own queue/topic infrastructure — it only resolves an existing one and fails fast if it's missing. Provisioning lives in Terraform (the `matrix`/`iac-matrix` repos), not here.