package outbox

import (
	"context"

	"commerce/internal/shared/events"

	"github.com/google/uuid"
)

// Publisher is the port ProcessBatch publishes through. Defined here, where
// it's consumed, rather than in internal/shared/aws or the publisher
// package, so this service can be tested with a fake instead of exercising
// real SQS.
type Publisher interface {
	Publish(ctx context.Context, envelopes []events.Envelope) (PublishResult, error)
}

// PublishResult is keyed by event id, not outbox row id - the publisher has
// no business knowing about database rows. ProcessBatch holds both (it
// claimed the rows) and does the id translation.
type PublishResult struct {
	PublishedIds []uuid.UUID
	FailedIds    []uuid.UUID
}
