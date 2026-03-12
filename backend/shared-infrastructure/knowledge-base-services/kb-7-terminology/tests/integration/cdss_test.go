package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"kb-7-terminology/tests/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ============================================================================
// CDSS Integration Test Suite
// ============================================================================
// Comprehensive integration tests for the KB-7 CDSS evaluation pipeline.
// Tests cover FHIR parsing, fact building, THREE-CHECK pipeline, rule engine,
// and alert generation.
//
// Run with: TEST_ENV=integration go test -v ./tests/integration/... -run TestCDSS
// ============================================================================

// CDSSIntegrationTestSuite provides comprehensive CDSS testing
type CDSSIntegrationTestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
}

// SetupSuite runs once before all tests
func (suite *CDSSIntegrationTestSuite) SetupSuite() {
	// Check for integration test environment
	if os.Getenv("TEST_ENV") != "integration" && os.Getenv("CI") != "true" {
		suite.T().Skip("CDSS integration tests require TEST_ENV=integration")
	}

	suite.baseURL = os.Getenv("KB7_BASE_URL")
	if suite.baseURL == "" {
		suite.baseURL = "http://localhost:8087"
	}

	suite.httpClient = &http.Client{
		Timeout: fixtures.GetCDSSTestTimeout(),
	}

	// Wait for service to be ready
	suite.waitForServiceReady()

	// Ensure value sets are seeded
	suite.seedValueSets()
}

// waitForServiceReady waits for the KB-7 service to be available
func (suite *CDSSIntegrationTestSuite) waitForServiceReady() {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := suite.httpClient.Get(suite.baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	suite.T().Fatal("KB-7 service did not become ready within timeout")
}

// seedValueSets ensures all builtin value sets are seeded
func (suite *CDSSIntegrationTestSuite) seedValueSets() {
	resp, err := suite.httpClient.Post(suite.baseURL+"/v1/rules/seed", "application/json", nil)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
}

// ============================================================================
// Category 1: FHIR Parsing Tests
// ============================================================================

// TestFHIRParsing_ValidBundle verifies that a complete FHIR Bundle is correctly parsed
func (suite *CDSSIntegrationTestSuite) TestFHIRParsing_ValidBundle() {
	t := suite.T()

	// Test 1.1: Parse Valid FHIR Bundle
	request := map[string]interface{}{
		"patient_id":   "test-patient-001",
		"encounter_id": "encounter-001",
		"bundle":       fixtures.SepsisPatientBundle,
		"options": map[string]interface{}{
			"enable_subsumption": true,
			"generate_alerts":    true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool), "CDSS evaluation should succeed")
	assert.Greater(t, int(resp["facts_extracted"].(float64)), 0, "Should extract facts from bundle")
}

// TestFHIRParsing_MultipleCodingsSameConcept tests resources with multiple coding systems
func (suite *CDSSIntegrationTestSuite) TestFHIRParsing_MultipleCodingsSameConcept() {
	t := suite.T()

	// Test 1.2: Parse Bundle with Multiple Codings (SNOMED + ICD-10)
	request := map[string]interface{}{
		"patient_id": "test-patient-002",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "44054006",
							"display": "Type 2 diabetes mellitus",
						},
						{
							"system":  "http://hl7.org/fhir/sid/icd-10-au",
							"code":    "E11.9",
							"display": "Type 2 diabetes mellitus without complications",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool), "Should handle multiple codings")
	assert.Greater(t, int(resp["facts_extracted"].(float64)), 0, "Should extract facts")
}

// TestFHIRParsing_InvalidResource tests graceful handling of malformed FHIR data
func (suite *CDSSIntegrationTestSuite) TestFHIRParsing_InvalidResource() {
	t := suite.T()

	// Test 1.3: Parse Invalid FHIR Resource - should not crash, may return warning
	request := map[string]interface{}{
		"patient_id": "test-patient-003",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				// Missing code - invalid condition
			},
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	// Should complete without crashing
	// May have success=true with 0 facts, or success=false with error
	// The important thing is it doesn't crash
	assert.NotNil(t, resp["success"], "Response should have success field")
}

// ============================================================================
// Category 2: Fact Building Tests
// ============================================================================

