package orderitem

import (
	dto "commerce/api/internal/dto/order-item"
	repo "commerce/internal/shared/repositories/order-item"
	"log/slog"
)

type OrderItemServiceI interface {
	GetById(id uint) (*dto.OrderItem, error)
	GetAllByOrder(orderId uint) ([]*dto.OrderItem, error)
	Save(orderItem dto.OrderItem) error
	Delete(id uint, hard bool) error
}

type OrderItemService struct {
	repo repo.OrderItemRepositoryI
}

func NewOrderItemService(repo repo.OrderItemRepositoryI) OrderItemServiceI {
	return &OrderItemService{repo: repo}
}

// Delete implements [OrderItemServiceI].
func (o *OrderItemService) Delete(id uint, hard bool) error {
	return o.repo.Delete(id, hard)
}

// GetAllByOrder implements [OrderItemServiceI].
func (o *OrderItemService) GetAllByOrder(orderId uint) ([]*dto.OrderItem, error) {
	models, err := o.repo.GetAllByOrder(orderId)
	if err != nil {
		slog.Error("Exception occurred retrieving items by order", "order-id", orderId, "error", err)
		return nil, err
	}
	orderItems := make([]*dto.OrderItem, 0, len(models))
	for _, model := range models {
		orderItems = append(orderItems, dto.FromModel(model))
	}
	return orderItems, nil
}

// GetById implements [OrderItemServiceI].
func (o *OrderItemService) GetById(id uint) (*dto.OrderItem, error) {
	model, err := o.repo.GetById(id)
	if err != nil {
		slog.Error("Exception occurred retrieving order-item by id", "id", id, "error", err)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// Save implements [OrderItemServiceI].
func (o *OrderItemService) Save(orderItem dto.OrderItem) error {
	return o.repo.Save(dto.ToModel(&orderItem))
}
