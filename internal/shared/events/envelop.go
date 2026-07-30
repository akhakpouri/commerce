package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Envelop struct {
	EventId   uuid.UUID       `json:"event_id"`
	EventType string          `json:"event_type"`
	OccuredAt time.Time       `json:"occured_at"`
	Aggregate Aggregate       `json:"aggregate"`
	Version   int             `json:"version"`
	Payload   json.RawMessage `json:"payload"`
}
