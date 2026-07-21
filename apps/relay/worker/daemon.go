package worker

import (
	"context"
	"fmt"

	"commerce/internal/shared/aws"
	"commerce/internal/shared/configs"
	queue_manager "commerce/internal/shared/managers/aws"
	manager "commerce/internal/shared/managers/transaction"
	outbox_repo "commerce/internal/shared/repositories/outbox"
	relay_manager "commerce/relay/internal/managers"
	"commerce/relay/internal/publisher"
	outbox_service "commerce/relay/internal/services/outbox"

	"gorm.io/gorm"
)

const QueueName = "commerce-queue"

// NewDaemon wires the outbox-to-SQS relay: resolves the queue URL - the queue
// itself is provisioned by iac-matrix, never created here - then assembles
// the publisher, outbox service, and poll loop behind RelayManagerI.
func NewDaemon(ctx context.Context, db *gorm.DB, cfg *configs.AWSConfig) (relay_manager.RelayManagerI, error) {
	sqsClient, err := aws.NewSqsClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating SQS client: %w", err)
	}

	queueManager := queue_manager.NewQueueManager(sqsClient)
	queueUrl, err := queueManager.GetUrl(ctx, QueueName)
	if err != nil {
		return nil, fmt.Errorf("resolving queue %q (has it been provisioned in iac-matrix?): %w", QueueName, err)
	}

	producer := aws.NewProducer(sqsClient, queueUrl)
	sqsPublisher := publisher.NewSqsPublisher(producer)

	outboxRepo := outbox_repo.NewOutboxRepository(db)
	txManager := manager.NewManager(db)
	outboxService := outbox_service.NewOutboxService(outboxRepo, txManager, sqsPublisher)

	return relay_manager.NewRelayManager(outboxService, relay_manager.DefaultInterval, relay_manager.DefaultBatchSize), nil
}
