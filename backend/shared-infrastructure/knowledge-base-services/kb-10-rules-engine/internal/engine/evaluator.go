// Package engine provides the core rule evaluation engine
package engine

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
)

// ConditionEvaluator evaluates rule conditions against patient context
type ConditionEvaluator struct {
	logger *logrus.Logger
}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator(logger *logrus.Logger) *ConditionEvaluator {
	return &ConditionEvaluator{
		logger: logger,
	}
}

// EvaluateConditions evaluates all conditions for a rule
func (e *ConditionEvaluator) EvaluateConditions(rule *models.Rule, ctx *models.EvaluationContext) (bool, []string, []string) {
	if len(rule.Conditions) == 0 {
		return true, nil, nil
	}

	results := make([]bool, len(rule.Conditions))
	conditionsMet := make([]string, 0)
	conditionsFailed := make([]string, 0)

	for i, condition := range rule.Conditions {
		result, err := e.EvaluateCondition(&condition, ctx)
		if err != nil {
			e.logger.WithError(err).WithFields(logrus.Fields{
				"rule_id":   rule.ID,
				"condition": condition.Field,
			}).Debug("Condition evaluation error")
			results[i] = false
			conditionsFailed = append(conditionsFailed, fmt.Sprintf("%s: error - %v", condition.Field, err))
		} else {
			results[i] = result
			if result {
				conditionsMet = append(conditionsMet, condition.Field)
			} else {
				conditionsFailed = append(conditionsFailed, condition.Field)
			}
		}
	}

	// Apply condition logic
	finalResult := e.applyConditionLogic(rule.ConditionLogic, results)

	return finalResult, conditionsMet, conditionsFailed
}

// EvaluateCondition evaluates a single condition
func (e *ConditionEvaluator) EvaluateCondition(cond *models.Condition, ctx *models.EvaluationContext) (bool, error) {
	// Handle CQL expressions (delegate to CQL engine)
	if cond.CQLExpr != "" {
		// CQL evaluation would be delegated to Vaidshala
		// For now, return false as CQL is not implemented inline
		return false, fmt.Errorf("CQL expressions require Vaidshala integration")
	}

	// Get the field value from context
	value, exists := e.getFieldValue(cond.Field, ctx)

	// Handle existence operators first
	switch cond.Operator {
	case models.OperatorEXISTS:
		return exists && value != nil, nil
	case models.OperatorNOTEXISTS:
		return !exists || value == nil, nil
	case models.OperatorISNULL:
		return !exists || value == nil, nil
	case models.OperatorISNOTNULL:
		return exists && value != nil, nil
	}

	// For other operators, the field must exist
	if !exists || value == nil {
		return false, nil
	}

	// Evaluate based on operator
	switch cond.Operator {
	case models.OperatorEQ:
		return e.evaluateEquals(value, cond.Value)
	case models.OperatorNEQ:
		result, err := e.evaluateEquals(value, cond.Value)
		return !result, err
	case models.OperatorGT:
		return e.evaluateGreaterThan(value, cond.Value)
	case models.OperatorGTE:
		return e.evaluateGreaterOrEqual(value, cond.Value)
	case models.OperatorLT:
		return e.evaluateLessThan(value, cond.Value)
	case models.OperatorLTE:
		return e.evaluateLessOrEqual(value, cond.Value)
	case models.OperatorCONTAINS:
		return e.evaluateContains(value, cond.Value)
	case models.OperatorNOTCONTAINS:
		result, err := e.evaluateContains(value, cond.Value)
		return !result, err
	case models.OperatorIN:
		return e.evaluateIn(value, cond.Value)
	case models.OperatorNOTIN:
		result, err := e.evaluateIn(value, cond.Value)
		return !result, err
	case models.OperatorBETWEEN:
		return e.evaluateBetween(value, cond.Value)
	case models.OperatorMATCHES:
		return e.evaluateMatches(value, cond.Value)
	case models.OperatorSTARTSWITH:
		return e.evaluateStartsWith(value, cond.Value)
	case models.OperatorENDSWITH:
		return e.evaluateEndsWith(value, cond.Value)
	case models.OperatorAGEGT:
		return e.evaluateAgeGreaterThan(value, cond.Value)
	case models.OperatorAGELT:
		return e.evaluateAgeLessThan(value, cond.Value)
	case models.OperatorAGEBETWEEN:
		return e.evaluateAgeBetween(value, cond.Value)
	case models.OperatorWITHINDAYS:
		return e.evaluateWithinDays(value, cond.Value)
	case models.OperatorBEFOREDAYS:
		return e.evaluateBeforeDays(value, cond.Value)
	case models.OperatorAFTERDAYS:
		return e.evaluateAfterDays(value, cond.Value)
	default:
		return false, fmt.Errorf("unknown operator: %s", cond.Operator)
	}
}

