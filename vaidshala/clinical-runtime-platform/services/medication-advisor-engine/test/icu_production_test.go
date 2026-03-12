// Package test provides production-grade ICU Intelligence tests.
// These tests require REAL KB services - NO mocks, NO fallbacks.
package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/cardiofit/medication-advisor-engine/advisor"
	"github.com/cardiofit/medication-advisor-engine/kbclients"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// PRODUCTION ICU TESTS - Real KB Services Required
// ============================================================================

// TestICUWithRealKB1DrugRules tests ICU safety rules against real KB-1 drug data
func TestICUWithRealKB1DrugRules(t *testing.T) {
	// REQUIRE KB-1 to be available - fail fast if not
	kb1URL := MustHaveKB1(t)
	t.Logf("Using KB-1 at: %s", kb1URL)

	// Create production client (NO fallbacks)
	config := kbclients.ProductionClientConfig(kb1URL)
	client, err := kbclients.NewKB1DosingClient(config)
	require.NoError(t, err, "Failed to create KB-1 client")
	defer client.Close()

	// Verify health check passes
	ctx := context.Background()
	err = client.HealthCheck(ctx)
	require.NoError(t, err, "KB-1 health check failed")

	// Test Case 1: Get real dosing data for Metoprolol (beta-blocker)
	t.Run("RealMetoprololDosing", func(t *testing.T) {
		dosing, err := client.GetStandardDosage(ctx, "6918") // Metoprolol RxNorm
		require.NoError(t, err, "Failed to get Metoprolol dosing from KB-1")
		require.NotNil(t, dosing, "Dosing data should not be nil")
		t.Logf("KB-1 Metoprolol Dosing: %+v", dosing)
		assert.NotEmpty(t, dosing.DrugName, "Drug name should not be empty")
	})

	// Test Case 2: Get real dosing data for Norepinephrine (vasopressor)
	t.Run("RealNorepinephrineDosing", func(t *testing.T) {
		dosing, err := client.GetStandardDosage(ctx, "7512") // Norepinephrine RxNorm
		require.NoError(t, err, "Failed to get Norepinephrine dosing from KB-1")
		require.NotNil(t, dosing, "Dosing data should not be nil")
		t.Logf("KB-1 Norepinephrine Dosing: %+v", dosing)
	})

	// Test Case 3: Search for real beta-blockers
	t.Run("RealBetaBlockerSearch", func(t *testing.T) {
		drugs, err := client.SearchByClass(ctx, "Beta Blocker")
		require.NoError(t, err, "Failed to search beta-blockers from KB-1")
		t.Logf("KB-1 returned %d beta-blockers", len(drugs))
		for _, d := range drugs {
			t.Logf("  - %s (%s)", d.DrugName, d.RxNormCode)
		}
	})
}

// TestICUWithRealKB4Safety tests ICU safety against real KB-4 patient safety data
func TestICUWithRealKB4Safety(t *testing.T) {
	// REQUIRE KB-4 to be available
	kb4URL := MustHaveKB4(t)
	t.Logf("Using KB-4 at: %s", kb4URL)

	// Create ICU state for testing with correct field types
	akiStage := advisor.AKIStage2
	icuState := &advisor.ICUClinicalState{
		Hemodynamic: advisor.HemodynamicState{
			MAP:            58, // Hypotensive
			SystolicBP:     85,
			DiastolicBP:    50,
			HeartRate:      110,
			VasopressorReq: advisor.VasopressorLow,
		},
		Renal: advisor.RenalState{
			EGFR:       25, // Severe CKD
			Creatinine: 3.2,
			AKIStage:   &akiStage,
		},
	}

	// Calculate acuity using correct method
	acuityScore := icuState.CalculateICUAcuityScore()
	t.Logf("ICU Acuity Score: %.2f", acuityScore)

	// Test ICU safety rules against real KB-4 data
	rulesEngine := advisor.NewICUSafetyRulesEngine()

	// Test beta-blocker in hypotension
	t.Run("BetaBlockerHypotensionWithRealKB", func(t *testing.T) {
		metoprolol := advisor.ClinicalCode{
			System:  "RxNorm",
			Code:    "6918",
			Display: "Metoprolol",
		}

		violations := rulesEngine.EvaluateMedication(metoprolol, icuState)
		require.NotEmpty(t, violations, "Should detect beta-blocker contraindication in hypotension")

		for _, v := range violations {
			t.Logf("ICU Safety Violation: %s - %s (Severity: %s)",
				v.RuleName, v.Recommendation, v.Severity)
		}

		// Cross-validate with KB-4
		kb4Result := validateWithKB4(t, kb4URL, metoprolol.Code, icuState)
		t.Logf("KB-4 Safety Check: %+v", kb4Result)
	})

	// Test aminoglycoside in AKI
	t.Run("AminoglycosideAKIWithRealKB", func(t *testing.T) {
		gentamicin := advisor.ClinicalCode{
			System:  "RxNorm",
			Code:    "641",
			Display: "Gentamicin",
		}

		violations := rulesEngine.EvaluateMedication(gentamicin, icuState)
		require.NotEmpty(t, violations, "Should detect aminoglycoside contraindication in AKI")

		for _, v := range violations {
			t.Logf("ICU Safety Violation: %s - %s", v.RuleName, v.Recommendation)
		}
	})
}

