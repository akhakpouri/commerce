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
**Status:** Active

Gin (`github.com/gin-gonic/gin`) chosen as the HTTP framework for the `api` module. Rationale: project already uses third-party dependencies (GORM, testify), and Gin's request binding, middleware chain, and structured error responses reduce handler boilerplate. Stdlib `net/http` was considered but Gin was preferred for development speed given the full service layer already in place.

Handlers live in `api/internal/handlers/`, one sub-package per domain. Routes wired in `api/main.go` (or a dedicated `router.go`). Gin is added to `api/go.mod` only — `internal/shared` stays dependency-free from HTTP concerns.

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
**Status:** Active — pending implementation

Unit tests cover the service layer only. Repository and DTO layers are not tested directly — repos are exercised through integration tests (future); DTOs are thin mappings with no logic to test.

**Mock library:** `github.com/stretchr/testify` (`testify/mock` + `testify/assert`/`require`). Added to `api/go.mod`. Hand-written mocks only — no code generation.

**Additional dependency:** `golang.org/x/crypto` added to `api/go.mod` to support bcrypt hash generation in `UserService` tests (needed to pre-populate `models.User.Password` so `CheckPassword` works without GORM hooks).

**Test file locations:** Co-located with each service, same package (white-box):
```
api/internal/services/
├── tax/tax_service_test.go
├── order/order_service_test.go
├── user/user_service_test.go
└── payment/payment_service_test.go
```

**Mock structure:** Each test file defines local mock structs by embedding `mock.Mock` and implementing the relevant repository/service interface. One mock per interface, one method stub per interface method.

**Mock placement:** Mocks live in the test file of the consumer, not next to the interface they implement. `MockOrderRepo` belongs in `order_service_test.go`, not in `internal/shared/repositories/order/`. Mocks are a testing artifact of the consumer — the repository package has no need for a mock of itself.

The only exception: if the same mock is needed across multiple test packages, extract it to `api/internal/mocks/`. That is not currently the case.

**Test suite pattern:** Use `testify/suite` for services with multiple tests that share mock setup (`OrderService`, `UserService`). Define mocks as suite fields and reset them in `SetupTest()` — this gives every test a clean mock with no leftover state. Use plain top-level functions for simpler cases (`TaxService`, `PaymentService`).

**Coverage targets per service:**

| Service | Key cases |
|---------|-----------|
| `TaxService` | `Calculate`: valid state, zero-tax state (AK), invalid state, zero amount; `GetStates`: sorted, count = 51 |
| `OrderService` | `Save`: correct SubTotal/Tax/Total written to repo, empty items, tax service error aborts save; `UpdateStatus`: all four valid statuses pass through, invalid status rejected before repo call |
| `UserService` | `Authenticate`: valid credentials, wrong password, user not found; `GetById`: found and not found; `Delete`: always calls repo with `hard=false` |
| `PaymentService` | `UpdateStatus`: all seven valid statuses pass through, invalid status rejected before repo call, repo error propagates |

**User test setup pattern:** Generate a bcrypt hash at `bcrypt.MinCost` (faster than `DefaultCost`) in a test helper, pre-populate `models.User.Password`, then assert `CheckPassword` behaves correctly — no GORM hooks involved.

---

## ADR-010 — Default sort order on all repository `Find` methods

**Date:** 2026-03-04
**Status:** Pending

All repository methods that return multiple records (e.g. `GetAll`, `GetByUserId`, `GetByProductId`) must apply an explicit `.Order(...)` clause before calling `Find()`. Without it, PostgreSQL returns rows in undefined order — results are non-deterministic across queries.

**Convention:** Default sort by `created_date ASC` unless the domain has a more natural ordering (e.g. `position`, `name`). Document any per-repo overrides inline.

**Example:**
```go
r.db.Order("created_date ASC").Find(&results)
```

**Action required:** Apply to all multi-record `Find` calls across all repositories once implementation is stabilised.

---