package payment

import (
	"commerce/internal/shared/models"
	"testing"
	"time"

	dto "commerce/api/internal/dto/payment"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setup(t *testing.T) (*MockPaymentRepositoryI, PaymentServiceI) {
	t.Helper()
	ctl := gomock.NewController(t)
	//the cleanup method replaces defer ctl.Finish(). It runs at the end of the test.
	t.Cleanup(ctl.Finish)
	mockRepo := NewMockPaymentRepositoryI(ctl)
	return mockRepo, NewPaymentService(mockRepo)
}

func TestGetById(t *testing.T) {
	id, empty := uint(1), uuid.New()
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().GetById(id).Return(&models.Payment{
		Base: models.Base{
			Id:          id,
			CreatedDate: time.Now(),
			UpdatedDate: time.Now(),
		},
		OrderId:              uint(123),
		Amount:               2356.25,
		Status:               models.PaymentStatusCompleted,
		GatewayTransactionId: empty.String(),
		GatewayResponse:      "correct",
	}, nil)

	payment, err := svc.GetById(id)
	assert.NoError(t, err)
	assert.NotNil(t, payment)
}

func TestGetByOrder(t *testing.T) {
	orderId := uint(1)
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().GetByOrder(orderId).Return([]*models.Payment{
		{
			Base: models.Base{
				Id:          uint(1),
				CreatedDate: time.Now(),
				UpdatedDate: time.Now(),
			},
			OrderId: orderId,
			Amount:  235.05,
		},
		{
			Base: models.Base{
				Id:          uint(2),
				CreatedDate: time.Now(),
				UpdatedDate: time.Now(),
			},
			OrderId: orderId,
			Amount:  1274.05,
		},
	}, nil)

	payments, err := svc.GetByOrder(orderId)
	assert.NoError(t, err)
	assert.NotEmpty(t, payments)
	assert.Equalf(t, 2, len(payments), "Payments count should be two (2)")
}

func TestDelete(t *testing.T) {
	id := uint(1)
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().Delete(id, false).Return(nil)
	err := svc.Delete(id, false)
	assert.NoError(t, err)
}

func TestSave(t *testing.T) {
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().Save(gomock.Any()).Return(nil)
	err := svc.Save(&dto.Payment{
		Id:      0,
		OrderId: 1,
		Amount:  125.250,
		Status:  "completed",
	})
	assert.NoError(t, err)
}

func TestUpdateStatus(t *testing.T) {
	mockRepo, svc := setup(t)
	mockRepo.EXPECT().UpdateStatus(uint(1), "completed").Return(nil)
	err := svc.UpdateStatus(uint(1), "completed")
	assert.NoError(t, err)

}
