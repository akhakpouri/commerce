package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Outbox struct {
	Base
	EventId       uuid.UUID      `gorm:"type:uuid; not null"`
	EventType     string         `gorm:"type:varchar(100); not null"`
	AggregateId   uint           `gorm:"not null"`
	AggregateType string         `gorm:"type:varchar(100); not null"`
	Payload       datatypes.JSON `gorm:"type:jsonb"`
	PublishedAt   time.Time
	Attempts      int `gorm:"type:integer;defualt:0"`
}
