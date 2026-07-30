# Relay slice 0 — make the outbox relay deployable

**Date:** 2026-07-30
**Status:** Approved by user, ready for implementation planning
**Related:** ADR-018 (event-driven backbone) + its 2026-07-01 and 2026-07-21 amendments, ADR-019 (observability), ADR-020 (event sourcing), issue #130
**Amends:** ADR-018's 2026-07-21 amendment (SNS swap is deferred, not done "before `ProcessBatch` goes live")

## Goal

Finish issue #130 to the point where `relay` is a deployable, testable service: a real event envelope on the wire, a publisher behind an interface, correct partial-failure handling, a Dockerfile, and an image in the publish pipeline.

**This is the unblocking slice.** The payment processor (slice 1), notifier (slice 2), and inventory (slice 3) all sit behind a relay that can actually ship.

## Decisions carried in

These were settled during design and are not re-opened here:

1. **Event naming is dotted lowercase** — `<aggregate>.<past-tense-verb>`. `order.placed` exists; `order.paid`, `payment.failed` follow. Chosen over PascalCase partly because SNS filter policies support prefix matching, so `{"event_type": [{"prefix": "order."}]}` becomes available later.
2. **SNS is deferred.** `order.placed` has exactly one consumer by design — the 2026-07-21 chaining amendment states only billing-service subscribes to it. Fan-out happens on `order.paid`, published by the payment processor's *own* relay clone. This relay stays on direct-to-SQS.
3. **Full ADR-018 envelope now**, not later. Adding `aggregate` and `version` after consumers exist is a breaking change across every deployed consumer.
4. **The envelope is the message body**, not a payload nested inside `aws.Message`.
5. **Publish-only** — a `v*` tag builds and pushes the relay image; the ECS service is rolled manually for now.

## Already done (verified 2026-07-30)

- `apps/relay` is in CI — `go work init` includes it and both Build and Test steps run it (landed with the `apps/` move).
- `Outbox.JobId` removed; zero references remain.
- `Outbox.Version int` added with `default:1`.
- Queue renamed to `commerce-payments-queue` in `daemon.go:19`.

## 1. Event envelope — `internal/shared/events`

New package. **It must not import `models` or GORM.** Consumers (notifier, inventory, payment processor) import this package to *parse* messages; dragging `gorm.io/datatypes` into a consumer that never touches a database is wrong, and it would make the wire contract depend on the storage schema.

```go
package events

type Aggregate struct {
    Type string `json:"type"`
    Id   uint   `json:"id"`
}

type Envelope struct {
    EventId    uuid.UUID       `json:"event_id"`
    EventType  string          `json:"event_type"`
    OccurredAt time.Time       `json:"occurred_at"`
    Aggregate  Aggregate       `json:"aggregate"`
    Version    int             `json:"version"`
    Payload    json.RawMessage `json:"payload"`
}
```

**Mapping lives in `apps/relay/internal/dto/outbox`**, alongside the existing `FromModel`/`ToModel`, keeping `events` dependency-free. The payment processor's relay clone will duplicate one small mapping function — accepted, in exchange for a contract package consumers can import cleanly.

**`OccurredAt` comes from the outbox row's `Base.CreatedDate`, not `time.Now()`.** The current `ToMessage` stamps publish time, so a backed-up or retrying relay reports timestamps minutes after the domain transaction actually committed. `CreatedDate` (`autoCreateTime`) is the moment the order was placed, which is what every consumer and every trace actually wants.

`AggregateType` stays PascalCase (`"Order"`) — it names a Go type, not a filter-policy string, so it is deliberately not subject to the dotted-lowercase convention.

## 2. Publisher port

`OutboxService` currently holds a concrete `*publisher.SqsPublisher` (`outbox_service.go:25`), which is the direct reason relay has zero tests — `ProcessBatch` cannot run without real AWS.

Define the interface **where it is consumed** (Go idiom), in `apps/relay/internal/services/outbox`:

```go
type Publisher interface {
    Publish(ctx context.Context, envelopes []events.Envelope) (PublishResult, error)
}

type PublishResult struct {
    PublishedIds []uuid.UUID  // event ids the broker accepted
    FailedIds    []uuid.UUID  // event ids it rejected
}
```

The result is keyed by **event id, not outbox row id** — the publisher has no business knowing about database rows. `ProcessBatch` holds both (it claimed the rows) and does the mapping.

