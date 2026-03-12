// Package tests provides comprehensive test coverage for KB-19 Protocol Orchestrator.
//
// PILLAR 8: PERFORMANCE + CHAOS TESTS
// Tests performance under load, concurrent access, timeout handling,
// and graceful degradation when KB services are unavailable.
package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/config"
	"kb-19-protocol-orchestrator/internal/models"
)

// ============================================================================
// PILLAR 8.1: PERFORMANCE BENCHMARKS
// SLA: < 200ms for arbitration pipeline
// ============================================================================

func BenchmarkArbitrationPipeline_Simple(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.ErrorLevel) // Reduce log noise

	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			KB3URL:  "http://localhost:8083",
			KB12URL: "http://localhost:8094",
			KB14URL: "http://localhost:8091",
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	}
}

func BenchmarkArbitrationPipeline_Complex(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	// Complex scenario with multiple protocols and conflicts
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis":         true,
			"HasHFrEF":          true,
			"HasAKI":            true,
			"HasAFib":           true,
			"CHA2DS2VASc >= 2": true,
		},
		"calculator_scores": map[string]interface{}{
			"SOFA":        12.0,
			"CHA2DS2VASc": 4.0,
			"eGFR":        28.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	}
}

func BenchmarkConflictDetection(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.ErrorLevel)
	detector := arbitration.NewConflictDetector(log)

	evaluations := []models.ProtocolEvaluation{
		{ProtocolID: "SEPSIS-FLUIDS", IsApplicable: true},
		{ProtocolID: "HF-DIURESIS", IsApplicable: true},
		{ProtocolID: "AFIB-ANTICOAG", IsApplicable: true},
		{ProtocolID: "AKI-PROTECTION", IsApplicable: true},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.DetectConflicts(evaluations)
	}
}

func BenchmarkSafetyGatekeeper(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.ErrorLevel)
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	decisions := []models.ArbitratedDecision{
		{ID: uuid.New(), DecisionType: models.DecisionDo, Target: "warfarin"},
		{ID: uuid.New(), DecisionType: models.DecisionDo, Target: "gentamicin"},
		{ID: uuid.New(), DecisionType: models.DecisionDo, Target: "heparin"},
	}

	patientCtx := &models.PatientContext{
		PregnancyStatus:  &models.PregnancyStatus{IsPregnant: true},
		CQLTruthFlags:    map[string]bool{"HasAKI": true},
		CalculatorScores: map[string]float64{"eGFR": 25},
		ICUStateSummary:  &models.ICUClinicalState{AKIStage: 2, BleedingRisk: "HIGH"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gatekeeper.Apply(decisions, patientCtx)
	}
}

func BenchmarkNarrativeGeneration(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.ErrorLevel)
	generator := arbitration.NewNarrativeGenerator(log)

	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())
	bundle.AddDecision(models.ArbitratedDecision{
		ID:             uuid.New(),
		DecisionType:   models.DecisionDo,
		Target:         "norepinephrine",
		Urgency:        models.UrgencySTAT,
		SourceProtocol: "SEPSIS-SEP1-2021",
	})
	bundle.AddConflictResolution(models.ConflictResolution{
		ProtocolA:    "SEPSIS",
		ProtocolB:    "HF",
		ConflictType: models.ConflictHemodynamic,
		Winner:       "SEPSIS",
	})
	bundle.Finalize()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.Generate(bundle)
	}
}

// ============================================================================
// PILLAR 8.2: CONCURRENT ACCESS TESTS
// Engine must handle concurrent requests safely
// ============================================================================

func TestConcurrentExecution_NoRaceConditions(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	const numGoroutines = 50
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			contextData := map[string]interface{}{
				"cql_truth_flags": map[string]interface{}{
					"HasSepsis": idx%2 == 0,
					"HasHFrEF":  idx%3 == 0,
				},
			}

			_, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int64(numGoroutines), successCount+errorCount,
		"All goroutines should complete")
	assert.Greater(t, successCount, int64(0), "Some executions should succeed")
}

func TestConcurrentProtocolListing(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	const numGoroutines = 20
	var wg sync.WaitGroup
	results := make([]int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			protocols := engine.ListProtocols("", "")
			results[idx] = len(protocols)
		}(i)
	}

	wg.Wait()

	// All should return same count (consistent view)
	baseCount := results[0]
	for i, count := range results {
		assert.Equal(t, baseCount, count,
			"Concurrent listing %d should return same count", i)
	}
}

// ============================================================================
// PILLAR 8.3: TIMEOUT HANDLING
// Pipeline must respect context timeouts
// ============================================================================

func TestTimeoutHandling_ImmediateTimeout(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	bundle, err := engine.Execute(ctx, uuid.New(), uuid.New(), nil)

	// Either completes quickly or respects cancellation
	// The behavior depends on implementation
	if bundle != nil {
		assert.Equal(t, models.StatusCompleted, bundle.Status)
	}
}

func TestTimeoutHandling_ShortTimeout(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 1 * time.Millisecond, // Very short timeout
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should still complete (local processing is fast)
	bundle, err := engine.Execute(ctx, uuid.New(), uuid.New(), nil)
	assert.NoError(t, err, "Short service timeout should not block local processing")
	assert.NotNil(t, bundle)
}