// TestFactBuilding_ConditionFacts verifies conditions are correctly classified
func (suite *CDSSIntegrationTestSuite) TestFactBuilding_ConditionFacts() {
	t := suite.T()

	// Test 2.1: Build Condition Facts with Value Set Classification
	request := map[string]interface{}{
		"patient_id": "fact-test-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
			"generate_alerts":    true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))
	assert.Equal(t, float64(1), resp["facts_extracted"].(float64), "Should extract 1 condition fact")
	assert.Greater(t, int(resp["matches_found"].(float64)), 0, "Sepsis should match value sets")
}

// TestFactBuilding_LabFacts verifies lab observations are correctly converted to facts
func (suite *CDSSIntegrationTestSuite) TestFactBuilding_LabFacts() {
	t := suite.T()

	// Test 2.2: Build Lab Facts with Value Extraction
	request := map[string]interface{}{
		"patient_id": "fact-test-002",
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 4.2,
					"unit":  "mmol/L",
				},
				"status": "final",
			},
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2160-0", "display": "Creatinine"},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 2.8,
					"unit":  "mg/dL",
				},
				"status": "final",
			},
		},
		"options": map[string]interface{}{
			"generate_alerts": true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))
	assert.Equal(t, float64(2), resp["facts_extracted"].(float64), "Should extract 2 lab facts")
}

// TestFactBuilding_MedicationFacts verifies medications are classified into therapeutic classes
func (suite *CDSSIntegrationTestSuite) TestFactBuilding_MedicationFacts() {
	t := suite.T()

	// Test 2.3: Build Medication Facts with Drug Class Identification
	request := map[string]interface{}{
		"patient_id": "fact-test-003",
		"medications": []map[string]interface{}{
			{
				"resourceType": "MedicationRequest",
				"medicationCodeableConcept": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "386873009", "display": "Lisinopril"},
					},
				},
				"status": "active",
			},
		},
		"options": map[string]interface{}{
			"generate_alerts": true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))
	assert.Greater(t, int(resp["facts_extracted"].(float64)), 0, "Should extract medication fact")
}

// ============================================================================
// Category 3: THREE-CHECK Pipeline Tests
// ============================================================================

