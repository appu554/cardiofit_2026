// Package transaction provides transaction management and conflict detection for KB-19.
// ConflictDetector MOVED FROM: medication-advisor-engine/advisor/conflicts.go
// as part of V3 architecture refactoring.
package transaction

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CONFLICT TYPES
// MOVED FROM: medication-advisor-engine/snapshot/manager.go lines 459-485
// =============================================================================

// Conflict represents a conflict between snapshot and current data
type Conflict struct {
	Type        ConflictType     `json:"type"`
	Severity    ConflictSeverity `json:"severity"`
	Field       string           `json:"field"`
	OldValue    interface{}      `json:"old_value,omitempty"`
	NewValue    interface{}      `json:"new_value,omitempty"`
	Description string           `json:"description"`
}

// ConflictType represents the type of conflict
type ConflictType string

const (
	ConflictTypeLab         ConflictType = "lab"
	ConflictTypeCondition   ConflictType = "condition"
	ConflictTypeAllergy     ConflictType = "allergy"
	ConflictTypeMedication  ConflictType = "medication"
	ConflictTypeDemographic ConflictType = "demographic"
	ConflictTypeVitals      ConflictType = "vitals"
)

// ConflictSeverity represents conflict severity (HARD = abort, SOFT = warn)
type ConflictSeverity string

const (
	ConflictSeverityHard ConflictSeverity = "hard" // Requires re-calculation
	ConflictSeveritySoft ConflictSeverity = "soft" // Warning only
)

// =============================================================================
// CLINICAL SNAPSHOT DATA TYPES (subset needed for conflict detection)
// MOVED FROM: medication-advisor-engine/snapshot/types.go
// =============================================================================

// ClinicalSnapshotData contains the actual clinical data captured at snapshot time
type ClinicalSnapshotData struct {
	Demographics PatientDemographics `json:"demographics"`
	LabResults   []LabResult         `json:"lab_results,omitempty"`
	Medications  []MedicationEntry   `json:"medications,omitempty"`
	Allergies    []AllergyEntry      `json:"allergies,omitempty"`
	Conditions   []ConditionEntry    `json:"conditions,omitempty"`
}

// PatientDemographics contains basic patient demographic information
type PatientDemographics struct {
	PatientID         uuid.UUID `json:"patient_id"`
	MRN               string    `json:"mrn"`
	DateOfBirth       time.Time `json:"date_of_birth"`
	Gender            string    `json:"gender"`
	WeightKg          *float64  `json:"weight_kg,omitempty"`
	HeightCm          *float64  `json:"height_cm,omitempty"`
	BMI               *float64  `json:"bmi,omitempty"`
	BSAm2             *float64  `json:"bsa_m2,omitempty"`
	Race              string    `json:"race,omitempty"`
	Ethnicity         string    `json:"ethnicity,omitempty"`
	PreferredLanguage string    `json:"preferred_language,omitempty"`
	SnapshotTime      time.Time `json:"snapshot_time"`
}

// LabResult represents a laboratory test result
type LabResult struct {
	ID             uuid.UUID       `json:"id"`
	TestName       string          `json:"test_name"`
	TestCode       string          `json:"test_code"`
	Value          interface{}     `json:"value"`
	Unit           string          `json:"unit"`
	ReferenceRange string          `json:"reference_range"`
	AbnormalFlag   string          `json:"abnormal_flag,omitempty"`
	Status         LabResultStatus `json:"status"`
	CollectedAt    time.Time       `json:"collected_at"`
	ReportedAt     time.Time       `json:"reported_at"`
	PerformingLab  string          `json:"performing_lab"`
	CriticalValue  bool            `json:"critical_value"`
	Comments       string          `json:"comments,omitempty"`
}

// LabResultStatus represents the status of a lab result
type LabResultStatus string

const (
	LabStatusFinal       LabResultStatus = "final"
	LabStatusPreliminary LabResultStatus = "preliminary"
	LabStatusCorrected   LabResultStatus = "corrected"
	LabStatusCancelled   LabResultStatus = "cancelled"
)

