// Package unit provides unit tests for the KB-10 Rules Engine
package unit

import (
	"context"
	"testing"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/cardiofit/kb-10-rules-engine/internal/engine"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEngine creates a test engine with a rule store
func setupTestEngine(t *testing.T) (*engine.RulesEngine, *models.RuleStore) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	store := models.NewRuleStore()
	cache := engine.NewCache(true, 5*time.Minute, logger)
	vaidshalaConfig := &config.VaidshalaConfig{
		URL:     "http://localhost:8096",
		Enabled: false,
	}

	// Create engine with nil db and metrics for unit testing
	rulesEngine := engine.NewRulesEngine(store, nil, cache, vaidshalaConfig, logger, nil)
	return rulesEngine, store
}

// addTestRules loads sample rules for testing
func addTestRules(store *models.RuleStore) {
	// Critical Hyperkalemia Rule
	hyperkalemiaRule := &models.Rule{
		ID:          "ALERT-LAB-K-CRITICAL-HIGH",
		Name:        "Critical Hyperkalemia Alert",
		Description: "Alerts when potassium level is critically elevated",
		Type:        models.RuleTypeAlert,
		Category:    "SAFETY",
		Severity:    models.SeverityCritical,
		Status:      models.StatusActive,
		Priority:    1,
		Conditions: []models.Condition{
			{
				Field:    "labs.potassium.value",
				Operator: models.OperatorGTE,
				Value:    6.5,
			},
		},
		ConditionLogic: models.LogicAND,
		Actions: []models.Action{
			{
				Type:     models.ActionTypeAlert,
				Message:  "CRITICAL: Potassium elevated - Risk of cardiac arrhythmia",
				Priority: "STAT",
			},
		},
		Tags: []string{"electrolyte", "critical", "cardiac-risk"},
	}
	store.Add(hyperkalemiaRule)

	// Hypoglycemia Rule
	hypoglycemiaRule := &models.Rule{
		ID:          "ALERT-LAB-GLUCOSE-CRITICAL-LOW",
		Name:        "Critical Hypoglycemia Alert",
		Description: "Alerts when glucose is critically low",
		Type:        models.RuleTypeAlert,
		Category:    "SAFETY",
		Severity:    models.SeverityCritical,
		Status:      models.StatusActive,
		Priority:    1,
		Conditions: []models.Condition{
			{
				Field:    "labs.glucose.value",
				Operator: models.OperatorLT,
				Value:    50.0,
			},
		},
		ConditionLogic: models.LogicAND,
		Actions: []models.Action{
			{
				Type:     models.ActionTypeAlert,
				Message:  "CRITICAL: Severe hypoglycemia",
				Priority: "STAT",
			},
		},
		Tags: []string{"glucose", "critical", "diabetic-emergency"},
	}
	store.Add(hypoglycemiaRule)

	// Sepsis Inference Rule
	sepsisRule := &models.Rule{
		ID:          "INFERENCE-SEPSIS-SUSPECTED",
		Name:        "Suspected Sepsis Inference",
		Description: "Infers suspected sepsis when SIRS criteria met",
		Type:        models.RuleTypeInference,
		Category:    "CLINICAL",
		Severity:    models.SeverityHigh,
		Status:      models.StatusActive,
		Priority:    10,
		Conditions: []models.Condition{
			{
				Field:    "vitals.temperature.value",
				Operator: models.OperatorGT,
				Value:    38.3,
			},
			{
				Field:    "vitals.heart_rate.value",
				Operator: models.OperatorGT,
				Value:    90.0,
			},
			{
				Field:    "labs.wbc.value",
				Operator: models.OperatorGT,
				Value:    12000.0,
			},
		},
		ConditionLogic: "((1 AND 2) OR 3)",
		Actions: []models.Action{
			{
				Type:    models.ActionTypeInference,
				Message: "Suspected Sepsis - SIRS criteria met",
			},
		},
		Tags: []string{"sepsis", "sirs", "inference"},
	}
	store.Add(sepsisRule)

	// Elderly Patient Rule (tests AGE_GT operator)
	elderlyRule := &models.Rule{
		ID:          "VALIDATION-ELDERLY-PATIENT",
		Name:        "Elderly Patient Flag",
		Description: "Flags elderly patients for additional monitoring",
		Type:        models.RuleTypeValidation,
		Category:    "GOVERNANCE",
		Severity:    models.SeverityModerate,
		Status:      models.StatusActive,
		Priority:    20,
		Conditions: []models.Condition{
			{
				Field:    "patient.age",
				Operator: models.OperatorGTE,
				Value:    65,
			},
		},
		ConditionLogic: models.LogicAND,
		Actions: []models.Action{
			{
				Type:    models.ActionTypeAlert,
				Message: "Elderly patient - consider age-appropriate dosing",
			},
		},
		Tags: []string{"geriatrics", "governance"},
	}
	store.Add(elderlyRule)

	// Inactive Rule (should not trigger)
	inactiveRule := &models.Rule{
		ID:          "ALERT-INACTIVE",
		Name:        "Inactive Test Rule",
		Type:        models.RuleTypeAlert,
		Category:    "SAFETY",
		Severity:    models.SeverityLow,
		Status:      models.StatusInactive,
		Priority:    100,
		Conditions: []models.Condition{
			{
				Field:    "labs.test.value",
				Operator: models.OperatorEQ,
				Value:    true,
			},
		},
		ConditionLogic: models.LogicAND,
		Actions: []models.Action{
			{
				Type:    models.ActionTypeAlert,
				Message: "This should never trigger",
			},
		},
	}
	store.Add(inactiveRule)

	// Sort by priority
	store.SortByPriority()
}

