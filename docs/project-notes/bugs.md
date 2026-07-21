# Bug Log

## BUG-030 — Injected base-DB repo used instead of the tx-bound repo inside a UoW `Execute` callback

**Files:** `api/internal/services/order/order_service.go`, `apps/relay/internal/services/outbox/outbox_service.go`
**Discovered:** 2026-07 (code review, twice — same pattern in two services)
**Status:** Fixed (both)

### Description
`OrderService`/`OutboxService` hold both a constructor-injected repo (bound to the base `*gorm.DB`) and a `manager.ManagerI` (the Unit-of-Work transaction manager, `internal/shared/managers/transaction`). Inside `manager.Execute(func(r manager.RepositoriesI) error {...})`, the callback was written using the injected `o.repo`/`o.repo.GetNextBatch`/`o.repo.MarkPublished` field instead of the tx-bound `r.Order()`/`r.Outbox()` accessors the callback receives.

### Impact
- **`OrderService.Save`:** the order save happened outside the transaction the outbox-event write was supposed to share — reintroduces the dual-write problem the transactional outbox exists to prevent.
- **`OutboxService.ProcessBatch`:** `GetNextBatch`'s `SELECT ... FOR UPDATE SKIP LOCKED` ran as its own autocommit statement on the base connection — the row lock was acquired and released before `MarkPublished` ever ran, so `manager.Execute`'s transaction wrapped nothing. Two relay workers/replicas could then claim and publish the same rows.
- Both variants passed their existing unit tests, because the mocks aliased the injected repo and the tx-bound repo to the same fake — the bug was invisible to the test double.

### Fix
Always call through the callback's `r manager.RepositoriesI` parameter (`r.Order()`, `r.Outbox()`) for every read/write that must participate in the transaction; never the struct's own injected `repo` field once inside `Execute`.

### Prevention
When writing a `manager.Execute` callback, grep the closure body for the service's own `repo`/`o.repo` field — any hit there is very likely meant to be `r.X()` instead.

---

## BUG-029 — `wg.Add(i)` instead of `wg.Add(1)` in SQS `Consumer` worker pool

**File:** `internal/shared/aws/consumer.go`
**Discovered:** 2026-07-10 (code review)
**Status:** Fixed (by author, mid-review)

### Description
`Consumer.Start` spawns `c.count` worker goroutines in a loop and calls `wg.Add(i)` using the loop index instead of `wg.Add(1)`. Each goroutine still only calls `wg.Done()` once (one `defer` per goroutine), so the total `Add` sum (`0+1+...+(n-1) = n(n-1)/2`) only matches the total `Done` calls (`n`) when `n == 3` — coincidence, not correctness.

### Impact
- `count = 1`: `Add` sum is 0, one `Done()` call → **panic: negative WaitGroup counter**.
- `count = 2`: `Add` sum is 1, two `Done()` calls → same panic.
- `count >= 4`: `Add` sum overshoots the `Done()` calls → `wg.Wait()` never reaches zero → `Start()` hangs forever, so graceful shutdown never completes.

### Fix
```go
wg.Add(1)
```
inside the loop (or `wg.Add(c.count)` once, before it).

---

## BUG-028 — Nil entries in `Consumer.recive()` messages slice on unmarshal failure

**File:** `internal/shared/aws/consumer.go`
**Discovered:** 2026-07-10 (code review)
**Status:** Fixed

### Description
```go
messages := make([]*Message, len(result.Messages))
for i, sqsMsg := range result.Messages {
    var msg Message
    if err := json.Unmarshal([]byte(*sqsMsg.Body), &msg); err != nil {
        slog.Info("Error unmarshaling message", "id", *sqsMsg.MessageId, "error", err)
        continue
    }
    ...
    messages[i] = &msg
}
```
The slice is pre-sized with `len(result.Messages)` nil pointers. On unmarshal failure, `continue` skips the `messages[i] = &msg` assignment but leaves that index `nil` in the returned slice — it's never trimmed. Same bug class as the `ids := make([]uint, len(outboxes))` fix already applied in `apps/relay/internal/services/outbox/outbox_service.go`.

### Impact
Nil entries flow straight into `msgChan` with no nil-check anywhere in `poll`/`worker`/`process`, then into the caller's `Handler`. A handler dereferencing any field on a nil `*Message` panics — an unrecovered goroutine panic kills the whole process. One malformed SQS message body is enough to trigger it.

### Fix
Build with `append` instead of pre-sized index assignment:
```go
messages := make([]*Message, 0, len(result.Messages))
for _, sqsMsg := range result.Messages {
    ...
    messages = append(messages, &msg)
}
```

