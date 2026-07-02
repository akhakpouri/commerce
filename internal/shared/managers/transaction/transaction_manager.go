package transaction

import (
	orderrepo "commerce/internal/shared/repositories/order"
	outboxrepo "commerce/internal/shared/repositories/outbox"

	"gorm.io/gorm"
)

type RepositoriesI interface {
	Order() orderrepo.OrderRepositoryI
	Outbox() outboxrepo.OutboxRepositoryI
}

type ManagerI interface {
	Execute(fn func(r RepositoriesI) error) error
}

type repositories struct {
	tx *gorm.DB
}

type Manager struct {
	db *gorm.DB
}

func NewManager(db *gorm.DB) ManagerI {
	return &Manager{db: db}
}

func (m *Manager) Execute(fn func(r RepositoriesI) error) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		return fn(&repositories{tx: tx})
	})
}

func (r *repositories) Order() orderrepo.OrderRepositoryI {
	return orderrepo.NewOrderRepository(r.tx)
}

func (r *repositories) Outbox() outboxrepo.OutboxRepositoryI {
	return outboxrepo.NewOutboxRepository(r.tx)
}
