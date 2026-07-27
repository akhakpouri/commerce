package category

import "commerce/internal/shared/models"

type Category struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
	ParentId    *uint  `json:"parent_id"`
	IsActive    bool   `json:"is_active"`
}

func FromModel(category *models.Category) *Category {
	return &Category{
		Id:          category.Id,
		Name:        category.Name,
		Description: category.Description,
		Slug:        category.Slug,
		ParentId:    category.ParentId,
		IsActive:    category.IsActive,
	}
}

func FromAllModels(categories []*models.Category) []*Category {
	dtos := make([]*Category, 0, len(categories))
	for _, category := range categories {
		dtos = append(dtos, FromModel(category))
	}
	return dtos
}

func ToModel(category *Category) *models.Category {
	return &models.Category{
		Name:        category.Name,
		Description: category.Description,
		Slug:        category.Slug,
		ParentId:    category.ParentId,
		IsActive:    category.IsActive,
	}
}
