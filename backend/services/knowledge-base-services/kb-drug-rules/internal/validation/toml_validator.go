package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"kb-drug-rules/internal/models"

	"github.com/BurntSushi/toml"
	"github.com/xeipuuv/gojsonschema"
)

// EnhancedTOMLValidator provides comprehensive TOML validation
type EnhancedTOMLValidator struct {
	jsonSchema       *gojsonschema.Schema
	clinicalRules    map[string]ClinicalRule
	requiredFields   []string
	warningRules     []WarningRule
}

// ClinicalRule represents a clinical validation rule
type ClinicalRule struct {
	Name        string
	Description string
	Validator   func(data map[string]interface{}) []string
}

// WarningRule represents a warning validation rule
type WarningRule struct {
	Name        string
	Description string
	Validator   func(data map[string]interface{}) []string
}

// NewEnhancedTOMLValidator creates a new enhanced TOML validator
func NewEnhancedTOMLValidator() *EnhancedTOMLValidator {
	validator := &EnhancedTOMLValidator{
		requiredFields: []string{
			"meta.drug_id",
			"meta.name",
			"meta.version",
			"meta.clinical_reviewer",
		},
		clinicalRules: make(map[string]ClinicalRule),
		warningRules:  make([]WarningRule, 0),
	}

	// Load JSON schema for structural validation
	validator.loadJSONSchema()
	
	// Initialize clinical rules
	validator.initializeClinicalRules()
	
	// Initialize warning rules
	validator.initializeWarningRules()

	return validator
}

// ValidateComprehensive performs comprehensive TOML validation
func (v *EnhancedTOMLValidator) ValidateComprehensive(tomlContent string) models.ValidationResult {
	result := models.ValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
		Score:    100.0,
	}

	// Phase 1: TOML Syntax Validation
	var parsed map[string]interface{}
	if _, err := toml.Decode(tomlContent, &parsed); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, 
			fmt.Sprintf("TOML syntax error: %v", err))
		result.Score = 0
		return result
	}

	// Phase 2: Required Fields Validation
	for _, field := range v.requiredFields {
		if !v.hasNestedField(parsed, field) {
			result.IsValid = false
			result.Errors = append(result.Errors, 
				fmt.Sprintf("Missing required field: %s", field))
			result.Score -= 20
		}
	}

	// Phase 3: Schema Validation (convert to JSON first)
	if v.jsonSchema != nil {
		jsonBytes, _ := json.Marshal(parsed)
		jsonDoc := gojsonschema.NewBytesLoader(jsonBytes)
		schemaResult, err := v.jsonSchema.Validate(jsonDoc)

		if err != nil {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Schema validation error: %v", err))
			result.Score -= 10
		} else if !schemaResult.Valid() {
			for _, desc := range schemaResult.Errors() {
				result.Errors = append(result.Errors, 
					fmt.Sprintf("Schema violation: %s", desc))
				result.IsValid = false
				result.Score -= 15
			}
		}
	}

	// Phase 4: Clinical Rules Validation
	clinicalWarnings := v.validateClinicalRules(parsed)
	result.Warnings = append(result.Warnings, clinicalWarnings...)
	result.Score -= float64(len(clinicalWarnings)) * 2

	// Phase 5: Warning Rules Validation
	warningMessages := v.validateWarningRules(parsed)
	result.Warnings = append(result.Warnings, warningMessages...)
	result.Score -= float64(len(warningMessages)) * 1

	// Phase 6: Version Format Validation
	if versionWarnings := v.validateVersionFormat(parsed); len(versionWarnings) > 0 {
		result.Warnings = append(result.Warnings, versionWarnings...)
		result.Score -= float64(len(versionWarnings)) * 3
	}

	// Ensure score doesn't go below 0
	if result.Score < 0 {
		result.Score = 0
	}

	return result
}

// hasNestedField checks if a nested field exists in data
func (v *EnhancedTOMLValidator) hasNestedField(data map[string]interface{}, fieldPath string) bool {
	parts := strings.Split(fieldPath, ".")
	current := data

	for i, part := range parts {
		if val, exists := current[part]; exists {
			if i == len(parts)-1 {
				// Final field - check if it's not empty
				switch v := val.(type) {
				case string:
					return strings.TrimSpace(v) != ""
				case nil:
					return false
				default:
					return true
				}
			} else {
				// Intermediate field - must be a map
				if nextMap, ok := val.(map[string]interface{}); ok {
					current = nextMap
				} else {
					return false
				}
			}
		} else {
			return false
		}
	}
	return true
}

