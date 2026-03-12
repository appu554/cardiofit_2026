// Package integration provides integration tests for KB-11 Population Health Engine.
package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cardiofit/kb-11-population-health/internal/analytics"
	"github.com/cardiofit/kb-11-population-health/internal/clients"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// ──────────────────────────────────────────────────────────────────────────────
// Population Snapshot Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestPopulationSnapshotStructure verifies population snapshot data structure.
func TestPopulationSnapshotStructure(t *testing.T) {
	t.Run("snapshot with complete data", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		assert.Equal(t, 10000, snapshot.TotalPatients)
		assert.Equal(t, 1700, snapshot.HighRiskCount)
		assert.Equal(t, 350, snapshot.RisingRiskCount)
		assert.InDelta(t, 0.42, snapshot.AverageRiskScore, 0.01)
		assert.NotEmpty(t, snapshot.RiskPercentages)
	})

	t.Run("risk percentages sum to 100", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		var total float64
		for _, pct := range snapshot.RiskPercentages {
			total += pct
		}

		assert.InDelta(t, 100.0, total, 0.1)
	})

	t.Run("high risk count matches distribution", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		highRiskPct := snapshot.RiskPercentages[models.RiskTierHigh]
		veryHighRiskPct := snapshot.RiskPercentages[models.RiskTierVeryHigh]
		expectedHighRisk := int(float64(snapshot.TotalPatients) * (highRiskPct + veryHighRiskPct) / 100)

		assert.InDelta(t, expectedHighRisk, snapshot.HighRiskCount, 50)
	})
}

// TestPopulationSnapshotCareGapMetrics verifies care gap metrics integration.
func TestPopulationSnapshotCareGapMetrics(t *testing.T) {
	t.Run("care gap metrics present", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		assert.NotNil(t, snapshot.CareGapMetrics)
		assert.Equal(t, 4500, snapshot.CareGapMetrics.TotalOpenGaps)
		assert.Equal(t, 2800, snapshot.CareGapMetrics.PatientsWithGaps)
		assert.InDelta(t, 1.6, snapshot.CareGapMetrics.AverageGapsPerPatient, 0.1)
	})

	t.Run("average gaps per patient calculation", func(t *testing.T) {
		metrics := &analytics.CareGapSnapshot{
			TotalOpenGaps:    1000,
			PatientsWithGaps: 500,
		}

		expected := float64(metrics.TotalOpenGaps) / float64(metrics.PatientsWithGaps)
		metrics.AverageGapsPerPatient = expected

		assert.InDelta(t, 2.0, metrics.AverageGapsPerPatient, 0.01)
	})
}

