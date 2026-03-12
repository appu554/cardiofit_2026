// Package models provides domain models for KB-19 Protocol Orchestrator.
//
// PatientContext represents the complete clinical snapshot for protocol arbitration.
// It aggregates data from multiple sources: Vaidshala CQL Engine, KB-8 Calculators,
// ICU Intelligence, and FHIR resources.
package models

import (
	"time"

	"github.com/google/uuid"
)

// PatientContext represents the complete clinical snapshot for a patient at a point in time.
// This is the primary input to the arbitration engine.
type PatientContext struct {
	// Core identifiers
	PatientID   uuid.UUID `json:"patient_id"`
	EncounterID uuid.UUID `json:"encounter_id"`

	// Demographics and patient characteristics
	Demographics Demographics `json:"demographics"`

	// Current vital signs
	Vitals VitalSigns `json:"vitals"`

	// Recent laboratory values
	Labs []LabValue `json:"labs"`

	// Active diagnoses/conditions
	Diagnoses []Diagnosis `json:"diagnoses"`

	// ICU clinical state summary (from ICU Intelligence module)
	ICUStateSummary *ICUClinicalState `json:"icu_state_summary,omitempty"`

	// Known comorbidities affecting protocol selection
	Comorbidities []Comorbidity `json:"comorbidities"`

	// Pregnancy status for teratogenic risk assessment
	PregnancyStatus *PregnancyStatus `json:"pregnancy_status,omitempty"`

	// Current active medications
	MedicationList []ActiveMedication `json:"medication_list"`

	// CQL truth flags from Vaidshala CQL Engine
	// Keys are CQL fact IDs, values indicate whether the condition is true
	// Example: {"HasHFrEF": true, "OnARNI": false, "HasAKI": true}
	CQLTruthFlags map[string]bool `json:"cql_truth_flags"`

	// Calculator scores from KB-8
	// Keys are calculator IDs, values are computed scores
	// Example: {"CHA2DS2VASc": 4.0, "SOFA": 8.0, "APACHE_II": 22.0}
	CalculatorScores map[string]float64 `json:"calculator_scores"`

	// Timestamp when this context was created
	Timestamp time.Time `json:"timestamp"`
}

// Demographics holds patient demographic information.
type Demographics struct {
	Age            int     `json:"age"`
	AgeUnit        string  `json:"age_unit"` // years, months, days
	Gender         string  `json:"gender"`   // male, female, other
	WeightKg       float64 `json:"weight_kg"`
	HeightCm       float64 `json:"height_cm"`
	BSA            float64 `json:"bsa"`             // Body Surface Area (m2)
	IBW            float64 `json:"ibw"`             // Ideal Body Weight (kg)
	Ethnicity      string  `json:"ethnicity"`       // For race-adjusted calculations
	IsPregnant     bool    `json:"is_pregnant"`
	GestationalAge *int    `json:"gestational_age"` // weeks, if pregnant
}

// VitalSigns represents current vital sign measurements.
type VitalSigns struct {
	SystolicBP     int     `json:"systolic_bp"`      // mmHg
	DiastolicBP    int     `json:"diastolic_bp"`     // mmHg
	MAP            int     `json:"map"`              // Mean Arterial Pressure, mmHg
	HeartRate      int     `json:"heart_rate"`       // bpm
	RespiratoryRate int    `json:"respiratory_rate"` // breaths/min
	Temperature    float64 `json:"temperature"`      // Celsius
	SpO2           int     `json:"spo2"`             // %
	FiO2           float64 `json:"fio2"`             // Fraction
	GCS            int     `json:"gcs"`              // Glasgow Coma Scale (3-15)
	MeasuredAt     time.Time `json:"measured_at"`
}

// LabValue represents a single laboratory result.
type LabValue struct {
	Code       string    `json:"code"`        // LOINC code
	Name       string    `json:"name"`        // Display name
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RefLow     float64   `json:"ref_low"`
	RefHigh    float64   `json:"ref_high"`
	IsAbnormal bool      `json:"is_abnormal"`
	Trend      string    `json:"trend"`       // rising, falling, stable
	MeasuredAt time.Time `json:"measured_at"`
}

// Diagnosis represents an active clinical diagnosis.
type Diagnosis struct {
	Code          string    `json:"code"`           // ICD-10 or SNOMED code
	System        string    `json:"system"`         // ICD10, SNOMEDCT
	Display       string    `json:"display"`
	Category      string    `json:"category"`       // acute, chronic, resolved
	Severity      string    `json:"severity"`       // mild, moderate, severe
	OnsetDate     time.Time `json:"onset_date"`
	IsPrimary     bool      `json:"is_primary"`
	IsConfirmed   bool      `json:"is_confirmed"`
}

