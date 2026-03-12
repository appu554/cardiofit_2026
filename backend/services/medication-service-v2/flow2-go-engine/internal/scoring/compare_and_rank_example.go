// Package scoring provides example usage of the enhanced compare-and-rank system
package scoring

import (
	"context"
	"fmt"
	"time"

	"flow2-go-engine/internal/models"
	"github.com/sirupsen/logrus"
)

// ExampleCompareAndRankUsage demonstrates how to use the enhanced compare-and-rank system
func ExampleCompareAndRankUsage() {
	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create compare-and-rank engine with KB configuration
	configPath := "internal/scoring/kb_config.yaml"
	engine := NewCompareAndRankEngine(configPath, logger)

	// Example 1: ASCVD patient with cardiovascular risk
	fmt.Println("=== Example 1: ASCVD Patient ===")
	ascvdRequest := createASCVDPatientRequest()
	ascvdResponse, err := engine.CompareAndRank(context.Background(), ascvdRequest)
	if err != nil {
		logger.WithError(err).Error("Failed to process ASCVD request")
		return
	}
	
	printRankingResults("ASCVD Patient", ascvdResponse)

	// Example 2: Budget-conscious patient
	fmt.Println("\n=== Example 2: Budget-Conscious Patient ===")
	budgetRequest := createBudgetPatientRequest()
	budgetResponse, err := engine.CompareAndRank(context.Background(), budgetRequest)
	if err != nil {
		logger.WithError(err).Error("Failed to process budget request")
		return
	}
	
	printRankingResults("Budget-Conscious Patient", budgetResponse)

	// Example 3: CKD patient with kidney disease
	fmt.Println("\n=== Example 3: CKD Patient ===")
	ckdRequest := createCKDPatientRequest()
	ckdResponse, err := engine.CompareAndRank(context.Background(), ckdRequest)
	if err != nil {
		logger.WithError(err).Error("Failed to process CKD request")
		return
	}
	
	printRankingResults("CKD Patient", ckdResponse)

	// Example 4: Demonstrate explainability features
	fmt.Println("\n=== Example 4: Explainability Features ===")
	demonstrateExplainability(ascvdResponse)
}

