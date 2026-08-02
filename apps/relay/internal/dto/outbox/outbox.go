package outbox

import (
	"encoding/json"
	"time"

	"commerce/internal/shared/events"
	"commerce/internal/shared/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Outbox struct {
	Id            uint       `json:"id"`
	EventId       uuid.UUID  `json:"event_id"`
	EventType     string     `json:"event_type"`
	AggregateId   uint       `json:"aggregate_id"`
	AggregateType string     `json:"aggregate_type"`
	Payload       string     `json:"payload"`
	PublishedAt   *time.Time `json:"published_at"`
	Attempts      int        `json:"attempts"`
	Version       int        `json:"version"`
	CreatedAt     time.Time  `json:"created_at"`
}

func FromModel(model *models.Outbox) *Outbox {
	return &Outbox{
		Id:            model.Id,
		EventId:       model.EventId,
		EventType:     model.EventType,
		AggregateId:   model.AggregateId,
		AggregateType: model.AggregateType,
		Payload:       string(model.Payload),
		PublishedAt:   model.PublishedAt,
		Attempts:      model.Attempts,
		Version:       model.Version,
		CreatedAt:     model.CreatedDate,
	}
}

func ToModel(dto *Outbox) *models.Outbox {
	return &models.Outbox{
		Base: models.Base{
			Id: dto.Id,
		},
		EventId:       dto.EventId,
		EventType:     dto.EventType,
		AggregateId:   dto.AggregateId,
		AggregateType: dto.AggregateType,
		Payload:       datatypes.JSON(dto.Payload),
		PublishedAt:   dto.PublishedAt,
		Attempts:      dto.Attempts,
		Version:       dto.Version,
	}
}

func FromAllModels(models []*models.Outbox) []*Outbox {
	var dtos []*Outbox
	for _, m := range models {
		dtos = append(dtos, FromModel(m))
	}
	return dtos
}

// ToEnvelope builds the wire-format event envelope for an outbox row.
// OccurredAt is the row's CreatedAt (the moment the domain transaction
// committed), not time.Now() - a backed-up or retrying relay must still
// report when the event actually happened, not when it happened to be sent.
func ToEnvelope(dto *Outbox) events.Envelope {
	return events.Envelope{
		EventId:   dto.EventId,
		EventType: dto.EventType,
		OccurredAt: dto.CreatedAt,
		Aggregate: events.Aggregate{
			Type: dto.AggregateType,
			Id:   dto.AggregateId,
		},
		Version: dto.Version,
		Payload: json.RawMessage(dto.Payload),
	}
}
