// Package unit provides performance tests for KB-10 Rules Engine
// CTO/CMO Spec Performance Targets:
// - Single evaluation: < 20ms
// - 100 concurrent: < 200ms p99
// - Cache hit: < 5ms
package unit

import (
	"context"
	"sort"
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

// createTestRulesEngine creates a RulesEngine for testing with a populated store
func createTestRulesEngine(logger *logrus.Logger, rules []*models.Rule) *engine.RulesEngine {
	store := models.NewRuleStore()
	for _, rule := range rules {
		store.Add(rule)
	}

	cache := engine.NewCache(false, 5*time.Minute, logger) // Disabled cache for perf tests

	vaidshalaConfig := &config.VaidshalaConfig{
		Enabled: false,
	}

	return engine.NewRulesEngine(store, nil, cache, vaidshalaConfig, logger, nil)
}

// =============================================================================
// SINGLE EVALUATION PERFORMANCE TESTS
// CTO/CMO Spec: "Single evaluation < 20ms"
// =============================================================================

func TestPerformance_SingleEvaluation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce log noise

	rule := &models.Rule{
		ID:       "PERF-SINGLE-001",
		Name:     "Single Evaluation Performance",
		Type:     models.RuleTypeAlert,
		Severity: models.SeverityCritical,
		Status:   models.StatusActive,
		Priority: 1,
		Conditions: []models.Condition{
			{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
			{Field: "labs.creatinine.value", Operator: "GT", Value: 2.0},
		},
		ConditionLogic: models.LogicAND,
		Actions: []models.Action{
			{Type: models.ActionTypeAlert, Message: "Critical Lab Alert"},
		},
	}

	eng := createTestRulesEngine(logger, []*models.Rule{rule})
	ctx := context.Background()

	evalContext := &models.EvaluationContext{
		PatientID: "perf-test-patient",
		Labs: map[string]models.LabValue{
			"potassium":  {Value: 6.8, Unit: "mEq/L"},
			"creatinine": {Value: 2.5, Unit: "mg/dL"},
		},
	}

	t.Run("Single evaluation under 20ms", func(t *testing.T) {
		var durations []time.Duration

		for i := 0; i < 100; i++ {
			start := time.Now()
			results, err := eng.EvaluateSpecific(ctx, []string{"PERF-SINGLE-001"}, evalContext)
			duration := time.Since(start)
			require.NoError(t, err)
			require.Len(t, results, 1)
			durations = append(durations, duration)
		}

		// Calculate statistics
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avg := total / time.Duration(len(durations))

		// Sort for percentiles
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		p50 := durations[len(durations)/2]
		p95 := durations[int(float64(len(durations))*0.95)]
		p99 := durations[int(float64(len(durations))*0.99)]
		max := durations[len(durations)-1]

		t.Logf("Single evaluation stats (100 iterations):")
		t.Logf("  Average: %v", avg)
		t.Logf("  P50: %v", p50)
		t.Logf("  P95: %v", p95)
		t.Logf("  P99: %v", p99)
		t.Logf("  Max: %v", max)

		// CTO/CMO spec: < 20ms
		assert.Less(t, p99.Milliseconds(), int64(20),
			"P99 single evaluation should be < 20ms (CTO/CMO spec)")
	})

	t.Run("Complex rule evaluation under 20ms", func(t *testing.T) {
		complexRule := &models.Rule{
			ID:       "PERF-COMPLEX-001",
			Name:     "Complex Rule Performance",
			Type:     models.RuleTypeAlert,
			Severity: models.SeverityCritical,
			Status:   models.StatusActive,
			Conditions: []models.Condition{
				{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
				{Field: "labs.creatinine.value", Operator: "GT", Value: 2.0},
				{Field: "labs.glucose.value", Operator: "LT", Value: 50.0},
				{Field: "conditions", Operator: "CONTAINS", Value: "diabetes"},
				{Field: "patient.age", Operator: "AGE_GT", Value: 65},
			},
			ConditionLogic: models.LogicAND,
			Actions: []models.Action{
				{Type: models.ActionTypeAlert, Message: "Multiple Critical Conditions"},
			},
		}

		eng := createTestRulesEngine(logger, []*models.Rule{complexRule})

		complexContext := &models.EvaluationContext{
			PatientID: "complex-perf-patient",
			Patient: models.PatientContext{
				DateOfBirth: time.Now().AddDate(-70, 0, 0),
			},
			Labs: map[string]models.LabValue{
				"potassium":  {Value: 6.8},
				"creatinine": {Value: 2.5},
				"glucose":    {Value: 45.0},
			},
			Conditions: []models.ConditionContext{
				{Code: "diabetes", Name: "Diabetes Mellitus"},
				{Code: "hypertension", Name: "Hypertension"},
				{Code: "ckd", Name: "Chronic Kidney Disease"},
			},
		}

		var durations []time.Duration
		for i := 0; i < 100; i++ {
			start := time.Now()
			results, err := eng.EvaluateSpecific(ctx, []string{"PERF-COMPLEX-001"}, complexContext)
			duration := time.Since(start)
			require.NoError(t, err)
			require.Len(t, results, 1)
			durations = append(durations, duration)
		}

		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		p99 := durations[int(float64(len(durations))*0.99)]

		assert.Less(t, p99.Milliseconds(), int64(20),
			"P99 complex rule evaluation should be < 20ms")
	})
}

// =============================================================================
// CONCURRENT PERFORMANCE TESTS
// CTO/CMO Spec: "100 concurrent < 200ms p99"
// =============================================================================

func TestPerformance_ConcurrentEvaluation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	rule := &models.Rule{
		ID:             "PERF-CONCURRENT-001",
		Name:           "Concurrent Performance Test",
		Type:           models.RuleTypeAlert,
		Status:         models.StatusActive,
		Conditions:     []models.Condition{{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5}},
		ConditionLogic: models.LogicAND,
		Actions:        []models.Action{{Type: models.ActionTypeAlert}},
	}

	eng := createTestRulesEngine(logger, []*models.Rule{rule})
	ctx := context.Background()

	evalContext := &models.EvaluationContext{
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8},
		},
	}

	t.Run("100 concurrent evaluations under 200ms p99", func(t *testing.T) {
		const numGoroutines = 100
		const iterations = 10

		var allDurations []time.Duration
		var mu sync.Mutex

		for iter := 0; iter < iterations; iter++ {
			var wg sync.WaitGroup

			start := time.Now()

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					eng.EvaluateSpecific(ctx, []string{"PERF-CONCURRENT-001"}, evalContext)
				}()
			}

			wg.Wait()
			totalDuration := time.Since(start)

			mu.Lock()
			allDurations = append(allDurations, totalDuration)
			mu.Unlock()
		}

		// Calculate p99 of total batch durations
		sort.Slice(allDurations, func(i, j int) bool {
			return allDurations[i] < allDurations[j]
		})
		p99 := allDurations[int(float64(len(allDurations))*0.99)]

		t.Logf("100 concurrent evaluations - p99 total time: %v", p99)
		assert.Less(t, p99.Milliseconds(), int64(200),
			"P99 for 100 concurrent evaluations should be < 200ms (CTO/CMO spec)")
	})
}

