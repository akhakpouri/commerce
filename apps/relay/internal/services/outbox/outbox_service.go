package outbox

import (
	"context"
	"log/slog"

	manager "commerce/internal/shared/managers/transaction"
	repo "commerce/internal/shared/repositories/outbox"
	dto "commerce/relay/internal/dto/outbox"
	"commerce/relay/internal/publisher"
)

type OutboxServiceI interface {
	Get(id uint) (*dto.Outbox, error)
	GetAll() ([]*dto.Outbox, error)
	GetNextBatch(limit int) ([]*dto.Outbox, error)
	MarkPublished(ids []uint) error
	Delete(id uint) error
	ProcessBatch(ctx context.Context, limit int) error
}

type OutboxService struct {
	repo      repo.OutboxRepositoryI
	manager   manager.ManagerI
	publisher *publisher.SqsPublisher
}

// ProcessBatch implements [OutboxServiceI]. It claims up to limit unpublished
// events, publishes them, and marks them published - all inside one
// transaction, so the SELECT ... FOR UPDATE SKIP LOCKED lock is held for the
// whole claim-publish-mark window instead of being released after the SELECT.
func (o *OutboxService) ProcessBatch(ctx context.Context, limit int) error {
	return o.manager.Execute(func(r manager.RepositoriesI) error {
		models, err := r.Outbox().GetNextBatch(limit)
		if err != nil {
			slog.Error("Exception occured getting the next batch.", "error", err, "limit", limit)
			return err
		}
		if len(models) == 0 {
			return nil
		}

		events := dto.FromAllModels(models)
		if err := o.publisher.Publish(ctx, events); err != nil {
			slog.Error("Exception occured publishing outbox batch.", "error", err)
			return err
		}

		ids := make([]uint, len(events))
		for i, event := range events {
			ids[i] = event.Id
		}
		return r.Outbox().MarkPublished(ids)
	})
}

// Delete implements [OutboxServiceI].
func (o *OutboxService) Delete(id uint) error {
	return o.repo.Delete(id)
}

// Get implements [OutboxServiceI].
func (o *OutboxService) Get(id uint) (*dto.Outbox, error) {
	model, err := o.repo.Get(id)
	if err != nil {
		slog.Error("Exception occured when getting the outbox.", "error", err, "id", id)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// GetAll implements [OutboxServiceI].
func (o *OutboxService) GetAll() ([]*dto.Outbox, error) {
	models, err := o.repo.GetAll()
	if err != nil {
		slog.Error("Exception occured when getting all outbox.", "error", err)
		return nil, err
	}
	dtos := dto.FromAllModels(models)
	return dtos, nil
}

// GetNextBatch implements [OutboxServiceI].
func (o *OutboxService) GetNextBatch(limit int) ([]*dto.Outbox, error) {
	models, err := o.repo.GetNextBatch(limit)
	if err != nil {
		slog.Error("Exception occured when getting next batch.", "error", err, "limit", limit)
		return nil, err
	}
	dtos := dto.FromAllModels(models)
	return dtos, nil
}

// MarkPublished implements [OutboxServiceI].
func (o *OutboxService) MarkPublished(ids []uint) error {
	return o.repo.MarkPublished(ids)
}

func NewOutboxService(
	repo repo.OutboxRepositoryI,
	manager manager.ManagerI,
	publisher *publisher.SqsPublisher) OutboxServiceI {
	return &OutboxService{
		repo:      repo,
		manager:   manager,
		publisher: publisher,
	}
}
