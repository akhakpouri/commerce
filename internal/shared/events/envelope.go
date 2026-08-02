package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Envelope struct {
	EventId    uuid.UUID       `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Aggregate  Aggregate       `json:"aggregate"`
	Version    int             `json:"version"`
	Payload    json.RawMessage `json:"payload"`
}