// getFieldValue retrieves a field value from the evaluation context using dot notation
func (e *ConditionEvaluator) getFieldValue(field string, ctx *models.EvaluationContext) (interface{}, bool) {
	parts := strings.Split(field, ".")
	if len(parts) == 0 {
		return nil, false
	}

	// Get the root object
	var current interface{}
	switch parts[0] {
	case "labs":
		current = ctx.Labs
	case "vitals":
		current = ctx.Vitals
	case "medications":
		current = ctx.Medications
	case "conditions":
		current = ctx.Conditions
	case "allergies":
		current = ctx.Allergies
	case "patient":
		current = e.patientToMap(ctx.Patient)
	case "encounter":
		current = e.encounterToMap(ctx.Encounter)
	case "custom_data":
		current = ctx.CustomData
	default:
		return nil, false
	}

	// Navigate through the remaining parts
	for i := 1; i < len(parts); i++ {
		if current == nil {
			return nil, false
		}

		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[parts[i]]
			if !ok {
				return nil, false
			}
			current = val
		case map[string]models.LabValue:
			val, ok := v[parts[i]]
			if !ok {
				return nil, false
			}
			current = e.labValueToMap(val)
		case map[string]models.VitalSign:
			val, ok := v[parts[i]]
			if !ok {
				return nil, false
			}
			current = e.vitalSignToMap(val)
		case []interface{}:
			// Handle array access or search within array
			return e.handleArrayAccess(v, parts[i:])
		case []models.MedicationContext:
			return e.searchMedications(v, parts[i:])
		case []models.ConditionContext:
			return e.searchConditions(v, parts[i:])
		case []models.AllergyContext:
			return e.searchAllergies(v, parts[i:])
		default:
			// Try reflection for struct access
			val := reflect.ValueOf(current)
			if val.Kind() == reflect.Struct {
				field := val.FieldByNameFunc(func(name string) bool {
					return strings.EqualFold(name, parts[i])
				})
				if field.IsValid() {
					current = field.Interface()
					continue
				}
			}
			return nil, false
		}
	}

	return current, true
}

// Helper conversion functions
func (e *ConditionEvaluator) patientToMap(p models.PatientContext) map[string]interface{} {
	return map[string]interface{}{
		"date_of_birth": p.DateOfBirth,
		"age":           p.Age,
		"gender":        p.Gender,
		"weight":        p.Weight,
		"height":        p.Height,
		"bsa":           p.BSA,
		"pregnant":      p.Pregnant,
		"lactating":     p.Lactating,
	}
}

func (e *ConditionEvaluator) encounterToMap(enc models.EncounterContext) map[string]interface{} {
	return map[string]interface{}{
		"type":       enc.Type,
		"class":      enc.Class,
		"status":     enc.Status,
		"start_date": enc.StartDate,
		"location":   enc.Location,
		"department": enc.Department,
		"provider":   enc.Provider,
	}
}

func (e *ConditionEvaluator) labValueToMap(lab models.LabValue) map[string]interface{} {
	return map[string]interface{}{
		"value":          lab.Value,
		"unit":           lab.Unit,
		"reference_min":  lab.ReferenceMin,
		"reference_max":  lab.ReferenceMax,
		"status":         lab.Status,
		"date":           lab.Date,
		"loinc_code":     lab.LoincCode,
		"interpretation": lab.Interpretation,
	}
}

func (e *ConditionEvaluator) vitalSignToMap(vital models.VitalSign) map[string]interface{} {
	return map[string]interface{}{
		"value":  vital.Value,
		"unit":   vital.Unit,
		"date":   vital.Date,
		"method": vital.Method,
	}
}

func (e *ConditionEvaluator) handleArrayAccess(arr []interface{}, remaining []string) (interface{}, bool) {
	if len(remaining) == 0 {
		return arr, true
	}
	// Check if any element matches
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			if val, exists := m[remaining[0]]; exists {
				return val, true
			}
		}
	}
	return nil, false
}

