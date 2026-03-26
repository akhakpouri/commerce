# Work Log

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
