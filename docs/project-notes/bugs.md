# Bug Log

## BUG-001 — `getPortFromEnv()` returns scan count instead of port number

**File:** `utils/internal/managers/config_manager.go`
**Discovered:** 2026-02-25
**Status:** Fixed

### Description
`fmt.Sscanf` returns the number of successfully scanned items, not the scanned value. The code was scanning into the `port` string variable (instead of an `int`) and returning `p` (always `1` on success), so `DB_PORT` env var was effectively ignored — port always resolved to `1`.

### Buggy Code
```go
p, err := fmt.Sscanf(port, "%d", &port) // scans into string, p = count
return p                                  // returns 1, not the port
```

### Fix
Replaced with `strconv.Atoi` which directly parses a string to int:
```go
p, err := strconv.Atoi(port)
return p
```

---

## BUG-002 — Missing `//go:embed` directive; `embed.FS` always empty

**File:** `utils/internal/managers/config_manager.go` (later refactored to `utils/main.go`)
**Discovered:** 2026-02-25
**Status:** Fixed

### Description
`var content embed.FS` was declared with no `//go:embed` directive. Go requires the directive on the line immediately preceding the variable. Without it the FS is empty and `content.ReadFile(...)` always returns "file does not exist".

### Fix
Added `//go:embed configs/config.json` directly above `var content embed.FS` in `utils/main.go` (embed responsibility moved there during refactor).

---

## BUG-003 — `//go:embed` directive bound to blank identifier `_`

**File:** `utils/main.go`
**Discovered:** 2026-02-25
**Status:** Fixed

### Description
```go
var content embed.FS        // no directive — always empty

//go:embed configs/config.json
var _ embed.FS              // directive discarded; _ is never read
```
The directive was attached to the wrong variable. `content` remained empty; `ReadFile` returned `*errors.errorString {s: "file does not exist"}`.

### Fix
```go
//go:embed configs/config.json
var content embed.FS
```

---

## BUG-004 — Wrong package name in `product/product.go`

**File:** `api/internal/dto/product/product.go`
**Discovered:** 2026-02-26
**Status:** Fixed

### Description
File declared `package dto` instead of `package product`. All other DTO sub-packages use their directory name as the package name. This causes an import conflict when callers import the package as `product`.

### Fix
Changed line 1 from `package dto` to `package product`.

---

## BUG-005 — Nil pointer dereference on `*time.Time` in `payment.FromModel`

**File:** `api/internal/dto/payment/payment.go`
**Discovered:** 2026-02-26
**Status:** Fixed

### Description
`payment.PaidAt.Format(...)` was called directly on a `*time.Time` field without a nil check. `PaidAt` is nullable — calling `.Format()` on a nil pointer panics at runtime.

### Fix
Wrapped in a nil check:
```go
PaidAt: func() string {
    if payment.PaidAt != nil {
        return payment.PaidAt.Format("01/02/2006 15:04:05")
    }
    return ""
}(),
```

---

## BUG-006 — Invalid Go time token `pm` in format string

**File:** `api/internal/dto/payment/payment.go`
**Discovered:** 2026-02-26
**Status:** Fixed

### Description
Format string `"01/02/2006 15:04pm"` was used. `pm` is not a valid Go time token — it is output as the literal string "pm". When combined with `15` (24-hour clock), this produces incorrect output like `"02/26/2026 14:04pm"`.

### Fix
Changed to `"01/02/2006 15:04:05"` (standard 24-hour format).

---

## BUG-007 — Format/parse layout mismatch causing silent nil on `PaidAt`

**File:** `api/internal/dto/payment/payment.go`
**Discovered:** 2026-02-26
**Status:** Fixed

### Description
`FromModel` formatted `PaidAt` with `"01/02/2006 15:04:05"` but `getTimeString` (used in `ToModel`) still used the old layout `"01/02/2006 15:04pm"`. `time.Parse` silently returns an error on mismatch, causing `getTimeString` to always return `nil` — `PaidAt` was never round-tripped correctly.

### Fix
Aligned both layouts to `"01/02/2006 15:04:05"`.

---

## BUG-008 — Duplicate `PaymentStatus` type in `order.go` and `payment.go`

**File:** `internal/shared/models/order.go`, `internal/shared/models/payment.go`
**Discovered:** 2026-02-26
**Status:** Fixed

