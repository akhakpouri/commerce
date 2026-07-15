package eventsourcing

import (
	"context"

	"github.com/google/uuid"
)

type EventStore interface {
	Append(ctx context.Context, streamId uuid.UUID, version int, events []Event) error
	Load(ctx context.Context, stremId uuid.UUID) ([]Event, error)
	LoadSince(ctx context.Context, globalSeq int64, limit int) ([]Event, error)
}
