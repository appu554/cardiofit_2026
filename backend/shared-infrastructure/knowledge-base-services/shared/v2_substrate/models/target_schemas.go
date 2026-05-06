package models

import "time"

// Target.Spec JSONB shapes — one struct per Target.Kind constant in enums.go.
//
// MedicineUse.Target.Spec is stored as JSON.RawMessage at the top level so
// the storage layer is shape-agnostic; these structs are the documented
// per-Kind contract that callers may use for type safety. Adding a new
// Target.Kind requires (a) a new constant in enums.go, (b) a new spec
// struct here, and (c) a delegated validator in
// validation/target_validator.go.

// TargetBPThresholdSpec — for Target{Kind: TargetKindBPThreshold}.
//
// Used for antihypertensive medicines where the target is keeping BP below a
// threshold. Both bounds are inclusive maxima.
//
// Example: {"systolic_max": 140, "diastolic_max": 90}
type TargetBPThresholdSpec struct {
	SystolicMax  int `json:"systolic_max"`
	DiastolicMax int `json:"diastolic_max"`
}

// TargetCompletionDateSpec — for Target{Kind: TargetKindCompletionDate}.
//
// Used for both antibiotic course completion AND deprescribing target dates.
// EndDate is the canonical target; DurationDays + Rationale are informational.
//
// Example: {"end_date": "2026-05-15T00:00:00Z", "duration_days": 7, "rationale": "amoxicillin course"}
type TargetCompletionDateSpec struct {
	EndDate      time.Time `json:"end_date"`
	DurationDays int       `json:"duration_days,omitempty"`
	Rationale    string    `json:"rationale,omitempty"`
}

// TargetSymptomResolutionSpec — for Target{Kind: TargetKindSymptomResolution}.
//
// Used for symptomatic (PRN) medicines where the target is the resolution
// of a specified symptom within a monitoring window.
//
// Example: {"target_symptom": "pain", "monitoring_window_days": 14, "snomed_code": "22253000"}
type TargetSymptomResolutionSpec struct {
	TargetSymptom        string `json:"target_symptom"`
	MonitoringWindowDays int    `json:"monitoring_window_days,omitempty"`
	SNOMEDCode           string `json:"snomed_code,omitempty"`
}

// TargetHbA1cBandSpec — for Target{Kind: TargetKindHbA1cBand}.
//
// Used for diabetes medicines where the target is keeping HbA1c within a band.
// Min and Max are both inclusive.
//
// Example: {"min": 6.5, "max": 8.0}
type TargetHbA1cBandSpec struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// TargetOpenSpec — for Target{Kind: TargetKindOpen}.
//
// Used for chronic, indefinite medicines where no specific numerical target
// applies. Rationale captures the clinical justification for ongoing use.
//
// Example: {"rationale": "long-term anticoagulation for AF"}
type TargetOpenSpec struct {
	Rationale string `json:"rationale,omitempty"`
}
