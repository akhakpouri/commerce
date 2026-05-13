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
**Status:** Active — implemented 2026-03-30

DB connection logic (DSN construction + `gorm.Open`) lives exclusively in `internal/shared/database`. Both `api` and `utils` use it — no duplication.

**`internal/shared/database/main.go`:**
- `Connect(cfg DbConfig) (*gorm.DB, error)` — builds DSN, opens and returns `*gorm.DB`
- `Migrate(cfg DbConfig)` — calls `Connect` internally, then runs `AutoMigrate`

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