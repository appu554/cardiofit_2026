// Package models contains domain models for KB-17 Population Registry
package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of clinical event
type EventType string

// Inbound event types (from upstream services via Kafka)
const (
	EventTypeDiagnosisCreated  EventType = "diagnosis.created"
	EventTypeDiagnosisUpdated  EventType = "diagnosis.updated"
	EventTypeLabResultCreated  EventType = "lab.result.created"
	EventTypeMedicationStarted EventType = "medication.started"
	EventTypeMedicationStopped EventType = "medication.stopped"
	EventTypeProblemAdded      EventType = "problem.added"
	EventTypeProblemResolved   EventType = "problem.resolved"
	EventTypeVitalRecorded     EventType = "vital.recorded"
	EventTypeEncounterClosed   EventType = "encounter.closed"
)

// Outbound event types (produced by KB-17)
const (
	EventTypeRegistryEnrolled       EventType = "registry.enrolled"
	EventTypeRegistryDisenrolled    EventType = "registry.disenrolled"
	EventTypeRegistryRiskChanged    EventType = "registry.risk_changed"
	EventTypeRegistryCareGapUpdated EventType = "registry.care_gap_updated"
	EventTypeRegistryMetricUpdated  EventType = "registry.metric_updated"
)

// IsInbound returns true if this is an inbound event type
func (e EventType) IsInbound() bool {
	switch e {
	case EventTypeDiagnosisCreated, EventTypeDiagnosisUpdated,
		EventTypeLabResultCreated, EventTypeMedicationStarted,
		EventTypeMedicationStopped, EventTypeProblemAdded,
		EventTypeProblemResolved, EventTypeVitalRecorded,
		EventTypeEncounterClosed:
		return true
	}
	return false
}

// IsOutbound returns true if this is an outbound event type
func (e EventType) IsOutbound() bool {
	switch e {
	case EventTypeRegistryEnrolled, EventTypeRegistryDisenrolled,
		EventTypeRegistryRiskChanged, EventTypeRegistryCareGapUpdated,
		EventTypeRegistryMetricUpdated:
		return true
	}
	return false
}

// ClinicalEvent represents an inbound clinical event from Kafka
type ClinicalEvent struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Source      string                 `json:"source"`       // e.g., "ehr", "lab-system"
	PatientID   string                 `json:"patient_id"`
	EncounterID string                 `json:"encounter_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
}

// GetDiagnosis extracts diagnosis data from the event
func (e *ClinicalEvent) GetDiagnosis() *Diagnosis {
	if e.Type != EventTypeDiagnosisCreated && e.Type != EventTypeDiagnosisUpdated {
		return nil
	}

	diagnosis := &Diagnosis{
		RecordedAt: e.Timestamp,
	}

	if code, ok := e.Data["code"].(string); ok {
		diagnosis.Code = code
	}
	if system, ok := e.Data["code_system"].(string); ok {
		diagnosis.CodeSystem = CodeSystem(system)
	}
	if display, ok := e.Data["display"].(string); ok {
		diagnosis.Display = display
	}
	if status, ok := e.Data["status"].(string); ok {
		diagnosis.Status = status
	}

	return diagnosis
}

// GetLabResult extracts lab result data from the event
func (e *ClinicalEvent) GetLabResult() *LabResult {
	if e.Type != EventTypeLabResultCreated {
		return nil
	}

	lab := &LabResult{
		EffectiveAt: e.Timestamp,
	}

	if code, ok := e.Data["code"].(string); ok {
		lab.Code = code
	}
	if system, ok := e.Data["code_system"].(string); ok {
		lab.CodeSystem = CodeSystem(system)
	}
	if display, ok := e.Data["display"].(string); ok {
		lab.Display = display
	}
	if value, ok := e.Data["value"]; ok {
		lab.Value = value
	}
	if unit, ok := e.Data["unit"].(string); ok {
		lab.Unit = unit
	}

	return lab
}

// GetMedication extracts medication data from the event
func (e *ClinicalEvent) GetMedication() *Medication {
	if e.Type != EventTypeMedicationStarted && e.Type != EventTypeMedicationStopped {
		return nil
	}

	med := &Medication{}

	if code, ok := e.Data["code"].(string); ok {
		med.Code = code
	}
	if system, ok := e.Data["code_system"].(string); ok {
		med.CodeSystem = CodeSystem(system)
	}
	if display, ok := e.Data["display"].(string); ok {
		med.Display = display
	}
	if status, ok := e.Data["status"].(string); ok {
		med.Status = status
	}

	return med
}

// RegistryEvent represents an outbound event produced by KB-17
type RegistryEvent struct {
	ID           string                 `json:"id"`
	Type         EventType              `json:"type"`
	RegistryCode RegistryCode           `json:"registry_code"`
	PatientID    string                 `json:"patient_id"`
	EnrollmentID uuid.UUID              `json:"enrollment_id,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Data         map[string]interface{} `json:"data"`
}