// loadJSONSchema loads the JSON schema for structural validation
func (v *EnhancedTOMLValidator) loadJSONSchema() {
	// Define the JSON schema for drug rules
	schemaJSON := `{
		"type": "object",
		"required": ["meta"],
		"properties": {
			"meta": {
				"type": "object",
				"required": ["drug_id", "name", "version", "clinical_reviewer"],
				"properties": {
					"drug_id": {"type": "string", "minLength": 1},
					"name": {"type": "string", "minLength": 1},
					"version": {"type": "string", "pattern": "^\\d+\\.\\d+(\\.\\d+)?$"},
					"clinical_reviewer": {"type": "string", "minLength": 1}
				}
			},
			"dose_calculation": {
				"type": "object",
				"properties": {
					"base_dose_mg": {"type": "number", "minimum": 0},
					"max_daily_dose_mg": {"type": "number", "minimum": 0}
				}
			}
		}
	}`

	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err == nil {
		v.jsonSchema = schema
	}
}

// initializeClinicalRules initializes clinical validation rules
func (v *EnhancedTOMLValidator) initializeClinicalRules() {
	// Rule: Dose limits should be reasonable
	v.clinicalRules["dose_limits"] = ClinicalRule{
		Name:        "Dose Limits Validation",
		Description: "Validates that dose limits are within reasonable clinical ranges",
		Validator: func(data map[string]interface{}) []string {
			warnings := []string{}
			
			if doseCalc, ok := data["dose_calculation"].(map[string]interface{}); ok {
				if baseDose, ok := doseCalc["base_dose_mg"].(float64); ok {
					if baseDose > 10000 { // 10g seems excessive for most drugs
						warnings = append(warnings, "Base dose exceeds 10g - please verify")
					}
					if baseDose < 0.001 { // Less than 1 microgram
						warnings = append(warnings, "Base dose is extremely low - please verify")
					}
				}
				
				if maxDose, ok := doseCalc["max_daily_dose_mg"].(float64); ok {
					if maxDose > 50000 { // 50g daily seems excessive
						warnings = append(warnings, "Maximum daily dose exceeds 50g - please verify")
					}
				}
			}
			
			return warnings
		},
	}

	// Rule: Version should follow semantic versioning
	v.clinicalRules["version_format"] = ClinicalRule{
		Name:        "Version Format Validation",
		Description: "Validates semantic versioning format",
		Validator: func(data map[string]interface{}) []string {
			warnings := []string{}
			
			if meta, ok := data["meta"].(map[string]interface{}); ok {
				if version, ok := meta["version"].(string); ok {
					matched, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?$`, version)
					if !matched {
						warnings = append(warnings, "Version should follow semantic versioning (e.g., 1.0.0)")
					}
				}
			}
			
			return warnings
		},
	}
}

// initializeWarningRules initializes warning validation rules
func (v *EnhancedTOMLValidator) initializeWarningRules() {
	// Warning: Missing evidence sources
	v.warningRules = append(v.warningRules, WarningRule{
		Name:        "Evidence Sources",
		Description: "Checks for evidence sources in metadata",
		Validator: func(data map[string]interface{}) []string {
			warnings := []string{}
			
			if meta, ok := data["meta"].(map[string]interface{}); ok {
				if _, hasEvidence := meta["evidence_sources"]; !hasEvidence {
					warnings = append(warnings, "Consider adding evidence_sources to metadata for clinical traceability")
				}
			}
			
			return warnings
		},
	})

	// Warning: Missing therapeutic class
	v.warningRules = append(v.warningRules, WarningRule{
		Name:        "Therapeutic Class",
		Description: "Checks for therapeutic class information",
		Validator: func(data map[string]interface{}) []string {
			warnings := []string{}
			
			if meta, ok := data["meta"].(map[string]interface{}); ok {
				if _, hasClass := meta["therapeutic_class"]; !hasClass {
					warnings = append(warnings, "Consider adding therapeutic_class to metadata for better categorization")
				}
			}
			
			return warnings
		},
	})
}

// validateClinicalRules runs all clinical validation rules
func (v *EnhancedTOMLValidator) validateClinicalRules(data map[string]interface{}) []string {
	var warnings []string
	
	for _, rule := range v.clinicalRules {
		ruleWarnings := rule.Validator(data)
		warnings = append(warnings, ruleWarnings...)
	}
	
	return warnings
}

// validateWarningRules runs all warning validation rules
func (v *EnhancedTOMLValidator) validateWarningRules(data map[string]interface{}) []string {
	var warnings []string
	
	for _, rule := range v.warningRules {
		ruleWarnings := rule.Validator(data)
		warnings = append(warnings, ruleWarnings...)
	}
	
	return warnings
}

// validateVersionFormat validates version format specifically
func (v *EnhancedTOMLValidator) validateVersionFormat(data map[string]interface{}) []string {
	warnings := []string{}
	
	if meta, ok := data["meta"].(map[string]interface{}); ok {
		if version, ok := meta["version"].(string); ok {
			// Check semantic versioning
			semverRegex := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
			if !semverRegex.MatchString(version) {
				warnings = append(warnings, "Version should follow semantic versioning format (major.minor.patch)")
			}
		}
	}
	
	return warnings
}

// ValidateTOMLSyntax validates only TOML syntax
func (v *EnhancedTOMLValidator) ValidateTOMLSyntax(tomlContent string) error {
	var parsed map[string]interface{}
	_, err := toml.Decode(tomlContent, &parsed)
	return err
}
