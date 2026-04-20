package models

import (
	"time"

	"github.com/google/uuid"
)

// ClinicianLabel is the discrete verdict shown to a clinician for one alert attribution.
// Vocabulary matches Gap 21 spec §6.1 and is the stable contract Sprint 2 ML attribution
// will produce with the same values.
type ClinicianLabel string

const (
	LabelPrevented                ClinicianLabel = "prevented"
	LabelNoEffectDetected         ClinicianLabel = "no_effect_detected"
	LabelOutcomeDespiteIntervention ClinicianLabel = "outcome_despite_intervention"
	LabelFragileEstimate          ClinicianLabel = "fragile_estimate"
	LabelInconclusive             ClinicianLabel = "inconclusive"
)

// AttributionVerdict is the output of the attribution engine for a single consolidated
// alert record. Sprint 1 fills in RiskDifference/RiskReductionPct via rule-based
// comparison against the patient's own pre-alert baseline. Sprint 2 (KB-28 Python)
// will replace the math but keep this struct.
type AttributionVerdict struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ConsolidatedRecordID uuid.UUID `gorm:"type:uuid;index;not null" json:"consolidated_record_id"`
	PatientID          string    `gorm:"size:100;index;not null" json:"patient_id"`
	CohortID           string    `gorm:"size:60;index;index:idx_av_cohort_label,priority:1" json:"cohort_id,omitempty"`

	// Attribution outputs
	ClinicianLabel     string    `gorm:"size:40;index;index:idx_av_cohort_label,priority:2;not null" json:"clinician_label"`
	TechnicalLabel     string    `gorm:"size:60" json:"technical_label"`
	RiskDifference     float64   `json:"risk_difference"`
	RiskReductionPct   float64   `json:"risk_reduction_pct"`
	CounterfactualRisk float64   `json:"counterfactual_risk"`
	ObservedOutcome    bool      `json:"observed_outcome"`
	PredictionWindowDays int     `json:"prediction_window_days"`

	// Provenance
	AttributionMethod  string    `gorm:"size:20;not null;default:'RULE_BASED'" json:"attribution_method"`
	MethodVersion      string    `gorm:"size:40" json:"method_version"`
	Rationale          string    `gorm:"type:text" json:"rationale"`
	LedgerEntryID      *uuid.UUID `gorm:"type:uuid" json:"ledger_entry_id,omitempty"`

	ComputedAt         time.Time `gorm:"autoCreateTime" json:"computed_at"`
}

func (AttributionVerdict) TableName() string { return "attribution_verdicts" }

// LedgerEntry is one append-only governance ledger entry. Sprint 1 uses HMAC-SHA256
// chain (each entry includes prior entry's hash); Sprint 2 layers Ed25519 signatures on top.
type LedgerEntry struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Sequence          int64     `gorm:"uniqueIndex;not null" json:"sequence"`
	EntryType         string    `gorm:"size:40;not null" json:"entry_type"`
	SubjectID         string    `gorm:"size:100;index" json:"subject_id"`
	PayloadJSON       string    `gorm:"type:text;not null" json:"payload_json"`
	PriorHash         string    `gorm:"size:64;not null" json:"prior_hash"`
	EntryHash         string    `gorm:"size:64;not null" json:"entry_hash"`
	CreatedAt         time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (LedgerEntry) TableName() string { return "governance_ledger_entries" }
