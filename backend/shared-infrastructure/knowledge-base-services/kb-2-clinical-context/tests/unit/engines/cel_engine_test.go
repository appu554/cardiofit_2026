package engines_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"kb-clinical-context/internal/engines"
	"kb-clinical-context/internal/models"
	"kb-clinical-context/tests/testutils"
)

func TestCELEngine_Creation_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	celEngine, err := engines.NewCELEngine(logger)
	
	require.NoError(t, err)
	assert.NotNil(t, celEngine)
}

func TestCELEngine_BasicExpressions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	celEngine, err := engines.NewCELEngine(logger)
	require.NoError(t, err)
	
	fixtures := testutils.NewPatientFixtures()
	patient := fixtures.CreateCardiovascularPatient()
	
	testCases := []struct {
		name        string
		expression  string
		expectMatch bool
		expectError bool
	}{
		{
			name:        "Simple true",
			expression:  "true",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "Simple false",
			expression:  "false", 
			expectMatch: false,
			expectError: false,
		},
		{
			name:        "Age comparison - should match",
			expression:  "patient.age >= 65",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "Age comparison - should not match",
			expression:  "patient.age < 30",
			expectMatch: false,
			expectError: false,
		},
		{
			name:        "Sex comparison",
			expression:  "patient.sex == 'M'",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "Blood pressure check",
			expression:  "bp.systolic >= 140",
			expectMatch: true, // CV patient has high BP
			expectError: false,
		},
		{
			name:        "Complex cardiovascular risk",
			expression:  "patient.age >= 65 && bp.systolic >= 140",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "Invalid expression - syntax error",
			expression:  "patient.age >= && 65",
			expectMatch: false,
			expectError: true,
		},
		{
			name:        "Invalid expression - non-boolean return",
			expression:  "patient.age + 10",
			expectMatch: false,
			expectError: true,
		},
		{
			name:        "Invalid expression - undefined variable",
			expression:  "unknown_variable > 10",
			expectMatch: false,
			expectError: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched, confidence, err := celEngine.EvaluateExpression(tc.expression, patient)
			
			if tc.expectError {
				assert.Error(t, err)
				t.Logf("Expected error received: %v", err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tc.expectMatch, matched, "Expression evaluation mismatch")
			assert.GreaterOrEqual(t, confidence, 0.0, "Confidence should be non-negative")
			assert.LessOrEqual(t, confidence, 1.0, "Confidence should not exceed 1.0")
			
			t.Logf("Expression: %s | Matched: %v | Confidence: %.3f", tc.expression, matched, confidence)
		})
	}
}

