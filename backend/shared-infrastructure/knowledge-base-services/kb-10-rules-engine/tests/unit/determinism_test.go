// Package unit provides determinism tests for KB-10 Rules Engine
// CTO/CMO Spec: "Rule evaluation is deterministic" - INVARIANT
package unit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/cardiofit/kb-10-rules-engine/internal/engine"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// createDeterminismTestEngine creates a RulesEngine for determinism testing
func createDeterminismTestEngine(logger *logrus.Logger, rules []*models.Rule) *engine.RulesEngine {
	store := models.NewRuleStore()
	for _, rule := range rules {
		store.Add(rule)
	}

	cache := engine.NewCache(false, 5*time.Minute, logger) // Disable cache for determinism tests

	vaidshalaConfig := &config.VaidshalaConfig{
		Enabled: false,
	}

	return engine.NewRulesEngine(store, nil, cache, vaidshalaConfig, logger, nil)
}

// =============================================================================
// DETERMINISM INVARIANT TESTS
// CTO/CMO Spec: "Rule evaluation is deterministic"
// Same inputs MUST produce same outputs, regardless of execution order or timing
// =============================================================================

func TestDeterminism_SameInputSameOutput(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	// Create test rule
	rule := &models.Rule{
		ID:       "DETERMINISM-001",
		Name:     "Determinism Test Rule",
		Type:     models.RuleTypeAlert,
		Category: "TESTING",
		Severity: models.SeverityCritical,
		Status:   models.StatusActive,
		Priority: 1,
		Conditions: []models.Condition{
			{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
		},
		ConditionLogic: models.LogicAND,
		Actions: []models.Action{
			{Type: models.ActionTypeAlert, Message: "Critical Hyperkalemia"},
		},
	}

	eng := createDeterminismTestEngine(logger, []*models.Rule{rule})

	// Fixed context - same every time
	evalContext := &models.EvaluationContext{
		PatientID: "patient-determinism-123",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8, Unit: "mEq/L"},
		},
	}

	t.Run("100 iterations produce identical results", func(t *testing.T) {
		var results []*models.EvaluationResult

		for i := 0; i < 100; i++ {
			resultList, err := eng.EvaluateSpecific(ctx, []string{"DETERMINISM-001"}, evalContext)
			require.NoError(t, err, "Iteration %d should not error", i)
			require.Len(t, resultList, 1, "Should return exactly 1 result")
			results = append(results, resultList[0])
		}

		// All results must be identical
		firstResult := results[0]
		for i, result := range results[1:] {
			assert.Equal(t, firstResult.Triggered, result.Triggered,
				"Iteration %d: Triggered state must match", i+1)
			assert.Equal(t, firstResult.RuleID, result.RuleID,
				"Iteration %d: RuleID must match", i+1)
			assert.Equal(t, firstResult.Severity, result.Severity,
				"Iteration %d: Severity must match", i+1)
			assert.Equal(t, firstResult.Message, result.Message,
				"Iteration %d: Message must match", i+1)
		}
	})

	t.Run("Order independence - same context in different sequences", func(t *testing.T) {
		// Create multiple rules
		rules := []*models.Rule{
			{
				ID: "ORDER-A", Name: "Rule A", Type: models.RuleTypeAlert,
				Status: models.StatusActive, Priority: 1,
				Conditions:     []models.Condition{{Field: "labs.potassium.value", Operator: "GTE", Value: 6.0}},
				ConditionLogic: models.LogicAND,
				Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "Rule A triggered"}},
			},
			{
				ID: "ORDER-B", Name: "Rule B", Type: models.RuleTypeAlert,
				Status: models.StatusActive, Priority: 2,
				Conditions:     []models.Condition{{Field: "labs.glucose.value", Operator: "LT", Value: 50.0}},
				ConditionLogic: models.LogicAND,
				Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "Rule B triggered"}},
			},
			{
				ID: "ORDER-C", Name: "Rule C", Type: models.RuleTypeAlert,
				Status: models.StatusActive, Priority: 3,
				Conditions:     []models.Condition{{Field: "labs.creatinine.value", Operator: "GT", Value: 2.0}},
				ConditionLogic: models.LogicAND,
				Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "Rule C triggered"}},
			},
		}

		eng := createDeterminismTestEngine(logger, rules)

		// Context that triggers all rules
		evalContext := &models.EvaluationContext{
			PatientID: "order-test-patient",
			Labs: map[string]models.LabValue{
				"potassium":  {Value: 6.5},
				"glucose":    {Value: 45.0},
				"creatinine": {Value: 2.5},
			},
		}

		// Evaluate multiple times
		var allResults [][]*models.EvaluationResult
		for i := 0; i < 10; i++ {
			results, err := eng.Evaluate(ctx, evalContext)
			require.NoError(t, err)
			allResults = append(allResults, results)
		}

		// Compare first result with all others
		firstRun := allResults[0]
		for i, run := range allResults[1:] {
			require.Len(t, run, len(firstRun), "Iteration %d: Should have same number of results", i+1)

			// Build maps for comparison
			firstMap := make(map[string]bool)
			runMap := make(map[string]bool)

			for _, r := range firstRun {
				firstMap[r.RuleID] = r.Triggered
			}
			for _, r := range run {
				runMap[r.RuleID] = r.Triggered
			}

			assert.Equal(t, firstMap, runMap,
				"Iteration %d: Triggered states must be identical", i+1)
		}
	})
}