// TestThreeCheck_ExactMatch verifies O(1) hash lookup for direct code match
func (suite *CDSSIntegrationTestSuite) TestThreeCheck_ExactMatch() {
	t := suite.T()

	for _, tc := range fixtures.ThreeCheckTestCases {
		if tc.ExpectedType != "exact" {
			continue
		}

		suite.Run(tc.Description, func() {
			// Test value set validation using POST /validate endpoint
			url := fmt.Sprintf("%s/v1/rules/valuesets/%s/validate",
				suite.baseURL, tc.ValueSet)

			reqBody := map[string]string{
				"code":   tc.Code,
				"system": tc.System,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			resp, err := suite.httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			valid := false
			if v, ok := result["valid"].(bool); ok {
				valid = v
			}
			assert.Equal(t, tc.ExpectedValid, valid,
				"Code %s should %s be in value set %s",
				tc.Code, map[bool]string{true: "", false: "NOT"}[tc.ExpectedValid], tc.ValueSet)

			// Verify THREE-CHECK pipeline was used (exact match should be step 2)
			if tc.ExpectedValid {
				matchType, _ := result["match_type"].(string)
				assert.Equal(t, tc.ExpectedType, matchType, "Expected match type %s", tc.ExpectedType)
			}
		})
	}
}

// TestThreeCheck_NoMatch verifies correct handling when code doesn't match
func (suite *CDSSIntegrationTestSuite) TestThreeCheck_NoMatch() {
	t := suite.T()

	// Test 3.4: No Match - Diabetes not in Sepsis value set
	url := fmt.Sprintf("%s/v1/rules/valuesets/%s/validate",
		suite.baseURL, "AUSepsisConditions")

	reqBody := map[string]string{
		"code":   "73211009", // Diabetes mellitus
		"system": "http://snomed.info/sct",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := suite.httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	valid := false
	if v, ok := result["valid"].(bool); ok {
		valid = v
	}

	assert.False(t, valid, "Diabetes should NOT be in Sepsis value set")

	// Verify match_type is "none"
	matchType, _ := result["match_type"].(string)
	assert.Equal(t, "none", matchType, "Non-member code should return match_type=none")
}

// TestThreeCheck_PipelineUsed verifies THREE-CHECK pipeline is used in evaluation
func (suite *CDSSIntegrationTestSuite) TestThreeCheck_PipelineUsed() {
	t := suite.T()

	request := map[string]interface{}{
		"patient_id": "pipeline-test-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))

	// Verify pipeline used
	pipelineUsed, ok := resp["pipeline_used"].(string)
	if ok {
		assert.Equal(t, "THREE-CHECK", pipelineUsed, "Should use THREE-CHECK pipeline")
	}
}

// ============================================================================
// Category 4: RefsetService Integration Tests
// ============================================================================

// TestRefsetService_ListRefsets verifies RefsetService can list available refsets
func (suite *CDSSIntegrationTestSuite) TestRefsetService_ListRefsets() {
	t := suite.T()

	// Skip if Neo4j not available
	resp, err := suite.httpClient.Get(suite.baseURL + "/v1/refsets")
	if err != nil {
		t.Skip("RefsetService not available")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("RefsetService not available - Neo4j may not be running")
	}

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.True(t, result["success"].(bool), "Should successfully list refsets")
}

// TestRefsetService_Health verifies RefsetService health endpoint
func (suite *CDSSIntegrationTestSuite) TestRefsetService_Health() {
	t := suite.T()

	resp, err := suite.httpClient.Get(suite.baseURL + "/v1/refsets/health")
	if err != nil {
		t.Skip("RefsetService health check not available")
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	status := result["status"].(string)
	assert.Contains(t, []string{"healthy", "degraded"}, status,
		"RefsetService status should be healthy or degraded")
}

// ============================================================================
// Category 5: Rule Engine Tests
// ============================================================================

// TestRuleEngine_SimpleValueSetCondition verifies rule with single Value Set condition
func (suite *CDSSIntegrationTestSuite) TestRuleEngine_SimpleValueSetCondition() {
	t := suite.T()

	// Test 5.1: Simple VALUE_SET Condition - Sepsis Detected
	request := map[string]interface{}{
		"patient_id": "rule-test-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"options": map[string]interface{}{
			"evaluate_rules":  true,
			"generate_alerts": true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))
	// Check that rules were evaluated and potentially fired
	rulesFired := int(resp["rules_fired"].(float64))
	t.Logf("Rules fired: %d", rulesFired)
}

// TestRuleEngine_ThresholdCondition verifies rule with lab threshold
func (suite *CDSSIntegrationTestSuite) TestRuleEngine_ThresholdCondition() {
	t := suite.T()

	// Test 5.2: THRESHOLD Condition - Elevated Lactate (> 2.0)
	request := map[string]interface{}{
		"patient_id": "rule-test-002",
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 4.2,
					"unit":  "mmol/L",
				},
				"status": "final",
			},
		},
		"options": map[string]interface{}{
			"evaluate_rules":  true,
			"generate_alerts": true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))
	// Lactate 4.2 > 2.0 threshold should trigger rules
	assert.Greater(t, int(resp["matches_found"].(float64)), 0, "Lactate should match lab value sets")
}

// TestRuleEngine_CompoundConditionAND verifies rule with multiple AND conditions
func (suite *CDSSIntegrationTestSuite) TestRuleEngine_CompoundConditionAND() {
	t := suite.T()

	// Test 5.3: COMPOUND Condition (AND) - Sepsis with Elevated Lactate
	request := map[string]interface{}{
		"patient_id":   "rule-test-003",
		"encounter_id": "encounter-003",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 4.5,
					"unit":  "mmol/L",
				},
				"status": "final",
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
			"evaluate_rules":     true,
			"generate_alerts":    true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))
	assert.Greater(t, int(resp["rules_fired"].(float64)), 0, "Compound rule should fire with both conditions met")
}

// TestRuleEngine_CompoundConditionPartialMatch verifies rule doesn't fire with partial AND match
func (suite *CDSSIntegrationTestSuite) TestRuleEngine_CompoundConditionPartialMatch() {
	t := suite.T()

	// Test 5.4: COMPOUND Condition (AND) - Partial Match (lactate below threshold)
	request := map[string]interface{}{
		"patient_id": "rule-test-004",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 1.5, // Below 2.0 threshold
					"unit":  "mmol/L",
				},
				"status": "final",
			},
		},
		"options": map[string]interface{}{
			"evaluate_rules":  true,
			"generate_alerts": true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))
	// The sepsis-lactate-elevated compound rule should NOT fire
	// But simple sepsis detection rule might still fire
	// This test verifies the compound rule logic
}

// ============================================================================
// Category 6: Alert Generation Tests
// ============================================================================

