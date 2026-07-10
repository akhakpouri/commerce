package outbox

import (
	"time"

	"commerce/internal/shared/aws"
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
	}
}

func FromAllModels(models []*models.Outbox) []*Outbox {
	var dtos []*Outbox
	for _, m := range models {
		dtos = append(dtos, FromModel(m))
	}
	return dtos
}

func ToMessage(dto *Outbox) *aws.Message {
	return &aws.Message{
		Id:        dto.EventId.String(),
		Type:      dto.EventType,
		Timestamp: time.Now().UTC(),
		Payload:   dto.Payload,
	}
}
