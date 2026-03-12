// Package unit provides concurrency tests for KB-10 Rules Engine
// CTO/CMO Spec: "100 concurrent evaluations < 200ms p99"
package unit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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

// createConcurrencyTestEngine creates a RulesEngine for concurrency testing
func createConcurrencyTestEngine(logger *logrus.Logger, rules []*models.Rule) *engine.RulesEngine {
	store := models.NewRuleStore()
	for _, rule := range rules {
		store.Add(rule)
	}

	cache := engine.NewCache(true, 5*time.Minute, logger) // Enable cache for realistic scenarios

	vaidshalaConfig := &config.VaidshalaConfig{
		Enabled: false,
	}

	return engine.NewRulesEngine(store, nil, cache, vaidshalaConfig, logger, nil)
}

// =============================================================================
// CONCURRENT RULE EVALUATION TESTS
// =============================================================================

func TestConcurrency_ParallelRuleEvaluation(t *testing.T) {
	logger := logrus.New()

	rule := &models.Rule{
		ID:       "CONCURRENT-001",
		Name:     "Concurrent Evaluation Test",
		Type:     models.RuleTypeAlert,
		Status:   models.StatusActive,
		Priority: 1,
		Conditions: []models.Condition{
			{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
		},
		ConditionLogic: models.LogicAND, // Use models constant ("AND")
		Actions:        []models.Action{{Type: models.ActionTypeAlert, Message: "Hyperkalemia Alert"}},
	}

	eng := createConcurrencyTestEngine(logger, []*models.Rule{rule})

	t.Run("100 concurrent evaluations complete without error", func(t *testing.T) {
		const numGoroutines = 100
		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		evalContext := &models.EvaluationContext{
			PatientID: "concurrent-test-patient",
			Labs: map[string]models.LabValue{
				"potassium": {Value: 6.8},
			},
		}

		ctx := context.Background()
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := eng.EvaluateSpecific(ctx, []string{"CONCURRENT-001"}, evalContext)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		assert.Equal(t, int64(numGoroutines), successCount, "All evaluations should succeed")
		assert.Equal(t, int64(0), errorCount, "No evaluations should fail")
		assert.Less(t, duration.Milliseconds(), int64(200),
			"100 concurrent evaluations should complete in < 200ms (CTO/CMO spec)")
	})

	t.Run("Concurrent evaluations with different contexts", func(t *testing.T) {
		const numGoroutines = 100
		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		ctx := context.Background()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				// Each goroutine has its own context with different patient ID
				evalCtx := &models.EvaluationContext{
					PatientID: fmt.Sprintf("patient-%d", idx),
					Labs: map[string]models.LabValue{
						"potassium": {Value: 6.8}, // Above threshold
					},
				}
				_, err := eng.EvaluateSpecific(ctx, []string{"CONCURRENT-001"}, evalCtx)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}(i)
		}

		wg.Wait()

		// Concurrency test: verify no errors or panics during concurrent execution
		assert.Equal(t, int64(numGoroutines), successCount, "All evaluations should complete")
		assert.Zero(t, errorCount, "No evaluations should error")
	})
}

// =============================================================================
// CONCURRENT RULE STORE ACCESS TESTS
// =============================================================================

func TestConcurrency_RuleStoreAccess(t *testing.T) {
	store := models.NewRuleStore()

	t.Run("Concurrent reads are safe", func(t *testing.T) {
		// Pre-populate store
		for i := 0; i < 100; i++ {
			store.Add(&models.Rule{
				ID:       fmt.Sprintf("RULE-%d", i),
				Name:     "Test Rule",
				Type:     models.RuleTypeAlert,
				Status:   models.StatusActive,
				Category: "TEST",
			})
		}

		var wg sync.WaitGroup
		const numReaders = 100

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				// Random read operations
				store.Get("RULE-0")
				store.GetByType(models.RuleTypeAlert)
				store.GetByCategory("TEST")
				store.Count()
			}(i)
		}

		wg.Wait()
		// No panic or deadlock means success
	})

	t.Run("Concurrent reads and writes are safe", func(t *testing.T) {
		freshStore := models.NewRuleStore()
		var wg sync.WaitGroup
		const numOperations = 100

		for i := 0; i < numOperations; i++ {
			wg.Add(2)

			// Writer
			go func(idx int) {
				defer wg.Done()
				freshStore.Add(&models.Rule{
					ID:     fmt.Sprintf("CONCURRENT-RULE-%d", idx%10),
					Name:   "Concurrent Rule",
					Type:   models.RuleTypeAlert,
					Status: models.StatusActive,
				})
			}(i)

			// Reader
			go func(idx int) {
				defer wg.Done()
				freshStore.Get(fmt.Sprintf("CONCURRENT-RULE-%d", idx%10))
				freshStore.Count()
			}(i)
		}

		wg.Wait()
		// No panic or deadlock means success
	})
}