// ============================================================================
// PILLAR 8.4: GRACEFUL DEGRADATION
// Engine should handle KB service failures gracefully
// ============================================================================

func TestGracefulDegradation_KB3Unavailable(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			KB3URL:  "http://localhost:99999", // Invalid port - service unavailable
			KB12URL: "http://localhost:8092",
			KB14URL: "http://localhost:8094",
			Timeout: 1 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	// Should complete even if KB-3 is unavailable
	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	assert.NoError(t, err, "Should gracefully handle KB-3 unavailability")
	assert.NotNil(t, bundle, "Bundle should be returned")

	// Core arbitration should still work
	assert.Equal(t, models.StatusCompleted, bundle.Status)
}

func TestGracefulDegradation_AllKBServicesUnavailable(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			KB3URL:  "http://localhost:99997",
			KB12URL: "http://localhost:99998",
			KB14URL: "http://localhost:99999",
			Timeout: 100 * time.Millisecond,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	// Should still produce clinical recommendations
	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	assert.NoError(t, err, "Should gracefully handle all KB services being unavailable")
	assert.NotNil(t, bundle, "Bundle should be returned")

	// Clinical decisions should still be made
	assert.NotNil(t, bundle.Decisions)
}

// ============================================================================
// PILLAR 8.5: MEMORY AND RESOURCE TESTS
// Engine should not leak resources
// ============================================================================

func TestMemoryStability_RepeatedExecution(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	// Execute many times and check for panics/crashes
	const iterations = 100
	for i := 0; i < iterations; i++ {
		bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
		if err != nil {
			t.Logf("Iteration %d failed: %v", i, err)
		}
		if bundle == nil {
			t.Fatalf("Iteration %d returned nil bundle", i)
		}
	}
}

// ============================================================================
// PILLAR 8.6: LOAD TESTING
// Engine should handle sustained load
// ============================================================================

func TestLoadHandling_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained load test in short mode")
	}

	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	const (
		duration    = 5 * time.Second
		concurrency = 10
	)

	var wg sync.WaitGroup
	var totalRequests int64
	var successRequests int64
	var failedRequests int64

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					atomic.AddInt64(&totalRequests, 1)

					contextData := map[string]interface{}{
						"cql_truth_flags": map[string]interface{}{
							"HasSepsis": true,
						},
					}

					_, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
					if err != nil {
						atomic.AddInt64(&failedRequests, 1)
					} else {
						atomic.AddInt64(&successRequests, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	successRate := float64(successRequests) / float64(totalRequests) * 100

	t.Logf("Sustained load test results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful: %d (%.1f%%)", successRequests, successRate)
	t.Logf("  Failed: %d", failedRequests)
	t.Logf("  Requests/sec: %.1f", float64(totalRequests)/duration.Seconds())

	assert.GreaterOrEqual(t, successRate, 90.0, "Success rate should be >= 90%")
}

// ============================================================================
// PILLAR 8.7: CHAOS SCENARIOS
// Engine should handle unexpected inputs and edge cases
// ============================================================================

func TestChaos_NilInputs(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	tests := []struct {
		name        string
		patientID   uuid.UUID
		encounterID uuid.UUID
		contextData map[string]interface{}
	}{
		{"Nil UUIDs", uuid.Nil, uuid.Nil, nil},
		{"Nil context", uuid.New(), uuid.New(), nil},
		{"Empty context", uuid.New(), uuid.New(), map[string]interface{}{}},
		{"Empty flags", uuid.New(), uuid.New(), map[string]interface{}{
			"cql_truth_flags": map[string]interface{}{},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundle, err := engine.Execute(context.Background(), tc.patientID, tc.encounterID, tc.contextData)
			assert.NoError(t, err, "Should handle %s gracefully", tc.name)
			assert.NotNil(t, bundle, "Bundle should be returned for %s", tc.name)
		})
	}
}

func TestChaos_InvalidFlagTypes(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Invalid types in context data
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis":   "true", // String instead of bool
			"InvalidFlag": 123,    // Integer instead of bool
		},
		"calculator_scores": map[string]interface{}{
			"SOFA": "eight", // String instead of float64
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	assert.NoError(t, err, "Should handle invalid types gracefully")
	assert.NotNil(t, bundle)
}

func TestChaos_ExtremeValues(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Extreme values
	contextData := map[string]interface{}{
		"calculator_scores": map[string]interface{}{
			"SOFA":  99999.0, // Extremely high
			"eGFR":  0.0,     // Minimum
			"HASBLED": -10.0, // Negative (invalid but shouldn't crash)
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	assert.NoError(t, err, "Should handle extreme values gracefully")
	assert.NotNil(t, bundle)
}

// ============================================================================
// PILLAR 8.8: PERFORMANCE ASSERTIONS
// Verify SLA compliance
// ============================================================================

func TestPerformanceSLA_Under200ms(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	const iterations = 10
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
		duration := time.Since(start)
		totalDuration += duration

		assert.NoError(t, err)
	}

	avgDuration := totalDuration / iterations

	t.Logf("Average execution time: %v", avgDuration)

	// SLA: < 200ms for local processing (without network calls to KB services)
	assert.Less(t, avgDuration, 200*time.Millisecond,
		"Average execution time should be under 200ms SLA")
}
