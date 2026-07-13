# Event-sourced Order aggregate — Postgres event store

**Date:** 2026-07-13
**Status:** Approved by user, ready for implementation planning
**Related:** ADR-020 (`docs/project-notes/decisions.md`), ADR-018 (transactional outbox), ADR-002 (GORM + PostgreSQL)

## Origin

Consolidated research on Auth0, DDD, the transactional outbox, and event sourcing — done across several external chat threads — was brought into a project session as fresh input (not pre-existing repo history; `docs/PROJECT_NOTES.md` and `examples/event_sourcing_demo.go` referenced in that research do not exist in this repo). Section 5 of that research ("Event Sourcing") was explicitly marked exploratory, with an open question: how event sourcing relates to the *existing* transactional outbox (ADR-018). This spec resolves that question by scoping the work to a standalone, isolated addition — not a replacement of anything live.

## Goal

Prove out an event-sourced persistence model for the `Order` aggregate — Postgres event store with optimistic concurrency, raise/mutate/rehydrate, and a schema-evolution/upcasting layer — as new, self-contained code. Explicitly **not** wiring it into the live `OrderService`, handlers, routes, or the outbox/SNS-SQS pipeline.

## Why not a full cutover

The live `Order` domain is substantial and already working: `models.Order`/`OrderItem` (GORM), `OrderRepositoryI`, `OrderService.Save` (computes subtotal/tax/total, persists `Order` + nested `OrderItems` via a GORM association, atomically emits an `OrderPlaced` outbox row through a transaction manager), DTOs, handlers, `orders:read`/`orders:write`-scoped routes, and existing unit tests/mocks (`api/internal/services/order/order_service_test.go`).

Concretely: `OrderServiceI.GetByUserId` lists all orders for a user. Pure event streams keyed by `stream_id` have no way to answer "all streams for user X" without a secondary index — that's exactly what a materialized-view/projection layer solves, and that work (along with outbox/SNS-SQS wiring) is explicitly out of scope here. Swapping `OrderService`'s persistence in this same pass would require solving the projection problem inline, which is a much bigger issue than "prove the event-sourcing pattern works."

## Package layout

`internal/shared/eventsourcing/` — new package. Generic infra and the `Order` aggregate live together in one package (not split into infra vs. domain sub-packages), matching the isolated scope of this work. Revisit the split if the package grows unwieldy once a second aggregate is event-sourced.

Rough file breakdown:
- `event.go` — `Event` (stored/loaded row shape) and `EventToAppend` (write-side; no `version`/`global_seq` yet — the store assigns those on `Append`)
- `store.go` — `EventStore` interface + Postgres implementation, `ErrConcurrencyConflict`
- `store_model.go` — GORM-mapped envelope struct for the events table. Lives here, not in `internal/shared/models/` — it's infra (no `Base`, immutable, append-only, no soft delete), not a domain model.
- `aggregate.go` — generic raise/mutate/rehydrate contract
- `upcaster.go` — registry keyed by `(event_type, schema_version)`
- `order.go` — `Order` aggregate: `OrderPlaced` + `OrderStatusChanged` events

**Migration:** the envelope struct is added to the existing model-registration list in `internal/shared/database/main.go` (same place `Outbox` is registered today) — `AutoMigrate` creates the table via the `utils` binary. No new migration mechanism needed.

## Events table schema

```
stream_id      uuid        not null
version        int         not null   -- 1-indexed, per-stream
event_type     varchar     not null
schema_version int         not null
payload        jsonb       not null
occurred_at    timestamp   not null
global_seq     bigint      autoincrement, PK
UNIQUE(stream_id, version)
```

`UNIQUE(stream_id, version)` is the entire optimistic-concurrency mechanism — enforced by Postgres, not application logic. `global_seq` gives future projections a single, gap-free position to poll forward from (unused by anything in this issue — `LoadSince` exists for that future consumer, not exercised end-to-end here).

## EventStore interface

```go
type EventStore interface {
    Append(ctx context.Context, streamId uuid.UUID, expectedVersion int, events []EventToAppend) error
    Load(ctx context.Context, streamId uuid.UUID) ([]Event, error)
    LoadSince(ctx context.Context, globalSeq int64, limit int) ([]Event, error)
}
```

