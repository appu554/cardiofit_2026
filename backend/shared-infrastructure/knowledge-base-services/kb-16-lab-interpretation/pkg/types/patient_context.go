// Package types provides shared type definitions for KB-16 Lab Interpretation
// Phase 3b.6: Enhanced PatientContext for conditional reference range selection
package types

import (
	"time"
)

// EnhancedPatientContext contains all patient information needed for context-aware
// lab interpretation and conditional reference range selection.
// Note: This is an enhanced version with additional clinical fields for Phase 3b.6.
type EnhancedPatientContext struct {
	// Core Patient Identification
	PatientID   string `json:"patient_id"`
	EncounterID string `json:"encounter_id,omitempty"`

	// Demographics
	Age         float64   `json:"age"`      // Age in years (can be fractional for infants)
	AgeInDays   int       `json:"age_days"` // Age in days (precise for neonates)
	Sex         string    `json:"sex"`      // M, F
	DateOfBirth time.Time `json:"dob,omitempty"`

	// Pregnancy Status (ACOG, ATA guidelines)
	IsPregnant      bool       `json:"is_pregnant"`
	Trimester       int        `json:"trimester,omitempty"`        // 1, 2, 3
	GestationalWeek int        `json:"gestational_week,omitempty"` // Current gestational age
	EstimatedDueDate *time.Time `json:"edd,omitempty"`
	IsPostpartum    bool       `json:"is_postpartum"`
	PostpartumWeeks int        `json:"postpartum_weeks,omitempty"`
	IsLactating     bool       `json:"is_lactating"`

	// Neonatal Parameters (AAP 2022 bilirubin guidelines)
	IsNeonate               bool   `json:"is_neonate"`                   // < 28 days of life
	GestationalAgeAtBirth   int    `json:"gestational_age_at_birth"`     // Weeks at birth
	HoursOfLife             int    `json:"hours_of_life,omitempty"`      // Hours since birth
	BirthWeight             int    `json:"birth_weight,omitempty"`       // Grams
	NeonatalRiskCategory    string `json:"neonatal_risk_category,omitempty"` // LOW, MEDIUM, HIGH

	// Renal Status (KDIGO guidelines)
	CKDStage       int     `json:"ckd_stage,omitempty"`       // 1-5
	EGFR           float64 `json:"egfr,omitempty"`            // mL/min/1.73m²
	IsOnDialysis   bool    `json:"is_on_dialysis"`
	DialysisType   string  `json:"dialysis_type,omitempty"`   // HD, PD
	LastDialysis   *time.Time `json:"last_dialysis,omitempty"`

	// Hepatic Status
	ChildPughScore int    `json:"child_pugh_score,omitempty"` // 5-15
	ChildPughClass string `json:"child_pugh_class,omitempty"` // A, B, C

	// Active Conditions (ICD-10 or SNOMED codes)
	Conditions []string `json:"conditions,omitempty"`

	// Active Medications (RxNorm codes)
	Medications []string `json:"medications,omitempty"`

	// Additional Clinical Context
	IsICU         bool   `json:"is_icu"`
	IsFasting     bool   `json:"is_fasting"`
	IsAmbulatory  bool   `json:"is_ambulatory"`
	RaceEthnicity string `json:"race_ethnicity,omitempty"` // For eGFR calculation

	// Baseline Values (for delta checks)
	BaselineCreatinine *float64 `json:"baseline_cr,omitempty"`
	BaselineHemoglobin *float64 `json:"baseline_hgb,omitempty"`

	// Context Metadata
	ContextSource   string    `json:"context_source,omitempty"`   // EMR, KB-2, Manual
	ContextDate     time.Time `json:"context_date,omitempty"`
	ContextProvider string    `json:"context_provider,omitempty"` // Who provided context
}

// NewEnhancedPatientContext creates an EnhancedPatientContext with sensible defaults
func NewEnhancedPatientContext(patientID string, age float64, sex string) *EnhancedPatientContext {
	ctx := &EnhancedPatientContext{
		PatientID:  patientID,
		Age:        age,
		Sex:        sex,
		IsNeonate:  age < 0.077, // < 28 days
		AgeInDays:  int(age * 365.25),
		ContextDate: time.Now(),
	}

	return ctx
}

// NewPatientContext is an alias for NewEnhancedPatientContext for backward compatibility
func NewPatientContext(patientID string, age float64, sex string) *EnhancedPatientContext {
	return NewEnhancedPatientContext(patientID, age, sex)
}

// SetPregnancy updates pregnancy status with trimester calculation
func (p *EnhancedPatientContext) SetPregnancy(isPregnant bool, gestationalWeek int) {
	p.IsPregnant = isPregnant
	p.GestationalWeek = gestationalWeek

	if isPregnant && gestationalWeek > 0 {
		switch {
		case gestationalWeek <= 13:
			p.Trimester = 1
		case gestationalWeek <= 27:
			p.Trimester = 2
		default:
			p.Trimester = 3
		}
	}
}