// MedicationEntry represents a medication in the snapshot
type MedicationEntry struct {
	ID             uuid.UUID        `json:"id"`
	MedicationName string           `json:"medication_name"`
	GenericName    string           `json:"generic_name"`
	RxNormCode     string           `json:"rxnorm_code,omitempty"`
	DoseMg         float64          `json:"dose_mg"`
	Unit           string           `json:"unit"`
	Route          string           `json:"route"`
	Frequency      string           `json:"frequency"`
	StartDate      time.Time        `json:"start_date"`
	EndDate        *time.Time       `json:"end_date,omitempty"`
	Status         MedicationStatus `json:"status"`
	Indication     string           `json:"indication"`
	PrescribedBy   string           `json:"prescribed_by"`
	Instructions   string           `json:"instructions,omitempty"`
	LastDoseTime   *time.Time       `json:"last_dose_time,omitempty"`
	AdherenceScore *float64         `json:"adherence_score,omitempty"`
}

// MedicationStatus represents the status of a medication
type MedicationStatus string

const (
	MedStatusActive       MedicationStatus = "active"
	MedStatusCompleted    MedicationStatus = "completed"
	MedStatusDiscontinued MedicationStatus = "discontinued"
	MedStatusHeld         MedicationStatus = "held"
)

// AllergyEntry represents an allergy in the snapshot
type AllergyEntry struct {
	ID           uuid.UUID       `json:"id"`
	Allergen     string          `json:"allergen"`
	AllergenType AllergenType    `json:"allergen_type"`
	Reaction     string          `json:"reaction"`
	Severity     AllergySeverity `json:"severity"`
	OnsetDate    *time.Time      `json:"onset_date,omitempty"`
	Status       AllergyStatus   `json:"status"`
	Notes        string          `json:"notes,omitempty"`
	ReportedBy   string          `json:"reported_by"`
	VerifiedBy   string          `json:"verified_by,omitempty"`
}

// AllergenType represents different types of allergens
type AllergenType string

const (
	AllergenDrug          AllergenType = "drug"
	AllergenFood          AllergenType = "food"
	AllergenEnvironmental AllergenType = "environmental"
	AllergenOther         AllergenType = "other"
)

// AllergySeverity represents allergy severity levels
type AllergySeverity string

const (
	AllergySeverityMild            AllergySeverity = "mild"
	AllergySeverityModerate        AllergySeverity = "moderate"
	AllergySeveritySevere          AllergySeverity = "severe"
	AllergySeverityLifeThreatening AllergySeverity = "life_threatening"
)

// AllergyStatus represents allergy status
type AllergyStatus string

const (
	AllergyStatusActive   AllergyStatus = "active"
	AllergyStatusInactive AllergyStatus = "inactive"
	AllergyStatusResolved AllergyStatus = "resolved"
)

// ConditionEntry represents a medical condition in the snapshot
type ConditionEntry struct {
	ID            uuid.UUID         `json:"id"`
	ConditionName string            `json:"condition_name"`
	ICD10Code     string            `json:"icd10_code,omitempty"`
	SNOMEDCT      string            `json:"snomed_ct,omitempty"`
	Status        ConditionStatus   `json:"status"`
	Severity      ConditionSeverity `json:"severity,omitempty"`
	OnsetDate     *time.Time        `json:"onset_date,omitempty"`
	DiagnosedDate *time.Time        `json:"diagnosed_date,omitempty"`
	ResolvedDate  *time.Time        `json:"resolved_date,omitempty"`
	DiagnosedBy   string            `json:"diagnosed_by"`
	Notes         string            `json:"notes,omitempty"`
	Stage         string            `json:"stage,omitempty"`
	Grade         string            `json:"grade,omitempty"`
}

// ConditionStatus represents medical condition status
type ConditionStatus string

