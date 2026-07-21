# Project Facts & Configuration

## Database

| Key      | Value           |
|----------|-----------------|
| Engine   | PostgreSQL 13+  |
| Database | `commerce`      |
| Schema   | `commerce`      |
| User     | `commerce`      |
| Password | `commerce@123`  |
| Host     | `localhost`     |
| Port     | `5432`          |
| SSL Mode | `disable`       |

Config file: `utils/configs/config.json` (gitignored; `config.example` committed as reference)
Embedded via `//go:embed configs/config.json` in `utils/main.go` and passed as `[]byte` to `managers.NewDbConfig`.

### Setup SQL
```sql
CREATE DATABASE commerce;
CREATE USER commerce WITH ENCRYPTED PASSWORD 'commerce@123';
GRANT ALL PRIVILEGES ON DATABASE commerce TO commerce;
\c commerce
CREATE SCHEMA commerce AUTHORIZATION commerce;
```

---

## Domain Models (all in `internal/shared/models`)

### Base (embedded by all models)

| Field         | Type        | Notes                                      |
|---------------|-------------|--------------------------------------------|
| `Id`          | `uint`      | Primary key, auto-increment                |
| `CreatedDate` | `time.Time` | Auto-set on insert                         |
| `UpdatedDate` | `time.Time` | Auto-set on update                         |
| `DeletedDate` | `time.Time` | Soft-delete marker — **not** `gorm.DeletedAt`; GORM does NOT auto-filter deleted records |

---

### Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                            User                                 │
│  Id · FirstName · LastName · Email · Password(bcrypt)           │
└────────┬───────────────────────┬──────────────────┬────────────┘
         │ 1:many                │ 1:many           │ 1:many
         ▼                       ▼                  ▼
  ┌─────────────┐         ┌────────────┐     ┌───────────┐
  │   Address   │◄────────│   Order    │     │  Review   │
  │─────────────│ shipping│────────────│     │───────────│
  │ UserId (FK) │ billing │ UserId(FK) │     │ UserId(FK)│
  │ Street      │         │ ShipAddr   │     │ ProductId │
  │ City        │         │ BillAddr   │     │ Rating    │
  │ State       │         │ OrderNum   │     │ Title     │
  │ PostalCode  │         │ Status     │     │ Comment   │
  │ Country     │         │ Payment    │     └─────┬─────┘
  │ IsDefault   │         │ SubTotal   │           │ many:1
  └─────────────┘         │ Tax·Total  │           ▼
                          └─────┬──────┘   ┌───────────────┐
                                │ 1:many   │    Product    │
                                ▼          │───────────────│
                         ┌────────────┐    │ Name · Price  │
                         │ OrderItem  │    │ Description   │
                         │────────────│    │ Sku (unique)  │
                         │ OrderId(FK)│    │ Stock         │
                         │ ProductId ─┼───►│ IsActive      │
                         │ Quantity   │    │ IsFeatured    │
                         │ UnitPrice  │    └───────┬───────┘
                         │ TaxAmount  │            │ 1:many (via junction)
                         └────────────┘            ▼
                                          ┌──────────────────┐
                                          │  ProductCategory │
                                          │──────────────────│
                                          │ ProductId (FK)   │
                                          │ CategoryId (FK)  │
                                          └────────┬─────────┘
                                                   │ many:1
                                                   ▼
                                          ┌──────────────────┐
                                          │    Category      │
                                          │──────────────────│
                                          │ Name · Slug      │
                                          │ Description      │
                                          │ ParentId (*uint) │◄─┐
                                          │ IsActive         │  │ self-ref
                                          │ Children []      │──┘ (tree)
                                          └──────────────────┘
