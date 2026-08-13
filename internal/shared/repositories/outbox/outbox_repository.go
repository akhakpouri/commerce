package outbox

import (
	model "commerce/internal/shared/models"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OutboxRepositoryI interface {
	Get(id uint) (*model.Outbox, error)
	GetAll() ([]*model.Outbox, error)
	GetNextBatch(limit int, maxAttempts int) ([]*model.Outbox, error)
	IncrementAttempts(ids []uint) error
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

func NewOutboxRepository(db *gorm.DB) OutboxRepositoryI {
	return &OutboxRepository{
		db: db,
	}
}
