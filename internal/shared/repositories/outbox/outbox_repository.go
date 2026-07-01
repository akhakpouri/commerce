package outbox

import (
	model "commerce/internal/shared/models"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type OutboxRepositoryI interface {
	Get(id uint) (*model.Outbox, error)
	GetAll() ([]*model.Outbox, error)
	GetNextBatch(limit int) ([]*model.Outbox, error)
	MarkPublished(ids []uint) error
	Delete(id uint) error
	Save(outbox *model.Outbox) error
}

type OutboxRepository struct {
	db *gorm.DB
}

// Save implements [OutboxRepositoryI].
func (o *OutboxRepository) Save(outbox *model.Outbox) error {
	
	return o.db.Create(outbox).Error
}

// MarkPublished implements [OutboxRepositoryI].
func (o *OutboxRepository) MarkPublished(ids []uint) error {
	return o.db.Model(&model.Outbox{}).Where("id IN ?", ids).Update("published_at", time.Now()).Error
}

// Delete implements [OutboxRepositoryI].
func (o *OutboxRepository) Delete(id uint) error {
	return o.db.Delete(model.Outbox{}, id).Error
}

// Get implements [OutboxRepositoryI].
func (o *OutboxRepository) Get(id uint) (*model.Outbox, error) {
	var outbox model.Outbox
	if err := o.db.First(&outbox, id).Error; err != nil {
		slog.Error("Couldn't find outbox.", "error", err, "id", id)
	}
	return &outbox, nil
}

// GetAll implements [OutboxRepositoryI].
func (o *OutboxRepository) GetAll() ([]*model.Outbox, error) {
	var outbox []*model.Outbox
	if err := o.db.Find(&outbox).Error; err != nil {
		slog.Error("Error retrieving all outbox from table", "error", err)
	}
	return outbox, nil
}

// GetNextBatch implements [OutboxRepositoryI].
func (o *OutboxRepository) GetNextBatch(limit int) ([]*model.Outbox, error) {
	var outbox []*model.Outbox
	if err := o.db.
		Where("published_at is null").
		Order("id").
		Limit(limit).
		Find(&outbox).Error; err != nil {
		slog.Error("Error retrieving outbox for next batch", "error", err, "limit", limit)
	}
	return outbox, nil
}

func NewOutboxRepository(db *gorm.DB) OutboxRepositoryI {
	return &OutboxRepository{
		db: db,
	}
}