// NewEnrollmentEvent creates an event for a new enrollment
func NewEnrollmentEvent(enrollment *RegistryPatient) *RegistryEvent {
	return &RegistryEvent{
		ID:           uuid.New().String(),
		Type:         EventTypeRegistryEnrolled,
		RegistryCode: enrollment.RegistryCode,
		PatientID:    enrollment.PatientID,
		EnrollmentID: enrollment.ID,
		Timestamp:    time.Now().UTC(),
		Data: map[string]interface{}{
			"status":            enrollment.Status,
			"risk_tier":         enrollment.RiskTier,
			"enrollment_source": enrollment.EnrollmentSource,
			"enrolled_at":       enrollment.EnrolledAt,
		},
	}
}

// NewDisenrollmentEvent creates an event for disenrollment
func NewDisenrollmentEvent(enrollment *RegistryPatient, reason string) *RegistryEvent {
	return &RegistryEvent{
		ID:           uuid.New().String(),
		Type:         EventTypeRegistryDisenrolled,
		RegistryCode: enrollment.RegistryCode,
		PatientID:    enrollment.PatientID,
		EnrollmentID: enrollment.ID,
		Timestamp:    time.Now().UTC(),
		Data: map[string]interface{}{
			"reason":         reason,
			"disenrolled_at": time.Now().UTC(),
		},
	}
}

// NewRiskChangedEvent creates an event for risk tier change
func NewRiskChangedEvent(enrollment *RegistryPatient, oldTier, newTier RiskTier) *RegistryEvent {
	return &RegistryEvent{
		ID:           uuid.New().String(),
		Type:         EventTypeRegistryRiskChanged,
		RegistryCode: enrollment.RegistryCode,
		PatientID:    enrollment.PatientID,
		EnrollmentID: enrollment.ID,
		Timestamp:    time.Now().UTC(),
		Data: map[string]interface{}{
			"old_risk_tier": oldTier,
			"new_risk_tier": newTier,
		},
	}
}

// NewCareGapEvent creates an event for care gap updates
func NewCareGapEvent(enrollment *RegistryPatient, action string, gapID string) *RegistryEvent {
	return &RegistryEvent{
		ID:           uuid.New().String(),
		Type:         EventTypeRegistryCareGapUpdated,
		RegistryCode: enrollment.RegistryCode,
		PatientID:    enrollment.PatientID,
		EnrollmentID: enrollment.ID,
		Timestamp:    time.Now().UTC(),
		Data: map[string]interface{}{
			"action":   action, // "added" or "closed"
			"gap_id":   gapID,
			"care_gaps": enrollment.CareGaps,
		},
	}
}

// ProcessEventRequest represents a request to process a clinical event via API
type ProcessEventRequest struct {
	EventType   EventType              `json:"event_type" binding:"required"`
	PatientID   string                 `json:"patient_id" binding:"required"`
	EncounterID string                 `json:"encounter_id,omitempty"`
	Data        map[string]interface{} `json:"data" binding:"required"`
}

// ProcessEventResponse represents the response for event processing
type ProcessEventResponse struct {
	Success           bool                       `json:"success"`
	EventID           string                     `json:"event_id"`
	ProcessedAt       time.Time                  `json:"processed_at"`
	EvaluationResults []CriteriaEvaluationResult `json:"evaluation_results,omitempty"`
	EnrollmentsCreated []uuid.UUID               `json:"enrollments_created,omitempty"`
	RiskChanges       []RiskChangeResult         `json:"risk_changes,omitempty"`
	Error             string                     `json:"error,omitempty"`
}

// RiskChangeResult represents a risk tier change that occurred
type RiskChangeResult struct {
	RegistryCode RegistryCode `json:"registry_code"`
	OldTier      RiskTier     `json:"old_tier"`
	NewTier      RiskTier     `json:"new_tier"`
}

// KafkaMessage represents a Kafka message structure
type KafkaMessage struct {
	Topic     string                 `json:"topic"`
	Key       string                 `json:"key"`
	Value     map[string]interface{} `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Partition int32                  `json:"partition,omitempty"`
	Offset    int64                  `json:"offset,omitempty"`
}

// KafkaTopics defines the Kafka topics used by KB-17
var KafkaTopics = struct {
	// Inbound topics
	DiagnosisEvents   string
	LabResultEvents   string
	MedicationEvents  string
	ProblemEvents     string
	VitalSignEvents   string
	EncounterEvents   string

	// Outbound topics
	RegistryEvents    string
}{
	DiagnosisEvents:  "clinical.diagnosis",
	LabResultEvents:  "clinical.lab-results",
	MedicationEvents: "clinical.medications",
	ProblemEvents:    "clinical.problems",
	VitalSignEvents:  "clinical.vital-signs",
	EncounterEvents:  "clinical.encounters",
	RegistryEvents:   "kb17.registry-events",
}
