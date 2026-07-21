package eventsourcing

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	SreamId   uuid.UUID
	EventType string
	Payload   string
	OccuredAt time.Time
}
