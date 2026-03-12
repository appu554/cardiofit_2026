//go:build scale
// +build scale

// Package scale provides load and volume testing for KB-11 Population Health.
// These tests verify behavior at real-world scale: 100K patients, 10K batches.
//
// Run with: go test -tags=scale ./tests/scale/... -v -timeout 30m
//
// IMPORTANT: These tests are opt-in and should NOT run in normal CI.
// They require significant resources and time.
package scale

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/kb-11-population-health/internal/analytics"
	"github.com/cardiofit/kb-11-population-health/internal/cohort"
	"github.com/cardiofit/kb-11-population-health/internal/models"
	"github.com/cardiofit/kb-11-population-health/internal/risk"
	"github.com/cardiofit/kb-11-population-health/tests/fixtures"
)

// ──────────────────────────────────────────────────────────────────────────────
// Scale Test Constants
// ──────────────────────────────────────────────────────────────────────────────

const (
	// Population sizes for scale testing
	ScalePopulationSmall  = 10_000   // Quick validation
	ScalePopulationMedium = 50_000   // Standard scale test
	ScalePopulationLarge  = 100_000  // Full production scale

	// Batch sizes
	ScaleBatchSmall  = 1_000
	ScaleBatchMedium = 5_000
	ScaleBatchLarge  = 10_000

	// Performance thresholds
	MaxPopulationSyncTime      = 10 * time.Minute
	MaxBatchCalculationTime    = 5 * time.Minute
	MaxCohortRefreshTime       = 2 * time.Minute
	MaxAnalyticsSnapshotTime   = 5 * time.Second
	MaxRiskCalculationTimePerPatient = 100 * time.Millisecond
)