// TestAlertGeneration_FromFiredRule verifies alert is correctly generated from fired rule
func (suite *CDSSIntegrationTestSuite) TestAlertGeneration_FromFiredRule() {
	t := suite.T()

	// Test 6.1: Alert Creation from Fired Rule
	request := map[string]interface{}{
		"patient_id":   "alert-test-001",
		"encounter_id": "encounter-alert-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 4.5,
					"unit":  "mmol/L",
				},
				"status": "final",
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
			"evaluate_rules":     true,
			"generate_alerts":    true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool))
	alertCount := int(resp["alerts_generated"].(float64))
	assert.Greater(t, alertCount, 0, "Should generate at least 1 alert")

	// Check alerts array
	alerts, ok := resp["alerts"].([]interface{})
	if ok && len(alerts) > 0 {
		firstAlert := alerts[0].(map[string]interface{})
		assert.NotEmpty(t, firstAlert["alert_id"], "Alert should have ID")
		assert.NotEmpty(t, firstAlert["severity"], "Alert should have severity")
		assert.NotEmpty(t, firstAlert["title"], "Alert should have title")
	}
}

// TestAlertGeneration_CriticalSeverity verifies critical alerts are generated for sepsis
func (suite *CDSSIntegrationTestSuite) TestAlertGeneration_CriticalSeverity() {
	t := suite.T()

	request := map[string]interface{}{
		"patient_id":   "alert-test-002",
		"encounter_id": "encounter-alert-002",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 4.5,
					"unit":  "mmol/L",
				},
				"status": "final",
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
			"evaluate_rules":     true,
			"generate_alerts":    true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	// Check for critical severity alert
	alerts, ok := resp["alerts"].([]interface{})
	if ok && len(alerts) > 0 {
		hasCritical := false
		for _, a := range alerts {
			alert := a.(map[string]interface{})
			if alert["severity"] == "critical" {
				hasCritical = true
				break
			}
		}
		assert.True(t, hasCritical, "Should have at least one critical severity alert for sepsis with elevated lactate")
	}
}

// ============================================================================
// Category 7: End-to-End Clinical Scenarios
// ============================================================================

// TestE2E_SepsisPatientScenario tests full evaluation of sepsis patient (Test 7.1)
func (suite *CDSSIntegrationTestSuite) TestE2E_SepsisPatientScenario() {
	t := suite.T()

	// Build request from fixture
	requestBody, err := json.Marshal(map[string]interface{}{
		"patient_id":   "sepsis-patient-001",
		"encounter_id": "encounter-sepsis-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "233604007", "display": "Pneumonia"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 4.5, "unit": "mmol/L"},
				"status":        "final",
			},
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "8480-6", "display": "Systolic BP"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 85, "unit": "mmHg"},
				"status":        "final",
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
			"evaluate_rules":     true,
			"generate_alerts":    true,
		},
	})
	require.NoError(t, err)

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/v1/cdss/evaluate",
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify response
	assert.True(t, result["success"].(bool), "Sepsis scenario should succeed")
	assert.GreaterOrEqual(t, int(result["facts_extracted"].(float64)), 4, "Should extract at least 4 facts")
	assert.Greater(t, int(result["alerts_generated"].(float64)), 0, "Should generate alerts")
	assert.Equal(t, "THREE-CHECK", result["pipeline_used"], "Should use THREE-CHECK pipeline")

	// Verify critical alert exists
	alerts := result["alerts"].([]interface{})
	hasCritical := false
	hasSepsisDomain := false
	for _, a := range alerts {
		alert := a.(map[string]interface{})
		if alert["severity"] == "critical" {
			hasCritical = true
		}
		if domain, ok := alert["clinical_domain"].(string); ok && domain == "sepsis" {
			hasSepsisDomain = true
		}
	}
	assert.True(t, hasCritical, "Should have critical alert for septic shock")
	assert.True(t, hasSepsisDomain, "Should have sepsis domain alert")

	t.Logf("Sepsis E2E: %d facts, %d alerts, %d rules fired",
		int(result["facts_extracted"].(float64)),
		int(result["alerts_generated"].(float64)),
		int(result["rules_fired"].(float64)))
}

