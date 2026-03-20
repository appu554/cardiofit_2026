package signals

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ClinicalSignalEnvelope is the canonical event wrapper for all clinical signals
// published to Kafka. Consumers deserialize this to route by SignalType.
type ClinicalSignalEnvelope struct {
	EventID    uuid.UUID       `json:"event_id"`
	PatientID  string          `json:"patient_id"`
	SignalType SignalType      `json:"signal_type"`
	Priority   bool            `json:"priority"`
	MeasuredAt time.Time       `json:"measured_at"`
	Source     SignalSource    `json:"source"`
	Confidence float64         `json:"confidence"`
	LOINCCode  string          `json:"loinc_code,omitempty"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
}

// KafkaTopic returns the target Kafka topic for this envelope.
func (e *ClinicalSignalEnvelope) KafkaTopic() string {
	if e.Priority {
		return TopicPriorityEvents
	}
	return TopicObservations
}

// ClinicalStateChangeEnvelope wraps medication, stratum, and protocol lifecycle
// events published to the clinical.state-changes.v1 topic.
type ClinicalStateChangeEnvelope struct {
	EventID    uuid.UUID       `json:"event_id"`
	PatientID  string          `json:"patient_id"`
	ChangeType string          `json:"change_type"`
	Timestamp  time.Time       `json:"timestamp"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
}

// KafkaTopic returns the target Kafka topic for state-change events.
func (e *ClinicalStateChangeEnvelope) KafkaTopic() string {
	return TopicStateChanges
}

// Kafka topic constants.
const (
	TopicObservations   = "clinical.observations.v1"
	TopicPriorityEvents = "clinical.priority-events.v1"
	TopicStateChanges   = "clinical.state-changes.v1"
	TopicSignalDLQ      = "clinical.signal-dlq.v1"
)
