package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	aws "commerce/internal/shared/aws"
	"commerce/internal/shared/events"
	outbox_service "commerce/relay/internal/services/outbox"

	"github.com/google/uuid"
)

const maxSendBatchSize = 10

// SqsPublisher adapts an SQS producer to the outbox service's Publisher port.
type SqsPublisher struct {
	producer *aws.Producer
}

func NewSqsPublisher(producer *aws.Producer) *SqsPublisher {
	return &SqsPublisher{producer: producer}
}

// Publish marshals each envelope as the SQS message body and sends in
// batches of 10 (SendMessageBatch's own limit). error is reserved for
// whole-call failures (network, auth, marshalling); a broker-side rejection
// of individual messages comes back as a non-empty PublishResult.FailedIds,
// not an error - ProcessBatch tells those two cases apart.
func (p *SqsPublisher) Publish(ctx context.Context, envelopes []events.Envelope) (outbox_service.PublishResult, error) {
	result := outbox_service.PublishResult{}

	for start := 0; start < len(envelopes); start += maxSendBatchSize {
		end := min(start+maxSendBatchSize, len(envelopes))

		messages := make([]*aws.RawMessage, end-start)
		for i, envelope := range envelopes[start:end] {
			body, err := json.Marshal(envelope)
			if err != nil {
				return outbox_service.PublishResult{}, fmt.Errorf("marshalling envelope %s: %w", envelope.EventId, err)
			}
			messages[i] = &aws.RawMessage{
				Id:   envelope.EventId.String(),
				Body: string(body),
				Attributes: map[string]string{
					"event_type": envelope.EventType,
				},
			}
		}

		batchResult, err := p.producer.SendBatch(ctx, messages)
		if err != nil {
			return outbox_service.PublishResult{}, fmt.Errorf("publishing outbox batch: %w", err)
		}

		for _, idStr := range batchResult.SuccessfulIds {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return outbox_service.PublishResult{}, fmt.Errorf("parsing published event id %q: %w", idStr, err)
			}
			result.PublishedIds = append(result.PublishedIds, id)
		}

		for _, failedEntry := range batchResult.Failed {
			id, err := uuid.Parse(failedEntry.Id)
			if err != nil {
				return outbox_service.PublishResult{}, fmt.Errorf("parsing failed event id %q: %w", failedEntry.Id, err)
			}
			result.FailedIds = append(result.FailedIds, id)
		}
	}

	return result, nil
}
