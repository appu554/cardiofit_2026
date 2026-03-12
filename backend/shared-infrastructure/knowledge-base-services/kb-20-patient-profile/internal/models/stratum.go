package models

import (
	"encoding/json"
)

// StratumResponse is returned by GET /patient/:id/stratum/:node_id.
// Finding F-03 (RED): ckd_substage provides G3a vs G3b visibility even
// when stratum_label remains DM_HTN_CKD.
type StratumResponse struct {
	PatientID           string           `json:"patient_id"`
	NodeID              string           `json:"node_id"`
	StratumLabel        string           `json:"stratum_label"`
	EGFR                *float64         `json:"egfr,omitempty"`
	EGFRSlope           float64          `json:"egfr_slope_per_year"`   // mL/min/1.73m² per year (linear regression)
	EGFRTrajectoryClass string           `json:"egfr_trajectory_class"` // STABLE, SLOW_DECLINE, RAPID_DECLINE, etc.
	CKDSubstage         string           `json:"ckd_substage,omitempty"`
	IsProvisional       bool             `json:"is_provisional"`
	ActiveModifiers     []ActiveModifier `json:"active_modifiers"`
	SafetyOverrides     []SafetyOverride `json:"safety_overrides"`
}

// ActiveModifier represents a context modifier active for this patient/node combination.
type ActiveModifier struct {
	ModifierID         string  `json:"modifier_id"`
	ModifierType       string  `json:"modifier_type"`
	DrugClassTrigger   string  `json:"drug_class_trigger"`
	Effect             string  `json:"effect"`
	TargetDifferential string  `json:"target_differential"`
	Magnitude          float64 `json:"magnitude"`
	CompletenessGrade  string  `json:"completeness_grade"`
	EffectiveMagnitude float64 `json:"effective_magnitude"`
}

// SafetyOverride represents a medication safety alert included in the stratum response.
type SafetyOverride struct {
	DrugClass      string `json:"drug_class"`
	AlertType      string `json:"alert_type"`
	Severity       string `json:"severity"`
	Message        string `json:"message"`
	RequiredAction string `json:"required_action"`
}

// Stratum label constants
const (
	StratumDMHTN      = "DM_HTN"
	StratumDMHTNCKD   = "DM_HTN_CKD"
	StratumDMHTNCKDHF = "DM_HTN_CKD_HF"
	StratumDMOnly     = "DM_ONLY"
	StratumHTNOnly    = "HTN_ONLY"
)

// CKD substage constants
const (
	CKDG1  = "G1"
	CKDG2  = "G2"
	CKDG3a = "G3a"
	CKDG3b = "G3b"
	CKDG4  = "G4"
	CKDG5  = "G5"
)

// MedicationThreshold defines an eGFR boundary and its medication implications.
type MedicationThreshold struct {
	EGFRBoundary       float64  `json:"egfr_boundary"`
	AffectedDrugClass  string   `json:"affected_drug_class"`
	RequiredAction     string   `json:"required_action"`
	MaxDoseMg          *float64 `json:"max_dose_mg,omitempty"`
}

// MedicationThresholds defines the clinically significant eGFR boundaries.
var MedicationThresholds = []MedicationThreshold{
	{EGFRBoundary: 60, AffectedDrugClass: DrugClassMetformin, RequiredAction: "MONITOR_RENAL_FUNCTION"},
	{EGFRBoundary: 45, AffectedDrugClass: DrugClassMetformin, RequiredAction: "CAP_DOSE_1500MG", MaxDoseMg: floatPtr(1500)},
	{EGFRBoundary: 30, AffectedDrugClass: DrugClassMetformin, RequiredAction: "REDUCE_DOSE_500_1000MG", MaxDoseMg: floatPtr(1000)},
	{EGFRBoundary: 30, AffectedDrugClass: DrugClassSGLT2I, RequiredAction: "EFFICACY_REDUCED_NOTE"},
	{EGFRBoundary: 15, AffectedDrugClass: DrugClassMetformin, RequiredAction: "DISCONTINUE"},
}

func floatPtr(v float64) *float64 {
	return &v
}

// MarshalJSON implements custom JSON marshaling for StratumResponse.
func (s StratumResponse) MarshalJSON() ([]byte, error) {
	type Alias StratumResponse
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(s),
	})
}