---

## BUG-027 — AWS config defaults in `apps/relay/configs/config.go` silently break non-local environments

**File:** `apps/relay/configs/config.go`
**Discovered:** 2026-07-10 (code review)
**Status:** Fixed

### Description
Three separate problems in the same config block:
1. `Endpoint` defaulted via `GetEnvOrDefault(..., "http://localhost:4566")`. Since the helper always returns non-empty, `internal/shared/aws/sqs.go`'s `if cfg.Endpoint != ""` guard was always true — the SQS client was **permanently pointed at LocalStack** unless something explicitly overrode it with a real endpoint URL. There's no way to "just leave it unset" and get real AWS.
2. `AccessKeyID` defaulted to the literal placeholder string `"your-access-key-id"` — a leftover LocalStack testing value baked into a code-level default.
3. `SecretAccessKey` was `GetEnvOrPanic` (required at startup). `sqs.go` already has a deliberate fallback — static credentials only `if cfg.AccessKeyID != "" && cfg.SecretAccessKey != ""`, otherwise the AWS SDK default credential chain (IAM role / `~/.aws/credentials` / `AWS_PROFILE`) takes over. Forcing `AWS_SECRET_ACCESS_KEY` to always be set defeats that fallback entirely.

### Fix
All three now default to `""` via `GetEnvOrDefault(key, "")`, except `Region` (kept `"us-east-1"` — a real, safe default, not a placeholder). Local/LocalStack testing sets real values explicitly in `configs/dev.env`; anywhere else, empty values fall through to the SDK's real defaults.

---

## BUG-026 — SQS message attribute `DataType` lowercase `"string"` instead of `"String"`

**File:** `internal/shared/aws/producer.go`
**Discovered:** 2026-07-10 (code review)
**Status:** Fixed

### Description
`Producer.Send`'s `MessageType` attribute used `DataType: aws_sdk.String("string")` (lowercase). SQS's `DataType` is case-sensitive and must be exactly `String`/`Number`/`Binary`. `SendBatch`, a few lines below, gets this right with capital `"String"` for the same semantic field — internal inconsistency was the tell.

### Impact
`Producer.Send` (the single-message path) would fail at the API with `InvalidParameterValue` the first time it was actually called.

### Fix
Capitalized to `"String"`.

---

## BUG-025 — `ConsumerConfig.Validate()` doesn't floor `Timeout` above 5

**File:** `internal/shared/configs/consumer_config.go`
**Discovered:** 2026-07-10 (code review)
**Status:** Fixed

### Description
`Validate()` only guarded `Timeout <= 0` (defaulting to 30). `Consumer.process()` (`internal/shared/aws/consumer.go`) derives the handler's local deadline as `context.WithTimeout(ctx, time.Duration(c.timeout-5)*time.Second)` — a 5-second reserve to call `delete()` before the SQS visibility timeout expires. A configured `Timeout` of 5 or less (a legitimate SQS visibility timeout value) produces a zero or negative duration, so `context.WithTimeout` creates an already-expired context — every message fails before the handler runs.

### Fix
```go
if cfg.Timeout <= 5 {
    cfg.Timeout = 30
}
```

---

## BUG-024 — `ConsumerConfig.Validate()` never checks `Url` is set

**File:** `internal/shared/configs/consumer_config.go`
**Discovered:** 2026-07-10 (code review)
**Status:** Fixed

### Description
`Validate()` applies defaults for `Count`/`Max`/`Timeout`/`WaitTime` but never checked `Url`. A missing queue URL wasn't caught at startup — it only surfaced later as an AWS API error the first time `Consumer.recive()` actually ran.

### Fix
```go
if cfg.Url == "" {
    panic("consumer queue URL is required")
}
```
Fails fast at construction, consistent with `GetEnvOrPanic`'s treatment of required config elsewhere in the codebase.

---

## BUG-023 — Empty JSON `{}` parses to zero-value `DbConfig` without triggering env var fallback

**File:** `utils/internal/managers/config_manager.go`
**Discovered:** 2026-04-20
**Status:** Fixed
**Branch:** feature/issue-99

### Description
The `utils` Dockerfile writes a dummy `{}` into `utils/configs/config.json` at build time so `//go:embed` succeeds (the real file is gitignored). The intent was for JSON parsing to fail at runtime, triggering `NewDbConfig`'s env var fallback path.

But `{}` is **valid JSON** — `json.Unmarshal([]byte("{}"), &cfg)` returns `nil` error and leaves `cfg` at its zero value. The fallback branch (which only fires on parse error) never ran. GORM then attempted to connect with an empty DSN, producing:

