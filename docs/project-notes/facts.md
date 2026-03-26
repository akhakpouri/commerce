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
| `DB_DRIVER` | Database driver (e.g. `postgres`) |
| `DB_SOURCE` | Full DSN connection string |

Config file: `api/configs/dev.env` — gitignored. All keys are required; missing key panics at startup via `GetEnvOrPanic`.

> **Note:** `dev.env` is loaded with a relative path — binary must be run from the `api/` directory. Known limitation, to be addressed before CI.

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
