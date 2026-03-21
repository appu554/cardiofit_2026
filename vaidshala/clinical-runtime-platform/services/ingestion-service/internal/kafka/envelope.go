package kafka

import (
	"time"

	"github.com/google/uuid"
)

// Envelope is the standard Kafka message wrapper used by the ingestion
// service. Every message produced to a Kafka topic is wrapped in this
// envelope so that consumers can route, trace, and filter without
// deserialising the full payload.
type Envelope struct {
	EventID          uuid.UUID              `json:"eventId"`
	EventType        string                 `json:"eventType"`
	SourceType       string                 `json:"sourceType"`
	PatientID        uuid.UUID              `json:"patientId"`
	TenantID         uuid.UUID              `json:"tenantId"`
	Timestamp        time.Time              `json:"timestamp"`
	FHIRResourceType string                 `json:"fhirResourceType"`
	FHIRResourceID   string                 `json:"fhirResourceId"`
	Payload          map[string]interface{} `json:"payload"`
	QualityScore     float64                `json:"qualityScore,omitempty"`
	Flags            []string               `json:"flags,omitempty"`
	TraceID          string                 `json:"traceId,omitempty"`
}
