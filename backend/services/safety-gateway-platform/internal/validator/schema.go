package validator

import (
	"fmt"
	"reflect"
	"strings"

	"safety-gateway-platform/pkg/types"
)

// SchemaValidator validates request schemas
type SchemaValidator struct {
	rules map[string]ValidationRule
}

// ValidationRule defines a validation rule for a field
type ValidationRule struct {
	Required    bool
	MinLength   int
	MaxLength   int
	Pattern     string
	AllowedValues []string
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() (*SchemaValidator, error) {
	validator := &SchemaValidator{
		rules: make(map[string]ValidationRule),
	}

	// Define validation rules
	validator.defineRules()

	return validator, nil
}

// defineRules defines validation rules for safety request fields
func (sv *SchemaValidator) defineRules() {
	sv.rules = map[string]ValidationRule{
		"RequestID": {
			Required:  true,
			MinLength: 36,
			MaxLength: 36,
		},
		"PatientID": {
			Required:  true,
			MinLength: 36,
			MaxLength: 36,
		},
		"ClinicianID": {
			Required:  true,
			MinLength: 36,
			MaxLength: 36,
		},
		"ActionType": {
			Required: true,
			AllowedValues: []string{
				"medication_order", "prescription", "medication_administration",
				"procedure_order", "lab_order", "diagnostic_order",
				"treatment_plan", "care_plan", "discharge_plan",
			},
		},
		"Priority": {
			Required: false,
			AllowedValues: []string{
				"low", "normal", "high", "urgent", "emergency",
			},
		},
		"Source": {
			Required:  false,
			MaxLength: 100,
		},
	}
}

// Validate validates a safety request against the schema
func (sv *SchemaValidator) Validate(req *types.SafetyRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Use reflection to validate fields
	v := reflect.ValueOf(req).Elem()
	t := reflect.TypeOf(req).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get validation rule for this field
		rule, exists := sv.rules[fieldName]
		if !exists {
			continue // No rule defined, skip validation
		}

		// Validate based on field type
		switch field.Kind() {
		case reflect.String:
			if err := sv.validateString(fieldName, field.String(), rule); err != nil {
				return err
			}
		case reflect.Slice:
			if err := sv.validateSlice(fieldName, field, rule); err != nil {
				return err
			}
		case reflect.Map:
			if err := sv.validateMap(fieldName, field, rule); err != nil {
				return err
			}
		}
	}

	// Additional custom validations
	if err := sv.validateCustomRules(req); err != nil {
		return err
	}

	return nil
}

