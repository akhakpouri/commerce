package orderitem

import "commerce/internal/shared/models"

type OrderItem struct {
	Id        uint    `json:"id"`
	OrderId   uint    `json:"order_id"`
	ProductId uint    `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

func FromModel(orderItem *models.OrderItem) *OrderItem {
	return &OrderItem{
		Id:        orderItem.Id,
		OrderId:   orderItem.OrderId,
		ProductId: orderItem.ProductId,
		Quantity:  orderItem.Quantity,
		UnitPrice: orderItem.UnitPrice,
	}
}

func ToModel(orderItem *OrderItem) *models.OrderItem {
	return &models.OrderItem{
		Base: models.Base{
			Id: orderItem.Id,
		},
		OrderId:   orderItem.OrderId,
		ProductId: orderItem.ProductId,
		Quantity:  orderItem.Quantity,
		UnitPrice: orderItem.UnitPrice,
	}
}
