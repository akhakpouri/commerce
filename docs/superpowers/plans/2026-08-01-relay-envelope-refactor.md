# Relay envelope/publisher refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the ad-hoc `aws.Message` send path with a proper event envelope (`events.Envelope`) as the SQS message body, behind a testable `Publisher` port, with partial-failure handling and a poison-row escape hatch in the outbox relay.

**Architecture:** `internal/shared/events.Envelope` becomes the transport-agnostic domain-event wire format (no SQS, no GORM). `internal/shared/aws.Message` is kept but narrowed to the **receive side only** (it already carries `ReceiptHandle`/`Attributes`, which are SQS transport concerns an event envelope shouldn't know about). The **send side** gets a new `aws.RawMessage` (pre-marshalled body + attributes), and a `Publisher` port is defined in `apps/relay/internal/services/outbox` (where it's consumed) so `OutboxService.ProcessBatch` can be unit-tested against a fake instead of real SQS.

**Tech Stack:** Go 1.26, GORM, AWS SDK v2 (SQS), `go.uber.org/mock` (gomock), `stretchr/testify`.

## Global Constraints

- `internal/shared/events` must never import `models` or GORM — consumers (future notifier/payment-processor) import it without dragging in a database dependency.
- The envelope carries the **full** ADR-018 shape now (`EventId`, `EventType`, `OccurredAt`, `Aggregate`, `Version`, `Payload`) — no partial rollout.
- The envelope is the SQS message **body** itself (marshalled directly into `RawMessage.Body`), not nested inside another wrapper.
- `aws.Message` is **not deleted** — it remains the receive-side type for `Consumer`. Only the send path changes.
- Claim → publish → mark, all inside one `manager.Execute` transaction, is unchanged and non-negotiable (ADR-018, 2026-07-01 amendment): a crash between publish and commit re-publishes a duplicate, which consumers must dedupe; the reverse order loses events silently.
- `Publisher.Publish`'s `error` return is reserved for **whole-call** failures (network, auth, marshalling). A broker-side rejection of individual messages is a **successful** call with non-empty `PublishResult.FailedIds` — not an error.
- `IncrementAttempts` must be a SQL-side increment (`gorm.Expr("attempts + 1")`), never read-modify-write — concurrent relay workers claim disjoint rows and must not race each other's counts.
- `maxAttempts` is configurable (env-driven), default `10`, and is **not** a package constant.
- The SQS message attribute carrying the event type is named `event_type` (not the old `MessageType`) — this is the literal string a future SNS filter policy will match on.
- Every module builds/tests from its own directory (root `CLAUDE.md`): `(cd internal/shared && go build ./... && go test ./...)`, `(cd apps/relay && go build ./... && go test ./...)`.

**Out of scope (deferred to a follow-up plan):** `docker/relay/Dockerfile`, `publish-images.yml`'s image matrix, cross-repo Terraform, and all documentation updates (ADR-018, `facts.md`, `issues.md` #130, `apps/relay/CLAUDE.md`, root `CLAUDE.md`).

---

### Task 1: `events.Envelope` — fix naming, lock the wire contract with a test

The struct was committed as `Envelop`/`OccuredAt` (commit `64eaf07`) but nothing consumes it yet. Fix the spelling now, before anything depends on it.

**Files:**
- Modify (rename): `internal/shared/events/envelop.go` → `internal/shared/events/envelope.go`
- Test: `internal/shared/events/envelope_test.go`
- Modify: `internal/shared/go.mod` (add `testify` as a direct test dependency)

**Interfaces:**
- Produces: `events.Envelope{EventId uuid.UUID, EventType string, OccurredAt time.Time, Aggregate events.Aggregate, Version int, Payload json.RawMessage}` — JSON tags: `event_id`, `event_type`, `occurred_at`, `aggregate`, `version`, `payload`. (`events.Aggregate{Type string, Id uint}` already has the correct shape and needs no change.)

- [ ] **Step 1: Add testify to `internal/shared`**

```bash
(cd internal/shared && go get github.com/stretchr/testify@v1.11.1)
```

- [ ] **Step 2: Write the failing test**

Create `internal/shared/events/envelope_test.go`:

```go
package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEnvelope_JSONRoundTrip(t *testing.T) {
	eventId := uuid.New()
	occurredAt := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	original := Envelope{
		EventId:    eventId,
		EventType:  "order.placed",
		OccurredAt: occurredAt,
		Aggregate:  Aggregate{Type: "Order", Id: 42},
		Version:    1,
		Payload:    json.RawMessage(`{"total_amount":125.5}`),
	}

	body, err := json.Marshal(original)
	assert.NoError(t, err)

	expected := `{
		"event_id":"` + eventId.String() + `",
		"event_type":"order.placed",
		"occurred_at":"2026-08-01T12:00:00Z",
		"aggregate":{"type":"Order","id":42},
		"version":1,
		"payload":{"total_amount":125.5}
	}`
	assert.JSONEq(t, expected, string(body))

	var decoded Envelope
	assert.NoError(t, json.Unmarshal(body, &decoded))
	assert.Equal(t, original, decoded)
}
```

- [ ] **Step 3: Run the test, confirm it fails to compile**

```bash
(cd internal/shared && go test ./events/... -run TestEnvelope_JSONRoundTrip -v)
```

Expected: FAIL — `undefined: Envelope` (the type is still named `Envelop`).

- [ ] **Step 4: Rename the file and fix the naming**

```bash
git mv internal/shared/events/envelop.go internal/shared/events/envelope.go
```

Replace the contents of `internal/shared/events/envelope.go` with:

```go
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Envelope struct {
	EventId    uuid.UUID       `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Aggregate  Aggregate       `json:"aggregate"`
	Version    int             `json:"version"`
	Payload    json.RawMessage `json:"payload"`
}
```

- [ ] **Step 5: Run the test, confirm it passes**

```bash
(cd internal/shared && go test ./events/... -v)
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/shared/events/envelope.go internal/shared/events/envelope_test.go internal/shared/go.mod internal/shared/go.sum
git commit -m "fix: rename events.Envelop to Envelope, OccuredAt to OccurredAt"
```

(Note: `git mv` followed by editing the destination file — there is no `envelop.go` left to `git rm`.)

---

### Task 2: Thread `maxAttempts` through relay config

No behavior to unit-test here — pure config plumbing, verified by build. `maxAttempts` must not be a package constant (Global Constraints), so it needs an env key and a config field, following the exact pattern already used for `Max_Messages`/`Worker_Count`.

**Files:**
- Modify: `internal/shared/constants/constants.go`
- Modify: `apps/relay/configs/config.go`
- Modify: `apps/relay/configs/dev.env`
- Modify: `apps/relay/configs/dev.env.example`

**Interfaces:**
- Produces: `constants.EnvKeys.OutboxMaxAttempts` (string `"Outbox_Max_Attempts"`), `configs.Config.MaxAttempts int` (default `10`)

- [ ] **Step 1: Add the env key**

In `internal/shared/constants/constants.go`, add to the `EnvKeys` var (after `WorkerCount`):

```go
	WorkerCount:        "Worker_Count",
	OutboxMaxAttempts:  "Outbox_Max_Attempts",
```

and to the `envKeys` struct (after `WorkerCount`):

```go
	WorkerCount        string
	OutboxMaxAttempts  string
```

- [ ] **Step 2: Add the config field**

In `apps/relay/configs/config.go`, add `MaxAttempts int` to the `Config` struct:

```go
type Config struct {
	Database    db.DbConfig
	Aws         cfg.AWSConfig
	Consumer    cfg.ConsumerConfig
	MaxAttempts int
}
```

and populate it in `NewConfig()`, alongside the existing `Consumer` block:

```go
	c := &Config{
		Database: *cfg.NewDbConfig(),
		Aws: cfg.AWSConfig{
			AccessKeyID:     cfg.GetEnvOrDefault(constants.EnvKeys.AWSAccessKeyID, ""),
			SecretAccessKey: cfg.GetEnvOrDefault(constants.EnvKeys.AWSSecretAccessKey, ""),
			Region:          cfg.GetEnvOrDefault(constants.EnvKeys.AWSRegion, "us-east-1"),
			Endpoint:        cfg.GetEnvOrDefault(constants.EnvKeys.AWSEndpoint, ""),
		},
		Consumer: cfg.ConsumerConfig{
			Max:      cfg.GetEnvOrDefaultToInt(constants.EnvKeys.MaxMessages, 10),
			Timeout:  cfg.GetEnvOrDefaultToInt(constants.EnvKeys.VisibilityTimeout, 30),
			WaitTime: cfg.GetEnvOrDefaultToInt(constants.EnvKeys.WaitTime, 30),
			Count:    cfg.GetEnvOrDefaultToInt(constants.EnvKeys.WorkerCount, 5),
		},
		MaxAttempts: cfg.GetEnvOrDefaultToInt(constants.EnvKeys.OutboxMaxAttempts, 10),
	}
```

- [ ] **Step 3: Add the env var to both dev.env files**

Append to `apps/relay/configs/dev.env`:

```
Outbox_Max_Attempts=10
```

Append to `apps/relay/configs/dev.env.example`:

```
Outbox_Max_Attempts=10
```

- [ ] **Step 4: Build to confirm it compiles**

```bash
(cd internal/shared && go build ./...)
(cd apps/relay && go build ./configs/...)
```

Expected: both succeed.

- [ ] **Step 5: Commit**

```bash
git add internal/shared/constants/constants.go apps/relay/configs/config.go apps/relay/configs/dev.env apps/relay/configs/dev.env.example
git commit -m "feat: add configurable Outbox_Max_Attempts to relay config"
```

---

### Task 3: Outbox repository — `IncrementAttempts` + `GetNextBatch` poison filter

No repository in this codebase has a unit test today (verified: zero `*_test.go` under any `repositories/` package) — they're thin GORM wrappers exercised indirectly through service-layer gomock tests. This task follows that convention; the new behavior gets exercised by Task 5's tests via the mock.

**Files:**
- Modify: `internal/shared/repositories/outbox/outbox_repository.go`

**Interfaces:**
- Produces: `OutboxRepositoryI.GetNextBatch(limit int, maxAttempts int) ([]*model.Outbox, error)`, `OutboxRepositoryI.IncrementAttempts(ids []uint) error`

- [ ] **Step 1: Update the interface and `GetNextBatch`**

In `internal/shared/repositories/outbox/outbox_repository.go`, change the interface:

```go
type OutboxRepositoryI interface {
	Get(id uint) (*model.Outbox, error)
	GetAll() ([]*model.Outbox, error)
	GetNextBatch(limit int, maxAttempts int) ([]*model.Outbox, error)
	IncrementAttempts(ids []uint) error
	MarkPublished(ids []uint) error
	Delete(id uint) error
	Save(outbox *model.Outbox) error
}
```

Replace `GetNextBatch`'s body:

```go
// GetNextBatch implements [OutboxRepositoryI]. Rows already at maxAttempts
// are excluded so a permanently-failing row stops consuming a batch slot on
// every poll once it's parked.
func (o *OutboxRepository) GetNextBatch(limit int, maxAttempts int) ([]*model.Outbox, error) {
	var outbox []*model.Outbox
	if err := o.db.Clauses(clause.Locking{
		Strength: "UPDATE",
		Options:  "SKIP LOCKED",
	}).Where("published_at is null AND attempts < ?", maxAttempts).
		Order("id").
		Limit(limit).
		Find(&outbox).Error; err != nil {
		slog.Error("Error retrieving outbox for next batch", "error", err, "limit", limit, "maxAttempts", maxAttempts)
		return nil, err
	}

	return outbox, nil
}
```

- [ ] **Step 2: Add `IncrementAttempts`**

Add this method (near `MarkPublished`):

```go
// IncrementAttempts implements [OutboxRepositoryI]. Uses a SQL-side
// increment rather than read-modify-write so concurrent relay workers
// claiming disjoint rows never race each other's attempt counts.
func (o *OutboxRepository) IncrementAttempts(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return o.db.Model(&model.Outbox{}).Where("id IN ?", ids).
		Update("attempts", gorm.Expr("attempts + 1")).Error
}
```

`gorm.Expr` needs the `gorm.io/gorm` package, which this file already imports.

- [ ] **Step 3: Build to confirm it compiles**

```bash
(cd internal/shared && go build ./...)
```

Expected: succeeds. (`apps/relay` will **not** build yet — its `OutboxService` still calls the old single-arg `GetNextBatch`. That's expected; it's fixed in Task 5.)

- [ ] **Step 4: Commit**

```bash
git add internal/shared/repositories/outbox/outbox_repository.go
git commit -m "feat: add IncrementAttempts and maxAttempts filter to outbox repository"
```

---

### Task 4: Outbox DTO — `Version` field + `ToEnvelope` (replaces `ToMessage`)

Removing `ToMessage` here breaks `apps/relay/internal/publisher` (it still calls `dto.ToMessage`) until Task 6 rewrites it — same kind of expected transient breakage as Task 3. This task's own package (`internal/dto/outbox`) builds and tests green in isolation regardless.

**Files:**
- Modify: `apps/relay/internal/dto/outbox/outbox.go`
- Test: `apps/relay/internal/dto/outbox/outbox_test.go`
- Modify: `apps/relay/go.mod` (add `testify`)

**Interfaces:**
- Consumes: `events.Envelope`, `events.Aggregate` (Task 1)
- Produces: `dto.Outbox{..., Version int}`, `func ToEnvelope(dto *Outbox) events.Envelope`

- [ ] **Step 1: Add testify to `apps/relay`**

```bash
(cd apps/relay && go get github.com/stretchr/testify@v1.11.1)
```

- [ ] **Step 2: Write the failing test**

Create `apps/relay/internal/dto/outbox/outbox_test.go`:

```go
package outbox