// =============================================================================
// CONDITION EVALUATION DETERMINISM
// =============================================================================

func TestDeterminism_ConditionEvaluation(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	t.Run("Numeric comparison is deterministic", func(t *testing.T) {
		condition := &models.Condition{
			Field:    "labs.potassium.value",
			Operator: "GTE",
			Value:    6.5,
		}
		ctx := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"potassium": {Value: 6.8},
			},
		}

		results := make([]bool, 100)
		for i := 0; i < 100; i++ {
			result, err := evaluator.EvaluateCondition(condition, ctx)
			require.NoError(t, err, "Iteration %d should not error", i)
			results[i] = result
		}

		// All results must be the same
		firstResult := results[0]
		for i, result := range results {
			assert.Equal(t, firstResult, result,
				"Iteration %d: Condition evaluation must be deterministic", i)
		}
	})

	t.Run("CONTAINS operator is deterministic", func(t *testing.T) {
		condition := &models.Condition{
			Field:    "medications",
			Operator: "CONTAINS",
			Value:    "metformin",
		}
		ctx := &models.EvaluationContext{
			Medications: []models.MedicationContext{
				{Code: "lisinopril", Name: "Lisinopril"},
				{Code: "metformin", Name: "Metformin"},
				{Code: "atorvastatin", Name: "Atorvastatin"},
			},
		}

		results := make([]bool, 100)
		for i := 0; i < 100; i++ {
			result, err := evaluator.EvaluateCondition(condition, ctx)
			require.NoError(t, err, "Iteration %d should not error", i)
			results[i] = result
		}

		// All results must be true (metformin is in the list)
		for i, result := range results {
			assert.True(t, result, "Iteration %d: CONTAINS should find metformin", i)
		}
	})

	t.Run("AGE_GT operator is deterministic", func(t *testing.T) {
		condition := &models.Condition{
			Field:    "patient.date_of_birth", // AGE_GT uses DOB to calculate age
			Operator: "AGE_GT",
			Value:    65,
		}
		dob := time.Now().AddDate(-70, 0, 0) // 70 years old
		ctx := &models.EvaluationContext{
			Patient: models.PatientContext{
				DateOfBirth: dob,
			},
		}

		results := make([]bool, 100)
		for i := 0; i < 100; i++ {
			result, err := evaluator.EvaluateCondition(condition, ctx)
			require.NoError(t, err, "Iteration %d should not error", i)
			results[i] = result
		}

		// All results must be true (70 > 65)
		for i, result := range results {
			assert.True(t, result, "Iteration %d: AGE_GT should be true for 70-year-old", i)
		}
	})
}

// =============================================================================
// PARALLEL DETERMINISM TESTS
// =============================================================================

