# Work Log

## Issue #82 — Payment handler

**Date:** 2026-04-08
**Status:** Done
**Branch:** feature/issue-82

Completed `PaymentHandler` and wired the nested `/orders/:order_id/payments` route.

- [x] `payment_handler.go` — `GetById`, `GetByOrder`, `Save`, `Delete`, `GetStatuses`, `UpdateStatus` implemented with Swagger annotations
- [x] `payment_handler.go` — `UpdateStatus` binds `dto.PaymentStatus` from request body; delegates validation to service
- [x] `payment_handler.go` — `GetStatuses` returns 404 if list is empty (in-memory — should never happen in practice)
- [x] `routes.go` — nested route `GET /api/orders/:order_id/payments` wired to `paymentHandler.GetByOrder`
- [x] `README.md` — updated status section to reflect all handlers complete; removed stale "hello, world!" note

---

## Issue #78 — User handler + nested address route

**Date:** 2026-04-06 → 2026-04-07
**Status:** Done
**Branch:** feature/issue-78

Completed `UserHandler` and wired the nested `/users/:user_id/addresses` route via `AddressHandler`.

- [x] `dto/user/authenticate.go` — new `Authenticate` DTO (email + password) for auth endpoints; `omitempty` removed from `Password` (see BUG-020)
- [x] `user_handler.go` — `GetById`, `GetAll`, `Authenticate`, `GetByEmail`, `Delete`, `Save` implemented with Swagger annotations
- [x] `user_handler.go` — `GetByEmail` returns 204 (existence check, no body — intentional)
- [x] `address_handler.go` — `GetByUserId` added; reads `user_id` param, delegates to `svc.GetAllByUserId`
- [x] `routes.go` — nested route `GET /api/users/:user_id/addresses` wired to `addressHandler.GetByUserId`
- [x] `api/internal/CLAUDE.md` — documented two-type `dto/user/` package and when to use each
- [x] `docs/project-notes/bugs.md` — BUG-020: `omitempty` on required fields silently drops values

---

## Issue #80 — Swagger annotation fixes + address handler completion

**Date:** 2026-04-02 → 2026-04-03
**Status:** Done
**Branch:** feature/issue-80

Fixed incorrect Swagger `@Router` path parameter syntax (`/:id` → `/{id}`) across all handlers, converted `address_handler` hard-delete to an optional query param, and completed the `AddressHandler` with a `Save` endpoint.

- [x] `address_handler.go` — route changed from `/:id/*hard` to `/:id`; `hard` now read via `c.DefaultQuery("hard", "false")`; `@Param hard query bool false` annotation added
- [x] `address_handler.go` — `@Router` updated to `{id}` OpenAPI syntax
- [x] `address_handler.go` — `POST /api/address` (`Save`) implemented with `@Param address body` annotation
- [x] `product_handler.go` — `@Router` updated to `{id}` for `GetById` and `Delete`; `@Param product body` added to `Save`
- [x] `category_handler.go` — `@Router` updated to `{id}` for `Delete`, `GetAllProductsByCategory`, `GetAllByParentId`, `GetById`; `@Param category body` added to `Save`
- [x] `api/internal/CLAUDE.md` — Swagger annotation rules updated with `{id}` vs `/:id` distinction, query param pattern for optional booleans, and tab-indented `@Param body` format

---

## Issue #80 — Category handler

**Date:** 2026-04-01
**Status:** Done
**Branch:** feature/issue-80

Completed `CategoryHandler` in `api/internal/handlers/category/category_handler.go`. Added `helpers.ParseParamToUint` to eliminate repeated `strconv.ParseUint` boilerplate.

- [x] `GET /api/category/` — `GetAll`
- [x] `GET /api/category/:id` — `GetById`
- [x] `GET /api/category/:id/children` — `GetAllByParentId`
- [x] `GET /api/category/:id/products` — `GetAllProductsByCategory` (delegates to `ProductService`)
- [x] `POST /api/category/` — `Save`
- [x] `DELETE /api/category/:id` — `Delete`
- [x] `api/internal/helpers/helper.go` — `ParseParamToUint` shared helper
- [x] Swagger docs regenerated; all endpoints annotated
- [x] `.vscode` untracked from git; `.gitignore` updated

---

## Issue #79 — Product handler

**Date:** 2026-04-01
**Status:** Done
**Branch:** feature/issue-79

Implemented the `ProductHandler` in `api/internal/handlers/product/product_handler.go` with full Swagger annotations. Wired DB connection via `container.Container` pattern (ADR-015). Also added `CategoryHandler` for the `GET /api/category/:id/products` endpoint (`GetAllByCategory` moved here per REST convention).

