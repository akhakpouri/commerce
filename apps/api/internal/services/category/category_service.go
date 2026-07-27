package category

import (
	dto "commerce/api/internal/dto/category"
	repo "commerce/internal/shared/repositories/category"
	"log/slog"
)

type CategoryServiceI interface {
	GetById(id uint) (*dto.Category, error)
	GetAll() ([]*dto.Category, error)
	GetAllByParentId(parentId uint) ([]*dto.Category, error)
	Save(category *dto.Category) error
	Delete(id uint, hard bool) error
}

type CategoryService struct {
	repo repo.CategoryRepositoryI
}

func NewCategoryService(repo repo.CategoryRepositoryI) CategoryServiceI {
	return &CategoryService{repo: repo}
}

// Delete implements [CategoryServiceI].
func (c *CategoryService) Delete(id uint, hard bool) error {
	return c.repo.Delete(id, hard)
}

// GetAll implements [CategoryServiceI].
func (c *CategoryService) GetAll() ([]*dto.Category, error) {
	models, err := c.repo.GetAll()
	if err != nil {
		slog.Error("Exception occured while getting all categories.", "error", err)
		return nil, err
	}
	return dto.FromAllModels(models), nil
}

// GetAllByParentId implements [CategoryServiceI].
func (c *CategoryService) GetAllByParentId(parentId uint) ([]*dto.Category, error) {
	models, err := c.repo.GetByParentId(parentId)
	if err != nil {
		slog.Error("Exception occured while getting all categories by parent.", "error", err)
		return nil, err
	}

	return dto.FromAllModels(models), nil
}

// GetById implements [CategoryServiceI].
func (c *CategoryService) GetById(id uint) (*dto.Category, error) {
	model, err := c.repo.GetById(id)
	if err != nil {
		slog.Error("Exception occured while getting category.", "error", err)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// Save implements [CategoryServiceI].
func (c *CategoryService) Save(category *dto.Category) error {
	model := dto.ToModel(category)
	return c.repo.Save(model)
}
