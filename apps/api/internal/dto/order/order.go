package order

import (
	orderitem "commerce/api/internal/dto/order-item"
	"commerce/internal/shared/models"
)

type Order struct {
	Id             uint                  `json:"id"`
	UserId         uint                  `json:"user_id"`
	Status         string                `json:"status"`
	TaxAmount      float64               `json:"tax_amount"`
	TotalAmount    float64               `json:"total_amount"`
	SubTotalAmount float64               `json:"sub_total_amount"`
	BillingState   string                `json:"billing_state"`
	OrderItems     []orderitem.OrderItem `json:"order_items,omitempty"`
}

func FromModel(order *models.Order) *Order {
	orderItems := make([]orderitem.OrderItem, len(order.OrderItems))
	for i, item := range order.OrderItems {
		orderItems[i] = *orderitem.FromModel(&item)
	}

	return &Order{
		Id:             order.Id,
		UserId:         order.UserId,
		TotalAmount:    order.TotalAmount,
		TaxAmount:      order.TaxAmount,
		Status:         string(order.Status),
		OrderItems:     orderItems,
		SubTotalAmount: order.SubTotalAmount,
		BillingState:   order.BillingAddress.State,
	}
}

func ToModel(order *Order) *models.Order {
	orderItems := make([]models.OrderItem, len(order.OrderItems))
	for i, item := range order.OrderItems {
		orderItems[i] = *orderitem.ToModel(&item)
	}

	return &models.Order{
		UserId:         order.UserId,
		TotalAmount:    order.TotalAmount,
		TaxAmount:      order.TaxAmount,
		Status:         models.OrderStatus(order.Status),
		SubTotalAmount: order.SubTotalAmount,
		OrderItems:     orderItems,
	}
}
