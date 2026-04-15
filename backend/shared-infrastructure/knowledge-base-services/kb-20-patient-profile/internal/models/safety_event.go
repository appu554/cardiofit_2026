package models

import (
	"time"

	"github.com/google/uuid"
)

// SafetyEvent is a queryable audit row for clinically significant
// safety events observed on a patient. Phase 8 P8-5: the summary-
// context service reads from this table to populate the confounder
// flags (IsAcuteIll, HasRecentTransfusion, HasRecentHypoglycaemia)
// that the MCU gate manager's V-06 stress-hyperglycaemia rule and
// its transfusion/hypoglycaemia guards depend on.
//
// The table is additive to the existing event-bus flow — every
// write happens alongside the existing eventBus.Publish call that
// emits SafetyAlertPayload onto Kafka, and the Kafka stream is
// unchanged. This table adds a second sink (persistent, queryable)
// so downstream summary-context reads can answer "has this patient
// had a recent acute event?" without replaying the Kafka log.
//
// EventType values (the V-06 rule and the hypoglycaemia guard read
// these exact strings):
//   - ACUTE_ILLNESS      — 7-day window for IsAcuteIll flag
//   - BLOOD_TRANSFUSION  — 90-day window for HasRecentTransfusion
//                          (transfusion inflates HbA1c transiently)
//   - HYPO_EVENT         — 30-day window for HasRecentHypoglycaemia
//                          (recent hypo → don't intensify glycaemic
//                          therapy even if HbA1c is above target)
//   - Other event types (EGFR_CRITICAL, POTASSIUM_HIGH, etc.) are
//     also persisted for audit but the summary-context service
//     does not derive flags from them today. Future MCU gate rules
//     can query this table for additional confounders without
//     changing the schema.
type SafetyEvent struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID  string    `gorm:"size:100;index;not null" json:"patient_id"`
	EventType  string    `gorm:"size:40;index;not null" json:"event_type"`
	Severity   string    `gorm:"size:20" json:"severity,omitempty"`
	Description string   `gorm:"type:text" json:"description,omitempty"`

	// LabType / OldValue / NewValue mirror the SafetyAlertPayload
	// fields that are set on lab-triggered safety events. Left empty
	// for non-lab events (hospitalisation, transfusion, etc.).
	LabType  string `gorm:"size:30" json:"lab_type,omitempty"`
	OldValue string `gorm:"size:40" json:"old_value,omitempty"`
	NewValue string `gorm:"size:40" json:"new_value,omitempty"`

	// ObservedAt is the clinical event time — when the acute
	// illness started, when the transfusion was given, when the
	// hypo event occurred. For lab-derived events this is the
	// lab's MeasuredAt; for direct clinical events this is the
	// event timestamp from the upstream source.
	ObservedAt time.Time `gorm:"index;not null" json:"observed_at"`

	// CreatedAt is when the row was written, independent of the
	// clinical event time. Used for audit + late-arriving events
	// where the ObservedAt is historical but the row is fresh.
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName returns the Postgres table name. GORM's default would
// pluralise to "safety_events" which matches the convention used by
// lab_entries and medication_states, but being explicit avoids a
// future surprise if the default pluralisation behavior changes.
func (SafetyEvent) TableName() string { return "safety_events" }

// SafetyEvent type constants. Importers should reference these
// rather than hardcoding strings, because the summary-context
// service queries for these exact values when deriving confounder
// flags. Drift between the writer and reader side would produce
// silent-false flags.
const (
	SafetyEventAcuteIllness     = "ACUTE_ILLNESS"
	SafetyEventBloodTransfusion = "BLOOD_TRANSFUSION"
	SafetyEventHypoEvent        = "HYPO_EVENT"
	SafetyEventEGFRCritical     = "EGFR_CRITICAL"
	SafetyEventPotassiumHigh    = "POTASSIUM_HIGH"
	SafetyEventHypertensiveCrisis = "HYPERTENSIVE_CRISIS"
)
