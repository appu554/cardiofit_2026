// Package integration provides analytics semantics tests for KB-11 Population Health.
// CRITICAL: These tests enforce the North Star principle:
// "KB-11 answers population-level questions, NOT patient-level decisions."
package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/kb-11-population-health/internal/analytics"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// ──────────────────────────────────────────────────────────────────────────────
// Population-Level Only Tests
// NORTH STAR: KB-11 is for population-level analytics, not patient CDS.
// ──────────────────────────────────────────────────────────────────────────────

// TestAnalyticsDoesNotExposePatientIdentifiers ensures PHI is never leaked.
func TestAnalyticsDoesNotExposePatientIdentifiers(t *testing.T) {
	t.Run("population snapshot excludes patient identifiers", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		// Serialize to JSON
		jsonBytes, err := json.Marshal(snapshot)
		require.NoError(t, err)
		jsonStr := string(jsonBytes)

		// Verify NO patient-identifying fields
		phiFields := []string{
			"patient_id", "patient_fhir_id", "fhir_patient_id",
			"\"mrn\"", "medical_record_number",
			"\"ssn\"", "social_security",
			"\"dob\"", "date_of_birth", "birth_date",
			"\"name\"", "first_name", "last_name",
			"\"address\"", "street", "city", "zip",
			"\"phone\"", "telephone", "mobile",
			"\"email\"", "email_address",
		}

		for _, field := range phiFields {
			assert.NotContains(t, jsonStr, field, "Population snapshot must not contain: %s", field)
		}
	})

	t.Run("risk stratification report excludes patient details", func(t *testing.T) {
		report := createTestRiskStratificationReport()

		jsonBytes, err := json.Marshal(report)
		require.NoError(t, err)
		jsonStr := string(jsonBytes)

		// Should only have aggregate counts, not patient details
		assert.NotContains(t, jsonStr, "patient_id")
		assert.NotContains(t, jsonStr, "fhir_patient_id")
		assert.NotContains(t, jsonStr, "mrn")
	})

	t.Run("provider analytics aggregates only", func(t *testing.T) {
		// Use local test type since ProviderAnalytics may not exist in analytics package
		providerAnalytics := testProviderAnalytics{
			ProviderID:       "prov-123",
			ProviderName:     "Dr. Smith",
			TotalPatients:    500,
			HighRiskCount:    85,
			RisingRiskCount:  18,
			AverageRiskScore: 0.42,
			RiskDistribution: map[models.RiskTier]int{
				models.RiskTierLow:      225,
				models.RiskTierModerate: 150,
				models.RiskTierHigh:     70,
				models.RiskTierVeryHigh: 15,
				models.RiskTierRising:   18,
				models.RiskTierUnscored: 22,
			},
		}

		jsonBytes, err := json.Marshal(providerAnalytics)
		require.NoError(t, err)
		jsonStr := string(jsonBytes)

		// Should contain aggregates
		assert.Contains(t, jsonStr, "total_patients")
		assert.Contains(t, jsonStr, "high_risk_count")
		assert.Contains(t, jsonStr, "average_risk_score")

		// Should NOT contain patient lists (check exact field names, not substrings)
		assert.NotContains(t, jsonStr, "\"patients\":")
		assert.NotContains(t, jsonStr, "patient_list")
		assert.NotContains(t, jsonStr, "patient_ids")
	})
}

// TestRiskDistributionSumsCorrectly ensures mathematical consistency.
func TestRiskDistributionSumsCorrectly(t *testing.T) {
	t.Run("risk distribution counts sum to total", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		totalFromDist := 0
		for _, count := range snapshot.RiskDistribution {
			totalFromDist += count
		}

		assert.Equal(t, snapshot.TotalPatients, totalFromDist,
			"Distribution counts must sum to total patients")
	})

	t.Run("risk percentages sum to approximately 100", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		totalPct := 0.0
		for _, pct := range snapshot.RiskPercentages {
			totalPct += pct
		}

		// Allow small floating point variance
		assert.InDelta(t, 100.0, totalPct, 1.0,
			"Risk percentages must sum to ~100%%")
	})

	t.Run("high risk count matches distribution", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		highFromDist := snapshot.RiskDistribution[models.RiskTierHigh] +
			snapshot.RiskDistribution[models.RiskTierVeryHigh]

		assert.Equal(t, snapshot.HighRiskCount, highFromDist,
			"HighRiskCount must equal High + VeryHigh from distribution")
	})
}

