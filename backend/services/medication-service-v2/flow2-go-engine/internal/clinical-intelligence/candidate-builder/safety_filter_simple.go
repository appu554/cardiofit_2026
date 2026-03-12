package candidatebuilder

import (
	"fmt"
	"log"
	"time"
)

// SafetyFilter implements patient-specific safety filtering - Step 2 of the filtering funnel
// This is the MOST CRITICAL safety gate that removes absolutely contraindicated drugs
type SafetyFilter struct {
	logger *log.Logger
}

// NewSafetyFilter creates a new safety filter
func NewSafetyFilter(logger *log.Logger) *SafetyFilter {
	return &SafetyFilter{
		logger: logger,
	}
}

// FilterByPatientContraindications removes drugs contraindicated for this specific patient
// Enhanced with additional safety checks for pregnancy, renal/hepatic impairment, and black box warnings
func (sf *SafetyFilter) FilterByPatientContraindications(
	candidatePool []Drug,
	patientFlags map[string]bool,
) ([]Drug, error) {

	startTime := time.Now()

	sf.logger.Printf("Starting enhanced patient-specific safety filtering: candidates=%d, patient_flags=%d",
		len(candidatePool), len(patientFlags))

	var safetyVettedPool []Drug
	var exclusionLog []ExclusionRecord
	excludedCount := 0

	for _, drug := range candidatePool {
		isContraindicated := false
		contraindicationReason := ""

		// Check legacy contraindications for backward compatibility
		for _, contraindication := range drug.Contraindications {
			// Check if patient has this contraindication flag set to true
			if patientFlag, exists := patientFlags[contraindication]; exists && patientFlag {
				isContraindicated = true
				contraindicationReason = contraindication

				// Create detailed exclusion record for audit trail
				exclusionRecord := ExclusionRecord{
					DrugName:        drug.Name,
					DrugCode:        drug.Code,
					ExclusionReason: contraindication,
					FilterStage:     "patient_contraindications",
					PatientFlag:     contraindication,
					Timestamp:       time.Now(),
					ClinicalReason:  sf.generateClinicalReason(drug, contraindication),
				}
				exclusionLog = append(exclusionLog, exclusionRecord)

				sf.logger.Printf("SAFETY FILTER EXCLUDED: %s (code: %s) due to patient contraindication: %s (flag: %v)",
					drug.Name, drug.Code, contraindication, patientFlag)
				break // No need to check other contraindications for this drug
			}
		}

		// Enhanced safety checks
		if !isContraindicated {
			isContraindicated, contraindicationReason = sf.checkEnhancedSafetyFlags(drug, patientFlags)
			if isContraindicated {
				exclusionRecord := ExclusionRecord{
					DrugName:        drug.Name,
					DrugCode:        drug.Code,
					ExclusionReason: contraindicationReason,
					FilterStage:     "enhanced_safety_checks",
					PatientFlag:     contraindicationReason,
					Timestamp:       time.Now(),
					ClinicalReason:  sf.generateClinicalReason(drug, contraindicationReason),
				}
				exclusionLog = append(exclusionLog, exclusionRecord)

				sf.logger.Printf("ENHANCED SAFETY FILTER EXCLUDED: %s (code: %s) due to: %s",
					drug.Name, drug.Code, contraindicationReason)
			}
		}

		if !isContraindicated {
			safetyVettedPool = append(safetyVettedPool, drug)
			sf.logger.Printf("SAFETY FILTER INCLUDED: %s (code: %s) - passed all safety checks",
				drug.Name, drug.Code)
		} else {
			excludedCount++
		}
	}

	processingTime := time.Since(startTime)
	safetyPassRate := sf.calculatePassRate(len(candidatePool), len(safetyVettedPool))

	sf.logger.Printf("Patient safety filtering completed: initial=%d, vetted=%d, excluded=%d, pass_rate=%.1f%%, time=%dms", 
		len(candidatePool), len(safetyVettedPool), excludedCount, safetyPassRate, processingTime.Milliseconds())

	// Clinical safety validation
	if len(safetyVettedPool) == 0 {
		sf.logger.Printf("WARNING: Safety filtering resulted in zero candidates - all %d drugs contraindicated for patient", 
			len(candidatePool))
		
		return safetyVettedPool, &FilterError{
			Stage:   "patient_contraindications",
			Message: fmt.Sprintf("all %d candidate drugs contraindicated for patient", len(candidatePool)),
		}
	}

	// Log safety filtering effectiveness
	if safetyPassRate < 50 {
		sf.logger.Printf("WARNING: Low safety pass rate (%.1f%%) - many drugs contraindicated for this patient", 
			safetyPassRate)
	}

	return safetyVettedPool, nil
}