func (e *ConditionEvaluator) searchMedications(meds []models.MedicationContext, remaining []string) (interface{}, bool) {
	if len(remaining) == 0 {
		return meds, true
	}
	// Search for matching medication
	for _, med := range meds {
		if strings.EqualFold(med.Name, remaining[0]) || med.Code == remaining[0] || med.RxNormCode == remaining[0] {
			return med, true
		}
	}
	return nil, false
}

func (e *ConditionEvaluator) searchConditions(conds []models.ConditionContext, remaining []string) (interface{}, bool) {
	if len(remaining) == 0 {
		return conds, true
	}
	// Search for matching condition
	for _, cond := range conds {
		if strings.EqualFold(cond.Name, remaining[0]) || cond.Code == remaining[0] ||
			cond.ICD10Code == remaining[0] || cond.SnomedCode == remaining[0] {
			return cond, true
		}
	}
	return nil, false
}

func (e *ConditionEvaluator) searchAllergies(allergies []models.AllergyContext, remaining []string) (interface{}, bool) {
	if len(remaining) == 0 {
		return allergies, true
	}
	for _, allergy := range allergies {
		if strings.EqualFold(allergy.Name, remaining[0]) || allergy.Code == remaining[0] {
			return allergy, true
		}
	}
	return nil, false
}

// Operator implementations

func (e *ConditionEvaluator) evaluateEquals(actual, expected interface{}) (bool, error) {
	actualVal, expectedVal := e.normalizeValues(actual, expected)
	return reflect.DeepEqual(actualVal, expectedVal), nil
}

func (e *ConditionEvaluator) evaluateGreaterThan(actual, expected interface{}) (bool, error) {
	actualNum, err := e.toFloat64(actual)
	if err != nil {
		return false, err
	}
	expectedNum, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}
	return actualNum > expectedNum, nil
}

func (e *ConditionEvaluator) evaluateGreaterOrEqual(actual, expected interface{}) (bool, error) {
	actualNum, err := e.toFloat64(actual)
	if err != nil {
		return false, err
	}
	expectedNum, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}
	return actualNum >= expectedNum, nil
}

func (e *ConditionEvaluator) evaluateLessThan(actual, expected interface{}) (bool, error) {
	actualNum, err := e.toFloat64(actual)
	if err != nil {
		return false, err
	}
	expectedNum, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}
	return actualNum < expectedNum, nil
}

func (e *ConditionEvaluator) evaluateLessOrEqual(actual, expected interface{}) (bool, error) {
	actualNum, err := e.toFloat64(actual)
	if err != nil {
		return false, err
	}
	expectedNum, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}
	return actualNum <= expectedNum, nil
}

func (e *ConditionEvaluator) evaluateContains(actual, expected interface{}) (bool, error) {
	actualStr := e.toString(actual)
	expectedStr := e.toString(expected)
	return strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr)), nil
}

func (e *ConditionEvaluator) evaluateIn(actual, expected interface{}) (bool, error) {
	// Expected should be a list
	var expectedList []interface{}
	switch v := expected.(type) {
	case []interface{}:
		expectedList = v
	case []string:
		for _, s := range v {
			expectedList = append(expectedList, s)
		}
	default:
		return false, fmt.Errorf("IN operator requires a list")
	}

	// Convert expected list to lowercase strings for comparison
	expectedStrings := make([]string, len(expectedList))
	for i, item := range expectedList {
		expectedStrings[i] = strings.ToLower(e.toString(item))
	}

	// Handle array of medications - check if ANY medication name is in the expected list
	if meds, ok := actual.([]models.MedicationContext); ok {
		for _, med := range meds {
			medName := strings.ToLower(med.Name)
			for _, expected := range expectedStrings {
				if medName == expected {
					return true, nil
				}
			}
		}
		return false, nil
	}

	// Handle array of conditions - check if ANY condition code/name is in the expected list
	if conds, ok := actual.([]models.ConditionContext); ok {
		for _, cond := range conds {
			condCode := strings.ToLower(cond.Code)
			condName := strings.ToLower(cond.Name)
			for _, expected := range expectedStrings {
				if condCode == expected || condName == expected {
					return true, nil
				}
			}
		}
		return false, nil
	}

	// Handle array of allergies - check if ANY allergy is in the expected list
	if allergies, ok := actual.([]models.AllergyContext); ok {
		for _, allergy := range allergies {
			allergyName := strings.ToLower(allergy.Name)
			allergyCode := strings.ToLower(allergy.Code)
			for _, expected := range expectedStrings {
				if allergyName == expected || allergyCode == expected {
					return true, nil
				}
			}
		}
		return false, nil
	}

	// Standard single value comparison
	actualStr := strings.ToLower(e.toString(actual))
	for _, expected := range expectedStrings {
		if actualStr == expected {
			return true, nil
		}
	}
	return false, nil
}

