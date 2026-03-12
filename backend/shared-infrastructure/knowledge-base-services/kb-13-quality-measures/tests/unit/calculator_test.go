// Package unit provides unit tests for KB-13 Quality Measures Engine components.
package unit

import (
	"testing"
	"math"

	"kb-13-quality-measures/internal/models"
)

// TestScoreCalculation_Proportion tests proportion scoring (most common).
func TestScoreCalculation_Proportion(t *testing.T) {
	testCases := []struct {
		name                 string
		numerator            int
		denominator          int
		denomExclusion       int
		denomException       int
		numerExclusion       int
		expectedScore        float64
		expectedPercentage   float64
	}{
		{
			name:               "BasicScore",
			numerator:          8,
			denominator:        10,
			denomExclusion:     0,
			denomException:     0,
			numerExclusion:     0,
			expectedScore:      0.8,
			expectedPercentage: 80.0,
		},
		{
			name:               "WithExclusions",
			numerator:          5,
			denominator:        10,
			denomExclusion:     2,
			denomException:     0,
			numerExclusion:     0,
			expectedScore:      0.625, // 5 / (10-2) = 5/8
			expectedPercentage: 62.5,
		},
		{
			name:               "WithExceptions",
			numerator:          7,
			denominator:        12,
			denomExclusion:     2,
			denomException:     2,
			numerExclusion:     0,
			expectedScore:      0.875, // 7 / (12-2-2) = 7/8
			expectedPercentage: 87.5,
		},
		{
			name:               "WithNumeratorExclusion",
			numerator:          6,
			denominator:        10,
			denomExclusion:     0,
			denomException:     0,
			numerExclusion:     1,
			expectedScore:      0.5, // (6-1) / 10 = 5/10
			expectedPercentage: 50.0,
		},
		{
			name:               "AllExclusions",
			numerator:          8,
			denominator:        15,
			denomExclusion:     3,
			denomException:     2,
			numerExclusion:     2,
			expectedScore:      0.6, // (8-2) / (15-3-2) = 6/10
			expectedPercentage: 60.0,
		},
		{
			name:               "PerfectScore",
			numerator:          100,
			denominator:        100,
			denomExclusion:     0,
			denomException:     0,
			numerExclusion:     0,
			expectedScore:      1.0,
			expectedPercentage: 100.0,
		},
		{
			name:               "ZeroScore",
			numerator:          0,
			denominator:        100,
			denomExclusion:     0,
			denomException:     0,
			numerExclusion:     0,
			expectedScore:      0.0,
			expectedPercentage: 0.0,
		},
		{
			name:               "ZeroDenominator",
			numerator:          0,
			denominator:        0,
			denomExclusion:     0,
			denomException:     0,
			numerExclusion:     0,
			expectedScore:      0.0, // Handle div-by-zero gracefully
			expectedPercentage: 0.0,
		},
		{
			name:               "AllExcluded",
			numerator:          5,
			denominator:        10,
			denomExclusion:     10, // All excluded
			denomException:     0,
			numerExclusion:     0,
			expectedScore:      0.0, // Adjusted denom is 0
			expectedPercentage: 0.0,
		},
		{
			name:               "CMS122Example",
			numerator:          5,
			denominator:        8,
			denomExclusion:     0,  // Denominator already excludes 2 from original 10
			denomException:     0,
			numerExclusion:     0,
			expectedScore:      0.625, // 5 / 8 = 0.625 (denominator is post-exclusion count)
			expectedPercentage: 62.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate score calculation logic
			adjustedDenom := tc.denominator - tc.denomExclusion - tc.denomException
			adjustedNum := tc.numerator - tc.numerExclusion

			var score float64
			if adjustedDenom > 0 {
				score = float64(adjustedNum) / float64(adjustedDenom)
			}

			tolerance := 0.001
			if math.Abs(score-tc.expectedScore) > tolerance {
				t.Errorf("Score: got %.4f, expected %.4f", score, tc.expectedScore)
			}

			percentage := score * 100
			if math.Abs(percentage-tc.expectedPercentage) > 0.1 {
				t.Errorf("Percentage: got %.2f, expected %.2f", percentage, tc.expectedPercentage)
			}
		})
	}
}