import (
	"encoding/json"
	"testing"
	"time"

	"commerce/internal/shared/events"
	"commerce/internal/shared/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestToEnvelope(t *testing.T) {
	eventId := uuid.New()
	createdAt := time.Date(2026, 8, 1, 9, 30, 0, 0, time.UTC)

	dto := FromModel(&models.Outbox{
		Base: models.Base{
			Id:          7,
			CreatedDate: createdAt,
		},
		EventId:       eventId,
		EventType:     "order.placed",
		AggregateId:   42,
		AggregateType: "Order",
		Payload:       datatypes.JSON(`{"total_amount":125.5}`),
		Version:       1,
	})

	envelope := ToEnvelope(dto)

	// OccurredAt must equal the row's CreatedDate (when the domain
	// transaction committed), not time.Now() - the regression guard for the
	// timestamp bug this refactor fixes.
	assert.Equal(t, events.Envelope{
		EventId:    eventId,
		EventType:  "order.placed",
		OccurredAt: createdAt,
		Aggregate:  events.Aggregate{Type: "Order", Id: 42},
		Version:    1,
		Payload:    json.RawMessage(`{"total_amount":125.5}`),
	}, envelope)
}
```

- [ ] **Step 3: Run the test, confirm it fails to compile**

```bash
(cd apps/relay && go test ./internal/dto/outbox/... -v)
```

Expected: FAIL — `dto.Version` undefined and/or `ToEnvelope` undefined.

- [ ] **Step 4: Implement**

Replace the full contents of `apps/relay/internal/dto/outbox/outbox.go`:

```go
package outbox

import (
	"encoding/json"
	"time"

	"commerce/internal/shared/events"
	"commerce/internal/shared/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Outbox struct {
	Id            uint       `json:"id"`
	EventId       uuid.UUID  `json:"event_id"`
	EventType     string     `json:"event_type"`
	AggregateId   uint       `json:"aggregate_id"`
	AggregateType string     `json:"aggregate_type"`
	Payload       string     `json:"payload"`
	PublishedAt   *time.Time `json:"published_at"`
	Attempts      int        `json:"attempts"`
	Version       int        `json:"version"`
	CreatedAt     time.Time  `json:"created_at"`
}

func FromModel(model *models.Outbox) *Outbox {
	return &Outbox{
		Id:            model.Id,
		EventId:       model.EventId,
		EventType:     model.EventType,
		AggregateId:   model.AggregateId,
		AggregateType: model.AggregateType,
		Payload:       string(model.Payload),
		PublishedAt:   model.PublishedAt,
		Attempts:      model.Attempts,
		Version:       model.Version,
		CreatedAt:     model.CreatedDate,
	}
}

func ToModel(dto *Outbox) *models.Outbox {
	return &models.Outbox{
		Base: models.Base{
			Id: dto.Id,
		},
		EventId:       dto.EventId,
		EventType:     dto.EventType,
		AggregateId:   dto.AggregateId,
		AggregateType: dto.AggregateType,
		Payload:       datatypes.JSON(dto.Payload),
		PublishedAt:   dto.PublishedAt,
		Attempts:      dto.Attempts,
		Version:       dto.Version,
	}
}

func FromAllModels(models []*models.Outbox) []*Outbox {
	var dtos []*Outbox
	for _, m := range models {
		dtos = append(dtos, FromModel(m))
	}
	return dtos
}

