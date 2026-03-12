//go:build conformance
// +build conformance

// Package cms provides CMS quality measure conformance tests.
// These tests validate regulatory compliance against fixed patient fixtures.
//
// CMS165 - Controlling High Blood Pressure
// NQF: 0018
// Measure Steward: NCQA
//
// IMPORTANT: These are regulatory conformance tests. Do not modify expected
// values without clinical governance approval. Any changes require:
// 1. Clinical review and sign-off
// 2. Updated fixture documentation
// 3. Audit trail entry
package cms

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// CMS165ExpectedResults holds expected conformance values for CMS165.
type CMS165ExpectedResults struct {
	MeasureID         string `yaml:"measure_id"`
	MeasureName       string `yaml:"measure_name"`
	MeasureVersion    string `yaml:"measure_version"`
	MeasurementYear   int    `yaml:"measurement_year"`
	MeasurementPeriod struct {
		Start string `yaml:"start"`
		End   string `yaml:"end"`
	} `yaml:"measurement_period"`
	PopulationCriteria struct {
		InitialPopulation struct {
			Description      string   `yaml:"description"`
			AgeMinimum       int      `yaml:"age_minimum"`
			AgeMaximum       int      `yaml:"age_maximum"`
			RequiredDiagnosis struct {
				CodeSystem string   `yaml:"code_system"`
				Codes      []string `yaml:"codes"`
			} `yaml:"required_diagnosis"`
			DiagnosisMustBeBefore string `yaml:"diagnosis_must_be_before"`
		} `yaml:"initial_population"`
		Numerator struct {
			Description       string `yaml:"description"`
			SystolicThreshold int    `yaml:"systolic_threshold"`
			DiastolicThreshold int   `yaml:"diastolic_threshold"`
		} `yaml:"numerator"`
	} `yaml:"population_criteria"`
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
		ID                   string  `yaml:"id"`
		Description          string  `yaml:"description"`
		InitialPopulation    int     `yaml:"initial_population"`
		Denominator          int     `yaml:"denominator"`
		DenominatorExclusion int     `yaml:"denominator_exclusion"`
		Numerator            int     `yaml:"numerator"`
		Score                float64 `yaml:"score"`
	} `yaml:"stratifications"`
	Validation struct {
		ScoreTolerance float64 `yaml:"score_tolerance"`
	} `yaml:"validation"`
}

