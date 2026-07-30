package product

import (
	dto "commerce/api/internal/dto/product"
	repo "commerce/internal/shared/repositories/product"
	"log/slog"
)

type ProductServiceI interface {
	GetById(id uint) (*dto.Product, error)
	GetAll() ([]*dto.Product, error)
	GetAllByCategory(categoryId uint) ([]*dto.Product, error)
	Save(product *dto.Product) error
	Delete(id uint, hard bool) error
}

type ProductService struct {
	repo repo.ProductRepositoryI
}

func NewProductService(repo repo.ProductRepositoryI) ProductServiceI {
	return &ProductService{repo: repo}
}

// Delete implements [ProductServiceI].
func (p *ProductService) Delete(id uint, hard bool) error {
	return p.repo.Delete(id, hard)
}

// GetAll implements [ProductServiceI].
func (p *ProductService) GetAll() ([]*dto.Product, error) {
	models, err := p.repo.GetAll()
	if err != nil {
		slog.Error("Exception thrown when getting all product", "error", err)
		return nil, err
	}
	return dto.FromAllModels(models), nil
}

// GetAllByCategory implements [ProductServiceI].
func (p *ProductService) GetAllByCategory(categoryId uint) ([]*dto.Product, error) {
	models, err := p.repo.GetAllByCategoryId(categoryId)
	if err != nil {
		slog.Error("Exception thrown when getting product by category", "error", err)
		return nil, err
	}
	return dto.FromAllModels(models), nil
}

// GetById implements [ProductServiceI].
func (p *ProductService) GetById(id uint) (*dto.Product, error) {
	model, err := p.repo.GetById(id)
	if err != nil {
		slog.Error("Exception thrown when getting product by id", "error", err)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// Save implements [ProductServiceI].
func (p *ProductService) Save(product *dto.Product) error {
	model := dto.ToModel(product)
	return p.repo.Save(model)
}
