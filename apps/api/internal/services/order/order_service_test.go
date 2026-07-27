package order

import (
	"fmt"
	"testing"
	"time"

	tax_service "commerce/api/internal/services/tax"
	"commerce/internal/shared/models"

	dto "commerce/api/internal/dto/order"
	orderitem "commerce/api/internal/dto/order-item"
	manager "commerce/internal/shared/managers/transaction"
	orderrepo "commerce/internal/shared/repositories/order"
	outboxrepo "commerce/internal/shared/repositories/outbox"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// fakeManager is a test double for manager.ManagerI: instead of opening a real
// transaction, its Execute simply runs the callback with fake repositories, so
// the service's transactional writes land on the mocks.
type fakeManager struct {
	repos manager.RepositoriesI
}

func (f *fakeManager) Execute(fn func(r manager.RepositoriesI) error) error {
	return fn(f.repos)
}

type fakeRepositories struct {
	order  orderrepo.OrderRepositoryI
	outbox outboxrepo.OutboxRepositoryI
}

func (f *fakeRepositories) Order() orderrepo.OrderRepositoryI   { return f.order }
func (f *fakeRepositories) Outbox() outboxrepo.OutboxRepositoryI { return f.outbox }

// fakeOutboxRepository satisfies OutboxRepositoryI; Save succeeds and the rest
// are unused stubs (the service Save path only touches Save).
type fakeOutboxRepository struct{}

func (fakeOutboxRepository) Save(*models.Outbox) error                  { return nil }
func (fakeOutboxRepository) Get(uint) (*models.Outbox, error)           { return nil, nil }
func (fakeOutboxRepository) GetAll() ([]*models.Outbox, error)          { return nil, nil }
func (fakeOutboxRepository) GetNextBatch(int) ([]*models.Outbox, error) { return nil, nil }
func (fakeOutboxRepository) MarkPublished([]uint) error                 { return nil }
func (fakeOutboxRepository) Delete(uint) error                          { return nil }

func setup(t *testing.T) (*MockOrderRepositoryI, OrderServiceI) {
	t.Helper()
	ctl := gomock.NewController(t)
	t.Cleanup(ctl.Finish)
	mockRepo := NewMockOrderRepositoryI(ctl)
	taxService := tax_service.NewTaxService()
	txm := &fakeManager{repos: &fakeRepositories{order: mockRepo, outbox: fakeOutboxRepository{}}}
	return mockRepo, NewOrderService(mockRepo, taxService, txm)
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

func TestGetStatuses(t *testing.T) {
	_, svc := setup(t)
	statuses := svc.GetStatuses()
	assert.NotEmpty(t, statuses)
	for i, status := range statuses {
		t.Logf("status %d %s", i, status.Status)
	}
}