### Description
`PaymentStatus` type and its constants were defined in both files. Go does not allow duplicate type definitions in the same package — compile error.

### Fix
Removed `PaymentStatus` from `order.go`. It now lives exclusively in `payment.go`. `Order` model references it from there. Also removed `Order.PaymentStatus` field; replaced with `Payments []Payment` association.

---

## BUG-009 — `GatewayTransactionId` marked `not null; unique` on `Payment`

**File:** `internal/shared/models/payment.go`
**Discovered:** 2026-02-26
**Status:** Fixed

### Description
`GatewayTransactionId` was tagged `gorm:"not null;unique"`. Failed and pending payments may not have a gateway transaction ID yet — `not null` would prevent inserting these rows.

### Fix
Removed `not null` constraint. Field is now nullable and unique only when populated.

---

## BUG-010 — Typo in `AddressRepository`: `GetByUsrerId` / `adress_repository.go`

**File:** `internal/shared/repositories/address/adress_repository.go`
**Discovered:** 2026-02-27
**Status:** Open

### Description
Two typos: the filename is `adress_repository.go` (missing an `d`) and the method is named `GetByUsrerId` (missing a `e`) in both the interface and implementation. Callers importing this package will reference the wrong name.

### Fix
Rename file to `address_repository.go`. Rename method to `GetByUserId` in both the interface and the implementation.

---

## BUG-011 — `Save` overwrites caller's data in address and category repos

**Files:** `internal/shared/repositories/address/adress_repository.go`, `internal/shared/repositories/category/category_repository.go`
**Discovered:** 2026-02-27
**Status:** Open

### Description
In both repos, the `Save` method passes `&address` (or `&category`) — a pointer-to-pointer — to `r.db.First(...)`. GORM scans the DB record into the struct the pointer points to, overwriting the caller's in-memory changes before `Save` is called. Updates become no-ops.

```go
// Buggy — fetches DB data into address, losing caller's changes
} else if err := r.db.First(&address, address.Id).Error; err != nil {
    return err
}
return r.db.Save(address).Error
```

### Fix
Use a separate variable to check existence without touching the caller's data:
```go
var existing models.Address
if err := r.db.First(&existing, address.Id).Error; err != nil {
    return err
}
return r.db.Save(address).Error
```

---

## BUG-012 — Read methods return soft-deleted records in address and category repos

**Files:** `internal/shared/repositories/address/adress_repository.go`, `internal/shared/repositories/category/category_repository.go`
**Discovered:** 2026-02-27
**Status:** Open

### Description
`GetById`, `GetAll`, `GetByUserId`, and `GetByParentId` do not filter on `deleted_date`. Because `Base.DeletedDate` is `time.Time` (not `gorm.DeletedAt`), GORM does not auto-filter soft-deleted records. All read queries return deleted records alongside active ones.

### Fix
Add `.Where("deleted_date = ?", time.Time{})` to every read query. Example:
```go
r.db.Where("deleted_date = ?", time.Time{}).First(&address, id)
r.db.Where("deleted_date = ?", time.Time{}).Find(&addresses)
```

---

## BUG-013 — `CategoryRepository.GetById` scans into `&category.Id` instead of `&category`

**File:** `internal/shared/repositories/category/category_repository.go`
**Discovered:** 2026-02-27
**Status:** Open

### Description
```go
r.db.First(&category.Id, id)
```
`&category.Id` is a `*uint`. GORM receives a scalar pointer instead of a struct pointer and cannot populate the full model. The returned `category` will have all fields at zero value except `Id`.

### Fix
```go
r.db.First(&category, id)
```

---

## BUG-015 — GORM `AutoMigrate` does not add constraints to existing tables

**Discovered:** 2026-03-10
**Status:** Known limitation

### Description
GORM's `AutoMigrate` only creates FK constraints when a table is first created. Adding `constraint:OnDelete:CASCADE` (or any constraint) to a model tag has no effect on tables that already exist in the database — the constraint is silently skipped.

### Workaround (dev)
Drop and recreate the schema, then re-run migrations:
```sql
DROP SCHEMA commerce CASCADE;
CREATE SCHEMA commerce AUTHORIZATION commerce;
```