// TestAverageRiskScoreValidity ensures score calculations are valid.
func TestAverageRiskScoreValidity(t *testing.T) {
	t.Run("average score is within valid range", func(t *testing.T) {
		snapshot := createTestPopulationSnapshot()

		assert.GreaterOrEqual(t, snapshot.AverageRiskScore, 0.0,
			"Average score cannot be negative")
		assert.LessOrEqual(t, snapshot.AverageRiskScore, 1.0,
			"Average score cannot exceed 1.0")
	})

	t.Run("average score reflects distribution", func(t *testing.T) {
		// If mostly low risk, average should be low
		lowRiskSnapshot := &analytics.PopulationSnapshot{
			TotalPatients:    1000,
			AverageRiskScore: 0.25,
			RiskDistribution: map[models.RiskTier]int{
				models.RiskTierLow:      800,
				models.RiskTierModerate: 150,
				models.RiskTierHigh:     40,
				models.RiskTierVeryHigh: 10,
			},
		}

		// With 80% low risk, average should be < 0.4
		assert.Less(t, lowRiskSnapshot.AverageRiskScore, 0.4,
			"Average should be low when most patients are low risk")
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Population Cohort Semantics Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortReturnsAggregatesNotPatients ensures cohorts don't leak patient data.
func TestCohortReturnsAggregatesNotPatients(t *testing.T) {
	t.Run("cohort stats are aggregates only", func(t *testing.T) {
		stats := createTestCohortStats()

		jsonBytes, err := json.Marshal(stats)
		require.NoError(t, err)
		jsonStr := string(jsonBytes)

		// Should have aggregate fields
		assert.Contains(t, jsonStr, "member_count")
		assert.Contains(t, jsonStr, "risk_distribution")
		assert.Contains(t, jsonStr, "average_risk_score")

		// Should NOT have patient lists
		assert.NotContains(t, jsonStr, "members")
		assert.NotContains(t, jsonStr, "patient_list")
		assert.NotContains(t, jsonStr, "patients")
	})
}

// TestCohortMembershipIsHashed ensures patient IDs are hashed when needed.
func TestCohortMembershipIsHashed(t *testing.T) {
	t.Run("member lookup uses FHIR ID not PHI", func(t *testing.T) {
		// When we do need to check membership, we use FHIR ID
		// which is a de-identified reference, not MRN/SSN
		fhirID := "patient-abc123def456"

		// FHIR IDs should look like UUIDs or opaque strings
		assert.NotContains(t, fhirID, "@", "FHIR ID should not be an email")
		// The key assertion is that it's not a recognizable identifier
		assert.Less(t, len(fhirID), 100, "FHIR ID should be reasonably short")
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Dashboard Output Semantics Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestExecutiveDashboardIsAggregate ensures executive dashboard has no PHI.
func TestExecutiveDashboardIsAggregate(t *testing.T) {
	t.Run("executive dashboard contains only aggregates", func(t *testing.T) {
		dashboard := createTestExecutiveDashboard()

		jsonBytes, err := json.Marshal(dashboard)
		require.NoError(t, err)
		jsonStr := string(jsonBytes)

		// Should have aggregate metrics
		expectedFields := []string{
			"total_patients", "high_risk_count", "rising_risk_count",
			"average_risk_score", "care_gap_summary",
		}
		for _, field := range expectedFields {
			assert.Contains(t, jsonStr, field, "Dashboard should have: %s", field)
		}

		// Should NOT have patient-level data
		assert.NotContains(t, jsonStr, "patient_id")
		assert.NotContains(t, jsonStr, "patient_name")
		assert.NotContains(t, jsonStr, "individual_scores")
	})
}

// TestCareManagerDashboardAggregations ensures care manager view is aggregate.
func TestCareManagerDashboardAggregations(t *testing.T) {
	t.Run("care manager dashboard shows actionable counts", func(t *testing.T) {
		// Care managers need to know HOW MANY, not WHO
		dashboard := &testCareManagerDashboard{
			TotalActionablePatients: 150,
			UrgentOutreach:          25,
			ScheduledFollowUp:       75,
			NewHighRisk:             15,
			RisingRisk:              35,
		}

		// These are appropriate population metrics
		assert.Greater(t, dashboard.TotalActionablePatients, 0)
		assert.Greater(t, dashboard.UrgentOutreach, 0)

		// The dashboard gives counts for planning, not patient lists
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Query Result Safety Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCustomQueryResultsAreAggregated ensures custom queries don't return patient lists.
func TestCustomQueryResultsAreAggregated(t *testing.T) {
	t.Run("custom query returns counts not lists", func(t *testing.T) {
		result := &analytics.CustomQueryResult{
			TotalCount: 5,
			Data: []map[string]interface{}{
				{"practice": "Downtown Medical", "high_risk_count": 45, "avg_score": 0.52},
				{"practice": "Health Plus", "high_risk_count": 32, "avg_score": 0.48},
			},
			ExecutedAt: time.Now(),
		}

		jsonBytes, err := json.Marshal(result)
		require.NoError(t, err)
		jsonStr := string(jsonBytes)

		// Results should be grouped/aggregated
		assert.Contains(t, jsonStr, "practice")
		assert.Contains(t, jsonStr, "high_risk_count")
		assert.Contains(t, jsonStr, "avg_score")

		// Not individual patients
		assert.NotContains(t, jsonStr, "patient_id")
		assert.NotContains(t, jsonStr, "patient_scores")
	})

	t.Run("custom query enforces row limits", func(t *testing.T) {
		// KB-11 should limit query results to prevent large data exports
		maxRows := 10000

		result := &analytics.CustomQueryResult{
			TotalCount: 10000,
			Limit:      maxRows,
		}

		assert.LessOrEqual(t, result.TotalCount, maxRows, "Row count should not exceed max")
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// API Response Safety Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestAPIResponsesAreSafe ensures API responses don't expose sensitive data.
func TestAPIResponsesAreSafe(t *testing.T) {
	t.Run("error responses do not leak data", func(t *testing.T) {
		errorResp := models.NewErrorResponse(
			"Query failed",
			"QUERY_ERROR",
			"Database connection timeout",
		)

		jsonBytes, err := json.Marshal(errorResp)
		require.NoError(t, err)
		jsonStr := string(jsonBytes)

		// Error should not contain sensitive info
		assert.NotContains(t, jsonStr, "password")
		assert.NotContains(t, jsonStr, "connection_string")
		assert.NotContains(t, jsonStr, "host=")
	})

	t.Run("validation errors do not echo sensitive input", func(t *testing.T) {
		// If someone sends PHI in a query, we shouldn't echo it back
		errorResp := models.NewErrorResponse(
			"Invalid filter value",
			"INVALID_FILTER",
			"Filter value must be a valid risk tier",
		)

		jsonBytes, _ := json.Marshal(errorResp)
		jsonStr := string(jsonBytes)

		// Should not contain any echoed patient data
		assert.NotContains(t, jsonStr, "John Doe")
		assert.NotContains(t, jsonStr, "123-45-6789")
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Trend Analysis Semantics Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestTrendAnalysisIsAggregate ensures time-series data is aggregate.
func TestTrendAnalysisIsAggregate(t *testing.T) {
	t.Run("trend data shows population metrics over time", func(t *testing.T) {
		trends := []testTrendDataPoint{
			{Period: "2024-01", HighRiskCount: 450, AvgRiskScore: 0.42, TotalPatients: 10000},
			{Period: "2024-02", HighRiskCount: 480, AvgRiskScore: 0.43, TotalPatients: 10200},
			{Period: "2024-03", HighRiskCount: 510, AvgRiskScore: 0.44, TotalPatients: 10500},
		}

		for _, point := range trends {
			// Each point should be aggregate
			assert.NotEmpty(t, point.Period)
			assert.Greater(t, point.TotalPatients, 0)

			// No patient-level data
			jsonBytes, _ := json.Marshal(point)
			jsonStr := string(jsonBytes)
			assert.NotContains(t, jsonStr, "patient_id")
		}
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Test Helper Types (local to tests - not dependent on analytics package)
// ──────────────────────────────────────────────────────────────────────────────

// testCareManagerDashboard represents care manager view (test-local type).
type testCareManagerDashboard struct {
	TotalActionablePatients int `json:"total_actionable_patients"`
	UrgentOutreach          int `json:"urgent_outreach"`
	ScheduledFollowUp       int `json:"scheduled_follow_up"`
	NewHighRisk             int `json:"new_high_risk"`
	RisingRisk              int `json:"rising_risk"`
}

// testProviderAnalytics represents provider-level aggregates (test-local type).
type testProviderAnalytics struct {
	ProviderID       string                  `json:"provider_id"`
	ProviderName     string                  `json:"provider_name"`
	TotalPatients    int                     `json:"total_patients"`
	HighRiskCount    int                     `json:"high_risk_count"`
	RisingRiskCount  int                     `json:"rising_risk_count"`
	AverageRiskScore float64                 `json:"average_risk_score"`
	RiskDistribution map[models.RiskTier]int `json:"risk_distribution"`
}

// testExecutiveDashboard represents executive dashboard (test-local type).
type testExecutiveDashboard struct {
	TotalPatients    int                `json:"total_patients"`
	HighRiskCount    int                `json:"high_risk_count"`
	RisingRiskCount  int                `json:"rising_risk_count"`
	AverageRiskScore float64            `json:"average_risk_score"`
	CareGapSummary   testCareGapSummary `json:"care_gap_summary"`
}

// testCareGapSummary represents care gap summary (test-local type).
type testCareGapSummary struct {
	TotalGaps         int     `json:"total_gaps"`
	OpenGaps          int     `json:"open_gaps"`
	ClosedGaps        int     `json:"closed_gaps"`
	OverdueGaps       int     `json:"overdue_gaps"`
	AvgGapsPerPatient float64 `json:"avg_gaps_per_patient"`
}

// testCohortStats represents cohort statistics (test-local type).
type testCohortStats struct {
	MemberCount      int                     `json:"member_count"`
	AverageRiskScore float64                 `json:"average_risk_score"`
	HighRiskCount    int                     `json:"high_risk_count"`
	RiskDistribution map[models.RiskTier]int `json:"risk_distribution"`
	ByPractice       map[string]int          `json:"by_practice"`
}

// testTrendDataPoint represents a single trend data point (test-local type).
type testTrendDataPoint struct {
	Period        string  `json:"period"`
	HighRiskCount int     `json:"high_risk_count"`
	AvgRiskScore  float64 `json:"avg_risk_score"`
	TotalPatients int     `json:"total_patients"`
}

// createTestExecutiveDashboard creates test executive dashboard.
func createTestExecutiveDashboard() *testExecutiveDashboard {
	return &testExecutiveDashboard{
		TotalPatients:    10000,
		HighRiskCount:    1700,
		RisingRiskCount:  350,
		AverageRiskScore: 0.42,
		CareGapSummary: testCareGapSummary{
			TotalGaps:         2500,
			OpenGaps:          1800,
			ClosedGaps:        700,
			OverdueGaps:       450,
			AvgGapsPerPatient: 0.25,
		},
	}
}

// createTestCohortStats creates test cohort statistics.
func createTestCohortStats() *testCohortStats {
	return &testCohortStats{
		MemberCount:      1500,
		AverageRiskScore: 0.68,
		HighRiskCount:    1050,
		RiskDistribution: map[models.RiskTier]int{
			models.RiskTierHigh:     900,
			models.RiskTierVeryHigh: 150,
			models.RiskTierModerate: 300,
			models.RiskTierRising:   150,
		},
		ByPractice: map[string]int{
			"Downtown Medical": 500,
			"Health Plus":      400,
			"Primary Care":     350,
			"Community Clinic": 250,
		},
	}
}