`error` is reserved for whole-call failures (network, auth, marshalling). A partial batch rejection is a successful call with a non-empty `FailedIds`, not an error.

ADR-018 point 2's "producer publishes through an interface" is finally satisfied by this, which makes the eventual SNS swap a new implementation plus one wiring line. That is a side benefit — the justification here is testability.

## 3. `ProcessBatch` — partial failures and the poison escape

Today any failure fails the whole batch, the transaction rolls back, and every row retries forever. `Producer.SendBatch` already computes `BatchResult{SuccessfulIds, Failed}` and `SqsPublisher.Publish` discards it by erroring whenever `len(result.Failed) > 0`. The data needed is already there, one layer down.

New shape, still inside the single `manager.Execute` transaction:

1. Claim up to `limit` rows (`FOR UPDATE SKIP LOCKED`, unchanged).
2. Map each to an `events.Envelope`.
3. `Publish` → `PublishResult`.
4. Build `eventId → rowId` from the claimed models.
5. `MarkPublished(successRowIds)` **and** `IncrementAttempts(failedRowIds)`.
6. Commit.

A whole-call `error` still rolls the transaction back and retries everything — correct, since nothing was accepted.

**Claim → publish → mark → commit ordering is unchanged and non-negotiable** (ADR-018, 2026-07-01 amendment): a crash in the publish→commit window re-publishes a duplicate, which consumers dedupe. The reverse order loses events silently.

**Poison escape:** `GetNextBatch` gains an `attempts < maxAttempts` filter so a permanently-failing row stops being re-claimed on every poll. Without it, `IncrementAttempts` only records the problem; the row still consumes a batch slot forever. Default `maxAttempts = 10`, configurable. A parked row is invisible until someone looks — log at `WARN` when a row is parked, and treat outbox-parked count as an ADR-019 metric when that lands.

## 4. Repository changes — `internal/shared/repositories/outbox`

```go
GetNextBatch(limit int, maxAttempts int) ([]*model.Outbox, error)  // + attempts < ? filter
IncrementAttempts(ids []uint) error                                // new
```

`IncrementAttempts` uses `gorm.Expr("attempts + 1")` — a read-modify-write would race with concurrent relay workers claiming disjoint rows.

`maxAttempts` is a relay config value (`configs.RelayConfig` or an env key alongside the existing interval/batch-size knobs), not a package constant — the same reasoning ADR-018's 2026-07-01 amendment applies to `batch` and worker count.

Out of scope but noted: `Get` and `GetAll` log errors then return `nil` error, so callers cannot detect failure. Pre-existing, unrelated to this slice, worth its own fix.

### Narrow `OutboxServiceI` while we're here

`OutboxServiceI` declares six methods. **`ProcessBatch` is the only one anything calls** — `relay_manager.go:48` is the sole consumer, and `Get`, `GetAll`, `GetNextBatch`, `MarkPublished`, and `Delete` are unused passthroughs to the repository (verified by grep).

This matters to the slice rather than being a drive-by cleanup: `GetNextBatch`'s signature change would otherwise have to be propagated through a service method with no callers, and `IncrementAttempts` would invite a matching passthrough for symmetry. Narrow the interface to:

```go
type OutboxServiceI interface {
    ProcessBatch(ctx context.Context, limit int) error
}
```

and delete the five passthrough implementations. `ProcessBatch` already reaches the repository through the transaction manager (`r.Outbox()`), so nothing it needs is lost.

## 5. Send path — `internal/shared/aws`

The envelope becomes the SQS body, so `Producer`'s send path changes from marshalling a `*Message` to accepting a pre-marshalled body.

**This is a signature change, not a new parallel API:** `SendBatch` has exactly one caller (`SqsPublisher`), and `Send` has **none**. Verified by grep. Changing the shape is cheaper than maintaining two paths.

```go
type RawMessage struct {
    Id         string            // batch-entry correlation token — the event id
    Body       string            // marshalled events.Envelope
    Attributes map[string]string
}

func (p *Producer) SendBatch(ctx context.Context, messages []*RawMessage) (*BatchResult, error)
```

Chunking at 10 and `BatchResult` collection are retained as-is.

Delete `Producer.Send` — it is dead code, and reviving it later from git is trivial.

**Message attributes:** currently `MessageType` (`producer.go:133`). Rename to `event_type` now, matching ADR-018, even though attributes are inert on a direct-to-queue send. It costs nothing today and is exactly the string SNS filter policies will match on later — getting it wrong then means subscriptions that silently match nothing.

