package address

import (
	"commerce/internal/shared/models"
	"time"

	"gorm.io/gorm"
)

type AddressRepositoryI interface {
	GetById(id uint) (*models.Address, error)
	GetByUserId(userId uint) ([]*models.Address, error)
	GetAll() ([]*models.Address, error)
	Save(address *models.Address) error
	Delete(id uint, hard bool) error
}

type AddressRepository struct {
	db *gorm.DB
}

// Repository constructor returns the interface
func NewAddressRepository(db *gorm.DB) AddressRepositoryI {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) GetById(id uint) (*models.Address, error) {
	var address models.Address
	if err := r.db.First(&address, id).Error; err != nil {
		return nil, err
	}
	return &address, nil
}

func (r *AddressRepository) GetByUserId(userId uint) ([]*models.Address, error) {
	var addresses []*models.Address
	if err := r.db.Where("user_id = ?", userId).Order("created_date desc").Find(&addresses).Error; err != nil {
		return nil, err
	}
	return addresses, nil
}

func (r *AddressRepository) GetAll() ([]*models.Address, error) {
	var addresses []*models.Address
	if err := r.db.Find(&addresses).Error; err != nil {
		return nil, err
	}
	return addresses, nil
}

func (r *AddressRepository) Save(address *models.Address) error {
	if address.Id == 0 {
		return r.db.Create(address).Error
	}
	return r.db.Save(address).Error
}

func (r *AddressRepository) Delete(id uint, hard bool) error {
	if hard {
		return r.db.Delete(&models.Address{}, id).Error
	}
	var address models.Address
	if err := r.db.First(&address, id).Error; err != nil {
		return err
	}
	address.DeletedDate = time.Now()
	return r.db.Save(&address).Error
}