// =============================================================================
// CONCURRENT CACHE ACCESS TESTS
// =============================================================================

func TestConcurrency_CacheAccess(t *testing.T) {
	logger := logrus.New()

	rule := &models.Rule{
		ID:             "CACHE-CONCURRENT-001",
		Name:           "Cache Concurrent Test",
		Type:           models.RuleTypeAlert,
		Status:         models.StatusActive,
		Conditions:     []models.Condition{{Field: "labs.glucose.value", Operator: "LT", Value: 50.0}},
		ConditionLogic: models.LogicAND, // Use models constant ("AND")
		Actions:        []models.Action{{Type: models.ActionTypeAlert}},
	}

	eng := createConcurrencyTestEngine(logger, []*models.Rule{rule})

	evalCtx := &models.EvaluationContext{
		PatientID: "cache-test-patient",
		Labs: map[string]models.LabValue{
			"glucose": {Value: 45.0},
		},
	}

	ctx := context.Background()

	t.Run("Concurrent cache hits are consistent", func(t *testing.T) {
		const numGoroutines = 100
		var wg sync.WaitGroup
		results := make([][]*models.EvaluationResult, numGoroutines)

		// First evaluation to populate cache
		firstResultList, _ := eng.EvaluateSpecific(ctx, []string{"CACHE-CONCURRENT-001"}, evalCtx)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				result, _ := eng.EvaluateSpecific(ctx, []string{"CACHE-CONCURRENT-001"}, evalCtx)
				results[idx] = result
			}(i)
		}

		wg.Wait()

		// All results should match the first result
		for i, resultList := range results {
			if len(resultList) > 0 && len(firstResultList) > 0 {
				assert.Equal(t, firstResultList[0].Triggered, resultList[0].Triggered,
					"Goroutine %d result should match first result", i)
			}
		}
	})
}

// =============================================================================
// CONCURRENT ALERT GENERATION TESTS
// =============================================================================

func TestConcurrency_AlertGeneration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel) // Enable debug logging

	rule := &models.Rule{
		ID:       "ALERT-CONCURRENT-001",
		Name:     "Concurrent Alert Test",
		Type:     models.RuleTypeAlert,
		Severity: models.SeverityCritical,
		Status:   models.StatusActive,
		Conditions: []models.Condition{
			{Field: "labs.lactate.value", Operator: "GTE", Value: 4.0},
		},
		ConditionLogic: models.LogicAND, // Use models constant ("AND")
		Actions: []models.Action{
			{Type: models.ActionTypeAlert, Message: "Sepsis Alert"},
		},
	}

	eng := createConcurrencyTestEngine(logger, []*models.Rule{rule})
	ctx := context.Background()

	// First verify single synchronous evaluation works
	t.Run("Single synchronous evaluation should trigger", func(t *testing.T) {
		evalCtx := &models.EvaluationContext{
			PatientID: "test-patient",
			Labs: map[string]models.LabValue{
				"lactate": {Value: 4.5},
			},
		}
		results, err := eng.EvaluateSpecific(ctx, []string{"ALERT-CONCURRENT-001"}, evalCtx)
		require.NoError(t, err, "Evaluation should not error")
		require.Len(t, results, 1, "Should have 1 result")
		t.Logf("RuleID: %s, Triggered: %v, ConditionsMet: %v, ConditionsFailed: %v",
			results[0].RuleID, results[0].Triggered, results[0].ConditionsMet, results[0].ConditionsFailed)
		assert.True(t, results[0].Triggered, "Rule should be triggered (4.5 >= 4.0)")
	})

	t.Run("Concurrent alert generation produces consistent results", func(t *testing.T) {
		const numGoroutines = 50
		var wg sync.WaitGroup
		var triggeredCount int64

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				evalCtx := &models.EvaluationContext{
					PatientID: fmt.Sprintf("patient-%d", idx),
					Labs: map[string]models.LabValue{
						"lactate": {Value: 4.5},
					},
				}
				results, err := eng.EvaluateSpecific(ctx, []string{"ALERT-CONCURRENT-001"}, evalCtx)
				if err == nil && len(results) > 0 && results[0].Triggered {
					atomic.AddInt64(&triggeredCount, 1)
				}
			}(i)
		}

		wg.Wait()

		// All evaluations should trigger (lactate 4.5 >= 4.0)
		assert.Equal(t, int64(numGoroutines), triggeredCount,
			"All concurrent evaluations should trigger alert")
	})
}