// ToEnvelope builds the wire-format event envelope for an outbox row.
// OccurredAt is the row's CreatedAt (the moment the domain transaction
// committed), not time.Now() - a backed-up or retrying relay must still
// report when the event actually happened, not when it happened to be sent.
func ToEnvelope(dto *Outbox) events.Envelope {
	return events.Envelope{
		EventId:   dto.EventId,
		EventType: dto.EventType,
		OccurredAt: dto.CreatedAt,
		Aggregate: events.Aggregate{
			Type: dto.AggregateType,
			Id:   dto.AggregateId,
		},
		Version: dto.Version,
		Payload: json.RawMessage(dto.Payload),
	}
}
```

Note: `ToMessage` and the `commerce/internal/shared/aws` import are gone — nothing in this file talks to SQS anymore.

- [ ] **Step 5: Run the test, confirm it passes**

```bash
(cd apps/relay && go test ./internal/dto/outbox/... -v)
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/relay/internal/dto/outbox/outbox.go apps/relay/internal/dto/outbox/outbox_test.go apps/relay/go.mod apps/relay/go.sum
git commit -m "feat: add Version to outbox DTO, replace ToMessage with ToEnvelope"
```

---

### Task 5: `OutboxService` — Publisher port, narrowed interface, `ProcessBatch` rewrite + tests

This is the core of the refactor. The `Publisher` port is defined here (where it's consumed), so `ProcessBatch` can be fully unit-tested with a hand-written fake — no real SQS/LocalStack needed. This task does not touch `apps/relay/internal/publisher` or `worker/daemon.go` yet, so the relay module as a whole will not build after this task (those callers still use the old signatures) — that's expected and resolved in Tasks 6–7. This package's own tests pass in isolation regardless, since Go only compiles a package's actual dependencies when you `go test` it directly.

**Files:**
- Create: `apps/relay/internal/services/outbox/publisher.go`
- Modify: `apps/relay/internal/services/outbox/outbox_service.go`
- Create: `apps/relay/internal/services/outbox/mock_outbox_repo_test.go`
- Create: `apps/relay/internal/services/outbox/outbox_service_test.go`
- Modify: `apps/relay/go.mod` (add `go.uber.org/mock`)

**Interfaces:**
- Consumes: `events.Envelope` (Task 1), `dto.ToEnvelope`/`dto.FromAllModels`/`dto.Outbox.Version` (Task 4), `repo.OutboxRepositoryI.GetNextBatch(limit, maxAttempts)` / `.IncrementAttempts(ids)` (Task 3)
- Produces: `outbox.Publisher` interface, `outbox.PublishResult{PublishedIds, FailedIds []uuid.UUID}`, `outbox.OutboxServiceI` (narrowed to just `ProcessBatch`), `NewOutboxService(manager manager.ManagerI, publisher Publisher, maxAttempts int) OutboxServiceI`

- [ ] **Step 1: Add gomock to `apps/relay`**

```bash
(cd apps/relay && go get go.uber.org/mock@v0.6.0)
```

- [ ] **Step 2: Define the `Publisher` port**

Create `apps/relay/internal/services/outbox/publisher.go`:

```go
package outbox

import (
	"context"

	"commerce/internal/shared/events"

	"github.com/google/uuid"
)

// Publisher is the port ProcessBatch publishes through. Defined here, where
// it's consumed, rather than in internal/shared/aws or the publisher
// package, so this service can be tested with a fake instead of exercising
// real SQS.
type Publisher interface {
	Publish(ctx context.Context, envelopes []events.Envelope) (PublishResult, error)
}

// PublishResult is keyed by event id, not outbox row id - the publisher has
// no business knowing about database rows. ProcessBatch holds both (it
// claimed the rows) and does the id translation.
type PublishResult struct {
	PublishedIds []uuid.UUID
	FailedIds    []uuid.UUID
}
```

- [ ] **Step 3: Rewrite `outbox_service.go`**

Replace the full contents of `apps/relay/internal/services/outbox/outbox_service.go`:

```go
package outbox

import (
	"context"
	"log/slog"

	"commerce/internal/shared/events"
	manager "commerce/internal/shared/managers/transaction"
	dto "commerce/relay/internal/dto/outbox"

	"github.com/google/uuid"
)

// OutboxServiceI is narrowed to ProcessBatch - the only method
// relay_manager.go (its sole caller) uses. Get/GetAll/GetNextBatch/
// MarkPublished/Delete were unused passthroughs to the repository.
type OutboxServiceI interface {
	ProcessBatch(ctx context.Context, limit int) error
}

type OutboxService struct {
	manager     manager.ManagerI
	publisher   Publisher
	maxAttempts int
}

func NewOutboxService(manager manager.ManagerI, publisher Publisher, maxAttempts int) OutboxServiceI {
	return &OutboxService{
		manager:     manager,
		publisher:   publisher,
		maxAttempts: maxAttempts,
	}
}

// ProcessBatch implements [OutboxServiceI]. It claims up to limit unpublished
// events (skipping rows already at maxAttempts), publishes them, and marks
// accepted rows published while bumping the attempt count on rejected ones -
// all inside one transaction, so the SELECT ... FOR UPDATE SKIP LOCKED lock
// is held for the whole claim-publish-mark window. A whole-call error (as
// opposed to a per-event rejection in PublishResult.FailedIds) rolls the
// transaction back and retries the entire batch next poll - correct, since
// nothing was accepted by the broker.
func (o *OutboxService) ProcessBatch(ctx context.Context, limit int) error {
	return o.manager.Execute(func(r manager.RepositoriesI) error {
		models, err := r.Outbox().GetNextBatch(limit, o.maxAttempts)
		if err != nil {
			slog.Error("Exception occured getting the next batch.", "error", err, "limit", limit)
			return err
		}
		if len(models) == 0 {
			return nil
		}

		dtos := dto.FromAllModels(models)
		envelopes := make([]events.Envelope, len(dtos))
		rowIdByEventId := make(map[uuid.UUID]uint, len(dtos))
		for i, d := range dtos {
			envelopes[i] = dto.ToEnvelope(d)
			rowIdByEventId[d.EventId] = d.Id
		}

		result, err := o.publisher.Publish(ctx, envelopes)
		if err != nil {
			slog.Error("Exception occured publishing outbox batch.", "error", err)
			return err
		}

		if len(result.FailedIds) > 0 {
			failedRowIds := make([]uint, len(result.FailedIds))
			for i, eventId := range result.FailedIds {
				failedRowIds[i] = rowIdByEventId[eventId]
			}
			slog.Warn("Outbox rows failed to publish and will be retried.", "rowIds", failedRowIds)
			if err := r.Outbox().IncrementAttempts(failedRowIds); err != nil {
				return err
			}
		}

		if len(result.PublishedIds) == 0 {
			return nil
		}

		publishedRowIds := make([]uint, len(result.PublishedIds))
		for i, eventId := range result.PublishedIds {
			publishedRowIds[i] = rowIdByEventId[eventId]
		}
		return r.Outbox().MarkPublished(publishedRowIds)
	})
}
```

- [ ] **Step 4: Generate the repository mock**

If `mockgen` isn't installed: `go install go.uber.org/mock/mockgen@v0.6.0`.

```bash
(cd apps/relay/internal/services/outbox && mockgen -source=../../../../../internal/shared/repositories/outbox/outbox_repository.go -destination=mock_outbox_repo_test.go -package=outbox)
```

If `mockgen` isn't available in the environment, create `apps/relay/internal/services/outbox/mock_outbox_repo_test.go` directly with this exact content (this is what the command above produces for the interface from Task 3):

```go
// Code generated by MockGen. DO NOT EDIT.
// Source: ../../../../../internal/shared/repositories/outbox/outbox_repository.go
//
// Generated by this command:
//
//	mockgen -source=../../../../../internal/shared/repositories/outbox/outbox_repository.go -destination=mock_outbox_repo_test.go -package=outbox
//

