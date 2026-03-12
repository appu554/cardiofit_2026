package scoring

import (
	"context"
	"testing"
	"time"

	"flow2-go-engine/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareAndRankEngine_BasicFunctionality(t *testing.T) {
	// Create engine with defaults
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	engine := NewCompareAndRankEngineWithDefaults(logger)
	
	// Create test request
	request := createTestCompareAndRankRequest()
	
	// Execute compare and rank
	response, err := engine.CompareAndRank(context.Background(), request)
	
	// Assertions
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Len(t, response.Ranked, 3) // Should have 3 candidates
	
	// Check ranking order (highest score first)
	for i := 0; i < len(response.Ranked)-1; i++ {
		assert.GreaterOrEqual(t, response.Ranked[i].FinalScore, response.Ranked[i+1].FinalScore,
			"Rankings should be in descending order by score")
		assert.Equal(t, i+1, response.Ranked[i].Rank, "Rank should match position")
	}
	
	// Check audit information
	assert.NotEmpty(t, response.Audit.ProfileUsed.Weights)
	assert.NotEmpty(t, response.Audit.ProfileUsed.Penalties)
	assert.Equal(t, 3, response.Audit.CandidatesProcessed)
	assert.GreaterOrEqual(t, response.Audit.CandidatesPruned, 0)
}

func TestCompareAndRankEngine_PhenotypeAwareWeights(t *testing.T) {
	engine := NewCompareAndRankEngineWithDefaults(logrus.New())
	
	testCases := []struct {
		phenotype      string
		expectedWeight string
	}{
		{"ASCVD", "ASCVD"},
		{"HF", "HF"},
		{"CKD", "CKD"},
		{"NONE", "NONE"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.phenotype, func(t *testing.T) {
			request := createTestCompareAndRankRequest()
			request.PatientContext.RiskPhenotype = tc.phenotype
			request.ConfigRef.WeightProfile = tc.expectedWeight
			
			response, err := engine.CompareAndRank(context.Background(), request)
			
			require.NoError(t, err)
			assert.Equal(t, tc.expectedWeight, response.Audit.ProfileUsed.Weights)
		})
	}
}

func TestCompareAndRankEngine_DominancePruning(t *testing.T) {
	engine := NewCompareAndRankEngineWithDefaults(logrus.New())
	
	// Create candidates where one clearly dominates another
	request := &models.CompareAndRankRequest{
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
			// Superior candidate
			{
				TherapyID: "superior",
				Efficacy: models.EfficacyDetail{
					ExpectedA1cDropPct: 1.5,
					CVBenefit:         true,
				},
				Safety: models.SafetyDetail{
					ResidualDDI:    "none",
					HypoPropensity: "low",
					WeightEffect:   "neutral",
				},
				Cost: models.CostDetail{
					MonthlyEstimate: 50.0,
					Currency:        "USD",
				},
				Availability: models.AvailabilityDetail{
					Tier:   1,
					OnHand: 100,
				},
				Adherence: models.AdherenceDetail{
					DosesPerDay: 1,
					PillBurden:  1,
				},
			},
			// Inferior candidate (dominated)
			{
				TherapyID: "inferior",
				Efficacy: models.EfficacyDetail{
					ExpectedA1cDropPct: 1.0, // Lower efficacy
					CVBenefit:         false,
				},
				Safety: models.SafetyDetail{
					ResidualDDI:    "moderate", // Worse safety
					HypoPropensity: "med",
					WeightEffect:   "gain",
				},
				Cost: models.CostDetail{
					MonthlyEstimate: 100.0, // Higher cost
					Currency:        "USD",
				},
				Availability: models.AvailabilityDetail{
					Tier:   2, // Worse availability
					OnHand: 10,
				},
				Adherence: models.AdherenceDetail{
					DosesPerDay: 2, // Worse adherence
					PillBurden:  2,
				},
			},
		},
		ConfigRef: models.ConfigReference{
			WeightProfile:    "NONE",
			PenaltiesProfile: "default",
		},
		RequestID: "test-dominance",
		Timestamp: time.Now(),
	}
	
	response, err := engine.CompareAndRank(context.Background(), request)
	
	require.NoError(t, err)
	// Should have pruned the dominated candidate or ranked superior first
	assert.Equal(t, "superior", response.Ranked[0].TherapyID)
	if len(response.Ranked) > 1 {
		assert.Greater(t, response.Ranked[0].FinalScore, response.Ranked[1].FinalScore)
	}
}

