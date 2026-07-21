package publisher

import (
	"context"
	"fmt"

	aws "commerce/internal/shared/aws"
	dto "commerce/relay/internal/dto/outbox"
)

const maxSendBatchSize = 10

// SqsPublisher adapts an SQS producer to the Publisher port.
type SqsPublisher struct {
	producer *aws.Producer
}

func NewSqsPublisher(producer *aws.Producer) *SqsPublisher {
	return &SqsPublisher{producer: producer}
}

// Publish sends events to SQS, chunking into batches of 10 (SendMessageBatch's
// own limit). Any failed entry - including a partial batch failure - fails the
// whole call, so the caller's transaction rolls back and every event in this
// batch is retried on the next poll; downstream consumers must be idempotent
// regardless, so re-sending an already-delivered event is safe.
func (p *SqsPublisher) Publish(ctx context.Context, events []*dto.Outbox) error {
	for start := 0; start < len(events); start += maxSendBatchSize {
		end := min(start+maxSendBatchSize, len(events))

		messages := make([]*aws.Message, end-start)
		for i, event := range events[start:end] {
			messages[i] = dto.ToMessage(event)
		}

		result, err := p.producer.SendBatch(ctx, messages)
		if err != nil {
			return fmt.Errorf("publishing outbox batch: %w", err)
		}
		if len(result.Failed) > 0 {
			return fmt.Errorf("publishing outbox batch: %d message(s) failed: %+v", len(result.Failed), result.Failed)
		}
	}
	return nil
}