```
failed to connect to `user=appuser database=port=0`:
    hostname resolving error: lookup user=: no such host
```

### Root cause
The fallback logic treated "unmarshal failed" as a proxy for "no config". `json.Unmarshal` only fails on malformed JSON — an empty/zero-value payload parses cleanly. A zero-value struct is not the same as an absent config.

### Fix
After successful unmarshal, check for zero-value and fall back explicitly:

```go
if cfg == (database.DbConfig{}) {
    slog.Info("Error parsing config file, falling back...")
    return getConfigFromEnv(), nil
}
```

Fallback now triggers for both invalid-JSON and empty-JSON cases, which is what the Dockerized flow needs.

---

## BUG-022 — `.Order()` placed after `.Find()` in GORM query chains

**Files:** `category_repository.go`, `order_repository.go`, `product_repository.go`, `user_repository.go`

### Symptom
Multi-record queries returned results in non-deterministic order despite `.Order()` being present in the chain.

### Root cause
GORM executes the SQL query when `.Find()` is called. Any method chained **after** `.Find()` modifies the `*gorm.DB` session but has no effect on the already-executed query. The pattern `r.db.Find(&results).Order("created_date desc")` silently discards the ordering.

### Fix
Move `.Order()` before `.Find()` in the chain:

```go
// wrong — Order is ignored
r.db.Find(&results).Order("created_date desc")

// correct — Order is applied
r.db.Order("created_date desc").Find(&results)
```

### Scope
Four `GetAll` methods had this bug. Filtered queries (`.Where(...).Order(...).Find(...)`) in other repos were already correct because the `.Order()` was naturally placed in the middle of the chain before `.Find()`.

---

## BUG-021 — Leftover `var content embed.FS` in `config_manager.go`

**File:** `utils/internal/managers/config_manager.go`
**Discovered:** 2026-04-09 (via golangci-lint)
**Status:** Fixed

### Description
After the ADR-003 embed refactor (issue #37), the `//go:embed` directive and file-reading responsibility were moved to `utils/main.go`. The now-unused `var content embed.FS` declaration and the `embed` import were left behind in `config_manager.go`, causing an `unused` lint error.

### Fix
Removed `var content embed.FS` and the `"embed"` import from `config_manager.go`.

---

## BUG-020 — `omitempty` on required DTO fields silently drops values

**File:** `api/internal/dto/user/authenticate.go`
**Discovered:** 2026-04-06
**Status:** Fixed

### Description
`Password` was tagged `json:"password,omitempty"`. With `omitempty`, an empty string is omitted during JSON binding — a request with no password would silently pass through to the service layer with a zero-value `Password` field instead of being rejected.

### Fix
Remove `omitempty` from any field that is required for the operation. Use `omitempty` only on genuinely optional response fields.

---

## BUG-019 — Wrong import alias used in `address_handler.go` Save annotation

**File:** `api/internal/handlers/address/address_handler.go`
**Discovered:** 2026-04-03
**Status:** Fixed

### Description
The `SaveAddress` Swagger annotation referenced `errdto.ErrorResponse` in `@Failure` lines, but the import alias in the file is `err_dto`. This would cause a swaggo parse error on doc regeneration.

### Fix
Updated `@Failure` annotations to use `err_dto.ErrorResponse`.

---

## BUG-018 — Swagger `@Router` annotations used Gin path syntax instead of OpenAPI syntax

**Files:** `address_handler.go`, `product_handler.go`, `category_handler.go`
**Discovered:** 2026-04-02
**Status:** Fixed

### Description
`@Router` annotations were written using Gin's route parameter syntax (e.g. `/api/products/:id`) instead of the OpenAPI specification syntax (e.g. `/api/products/{id}`). Swaggo requires OpenAPI syntax — curly braces, not colons. The generated docs rendered path params incorrectly in Swagger UI.

### Fix
Replace `/:id` with `/{id}` in all `@Router` annotations. Gin route registration in `RegisterRoutes` is unaffected and still uses `/:id`.

---

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

### Edge case — handler with no local DTO variable
If the handler never needs a local variable of the DTO type (e.g. a GET that simply returns what the service gives back), the `var x []dto.X` trick isn't natural. The explicit typed declaration `var x []dto.X = h.svc.GetAll()` satisfies the compiler but triggers staticcheck ST1023 ("type can be inferred"). Fix: suppress with `//nolint:staticcheck`:

```go
var states []dto.Tax = h.svc.GetAll() //nolint:staticcheck
```

This is a deliberate trade-off — the explicit type keeps the import alive for swaggo; the nolint silences the redundant-type check.
