package scoring

import (
	"context"
	"fmt"
	"testing"
	"time"

	"flow2-go-engine/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompareAndRankIntegration tests the complete compare-and-rank workflow
func TestCompareAndRankIntegration(t *testing.T) {
	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

	// Create engine with defaults (no KB config file needed for test)
	engine := NewCompareAndRankEngineWithDefaults(logger)

	// Test Case 1: ASCVD Patient - Should prioritize CV benefit medications
	t.Run("ASCVD_Patient_Prioritizes_CV_Benefit", func(t *testing.T) {
		request := createASCVDTestRequest()
		response, err := engine.CompareAndRank(context.Background(), request)

		require.NoError(t, err)
		require.NotEmpty(t, response.Ranked)

		// Verify ASCVD profile was used
		assert.Equal(t, "ASCVD", response.Audit.ProfileUsed.Weights)

		// Top recommendation should have CV benefit or high efficacy
		topRanked := response.Ranked[0]
		assert.True(t, 
			topRanked.SubScores.Efficacy.CVBenefit || 
			topRanked.SubScores.Efficacy.Score > 0.7,
			"ASCVD patient should get CV benefit or high efficacy medication")

		// Verify explainability
		assert.NotEmpty(t, topRanked.Contributions)
		assert.NotEmpty(t, topRanked.Notes)
		assert.True(t, topRanked.EligibilityFlags.TopSlotEligible)
	})

	// Test Case 2: Budget Mode - Should prioritize cost-effective options
	t.Run("Budget_Mode_Prioritizes_Cost", func(t *testing.T) {
		request := createBudgetTestRequest()
		response, err := engine.CompareAndRank(context.Background(), request)

		require.NoError(t, err)
		require.NotEmpty(t, response.Ranked)

		// Verify budget profile was used
		assert.Equal(t, "BUDGET_MODE", response.Audit.ProfileUsed.Weights)

		// Top recommendation should have good cost score
		topRanked := response.Ranked[0]
		assert.Greater(t, topRanked.SubScores.Cost.Score, 0.6,
			"Budget mode should prioritize cost-effective options")

		// Verify cost is a major contributor
		costContribution := findContribution(topRanked.Contributions, "cost")
		require.NotNil(t, costContribution)
		assert.Greater(t, costContribution.Weight, 0.15, // Budget mode has 22% cost weight
			"Cost should have significant weight in budget mode")
	})

	// Test Case 3: Dominance Pruning - Should remove dominated options
	t.Run("Dominance_Pruning_Removes_Inferior_Options", func(t *testing.T) {
		request := createDominanceTestRequest()
		response, err := engine.CompareAndRank(context.Background(), request)

		require.NoError(t, err)

		// Should have pruned some candidates
		assert.Greater(t, response.Audit.CandidatesPruned, 0,
			"Should have pruned dominated candidates")

		// Remaining candidates should be non-dominated
		assert.LessOrEqual(t, len(response.Ranked), 2,
			"Should have few non-dominated candidates")
	})

	// Test Case 4: Preference Knockout - Should respect patient preferences
	t.Run("Preference_Knockout_Respects_Patient_Choices", func(t *testing.T) {
		request := createPreferenceTestRequest()
		response, err := engine.CompareAndRank(context.Background(), request)

		require.NoError(t, err)
		require.NotEmpty(t, response.Ranked)

		// No injectable medications should be in top slots
		for _, ranked := range response.Ranked {
			if ranked.EligibilityFlags.TopSlotEligible {
				// Check that it's not injectable (route should be "po")
				assert.NotEqual(t, "sc", ranked.SafetyVerified.FinalDose.Route,
					"Injectable should not be top-slot eligible when patient avoids injectables")
			}
		}
	})

	// Test Case 5: Performance Validation
	t.Run("Performance_Under_200ms", func(t *testing.T) {
		request := createPerformanceTestRequest()
		
		start := time.Now()
		response, err := engine.CompareAndRank(context.Background(), request)
		duration := time.Since(start)

		require.NoError(t, err)
		require.NotEmpty(t, response.Ranked)

		// Should complete within 200ms
		assert.Less(t, duration.Milliseconds(), int64(200),
			"Compare-and-rank should complete within 200ms")

		// Audit should track processing time
		assert.Greater(t, response.Audit.ProcessingTime.Nanoseconds(), int64(0))
	})
}

// Helper functions to create test requests

func createASCVDTestRequest() *models.CompareAndRankRequest {
	return &models.CompareAndRankRequest{
		PatientContext: models.PatientRiskContext{
			RiskPhenotype: "ASCVD",
			ResourceTier:  "standard",
			Preferences: models.JITPatientPreferences{
				AvoidInjectables:   false,
				OnceDailyPreferred: true,
				CostSensitivity:    "low",
			},
		},
		Candidates: []models.EnhancedProposal{
			createTestEnhancedProposal("semaglutide", 1.5, true, false, false, 800.0, 2, "sc"),
			createTestEnhancedProposal("empagliflozin", 0.8, true, true, true, 400.0, 1, "po"),
			createTestEnhancedProposal("glimepiride", 1.0, false, false, false, 25.0, 1, "po"),
		},
		ConfigRef: models.ConfigReference{
			WeightProfile:    "ASCVD",
			PenaltiesProfile: "default",
		},
		RequestID: "test-ascvd",
		Timestamp: time.Now(),
	}
}

func createBudgetTestRequest() *models.CompareAndRankRequest {
	request := createASCVDTestRequest()
	request.PatientContext.RiskPhenotype = "NONE"
	request.PatientContext.ResourceTier = "minimal"
	request.PatientContext.Preferences.CostSensitivity = "high"
	request.ConfigRef.WeightProfile = "BUDGET_MODE"
	request.RequestID = "test-budget"
	return request
}

func createDominanceTestRequest() *models.CompareAndRankRequest {
	return &models.CompareAndRankRequest{
		PatientContext: models.PatientRiskContext{
			RiskPhenotype: "NONE",
			ResourceTier:  "standard",
			Preferences: models.JITPatientPreferences{
				AvoidInjectables:   false,
				OnceDailyPreferred: true,
				CostSensitivity:    "medium",
			},
		},
		Candidates: []models.EnhancedProposal{
			// Superior option
			createTestEnhancedProposal("superior", 1.5, true, false, false, 50.0, 1, "po"),
			// Inferior option (dominated)
			createTestEnhancedProposal("inferior", 1.0, false, false, false, 100.0, 2, "po"),
			// Another inferior option
			createTestEnhancedProposal("also_inferior", 0.8, false, false, false, 150.0, 3, "po"),
		},
		ConfigRef: models.ConfigReference{
			WeightProfile:    "NONE",
			PenaltiesProfile: "default",
		},
		RequestID: "test-dominance",
		Timestamp: time.Now(),
	}
}

func createPreferenceTestRequest() *models.CompareAndRankRequest {
	return &models.CompareAndRankRequest{
		PatientContext: models.PatientRiskContext{
			RiskPhenotype: "NONE",
			ResourceTier:  "standard",
			Preferences: models.JITPatientPreferences{
				AvoidInjectables:   true, // Key preference
				OnceDailyPreferred: true,
				CostSensitivity:    "medium",
			},
		},
		Candidates: []models.EnhancedProposal{
			createTestEnhancedProposal("injectable", 1.5, true, false, false, 800.0, 2, "sc"), // Should be filtered
			createTestEnhancedProposal("oral", 1.2, false, false, false, 400.0, 1, "po"),      // Should be preferred
		},
		ConfigRef: models.ConfigReference{
			WeightProfile:    "NONE",
			PenaltiesProfile: "default",
		},
		RequestID: "test-preference",
		Timestamp: time.Now(),
	}
}

func createPerformanceTestRequest() *models.CompareAndRankRequest {
	// Create request with many candidates to test performance
	candidates := make([]models.EnhancedProposal, 20)
	for i := 0; i < 20; i++ {
		candidates[i] = createTestEnhancedProposal(
			fmt.Sprintf("med_%d", i),
			0.5+float64(i)*0.1,
			i%3 == 0, // CV benefit every 3rd
			i%4 == 0, // HF benefit every 4th
			i%5 == 0, // CKD benefit every 5th
			float64(50+i*20),
			1+(i%4),
			"po",
		)
	}

	return &models.CompareAndRankRequest{
		PatientContext: models.PatientRiskContext{
			RiskPhenotype: "NONE",
			ResourceTier:  "standard",
			Preferences: models.JITPatientPreferences{
				AvoidInjectables:   false,
				OnceDailyPreferred: true,
				CostSensitivity:    "medium",
			},
		},
		Candidates: candidates,
		ConfigRef: models.ConfigReference{
			WeightProfile:    "NONE",
			PenaltiesProfile: "default",
		},
		RequestID: "test-performance",
		Timestamp: time.Now(),
	}
}

func createTestEnhancedProposal(therapyID string, a1cDrop float64, cvBenefit, hfBenefit, ckdBenefit bool, cost float64, tier int, route string) models.EnhancedProposal {
	return models.EnhancedProposal{
		TherapyID: therapyID,
		Class:     "antidiabetic",
		Agent:     therapyID,
		Regimen: models.RegimenDetail{
			Form:      "tablet",
			Frequency: "daily",
			IsFDC:     false,
			PillCount: 1,
		},
		Dose: models.DoseDetail{
			Amount:    500,
			Unit:      "mg",
			Frequency: "daily",
			Route:     route,
			Rationale: "test dose",
		},
		Efficacy: models.EfficacyDetail{
			ExpectedA1cDropPct: a1cDrop,
			CVBenefit:         cvBenefit,
			HFBenefit:         hfBenefit,
			CKDBenefit:        ckdBenefit,
		},
		Safety: models.SafetyDetail{
			ResidualDDI:    "none",
			HypoPropensity: "low",
			WeightEffect:   "neutral",
		},
		Suitability: models.SuitabilityDetail{
			RenalFit:   true,
			HepaticFit: true,
		},
		Adherence: models.AdherenceDetail{
			DosesPerDay:      1,
			PillBurden:       1,
			RequiresDevice:   route != "po",
			RequiresTraining: route == "sc",
		},
		Availability: models.AvailabilityDetail{
			Tier:         tier,
			OnHand:       100,
			LeadTimeDays: 0,
		},
		Cost: models.CostDetail{
			MonthlyEstimate: cost,
			Currency:        "USD",
		},
		Preferences: models.PreferencesDetail{
			AvoidInjectables:   false,
			OnceDailyPreferred: true,
			CostSensitivity:    "medium",
		},
		Provenance: models.ProvenanceDetail{
			KBVersions: map[string]string{
				"test": "v1.0",
			},
		},
	}
}

// Helper function to find a contribution by factor name
func findContribution(contributions []models.ScoreContribution, factor string) *models.ScoreContribution {
	for _, contrib := range contributions {
		if contrib.Factor == factor {
			return &contrib
		}
	}
	return nil
}