// TestRulesEngine_Evaluate tests basic rule evaluation
func TestRulesEngine_Evaluate(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	tests := []struct {
		name              string
		context           *models.EvaluationContext
		expectedTriggered int
		expectedRuleIDs   []string
	}{
		{
			name: "Critical hyperkalemia triggers alert",
			context: &models.EvaluationContext{
				PatientID: "patient-001",
				Labs: map[string]models.LabValue{
					"potassium": {Value: 6.8, Unit: "mEq/L"},
				},
				Timestamp: time.Now(),
			},
			expectedTriggered: 1,
			expectedRuleIDs:   []string{"ALERT-LAB-K-CRITICAL-HIGH"},
		},
		{
			name: "Critical hypoglycemia triggers alert",
			context: &models.EvaluationContext{
				PatientID: "patient-002",
				Labs: map[string]models.LabValue{
					"glucose": {Value: 45.0, Unit: "mg/dL"},
				},
				Timestamp: time.Now(),
			},
			expectedTriggered: 1,
			expectedRuleIDs:   []string{"ALERT-LAB-GLUCOSE-CRITICAL-LOW"},
		},
		{
			name: "Normal values trigger no alerts",
			context: &models.EvaluationContext{
				PatientID: "patient-003",
				Labs: map[string]models.LabValue{
					"potassium": {Value: 4.5, Unit: "mEq/L"},
					"glucose":   {Value: 100.0, Unit: "mg/dL"},
				},
				Timestamp: time.Now(),
			},
			expectedTriggered: 0,
			expectedRuleIDs:   []string{},
		},
		{
			name: "Multiple critical values trigger multiple alerts",
			context: &models.EvaluationContext{
				PatientID: "patient-004",
				Labs: map[string]models.LabValue{
					"potassium": {Value: 7.0, Unit: "mEq/L"},
					"glucose":   {Value: 40.0, Unit: "mg/dL"},
				},
				Timestamp: time.Now(),
			},
			expectedTriggered: 2,
			expectedRuleIDs:   []string{"ALERT-LAB-K-CRITICAL-HIGH", "ALERT-LAB-GLUCOSE-CRITICAL-LOW"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := eng.Evaluate(context.Background(), tt.context)
			require.NoError(t, err)

			// Count triggered rules
			triggeredCount := 0
			triggeredIDs := make(map[string]bool)
			for _, r := range results {
				if r.Triggered {
					triggeredCount++
					triggeredIDs[r.RuleID] = true
				}
			}

			assert.Equal(t, tt.expectedTriggered, triggeredCount,
				"Expected %d triggered rules, got %d", tt.expectedTriggered, triggeredCount)

			for _, expectedID := range tt.expectedRuleIDs {
				assert.True(t, triggeredIDs[expectedID],
					"Expected rule %s to trigger", expectedID)
			}
		})
	}
}

