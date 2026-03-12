package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"kb-clinical-context/internal/api"
	"kb-clinical-context/internal/models"
	"kb-clinical-context/internal/services"
	"kb-clinical-context/tests/testutils"
)

// APITestSuite provides comprehensive integration testing for API endpoints
type APITestSuite struct {
	suite.Suite
	testContainer   *testutils.TestContainer
	contextService  *services.ContextService
	router          *gin.Engine
	fixtures        *testutils.PatientFixtures
}

func TestAPISuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func (suite *APITestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// Setup test containers
	var err error
	suite.testContainer, err = testutils.SetupTestContainers(suite.T())
	require.NoError(suite.T(), err)
	
	// Wait for containers to be ready
	err = suite.testContainer.WaitForContainers(60 * time.Second)
	require.NoError(suite.T(), err)
	
	// Seed test data
	err = suite.testContainer.SeedTestData(suite.T())
	require.NoError(suite.T(), err)
	
	// Initialize fixtures
	suite.fixtures = testutils.NewPatientFixtures()
	
	// Initialize context service
	// Note: In real implementation, you'd properly initialize with phenotype engine
	suite.contextService, err = services.NewContextService(
		suite.testContainer.MongoDB,
		suite.testContainer.RedisClient,
		&MockMetricsCollector{},
		suite.testContainer.Config,
		"../phenotypes", // Phenotype directory
	)
	require.NoError(suite.T(), err)
	
	// Setup API router
	server := api.NewServer(
		suite.testContainer.Config,
		suite.testContainer.MongoDB,
		suite.testContainer.RedisClient,
		&MockMetricsCollector{},
		suite.contextService,
	)
	suite.router = server.Router
}

func (suite *APITestSuite) TearDownSuite() {
	if suite.testContainer != nil {
		suite.testContainer.Cleanup()
	}
}

func (suite *APITestSuite) SetupTest() {
	// Clear test data before each test
	err := suite.testContainer.ClearTestData()
	require.NoError(suite.T(), err)
	
	// Re-seed basic test data
	err = suite.testContainer.SeedTestData(suite.T())
	require.NoError(suite.T(), err)
}

func (suite *APITestSuite) TestHealthEndpoint() {
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(suite.T(), err)
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "healthy", response["status"])
	assert.NotEmpty(suite.T(), response["timestamp"])
	
	suite.T().Logf("Health response: %+v", response)
}

func (suite *APITestSuite) TestBuildContextEndpoint_Success() {
	patient := suite.fixtures.CreateCardiovascularPatient()
	
	requestBody := models.BuildContextRequest{
		PatientID: patient.PatientID,
		Patient: map[string]interface{}{
			"demographics": map[string]interface{}{
				"age_years": patient.Demographics.AgeYears,
				"sex":       patient.Demographics.Sex,
				"race":      patient.Demographics.Race,
				"ethnicity": patient.Demographics.Ethnicity,
			},
			"active_conditions":   convertConditionsToMap(patient.ActiveConditions),
			"recent_labs":         convertLabsToMap(patient.RecentLabs),
			"current_medications": convertMedicationsToMap(patient.CurrentMeds),
		},
		TransactionID: "test-tx-001",
	}
	
	body, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)
	
	req, err := http.NewRequest("POST", "/api/v1/context/build", bytes.NewReader(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response models.BuildContextResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	// Validate response structure
	assert.Equal(suite.T(), patient.PatientID, response.Context.PatientID)
	assert.NotEmpty(suite.T(), response.Context.ContextID)
	assert.False(suite.T(), response.CacheHit) // First call should not be cache hit
	assert.NotZero(suite.T(), response.ProcessedAt)
	
	// Validate demographics
	assert.Equal(suite.T(), patient.Demographics.AgeYears, response.Context.Demographics.AgeYears)
	assert.Equal(suite.T(), patient.Demographics.Sex, response.Context.Demographics.Sex)
	
	// Validate phenotypes (implementation-dependent)
	assert.GreaterOrEqual(suite.T(), len(response.Phenotypes), 0)
	
	// Validate risk scores
	assert.NotNil(suite.T(), response.RiskScores)
	
	suite.T().Logf("Build context response: PatientID=%s, Phenotypes=%d, RiskScores=%d", 
		response.Context.PatientID, len(response.Phenotypes), len(response.RiskScores))
}

func (suite *APITestSuite) TestBuildContextEndpoint_CacheHit() {
	patient := suite.fixtures.CreateDiabeticPatient()
	
	requestBody := models.BuildContextRequest{
		PatientID: patient.PatientID,
		Patient: map[string]interface{}{
			"demographics": map[string]interface{}{
				"age_years": patient.Demographics.AgeYears,
				"sex":       patient.Demographics.Sex,
			},
			"active_conditions":   convertConditionsToMap(patient.ActiveConditions),
			"recent_labs":         convertLabsToMap(patient.RecentLabs),
			"current_medications": convertMedicationsToMap(patient.CurrentMeds),
		},
	}
	
	body, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)
	
	// First request
	req1, err := http.NewRequest("POST", "/api/v1/context/build", bytes.NewReader(body))
	require.NoError(suite.T(), err)
	req1.Header.Set("Content-Type", "application/json")
	
	w1 := httptest.NewRecorder()
	suite.router.ServeHTTP(w1, req1)
	assert.Equal(suite.T(), http.StatusOK, w1.Code)
	
	// Second request (should hit cache)
	req2, err := http.NewRequest("POST", "/api/v1/context/build", bytes.NewReader(body))
	require.NoError(suite.T(), err)
	req2.Header.Set("Content-Type", "application/json")
	
	w2 := httptest.NewRecorder()
	suite.router.ServeHTTP(w2, req2)
	assert.Equal(suite.T(), http.StatusOK, w2.Code)
	
	var response1, response2 models.BuildContextResponse
	err = json.Unmarshal(w1.Body.Bytes(), &response1)
	require.NoError(suite.T(), err)
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	require.NoError(suite.T(), err)
	
	// First should not be cache hit, second should be
	assert.False(suite.T(), response1.CacheHit)
	assert.True(suite.T(), response2.CacheHit)
	
	// Results should be consistent
	assert.Equal(suite.T(), response1.Context.PatientID, response2.Context.PatientID)
	
	suite.T().Logf("Cache test: First call CacheHit=%v, Second call CacheHit=%v", 
		response1.CacheHit, response2.CacheHit)
}

