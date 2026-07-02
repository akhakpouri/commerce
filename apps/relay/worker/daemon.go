package worker

import (
	manager "commerce/internal/shared/managers/transaction"
	outbox_repo "commerce/internal/shared/repositories/outbox"
	outbox_service "commerce/relay/internal/services/outbox"
	"context"
	"time"

	"gorm.io/gorm"
)

type Daemon struct {
	OutboxService outbox_service.OutboxServiceI
	interval      time.Duration
}

func NewDaemon(db *gorm.DB, interval time.Duration) *Daemon {
	outboxRepo := outbox_repo.NewOutboxRepository(db)
	manager := manager.NewManager(db)
	outboxService := outbox_service.NewOutboxService(outboxRepo, manager)

	return &Daemon{
		OutboxService: outboxService,
		interval:      interval,
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			d.OutboxService.ProcessBatch(10)
		}
	}
}
