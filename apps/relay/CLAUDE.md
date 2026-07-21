# CLAUDE.md — apps/relay

## Overview & Purpose

Standalone Go workspace module (`commerce/relay`) that drains the transactional outbox (`commerce.outbox`, defined in `internal/shared`) to the event broker. **Producer-only**, per ADR-018 — this app never consumes. Downstream consumers (`notifier`, and future `billing`/`shipping`) are separate apps, not part of this module.

## Structure

- `main.go` — composition root. Connects to Postgres, builds the relay via `worker.NewDaemon`, then calls `Start(ctx)` **synchronously** (blocks until `ctx` is canceled via `signal.NotifyContext` — no goroutine, no sleep-based shutdown race).
- `worker/daemon.go` — `NewDaemon(ctx, db, awsCfg) (managers.RelayManagerI, error)`: the module's one wiring function. Creates the SQS client, **resolves** the queue URL via `GetUrl` (never creates it — queue/topic infra is owned by Terraform in `matrix`/`iac-matrix`; fails fast if it hasn't been provisioned yet), then assembles the publisher, outbox service, and `RelayManager`.
- `internal/managers/relay_manager.go` — `RelayManagerI` is one method: `Start(ctx) error`. Ticks on `interval` (`DefaultInterval` = 5s), calling `OutboxService.ProcessBatch(ctx, batchSize)` (`DefaultBatchSize` = 10) each tick until `ctx` is done.
- `internal/managers/consumer_manager.go` — **not wired into `RelayManager`.** Scaffolding for a future, separate consumer app (SQS listener + message-type dispatch); left in place but deliberately unused, per the producer-only decision. Don't wire it into this module.
- `internal/publisher/sqs_publisher.go` — `SqsPublisher.Publish(ctx, events)`: chunks events into ≤10-message SQS batches (the SDK's own `SendMessageBatch` limit), all-or-nothing per chunk — a partial-batch failure fails the whole call, so the caller's transaction rolls back and the *entire* poll retries next tick (safe, since consumers must dedupe on `event_id` regardless). `OutboxService` depends on the **concrete** `*publisher.SqsPublisher` type, not an interface — there's no gomock seam here by design; testing this service means exercising a real `aws.Producer`/SQS (or LocalStack), not a mock.
  - **Known gap:** this publishes straight to one SQS queue. Per the ADR-018 amendment (2026-07-21, `docs/project-notes/decisions.md`), the *resolved* design is relay → **SNS** topic (`commerce-domain-events`) → per-consumer SQS queue — `SqsPublisher` is expected to be replaced with an SNS publisher (topic ARN, `sns:Publish`) before this goes live. Tracked in `docs/project-notes/issues.md` #130.
- `internal/services/outbox/outbox_service.go` — `ProcessBatch(ctx, limit)`: inside one `manager.Execute` (Unit-of-Work transaction, `internal/shared/managers/transaction`), claims via `r.Outbox().GetNextBatch(limit)` (`FOR UPDATE SKIP LOCKED`), no-ops on an empty batch, publishes, then `r.Outbox().MarkPublished(ids)` — publish-before-mark is the at-least-once guarantee (ADR-018 non-negotiable #4).
- `internal/dto/outbox/outbox.go` — `Outbox` DTO; `ToMessage` converts a DTO row into `aws.Message` for the publisher.
- `configs/config.go` — relay's own `Config{Database, Aws}`, built on `internal/shared/configs` primitives (same pattern as `api/configs`).

## Conventions / gotchas

- **Queue infra is IaC-owned.** The relay only ever calls `GetUrl` — it never creates queues, topics, or subscriptions. If the named resource doesn't exist yet in AWS, the relay fails fast at startup rather than silently provisioning something Terraform doesn't know about.
- **One poll = one transaction.** Claim, publish, and mark must all go through the tx-bound `r.Outbox()` handed into the `manager.Execute` callback — never the struct's own injected `repo` field once inside `Execute`. Using the base-DB repo there breaks atomicity and silently releases the `SKIP LOCKED` row lock before the publish completes (this exact bug hit both this service and `OrderService.Save` — see `docs/project-notes/bugs.md` BUG-030).
- **Concurrency model is "Model B"** (ADR-018 amendment 2026-07-01): N autonomous workers, each with its own DB session/transaction, no shared coordinator. Not yet implemented as multiple workers in this module — today's `RelayManager.Start` is a single ticker loop; scaling to N workers is future work.
- **Not yet in CI.** `go.work` includes `./apps/relay`, but `.github/workflows/go.yml`'s `go work init` step and `publish-images.yml`'s image matrix do not. No `docker/relay/Dockerfile` exists yet either. Tracked in `docs/project-notes/issues.md` #130.

## Status

In progress — `docs/project-notes/issues.md` #130. Claim → publish → mark works end-to-end against direct SQS; the SNS swap, CI wiring, and Dockerfile are the open items.
