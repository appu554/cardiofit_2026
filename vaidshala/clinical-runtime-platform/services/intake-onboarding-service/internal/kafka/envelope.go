package kafka

import (
	"time"

	"github.com/google/uuid"
)

// Envelope is the standard Kafka message wrapper for intake events.
type Envelope struct {
	EventID    uuid.UUID              `json:"event_id"`
	EventType  string                 `json:"event_type"`
	SourceType string                 `json:"source_type"`
	PatientID  uuid.UUID              `json:"patient_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Payload    map[string]interface{} `json:"payload"`
}