- **`Append`** assigns `expectedVersion+1, expectedVersion+2, ...` to the batch and inserts in one DB transaction. A Postgres unique-violation (`23505`) on `(stream_id, version)` maps to a typed `ErrConcurrencyConflict`. The caller (aggregate-level save) reloads the aggregate from the store and **re-derives the command against fresh state** — never blind-resends the same event. This is the load-bearing nuance from the source research: retrying blindly reproduces whatever bug the conflict was protecting against (e.g. an invalid status transition slipping through); reload-and-re-validate is what actually prevents it.
- **`Load`** — full stream by `stream_id`, in version order, for rehydration.
- **`LoadSince`** — by `global_seq`, for future projections. Included now for interface completeness; no caller in this issue exercises it.

## Aggregate pattern

Generic contract: `raise(eventType, schemaVersion, payload)` appends to an in-memory uncommitted-events list *and* calls `mutate()` immediately (so aggregate invariants apply on the live path exactly as they will on replay). `rehydrate(events)` calls `mutate()` for each loaded (and upcasted) event without adding to the uncommitted list. **Same `mutate()`, both paths** — this is the core invariant from the source research: live commands and historical replay can never disagree because they go through identical code.

### Order aggregate events (scope for this issue)

- **`OrderPlaced`** — creation. Snapshotted line items (product id/name/price/qty *at order time*, per the DDD principle that `Order` owns immutable snapshots rather than live references into the `Product` aggregate — a later price change must never rewrite historical orders). Also carries billing state and the computed subtotal/tax/total.
- **`OrderStatusChanged`** — enforces the same valid-transition rules as today's `models.OrderStatus` (Pending → Shipped → Delivered, or → Cancelled). Invalid transitions are rejected inside `mutate()` *before* the event is raised, so they never reach the store.

`OrderStatusChanged` is included specifically so the concurrency-conflict test has a real business invariant to exercise — two concurrent status changes racing on the same stream, where reload-and-retry means re-validating the transition against the winner's new state — rather than a bare mechanical version-conflict check with nothing at stake.

## Upcasting

Registry keyed by `(event_type, schema_version)`, dispatched at the deserialization boundary — before `mutate()` ever sees the payload. One entry for this issue, reusing the exact worked example from the source research (already validated there): `OrderPlaced` schema_version 1 (`TotalCents`) → schema_version 2 (`TotalAmount`). Old rows are never rewritten; the transform runs on read.

## Testing strategy

No code in this repo below the service layer has ever been tested against a real database — services are unit-tested via gomock against repository interfaces (ADR-014), and nothing under `internal/shared` currently has any `*_test.go` files. This introduces a new test category. Two tiers:

1. **In-memory fake `EventStore`** (same interface) for aggregate-level tests — fast, no DB:
   - Rehydration correctness
   - Mixed-schema stream through the upcaster (one `OrderPlaced` stored as `TotalCents`, another as `TotalAmount`, both rehydrate to the same current shape)
   - Invalid status transitions rejected before raising
2. **Postgres-integration tests** for the real store — needs the actual `UNIQUE(stream_id, version)` constraint to prove anything, so a fake can't substitute:
   - Normal `Append`
   - Concurrency-conflict + reload-and-retry: two goroutines racing `OrderStatusChanged` on the same stream; second writer gets `ErrConcurrencyConflict`, reloads, re-validates its transition against the now-current state, and either succeeds or correctly rejects

## Explicit non-goals (this issue)

- Wiring the event store into `OrderService`, handlers, or routes
- Materialized views / projections (blocks `GetByUserId` and any list-style read)
- Outbox/SNS-SQS integration — no decision yet on whether the event store eventually feeds the outbox, replaces it, or the two stay parallel
- Snapshotting
- A second event-sourced aggregate (e.g. `Stock`, mentioned in the source research as the inventory/oversell scenario) — `Order` only, for this issue

## Consequences

- `Order` has two coexisting, disconnected persistence paths after this lands: the legacy GORM row via `OrderService` (live traffic), and the new isolated event-sourced aggregate (proven in tests, not reachable from the API). Deliberate, not an oversight.
- The projection gap is a named, known prerequisite for ever wiring this into live traffic — tracked as follow-up work, not solved here.
- A future ADR is needed before the event store and the outbox are connected.
