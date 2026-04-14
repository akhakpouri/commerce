package order

import (
	"commerce/internal/shared/models"
	"time"

	"gorm.io/gorm"
)

type OrderRepositoryI interface {
	GetById(id uint) (*models.Order, error)
	GetAll() ([]*models.Order, error)
	GetAllByUserId(userId uint) ([]*models.Order, error)
	Save(order *models.Order) error
	Delete(id uint, hard bool) error
	UpdateStatus(id uint, status string) error
}

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepositoryI {
	return &OrderRepository{db: db}
}

// Delete implements [OrderRepositoryI].
func (o *OrderRepository) Delete(id uint, hard bool) error {
	if hard {
		return o.db.Delete(models.Order{}, id).Error
	}
	var order models.Order
	if err := o.db.First(&order, id).Error; err != nil {
		return err
	}
	order.DeletedDate = time.Now()
	return o.db.Save(&order).Error
}

// GetAll implements [OrderRepositoryI].
func (o *OrderRepository) GetAll() ([]*models.Order, error) {
	var orders []*models.Order
	if err := o.db.Order("created_date desc").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// GetAllByUserId implements [OrderRepositoryI].
func (o *OrderRepository) GetAllByUserId(userId uint) ([]*models.Order, error) {
	var orders []*models.Order
	if err := o.db.
		Preload("BillingAddress").
		Where("user_id = ?", userId).
		Order("created_date desc").
		Find(&orders).
		Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// GetById implements [OrderRepositoryI].
func (o *OrderRepository) GetById(id uint) (*models.Order, error) {
	var order models.Order
	if err := o.db.Preload("BillingAddress").First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// Save implements [OrderRepositoryI].
func (o *OrderRepository) Save(order *models.Order) error {
	if order.Id == 0 {
		return o.db.Create(order).Error
	}
	return o.db.Save(order).Error
}

// UpdateStatus implements [OrderRepositoryI].
func (o *OrderRepository) UpdateStatus(id uint, status string) error {
	return o.db.
		Model(&models.Order{}).
		Where("id = ?", id).
		Update("order_status", status).Error
}