const (
	ConditionStatusActive     ConditionStatus = "active"
	ConditionStatusRecurrence ConditionStatus = "recurrence"
	ConditionStatusRelapse    ConditionStatus = "relapse"
	ConditionStatusInactive   ConditionStatus = "inactive"
	ConditionStatusRemission  ConditionStatus = "remission"
	ConditionStatusResolved   ConditionStatus = "resolved"
)

// ConditionSeverity represents medical condition severity
type ConditionSeverity string

const (
	ConditionSeverityMild     ConditionSeverity = "mild"
	ConditionSeverityModerate ConditionSeverity = "moderate"
	ConditionSeveritySevere   ConditionSeverity = "severe"
)

// =============================================================================
// CONFLICT DETECTOR
// MOVED FROM: medication-advisor-engine/advisor/conflicts.go lines 1-347
// =============================================================================

// ConflictDetector detects and classifies conflicts between snapshot and current data
type ConflictDetector struct {
	// Thresholds for critical values
	criticalLabThresholds map[string]LabThreshold
}

// LabThreshold defines critical thresholds for a lab value
type LabThreshold struct {
	LowCritical  float64
	HighCritical float64
	Unit         string
}

// NewConflictDetector creates a new conflict detector with default thresholds
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{
		criticalLabThresholds: map[string]LabThreshold{
			"potassium":   {LowCritical: 2.5, HighCritical: 6.5, Unit: "mEq/L"},
			"sodium":      {LowCritical: 120, HighCritical: 160, Unit: "mEq/L"},
			"creatinine":  {LowCritical: 0.0, HighCritical: 10.0, Unit: "mg/dL"},
			"glucose":     {LowCritical: 40, HighCritical: 500, Unit: "mg/dL"},
			"hemoglobin":  {LowCritical: 7.0, HighCritical: 20.0, Unit: "g/dL"},
			"platelets":   {LowCritical: 50000, HighCritical: 1000000, Unit: "/uL"},
			"inr":         {LowCritical: 0.0, HighCritical: 5.0, Unit: "ratio"},
		},
	}
}

// ConflictClassification represents the classification result
type ConflictClassification struct {
	HasHardConflicts      bool       `json:"has_hard_conflicts"`
	HasSoftConflicts      bool       `json:"has_soft_conflicts"`
	HardConflicts         []Conflict `json:"hard_conflicts"`
	SoftConflicts         []Conflict `json:"soft_conflicts"`
	Recommendation        string     `json:"recommendation"` // proceed, warn, abort
	RequiresRecalculation bool       `json:"requires_recalculation"`
	AnalyzedAt            time.Time  `json:"analyzed_at"`
}

// ClassifyConflicts classifies detected changes into hard and soft conflicts
func (cd *ConflictDetector) ClassifyConflicts(
	snapshotData ClinicalSnapshotData,
	currentData ClinicalSnapshotData,
) *ConflictClassification {

	result := &ConflictClassification{
		HardConflicts: []Conflict{},
		SoftConflicts: []Conflict{},
		AnalyzedAt:    time.Now(),
	}

	// Detect lab changes
	cd.detectLabConflicts(snapshotData.LabResults, currentData.LabResults, result)

	// Detect condition changes
	cd.detectConditionConflicts(snapshotData.Conditions, currentData.Conditions, result)

	// Detect allergy changes
	cd.detectAllergyConflicts(snapshotData.Allergies, currentData.Allergies, result)

	// Detect demographic changes
	cd.detectDemographicConflicts(snapshotData.Demographics, currentData.Demographics, result)

	// Detect medication changes
	cd.detectMedicationConflicts(snapshotData.Medications, currentData.Medications, result)

	// Set flags and recommendation
	result.HasHardConflicts = len(result.HardConflicts) > 0
	result.HasSoftConflicts = len(result.SoftConflicts) > 0
	result.RequiresRecalculation = result.HasHardConflicts

	if result.HasHardConflicts {
		result.Recommendation = "abort"
	} else if result.HasSoftConflicts {
		result.Recommendation = "warn"
	} else {
		result.Recommendation = "proceed"
	}

	return result
}