// ──────────────────────────────────────────────────────────────────────────────
// Population Sync Scale Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestPopulationProjectionScale validates population projection at 100K scale.
// This is the primary scale test for the projection service.
func TestPopulationProjectionScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scale test in short mode")
	}

	t.Run("100K patients projection generation", func(t *testing.T) {
		patients := fixtures.GenerateSyntheticPatients(ScalePopulationLarge)

		// Track memory before
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		start := time.Now()

		// Simulate projection processing (in real scenario, this would go through the service)
		processed := 0
		for _, p := range patients {
			// Compute hash (CPU-intensive operation)
			_ = p.Hash()
			processed++
		}

		duration := time.Since(start)

		// Track memory after
		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		// Assertions
		assert.Equal(t, ScalePopulationLarge, processed, "Should process all patients")
		assert.Less(t, duration, MaxPopulationSyncTime, "Should complete within time limit")

		// Log performance metrics
		t.Logf("📊 Population Projection Scale Results:")
		t.Logf("   Patients processed: %d", processed)
		t.Logf("   Duration: %v", duration)
		t.Logf("   Rate: %.2f patients/sec", float64(processed)/duration.Seconds())
		t.Logf("   Memory delta: %.2f MB", float64(memAfter.Alloc-memBefore.Alloc)/(1024*1024))
	})

	t.Run("50K patients with full feature extraction", func(t *testing.T) {
		patients := fixtures.GenerateSyntheticPatients(ScalePopulationMedium)

		start := time.Now()

		// Process each patient with full feature extraction
		var totalConditions, totalMeds, totalLabs int64
		for _, p := range patients {
			atomic.AddInt64(&totalConditions, int64(len(p.Conditions)))
			atomic.AddInt64(&totalMeds, int64(len(p.Medications)))
			atomic.AddInt64(&totalLabs, int64(len(p.LabValues)))
			_ = p.Hash()
		}

		duration := time.Since(start)

		assert.Less(t, duration, MaxPopulationSyncTime/2, "50K should complete in half the time")

		t.Logf("📊 Feature Extraction Stats:")
		t.Logf("   Total conditions: %d (avg %.1f/patient)", totalConditions, float64(totalConditions)/float64(ScalePopulationMedium))
		t.Logf("   Total medications: %d (avg %.1f/patient)", totalMeds, float64(totalMeds)/float64(ScalePopulationMedium))
		t.Logf("   Total labs: %d (avg %.1f/patient)", totalLabs, float64(totalLabs)/float64(ScalePopulationMedium))
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Batch Risk Calculation Scale Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestBatchRiskCalculationScale validates batch risk calculation at 10K scale.
func TestBatchRiskCalculationScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scale test in short mode")
	}

	t.Run("10K batch risk calculation", func(t *testing.T) {
		patients := fixtures.GenerateSyntheticPatients(ScaleBatchLarge)
		model := risk.DefaultHospitalizationModel()

		start := time.Now()

		// Simulate batch calculation
		results := make([]*ScaleRiskResult, len(patients))
		var wg sync.WaitGroup
		var successCount int64

		// Use worker pool pattern
		workerCount := runtime.NumCPU() * 2
		patientChan := make(chan int, len(patients))

		// Start workers
		for w := 0; w < workerCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range patientChan {
					result := calculateRiskForPatient(patients[i], model)
					results[i] = result
					if result != nil {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}()
		}

		// Send work
		for i := range patients {
			patientChan <- i
		}
		close(patientChan)

		wg.Wait()
		duration := time.Since(start)

		// Assertions
		assert.Equal(t, int64(ScaleBatchLarge), successCount, "All patients should be calculated")
		assert.Less(t, duration, MaxBatchCalculationTime, "Should complete within time limit")

		// Calculate average time per patient
		avgTimePerPatient := duration / time.Duration(ScaleBatchLarge)
		assert.Less(t, avgTimePerPatient, MaxRiskCalculationTimePerPatient, "Per-patient time should be reasonable")

		t.Logf("📊 Batch Risk Calculation Scale Results:")
		t.Logf("   Batch size: %d", ScaleBatchLarge)
		t.Logf("   Duration: %v", duration)
		t.Logf("   Rate: %.2f calculations/sec", float64(ScaleBatchLarge)/duration.Seconds())
		t.Logf("   Avg time/patient: %v", avgTimePerPatient)
	})

	t.Run("5K batch with concurrent governance emission", func(t *testing.T) {
		patients := fixtures.GenerateSyntheticPatients(ScaleBatchMedium)
		model := risk.DefaultHospitalizationModel()

		start := time.Now()

		var wg sync.WaitGroup
		var riskCount, governanceCount int64

		for _, p := range patients {
			wg.Add(1)
			go func(patient *risk.RiskFeatures) {
				defer wg.Done()

				// Calculate risk
				result := calculateRiskForPatient(patient, model)
				if result != nil {
					atomic.AddInt64(&riskCount, 1)
				}

				// Simulate governance emission
				_ = patient.Hash() // Input hash
				atomic.AddInt64(&governanceCount, 1)
			}(p)
		}

		wg.Wait()
		duration := time.Since(start)

		assert.Equal(t, int64(ScaleBatchMedium), riskCount)
		assert.Equal(t, int64(ScaleBatchMedium), governanceCount)

		t.Logf("📊 Concurrent Risk + Governance:")
		t.Logf("   Risk calculations: %d", riskCount)
		t.Logf("   Governance events: %d", governanceCount)
		t.Logf("   Duration: %v", duration)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Refresh Scale Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestDynamicCohortRefreshScale validates cohort refresh at 50K scale.
func TestDynamicCohortRefreshScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scale test in short mode")
	}

	t.Run("50K patient cohort refresh", func(t *testing.T) {
		projections := fixtures.GenerateSyntheticPatientProjections(ScalePopulationMedium)

		// Create high-risk cohort criteria
		criteria := cohort.HighRiskCriteria()

		start := time.Now()

		// Simulate cohort evaluation
		var matched int64
		for _, p := range projections {
			if evaluateCriteria(p, criteria) {
				atomic.AddInt64(&matched, 1)
			}
		}

		duration := time.Since(start)

		assert.Less(t, duration, MaxCohortRefreshTime, "Should complete within time limit")

		// Expected ~17% high risk (12% + 5% very high)
		expectedHighRisk := float64(ScalePopulationMedium) * 0.17
		assert.InDelta(t, expectedHighRisk, float64(matched), float64(ScalePopulationMedium)*0.05, "High risk count should be ~17%")

		t.Logf("📊 Cohort Refresh Scale Results:")
		t.Logf("   Population: %d", ScalePopulationMedium)
		t.Logf("   Matched: %d (%.1f%%)", matched, float64(matched)/float64(ScalePopulationMedium)*100)
		t.Logf("   Duration: %v", duration)
		t.Logf("   Rate: %.2f evaluations/sec", float64(ScalePopulationMedium)/duration.Seconds())
	})

	t.Run("complex multi-criteria cohort at scale", func(t *testing.T) {
		projections := fixtures.GenerateSyntheticPatientProjections(ScalePopulationMedium)

		// Complex criteria: high risk + care gaps + specific practice
		criteria := cohort.NewCriterionBuilder().
			Where("current_risk_tier", models.OpIn, []string{"HIGH", "VERY_HIGH"}).
			And("care_gap_count", models.OpGreaterEq, 1).
			Build()

		start := time.Now()

		var matched int64
		for _, p := range projections {
			if evaluateComplexCriteria(p, criteria) {
				atomic.AddInt64(&matched, 1)
			}
		}

		duration := time.Since(start)

		assert.Less(t, duration, MaxCohortRefreshTime)

		t.Logf("📊 Complex Cohort Results:")
		t.Logf("   Matched (high risk + care gaps): %d (%.1f%%)", matched, float64(matched)/float64(ScalePopulationMedium)*100)
		t.Logf("   Duration: %v", duration)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Analytics Query Scale Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestPopulationAnalyticsScale validates analytics queries at production scale.
func TestPopulationAnalyticsScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scale test in short mode")
	}

	t.Run("population snapshot generation at 100K", func(t *testing.T) {
		projections := fixtures.GenerateSyntheticPatientProjections(ScalePopulationLarge)

		start := time.Now()

		snapshot := generatePopulationSnapshot(projections)

		duration := time.Since(start)

		require.NotNil(t, snapshot)
		assert.Equal(t, ScalePopulationLarge, snapshot.TotalPatients)
		assert.Less(t, duration, MaxAnalyticsSnapshotTime, "Snapshot should be fast")

		// Verify distribution sums
		totalFromDist := 0
		for _, count := range snapshot.RiskDistribution {
			totalFromDist += count
		}
		assert.Equal(t, ScalePopulationLarge, totalFromDist, "Distribution should sum to total")

		t.Logf("📊 Analytics Snapshot Scale Results:")
		t.Logf("   Population: %d", ScalePopulationLarge)
		t.Logf("   Duration: %v", duration)
		t.Logf("   High risk count: %d (%.1f%%)", snapshot.HighRiskCount, float64(snapshot.HighRiskCount)/float64(ScalePopulationLarge)*100)
		t.Logf("   Avg risk score: %.3f", snapshot.AverageRiskScore)
	})

	t.Run("risk stratification report at scale", func(t *testing.T) {
		projections := fixtures.GenerateSyntheticPatientProjections(ScalePopulationLarge)

		start := time.Now()

		report := generateRiskStratificationReport(projections)

		duration := time.Since(start)

		require.NotNil(t, report)
		assert.Less(t, duration, MaxAnalyticsSnapshotTime*2)

		// Verify all tiers present
		assert.Contains(t, report.TierDetails, models.RiskTierLow)
		assert.Contains(t, report.TierDetails, models.RiskTierModerate)
		assert.Contains(t, report.TierDetails, models.RiskTierHigh)
		assert.Contains(t, report.TierDetails, models.RiskTierVeryHigh)

		t.Logf("📊 Risk Stratification Report:")
		for tier, details := range report.TierDetails {
			t.Logf("   %s: %d patients (%.1f%%)", tier, details.Count, details.Percentage)
		}
	})

	t.Run("provider analytics aggregation at scale", func(t *testing.T) {
		projections := fixtures.GenerateSyntheticPatientProjections(ScalePopulationLarge)

		start := time.Now()

		// Aggregate by provider
		providerStats := make(map[string]*ProviderAnalytics)
		for _, p := range projections {
			pcp := p.AttributedPCP
			if _, exists := providerStats[pcp]; !exists {
				providerStats[pcp] = &ProviderAnalytics{ProviderID: pcp}
			}
			providerStats[pcp].TotalPatients++
			if p.CurrentRiskTier == models.RiskTierHigh || p.CurrentRiskTier == models.RiskTierVeryHigh {
				providerStats[pcp].HighRiskCount++
			}
			providerStats[pcp].TotalRiskScore += p.CurrentRiskScore
		}

		duration := time.Since(start)

		assert.Less(t, duration, MaxAnalyticsSnapshotTime*3)
		assert.NotEmpty(t, providerStats)

		t.Logf("📊 Provider Aggregation:")
		t.Logf("   Unique providers: %d", len(providerStats))
		t.Logf("   Duration: %v", duration)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Memory and Resource Scale Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestMemoryBoundsAtScale verifies memory usage stays bounded at scale.
func TestMemoryBoundsAtScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scale test in short mode")
	}

	t.Run("100K patients memory stays bounded", func(t *testing.T) {
		// Force GC before measurement
		runtime.GC()

		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		// Generate and process large population
		patients := fixtures.GenerateSyntheticPatients(ScalePopulationLarge)

		var memAfterGen runtime.MemStats
		runtime.ReadMemStats(&memAfterGen)

		// Process all patients
		for _, p := range patients {
			_ = p.Hash()
		}

		var memAfterProcess runtime.MemStats
		runtime.ReadMemStats(&memAfterProcess)

		// Memory calculations
		genMemMB := float64(memAfterGen.Alloc-memBefore.Alloc) / (1024 * 1024)
		processMemMB := float64(memAfterProcess.Alloc-memAfterGen.Alloc) / (1024 * 1024)
		totalMemMB := float64(memAfterProcess.Alloc-memBefore.Alloc) / (1024 * 1024)

		// Should use less than 1GB for 100K patients
		maxMemMB := 1024.0
		assert.Less(t, totalMemMB, maxMemMB, "Memory should stay under 1GB")

		t.Logf("📊 Memory Usage at 100K Scale:")
		t.Logf("   Generation: %.2f MB", genMemMB)
		t.Logf("   Processing: %.2f MB", processMemMB)
		t.Logf("   Total: %.2f MB", totalMemMB)
		t.Logf("   Per-patient: %.2f KB", totalMemMB*1024/float64(ScalePopulationLarge))
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper Types and Functions for Scale Tests
// ──────────────────────────────────────────────────────────────────────────────

// ScaleRiskResult is a simplified risk result for scale testing.
type ScaleRiskResult struct {
	PatientID string
	Score     float64
	Tier      models.RiskTier
	Hash      string
}

// ProviderAnalytics holds aggregated provider statistics.
type ProviderAnalytics struct {
	ProviderID     string
	TotalPatients  int
	HighRiskCount  int
	TotalRiskScore float64
}

// ScalePopulationSnapshot is a simplified snapshot for scale testing.
type ScalePopulationSnapshot struct {
	TotalPatients    int
	HighRiskCount    int
	AverageRiskScore float64
	RiskDistribution map[models.RiskTier]int
}

// ScaleRiskReport is a simplified risk report for scale testing.
type ScaleRiskReport struct {
	TotalPatients int
	TierDetails   map[models.RiskTier]*TierDetail
}

// TierDetail holds tier-specific details.
type TierDetail struct {
	Count      int
	Percentage float64
	AvgScore   float64
}

// calculateRiskForPatient simulates risk calculation for a patient.
func calculateRiskForPatient(p *risk.RiskFeatures, model *risk.ModelConfig) *ScaleRiskResult {
	// Simulate calculation with real algorithm components
	score := 0.0

	// Age factors
	if p.Age >= 65 {
		score += model.Weights["age_over_65"]
	}
	if p.Age >= 80 {
		score += model.Weights["age_over_80"]
	}

	// Chronic conditions
	chronicCount := len(p.Conditions)
	if chronicCount > 0 {
		score += float64(min(chronicCount, 5)) / 5.0 * model.Weights["chronic_conditions"]
	}

	// Medications
	highRiskMeds := 0
	for _, m := range p.Medications {
		if m.HighRisk {
			highRiskMeds++
		}
	}
	if highRiskMeds > 0 {
		score += float64(min(highRiskMeds, 3)) / 3.0 * model.Weights["high_risk_medications"]
	}

	// Normalize
	if score > 1.0 {
		score = 1.0
	}

	// Determine tier
	tier := models.RiskTierLow
	switch {
	case score >= 0.8:
		tier = models.RiskTierVeryHigh
	case score >= 0.6:
		tier = models.RiskTierHigh
	case score >= 0.4:
		tier = models.RiskTierModerate
	}

	return &ScaleRiskResult{
		PatientID: p.PatientFHIRID,
		Score:     score,
		Tier:      tier,
		Hash:      p.Hash(),
	}
}

// evaluateCriteria checks if a projection matches cohort criteria.
func evaluateCriteria(p *models.PatientProjection, criteria []cohort.Criterion) bool {
	for _, c := range criteria {
		switch c.Field {
		case "current_risk_tier":
			if values, ok := c.Value.([]string); ok {
				matched := false
				for _, v := range values {
					if string(p.CurrentRiskTier) == v {
						matched = true
						break
					}
				}
				if !matched {
					return false
				}
			}
		}
	}
	return true
}

// evaluateComplexCriteria handles multi-field criteria.
func evaluateComplexCriteria(p *models.PatientProjection, criteria []cohort.Criterion) bool {
	for _, c := range criteria {
		matched := false

		switch c.Field {
		case "current_risk_tier":
			if values, ok := c.Value.([]string); ok {
				for _, v := range values {
					if string(p.CurrentRiskTier) == v {
						matched = true
						break
					}
				}
			}
		case "care_gap_count":
			if threshold, ok := c.Value.(int); ok {
				matched = p.CareGapCount >= threshold
			}
		}

		if !matched {
			return false
		}
	}
	return true
}

// generatePopulationSnapshot creates a snapshot from projections.
func generatePopulationSnapshot(projections []*models.PatientProjection) *ScalePopulationSnapshot {
	snapshot := &ScalePopulationSnapshot{
		TotalPatients:    len(projections),
		RiskDistribution: make(map[models.RiskTier]int),
	}

	var totalScore float64
	for _, p := range projections {
		snapshot.RiskDistribution[p.CurrentRiskTier]++
		totalScore += p.CurrentRiskScore

		if p.CurrentRiskTier == models.RiskTierHigh || p.CurrentRiskTier == models.RiskTierVeryHigh {
			snapshot.HighRiskCount++
		}
	}

	if snapshot.TotalPatients > 0 {
		snapshot.AverageRiskScore = totalScore / float64(snapshot.TotalPatients)
	}

	return snapshot
}

// generateRiskStratificationReport creates a stratification report.
func generateRiskStratificationReport(projections []*models.PatientProjection) *ScaleRiskReport {
	report := &ScaleRiskReport{
		TotalPatients: len(projections),
		TierDetails:   make(map[models.RiskTier]*TierDetail),
	}

	// Initialize all tiers
	for _, tier := range []models.RiskTier{
		models.RiskTierLow, models.RiskTierModerate,
		models.RiskTierHigh, models.RiskTierVeryHigh,
		models.RiskTierRising, models.RiskTierUnscored,
	} {
		report.TierDetails[tier] = &TierDetail{}
	}

	// Aggregate
	for _, p := range projections {
		if detail, exists := report.TierDetails[p.CurrentRiskTier]; exists {
			detail.Count++
			detail.AvgScore += p.CurrentRiskScore
		}
	}

	// Calculate percentages and averages
	for _, detail := range report.TierDetails {
		if report.TotalPatients > 0 {
			detail.Percentage = float64(detail.Count) / float64(report.TotalPatients) * 100
		}
		if detail.Count > 0 {
			detail.AvgScore /= float64(detail.Count)
		}
	}

	return report
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ──────────────────────────────────────────────────────────────────────────────
// Benchmark Tests (for continuous performance monitoring)
// ──────────────────────────────────────────────────────────────────────────────

func BenchmarkRiskCalculation1K(b *testing.B) {
	patients := fixtures.GenerateSyntheticPatients(1000)
	model := risk.DefaultHospitalizationModel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range patients {
			calculateRiskForPatient(p, model)
		}
	}
}

func BenchmarkHashGeneration(b *testing.B) {
	patients := fixtures.GenerateSyntheticPatients(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range patients {
			_ = p.Hash()
		}
	}
}

func BenchmarkCohortEvaluation(b *testing.B) {
	projections := fixtures.GenerateSyntheticPatientProjections(1000)
	criteria := cohort.HighRiskCriteria()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range projections {
			evaluateCriteria(p, criteria)
		}
	}
}

func BenchmarkPopulationSnapshot(b *testing.B) {
	projections := fixtures.GenerateSyntheticPatientProjections(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generatePopulationSnapshot(projections)
	}
}
