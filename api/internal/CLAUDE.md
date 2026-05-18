# CLAUDE.md

This file provides guidance to Claude Code with respect to the `api/internal` directory.

## Overview & Purpose
Application layer for the HTTP API. Contains four packages: `dto` (request/response shapes), `services` (business logic), `handlers` (Gin HTTP layer), and `auth` (Auth0 JWT validation + scope guards). See ADR-004, ADR-017.

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

**`dto/user/`** contains two types: `User` (response shape — used for reads/writes) and `Authenticate` (auth request — email + password only). Use `Authenticate` for login/auth endpoints, not `User`.

**Structure:**
```
dto/
├── user/user.go
├── user/authenticate.go
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

### `auth`
JWT validation against Auth0's JWKS + per-route scope enforcement. See ADR-017.

**Files (`api/internal/auth/`):**
- `validator.go` — `NewValidator(domain, audience)` builds the v3 `*validator.Validator` with `jwks.NewCachingProvider` (RS256, 5-min cache, issuer = `https://<domain>/` with trailing slash).
- `middleware.go` — `NewMiddleware(*validator.Validator)` wraps it in the v3 `JWTMiddleware`; `Gin(*JWTMiddleware)` adapts it to a `gin.HandlerFunc` that on success stashes an `*Identity` under `constants.ContextKeys.Identity`, on failure writes a JSON 401 and aborts.
- `claims.go` — `Claim{Scope string}` is the custom claims type (implements v3's `validator.CustomClaims`). `Validate(ctx)` rejects leading/trailing whitespace and double spaces; `HasScope(s)` does exact-match on space-split tokens.
- `identity.go` — `Identity{Subject, Scopes []string, ExpiresAt}` — what handlers retrieve from the context.
- `scope.go` — typed scope constants. Always reference `auth.Scopes.Orders.Read` instead of string literals — the package is the source of truth for scope spellings (which intentionally match the iac-matrix Terraform).

**Wiring pattern in `server/router/routes.go`:**
```go
v, _ := auth.NewValidator(cfg.Auth.Domain, cfg.Auth.Audience)
mw, _ := auth.NewMiddleware(v)
ginAuth := auth.Gin(mw)

authedApi := api.Group("", ginAuth)         // every route under this group requires a valid JWT
orderHandler.RegisterRoutes(authedApi.Group("/orders"))
```

**Per-route scope enforcement** — handlers attach `auth.RequireScope(...)` per route inside `RegisterRoutes`, NOT in the router:
```go
func (h *OrderHandler) RegisterRoutes(rg *gin.RouterGroup) {
    rg.GET("/:id",     auth.RequireScope(auth.Scopes.Orders.Read),  h.GetById)
    rg.POST("/",       auth.RequireScope(auth.Scopes.Orders.Write), h.Save)
    ...
}
```
This keeps the read/write classification next to the route definition rather than scattered. `RequireScope` returns 401 if no identity is present, 403 if the scope is missing.

**Policy: every route gets a scope.** There are no anonymous endpoints (one documented exception: `/api/tax/*` — see facts.md). `GET` uses the domain's `:read` scope, mutations use `:write`. `address` rides under `users:*` (no `address:*` scope). Nested resources use the parent's scope (`/users/:id/<nested>` → `users:read`, `/category/:id/products` → `category:read`). Full table + exceptions list in `docs/project-notes/facts.md` under "Route-to-scope policy".

**Reading identity inside a handler:**
```go
v, _ := c.Get(constants.ContextKeys.Identity)
id := v.(*auth.Identity)   // *Identity is guaranteed if RequireScope or just Gin() ran
```

**Swagger:** every endpoint behind `ginAuth` must declare `@Security BearerAuth` and document `@Failure 401` + `@Failure 403`. The security definition itself lives once in `api/main.go` (`@securityDefinitions.apikey BearerAuth`). After annotation changes regenerate with `(cd api && go generate ./...)`.

**Gotchas:**
- Auth0 dev tokens have `aud` and `iss` baked in at issue time — `iss` must exactly match `https://<domain>/` *including the trailing slash*.
- `Identity.Scopes` is parsed from the `scope` claim by whitespace split. M2M tokens with no granted scopes carry `scope: []` and every `RequireScope` check 403s — granting scopes is an Auth0-side change (M2M client → APIs → toggle scopes), not a code change.
- Swagger UI uses an OpenAPI 2.0 `apiKey` scheme (swaggo limitation) — the Authorize input must contain the literal string `Bearer <token>`. See `swagger_bearer_apikey_quirk` memory note.
- `@Router` paths in handler annotations are not cross-checked against actual `RouterGroup` prefixes — a typo silently produces a Swagger UI that calls the wrong URL. Always grep both sides after edits.

**Structure:**
```
auth/
├── validator.go
├── middleware.go
├── middleware_test.go
├── claims.go
├── claims_test.go
├── identity.go
└── scope.go
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
| ADR-017 | Authorization via Auth0 (managed IdP, no in-tree auth server) |

Full details in `docs/project-notes/decisions.md`.