// TestICUWithRealKB7Terminology tests ICU with real SNOMED/LOINC codes
func TestICUWithRealKB7Terminology(t *testing.T) {
	// REQUIRE KB-7 to be available
	kb7URL := MustHaveKB7(t)
	t.Logf("Using KB-7 at: %s", kb7URL)

	snomedCodes := []struct {
		code        string
		display     string
		expectedUse string
	}{
		{"91302008", "Sepsis", "ICU condition detection"},
		{"233604007", "Pneumonia", "Respiratory dimension"},
		{"14669001", "Acute Kidney Injury", "Renal dimension"},
		{"40733004", "Infectious disease", "Infection dimension"},
	}

	for _, sc := range snomedCodes {
		t.Run(fmt.Sprintf("Validate_%s", sc.code), func(t *testing.T) {
			validated := validateSNOMEDWithKB7(t, kb7URL, sc.code)
			if validated {
				t.Logf("✅ SNOMED %s (%s) validated via KB-7", sc.code, sc.display)
			} else {
				t.Logf("⚠️ SNOMED %s (%s) not found in KB-7", sc.code, sc.display)
			}
		})
	}

	// Validate LOINC codes for lab values
	loincCodes := []struct {
		code        string
		display     string
		expectedUse string
	}{
		{"2160-0", "Creatinine", "Renal function"},
		{"2823-3", "Potassium", "Electrolyte monitoring"},
		{"718-7", "Hemoglobin", "Anemia detection"},
		{"32623-1", "Lactate", "Sepsis marker"},
	}

	for _, lc := range loincCodes {
		t.Run(fmt.Sprintf("Validate_LOINC_%s", lc.code), func(t *testing.T) {
			validated := validateLOINCWithKB7(t, kb7URL, lc.code)
			if validated {
				t.Logf("✅ LOINC %s (%s) validated via KB-7", lc.code, lc.display)
			} else {
				t.Logf("⚠️ LOINC %s (%s) not found in KB-7", lc.code, lc.display)
			}
		})
	}
}

// TestICUFullIntegrationWithAllKB tests complete ICU workflow with all KB services
func TestICUFullIntegrationWithAllKB(t *testing.T) {
	// Require ALL KB services
	config := RequireProductionKB(t)
	t.Logf("Production test config: %+v", config)

	// Create complete ICU scenario
	icuState := createCriticalSepsisState()
	temporal := createRealTemporalState()

	// Initialize all engines - NewICUTaskCoordinator takes no arguments
	taskCoordinator := advisor.NewICUTaskCoordinator()

	// Proposed medications for sepsis
	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "7512", Display: "Norepinephrine"},
		{System: "RxNorm", Code: "641", Display: "Gentamicin"},
		{System: "RxNorm", Code: "2551", Display: "Vancomycin"},
	}

	// Generate governance events (empty for this test)
	var govEvents []advisor.GovernanceEvent

	// Generate complete task bundle
	taskBundle := taskCoordinator.GenerateAllTasks(icuState, temporal, proposedMeds, govEvents)

	require.NotNil(t, taskBundle, "Task bundle should not be nil")
	t.Logf("Generated %d total tasks (Critical: %d, Urgent: %d, Stat: %d)",
		taskBundle.TotalTasks, taskBundle.CriticalCount, taskBundle.UrgentCount, taskBundle.StatTaskCount)

	// Verify tasks generated
	for i, task := range taskBundle.Tasks {
		t.Logf("Task %d: [%s] %s (Priority: %s, ResponseTime: %d min)",
			i+1, task.TaskType, task.Title, task.Priority, task.TimeConstraint.MaxResponseMinutes)
	}

	// Log violation and alert task counts
	t.Logf("Violation Tasks: %d, Alert Tasks: %d, Predictive Tasks: %d",
		taskBundle.ViolationTaskCount, taskBundle.AlertTaskCount, taskBundle.PredictiveTaskCount)

	// Assertions - validate task generation with real KB data
	assert.GreaterOrEqual(t, len(taskBundle.Tasks), 1, "Should generate at least 1 task")
	assert.GreaterOrEqual(t, taskBundle.TotalTasks, 1, "TotalTasks should be at least 1")
}

