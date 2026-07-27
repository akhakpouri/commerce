package payment

import (
	"commerce/internal/shared/models"
	"time"
)

type Payment struct {
	Id            uint    `json:"id"`
	OrderId       uint    `json:"order_id"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	Status        string  `json:"status"`
	Currency      string  `json:"currency"`
	Gateway       string  `json:"gateway"`
	PaidAt        string  `json:"paid_at"`
}

func FromModel(payment *models.Payment) *Payment {
	return &Payment{
		Id:            payment.Id,
		OrderId:       payment.OrderId,
		Amount:        payment.Amount,
		PaymentMethod: string(payment.PaymentMethod),
		Status:        string(payment.Status),
		Currency:      payment.Currency,
		Gateway:       string(payment.PaymentGateway),
		PaidAt: func() string {
			if payment.PaidAt != nil {
				return payment.PaidAt.Format("01/02/2006 15:04:05")
			}
			return ""
		}(),
	}
}

func ToModel(payment *Payment) *models.Payment {
	return &models.Payment{
		OrderId:        payment.OrderId,
		Amount:         payment.Amount,
		PaymentMethod:  models.PaymentMethod(payment.PaymentMethod),
		Status:         models.PaymentStatus(payment.Status),
		Currency:       payment.Currency,
		PaymentGateway: models.PaymentGateway(payment.Gateway),
		PaidAt:         getTimeString(payment.PaidAt),
	}
}

func getTimeString(t string) *time.Time {
	layout := "01/02/2006 15:04:05"
	parsedTime, err := time.Parse(layout, t)
	if err != nil {
		return nil
	}
	return &parsedTime
}