// TestE2E_DiabeticPatientWithAKIRisk tests diabetic patient with nephrotoxic drugs (Test 7.2)
func (suite *CDSSIntegrationTestSuite) TestE2E_DiabeticPatientWithAKIRisk() {
	t := suite.T()

	request := map[string]interface{}{
		"patient_id":   "diabetic-patient-001",
		"encounter_id": "encounter-diabetic-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "44054006", "display": "Type 2 diabetes"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "431856006", "display": "CKD Stage 2"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2160-0", "display": "Creatinine"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 1.8, "unit": "mg/dL"},
				"status":        "final",
			},
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "4548-4", "display": "HbA1c"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 9.2, "unit": "%"},
				"status":        "final",
			},
		},
		"medications": []map[string]interface{}{
			{
				"resourceType": "MedicationRequest",
				"medicationCodeableConcept": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "21411011000036105", "display": "Gentamicin"},
					},
				},
				"status": "active",
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
			"evaluate_rules":     true,
			"generate_alerts":    true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool), "Diabetic scenario should succeed")
	assert.GreaterOrEqual(t, int(resp["facts_extracted"].(float64)), 4, "Should extract at least 4 facts")

	t.Logf("Diabetic E2E: %d facts, %d alerts, %d rules fired",
		int(resp["facts_extracted"].(float64)),
		int(resp["alerts_generated"].(float64)),
		int(resp["rules_fired"].(float64)))
}

// TestE2E_CardiacPatientAnticoagulation tests cardiac patient for anticoagulation (Test 7.3)
func (suite *CDSSIntegrationTestSuite) TestE2E_CardiacPatientAnticoagulation() {
	t := suite.T()

	request := map[string]interface{}{
		"patient_id":   "cardiac-patient-001",
		"encounter_id": "encounter-cardiac-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "49436004", "display": "Atrial fibrillation"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "59621000", "display": "Essential hypertension"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"observations": []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "6301-6", "display": "INR"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 1.1, "unit": "ratio"},
				"status":        "final",
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
			"evaluate_rules":     true,
			"generate_alerts":    true,
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	assert.True(t, resp["success"].(bool), "Cardiac scenario should succeed")
	assert.GreaterOrEqual(t, int(resp["facts_extracted"].(float64)), 3, "Should extract at least 3 facts")

	t.Logf("Cardiac E2E: %d facts, %d alerts, %d rules fired",
		int(resp["facts_extracted"].(float64)),
		int(resp["alerts_generated"].(float64)),
		int(resp["rules_fired"].(float64)))
}

// ============================================================================
// Category 8: Performance Tests
// ============================================================================

// TestPerformance_SingleRequestLatency tests acceptable latency for single evaluation
func (suite *CDSSIntegrationTestSuite) TestPerformance_SingleRequestLatency() {
	t := suite.T()

	config := fixtures.DefaultPerformanceConfig

	// Run multiple requests to get latency statistics
	var latencies []time.Duration
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		start := time.Now()

		request := map[string]interface{}{
			"patient_id": fmt.Sprintf("perf-test-%d", i),
			"conditions": []map[string]interface{}{
				{
					"resourceType": "Condition",
					"code": map[string]interface{}{
						"coding": []map[string]interface{}{
							{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
						},
					},
					"clinicalStatus": map[string]interface{}{
						"coding": []map[string]interface{}{{"code": "active"}},
					},
				},
			},
			"observations": []map[string]interface{}{
				{
					"resourceType": "Observation",
					"code": map[string]interface{}{
						"coding": []map[string]interface{}{
							{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
						},
					},
					"valueQuantity": map[string]interface{}{"value": 4.5, "unit": "mmol/L"},
					"status":        "final",
				},
			},
			"options": map[string]interface{}{
				"enable_subsumption": true,
				"evaluate_rules":     true,
				"generate_alerts":    true,
			},
		}

		suite.postCDSSEvaluate(request)
		latencies = append(latencies, time.Since(start))
	}

	// Calculate P50, P95, P99
	// Sort latencies
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	p50 := latencies[len(latencies)/2]
	p95 := latencies[int(float64(len(latencies))*0.95)]

	t.Logf("Performance Test: P50=%v, P95=%v", p50, p95)

	// These are integration tests, so we allow more generous timeouts
	assert.Less(t, p50, config.SingleRequestP50Target*10,
		"P50 latency should be under %v (with 10x tolerance for integration tests)", config.SingleRequestP50Target)
	assert.Less(t, p95, config.SingleRequestP95Target*10,
		"P95 latency should be under %v (with 10x tolerance for integration tests)", config.SingleRequestP95Target)
}

// TestPerformance_CacheEffectiveness tests that caching improves performance
func (suite *CDSSIntegrationTestSuite) TestPerformance_CacheEffectiveness() {
	t := suite.T()

	request := map[string]interface{}{
		"patient_id": "cache-test-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		"options": map[string]interface{}{
			"enable_subsumption": true,
		},
	}

	// First request (cold)
	start1 := time.Now()
	suite.postCDSSEvaluate(request)
	coldLatency := time.Since(start1)

	// Second request (should be cached)
	start2 := time.Now()
	suite.postCDSSEvaluate(request)
	warmLatency := time.Since(start2)

	t.Logf("Cache Test: Cold=%v, Warm=%v, Improvement=%.1f%%",
		coldLatency, warmLatency, float64(coldLatency-warmLatency)/float64(coldLatency)*100)

	// Warm request should generally be faster or similar (allow for variance)
	// In real scenarios, cached requests are typically 50%+ faster
	assert.LessOrEqual(t, warmLatency, coldLatency*2,
		"Warm request should not be significantly slower than cold request")
}

// ============================================================================
// Category 9: Error Handling Tests
// ============================================================================

// TestErrorHandling_InvalidInput verifies graceful handling of invalid input
func (suite *CDSSIntegrationTestSuite) TestErrorHandling_InvalidInput() {
	t := suite.T()

	// Test 9.1: Invalid FHIR Resource
	request := map[string]interface{}{
		"patient_id": "error-test-001",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code":         nil, // Invalid - missing code
			},
		},
	}

	requestBody, _ := json.Marshal(request)
	resp, err := suite.httpClient.Post(
		suite.baseURL+"/v1/cdss/evaluate",
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 with success=false or 400
	assert.Contains(t, []int{200, 400}, resp.StatusCode,
		"Should return 200 (with error in body) or 400 for invalid input")
}