// detectLabConflicts detects HARD conflicts from lab value changes
func (cd *ConflictDetector) detectLabConflicts(
	snapshotLabs []LabResult,
	currentLabs []LabResult,
	result *ConflictClassification,
) {
	// Create map of snapshot labs
	snapshotLabMap := make(map[string]LabResult)
	for _, lab := range snapshotLabs {
		snapshotLabMap[lab.TestCode] = lab
	}

	for _, currentLab := range currentLabs {
		snapshotLab, exists := snapshotLabMap[currentLab.TestCode]

		// New critical lab = HARD conflict
		if currentLab.CriticalValue && (!exists || !snapshotLab.CriticalValue) {
			result.HardConflicts = append(result.HardConflicts, Conflict{
				Type:        ConflictTypeLab,
				Severity:    ConflictSeverityHard,
				Field:       currentLab.TestName,
				OldValue:    getLabValue(snapshotLab),
				NewValue:    currentLab.Value,
				Description: "Lab value became critical since snapshot",
			})
			continue
		}

		// Significant change in lab value = HARD conflict
		if exists {
			oldVal, oldOk := getNumericValue(snapshotLab.Value)
			newVal, newOk := getNumericValue(currentLab.Value)

			if oldOk && newOk {
				percentChange := abs((newVal - oldVal) / oldVal * 100)
				if percentChange > 50 { // >50% change is significant
					result.HardConflicts = append(result.HardConflicts, Conflict{
						Type:        ConflictTypeLab,
						Severity:    ConflictSeverityHard,
						Field:       currentLab.TestName,
						OldValue:    oldVal,
						NewValue:    newVal,
						Description: "Significant lab value change (>50%)",
					})
				}
			}
		}
	}
}

// detectConditionConflicts detects HARD conflicts from new conditions
func (cd *ConflictDetector) detectConditionConflicts(
	snapshotConditions []ConditionEntry,
	currentConditions []ConditionEntry,
	result *ConflictClassification,
) {
	// Create set of snapshot condition codes
	snapshotCodes := make(map[string]bool)
	for _, c := range snapshotConditions {
		snapshotCodes[c.SNOMEDCT] = true
		snapshotCodes[c.ICD10Code] = true
	}

	for _, current := range currentConditions {
		if current.Status != ConditionStatusActive {
			continue
		}

		isNew := true
		if current.SNOMEDCT != "" && snapshotCodes[current.SNOMEDCT] {
			isNew = false
		}
		if current.ICD10Code != "" && snapshotCodes[current.ICD10Code] {
			isNew = false
		}

		if isNew {
			// New active condition = HARD conflict
			result.HardConflicts = append(result.HardConflicts, Conflict{
				Type:        ConflictTypeCondition,
				Severity:    ConflictSeverityHard,
				Field:       "conditions",
				NewValue:    current.ConditionName,
				Description: "New condition diagnosed since snapshot",
			})
		}
	}
}

// detectAllergyConflicts detects HARD conflicts from new allergies
func (cd *ConflictDetector) detectAllergyConflicts(
	snapshotAllergies []AllergyEntry,
	currentAllergies []AllergyEntry,
	result *ConflictClassification,
) {
	// Create set of snapshot allergens
	snapshotAllergens := make(map[string]bool)
	for _, a := range snapshotAllergies {
		snapshotAllergens[a.Allergen] = true
	}

	for _, current := range currentAllergies {
		if current.Status != AllergyStatusActive {
			continue
		}

		if !snapshotAllergens[current.Allergen] {
			// New allergy = HARD conflict (especially for drug allergies)
			severity := ConflictSeverityHard
			if current.AllergenType != AllergenDrug {
				severity = ConflictSeveritySoft
			}

			result.HardConflicts = append(result.HardConflicts, Conflict{
				Type:        ConflictTypeAllergy,
				Severity:    severity,
				Field:       "allergies",
				NewValue:    current.Allergen,
				Description: "New allergy reported since snapshot",
			})
		}
	}
}