- [x] `GET /api/products/` — `GetAll`
- [x] `GET /api/products/:id` — `GetById`
- [x] `POST /api/products/` — `Save`
- [x] `DELETE /api/products/:id` — `Delete`
- [x] `GET /api/category/:id/products` — `GetAllByCategory` (CategoryHandler)
- [x] Container pattern implemented (`api/container/container.go`) — all services wired via `NewContainer(*gorm.DB)`
- [x] `api/internal/dto/err/error.go` — shared error response DTO
- [x] Swagger docs regenerated

**Note:** `GetAllByCategory` was intentionally placed in `CategoryHandler` (not `ProductHandler`) — nested resource convention (`/category/:id/products`).

---

## Issue #44 — HTTP handlers (Gin)

**Date:** 2026-03-26
**Last updated:** 2026-03-26
**Status:** In progress
**Branch:** feature/issue-73

Implement HTTP handlers using Gin (ADR-004). Service layer and DTOs are complete — handlers are the next layer.

- [x] Add `github.com/gin-gonic/gin` to `api/go.mod`
- [x] Wire router in `api/main.go`
- [x] `api/server/server.go` — `Server` struct with graceful shutdown (`Run()`)
- [x] `api/server/router/routes.go` — `RegisterRoutes()` composition root for HTTP layer
- [x] `api/configs/config.go` — env-based config, `GetEnvOrPanic`, CORS middleware
- [x] `api/internal/constants/constants.go` — typed env key + header name constants
- [x] `api/internal/handlers/tax/tax_handler.go` — first handler group (`GET /api/v1/taxes/`)
- [ ] Wire DB connection (`databaseConfig` exists but unused — needed before repo-backed handlers)
- [ ] Implement remaining handler groups (one per domain)

---

## Issue #69–#73 — Service layer unit tests (ADR-014)

**Date:** 2026-03-12
**Last updated:** 2026-03-26
**Status:** Done
**Branch:** feature/issue-73

- [x] #69 — Add test dependencies to `api/go.mod` (testify, bcrypt)
- [x] #70 — Unit tests for `TaxService`
- [x] #71 — Unit tests for `OrderService`
- [x] #72 — Unit tests for `UserService`
- [x] #73 — Unit tests for `PaymentService`

#69 must be completed before any test file work begins. See ADR-014 in `decisions.md` for mock strategy and full test case matrix.

**OrderService test cases (feature/issue-73):**
- `TestGetbyId` — happy path
- `TestGetbyIdError` — repo error propagation
- `TestGetAllByUser` — happy path, asserts count
- `TestGetAllByUserError` — repo error propagation
- `TestDelete` — soft delete
- `TestDeleteHard` — hard delete
- `TestDeleteError` — repo error propagation
- `TestSave` — verifies SubTotal/Tax/Total computed correctly before persist (DoAndReturn on model)
- `TestSaveInvalidState` — invalid BillingState → tax service errors → repo never called
- `TestUpdateStatus` — valid status → repo called
- `TestUpdateStatusInvalid` — invalid status → error, repo never called
- `TestUpdateStatusRepoError` — repo error propagation

**Key testing notes:**
- `Save` takes `dto.Order` by value — assert computed amounts inside `DoAndReturn` on the model, not on the caller's variable
- Use `assert.InDelta` for tax/total (floating point); `assert.Equal` is safe for subtotal (integer arithmetic)
- MD tax rate is `0.06` — subtotal 40.00 → tax 2.40 → total 42.40

---

## Issue #66 — Compute SubTotal, Tax, and Total in OrderService.Save

**Date:** 2026-03-11
**Status:** Done

- [x] Add `SubTotalAmount`, `TaxAmount`, `BillingState` to `Order` DTO; updated `ToModel`/`FromModel`
- [x] Inject `TaxServiceI` into `OrderService` constructor
- [x] Compute `SubTotalAmount`, call `TaxService.Calculate`, set `TotalAmount` in `Save`
- [x] `calculateTax` returns `(float64, error)` — no pointer, error propagated in `Save`
- [x] `GetById` in order repo preloads `BillingAddress` so `BillingState` is populated

---

## Issue #65 — Implement TaxService

**Date:** 2026-03-11
**Status:** Done

- [x] Create `api/internal/services/tax/tax_service.go`
- [x] `TaxServiceI` interface with `Calculate(amount float64, state string) (*float64, error)` and `GetStates() []string`
- [x] In-memory `map[string]dto.Tax` implementation; error on unknown state
- [x] `float64` throughout for precision
- [x] `GetStates` returns sorted keys

---

## Issue — Repository + Service layer implementation (ADR-008, ADR-009)

**Date:** 2026-02-27
**Last updated:** 2026-03-12
**Status:** Done
**Branch:** `feature/issue-22`

Implementing the repository layer (ADR-009) and service layer (ADR-008). See both ADRs in `decisions.md` for full interface signatures and implementation notes.

