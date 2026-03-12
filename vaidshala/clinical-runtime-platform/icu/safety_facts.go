package icu

import "time"

// SafetyFacts contains all clinical facts needed for dominance classification.
// These facts are gathered from real-time monitoring and FHIR resources.
//
// ARCHITECTURE NOTE:
// This struct is the INPUT to ClassifyDominanceState(). It is populated by:
//   - Real-time vital sign streams
//   - FHIR Observation resources (via KB-7 snapshots)
//   - ADT events (admission/discharge/transfer)
//   - Medication administration records
//
// The facts are IMMUTABLE once gathered - the classifier operates on a point-in-time snapshot.
type SafetyFacts struct {
	// ═══════════════════════════════════════════════════════════════════════════
	// CONTEXT FLAGS - Determine if dominance evaluation is applicable
	// ═══════════════════════════════════════════════════════════════════════════

	// IsInICU indicates patient is currently in an ICU location.
	// Derived from: ADT Location, Encounter.location.physicalType
	IsInICU bool `json:"is_in_icu"`

	// IsCodeActive indicates Code Blue/ACLS is in progress.
	// Derived from: Code Blue alert, resuscitation order active
	IsCodeActive bool `json:"is_code_active"`

	// IsCriticallyUnstable is derived from vital sign trend analysis.
	// True if: rapid deterioration detected by early warning score
	IsCriticallyUnstable bool `json:"is_critically_unstable"`

	// ═══════════════════════════════════════════════════════════════════════════
	// NEUROLOGIC PARAMETERS (Priority 1: NEUROLOGIC_COLLAPSE)
	// Triggers: GCS <8, Active seizure, ICP >20, Herniation signs
	// ═══════════════════════════════════════════════════════════════════════════

	// GCS is the Glasgow Coma Scale score (3-15).
	// Source: FHIR Observation, LOINC 9269-2
	GCS int `json:"gcs"`

	// HasActiveSeizure indicates witnessed seizure activity.
	// Source: Clinical documentation, EEG monitoring
	HasActiveSeizure bool `json:"has_active_seizure"`

	// ICP is intracranial pressure in mmHg.
	// Source: ICP monitor, FHIR Observation
	ICP float64 `json:"icp"`

	// HasHerniationSigns indicates Cushing's triad or blown pupil.
	// Source: Neuro exam, vital sign pattern (HTN + bradycardia + irregular resp)
	HasHerniationSigns bool `json:"has_herniation_signs"`

	// ═══════════════════════════════════════════════════════════════════════════
	// HEMODYNAMIC PARAMETERS (Priority 2: SHOCK)
	// Triggers: MAP <65, Lactate >4, Vasopressor requirement, Septic shock
	// ═══════════════════════════════════════════════════════════════════════════

	// MAP is mean arterial pressure in mmHg.
	// Source: Arterial line or calculated from NIBP
	// Formula: MAP = DBP + (SBP - DBP) / 3
	MAP float64 `json:"map"`

	// Lactate is serum lactate in mmol/L.
	// Source: ABG, point-of-care testing, LOINC 2524-7
	Lactate float64 `json:"lactate"`

	// OnVasopressors indicates any vasopressor infusion is active.
	// Includes: Norepinephrine, Epinephrine, Vasopressin, Dopamine, Phenylephrine
	OnVasopressors bool `json:"on_vasopressors"`

	// HasSepticShock indicates Sepsis-3 criteria met with vasopressor requirement.
	// Source: qSOFA, SOFA score, clinical documentation
	HasSepticShock bool `json:"has_septic_shock"`

	// ═══════════════════════════════════════════════════════════════════════════
	// RESPIRATORY PARAMETERS (Priority 3: HYPOXIA)
	// Triggers: SpO2 <88%, P/F ratio <100, FiO2 >0.6
	// ═══════════════════════════════════════════════════════════════════════════

	// SpO2 is oxygen saturation percentage (0-100).
	// Source: Pulse oximetry, LOINC 2708-6
	SpO2 float64 `json:"spo2"`

	// PFRatio is PaO2/FiO2 ratio (Berlin ARDS criteria).
	// Source: ABG PaO2 / ventilator FiO2
	// <100 = Severe ARDS, 100-200 = Moderate, 200-300 = Mild
	PFRatio float64 `json:"pf_ratio"`

	// FiO2 is fraction of inspired oxygen (0.21 to 1.0).
	// Source: Ventilator settings, supplemental O2 device
	FiO2 float64 `json:"fio2"`

	// OnMechanicalVent indicates patient is intubated/ventilated.
	// Source: Ventilator flowsheet, procedure documentation
	OnMechanicalVent bool `json:"on_mechanical_vent"`

	// ═══════════════════════════════════════════════════════════════════════════
	// BLEEDING PARAMETERS (Priority 4: ACTIVE_BLEED)
	// Triggers: Hgb drop >2g/dL/6h, Active transfusion, Surgical bleeding
	// ═══════════════════════════════════════════════════════════════════════════

	// HgbDrop6h is hemoglobin drop in last 6 hours (g/dL).
	// Source: Serial CBC comparison
	HgbDrop6h float64 `json:"hgb_drop_6h"`

	// CurrentHgb is current hemoglobin level (g/dL).
	// Source: CBC, LOINC 718-7
	CurrentHgb float64 `json:"current_hgb"`

	// HasActiveTransfusion indicates PRBC transfusion in progress.
	// Source: Blood bank, MAR
	HasActiveTransfusion bool `json:"has_active_transfusion"`

	// HasSurgicalBleeding indicates post-op or traumatic bleeding.
	// Source: Surgical documentation, drain output
	HasSurgicalBleeding bool `json:"has_surgical_bleeding"`

	// INR is International Normalized Ratio.
	// Source: Coagulation panel, LOINC 6301-6
	INR float64 `json:"inr"`

	// HasActiveBleeding indicates clinically evident bleeding.
	// Source: Clinical documentation, GI bleed, wound assessment
	HasActiveBleeding bool `json:"has_active_bleeding"`

	// ═══════════════════════════════════════════════════════════════════════════
	// CARDIAC OUTPUT PARAMETERS (Priority 5: LOW_OUTPUT_FAILURE)
	// Triggers: CI <2.0, ScvO2 <60%, Inotrope escalation, AKI + ALF
	// ═══════════════════════════════════════════════════════════════════════════

	// CardiacIndex is cardiac index in L/min/m².
	// Source: Swan-Ganz, PICCO, echocardiography
	// Normal: 2.5-4.0, <2.0 = cardiogenic shock
	CardiacIndex float64 `json:"cardiac_index"`

	// ScvO2 is central venous oxygen saturation (%).
	// Source: Central line blood gas
	// <60% indicates inadequate oxygen delivery
	ScvO2 float64 `json:"scvo2"`

	// OnInotropeEscalation indicates inotrope dose is increasing.
	// Source: Medication pump trends, pharmacy
	OnInotropeEscalation bool `json:"on_inotrope_escalation"`

	// HasAKI indicates acute kidney injury (KDIGO stage ≥2).
	// Source: Creatinine trend, urine output, KDIGO criteria
	HasAKI bool `json:"has_aki"`

	// HasALF indicates acute liver failure.
	// Source: LFTs, INR, encephalopathy assessment
	HasALF bool `json:"has_alf"`

	// ═══════════════════════════════════════════════════════════════════════════
	// METADATA
	// ═══════════════════════════════════════════════════════════════════════════

	// PatientID is the FHIR Patient resource ID.
	PatientID string `json:"patient_id"`

	// EncounterID is the current FHIR Encounter resource ID.
	EncounterID string `json:"encounter_id"`

	// Timestamp is when these facts were gathered.
	Timestamp time.Time `json:"timestamp"`

	// SourceSystem identifies the originating system.
	SourceSystem string `json:"source_system"`
}