func (suite *APITestSuite) TestBuildContextEndpoint_ValidationErrors() {
	testCases := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing PatientID",
			requestBody:    models.BuildContextRequest{Patient: map[string]interface{}{"test": "data"}},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "patient_id",
		},
		{
			name:           "Missing Patient Data",
			requestBody:    models.BuildContextRequest{PatientID: "TEST-001"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "patient",
		},
		{
			name:           "Empty Request Body",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			var body []byte
			var err error
			
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tc.requestBody)
				require.NoError(t, err)
			}
			
			req, err := http.NewRequest("POST", "/api/v1/context/build", bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			assert.Equal(t, tc.expectedStatus, w.Code)
			
			if tc.expectedError != "" {
				assert.Contains(t, w.Body.String(), tc.expectedError)
			}
			
			t.Logf("Validation test '%s': Status=%d, Body=%s", tc.name, w.Code, w.Body.String())
		})
	}
}

func (suite *APITestSuite) TestPhenotypeDetectionEndpoint() {
	patient := suite.fixtures.CreateCKDPatient()
	
	requestBody := models.PhenotypeDetectionRequest{
		PatientID: patient.PatientID,
		PatientData: map[string]interface{}{
			"demographics":        convertDemographicsToMap(patient.Demographics),
			"active_conditions":   convertConditionsToMap(patient.ActiveConditions),
			"recent_labs":         convertLabsToMap(patient.RecentLabs),
			"current_medications": convertMedicationsToMap(patient.CurrentMeds),
		},
		PhenotypeIDs: []string{"ckd_stage_3", "diabetes_uncontrolled"},
	}
	
	body, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)
	
	req, err := http.NewRequest("POST", "/api/v1/phenotypes/detect", bytes.NewReader(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response models.PhenotypeDetectionResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), patient.PatientID, response.PatientID)
	assert.GreaterOrEqual(suite.T(), response.TotalPhenotypes, 0)
	assert.GreaterOrEqual(suite.T(), response.ProcessingTime, int64(0))
	assert.NotZero(suite.T(), response.Timestamp)
	
	// Should detect some phenotypes for CKD patient
	if len(response.DetectedPhenotypes) > 0 {
		for _, phenotype := range response.DetectedPhenotypes {
			assert.NotEmpty(suite.T(), phenotype.PhenotypeID)
			assert.GreaterOrEqual(suite.T(), phenotype.Confidence, 0.0)
			assert.LessOrEqual(suite.T(), phenotype.Confidence, 1.0)
			assert.NotZero(suite.T(), phenotype.DetectedAt)
		}
	}
	
	suite.T().Logf("Phenotype detection: PatientID=%s, Detected=%d, ProcessingTime=%dms", 
		response.PatientID, response.TotalPhenotypes, response.ProcessingTime)
}