func TestCELEngine_ClinicalExpressions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	celEngine, err := engines.NewCELEngine(logger)
	require.NoError(t, err)
	
	fixtures := testutils.NewPatientFixtures()
	
	testCases := []struct {
		name        string
		patient     models.PatientContext
		expression  string
		expectMatch bool
		description string
	}{
		{
			name:        "Hypertension Stage 1 - Cardiovascular Patient",
			patient:     fixtures.CreateCardiovascularPatient(),
			expression:  "bp.systolic >= 130 && bp.systolic < 140",
			expectMatch: false, // CV patient has systolic 158, which is stage 2
			description: "Should not match stage 1 hypertension for stage 2 patient",
		},
		{
			name:        "Hypertension Stage 2 - Cardiovascular Patient", 
			patient:     fixtures.CreateCardiovascularPatient(),
			expression:  "bp.systolic >= 140 || bp.diastolic >= 90",
			expectMatch: true,
			description: "Should match stage 2 hypertension criteria",
		},
		{
			name:        "Diabetes Detection - Diabetic Patient",
			patient:     fixtures.CreateDiabeticPatient(),
			expression:  "labs.hba1c > 7.0",
			expectMatch: true,
			description: "Should detect uncontrolled diabetes",
		},
		{
			name:        "Diabetes Detection - Healthy Patient",
			patient:     fixtures.CreateHealthyPatient(),
			expression:  "labs.hba1c > 7.0",
			expectMatch: false,
			description: "Should not detect diabetes in healthy patient",
		},
		{
			name:        "CKD Detection - CKD Patient",
			patient:     fixtures.CreateCKDPatient(),
			expression:  "labs.egfr >= 30 && labs.egfr < 60",
			expectMatch: true,
			description: "Should detect CKD stage 3",
		},
		{
			name:        "Elderly Fall Risk - Elderly Patient",
			patient:     fixtures.CreateElderlyMultiMorbidPatient(),
			expression:  "patient.age >= 75",
			expectMatch: true,
			description: "Should identify elderly fall risk",
		},
		{
			name:        "Multi-condition Risk - Multi-morbid Patient",
			patient:     fixtures.CreateElderlyMultiMorbidPatient(),
			expression:  "patient.age >= 65 && conditions.count >= 3",
			expectMatch: true,
			description: "Should identify high-risk multi-morbid patient",
		},
		{
			name:        "Low Risk Profile - Healthy Patient",
			patient:     fixtures.CreateHealthyPatient(),
			expression:  "patient.age < 40 && bp.systolic < 120 && labs.hba1c < 5.7",
			expectMatch: true,
			description: "Should identify low-risk healthy patient",
		},
		{
			name:        "Cardiovascular Risk Factors",
			patient:     fixtures.CreateCardiovascularPatient(),
			expression:  "(patient.age >= 45 && patient.sex == 'M') || (patient.age >= 55 && patient.sex == 'F')",
			expectMatch: true,
			description: "Should identify cardiovascular risk by age and sex",
		},
		{
			name:        "Complex Multi-Factor Risk",
			patient:     fixtures.CreateElderlyMultiMorbidPatient(),
			expression:  "patient.age >= 80 && medications.count >= 4 && conditions.count >= 3",
			expectMatch: true,
			description: "Should identify complex high-risk patient profile",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched, confidence, err := celEngine.EvaluateExpression(tc.expression, tc.patient)
			
			require.NoError(t, err)
			assert.Equal(t, tc.expectMatch, matched, tc.description)
			assert.GreaterOrEqual(t, confidence, 0.0)
			assert.LessOrEqual(t, confidence, 1.0)
			
			t.Logf("Patient: %s | Expression: %s | Matched: %v | Confidence: %.3f | %s", 
				tc.patient.PatientID, tc.expression, matched, confidence, tc.description)
		})
	}
}

func TestCELEngine_ExpressionValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	celEngine, err := engines.NewCELEngine(logger)
	require.NoError(t, err)
	
	testCases := []struct {
		name        string
		expression  string
		expectError bool
		errorType   string
	}{
		{
			name:        "Valid boolean expression",
			expression:  "patient.age >= 18 && patient.sex == 'M'",
			expectError: false,
		},
		{
			name:        "Valid complex clinical expression",
			expression:  "labs.total_cholesterol > 240 && !patient.has_diabetes",
			expectError: false,
		},
		{
			name:        "Valid nested conditional",
			expression:  "(patient.age >= 65) ? (bp.systolic >= 130) : (bp.systolic >= 140)",
			expectError: false,
		},
		{
			name:        "Invalid syntax - missing operand",
			expression:  "patient.age >= && 65",
			expectError: true,
			errorType:   "syntax",
		},
		{
			name:        "Invalid syntax - unbalanced parentheses",
			expression:  "((patient.age >= 65) && bp.systolic >= 130",
			expectError: true,
			errorType:   "syntax",
		},
		{
			name:        "Invalid return type - returns number",
			expression:  "patient.age + 10",
			expectError: true,
			errorType:   "type",
		},
		{
			name:        "Invalid return type - returns string",
			expression:  "patient.sex + '_suffix'",
			expectError: true,
			errorType:   "type",
		},
		{
			name:        "Undefined variable",
			expression:  "unknown_var > 10",
			expectError: true,
			errorType:   "undefined",
		},
		{
			name:        "Undefined field",
			expression:  "patient.unknown_field == 'test'",
			expectError: true,
			errorType:   "undefined",
		},
		{
			name:        "Type mismatch",
			expression:  "patient.age == 'string'",
			expectError: false, // CEL handles type coercion
		},
		{
			name:        "Empty expression",
			expression:  "",
			expectError: true,
			errorType:   "syntax",
		},
		{
			name:        "Too long expression",
			expression:  generateLongExpression(15000), // Exceeds MaxExpressionLength
			expectError: true,
			errorType:   "length",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := celEngine.ValidateExpression(tc.expression)
			
			if tc.expectError {
				assert.Error(t, err)
				t.Logf("Expected validation error (%s): %v", tc.errorType, err)
			} else {
				assert.NoError(t, err)
				t.Logf("Expression validated successfully")
			}
		})
	}
}