func TestCompareAndRankEngine_KnockoutRules(t *testing.T) {
	engine := NewCompareAndRankEngineWithDefaults(logrus.New())
	
	// Test injectable knockout
	request := createTestCompareAndRankRequest()
	request.PatientContext.Preferences.AvoidInjectables = true
	
	// Make one candidate injectable
	request.Candidates[0].Dose.Route = "sc" // Subcutaneous injection
	
	response, err := engine.CompareAndRank(context.Background(), request)
	
	require.NoError(t, err)
	// Injectable candidate should be filtered out
	for _, ranked := range response.Ranked {
		assert.NotEqual(t, "sc", ranked.SafetyVerified.FinalDose.Route)
	}
}

func TestCompareAndRankEngine_ExplainabilityFeatures(t *testing.T) {
	engine := NewCompareAndRankEngineWithDefaults(logrus.New())
	
	request := createTestCompareAndRankRequest()
	response, err := engine.CompareAndRank(context.Background(), request)
	
	require.NoError(t, err)
	require.NotEmpty(t, response.Ranked)
	
	topRanked := response.Ranked[0]
	
	// Check contributions
	assert.Len(t, topRanked.Contributions, 6) // Should have 6 factors
	
	expectedFactors := []string{"efficacy", "safety", "availability", "cost", "adherence", "preference"}
	actualFactors := make([]string, len(topRanked.Contributions))
	for i, contrib := range topRanked.Contributions {
		actualFactors[i] = contrib.Factor
	}
	
	for _, expected := range expectedFactors {
		assert.Contains(t, actualFactors, expected)
	}
	
	// Check each contribution has required fields
	for _, contrib := range topRanked.Contributions {
		assert.NotEmpty(t, contrib.Factor)
		assert.GreaterOrEqual(t, contrib.Value, 0.0)
		assert.LessOrEqual(t, contrib.Value, 1.0)
		assert.Greater(t, contrib.Weight, 0.0)
		assert.NotEmpty(t, contrib.Note)
	}
	
	// Check eligibility flags
	assert.NotNil(t, topRanked.EligibilityFlags)
	
	// Check notes
	assert.NotEmpty(t, topRanked.Notes)
	
	// Check audit info
	assert.NotNil(t, topRanked.AuditInfo)
	assert.NotEmpty(t, topRanked.AuditInfo.RawInputs)
}

func TestCompareAndRankEngine_TieBreakers(t *testing.T) {
	engine := NewCompareAndRankEngineWithDefaults(logrus.New())
	
	// Create candidates with identical final scores but different sub-scores
	request := createTestCompareAndRankRequest()
	
	// Modify candidates to have very similar total scores but different safety scores
	request.Candidates[0].Safety.ResidualDDI = "none"    // Better safety
	request.Candidates[1].Safety.ResidualDDI = "moderate" // Worse safety
	
	response, err := engine.CompareAndRank(context.Background(), request)
	
	require.NoError(t, err)
	require.Len(t, response.Ranked, 3)
	
	// If scores are very close, tie-breakers should apply
	// The candidate with better safety should rank higher if other factors are similar
	if response.Ranked[0].FinalScore - response.Ranked[1].FinalScore < 0.1 {
		assert.GreaterOrEqual(t, response.Ranked[0].SubScores.Safety.Score, 
			response.Ranked[1].SubScores.Safety.Score)
	}
}

// Helper function to create test request
func createTestCompareAndRankRequest() *models.CompareAndRankRequest {
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
			createTestProposal("metformin", 1.2, "none", "low", 25.0, 1, 1),
			createTestProposal("glipizide", 1.0, "moderate", "high", 15.0, 2, 2),
			createTestProposal("insulin", 1.8, "none", "high", 100.0, 1, 4),
		},
		ConfigRef: models.ConfigReference{
			WeightProfile:    "NONE",
			PenaltiesProfile: "default",
		},
		RequestID: "test-request",
		Timestamp: time.Now(),
	}
}

// Helper function to create test proposal
func createTestProposal(therapyID string, a1cDrop float64, ddi, hypo string, cost float64, tier, dosesPerDay int) models.EnhancedProposal {
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
			Route:     "po",
			Rationale: "standard dose",
		},
		Efficacy: models.EfficacyDetail{
			ExpectedA1cDropPct: a1cDrop,
			CVBenefit:         false,
			HFBenefit:         false,
			CKDBenefit:        false,
		},
		Safety: models.SafetyDetail{
			ResidualDDI:    ddi,
			HypoPropensity: hypo,
			WeightEffect:   "neutral",
		},
		Suitability: models.SuitabilityDetail{
			RenalFit:   true,
			HepaticFit: true,
		},
		Adherence: models.AdherenceDetail{
			DosesPerDay:      dosesPerDay,
			PillBurden:       1,
			RequiresDevice:   false,
			RequiresTraining: false,
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
				"drug_master": "v1.0",
				"ddi":         "v1.0",
			},
		},
	}
}