// CMS165PatientFixtures holds the patient test data for CMS165.
type CMS165PatientFixtures struct {
	FixtureMetadata struct {
		MeasureID         string `json:"measure_id"`
		MeasureName       string `json:"measure_name"`
		Description       string `json:"description"`
		Version           string `json:"version"`
		MeasurementPeriod struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"measurement_period"`
		PatientCount int    `json:"patient_count"`
		CreatedAt    string `json:"created_at"`
	} `json:"fixture_metadata"`
	Patients []CMS165Patient `json:"patients"`
	Summary  struct {
		TotalPatients          int     `json:"total_patients"`
		InitialPopulation      int     `json:"initial_population"`
		Denominator            int     `json:"denominator"`
		DenominatorExclusions  int     `json:"denominator_exclusions"`
		Numerator              int     `json:"numerator"`
		ExpectedScore          float64 `json:"expected_score"`
		StratificationBreakdown map[string]struct {
			IP     int `json:"ip"`
			Denom  int `json:"denom"`
			Excl   int `json:"excl"`
			Numer  int `json:"numer"`
		} `json:"stratification_breakdown"`
	} `json:"summary"`
}

// CMS165Patient represents a test patient for CMS165.
type CMS165Patient struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	Demographics struct {
		DateOfBirth             string `json:"date_of_birth"`
		Gender                  string `json:"gender"`
		AgeAtMeasurementStart   int    `json:"age_at_measurement_start"`
	} `json:"demographics"`
	Conditions []struct {
		Code      string `json:"code"`
		System    string `json:"system"`
		Display   string `json:"display"`
		OnsetDate string `json:"onset_date"`
	} `json:"conditions"`
	VitalSigns []struct {
		Type     string `json:"type"`
		Systolic struct {
			Code   string  `json:"code"`
			System string  `json:"system"`
			Value  float64 `json:"value"`
			Unit   string  `json:"unit"`
		} `json:"systolic"`
		Diastolic struct {
			Code   string  `json:"code"`
			System string  `json:"system"`
			Value  float64 `json:"value"`
			Unit   string  `json:"unit"`
		} `json:"diastolic"`
		Date string `json:"date"`
	} `json:"vital_signs"`
	Exclusions []string `json:"exclusions"`
	Expected   struct {
		InitialPopulation    bool   `json:"initial_population"`
		Denominator          bool   `json:"denominator"`
		DenominatorExclusion bool   `json:"denominator_exclusion"`
		Numerator            bool   `json:"numerator"`
		Stratification       string `json:"stratification"`
		Rationale            string `json:"rationale"`
	} `json:"expected"`
}

// loadCMS165ExpectedResults loads the expected results from YAML fixture.
func loadCMS165ExpectedResults(t *testing.T) *CMS165ExpectedResults {
	t.Helper()

	fixtureDir := filepath.Join("fixtures")
	yamlPath := filepath.Join(fixtureDir, "cms165_expected.yaml")

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load CMS165 expected results: %v", err)
	}

	var expected CMS165ExpectedResults
	if err := yaml.Unmarshal(data, &expected); err != nil {
		t.Fatalf("Failed to parse CMS165 expected results: %v", err)
	}

	return &expected
}

// loadCMS165PatientFixtures loads the patient fixtures from JSON.
func loadCMS165PatientFixtures(t *testing.T) *CMS165PatientFixtures {
	t.Helper()

	fixtureDir := filepath.Join("fixtures")
	jsonPath := filepath.Join(fixtureDir, "cms165_patients.json")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to load CMS165 patient fixtures: %v", err)
	}

	var fixtures CMS165PatientFixtures
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("Failed to parse CMS165 patient fixtures: %v", err)
	}

	return &fixtures
}

// TestCMS165_Conformance_FixtureIntegrity verifies fixture data consistency.
func TestCMS165_Conformance_FixtureIntegrity(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)
	fixtures := loadCMS165PatientFixtures(t)

	t.Run("MeasureIDMatch", func(t *testing.T) {
		if expected.MeasureID != fixtures.FixtureMetadata.MeasureID {
			t.Errorf("Measure ID mismatch: expected %s, fixtures %s",
				expected.MeasureID, fixtures.FixtureMetadata.MeasureID)
		}
	})

	t.Run("PatientCountMatch", func(t *testing.T) {
		if len(fixtures.Patients) != fixtures.FixtureMetadata.PatientCount {
			t.Errorf("Patient count mismatch: metadata says %d, actual %d",
				fixtures.FixtureMetadata.PatientCount, len(fixtures.Patients))
		}
	})

	t.Run("SummaryConsistency", func(t *testing.T) {
		// Verify summary matches expected
		if fixtures.Summary.InitialPopulation != expected.ExpectedResults.InitialPopulation {
			t.Errorf("Initial population mismatch: summary %d, expected %d",
				fixtures.Summary.InitialPopulation, expected.ExpectedResults.InitialPopulation)
		}
		if fixtures.Summary.Numerator != expected.ExpectedResults.Numerator {
			t.Errorf("Numerator mismatch: summary %d, expected %d",
				fixtures.Summary.Numerator, expected.ExpectedResults.Numerator)
		}
	})

	t.Run("UniquePatientIDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for _, p := range fixtures.Patients {
			if ids[p.ID] {
				t.Errorf("Duplicate patient ID: %s", p.ID)
			}
			ids[p.ID] = true
		}
	})

	t.Run("AllPatientsHaveExpectedFields", func(t *testing.T) {
		for _, p := range fixtures.Patients {
			if p.Demographics.DateOfBirth == "" {
				t.Errorf("Patient %s missing date of birth", p.ID)
			}
			if len(p.VitalSigns) == 0 {
				t.Errorf("Patient %s has no vital signs", p.ID)
			}
			if p.Expected.Stratification == "" {
				t.Errorf("Patient %s missing stratification assignment", p.ID)
			}
		}
	})
}

// TestCMS165_Conformance_PopulationCounts verifies population calculations.
func TestCMS165_Conformance_PopulationCounts(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)
	fixtures := loadCMS165PatientFixtures(t)

	// Count populations from patient expected values
	var ip, denom, denomExcl, numer int
	for _, p := range fixtures.Patients {
		if p.Expected.InitialPopulation {
			ip++
		}
		if p.Expected.Denominator {
			denom++
		}
		if p.Expected.DenominatorExclusion {
			denomExcl++
		}
		if p.Expected.Numerator {
			numer++
		}
	}

	t.Run("InitialPopulation", func(t *testing.T) {
		if ip != expected.ExpectedResults.InitialPopulation {
			t.Errorf("Initial population: got %d, expected %d", ip, expected.ExpectedResults.InitialPopulation)
		}
	})

	t.Run("Denominator", func(t *testing.T) {
		// Denominator should equal IP minus exclusions for CMS165
		expectedDenom := expected.ExpectedResults.Denominator - expected.ExpectedResults.DenominatorExclusion
		actualDenom := denom - denomExcl
		if actualDenom != expectedDenom {
			t.Errorf("Effective denominator: got %d, expected %d", actualDenom, expectedDenom)
		}
	})

	t.Run("DenominatorExclusion", func(t *testing.T) {
		if denomExcl != expected.ExpectedResults.DenominatorExclusion {
			t.Errorf("Denominator exclusions: got %d, expected %d", denomExcl, expected.ExpectedResults.DenominatorExclusion)
		}
	})

	t.Run("Numerator", func(t *testing.T) {
		if numer != expected.ExpectedResults.Numerator {
			t.Errorf("Numerator: got %d, expected %d", numer, expected.ExpectedResults.Numerator)
		}
	})
}

// TestCMS165_Conformance_ScoreCalculation verifies the score calculation.
func TestCMS165_Conformance_ScoreCalculation(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)
	fixtures := loadCMS165PatientFixtures(t)

	// Calculate score: Numerator / (Denominator - Exclusions - Exceptions)
	effectiveDenom := expected.ExpectedResults.Denominator -
		expected.ExpectedResults.DenominatorExclusion -
		expected.ExpectedResults.DenominatorException

	var calculatedScore float64
	if effectiveDenom > 0 {
		calculatedScore = float64(expected.ExpectedResults.Numerator) / float64(effectiveDenom)
	}

	tolerance := expected.Validation.ScoreTolerance
	if tolerance == 0 {
		tolerance = 0.001
	}

	t.Run("ScoreMatchesExpected", func(t *testing.T) {
		if math.Abs(calculatedScore-expected.ExpectedResults.Score) > tolerance {
			t.Errorf("Score calculation: got %.4f, expected %.4f (tolerance %.4f)",
				calculatedScore, expected.ExpectedResults.Score, tolerance)
		}
	})

	t.Run("ScoreMatchesFixtureSummary", func(t *testing.T) {
		if math.Abs(fixtures.Summary.ExpectedScore-expected.ExpectedResults.Score) > tolerance {
			t.Errorf("Fixture summary score mismatch: got %.4f, expected %.4f",
				fixtures.Summary.ExpectedScore, expected.ExpectedResults.Score)
		}
	})

	t.Run("PerformanceRateConsistent", func(t *testing.T) {
		expectedPerfRate := expected.ExpectedResults.Score * 100
		if math.Abs(expectedPerfRate-expected.ExpectedResults.PerformanceRate) > 0.1 {
			t.Errorf("Performance rate inconsistent with score: %.2f%% vs %.2f%%",
				expected.ExpectedResults.PerformanceRate, expectedPerfRate)
		}
	})
}

// TestCMS165_Conformance_AgeEligibility verifies age-based eligibility.
func TestCMS165_Conformance_AgeEligibility(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)
	fixtures := loadCMS165PatientFixtures(t)

	minAge := expected.PopulationCriteria.InitialPopulation.AgeMinimum
	maxAge := expected.PopulationCriteria.InitialPopulation.AgeMaximum

	for _, p := range fixtures.Patients {
		t.Run(p.ID, func(t *testing.T) {
			age := p.Demographics.AgeAtMeasurementStart

			// Verify age is within valid range for initial population
			if p.Expected.InitialPopulation {
				if age < minAge || age > maxAge {
					t.Errorf("Patient %s age %d outside valid range [%d-%d] but in initial population",
						p.ID, age, minAge, maxAge)
				}
			}

			// Verify correct stratification assignment
			var expectedStrat string
			switch {
			case age >= 18 && age <= 44:
				expectedStrat = "age-18-44"
			case age >= 45 && age <= 64:
				expectedStrat = "age-45-64"
			case age >= 65 && age <= 85:
				expectedStrat = "age-65-85"
			}

			if p.Expected.Stratification != expectedStrat && p.Expected.InitialPopulation {
				t.Errorf("Patient %s age %d stratification: got %s, expected %s",
					p.ID, age, p.Expected.Stratification, expectedStrat)
			}
		})
	}
}

// TestCMS165_Conformance_HypertensionDiagnosis verifies HTN diagnosis requirement.
func TestCMS165_Conformance_HypertensionDiagnosis(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)
	fixtures := loadCMS165PatientFixtures(t)

	validHTNCodes := expected.PopulationCriteria.InitialPopulation.RequiredDiagnosis.Codes
	diagnosisCutoff := expected.PopulationCriteria.InitialPopulation.DiagnosisMustBeBefore

	for _, p := range fixtures.Patients {
		t.Run(p.ID, func(t *testing.T) {
			if !p.Expected.InitialPopulation {
				return // Skip patients not in initial population
			}

			// Check for valid HTN diagnosis
			hasValidHTN := false
			var earliestHTNDate string

			for _, cond := range p.Conditions {
				// Check if condition matches any valid HTN code pattern
				for _, validCode := range validHTNCodes {
					if strings.HasSuffix(validCode, ".*") {
						prefix := strings.TrimSuffix(validCode, ".*")
						if strings.HasPrefix(cond.Code, prefix) {
							hasValidHTN = true
							if earliestHTNDate == "" || cond.OnsetDate < earliestHTNDate {
								earliestHTNDate = cond.OnsetDate
							}
						}
					} else if cond.Code == validCode {
						hasValidHTN = true
						if earliestHTNDate == "" || cond.OnsetDate < earliestHTNDate {
							earliestHTNDate = cond.OnsetDate
						}
					}
				}
			}

			if !hasValidHTN {
				t.Errorf("Patient %s in initial population but no valid HTN diagnosis", p.ID)
			}

			// Verify diagnosis before cutoff date
			if diagnosisCutoff != "" && earliestHTNDate >= diagnosisCutoff {
				t.Errorf("Patient %s HTN diagnosis date %s not before cutoff %s",
					p.ID, earliestHTNDate, diagnosisCutoff)
			}
		})
	}
}

// TestCMS165_Conformance_BPThreshold verifies BP threshold logic.
func TestCMS165_Conformance_BPThreshold(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)
	fixtures := loadCMS165PatientFixtures(t)

	systolicThreshold := expected.PopulationCriteria.Numerator.SystolicThreshold
	diastolicThreshold := expected.PopulationCriteria.Numerator.DiastolicThreshold

	for _, p := range fixtures.Patients {
		t.Run(p.ID, func(t *testing.T) {
			// Skip excluded patients
			if p.Expected.DenominatorExclusion {
				return
			}

			if len(p.VitalSigns) == 0 {
				return
			}

			// Find most recent BP
			var latestBP *struct {
				Type     string `json:"type"`
				Systolic struct {
					Code   string  `json:"code"`
					System string  `json:"system"`
					Value  float64 `json:"value"`
					Unit   string  `json:"unit"`
				} `json:"systolic"`
				Diastolic struct {
					Code   string  `json:"code"`
					System string  `json:"system"`
					Value  float64 `json:"value"`
					Unit   string  `json:"unit"`
				} `json:"diastolic"`
				Date string `json:"date"`
			}
			var latestDate string

			for i := range p.VitalSigns {
				vs := &p.VitalSigns[i]
				if vs.Type == "blood_pressure" {
					if latestDate == "" || vs.Date > latestDate {
						latestDate = vs.Date
						latestBP = vs
					}
				}
			}

			if latestBP == nil {
				if p.Expected.Numerator {
					t.Errorf("Patient %s expected in numerator but no BP found", p.ID)
				}
				return
			}

			systolic := latestBP.Systolic.Value
			diastolic := latestBP.Diastolic.Value

			// BP must be < 140/90 for numerator inclusion
			bpControlled := systolic < float64(systolicThreshold) && diastolic < float64(diastolicThreshold)

			if p.Expected.Numerator && !bpControlled {
				t.Errorf("Patient %s expected in numerator but BP %.0f/%.0f is not controlled (<140/90)",
					p.ID, systolic, diastolic)
			}

			if !p.Expected.Numerator && bpControlled && !p.Expected.DenominatorExclusion {
				t.Errorf("Patient %s BP %.0f/%.0f is controlled but not in numerator",
					p.ID, systolic, diastolic)
			}
		})
	}
}

// TestCMS165_Conformance_Exclusions verifies exclusion criteria.
func TestCMS165_Conformance_Exclusions(t *testing.T) {
	fixtures := loadCMS165PatientFixtures(t)

	exclusionCodes := map[string]string{
		"Z33.1": "pregnancy",
		"N18.6": "esrd",
		"Z99.2": "dialysis",
		"Z94.0": "kidney_transplant",
		"Z51.5": "hospice",
	}

	for _, p := range fixtures.Patients {
		t.Run(p.ID, func(t *testing.T) {
			// Check if patient has any exclusion conditions
			hasExclusionCondition := false
			for _, cond := range p.Conditions {
				if _, isExclusion := exclusionCodes[cond.Code]; isExclusion {
					hasExclusionCondition = true
					break
				}
			}

			// Verify exclusion status matches conditions
			if hasExclusionCondition && !p.Expected.DenominatorExclusion {
				t.Errorf("Patient %s has exclusion condition but not marked as excluded", p.ID)
			}

			// Verify exclusion array matches expected status
			if len(p.Exclusions) > 0 && !p.Expected.DenominatorExclusion {
				t.Errorf("Patient %s has exclusions %v but not marked as excluded",
					p.ID, p.Exclusions)
			}

			// Excluded patients should not be in numerator
			if p.Expected.DenominatorExclusion && p.Expected.Numerator {
				t.Errorf("Patient %s is excluded but also in numerator", p.ID)
			}
		})
	}
}

// TestCMS165_Conformance_Stratification verifies age stratification counts.
func TestCMS165_Conformance_Stratification(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)
	fixtures := loadCMS165PatientFixtures(t)

	// Count patients by stratification
	stratCounts := make(map[string]struct {
		ip, denom, excl, numer int
	})

	for _, p := range fixtures.Patients {
		strat := p.Expected.Stratification
		counts := stratCounts[strat]

		if p.Expected.InitialPopulation {
			counts.ip++
		}
		if p.Expected.Denominator {
			counts.denom++
		}
		if p.Expected.DenominatorExclusion {
			counts.excl++
		}
		if p.Expected.Numerator {
			counts.numer++
		}

		stratCounts[strat] = counts
	}

	for _, expStrat := range expected.Stratifications {
		t.Run(expStrat.ID, func(t *testing.T) {
			actual, exists := stratCounts[expStrat.ID]
			if !exists {
				t.Fatalf("Stratification %s not found in patient data", expStrat.ID)
			}

			if actual.ip != expStrat.InitialPopulation {
				t.Errorf("Initial population: got %d, expected %d", actual.ip, expStrat.InitialPopulation)
			}

			if actual.excl != expStrat.DenominatorExclusion {
				t.Errorf("Denominator exclusions: got %d, expected %d", actual.excl, expStrat.DenominatorExclusion)
			}

			if actual.numer != expStrat.Numerator {
				t.Errorf("Numerator: got %d, expected %d", actual.numer, expStrat.Numerator)
			}

			// Verify stratification score
			effectiveDenom := expStrat.Denominator - expStrat.DenominatorExclusion
			if effectiveDenom > 0 {
				expectedScore := float64(expStrat.Numerator) / float64(effectiveDenom)
				if math.Abs(expectedScore-expStrat.Score) > 0.01 {
					t.Errorf("Score mismatch: calculated %.3f, expected %.3f",
						expectedScore, expStrat.Score)
				}
			}
		})
	}
}

// TestCMS165_Conformance_MeasurementPeriod verifies BP dates are within period.
func TestCMS165_Conformance_MeasurementPeriod(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)
	fixtures := loadCMS165PatientFixtures(t)

	periodStart, _ := time.Parse("2006-01-02", expected.MeasurementPeriod.Start)
	periodEnd, _ := time.Parse("2006-01-02", expected.MeasurementPeriod.End)

	for _, p := range fixtures.Patients {
		t.Run(p.ID, func(t *testing.T) {
			for _, vs := range p.VitalSigns {
				if vs.Type != "blood_pressure" {
					continue
				}

				bpDate, err := time.Parse("2006-01-02", vs.Date)
				if err != nil {
					t.Errorf("Invalid BP date format: %s", vs.Date)
					continue
				}

				if bpDate.Before(periodStart) || bpDate.After(periodEnd) {
					t.Errorf("BP date %s outside measurement period [%s, %s]",
						vs.Date, expected.MeasurementPeriod.Start, expected.MeasurementPeriod.End)
				}
			}
		})
	}
}

// TestCMS165_Conformance_MostRecentBPOnly verifies only most recent BP is used.
func TestCMS165_Conformance_MostRecentBPOnly(t *testing.T) {
	fixtures := loadCMS165PatientFixtures(t)

	for _, p := range fixtures.Patients {
		t.Run(p.ID, func(t *testing.T) {
			if len(p.VitalSigns) < 2 {
				return // Need multiple BPs to test this
			}

			// Find most recent BP
			var latestDate string
			var latestBP struct {
				systolic, diastolic float64
			}

			for _, vs := range p.VitalSigns {
				if vs.Type == "blood_pressure" {
					if latestDate == "" || vs.Date > latestDate {
						latestDate = vs.Date
						latestBP.systolic = vs.Systolic.Value
						latestBP.diastolic = vs.Diastolic.Value
					}
				}
			}

			// Check first BP (older) - if controlled but patient not in numerator,
			// that's correct because most recent is used
			if len(p.VitalSigns) > 1 && !p.Expected.DenominatorExclusion {
				firstBP := p.VitalSigns[0]
				if firstBP.Date != latestDate {
					firstControlled := firstBP.Systolic.Value < 140 && firstBP.Diastolic.Value < 90
					latestControlled := latestBP.systolic < 140 && latestBP.diastolic < 90

					// If statuses differ, verify numerator reflects most recent
					if firstControlled != latestControlled {
						if p.Expected.Numerator != latestControlled {
							t.Errorf("Numerator status should reflect most recent BP (date %s), not older readings",
								latestDate)
						}
					}
				}
			}
		})
	}
}

// TestCMS165_Conformance_DeterministicResults verifies deterministic output.
func TestCMS165_Conformance_DeterministicResults(t *testing.T) {
	// Load fixtures multiple times and verify identical results
	expected1 := loadCMS165ExpectedResults(t)
	expected2 := loadCMS165ExpectedResults(t)

	fixtures1 := loadCMS165PatientFixtures(t)
	fixtures2 := loadCMS165PatientFixtures(t)

	t.Run("ExpectedResultsIdentical", func(t *testing.T) {
		if expected1.ExpectedResults.Score != expected2.ExpectedResults.Score {
			t.Error("Non-deterministic: Expected results differ between loads")
		}
	})

	t.Run("PatientCountIdentical", func(t *testing.T) {
		if len(fixtures1.Patients) != len(fixtures2.Patients) {
			t.Error("Non-deterministic: Patient count differs between loads")
		}
	})

	t.Run("SummaryIdentical", func(t *testing.T) {
		if fixtures1.Summary.ExpectedScore != fixtures2.Summary.ExpectedScore {
			t.Error("Non-deterministic: Summary scores differ between loads")
		}
	})
}

// TestCMS165_Conformance_MeasureModel verifies KB-13 measure model compatibility.
func TestCMS165_Conformance_MeasureModel(t *testing.T) {
	expected := loadCMS165ExpectedResults(t)

	t.Run("MeasureIDFormat", func(t *testing.T) {
		// CMS measure IDs should follow CMS###v## pattern
		if !strings.HasPrefix(expected.MeasureID, "CMS") {
			t.Errorf("Invalid measure ID format: %s", expected.MeasureID)
		}
	})

	t.Run("MeasureVersionFormat", func(t *testing.T) {
		// Version should be semantic versioning
		parts := strings.Split(expected.MeasureVersion, ".")
		if len(parts) != 3 {
			t.Errorf("Invalid version format: %s (expected x.y.z)", expected.MeasureVersion)
		}
	})

	t.Run("RequiredPopulations", func(t *testing.T) {
		// CMS165 should have all standard populations defined
		results := expected.ExpectedResults
		if results.InitialPopulation == 0 {
			t.Error("Initial population should be defined")
		}
		if results.Denominator == 0 {
			t.Error("Denominator should be defined")
		}
	})

	t.Run("StratificationsDefined", func(t *testing.T) {
		// CMS165 requires age stratification
		if len(expected.Stratifications) == 0 {
			t.Error("Age stratifications should be defined")
		}

		// Verify expected stratification IDs
		expectedStrats := map[string]bool{
			"age-18-44": false,
			"age-45-64": false,
			"age-65-85": false,
		}

		for _, strat := range expected.Stratifications {
			if _, exists := expectedStrats[strat.ID]; exists {
				expectedStrats[strat.ID] = true
			}
		}

		for id, found := range expectedStrats {
			if !found {
				t.Errorf("Missing expected stratification: %s", id)
			}
		}
	})
}

// TestCMS165_Conformance_BothBPComponentsRequired verifies both systolic and diastolic are needed.
func TestCMS165_Conformance_BothBPComponentsRequired(t *testing.T) {
	fixtures := loadCMS165PatientFixtures(t)

	for _, p := range fixtures.Patients {
		t.Run(p.ID, func(t *testing.T) {
			for _, vs := range p.VitalSigns {
				if vs.Type != "blood_pressure" {
					continue
				}

				// Both components must be present
				if vs.Systolic.Value == 0 || vs.Diastolic.Value == 0 {
					t.Errorf("BP reading missing component: systolic=%.0f, diastolic=%.0f",
						vs.Systolic.Value, vs.Diastolic.Value)
				}

				// Both must have correct LOINC codes
				if vs.Systolic.Code != "8480-6" {
					t.Errorf("Invalid systolic LOINC code: %s (expected 8480-6)", vs.Systolic.Code)
				}
				if vs.Diastolic.Code != "8462-4" {
					t.Errorf("Invalid diastolic LOINC code: %s (expected 8462-4)", vs.Diastolic.Code)
				}
			}
		})
	}
}

// TestCMS165_Conformance_BoundaryConditions tests edge cases.
func TestCMS165_Conformance_BoundaryConditions(t *testing.T) {
	fixtures := loadCMS165PatientFixtures(t)

	t.Run("ExactlyAtThreshold", func(t *testing.T) {
		// Find patients with BP exactly at or just below threshold
		for _, p := range fixtures.Patients {
			if p.Expected.DenominatorExclusion {
				continue
			}

			for _, vs := range p.VitalSigns {
				if vs.Type != "blood_pressure" {
					continue
				}

				// Test boundary: 139/89 should be controlled, 140/90 should not
				systolic := vs.Systolic.Value
				diastolic := vs.Diastolic.Value

				// Patient CMS165-P006 has BP 138/86 - should be in numerator
				if p.ID == "CMS165-P006" {
					if systolic >= 140 || diastolic >= 90 {
						t.Errorf("Boundary patient %s BP should be < 140/90 but is %.0f/%.0f",
							p.ID, systolic, diastolic)
					}
					if !p.Expected.Numerator {
						t.Error("Patient CMS165-P006 with BP 138/86 should be in numerator")
					}
				}
			}
		}
	})
}