// TestCalculationResult_Structure tests the CalculationResult model.
func TestCalculationResult_Structure(t *testing.T) {
	result := &models.CalculationResult{
		ID:                   "test-result-001",
		MeasureID:            "CMS122v12",
		ReportType:           models.ReportSummary,
		InitialPopulation:    100,
		Denominator:          90,
		DenominatorExclusion: 10,
		DenominatorException: 5,
		Numerator:            60,
		NumeratorExclusion:   2,
		Score:                0.773, // (60-2) / (90-10-5) = 58/75
	}

	t.Run("ID", func(t *testing.T) {
		if result.ID == "" {
			t.Error("Result ID should not be empty")
		}
	})

	t.Run("MeasureID", func(t *testing.T) {
		if result.MeasureID != "CMS122v12" {
			t.Errorf("MeasureID: got %s, expected CMS122v12", result.MeasureID)
		}
	})

	t.Run("ReportType", func(t *testing.T) {
		if result.ReportType != models.ReportSummary {
			t.Errorf("ReportType: got %s, expected summary", result.ReportType)
		}
	})

	t.Run("PopulationConsistency", func(t *testing.T) {
		// Initial population >= Denominator
		if result.InitialPopulation < result.Denominator {
			t.Error("Initial population should be >= denominator")
		}

		// Denominator >= Numerator
		if result.Denominator < result.Numerator {
			t.Error("Denominator should be >= numerator")
		}
	})

	t.Run("ScoreRange", func(t *testing.T) {
		if result.Score < 0 || result.Score > 1 {
			t.Errorf("Score should be between 0 and 1, got %.4f", result.Score)
		}
	})
}

// TestStratificationResult tests stratification result calculation.
func TestStratificationResult(t *testing.T) {
	testCases := []struct {
		name          string
		stratID       string
		component     string
		denominator   int
		numerator     int
		expectedScore float64
	}{
		{
			name:          "Age18-44",
			stratID:       "age-18-44",
			component:     "18-44",
			denominator:   30,
			numerator:     20,
			expectedScore: 0.667,
		},
		{
			name:          "Age45-64",
			stratID:       "age-45-64",
			component:     "45-64",
			denominator:   40,
			numerator:     28,
			expectedScore: 0.7,
		},
		{
			name:          "Age65-75",
			stratID:       "age-65-75",
			component:     "65-75",
			denominator:   30,
			numerator:     15,
			expectedScore: 0.5,
		},
		{
			name:          "EmptyStratum",
			stratID:       "age-0-17",
			component:     "0-17",
			denominator:   0,
			numerator:     0,
			expectedScore: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strat := models.StratificationResult{
				StratificationID: tc.stratID,
				Component:        tc.component,
				Denominator:      tc.denominator,
				Numerator:        tc.numerator,
			}

			// Calculate score for stratification
			if strat.Denominator > 0 {
				strat.Score = float64(strat.Numerator) / float64(strat.Denominator)
			}

			tolerance := 0.001
			if math.Abs(strat.Score-tc.expectedScore) > tolerance {
				t.Errorf("Stratification score: got %.4f, expected %.4f", strat.Score, tc.expectedScore)
			}
		})
	}
}

// TestPopulationType_Values tests population type constants.
func TestPopulationType_Values(t *testing.T) {
	expectedTypes := map[models.PopulationType]string{
		models.PopulationInitial:              "initial-population",
		models.PopulationDenominator:          "denominator",
		models.PopulationDenominatorExclusion: "denominator-exclusion",
		models.PopulationDenominatorException: "denominator-exception",
		models.PopulationNumerator:            "numerator",
		models.PopulationNumeratorExclusion:   "numerator-exclusion",
	}

	for popType, expectedStr := range expectedTypes {
		if string(popType) != expectedStr {
			t.Errorf("PopulationType %v: got %s, expected %s", popType, string(popType), expectedStr)
		}
	}
}

// TestScoringType_Values tests scoring type constants.
func TestScoringType_Values(t *testing.T) {
	expectedTypes := map[models.ScoringType]string{
		models.ScoringProportion: "proportion",
		models.ScoringRatio:      "ratio",
		models.ScoringContinuous: "continuous",
		models.ScoringComposite:  "composite",
	}

	for scoringType, expectedStr := range expectedTypes {
		if string(scoringType) != expectedStr {
			t.Errorf("ScoringType %v: got %s, expected %s", scoringType, string(scoringType), expectedStr)
		}
	}
}

// TestMeasureType_Values tests measure type constants.
func TestMeasureType_Values(t *testing.T) {
	expectedTypes := map[models.MeasureType]string{
		models.MeasureTypeProcess:      "PROCESS",
		models.MeasureTypeOutcome:      "OUTCOME",
		models.MeasureTypeStructure:    "STRUCTURE",
		models.MeasureTypeEfficiency:   "EFFICIENCY",
		models.MeasureTypeComposite:    "COMPOSITE",
		models.MeasureTypeIntermediate: "INTERMEDIATE",
	}

	for measureType, expectedStr := range expectedTypes {
		if string(measureType) != expectedStr {
			t.Errorf("MeasureType %v: got %s, expected %s", measureType, string(measureType), expectedStr)
		}
	}
}

