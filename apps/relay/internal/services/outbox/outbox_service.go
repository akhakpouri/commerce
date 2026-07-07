package outbox

import (
	manager "commerce/internal/shared/managers/transaction"
	repo "commerce/internal/shared/repositories/outbox"
	dto "commerce/relay/internal/dto/outbox"
	"fmt"
	"log/slog"
)

type OutboxServiceI interface {
	Get(id uint) (*dto.Outbox, error)
	GetAll() ([]*dto.Outbox, error)
	GetNextBatch(limit int) ([]*dto.Outbox, error)
	MarkPublished(ids []uint) error
	Delete(id uint) error
	ProcessBatch(limit int) error
}

type OutboxService struct {
	repo    repo.OutboxRepositoryI
	manager manager.ManagerI
}

// ProcessBatch implements [OutboxServiceI].
func (o *OutboxService) ProcessBatch(limit int) error {
	return o.manager.Execute(func(r manager.RepositoriesI) error {
		outboxes, err := r.Outbox().GetNextBatch(limit)
		if err != nil {
			slog.Error("Exception occured getting the next batch.", "error", err, "limit", limit)
			return err
		}
		ids := make([]uint, 0, len(outboxes))
		for i, outbox := range outboxes {
			fmt.Printf("current index is %d. aggregate-id is %d, event-id is %d, event-type is %s, payload is %s\n",
				i, outbox.AggregateId,
				outbox.EventId,
				outbox.EventType,
				outbox.Payload)
			ids = append(ids, outbox.Id)
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

func NewOutboxService(repo repo.OutboxRepositoryI, manager manager.ManagerI) OutboxServiceI {
	return &OutboxService{
		repo:    repo,
		manager: manager,
	}
}