// =============================================================================
// CACHE PERFORMANCE TESTS
// CTO/CMO Spec: "Cache hit < 5ms"
// =============================================================================

func TestPerformance_CacheHit(t *testing.T) {
	logger := logrus.New()
	c := engine.NewCache(true, 5*time.Minute, logger)
	defer c.Close()

	evalContext := &models.EvaluationContext{
		PatientID: "cache-perf-patient",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8},
		},
	}

	ruleIDs := []string{"CACHE-PERF-001"}
	testResults := []*models.EvaluationResult{
		{
			RuleID:    "CACHE-PERF-001",
			RuleName:  "Cache Performance Test",
			RuleType:  models.RuleTypeAlert,
			Triggered: true,
			Severity:  models.SeverityCritical,
		},
	}

	t.Run("Cache hit under 5ms", func(t *testing.T) {
		c.Set(evalContext, ruleIDs, testResults)

		var durations []time.Duration
		for i := 0; i < 1000; i++ {
			start := time.Now()
			_, found := c.Get(evalContext, ruleIDs)
			duration := time.Since(start)
			require.True(t, found)
			durations = append(durations, duration)
		}

		// Calculate statistics
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		p50 := durations[len(durations)/2]
		p95 := durations[int(float64(len(durations))*0.95)]
		p99 := durations[int(float64(len(durations))*0.99)]

		t.Logf("Cache hit stats (1000 iterations):")
		t.Logf("  P50: %v", p50)
		t.Logf("  P95: %v", p95)
		t.Logf("  P99: %v", p99)

		// CTO/CMO spec: < 5ms
		assert.Less(t, p99.Milliseconds(), int64(5),
			"P99 cache hit should be < 5ms (CTO/CMO spec)")
	})

	t.Run("Cache miss under 5ms", func(t *testing.T) {
		var durations []time.Duration
		for i := 0; i < 1000; i++ {
			missContext := &models.EvaluationContext{PatientID: "nonexistent"}
			start := time.Now()
			c.Get(missContext, []string{"MISS-RULE"})
			duration := time.Since(start)
			durations = append(durations, duration)
		}

		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		p99 := durations[int(float64(len(durations))*0.99)]

		assert.Less(t, p99.Milliseconds(), int64(5),
			"P99 cache miss should be < 5ms")
	})
}

