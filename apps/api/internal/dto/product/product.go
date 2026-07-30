package product

import (
	"commerce/api/internal/dto/category"
	"commerce/api/internal/dto/review"
	"commerce/internal/shared/models"
)

type Product struct {
	Id          uint                `json:"id"`
	Name        string              `json:"name"`
	Price       float32             `json:"price"`
	Description string              `json:"description"`
	Sku         string              `json:"sku"`
	Stock       int                 `json:"stock"`
	IsActive    bool                `json:"is_active"`
	IsFeatured  bool                `json:"is_featured"`
	Categories  []category.Category `json:"categories,omitempty"`
	Reviews     []review.Review     `json:"reviews,omitempty"`
}

func FromModel(product *models.Product) *Product {
	categories := make([]category.Category, len(product.ProductCategories))
	for i, pc := range product.ProductCategories {
		categories[i] = *category.FromModel(&pc.Category)
	}

	reviews := make([]review.Review, len(product.Reviews))
	for i, r := range product.Reviews {
		reviews[i] = *review.FromModel(&r)
	}

	return &Product{
		Id:          product.Id,
		Name:        product.Name,
		Price:       product.Price,
		Description: product.Description,
		Sku:         product.Sku,
		Stock:       product.Stock,
		IsActive:    product.IsActive,
		IsFeatured:  product.IsFeatured,
		Categories:  categories,
		Reviews:     reviews,
	}
}

func FromAllModels(products []*models.Product) []*Product {
	dtos := make([]*Product, 0, len(products))
	for _, product := range products {
		dtos = append(dtos, FromModel(product))
	}
	return dtos
}

func ToModel(product *Product) *models.Product {
	return &models.Product{
		Name:        product.Name,
		Price:       product.Price,
		Description: product.Description,
		Sku:         product.Sku,
		Stock:       product.Stock,
		IsActive:    product.IsActive,
		IsFeatured:  product.IsFeatured,
	}
}
