# CLAUDE.md

This file provides guidance to Claude Code with respect to the `internal/shared` directory.

## Overview & Purpose
Shared library used by both `api` and `utils` modules. Contains all GORM models and the database connection/migration logic. All external dependencies (GORM, PostgreSQL driver, bcrypt) are pinned here.

## Packages

### `database`
- `Connect(cfg DbConfig) (*gorm.DB, error)` — builds DSN from `DbConfig` fields and opens a GORM+PostgreSQL connection. Used by both `api` and `utils` — single source of truth for DSN construction (ADR-015).
- `Migrate(cfg DbConfig)` — calls `Connect` internally, then runs `AutoMigrate` on all registered models. Used by `utils` only.
- `setup.go` — the only place to register new models for migration
- `DbConfig` — connection struct; all fields including `Schema` are dynamic (no hardcoded values). `Port` is `int`.

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