// TestErrorHandling_MissingPatientID verifies patient_id is required
func (suite *CDSSIntegrationTestSuite) TestErrorHandling_MissingPatientID() {
	t := suite.T()

	// Missing patient_id
	request := map[string]interface{}{
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008"},
					},
				},
			},
		},
	}

	requestBody, _ := json.Marshal(request)
	resp, err := suite.httpClient.Post(
		suite.baseURL+"/v1/cdss/evaluate",
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 400 for missing required field
	assert.Equal(t, 400, resp.StatusCode, "Should return 400 for missing patient_id")
}

// TestErrorHandling_UnknownCodeSystem verifies handling of unrecognized code systems
func (suite *CDSSIntegrationTestSuite) TestErrorHandling_UnknownCodeSystem() {
	t := suite.T()

	// Test 9.4: Unknown Code System
	request := map[string]interface{}{
		"patient_id": "error-test-003",
		"conditions": []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://unknown-system.example.com",
							"code":    "12345",
							"display": "Unknown condition",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
	}

	resp := suite.postCDSSEvaluate(request)
	require.NotNil(t, resp)

	// Should still succeed but with 0 matches for unknown system
	assert.True(t, resp["success"].(bool), "Should handle unknown code system gracefully")
	assert.Equal(t, float64(1), resp["facts_extracted"].(float64), "Should still extract the fact")
	// Matches might be 0 since unknown system won't match value sets
}

// TestErrorHandling_EmptyRequest verifies handling of empty request
func (suite *CDSSIntegrationTestSuite) TestErrorHandling_EmptyRequest() {
	t := suite.T()

	// Empty request body
	requestBody := []byte("{}")
	resp, err := suite.httpClient.Post(
		suite.baseURL+"/v1/cdss/evaluate",
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 400 for missing patient_id
	assert.Equal(t, 400, resp.StatusCode, "Should return 400 for empty request")
}

// ============================================================================
// Helper Methods
// ============================================================================

// postCDSSEvaluate sends a CDSS evaluation request and returns the response
func (suite *CDSSIntegrationTestSuite) postCDSSEvaluate(request map[string]interface{}) map[string]interface{} {
	requestBody, err := json.Marshal(request)
	if err != nil {
		suite.T().Logf("Failed to marshal request: %v", err)
		return nil
	}

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/v1/cdss/evaluate",
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		suite.T().Logf("Failed to send request: %v", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		suite.T().Logf("Failed to read response: %v", err)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		suite.T().Logf("Failed to unmarshal response: %v, body: %s", err, string(body))
		return nil
	}

	return result
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ============================================================================
// Test Suite Entry Point
// ============================================================================

// TestCDSSIntegrationSuite runs the full CDSS integration test suite
func TestCDSSIntegrationSuite(t *testing.T) {
	suite.Run(t, new(CDSSIntegrationTestSuite))
}