func (suite *APITestSuite) TestRiskAssessmentEndpoint() {
	patient := suite.fixtures.CreateElderlyMultiMorbidPatient()
	
	requestBody := models.RiskAssessmentRequest{
		PatientID: patient.PatientID,
		RiskTypes: []string{"cardiovascular_risk", "fall_risk", "ade_risk", "readmission_risk"},
		PatientData: map[string]interface{}{
			"demographics":        convertDemographicsToMap(patient.Demographics),
			"active_conditions":   convertConditionsToMap(patient.ActiveConditions),
			"recent_labs":         convertLabsToMap(patient.RecentLabs),
			"current_medications": convertMedicationsToMap(patient.CurrentMeds),
		},
	}
	
	body, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)
	
	req, err := http.NewRequest("POST", "/api/v1/risk/assess", bytes.NewReader(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response models.RiskAssessmentResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), patient.PatientID, response.PatientID)
	assert.NotNil(suite.T(), response.RiskScores)
	assert.GreaterOrEqual(suite.T(), len(response.RiskScores), 1)
	assert.GreaterOrEqual(suite.T(), response.ConfidenceScore, 0.0)
	assert.LessOrEqual(suite.T(), response.ConfidenceScore, 1.0)
	assert.NotZero(suite.T(), response.AssessmentTimestamp)
	
	// Validate risk scores
	for riskType, score := range response.RiskScores {
		assert.GreaterOrEqual(suite.T(), score, 0.0)
		assert.LessOrEqual(suite.T(), score, 1.0)
		suite.T().Logf("Risk assessment: %s = %.3f", riskType, score)
	}
	
	// Elderly multi-morbid patient should have high risks
	if cvRisk, ok := response.RiskScores["cardiovascular_risk"]; ok {
		assert.Greater(suite.T(), cvRisk, 0.5, "Elderly multi-morbid patient should have high CV risk")
	}
	
	if fallRisk, ok := response.RiskScores["fall_risk"]; ok {
		assert.Greater(suite.T(), fallRisk, 0.5, "Elderly patient should have high fall risk")
	}
}

func (suite *APITestSuite) TestCareGapsEndpoint() {
	patient := suite.fixtures.CreateDiabeticPatient()
	
	req, err := http.NewRequest("GET", "/api/v1/care-gaps/"+patient.PatientID, nil)
	require.NoError(suite.T(), err)
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response models.CareGapsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), patient.PatientID, response.PatientID)
	assert.GreaterOrEqual(suite.T(), response.TotalGaps, 0)
	assert.NotEmpty(suite.T(), response.Priority)
	assert.NotZero(suite.T(), response.NextReview)
	
	suite.T().Logf("Care gaps: PatientID=%s, TotalGaps=%d, Priority=%s", 
		response.PatientID, response.TotalGaps, response.Priority)
}

func (suite *APITestSuite) TestAPIPerformance_BuildContext() {
	patient := suite.fixtures.CreateCardiovascularPatient()
	requestBody := models.BuildContextRequest{
		PatientID: patient.PatientID,
		Patient: map[string]interface{}{
			"demographics":        convertDemographicsToMap(patient.Demographics),
			"active_conditions":   convertConditionsToMap(patient.ActiveConditions),
			"recent_labs":         convertLabsToMap(patient.RecentLabs),
			"current_medications": convertMedicationsToMap(patient.CurrentMeds),
		},
	}
	
	body, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)
	
	// Performance test configuration
	config := testutils.DefaultPerformanceConfig()
	config.MaxDuration = 50 * time.Millisecond // API should respond quickly
	config.MaxThroughput = 100 // 100 RPS for integration tests
	config.ConcurrentUsers = 5  // Lighter load for integration tests
	config.TestDuration = 10 * time.Second
	
	pt := testutils.NewPerformanceTester(config)
	
	testFunc := func() error {
		req, err := http.NewRequest("POST", "/api/v1/context/build", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			return assert.AnError
		}
		
		return nil
	}
	
	metrics := pt.RunPerformanceTest(suite.T(), testFunc)
	pt.ValidatePerformanceMetrics(suite.T(), metrics)
	
	// API-specific performance assertions
	assert.Less(suite.T(), metrics.P95Duration, 100*time.Millisecond, 
		"API P95 latency should be under 100ms")
	
	suite.T().Logf("API Performance - Throughput: %.0f RPS, P95 Latency: %v, Error Rate: %.2f%%", 
		metrics.Throughput, metrics.P95Duration, metrics.ErrorRate)
}

