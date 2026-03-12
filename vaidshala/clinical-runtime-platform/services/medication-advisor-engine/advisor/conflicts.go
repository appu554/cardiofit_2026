package advisor

import (
	"time"

	"github.com/cardiofit/medication-advisor-engine/snapshot"
)

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
	HasHardConflicts  bool                `json:"has_hard_conflicts"`
	HasSoftConflicts  bool                `json:"has_soft_conflicts"`
	HardConflicts     []snapshot.Conflict `json:"hard_conflicts"`
	SoftConflicts     []snapshot.Conflict `json:"soft_conflicts"`
	Recommendation    string              `json:"recommendation"` // proceed, warn, abort
	RequiresRecalculation bool            `json:"requires_recalculation"`
	AnalyzedAt        time.Time           `json:"analyzed_at"`
}

// ClassifyConflicts classifies detected changes into hard and soft conflicts
func (cd *ConflictDetector) ClassifyConflicts(
	snapshotData snapshot.ClinicalSnapshotData,
	currentData snapshot.ClinicalSnapshotData,
) *ConflictClassification {

	result := &ConflictClassification{
		HardConflicts:  []snapshot.Conflict{},
		SoftConflicts:  []snapshot.Conflict{},
		AnalyzedAt:     time.Now(),
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
	snapshotLabs []snapshot.LabResult,
	currentLabs []snapshot.LabResult,
	result *ConflictClassification,
) {
	// Create map of snapshot labs
	snapshotLabMap := make(map[string]snapshot.LabResult)
	for _, lab := range snapshotLabs {
		snapshotLabMap[lab.TestCode] = lab
	}

	for _, currentLab := range currentLabs {
		snapshotLab, exists := snapshotLabMap[currentLab.TestCode]

		// New critical lab = HARD conflict
		if currentLab.CriticalValue && (!exists || !snapshotLab.CriticalValue) {
			result.HardConflicts = append(result.HardConflicts, snapshot.Conflict{
				Type:        snapshot.ConflictTypeLab,
				Severity:    snapshot.ConflictSeverityHard,
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
					result.HardConflicts = append(result.HardConflicts, snapshot.Conflict{
						Type:        snapshot.ConflictTypeLab,
						Severity:    snapshot.ConflictSeverityHard,
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
	snapshotConditions []snapshot.ConditionEntry,
	currentConditions []snapshot.ConditionEntry,
	result *ConflictClassification,
) {
	// Create set of snapshot condition codes
	snapshotCodes := make(map[string]bool)
	for _, c := range snapshotConditions {
		snapshotCodes[c.SNOMEDCT] = true
		snapshotCodes[c.ICD10Code] = true
	}

	for _, current := range currentConditions {
		if current.Status != snapshot.ConditionStatusActive {
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
			result.HardConflicts = append(result.HardConflicts, snapshot.Conflict{
				Type:        snapshot.ConflictTypeCondition,
				Severity:    snapshot.ConflictSeverityHard,
				Field:       "conditions",
				NewValue:    current.ConditionName,
				Description: "New condition diagnosed since snapshot",
			})
		}
	}
}

// detectAllergyConflicts detects HARD conflicts from new allergies
func (cd *ConflictDetector) detectAllergyConflicts(
	snapshotAllergies []snapshot.AllergyEntry,
	currentAllergies []snapshot.AllergyEntry,
	result *ConflictClassification,
) {
	// Create set of snapshot allergens
	snapshotAllergens := make(map[string]bool)
	for _, a := range snapshotAllergies {
		snapshotAllergens[a.Allergen] = true
	}

	for _, current := range currentAllergies {
		if current.Status != snapshot.AllergyStatusActive {
			continue
		}

		if !snapshotAllergens[current.Allergen] {
			// New allergy = HARD conflict (especially for drug allergies)
			severity := snapshot.ConflictSeverityHard
			if current.AllergenType != snapshot.AllergenDrug {
				severity = snapshot.ConflictSeveritySoft
			}

			result.HardConflicts = append(result.HardConflicts, snapshot.Conflict{
				Type:        snapshot.ConflictTypeAllergy,
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
	snapshotDemo snapshot.PatientDemographics,
	currentDemo snapshot.PatientDemographics,
	result *ConflictClassification,
) {
	// Weight change = SOFT conflict (may affect dosing)
	if snapshotDemo.WeightKg != nil && currentDemo.WeightKg != nil {
		weightDiff := abs(*currentDemo.WeightKg - *snapshotDemo.WeightKg)
		if weightDiff > 5 { // >5kg change
			result.SoftConflicts = append(result.SoftConflicts, snapshot.Conflict{
				Type:        snapshot.ConflictTypeDemographic,
				Severity:    snapshot.ConflictSeveritySoft,
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
			result.SoftConflicts = append(result.SoftConflicts, snapshot.Conflict{
				Type:        snapshot.ConflictTypeDemographic,
				Severity:    snapshot.ConflictSeveritySoft,
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
	snapshotMeds []snapshot.MedicationEntry,
	currentMeds []snapshot.MedicationEntry,
	result *ConflictClassification,
) {
	// Create set of snapshot medication codes
	snapshotCodes := make(map[string]bool)
	for _, m := range snapshotMeds {
		snapshotCodes[m.RxNormCode] = true
	}

	for _, current := range currentMeds {
		if current.Status != snapshot.MedStatusActive {
			continue
		}

		if !snapshotCodes[current.RxNormCode] {
			// New medication = SOFT conflict (potential interaction)
			result.SoftConflicts = append(result.SoftConflicts, snapshot.Conflict{
				Type:        snapshot.ConflictTypeMedication,
				Severity:    snapshot.ConflictSeveritySoft,
				Field:       "medications",
				NewValue:    current.MedicationName,
				Description: "New medication added (check for interactions)",
			})
		}
	}
}

// Helper functions

func getLabValue(lab snapshot.LabResult) interface{} {
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
