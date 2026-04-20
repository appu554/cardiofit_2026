package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// OverlapStatus is the hard-guard outcome of the propensity overlap check.
// Spec §6.1: CATE_INCONCLUSIVE_NO_OVERLAP is returned verbatim when overlap fails.
type OverlapStatus string

const (
	OverlapPass             OverlapStatus = "OVERLAP_PASS"
	OverlapBelowFloor       OverlapStatus = "OVERLAP_BELOW_FLOOR"   // propensity < band[0]
	OverlapAboveCeiling     OverlapStatus = "OVERLAP_ABOVE_CEILING" // propensity > band[1]
	OverlapInsufficientData OverlapStatus = "OVERLAP_INSUFFICIENT_DATA"
)

// LearnerType identifies which CATE estimator produced the estimate. Sprint 1 ships
// BASELINE_DIFF_MEANS only; Sprint 2 adds S/T/X/DR/R/CAUSAL_FOREST behind the same
// contract. The enum is the vehicle for per-cohort primary-learner selection (§6.1).
type LearnerType string

const (
	LearnerBaselineDiffMeans LearnerType = "BASELINE_DIFF_MEANS"
	LearnerS                 LearnerType = "S_LEARNER"
	LearnerT                 LearnerType = "T_LEARNER"
	LearnerX                 LearnerType = "X_LEARNER"
	LearnerDR                LearnerType = "DR_LEARNER"
	LearnerR                 LearnerType = "R_LEARNER"
	LearnerCausalForest      LearnerType = "CAUSAL_FOREST"
)

// CATEConfidenceLabel is the clinician-facing confidence tier derived from CI width +
// overlap status. Populated by ConfidenceLabel() on read.
type CATEConfidenceLabel string

const (
	CATEConfidenceHigh   CATEConfidenceLabel = "HIGH"
	CATEConfidenceMedium CATEConfidenceLabel = "MEDIUM"
	CATEConfidenceLow    CATEConfidenceLabel = "LOW"
)

// FeatureContribution is one row in the top-K feature attribution table. Sprint 1 uses
// cohort-bucket-membership deltas; Sprint 2 replaces with SHAP without changing the shape.
type FeatureContribution struct {
	FeatureKey   string  `json:"feature_key"`
	Contribution float64 `json:"contribution"` // signed: positive pushes CATE up
	PatientValue float64 `json:"patient_value"`
	CohortMean   float64 `json:"cohort_mean"`
}

// CATEEstimate is the per-patient × per-intervention causal estimate.
// This is the stable Sprint 1 → Sprint N contract. Every downstream consumer reads
// this shape only.
type CATEEstimate struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ConsolidatedRecordID uuid.UUID `gorm:"type:uuid;index;not null" json:"consolidated_record_id"`
	PatientID            string    `gorm:"size:100;index;not null" json:"patient_id"`
	CohortID             string    `gorm:"size:60;index;not null" json:"cohort_id"`
	InterventionID       string    `gorm:"size:80;index;not null" json:"intervention_id"`
	LearnerType          LearnerType `gorm:"size:30;not null" json:"learner_type"`

	PointEstimate float64 `json:"point_estimate"`
	CILower       float64 `json:"ci_lower"`
	CIUpper       float64 `json:"ci_upper"`
	HorizonDays   int     `json:"horizon_days"`

	// Propensity and overlap
	Propensity    float64       `json:"propensity"`
	OverlapStatus OverlapStatus `gorm:"size:40;index;not null" json:"overlap_status"`

	// Cohort context (shown in explanation layer Sprint 3)
	TrainingN      int `json:"training_n"`
	CohortTreatedN int `json:"cohort_treated_n"`
	CohortControlN int `json:"cohort_control_n"`

	// Feature contributions (top-K, signed)
	FeatureContributionsJSON string         `gorm:"type:text" json:"-"`
	FeatureContributionKeys  pq.StringArray `gorm:"type:text[]" json:"feature_contribution_keys"`

	// Provenance
	ModelVersion  string     `gorm:"size:40" json:"model_version"`
	LedgerEntryID *uuid.UUID `gorm:"type:uuid" json:"ledger_entry_id,omitempty"`
	ComputedAt    time.Time  `gorm:"autoCreateTime;index" json:"computed_at"`
}

func (CATEEstimate) TableName() string { return "cate_estimates" }

// IsActionable is the single short-circuit used by the recommender (Sprint 3).
// Spec §6.1: CATE estimates without overlap pass are never shown, regardless of point value.
func (e CATEEstimate) IsActionable() bool {
	return e.OverlapStatus == OverlapPass
}

// ConfidenceLabel derives the 3-tier label from CI width + overlap status. Spec §6.1.
// Width thresholds are deliberately not per-cohort in Sprint 1; Sprint 3 explanation
// layer will YAML-configure per market.
func (e CATEEstimate) ConfidenceLabel() CATEConfidenceLabel {
	if e.OverlapStatus != OverlapPass {
		return CATEConfidenceLow
	}
	width := e.CIUpper - e.CILower
	switch {
	case width <= 0.06:
		return CATEConfidenceHigh
	case width <= 0.20:
		return CATEConfidenceMedium
	default:
		return CATEConfidenceLow
	}
}

// CATEPrimaryLearnerAssignment records which learner is the cohort × intervention × horizon
// primary. Sprint 1 populates once at service start from cate_parameters.yaml; Sprint 2
// makes it an output of the Qini-based selection pipeline (spec §6.1). Appended to ledger
// as CATE_LEARNER_ASSIGNMENT.
type CATEPrimaryLearnerAssignment struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CohortID       string      `gorm:"size:60;uniqueIndex:idx_cohort_intv_horizon,priority:1;not null" json:"cohort_id"`
	InterventionID string      `gorm:"size:80;uniqueIndex:idx_cohort_intv_horizon,priority:2;not null" json:"intervention_id"`
	HorizonDays    int         `gorm:"uniqueIndex:idx_cohort_intv_horizon,priority:3;not null" json:"horizon_days"`
	LearnerType    LearnerType `gorm:"size:30;not null" json:"learner_type"`
	AssignedAt     time.Time   `gorm:"autoCreateTime" json:"assigned_at"`
	LedgerEntryID  *uuid.UUID  `gorm:"type:uuid" json:"ledger_entry_id,omitempty"`
}

func (CATEPrimaryLearnerAssignment) TableName() string { return "cate_primary_learner_assignments" }
