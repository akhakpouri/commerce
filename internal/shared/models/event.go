package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Event struct {
	StreamId      uuid.UUID      `gorm:"type:uuid; uniqueIndex:idx_stream_version; not null"`
	Version       int            `gorm:"uniqueIndex:idx_stream_version; not null"`
	EventType     int            `gorm:"not null"`
	SchemaVersion int            `gorm:"not null"`
	Payload       datatypes.JSON `gorm:"type:jsonb; not null"`
	OccuredAt     time.Time      `gorm:"not null"`
	GlobalSequnce int64          `gorm:"type:bigint; column:global_seq; primaryKey; autoIncrement; not null"`
}