// validateString validates a string field
func (sv *SchemaValidator) validateString(fieldName, value string, rule ValidationRule) error {
	// Required validation
	if rule.Required && value == "" {
		return fmt.Errorf("field '%s' is required", fieldName)
	}

	// Skip further validation if field is empty and not required
	if value == "" && !rule.Required {
		return nil
	}

	// Length validation
	if rule.MinLength > 0 && len(value) < rule.MinLength {
		return fmt.Errorf("field '%s' must be at least %d characters", fieldName, rule.MinLength)
	}

	if rule.MaxLength > 0 && len(value) > rule.MaxLength {
		return fmt.Errorf("field '%s' must be at most %d characters", fieldName, rule.MaxLength)
	}

	// Allowed values validation
	if len(rule.AllowedValues) > 0 {
		allowed := false
		for _, allowedValue := range rule.AllowedValues {
			if value == allowedValue {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("field '%s' has invalid value '%s', allowed values: %s",
				fieldName, value, strings.Join(rule.AllowedValues, ", "))
		}
	}

	return nil
}

// validateSlice validates a slice field
func (sv *SchemaValidator) validateSlice(fieldName string, field reflect.Value, rule ValidationRule) error {
	if field.IsNil() {
		return nil // Nil slices are allowed
	}

	length := field.Len()

	// Validate slice elements if they are strings
	if field.Type().Elem().Kind() == reflect.String {
		for i := 0; i < length; i++ {
			element := field.Index(i).String()
			if element == "" {
				return fmt.Errorf("field '%s' contains empty string at index %d", fieldName, i)
			}
			
			// Skip UUID validation for ID fields - allow names instead of UUIDs
			// This allows medication names, condition names, etc. instead of requiring UUIDs
		}
	}

	return nil
}

// validateMap validates a map field
func (sv *SchemaValidator) validateMap(fieldName string, field reflect.Value, rule ValidationRule) error {
	if field.IsNil() {
		return nil // Nil maps are allowed
	}

	// Validate map keys and values
	for _, key := range field.MapKeys() {
		keyStr := key.String()
		value := field.MapIndex(key)

		// Validate key
		if keyStr == "" {
			return fmt.Errorf("field '%s' contains empty key", fieldName)
		}

		if len(keyStr) > 100 {
			return fmt.Errorf("field '%s' contains key that is too long: %s", fieldName, keyStr)
		}

		// Validate value if it's a string
		if value.Kind() == reflect.String {
			valueStr := value.String()
			if len(valueStr) > 1000 {
				return fmt.Errorf("field '%s' contains value that is too long for key '%s'", fieldName, keyStr)
			}
		}
	}

	return nil
}

// validateCustomRules validates custom business rules
func (sv *SchemaValidator) validateCustomRules(req *types.SafetyRequest) error {
	// Validate that at least one of medication, condition, or allergy IDs is provided
	// for certain action types
	actionTypesRequiringIDs := []string{
		"medication_order", "prescription", "medication_administration",
	}

	for _, actionType := range actionTypesRequiringIDs {
		if req.ActionType == actionType {
			if len(req.MedicationIDs) == 0 && len(req.ConditionIDs) == 0 && len(req.AllergyIDs) == 0 {
				return fmt.Errorf("action type '%s' requires at least one medication, condition, or allergy ID", actionType)
			}
		}
	}

	// Validate medication-specific rules
	if len(req.MedicationIDs) > 0 {
		// Limit number of medications per request
		if len(req.MedicationIDs) > 20 {
			return fmt.Errorf("too many medications in single request (max 20)")
		}

		// Check for duplicate medication IDs
		seen := make(map[string]bool)
		for _, medID := range req.MedicationIDs {
			if seen[medID] {
				return fmt.Errorf("duplicate medication ID: %s", medID)
			}
			seen[medID] = true
		}
	}

	// Validate condition-specific rules
	if len(req.ConditionIDs) > 0 {
		// Limit number of conditions per request
		if len(req.ConditionIDs) > 50 {
			return fmt.Errorf("too many conditions in single request (max 50)")
		}

		// Check for duplicate condition IDs
		seen := make(map[string]bool)
		for _, condID := range req.ConditionIDs {
			if seen[condID] {
				return fmt.Errorf("duplicate condition ID: %s", condID)
			}
			seen[condID] = true
		}
	}

	// Validate allergy-specific rules
	if len(req.AllergyIDs) > 0 {
		// Limit number of allergies per request
		if len(req.AllergyIDs) > 30 {
			return fmt.Errorf("too many allergies in single request (max 30)")
		}

		// Check for duplicate allergy IDs
		seen := make(map[string]bool)
		for _, allergyID := range req.AllergyIDs {
			if seen[allergyID] {
				return fmt.Errorf("duplicate allergy ID: %s", allergyID)
			}
			seen[allergyID] = true
		}
	}

	// Validate context map
	if len(req.Context) > 20 {
		return fmt.Errorf("too many context entries (max 20)")
	}

	return nil
}

// AddRule adds a custom validation rule
func (sv *SchemaValidator) AddRule(fieldName string, rule ValidationRule) {
	sv.rules[fieldName] = rule
}

// GetRules returns all validation rules
func (sv *SchemaValidator) GetRules() map[string]ValidationRule {
	return sv.rules
}