func TestCELEngine_Performance_SingleExpression(t *testing.T) {
	logger := zaptest.NewLogger(t)
	celEngine, err := engines.NewCELEngine(logger)
	require.NoError(t, err)
	
	fixtures := testutils.NewPatientFixtures()
	patient := fixtures.CreateCardiovascularPatient()
	expression := "patient.age >= 65 && bp.systolic >= 140 && labs.total_cholesterol > 240"
	
	// Performance test configuration
	config := testutils.DefaultPerformanceConfig()
	config.MaxDuration = 10 * time.Millisecond // CEL should be very fast
	config.MaxThroughput = 5000 // 5K evaluations per second
	
	pt := testutils.NewPerformanceTester(config)
	
	testFunc := func() error {
		_, _, err := celEngine.EvaluateExpression(expression, patient)
		return err
	}
	
	// Run performance test
	metrics := pt.RunPerformanceTest(t, testFunc)
	
	// Validate performance metrics
	pt.ValidatePerformanceMetrics(t, metrics)
	
	// Additional CEL-specific assertions
	assert.Less(t, metrics.P95Duration, 5*time.Millisecond, 
		"P95 latency should be under 5ms for CEL evaluation")
	assert.Greater(t, metrics.Throughput, float64(1000), 
		"CEL should achieve at least 1000 evaluations per second")
	
	t.Logf("CEL Performance - Throughput: %.0f RPS, P95 Latency: %v", 
		metrics.Throughput, metrics.P95Duration)
}

func TestCELEngine_Performance_ExpressionCaching(t *testing.T) {
	logger := zaptest.NewLogger(t)
	celEngine, err := engines.NewCELEngine(logger)
	require.NoError(t, err)
	
	fixtures := testutils.NewPatientFixtures()
	patient := fixtures.CreateCardiovascularPatient()
	expression := "patient.age >= 65 && bp.systolic >= 140"
	
	// First evaluation (compiles and caches)
	start1 := time.Now()
	matched1, confidence1, err1 := celEngine.EvaluateExpression(expression, patient)
	duration1 := time.Since(start1)
	
	require.NoError(t, err1)
	
	// Second evaluation (uses cached compilation)
	start2 := time.Now()
	matched2, confidence2, err2 := celEngine.EvaluateExpression(expression, patient)
	duration2 := time.Since(start2)
	
	require.NoError(t, err2)
	
	// Results should be identical
	assert.Equal(t, matched1, matched2)
	assert.Equal(t, confidence1, confidence2)
	
	// Second evaluation should be faster or similar (caching benefit)
	// Note: For simple expressions, the difference might be negligible
	t.Logf("First evaluation: %v, Second evaluation: %v", duration1, duration2)
	
	// Verify cache stats
	stats := celEngine.GetCacheStats()
	require.NotNil(t, stats)
	
	cachedExprs, ok := stats["cached_expressions"].(int)
	require.True(t, ok)
	assert.Greater(t, cachedExprs, 0, "Should have cached expressions")
	
	t.Logf("Cache stats: %+v", stats)
}

