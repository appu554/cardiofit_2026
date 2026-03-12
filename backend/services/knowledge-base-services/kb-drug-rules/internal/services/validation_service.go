package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-drug-rules/internal/models"
)

// validationService implements the ValidationService interface
type validationService struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// NewValidationService creates a new validation service
func NewValidationService(db *gorm.DB, logger *logrus.Logger) ValidationService {
	return &validationService{
		db:     db,
		logger: logger,
	}
}

// ValidateRuleContent validates the content of drug rules
func (v *validationService) ValidateRuleContent(content *models.DrugRuleContent, regions []string) (*models.ValidationResponse, error) {
	response := &models.ValidationResponse{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Info:     []string{},
	}

	// Schema validation
	if err := v.ValidateSchema(content); err != nil {
		response.Valid = false
		response.Errors = append(response.Errors, fmt.Sprintf("Schema validation failed: %s", err.Error()))
	}

	// Expression validation
	if err := v.ValidateExpressions(&content.DoseCalculation); err != nil {
		response.Valid = false
		response.Errors = append(response.Errors, fmt.Sprintf("Expression validation failed: %s", err.Error()))
	}

	// Cross-reference validation
	if err := v.ValidateCrossReferences(content); err != nil {
		response.Warnings = append(response.Warnings, fmt.Sprintf("Cross-reference warning: %s", err.Error()))
	}

	// Clinical safety validation
	if err := v.ValidateClinicalSafety(content); err != nil {
		response.Valid = false
		response.Errors = append(response.Errors, fmt.Sprintf("Clinical safety validation failed: %s", err.Error()))
	}

	// Regional validation
	if err := v.validateRegionalVariations(content, regions); err != nil {
		response.Warnings = append(response.Warnings, fmt.Sprintf("Regional validation warning: %s", err.Error()))
	}

	// Add info messages
	response.Info = append(response.Info, fmt.Sprintf("Validated %d monitoring requirements", len(content.MonitoringRequirements)))
	response.Info = append(response.Info, fmt.Sprintf("Validated %d regional variations", len(content.RegionalVariations)))

	v.logger.WithFields(logrus.Fields{
		"valid":    response.Valid,
		"errors":   len(response.Errors),
		"warnings": len(response.Warnings),
		"regions":  regions,
	}).Debug("Rule content validation completed")

	return response, nil
}

