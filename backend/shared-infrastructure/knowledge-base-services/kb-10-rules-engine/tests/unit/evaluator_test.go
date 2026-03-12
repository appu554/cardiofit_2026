// Package unit provides unit tests for the rules engine
package unit

import (
	"testing"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/engine"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestConditionEvaluator_NumericOperators(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	evaluator := engine.NewConditionEvaluator(logger)

	tests := []struct {
		name      string
		condition models.Condition
		context   *models.EvaluationContext
		expected  bool
	}{
		{
			name: "GTE operator - true when value equals threshold",
			condition: models.Condition{
				Field:    "labs.potassium.value",
				Operator: "GTE",
				Value:    6.5,
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"potassium": {Value: 6.5},
				},
			},
			expected: true,
		},
		{
			name: "GTE operator - true when value exceeds threshold",
			condition: models.Condition{
				Field:    "labs.potassium.value",
				Operator: "GTE",
				Value:    6.5,
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"potassium": {Value: 6.8},
				},
			},
			expected: true,
		},
		{
			name: "GTE operator - false when value below threshold",
			condition: models.Condition{
				Field:    "labs.potassium.value",
				Operator: "GTE",
				Value:    6.5,
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"potassium": {Value: 5.0},
				},
			},
			expected: false,
		},
		{
			name: "LT operator - true when value below threshold",
			condition: models.Condition{
				Field:    "labs.glucose.value",
				Operator: "LT",
				Value:    50,
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"glucose": {Value: 45},
				},
			},
			expected: true,
		},
		{
			name: "LT operator - false when value at threshold",
			condition: models.Condition{
				Field:    "labs.glucose.value",
				Operator: "LT",
				Value:    50,
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"glucose": {Value: 50},
				},
			},
			expected: false,
		},
		{
			name: "EQ operator - true when equal",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "EQ",
				Value:    "male",
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "male"},
			},
			expected: true,
		},
		{
			name: "EQ operator - false when not equal",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "EQ",
				Value:    "male",
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "female"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateCondition(&tt.condition, tt.context)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_ExistenceOperators(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	tests := []struct {
		name      string
		condition models.Condition
		context   *models.EvaluationContext
		expected  bool
	}{
		{
			name: "EXISTS operator - true when field exists",
			condition: models.Condition{
				Field:    "labs.troponin",
				Operator: "EXISTS",
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"troponin": {Value: 0.05},
				},
			},
			expected: true,
		},
		{
			name: "EXISTS operator - false when field missing",
			condition: models.Condition{
				Field:    "labs.troponin",
				Operator: "EXISTS",
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{},
			},
			expected: false,
		},
		{
			name: "IS_NULL operator - true when field missing",
			condition: models.Condition{
				Field:    "labs.creatinine",
				Operator: "IS_NULL",
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{},
			},
			expected: true,
		},
		{
			name: "IS_NOT_NULL operator - true when field exists",
			condition: models.Condition{
				Field:    "labs.potassium",
				Operator: "IS_NOT_NULL",
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"potassium": {Value: 4.5},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateCondition(&tt.condition, tt.context)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_StringOperators(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	tests := []struct {
		name      string
		condition models.Condition
		context   *models.EvaluationContext
		expected  bool
	}{
		{
			name: "CONTAINS operator - true when substring found",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "CONTAINS",
				Value:    "mal",
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "male"},
			},
			expected: true,
		},
		{
			name: "STARTS_WITH operator - true when prefix matches",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "STARTS_WITH",
				Value:    "fem",
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "female"},
			},
			expected: true,
		},
		{
			name: "ENDS_WITH operator - true when suffix matches",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "ENDS_WITH",
				Value:    "ale",
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "female"},
			},
			expected: true,
		},
		{
			name: "MATCHES operator - true when regex matches",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "MATCHES",
				Value:    "^(male|female)$",
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "male"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateCondition(&tt.condition, tt.context)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_ListOperators(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	tests := []struct {
		name      string
		condition models.Condition
		context   *models.EvaluationContext
		expected  bool
	}{
		{
			name: "IN operator - true when value in list",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "IN",
				Value:    []interface{}{"male", "female"},
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "male"},
			},
			expected: true,
		},
		{
			name: "IN operator - false when value not in list",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "IN",
				Value:    []interface{}{"male", "female"},
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "unknown"},
			},
			expected: false,
		},
		{
			name: "NOT_IN operator - true when value not in list",
			condition: models.Condition{
				Field:    "patient.gender",
				Operator: "NOT_IN",
				Value:    []interface{}{"unknown"},
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Gender: "male"},
			},
			expected: true,
		},
		{
			name: "BETWEEN operator - true when value in range",
			condition: models.Condition{
				Field:    "labs.potassium.value",
				Operator: "BETWEEN",
				Value:    []interface{}{3.5, 5.0},
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"potassium": {Value: 4.2},
				},
			},
			expected: true,
		},
		{
			name: "BETWEEN operator - false when value outside range",
			condition: models.Condition{
				Field:    "labs.potassium.value",
				Operator: "BETWEEN",
				Value:    []interface{}{3.5, 5.0},
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"potassium": {Value: 6.5},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateCondition(&tt.condition, tt.context)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_AgeOperators(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	// Create a date of birth for a 70-year-old
	dob := time.Now().AddDate(-70, 0, 0)

	tests := []struct {
		name      string
		condition models.Condition
		context   *models.EvaluationContext
		expected  bool
	}{
		{
			name: "AGE_GT operator - true when age exceeds threshold",
			condition: models.Condition{
				Field:    "patient.date_of_birth",
				Operator: "AGE_GT",
				Value:    65,
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{DateOfBirth: dob},
			},
			expected: true,
		},
		{
			name: "AGE_LT operator - false when age exceeds threshold",
			condition: models.Condition{
				Field:    "patient.date_of_birth",
				Operator: "AGE_LT",
				Value:    65,
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{DateOfBirth: dob},
			},
			expected: false,
		},
		{
			name: "AGE_BETWEEN operator - true when age in range",
			condition: models.Condition{
				Field:    "patient.date_of_birth",
				Operator: "AGE_BETWEEN",
				Value:    []interface{}{65, 75},
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{DateOfBirth: dob},
			},
			expected: true,
		},
		{
			name: "AGE_GT using direct age field",
			condition: models.Condition{
				Field:    "patient.age",
				Operator: "AGE_GT",
				Value:    65,
			},
			context: &models.EvaluationContext{
				Patient: models.PatientContext{Age: 70},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateCondition(&tt.condition, tt.context)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_TemporalOperators(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	recentDate := time.Now().AddDate(0, 0, -5)  // 5 days ago
	oldDate := time.Now().AddDate(0, 0, -100)   // 100 days ago

	tests := []struct {
		name      string
		condition models.Condition
		context   *models.EvaluationContext
		expected  bool
	}{
		{
			name: "WITHIN_DAYS operator - true when date is recent",
			condition: models.Condition{
				Field:    "labs.hba1c.date",
				Operator: "WITHIN_DAYS",
				Value:    30,
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"hba1c": {Value: 6.5, Date: recentDate},
				},
			},
			expected: true,
		},
		{
			name: "WITHIN_DAYS operator - false when date is old",
			condition: models.Condition{
				Field:    "labs.hba1c.date",
				Operator: "WITHIN_DAYS",
				Value:    30,
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"hba1c": {Value: 6.5, Date: oldDate},
				},
			},
			expected: false,
		},
		{
			name: "BEFORE_DAYS operator - true when date is old",
			condition: models.Condition{
				Field:    "labs.hba1c.date",
				Operator: "BEFORE_DAYS",
				Value:    30,
			},
			context: &models.EvaluationContext{
				Labs: map[string]models.LabValue{
					"hba1c": {Value: 6.5, Date: oldDate},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateCondition(&tt.condition, tt.context)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRuleStore_BasicOperations(t *testing.T) {
	store := models.NewRuleStore()

	// Create test rule
	rule := &models.Rule{
		ID:       "TEST-RULE-001",
		Name:     "Test Rule",
		Type:     models.RuleTypeAlert,
		Category: "TEST",
		Severity: models.SeverityCritical,
		Status:   models.StatusActive,
		Priority: 1,
	}

	// Test Add
	store.Add(rule)
	assert.Equal(t, 1, store.Count())

	// Test Get
	retrieved, exists := store.Get("TEST-RULE-001")
	assert.True(t, exists)
	assert.Equal(t, rule.Name, retrieved.Name)

	// Test GetByType
	byType := store.GetByType(models.RuleTypeAlert)
	assert.Len(t, byType, 1)

	// Test GetByCategory
	byCategory := store.GetByCategory("TEST")
	assert.Len(t, byCategory, 1)

	// Test Remove
	removed := store.Remove("TEST-RULE-001")
	assert.True(t, removed)
	assert.Equal(t, 0, store.Count())
}

func TestRuleStore_Query(t *testing.T) {
	store := models.NewRuleStore()

	// Add test rules
	rules := []*models.Rule{
		{ID: "R1", Type: models.RuleTypeAlert, Category: "SAFETY", Status: models.StatusActive, Priority: 1},
		{ID: "R2", Type: models.RuleTypeAlert, Category: "CLINICAL", Status: models.StatusActive, Priority: 2},
		{ID: "R3", Type: models.RuleTypeInference, Category: "SAFETY", Status: models.StatusActive, Priority: 3},
		{ID: "R4", Type: models.RuleTypeAlert, Category: "SAFETY", Status: models.StatusInactive, Priority: 4},
	}

	for _, r := range rules {
		store.Add(r)
	}

	// Query by type
	filter := &models.Filter{Types: []string{models.RuleTypeAlert}}
	result := store.Query(filter)
	assert.Len(t, result, 3)

	// Query by category
	filter = &models.Filter{Categories: []string{"SAFETY"}}
	result = store.Query(filter)
	assert.Len(t, result, 3)

	// Query by type and category
	filter = &models.Filter{
		Types:      []string{models.RuleTypeAlert},
		Categories: []string{"SAFETY"},
	}
	result = store.Query(filter)
	assert.Len(t, result, 2)

	// Query by status
	filter = &models.Filter{Statuses: []string{models.StatusActive}}
	result = store.Query(filter)
	assert.Len(t, result, 3)

	// Query with limit
	filter = &models.Filter{Limit: 2}
	result = store.Query(filter)
	assert.Len(t, result, 2)
}

func TestRule_Validation(t *testing.T) {
	tests := []struct {
		name    string
		rule    models.Rule
		wantErr bool
	}{
		{
			name: "valid rule",
			rule: models.Rule{
				ID:         "TEST-001",
				Name:       "Test Rule",
				Type:       models.RuleTypeAlert,
				Conditions: []models.Condition{{Field: "test", Operator: "EQ", Value: "value"}},
				Actions:    []models.Action{{Type: models.ActionTypeAlert}},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			rule: models.Rule{
				Name:       "Test Rule",
				Type:       models.RuleTypeAlert,
				Conditions: []models.Condition{{Field: "test", Operator: "EQ", Value: "value"}},
				Actions:    []models.Action{{Type: models.ActionTypeAlert}},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			rule: models.Rule{
				ID:         "TEST-001",
				Type:       models.RuleTypeAlert,
				Conditions: []models.Condition{{Field: "test", Operator: "EQ", Value: "value"}},
				Actions:    []models.Action{{Type: models.ActionTypeAlert}},
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			rule: models.Rule{
				ID:         "TEST-001",
				Name:       "Test Rule",
				Type:       "INVALID",
				Conditions: []models.Condition{{Field: "test", Operator: "EQ", Value: "value"}},
				Actions:    []models.Action{{Type: models.ActionTypeAlert}},
			},
			wantErr: true,
		},
		{
			name: "missing conditions (for non-suppression rule)",
			rule: models.Rule{
				ID:      "TEST-001",
				Name:    "Test Rule",
				Type:    models.RuleTypeAlert,
				Actions: []models.Action{{Type: models.ActionTypeAlert}},
			},
			wantErr: true,
		},
		{
			name: "missing actions",
			rule: models.Rule{
				ID:         "TEST-001",
				Name:       "Test Rule",
				Type:       models.RuleTypeAlert,
				Conditions: []models.Condition{{Field: "test", Operator: "EQ", Value: "value"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
