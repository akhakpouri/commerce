package worker

import (
	queue_manager "commerce/internal/shared/managers/aws"
	manager "commerce/internal/shared/managers/transaction"
	outbox_repo "commerce/internal/shared/repositories/outbox"
	relay_manager "commerce/relay/internal/managers/relay"
	outbox_service "commerce/relay/internal/services/outbox"
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"gorm.io/gorm"
)

type Daemon struct {
	OutboxService outbox_service.OutboxServiceI
	interval      time.Duration
}

func NewDaemon(db *gorm.DB, client *sqs.Client, interval time.Duration) *Daemon {
	outboxRepo := outbox_repo.NewOutboxRepository(db)
	manager := manager.NewManager(db)
	queueManager := queue_manager.NewQueueManager(client)

	relayManager := relay_manager.NewRelayManager("commerce-queue", queueManager)
	outboxService := outbox_service.NewOutboxService(outboxRepo, manager, relayManager)

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
			if err := d.OutboxService.ProcessBatch(10); err != nil {
				slog.Error("Exception occured when processing batch.", "error", err)
			}
		}
	}
}