// ValidateSchema validates the schema structure
func (v *validationService) ValidateSchema(content *models.DrugRuleContent) error {
	// Validate required fields
	if content.Meta.DrugName == "" {
		return fmt.Errorf("drug_name is required")
	}

	if len(content.Meta.TherapeuticClass) == 0 {
		return fmt.Errorf("therapeutic_class is required")
	}

	// Validate dose calculation
	if content.DoseCalculation.BaseFormula == "" {
		return fmt.Errorf("base_formula is required")
	}

	if content.DoseCalculation.MaxDailyDose <= 0 {
		return fmt.Errorf("max_daily_dose must be positive")
	}

	if content.DoseCalculation.MinDailyDose < 0 {
		return fmt.Errorf("min_daily_dose cannot be negative")
	}

	if content.DoseCalculation.MaxDailyDose <= content.DoseCalculation.MinDailyDose {
		return fmt.Errorf("max_daily_dose must be greater than min_daily_dose")
	}

	// Validate adjustment factors
	for i, factor := range content.DoseCalculation.AdjustmentFactors {
		if factor.Factor == "" {
			return fmt.Errorf("adjustment_factor[%d].factor is required", i)
		}
		if factor.Condition == "" {
			return fmt.Errorf("adjustment_factor[%d].condition is required", i)
		}
		if factor.Multiplier <= 0 {
			return fmt.Errorf("adjustment_factor[%d].multiplier must be positive", i)
		}
	}

	// Validate renal adjustments
	if content.DoseCalculation.RenalAdjustment != nil {
		for i, threshold := range content.DoseCalculation.RenalAdjustment.EGFRThresholds {
			if threshold.MinEGFR < 0 || threshold.MaxEGFR < 0 {
				return fmt.Errorf("renal_adjustment.egfr_thresholds[%d]: eGFR values cannot be negative", i)
			}
			if threshold.MinEGFR >= threshold.MaxEGFR {
				return fmt.Errorf("renal_adjustment.egfr_thresholds[%d]: min_egfr must be less than max_egfr", i)
			}
			if threshold.DoseMultiplier <= 0 {
				return fmt.Errorf("renal_adjustment.egfr_thresholds[%d]: dose_multiplier must be positive", i)
			}
		}
	}

	// Validate hepatic adjustments
	if content.DoseCalculation.HepaticAdjustment != nil {
		validChildPughClasses := map[string]bool{"A": true, "B": true, "C": true}
		for i, adj := range content.DoseCalculation.HepaticAdjustment.ChildPughAdjustments {
			if !validChildPughClasses[adj.ChildPughClass] {
				return fmt.Errorf("hepatic_adjustment.child_pugh_adjustments[%d]: invalid child_pugh_class '%s'", i, adj.ChildPughClass)
			}
			if adj.DoseMultiplier <= 0 {
				return fmt.Errorf("hepatic_adjustment.child_pugh_adjustments[%d]: dose_multiplier must be positive", i)
			}
		}
	}

	// Validate age adjustments
	for i, adj := range content.DoseCalculation.AgeAdjustments {
		if adj.MinAge < 0 || adj.MaxAge < 0 {
			return fmt.Errorf("age_adjustments[%d]: age values cannot be negative", i)
		}
		if adj.MinAge >= adj.MaxAge {
			return fmt.Errorf("age_adjustments[%d]: min_age must be less than max_age", i)
		}
		if adj.DoseMultiplier <= 0 {
			return fmt.Errorf("age_adjustments[%d]: dose_multiplier must be positive", i)
		}
	}

	// Validate contraindications
	for i, contraindication := range content.SafetyVerification.Contraindications {
		if contraindication.Condition == "" {
			return fmt.Errorf("contraindications[%d].condition is required", i)
		}
		validSeverities := map[string]bool{"absolute": true, "relative": true}
		if !validSeverities[contraindication.Severity] {
			return fmt.Errorf("contraindications[%d]: invalid severity '%s'", i, contraindication.Severity)
		}
	}

	// Validate warnings
	for i, warning := range content.SafetyVerification.Warnings {
		if warning.Description == "" {
			return fmt.Errorf("warnings[%d].description is required", i)
		}
		validSeverities := map[string]bool{"black_box": true, "serious": true, "moderate": true}
		if !validSeverities[warning.Severity] {
			return fmt.Errorf("warnings[%d]: invalid severity '%s'", i, warning.Severity)
		}
	}

	// Validate monitoring requirements
	for i, req := range content.MonitoringRequirements {
		if req.Parameter == "" {
			return fmt.Errorf("monitoring_requirements[%d].parameter is required", i)
		}
		if req.Frequency == "" {
			return fmt.Errorf("monitoring_requirements[%d].frequency is required", i)
		}
		validTypes := map[string]bool{"lab": true, "vital": true, "symptom": true, "efficacy": true}
		if !validTypes[req.Type] {
			return fmt.Errorf("monitoring_requirements[%d]: invalid type '%s'", i, req.Type)
		}
	}

	return nil
}

// ValidateExpressions validates mathematical expressions in dose calculations
func (v *validationService) ValidateExpressions(doseCalc *models.DoseCalculation) error {
	// Validate base formula
	if err := v.validateMathExpression(doseCalc.BaseFormula); err != nil {
		return fmt.Errorf("base_formula validation failed: %w", err)
	}

	// Validate adjustment factor expressions
	for i, factor := range doseCalc.AdjustmentFactors {
		if strings.Contains(factor.Condition, "=") || strings.Contains(factor.Condition, ">") || strings.Contains(factor.Condition, "<") {
			if err := v.validateConditionExpression(factor.Condition); err != nil {
				return fmt.Errorf("adjustment_factor[%d].condition validation failed: %w", i, err)
			}
		}
	}

	return nil
}

// ValidateCrossReferences validates references to other services
func (v *validationService) ValidateCrossReferences(content *models.DrugRuleContent) error {
	// Validate ICD-10 codes in contraindications
	for _, contraindication := range content.SafetyVerification.Contraindications {
		if contraindication.ICD10Code != "" {
			if err := v.validateICD10Code(contraindication.ICD10Code); err != nil {
				return fmt.Errorf("invalid ICD-10 code '%s': %w", contraindication.ICD10Code, err)
			}
		}
	}

	// Validate SNOMED codes
	for _, contraindication := range content.SafetyVerification.Contraindications {
		if contraindication.SNOMEDCode != "" {
			if err := v.validateSNOMEDCode(contraindication.SNOMEDCode); err != nil {
				return fmt.Errorf("invalid SNOMED code '%s': %w", contraindication.SNOMEDCode, err)
			}
		}
	}

	// Validate LOINC codes in lab monitoring
	for _, labMonitoring := range content.SafetyVerification.LabMonitoring {
		if labMonitoring.LOINCCode != "" {
			if err := v.validateLOINCCode(labMonitoring.LOINCCode); err != nil {
				return fmt.Errorf("invalid LOINC code '%s': %w", labMonitoring.LOINCCode, err)
			}
		}
	}

	return nil
}