// TestClinicalDomain_Values tests clinical domain constants.
func TestClinicalDomain_Values(t *testing.T) {
	domains := []models.ClinicalDomain{
		models.DomainDiabetes,
		models.DomainCardiovascular,
		models.DomainRespiratory,
		models.DomainPreventive,
		models.DomainBehavioralHealth,
		models.DomainMaternal,
		models.DomainPediatric,
		models.DomainPatientSafety,
	}

	for _, domain := range domains {
		if string(domain) == "" {
			t.Errorf("Clinical domain should not be empty: %v", domain)
		}
	}
}

// TestQualityProgram_Values tests quality program constants.
func TestQualityProgram_Values(t *testing.T) {
	programs := map[models.QualityProgram]string{
		models.ProgramHEDIS:  "HEDIS",
		models.ProgramCMS:    "CMS",
		models.ProgramMIPS:   "MIPS",
		models.ProgramACO:    "ACO",
		models.ProgramPCMH:   "PCMH",
		models.ProgramNQF:    "NQF",
		models.ProgramCustom: "CUSTOM",
	}

	for program, expectedStr := range programs {
		if string(program) != expectedStr {
			t.Errorf("QualityProgram %v: got %s, expected %s", program, string(program), expectedStr)
		}
	}
}

// TestCareGap_NewCareGap tests care gap creation.
func TestCareGap_NewCareGap(t *testing.T) {
	gap := models.NewCareGap(
		"CMS122v12",
		"patient-123",
		"missing-hba1c",
		"No HbA1c test in measurement period",
		models.PriorityHigh,
	)

	t.Run("MeasureID", func(t *testing.T) {
		if gap.MeasureID != "CMS122v12" {
			t.Errorf("MeasureID: got %s, expected CMS122v12", gap.MeasureID)
		}
	})

	t.Run("SubjectID", func(t *testing.T) {
		if gap.SubjectID != "patient-123" {
			t.Errorf("SubjectID: got %s, expected patient-123", gap.SubjectID)
		}
	})

	t.Run("GapType", func(t *testing.T) {
		if gap.GapType != "missing-hba1c" {
			t.Errorf("GapType: got %s, expected missing-hba1c", gap.GapType)
		}
	})

	t.Run("Priority", func(t *testing.T) {
		if gap.Priority != models.PriorityHigh {
			t.Errorf("Priority: got %s, expected high", gap.Priority)
		}
	})

	t.Run("Status", func(t *testing.T) {
		if gap.Status != models.CareGapStatusOpen {
			t.Errorf("Status: got %s, expected open", gap.Status)
		}
	})

	// 🔴 CRITICAL: KB-13 care gaps are NOT authoritative
	t.Run("SourceIsQualityMeasure", func(t *testing.T) {
		if gap.Source != models.CareGapSourceQualityMeasure {
			t.Errorf("Source: got %s, expected QUALITY_MEASURE", gap.Source)
		}
	})

	t.Run("IsNotAuthoritative", func(t *testing.T) {
		if gap.IsAuthoritative {
			t.Error("KB-13 care gaps should NOT be authoritative")
		}
	})

	t.Run("TimestampsSet", func(t *testing.T) {
		if gap.CreatedAt.IsZero() {
			t.Error("CreatedAt should be set")
		}
		if gap.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should be set")
		}
	})
}

// TestCareGapPriority_Values tests care gap priority levels.
func TestCareGapPriority_Values(t *testing.T) {
	priorities := []struct {
		priority models.Priority
		expected string
	}{
		{models.PriorityCritical, "critical"},
		{models.PriorityHigh, "high"},
		{models.PriorityMedium, "medium"},
		{models.PriorityLow, "low"},
	}

	for _, tc := range priorities {
		if string(tc.priority) != tc.expected {
			t.Errorf("Priority %v: got %s, expected %s", tc.priority, string(tc.priority), tc.expected)
		}
	}
}

// TestCareGapStatus_Values tests care gap status values.
func TestCareGapStatus_Values(t *testing.T) {
	statuses := []struct {
		status   models.CareGapStatus
		expected string
	}{
		{models.CareGapStatusOpen, "open"},
		{models.CareGapStatusInProgress, "in-progress"},
		{models.CareGapStatusClosed, "closed"},
		{models.CareGapStatusDeferred, "deferred"},
	}

	for _, tc := range statuses {
		if string(tc.status) != tc.expected {
			t.Errorf("Status %v: got %s, expected %s", tc.status, string(tc.status), tc.expected)
		}
	}
}

