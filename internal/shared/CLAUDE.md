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