`aws.Message` remains the **receive-side** type; it already carries `ReceiptHandle` and `Attributes`, which are transport concerns. It is no longer used for sending.

## 6. Ship it

**`docker/relay/Dockerfile`** — clone the `utils` shape: `golang:1.26.5-alpine` builder → `alpine:latest` runtime, non-root `appuser`, workspace-root build context (ADR-016). No `EXPOSE` — the relay listens on nothing. Sibling `go.mod` copies for `apps/api` and `apps/utils` are required (the go.work validation rule in `docker/CLAUDE.md`); no dummy config file is needed since ADR-023 removed all embedding.

**`publish-images.yml`** — add to the image matrix:

```yaml
- service: relay
  repository: commerce-relay-registry
```

The `deploy` job is **not** touched. It currently runs a `utils` migration task and then rolls `commerce-api`; inserting a relay roll needs a deliberate answer about ordering relative to migrations, which is out of scope here.

**Cross-repo (`matrix`):** ECR repo `commerce-relay-registry` (IMMUTABLE, matching `api`/`utils`), ECS task definition, and an IAM task role with `sqs:SendMessage` + `sqs:GetQueueUrl` on `commerce-payments-queue` — **not** `sns:Publish`, since SNS is deferred.

## 7. Tests — relay's first

The Publisher port makes these possible. gomock for the repository, matching the `mock_*_test.go` pattern in `apps/api/internal/services`; a hand-written fake for `Publisher`.

| Case | Asserts |
|------|---------|
| Empty claim | publisher never called; no marks, no attempt bumps |
| All accepted | every row marked published; no attempt bumps |
| Partial rejection | **only** accepted rows marked; rejected rows' attempts incremented |
| Whole-call error | error propagates; transaction rolls back; nothing marked |
| Envelope mapping | `OccurredAt` equals the row's `CreatedDate`, **not** now; aggregate type/id populated |

That last case is the regression guard for the timestamp bug this slice fixes.

## Documentation

- **ADR-018** — record the dotted-lowercase naming convention; update the chaining amendment's event names (`OrderPaid` → `order.paid`, `PaymentFailed` → `payment.failed`).
- **ADR-018's 2026-07-21 SNS amendment** — amend to record that the swap is *deferred*, with the reason (`order.placed` has a single consumer by design) and the trigger that would reverse it (a second `order.placed` consumer — most likely an async inventory reservation).
- **`facts.md`** — envelope shape, event naming convention, queue name, `maxAttempts` default.
- **`issues.md` #130** — tick the CI item (already true), record what this slice closed and what it deliberately deferred.
- **`docker/CLAUDE.md`** — add `relay/` to the structure block and drop the "No `docker/relay/Dockerfile` yet" note.
- **Root `CLAUDE.md`** — the ⚠️ relay gap note becomes narrower: image now published, service roll still manual.

## Non-goals

- SNS, fan-out, and filter policies — deferred with a recorded trigger.
- The payment processor, notifier, and inventory services — slices 1–3.
- A relay deploy step in the `deploy` job.
- `VisibilityExtender`'s three known gaps (#130) — consumer-side, and this slice ships no consumer.
- Any ADR-019 observability beyond logging a parked row.
- Fixing `Get`/`GetAll` error swallowing.

## Open item to confirm before implementing

`daemon.go:19` reads `commerce-payments-queue` (plural). The queue name was described as `commerce-payment-queue` (singular), and #130 records that the live queue's name was never reconciled against the Terraform draft. `queueManager.GetUrl` resolves by exact name, so a mismatch fails at daemon startup with a message that reads like missing infrastructure rather than a typo. Confirm against live AWS and `matrix` before this ships.

## Verification

```bash
(cd internal/shared && go build ./... && go test ./...)
(cd apps/relay     && go build ./... && go test ./...)
(cd apps/api       && go build ./... && go test ./...)
(cd apps/utils     && go build ./... && go test ./...)
docker build -f docker/relay/Dockerfile .
docker build -f docker/api/Dockerfile .
docker build -f docker/utils/Dockerfile .
```

End to end: place an order via the API, confirm one `commerce.outbox` row, run `(cd apps/relay && go run .)`, and confirm the message lands on `commerce-payments-queue` with a body that parses as `events.Envelope` — `aggregate.id` matching the order and `occurred_at` matching the row's `created_date`, not the publish time.
