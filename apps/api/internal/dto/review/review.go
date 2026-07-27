package review

import "commerce/internal/shared/models"

type Review struct {
	Id        uint   `json:"id"`
	ProductId uint   `json:"product_id"`
	UserId    uint   `json:"user_id"`
	Rating    int    `json:"rating"`
	Title     string `json:"title"`
	Comment   string `json:"comment"`
}

func FromModel(review *models.Review) *Review {
	return &Review{
		Id:        review.Id,
		ProductId: review.ProductId,
		UserId:    review.UserId,
		Rating:    review.Rating,
		Title:     review.Title,
		Comment:   review.Comment,
	}
}

func FromAllModels(reviews []*models.Review) []*Review {
	dtos := make([]*Review, 0, len(reviews))
	for _, review := range reviews {
		dtos = append(dtos, FromModel(review))
	}
	return dtos
}

func ToModel(review *Review) *models.Review {
	return &models.Review{
		ProductId: review.ProductId,
		UserId:    review.UserId,
		Rating:    review.Rating,
		Title:     review.Title,
		Comment:   review.Comment,
	}
}
