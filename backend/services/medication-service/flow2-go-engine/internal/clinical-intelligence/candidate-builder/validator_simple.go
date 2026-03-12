package candidatebuilder

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

// InputValidator implements comprehensive input validation for candidate builder
type InputValidator struct {
	logger *log.Logger
}

// NewInputValidator creates a new input validator
func NewInputValidator(logger *log.Logger) *InputValidator {
	return &InputValidator{
		logger: logger,
	}
}

// ValidateInputs performs comprehensive validation of all inputs before filtering
func (iv *InputValidator) ValidateInputs(input CandidateBuilderInput) error {
	iv.logger.Printf("Starting input validation: request_id=%s, patient_id=%s", 
		input.RequestID, input.PatientID)

	// Validate request metadata
	if err := iv.validateRequestMetadata(input); err != nil {
		return &ValidationError{Field: "request_metadata", Message: err.Error()}
	}

	// Validate patient flags
	if err := iv.ValidatePatientFlags(input.PatientFlags); err != nil {
		return &ValidationError{Field: "patient_flags", Message: err.Error()}
	}

	// Validate recommended drug classes
	if err := iv.validateRecommendedDrugClasses(input.RecommendedDrugClasses); err != nil {
		return &ValidationError{Field: "recommended_drug_classes", Message: err.Error()}
	}

	// Validate drug master list
	if err := iv.ValidateDrugMasterList(input.DrugMasterList); err != nil {
		return &ValidationError{Field: "drug_master_list", Message: err.Error()}
	}

	// Validate active medications
	if err := iv.ValidateActiveMedications(input.ActiveMedications); err != nil {
		return &ValidationError{Field: "active_medications", Message: err.Error()}
	}

	// Validate DDI rules
	if err := iv.ValidateDDIRules(input.DDIRules); err != nil {
		return &ValidationError{Field: "ddi_rules", Message: err.Error()}
	}

	iv.logger.Printf("Input validation completed successfully: patient_flags=%d, drugs=%d, active_meds=%d, ddi_rules=%d, classes=%d", 
		len(input.PatientFlags), len(input.DrugMasterList), len(input.ActiveMedications), 
		len(input.DDIRules), len(input.RecommendedDrugClasses))

	return nil
}

// validateRequestMetadata validates basic request information
func (iv *InputValidator) validateRequestMetadata(input CandidateBuilderInput) error {
	if input.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}

	if input.PatientID == "" {
		return fmt.Errorf("patient_id is required")
	}

	// Validate request ID format (should be UUID-like)
	if len(input.RequestID) < 10 {
		return fmt.Errorf("request_id appears to be invalid format: %s", input.RequestID)
	}

	// Validate patient ID format
	if len(input.PatientID) < 10 {
		return fmt.Errorf("patient_id appears to be invalid format: %s", input.PatientID)
	}

	return nil
}

// ValidatePatientFlags validates patient safety flags structure and content
func (iv *InputValidator) ValidatePatientFlags(flags map[string]bool) error {
	if flags == nil {
		return fmt.Errorf("patient_flags cannot be nil - safety filtering requires patient contraindication data")
	}

	if len(flags) == 0 {
		iv.logger.Printf("WARNING: Patient flags map is empty - this may indicate missing safety data")
	}

	// Validate flag structure and types
	invalidFlags := []string{}
	for flag, value := range flags {
		if flag == "" {
			invalidFlags = append(invalidFlags, "empty flag key detected")
			continue
		}

		// Ensure boolean values
		if reflect.TypeOf(value).Kind() != reflect.Bool {
			iv.logger.Printf("WARNING: Non-boolean patient flag detected: %s (type: %s)", 
				flag, reflect.TypeOf(value).String())
		}

		// Validate known safety flag patterns
		if !iv.isValidSafetyFlag(flag) {
			iv.logger.Printf("DEBUG: Unknown safety flag pattern: %s", flag)
		}
	}

	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid patient flags: %s", strings.Join(invalidFlags, ", "))
	}

	return nil
}

// validateRecommendedDrugClasses validates therapeutic class recommendations
func (iv *InputValidator) validateRecommendedDrugClasses(classes []string) error {
	// Note: Empty classes array is valid (means broad search)
	if classes == nil {
		return fmt.Errorf("recommended_drug_classes cannot be nil")
	}

	// Validate each class
	for i, class := range classes {
		if class == "" {
			return fmt.Errorf("empty drug class at index %d", i)
		}

		if !iv.isValidTherapeuticClass(class) {
			iv.logger.Printf("WARNING: Unknown therapeutic class detected: %s", class)
		}
	}

	return nil
}

// ValidateDrugMasterList validates the drug master list from kb_drug_master_v1
func (iv *InputValidator) ValidateDrugMasterList(drugs []Drug) error {
	if drugs == nil {
		return fmt.Errorf("drug_master_list cannot be nil - no drugs available for filtering")
	}

	if len(drugs) == 0 {
		return fmt.Errorf("drug_master_list is empty - no drugs available for filtering")
	}

	// Validate each drug entry
	for i, drug := range drugs {
		if err := iv.validateDrugEntry(drug, i); err != nil {
			return fmt.Errorf("invalid drug at index %d: %w", i, err)
		}
	}

	iv.logger.Printf("Drug master list validation completed: %d drugs validated", len(drugs))
	return nil
}