// TestRulesEngine_EvaluateByType tests filtering by rule type
func TestRulesEngine_EvaluateByType(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	ctx := &models.EvaluationContext{
		PatientID: "patient-001",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8, Unit: "mEq/L"},
			"wbc":       {Value: 15000.0, Unit: "10*3/uL"},
		},
		Vitals: map[string]models.VitalSign{
			"temperature": {Value: 39.0, Unit: "Cel"},
			"heart_rate":  {Value: 110.0, Unit: "/min"},
		},
		Timestamp: time.Now(),
	}

	// Test ALERT type only
	alertResults, err := eng.EvaluateByType(context.Background(), models.RuleTypeAlert, ctx)
	require.NoError(t, err)

	alertTriggered := 0
	for _, r := range alertResults {
		if r.Triggered {
			alertTriggered++
			assert.Equal(t, models.RuleTypeAlert, r.RuleType)
		}
	}
	assert.GreaterOrEqual(t, alertTriggered, 1, "Should trigger at least 1 ALERT rule")

	// Test INFERENCE type only
	inferenceResults, err := eng.EvaluateByType(context.Background(), models.RuleTypeInference, ctx)
	require.NoError(t, err)

	inferenceTriggered := 0
	for _, r := range inferenceResults {
		if r.Triggered {
			inferenceTriggered++
			assert.Equal(t, models.RuleTypeInference, r.RuleType)
		}
	}
	assert.GreaterOrEqual(t, inferenceTriggered, 1, "Should trigger at least 1 INFERENCE rule (sepsis)")
}

// TestRulesEngine_EvaluateByCategory tests filtering by category
func TestRulesEngine_EvaluateByCategory(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	ctx := &models.EvaluationContext{
		PatientID: "patient-001",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8, Unit: "mEq/L"},
		},
		Timestamp: time.Now(),
	}

	// Test SAFETY category
	safetyResults, err := eng.EvaluateByCategory(context.Background(), "SAFETY", ctx)
	require.NoError(t, err)

	for _, r := range safetyResults {
		if r.Triggered {
			assert.Equal(t, "SAFETY", r.Category)
		}
	}
}

// TestRulesEngine_ConditionLogic tests complex condition logic
func TestRulesEngine_ConditionLogic(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	tests := []struct {
		name           string
		context        *models.EvaluationContext
		expectedSepsis bool
	}{
		{
			name: "Sepsis with fever and tachycardia (1 AND 2)",
			context: &models.EvaluationContext{
				PatientID: "patient-001",
				Vitals: map[string]models.VitalSign{
					"temperature": {Value: 39.0, Unit: "Cel"},
					"heart_rate":  {Value: 110.0, Unit: "/min"},
				},
				Labs: map[string]models.LabValue{
					"wbc": {Value: 8000.0, Unit: "10*3/uL"}, // Normal WBC
				},
				Timestamp: time.Now(),
			},
			expectedSepsis: true, // (1 AND 2) is true
		},
		{
			name: "Sepsis with elevated WBC only (condition 3)",
			context: &models.EvaluationContext{
				PatientID: "patient-002",
				Vitals: map[string]models.VitalSign{
					"temperature": {Value: 37.0, Unit: "Cel"}, // Normal
					"heart_rate":  {Value: 70.0, Unit: "/min"}, // Normal
				},
				Labs: map[string]models.LabValue{
					"wbc": {Value: 15000.0, Unit: "10*3/uL"}, // Elevated
				},
				Timestamp: time.Now(),
			},
			expectedSepsis: true, // Condition 3 alone is true
		},
		{
			name: "No sepsis - normal values",
			context: &models.EvaluationContext{
				PatientID: "patient-003",
				Vitals: map[string]models.VitalSign{
					"temperature": {Value: 37.0, Unit: "Cel"},
					"heart_rate":  {Value: 70.0, Unit: "/min"},
				},
				Labs: map[string]models.LabValue{
					"wbc": {Value: 8000.0, Unit: "10*3/uL"},
				},
				Timestamp: time.Now(),
			},
			expectedSepsis: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := eng.Evaluate(context.Background(), tt.context)
			require.NoError(t, err)

			sepsisTriggered := false
			for _, r := range results {
				if r.Triggered && r.RuleID == "INFERENCE-SEPSIS-SUSPECTED" {
					sepsisTriggered = true
					break
				}
			}

			assert.Equal(t, tt.expectedSepsis, sepsisTriggered,
				"Expected sepsis trigger: %v, got: %v", tt.expectedSepsis, sepsisTriggered)
		})
	}
}

