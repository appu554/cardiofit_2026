package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

// FormValidationError describes a single slot value validation failure.
type FormValidationError struct {
	SlotName string `json:"slot_name"`
	Message  string `json:"message"`
}

func (e FormValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.SlotName, e.Message)
}

// ValidateSlotValue checks that a raw JSON value conforms to the slot's
// expected data type and constraints (range, allowed values, etc.).
// Returns nil if valid, or a FormValidationError describing the problem.
func ValidateSlotValue(slotName string, raw json.RawMessage) error {
	def, ok := slots.LookupSlot(slotName)
	if !ok {
		return &FormValidationError{SlotName: slotName, Message: "unknown slot"}
	}

	switch def.DataType {
	case "numeric":
		return validateNumeric(slotName, raw, def)
	case "integer":
		return validateInteger(slotName, raw)
	case "boolean":
		return validateBoolean(slotName, raw)
	case "coded_choice":
		return validateCodedChoice(slotName, raw)
	case "text":
		return validateText(slotName, raw)
	case "list":
		return validateList(slotName, raw)
	case "date":
		return validateText(slotName, raw) // dates arrive as ISO strings
	default:
		return nil // unknown type — accept
	}
}

func validateNumeric(slotName string, raw json.RawMessage, def slots.SlotDefinition) error {
	var val float64
	if err := json.Unmarshal(raw, &val); err != nil {
		// Try string-encoded number
		var s string
		if err2 := json.Unmarshal(raw, &s); err2 != nil {
			return &FormValidationError{SlotName: slotName, Message: "expected a number"}
		}
		return &FormValidationError{SlotName: slotName, Message: "expected a number, got string"}
	}

	// Plausibility ranges for known clinical slots
	ranges := numericRanges()
	if r, ok := ranges[slotName]; ok {
		if val < r.min || val > r.max {
			return &FormValidationError{
				SlotName: slotName,
				Message:  fmt.Sprintf("value %.2f outside plausible range [%.0f, %.0f]", val, r.min, r.max),
			}
		}
	}
	return nil
}

func validateInteger(slotName string, raw json.RawMessage) error {
	var val float64
	if err := json.Unmarshal(raw, &val); err != nil {
		return &FormValidationError{SlotName: slotName, Message: "expected an integer"}
	}
	if val != float64(int64(val)) {
		return &FormValidationError{SlotName: slotName, Message: "expected an integer, got decimal"}
	}
	return nil
}

func validateBoolean(slotName string, raw json.RawMessage) error {
	var val bool
	if err := json.Unmarshal(raw, &val); err != nil {
		// Also accept "true"/"false" strings
		var s string
		if err2 := json.Unmarshal(raw, &s); err2 != nil {
			return &FormValidationError{SlotName: slotName, Message: "expected true or false"}
		}
		s = strings.ToLower(s)
		if s != "true" && s != "false" && s != "yes" && s != "no" {
			return &FormValidationError{SlotName: slotName, Message: "expected true/false or yes/no"}
		}
	}
	return nil
}

func validateCodedChoice(slotName string, raw json.RawMessage) error {
	var val string
	if err := json.Unmarshal(raw, &val); err != nil {
		return &FormValidationError{SlotName: slotName, Message: "expected a string choice"}
	}
	if strings.TrimSpace(val) == "" {
		return &FormValidationError{SlotName: slotName, Message: "choice cannot be empty"}
	}
	return nil
}

func validateText(slotName string, raw json.RawMessage) error {
	var val string
	if err := json.Unmarshal(raw, &val); err != nil {
		return &FormValidationError{SlotName: slotName, Message: "expected a string"}
	}
	if len(val) > 2000 {
		return &FormValidationError{SlotName: slotName, Message: "text exceeds 2000 character limit"}
	}
	return nil
}

func validateList(slotName string, raw json.RawMessage) error {
	var val []interface{}
	if err := json.Unmarshal(raw, &val); err != nil {
		// Also accept a single string
		var s string
		if err2 := json.Unmarshal(raw, &s); err2 != nil {
			return &FormValidationError{SlotName: slotName, Message: "expected an array"}
		}
	}
	return nil
}

type numericRange struct {
	min, max float64
}

func numericRanges() map[string]numericRange {
	return map[string]numericRange{
		"age":                      {0, 130},
		"height":                   {30, 280},  // cm
		"weight":                   {1, 400},   // kg
		"bmi":                      {8, 80},
		"fbg":                      {20, 600},  // mg/dL
		"hba1c":                    {3, 20},    // %
		"ppbg":                     {20, 600},  // mg/dL
		"egfr":                     {0, 200},   // mL/min/1.73m²
		"serum_creatinine":         {0.1, 30},  // mg/dL
		"uacr":                     {0, 5000},  // mg/g
		"serum_potassium":          {1, 10},    // mEq/L
		"systolic_bp":              {50, 300},  // mmHg
		"diastolic_bp":             {20, 200},  // mmHg
		"heart_rate":               {20, 250},  // bpm
		"lvef":                     {5, 90},    // %
		"total_cholesterol":        {50, 500},  // mg/dL
		"ldl":                      {10, 400},  // mg/dL
		"hdl":                      {5, 150},   // mg/dL
		"triglycerides":            {20, 2000}, // mg/dL
		"adherence_score":          {0, 1},
		"exercise_minutes_week":    {0, 2000},
		"sleep_hours":              {0, 24},
		"diabetes_duration_years":  {0, 80},
		"hypoglycemia_episodes":    {0, 100},
		"medication_count":         {0, 50},
		"bariatric_surgery_months": {0, 600},
		"mi_stroke_days":           {0, 36500},
	}
}