func TestDeterminism_ParallelEvaluation(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	rule := &models.Rule{
		ID:       "PARALLEL-DET-001",
		Name:     "Parallel Determinism Test",
		Type:     models.RuleTypeAlert,
		Severity: models.SeverityCritical,
		Status:   models.StatusActive,
		Conditions: []models.Condition{
			{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
		},
		ConditionLogic: models.LogicAND,
		Actions: []models.Action{
			{Type: models.ActionTypeAlert, Message: "Critical Alert"},
		},
	}

	eng := createDeterminismTestEngine(logger, []*models.Rule{rule})

	evalContext := &models.EvaluationContext{
		PatientID: "parallel-det-patient",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8},
		},
	}

	t.Run("Concurrent evaluations produce identical results", func(t *testing.T) {
		const numGoroutines = 100
		var wg sync.WaitGroup
		results := make([]*models.EvaluationResult, numGoroutines)
		var mu sync.Mutex

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				resultList, err := eng.EvaluateSpecific(ctx, []string{"PARALLEL-DET-001"}, evalContext)
				require.NoError(t, err)
				require.Len(t, resultList, 1)

				mu.Lock()
				results[idx] = resultList[0]
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// All results must be identical
		firstResult := results[0]
		for i, result := range results[1:] {
			assert.Equal(t, firstResult.Triggered, result.Triggered,
				"Goroutine %d: Triggered state must match", i+1)
			assert.Equal(t, firstResult.RuleID, result.RuleID,
				"Goroutine %d: RuleID must match", i+1)
			assert.Equal(t, firstResult.Severity, result.Severity,
				"Goroutine %d: Severity must match", i+1)
		}
	})
}

// =============================================================================
// EDGE CASE DETERMINISM TESTS
// =============================================================================

func TestDeterminism_EdgeCases(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	t.Run("Boundary value - exactly equal", func(t *testing.T) {
		condition := &models.Condition{
			Field:    "labs.potassium.value",
			Operator: "GTE",
			Value:    6.5,
		}
		ctx := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"potassium": {Value: 6.5}, // Exactly at boundary
			},
		}

		results := make([]bool, 100)
		for i := 0; i < 100; i++ {
			result, err := evaluator.EvaluateCondition(condition, ctx)
			require.NoError(t, err, "Iteration %d should not error", i)
			results[i] = result
		}

		// All must be true (>= includes equality)
		for i, result := range results {
			assert.True(t, result, "Iteration %d: GTE at boundary must be true", i)
		}
	})

	t.Run("Floating point precision", func(t *testing.T) {
		condition := &models.Condition{
			Field:    "labs.test.value",
			Operator: "EQ",
			Value:    0.1 + 0.2, // Classic floating point issue: 0.30000000000000004
		}
		ctx := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"test": {Value: 0.3},
			},
		}

		// Run multiple times to check consistency
		results := make([]bool, 100)
		for i := 0; i < 100; i++ {
			result, err := evaluator.EvaluateCondition(condition, ctx)
			require.NoError(t, err, "Iteration %d should not error", i)
			results[i] = result
		}

		// All results must be the same (whether true or false)
		firstResult := results[0]
		for i, result := range results {
			assert.Equal(t, firstResult, result,
				"Iteration %d: Floating point comparison must be deterministic", i)
		}
	})

	t.Run("Empty collections", func(t *testing.T) {
		condition := &models.Condition{
			Field:    "medications",
			Operator: "CONTAINS",
			Value:    "metformin",
		}
		ctx := &models.EvaluationContext{
			Medications: []models.MedicationContext{}, // Empty
		}

		results := make([]bool, 100)
		for i := 0; i < 100; i++ {
			result, err := evaluator.EvaluateCondition(condition, ctx)
			require.NoError(t, err, "Iteration %d should not error", i)
			results[i] = result
		}

		// All must be false
		for i, result := range results {
			assert.False(t, result, "Iteration %d: Empty collection CONTAINS must be false", i)
		}
	})

	t.Run("Nil/missing field", func(t *testing.T) {
		condition := &models.Condition{
			Field:    "labs.nonexistent.value",
			Operator: "GTE",
			Value:    1.0,
		}
		ctx := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"potassium": {Value: 6.5},
			},
		}

		results := make([]bool, 100)
		for i := 0; i < 100; i++ {
			result, err := evaluator.EvaluateCondition(condition, ctx)
			require.NoError(t, err, "Iteration %d should not error", i)
			results[i] = result
		}

		// All must be the same (likely false for missing field)
		firstResult := results[0]
		for i, result := range results {
			assert.Equal(t, firstResult, result,
				"Iteration %d: Missing field evaluation must be deterministic", i)
		}
	})
}