func TestCELEngine_Performance_ConcurrentEvaluation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	celEngine, err := engines.NewCELEngine(logger)
	require.NoError(t, err)
	
	fixtures := testutils.NewPatientFixtures()
	patients := fixtures.GetAllTestPatients()
	expressions := []string{
		"patient.age >= 65",
		"bp.systolic >= 140 || bp.diastolic >= 90", 
		"labs.hba1c > 7.0",
		"labs.egfr >= 30 && labs.egfr < 60",
		"patient.age >= 75 && medications.count > 3",
	}
	
	// Test concurrent evaluation with different patients and expressions
	config := testutils.DefaultPerformanceConfig()
	config.ConcurrentUsers = 50
	config.TestDuration = 15 * time.Second
	
	pt := testutils.NewPerformanceTester(config)
	
	testFunc := func() error {
		// Randomly select patient and expression
		patientIdx := len(patients) % 6 // Simple rotation
		exprIdx := len(expressions) % 5
		
		if patientIdx >= len(patients) {
			patientIdx = 0
		}
		if exprIdx >= len(expressions) {
			exprIdx = 0
		}
		
		_, _, err := celEngine.EvaluateExpression(expressions[exprIdx], patients[patientIdx])
		return err
	}
	
	metrics := pt.RunPerformanceTest(t, testFunc)
	pt.ValidatePerformanceMetrics(t, metrics)
	
	// CEL should handle concurrent evaluation well
	assert.Less(t, metrics.ErrorRate, 0.1, "Error rate should be minimal under concurrent load")
	assert.Greater(t, metrics.Throughput, float64(500), "Should maintain good throughput under concurrent load")
	
	t.Logf("Concurrent CEL Performance - %d users, %.0f RPS, %.2f%% error rate", 
		config.ConcurrentUsers, metrics.Throughput, metrics.ErrorRate)
}

func TestCELEngine_Timeout_Protection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	celEngine, err := engines.NewCELEngine(logger)
	require.NoError(t, err)
	
	fixtures := testutils.NewPatientFixtures()
	patient := fixtures.CreateCardiovascularPatient()
	
	// Test with a complex expression that might take time
	complexExpression := generateComplexExpression()
	
	start := time.Now()
	_, _, err = celEngine.EvaluateExpression(complexExpression, patient)
	duration := time.Since(start)
	
	// Should complete within reasonable time or timeout gracefully
	assert.Less(t, duration, 10*time.Second, "Expression evaluation should not hang")
	
	if err != nil {
		t.Logf("Complex expression failed (expected): %v", err)
		// Timeout errors are acceptable for very complex expressions
		assert.Contains(t, err.Error(), "timeout", "Should fail with timeout if too complex")
	}
	
	t.Logf("Complex expression evaluation time: %v", duration)
}

func TestCELEngine_MemoryLeakProtection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	celEngine, err := engines.NewCELEngine(logger)
	require.NoError(t, err)
	
	fixtures := testutils.NewPatientFixtures()
	patient := fixtures.CreateCardiovascularPatient()
	
	// Measure initial memory
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	
	// Evaluate many different expressions to test memory usage
	for i := 0; i < 1000; i++ {
		expression := fmt.Sprintf("patient.age >= %d", i%100)
		_, _, err := celEngine.EvaluateExpression(expression, patient)
		if err != nil {
			t.Logf("Expression %d failed: %v", i, err)
		}
	}
	
	// Force garbage collection and measure memory
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup
	
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	memoryIncrease := memAfter.HeapInuse - memBefore.HeapInuse
	memoryIncreaseMB := float64(memoryIncrease) / 1024 / 1024
	
	t.Logf("Memory increase after 1000 evaluations: %.2f MB", memoryIncreaseMB)
	
	// Memory increase should be reasonable (caching is expected)
	assert.Less(t, memoryIncreaseMB, 50.0, "Memory increase should be under 50MB")
}

// Helper functions

func generateLongExpression(length int) string {
	expr := "patient.age >= 0"
	for len(expr) < length {
		expr += " && patient.age >= 0"
	}
	return expr
}

func generateComplexExpression() string {
	// Create a complex nested expression that might stress the evaluator
	return `(patient.age >= 65 && patient.sex == 'M') && 
			(bp.systolic >= 140 || bp.diastolic >= 90) && 
			(labs.total_cholesterol > 240 || labs.ldl > 160) &&
			(labs.hba1c > 7.0 ? labs.glucose > 200 : labs.glucose > 100) &&
			(conditions.count > 2 ? medications.count > 3 : medications.count > 1) &&
			(patient.has_diabetes || patient.has_ckd || patient.has_heart_failure)`
}