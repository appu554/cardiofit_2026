package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// EventStatus represents the status of an outbox event
type EventStatus string

const (
	EventStatusPending     EventStatus = "pending"
	EventStatusPublished   EventStatus = "published"
	EventStatusFailed      EventStatus = "failed"
	EventStatusDeadLetter  EventStatus = "dead_letter"
)

// MedicalContext represents medical priority context for circuit breaker
type MedicalContext string

const (
	MedicalContextCritical    MedicalContext = "critical"
	MedicalContextUrgent      MedicalContext = "urgent"
	MedicalContextRoutine     MedicalContext = "routine"
	MedicalContextBackground  MedicalContext = "background"
)

// Metadata represents JSON metadata for events
type Metadata map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for database retrieval
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, m)
}

// OutboxEvent represents an event in the outbox table
type OutboxEvent struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	ServiceName     string         `json:"service_name" db:"service_name"`
	EventType       string         `json:"event_type" db:"event_type"`
	EventData       string         `json:"event_data" db:"event_data"`
	Topic           string         `json:"topic" db:"topic"`
	CorrelationID   *string        `json:"correlation_id,omitempty" db:"correlation_id"`
	Priority        int32          `json:"priority" db:"priority"`
	Metadata        Metadata       `json:"metadata,omitempty" db:"metadata"`
	MedicalContext  MedicalContext `json:"medical_context" db:"medical_context"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	PublishedAt     *time.Time     `json:"published_at,omitempty" db:"published_at"`
	RetryCount      int32          `json:"retry_count" db:"retry_count"`
	Status          EventStatus    `json:"status" db:"status"`
	ErrorMessage    *string        `json:"error_message,omitempty" db:"error_message"`
	NextRetryAt     *time.Time     `json:"next_retry_at,omitempty" db:"next_retry_at"`
}

// NewOutboxEvent creates a new outbox event
func NewOutboxEvent(serviceName, eventType, eventData, topic string, priority int32, medicalContext MedicalContext) *OutboxEvent {
	return &OutboxEvent{
		ID:             uuid.New(),
		ServiceName:    serviceName,
		EventType:      eventType,
		EventData:      eventData,
		Topic:          topic,
		Priority:       priority,
		MedicalContext: medicalContext,
		CreatedAt:      time.Now().UTC(),
		RetryCount:     0,
		Status:         EventStatusPending,
	}
}

// IsCritical returns true if the event has critical medical context
func (e *OutboxEvent) IsCritical() bool {
	return e.MedicalContext == MedicalContextCritical
}

// IsUrgent returns true if the event has urgent medical context
func (e *OutboxEvent) IsUrgent() bool {
	return e.MedicalContext == MedicalContextUrgent
}

// CanRetry returns true if the event can be retried based on retry count and status
func (e *OutboxEvent) CanRetry(maxRetries int32) bool {
	return e.Status == EventStatusFailed && e.RetryCount < maxRetries
}

// IncrementRetryCount increments the retry count and updates next retry time
func (e *OutboxEvent) IncrementRetryCount(nextRetryAt time.Time) {
	e.RetryCount++
	e.NextRetryAt = &nextRetryAt
}

// MarkPublished marks the event as successfully published
func (e *OutboxEvent) MarkPublished() {
	now := time.Now().UTC()
	e.Status = EventStatusPublished
	e.PublishedAt = &now
	e.ErrorMessage = nil
}

// MarkFailed marks the event as failed with an error message
func (e *OutboxEvent) MarkFailed(errorMessage string) {
	e.Status = EventStatusFailed
	e.ErrorMessage = &errorMessage
}

// MarkDeadLetter marks the event as dead letter (max retries exceeded)
func (e *OutboxEvent) MarkDeadLetter(errorMessage string) {
	e.Status = EventStatusDeadLetter
	e.ErrorMessage = &errorMessage
}

// TableName returns the table name for the service
func (e *OutboxEvent) TableName() string {
	return "outbox_events_" + e.ServiceName
}

// OutboxStats represents statistics for outbox queues
type OutboxStats struct {
	ServiceName              string            `json:"service_name"`
	QueueDepths              map[string]int64  `json:"queue_depths"`
	TotalProcessed24h        int64             `json:"total_processed_24h"`
	DeadLetterCount          int64             `json:"dead_letter_count"`
	SuccessRates             map[string]float64 `json:"success_rates"`
	CriticalEventsProcessed  int64             `json:"critical_events_processed"`
	NonCriticalEventsDropped int64             `json:"non_critical_events_dropped"`
}

// CircuitBreakerState represents the state of the medical circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "CLOSED"
	CircuitBreakerOpen     CircuitBreakerState = "OPEN"
	CircuitBreakerHalfOpen CircuitBreakerState = "HALF_OPEN"
)

// CircuitBreakerStatus represents the status of the medical circuit breaker
type CircuitBreakerStatus struct {
	Enabled                  bool                `json:"enabled"`
	State                    CircuitBreakerState `json:"state"`
	CurrentLoad              float64             `json:"current_load"`
	TotalRequests            int64               `json:"total_requests"`
	FailedRequests           int64               `json:"failed_requests"`
	CriticalEventsProcessed  int64               `json:"critical_events_processed"`
	NonCriticalEventsDropped int64               `json:"non_critical_events_dropped"`
	NextRetryAt              *time.Time          `json:"next_retry_at,omitempty"`
}