// ICUClinicalState represents the 8-dimensional ICU clinical state.
// This is provided by the ICU Intelligence module in Vaidshala.
type ICUClinicalState struct {
	// Hemodynamic state
	ShockState          string  `json:"shock_state"`          // NONE, COMPENSATED, UNCOMPENSATED
	VasopressorScore    float64 `json:"vasopressor_score"`    // 0-4
	FluidBalance        float64 `json:"fluid_balance_ml"`     // Net fluid balance in mL
	CardiacOutput       string  `json:"cardiac_output"`       // LOW, NORMAL, HIGH

	// Respiratory state
	VentilationMode     string  `json:"ventilation_mode"`     // NONE, NIPPV, MECHANICAL
	PFRatio             float64 `json:"pf_ratio"`             // PaO2/FiO2
	ARDSSeverity        string  `json:"ards_severity"`        // NONE, MILD, MODERATE, SEVERE

	// Renal state
	AKIStage            int     `json:"aki_stage"`            // 0-3 (KDIGO)
	RRTStatus           string  `json:"rrt_status"`           // NONE, INTERMITTENT, CONTINUOUS
	UrineOutput         float64 `json:"urine_output"`         // mL/kg/hr

	// Hepatic state
	ChildPughClass      string  `json:"child_pugh_class"`     // A, B, C
	ChildPughScore      int     `json:"child_pugh_score"`     // 5-15
	Bilirubin           float64 `json:"bilirubin"`            // mg/dL
	INR                 float64 `json:"inr"`

	// Neurological state
	SedationLevel       string  `json:"sedation_level"`       // RASS score category
	RASSScore           int     `json:"rass_score"`           // -5 to +4
	HasDelirium         bool    `json:"has_delirium"`
	CAMICUPositive      bool    `json:"cam_icu_positive"`

	// Coagulation state
	DICScore            int     `json:"dic_score"`            // ISTH DIC score
	PlateletsLow        bool    `json:"platelets_low"`        // <50,000
	BleedingRisk        string  `json:"bleeding_risk"`        // LOW, MODERATE, HIGH

	// Infection state
	SepsisStatus        string  `json:"sepsis_status"`        // NONE, SIRS, SEPSIS, SEPTIC_SHOCK
	SOFAScore           int     `json:"sofa_score"`           // 0-24
	QSOFAScore          int     `json:"qsofa_score"`          // 0-3
	InfectionSite       string  `json:"infection_site"`       // e.g., PULMONARY, ABDOMINAL

	// Metabolic state
	GlucoseControl      string  `json:"glucose_control"`      // HYPOGLYCEMIC, NORMAL, HYPERGLYCEMIC
	LactateLevel        float64 `json:"lactate_level"`        // mmol/L
	AcidBaseStatus      string  `json:"acid_base_status"`     // ACIDOSIS, NORMAL, ALKALOSIS

	// Timestamp
	EvaluatedAt         time.Time `json:"evaluated_at"`
}

// Comorbidity represents a known comorbid condition.
type Comorbidity struct {
	Code          string `json:"code"`           // SNOMED or ICD-10
	Display       string `json:"display"`
	Category      string `json:"category"`       // cardiac, renal, hepatic, etc.
	CharlsonScore int    `json:"charlson_score"` // Contribution to Charlson Comorbidity Index
}

// PregnancyStatus represents pregnancy-related clinical information.
type PregnancyStatus struct {
	IsPregnant        bool      `json:"is_pregnant"`
	GestationalAge    int       `json:"gestational_age"`     // weeks
	Trimester         int       `json:"trimester"`           // 1, 2, 3
	EDD               time.Time `json:"edd"`                 // Estimated Due Date
	HighRiskPregnancy bool      `json:"high_risk_pregnancy"`
	Complications     []string  `json:"complications"`       // preeclampsia, gestational diabetes, etc.
}

// ActiveMedication represents a currently active medication.
type ActiveMedication struct {
	RxNormCode     string    `json:"rxnorm_code"`
	Name           string    `json:"name"`
	Dose           float64   `json:"dose"`
	DoseUnit       string    `json:"dose_unit"`
	Route          string    `json:"route"`           // PO, IV, SC, etc.
	Frequency      string    `json:"frequency"`       // daily, BID, TID, etc.
	StartDate      time.Time `json:"start_date"`
	IsHighAlert    bool      `json:"is_high_alert"`
	DrugCategory   string    `json:"drug_category"`   // anticoagulant, antihypertensive, etc.
}

// NewPatientContext creates a new PatientContext with initialized maps and timestamp.
func NewPatientContext(patientID, encounterID uuid.UUID) *PatientContext {
	return &PatientContext{
		PatientID:        patientID,
		EncounterID:      encounterID,
		CQLTruthFlags:    make(map[string]bool),
		CalculatorScores: make(map[string]float64),
		Timestamp:        time.Now(),
	}
}

// HasCriticalVitals returns true if any vital sign is critically abnormal.
func (ctx *PatientContext) HasCriticalVitals() bool {
	v := ctx.Vitals
	return v.SystolicBP < 90 || v.SystolicBP > 180 ||
		v.HeartRate < 40 || v.HeartRate > 150 ||
		v.SpO2 < 88 ||
		v.GCS < 9
}

// IsICU returns true if the patient appears to be in an ICU setting.
func (ctx *PatientContext) IsICU() bool {
	return ctx.ICUStateSummary != nil
}

// HasDiagnosis checks if the patient has a specific diagnosis by code.
func (ctx *PatientContext) HasDiagnosis(code string) bool {
	for _, dx := range ctx.Diagnoses {
		if dx.Code == code {
			return true
		}
	}
	return false
}

// GetCQLFlag returns the value of a CQL truth flag, defaulting to false if not present.
func (ctx *PatientContext) GetCQLFlag(flagID string) bool {
	if val, ok := ctx.CQLTruthFlags[flagID]; ok {
		return val
	}
	return false
}

// GetCalculatorScore returns the value of a calculator score, defaulting to 0 if not present.
func (ctx *PatientContext) GetCalculatorScore(calculatorID string) float64 {
	if val, ok := ctx.CalculatorScores[calculatorID]; ok {
		return val
	}
	return 0
}
