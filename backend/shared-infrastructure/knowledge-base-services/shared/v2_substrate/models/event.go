package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event represents a v2 substrate Event entity for a Resident — something
// that occurred and has legal, regulatory, or workflow significance.
//
// Event vs Observation: an Observation is a clinical fact about the
// resident's state (e.g. post-fall blood pressure). An Event is a
// thing that occurred (e.g. the fall itself, mandatory-reportable under
// the Quality Indicator Program). Per Layer 2 doc §1.5.
//
// Canonical storage: kb-20-patient-profile (events table, greenfield in
// migration 009).
//
// FHIR boundary: clinical / care transition / administrative events map
// to AU FHIR Encounter; system events (rule_fire, recommendation_*, etc.)
// map to AU FHIR Communication. Routing is by EventType bucket; see
// shared/v2_substrate/fhir/event_mapper.go. Vaidshala-specific fields
// (Severity, ReportableUnder, TriggeredStateChanges, DescriptionStructured,
// related-entity refs) are encoded as Vaidshala-namespaced extensions.
type Event struct {
	ID                    uuid.UUID              `json:"id"`
	EventType             string                 `json:"event_type"` // see EventType* constants
	OccurredAt            time.Time              `json:"occurred_at"`
	OccurredAtFacility    *uuid.UUID             `json:"occurred_at_facility,omitempty"` // some system events have no facility
	ResidentID            uuid.UUID              `json:"resident_id"`
	ReportedByRef         uuid.UUID              `json:"reported_by_ref"` // Role.id
	WitnessedByRefs       []uuid.UUID            `json:"witnessed_by_refs,omitempty"`
	Severity              string                 `json:"severity,omitempty"` // see EventSeverity* constants
	DescriptionStructured json.RawMessage        `json:"description_structured,omitempty"`
	DescriptionFreeText   string                 `json:"description_free_text,omitempty"`
	RelatedObservations   []uuid.UUID            `json:"related_observations,omitempty"`
	RelatedMedicationUses []uuid.UUID            `json:"related_medication_uses,omitempty"`
	TriggeredStateChanges []TriggeredStateChange `json:"triggered_state_changes,omitempty"`
	ReportableUnder       []string               `json:"reportable_under,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// TriggeredStateChange describes one state-machine transition that this
// Event drove. StateMachine is one of the EventStateMachine* constants;
// StateChange is opaque JSON whose shape is per-state-machine (validated
// downstream by the state-machine evaluator, not at the Event level).
type TriggeredStateChange struct {
	StateMachine string          `json:"state_machine"`
	StateChange  json.RawMessage `json:"state_change"`
}

// Event types (per Layer 2 doc §1.5).
//
// Bucketed for FHIR routing: Clinical / Care transitions / Administrative
// map to FHIR Encounter; System events map to FHIR Communication.
// IsClinicalEventType / IsCareTransitionEventType /
// IsAdministrativeEventType / IsSystemEventType report the bucket.
const (
	// Clinical events
	EventTypeFall                = "fall"                 // QI Program reportable
	EventTypePressureInjury      = "pressure_injury"      // QI Program reportable
	EventTypeBehaviouralIncident = "behavioural_incident" // restrictive practice trigger
	EventTypeMedicationError     = "medication_error"     // SIRS reportable
	EventTypeAdverseDrugEvent    = "adverse_drug_event"   // causality assessment trigger

	// Care transitions
	EventTypeHospitalAdmission              = "hospital_admission"  // MHR-trackable
	EventTypeHospitalDischarge              = "hospital_discharge"  // reconciliation trigger
	EventTypeGPVisit                        = "GP_visit"
	EventTypeSpecialistVisit                = "specialist_visit"
	EventTypeEmergencyDepartmentPresentation = "emergency_department_presentation"
	EventTypeEndOfLifeRecognition           = "end_of_life_recognition" // care intensity transition
	EventTypeDeath                          = "death"

	// Administrative events
	EventTypeAdmissionToFacility       = "admission_to_facility"
	EventTypeTransferBetweenFacilities = "transfer_between_facilities"
	EventTypeCarePlanningMeeting       = "care_planning_meeting"
	EventTypeFamilyMeeting             = "family_meeting"

	// System events (for EvidenceTrace)
	EventTypeRuleFire                  = "rule_fire"
	EventTypeRecommendationSubmitted   = "recommendation_submitted"
	EventTypeRecommendationDecided     = "recommendation_decided"
	EventTypeMonitoringPlanActivated   = "monitoring_plan_activated"
	EventTypeConsentGrantedOrWithdrawn = "consent_granted_or_withdrawn"
	EventTypeCredentialVerifiedOrExpired = "credential_verified_or_expired"
)

// IsValidEventType reports whether s is one of the recognised EventType values.
func IsValidEventType(s string) bool {
	return IsClinicalEventType(s) || IsCareTransitionEventType(s) ||
		IsAdministrativeEventType(s) || IsSystemEventType(s)
}

// IsClinicalEventType reports whether s is a Clinical-bucket event type.
func IsClinicalEventType(s string) bool {
	switch s {
	case EventTypeFall, EventTypePressureInjury, EventTypeBehaviouralIncident,
		EventTypeMedicationError, EventTypeAdverseDrugEvent:
		return true
	}
	return false
}

// IsCareTransitionEventType reports whether s is a Care-transitions-bucket event type.
func IsCareTransitionEventType(s string) bool {
	switch s {
	case EventTypeHospitalAdmission, EventTypeHospitalDischarge,
		EventTypeGPVisit, EventTypeSpecialistVisit,
		EventTypeEmergencyDepartmentPresentation,
		EventTypeEndOfLifeRecognition, EventTypeDeath:
		return true
	}
	return false
}

// IsAdministrativeEventType reports whether s is an Administrative-bucket event type.
func IsAdministrativeEventType(s string) bool {
	switch s {
	case EventTypeAdmissionToFacility, EventTypeTransferBetweenFacilities,
		EventTypeCarePlanningMeeting, EventTypeFamilyMeeting:
		return true
	}
	return false
}

// IsSystemEventType reports whether s is a System-bucket event type
// (i.e. routed to FHIR Communication, not Encounter).
func IsSystemEventType(s string) bool {
	switch s {
	case EventTypeRuleFire, EventTypeRecommendationSubmitted,
		EventTypeRecommendationDecided, EventTypeMonitoringPlanActivated,
		EventTypeConsentGrantedOrWithdrawn,
		EventTypeCredentialVerifiedOrExpired:
		return true
	}
	return false
}

// Event severity (per Layer 2 doc §1.5: minor | moderate | major | sentinel).
// "sentinel" carries SIRS (Serious Incident Response Scheme) connotation.
const (
	EventSeverityMinor    = "minor"
	EventSeverityModerate = "moderate"
	EventSeverityMajor    = "major"
	EventSeveritySentinel = "sentinel"
)

// IsValidEventSeverity reports whether s is one of the recognised
// EventSeverity values.
func IsValidEventSeverity(s string) bool {
	switch s {
	case EventSeverityMinor, EventSeverityModerate,
		EventSeverityMajor, EventSeveritySentinel:
		return true
	}
	return false
}

// State-machine identifiers used in TriggeredStateChange.StateMachine.
// Per Layer 2 doc §1.5.
const (
	EventStateMachineRecommendation = "Recommendation"
	EventStateMachineMonitoring     = "Monitoring"
	EventStateMachineAuthorisation  = "Authorisation"
	EventStateMachineConsent        = "Consent"
	EventStateMachineClinicalState  = "ClinicalState"
)

// IsValidEventStateMachine reports whether s is a recognised state-machine
// identifier for TriggeredStateChange.StateMachine.
func IsValidEventStateMachine(s string) bool {
	switch s {
	case EventStateMachineRecommendation, EventStateMachineMonitoring,
		EventStateMachineAuthorisation, EventStateMachineConsent,
		EventStateMachineClinicalState:
		return true
	}
	return false
}