// validateDrugEntry validates a single drug entry
func (iv *InputValidator) validateDrugEntry(drug Drug, index int) error {
	if drug.Code == "" {
		return fmt.Errorf("drug code is required")
	}

	if drug.Name == "" {
		return fmt.Errorf("drug name is required")
	}

	if len(drug.TherapeuticClasses) == 0 {
		iv.logger.Printf("WARNING: Drug %s (code: %s) missing therapeutic classes - may affect class filtering",
			drug.Name, drug.Code)
	}

	// Validate contraindications array
	if drug.Contraindications == nil {
		iv.logger.Printf("DEBUG: Drug %s has no contraindications defined", drug.Name)
	}

	return nil
}

// ValidateActiveMedications validates patient's current medications
func (iv *InputValidator) ValidateActiveMedications(medications []ActiveMedication) error {
	if medications == nil {
		return fmt.Errorf("active_medications cannot be nil - DDI checking requires current medication data")
	}

	// Empty active medications is valid (patient not on any medications)
	if len(medications) == 0 {
		iv.logger.Printf("Patient has no active medications - DDI filtering will be skipped")
		return nil
	}

	// Validate each medication
	for i, med := range medications {
		if err := iv.validateActiveMedication(med, i); err != nil {
			return fmt.Errorf("invalid active medication at index %d: %w", i, err)
		}
	}

	return nil
}

// validateActiveMedication validates a single active medication
func (iv *InputValidator) validateActiveMedication(med ActiveMedication, index int) error {
	if med.MedicationCode == "" {
		return fmt.Errorf("medication_code is required")
	}

	if med.Name == "" {
		return fmt.Errorf("medication name is required")
	}

	if !med.IsActive {
		iv.logger.Printf("DEBUG: Inactive medication in active list: %s", med.Name)
	}

	// Validate start date
	if med.StartDate.IsZero() {
		iv.logger.Printf("DEBUG: Missing start date for active medication: %s", med.Name)
	}

	return nil
}

// ValidateDDIRules validates drug-drug interaction rules
func (iv *InputValidator) ValidateDDIRules(rules []DrugInteraction) error {
	if rules == nil {
		return fmt.Errorf("ddi_rules cannot be nil - DDI filtering requires interaction data")
	}

	if len(rules) == 0 {
		iv.logger.Printf("WARNING: DDI rules list is empty - DDI filtering will be ineffective")
		return nil
	}

	// Validate each DDI rule
	for i, rule := range rules {
		if err := iv.validateDDIRule(rule, i); err != nil {
			return fmt.Errorf("invalid DDI rule at index %d: %w", i, err)
		}
	}

	return nil
}

// validateDDIRule validates a single DDI rule
func (iv *InputValidator) validateDDIRule(rule DrugInteraction, index int) error {
	if rule.ID == "" {
		return fmt.Errorf("DDI rule ID is required")
	}

	if rule.Drug1 == "" || rule.Drug2 == "" {
		return fmt.Errorf("both drug1 and drug2 are required for DDI rule")
	}

	if rule.Severity == "" {
		return fmt.Errorf("severity is required for DDI rule")
	}

	// Validate severity values
	validSeverities := []string{"Contraindicated", "Major", "Moderate", "Minor"}
	if !iv.contains(validSeverities, string(rule.Severity)) {
		return fmt.Errorf("invalid severity '%s', must be one of: %s", rule.Severity, strings.Join(validSeverities, ", "))
	}

	if rule.Description == "" {
		iv.logger.Printf("DEBUG: DDI rule %s missing description", rule.ID)
	}

	return nil
}

// Helper functions for validation

// isValidSafetyFlag checks if a safety flag follows expected patterns
func (iv *InputValidator) isValidSafetyFlag(flag string) bool {
	knownPatterns := []string{
		"has_history_of_",
		"is_",
		"has_",
		"requires_",
		"contraindicated_for_",
	}

	for _, pattern := range knownPatterns {
		if strings.HasPrefix(flag, pattern) {
			return true
		}
	}

	return false
}

// isValidTherapeuticClass checks if a therapeutic class is recognized
func (iv *InputValidator) isValidTherapeuticClass(class string) bool {
	knownClasses := []string{
		"ACE_INHIBITOR", "ARB", "THIAZIDE_DIURETIC", "BETA_BLOCKER",
		"CALCIUM_CHANNEL_BLOCKER", "ANTIBIOTIC", "ANTIDIABETIC",
		"ANTICOAGULANT", "ANTIPLATELET", "STATIN", "NSAID",
	}

	return iv.contains(knownClasses, class)
}

// contains checks if a slice contains a specific string
func (iv *InputValidator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