// createASCVDPatientRequest creates a request for an ASCVD patient
func createASCVDPatientRequest() *models.CompareAndRankRequest {
	return &models.CompareAndRankRequest{
		PatientContext: models.PatientRiskContext{
			RiskPhenotype: "ASCVD", // Cardiovascular disease
			ResourceTier:  "standard",
			Preferences: models.JITPatientPreferences{
				AvoidInjectables:   false, // Willing to use injectables for CV benefit
				OnceDailyPreferred: true,
				CostSensitivity:    "low", // Less cost-sensitive due to high risk
			},
		},
		Candidates: []models.EnhancedProposal{
			// GLP-1 RA with CV benefit
			{
				TherapyID: "semaglutide_weekly",
				Class:     "GLP-1_RA",
				Agent:     "semaglutide",
				Regimen: models.RegimenDetail{
					Form:      "injection",
					Frequency: "weekly",
					IsFDC:     false,
					PillCount: 0,
				},
				Dose: models.DoseDetail{
					Amount:    1.0,
					Unit:      "mg",
					Frequency: "weekly",
					Route:     "sc",
					Rationale: "CV outcome benefit",
				},
				Efficacy: models.EfficacyDetail{
					ExpectedA1cDropPct: 1.5,
					CVBenefit:         true, // Key benefit for ASCVD
					HFBenefit:         false,
					CKDBenefit:        false,
				},
				Safety: models.SafetyDetail{
					ResidualDDI:    "none",
					HypoPropensity: "low",
					WeightEffect:   "loss", // Additional benefit
				},
				Suitability: models.SuitabilityDetail{
					RenalFit:   true,
					HepaticFit: true,
				},
				Adherence: models.AdherenceDetail{
					DosesPerDay:      0, // Weekly dosing
					PillBurden:       0,
					RequiresDevice:   true,
					RequiresTraining: true,
				},
				Availability: models.AvailabilityDetail{
					Tier:         2, // Non-preferred but available
					OnHand:       50,
					LeadTimeDays: 1,
				},
				Cost: models.CostDetail{
					MonthlyEstimate: 800.0, // Expensive but CV benefit justifies
					Currency:        "USD",
					PatientCopay:    100.0,
				},
				Preferences: models.PreferencesDetail{
					AvoidInjectables:   false,
					OnceDailyPreferred: true,
					CostSensitivity:    "low",
				},
				Provenance: models.ProvenanceDetail{
					KBVersions: map[string]string{
						"drug_master": "v1.2",
						"cv_outcomes": "v2.1",
					},
				},
			},
			// SGLT2i with CV benefit
			{
				TherapyID: "empagliflozin",
				Class:     "SGLT2i",
				Agent:     "empagliflozin",
				Regimen: models.RegimenDetail{
					Form:      "tablet",
					Frequency: "daily",
					IsFDC:     false,
					PillCount: 1,
				},
				Dose: models.DoseDetail{
					Amount:    10,
					Unit:      "mg",
					Frequency: "daily",
					Route:     "po",
					Rationale: "CV and HF benefit",
				},
				Efficacy: models.EfficacyDetail{
					ExpectedA1cDropPct: 0.8,
					CVBenefit:         true,
					HFBenefit:         true, // Additional benefit
					CKDBenefit:        true,
				},
				Safety: models.SafetyDetail{
					ResidualDDI:    "none",
					HypoPropensity: "low",
					WeightEffect:   "loss",
				},
				Suitability: models.SuitabilityDetail{
					RenalFit:   true,
					HepaticFit: true,
				},
				Adherence: models.AdherenceDetail{
					DosesPerDay:      1,
					PillBurden:       1,
					RequiresDevice:   false,
					RequiresTraining: false,
				},
				Availability: models.AvailabilityDetail{
					Tier:         1, // Preferred
					OnHand:       200,
					LeadTimeDays: 0,
				},
				Cost: models.CostDetail{
					MonthlyEstimate: 400.0,
					Currency:        "USD",
					PatientCopay:    50.0,
				},
				Preferences: models.PreferencesDetail{
					AvoidInjectables:   false,
					OnceDailyPreferred: true,
					CostSensitivity:    "low",
				},
				Provenance: models.ProvenanceDetail{
					KBVersions: map[string]string{
						"drug_master": "v1.2",
						"cv_outcomes": "v2.1",
					},
				},
			},
			// Traditional sulfonylurea (lower cost, less CV benefit)
			{
				TherapyID: "glimepiride",
				Class:     "sulfonylurea",
				Agent:     "glimepiride",
				Regimen: models.RegimenDetail{
					Form:      "tablet",
					Frequency: "daily",
					IsFDC:     false,
					PillCount: 1,
				},
				Dose: models.DoseDetail{
					Amount:    2,
					Unit:      "mg",
					Frequency: "daily",
					Route:     "po",
					Rationale: "cost-effective option",
				},
				Efficacy: models.EfficacyDetail{
					ExpectedA1cDropPct: 1.0,
					CVBenefit:         false, // No CV benefit
					HFBenefit:         false,
					CKDBenefit:        false,
				},
				Safety: models.SafetyDetail{
					ResidualDDI:    "none",
					HypoPropensity: "high", // Major safety concern
					WeightEffect:   "gain",
				},
				Suitability: models.SuitabilityDetail{
					RenalFit:   true,
					HepaticFit: true,
				},
				Adherence: models.AdherenceDetail{
					DosesPerDay:      1,
					PillBurden:       1,
					RequiresDevice:   false,
					RequiresTraining: false,
				},
				Availability: models.AvailabilityDetail{
					Tier:         1,
					OnHand:       500,
					LeadTimeDays: 0,
				},
				Cost: models.CostDetail{
					MonthlyEstimate: 25.0, // Very cheap
					Currency:        "USD",
					PatientCopay:    5.0,
				},
				Preferences: models.PreferencesDetail{
					AvoidInjectables:   false,
					OnceDailyPreferred: true,
					CostSensitivity:    "low",
				},
				Provenance: models.ProvenanceDetail{
					KBVersions: map[string]string{
						"drug_master": "v1.2",
					},
				},
			},
		},
		ConfigRef: models.ConfigReference{
			WeightProfile:    "ASCVD", // Use ASCVD-specific weights
			PenaltiesProfile: "default",
		},
		RequestID: "example-ascvd-001",
		Timestamp: time.Now(),
	}
}