```

---

### Relationship Summary

| From         | To              | Type          | FK field(s)                              |
|--------------|-----------------|---------------|------------------------------------------|
| `Address`    | `User`          | many:1        | `Address.UserId`                         |
| `Order`      | `User`          | many:1        | `Order.UserId`                           |
| `Order`      | `Address`       | many:1 (×2)   | `Order.ShippingAddressId`, `Order.BillingAddressId` |
| `Order`      | `OrderItem`     | 1:many        | `OrderItem.OrderId`                      |
| `OrderItem`  | `Product`       | many:1        | `OrderItem.ProductId`                    |
| `Review`     | `User`          | many:1        | `Review.UserId`                          |
| `Review`     | `Product`       | many:1        | `Review.ProductId`                       |
| `Product`    | `Category`      | many:many     | via `ProductCategory` junction           |
| `Category`   | `Category`      | self-ref tree | `Category.ParentId` (`*uint`, nullable)  |

---

### Model Notes

**User** (`users`)
- `BeforeCreate` hook: bcrypt-hashes `Password`; rejects empty password
- `BeforeUpdate` hook: re-hashes only if `Password` field changed
- `CheckPassword(string) bool` — bcrypt comparison helper
- `FullName() string` — concatenates `FirstName + LastName`

**Category** (`categories`)
- `ParentId *uint` is nullable — `nil` means root category
- Self-referential `Children []Category` enables an unlimited-depth tree

**Order** (`orders`)
- `Status OrderStatus` — enum: `pending`, `shipped`, `delivered`, `cancelled`
- `Payments []Payment` — 1:many association (replaces old `PaymentStatus` field)
- References `Address` twice (shipping + billing) via explicit FK fields

**Payment** (`payments`)
- `PaymentStatus` type lives here — referenced by `Order` via association, not a field
- Status enum: `pending`, `authorized`, `captured`, `failed`, `refunded`, `partially_refunded`
- `PaymentGateway` enum: `stripe`, `paypal`, `square`, `authorize_net`
- `PaymentMethod` enum: `credit_card`, `debit_card`, `paypal`, `bank_transfer`
- `GatewayTransactionId` — nullable (failed/pending payments may not have one)
- `PaidAt *time.Time` — nullable, only set when payment is captured

**ProductCategory** (`product_categories`)
- Pure junction table; carries its own `Base` (Id + timestamps)

---

---

## DTO Layer (`api/internal/dto/`, ADR-008)

- Each DTO lives in its own sub-package: `api/internal/dto/<name>/<name>.go`
- Package name must match directory name (e.g. `package product` not `package dto`)
- Each package exposes `FromModel(...)` and `ToModel(...)` only — no business logic
- Enums: `string(models.SomeEnum)` to convert to string; `models.SomeEnum(str)` to convert back
- `*time.Time` fields: always nil-check before calling `.Format()` or panic occurs
- Time format in use: `"01/02/2006 15:04:05"` — Go reference time, 24-hour clock
- Format and parse layouts must be identical — mismatch causes silent `nil` on parse

---

## Environment Variables

### `utils` — DB config fallback (when `config.json` parse fails)
`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `DB_SCHEMA`

### `api` — required at startup (loaded from `api/configs/dev.env` via godotenv)

| Key | Purpose |
|-----|---------|
| `ENV` | Environment name (e.g. `development`) |
| `SERVER_ADDRESS` | Bind address (e.g. `:8080`) |
| `CORS_ALLOWED_ORIGIN` | Exact-match allowed origin (e.g. `http://localhost:3000`) |
| `DB_HOST` | Database host |
| `DB_PORT` | Database port — parsed to `int` at startup; invalid value panics |
| `DB_USER` | Database user |
| `DB_PASSWORD` | Database password |
| `DB_NAME` | Database name |
| `DB_SSLMODE` | SSL mode (e.g. `disable`) |
| `DB_SCHEMA` | Schema name (e.g. `commerce`) |
| `AUTH_DOMAIN` | Auth0 tenant domain (e.g. `dev-y7vm6nwrj5uw2n2e.us.auth0.com`). Issuer URL is `https://<domain>/` (trailing slash); JWKS at `https://<domain>/.well-known/jwks.json`. |
| `AUTH_AUDIENCE` | Auth0 API audience identifier (e.g. `urn:commerce-api`). Tokens carry this in their `aud` claim. |

Config file: `api/configs/dev.env` — gitignored (contains credentials). `api/configs/dev.env.example` is committed as a reference. All keys are required; missing key panics at startup via `GetEnvOrPanic`.

`databaseConfig.Connect()` converts to `database.DbConfig` and delegates to `database.Connect()` in `internal/shared`. See ADR-015.

> **Note:** `dev.env` is loaded with a relative path — binary must be run from the `api/` directory. In production (container), env vars are injected directly and `dev.env` is not required — `NewConfig()` skips `godotenv.Load` when the file is absent.

---

## Postman Integration

Postman is the primary tool for API integration testing. The collection is tied to this git repo via Postman's Git integration and auto-generated from `api/docs/swagger.json`.

| Path | Purpose |
|------|---------|
| `api/docs/postman/collections/` | Request collections (generated from swagger) |
| `api/docs/postman/environments/` | Environment configs (non-secret vars) |
| `api/docs/postman/flows/` | Postman Flows |
| `api/docs/postman/globals/` | Global variables |
| `api/docs/postman/mocks/` | Mock server configs |
| `api/docs/postman/specs/` | Linked API spec snapshots |

