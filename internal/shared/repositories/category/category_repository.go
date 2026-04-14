package category

import (
	"commerce/internal/shared/models"
	"time"

	"gorm.io/gorm"
)

type CategoryRepositoryI interface {
	GetById(id uint) (*models.Category, error)
	GetByParentId(parentId uint) ([]*models.Category, error)
	GetAll() ([]*models.Category, error)
	Save(category *models.Category) error
	Delete(id uint, hard bool) error
}

type CategoryRepository struct {
	db *gorm.DB
}

// Repository constructor returns the interface
func NewCategoryRepository(db *gorm.DB) CategoryRepositoryI {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) GetById(id uint) (*models.Category, error) {
	var category models.Category
	if err := r.db.First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) GetByParentId(parentId uint) ([]*models.Category, error) {
	var categories []*models.Category
	if err := r.db.Where("parent_id = ?", parentId).Order("name asc").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) GetAll() ([]*models.Category, error) {
	var categories []*models.Category
	if err := r.db.Order("name asc").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) Save(category *models.Category) error {
	if category.Id == 0 {
		return r.db.Create(category).Error
	}
	return r.db.Save(category).Error
}

func (r *CategoryRepository) Delete(id uint, hard bool) error {
	if hard {
		return r.db.Delete(&models.Category{}, id).Error
	}
	var category models.Category
	if err := r.db.First(&category, id).Error; err != nil {
		return err
	}
	category.DeletedDate = time.Now()
	return r.db.Save(&category).Error
}