**Repository layer** (`internal/shared/repositories/`) — owns GORM queries, returns models:
- [x] `repositories/user/user_repository.go`
- [x] `repositories/address/address_repository.go`
- [x] `repositories/product/product_repository.go`
- [x] `repositories/category/category_repository.go`
- [x] `repositories/review/review_repository.go`
- [x] `repositories/order/order_repository.go`
- [x] `repositories/payment/payment_repository.go`

**Note — `Save` method primary key retention:**
GORM mutates the pointer passed to `Save` in place — the generated primary key is written back onto the struct automatically. No signature change needed. Callers just need to hold onto the pointer they pass in and read the ID from it after `Save` returns. No action required — awareness only.

**Service layer** (`api/internal/services/`) — owns business logic, returns DTOs:
- [x] `services/address/address_service.go`
- [x] `services/category/category_service.go`
- [x] `services/user/user_service.go`
- [x] `services/product/product_service.go`
- [x] `services/review/review_service.go`
- [x] `services/order-item/order_item_service.go`
- [x] `services/order/order_service.go`
- [x] `services/payment/payment_service.go`

**Repo additions required before services can be completed:**
- `user_repository.go` — add `GetByEmail(email string) (*models.User, error)` (needed by `UserService.Authenticate`)
- `order_repository.go` — add `GetByUserId(userId uint) ([]*models.Order, error)` (needed by `OrderService.GetByUserId`)

**Service design notes (feature/issue-26):**

`UserService` — interface: `GetById`, `GetAll`, `Save`, `Delete(id, hard)`, `Authenticate(email, password)`
- `Authenticate`: `repo.GetByEmail` → `model.CheckPassword(password)` → return `errors.New("invalid credentials")` if false

`ProductService` — interface: `GetById`, `GetAll`, `GetByCategory(categoryId)`, `Save`, `Delete(id, hard)`
- `GetByCategory` lives here (not CategoryService) — returns products; category is just a filter
- `GetByOrder` was considered and rejected — `OrderItem` DTO already carries the product info needed at order time; no need to re-fetch

`ReviewService` — interface: `GetById`, `GetByProductId`, `Save`, `Delete(id, hard)`
- `GetByProductId` returns `[]*dto.Review`

`OrderService` — interface: `GetById`, `GetByUserId`, `Save`, `Delete(id, hard)`, `UpdateStatus(id, status)`
- Injects both `OrderRepositoryI` and `OrderItemRepositoryI` (per CLAUDE.md)
- `UpdateStatus`: validate status string against `models.OrderStatus` consts before calling repo
- Valid statuses: `pending`, `shipped`, `delivered`, `cancelled`

`PaymentService` — interface: `GetById`, `GetByOrderId`, `Save`, `Delete(id, hard)`, `UpdateStatus(id, status)`
- `GetByOrderId` maps to `repo.GetByOrder`
- `UpdateStatus`: validate against `models.PaymentStatus` consts before calling repo
- Valid statuses: `pending`, `completed`, `authorized`, `captured`, `failed`, `refunded`, `partially_refunded`

**Consistency rules (follow address/category pattern):**
- Return `[]*dto.X` for slices
- Log errors with `slog.Error(...)` before returning
- Constructor returns the interface type
- Import alias: `userdto "commerce/api/internal/dto/user"`, `userrepo "commerce/internal/shared/repositories/user"` etc.

---

## Issue #38 — Payment model implementation

**Date:** 2026-02-26
**Status:** Done
**Branch:** `feature/issue-9`
**GitHub Issue:** #9

Designing and implementing the `Payment` entity as per ADR-007. Model lives in `internal/shared/models/payment.go` and must be registered in `internal/shared/database/setup.go`.

**Scope:**
- [x] `Payment` model with all fields from ADR-007
- [x] Register model for GORM AutoMigrate
- [x] Update `Order` model if needed (e.g., `Payments []Payment` association)

See ADR-007 in `decisions.md` for full field list and rationale.

---

## Issue #37 — ADR-003 embed fix

**Status:** Done
**Branch:** `feature/issue-37`

Resolved three bugs related to the `//go:embed` config setup (see BUG-002, BUG-003):

1. `config_manager.go` had `var content embed.FS` with no `//go:embed` directive — FS was always empty.
2. Embed responsibility was refactored: `NewDbConfig` now accepts `[]byte`; file reading and embedding moved to `utils/main.go`.
3. In `main.go`, the `//go:embed` directive was attached to `var _ embed.FS` (blank identifier) instead of `var content embed.FS` — fixed by moving the directive to the correct variable.
4. Fixed fallback logic: env var path now returns `nil` error so the caller can proceed.
5. Restored `utils/configs/config.json` as the canonical config location; updated `.gitignore` to match.

---

## Issue #34 — (merged)

**Branch:** `feature/issue-33`
**Merged commit:** `82a534f`
**Status:** Done

---

## Issue #33 — (merged)

**Status:** Done
**Notes:** Readme update included (`5c69c89`), config file removed (`109803b`).
