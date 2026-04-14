package orderitem

import (
	"commerce/internal/shared/models"
	"time"

	"gorm.io/gorm"
)

type OrderItemRepositoryI interface {
	GetById(id uint) (*models.OrderItem, error)
	GetAllByOrder(orderId uint) ([]*models.OrderItem, error)
	Save(item *models.OrderItem) error
	Delete(id uint, hard bool) error
}

type OrderItemRepository struct {
	db *gorm.DB
}

func NewOrderItemRepository(db *gorm.DB) OrderItemRepositoryI {
	return &OrderItemRepository{db: db}
}

func (r *OrderItemRepository) GetById(id uint) (*models.OrderItem, error) {
	var item models.OrderItem
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *OrderItemRepository) GetAllByOrder(orderId uint) ([]*models.OrderItem, error) {
	var items []*models.OrderItem
	if err := r.db.Where("order_id = ?", orderId).
		Order("created_date desc").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *OrderItemRepository) Save(item *models.OrderItem) error {
	if item.Id == 0 {
		return r.db.Create(item).Error
	}
	return r.db.Save(item).Error
}

func (r *OrderItemRepository) Delete(id uint, hard bool) error {
	if hard {
		return r.db.Delete(&models.OrderItem{}, id).Error
	}
	var orderItem models.OrderItem
	if err := r.db.First(&orderItem, id).Error; err != nil {
		return err
	}
	orderItem.DeletedDate = time.Now()
	return r.db.Save(&orderItem).Error
}
