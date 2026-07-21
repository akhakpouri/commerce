# CLAUDE.md

This file provides guidance to Claude Code with respect to the `internal/shared` directory.

## Overview & Purpose
Shared library used by both `api` and `utils` modules. Contains all GORM models and the database connection/migration logic. All external dependencies (GORM, PostgreSQL driver, bcrypt) are pinned here.

## Packages

### `database`
**Thin shim over `github.com/akhakpouri/gorm-kit`** (issue #127, ADR-015 amendment). Connection/DSN logic now lives in that external module, not here.
- `Migrate(cfg database.DbConfig)` — delegates to `pg.Connect(cfg)` then gorm-kit's driver-agnostic `database.Migrate(db, models...)`. The **model registration list is inlined in `main.go`** (the old `setup.go` was removed) — that's the only place to register new models for migration. Used by `utils` only.
- To **connect** (no migrate), call `pg.Connect(cfg)` from `gorm-kit/pg` directly — that's what `api` does. There is no local `Connect` anymore.
- `DbConfig` — gorm-kit's type (`github.com/akhakpouri/gorm-kit/database`); same fields, all dynamic including `Schema`, `Port` is `int`. The old local `db_config.go` was removed.
- Deps: `gorm.io/driver/postgres` is pulled transitively via `gorm-kit/pg` (no longer a direct dep); `gorm.io/gorm` stays direct (models/repos use `*gorm.DB`).

### `models`
Eight domain models, all embedding `Base`:

- `Base` — `Id uint` (auto-increment PK), `CreatedDate`, `UpdatedDate`, `DeletedDate` (all `time.Time`)
- **Important:** `DeletedDate` is `time.Time`, NOT `gorm.DeletedAt` — GORM does not auto-filter soft-deleted records
- Every model implements `TableName() string` to explicitly set the table name
- Full relationship diagram: see `docs/project-notes/facts.md`

**Notable model behaviour:**
- `User` — `BeforeCreate`/`BeforeUpdate` hooks auto-bcrypt the `Password` field; `CheckPassword(string) bool` for verification
- `Category` — self-referential via `ParentId *uint` (nullable); supports unlimited-depth tree
- `Order` — uses string enum type `OrderStatus` defined in `order.go`; payment state is read via `Payments []Payment` association (see ADR-007)

### `repositories`
One sub-package per domain. Each defines an interface (`XxxRepositoryI`) and a concrete struct (`XxxRepository`). Constructor takes `*gorm.DB` and returns the interface.

### `configs`
Shared config primitives used by `api/configs` and `apps/relay/configs` (both build their own `Config` on top of this rather than duplicating fields). `GetEnvOrPanic`/`GetEnvOrDefault` helpers; `DatabaseConfig` (`Connect()` delegates to `gorm-kit/pg.Connect`, same as the `database` package shim, just callable straight from a config struct); `AWSConfig`; `ConsumerConfig` (see `aws` below — `Validate()` applies defaults and enforces two invariants the `Consumer` code depends on: `Url` must be non-empty — panics otherwise — and `Timeout` must be `> 5` — floored to 30 if not, because `Consumer.process()` derives the handler's deadline as `Timeout - 5`).

### `aws`
SQS producer/consumer toolkit (added 2026-07, user-designed — not scaffolded from a template). **Wired into `apps/relay`** as of 2026-07-21 — `OutboxService.ProcessBatch` publishes through `apps/relay/internal/publisher.SqsPublisher` (`Producer.SendBatch`), inside the same transaction as the outbox claim/mark. This is a direct-SQS producer; per the ADR-018 amendment (2026-07-21, `docs/project-notes/decisions.md`), the *resolved* design is relay → **SNS** topic → per-consumer SQS, so `SqsPublisher` is expected to be replaced with an SNS publisher before this goes live — tracked in `docs/project-notes/issues.md` #130. The `Consumer`/`ConsumerManager` side of this toolkit remains unwired (producer-only scope for `apps/relay`; consumers are separate apps).

- `NewSqsClient(ctx, *configs.AWSConfig)` — client factory. `AccessKeyID`/`SecretAccessKey` install a static credentials provider **only if both are non-empty**; otherwise the AWS SDK's default credential chain takes over (IAM role, `~/.aws/credentials`, `AWS_PROFILE`). `Endpoint` overrides `BaseEndpoint` for LocalStack **only if set** — leave it unset to hit real AWS. Callers must pass through empty strings, not placeholder defaults, or this fallback never triggers (see bugs.md BUG-027).
- `Producer` — `Send` (single message), `SendFIFOMessage`, `SendBatch` (hard-capped at 10 per SQS's own limit). Message attribute `DataType` must be exactly `"String"`/`"Number"`/`"Binary"` — case-sensitive, SQS rejects anything else (bugs.md BUG-026).
- `Consumer` — worker-pool pattern: one `poll()` goroutine long-polls `ReceiveMessage` and fans results into a buffered channel; `Count` `worker()` goroutines drain it concurrently. `Start(ctx)` blocks until `ctx` is canceled, then waits for in-flight workers via `sync.WaitGroup` before returning — mirrors the `signal.NotifyContext` shutdown shape in `apps/relay`.
- `QueueMonitor` — `GetQueueStats`/`HealthCheck` via `GetQueueAttributes` (queue depth, in-flight count, delayed count).
- `VisibilityExtender` — `StartVisibilityHeartbeat` periodically renews a message's visibility timeout so a still-working consumer isn't raced by a second one picking up the same message after the timeout lapses (lease-renewal pattern, same idea as Kafka consumer heartbeats / distributed-lock lease extension). **Not connected to `Consumer.process()` yet**, and has three known gaps before it should be relied on: no per-message start/stop lifecycle (needs its own `context.WithCancel`, canceled the instant the handler returns), no retry on a single failed `ExtendVisibility` call (one transient error silently ends renewal for the rest of that message), and no cap on total extension time (a hung, not-crashed handler would renew forever, making that message permanently unrecoverable). Track in `docs/project-notes/issues.md` #130.

```
repositories/
├── user/user_repository.go
├── address/address_repository.go
├── product/product_repository.go
├── category/category_repository.go
├── review/review_repository.go
├── order/order_repository.go
├── order_item/order_item_repository.go
└── payment/payment_repository.go
```

**Soft-delete:** `DeletedDate` is `time.Time` — repos manually set it and call `Save`. GORM's built-in soft-delete filtering does NOT apply.

**Hard-delete:** pass `hard: true` to `Delete` — executes a permanent `DELETE`.

## Adding Dependencies
```bash
cd internal/shared
go get gorm.io/gorm gorm.io/driver/postgres
```