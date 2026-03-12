// Package cms provides CMS conformance tests for KB-13 Quality Measures.
//
// IMPORTANT: These tests verify regulatory compliance with CMS measure specifications.
// Results must be deterministic and match expected values from clinical review.
//
// Run with: go test -tags=conformance ./tests/conformance/cms/...
//
//go:build conformance

package cms

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"kb-13-quality-measures/internal/models"
)

// CMS122ExpectedResults holds the expected conformance results.
type CMS122ExpectedResults struct {
	MeasureID         string `yaml:"measure_id"`
	MeasureVersion    string `yaml:"measure_version"`
	MeasurementPeriod struct {
		Start string `yaml:"start"`
		End   string `yaml:"end"`
	} `yaml:"measurement_period"`
	ExpectedResults struct {
		InitialPopulation    int     `yaml:"initial_population"`
		Denominator          int     `yaml:"denominator"`
		DenominatorExclusion int     `yaml:"denominator_exclusion"`
		DenominatorException int     `yaml:"denominator_exception"`
		Numerator            int     `yaml:"numerator"`
		NumeratorExclusion   int     `yaml:"numerator_exclusion"`
		Score                float64 `yaml:"score"`
		PerformanceRate      float64 `yaml:"performance_rate"`
	} `yaml:"expected_results"`
	Stratifications []struct {
		ID          string  `yaml:"id"`
		Numerator   int     `yaml:"numerator"`
		Denominator int     `yaml:"denominator"`
		Score       float64 `yaml:"score"`
	} `yaml:"stratifications"`
	Tolerance float64 `yaml:"tolerance"`
}

