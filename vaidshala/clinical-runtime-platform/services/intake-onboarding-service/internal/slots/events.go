package slots

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SlotEvent represents a single immutable slot fill event.
type SlotEvent struct {
	ID             uuid.UUID       `json:"id"`
	PatientID      uuid.UUID       `json:"patient_id"`
	SlotName       string          `json:"slot_name"`
	Domain         string          `json:"domain"`
	Value          json.RawMessage `json:"value"`
	ExtractionMode string          `json:"extraction_mode"` // BUTTON, REGEX, NLU, DEVICE
	Confidence     float64         `json:"confidence"`
	SafetyResult   json.RawMessage `json:"safety_result,omitempty"`
	SourceChannel  string          `json:"source_channel"` // APP, WHATSAPP, ASHA
	FHIRResourceID string          `json:"fhir_resource_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// SlotValue represents the current value of a slot (derived from latest event).
type SlotValue struct {
	Value          json.RawMessage `json:"value"`
	ExtractionMode string          `json:"extraction_mode"`
	Confidence     float64         `json:"confidence"`
	FHIRResourceID string          `json:"fhir_resource_id"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// EventStore is the interface for slot event storage.
type EventStore interface {
	Append(ctx context.Context, event SlotEvent) error
	CurrentValues(ctx context.Context, patientID uuid.UUID) (map[string]SlotValue, error)
	SlotHistory(ctx context.Context, patientID uuid.UUID, slotName string) ([]SlotEvent, error)
}

// PgEventStore implements EventStore backed by PostgreSQL.
type PgEventStore struct {
	pool *pgxpool.Pool
}

// NewPgEventStore creates a new PostgreSQL-backed event store.
func NewPgEventStore(pool *pgxpool.Pool) *PgEventStore {
	return &PgEventStore{pool: pool}
}

// Append inserts a new slot event (append-only, never updates).
func (s *PgEventStore) Append(ctx context.Context, event SlotEvent) error {
	query := `
		INSERT INTO slot_events (patient_id, slot_name, domain, value, extraction_mode,
			confidence, safety_result, source_channel, fhir_resource_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	return s.pool.QueryRow(ctx, query,
		event.PatientID, event.SlotName, event.Domain, event.Value,
		event.ExtractionMode, event.Confidence, event.SafetyResult,
		event.SourceChannel, event.FHIRResourceID,
	).Scan(&event.ID, &event.CreatedAt)
}

// CurrentValues returns the latest value for each slot for a patient.
// Uses the current_slots view (DISTINCT ON patient_id, slot_name ORDER BY created_at DESC).
func (s *PgEventStore) CurrentValues(ctx context.Context, patientID uuid.UUID) (map[string]SlotValue, error) {
	query := `
		SELECT slot_name, value, extraction_mode, confidence, fhir_resource_id, created_at
		FROM current_slots
		WHERE patient_id = $1`

	rows, err := s.pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]SlotValue)
	for rows.Next() {
		var name string
		var sv SlotValue
		if err := rows.Scan(&name, &sv.Value, &sv.ExtractionMode, &sv.Confidence, &sv.FHIRResourceID, &sv.UpdatedAt); err != nil {
			return nil, err
		}
		result[name] = sv
	}
	return result, rows.Err()
}

// SlotHistory returns all events for a slot in chronological order.
func (s *PgEventStore) SlotHistory(ctx context.Context, patientID uuid.UUID, slotName string) ([]SlotEvent, error) {
	query := `
		SELECT id, patient_id, slot_name, domain, value, extraction_mode,
			confidence, safety_result, source_channel, fhir_resource_id, created_at
		FROM slot_events
		WHERE patient_id = $1 AND slot_name = $2
		ORDER BY created_at ASC`

	rows, err := s.pool.Query(ctx, query, patientID, slotName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SlotEvent
	for rows.Next() {
		var e SlotEvent
		if err := rows.Scan(&e.ID, &e.PatientID, &e.SlotName, &e.Domain, &e.Value,
			&e.ExtractionMode, &e.Confidence, &e.SafetyResult, &e.SourceChannel,
			&e.FHIRResourceID, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