### Fix (staging/prod)
Add constraints manually via `ALTER TABLE`:
```sql
ALTER TABLE commerce.order_items
  ADD CONSTRAINT fk_order_items_order
  FOREIGN KEY (order_id) REFERENCES commerce.orders(id) ON DELETE CASCADE;
```
Repeat for each relationship. A dedicated SQL migration script should be maintained for non-dev environments.

---

## BUG-014 — `CategoryRepository.Delete` soft branch performs a hard delete

**File:** `internal/shared/repositories/category/category_repository.go`
**Discovered:** 2026-02-27
**Status:** Open

### Description
```go
// intended as soft delete
return r.db.Delete(&models.Category{}, id).Error
// intended as hard delete
return r.db.Unscoped().Delete(&models.Category{}, id).Error
```
Because `Base.DeletedDate` is `time.Time` (not `gorm.DeletedAt`), GORM has no soft-delete awareness. `r.db.Delete(...)` issues a SQL `DELETE` statement regardless — both branches do a hard delete. `Unscoped()` is also a no-op here (it only bypasses `gorm.DeletedAt` filtering).

### Fix
Soft branch must manually set `deleted_date`:
```go
// soft
return r.db.Model(&models.Category{}).Where("id = ?", id).Update("deleted_date", time.Now()).Error
// hard
return r.db.Delete(&models.Category{}, id).Error
```

---

## BUG-015 — gopls shows stale red highlights after code generation

**Tool:** VS Code + gopls
**Discovered:** 2026-03-27
**Status:** Known — workaround documented

### Description
After running a code generation step (e.g. `swag init` generating `api/docs/`), gopls may continue to show red highlights in files that import the newly generated package. The build is clean and `golangci-lint` reports no issues — the error is purely a stale gopls cache.

### Symptom
Red highlights in handler or other files with no corresponding lint or build error. Typically triggered after generating the `docs` package or other `go generate` output.

### Fix
**VS Code command palette → `Go: Restart Language Server`**
If highlights persist: **`Developer: Reload Window`**

No code changes required.

---

## BUG-016 — Swaggo `ParseComment` error on `@Success` annotation

**Tool:** `swag init` (swaggo)
**Discovered:** 2026-03-31
**Status:** Known — rules documented

### Description
`swag init` throws `ParseComment error: can not parse response comment` when `@Success` annotations are malformed or placed incorrectly.

### Root causes (all three can trigger the same error)

**1. Missing status code** — status code is required, not optional:
```go
// @Success {array} dto.Tax       ← wrong
// @Success 200 {array} string    ← correct
```

**2. Annotation outside the contiguous comment block** — all swaggo tags must be in one unbroken `//` block. A misaligned line breaks the block:
```go
// @Router  /api/taxes [get]
//                              ← blank line breaks the block
// @Success 200 {array} string  ← swaggo doesn't see this
```

**3. Annotation on a non-handler function** — swaggo only processes functions with `*gin.Context` signature. Placing annotations on `RegisterRoutes(*gin.RouterGroup)` is silently ignored or causes parse errors. Always annotate the named handler method, not `RegisterRoutes`.

### Fix
- Ensure `@Success` format is: `@Success <status> {type} <model>`
- Keep all annotations in one contiguous comment block with no blank lines
- Place annotations on `func (h *XxxHandler) MethodName(c *gin.Context)` only

---

## BUG-017 — Swaggo `cannot find type definition` for DTO types

**Tool:** `swag init` (swaggo)
**Discovered:** 2026-03-31
**Status:** Known — fix documented

### Description
`swag init` throws `cannot find type definition: dto.Product` (or similar) even when `--parseInternal` is set.

### Root cause
Swaggo resolves annotation types (`dto.Product`) by looking for an import with that alias in the current file. If the handler doesn't explicitly import the DTO package — which is common when the service returns the DTO type and Go infers it — swaggo has no mapping from `dto` to a package path.

Simply adding the import isn't enough either: Go's compiler rejects unused imports, so the import must also be referenced in actual code.

### Fix
Import the DTO package AND explicitly declare the variable type in the handler so the import is used by the compiler:

```go
import dto "commerce/api/internal/dto/product"

func (h *ProductHandler) GetAll(c *gin.Context) {
    var products []*dto.Product
    var err error
    products, err = h.svc.GetAll()
    ...
}
```

`var products []*dto.Product` satisfies both requirements: the compiler sees the import as used, and swaggo can resolve `dto.Product` via the import alias.