func (e *ConditionEvaluator) evaluateBetween(actual, expected interface{}) (bool, error) {
	actualNum, err := e.toFloat64(actual)
	if err != nil {
		return false, err
	}

	// Expected should be [min, max]
	var min, max float64
	switch v := expected.(type) {
	case []interface{}:
		if len(v) != 2 {
			return false, fmt.Errorf("BETWEEN requires [min, max]")
		}
		min, err = e.toFloat64(v[0])
		if err != nil {
			return false, err
		}
		max, err = e.toFloat64(v[1])
		if err != nil {
			return false, err
		}
	case map[string]interface{}:
		minVal, ok1 := v["min"]
		maxVal, ok2 := v["max"]
		if !ok1 || !ok2 {
			return false, fmt.Errorf("BETWEEN requires min and max")
		}
		min, err = e.toFloat64(minVal)
		if err != nil {
			return false, err
		}
		max, err = e.toFloat64(maxVal)
		if err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("BETWEEN requires [min, max] or {min, max}")
	}

	return actualNum >= min && actualNum <= max, nil
}

func (e *ConditionEvaluator) evaluateMatches(actual, expected interface{}) (bool, error) {
	actualStr := e.toString(actual)
	pattern := e.toString(expected)

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return regex.MatchString(actualStr), nil
}

func (e *ConditionEvaluator) evaluateStartsWith(actual, expected interface{}) (bool, error) {
	actualStr := strings.ToLower(e.toString(actual))
	expectedStr := strings.ToLower(e.toString(expected))
	return strings.HasPrefix(actualStr, expectedStr), nil
}

func (e *ConditionEvaluator) evaluateEndsWith(actual, expected interface{}) (bool, error) {
	actualStr := strings.ToLower(e.toString(actual))
	expectedStr := strings.ToLower(e.toString(expected))
	return strings.HasSuffix(actualStr, expectedStr), nil
}

func (e *ConditionEvaluator) evaluateAgeGreaterThan(actual, expected interface{}) (bool, error) {
	age, err := e.calculateAge(actual)
	if err != nil {
		return false, err
	}
	threshold, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}
	return float64(age) > threshold, nil
}

func (e *ConditionEvaluator) evaluateAgeLessThan(actual, expected interface{}) (bool, error) {
	age, err := e.calculateAge(actual)
	if err != nil {
		return false, err
	}
	threshold, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}
	return float64(age) < threshold, nil
}

func (e *ConditionEvaluator) evaluateAgeBetween(actual, expected interface{}) (bool, error) {
	age, err := e.calculateAge(actual)
	if err != nil {
		return false, err
	}

	var min, max float64
	switch v := expected.(type) {
	case []interface{}:
		if len(v) != 2 {
			return false, fmt.Errorf("AGE_BETWEEN requires [min, max]")
		}
		min, err = e.toFloat64(v[0])
		if err != nil {
			return false, err
		}
		max, err = e.toFloat64(v[1])
		if err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("AGE_BETWEEN requires [min, max]")
	}

	return float64(age) >= min && float64(age) <= max, nil
}

func (e *ConditionEvaluator) evaluateWithinDays(actual, expected interface{}) (bool, error) {
	date, err := e.toTime(actual)
	if err != nil {
		return false, err
	}
	days, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}

	cutoff := time.Now().AddDate(0, 0, -int(days))
	return date.After(cutoff), nil
}

func (e *ConditionEvaluator) evaluateBeforeDays(actual, expected interface{}) (bool, error) {
	date, err := e.toTime(actual)
	if err != nil {
		return false, err
	}
	days, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}

	cutoff := time.Now().AddDate(0, 0, -int(days))
	return date.Before(cutoff), nil
}