**Secrets** are stored in the Postman Vault — never committed to the repo. A `.gitignore` under `api/docs/postman/` enforces this.

To keep the collection in sync: regenerate `swagger.json` after annotation changes (`cd api && go generate ./...`), then re-import the spec in Postman.

---

## Auth0 (ADR-017)

Tenant is managed in Terraform via the `auth0/auth0` provider — source of truth lives in the **`akhakpouri/iac-matrix`** repo (issue #6, landed 2026-05-05). Do not create or rename Auth0 resources from the dashboard; round-trip them through Terraform.

| Field | Value |
|-------|-------|
| Tenant domain | `dev-y7vm6nwrj5uw2n2e.us.auth0.com` |
| API audience | `urn:commerce-api` |
| Issuer (`iss` claim) | `https://dev-y7vm6nwrj5uw2n2e.us.auth0.com/` (trailing slash) |
| JWKS endpoint | `https://dev-y7vm6nwrj5uw2n2e.us.auth0.com/.well-known/jwks.json` |
| Signing alg | RS256 |
| Validation library | `github.com/auth0/go-jwt-middleware/v3` |

### Scopes (defined in iac-matrix Terraform)

```
category:read   category:write
orders:read     orders:write
payment:read    payment:write
products:read   products:write
reviews:read    reviews:write
users:read      users:write    users:delete
```

`users:delete` is the only delete-class scope. Given ADR-011 forbids hard-deletes via the API, it gates the soft-delete code path. Pluralization is intentionally inconsistent with the model names (`Category`, `Payment` are singular models but `category` / `payment` scopes are too); revisit before scaling.

### Route-to-scope policy (decided 2026-05-18 as part of #114)

Every route on the API requires a scope. There are no anonymous endpoints — a public-facing storefront still rides on a token (SPA's Auth0 client or a BFF M2M), just one granted only `:read` scopes.

| Route shape | Scope |
|-------------|-------|
| `GET <domain>` / `GET <domain>/:id` | `<domain>:read` |
| `POST` / `PATCH` / `PUT` / `DELETE` on `<domain>` | `<domain>:write` |
| `DELETE /api/user/:id` (soft-delete) | `users:delete` (only delete-class scope) |
| Any `address` route | `users:*` (no `address:*` scope exists; address lives under users) |
| `GET /api/category/:id/products` | `products:read` (leaf resource wins) |
| `GET /api/products/:id/reviews` | `reviews:read` (leaf resource wins) |
| `GET /api/users/:id/orders` | `orders:read` (leaf resource wins) |
| `GET /api/orders/:id/payments` | `payment:read` (leaf resource wins) |
| `GET /api/users/:id/addresses` | `users:read` (address has no own scope; rides under users) |

**Nested-route rule: leaf resource wins.** A route is scoped by the resource it returns, not by the resource it's mounted under. The exception is `address` routes, which always use `users:*` because no `address:*` scope exists.

Rationale: tightening public→scoped is a breaking change; loosening scoped→public is not. Default stricter.

**Public exceptions** (carve-outs from the "every route gets a scope" rule):

| Route | Why |
|-------|-----|
| `/api/tax/*` (all methods) | Pure reference data / utility computation — no identity dependence, no user-owned data. Useful for guest-checkout tax estimates. There is no `tax:*` scope in iac-matrix, so scoping it would require a Terraform change first. |

When adding a new public exception: it must be listed in this table with a one-line "Why" so the deviation is auditable. If the rationale doesn't survive scrutiny, default to scoping the route instead.

### M2M test client status

The auto-created Auth0 "Test Application" used to validate the middleware end-to-end on 2026-05-13 was **deleted** afterward. A proper M2M Application is not yet provisioned — when it lands, do it in iac-matrix (`auth0_client` + `auth0_client_grant` for scopes) rather than the dashboard.

Until then: local testing against scope-protected routes (Swagger UI, curl) will return `scope: []` (so 403) or 401, depending on whether you have a token at all. Don't waste time debugging — the missing M2M app is the cause.

### Debugging gotchas

Two cost-real-time issues encountered while landing #113. Recorded here so the next person doesn't re-spend the time:

1. **Swagger UI does not auto-prepend `Bearer `.** Swaggo emits OpenAPI 2.0, where our security scheme is `apiKey` (not `http bearer`). The Authorize input is sent verbatim as the `Authorization` header, so the user must type `Bearer <token>`, not just `<token>`. A bare token produces "Failed to validate JWT" with underlying error `jwt missing`. Fix would be migrating to swag v3 with a proper `http bearer` scheme; not blocking.
2. **`@Router` annotation paths are not validated against actual Gin routes.** Swaggo trusts whatever string you write. If the annotation says `/api/order/...` (singular) but `RegisterRoutes` mounts at `/api/orders` (plural), Swagger UI will silently call the wrong URL and you'll see 404s from a route the server has correctly registered. Diagnose by checking the browser Network tab against `[GIN-debug]` startup output. Fix is to align both sides; there's no built-in linter.

---

## Module Paths

| Module            | Go module name            |
|-------------------|---------------------------|
| `api`             | `commerce/api`            |
| `utils`           | `commerce/utils`          |
| `internal/shared` | `commerce/internal/shared`|

---

## Linter

Tool: `golangci-lint` — config at `.golangci.yml` (workspace root)
Enabled rules: `errcheck`, `ineffassign`, `unused`, `govet`, `staticcheck`

---

## Event-Driven Architecture (ADR-018) — design reference

> Status: **designed, not yet implemented.** Names below are the planned canonical identifiers.
> **Visual:** open `docs/project-notes/event-driven-architecture.drawio` in draw.io / diagrams.net (VS Code "Draw.io Integration" extension renders it inline).

### Topology (one line)

`api` (writes outbox in order txn) → `relay` (outbox → SNS) → SNS topic → per-consumer SQS queue (filtered) → worker app → side effect. At-least-once; consumers dedupe on `event_id`.

### Planned applications (Go workspace modules, siblings to `api`/`utils`)

| Module | Role | Compute | AWS perms |
|--------|------|---------|-----------|
| `relay` | Drains `commerce.outbox` → SNS. `FOR UPDATE SKIP LOCKED`, N autonomous workers (per-worker tx, see below). Safe at 1 or N replicas. | ECS Fargate | `sns:Publish` on topic |
| `notifier` | Consumes notifications queue → SES email. | ECS Fargate | `sqs:Receive`/`Delete`/`GetQueueAttributes` + `ses:SendEmail` |
| `shipping` *(later)* | Consumes shipping queue → status transition. | ECS Fargate | `sqs:*` on its queue |
| `apps/janitor` | Prunes published `outbox` rows older than 7d, in batches. | **AWS Lambda**, EventBridge-scheduled daily | `secretsmanager:GetSecretValue` + VPC/ENI; **no SNS/SQS** |

ECS workers each get: own `go.mod` (`go 1.26.5`), `docker/<name>/Dockerfile`, ECR repo `commerce-<name>-registry` (IMMUTABLE, sha tags), entry in `publish-images.yml` matrix, and an entry in the CI `go work init` `use` list.

**`apps/janitor` diverges** (ADR-018 has the full list): Lambda not ECS → handler entrypoint (`aws-lambda-go`), VPC-attached for RDS reachability, packaged as a container image on `public.ecr.aws/lambda/provided:al2023` (recommended, reuses ECR/OIDC) or a `bootstrap` zip. Still a Go workspace module (add to `go.work` + CI use-list). `apps/` is a new path convention — open sub-decision whether other workers join it.

### AWS resources (owned by `matrix`, `aws/commerce/`)

| Resource | Canonical name |
|----------|----------------|
| SNS topic (single bus) | `commerce-domain-events` |
| SQS queue (notifier) | `commerce-notifications-queue` |
| SQS DLQ (notifier) | `commerce-notifications-dlq` (redrive `maxReceiveCount = 5`) |
| Subscription | topic → notifications queue, **raw delivery on**, filter `{event_type: ["OrderPlaced"]}` |

### Event envelope (`internal/shared/events/`)

`event_id` (uuid, idempotency key) · `event_type` (string, = SNS message attribute) · `occurred_at` · `aggregate {type,id}` · `version` (int) · `payload` (json). Same shape in outbox and on the wire.

### Outbox table — `commerce.outbox`

| column | type | note |
|--------|------|------|
| `id` | bigserial PK | relay order key |
| `event_id` | uuid unique | dedupe/idempotency |
| `event_type` | varchar | SNS attribute |
| `aggregate_type` / `aggregate_id` | varchar / bigint | tracing |
| `payload` | jsonb | the envelope |
| `created_at` | timestamptz | |
| `published_at` | timestamptz **null** | NULL = unpublished (relay's queue) |
| `attempts` | int | bump on publish failure |

Partial index: `WHERE published_at IS NULL`. Registered as a GORM model and migrated by `utils` (added to the `database` shim's migrate list).

### Non-negotiables

1. Outbox INSERT is in the **same `db.Transaction`** as the state change (no dual-write).
2. Consumers are **idempotent** — dedupe on `event_id` before the side effect (SNS→SQS is at-least-once).
3. Relay poll uses `FOR UPDATE SKIP LOCKED` (safe horizontal scaling, no double-publish).
4. Relay publishes to SNS **before** `UPDATE published_at`/commit — at-least-once; reversing it silently loses events (ADR-018 amendment 2026-07-01).
5. **Cross-service ordering dependencies are event chains, not shared subscriptions.** If consumer B must act only after consumer A succeeds, B subscribes to a *new* event A publishes on success (e.g. `OrderPaid`), not to the same event A consumes. (ADR-018 amendment 2026-07-21.)

### Relay concurrency — "Model B" (ADR-018 amendment 2026-07-01, with #130)

N **autonomous** worker goroutines, **each with its own DB session + transaction** — no coordinator, no shared tx. Per worker, looped: `BEGIN → SELECT … WHERE published_at IS NULL ORDER BY id LIMIT <batch> FOR UPDATE SKIP LOCKED → publish each to SNS → UPDATE published_at (only SNS-accepted rows) → COMMIT`.

- `SKIP LOCKED` hands each claim a **disjoint** row set — same mechanism across goroutines *and* replicas; no leader election.
- **Two throttles, both configurable** (default `batch` 50): `batch` = rows per claim; **W** (worker count) = concurrent claims/publishes in flight.
- **Connection pool must be sized ≥ W** — each worker pins a connection for the whole claim→publish→commit window (incl. SNS latency). Undersize → starvation.
- Per-row publish failure → that row stays `published_at IS NULL` (bump `attempts`), retried next loop.
- **No ordering guarantee** (workers interleave). Fine while slices are order-independent; a per-aggregate-FIFO need reverses the choice (single partitioned claimer / SNS FIFO).
- Rejected "Model A" (1 coordinator + publish-only workers over a shared tx): `*gorm.DB` tx isn't concurrency-safe; tx-per-worker sidesteps it.

### Cross-service sequencing — event chaining, not shared fan-out (ADR-018 amendment 2026-07-21)

Example: payment must clear before shipping/notification act. Only **billing-service** subscribes to `OrderPlaced`. On success it publishes `OrderPaid` (own outbox + own relay-clone); on failure, `PaymentFailed`. **Shipping/notification subscribe to `OrderPaid`, not `OrderPlaced`** — they can't run early because the event doesn't exist until billing emits it. Fan-out (parallel, independent) stays the default for order-independent reactions; chaining is the deliberate exception when one consumer's output must gate another's input. Any service that both consumes and produces (billing, later others) needs its own outbox + relay pair, not a bare consumer.

### First slice

`OrderPlaced → email`: `OrderService.Save` emits one `OrderPlaced` outbox row → `relay` → `notifications` queue → `notifier` → SES. Later events clone the relay/notifier skeleton + a queue/filter; backbone unchanged.

---

## Observability (ADR-019) — design reference

> Status: **deferred** except Phase 0. Design issue #136; per-phase issues track the rest. Promote per phase as they close.

### Stack

OpenTelemetry instrumentation (Go apps) → **ADOT collector** (ECS sidecar) → CloudWatch (logs + metrics) + X-Ray (traces). Vendor-neutral — swappable backend later.

### Three pillars

| Pillar | What | How |
|--------|------|-----|
| Logs | Structured `slog` JSON; every line carries `correlation_id` + `event_id` | CloudWatch log groups (exist) |
| Metrics | API RED (rate/error/latency) + backbone: outbox lag, relay pub success/fail, SQS depth + `ApproximateAgeOfOldestMessage`, DLQ depth, consumer duration + success/fail, dedupe hits | CloudWatch EMF or OTel |
| Traces | One trace across the async flow via W3C `traceparent` in the envelope | OTel → X-Ray |

### Phases

- **Phase 0 (rides with #130, NOT deferred):** add `correlation_id` + `traceparent` to event envelope + `commerce.outbox` row; generate `correlation_id` at API edge. Carrier only — nothing reads it yet.
- **Phase 1:** structured `slog` schema + CloudWatch alarms (DLQ depth, outbox lag, queue age).
- **Phase 2:** RED metrics + per-app dashboards.
- **Phase 3:** full OTel distributed tracing.

### Non-negotiable

The envelope carrier (`correlation_id` + `traceparent`) must land **in #130** — retrofitting trace propagation across a live event stream means migrating in-flight events. Everything Phase 1+ is deferrable; the carrier is not.
