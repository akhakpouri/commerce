package review

import (
	dto "commerce/api/internal/dto/review"
	repo "commerce/internal/shared/repositories/review"
	"log/slog"
)

type ReviewServiceI interface {
	GetById(id uint) (*dto.Review, error)
	GetAllByProduct(productId uint) ([]*dto.Review, error)
	Save(review *dto.Review) error
	Delete(id uint, hard bool) error
}

type ReviewService struct {
	repo repo.ReviewRepositoryI
}

func NewReviewService(repo repo.ReviewRepositoryI) ReviewServiceI {
	return &ReviewService{repo: repo}
}

// Delete implements [ReviewServiceI].
func (r *ReviewService) Delete(id uint, hard bool) error {
	return r.repo.Delete(id, hard)
}

// GetAllByProduct implements [ReviewServiceI].
func (r *ReviewService) GetAllByProduct(productId uint) ([]*dto.Review, error) {
	models, err := r.repo.GetByProductId(productId)
	if err != nil {
		slog.Error("Exception occured in get reviews by product", "productId", productId, "error", err)
		return nil, err
	}
	return dto.FromAllModels(models), nil
}

// GetById implements [ReviewServiceI].
func (r *ReviewService) GetById(id uint) (*dto.Review, error) {
	model, err := r.repo.GetById(id)
	if err != nil {
		slog.Error("Exception occured retreving review by id", "errro", err)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// Save implements [ReviewServiceI].
func (r *ReviewService) Save(review *dto.Review) error {
	model := dto.ToModel(review)
	return r.repo.Save(model)
}