// generateClinicalReason generates human-readable clinical reasoning for exclusions
func (sf *SafetyFilter) generateClinicalReason(drug Drug, contraindication string) string {
	clinicalReasons := map[string]string{
		"ANGIOEDEMA_HISTORY":     fmt.Sprintf("%s contraindicated due to angioedema history - risk of life-threatening airway swelling", drug.Name),
		"PREGNANCY":              fmt.Sprintf("%s contraindicated in pregnancy - potential teratogenic effects", drug.Name),
		"SEVERE_KIDNEY_DISEASE":  fmt.Sprintf("%s contraindicated with severe kidney disease - risk of drug accumulation", drug.Name),
		"SEVERE_LIVER_DISEASE":   fmt.Sprintf("%s contraindicated with severe liver disease - impaired drug metabolism", drug.Name),
		"HEART_BLOCK":            fmt.Sprintf("%s contraindicated with heart block - risk of cardiac conduction issues", drug.Name),
		"ASTHMA":                 fmt.Sprintf("%s contraindicated with asthma - risk of bronchospasm", drug.Name),
		"ALLERGY_HISTORY":        fmt.Sprintf("%s contraindicated due to known allergy - risk of allergic reaction", drug.Name),
	}

	if reason, exists := clinicalReasons[contraindication]; exists {
		return reason
	}

	return fmt.Sprintf("%s contraindicated due to patient condition: %s", drug.Name, contraindication)
}

// calculatePassRate calculates the percentage of drugs that passed filtering
func (sf *SafetyFilter) calculatePassRate(initial, passed int) float64 {
	if initial == 0 {
		return 0
	}
	return float64(passed) / float64(initial) * 100
}

// getActiveFlagsCount counts how many patient flags are set to true
func (sf *SafetyFilter) getActiveFlagsCount(flags map[string]bool) int {
	count := 0
	for _, value := range flags {
		if value {
			count++
		}
	}
	return count
}

// GetSupportedContraindications returns list of supported contraindication flags
func (sf *SafetyFilter) GetSupportedContraindications() []string {
	return []string{
		"ANGIOEDEMA_HISTORY",
		"PREGNANCY",
		"BREASTFEEDING",
		"SEVERE_KIDNEY_DISEASE",
		"SEVERE_LIVER_DISEASE",
		"HEART_BLOCK",
		"ASTHMA",
		"COPD",
		"ALLERGY_HISTORY",
		"BLEEDING_DISORDER",
		"THROMBOCYTOPENIA",
		"HYPERKALEMIA",
		"HYPONATREMIA",
		"GOUT",
		"DIABETES_TYPE_1",
		"HEART_FAILURE",
		"MYOCARDIAL_INFARCTION_RECENT",
		"STROKE_RECENT",
		"SURGERY_RECENT",
		"ELDERLY_FRAIL",
		"PEDIATRIC",
	}
}

// ValidatePatientFlags validates patient safety flags structure
func (sf *SafetyFilter) ValidatePatientFlags(flags map[string]bool) error {
	if flags == nil {
		return fmt.Errorf("patient flags cannot be nil")
	}

	supportedFlags := sf.GetSupportedContraindications()
	unknownFlags := []string{}

	for flag := range flags {
		if !sf.contains(supportedFlags, flag) {
			unknownFlags = append(unknownFlags, flag)
		}
	}

	if len(unknownFlags) > 0 {
		sf.logger.Printf("WARNING: Unknown patient safety flags detected: %v", unknownFlags)
	}

	return nil
}

// checkEnhancedSafetyFlags performs additional safety checks beyond basic contraindications
func (sf *SafetyFilter) checkEnhancedSafetyFlags(drug Drug, patientFlags map[string]bool) (bool, string) {
	// Check pregnancy category for pregnant patients (if field exists)
	if patientFlags["is_pregnant"] && (drug.PregnancyCategory == "X" || drug.PregnancyCategory == "D") {
		return true, fmt.Sprintf("pregnancy_category_%s", drug.PregnancyCategory)
	}

	// Check renal adjustment for patients with kidney disease (if field exists)
	if patientFlags["has_kidney_disease"] && drug.RenalAdjustment {
		return true, "renal_adjustment_required"
	}

	// Check hepatic adjustment for patients with liver disease (if field exists)
	if patientFlags["has_liver_disease"] && drug.HepaticAdjustment {
		return true, "hepatic_adjustment_required"
	}

	// Check black box warning for high-risk patients (if field exists)
	if patientFlags["high_risk_patient"] && drug.BlackBoxWarning {
		return true, "black_box_warning"
	}

	// Check enhanced contraindication codes (if field exists)
	for _, code := range drug.ContraindicationCodes {
		if patientFlags[code] {
			return true, code
		}
	}

	// Check allergy codes (if field exists)
	for _, code := range drug.AllergyCodes {
		if patientFlags[code] {
			return true, fmt.Sprintf("allergy_%s", code)
		}
	}

	return false, ""
}

// contains checks if a slice contains a specific string
func (sf *SafetyFilter) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
