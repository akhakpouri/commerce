package payment

import (
	"commerce/internal/shared/models"
	"time"

	"gorm.io/gorm"
)

type PaymentRepositoryI interface {
	GetById(id uint) (*models.Payment, error)
	GetAll() ([]*models.Payment, error)
	GetByOrder(orderId uint) ([]*models.Payment, error)
	Save(payment *models.Payment) error
	Delete(id uint, hard bool) error
	UpdateStatus(id uint, status string) error
}

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepositoryI {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) GetById(id uint) (*models.Payment, error) {
	payment := models.Payment{}
	if err := r.db.First(&payment, id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) GetAll() ([]*models.Payment, error) {
	payments := []*models.Payment{}
	if err := r.db.Order("created_date desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *PaymentRepository) GetByOrder(orderId uint) ([]*models.Payment, error) {
	payments := []*models.Payment{}
	if err := r.db.Where("order_id = ?", orderId).Order("created_date desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *PaymentRepository) Save(payment *models.Payment) error {
	if payment.Id == 0 {
		return r.db.Create(payment).Error
	}
	return r.db.Save(payment).Error
}

// UpdateStauts implements [PaymentRepositoryI].
func (r *PaymentRepository) UpdateStatus(id uint, status string) error {
	model, err := r.GetById(id)
	if err != nil {
		return err
	}
	model.Status = models.PaymentStatus(status)
	return r.Save(model)
}

func (r *PaymentRepository) Delete(id uint, hard bool) error {
	if hard {
		return r.db.Delete(&models.Payment{}, id).Error
	}
	payment := models.Payment{}
	if err := r.db.First(&payment, id).Error; err != nil {
		return err
	}
	payment.DeletedDate = time.Now()
	return r.db.Save(&payment).Error
}