// NewSafetyFacts creates a SafetyFacts with safe defaults.
// All numeric values default to "normal" ranges that won't trigger dominance.
func NewSafetyFacts(patientID, encounterID string) *SafetyFacts {
	return &SafetyFacts{
		PatientID:   patientID,
		EncounterID: encounterID,
		Timestamp:   time.Now(),

		// Default to non-ICU context
		IsInICU:              false,
		IsCodeActive:         false,
		IsCriticallyUnstable: false,

		// Neurologic - normal values
		GCS:                15,    // Alert and oriented
		HasActiveSeizure:   false,
		ICP:                10,    // Normal ICP
		HasHerniationSigns: false,

		// Hemodynamic - normal values
		MAP:            80,    // Normal MAP
		Lactate:        1.0,   // Normal lactate
		OnVasopressors: false,
		HasSepticShock: false,

		// Respiratory - normal values
		SpO2:             98,    // Normal saturation
		PFRatio:          400,   // Normal P/F ratio
		FiO2:             0.21,  // Room air
		OnMechanicalVent: false,

		// Bleeding - normal values
		HgbDrop6h:          0,
		CurrentHgb:         14,   // Normal hemoglobin
		HasActiveTransfusion: false,
		HasSurgicalBleeding: false,
		INR:                1.0,  // Normal INR
		HasActiveBleeding:  false,

		// Cardiac - normal values
		CardiacIndex:        3.0,  // Normal CI
		ScvO2:               70,   // Normal ScvO2
		OnInotropeEscalation: false,
		HasAKI:              false,
		HasALF:              false,
	}
}

// IsICUContext returns true if dominance evaluation should be performed.
// Per CTO/CMO directive: Only assert dominance in ICU/Code contexts.
func (f *SafetyFacts) IsICUContext() bool {
	return f.IsInICU || f.IsCodeActive || f.IsCriticallyUnstable
}

// Validate checks if required fields are populated.
func (f *SafetyFacts) Validate() error {
	if f.PatientID == "" {
		return ErrMissingPatientID
	}
	if f.EncounterID == "" {
		return ErrMissingEncounterID
	}
	if f.GCS < 3 || f.GCS > 15 {
		return ErrInvalidGCS
	}
	return nil
}

// Common validation errors
var (
	ErrMissingPatientID   = &SafetyFactsError{Field: "PatientID", Message: "patient ID is required"}
	ErrMissingEncounterID = &SafetyFactsError{Field: "EncounterID", Message: "encounter ID is required"}
	ErrInvalidGCS         = &SafetyFactsError{Field: "GCS", Message: "GCS must be between 3 and 15"}
)

// SafetyFactsError represents a validation error in SafetyFacts.
type SafetyFactsError struct {
	Field   string
	Message string
}

func (e *SafetyFactsError) Error() string {
	return "SafetyFacts validation error: " + e.Field + " - " + e.Message
}
