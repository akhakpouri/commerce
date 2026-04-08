package payment

import (
	dto "commerce/api/internal/dto/payment"
	model "commerce/internal/shared/models"
	repo "commerce/internal/shared/repositories/payment"
	"fmt"
	"log/slog"
)

type PaymentServiceI interface {
	GetById(id uint) (*dto.Payment, error)
	GetByOrder(orderId uint) ([]*dto.Payment, error)
	GetStatuses() []dto.PaymentStatus
	Delete(id uint, hard bool) error
	Save(payment *dto.Payment) error
	UpdateStatus(id uint, status string) error
}

type PaymentService struct {
	repo repo.PaymentRepositoryI
}

func NewPaymentService(repo repo.PaymentRepositoryI) PaymentServiceI {
	return &PaymentService{repo: repo}
}

// Delete implements [PaymentServiceI].
func (p *PaymentService) Delete(id uint, hard bool) error {
	return p.repo.Delete(id, hard)
}

// GetById implements [PaymentServiceI].
func (p *PaymentService) GetById(id uint) (*dto.Payment, error) {
	model, err := p.repo.GetById(id)
	if err != nil {
		slog.Error("Exception occured when getting payment by id", "id", id, "error", err)
		return nil, err
	}
	return dto.FromModel(model), nil
}

// GetByOrder implements [PaymentServiceI].
func (p *PaymentService) GetByOrder(orderId uint) ([]*dto.Payment, error) {
	models, err := p.repo.GetByOrder(orderId)
	if err != nil {
		slog.Error("Exception occured when getting payments by order", "orderId", orderId, "error", err)
		return nil, err
	}
	payments := make([]*dto.Payment, 0, len(models))
	for _, model := range models {
		payments = append(payments, dto.FromModel(model))
	}
	return payments, nil
}

// GetStatuses implements [PaymentServiceI].
func (p *PaymentService) GetStatuses() []dto.PaymentStatus {
	var statuses []dto.PaymentStatus
	for k := range validStatuses {
		statuses = append(statuses, dto.PaymentStatus{Status: string(k)})
	}
	return statuses
}

// Save implements [PaymentServiceI].
func (p *PaymentService) Save(payment *dto.Payment) error {
	dto := dto.ToModel(payment)
	return p.repo.Save(dto)
}

// UpdateStatus implements [PaymentServiceI].
func (p *PaymentService) UpdateStatus(id uint, status string) error {
	if !isPaymentStatusValid(status) {
		slog.Error("Payment status doesn't exist.", "status", status)
		return fmt.Errorf("invalid payment status: %s", status)
	}
	return p.repo.UpdateStatus(id, status)
}

var validStatuses = map[model.PaymentStatus]struct{}{
	model.PaymentStatusPending:           {},
	model.PaymentStatusCompleted:         {},
	model.PaymentStatusAuthorized:        {},
	model.PaymentStatusCaptured:          {},
	model.PaymentStatusFailed:            {},
	model.PaymentStatusRefunded:          {},
	model.PaymentStatusPartiallyRefunded: {},
}

func isPaymentStatusValid(status string) bool {
	_, ok := validStatuses[model.PaymentStatus(status)]
	return ok
}