// =============================================================================
// MULTI-CONDITION DETERMINISM
// =============================================================================

func TestDeterminism_MultipleConditions(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()

	t.Run("AND logic is deterministic", func(t *testing.T) {
		rule := &models.Rule{
			ID:       "MULTI-AND-001",
			Name:     "Multi-condition AND Test",
			Type:     models.RuleTypeAlert,
			Status:   models.StatusActive,
			Conditions: []models.Condition{
				{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
				{Field: "labs.creatinine.value", Operator: "GT", Value: 2.0},
				{Field: "conditions", Operator: "CONTAINS", Value: "diabetes"},
			},
			ConditionLogic: models.LogicAND, // AND
			Actions:        []models.Action{{Type: models.ActionTypeAlert}},
		}

		eng := createDeterminismTestEngine(logger, []*models.Rule{rule})

		evalContext := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"potassium":  {Value: 6.8},
				"creatinine": {Value: 2.5},
			},
			Conditions: []models.ConditionContext{
				{Code: "diabetes", Name: "Diabetes Mellitus"},
			},
		}

		var results []*models.EvaluationResult
		for i := 0; i < 50; i++ {
			resultList, err := eng.EvaluateSpecific(ctx, []string{"MULTI-AND-001"}, evalContext)
			require.NoError(t, err)
			require.Len(t, resultList, 1)
			results = append(results, resultList[0])
		}

		// All must match
		firstResult := results[0]
		for i, result := range results[1:] {
			assert.Equal(t, firstResult.Triggered, result.Triggered,
				"Iteration %d: AND logic must be deterministic", i+1)
		}
	})

	t.Run("OR logic is deterministic", func(t *testing.T) {
		rule := &models.Rule{
			ID:       "MULTI-OR-001",
			Name:     "Multi-condition OR Test",
			Type:     models.RuleTypeAlert,
			Status:   models.StatusActive,
			Conditions: []models.Condition{
				{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
				{Field: "labs.sodium.value", Operator: "LT", Value: 130.0},
			},
			ConditionLogic: "OR",
			Actions:        []models.Action{{Type: models.ActionTypeAlert}},
		}

		eng := createDeterminismTestEngine(logger, []*models.Rule{rule})

		evalContext := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"potassium": {Value: 6.0}, // Below threshold
				"sodium":    {Value: 125}, // Below threshold - should trigger OR
			},
		}

		var results []*models.EvaluationResult
		for i := 0; i < 50; i++ {
			resultList, err := eng.EvaluateSpecific(ctx, []string{"MULTI-OR-001"}, evalContext)
			require.NoError(t, err)
			require.Len(t, resultList, 1)
			results = append(results, resultList[0])
		}

		// All must be triggered (sodium < 130 satisfies OR)
		for i, result := range results {
			assert.True(t, result.Triggered,
				"Iteration %d: OR logic should trigger when one condition is true", i)
		}
	})
}

// =============================================================================
// TIMESTAMP INDEPENDENCE
// =============================================================================

func TestDeterminism_TimestampIndependence(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	t.Run("WITHIN_DAYS is consistent for same relative date", func(t *testing.T) {
		now := time.Now()
		labDate := now.AddDate(0, 0, -10) // 10 days ago

		condition := &models.Condition{
			Field:    "labs.hba1c.date",
			Operator: "WITHIN_DAYS",
			Value:    30, // Within 30 days
		}
		ctx := &models.EvaluationContext{
			Labs: map[string]models.LabValue{
				"hba1c": {Value: 6.5, Date: labDate},
			},
		}

		results := make([]bool, 100)
		for i := 0; i < 100; i++ {
			result, err := evaluator.EvaluateCondition(condition, ctx)
			require.NoError(t, err, "Iteration %d should not error", i)
			results[i] = result
		}

		// All must be true (10 days < 30 days)
		for i, result := range results {
			assert.True(t, result,
				"Iteration %d: WITHIN_DAYS should be true for 10-day-old lab", i)
		}
	})
}
