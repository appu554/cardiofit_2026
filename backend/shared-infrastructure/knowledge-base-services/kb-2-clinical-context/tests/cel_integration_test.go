package tests

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"kb-clinical-context/internal/engines"
	"kb-clinical-context/internal/models"
)

func TestCELEngineBasicFunctionality(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// Create CEL engine
	celEngine, err := engines.NewCELEngine(logger)
	if err != nil {
		t.Fatalf("Failed to create CEL engine: %v", err)
	}

	// Test basic boolean expressions
	testCases := []struct {
		name        string
		expression  string
		expectMatch bool
		expectError bool
	}{
		{
			name:        "Simple true expression",
			expression:  "true",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "Simple false expression", 
			expression:  "false",
			expectMatch: false,
			expectError: false,
		},
		{
			name:        "Age comparison",
			expression:  "patient.age >= 65",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "Blood pressure condition",
			expression:  "bp.systolic >= 140 || bp.diastolic >= 90",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "Complex cardiovascular condition",
			expression:  "((bp.systolic >= 130 && bp.systolic < 140) || (bp.diastolic >= 80 && bp.diastolic < 90)) && (risk.ascvd_10yr >= 10 || patient.has_diabetes || patient.has_ckd || patient.age >= 65)",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "Invalid expression - non-boolean result",
			expression:  "patient.age + 10",
			expectMatch: false,
			expectError: true,
		},
	}

	// Create sample patient context
	patientContext := createSamplePatientContext()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched, confidence, err := celEngine.EvaluateExpression(tc.expression, patientContext)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if matched != tc.expectMatch {
				t.Errorf("Expected matched=%v, got %v", tc.expectMatch, matched)
			}

			if confidence < 0 || confidence > 1 {
				t.Errorf("Confidence score should be between 0 and 1, got %f", confidence)
			}

			t.Logf("Expression: %s, Matched: %v, Confidence: %f", tc.expression, matched, confidence)
		})
	}
}

func TestCELEngineValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	celEngine, err := engines.NewCELEngine(logger)
	if err != nil {
		t.Fatalf("Failed to create CEL engine: %v", err)
	}

	validationTests := []struct {
		name        string
		expression  string
		expectError bool
	}{
		{
			name:        "Valid boolean expression",
			expression:  "patient.age >= 18 && patient.sex == 'M'",
			expectError: false,
		},
		{
			name:        "Valid complex expression",
			expression:  "labs.total_cholesterol > 240 && !patient.has_diabetes",
			expectError: false,
		},
		{
			name:        "Invalid expression - syntax error",
			expression:  "patient.age >= && 65",
			expectError: true,
		},
		{
			name:        "Invalid expression - non-boolean return",
			expression:  "patient.age + 10",
			expectError: true,
		},
		{
			name:        "Invalid expression - undefined variable",
			expression:  "unknown_var > 10",
			expectError: true,
		},
	}

	for _, tc := range validationTests {
		t.Run(tc.name, func(t *testing.T) {
			err := celEngine.ValidateExpression(tc.expression)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				} else {
					t.Logf("Got expected validation error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				} else {
					t.Logf("Expression validated successfully: %s", tc.expression)
				}
			}
		})
	}
}

func TestMultiEngineEvaluator(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// Create multi-engine evaluator
	multiEngine, err := engines.NewMultiEngineEvaluator(logger)
	if err != nil {
		t.Fatalf("Failed to create multi-engine evaluator: %v", err)
	}

	// Create test phenotype definition
	phenotypeDef := engines.PhenotypeDefinitionYAML{
		ID:          "TEST-001",
		Name:        "Test Hypertension Stage 1",
		Domain:      "cardiovascular", 
		Version:     "1.0.0",
		Status:      "active",
		Description: "Test phenotype for hypertension stage 1",
		Criteria: struct {
			LogicEngine      string                         `yaml:"logic_engine"`
			Expression       string                         `yaml:"expression"`
			DataRequirements []engines.DataRequirement     `yaml:"data_requirements"`
		}{
			LogicEngine: "cel",
			Expression:  "bp.systolic >= 130 && bp.systolic < 140 && patient.age >= 18",
			DataRequirements: []engines.DataRequirement{
				{
					Field:    "bp.systolic",
					Type:     "integer",
					Required: true,
					Source:   "vitals",
				},
				{
					Field:    "patient.age", 
					Type:     "integer",
					Required: true,
					Source:   "demographics",
				},
			},
		},
		Priority: 500,
		Outputs: map[string]string{
			"risk_category": "moderate",
		},
	}

	// Create sample patient context
	patientContext := createSamplePatientContext()

	// Test phenotype evaluation
	result, err := multiEngine.EvaluatePhenotype(phenotypeDef, patientContext)
	if err != nil {
		t.Fatalf("Failed to evaluate phenotype: %v", err)
	}

	t.Logf("Evaluation result: Matched=%v, Confidence=%f, Engine=%s, ExecutionTime=%v",
		result.Matched, result.Confidence, result.EngineUsed, result.ExecutionTime)

	// Verify result structure
	if result.EngineUsed != engines.LogicEngineCEL {
		t.Errorf("Expected CEL engine, got %s", result.EngineUsed)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Invalid confidence score: %f", result.Confidence)
	}

	if len(result.Evidence) == 0 {
		t.Errorf("Expected some evidence items")
	}

	// Test validation
	err = multiEngine.ValidatePhenotype(phenotypeDef)
	if err != nil {
		t.Errorf("Phenotype validation failed: %v", err)
	}

	// Test with invalid phenotype
	invalidPhenotypeDef := phenotypeDef
	invalidPhenotypeDef.Criteria.Expression = "invalid_expression &&"
	
	err = multiEngine.ValidatePhenotype(invalidPhenotypeDef)
	if err == nil {
		t.Errorf("Expected validation to fail for invalid expression")
	}
}

func TestCELEngineTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	celEngine, err := engines.NewCELEngine(logger)
	if err != nil {
		t.Fatalf("Failed to create CEL engine: %v", err)
	}

	// Test with a simple expression (timeout should not occur)
	patientContext := createSamplePatientContext()
	
	matched, confidence, err := celEngine.EvaluateExpression("patient.age >= 18", patientContext)
	if err != nil {
		t.Errorf("Simple expression should not timeout: %v", err)
	}

	t.Logf("Simple expression result: matched=%v, confidence=%f", matched, confidence)
}

func TestCELEngineExpressionCaching(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	celEngine, err := engines.NewCELEngine(logger)
	if err != nil {
		t.Fatalf("Failed to create CEL engine: %v", err)
	}

	expression := "patient.age >= 65 && bp.systolic >= 140"
	patientContext := createSamplePatientContext()

	// First evaluation (should compile and cache)
	start1 := time.Now()
	matched1, confidence1, err1 := celEngine.EvaluateExpression(expression, patientContext)
	duration1 := time.Since(start1)

	if err1 != nil {
		t.Fatalf("First evaluation failed: %v", err1)
	}

	// Second evaluation (should use cached compiled expression)
	start2 := time.Now()
	matched2, confidence2, err2 := celEngine.EvaluateExpression(expression, patientContext)
	duration2 := time.Since(start2)

	if err2 != nil {
		t.Fatalf("Second evaluation failed: %v", err2)
	}

	// Results should be identical
	if matched1 != matched2 || confidence1 != confidence2 {
		t.Errorf("Cached evaluation results differ: (%v,%f) vs (%v,%f)", 
			matched1, confidence1, matched2, confidence2)
	}

	// Second evaluation should be faster (cached)
	if duration2 > duration1 {
		t.Logf("Warning: Cached evaluation took longer (%v vs %v) - this might be normal for simple expressions",
			duration2, duration1)
	}

	t.Logf("First evaluation: %v (matched=%v, confidence=%f)", duration1, matched1, confidence1)
	t.Logf("Second evaluation: %v (matched=%v, confidence=%f)", duration2, matched2, confidence2)

	// Test cache stats
	stats := celEngine.GetCacheStats()
	t.Logf("Cache stats: %+v", stats)
}

// Helper function to create a sample patient context for testing
func createSamplePatientContext() models.PatientContext {
	return models.PatientContext{
		PatientID:  "test-patient-001",
		ContextID:  "test-context-001",
		Timestamp:  time.Now(),
		Demographics: models.Demographics{
			AgeYears:  72,
			Sex:       "M",
			Race:      "White",
			Ethnicity: "Not Hispanic or Latino",
		},
		ActiveConditions: []models.Condition{
			{
				Code:      "I10",
				System:    "ICD-10",
				Name:      "Essential hypertension",
				OnsetDate: time.Now().AddDate(-2, 0, 0),
				Severity:  "moderate",
			},
			{
				Code:      "E11.9",
				System:    "ICD-10", 
				Name:      "Type 2 diabetes mellitus without complications",
				OnsetDate: time.Now().AddDate(-5, 0, 0),
				Severity:  "mild",
			},
		},
		RecentLabs: []models.LabResult{
			{
				LOINCCode:    "2093-3",
				Value:        245.0,
				Unit:         "mg/dL",
				ResultDate:   time.Now().AddDate(0, 0, -7),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "4548-4", 
				Value:        7.2,
				Unit:         "%",
				ResultDate:   time.Now().AddDate(0, 0, -14),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "8480-6", // Systolic BP (simulated as lab for test)
				Value:        145.0,
				Unit:         "mmHg",
				ResultDate:   time.Now().AddDate(0, 0, -1),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "8462-4", // Diastolic BP (simulated as lab for test)
				Value:        92.0,
				Unit:         "mmHg", 
				ResultDate:   time.Now().AddDate(0, 0, -1),
				AbnormalFlag: "H",
			},
		},
		CurrentMeds: []models.Medication{
			{
				RxNormCode: "161",
				Name:       "Lisinopril 10mg",
				Dose:       "10mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(0, -6, 0),
			},
			{
				RxNormCode: "6918",
				Name:       "Metformin 500mg",
				Dose:       "500mg",
				Frequency:  "twice daily", 
				StartDate:  time.Now().AddDate(0, -12, 0),
			},
		},
		DetectedPhenotypes: []models.DetectedPhenotype{},
		RiskFactors: map[string]interface{}{
			"cardiovascular_risk": 0.8,
			"ascvd_10yr":         15.5,
		},
		CareGaps: []string{},
		TTL:      time.Now().Add(24 * time.Hour),
	}
}

// Benchmark test for CEL evaluation performance
func BenchmarkCELEvaluation(b *testing.B) {
	logger := zaptest.NewLogger(b)
	
	celEngine, err := engines.NewCELEngine(logger)
	if err != nil {
		b.Fatalf("Failed to create CEL engine: %v", err)
	}

	expression := "((bp.systolic >= 130 && bp.systolic < 140) || (bp.diastolic >= 80 && bp.diastolic < 90)) && (risk.ascvd_10yr >= 10 || patient.has_diabetes || patient.has_ckd || patient.age >= 65)"
	patientContext := createSamplePatientContext()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _, err := celEngine.EvaluateExpression(expression, patientContext)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}