// createBudgetPatientRequest creates a request for a budget-conscious patient
func createBudgetPatientRequest() *models.CompareAndRankRequest {
	request := createASCVDPatientRequest()
	
	// Modify for budget constraints
	request.PatientContext.RiskPhenotype = "NONE" // No high-risk phenotype
	request.PatientContext.ResourceTier = "minimal" // Budget constraints
	request.PatientContext.Preferences.CostSensitivity = "high"
	request.ConfigRef.WeightProfile = "BUDGET_MODE" // Use budget-aware weights
	request.RequestID = "example-budget-001"
	
	return request
}

// createCKDPatientRequest creates a request for a CKD patient
func createCKDPatientRequest() *models.CompareAndRankRequest {
	request := createASCVDPatientRequest()
	
	// Modify for CKD
	request.PatientContext.RiskPhenotype = "CKD"
	request.ConfigRef.WeightProfile = "CKD"
	request.RequestID = "example-ckd-001"
	
	// Emphasize SGLT2i benefit for CKD
	for i := range request.Candidates {
		if request.Candidates[i].TherapyID == "empagliflozin" {
			request.Candidates[i].Efficacy.CKDBenefit = true
		}
	}
	
	return request
}

// printRankingResults prints the ranking results in a readable format
func printRankingResults(scenario string, response *models.CompareAndRankResponse) {
	fmt.Printf("Scenario: %s\n", scenario)
	fmt.Printf("Candidates processed: %d, Pruned: %d\n", 
		response.Audit.CandidatesProcessed, response.Audit.CandidatesPruned)
	fmt.Printf("Weight profile used: %s\n", response.Audit.ProfileUsed.Weights)
	fmt.Printf("Processing time: %v\n", response.Audit.ProcessingTime)
	
	fmt.Println("\nRanking Results:")
	for i, ranked := range response.Ranked {
		fmt.Printf("%d. %s (Score: %.3f)\n", i+1, ranked.TherapyID, ranked.FinalScore)
		fmt.Printf("   Efficacy: %.3f, Safety: %.3f, Cost: %.3f\n",
			ranked.SubScores.Efficacy.Score,
			ranked.SubScores.Safety.Score,
			ranked.SubScores.Cost.Score)
		fmt.Printf("   Top slot eligible: %v\n", ranked.EligibilityFlags.TopSlotEligible)
		if len(ranked.Notes) > 0 {
			fmt.Printf("   Notes: %s\n", ranked.Notes[0])
		}
		fmt.Println()
	}
}

// demonstrateExplainability shows the explainability features
func demonstrateExplainability(response *models.CompareAndRankResponse) {
	if len(response.Ranked) == 0 {
		return
	}
	
	topRanked := response.Ranked[0]
	fmt.Printf("Top Recommendation: %s (Score: %.3f)\n", topRanked.TherapyID, topRanked.FinalScore)
	
	fmt.Println("\nScore Contributions:")
	for _, contrib := range topRanked.Contributions {
		fmt.Printf("- %s: %.3f (weight: %.2f, contribution: %.3f)\n",
			contrib.Factor, contrib.Value, contrib.Weight, contrib.Contribution)
		fmt.Printf("  Note: %s\n", contrib.Note)
	}
	
	fmt.Println("\nClinical Notes:")
	for _, note := range topRanked.Notes {
		fmt.Printf("- %s\n", note)
	}
	
	if !topRanked.EligibilityFlags.TopSlotEligible {
		fmt.Println("\nEligibility Concerns:")
		for _, reason := range topRanked.EligibilityFlags.Reasons {
			fmt.Printf("- %s\n", reason)
		}
	}
}