// TestRulesEngine_InactiveRulesSkipped tests that inactive rules are not evaluated
func TestRulesEngine_InactiveRulesSkipped(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	ctx := &models.EvaluationContext{
		PatientID: "patient-001",
		Labs: map[string]models.LabValue{
			"test": {Value: 1.0}, // Value representing "true"
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	for _, r := range results {
		assert.NotEqual(t, "ALERT-INACTIVE", r.RuleID,
			"Inactive rule should not be evaluated")
	}
}

// TestRulesEngine_PriorityOrdering tests that rules are evaluated in priority order
func TestRulesEngine_PriorityOrdering(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	ctx := &models.EvaluationContext{
		PatientID: "patient-001",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8, Unit: "mEq/L"},
			"wbc":       {Value: 15000.0, Unit: "10*3/uL"},
		},
		Vitals: map[string]models.VitalSign{
			"temperature": {Value: 39.0, Unit: "Cel"},
			"heart_rate":  {Value: 110.0, Unit: "/min"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	// Verify we got results back
	assert.NotEmpty(t, results, "Should have evaluation results")
}

// TestRulesEngine_EvaluateSpecific tests evaluating specific rules by ID
func TestRulesEngine_EvaluateSpecific(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	ctx := &models.EvaluationContext{
		PatientID: "patient-001",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8, Unit: "mEq/L"},
			"glucose":   {Value: 40.0, Unit: "mg/dL"},
		},
		Timestamp: time.Now(),
	}

	// Evaluate only hyperkalemia rule
	ruleIDs := []string{"ALERT-LAB-K-CRITICAL-HIGH"}
	results, err := eng.EvaluateSpecific(context.Background(), ruleIDs, ctx)
	require.NoError(t, err)

	assert.Len(t, results, 1, "Should only evaluate 1 specific rule")
	assert.Equal(t, "ALERT-LAB-K-CRITICAL-HIGH", results[0].RuleID)
	assert.True(t, results[0].Triggered)
}

// TestRulesEngine_ResultMetadata tests that results contain proper metadata
func TestRulesEngine_ResultMetadata(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	ctx := &models.EvaluationContext{
		PatientID: "patient-001",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8, Unit: "mEq/L"},
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), ctx)
	require.NoError(t, err)

	for _, r := range results {
		if r.Triggered && r.RuleID == "ALERT-LAB-K-CRITICAL-HIGH" {
			assert.Equal(t, "Critical Hyperkalemia Alert", r.RuleName)
			assert.Equal(t, models.SeverityCritical, r.Severity)
			assert.Equal(t, "SAFETY", r.Category)
			assert.NotZero(t, r.ExecutedAt)
			return
		}
	}
	t.Error("Expected hyperkalemia rule to trigger with metadata")
}

// TestRulesEngine_ElderlyPatient tests age-based conditions
func TestRulesEngine_ElderlyPatient(t *testing.T) {
	eng, store := setupTestEngine(t)
	addTestRules(store)

	// Test elderly patient
	elderlyCtx := &models.EvaluationContext{
		PatientID: "patient-elderly",
		Patient: models.PatientContext{
			Age: 75,
		},
		Timestamp: time.Now(),
	}

	results, err := eng.Evaluate(context.Background(), elderlyCtx)
	require.NoError(t, err)

	elderlyTriggered := false
	for _, r := range results {
		if r.Triggered && r.RuleID == "VALIDATION-ELDERLY-PATIENT" {
			elderlyTriggered = true
			break
		}
	}
	assert.True(t, elderlyTriggered, "Elderly patient rule should trigger")

	// Test young patient
	youngCtx := &models.EvaluationContext{
		PatientID: "patient-young",
		Patient: models.PatientContext{
			Age: 45,
		},
		Timestamp: time.Now(),
	}

	results2, err := eng.Evaluate(context.Background(), youngCtx)
	require.NoError(t, err)

	youngTriggered := false
	for _, r := range results2 {
		if r.Triggered && r.RuleID == "VALIDATION-ELDERLY-PATIENT" {
			youngTriggered = true
			break
		}
	}
	assert.False(t, youngTriggered, "Elderly patient rule should NOT trigger for young patient")
}