// Package outbox is a generated GoMock package.
package outbox

import (
	models "commerce/internal/shared/models"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockOutboxRepositoryI is a mock of OutboxRepositoryI interface.
type MockOutboxRepositoryI struct {
	ctrl     *gomock.Controller
	recorder *MockOutboxRepositoryIMockRecorder
	isgomock struct{}
}

// MockOutboxRepositoryIMockRecorder is the mock recorder for MockOutboxRepositoryI.
type MockOutboxRepositoryIMockRecorder struct {
	mock *MockOutboxRepositoryI
}

// NewMockOutboxRepositoryI creates a new mock instance.
func NewMockOutboxRepositoryI(ctrl *gomock.Controller) *MockOutboxRepositoryI {
	mock := &MockOutboxRepositoryI{ctrl: ctrl}
	mock.recorder = &MockOutboxRepositoryIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOutboxRepositoryI) EXPECT() *MockOutboxRepositoryIMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockOutboxRepositoryI) Delete(id uint) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockOutboxRepositoryIMockRecorder) Delete(id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockOutboxRepositoryI)(nil).Delete), id)
}

// Get mocks base method.
func (m *MockOutboxRepositoryI) Get(id uint) (*models.Outbox, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", id)
	ret0, _ := ret[0].(*models.Outbox)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockOutboxRepositoryIMockRecorder) Get(id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockOutboxRepositoryI)(nil).Get), id)
}

// GetAll mocks base method.
func (m *MockOutboxRepositoryI) GetAll() ([]*models.Outbox, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll")
	ret0, _ := ret[0].([]*models.Outbox)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAll indicates an expected call of GetAll.
func (mr *MockOutboxRepositoryIMockRecorder) GetAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockOutboxRepositoryI)(nil).GetAll))
}

// GetNextBatch mocks base method.
func (m *MockOutboxRepositoryI) GetNextBatch(limit, maxAttempts int) ([]*models.Outbox, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNextBatch", limit, maxAttempts)
	ret0, _ := ret[0].([]*models.Outbox)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNextBatch indicates an expected call of GetNextBatch.
func (mr *MockOutboxRepositoryIMockRecorder) GetNextBatch(limit, maxAttempts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNextBatch", reflect.TypeOf((*MockOutboxRepositoryI)(nil).GetNextBatch), limit, maxAttempts)
}

// IncrementAttempts mocks base method.
func (m *MockOutboxRepositoryI) IncrementAttempts(ids []uint) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IncrementAttempts", ids)
	ret0, _ := ret[0].(error)
	return ret0
}

// IncrementAttempts indicates an expected call of IncrementAttempts.
func (mr *MockOutboxRepositoryIMockRecorder) IncrementAttempts(ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IncrementAttempts", reflect.TypeOf((*MockOutboxRepositoryI)(nil).IncrementAttempts), ids)
}

// MarkPublished mocks base method.
func (m *MockOutboxRepositoryI) MarkPublished(ids []uint) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MarkPublished", ids)
	ret0, _ := ret[0].(error)
	return ret0
}

// MarkPublished indicates an expected call of MarkPublished.
func (mr *MockOutboxRepositoryIMockRecorder) MarkPublished(ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MarkPublished", reflect.TypeOf((*MockOutboxRepositoryI)(nil).MarkPublished), ids)
}

// Save mocks base method.
func (m *MockOutboxRepositoryI) Save(outbox *models.Outbox) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Save", outbox)
	ret0, _ := ret[0].(error)
	return ret0
}

// Save indicates an expected call of Save.
func (mr *MockOutboxRepositoryIMockRecorder) Save(outbox any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*MockOutboxRepositoryI)(nil).Save), outbox)
}
```

- [ ] **Step 5: Write the failing tests**

Create `apps/relay/internal/services/outbox/outbox_service_test.go`:

```go
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"commerce/internal/shared/events"
	manager "commerce/internal/shared/managers/transaction"
	"commerce/internal/shared/models"
	orderrepo "commerce/internal/shared/repositories/order"
	repo "commerce/internal/shared/repositories/outbox"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/datatypes"
)

// fakeManager mirrors the pattern in apps/api/internal/services/order:
// instead of opening a real transaction, Execute runs the callback with fake
// repositories, so the service's transactional calls land on the mock.
type fakeManager struct {
	repos manager.RepositoriesI
}

func (f *fakeManager) Execute(fn func(r manager.RepositoriesI) error) error {
	return fn(f.repos)
}

type fakeRepositories struct {
	outbox repo.OutboxRepositoryI
}

// Order() is part of manager.RepositoriesI (shared with apps/api) but
// ProcessBatch never calls it - returning nil is safe.
func (f *fakeRepositories) Order() orderrepo.OrderRepositoryI { return nil }
func (f *fakeRepositories) Outbox() repo.OutboxRepositoryI    { return f.outbox }

// fakePublisher is a hand-written fake for the Publisher port - per the
// design spec, this service is tested against a fake, not a mock of SQS.
type fakePublisher struct {
	envelopes []events.Envelope
	result    PublishResult
	err       error
	calls     int
}