// =============================================================================
// CONCURRENT MULTI-RULE EVALUATION TESTS
// =============================================================================

func TestConcurrency_MultiRuleEvaluation(t *testing.T) {
	logger := logrus.New()

	rules := []*models.Rule{
		{
			ID: "MULTI-RULE-A", Name: "Potassium Check", Type: models.RuleTypeAlert,
			Status: models.StatusActive, Priority: 1,
			Conditions:     []models.Condition{{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5}},
			ConditionLogic: models.LogicAND, // Use models constant ("AND")
			Actions:        []models.Action{{Type: models.ActionTypeAlert}},
		},
		{
			ID: "MULTI-RULE-B", Name: "Glucose Check", Type: models.RuleTypeAlert,
			Status: models.StatusActive, Priority: 2,
			Conditions:     []models.Condition{{Field: "labs.glucose.value", Operator: "LT", Value: 50.0}},
			ConditionLogic: models.LogicAND, // Use models constant ("AND")
			Actions:        []models.Action{{Type: models.ActionTypeAlert}},
		},
		{
			ID: "MULTI-RULE-C", Name: "Creatinine Check", Type: models.RuleTypeAlert,
			Status: models.StatusActive, Priority: 3,
			Conditions:     []models.Condition{{Field: "labs.creatinine.value", Operator: "GT", Value: 2.0}},
			ConditionLogic: models.LogicAND, // Use models constant ("AND")
			Actions:        []models.Action{{Type: models.ActionTypeAlert}},
		},
	}

	eng := createConcurrencyTestEngine(logger, rules)
	ctx := context.Background()

	evalCtx := &models.EvaluationContext{
		PatientID: "multi-rule-patient",
		Labs: map[string]models.LabValue{
			"potassium":  {Value: 6.8},
			"glucose":    {Value: 45.0},
			"creatinine": {Value: 2.5},
		},
	}

	t.Run("Concurrent multi-rule evaluation", func(t *testing.T) {
		const numGoroutines = 50
		var wg sync.WaitGroup
		var totalTriggered int64

		ruleIDs := []string{"MULTI-RULE-A", "MULTI-RULE-B", "MULTI-RULE-C"}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results, err := eng.EvaluateSpecific(ctx, ruleIDs, evalCtx)
				if err == nil {
					for _, result := range results {
						if result.Triggered {
							atomic.AddInt64(&totalTriggered, 1)
						}
					}
				}
			}()
		}

		wg.Wait()

		// Each goroutine should trigger all 3 rules
		expectedTriggered := int64(numGoroutines * len(rules))
		assert.Equal(t, expectedTriggered, totalTriggered,
			"All rules should trigger for each goroutine")
	})
}

// =============================================================================
// CONTEXT CANCELLATION TESTS
// =============================================================================

func TestConcurrency_ContextCancellation(t *testing.T) {
	logger := logrus.New()

	rule := &models.Rule{
		ID:             "CANCEL-001",
		Name:           "Cancellation Test",
		Type:           models.RuleTypeAlert,
		Status:         models.StatusActive,
		Conditions:     []models.Condition{{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5}},
		ConditionLogic: models.LogicAND, // Use models constant ("AND")
		Actions:        []models.Action{{Type: models.ActionTypeAlert}},
	}

	eng := createConcurrencyTestEngine(logger, []*models.Rule{rule})

	evalCtx := &models.EvaluationContext{
		PatientID: "cancel-test-patient",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8},
		},
	}

	t.Run("Evaluation respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		var cancelledCount int64

		// Start many goroutines
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Check context before evaluation
				select {
				case <-ctx.Done():
					atomic.AddInt64(&cancelledCount, 1)
					return
				default:
				}

				// Simulate some work before evaluation
				time.Sleep(time.Millisecond)

				select {
				case <-ctx.Done():
					atomic.AddInt64(&cancelledCount, 1)
					return
				default:
					eng.EvaluateSpecific(ctx, []string{"CANCEL-001"}, evalCtx)
				}
			}()
		}

		// Cancel after short delay
		time.Sleep(5 * time.Millisecond)
		cancel()

		wg.Wait()

		t.Logf("Cancelled %d goroutines", cancelledCount)
		// Some goroutines should be cancelled
	})

	t.Run("Timeout prevents long-running evaluation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		const numGoroutines = 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
					eng.EvaluateSpecific(ctx, []string{"CANCEL-001"}, evalCtx)
				}
			}()
		}

		wg.Wait()
		// All should complete or be cancelled - no hanging
	})
}