func (e *ConditionEvaluator) evaluateAfterDays(actual, expected interface{}) (bool, error) {
	date, err := e.toTime(actual)
	if err != nil {
		return false, err
	}
	days, err := e.toFloat64(expected)
	if err != nil {
		return false, err
	}

	cutoff := time.Now().AddDate(0, 0, -int(days))
	return date.After(cutoff), nil
}

// Helper functions

func (e *ConditionEvaluator) normalizeValues(a, b interface{}) (interface{}, interface{}) {
	// Try to normalize both to the same type
	if aNum, err := e.toFloat64(a); err == nil {
		if bNum, err := e.toFloat64(b); err == nil {
			return aNum, bNum
		}
	}
	return strings.ToLower(e.toString(a)), strings.ToLower(e.toString(b))
}

func (e *ConditionEvaluator) toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	case map[string]interface{}:
		if numVal, ok := val["value"]; ok {
			return e.toFloat64(numVal)
		}
	}
	return 0, fmt.Errorf("cannot convert %T to float64", v)
}

func (e *ConditionEvaluator) toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (e *ConditionEvaluator) toTime(v interface{}) (time.Time, error) {
	switch val := v.(type) {
	case time.Time:
		return val, nil
	case string:
		// Try common date formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02",
			"01/02/2006",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse date: %s", val)
	case map[string]interface{}:
		if dateVal, ok := val["date"]; ok {
			return e.toTime(dateVal)
		}
	}
	return time.Time{}, fmt.Errorf("cannot convert %T to time", v)
}

func (e *ConditionEvaluator) calculateAge(v interface{}) (int, error) {
	// First check if it's already an age (integer)
	if age, ok := v.(int); ok {
		return age, nil
	}

	// Try to parse as date of birth
	dob, err := e.toTime(v)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	years := now.Year() - dob.Year()
	if now.YearDay() < dob.YearDay() {
		years--
	}
	return years, nil
}

// applyConditionLogic applies the condition logic (AND/OR/custom) to results
func (e *ConditionEvaluator) applyConditionLogic(logic string, results []bool) bool {
	if len(results) == 0 {
		return true
	}

	switch strings.ToUpper(logic) {
	case models.LogicAND, "":
		for _, r := range results {
			if !r {
				return false
			}
		}
		return true
	case models.LogicOR:
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	default:
		// Handle custom logic expressions like "((1 AND 2) OR 3) AND 4"
		return e.evaluateCustomLogic(logic, results)
	}
}

// evaluateCustomLogic evaluates custom logic expressions
func (e *ConditionEvaluator) evaluateCustomLogic(expr string, results []bool) bool {
	// Simple parser for expressions like "((1 AND 2) OR 3) AND 4"
	// Replace condition numbers with their boolean values
	expr = strings.ToUpper(expr)

	for i := len(results); i >= 1; i-- {
		val := "FALSE"
		if results[i-1] {
			val = "TRUE"
		}
		expr = strings.ReplaceAll(expr, fmt.Sprintf("%d", i), val)
	}

	// Simple evaluation (very basic - doesn't handle all edge cases)
	return e.evaluateBoolExpr(expr)
}

func (e *ConditionEvaluator) evaluateBoolExpr(expr string) bool {
	expr = strings.TrimSpace(expr)

	// Handle parentheses recursively
	for strings.Contains(expr, "(") {
		// Find innermost parentheses
		start := strings.LastIndex(expr, "(")
		end := strings.Index(expr[start:], ")") + start
		if end <= start {
			return false
		}
		inner := expr[start+1 : end]
		result := e.evaluateBoolExpr(inner)
		resultStr := "FALSE"
		if result {
			resultStr = "TRUE"
		}
		expr = expr[:start] + resultStr + expr[end+1:]
	}

	// Evaluate AND first (higher precedence)
	if strings.Contains(expr, " AND ") {
		parts := strings.Split(expr, " AND ")
		for _, part := range parts {
			if !e.evaluateBoolExpr(strings.TrimSpace(part)) {
				return false
			}
		}
		return true
	}

	// Then evaluate OR
	if strings.Contains(expr, " OR ") {
		parts := strings.Split(expr, " OR ")
		for _, part := range parts {
			if e.evaluateBoolExpr(strings.TrimSpace(part)) {
				return true
			}
		}
		return false
	}

	// Base case - compare case-insensitively
	return strings.ToUpper(strings.TrimSpace(expr)) == "TRUE"
}
