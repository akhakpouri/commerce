# Architectural Decision Records

## ADR-001 — Go workspace with three modules

**Date:** (pre-existing)
**Status:** Closed — verified 2026-02-25

Split into `api`, `utils`, and `internal/shared` modules under a single `go.work` workspace. All external dependencies (GORM, PostgreSQL driver, bcrypt) are pinned to `internal/shared` only. `api` and `utils` consume `internal/shared` as a local dependency.

---

## ADR-002 — GORM + PostgreSQL for persistence

**Date:** (pre-existing)
**Status:** Closed — verified 2026-02-25

GORM is the ORM. All models embed a `Base` struct (`internal/shared/models/base.go`) providing:
- `Id uint` — auto-increment primary key (not UUID)
- `CreatedDate time.Time` — auto-set on create
- `UpdatedDate time.Time` — auto-set on update
- `DeletedDate time.Time` — indexed, but typed as `time.Time` not `gorm.DeletedAt`

> **Note:** `DeletedDate` uses `time.Time`, not `gorm.DeletedAt`. GORM's automatic soft-delete filtering requires `gorm.DeletedAt`. Current implementation does NOT auto-filter deleted records unless queries are written manually.

All tables live in the `commerce` PostgreSQL schema.

---

## ADR-003 — Embedded config with env var fallback

**Date:** (pre-existing)
**Status:** Closed — verified 2026-02-25

`utils/main.go` embeds `configs/config.json` at compile time via `//go:embed` and passes the raw bytes to `managers.NewDbConfig([]byte)`. If JSON parsing fails, `NewDbConfig` falls back to environment variables (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `DB_SCHEMA`) and returns `nil` error.

Config file: `utils/configs/config.json` — gitignored (contains credentials). `utils/configs/config.example` is committed as a reference.

> **Note:** Embed responsibility lives in `utils/main.go`, not `config_manager.go`. `NewDbConfig` accepts `[]byte` and has no knowledge of the filesystem.

---

## ADR-004 — Gin as the HTTP framework

**Date:** 2026-03-26
**Status:** Active — skeleton implemented 2026-03-26

Gin (`github.com/gin-gonic/gin`) chosen as the HTTP framework for the `api` module. Rationale: project already uses third-party dependencies (GORM, testify), and Gin's request binding, middleware chain, and structured error responses reduce handler boilerplate. Stdlib `net/http` was considered but Gin was preferred for development speed given the full service layer already in place. Gin is added to `api/go.mod` only — `internal/shared` stays dependency-free from HTTP concerns.

### Server structure (implemented)

```
api/
├── main.go                      # composition root: config → router → server
├── configs/
│   ├── config.go                # NewConfig(), GetEnvOrPanic(), CorsNew()
│   ├── dev.env                  # gitignored — local env vars (dev.env.example committed)
│   └── dev.env.example          # committed reference template
├── server/
│   ├── server.go                # Server struct, Run() with graceful shutdown
│   └── router/
│       └── routes.go            # RegisterRoutes() — all handler wiring lives here
└── internal/
    ├── constants/constants.go   # typed env key + header name constants
    └── handlers/
        └── tax/tax_handler.go   # first handler group (no repo dependency)
```

**`main.go` flow:** `NewConfig()` → `gin.Default()` → attach CORS middleware → `RegisterRoutes(router)` → `NewServer(...).Run()`

**`Server.Run()`:** starts `http.Server` in a goroutine, blocks on `SIGINT`/`SIGTERM`, then calls `srv.Shutdown(ctx)` with a 30-second timeout. No `<-ctx.Done()` after `Shutdown` — `Shutdown` already blocks until drain or timeout.

**Config:** `NewConfig()` attempts `godotenv.Load("configs/dev.env")` only if the file exists (checked via `os.Stat`). If absent, env vars are read directly from the environment via `GetEnvOrPanic` — missing vars panic at startup. `dev.env` is gitignored; `dev.env.example` is committed as a reference. See facts.md for the full env key list.

**CORS:** configured via `github.com/gin-contrib/cors`. `AllowOriginFunc` does exact-match on `CORS_ALLOWED_ORIGIN`. Methods: GET, POST, PUT, DELETE.

**Handler pattern:** each handler struct holds an injected service interface. `RegisterRoutes(rg *gin.RouterGroup)` wires the routes. `routes.go` is the only place that constructs services and handlers — it is the composition root for the HTTP layer.

> **Note:** `configs/dev.env` is loaded with a relative path — binary must be run from `api/` when using a local env file.

---

## ADR-015 — Consolidated DB connection in `internal/shared/database`