// detectDemographicConflicts detects SOFT conflicts from demographic changes
func (cd *ConflictDetector) detectDemographicConflicts(
	snapshotDemo PatientDemographics,
	currentDemo PatientDemographics,
	result *ConflictClassification,
) {
	// Weight change = SOFT conflict (may affect dosing)
	if snapshotDemo.WeightKg != nil && currentDemo.WeightKg != nil {
		weightDiff := abs(*currentDemo.WeightKg - *snapshotDemo.WeightKg)
		if weightDiff > 5 { // >5kg change
			result.SoftConflicts = append(result.SoftConflicts, Conflict{
				Type:        ConflictTypeDemographic,
				Severity:    ConflictSeveritySoft,
				Field:       "weight",
				OldValue:    *snapshotDemo.WeightKg,
				NewValue:    *currentDemo.WeightKg,
				Description: "Weight changed by >5kg (may affect dosing)",
			})
		}
	}

	// Height change = SOFT conflict (BSA calculation)
	if snapshotDemo.HeightCm != nil && currentDemo.HeightCm != nil {
		if *currentDemo.HeightCm != *snapshotDemo.HeightCm {
			result.SoftConflicts = append(result.SoftConflicts, Conflict{
				Type:        ConflictTypeDemographic,
				Severity:    ConflictSeveritySoft,
				Field:       "height",
				OldValue:    *snapshotDemo.HeightCm,
				NewValue:    *currentDemo.HeightCm,
				Description: "Height updated (affects BSA calculation)",
			})
		}
	}
}

// detectMedicationConflicts detects conflicts from medication changes
func (cd *ConflictDetector) detectMedicationConflicts(
	snapshotMeds []MedicationEntry,
	currentMeds []MedicationEntry,
	result *ConflictClassification,
) {
	// Create set of snapshot medication codes
	snapshotCodes := make(map[string]bool)
	for _, m := range snapshotMeds {
		snapshotCodes[m.RxNormCode] = true
	}

	for _, current := range currentMeds {
		if current.Status != MedStatusActive {
			continue
		}

		if !snapshotCodes[current.RxNormCode] {
			// New medication = SOFT conflict (potential interaction)
			result.SoftConflicts = append(result.SoftConflicts, Conflict{
				Type:        ConflictTypeMedication,
				Severity:    ConflictSeveritySoft,
				Field:       "medications",
				NewValue:    current.MedicationName,
				Description: "New medication added (check for interactions)",
			})
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func getLabValue(lab LabResult) interface{} {
	if lab.Value == nil {
		return nil
	}
	return lab.Value
}

func getNumericValue(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// =============================================================================
// CLASSIFICATION METHODS
// =============================================================================

// IsAbortRequired returns true if hard conflicts require aborting the workflow
func (cc *ConflictClassification) IsAbortRequired() bool {
	return cc.HasHardConflicts
}

// IsWarningRequired returns true if soft conflicts require warning the user
func (cc *ConflictClassification) IsWarningRequired() bool {
	return cc.HasSoftConflicts && !cc.HasHardConflicts
}

// CanProceed returns true if workflow can proceed without issues
func (cc *ConflictClassification) CanProceed() bool {
	return !cc.HasHardConflicts && !cc.HasSoftConflicts
}

// GetSummary returns a human-readable summary of conflicts
func (cc *ConflictClassification) GetSummary() string {
	if cc.CanProceed() {
		return "No conflicts detected. Safe to proceed."
	}

	summary := ""
	if cc.HasHardConflicts {
		summary = "ABORT REQUIRED: "
		for _, c := range cc.HardConflicts {
			summary += c.Description + "; "
		}
	} else if cc.HasSoftConflicts {
		summary = "WARNING: "
		for _, c := range cc.SoftConflicts {
			summary += c.Description + "; "
		}
	}

	return summary
}