func (f *fakePublisher) Publish(_ context.Context, envelopes []events.Envelope) (PublishResult, error) {
	f.calls++
	f.envelopes = envelopes
	return f.result, f.err
}

func newMockRepo(t *testing.T) *MockOutboxRepositoryI {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return NewMockOutboxRepositoryI(ctrl)
}

func setup(mockRepo *MockOutboxRepositoryI, publisher *fakePublisher, maxAttempts int) OutboxServiceI {
	txm := &fakeManager{repos: &fakeRepositories{outbox: mockRepo}}
	return NewOutboxService(txm, publisher, maxAttempts)
}

func TestProcessBatch_EmptyClaim(t *testing.T) {
	mockRepo := newMockRepo(t)
	publisher := &fakePublisher{}
	svc := setup(mockRepo, publisher, 10)

	mockRepo.EXPECT().GetNextBatch(10, 10).Return(nil, nil)

	err := svc.ProcessBatch(context.Background(), 10)

	assert.NoError(t, err)
	assert.Equal(t, 0, publisher.calls)
}

func TestProcessBatch_AllAccepted(t *testing.T) {
	mockRepo := newMockRepo(t)
	eventId := uuid.New()
	createdAt := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	row := &models.Outbox{
		Base:          models.Base{Id: 1, CreatedDate: createdAt},
		EventId:       eventId,
		EventType:     "order.placed",
		AggregateId:   42,
		AggregateType: "Order",
		Payload:       datatypes.JSON(`{"total_amount":125.5}`),
		Version:       1,
	}
	publisher := &fakePublisher{result: PublishResult{PublishedIds: []uuid.UUID{eventId}}}
	svc := setup(mockRepo, publisher, 10)

	mockRepo.EXPECT().GetNextBatch(10, 10).Return([]*models.Outbox{row}, nil)
	mockRepo.EXPECT().MarkPublished([]uint{1}).Return(nil)

	err := svc.ProcessBatch(context.Background(), 10)

	assert.NoError(t, err)
	assert.Equal(t, 1, publisher.calls)
	assert.Equal(t, []events.Envelope{{
		EventId:    eventId,
		EventType:  "order.placed",
		OccurredAt: createdAt,
		Aggregate:  events.Aggregate{Type: "Order", Id: 42},
		Version:    1,
		Payload:    json.RawMessage(`{"total_amount":125.5}`),
	}}, publisher.envelopes)
}

func TestProcessBatch_PartialRejection(t *testing.T) {
	mockRepo := newMockRepo(t)
	acceptedId := uuid.New()
	rejectedId := uuid.New()
	rows := []*models.Outbox{
		{Base: models.Base{Id: 1}, EventId: acceptedId, EventType: "order.placed", AggregateType: "Order", Payload: datatypes.JSON(`{}`)},
		{Base: models.Base{Id: 2}, EventId: rejectedId, EventType: "order.placed", AggregateType: "Order", Payload: datatypes.JSON(`{}`)},
	}
	publisher := &fakePublisher{result: PublishResult{
		PublishedIds: []uuid.UUID{acceptedId},
		FailedIds:    []uuid.UUID{rejectedId},
	}}
	svc := setup(mockRepo, publisher, 10)

	mockRepo.EXPECT().GetNextBatch(10, 10).Return(rows, nil)
	mockRepo.EXPECT().MarkPublished([]uint{1}).Return(nil)
	mockRepo.EXPECT().IncrementAttempts([]uint{2}).Return(nil)

	err := svc.ProcessBatch(context.Background(), 10)

	assert.NoError(t, err)
}

func TestProcessBatch_WholeCallError(t *testing.T) {
	mockRepo := newMockRepo(t)
	row := &models.Outbox{Base: models.Base{Id: 1}, EventId: uuid.New(), AggregateType: "Order", Payload: datatypes.JSON(`{}`)}
	publisher := &fakePublisher{err: errors.New("sqs unavailable")}
	svc := setup(mockRepo, publisher, 10)

	mockRepo.EXPECT().GetNextBatch(10, 10).Return([]*models.Outbox{row}, nil)
	// Deliberately no MarkPublished/IncrementAttempts expectation - gomock
	// fails the test if either is called, proving nothing gets marked on a
	// whole-call error.

	err := svc.ProcessBatch(context.Background(), 10)

	assert.Error(t, err)
}
```

- [ ] **Step 6: Run the tests, confirm they fail first (if written before Step 3), then pass**

```bash
(cd apps/relay && go test ./internal/services/outbox/... -v)
```

Expected: all four tests PASS (`TestProcessBatch_EmptyClaim`, `TestProcessBatch_AllAccepted`, `TestProcessBatch_PartialRejection`, `TestProcessBatch_WholeCallError`).

- [ ] **Step 7: Commit**

```bash
git add apps/relay/internal/services/outbox/ apps/relay/go.mod apps/relay/go.sum
git commit -m "feat: define Publisher port, rewrite ProcessBatch with partial-failure handling and poison escape"
```

---

### Task 6: Send path — `aws.RawMessage`, delete dead `Producer.Send`, rewrite `SqsPublisher`

`Producer.SendBatch` has exactly one caller (`SqsPublisher`); `Producer.Send` has none (verified by grep). Neither this package nor `SqsPublisher` has automated tests today — a real `Producer`/SQS test needs LocalStack, which is out of scope here (same as before this refactor); this task is verified by build only.

**Files:**
- Modify: `internal/shared/aws/producer.go`
- Modify: `apps/relay/internal/publisher/sqs_publisher.go`

**Interfaces:**
- Consumes: `outbox.Publisher`, `outbox.PublishResult` (Task 5), `events.Envelope` (Task 1)
- Produces: `aws.RawMessage{Id, Body string, Attributes map[string]string}`, `Producer.SendBatch(ctx, []*RawMessage) (*BatchResult, error)`

- [ ] **Step 1: Add `RawMessage`, change `SendBatch`, delete `Send`**

Replace the full contents of `internal/shared/aws/producer.go`:

```go
package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

// purpose of the producer is to send messages to the queue
type Producer struct {
	client *sqs.Client
	url    string
}

