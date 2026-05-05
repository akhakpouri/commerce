# Work Log

## Issue #113 — Auth0 JWT validation middleware

**Date:** 2026-05-05
**Status:** In progress (paused mid-task)
**Branch:** `feature/issue-113`

First implementation slice of ADR-017. Wires Auth0 config into the API and stands up the JWT validation middleware skeleton. Unblocked by iac-matrix#6 (Auth0 tenant + API resource server landed via Terraform 2026-05-05).

- [x] `api/configs/dev.env.example` — `AUTH_DOMAIN`, `AUTH_AUDIENCE` keys added
- [x] `api/configs/dev.env` — local values populated (gitignored)
- [x] `api/internal/constants/constants.go` — `EnvKeys.AuthDomain`, `EnvKeys.AuthAudience` added to typed struct
- [x] `api/configs/config.go` — exported `AuthConfig` struct (`Domain`, `Audience`); wired into `Config` via `NewConfig()` + `GetEnvOrPanic`
- [x] `api/go.mod` — `github.com/auth0/go-jwt-middleware/v3` added (note: **v3**, not v2 as originally planned)
- [x] `api/internal/middleware/auth/claims.go` — stateless `Claim{Scope string}` type implementing v3's `validator.CustomClaims` interface (`Validate(ctx) error`); plus `HasScope(string) bool` helper using `strings.SplitSeq`
- [ ] `api/internal/middleware/auth/` — actual `gin.HandlerFunc`: bearer extraction, `validator.New(...)` with `jwks.NewCachingProvider` (5min TTL), validate `iss` (`https://<domain>/` — trailing slash) + `aud`, `c.Set(claims, *Claim)` on success, abort 401 on failure
- [ ] Unit tests — `httptest.Server` fake JWKS + RSA-signed test tokens; cases: valid / expired / wrong aud / wrong iss / bad sig / missing or malformed Authorization header
- [ ] Swagger `BearerAuth` security definition in `main.go`; regenerate `docs/`
- [ ] (Defer) Route wiring — applying middleware to specific routes is #114's job

**Open architectural nits to revisit:**
- `Claim.Validate` includes whitespace-format checks on the scope string. Auth0 will not return malformed scope strings, so these branches are effectively dead code — keep, drop, or downgrade to log-only is a style call.
- Style consistency in `configs/config.go`: `AuthConfig` is exported while `serverConfig`/`databaseConfig` are unexported. Either pattern is fine; pick one when the second consumer of `AuthConfig` lands.

---

## Issue #114 — Scope-check guard + per-route classification

**Date:** 2026-04-27 (opened) / pending
**Status:** Blocked on #113
**Branch:** —

Per-route guard that asserts the JWT has a required scope before the handler runs. Consumes the `*Claim` stashed on `*gin.Context` by #113.

- [ ] `RequireScope("orders:write")` helper in `api/internal/middleware/auth/`
- [ ] Per-route classification matrix in `routes.go` — public / user-auth / M2M-with-scope
- [ ] Reconcile `users:delete` scope with ADR-011 (only delete-class scope; likely gates soft-delete; consider rename to `users:deactivate` upstream in iac-matrix)
- [ ] Address pluralization inconsistency in scope names (`category`/`payment` singular vs others plural) — fix in iac-matrix Terraform before tokens are issued at scale
- [ ] Unit tests per ADR-014

---

## Issue #115 — Map Auth0 `sub` claim → domain `users` row

**Date:** 2026-04-27 (opened) / pending
**Status:** Blocked on #113
**Branch:** —

First-time login creates a `users` row keyed by Auth0 `sub`. Commerce profile fields (name, addresses, etc.) continue to live in `users` — Auth0 owns identity, this repo owns the domain user.

- [ ] Add `Auth0Sub string` column to `User` model in `internal/shared/models/user.go` (unique index)
- [ ] Lookup-or-create helper invoked from JWT middleware after successful validation
- [ ] Migration / AutoMigrate updates
- [ ] Unit tests per ADR-014

