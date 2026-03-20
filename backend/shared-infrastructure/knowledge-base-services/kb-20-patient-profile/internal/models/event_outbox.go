package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventOutboxEntry is the GORM model for the event_outbox table.
// Events are written atomically in the same DB transaction as the data change,
// then a background poller delivers them to subscribers and marks published_at.
type EventOutboxEntry struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventType   string          `gorm:"type:text;not null"`
	PatientID   string          `gorm:"type:text;not null"`
	Payload     json.RawMessage `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time       `gorm:"not null;default:now()"`
	PublishedAt       *time.Time
	KafkaPublishedAt  *time.Time `gorm:"column:kafka_published_at"`
}

// TableName returns the database table name for GORM.
func (EventOutboxEntry) TableName() string {
	return "event_outbox"
}