func NewProducer(client *sqs.Client, url string) *Producer {
	return &Producer{
		client: client,
		url:    url,
	}
}

// RawMessage is a pre-marshalled SQS send: Body is the wire payload exactly
// as it will appear in the message body (the caller owns marshalling - e.g.
// an events.Envelope), Id correlates batch entries to their SendBatch
// result, and Attributes become SQS message attributes verbatim.
type RawMessage struct {
	Id         string
	Body       string
	Attributes map[string]string
}

// SendFIFOMessage sends a message to a FIFO queue with ordering guarantees
func (p *Producer) SendFIFOMessage(
	ctx context.Context,
	message *Message,
	messageGroupId string,
	deduplicationId string,
) (string, error) {
	if message.Id == "" {
		message.Id = uuid.New().String()
	}
	message.Timestamp = time.Now().UTC()

	body, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("marshaling message: %w", err)
	}

	// Use message ID as deduplication ID if not provided
	if deduplicationId == "" {
		deduplicationId = message.Id
	}

	input := &sqs.SendMessageInput{
		QueueUrl:               aws_sdk.String(p.url),
		MessageBody:            aws_sdk.String(string(body)),
		MessageGroupId:         aws_sdk.String(messageGroupId),
		MessageDeduplicationId: aws_sdk.String(deduplicationId),
	}

	result, err := p.client.SendMessage(ctx, input)
	if err != nil {
		return "", fmt.Errorf("sending FIFO message: %w", err)
	}

	return *result.MessageId, nil
}

func (p *Producer) SendBatch(
	ctx context.Context,
	messages []*RawMessage) (*BatchResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages are empty")
	} else if len(messages) > 10 {
		return nil, fmt.Errorf("batch size exceeds maximum of ten (10) messages")
	}

	entries := make([]types.SendMessageBatchRequestEntry, len(messages))

	for i, msg := range messages {
		if msg.Id == "" {
			msg.Id = uuid.New().String()
		}

		var attrs map[string]types.MessageAttributeValue
		if len(msg.Attributes) > 0 {
			attrs = make(map[string]types.MessageAttributeValue, len(msg.Attributes))
			for key, value := range msg.Attributes {
				attrs[key] = types.MessageAttributeValue{
					DataType:    aws_sdk.String("String"),
					StringValue: aws_sdk.String(value),
				}
			}
		}

		entries[i] = types.SendMessageBatchRequestEntry{
			Id:                aws_sdk.String(msg.Id),
			MessageBody:       aws_sdk.String(msg.Body),
			MessageAttributes: attrs,
		}
	}

	input := &sqs.SendMessageBatchInput{
		QueueUrl: aws_sdk.String(p.url),
		Entries:  entries,
	}

	result, err := p.client.SendMessageBatch(ctx, input)
	if err != nil {
		slog.Error("exception occured during send batch message", "error", err)
		return nil, fmt.Errorf("send batch message: %w", err)
	}

	successIds := make([]string, 0, len(result.Successful))
	failed := make([]BatchError, 0, len(result.Failed))

	for _, success := range result.Successful {
		successIds = append(successIds, *success.Id)
	}

	for _, f := range result.Failed {
		failed = append(failed, BatchError{
			Id:      *f.Id,
			Code:    *f.Code,
			Message: *f.Message,
		})
	}

	batchResult := &BatchResult{
		SuccessfulIds: successIds,
		Failed:        failed,
	}

	return batchResult, nil
}
```

(`Message`, `Consumer`, `VisibilityExtender`, `QueueMonitor` in the rest of the package are untouched — `Message` remains the receive-side type.)

- [ ] **Step 2: Rewrite `SqsPublisher`**

Replace the full contents of `apps/relay/internal/publisher/sqs_publisher.go`:

```go
package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	aws "commerce/internal/shared/aws"
	"commerce/internal/shared/events"
	outbox_service "commerce/relay/internal/services/outbox"

	"github.com/google/uuid"
)

const maxSendBatchSize = 10

// SqsPublisher adapts an SQS producer to the outbox service's Publisher port.
type SqsPublisher struct {
	producer *aws.Producer
}

func NewSqsPublisher(producer *aws.Producer) *SqsPublisher {
	return &SqsPublisher{producer: producer}
}

// Publish marshals each envelope as the SQS message body and sends in
// batches of 10 (SendMessageBatch's own limit). error is reserved for
// whole-call failures (network, auth, marshalling); a broker-side rejection
// of individual messages comes back as a non-empty PublishResult.FailedIds,
// not an error - ProcessBatch tells those two cases apart.
func (p *SqsPublisher) Publish(ctx context.Context, envelopes []events.Envelope) (outbox_service.PublishResult, error) {
	result := outbox_service.PublishResult{}

	for start := 0; start < len(envelopes); start += maxSendBatchSize {
		end := min(start+maxSendBatchSize, len(envelopes))

		messages := make([]*aws.RawMessage, end-start)
		for i, envelope := range envelopes[start:end] {
			body, err := json.Marshal(envelope)
			if err != nil {
				return outbox_service.PublishResult{}, fmt.Errorf("marshalling envelope %s: %w", envelope.EventId, err)
			}
			messages[i] = &aws.RawMessage{
				Id:   envelope.EventId.String(),
				Body: string(body),
				Attributes: map[string]string{
					"event_type": envelope.EventType,
				},
			}
		}

		batchResult, err := p.producer.SendBatch(ctx, messages)
		if err != nil {
			return outbox_service.PublishResult{}, fmt.Errorf("publishing outbox batch: %w", err)
		}

		for _, idStr := range batchResult.SuccessfulIds {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return outbox_service.PublishResult{}, fmt.Errorf("parsing published event id %q: %w", idStr, err)
			}
			result.PublishedIds = append(result.PublishedIds, id)
		}

		for _, failedEntry := range batchResult.Failed {
			id, err := uuid.Parse(failedEntry.Id)
			if err != nil {
				return outbox_service.PublishResult{}, fmt.Errorf("parsing failed event id %q: %w", failedEntry.Id, err)
			}
			result.FailedIds = append(result.FailedIds, id)
		}
	}

	return result, nil
}
```

- [ ] **Step 3: Build to confirm it compiles**

```bash
(cd internal/shared && go build ./...)
(cd apps/relay && go build ./internal/publisher/... ./internal/services/outbox/...)
```

Expected: both succeed. (`apps/relay`'s `worker` package still won't build — fixed in Task 7.)

- [ ] **Step 4: Commit**

```bash
git add internal/shared/aws/producer.go apps/relay/internal/publisher/sqs_publisher.go
git commit -m "feat: replace aws.Message send path with RawMessage, rewrite SqsPublisher against the Publisher port"
```

---

### Task 7: Wire it up — `worker/daemon.go` + `main.go`

The last piece: thread `maxAttempts` in, drop the now-unused `outboxRepo` construction (the service no longer takes a raw repository — it reaches the repo only through the transaction manager inside `ProcessBatch`).

**Files:**
- Modify: `apps/relay/worker/daemon.go`
- Modify: `apps/relay/main.go`

**Interfaces:**
- Consumes: `outbox_service.NewOutboxService(manager, publisher, maxAttempts)` (Task 5), `configs.Config.MaxAttempts` (Task 2)

- [ ] **Step 1: Update `daemon.go`**

Replace the full contents of `apps/relay/worker/daemon.go`:

```go
package worker

