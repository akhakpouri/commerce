# CLAUDE.md

This file provides guidance to Claude Code with respect to the `api/internal` directory.

## Overview & Purpose
Application layer for the HTTP API. Contains three packages: `dto` (request/response shapes), `services` (business logic), and `handlers` (Gin HTTP layer). See ADR-004.

**Hard rule:** Services must never import or reference `gorm.io/gorm` directly. All DB access goes through repository interfaces injected at construction time.

---

## Packages

### `dto`
Thin data containers for API payloads. One sub-package per domain under `api/internal/dto/<name>/`.

**Responsibilities:**
- JSON shape (`json` tags)
- Input validation (`validate` tags)
- Mapping to/from models (`ToModel()` / `FromModel()`)

**No business logic in DTOs.** See ADR-008.

**Mapping convention:**
- `ToModel()` — method on request DTOs, returns a model
- `FromModel(...)` — standalone function on response DTOs, accepts a model

**Exception:** `dto/tax/tax.go` has no backing model — it is a plain data container used only by `TaxService`. No `ToModel()`/`FromModel()` required.

**Structure:**
```
dto/
├── user/user.go
├── address/address.go
├── product/product.go
├── category/category.go
├── review/review.go
├── order/order.go
├── order-item/order_item.go
├── payment/payment.go
└── tax/tax.go
```

---

### `services`
Business logic layer. One sub-package per domain under `api/internal/services/<name>/`.

**Responsibilities:**
- Enforce business rules (e.g. order must exist before payment, status validation)
- Orchestrate repository calls
- Map between models and DTOs
- Return DTOs to callers — never raw models

**Pattern:** Each file defines an interface (`XxxServiceI`) and a concrete struct (`XxxService`) that implements it.

**Constructor:** takes one or more repository interfaces, returns the service interface:
```go
func NewXxxService(repo xrepo.XxxRepositoryI) XxxServiceI {
    return &XxxService{repo: repo}
}
```

**Exception:** `TaxService` has no repository — it operates entirely on an in-memory map. Its constructor takes no parameters: `func NewTaxService() TaxServiceI`.

**Import aliasing** — service, repo, and DTO packages share the same domain name. Alias at the import site:
```go
import (
    userdto  "commerce/api/internal/dto/user"
    userrepo "commerce/internal/shared/repositories/user"
)
```

**Structure:**
```
services/
├── user/user_service.go
├── address/address_service.go
├── product/product_service.go
├── category/category_service.go
├── review/review_service.go
├── order-item/order_item_service.go
├── order/order_service.go
├── payment/payment_service.go
└── tax/tax_service.go
```

**Notable implementation rules (from ADR-008, ADR-013):**
- `UserService.Authenticate` — fetch by email, call `model.CheckPassword(password)`, return error if false
- `AddressService.SetDefault` — clear existing default for user, then set new one
- `OrderService.Save` — atomicity is achieved via GORM's association create: pass the full `models.Order` with nested `[]OrderItem` to `repo.Save`; GORM inserts both in a single transaction. No manual transaction needed.
- `OrderService.Save` — before persisting, compute `SubTotalAmount = Σ (quantity × unit_price)`, call `TaxService.Calculate` for `TaxAmount`, set `TotalAmount = SubTotalAmount + TaxAmount`. Inject `TaxServiceI` alongside `OrderRepositoryI`.
- `OrderService.UpdateStatus` / `PaymentService.UpdateStatus` — validate input string against model enum constants before calling repo

---

### `handlers`
Gin HTTP layer. One sub-package per domain under `api/internal/handlers/<name>/`.

**Pattern:**
```go
type ProductHandler struct { svc ProductServiceI }

func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
    rg.GET("/", h.GetAll)
    rg.GET("/:id", h.GetById)
    // ...
}

func (h *ProductHandler) GetAll(c *gin.Context) { ... }
func (h *ProductHandler) GetById(c *gin.Context) { ... }
```

**Rules:**
- `RegisterRoutes` is a pure routing table — no logic, just wires methods to routes
- Each endpoint is a named method with `*gin.Context` signature — never inline in `RegisterRoutes`
- Always `return` after `c.JSON(...)` on error paths — execution falls through otherwise
- Path params via `c.Param("id")` — always a string, parse with `strconv.ParseUint` for `uint`

**Swagger annotation rules:**
- Annotations must be on methods with `*gin.Context` signature — swaggo ignores `RegisterRoutes`
- All annotations must be in one contiguous comment block (no blank lines between them)
- `@Success` format is always: `@Success <status> {type} <model>` — status code is required
- Type tokens: `{object}` for a single struct, `{array}` for a slice
- The model type must be imported AND used in actual Go code — swaggo resolves types via import aliases; Go rejects unused imports. Fix: explicitly declare the variable type (`var products []*dto.Product`) so the import is used by both the compiler and swaggo (see BUG-017)
- `GetStates` returns `[]string` — annotate as `{array} string`, not `{array} dto.Tax`
- `@Router` path params use OpenAPI syntax `{id}`, NOT Gin's `/:id` — e.g. `@Router /api/products/{id} [get]`; Gin route registration still uses `/:id`
- Optional boolean flags belong as query params, not path wildcards — use `c.DefaultQuery("hard", "false") == "true"` and annotate with `@Param hard query bool false "..."`
- Regenerate after any annotation change: `(cd api && go generate ./...)`

**Structure:**
```
handlers/
├── tax/tax_handler.go
├── product/product_handler.go
└── ...
```

---

## Key ADRs
| ADR | Title |
|-----|-------|
| ADR-004 | Gin as the HTTP framework |
| ADR-008 | Thin DTOs with service-layer mapping and business logic |
| ADR-009 | Repository pattern for data access |
| ADR-013 | Order amount calculation strategy (SubTotal, Tax, Total) |
| ADR-015 | Consolidated DB connection in `internal/shared/database` |

Full details in `docs/project-notes/decisions.md`.
