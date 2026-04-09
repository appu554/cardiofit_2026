package models

import "time"

// GatingVerdict is the renal safety classification for a single medication.
type GatingVerdict string

const (
	VerdictContraindicated  GatingVerdict = "CONTRAINDICATED"
	VerdictDoseReduce       GatingVerdict = "DOSE_REDUCE"
	VerdictMonitorEscalate  GatingVerdict = "MONITOR_ESCALATE"
	VerdictAnticipatory     GatingVerdict = "ANTICIPATORY_ALERT"
	VerdictCleared          GatingVerdict = "CLEARED"
	VerdictInsufficientData GatingVerdict = "INSUFFICIENT_DATA"
)

// RenalDrugRule defines eGFR thresholds for a single drug class.
type RenalDrugRule struct {
	DrugClass              string  `yaml:"drug_class" json:"drug_class"`
	ContraindicatedBelow   float64 `yaml:"contraindicated_below" json:"contraindicated_below"`
	DoseReduceBelow        float64 `yaml:"dose_reduce_below" json:"dose_reduce_below"`
	MaxDoseReducedMg       float64 `yaml:"max_dose_reduced_mg" json:"max_dose_reduced_mg"`
	MonitorEscalateBelow   float64 `yaml:"monitor_escalate_below" json:"monitor_escalate_below"`
	RequiresPotassiumCheck bool    `yaml:"requires_potassium_check" json:"requires_potassium_check"`
	PotassiumContraAbove   float64 `yaml:"potassium_contra_above" json:"potassium_contra_above"`
	EfficacyCliffBelow     float64 `yaml:"efficacy_cliff_below" json:"efficacy_cliff_below"`
	SubstituteClass        string  `yaml:"substitute_class" json:"substitute_class"`
	AnticipateMonths       int     `yaml:"anticipate_months" json:"anticipate_months"`
	SourceGuideline        string  `yaml:"source_guideline" json:"source_guideline"`
	InitiationMinEGFR      float64 `yaml:"initiation_min_egfr,omitempty" json:"initiation_min_egfr,omitempty"`
	ContinuationMinEGFR    float64 `yaml:"continuation_min_egfr,omitempty" json:"continuation_min_egfr,omitempty"`
}

// RenalStatus holds the patient's current renal state for gating decisions.
type RenalStatus struct {
	EGFR                float64    `json:"egfr"`
	EGFRSlope           float64    `json:"egfr_slope"`
	EGFRMeasuredAt      time.Time  `json:"egfr_measured_at"`
	EGFRDataPoints      int        `json:"egfr_data_points"`
	Potassium           *float64   `json:"potassium,omitempty"`
	PotassiumMeasuredAt *time.Time `json:"potassium_measured_at,omitempty"`
	ACR                 *float64   `json:"acr,omitempty"`
	CKDStage            string     `json:"ckd_stage"`
	IsRapidDecliner     bool       `json:"is_rapid_decliner"`
}

// MedicationGatingResult is the full gating output for one medication.
type MedicationGatingResult struct {
	DrugClass           string        `json:"drug_class"`
	DrugName            string        `json:"drug_name,omitempty"`
	CurrentDoseMg       float64       `json:"current_dose_mg,omitempty"`
	Verdict             GatingVerdict `json:"verdict"`
	Reason              string        `json:"reason"`
	ClinicalAction      string        `json:"clinical_action"`
	MaxSafeDoseMg       *float64      `json:"max_safe_dose_mg,omitempty"`
	SubstituteClass     string        `json:"substitute_class,omitempty"`
	MonitoringRequired  []string      `json:"monitoring_required,omitempty"`
	MonitoringFrequency string        `json:"monitoring_frequency,omitempty"`
	TimeToThreshold     *float64      `json:"time_to_threshold_months,omitempty"`
	SourceGuideline     string        `json:"source_guideline"`
	EGFR                float64       `json:"egfr_at_evaluation"`
	EvaluatedAt         time.Time     `json:"evaluated_at"`
}

// PatientGatingReport is the full renal safety report for a patient.
type PatientGatingReport struct {
	PatientID              string                   `json:"patient_id"`
	RenalStatus            RenalStatus              `json:"renal_status"`
	MedicationResults      []MedicationGatingResult  `json:"medication_results"`
	HasContraindicated     bool                     `json:"has_contraindicated"`
	HasDoseReduce          bool                     `json:"has_dose_reduce"`
	StaleEGFR              bool                     `json:"stale_egfr"`
	StaleEGFRDays          int                      `json:"stale_egfr_days,omitempty"`
	OverallUrgency         string                   `json:"overall_urgency"`
	BlockedRecommendations []string                 `json:"blocked_recommendations,omitempty"`
}
