package review

import (
	"commerce/internal/shared/models"
	"time"

	"gorm.io/gorm"
)

type ReviewRepositoryI interface {
	GetById(id uint) (*models.Review, error)
	GetByProductId(productId uint) ([]*models.Review, error)
	Save(review *models.Review) error
	Delete(id uint, hard bool) error
}

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) ReviewRepositoryI {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) GetById(id uint) (*models.Review, error) {
	var review models.Review
	if err := r.db.First(&review, id).Error; err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *ReviewRepository) GetByProductId(productId uint) ([]*models.Review, error) {
	var reviews []*models.Review
	if err := r.db.Where("product_id = ?", productId).Order("created_date desc").Find(&reviews).Error; err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *ReviewRepository) Save(review *models.Review) error {
	if review.Id == 0 {
		return r.db.Create(review).Error
	}
	return r.db.Save(review).Error
}

func (r *ReviewRepository) Delete(id uint, hard bool) error {
	if hard {
		return r.db.Delete(&models.Review{}, id).Error
	} else {
		var review models.Review
		if err := r.db.First(&review, id).Error; err != nil {
			return err
		}
		review.DeletedDate = time.Now()
		return r.db.Save(&review).Error
	}
}
