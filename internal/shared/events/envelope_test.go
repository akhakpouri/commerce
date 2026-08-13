package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEnvelope_JSONRoundTrip(t *testing.T) {
	eventId := uuid.New()
	occurredAt := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	original := Envelope{
		EventId:    eventId,
		EventType:  "order.placed",
		OccurredAt: occurredAt,
		Aggregate:  Aggregate{Type: "Order", Id: 42},
		Version:    1,
		Payload:    json.RawMessage(`{"total_amount":125.5}`),
	}

	body, err := json.Marshal(original)
	assert.NoError(t, err)

	expected := `{
		"event_id":"` + eventId.String() + `",
		"event_type":"order.placed",
		"occurred_at":"2026-08-01T12:00:00Z",
		"aggregate":{"type":"Order","id":42},
		"version":1,
		"payload":{"total_amount":125.5}
	}`
	assert.JSONEq(t, expected, string(body))

	var decoded Envelope
	assert.NoError(t, json.Unmarshal(body, &decoded))
	assert.Equal(t, original, decoded)
}
