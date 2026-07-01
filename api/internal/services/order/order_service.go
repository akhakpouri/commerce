package order

import (
	dto "commerce/api/internal/dto/order"
	tax_service "commerce/api/internal/services/tax"
	manager "commerce/internal/shared/managers/transaction"
	models "commerce/internal/shared/models"
	repo "commerce/internal/shared/repositories/order"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type OrderServiceI interface {
	GetById(id uint) (*dto.Order, error)
	GetByUserId(userId uint) ([]*dto.Order, error)
	GetStatuses() []dto.OrderStatus
	Save(order dto.Order) error
	Delete(id uint, hard bool) error
	UpdateStatus(id uint, status string) error
}

type OrderService struct {
	repo       repo.OrderRepositoryI
	taxService tax_service.TaxServiceI
	manager    manager.ManagerI
}

func NewOrderService(repo repo.OrderRepositoryI, taxService tax_service.TaxServiceI, manager manager.ManagerI) OrderServiceI {
	return &OrderService{
		repo:       repo,
		taxService: taxService,
		manager:    manager,
	}
}

// Delete implements [OrderServiceI].
func (o *OrderService) Delete(id uint, hard bool) error {
	return o.repo.Delete(id, hard)
}

// GetById implements [OrderServiceI].
func (o *OrderService) GetById(id uint) (*dto.Order, error) {
	model, err := o.repo.GetById(id)
	if err != nil {
		slog.Error("Exception occurred getting order by id.", "id", id, "error", err)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// GetStatuses implements [OrderServiceI].
func (o *OrderService) GetStatuses() []dto.OrderStatus {
	statuses := []dto.OrderStatus{}

	for key := range validStatuses {
		statuses = append(statuses, dto.OrderStatus{Status: string(key)})
	}
	return statuses
}

// GetByUserId implements [OrderServiceI].
func (o *OrderService) GetByUserId(userId uint) ([]*dto.Order, error) {
	models, err := o.repo.GetAllByUserId(userId)
	if err != nil {
		slog.Error("Exception occurred getting orders by user", "userId", userId, "error", err)
		return nil, err
	}
	orders := make([]*dto.Order, 0, len(models))
	for _, model := range models {
		orders = append(orders, dto.FromModel(model))
	}
	return orders, nil
}

// Save implements [OrderServiceI].
func (o *OrderService) Save(order dto.Order) error {
	order.SubTotalAmount = calculateSubTotalAmount(&order)
	tax, err := o.calculateTax(&order)
	if err != nil {
		return err
	}
	order.TaxAmount = tax
	order.TotalAmount = calculateTotalAmount(&order)
	model := dto.ToModel(&order)
	return o.manager.Execute(func(r manager.RepositoriesI) error {

		if err := r.Order().Save(model); err != nil {
			slog.Error("Exception occured when saving order", "error", err)
			return err
		}

		payload, err := json.Marshal(model)
		if err != nil {
			slog.Error("Exception occured when converting model to json")
			return err
		}
		event := &models.Outbox{
			EventId:       uuid.New(),
			EventType:     "OrderPlaced",
			AggregateId:   model.Id,
			AggregateType: "Order",
			Payload:       payload,
		}

		if err := r.Outbox().Save(event); err != nil {
			slog.Error("Exception occured when saving outbox event", "error", err)
			return err
		}
		return nil
	})
}

// UpdateStatus implements [OrderServiceI].
func (o *OrderService) UpdateStatus(id uint, status string) error {
	if !isOrderStatusValid(status) {
		slog.Error("Order status doesn't exist.", "status", status)
		return fmt.Errorf("invalid order status: %s", status)
	}
	return o.repo.UpdateStatus(id, status)
}

var validStatuses = map[models.OrderStatus]struct{}{
	models.OrderStatusPending:   {},
	models.OrderStatusDelivered: {},
	models.OrderStatusShipped:   {},
	models.OrderStatusCancelled: {},
}

func isOrderStatusValid(status string) bool {
	_, ok := validStatuses[models.OrderStatus(status)]
	return ok
}

func (o *OrderService) calculateTax(order *dto.Order) (float64, error) {
	tax, err := o.taxService.Calculate(order.SubTotalAmount, order.BillingState)
	if err != nil {
		slog.Error("Exception occured when calculating order tax.", "order-id", order.Id, "state", order.BillingState)
		return 0, err
	}
	return *tax, nil
}

func calculateTotalAmount(order *dto.Order) float64 {
	return order.SubTotalAmount + order.TaxAmount
}

func calculateSubTotalAmount(o *dto.Order) float64 {
	total := 0.00

	for _, item := range o.OrderItems {
		total += (item.UnitPrice * float64(item.Quantity))
	}
	return total
}
