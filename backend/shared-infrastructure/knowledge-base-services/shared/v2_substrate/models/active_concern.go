package models

import (
	"time"

	"github.com/google/uuid"
)

// ActiveConcern represents an open clinical question for a Resident — a
// time-windowed concern with a default expected resolution. Per Layer 2 doc
// §2.3 (lines 508-533).
//
// Active concerns gate downstream rule firing: some rules require an open
// concern of a particular type to fire (e.g. "antibiotic course completion"
// only matters while antibiotic_course_active is open); others are
// suppressed inside concern windows (e.g. baselines computed during
// post_fall_24h are contaminated by acute readings and excluded).
//
// Lifecycle:
//   - opened by an Event (e.g. fall → post_fall_72h) or a MedicineUse
//     insert (e.g. ATC J01 antibiotic → antibiotic_course_active), with
//     ExpectedResolutionAt = StartedAt + concern_type_triggers.default_window_hours
//   - resolved when stop criteria fire (resolved_stop_criteria), an
//     escalation Event lands (escalated), or the SweepExpired cron passes
//     ExpectedResolutionAt without resolution (expired_unresolved). The
//     last is terminal — no transition out of expired_unresolved.
//
// Canonical storage: kb-20-patient-profile (active_concerns table, migration
// 015). Concerns are NOT mapped to a native FHIR concept; the closest
// analogue is FHIR Condition, used for ingress/egress at integration
// boundaries (see shared/v2_substrate/fhir/active_concern_mapper.go).
type ActiveConcern struct {
	ID                          uuid.UUID  `json:"id"`
	ResidentID                  uuid.UUID  `json:"resident_id"`
	ConcernType                 string     `json:"concern_type"` // see ActiveConcern* constants
	StartedAt                   time.Time  `json:"started_at"`
	StartedByEventRef           *uuid.UUID `json:"started_by_event_ref,omitempty"`
	ExpectedResolutionAt        time.Time  `json:"expected_resolution_at"`
	OwnerRoleRef                *uuid.UUID `json:"owner_role_ref,omitempty"`
	RelatedMonitoringPlanRef    *uuid.UUID `json:"related_monitoring_plan_ref,omitempty"`
	ResolutionStatus            string     `json:"resolution_status"` // see ResolutionStatus* constants
	ResolvedAt                  *time.Time `json:"resolved_at,omitempty"`
	ResolutionEvidenceTraceRef  *uuid.UUID `json:"resolution_evidence_trace_ref,omitempty"`
	Notes                       string     `json:"notes,omitempty"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}

// ActiveConcern types (per Layer 2 doc §2.3 + concern_type_triggers seed in
// migration 015). Each type maps to a default_window_hours value and a
// trigger source (event type, ATC class, or manual open).
const (
	// 72-hour post-fall watch — opened by a fall Event. Watches for
	// delayed head injury, post-fall vitals, follow-up assessment.
	ActiveConcernPostFall72h = "post_fall_72h"

	// 24-hour post-fall watch — opened by a fall Event. Used by the
	// systolic-BP baseline exclusion list (Layer 2 §2.2 sysBP config).
	// Emitted alongside post_fall_72h with a tighter window.
	ActiveConcernPostFall24h = "post_fall_24h"

	// 72-hour reconciliation watch after hospital discharge.
	ActiveConcernPostHospitalDischarge72h = "post_hospital_discharge_72h"

	// Antibiotic course active — opened by ATC J01 MedicineUse insert.
	// Watches for C. diff, course completion, missed doses.
	ActiveConcernAntibioticCourseActive = "antibiotic_course_active"

	// 14-day initial psychotropic titration window — opened by ATC N05
	// MedicineUse insert. Resolved by 3 consecutive zero-agitation days.
	ActiveConcernNewPsychotropicTitration = "new_psychotropic_titration_window"

	// Manually-opened acute infection window (72h).
	ActiveConcernAcuteInfectionActive = "acute_infection_active"

	// 30-day end-of-life recognition window — opened by
	// end_of_life_recognition Event before palliative tagging.
	ActiveConcernEndOfLifeRecognition = "end_of_life_recognition_window"

	// 14-day post-deprescribing watch — opened by recommendation
	// lifecycle when a deprescribing recommendation completes.
	ActiveConcernPostDeprescribingMonitoring = "post_deprescribing_monitoring"

	// 72-hour pre-event warning window — manually opened by Layer 3
	// trajectory rules when a warning threshold is crossed.
	ActiveConcernPreEventWarning = "pre_event_warning_window"

	// 14-day deferred-recommendation watch — recommendation awaiting
	// SDM consent.
	ActiveConcernAwaitingConsentReview = "awaiting_consent_review"

	// 30-day deferred-recommendation watch — recommendation awaiting
	// specialist consult.
	ActiveConcernAwaitingSpecialistInput = "awaiting_specialist_input"
)

// IsValidActiveConcernType reports whether s is one of the recognised
// ActiveConcern types. Validators reject unknown types at the model
// boundary; the storage layer additionally enforces this via the
// concern_type_triggers PK FK relationship.
func IsValidActiveConcernType(s string) bool {
	switch s {
	case ActiveConcernPostFall72h, ActiveConcernPostFall24h,
		ActiveConcernPostHospitalDischarge72h,
		ActiveConcernAntibioticCourseActive,
		ActiveConcernNewPsychotropicTitration,
		ActiveConcernAcuteInfectionActive,
		ActiveConcernEndOfLifeRecognition,
		ActiveConcernPostDeprescribingMonitoring,
		ActiveConcernPreEventWarning,
		ActiveConcernAwaitingConsentReview,
		ActiveConcernAwaitingSpecialistInput:
		return true
	}
	return false
}

// ResolutionStatus values for an ActiveConcern. The state machine:
//
//	open → resolved_stop_criteria   (stop criteria fired e.g. 3-day-zero-agitation)
//	open → escalated                 (escalation Event landed e.g. ED presentation)
//	open → expired_unresolved        (SweepExpired cron observed past ExpectedResolutionAt)
//
// expired_unresolved is terminal; no transition out. The other terminal
// states (resolved_stop_criteria, escalated) are also terminal once
// reached but the SweepExpired idempotently leaves them alone.
const (
	ResolutionStatusOpen                = "open"
	ResolutionStatusResolvedStopCriteria = "resolved_stop_criteria"
	ResolutionStatusEscalated           = "escalated"
	ResolutionStatusExpiredUnresolved   = "expired_unresolved"
)

// IsValidResolutionStatus reports whether s is one of the recognised
// ResolutionStatus values.
func IsValidResolutionStatus(s string) bool {
	switch s {
	case ResolutionStatusOpen, ResolutionStatusResolvedStopCriteria,
		ResolutionStatusEscalated, ResolutionStatusExpiredUnresolved:
		return true
	}
	return false
}

// IsTerminalResolutionStatus reports whether s is a terminal status
// (no outbound transitions). Used by the validator to reject illegal
// status transitions in PATCH /active-concerns/:id.
func IsTerminalResolutionStatus(s string) bool {
	switch s {
	case ResolutionStatusResolvedStopCriteria,
		ResolutionStatusEscalated,
		ResolutionStatusExpiredUnresolved:
		return true
	}
	return false
}

// IsValidResolutionTransition reports whether moving from `from` to `to`
// is a legal status transition. The only legal source for non-trivial
// transitions is "open"; once a concern reaches a terminal status, no
// further transitions are allowed.
//
// Self-transitions (from == to) are rejected as no-ops; callers that
// want to update notes/evidence on an already-resolved concern should
// not change resolution_status at all.
func IsValidResolutionTransition(from, to string) bool {
	if !IsValidResolutionStatus(from) || !IsValidResolutionStatus(to) {
		return false
	}
	if from == to {
		return false
	}
	if from != ResolutionStatusOpen {
		return false
	}
	// from == open → any other valid status is reachable.
	return true
}

// EventTypeConcernExpiredUnresolved is the cascade Event type emitted by
// the SweepExpired cron when an open concern passes its
// ExpectedResolutionAt without resolution. Layer 3 rules (e.g. fall risk
// reassessment) consume this event.
//
// Defined here rather than in event.go to keep the concern-lifecycle
// vocabulary in one file; the event type is registered through the
// IsSystemEventType branch via the IsValidConcernCascadeEventType helper
// because system_events route to FHIR Communication.
const EventTypeConcernExpiredUnresolved = "concern_expired_unresolved"

// IsValidConcernCascadeEventType reports whether s is one of the cascade
// event types the active-concern engine produces. Currently only
// concern_expired_unresolved; reserved for future cascade events.
func IsValidConcernCascadeEventType(s string) bool {
	return s == EventTypeConcernExpiredUnresolved
}
