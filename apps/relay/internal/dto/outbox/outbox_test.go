package outbox

import (
	"encoding/json"
	"testing"
	"time"

	"commerce/internal/shared/events"
	"commerce/internal/shared/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestToEnvelope(t *testing.T) {
	eventId := uuid.New()
	createdAt := time.Date(2026, 8, 1, 9, 30, 0, 0, time.UTC)

	dto := FromModel(&models.Outbox{
		Base: models.Base{
			Id:          7,
			CreatedDate: createdAt,
		},
		EventId:       eventId,
		EventType:     "order.placed",
		AggregateId:   42,
		AggregateType: "Order",
		Payload:       datatypes.JSON(`{"total_amount":125.5}`),
		Version:       1,
	})

	envelope := ToEnvelope(dto)

	// OccurredAt must equal the row's CreatedDate (when the domain
	// transaction committed), not time.Now() - the regression guard for the
	// timestamp bug this refactor fixes.
	assert.Equal(t, events.Envelope{
		EventId:    eventId,
		EventType:  "order.placed",
		OccurredAt: createdAt,
		Aggregate:  events.Aggregate{Type: "Order", Id: 42},
		Version:    1,
		Payload:    json.RawMessage(`{"total_amount":125.5}`),
	}, envelope)
}
