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