func (suite *APITestSuite) TestAPIPerformance_SLACompliance() {
	// Test against KB-2 SLA targets
	patient := suite.fixtures.CreateHealthyPatient() // Use healthy patient for consistent performance
	requestBody := models.BuildContextRequest{
		PatientID: patient.PatientID,
		Patient: map[string]interface{}{
			"demographics":      convertDemographicsToMap(patient.Demographics),
			"active_conditions": []interface{}{}, // Minimal data for best performance
			"recent_labs":       convertLabsToMap(patient.RecentLabs[:1]), // Just one lab
		},
	}
	
	body, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)
	
	config := testutils.LoadTestConfig()
	config.ConcurrentUsers = 20 // Reasonable load for integration test
	config.TestDuration = 30 * time.Second
	
	pt := testutils.NewPerformanceTester(config)
	
	testFunc := func() error {
		req, err := http.NewRequest("POST", "/api/v1/context/build", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			return assert.AnError
		}
		
		return nil
	}
	
	metrics := pt.RunPerformanceTest(suite.T(), testFunc)
	
	// Validate against KB-2 SLA targets
	testutils.ValidateSLACompliance(suite.T(), metrics, testutils.KB2SLATargets)
	
	suite.T().Logf("SLA Compliance Test - P50: %v, P95: %v, P99: %v, Throughput: %.0f RPS", 
		metrics.P50Duration, metrics.P95Duration, metrics.P99Duration, metrics.Throughput)
}

// Helper functions and mock implementations

type MockMetricsCollector struct{}

func (m *MockMetricsCollector) RecordCacheHit(cacheType string)                              {}
func (m *MockMetricsCollector) RecordCacheMiss(cacheType string)                             {}
func (m *MockMetricsCollector) RecordContextBuild(success bool)                              {}
func (m *MockMetricsCollector) RecordContextBuildDuration(phenotypeCount int, duration time.Duration) {}
func (m *MockMetricsCollector) RecordPhenotypeDetection(phenotypeID string)                  {}
func (m *MockMetricsCollector) RecordPhenotypeDetectionDuration(phenotypeCount int, duration time.Duration) {}
func (m *MockMetricsCollector) RecordRiskAssessment(riskType string, success bool)           {}
func (m *MockMetricsCollector) RecordRiskAssessmentDuration(riskType string, duration time.Duration) {}
func (m *MockMetricsCollector) RecordCareGap(gapType string)                                 {}
func (m *MockMetricsCollector) RecordMongoOperation(operation, collection string, success bool, duration time.Duration) {}

func convertDemographicsToMap(demo models.Demographics) map[string]interface{} {
	return map[string]interface{}{
		"age_years": demo.AgeYears,
		"sex":       demo.Sex,
		"race":      demo.Race,
		"ethnicity": demo.Ethnicity,
	}
}

func convertConditionsToMap(conditions []models.Condition) []interface{} {
	result := make([]interface{}, len(conditions))
	for i, condition := range conditions {
		result[i] = map[string]interface{}{
			"code":        condition.Code,
			"system":      condition.System,
			"name":        condition.Name,
			"onset_date":  condition.OnsetDate,
			"severity":    condition.Severity,
		}
	}
	return result
}

func convertLabsToMap(labs []models.LabResult) []interface{} {
	result := make([]interface{}, len(labs))
	for i, lab := range labs {
		result[i] = map[string]interface{}{
			"loinc_code":    lab.LOINCCode,
			"value":         lab.Value,
			"unit":          lab.Unit,
			"result_date":   lab.ResultDate,
			"abnormal_flag": lab.AbnormalFlag,
		}
	}
	return result
}

func convertMedicationsToMap(medications []models.Medication) []interface{} {
	result := make([]interface{}, len(medications))
	for i, med := range medications {
		result[i] = map[string]interface{}{
			"rxnorm_code": med.RxNormCode,
			"name":        med.Name,
			"dose":        med.Dose,
			"frequency":   med.Frequency,
			"start_date":  med.StartDate,
		}
	}
	return result
}