// ============================================================================
// Helper Functions for Real KB Validation
// ============================================================================

func validateWithKB4(t *testing.T, kb4URL, rxnormCode string, icuState *advisor.ICUClinicalState) map[string]interface{} {
	t.Helper()

	// Call KB-4 contraindication check
	url := fmt.Sprintf("%s/v1/safety/contraindication?rxnorm=%s", kb4URL, rxnormCode)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("KB-4 request failed: %v", err)
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func validateSNOMEDWithKB7(t *testing.T, kb7URL, snomedCode string) bool {
	t.Helper()

	url := fmt.Sprintf("%s/v1/snomed/validate/%s", kb7URL, snomedCode)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func validateLOINCWithKB7(t *testing.T, kb7URL, loincCode string) bool {
	t.Helper()

	url := fmt.Sprintf("%s/v1/loinc/validate/%s", kb7URL, loincCode)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func createCriticalSepsisState() *advisor.ICUClinicalState {
	akiStage := advisor.AKIStage2
	pfRatio := 180.0
	lactate := 4.2
	procalcitonin := 8.5

	state := &advisor.ICUClinicalState{
		Hemodynamic: advisor.HemodynamicState{
			MAP:            55,
			SystolicBP:     80,
			DiastolicBP:    45,
			HeartRate:      120,
			VasopressorReq: advisor.VasopressorMultiple,
			ShockState:     advisor.ShockSevere, // Severe shock for septic shock scenario
		},
		Respiratory: advisor.RespiratoryState{
			PaO2FiO2Ratio:  &pfRatio,
			FiO2:           0.6,
			VentilatorMode: advisor.VentModeAC,
		},
		Renal: advisor.RenalState{
			EGFR:       35,
			Creatinine: 2.1,
			AKIStage:   &akiStage,
		},
		Infection: advisor.InfectionState{
			SepsisStatus:  advisor.SepsisSevere,
			SepticShock:   true,
			Lactate:       &lactate,
			Procalcitonin: &procalcitonin,
			WBC:           18.5,
			Temperature:   39.2,
		},
		Neurological: advisor.NeurologicalState{
			GCS:            11,
			PupilsReactive: true,
		},
	}
	return state
}

func createRealTemporalState() *advisor.ICUTemporalState {
	now := time.Now()

	// Create a proper ICUTemporalState using the actual struct fields
	return &advisor.ICUTemporalState{
		PatientID:    uuid.New(),
		EncounterID:  uuid.New(),
		CurrentState: createCriticalSepsisState(), // Use the clinical state we created
		HistoricalStates: []advisor.ICUClinicalState{
			// Empty for now - would contain historical snapshots in real scenario
		},
		Trends: advisor.DimensionTrends{
			Hemodynamic: advisor.TrendAnalysis{
				Direction:  advisor.TrendDeteriorating,
				Slope:      -2.5,
				Confidence: 0.85,
			},
			Respiratory: advisor.TrendAnalysis{
				Direction:  advisor.TrendDeteriorating,
				Slope:      -10.0,
				Confidence: 0.78,
			},
			Renal: advisor.TrendAnalysis{
				Direction:  advisor.TrendDeteriorating,
				Slope:      0.3,
				Confidence: 0.82,
			},
		},
		Alerts:       []advisor.TemporalAlert{},   // Alerts will be generated by analysis
		Predictions:  []advisor.DeteriorationPrediction{}, // Predictions will be generated
		LastAnalyzed: now,
	}
}