// TestCareGapSource_CriticalDistinction tests the critical KB-13 vs KB-9 distinction.
func TestCareGapSource_CriticalDistinction(t *testing.T) {
	t.Run("QualityMeasureSource", func(t *testing.T) {
		if string(models.CareGapSourceQualityMeasure) != "QUALITY_MEASURE" {
			t.Errorf("QualityMeasure source should be QUALITY_MEASURE")
		}
	})

	t.Run("PatientCDSSource", func(t *testing.T) {
		if string(models.CareGapSourcePatientCDS) != "PATIENT_CDS" {
			t.Errorf("PatientCDS source should be PATIENT_CDS")
		}
	})

	// 🔴 CRITICAL: Verify KB-13 gaps are created with correct source
	t.Run("KB13GapsAreNotAuthoritative", func(t *testing.T) {
		gap := models.NewCareGap("CMS165", "patient-456", "bp-gap", "Missing BP", models.PriorityMedium)

		if gap.Source != models.CareGapSourceQualityMeasure {
			t.Error("KB-13 gaps MUST have QUALITY_MEASURE source")
		}

		if gap.IsAuthoritative {
			t.Error("KB-13 gaps MUST NOT be authoritative (KB-9 is authoritative)")
		}
	})
}

// TestMeasure_Structure tests measure model structure.
func TestMeasure_Structure(t *testing.T) {
	measure := &models.Measure{
		ID:          "CMS122v12",
		Version:     "12.0.000",
		Name:        "cms122-diabetes-hba1c",
		Title:       "Diabetes: Hemoglobin A1c Poor Control (>9%)",
		Description: "Percentage of patients with diabetes with HbA1c poor control",
		Type:        models.MeasureTypeProcess,
		Scoring:     models.ScoringProportion,
		Domain:      models.DomainDiabetes,
		Program:     models.ProgramCMS,
		NQFNumber:   "0059",
		CMSNumber:   "CMS122",
		MeasurementPeriod: models.MeasurementPeriod{
			Type:     "calendar",
			Duration: "P1Y",
			Anchor:   "year",
		},
		ImprovementNotation: "decrease", // Lower is better for poor control
		Active:              true,
	}

	t.Run("RequiredFields", func(t *testing.T) {
		if measure.ID == "" {
			t.Error("ID should not be empty")
		}
		if measure.Name == "" {
			t.Error("Name should not be empty")
		}
		if measure.Type == "" {
			t.Error("Type should not be empty")
		}
		if measure.Domain == "" {
			t.Error("Domain should not be empty")
		}
		if measure.Program == "" {
			t.Error("Program should not be empty")
		}
	})

	t.Run("MeasurementPeriod", func(t *testing.T) {
		if measure.MeasurementPeriod.Type == "" {
			t.Error("MeasurementPeriod.Type should not be empty")
		}
		if measure.MeasurementPeriod.Duration == "" {
			t.Error("MeasurementPeriod.Duration should not be empty")
		}
	})
}

// TestExecutionContextVersion tests the audit context structure.
func TestExecutionContextVersion(t *testing.T) {
	// 🟡 REQUIRED per CTO/CMO gate: All calculations must include execution context
	ctx := models.ExecutionContextVersion{
		KB13Version:        "1.0.0",
		CQLLibraryVersion:  "3.4.0",
		TerminologyVersion: "2024-01",
		MeasureYAMLVersion: "12.0.000",
	}

	t.Run("KB13Version", func(t *testing.T) {
		if ctx.KB13Version == "" {
			t.Error("KB13Version should be set for audit")
		}
	})

	t.Run("CQLLibraryVersion", func(t *testing.T) {
		if ctx.CQLLibraryVersion == "" {
			t.Error("CQLLibraryVersion should be set for audit")
		}
	})

	t.Run("TerminologyVersion", func(t *testing.T) {
		if ctx.TerminologyVersion == "" {
			t.Error("TerminologyVersion should be set for audit (KB-7 version)")
		}
	})

	t.Run("MeasureYAMLVersion", func(t *testing.T) {
		if ctx.MeasureYAMLVersion == "" {
			t.Error("MeasureYAMLVersion should be set for audit")
		}
	})
}

// TestReportType_Values tests report type constants.
func TestReportType_Values(t *testing.T) {
	types := []struct {
		reportType models.ReportType
		expected   string
	}{
		{models.ReportIndividual, "individual"},
		{models.ReportSubjectList, "subject-list"},
		{models.ReportSummary, "summary"},
		{models.ReportDataExchange, "data-exchange"},
	}

	for _, tc := range types {
		if string(tc.reportType) != tc.expected {
			t.Errorf("ReportType %v: got %s, expected %s", tc.reportType, string(tc.reportType), tc.expected)
		}
	}
}