// SetEGFR updates eGFR and auto-calculates CKD stage
func (p *EnhancedPatientContext) SetEGFR(egfr float64) {
	p.EGFR = egfr

	// KDIGO CKD staging based on eGFR
	switch {
	case egfr >= 90:
		p.CKDStage = 1 // G1: Normal or high
	case egfr >= 60:
		p.CKDStage = 2 // G2: Mildly decreased
	case egfr >= 45:
		p.CKDStage = 3 // G3a: Mildly to moderately decreased
	case egfr >= 30:
		p.CKDStage = 3 // G3b: Moderately to severely decreased
	case egfr >= 15:
		p.CKDStage = 4 // G4: Severely decreased
	default:
		p.CKDStage = 5 // G5: Kidney failure
	}
}

// SetNeonatalStatus updates neonatal parameters for bilirubin interpretation
func (p *EnhancedPatientContext) SetNeonatalStatus(gestationalAgeAtBirth, hoursOfLife int) {
	p.IsNeonate = true
	p.GestationalAgeAtBirth = gestationalAgeAtBirth
	p.HoursOfLife = hoursOfLife
	p.AgeInDays = hoursOfLife / 24

	// Determine risk category for bilirubin nomogram
	switch {
	case gestationalAgeAtBirth >= 38:
		p.NeonatalRiskCategory = "LOW"
	case gestationalAgeAtBirth >= 35:
		p.NeonatalRiskCategory = "MEDIUM"
	default:
		p.NeonatalRiskCategory = "HIGH"
	}
}

// IsPregnancyActive returns true if patient is currently pregnant
func (p *EnhancedPatientContext) IsPregnancyActive() bool {
	return p.IsPregnant && p.Trimester > 0
}

// IsCKDPatient returns true if patient has CKD Stage 3+
func (p *EnhancedPatientContext) IsCKDPatient() bool {
	return p.CKDStage >= 3
}

// IsDialysisPatient returns true if patient is on dialysis
func (p *EnhancedPatientContext) IsDialysisPatient() bool {
	return p.IsOnDialysis
}

// IsNeonateWithinNomogramAge returns true if neonate is within bilirubin nomogram range (0-120h)
func (p *EnhancedPatientContext) IsNeonateWithinNomogramAge() bool {
	return p.IsNeonate && p.HoursOfLife > 0 && p.HoursOfLife <= 120
}

// GetAgeCategory returns a human-readable age category
func (p *EnhancedPatientContext) GetAgeCategory() string {
	switch {
	case p.IsNeonate:
		return "Neonate"
	case p.Age < 2:
		return "Infant"
	case p.Age < 12:
		return "Pediatric"
	case p.Age < 18:
		return "Adolescent"
	case p.Age < 65:
		return "Adult"
	default:
		return "Geriatric"
	}
}

// ContextSummary returns a human-readable summary of patient context
func (p *EnhancedPatientContext) ContextSummary() string {
	summary := p.GetAgeCategory()

	if p.Sex == "M" {
		summary += " Male"
	} else if p.Sex == "F" {
		summary += " Female"
	}

	if p.IsPregnancyActive() {
		summary += ", Pregnant T" + string(rune('0'+p.Trimester))
	}

	if p.IsOnDialysis {
		summary += ", Dialysis"
	} else if p.CKDStage >= 3 {
		summary += ", CKD Stage " + string(rune('0'+p.CKDStage))
	}

	if p.IsNeonate && p.HoursOfLife > 0 {
		summary += ", " + string(rune('0'+(p.HoursOfLife/10))) + string(rune('0'+(p.HoursOfLife%10))) + "h of life"
	}

	return summary
}

// HasRiskFactor returns true if patient has any high-risk conditions
func (p *EnhancedPatientContext) HasRiskFactor(factors ...string) bool {
	factorSet := make(map[string]bool)
	for _, f := range factors {
		factorSet[f] = true
	}

	for _, cond := range p.Conditions {
		if factorSet[cond] {
			return true
		}
	}

	// Check implied risk factors
	if p.IsPregnant && factorSet["pregnancy"] {
		return true
	}
	if p.CKDStage >= 4 && factorSet["ckd_severe"] {
		return true
	}
	if p.IsOnDialysis && factorSet["dialysis"] {
		return true
	}

	return false
}

// Clone creates a deep copy of the EnhancedPatientContext
func (p *EnhancedPatientContext) Clone() *EnhancedPatientContext {
	clone := *p

	// Deep copy slices
	if p.Conditions != nil {
		clone.Conditions = make([]string, len(p.Conditions))
		copy(clone.Conditions, p.Conditions)
	}
	if p.Medications != nil {
		clone.Medications = make([]string, len(p.Medications))
		copy(clone.Medications, p.Medications)
	}

	return &clone
}
