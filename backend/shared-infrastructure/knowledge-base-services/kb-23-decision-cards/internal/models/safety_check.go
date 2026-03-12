package models

// PatientStateEntry is the structured representation of KB-20 patient data
// for CTL Panel 1. It replaces the prose-based ClinicianSummary with
// machine-readable fields that the UI can render directly.
type PatientStateEntry struct {
	Stratum     string   `json:"stratum"`
	CKDSubstage *string  `json:"ckd_substage,omitempty"`
	EGFRValue   float64  `json:"egfr_value,omitempty"`
	LatestHbA1c float64  `json:"latest_hba1c,omitempty"`
	LatestFBG   float64  `json:"latest_fbg,omitempty"`
	WeightKg    float64  `json:"weight_kg,omitempty"`
	Medications []string `json:"medications,omitempty"`
	IsAcuteIll  bool     `json:"is_acute_illness"`
}

// SafetyCheckEntry is the structured representation of a single safety
// evaluation for CTL Panel 3. Each entry records a gate decision, its
// rationale, and any relevant observation reliability or safety flags.
type SafetyCheckEntry struct {
	CheckType             string             `json:"check_type"`
	Gate                  MCUGate            `json:"gate"`
	GateRationale         string             `json:"gate_rationale"`
	ObservationReliability ObservationReliability `json:"observation_reliability"`
	SafetyFlags           []SafetyFlagEntry  `json:"safety_flags,omitempty"`
	StressHyperglycaemia  bool               `json:"stress_hyperglycaemia"`
	HysteresisApplied     bool               `json:"hysteresis_applied"`
	DoseAdjustmentNotes   string             `json:"dose_adjustment_notes,omitempty"`
}

// ConditionCriterion is a single guideline criterion evaluated for CTL Panel 2.
// Each recommendation can have multiple criteria; the overall ConditionStatus
// is derived from the set of criteria met/unmet.
type ConditionCriterion struct {
	CriterionID string          `json:"criterion_id"`
	Description string          `json:"description"`
	Status      ConditionStatus `json:"status"`
	Evidence    string          `json:"evidence,omitempty"`
}

// ReasoningStepEntry is a single step in the KB-22 Bayesian reasoning chain
// for CTL Panel 4. Each step records a question that contributed meaningful
// information gain (|IG| > 0.01) to the diagnostic inference.
type ReasoningStepEntry struct {
	StepNumber      int     `json:"step_number"`
	QuestionID      string  `json:"question_id"`
	QuestionText    string  `json:"question_text"`
	Answer          string  `json:"answer"`
	InformationGain float64 `json:"information_gain"`
	TopDifferential string  `json:"top_differential"`
	TopPosterior    float64 `json:"top_posterior"`
}