// TestPopulationSnapshotAttribution verifies attribution statistics.
func TestPopulationSnapshotAttribution(t *testing.T) {
	t.Run("attribution stats present", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		assert.NotNil(t, snapshot.AttributionStats)
		assert.Equal(t, 85, snapshot.AttributionStats.TotalPCPs)
		assert.Equal(t, 12, snapshot.AttributionStats.TotalPractices)
		assert.Equal(t, 450, snapshot.AttributionStats.UnattributedCount)
	})

	t.Run("unattributed percentage calculation", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		unattributedPct := float64(snapshot.AttributionStats.UnattributedCount) / float64(snapshot.TotalPatients) * 100
		assert.InDelta(t, 4.5, unattributedPct, 0.1)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Risk Stratification Report Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestRiskStratificationReport verifies risk stratification report structure.
func TestRiskStratificationReport(t *testing.T) {
	t.Run("report has all risk tiers", func(t *testing.T) {
		report := createTestRiskStratificationReport()

		expectedTiers := []models.RiskTier{
			models.RiskTierLow,
			models.RiskTierModerate,
			models.RiskTierHigh,
			models.RiskTierVeryHigh,
			models.RiskTierRising,
		}

		for _, tier := range expectedTiers {
			_, exists := report.Distribution[tier]
			assert.True(t, exists, "Distribution should contain tier %s", tier)
		}
	})

	t.Run("distribution counts match total", func(t *testing.T) {
		report := createTestRiskStratificationReport()

		var total int
		for _, details := range report.Distribution {
			total += details.Count
		}

		assert.Equal(t, 10000, total)
	})

	t.Run("percentages sum to 100", func(t *testing.T) {
		report := createTestRiskStratificationReport()

		var totalPct float64
		for _, details := range report.Distribution {
			totalPct += details.Percentage
		}

		assert.InDelta(t, 100.0, totalPct, 0.1)
	})
}

// TestTierDetails verifies individual tier details data.
func TestTierDetails(t *testing.T) {
	t.Run("tier details has required fields", func(t *testing.T) {
		details := &analytics.TierDetails{
			Tier:          models.RiskTierHigh,
			Count:         1500,
			Percentage:    15.0,
			AverageScore:  0.75,
			AverageAge:    68.5,
			TopConditions: []string{"diabetes", "heart_failure"},
		}

		assert.Equal(t, models.RiskTierHigh, details.Tier)
		assert.Equal(t, 1500, details.Count)
		assert.InDelta(t, 15.0, details.Percentage, 0.01)
		assert.InDelta(t, 0.75, details.AverageScore, 0.01)
		assert.InDelta(t, 68.5, details.AverageAge, 0.1)
		assert.Len(t, details.TopConditions, 2)
	})
}

// TestHighRiskBreakdown verifies high-risk patient breakdown.
func TestHighRiskBreakdown(t *testing.T) {
	t.Run("breakdown by contributing factors", func(t *testing.T) {
		report := createTestRiskStratificationReport()

		assert.NotNil(t, report.HighRiskBreakdown)
		assert.NotEmpty(t, report.HighRiskBreakdown.ByConditionCount)
		assert.NotEmpty(t, report.HighRiskBreakdown.ByAge)
	})

	t.Run("condition breakdown sums correctly", func(t *testing.T) {
		breakdown := &analytics.HighRiskBreakdown{
			TotalHighRisk: 1500,
			ByConditionCount: map[string]int{
				"diabetes":      450,
				"heart_failure": 380,
				"copd":          320,
				"ckd":           350,
			},
		}

		var total int
		for _, count := range breakdown.ByConditionCount {
			total += count
		}

		assert.Equal(t, 1500, total)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Provider Analytics Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestProviderPanelAnalytics verifies provider panel analytics structure.
func TestProviderPanelAnalytics(t *testing.T) {
	t.Run("provider analytics with all fields", func(t *testing.T) {
		providerAnalytics := createTestProviderPanelAnalytics("provider-123")

		assert.Equal(t, "provider-123", providerAnalytics.ProviderID)
		assert.Equal(t, "Dr. Smith", providerAnalytics.ProviderName)
		assert.Equal(t, 250, providerAnalytics.PanelSize)
		assert.Equal(t, 35, providerAnalytics.HighRiskCount)
		assert.Equal(t, 12, providerAnalytics.RisingRiskCount)
		assert.InDelta(t, 0.38, providerAnalytics.AverageRiskScore, 0.01)
	})

	t.Run("compared to average is present", func(t *testing.T) {
		providerAnalytics := createTestProviderPanelAnalytics("provider-123")

		assert.NotNil(t, providerAnalytics.ComparedToAverage)
		assert.Greater(t, providerAnalytics.ComparedToAverage.RiskScoreDiff, 0.0)
	})

	t.Run("high risk percentage calculation", func(t *testing.T) {
		providerAnalytics := createTestProviderPanelAnalytics("provider-123")

		highRiskPct := float64(providerAnalytics.HighRiskCount) / float64(providerAnalytics.PanelSize) * 100
		assert.InDelta(t, 14.0, highRiskPct, 0.5)
	})
}

// TestProviderComparison verifies provider comparison functionality.
func TestProviderComparison(t *testing.T) {
	t.Run("compare two providers", func(t *testing.T) {
		provider1 := createTestProviderPanelAnalytics("provider-1")
		provider2 := createTestProviderPanelAnalytics("provider-2")
		provider2.AverageRiskScore = 0.45
		provider2.HighRiskCount = 45

		// Provider 2 has higher risk
		assert.Greater(t, provider2.AverageRiskScore, provider1.AverageRiskScore)
		assert.Greater(t, provider2.HighRiskCount, provider1.HighRiskCount)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Practice Analytics Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestPracticeAnalytics verifies practice-level analytics structure.
func TestPracticeAnalytics(t *testing.T) {
	t.Run("practice analytics with all fields", func(t *testing.T) {
		practiceAnalytics := createTestPracticeAnalytics("practice-456")

		assert.Equal(t, "practice-456", practiceAnalytics.PracticeID)
		assert.Equal(t, "Cardiology Associates", practiceAnalytics.PracticeName)
		assert.Equal(t, 8, practiceAnalytics.ProviderCount)
		assert.Equal(t, 2000, practiceAnalytics.TotalPatients)
		assert.Equal(t, 280, practiceAnalytics.HighRiskCount)
		assert.InDelta(t, 0.40, practiceAnalytics.AverageRiskScore, 0.01)
	})

	t.Run("average panel size calculation", func(t *testing.T) {
		practiceAnalytics := createTestPracticeAnalytics("practice-456")

		avgPanelSize := practiceAnalytics.TotalPatients / practiceAnalytics.ProviderCount
		assert.Equal(t, 250, avgPanelSize)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Population Filter Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestPopulationFilter verifies population filter structure and defaults.
func TestPopulationFilter(t *testing.T) {
	t.Run("empty filter returns all", func(t *testing.T) {
		filter := &analytics.PopulationFilter{}

		assert.Empty(t, filter.Practice)
		assert.Empty(t, filter.PCP)
		assert.Zero(t, filter.MinAge)
		assert.Zero(t, filter.MaxAge)
		assert.False(t, filter.WithCareGaps)
		assert.Empty(t, filter.RiskTiers)
	})

	t.Run("filter by practice", func(t *testing.T) {
		filter := &analytics.PopulationFilter{
			Practice: "cardiology-associates",
		}

		assert.Equal(t, "cardiology-associates", filter.Practice)
	})

	t.Run("filter by age range", func(t *testing.T) {
		filter := &analytics.PopulationFilter{
			MinAge: 65,
			MaxAge: 85,
		}

		assert.Equal(t, 65, filter.MinAge)
		assert.Equal(t, 85, filter.MaxAge)
	})

	t.Run("filter by risk tiers", func(t *testing.T) {
		filter := &analytics.PopulationFilter{
			RiskTiers: []models.RiskTier{
				models.RiskTierHigh,
				models.RiskTierVeryHigh,
			},
		}

		assert.Len(t, filter.RiskTiers, 2)
		assert.Contains(t, filter.RiskTiers, models.RiskTierHigh)
		assert.Contains(t, filter.RiskTiers, models.RiskTierVeryHigh)
	})

	t.Run("filter with care gaps", func(t *testing.T) {
		filter := &analytics.PopulationFilter{
			WithCareGaps: true,
		}

		assert.True(t, filter.WithCareGaps)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// KB-13 Care Gap Client Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCareGapClientStructures verifies KB-13 client data structures.
func TestCareGapClientStructures(t *testing.T) {
	t.Run("patient care gap summary", func(t *testing.T) {
		summary := &clients.PatientCareGapSummary{
			PatientFHIRID:   "patient-123",
			TotalGapCount:   3,
			OpenGapCount:    2,
			ClosedGapCount:  1,
			OverdueGapCount: 1,
			CriticalGaps:    1,
			LastUpdated:     time.Now().UTC(),
		}

		assert.Equal(t, "patient-123", summary.PatientFHIRID)
		assert.Equal(t, 3, summary.TotalGapCount)
		assert.Equal(t, 2, summary.OpenGapCount)
		assert.Equal(t, 1, summary.ClosedGapCount)
		assert.Equal(t, 1, summary.OverdueGapCount)
		assert.Equal(t, 1, summary.CriticalGaps)
	})

	t.Run("population care gap metrics", func(t *testing.T) {
		metrics := &clients.PopulationCareGapMetrics{
			TotalPatients:         10000,
			PatientsWithGaps:      2800,
			TotalOpenGaps:         4500,
			AverageGapsPerPatient: 1.6,
			GapsByCategory: map[string]int{
				"preventive": 850,
				"chronic":    720,
				"medication": 680,
			},
			CalculatedAt: time.Now().UTC(),
		}

		assert.Equal(t, 10000, metrics.TotalPatients)
		assert.Equal(t, 2800, metrics.PatientsWithGaps)
		assert.Equal(t, 4500, metrics.TotalOpenGaps)
		assert.InDelta(t, 1.6, metrics.AverageGapsPerPatient, 0.01)
		assert.Len(t, metrics.GapsByCategory, 3)
	})

	t.Run("care gap trends", func(t *testing.T) {
		trend := &clients.CareGapTrend{
			Period:      "2024-01",
			OpenGaps:    2800,
			ClosedGaps:  1700,
			NewGaps:     500,
			ClosureRate: 0.378,
		}

		assert.Equal(t, "2024-01", trend.Period)
		assert.Equal(t, 2800, trend.OpenGaps)
		assert.Equal(t, 1700, trend.ClosedGaps)
		assert.InDelta(t, 0.378, trend.ClosureRate, 0.001)
	})
}

// TestCareGapFilter verifies care gap filter structure.
func TestCareGapFilter(t *testing.T) {
	t.Run("filter by practice", func(t *testing.T) {
		filter := &clients.CareGapFilter{
			Practice: "practice-123",
		}

		assert.Equal(t, "practice-123", filter.Practice)
	})

	t.Run("filter by category", func(t *testing.T) {
		filter := &clients.CareGapFilter{
			Category: "preventive",
		}

		assert.Equal(t, "preventive", filter.Category)
	})

	t.Run("filter by priority", func(t *testing.T) {
		filter := &clients.CareGapFilter{
			Priority: "high",
		}

		assert.Equal(t, "high", filter.Priority)
	})

	t.Run("filter by PCP", func(t *testing.T) {
		filter := &clients.CareGapFilter{
			PCP: "provider-456",
		}

		assert.Equal(t, "provider-456", filter.PCP)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Dashboard Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestExecutiveDashboard verifies executive dashboard data structure.
func TestExecutiveDashboard(t *testing.T) {
	t.Run("executive dashboard has key metrics", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		// Verify executive-level metrics exist
		assert.Greater(t, snapshot.TotalPatients, 0)
		assert.Greater(t, snapshot.HighRiskCount, 0)
		assert.Greater(t, snapshot.RisingRiskCount, 0)
		assert.Greater(t, snapshot.AverageRiskScore, 0.0)
		assert.NotEmpty(t, snapshot.RiskPercentages)
	})

	t.Run("high risk percentage calculation", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		// HighRiskCount = 1700, TotalPatients = 10000 → 17%
		highRiskPct := float64(snapshot.HighRiskCount) / float64(snapshot.TotalPatients) * 100
		assert.InDelta(t, 17.0, highRiskPct, 0.5)
	})
}

// TestCareManagerDashboard verifies care manager dashboard data structure.
func TestCareManagerDashboard(t *testing.T) {
	t.Run("care manager dashboard has actionable counts", func(t *testing.T) {
		report := createTestRiskStratificationReport()

		// Calculate actionable patient count (high + very_high + rising)
		var actionable int
		if d, ok := report.Distribution[models.RiskTierHigh]; ok {
			actionable += d.Count
		}
		if d, ok := report.Distribution[models.RiskTierVeryHigh]; ok {
			actionable += d.Count
		}
		if d, ok := report.Distribution[models.RiskTierRising]; ok {
			actionable += d.Count
		}

		assert.Greater(t, actionable, 0)
		assert.Equal(t, 2000, actionable) // 1000 + 500 + 500
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Caching Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestAnalyticsCacheKey verifies cache key generation.
func TestAnalyticsCacheKey(t *testing.T) {
	t.Run("different filters produce different keys", func(t *testing.T) {
		filter1 := &analytics.PopulationFilter{Practice: "practice-1"}
		filter2 := &analytics.PopulationFilter{Practice: "practice-2"}

		key1 := generateCacheKey("snapshot", filter1)
		key2 := generateCacheKey("snapshot", filter2)

		assert.NotEqual(t, key1, key2)
	})

	t.Run("same filter produces same key", func(t *testing.T) {
		filter := &analytics.PopulationFilter{
			Practice: "practice-1",
			MinAge:   65,
		}

		key1 := generateCacheKey("snapshot", filter)
		key2 := generateCacheKey("snapshot", filter)

		assert.Equal(t, key1, key2)
	})

	t.Run("nil filter produces consistent key", func(t *testing.T) {
		key1 := generateCacheKey("snapshot", nil)
		key2 := generateCacheKey("snapshot", nil)

		assert.Equal(t, key1, key2)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Test Helpers
// ──────────────────────────────────────────────────────────────────────────────

func createTestPopulationSnapshot() *analytics.PopulationSnapshot {
	return &analytics.PopulationSnapshot{
		TotalPatients:    10000,
		ActivePatients:   10000,
		HighRiskCount:    1700, // High 1200 + VeryHigh 500 = 1700
		RisingRiskCount:  350,
		AverageRiskScore: 0.42,
		RiskDistribution: map[models.RiskTier]int{
			models.RiskTierLow:      4500,
			models.RiskTierModerate: 3000,
			models.RiskTierHigh:     1200,
			models.RiskTierVeryHigh: 500,
			models.RiskTierRising:   350,
			models.RiskTierUnscored: 450,
		},
		RiskPercentages: map[models.RiskTier]float64{
			models.RiskTierLow:      45.0,
			models.RiskTierModerate: 30.0,
			models.RiskTierHigh:     12.0,
			models.RiskTierVeryHigh: 5.0,
			models.RiskTierRising:   3.5,
			models.RiskTierUnscored: 4.5,
		},
		CareGapMetrics: &analytics.CareGapSnapshot{
			TotalOpenGaps:         4500,
			PatientsWithGaps:      2800,
			AverageGapsPerPatient: 1.6,
		},
		AttributionStats: &analytics.AttributionSnapshot{
			TotalPCPs:         85,
			TotalPractices:    12,
			UnattributedCount: 450,
		},
		CalculatedAt: time.Now().UTC(),
	}
}

func createTestRiskStratificationReport() *analytics.RiskStratificationReport {
	return &analytics.RiskStratificationReport{
		Distribution: map[models.RiskTier]*analytics.TierDetails{
			models.RiskTierLow: {
				Tier:         models.RiskTierLow,
				Count:        4500,
				Percentage:   45.0,
				AverageScore: 0.15,
				AverageAge:   55.0,
			},
			models.RiskTierModerate: {
				Tier:         models.RiskTierModerate,
				Count:        3000,
				Percentage:   30.0,
				AverageScore: 0.45,
				AverageAge:   62.0,
			},
			models.RiskTierHigh: {
				Tier:         models.RiskTierHigh,
				Count:        1000,
				Percentage:   10.0,
				AverageScore: 0.72,
				AverageAge:   68.0,
			},
			models.RiskTierVeryHigh: {
				Tier:         models.RiskTierVeryHigh,
				Count:        500,
				Percentage:   5.0,
				AverageScore: 0.88,
				AverageAge:   72.0,
			},
			models.RiskTierRising: {
				Tier:         models.RiskTierRising,
				Count:        500,
				Percentage:   5.0,
				AverageScore: 0.55,
				AverageAge:   60.0,
			},
			models.RiskTierUnscored: {
				Tier:         models.RiskTierUnscored,
				Count:        500,
				Percentage:   5.0,
				AverageScore: 0.0,
				AverageAge:   45.0,
			},
		},
		RisingRiskPatients: []analytics.RisingRiskSummary{
			{
				PatientFHIRID: "patient-001",
				CurrentScore:  0.65,
				PreviousScore: 0.45,
				RisingRate:    0.20,
				DaysRising:    14,
				AttributedPCP: "provider-001",
			},
		},
		HighRiskBreakdown: &analytics.HighRiskBreakdown{
			TotalHighRisk: 1500,
			ByConditionCount: map[string]int{
				"diabetes":      450,
				"heart_failure": 350,
				"copd":          300,
				"ckd":           400,
			},
			ByAge: map[string]int{
				"18-44": 100,
				"45-64": 400,
				"65-74": 500,
				"75+":   500,
			},
			WithRecentAdmit:   150,
			WithCareGaps:      320,
			AverageConditions: 3.2,
		},
		ReportDate: time.Now().UTC(),
	}
}

func createTestProviderPanelAnalytics(providerID string) *analytics.ProviderPanelAnalytics {
	return &analytics.ProviderPanelAnalytics{
		ProviderID:       providerID,
		ProviderName:     "Dr. Smith",
		PanelSize:        250,
		HighRiskCount:    35,
		RisingRiskCount:  12,
		AverageRiskScore: 0.38,
		ComparedToAverage: &analytics.ComparisonMetrics{
			HighRiskPercentDiff: 2.0,
			RiskScoreDiff:       0.05,
			Percentile:          75,
		},
		RiskDistribution: map[models.RiskTier]int{
			models.RiskTierLow:      110,
			models.RiskTierModerate: 78,
			models.RiskTierHigh:     30,
			models.RiskTierVeryHigh: 5,
			models.RiskTierRising:   12,
			models.RiskTierUnscored: 15,
		},
		CalculatedAt: time.Now().UTC(),
	}
}

func createTestPracticeAnalytics(practiceID string) *analytics.PracticeAnalytics {
	return &analytics.PracticeAnalytics{
		PracticeID:       practiceID,
		PracticeName:     "Cardiology Associates",
		ProviderCount:    8,
		TotalPatients:    2000,
		HighRiskCount:    280,
		AverageRiskScore: 0.40,
		RiskDistribution: map[models.RiskTier]int{
			models.RiskTierLow:      880,
			models.RiskTierModerate: 625,
			models.RiskTierHigh:     240,
			models.RiskTierVeryHigh: 40,
			models.RiskTierRising:   95,
			models.RiskTierUnscored: 120,
		},
		CalculatedAt: time.Now().UTC(),
	}
}

// generateCacheKey creates a cache key for analytics data.
func generateCacheKey(prefix string, filter *analytics.PopulationFilter) string {
	if filter == nil {
		return prefix + ":all"
	}

	key := prefix
	if filter.Practice != "" {
		key += ":practice:" + filter.Practice
	}
	if filter.PCP != "" {
		key += ":pcp:" + filter.PCP
	}
	if filter.MinAge > 0 {
		key += ":min_age:" + string(rune(filter.MinAge))
	}
	if filter.MaxAge > 0 {
		key += ":max_age:" + string(rune(filter.MaxAge))
	}

	return key
}
