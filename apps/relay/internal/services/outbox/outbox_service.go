package outbox

import (
	repo "commerce/internal/shared/repositories/outbox"
	dto "commerce/relay/internal/dto/outbox"
	"log/slog"
)

type OutboxServiceI interface {
	Get(id uint) (*dto.Outbox, error)
	GetAll() ([]*dto.Outbox, error)
	GetNextBatch(limit int) ([]*dto.Outbox, error)
	MarkPublished(ids []uint) error
	Delete(id uint) error
}

type OutboxService struct {
	repo repo.OutboxRepositoryI
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

func NewOutboxService(repo repo.OutboxRepositoryI) OutboxServiceI {
	return &OutboxService{
		repo: repo,
	}
}