// =============================================================================
// RACE CONDITION TESTS
// =============================================================================

func TestConcurrency_RaceConditions(t *testing.T) {
	logger := logrus.New()

	rule := &models.Rule{
		ID:     "RACE-001",
		Name:   "Race Condition Test",
		Type:   models.RuleTypeAlert,
		Status: models.StatusActive,
		Conditions: []models.Condition{
			{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
		},
		ConditionLogic: models.LogicAND, // Use models constant ("AND")
		Actions:        []models.Action{{Type: models.ActionTypeAlert}},
	}

	eng := createConcurrencyTestEngine(logger, []*models.Rule{rule})
	ctx := context.Background()

	t.Run("No data races in rule evaluation", func(t *testing.T) {
		// Shared context - potential race if not handled properly
		sharedContext := &models.EvaluationContext{
			PatientID: "race-test-patient",
			Labs: map[string]models.LabValue{
				"potassium": {Value: 6.8},
			},
		}

		var wg sync.WaitGroup
		const numGoroutines = 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					eng.EvaluateSpecific(ctx, []string{"RACE-001"}, sharedContext)
				}
			}()
		}

		wg.Wait()
		// Run with -race flag to detect data races
	})
}

// =============================================================================
// THROUGHPUT TESTS
// =============================================================================

func TestConcurrency_Throughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	logger := logrus.New()

	rule := &models.Rule{
		ID:             "THROUGHPUT-001",
		Name:           "Throughput Test",
		Type:           models.RuleTypeAlert,
		Status:         models.StatusActive,
		Conditions:     []models.Condition{{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5}},
		ConditionLogic: models.LogicAND, // Use models constant ("AND")
		Actions:        []models.Action{{Type: models.ActionTypeAlert}},
	}

	eng := createConcurrencyTestEngine(logger, []*models.Rule{rule})
	bgCtx := context.Background()

	evalCtx := &models.EvaluationContext{
		PatientID: "throughput-patient",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8},
		},
	}

	t.Run("Measure throughput at different concurrency levels", func(t *testing.T) {
		concurrencyLevels := []int{1, 10, 50, 100}

		for _, numWorkers := range concurrencyLevels {
			const totalEvaluations = 1000
			evaluationsPerWorker := totalEvaluations / numWorkers

			start := time.Now()
			var wg sync.WaitGroup

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < evaluationsPerWorker; j++ {
						eng.EvaluateSpecific(bgCtx, []string{"THROUGHPUT-001"}, evalCtx)
					}
				}()
			}

			wg.Wait()
			duration := time.Since(start)

			throughput := float64(totalEvaluations) / duration.Seconds()
			t.Logf("Concurrency %d: %.2f evaluations/sec (%.2fms total)",
				numWorkers, throughput, float64(duration.Nanoseconds())/1e6)
		}
	})
}

// =============================================================================
// STRESS TESTS
// =============================================================================

func TestConcurrency_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	logger := logrus.New()

	rules := make([]*models.Rule, 20)
	ruleIDs := make([]string, 20)
	for i := 0; i < 20; i++ {
		ruleID := fmt.Sprintf("STRESS-%d", i)
		rules[i] = &models.Rule{
			ID:     ruleID,
			Name:   "Stress Test Rule",
			Type:   models.RuleTypeAlert,
			Status: models.StatusActive,
			Conditions: []models.Condition{
				{Field: "labs.potassium.value", Operator: "GTE", Value: 6.5},
			},
			ConditionLogic: models.LogicAND, // Use models constant ("AND")
			Actions:        []models.Action{{Type: models.ActionTypeAlert}},
		}
		ruleIDs[i] = ruleID
	}

	eng := createConcurrencyTestEngine(logger, rules)

	evalCtx := &models.EvaluationContext{
		PatientID: "stress-test-patient",
		Labs: map[string]models.LabValue{
			"potassium": {Value: 6.8},
		},
	}

	t.Run("Sustained high concurrency", func(t *testing.T) {
		const numGoroutines = 200
		const duration = 5 * time.Second
		var evaluationCount int64
		var errorCount int64

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						results, err := eng.EvaluateSpecific(ctx, ruleIDs, evalCtx)
						if err != nil {
							atomic.AddInt64(&errorCount, 1)
						} else {
							atomic.AddInt64(&evaluationCount, int64(len(results)))
						}
					}
				}
			}()
		}

		wg.Wait()

		t.Logf("Stress test: %d evaluations, %d errors in %v",
			evaluationCount, errorCount, duration)
		require.Zero(t, errorCount, "No errors during stress test")
	})
}