// =============================================================================
// RULE LOADING PERFORMANCE TESTS
// =============================================================================

func TestPerformance_RuleLoading(t *testing.T) {
	t.Run("Load 1000 rules under 100ms", func(t *testing.T) {
		store := models.NewRuleStore()

		// Create 1000 rules
		rules := make([]*models.Rule, 1000)
		for i := 0; i < 1000; i++ {
			rules[i] = &models.Rule{
				ID:       "LOAD-PERF-" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
				Name:     "Load Performance Rule",
				Type:     models.RuleTypeAlert,
				Category: "PERFORMANCE",
				Status:   models.StatusActive,
				Priority: i % 10,
				Conditions: []models.Condition{
					{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
				},
				ConditionLogic: models.LogicAND,
				Actions: []models.Action{
					{Type: models.ActionTypeAlert},
				},
			}
		}

		start := time.Now()
		for _, rule := range rules {
			store.Add(rule)
		}
		duration := time.Since(start)

		t.Logf("Loaded 1000 rules in %v", duration)
		assert.Less(t, duration.Milliseconds(), int64(100),
			"Loading 1000 rules should take < 100ms")
		assert.Equal(t, 1000, store.Count())
	})

	t.Run("Query by type performance", func(t *testing.T) {
		store := models.NewRuleStore()

		// Add mixed rules
		ruleTypes := []string{models.RuleTypeAlert, models.RuleTypeInference, models.RuleTypeSuppression}
		for i := 0; i < 1000; i++ {
			store.Add(&models.Rule{
				ID:     "QUERY-PERF-" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
				Name:   "Query Performance Rule",
				Type:   ruleTypes[i%3],
				Status: models.StatusActive,
				Actions: []models.Action{
					{Type: models.ActionTypeAlert},
				},
			})
		}

		var durations []time.Duration
		for i := 0; i < 100; i++ {
			start := time.Now()
			store.GetByType(models.RuleTypeAlert)
			durations = append(durations, time.Since(start))
		}

		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		p99 := durations[int(float64(len(durations))*0.99)]

		t.Logf("Query by type p99: %v", p99)
		assert.Less(t, p99.Milliseconds(), int64(10),
			"Query by type should be < 10ms")
	})
}

// =============================================================================
// OPERATOR PERFORMANCE TESTS
// =============================================================================

func TestPerformance_Operators(t *testing.T) {
	logger := logrus.New()
	evaluator := engine.NewConditionEvaluator(logger)

	operators := []struct {
		name      string
		condition models.Condition
		context   *models.EvaluationContext
	}{
		{
			name: "EQ operator",
			condition: models.Condition{
				Field: "patient.gender", Operator: "EQ", Value: "male",
			},
			context: &models.EvaluationContext{Patient: models.PatientContext{Gender: "male"}},
		},
		{
			name: "GTE operator",
			condition: models.Condition{
				Field: "labs.potassium.value", Operator: "GTE", Value: 6.5,
			},
			context: &models.EvaluationContext{Labs: map[string]models.LabValue{"potassium": {Value: 6.8}}},
		},
		{
			name: "CONTAINS operator",
			condition: models.Condition{
				Field: "conditions", Operator: "CONTAINS", Value: "diabetes",
			},
			context: &models.EvaluationContext{
				Conditions: []models.ConditionContext{
					{Code: "hypertension", Name: "Hypertension"},
					{Code: "diabetes", Name: "Diabetes Mellitus"},
					{Code: "ckd", Name: "Chronic Kidney Disease"},
				},
			},
		},
		{
			name: "IN operator",
			condition: models.Condition{
				Field: "patient.gender", Operator: "IN", Value: []interface{}{"male", "female"},
			},
			context: &models.EvaluationContext{Patient: models.PatientContext{Gender: "male"}},
		},
		{
			name: "WITHIN_DAYS operator",
			condition: models.Condition{
				Field: "labs.hba1c.date", Operator: "WITHIN_DAYS", Value: 30,
			},
			context: &models.EvaluationContext{Labs: map[string]models.LabValue{
				"hba1c": {Value: 6.5, Date: time.Now().AddDate(0, 0, -10)},
			}},
		},
	}

	for _, op := range operators {
		t.Run(op.name+" performance", func(t *testing.T) {
			var durations []time.Duration
			for i := 0; i < 1000; i++ {
				start := time.Now()
				evaluator.EvaluateCondition(&op.condition, op.context)
				durations = append(durations, time.Since(start))
			}

			sort.Slice(durations, func(i, j int) bool {
				return durations[i] < durations[j]
			})
			p99 := durations[int(float64(len(durations))*0.99)]

			t.Logf("%s p99: %v", op.name, p99)
			assert.Less(t, p99.Microseconds(), int64(1000),
				"%s p99 should be < 1ms", op.name)
		})
	}
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

func BenchmarkSingleRuleEvaluation(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel) // Disable logging for benchmarks

	rule := &models.Rule{
		ID:             "BENCH-001",
		Name:           "Benchmark Rule",
		Type:           models.RuleTypeAlert,
		Status:         models.StatusActive,
		Conditions:     []models.Condition{{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5}},
		ConditionLogic: models.LogicAND,
		Actions:        []models.Action{{Type: models.ActionTypeAlert}},
	}

	eng := createTestRulesEngine(logger, []*models.Rule{rule})
	ctx := context.Background()

	evalContext := &models.EvaluationContext{
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.EvaluateSpecific(ctx, []string{"BENCH-001"}, evalContext)
	}
}

func BenchmarkComplexRuleEvaluation(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	rule := &models.Rule{
		ID:     "BENCH-COMPLEX-001",
		Name:   "Complex Benchmark Rule",
		Type:   models.RuleTypeAlert,
		Status: models.StatusActive,
		Conditions: []models.Condition{
			{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
			{Field: "labs.creatinine.value", Operator: "GT", Value: 2.0},
			{Field: "conditions", Operator: "CONTAINS", Value: "diabetes"},
			{Field: "patient.age", Operator: "AGE_GT", Value: 65},
		},
		ConditionLogic: models.LogicAND,
		Actions:        []models.Action{{Type: models.ActionTypeAlert}},
	}

	eng := createTestRulesEngine(logger, []*models.Rule{rule})
	ctx := context.Background()

	evalContext := &models.EvaluationContext{
		Patient: models.PatientContext{DateOfBirth: time.Now().AddDate(-70, 0, 0)},
		Labs:    map[string]models.LabValue{"potassium": {Value: 6.8}, "creatinine": {Value: 2.5}},
		Conditions: []models.ConditionContext{
			{Code: "diabetes", Name: "Diabetes Mellitus"},
			{Code: "hypertension", Name: "Hypertension"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.EvaluateSpecific(ctx, []string{"BENCH-COMPLEX-001"}, evalContext)
	}
}

func BenchmarkCacheHit(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	c := engine.NewCache(true, 5*time.Minute, logger)
	defer c.Close()

	evalContext := &models.EvaluationContext{PatientID: "bench-patient"}
	ruleIDs := []string{"BENCH-CACHE"}
	testResults := []*models.EvaluationResult{{RuleID: "BENCH-CACHE", Triggered: true}}
	c.Set(evalContext, ruleIDs, testResults)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(evalContext, ruleIDs)
	}
}

func BenchmarkConditionEvaluation(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	evaluator := engine.NewConditionEvaluator(logger)

	condition := models.Condition{
		Field:    "labs.potassium.value",
		Operator: "GTE",
		Value:    6.5,
	}
	evalContext := &models.EvaluationContext{
		Labs: map[string]models.LabValue{"potassium": {Value: 6.8}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.EvaluateCondition(&condition, evalContext)
	}
}

func BenchmarkRuleStoreQuery(b *testing.B) {
	store := models.NewRuleStore()
	for i := 0; i < 1000; i++ {
		store.Add(&models.Rule{
			ID:       "BENCH-STORE-" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
			Type:     models.RuleTypeAlert,
			Category: "BENCHMARK",
			Status:   models.StatusActive,
			Actions:  []models.Action{{Type: models.ActionTypeAlert}},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetByType(models.RuleTypeAlert)
	}
}