**Date:** 2026-03-30
**Status:** Active — implemented 2026-03-30; **amended 2026-06-25 (issue #127)** — DSN construction + `Connect` delegated to the external `gorm-kit` module.

DB connection logic (DSN construction + `gorm.Open`) lives exclusively in `internal/shared/database`. Both `api` and `utils` use it — no duplication.

**`internal/shared/database/main.go`:**
- `Connect(cfg DbConfig) (*gorm.DB, error)` — builds DSN, opens and returns `*gorm.DB`
- `Migrate(cfg DbConfig)` — calls `Connect` internally, then runs `AutoMigrate`

**Amendment (2026-06-25, issue #127):** the DSN-construction and `gorm.Open` logic was extracted into a standalone reusable module — **`github.com/akhakpouri/gorm-kit`** (`pg.Connect` + driver-agnostic `database.Migrate(db, models...)`; same `DbConfig` shape) — to stop re-writing it in every new Go+GORM project. `internal/shared/database` is now a **thin shim**:
- `database/main.go` keeps `Migrate(cfg)` but delegates: `pg.Connect(cfg)` → `gorm-kit database.Migrate(db, <models>)`. The **model registration list now lives in this shim** (the old `setup.go`/`db_config.go` were removed; `DbConfig` is gorm-kit's type, re-exported via import).
- `api/configs/config.go` calls `pg.Connect` directly with gorm-kit's `database.DbConfig`.
- The "consolidated, no duplication" principle of ADR-015 is **unchanged** — connection logic still has a single source of truth; that source just moved out-of-tree to gorm-kit. `gorm.io/driver/postgres` is no longer a direct dep (pulled via `gorm-kit/pg`); `gorm.io/gorm` stays direct (models/repos use `*gorm.DB`).
- Migrate-gate contract preserved: the shim's `log.Fatal` on migration failure → `os.Exit(1)`, so CI still fails the deploy on a bad migration.

See `gorm-kit` repo at `~/code/go/gorm-kit` (single module, Postgres-first, `pg/` + `database/` + `mysql/` packages; MIT).

**Config loading stays per-module** (each module owns how it reads config):
- `utils` — reads `configs/config.json` (embedded) → `database.DbConfig` → `database.Migrate(cfg)`
- `api` — reads `configs/dev.env` (godotenv) → `databaseConfig` → `databaseConfig.Connect()` which converts to `database.DbConfig` and delegates to `database.Connect(cfg)`

**`api/configs/databaseConfig`** remains its own independent struct (no JSON tags, loaded from env vars). `Port` is `int` — parsed from `DB_PORT` at startup via `strconv.Atoi`; invalid value panics immediately. `Connect()` is the only method — `ConnectionString()` was removed.

**Why not reuse `database.DbConfig` directly in `api`:** config loading strategies differ per module (JSON vs `.env`). Keeping `databaseConfig` separate avoids coupling the `api` config layer to the shared library's struct tags and field conventions.

---

## ADR-005 — bcrypt password hashing via GORM hooks

**Date:** (pre-existing)
**Status:** Active

`User` model uses `BeforeCreate` and `BeforeUpdate` GORM hooks to automatically hash the `Password` field with bcrypt. A `CheckPassword()` method is provided for verification. Hashing is transparent to callers.

**Clarification (2026-02-26):** Moving this logic to the service layer (per ADR-008) was considered and rejected. Password hashing is a persistence invariant — it must hold regardless of which service or consumer writes a `User`. Keeping it in the model hook makes it unconditional and impossible to accidentally bypass.

---

## ADR-006 — Shell script installation with compile-time config embedding

**Date:** 2026-02-26
**Status:** Closed

`utils/install.sh` is the chosen installation mechanism for the migration binary. It builds the binary with `configs/config.json` embedded at compile time and installs it to `$GOPATH/bin/commerce-tools/` alongside a copy of the `configs/` directory.

**Workflow:** edit `config.json` → run `install.sh` → binary is built with credentials baked in → migrations run immediately.

**Why this over alternatives:**
- `go install` alone can't embed a local config file at the user's `$GOBIN` without a build step
- Runtime `--config` flag was considered but rejected as unnecessary complexity for this use case
- Custom install dir (`commerce-tools/`) keeps the binary isolated from other Go tools in `$GOBIN`

**Tradeoff:** targeting a different database requires editing `config.json` and re-running `install.sh` (rebuild required). This is acceptable given the tool's purpose as a one-time migration runner, not a frequently reconfigured service.

**Fix (2026-02-26):** `cp configs` corrected to `cp -r configs` — directory copy was silently failing without the `-r` flag.

---

## ADR-007 — Payment model as a separate table with audit trail

**Date:** 2026-02-26
**Status:** Active — implemented and migrated 2026-02-26

Rather than extending `Order` with more payment fields, `Payment` is its own table with a many-to-one relationship to `Order`. This preserves the full history of payment attempts (retries, refunds) rather than overwriting a single status.

**Fields:** `OrderId` (FK), `Amount`, `Currency`, `Status`, `Gateway`, `GatewayTransactionId`, `GatewayResponse`, `PaymentMethod`, `PaidAt` (nullable).

**Status enum:** `pending | authorized | captured | failed | refunded | partially_refunded`

**Key decisions:**
- `Order.PaymentStatus` field was removed during implementation — replaced with `Payments []Payment` association. Payment state is read by querying the `payments` table directly.
- No separate `PaymentMethod` model for MVP — gateway tokens (e.g., Stripe `pm_...`) stored as a string field on `Payment`.
- Refunds handled via status + `RefundedAmount` on the existing `Payment` row (not separate rows) for MVP simplicity.
- Actual card data never stored — delegated entirely to the payment gateway (PCI compliance).
- `Payment` is NOT tied directly to `User` — user is reachable via `Payment → Order → UserId`. Adding a direct `UserId` FK would be redundant denormalization.
- Payments are cascade-deleted when their parent `Order` is deleted. This simplifies the data model at the cost of full audit trail preservation — acceptable for MVP. Revisit if financial audit requirements tighten post-MVP.
- `Payment.Order` uses `OnDelete:CASCADE` to match `Order.Payments` — both sides must agree to avoid constraint conflicts.

**Follow-up (post-MVP):** Introduce a `PaymentMethod` model to support saved payment methods per user:
- `PaymentMethod` belongs to `User` (stores gateway token, card brand, last 4, expiry)
- `Payment.PaymentMethodId` — optional FK to `PaymentMethod` (nullable for one-off payments)
- On user delete → CASCADE delete `PaymentMethod`; SET NULL on `Payment.PaymentMethodId`
- This is the correct solution for "reuse a saved card on a new order" without adding `UserId` to `Payment`

---
## ADR-008 — Thin DTOs with service-layer mapping and business logic

**Date:** 2026-02-26
**Status:** Done

API payloads are represented as DTOs (request/response structs) living in `api/internal/dto/`. DTOs are plain data containers — json tags, validation tags, and mapping methods only. Business logic lives exclusively in `api/internal/services/`.

**Layer responsibilities:**

| Concern | Layer |
|---|---|
| JSON shape / validation tags | DTO (`api/internal/dto/`) |
| Mapping DTO ↔ model | DTO methods (`ToModel()` / `FromModel()`) |
| Business rules (e.g. order must exist, not already paid) | Service |
| Password hashing, GORM hook behaviour | Model |
| DB persistence | Repository (via GORM) — services never import or reference GORM directly |

**Mapping convention:** `ToModel()` as a method on request DTOs; standalone `FromModel()` functions for response DTOs.

**Why not business logic in DTOs:**
- GORM hooks on models (e.g. `User.BeforeCreate` bcrypt) already own some business logic — duplicating concerns in DTOs creates conflicts.
- DTOs live in `api/`; if logic lives there it can't be reused by other consumers (CLI, workers) without creating cross-module coupling.

**Why not logic in models:**
- Models are shared across all consumers via `internal/shared` — embedding API-specific rules there pollutes the shared library.

### Service Layer Design (2026-02-27)

**Structure:** One sub-package per domain, mirroring the DTO layout.
```
api/internal/services/
├── user/user_service.go
├── address/address_service.go
├── product/product_service.go
├── category/category_service.go
├── review/review_service.go
├── order-item/order_item_service.go
├── order/order_service.go
└── payment/payment_service.go
```

**Pattern:** Each file defines an interface (`XxxServiceI`) and a concrete struct (`XxxService`) that implements it. Constructor takes a repository interface and returns the service interface: `func NewXxxService(repo XxxRepositoryI) XxxServiceI`. Services never hold `*gorm.DB` directly — see ADR-009.

**DTO import aliasing** — service package, repo package, and DTO package all share the same domain name (e.g. all `package user`). Alias at the import site: `userdto "commerce/api/internal/dto/user"`, `userrepo "commerce/internal/shared/repositories/user"`.

**Interface signatures:**

```go
// UserService
GetById(id uint) (*userdto.User, error)
GetByEmail(email string) (*userdto.User, error)
Create(dto *userdto.User) (*userdto.User, error)
Update(id uint, dto *userdto.User) (*userdto.User, error)
Delete(id uint) error
Authenticate(email, password string) (*userdto.User, error)

// AddressService
GetById(id uint) (*addressdto.Address, error)
GetByUserId(userId uint) ([]addressdto.Address, error)
Create(dto *addressdto.Address) (*addressdto.Address, error)
Update(id uint, dto *addressdto.Address) (*addressdto.Address, error)
Delete(id uint) error
SetDefault(userId uint, addressId uint) error

// ProductService
GetById(id uint) (*productdto.Product, error)
GetAll() ([]productdto.Product, error)
Create(dto *productdto.Product) (*productdto.Product, error)
Update(id uint, dto *productdto.Product) (*productdto.Product, error)
Delete(id uint) error

// CategoryService
GetById(id uint) (*categorydto.Category, error)
GetAll() ([]categorydto.Category, error)
Create(dto *categorydto.Category) (*categorydto.Category, error)
Update(id uint, dto *categorydto.Category) (*categorydto.Category, error)
Delete(id uint) error

// ReviewService
GetById(id uint) (*reviewdto.Review, error)
GetByProductId(productId uint) ([]reviewdto.Review, error)
Create(dto *reviewdto.Review) (*reviewdto.Review, error)
Update(id uint, dto *reviewdto.Review) (*reviewdto.Review, error)
Delete(id uint) error

// OrderService
GetById(id uint) (*orderdto.Order, error)
GetByUserId(userId uint) ([]orderdto.Order, error)
Create(dto *orderdto.Order) (*orderdto.Order, error)  // must create OrderItems in same transaction
UpdateStatus(id uint, status string) (*orderdto.Order, error)
Delete(id uint) error

// PaymentService
GetById(id uint) (*paymentdto.Payment, error)
GetByOrderId(orderId uint) ([]paymentdto.Payment, error)
Create(dto *paymentdto.Payment) (*paymentdto.Payment, error)
UpdateStatus(id uint, status string) (*paymentdto.Payment, error)
Delete(id uint) error
```

**`ToModel()` must include `Id` via `Base{}`** — the repo's `Save` uses `Id == 0` to distinguish create vs update. If `ToModel` omits the `Id`, updates silently insert a new row instead. Always map it:
```go
Base: models.Base{Id: dto.Id},
```
**Action required:** Audit all DTO `ToModel()` functions — as of 2026-03-09, only `order-item` has been fixed; all others are missing this.

**Notable implementation notes (service layer):**
- `UserService.Authenticate` — fetch by email, call `model.CheckPassword(password)`, return error if false.
- `AddressService.SetDefault` — call `repo.ClearDefault(userId)` then `repo.SetDefault(addressId)`.
- `OrderService.Create` — open a `db.Transaction(...)` and pass it down to create `Order` + all `OrderItems` atomically.
- `OrderService.UpdateStatus` / `PaymentService.UpdateStatus` — validate input string against model enum constants before calling repo.

---

## ADR-009 — Repository pattern for data access

**Date:** 2026-02-27
**Status:** Done

A repository layer is introduced between services and GORM. Services never hold `*gorm.DB` directly; they depend on repository interfaces.

**Layering:**
```
Handler → Service → Repository → GORM → DB
           (why)      (how)
```

**Location:** `internal/shared/repositories/` — sits alongside models in the shared module. GORM is already a dependency there, and repos are model-specific with no API concerns.

**Structure:** One sub-package per domain, same pattern as models and DTOs.
```
internal/shared/repositories/
├── user/user_repository.go
├── address/address_repository.go
├── product/product_repository.go
├── category/category_repository.go
├── review/review_repository.go
├── order/order_repository.go
└── payment/payment_repository.go
```
`OrderItem` has no dedicated repo — managed within `order/`.

**Pattern:** Each file defines an interface (`XxxRepositoryI`) and a concrete struct (`XxxRepository`). Constructor takes `*gorm.DB` and returns the interface: `func NewXxxRepository(db *gorm.DB) XxxRepositoryI`.

**Method naming:** `Find...` for reads, `Create`, `Update`, `SoftDelete` for writes.

**Soft-delete** — repos own the soft-delete logic so services don't need to know about it:
- All `Find...` methods filter: `.Where("deleted_date = ?", time.Time{})`
- `SoftDelete` sets: `.Update("deleted_date", time.Now())`

**Why repositories in `internal/shared/` and not `api/internal/`:**
- GORM is already a dependency of `internal/shared` — no new dependency introduced.
- Repos are model-specific (no API concerns) — they belong near models, not near handlers.
- Future consumers (e.g. a worker module) can reuse repos without importing the `api` module.

**Why not embed queries directly in services:**
- Services become testable without a real DB — inject a mock repo instead.
- Query logic is centralized; soft-delete filtering isn't scattered across services.
- Swapping GORM for another persistence mechanism only touches the repo layer.

---

## ADR-011 — Users cannot be hard-deleted via the API

**Date:** 2026-03-05
**Status:** Active

`UserService.Delete` soft-deletes only (`hard: false` hardcoded). Hard-delete is available at the repository level but intentionally not exposed through the service or any API endpoint.

**Rationale:** User records are referenced by orders, reviews, and addresses. Hard-deleting a user would orphan those records. Soft-delete preserves referential integrity and audit history.

---

## ADR-012 — Cascade constraints on all foreign key relationships

**Date:** 2026-03-10
**Status:** Closed — verified 2026-03-12

All models with foreign key relationships define explicit `OnDelete` constraints via GORM struct tags on association fields (not scalar FK columns). `foreignKey` tag values always use the Go struct field name (PascalCase) — GORM converts to snake_case for the DB column automatically.

**Constraint rules per relationship:**

| Parent | Child | Action |
|--------|-------|--------|
| `User` | `Address` | CASCADE |
| `User` | `Order` | CASCADE |
| `User` | `Review` | CASCADE |
| `Order` | `OrderItem` | CASCADE |
| `Order` | `Payment` | CASCADE (see ADR-007) |
| `Product` | `Review` | CASCADE |
| `Product` | `ProductCategory` | CASCADE |
| `Category` | `ProductCategory` | CASCADE |
| `Category` | `Category` (children) | CASCADE |
| `Address` | `Order` (shipping/billing) | RESTRICT |

**Key implementation notes:**
- Constraints live on association fields only (e.g. `User User`, `Order Order`) — scalar FK fields (e.g. `UserId uint`) just have `gorm:"not null"`
- `OnDelete:RESTRICT` on `Order.ShippingAddress` / `Order.BillingAddress` — prevents deleting an address that is still tied to an order
- GORM `AutoMigrate` only applies constraints on table creation, not to existing tables — see BUG-015 for the workaround

---

## ADR-013 — Order amount calculation strategy

**Date:** 2026-03-11
**Status:** Closed — verified 2026-03-12

Order amounts are split into three fields on the `Order` model: `SubTotalAmount`, `TaxAmount`, `TotalAmount`. Each is calculated differently.

| Field | Source | Where |
|-------|--------|-------|
| `SubTotalAmount` | `Σ (quantity × unit_price)` across all `OrderItems` | `OrderService.Save` |
| `TaxAmount` | `SubTotalAmount × rate` for the given state | `TaxService` (injected into `OrderService`) |
| `TotalAmount` | `SubTotalAmount + TaxAmount` | `OrderService.Save` |

**`TotalAmount` — service vs. DB generated column:**
A PostgreSQL `GENERATED ALWAYS AS (sub_total_amount + tax_amount) STORED` column was considered but rejected:
- GORM `AutoMigrate` does not add generated columns — requires a manual migration
- GORM needs special read-only tags (`<-:false`) to avoid writing the column
- The consistency benefit is minimal since `OrderService.Save` is the only write path
- One line of service code is clearer than schema complexity

Decision: calculate `TotalAmount` in the service layer.

**`TaxService` — rate source:**
- An external tax rate API was considered and rejected for MVP: adds a network dependency, latency, and a failure mode on every order creation
- Tax rates are stored as an in-memory `map[string]float64` (state abbreviation → rate), loaded at startup from a config file or hardcoded constants
- `TaxService` is behind an interface — swapping to an external source later is a one-file change

**`TaxService` interface:**
```go
type TaxServiceI interface {
    Calculate(subTotal float64, state string) (float64, error)
}
```

**Order DTO update required:** add `SubTotalAmount` and `TaxAmount` fields; `ToModel` must map them. `TotalAmount` remains on both DTO and model.

---

## ADR-014 — Unit testing strategy for the service layer

**Date:** 2026-03-12
**Status:** Done — all service tests implemented 2026-03-26

Unit tests cover the service layer only. Repository and DTO layers are not tested directly — repos are exercised through integration tests (future); DTOs are thin mappings with no logic to test.

**Mock library:** `go.uber.org/mock/gomock` with `mockgen` for code generation. `github.com/stretchr/testify/assert` for assertions. Both added to `api/go.mod`.

**Additional dependency:** `golang.org/x/crypto` added to `api/go.mod` to support bcrypt hash generation in `UserService` tests (needed to pre-populate `models.User.Password` so `CheckPassword` works without GORM hooks).

**Test file locations:** Co-located with each service, same package (white-box):
```
api/internal/services/
├── tax/tax_service_test.go
├── order/order_service_test.go         + mock_order_repo_test.go
├── user/user_service_test.go           + mock_user_repository.go
└── payment/payment_service_test.go     + mock_payment_repo_test.go
```

**Mock placement:** Generated mocks live alongside the test file of the consumer (`_test.go` package). `MockOrderRepo` belongs in `order/`, not in `internal/shared/repositories/order/`.

**Key testing notes:**
- `OrderService.Save` takes `dto.Order` by value — assert computed amounts inside `DoAndReturn` on the `*models.Order` passed to `repo.Save`, not on the caller's variable (it is never mutated)
- Use `assert.InDelta` for tax/total comparisons (floating point); `assert.Equal` is safe for subtotal (integer arithmetic)
- MD tax rate is `0.06` — subtotal 40.00 → tax 2.40 → total 42.40
- For invalid-state / invalid-status tests: register no `EXPECT` on the repo — gomock fails the test automatically if an unexpected call occurs

**Coverage implemented:**

| Service | Cases |
|---------|-------|
| `TaxService` | `Calculate`: valid state, zero-tax state (DE), invalid state, zero amount; `GetStates`: count = 51 |
| `OrderService` | `GetById`, `GetById` error, `GetAllByUser`, `GetAllByUser` error, `Delete` (soft), `Delete` (hard), `Delete` error, `Save` (amounts verified), `SaveInvalidState`, `UpdateStatus`, `UpdateStatusInvalid`, `UpdateStatusRepoError` |
| `UserService` | `Authenticate`, `InvalidAuthentication`, `GetById`, `GetAll`, `Save`, `Delete` |
| `PaymentService` | `GetById`, `GetByOrder`, `Delete`, `Save`, `UpdateStatus` |

---

## ADR-010 — Default sort order on all repository `Find` methods

**Date:** 2026-03-04
**Status:** Closed — implemented 2026-04-14 (issue #43)

All repository methods that return multiple records (e.g. `GetAll`, `GetByUserId`, `GetByProductId`) must apply an explicit `.Order(...)` clause before calling `Find()`. Without it, PostgreSQL returns rows in undefined order — results are non-deterministic across queries.

**Convention:** Default sort by `created_date ASC` unless the domain has a more natural ordering (e.g. `position`, `name`). Document any per-repo overrides inline.

**Example:**
```go
r.db.Order("created_date ASC").Find(&results)
```

**Action required:** Apply to all multi-record `Find` calls across all repositories once implementation is stabilised.

---

## ADR-016 — Centralized Docker structure with workspace-root build context

**Date:** 2026-04-14
**Status:** Active

Dockerfiles live in a centralized `docker/` directory with one subdirectory per service. `.dockerignore` and `docker-compose.yaml` live at the workspace root.

### Structure

```
docker/
├── api/
│   └── Dockerfile
├── utils/
│   └── Dockerfile
.dockerignore          # workspace root — applies to all builds
docker-compose.yaml    # workspace root — orchestrates the stack
```

### Build context

The build context is always the **workspace root**, not the Dockerfile's directory. This is required because the Go workspace (`go.work`) and all three modules (`api/`, `utils/`, `internal/shared/`) must be accessible during the build. The `-f` flag points to the Dockerfile:

```bash
docker build -f docker/api/Dockerfile .
docker build -f docker/utils/Dockerfile .
```

All `COPY` paths inside Dockerfiles are relative to the workspace root context, not the Dockerfile location.

### docker-compose.yaml

Each service uses `build.context: .` (workspace root) and `build.dockerfile: docker/<service>/Dockerfile`.

### Alternatives considered

- **`Dockerfile.api` / `Dockerfile.utils` at workspace root** — rejected; clutters the root as more services are added.
- **Dockerfile inside each module (`api/Dockerfile`, `utils/Dockerfile`)** — rejected; the build context would need to be `..` (parent traversal), which is less explicit and breaks some CI tooling.

### Postgres is managed externally (not in compose)

**Updated:** 2026-04-20 (issue #99)

`docker-compose.yaml` does not include a Postgres service. Postgres is provisioned separately — local dev uses an existing host install reached via `host.docker.internal`; other environments are handled by IaC. Compose only builds and runs the `api` and `utils` containers.

DB connection values (host, port, credentials, schema) are supplied via a root `.env` file (gitignored); `.env.example` is committed as a reference.

**Why:** Keeps compose focused on application containers. Provisioning and teardown of the DB is already handled by the user's existing tooling; duplicating that inside compose would couple local dev to a specific DB lifecycle for no real gain.

**Dependency ordering:** `api` uses `depends_on: utils` with `condition: service_completed_successfully` — `api` starts only after `utils` (migrations) exits with code 0. If migrations fail, `api` never starts.

---

## ADR-017 — Authorization via Auth0 (managed identity provider)

**Date:** 2026-04-22 (proposed) / revised 2026-04-27 / implementation begun 2026-05-05
**Status:** Accepted — supersedes the in-tree auth-server draft. Implementation issues #113–#116 opened (replacing #108/#109/#110, which described the rejected build-in-tree approach). Tenant config (iac-matrix#6) landed 2026-05-05.

All authorization concerns are delegated to **Auth0**. Tokens are issued by Auth0; consuming services (this API, the upcoming Python API, future frontends) validate them locally against Auth0's JWKS. There is no dedicated authentication-server in this codebase or alongside it.

### Two-track model (unchanged from the prior draft)

1. **Storefront users** — Auth0 Universal Login (email/password today; social federation available later without code changes). Auth0 issues a user JWT.
2. **Machine-to-machine clients** (partners, internal services) — Auth0 M2M Applications using OAuth 2.0 client credentials grant. Auth0 issues a scoped JWT.

Both flows produce JWTs validated by the same middleware in each consuming service.

### Why Auth0 (vs. AWS Cognito, self-hosted Zitadel, build-from-scratch)

- **Build-from-scratch** rejected: OAuth/OIDC is security-critical surface area (token signing, JWKS, key rotation, refresh, revocation, discovery). No business value in reimplementing a solved problem.
- **Self-hosted Zitadel** rejected: operational cost not justified at this stage; the portability argument is mostly theoretical until a concrete reason to migrate appears.
- **AWS Cognito** considered: cheaper, but rougher DX and weaker on the M2M side for a multi-app ecosystem.
- **Auth0** chosen on developer experience, SDK/library quality across Go/Python/SPA, and clean M2M support. Free tier covers early use; pricing to be revisited before scale.

### Why no dedicated authentication-server

A separate `authentication-server` service was considered and rejected. With a managed provider, that service collapses to a thin proxy over Auth0's Management API with no runtime responsibility. **Auth0 is the auth server.** Each consumer validates tokens locally against Auth0's JWKS — there is no central runtime auth dependency to operate.

If a real cross-cutting need emerges later (complex policy engine, multi-tenant onboarding flows, federation glue), a service can be extracted at that point. We don't pre-build it.

### Threat-model clarification (unchanged)

"Stop random websites from hitting the API" is not served by OAuth. Anyone with `curl` can hit any public endpoint — the control is authorizing the *request*, not the *origin*. CORS helps only against cross-origin scripts in other users' browsers; it is not a security boundary for servers. Defense remains: (a) every non-public route requires a valid bearer token, (b) rate limiting, (c) WAF at ingress if deployed. This ADR covers (a).

### Implementation surface

| Component | Where it lives | Notes |
|---|---|---|
| Auth0 tenant config (API, scopes, M2M apps, Actions) | Terraform, `auth0/auth0` provider — **`akhakpouri/iac-matrix` repo, landed 2026-05-05 (issue #6)** | Single source of truth for auth infra. API audience: `urn:commerce-api`. |
| JWT validation middleware (Go) | `api/internal/middleware/auth/` | Uses `github.com/auth0/go-jwt-middleware/v3` (v3, not v2); validates sig via JWKS, `iss`, `aud`, `exp`. `Claim` type implements `validator.CustomClaims`. |
| Scope-check helper (Go) | Same package | Per-route guard, e.g. `RequireScope("orders:write")` |
| Domain user mapping | commerce-api `users` table keyed by Auth0 `sub` claim | First-time login creates row; commerce profile fields stay here |
| Custom claims (if needed) | Auth0 Action (JS, runs inside Auth0) | e.g. embed internal `user_id` for cheap lookup |
| Frontend login | Auth0 SPA SDK in each frontend repo | Universal Login redirect; SDK attaches `Authorization: Bearer` |
| Python API | Equivalent JWT middleware in that repo | `authlib` or `python-jose` |

### What is NOT built in this codebase (compared to the prior draft)

The in-tree-auth-server draft of this ADR proposed all of the following. None of them are now needed:

- Token issuance, signing, or JWKS endpoint
- A user database for auth (Auth0 owns identity; commerce-api keeps domain user rows mapped by `sub`)
- `/auth/login`, `/auth/register`, `/oauth/token` handlers
- `ApiClient` model + repository + AutoMigrate registration
- Refresh-token rotation logic
- `JWT_SIGNING_KEY` env var (Auth0 holds the keys; consumers only need the public JWKS)
- `utils` CLI subcommand to register API clients (M2M apps live in Auth0)

### Open decisions

**Resolved (2026-05-05, via iac-matrix#6):**
- Scope vocabulary — finalized as `{category,orders,payment,products,reviews,users}:{read,write}` plus `users:delete`. Note `users:delete` is the only delete scope; given ADR-011 forbids hard-deletes via the API, it likely gates the soft-delete path. Mixed plural/singular (`category`/`payment` singular, others plural) is a known wart — revisit before tokens issue at scale.
- Auth0 tenant managed via Terraform from day one — `auth0/auth0` provider in the `akhakpouri/iac-matrix` repo. Dashboard is read-only.
- Audience identifier — `urn:commerce-api` (URN form, not URL — Auth0 just compares strings).

**Still open (to resolve before scope guard #114 lands):**
- Per-route classification — public / user-auth / M2M-auth + required scope. Will live in `routes.go`.
- Token expiry — Auth0 default (24h access, configurable) likely fine; revisit if needed.

### Consequences

- New `api/internal/middleware/auth/` package
- `routes.go` classifies each route as public / user-auth / M2M-auth-with-scope
- New env vars in `dev.env.example`: `AUTH_DOMAIN`, `AUTH_AUDIENCE` (no secret — JWKS is public). Note: keys use `AUTH_` prefix, not `AUTH0_` as originally drafted.
- Terraform module describing the Auth0 tenant (location TBD — likely `infra/auth0/` or a separate infra repo if one is established)
- Frontend repos (when created) integrate the Auth0 SPA SDK
- Python API (when created) implements equivalent JWT middleware
- **ADR-005 implication:** bcrypt password hashing on `User.Password` becomes unused for auth once Auth0 cutover is complete. `User.Password` can be deprecated/removed in a follow-up — tracked separately, not in scope here.

---

## ADR-018 — Event-driven backbone: transactional outbox + SNS/SQS fan-out, workers as separate apps

**Date:** 2026-06-25
**Status:** Accepted — not yet implemented (design under review)

### Context

The app is moving beyond CRUD toward a POS-style flow: purchase → ship → notify, expanding over time (payments, shipping, notifications, more). These side effects should be **decoupled** from the request path and from each other, and a second consumer ecosystem is coming (the planned Python API + frontends per ADR-017). We want an event-driven architecture that **starts small on the infrastructure we already run** (single ECS-deployed `api`, shared RDS Postgres in `matrix`) and grows without rewrites.

The domain already has the state machine the events ride on: `Order.Status` (pending → shipped → delivered → cancelled) and `Payment.Status`. "Events" are largely these transitions made explicit.

### Decision

**1. Transactional outbox in Postgres as the source of truth for events.**
When a domain state change happens, the producing service writes the event into a `commerce.outbox` table **in the same DB transaction** as the state change. This eliminates the dual-write problem (state saved but event lost, or vice versa). Publishing to the broker is deferred — the DB commit is the only thing that must be atomic with the business change.

**2. SNS + SQS fan-out as the broker (not EventBridge, not Kafka — for now).**
- A **single SNS topic** `commerce-domain-events` carries all domain events. Each event sets a `event_type` SNS **message attribute** (`OrderPlaced`, `OrderShipped`, …).
- **One SQS queue per consumer**, each subscribed to the topic with a **filter policy** on `event_type`, with **raw message delivery on** (SQS body = bare event envelope). Each queue has a **DLQ** (`maxReceiveCount = 5`).
- Adding a consumer = new queue + subscription + filter. **Producer and relay never change.** That additivity is the main reason for the single-topic design.
- EventBridge was considered and deferred: it wins when you need content-based routing rules, archive/replay, or many AWS/SaaS targets. SNS+SQS is cheaper, higher-throughput, lower-latency, and sufficient for a handful of event types. The producer publishes through an interface, so a later swap to EventBridge is a small, contained change.

**3. A standalone `relay` app drains the outbox to SNS.**
Polls `SELECT ... WHERE published_at IS NULL ORDER BY id LIMIT N FOR UPDATE SKIP LOCKED`, publishes each row to SNS with the `event_type` attribute, sets `published_at`. `SKIP LOCKED` is **mandatory** so the relay can run >1 replica (or coexist with API replicas) without double-publishing. Chosen as a separate ECS app (1 replica) to keep `api` HTTP-only and sidestep multi-replica relay coordination; an in-`api` goroutine was the rejected smaller alternative (safe with `SKIP LOCKED`, but couples relay lifecycle to the HTTP service).

**4. Workers are separate applications in this repo** (Go workspace modules, siblings to `api`/`utils`): `relay/`, `notifier/`, later `shipping/`. Each: own `go.mod`, `docker/<name>/Dockerfile`, ECR repo (`commerce-<name>-registry`, IMMUTABLE, sha tags), ECS task def + IAM task role in `matrix`. **NOT microservices split by domain** — they're workers split by *side effect*, sharing `internal/shared` (models, events, DB). Domain logic stays in `api`.

**5. At-least-once delivery → consumers must be idempotent.** SNS→SQS can redeliver. Every consumer dedupes on the envelope's `event_id` (a `processed_events` table or a naturally-idempotent side effect) before acting.

### Event envelope (contract, in `internal/shared/events/`)

`event_id` (uuid — idempotency key) · `event_type` (string — also the SNS attribute) · `occurred_at` · `aggregate` `{type,id}` · `version` (int — per-type schema version) · `payload` (json). The same shape is stored in the outbox and put on the wire.

### Outbox table (`commerce.outbox`, GORM model, migrated by `utils`)

`id` bigserial PK · `event_id` uuid unique · `event_type` · `aggregate_type`/`aggregate_id` · `payload` jsonb · `created_at` · `published_at` (nullable — NULL = the relay's work queue) · `attempts`. Partial index `WHERE published_at IS NULL` keeps the relay poll cheap; periodic cleanup of published rows is a later concern.

### Cross-repo split (this repo vs `matrix`)

- **This repo:** outbox model + migration (via `utils`), `events` contracts, `relay`/`notifier`/`shipping` apps + Dockerfiles + `publish-images.yml` matrix entries.
- **`matrix` (`aws/commerce/`):** SNS topic, SQS queues + DLQs, subscriptions + filter policies, ECS task defs/services, IAM task roles (relay → `sns:Publish`; notifier → `sqs:Receive`/`Delete`/`GetQueueAttributes` + `ses:SendEmail`), and OIDC/ECR plumbing matching the existing `api`/`utils` pattern.

### First slice (walking skeleton): `OrderPlaced → email`

`internal/shared` events package + `Outbox` model → `OrderService.Save` writes one `OrderPlaced` row in its txn → `relay` publishes → `notifier` consumes `commerce-notifications-queue` (filter `OrderPlaced`) → SES email. Proves the entire pipe with one event and one side effect. Everything later (`OrderShipped`, `shipping` worker, payment events) clones the relay/notifier skeleton + a queue/filter — no backbone change.

### Failure semantics (consumer side)

SQS never observes the consumer's exception — it only tracks whether the message was **deleted** before its **visibility timeout** expired. Failure is the *absence* of a successful `DeleteMessage`, not an active signal.

**Retry loop:** `ReceiveMessage` makes the message invisible for the visibility timeout. On success the consumer calls `DeleteMessage`. On exception/crash it doesn't delete → the message reappears when the timeout expires, `ReceiveCount` increments, and it's redelivered. Retries are **fixed-interval** (one visibility timeout each), not exponential, unless the consumer calls `ChangeMessageVisibility` to back off per attempt.

**DLQ as circuit breaker:** redrive `maxReceiveCount = 5`. On the 6th delivery SQS routes the message to `commerce-<consumer>-dlq` instead of redelivering — a poison message is quarantined after 5 tries rather than looping forever. Nothing drains the DLQ automatically: a **CloudWatch alarm on DLQ depth > 0** pages a human; after the fix you **redrive** the DLQ back to the source queue to reprocess.

**Blast radius:** a consumer failure does **not** touch the `outbox` row (already `published_at`-stamped — the relay is done), does not roll back the order, and does not block other messages on the queue (standard SQS processes them concurrently). Postgres is unaware any retry is happening.

**Exactly-once is impossible past the broker; close the gap with idempotency.** The outbox gives exactly-once *into SNS*; SNS→SQS→consumer is **at-least-once**. Because the side effect (SES, a shipping API) isn't transactional with the dedup write, a redelivery can repeat it. Mitigations, in order of strength:
- Consumer **dedupes on `event_id`** (a `processed_events` table) before acting — mandatory.
- Operation **ordering is a deliberate trade-off:** *side-effect-first then mark-processed* risks a duplicate on crash-in-between (chosen default for notifications — a dup email beats a lost one); *mark-processed-first* risks a silently-lost side effect. Pick per consumer and document it.
- For consequential side effects (charge, ship), also pass a downstream **idempotency key** (Stripe/SES token) so the provider collapses the duplicate request, not just our dedup table.

**Two settings that make-or-break it:**
1. **Visibility timeout > worst-case processing time** — otherwise SQS redelivers while the first attempt is still running → needless double-processing.
2. Default retry cadence is fixed; add **per-attempt backoff** via `ChangeMessageVisibility` for failures expected to be transient (downstream throttling).

### Outbox retention — `apps/janitor` (Lambda, daily)

Published outbox rows are never reread by the relay (partial index excludes them) but accumulate as physical bloat. They're pruned by a dedicated app, **`apps/janitor`**. Decision (2026-06-25): janitor runs as a **scheduled AWS Lambda**, *not* an ECS task — a once-a-day, short-lived, infrequent DB sweep is a textbook Lambda fit (no idle Fargate cost, no long-running process).

- **Trigger:** EventBridge Scheduler cron, once daily, invokes the Lambda. It's a *scheduled single invocation*, not a poll loop; within one invocation it loops batched deletes until none remain.
- **Action:** batched `DELETE FROM commerce.outbox WHERE published_at IS NOT NULL AND published_at < now() - interval '7 days'` in chunks (~5k), each a short txn, looping until 0 rows — avoids the long lock / WAL spike of one giant delete.
- **Hard delete** — events are plumbing, not the audit source of truth. If that changes, switch to archive-before-delete (an `outbox_archive` table / S3).
- **Retention window:** 7 days of published history kept for forensics, then pruned.

**Lambda divergences from the ECS fleet (captured so they aren't rediscovered):**
- **Packaging differs from the image pipeline.** (a) *Recommended:* container image on `public.ecr.aws/lambda/provided:al2023` pushed to ECR — reuses the existing ECR/OIDC publish shape. (b) Zip of a `bootstrap` binary on `provided.al2023` — lighter, but doesn't fit the `publish-images.yml` image matrix. Either way the entrypoint is a Lambda handler (`aws-lambda-go/lambda.Start`), not a run-and-exit `main()` like `utils`.
- **Must be VPC-attached to reach RDS** (private subnets). Needs subnet + security-group config; the RDS SG must allow 5432 from the Lambda SG. (#1 thing forgotten with Lambda+RDS.)
- **IAM execution role:** `secretsmanager:GetSecretValue` on `/commerce-api/rds/psql` + the VPC/ENI-managed policy. **No SNS/SQS perms** — janitor only touches Postgres.
- **15-min Lambda timeout** is ample; if a backlog ever exceeds it, it deletes what fits and the next day catches up (self-healing).
- **Path convention:** `apps/janitor` introduces an `apps/` grouping not used by the current flat layout (`api/`, `utils/`). Open sub-decision: move the other new workers (`relay`, `notifier`, `shipping`) under `apps/` for consistency, or let janitor be the lone exception. Still a Go workspace module either way → add to `go.work` + the CI `go work init` use-list.

### Consequences

- New workspace modules → update the CI `go work init` `use` list (the duplicated-list gotcha in CLAUDE.md) and pin each to `go 1.26.5` (the gorm-kit floor, ADR-015 amendment).
- New AWS surface in `matrix`: SNS/SQS/SES, more IAM, more ECS tasks — operational cost grows.
- `api` gains a hard dependency on the outbox write succeeding inside the order txn; a bug there blocks order creation (acceptable — correctness over availability for purchases).
- Consumers carry an idempotency/dedup obligation forever; document it per worker.
- Re-evaluate EventBridge if event types proliferate or replay/routing needs appear.

### Amendment (2026-07-01, with #130) — relay internal concurrency: autonomous workers, per-worker transaction ("Model B")

Point 3 fixed the relay's *external* shape (separate app, `SKIP LOCKED`, safe at N replicas) but not how it parallelizes *internally*. Decided:

**The relay runs N autonomous worker goroutines, each owning its own DB session and transaction — no coordinator, no shared transaction.** Each worker loops independently:

```
BEGIN
SELECT ... FROM commerce.outbox
  WHERE published_at IS NULL ORDER BY id LIMIT <batch>
  FOR UPDATE SKIP LOCKED
publish each row to SNS
UPDATE published_at = now() on the rows that SNS accepted
COMMIT   -- releases the row locks
```

`SKIP LOCKED` is the *entire* coordination mechanism. It hands each concurrent claim a **disjoint** row set, and it does so identically whether the competing claimers are goroutines in one process or separate ECS replicas — the same one line scales both axes (intra-process pool **and** horizontal replicas) with no leader election or distributed lock.

**Ordering — publish *before* commit (non-negotiable).** The claim→publish→mark→commit order makes delivery **at-least-once**. A crash in the publish→commit window leaves the row `published_at IS NULL`, so the next loop re-publishes → a *duplicate* (tolerable; consumers dedupe on `event_id` per point 5). The reverse order (mark/commit then publish) was **rejected**: a crash there marks the row published but never sends it → a *silently lost event*, which defeats the outbox. Duplicates are recoverable; loss is not.

**Rejected alternative — "Model A" (one coordinator claims + fans rows out to publish-only workers over a single shared tx).** More moving parts (fan-out, result collection), and a shared `*gorm.DB` transaction is **not safe for concurrent use across goroutines**. Model B's tx-per-worker dissolves that hazard entirely — concurrent DB access is just normal connection-pool usage — and each worker is a self-contained mini-relay.

**Load throttling is two independent knobs**, both configurable (default batch 50): `batch` bounds how many rows one claim locks; the **worker count W** bounds how many claims/SNS publishes are in flight at once. Batch size is kept modest deliberately — the row locks and the DB connection are held for the *whole* window including the SNS network round-trip, so a large batch = a long transaction pinning a connection on I/O.

**The one condition that reverses this choice: a per-aggregate ordering requirement.** Model B's workers interleave freely, so it provides **no ordering guarantee** across events. Accepted because the current slices are order-independent (`OrderPlaced → email` doesn't depend on sequencing). If strict per-aggregate FIFO ever becomes required, revisit — that forces a single claimer partitioning by aggregate key (back toward Model A) and/or an SNS FIFO topic.

**Consequences:** the relay's DB connection pool must be sized **≥ W** (every worker holds a connection for its full claim→publish→commit window, SNS latency included) — undersize it and workers starve. Only SNS-accepted IDs are marked; a per-row publish failure leaves that row `published_at IS NULL` (bump `attempts`) for a free retry next loop. Batch stays small to keep transactions short.

### Amendment (2026-07-21) — cross-service sequencing via event chaining, not shared fan-out

**Context:** ADR-018's fan-out has every consumer subscribe independently to the same event and process in parallel — correct for the first slice (`OrderPlaced → email`), which has no ordering dependency. But some future workflows do: **payment must clear before shipping/notification act on an order** (`OrderPlaced → billing → {shipping, notification}`). Subscribing billing, shipping, and notification all to `OrderPlaced` would run them concurrently — shipping and notification would race ahead of an unconfirmed payment.

**Decision: model an ordering dependency as an event chain (choreography), not a shared subscription.**
- Relay publishes `OrderPlaced` to the SNS topic as usual. Only **billing-service** subscribes to it.
- Billing-service processes the payment. For idempotency it tracks processed events in its own dedupe table before acting (ADR-018 point 5 — same obligation as any consumer).
- When billing's transaction completes, it publishes a **new** event — `OrderPaid` on success, `PaymentFailed` on failure — using the same transactional-outbox pattern the whole backbone runs on: billing gets its own `outbox` table + its own relay-clone (point 4 — "later events clone the relay/notifier skeleton... no backbone change"), writing the follow-up event in the same DB transaction as the payment outcome.
- **Notification-service and shipping-service subscribe to `OrderPaid`, not `OrderPlaced`.** They structurally cannot act before payment is confirmed, because the event they react to doesn't exist until billing emits it. `PaymentFailed` can fan out to its own subscriber(s) (e.g. a "payment failed" email) instead of shipping/notifying.

**Rationale:** preserves the additivity from point 2 — adding billing didn't change the `api`/`relay` publisher, it just became a new subscriber and, separately, a new publisher of its own event. Fan-out (parallel, independent reactions) stays the default; chaining is the deliberate exception used only where one consumer's output must gate another's input.

**Rejected alternative — central orchestrator** (e.g. Step Functions) explicitly calling billing → shipping → notification. More visible control flow, but re-couples the system around a component that must know the whole workflow, undermining "producer/relay never change." Fallback if choreography chains get too deep or branchy to reason about.

**Consequence — general rule going forward:** any service that both consumes and produces events (billing today, others later) needs its own transactional-outbox + relay pair, not a bare SQS consumer. Before wiring a new consumer, check whether its output must gate another consumer's input; if yes, chain events (new event, narrower subscription) — don't fan the same event out to both.

**Status:** decided, not yet implemented — billing-service doesn't exist yet. This fixes the pattern for when it's built.

### Amendment (2026-07-21) — resolved: relay publishes to SNS, not direct-to-SQS

Closes the open question carried since 2026-07-10 (`issues.md` #130): whether `relay` should publish straight to one SQS queue or through SNS. **Decided: SNS**, per the original design in point 2 and the Terraform already drafted (`matrix/aws/commerce/sns-sqs.tf` — topic `commerce-domain-events`). The relay code built against direct SQS (`apps/relay/internal/publisher.SqsPublisher`, resolving a queue named `commerce-queue`) needs to be replaced with an SNS publisher (topic ARN, `sns:Publish`) before `ProcessBatch` goes live — tracked as follow-up work under #130, not yet done.

---

## ADR-019 — Observability for the event-driven backbone (logs, metrics, distributed tracing)

**Date:** 2026-06-26
**Status:** Accepted (deferred) — sketched 2026-06-26; only Phase 0 rides with ADR-018 #130, the rest is shelved. Tracking: #136 (design) + per-phase issues.

### Context

ADR-018 fans a single user action across api → outbox → relay → SNS → SQS → notifier → SES — separate processes across an async boundary. Logs alone can't answer "where did this order's notification go?" once delivery is asynchronous. Observability stops being optional the moment the event backbone ships; it's what makes async debuggable.

### Decision

Adopt the three pillars, AWS-native and vendor-neutral, rolled out in phases:

- **Structured logging** — standardize the existing `slog` usage on a JSON schema where every line in a flow carries `correlation_id` + `event_id`. CloudWatch log groups already exist.
- **Metrics — RED + backbone signals** — API rate/error/latency (Gin middleware); **outbox lag** (`published_at IS NULL` count + age of oldest), relay publish success/fail, **SQS depth + `ApproximateAgeOfOldestMessage`**, **DLQ depth**, consumer duration + success/fail, dedupe hits. Emit via CloudWatch **EMF** (metrics-from-logs, no agent) or OTel.
- **Distributed tracing** — one trace across the whole async flow, via **W3C `traceparent`** propagated through the event envelope so the consumer span links to the producer span.

**Stack:** OpenTelemetry instrumentation in the Go apps → **ADOT collector** (ECS sidecar) → CloudWatch (logs + metrics) + X-Ray (traces). OTel keeps the backend swappable (Grafana/Datadog later, no re-instrumentation).

### Phasing

- **Phase 0 — rides with ADR-018 #130, NOT deferred:** reserve `correlation_id` + `traceparent` on the event envelope **and** the `commerce.outbox` row; generate `correlation_id` at the API edge in `OrderService.Save`. Nothing reads them yet.
- **Phase 1:** structured `slog` schema + CloudWatch alarms (DLQ depth, outbox lag, queue age).
- **Phase 2:** RED metrics + per-app dashboards.
- **Phase 3:** full OTel distributed tracing across the async boundary.

### Why Phase 0 can't wait

The envelope is defined in #130. The trace/correlation **carrier must exist from the first event** — retrofitting it across a *live* event stream means migrating in-flight events. Two fields now vs. a painful migration later. Everything Phase 1+ is genuinely deferrable; the carrier is not.

### Consequences

- The event envelope schema (ADR-018) carries two observability fields from day one even though nothing consumes them until Phase 3.
- ADOT collector sidecar adds a container to each ECS task def (`matrix`) when Phase 2/3 land — more cost, more IAM (X-Ray/CloudWatch put permissions).
- Promote this ADR from "deferred" to "implemented" per phase as issues close.

---

## ADR-020 — Event-sourced Order aggregate (isolated Postgres event store)

**Date:** 2026-07-13
**Status:** Accepted — isolated scope only. Not wired into the live `OrderService`, handlers, or routes.

### Context

Consolidated research (Auth0, DDD, the outbox pattern, and event sourcing specifically) done across several external chat threads was brought into a project session and evaluated against what's actually in this repo. Event sourcing was flagged in that research as exploratory — "not yet built" — with an explicitly open question: how it relates to the *existing* transactional outbox (ADR-018). This ADR resolves that question for the current issue by scoping event sourcing to a standalone addition, not a replacement.

The live `Order` domain today is substantial: `models.Order`/`OrderItem` (GORM), `OrderRepositoryI`, `OrderService.Save` (computes subtotal/tax/total, persists `Order` + nested `OrderItems` via a GORM association, atomically emits an `OrderPlaced` outbox row via a transaction manager), DTOs, handlers, routes gated by `orders:read`/`orders:write` scopes, and existing unit tests/mocks. `OrderServiceI.GetByUserId` (list orders by user) has no answer under pure event streams without a secondary index — that's exactly what a materialized-views/projection layer would solve, and that work is explicitly out of scope for the originating issue (same status as the outbox/SNS-SQS wiring). A full cutover was rejected for this reason: it would require solving the projection problem inline, expanding scope well past what was asked.

### Decision

Build a self-contained event-sourced `Order` aggregate and generic Postgres-backed event store as new, isolated code. The existing `Order`/`OrderItem` models, `OrderRepositoryI`, `OrderService`, handlers, and routes are **unchanged** and continue serving live traffic. Whether/how the event store eventually feeds the outbox, replaces `OrderService`'s persistence, or gets wired into handlers is deferred to a follow-up issue once the projection question (ADR needed) is resolved.

**Package:** `internal/shared/eventsourcing` — generic infra (`EventStore`, upcaster registry, aggregate contract) and the `Order` aggregate live together in one package, not split, per the isolated scope of this ADR (revisit if the package grows unwieldy). The GORM-mapped envelope struct itself (`models.Event`) lives in `internal/shared/models/` instead, alongside every other model — consistent with the existing `Outbox` precedent (also an envelope-style table, also in `models/`). It does **not** embed `Base`: `Base`'s `UpdatedDate`/`DeletedDate` imply mutability an append-only log must never have (unlike `Outbox`, whose rows are legitimately mutated after insert). `models.Event` defines its own `GlobalSeq` (autoincrement PK) and a plain `OccurredAt`, with no update/soft-delete columns.

**Events table** — registered via `AutoMigrate` in `internal/shared/database/main.go`'s existing model list (same mechanism as `Outbox`), not a new migration path:

```
stream_id      uuid        not null
version        int         not null   -- 1-indexed, per-stream
event_type     varchar     not null
schema_version int         not null
payload        jsonb       not null
occurred_at    timestamp   not null
global_seq     bigint      autoincrement, PK
UNIQUE(stream_id, version)
```

**EventStore** — `Append` (optimistic concurrency: assigns `expectedVersion+1, +2, ...` to a batch in one transaction; a Postgres unique-violation on `(stream_id, version)` maps to a typed `ErrConcurrencyConflict` — caller reloads the aggregate and re-derives the command, never blind-resends), `Load` (by stream), `LoadSince` (by `global_seq`, for future projections — unused by anything in this issue).

**Order aggregate** — `raise()`/`mutate()`/`rehydrate()`: `mutate()` is the single state-transition function, called identically for live commands and event replay. Two events for this issue: `OrderPlaced` (creation, snapshotted line items — product id/name/price/qty at order time, per the DDD note that `Order` owns immutable snapshots rather than live `Product` references) and `OrderStatusChanged` (enforces the same valid-transition rules as today's `models.OrderStatus`; invalid transitions are rejected before raising, so they never reach the store). `OrderStatusChanged` exists specifically so the concurrency-conflict test has a real business invariant to exercise (two concurrent status changes racing on one stream), not just a mechanical version check.

**Upcasting** — registry keyed by `(event_type, schema_version)`, dispatched at the deserialization boundary before `mutate()` sees the payload. One entry for this issue, reusing the worked example from the source research: `OrderPlaced` v1 (`TotalCents`) → v2 (`TotalAmount`).

**Testing** — two tiers, since no code in this repo below the service layer (mocked via gomock) has ever been tested against a real DB:
- In-memory fake `EventStore` (same interface) for rehydration, mixed-schema/upcasting, and invalid-transition tests — fast, no DB.
- Postgres-integration tests for the real store: normal `Append`, and concurrency-conflict + reload-and-retry (two goroutines racing `OrderStatusChanged` on the same stream) — this one needs the actual `UNIQUE(stream_id, version)` constraint to prove anything; a fake can't stand in for it.

### Consequences

- `Order` has two coexisting, unconnected persistence paths until a follow-up issue decides how (or whether) to cut over: the legacy GORM row via `OrderService`, and the new isolated event-sourced aggregate. This is deliberate, not an oversight — see Context.
- The projection/materialized-view gap (`GetByUserId` and friends) is a known, named prerequisite for ever wiring this into live traffic — not solved here.
- Relationship to ADR-018 stays exactly as it is today: `OrderService.Save`'s atomic `Order` + `Outbox` commit is untouched. A future ADR is needed before the event store and the outbox are connected.
- New test category for this repo (Postgres-integration tests) — needs a documented way to run against a local dev DB; not previously required by any existing test.

---