import (
	"context"
	"fmt"

	"commerce/internal/shared/aws"
	"commerce/internal/shared/configs"
	queue_manager "commerce/internal/shared/managers/aws"
	manager "commerce/internal/shared/managers/transaction"
	relay_manager "commerce/relay/internal/managers"
	"commerce/relay/internal/publisher"
	outbox_service "commerce/relay/internal/services/outbox"

	"gorm.io/gorm"
)

const QueueName = "commerce-payments-queue"

// NewDaemon wires the outbox-to-SQS relay: resolves the queue URL - the queue
// itself is provisioned by iac-matrix, never created here - then assembles
// the publisher, outbox service, and poll loop behind RelayManagerI.
func NewDaemon(ctx context.Context, db *gorm.DB, cfg *configs.AWSConfig, maxAttempts int) (relay_manager.RelayManagerI, error) {
	sqsClient, err := aws.NewSqsClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating SQS client: %w", err)
	}

	queueManager := queue_manager.NewQueueManager(sqsClient)
	queueUrl, err := queueManager.GetUrl(ctx, QueueName)
	if err != nil {
		return nil, fmt.Errorf("resolving queue %q (has it been provisioned in iac-matrix?): %w", QueueName, err)
	}

	producer := aws.NewProducer(sqsClient, queueUrl)
	sqsPublisher := publisher.NewSqsPublisher(producer)

	txManager := manager.NewManager(db)
	outboxService := outbox_service.NewOutboxService(txManager, sqsPublisher, maxAttempts)

	return relay_manager.NewRelayManager(outboxService, relay_manager.DefaultInterval, relay_manager.DefaultBatchSize), nil
}
```

(The `outbox_repo "commerce/internal/shared/repositories/outbox"` import and the `outboxRepo := outbox_repo.NewOutboxRepository(db)` line are gone — `OutboxService` no longer holds a raw repository.)

- [ ] **Step 2: Update `main.go`'s call site**

In `apps/relay/main.go`, change:

```go
	relay, err := daemon.NewDaemon(ctx, db, &config.Aws)
```

to:

```go
	relay, err := daemon.NewDaemon(ctx, db, &config.Aws, config.MaxAttempts)
```

- [ ] **Step 3: Build the whole module to confirm it's green again**

```bash
(cd apps/relay && go build ./...)
```

Expected: succeeds — this is the first point since Task 3 that the entire `apps/relay` module compiles.

- [ ] **Step 4: Commit**

```bash
git add apps/relay/worker/daemon.go apps/relay/main.go
git commit -m "feat: thread maxAttempts into relay wiring, drop unused outbox repo construction"
```

---

### Task 8: Full verification across all modules

**Files:** none (verification only)

- [ ] **Step 1: Run every module's build and test suite**

```bash
(cd internal/shared && go build ./... && go test ./...)
(cd apps/relay     && go build ./... && go test ./...)
(cd apps/api       && go build ./... && go test ./...)
(cd apps/utils     && go build ./... && go test ./...)
```

Expected: all succeed. `apps/api` and `apps/utils` are unaffected by this refactor (neither imports `internal/shared/aws`'s send path or `apps/relay`), so this mainly confirms no cross-module regression slipped in.

- [ ] **Step 2: Sanity-grep for stale references**

```bash
grep -rn "Envelop\b\|OccuredAt\|ToMessage\b" --include="*.go" . || echo "clean"
```

Expected: no matches (or only unrelated hits you inspect by hand — there should be none, since Tasks 1 and 4 removed every reference).

- [ ] **Step 3: `go mod tidy` + `go work sync`**

```bash
(cd internal/shared && go mod tidy)
(cd apps/relay && go mod tidy)
go work sync
```

Expected: no unexpected diffs (only the `testify`/`go.uber.org/mock` additions from earlier tasks settling).

- [ ] **Step 4: Commit if `go mod tidy` changed anything**

```bash
git status --short
# only if there are changes:
git add -A
git commit -m "chore: go mod tidy after envelope/publisher refactor"
```

## Non-goals

- `docker/relay/Dockerfile`, `publish-images.yml` image matrix, cross-repo Terraform (ECR repo, ECS task role) — deferred to a follow-up "ship it" plan.
- Documentation updates (ADR-018 naming convention record, `facts.md`, `issues.md` #130, `apps/relay/CLAUDE.md`, root `CLAUDE.md`'s relay-gap note) — deferred to the same follow-up.
- Fixing `Get`/`GetAll`'s error-swallowing in the outbox repository (pre-existing, unrelated).
- `VisibilityExtender`'s three known gaps and `Consumer`/`ConsumerManager` wiring — consumer-side, out of scope for a producer-only relay.
- Any ADR-019 observability beyond the existing `slog.Warn` on a retried batch.
