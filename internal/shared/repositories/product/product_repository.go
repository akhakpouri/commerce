package product

import (
	"commerce/internal/shared/models"
	"time"

	"gorm.io/gorm"
)

type ProductRepositoryI interface {
	GetById(id uint) (*models.Product, error)
	GetAll() ([]*models.Product, error)
	GetAllByCategoryId(categoryId uint) ([]*models.Product, error)
	Save(product *models.Product) error
	Delete(id uint, hard bool) error
}

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepositoryI {
	return &ProductRepository{db: db}
}

// Delete implements [ProductRepositoryI].
func (p *ProductRepository) Delete(id uint, hard bool) error {
	if hard {
		return p.db.Delete(models.Product{}, id).Error
	}
	var product models.Product
	if err := p.db.First(&product, id).Error; err != nil {
		return err
	}
	product.DeletedDate = time.Now()
	return p.db.Save(&product).Error
}

// GetAll implements [ProductRepositoryI].
func (p *ProductRepository) GetAll() ([]*models.Product, error) {
	var products []*models.Product
	if err := p.db.Order("created_date desc").Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// GetAllByCategoryId implements [ProductRepositoryI].
func (p *ProductRepository) GetAllByCategoryId(categoryId uint) ([]*models.Product, error) {
	var products []*models.Product
	if err := p.db.
		Joins("JOIN product_categories on product_categories.product_id = products.id").
		Where("product_categories.category_id = ?", categoryId).
		Order("name").
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// GetById implements [ProductRepositoryI].
func (p *ProductRepository) GetById(id uint) (*models.Product, error) {
	var product models.Product
	if err := p.db.First(&product, id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

// Save implements [ProductRepositoryI].
func (p *ProductRepository) Save(product *models.Product) error {
	if product.Id == 0 {
		return p.db.Create(product).Error
	}
	return p.db.Save(product).Error
}
