package order

import (
	"fmt"
	"testing"
	"time"

	tax_service "commerce/api/internal/services/tax"
	"commerce/internal/shared/models"

	dto "commerce/api/internal/dto/order"
	orderitem "commerce/api/internal/dto/order-item"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setup(t *testing.T) (*MockOrderRepositoryI, OrderServiceI) {
	t.Helper()
	ctl := gomock.NewController(t)
	t.Cleanup(ctl.Finish)
	mockRepo := NewMockOrderRepositoryI(ctl)
	taxService := tax_service.NewTaxService()
	return mockRepo, NewOrderService(mockRepo, taxService)
}

func TestGetbyId(t *testing.T) {
	id := uint(1)
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().GetById(id).Return(&models.Order{
		Base: models.Base{
			Id:          1,
			CreatedDate: time.Now(),
			UpdatedDate: time.Now(),
		},
		UserId:         1,
		SubTotalAmount: 125.25,
		TaxAmount:      25.30,
		ShippingAddress: models.Address{
			Street:  "123 foo street",
			City:    "Foo city",
			State:   "MD",
			Country: "USA",
		},
	}, nil)
	order, err := svc.GetById(id)
	assert.NoError(t, err)
	assert.NotNil(t, order)
}

func TestDelete(t *testing.T) {
	id := uint(1)
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().Delete(id, false).Return(nil)
	err := svc.Delete(id, false)
	assert.NoError(t, err)
}

func TestGetAllByUser(t *testing.T) {
	userId := uint(1)
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().GetAllByUserId(userId).Return([]*models.Order{
		{
			Base: models.Base{
				Id:          1,
				CreatedDate: time.Now(),
				UpdatedDate: time.Now(),
			},
			UserId:         1,
			SubTotalAmount: 123.55,
		},
		{
			Base: models.Base{
				Id:          2,
				CreatedDate: time.Now(),
				UpdatedDate: time.Now(),
			},
			UserId:         1,
			SubTotalAmount: 125.55},
	}, nil)
	orders, err := svc.GetByUserId(userId)
	assert.NoError(t, err)
	assert.NotNil(t, orders)
	assert.Equal(t, 2, len(orders), "order count must equal two (2)")
}

func TestSave(t *testing.T) {
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().Save(gomock.Any()).DoAndReturn(func(m *models.Order) error {
		assert.Equal(t, 40.00, m.SubTotalAmount, "sub total amount is not correct.")
		assert.InDelta(t, 2.40, m.TaxAmount, 0.001, "tax amount isn't correct.")
		assert.InDelta(t, 42.40, m.TotalAmount, 0.001, "total amount is not correct.")
		return nil
	})
	order := dto.Order{
		Id: 0,
		OrderItems: []orderitem.OrderItem{
			{
				Id:        0,
				ProductId: 1,
				Quantity:  2,
				UnitPrice: 5,
			},
			{
				Id:        0,
				ProductId: 2,
				Quantity:  3,
				UnitPrice: 10,
			},
		},
		Status:       "Pending",
		BillingState: "MD",
	}

	err := svc.Save(order)
	assert.NoError(t, err)
}

func TestSaveInvalidState(t *testing.T) {
	_, svc := setup(t)
	order := dto.Order{
		Id: 0,
		OrderItems: []orderitem.OrderItem{
			{
				Id:        0,
				ProductId: 1,
				Quantity:  2,
				UnitPrice: 5,
			},
			{
				Id:        0,
				ProductId: 2,
				Quantity:  3,
				UnitPrice: 10,
			},
		},
		Status:       "Pending",
		BillingState: "NOTFOUND",
	}

	err := svc.Save(order)
	assert.Error(t, err)
}

func TestUpdateStatus(t *testing.T) {
	id := uint(1)
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().UpdateStatus(id, string(models.OrderStatusShipped)).Return(nil)
	err := svc.UpdateStatus(id, string(models.OrderStatusShipped))
	assert.NoError(t, err)
}

func TestUpdateStatusInvalid(t *testing.T) {
	_, svc := setup(t)
	err := svc.UpdateStatus(uint(1), "INVALID")
	assert.Error(t, err)
}

func TestUpdateStatusRepoError(t *testing.T) {
	id := uint(1)
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().UpdateStatus(id, string(models.OrderStatusShipped)).Return(fmt.Errorf("db error"))
	err := svc.UpdateStatus(id, string(models.OrderStatusShipped))
	assert.Error(t, err)
}
