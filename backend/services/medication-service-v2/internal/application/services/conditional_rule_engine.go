package services

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"medication-service-v2/internal/domain/entities"
)

// ConditionalRuleEngine evaluates and executes conditional rules
type ConditionalRuleEngine struct {
	ruleRepository      ConditionalRuleRepository
	functionRegistry    *FunctionRegistry
	performanceMetrics  *PerformanceMetrics
	cacheEnabled        bool
	evaluationCache     map[string]*EvaluationResult
}

// ConditionalRuleRepository defines storage operations for conditional rules
type ConditionalRuleRepository interface {
	Save(ctx context.Context, rule *entities.ConditionalRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ConditionalRule, error)
	GetByProtocol(ctx context.Context, protocolID string) ([]*entities.ConditionalRule, error)
	List(ctx context.Context, filters RuleFilters) ([]*entities.ConditionalRule, error)
	Update(ctx context.Context, rule *entities.ConditionalRule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// RuleFilters defines filters for rule queries
type RuleFilters struct {
	ProtocolID      string    `json:"protocol_id,omitempty"`
	ValidationLevel entities.ValidationLevel `json:"validation_level,omitempty"`
	Priority        *int      `json:"priority,omitempty"`
	CacheEnabled    *bool     `json:"cache_enabled,omitempty"`
	Limit          int       `json:"limit,omitempty"`
	Offset         int       `json:"offset,omitempty"`
}

// EvaluationResult represents the result of rule evaluation
type EvaluationResult struct {
	RuleID         uuid.UUID              `json:"rule_id"`
	Condition      bool                   `json:"condition_met"`
	ResolvedFields map[string]interface{} `json:"resolved_fields"`
	ProcessingTime time.Duration          `json:"processing_time"`
	CacheUsed      bool                   `json:"cache_used"`
	Errors         []string               `json:"errors,omitempty"`
	Warnings       []string               `json:"warnings,omitempty"`
}

// FunctionRegistry manages custom evaluation functions
type FunctionRegistry struct {
	functions map[string]RuleFunction
}

// RuleFunction represents a custom function that can be used in rule evaluation
type RuleFunction func(ctx context.Context, args []interface{}, patientContext entities.PatientContext) (interface{}, error)

// PerformanceMetrics tracks rule engine performance
type PerformanceMetrics struct {
	TotalEvaluations int64         `json:"total_evaluations"`
	AverageTime      time.Duration `json:"average_time"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	ErrorRate        float64       `json:"error_rate"`
}

// NewConditionalRuleEngine creates a new conditional rule engine
func NewConditionalRuleEngine(ruleRepo ConditionalRuleRepository) *ConditionalRuleEngine {
	engine := &ConditionalRuleEngine{
		ruleRepository:     ruleRepo,
		functionRegistry:   NewFunctionRegistry(),
		performanceMetrics: &PerformanceMetrics{},
		cacheEnabled:       true,
		evaluationCache:    make(map[string]*EvaluationResult),
	}

	// Register built-in functions
	engine.registerBuiltinFunctions()

	return engine
}

// NewFunctionRegistry creates a new function registry
func NewFunctionRegistry() *FunctionRegistry {
	return &FunctionRegistry{
		functions: make(map[string]RuleFunction),
	}
}

// EvaluateRules evaluates all rules for a given protocol and patient context
func (e *ConditionalRuleEngine) EvaluateRules(ctx context.Context, protocolID string, patientContext entities.PatientContext) ([]*EvaluationResult, error) {
	// Get rules for protocol
	rules, err := e.ruleRepository.GetByProtocol(ctx, protocolID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rules for protocol")
	}

	// Sort rules by priority (higher priority first)
	e.sortRulesByPriority(rules)

	// Evaluate each rule
	results := make([]*EvaluationResult, 0, len(rules))
	for _, rule := range rules {
		result, err := e.EvaluateRule(ctx, rule, patientContext)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate rule %s", rule.ID)
		}
		results = append(results, result)
	}

	return results, nil
}

// EvaluateRule evaluates a single conditional rule
func (e *ConditionalRuleEngine) EvaluateRule(ctx context.Context, rule *entities.ConditionalRule, patientContext entities.PatientContext) (*EvaluationResult, error) {
	startTime := time.Now()

	// Check cache
	cacheKey := e.generateCacheKey(rule, patientContext)
	if e.cacheEnabled && rule.CacheEnabled {
		if cached, exists := e.evaluationCache[cacheKey]; exists {
			cached.CacheUsed = true
			return cached, nil
		}
	}

	result := &EvaluationResult{
		RuleID:         rule.ID,
		ResolvedFields: make(map[string]interface{}),
		Errors:         make([]string, 0),
		Warnings:       make([]string, 0),
		CacheUsed:      false,
	}

	// Evaluate condition
	if rule.Condition != nil {
		conditionMet, err := e.evaluateCondition(ctx, *rule.Condition, patientContext)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("condition evaluation failed: %v", err))
			result.Condition = false
		} else {
			result.Condition = conditionMet
		}
	} else {
		result.Condition = true // No condition means always applies
	}

	// If condition is met, resolve fields
	if result.Condition {
		resolvedFields, err := e.resolveFields(ctx, rule.Fields, patientContext)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("field resolution failed: %v", err))
		} else {
			result.ResolvedFields = resolvedFields
		}
	}

	// Set processing time
	result.ProcessingTime = time.Since(startTime)

	// Cache result if enabled
	if e.cacheEnabled && rule.CacheEnabled {
		e.evaluationCache[cacheKey] = result
		
		// Set cache expiry
		if rule.CacheTTL > 0 {
			go e.expireCacheEntry(cacheKey, rule.CacheTTL)
		}
	}

	// Update performance metrics
	e.updatePerformanceMetrics(result)

	return result, nil
}

// evaluateCondition evaluates a rule condition
func (e *ConditionalRuleEngine) evaluateCondition(ctx context.Context, condition entities.RuleCondition, patientContext entities.PatientContext) (bool, error) {
	// Handle sub-conditions with logical operators
	if len(condition.SubConditions) > 0 {
		return e.evaluateLogicalCondition(ctx, condition, patientContext)
	}

	// Evaluate single condition
	return e.evaluateSingleCondition(ctx, condition, patientContext)
}

// evaluateLogicalCondition evaluates conditions with logical operators
func (e *ConditionalRuleEngine) evaluateLogicalCondition(ctx context.Context, condition entities.RuleCondition, patientContext entities.PatientContext) (bool, error) {
	results := make([]bool, len(condition.SubConditions))
	
	// Evaluate all sub-conditions
	for i, subCondition := range condition.SubConditions {
		result, err := e.evaluateCondition(ctx, subCondition, patientContext)
		if err != nil {
			return false, err
		}
		results[i] = result
	}

	// Apply logical operator
	switch strings.ToLower(condition.LogicalOperator) {
	case "and":
		for _, result := range results {
			if !result {
				return false, nil
			}
		}
		return true, nil
	case "or":
		for _, result := range results {
			if result {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported logical operator: %s", condition.LogicalOperator)
	}
}

// evaluateSingleCondition evaluates a single condition
func (e *ConditionalRuleEngine) evaluateSingleCondition(ctx context.Context, condition entities.RuleCondition, patientContext entities.PatientContext) (bool, error) {
	// Get field value
	fieldValue, err := e.getFieldValue(ctx, condition.Field, patientContext)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get field value for %s", condition.Field)
	}

	// Evaluate based on operator
	return e.evaluateOperator(fieldValue, condition.Operator, condition.Value)
}

// getFieldValue gets the value of a field from patient context
func (e *ConditionalRuleEngine) getFieldValue(ctx context.Context, fieldPath string, patientContext entities.PatientContext) (interface{}, error) {
	// Handle nested field paths (e.g., "renal_function.egfr")
	parts := strings.Split(fieldPath, ".")
	
	// Start with patient context as root
	var current interface{} = patientContext
	
	for _, part := range parts {
		current = e.getNestedFieldValue(current, part)
		if current == nil {
			return nil, fmt.Errorf("field path %s not found", fieldPath)
		}
	}

	// Handle function calls (e.g., "bmi()")
	if strings.HasSuffix(fieldPath, "()") {
		funcName := strings.TrimSuffix(fieldPath, "()")
		if function, exists := e.functionRegistry.functions[funcName]; exists {
			return function(ctx, []interface{}{}, patientContext)
		}
		return nil, fmt.Errorf("function %s not found", funcName)
	}

	return current, nil
}

// getNestedFieldValue gets a nested field value using reflection
func (e *ConditionalRuleEngine) getNestedFieldValue(obj interface{}, fieldName string) interface{} {
	if obj == nil {
		return nil
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	// Try to find field by name (case-insensitive)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if strings.EqualFold(field.Name, fieldName) {
			fieldValue := v.Field(i)
			if fieldValue.CanInterface() {
				return fieldValue.Interface()
			}
		}
	}

	return nil
}

// evaluateOperator evaluates an operator against two values
func (e *ConditionalRuleEngine) evaluateOperator(left interface{}, operator string, right interface{}) (bool, error) {
	switch strings.ToLower(operator) {
	case "==", "eq":
		return e.compareEqual(left, right), nil
	case "!=", "ne":
		return !e.compareEqual(left, right), nil
	case ">", "gt":
		return e.compareGreater(left, right)
	case "<", "lt":
		return e.compareLess(left, right)
	case ">=", "gte":
		return e.compareGreaterEqual(left, right)
	case "<=", "lte":
		return e.compareLessEqual(left, right)
	case "in":
		return e.compareIn(left, right)
	case "not_in":
		return !e.compareIn(left, right), nil
	case "contains":
		return e.compareContains(left, right)
	case "matches":
		return e.compareRegex(left, right)
	case "between":
		return e.compareBetween(left, right)
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// Comparison helper methods
func (e *ConditionalRuleEngine) compareEqual(left, right interface{}) bool {
	return reflect.DeepEqual(left, right)
}

func (e *ConditionalRuleEngine) compareGreater(left, right interface{}) (bool, error) {
	leftNum, rightNum, err := e.convertToNumbers(left, right)
	if err != nil {
		return false, err
	}
	return leftNum > rightNum, nil
}

func (e *ConditionalRuleEngine) compareLess(left, right interface{}) (bool, error) {
	leftNum, rightNum, err := e.convertToNumbers(left, right)
	if err != nil {
		return false, err
	}
	return leftNum < rightNum, nil
}

func (e *ConditionalRuleEngine) compareGreaterEqual(left, right interface{}) (bool, error) {
	leftNum, rightNum, err := e.convertToNumbers(left, right)
	if err != nil {
		return false, err
	}
	return leftNum >= rightNum, nil
}

func (e *ConditionalRuleEngine) compareLessEqual(left, right interface{}) (bool, error) {
	leftNum, rightNum, err := e.convertToNumbers(left, right)
	if err != nil {
		return false, err
	}
	return leftNum <= rightNum, nil
}

func (e *ConditionalRuleEngine) compareIn(left, right interface{}) (bool, error) {
	// Convert right to slice
	rightValue := reflect.ValueOf(right)
	if rightValue.Kind() != reflect.Slice && rightValue.Kind() != reflect.Array {
		return false, fmt.Errorf("right operand must be array/slice for 'in' operator")
	}

	for i := 0; i < rightValue.Len(); i++ {
		if e.compareEqual(left, rightValue.Index(i).Interface()) {
			return true, nil
		}
	}

	return false, nil
}

func (e *ConditionalRuleEngine) compareContains(left, right interface{}) (bool, error) {
	leftStr, ok := left.(string)
	if !ok {
		return false, fmt.Errorf("left operand must be string for 'contains' operator")
	}

	rightStr, ok := right.(string)
	if !ok {
		return false, fmt.Errorf("right operand must be string for 'contains' operator")
	}

	return strings.Contains(leftStr, rightStr), nil
}

func (e *ConditionalRuleEngine) compareRegex(left, right interface{}) (bool, error) {
	// Implementation would use regexp package
	return false, fmt.Errorf("regex comparison not implemented")
}

func (e *ConditionalRuleEngine) compareBetween(left, right interface{}) (bool, error) {
	// Expect right to be array of [min, max]
	rightValue := reflect.ValueOf(right)
	if rightValue.Kind() != reflect.Slice || rightValue.Len() != 2 {
		return false, fmt.Errorf("right operand must be array of [min, max] for 'between' operator")
	}

	leftNum, err := e.convertToNumber(left)
	if err != nil {
		return false, err
	}

	minNum, err := e.convertToNumber(rightValue.Index(0).Interface())
	if err != nil {
		return false, err
	}

	maxNum, err := e.convertToNumber(rightValue.Index(1).Interface())
	if err != nil {
		return false, err
	}

	return leftNum >= minNum && leftNum <= maxNum, nil
}

// convertToNumbers converts two values to float64
func (e *ConditionalRuleEngine) convertToNumbers(left, right interface{}) (float64, float64, error) {
	leftNum, err := e.convertToNumber(left)
	if err != nil {
		return 0, 0, err
	}

	rightNum, err := e.convertToNumber(right)
	if err != nil {
		return 0, 0, err
	}

	return leftNum, rightNum, nil
}

// convertToNumber converts a value to float64
func (e *ConditionalRuleEngine) convertToNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", value)
	}
}

// resolveFields resolves field requirements for a rule
func (e *ConditionalRuleEngine) resolveFields(ctx context.Context, fields []entities.FieldRequirement, patientContext entities.PatientContext) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})

	for _, field := range fields {
		value, err := e.getFieldValue(ctx, field.Name, patientContext)
		if err != nil {
			if field.Required {
				return nil, errors.Wrapf(err, "required field %s not available", field.Name)
			}
			// Use default value for optional fields
			if field.DefaultValue != nil {
				resolved[field.Name] = field.DefaultValue
			}
			continue
		}

		// Validate field value
		if err := e.validateFieldValue(value, field); err != nil {
			if field.Required {
				return nil, errors.Wrapf(err, "validation failed for required field %s", field.Name)
			}
			continue
		}

		resolved[field.Name] = value
	}

	return resolved, nil
}

// validateFieldValue validates a field value against its requirements
func (e *ConditionalRuleEngine) validateFieldValue(value interface{}, field entities.FieldRequirement) error {
	// Type validation
	if !e.isValidType(value, field.Type) {
		return fmt.Errorf("field %s has invalid type, expected %s", field.Name, field.Type)
	}

	// Range validation for numeric fields
	if field.ValidRange != nil && field.Type == entities.FieldTypeNumber {
		num, err := e.convertToNumber(value)
		if err != nil {
			return err
		}

		if field.ValidRange.Min != nil && num < *field.ValidRange.Min {
			return fmt.Errorf("field %s value %v is below minimum %v", field.Name, num, *field.ValidRange.Min)
		}

		if field.ValidRange.Max != nil && num > *field.ValidRange.Max {
			return fmt.Errorf("field %s value %v is above maximum %v", field.Name, num, *field.ValidRange.Max)
		}
	}

	// Valid values validation
	if len(field.ValidValues) > 0 && field.Type == entities.FieldTypeString {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %s expected string value", field.Name)
		}

		valid := false
		for _, validValue := range field.ValidValues {
			if str == validValue {
				valid = true
				break
			}
		}

		if !valid {
			return fmt.Errorf("field %s value '%s' is not in valid values %v", field.Name, str, field.ValidValues)
		}
	}

	return nil
}

// isValidType checks if a value matches the expected field type
func (e *ConditionalRuleEngine) isValidType(value interface{}, fieldType entities.FieldType) bool {
	switch fieldType {
	case entities.FieldTypeNumber:
		_, err := e.convertToNumber(value)
		return err == nil
	case entities.FieldTypeString:
		_, ok := value.(string)
		return ok
	case entities.FieldTypeBoolean:
		_, ok := value.(bool)
		return ok
	case entities.FieldTypeDate:
		_, ok := value.(time.Time)
		return ok
	case entities.FieldTypeArray:
		v := reflect.ValueOf(value)
		return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
	case entities.FieldTypeObject:
		v := reflect.ValueOf(value)
		return v.Kind() == reflect.Map || v.Kind() == reflect.Struct
	default:
		return false
	}
}

// Helper methods

func (e *ConditionalRuleEngine) generateCacheKey(rule *entities.ConditionalRule, patientContext entities.PatientContext) string {
	return fmt.Sprintf("rule:%s:patient:%s", rule.ID.String(), patientContext.PatientID)
}

func (e *ConditionalRuleEngine) expireCacheEntry(cacheKey string, ttl time.Duration) {
	time.Sleep(ttl)
	delete(e.evaluationCache, cacheKey)
}

func (e *ConditionalRuleEngine) sortRulesByPriority(rules []*entities.ConditionalRule) {
	// Simple bubble sort by priority (higher priority first)
	for i := 0; i < len(rules)-1; i++ {
		for j := 0; j < len(rules)-i-1; j++ {
			if rules[j].Priority < rules[j+1].Priority {
				rules[j], rules[j+1] = rules[j+1], rules[j]
			}
		}
	}
}

func (e *ConditionalRuleEngine) updatePerformanceMetrics(result *EvaluationResult) {
	e.performanceMetrics.TotalEvaluations++
	
	// Update average time (simple moving average)
	totalTime := time.Duration(e.performanceMetrics.TotalEvaluations) * e.performanceMetrics.AverageTime
	e.performanceMetrics.AverageTime = (totalTime + result.ProcessingTime) / time.Duration(e.performanceMetrics.TotalEvaluations)

	// Update cache hit rate
	if result.CacheUsed {
		// Increment cache hits in performance calculation
	}

	// Update error rate
	if len(result.Errors) > 0 {
		// Update error rate calculation
	}
}

// Register built-in functions
func (e *ConditionalRuleEngine) registerBuiltinFunctions() {
	// BMI calculation
	e.functionRegistry.RegisterFunction("bmi", func(ctx context.Context, args []interface{}, patientContext entities.PatientContext) (interface{}, error) {
		if patientContext.Weight > 0 && patientContext.Height > 0 {
			heightM := patientContext.Height / 100 // Convert cm to m
			bmi := patientContext.Weight / (heightM * heightM)
			return math.Round(bmi*10) / 10, nil // Round to 1 decimal place
		}
		return nil, fmt.Errorf("weight and height required for BMI calculation")
	})

	// BSA (Body Surface Area) calculation
	e.functionRegistry.RegisterFunction("bsa", func(ctx context.Context, args []interface{}, patientContext entities.PatientContext) (interface{}, error) {
		if patientContext.Weight > 0 && patientContext.Height > 0 {
			// Mosteller formula
			bsa := math.Sqrt((patientContext.Weight * patientContext.Height) / 3600)
			return math.Round(bsa*100) / 100, nil // Round to 2 decimal places
		}
		return nil, fmt.Errorf("weight and height required for BSA calculation")
	})

	// Creatinine clearance estimation
	e.functionRegistry.RegisterFunction("creatinine_clearance", func(ctx context.Context, args []interface{}, patientContext entities.PatientContext) (interface{}, error) {
		if patientContext.RenalFunction != nil {
			return patientContext.RenalFunction.CreatinineClearance, nil
		}
		return nil, fmt.Errorf("renal function data not available")
	})
}

// RegisterFunction registers a custom function
func (fr *FunctionRegistry) RegisterFunction(name string, function RuleFunction) {
	fr.functions[name] = function
}

// GetFunction gets a registered function
func (fr *FunctionRegistry) GetFunction(name string) (RuleFunction, bool) {
	function, exists := fr.functions[name]
	return function, exists
}