---

## Issue #116 — Deprecate `User.Password` + bcrypt hooks

**Date:** 2026-04-27 (opened) / pending — post-cutover follow-up
**Status:** Blocked on Auth0 cutover (after #113–#115)
**Branch:** —

Once Auth0 owns authentication, `User.Password` and the bcrypt `BeforeCreate` / `BeforeUpdate` hooks are dead weight. Remove them and supersede ADR-005.

- [ ] Drop `Password` field from `User` model
- [ ] Drop `BeforeCreate` / `BeforeUpdate` bcrypt hooks
- [ ] Drop `CheckPassword` method
- [ ] Drop `dto.Authenticate` and any handler/service code referencing it
- [ ] Migration to drop the `password` column
- [ ] Author ADR-018 (or amend ADR-005) marking ADR-005 superseded

---

## Issue #108 — ADR: authorization strategy (user JWT + OAuth 2.0 client credentials)

**Date:** 2026-04-22
**Status:** Closed — superseded by ADR-017 revision (2026-04-27); see #113–#116
**Branch:** —

Original prerequisite for #109/#110, which described a build-in-tree auth approach. Pivoted to managed IdP — see ADR-017 in `decisions.md`.

---

## Issue #109 — User authentication (JWT bearer tokens)

**Date:** 2026-04-22
**Status:** Closed — superseded by ADR-017 revision (2026-04-27); replaced by #113/#115
**Branch:** —

Originally would have built `/auth/login` and `/auth/register` in-tree against the existing `User` model + bcrypt. Auth0 owns these flows now (Universal Login).

---

## Issue #110 — OAuth 2.0 client credentials (M2M authorization)

**Date:** 2026-04-22
**Status:** Closed — superseded by ADR-017 revision (2026-04-27); replaced by #113/#114
**Branch:** —

Originally would have built `/oauth/token` with an `ApiClient` model + secret hashing in-tree. Auth0 M2M Applications cover this; no in-tree token issuer.

---

## Issue #99 — docker-compose.yaml for local development

**Date:** 2026-04-20
**Status:** In progress
**Branch:** feature/issue-99

Added `docker-compose.yaml` at the workspace root to build and run the `api` and `utils` containers with one command. Postgres is managed outside compose (see ADR-016 amendment).

- [x] `docker-compose.yaml` — `api` + `utils` services using `docker/api/Dockerfile` and `docker/utils/Dockerfile`
- [x] `api` uses `depends_on: utils` with `condition: service_completed_successfully` — waits for migrations to succeed before starting
- [x] Both services load config from root `.env` via `env_file`
- [x] `.env.example` committed as reference; `.env` gitignored
- [x] `utils/internal/managers/config_manager.go` — zero-value `DbConfig` now triggers env var fallback (see BUG-023)
- [ ] Verify end-to-end: `docker compose up` against externally-reachable Postgres
- [ ] Scrub example credentials in `.env.example` before committing

---

## Issue #98 — Dockerfile for utils migration runner

**Date:** 2026-04-15
**Status:** Done
**Branch:** feature/issue-98

Added `docker/utils/Dockerfile` — multi-stage build for the migration runner binary.

- [x] Multi-stage build: `golang:1.26-alpine` builder → `alpine:latest` runtime
- [x] Builder compiles from workspace root with `go.work` resolving all modules
- [x] Sibling module `api/go.mod` copied for workspace validation (no source)
- [x] Dummy `config.json` (`{}`) created during build so `//go:embed` succeeds — at runtime, JSON parse fails and `NewDbConfig` falls back to env vars (see #105 for future cleanup)
- [x] Runs as non-root user (`appuser`)
- [x] Image builds cleanly with `docker build -f docker/utils/Dockerfile .`

---

## Issue #97 — Dockerfile for api service

**Date:** 2026-04-14 → 2026-04-15
**Status:** Done
**Branch:** feature/issue-97

Added `docker/api/Dockerfile`, `.dockerignore`, `docker/CLAUDE.md`, and ADR-016.

- [x] Multi-stage build: `golang:1.26-alpine` builder → `alpine:latest` runtime
- [x] Sibling module `utils/go.mod` copied for workspace validation (no source)
- [x] Runs as non-root user (`appuser`)
- [x] `EXPOSE 8080` declared
- [x] `.dockerignore` at workspace root
- [x] `docker/CLAUDE.md` documenting build context, conventions, and layering strategy
- [x] ADR-016 — centralized Docker structure with workspace-root build context

---

## Issue #95 — Support env var config without dev.env file

**Date:** 2026-04-13
**Status:** Done
**Branch:** feature/issue-95

Removed the hard dependency on `api/configs/dev.env` so the binary starts correctly in containerized environments where env vars are injected at runtime (ECS Fargate / AWS Secrets Manager).

- [x] `configs/config.go` — `godotenv.Load` now wrapped in `os.Stat` check; only loads `dev.env` if the file exists; falls back to env vars silently
- [x] `api/configs/dev.env` — removed from git tracking (`git rm --cached`); added to `.gitignore`
- [x] `api/configs/dev.env.example` — committed as reference template with `DB_PASSWORD` left blank
- [x] `docs/project-notes/facts.md` — updated env vars section to reflect gitignored status and container fallback behaviour
- [x] `docs/project-notes/decisions.md` — updated ADR-004 config description and structure diagram; removed stale "known limitation" note

---

## Lint cleanup — all modules

**Date:** 2026-04-09
**Status:** Done

Ran `golangci-lint run ./...` across all three modules after all handlers were complete. Fixed 6 issues:

- [x] `address_handler.go` — `ineffassign`: `var address = &dto.Address{}` → `var address *dto.Address`
- [x] `category_handler.go` — `ineffassign`: `var products = []*product_dto.Product{}` → `var products []*product_dto.Product`
- [x] `tax_handler.go` — `ineffassign`: `states := []dto.Tax{}` + reassign → `var states []dto.Tax = h.svc.GetAll() //nolint:staticcheck` (explicit type required to keep `dto` import alive for swaggo — see BUG-017 edge case)
- [x] `order_handler.go` — `S1021`: merged `var statuses` declaration + assignment → `statuses := h.svc.GetStatuses()`
- [x] `payment_handler.go` — `S1021`: same merge
- [x] `config_manager.go` — `unused`: removed leftover `var content embed.FS` and `"embed"` import (see BUG-021)

---

## Issue #81 — Order handler

**Date:** 2026-04-09
**Status:** Done
**Branch:** feature/issue-81

Completed `OrderHandler` and wired the nested `/users/:user_id/orders` route.

- [x] `dto/order/order_status.go` — new `OrderStatus` DTO for `UpdateStatus` request body
- [x] `order_service.go` — added `GetStatuses() []dto.OrderStatus` to `OrderServiceI` and implementation
- [x] `order_handler.go` — `GetById`, `GetByUser`, `Save`, `Delete`, `GetStatuses`, `UpdateStatus` implemented with Swagger annotations
- [x] `order_handler.go` — `Delete` supports `?hard=true` query param consistent with other handlers
- [x] `routes.go` — `orderHandler.RegisterRoutes(api.Group("/orders"))` wired
- [x] `routes.go` — nested route `GET /api/users/:user_id/orders` wired to `orderHandler.GetByUser`

---

## Issue #84 — Review handler

**Date:** 2026-04-08
**Status:** Done
**Branch:** feature/issue-84

Completed `ReviewHandler` and wired the nested `/products/:id/reviews` route.

- [x] `review_handler.go` — `GetById`, `GetAllByProduct`, `Save`, `Delete` implemented with Swagger annotations
- [x] `review_handler.go` — `Delete` supports `?hard=true` query param consistent with other handlers
- [x] `routes.go` — nested route `GET /api/products/:id/reviews` wired to `reviewHandler.GetAllByProduct`; uses `:id` to avoid wildcard conflict with existing `GET /api/products/:id`

---

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