// ValidateClinicalSafety validates clinical safety rules
func (v *validationService) ValidateClinicalSafety(content *models.DrugRuleContent) error {
	// Check for dangerous dose combinations
	if content.DoseCalculation.MaxDailyDose > 10000 { // Example threshold
		return fmt.Errorf("max_daily_dose exceeds safety threshold (10000mg)")
	}

	// Check for conflicting contraindications and indications
	for _, contraindication := range content.SafetyVerification.Contraindications {
		if contraindication.Severity == "absolute" && strings.Contains(strings.ToLower(contraindication.Condition), "pregnancy") {
			// Check if there are any pregnancy-related dose adjustments
			for _, specialPop := range content.DoseCalculation.SpecialPopulations {
				if specialPop.Population == "pregnancy" && !specialPop.Contraindicated {
					return fmt.Errorf("conflicting pregnancy rules: absolute contraindication but dose adjustment provided")
				}
			}
		}
	}

	// Validate critical lab value thresholds
	for _, labMonitoring := range content.SafetyVerification.LabMonitoring {
		if labMonitoring.CriticalValues.Low != nil && labMonitoring.CriticalValues.High != nil {
			if *labMonitoring.CriticalValues.Low >= *labMonitoring.CriticalValues.High {
				return fmt.Errorf("invalid critical values for %s: low >= high", labMonitoring.Parameter)
			}
		}
	}

	return nil
}

// Helper methods

func (v *validationService) validateRegionalVariations(content *models.DrugRuleContent, regions []string) error {
	// Check if all specified regions have variations defined
	for _, region := range regions {
		if _, exists := content.RegionalVariations[region]; !exists {
			return fmt.Errorf("no regional variation defined for region: %s", region)
		}
	}

	// Validate each regional variation
	for region, variation := range content.RegionalVariations {
		if variation.DoseCalculation != nil {
			if err := v.ValidateExpressions(variation.DoseCalculation); err != nil {
				return fmt.Errorf("regional variation for %s: %w", region, err)
			}
		}
	}

	return nil
}

func (v *validationService) validateMathExpression(expression string) error {
	// Simple validation for mathematical expressions
	// In production, use a proper expression parser
	if expression == "" {
		return fmt.Errorf("expression cannot be empty")
	}

	// Check for basic mathematical operators and variables
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\s\+\-\*\/\(\)\.\,]+$`)
	if !validPattern.MatchString(expression) {
		return fmt.Errorf("expression contains invalid characters")
	}

	// Check for balanced parentheses
	openCount := strings.Count(expression, "(")
	closeCount := strings.Count(expression, ")")
	if openCount != closeCount {
		return fmt.Errorf("unbalanced parentheses in expression")
	}

	return nil
}

func (v *validationService) validateConditionExpression(condition string) error {
	// Simple validation for condition expressions
	// In production, use a proper condition parser
	if condition == "" {
		return fmt.Errorf("condition cannot be empty")
	}

	// Check for valid comparison operators
	hasComparison := strings.Contains(condition, "=") || 
		strings.Contains(condition, ">") || 
		strings.Contains(condition, "<") ||
		strings.Contains(condition, ">=") ||
		strings.Contains(condition, "<=") ||
		strings.Contains(condition, "!=")

	if !hasComparison {
		return fmt.Errorf("condition must contain a comparison operator")
	}

	return nil
}

func (v *validationService) validateICD10Code(code string) error {
	// Basic ICD-10 code validation
	// ICD-10 codes are typically 3-7 characters: letter + 2 digits + optional decimal + 1-4 more characters
	pattern := regexp.MustCompile(`^[A-Z][0-9]{2}(\.[0-9A-Z]{1,4})?$`)
	if !pattern.MatchString(code) {
		return fmt.Errorf("invalid ICD-10 code format")
	}
	return nil
}

func (v *validationService) validateSNOMEDCode(code string) error {
	// Basic SNOMED CT code validation
	// SNOMED codes are typically 6-18 digit numbers
	pattern := regexp.MustCompile(`^[0-9]{6,18}$`)
	if !pattern.MatchString(code) {
		return fmt.Errorf("invalid SNOMED code format")
	}
	return nil
}

func (v *validationService) validateLOINCCode(code string) error {
	// Basic LOINC code validation
	// LOINC codes are typically in format: NNNNN-N
	pattern := regexp.MustCompile(`^[0-9]{4,5}-[0-9]$`)
	if !pattern.MatchString(code) {
		return fmt.Errorf("invalid LOINC code format")
	}
	return nil
}