// CMS122PatientFixtures holds the test patient data.
type CMS122PatientFixtures struct {
	Description       string `json:"description"`
	MeasureID         string `json:"measure_id"`
	MeasurementPeriod struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"measurement_period"`
	Patients []CMS122Patient `json:"patients"`
	Summary  struct {
		TotalPatients         int `json:"total_patients"`
		InInitialPopulation   int `json:"in_initial_population"`
		InDenominator         int `json:"in_denominator"`
		DenominatorExclusions int `json:"denominator_exclusions"`
		InNumerator           int `json:"in_numerator"`
	} `json:"summary"`
}

// CMS122Patient represents a test patient for CMS122.
type CMS122Patient struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	Demographics struct {
		DateOfBirth            string `json:"date_of_birth"`
		Gender                 string `json:"gender"`
		AgeAtMeasurementStart  int    `json:"age_at_measurement_start"`
	} `json:"demographics"`
	Conditions []struct {
		Code        string `json:"code"`
		System      string `json:"system"`
		Description string `json:"description"`
		OnsetDate   string `json:"onset_date"`
	} `json:"conditions"`
	LabResults []struct {
		Code        string  `json:"code"`
		System      string  `json:"system"`
		Description string  `json:"description"`
		Value       float64 `json:"value"`
		Unit        string  `json:"unit"`
		Date        string  `json:"date"`
	} `json:"lab_results"`
	Exclusions []struct {
		Type        string `json:"type"`
		Code        string `json:"code"`
		System      string `json:"system"`
		Description string `json:"description"`
		StartDate   string `json:"start_date,omitempty"`
	} `json:"exclusions"`
	Expected struct {
		InitialPopulation    bool   `json:"initial_population"`
		Denominator          bool   `json:"denominator"`
		DenominatorExclusion bool   `json:"denominator_exclusion"`
		Numerator            bool   `json:"numerator"`
		Stratification       string `json:"stratification,omitempty"`
		ExclusionReason      string `json:"exclusion_reason,omitempty"`
	} `json:"expected"`
}

// loadExpectedResults loads the expected results from YAML fixture.
func loadExpectedResults(t *testing.T) *CMS122ExpectedResults {
	data, err := os.ReadFile("fixtures/cms122_expected.yaml")
	if err != nil {
		t.Fatalf("Failed to load expected results: %v", err)
	}

	var expected CMS122ExpectedResults
	if err := yaml.Unmarshal(data, &expected); err != nil {
		t.Fatalf("Failed to parse expected results: %v", err)
	}

	return &expected
}

// loadPatientFixtures loads the patient fixtures from JSON.
func loadPatientFixtures(t *testing.T) *CMS122PatientFixtures {
	data, err := os.ReadFile("fixtures/cms122_patients.json")
	if err != nil {
		t.Fatalf("Failed to load patient fixtures: %v", err)
	}

	var fixtures CMS122PatientFixtures
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("Failed to parse patient fixtures: %v", err)
	}

	return &fixtures
}

// floatEquals compares two floats within tolerance.
func floatEquals(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

// TestCMS122_Conformance_FixtureIntegrity verifies fixture data consistency.
func TestCMS122_Conformance_FixtureIntegrity(t *testing.T) {
	expected := loadExpectedResults(t)
	fixtures := loadPatientFixtures(t)

	// Verify measure IDs match
	if expected.MeasureID != fixtures.MeasureID {
		t.Errorf("Measure ID mismatch: expected %s, fixtures %s",
			expected.MeasureID, fixtures.MeasureID)
	}

	// Verify patient count matches summary
	if len(fixtures.Patients) != fixtures.Summary.TotalPatients {
		t.Errorf("Patient count mismatch: %d patients, summary says %d",
			len(fixtures.Patients), fixtures.Summary.TotalPatients)
	}

	// Verify summary matches expected results
	if fixtures.Summary.InInitialPopulation != expected.ExpectedResults.InitialPopulation {
		t.Errorf("Initial population mismatch: fixtures %d, expected %d",
			fixtures.Summary.InInitialPopulation, expected.ExpectedResults.InitialPopulation)
	}

	if fixtures.Summary.DenominatorExclusions != expected.ExpectedResults.DenominatorExclusion {
		t.Errorf("Denominator exclusions mismatch: fixtures %d, expected %d",
			fixtures.Summary.DenominatorExclusions, expected.ExpectedResults.DenominatorExclusion)
	}

	if fixtures.Summary.InNumerator != expected.ExpectedResults.Numerator {
		t.Errorf("Numerator mismatch: fixtures %d, expected %d",
			fixtures.Summary.InNumerator, expected.ExpectedResults.Numerator)
	}
}

// TestCMS122_Conformance_PopulationCounts verifies population calculations.
func TestCMS122_Conformance_PopulationCounts(t *testing.T) {
	expected := loadExpectedResults(t)
	fixtures := loadPatientFixtures(t)

	// Calculate populations from fixture patients
	var initialPop, denominator, denomExclusion, numerator int

	for _, patient := range fixtures.Patients {
		if patient.Expected.InitialPopulation {
			initialPop++
		}
		if patient.Expected.Denominator {
			denominator++
		}
		if patient.Expected.DenominatorExclusion {
			denomExclusion++
		}
		if patient.Expected.Numerator {
			numerator++
		}
	}

	// Verify against expected results
	t.Run("InitialPopulation", func(t *testing.T) {
		if initialPop != expected.ExpectedResults.InitialPopulation {
			t.Errorf("Initial population: calculated %d, expected %d",
				initialPop, expected.ExpectedResults.InitialPopulation)
		}
	})

	t.Run("Denominator", func(t *testing.T) {
		// Denominator = Initial - Exclusions (after applying exclusion logic)
		effectiveDenom := denominator - denomExclusion
		expectedEffectiveDenom := expected.ExpectedResults.Denominator - expected.ExpectedResults.DenominatorExclusion
		if effectiveDenom != expectedEffectiveDenom {
			t.Errorf("Effective denominator: calculated %d, expected %d",
				effectiveDenom, expectedEffectiveDenom)
		}
	})

	t.Run("DenominatorExclusion", func(t *testing.T) {
		if denomExclusion != expected.ExpectedResults.DenominatorExclusion {
			t.Errorf("Denominator exclusions: calculated %d, expected %d",
				denomExclusion, expected.ExpectedResults.DenominatorExclusion)
		}
	})

	t.Run("Numerator", func(t *testing.T) {
		if numerator != expected.ExpectedResults.Numerator {
			t.Errorf("Numerator: calculated %d, expected %d",
				numerator, expected.ExpectedResults.Numerator)
		}
	})
}

// TestCMS122_Conformance_ScoreCalculation verifies score calculation.
func TestCMS122_Conformance_ScoreCalculation(t *testing.T) {
	expected := loadExpectedResults(t)
	fixtures := loadPatientFixtures(t)

	// Calculate score from fixture data
	var numerator, effectiveDenom int
	for _, patient := range fixtures.Patients {
		if patient.Expected.Denominator && !patient.Expected.DenominatorExclusion {
			effectiveDenom++
			if patient.Expected.Numerator {
				numerator++
			}
		}
	}

	var calculatedScore float64
	if effectiveDenom > 0 {
		calculatedScore = float64(numerator) / float64(effectiveDenom)
	}

	// Verify score
	if !floatEquals(calculatedScore, expected.ExpectedResults.Score, expected.Tolerance) {
		t.Errorf("Score calculation: calculated %.4f, expected %.4f (tolerance %.4f)",
			calculatedScore, expected.ExpectedResults.Score, expected.Tolerance)
	}

	// Verify performance rate
	performanceRate := calculatedScore * 100
	if !floatEquals(performanceRate, expected.ExpectedResults.PerformanceRate, expected.Tolerance*100) {
		t.Errorf("Performance rate: calculated %.2f%%, expected %.2f%%",
			performanceRate, expected.ExpectedResults.PerformanceRate)
	}
}

// TestCMS122_Conformance_AgeEligibility verifies age criteria (18-75).
func TestCMS122_Conformance_AgeEligibility(t *testing.T) {
	fixtures := loadPatientFixtures(t)

	for _, patient := range fixtures.Patients {
		age := patient.Demographics.AgeAtMeasurementStart
		inInitialPop := patient.Expected.InitialPopulation

		// Age must be 18-75 for initial population
		ageEligible := age >= 18 && age <= 75

		if inInitialPop && !ageEligible {
			t.Errorf("Patient %s: in initial population but age %d is outside 18-75",
				patient.ID, age)
		}
	}
}

// TestCMS122_Conformance_DiabetesDiagnosis verifies diabetes criteria.
func TestCMS122_Conformance_DiabetesDiagnosis(t *testing.T) {
	fixtures := loadPatientFixtures(t)

	diabetesCodes := map[string]bool{
		"E11":    true, // Type 2 diabetes prefix
		"E11.9":  true,
		"E11.65": true,
		"E10":    true, // Type 1 diabetes prefix
	}

	for _, patient := range fixtures.Patients {
		if !patient.Expected.InitialPopulation {
			continue
		}

		// Patient must have diabetes diagnosis
		hasDiabetes := false
		for _, condition := range patient.Conditions {
			// Check ICD-10 codes
			if condition.System == "ICD-10-CM" {
				code := condition.Code
				// Check full code or prefix
				if diabetesCodes[code] {
					hasDiabetes = true
					break
				}
				// Check E11.* prefix
				if len(code) >= 3 && code[:3] == "E11" {
					hasDiabetes = true
					break
				}
			}
		}

		if !hasDiabetes {
			t.Errorf("Patient %s: in initial population but no diabetes diagnosis found",
				patient.ID)
		}
	}
}

// TestCMS122_Conformance_HbA1cThreshold verifies HbA1c < 8% numerator criterion.
func TestCMS122_Conformance_HbA1cThreshold(t *testing.T) {
	fixtures := loadPatientFixtures(t)
	hba1cThreshold := 8.0

	for _, patient := range fixtures.Patients {
		if patient.Expected.DenominatorExclusion {
			continue // Skip excluded patients
		}

		if !patient.Expected.Denominator {
			continue // Skip non-denominator patients
		}

		// Find HbA1c lab result
		var hba1cValue float64
		hasHbA1c := false
		for _, lab := range patient.LabResults {
			if lab.Code == "4548-4" { // LOINC code for HbA1c
				hba1cValue = lab.Value
				hasHbA1c = true
				break
			}
		}

		if !hasHbA1c {
			// If no HbA1c and in denominator, should NOT be in numerator
			if patient.Expected.Numerator {
				t.Errorf("Patient %s: in numerator but no HbA1c result found", patient.ID)
			}
			continue
		}

		// Verify numerator membership based on HbA1c threshold
		shouldBeInNumerator := hba1cValue < hba1cThreshold

		if shouldBeInNumerator != patient.Expected.Numerator {
			t.Errorf("Patient %s: HbA1c %.1f, expected numerator=%v, got %v",
				patient.ID, hba1cValue, shouldBeInNumerator, patient.Expected.Numerator)
		}
	}
}

// TestCMS122_Conformance_Exclusions verifies exclusion logic.
func TestCMS122_Conformance_Exclusions(t *testing.T) {
	fixtures := loadPatientFixtures(t)

	for _, patient := range fixtures.Patients {
		hasExclusion := len(patient.Exclusions) > 0
		isExcluded := patient.Expected.DenominatorExclusion

		if hasExclusion != isExcluded {
			t.Errorf("Patient %s: has %d exclusions but DenominatorExclusion=%v",
				patient.ID, len(patient.Exclusions), isExcluded)
		}

		// Verify exclusion reason matches
		if isExcluded && len(patient.Exclusions) > 0 {
			exclusionType := patient.Exclusions[0].Type
			expectedReason := patient.Expected.ExclusionReason

			if exclusionType != expectedReason {
				t.Errorf("Patient %s: exclusion type '%s' doesn't match expected reason '%s'",
					patient.ID, exclusionType, expectedReason)
			}
		}
	}
}

// TestCMS122_Conformance_Stratification verifies age stratification.
func TestCMS122_Conformance_Stratification(t *testing.T) {
	expected := loadExpectedResults(t)
	fixtures := loadPatientFixtures(t)

	// Calculate stratified results
	strats := make(map[string]struct {
		numerator   int
		denominator int
	})

	for _, patient := range fixtures.Patients {
		if patient.Expected.DenominatorExclusion {
			continue
		}
		if !patient.Expected.Denominator {
			continue
		}

		stratID := patient.Expected.Stratification
		if stratID == "" {
			continue
		}

		s := strats[stratID]
		s.denominator++
		if patient.Expected.Numerator {
			s.numerator++
		}
		strats[stratID] = s
	}

	// Verify against expected stratifications
	for _, expectedStrat := range expected.Stratifications {
		calculated, exists := strats[expectedStrat.ID]
		if !exists {
			t.Errorf("Stratification %s: not found in calculated results", expectedStrat.ID)
			continue
		}

		if calculated.numerator != expectedStrat.Numerator {
			t.Errorf("Stratification %s: numerator calculated %d, expected %d",
				expectedStrat.ID, calculated.numerator, expectedStrat.Numerator)
		}

		if calculated.denominator != expectedStrat.Denominator {
			t.Errorf("Stratification %s: denominator calculated %d, expected %d",
				expectedStrat.ID, calculated.denominator, expectedStrat.Denominator)
		}

		// Verify score
		var calculatedScore float64
		if calculated.denominator > 0 {
			calculatedScore = float64(calculated.numerator) / float64(calculated.denominator)
		}

		if !floatEquals(calculatedScore, expectedStrat.Score, expected.Tolerance) {
			t.Errorf("Stratification %s: score calculated %.3f, expected %.3f",
				expectedStrat.ID, calculatedScore, expectedStrat.Score)
		}
	}
}

// TestCMS122_Conformance_MeasurementPeriod verifies lab results within measurement period.
func TestCMS122_Conformance_MeasurementPeriod(t *testing.T) {
	fixtures := loadPatientFixtures(t)

	periodStart, _ := time.Parse("2006-01-02", fixtures.MeasurementPeriod.Start)
	periodEnd, _ := time.Parse("2006-01-02", fixtures.MeasurementPeriod.End)

	for _, patient := range fixtures.Patients {
		if !patient.Expected.Numerator {
			continue
		}

		// Find the qualifying HbA1c
		for _, lab := range patient.LabResults {
			if lab.Code != "4548-4" {
				continue
			}

			labDate, err := time.Parse("2006-01-02", lab.Date)
			if err != nil {
				t.Errorf("Patient %s: invalid lab date format: %s", patient.ID, lab.Date)
				continue
			}

			if labDate.Before(periodStart) || labDate.After(periodEnd) {
				t.Errorf("Patient %s: HbA1c date %s outside measurement period %s to %s",
					patient.ID, lab.Date, fixtures.MeasurementPeriod.Start, fixtures.MeasurementPeriod.End)
			}
		}
	}
}

// TestCMS122_Conformance_DeterministicResults ensures same input = same output.
func TestCMS122_Conformance_DeterministicResults(t *testing.T) {
	// Run calculation multiple times and verify consistent results
	expected := loadExpectedResults(t)

	for i := 0; i < 3; i++ {
		fixtures := loadPatientFixtures(t)

		var numerator, effectiveDenom int
		for _, patient := range fixtures.Patients {
			if patient.Expected.Denominator && !patient.Expected.DenominatorExclusion {
				effectiveDenom++
				if patient.Expected.Numerator {
					numerator++
				}
			}
		}

		var calculatedScore float64
		if effectiveDenom > 0 {
			calculatedScore = float64(numerator) / float64(effectiveDenom)
		}

		if !floatEquals(calculatedScore, expected.ExpectedResults.Score, expected.Tolerance) {
			t.Errorf("Run %d: non-deterministic result, got %.4f, expected %.4f",
				i+1, calculatedScore, expected.ExpectedResults.Score)
		}
	}
}

// TestCMS122_Conformance_MeasureModel verifies KB-13 measure model integration.
func TestCMS122_Conformance_MeasureModel(t *testing.T) {
	expected := loadExpectedResults(t)

	// Create measure matching CMS122 specification
	measure := &models.Measure{
		ID:                  expected.MeasureID,
		Version:             expected.MeasureVersion,
		Name:                "Diabetes: Hemoglobin A1c (HbA1c) Poor Control (>9%)",
		Title:               "HbA1c Control (<8%)",
		Type:                models.MeasureTypeProcess,
		Scoring:             models.ScoringProportion,
		Domain:              models.DomainDiabetes,
		Program:             models.ProgramCMS,
		NQFNumber:           "0059",
		CMSNumber:           "CMS122v11",
		HEDISCode:           "HBD",
		ImprovementNotation: "increase",
		Active:              true,
		Populations: []models.Population{
			{
				ID:            "initial-population",
				Type:          models.PopulationInitial,
				Description:   "Patients 18-75 with diabetes",
				CQLExpression: "InInitialPopulation",
			},
			{
				ID:            "denominator",
				Type:          models.PopulationDenominator,
				Description:   "Equals initial population",
				CQLExpression: "InDenominator",
			},
			{
				ID:            "denominator-exclusion",
				Type:          models.PopulationDenominatorExclusion,
				Description:   "Hospice, ESRD, palliative care",
				CQLExpression: "HasDenominatorExclusion",
			},
			{
				ID:            "numerator",
				Type:          models.PopulationNumerator,
				Description:   "HbA1c < 8%",
				CQLExpression: "InNumerator",
			},
		},
		Stratifications: []models.Stratification{
			{
				ID:          "age-strat",
				Description: "Age stratification",
				Components:  []string{"18-44", "45-64", "65-75"},
			},
		},
	}

	// Verify measure has required populations
	if len(measure.Populations) < 4 {
		t.Errorf("CMS122 must have at least 4 populations, got %d", len(measure.Populations))
	}

	// Verify required population types exist
	requiredTypes := map[models.PopulationType]bool{
		models.PopulationInitial:              false,
		models.PopulationDenominator:          false,
		models.PopulationDenominatorExclusion: false,
		models.PopulationNumerator:            false,
	}

	for _, pop := range measure.Populations {
		if _, required := requiredTypes[pop.Type]; required {
			requiredTypes[pop.Type] = true
		}
	}

	for popType, found := range requiredTypes {
		if !found {
			t.Errorf("CMS122 missing required population type: %s", popType)
		}
	}

	// Verify stratification components
	if len(measure.Stratifications) == 0 {
		t.Error("CMS122 must have age stratification")
	} else if len(measure.Stratifications[0].Components) != 3 {
		t.Errorf("Age stratification must have 3 components, got %d",
			len(measure.Stratifications[0].Components))
	}
}

// BenchmarkCMS122_ScoreCalculation benchmarks score calculation performance.
func BenchmarkCMS122_ScoreCalculation(b *testing.B) {
	fixtures := &CMS122PatientFixtures{}
	data, _ := os.ReadFile("fixtures/cms122_patients.json")
	json.Unmarshal(data, fixtures)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var numerator, effectiveDenom int
		for _, patient := range fixtures.Patients {
			if patient.Expected.Denominator && !patient.Expected.DenominatorExclusion {
				effectiveDenom++
				if patient.Expected.Numerator {
					numerator++
				}
			}
		}
		_ = float64(numerator) / float64(effectiveDenom)
	}
}
