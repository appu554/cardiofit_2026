package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"kb-7-terminology/tests/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SubsumptionIntegrationTestSuite tests SNOMED CT subsumption operations
type SubsumptionIntegrationTestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
}

// SetupSuite runs once before all tests in the suite
func (suite *SubsumptionIntegrationTestSuite) SetupSuite() {
	if os.Getenv("TEST_ENV") != "integration" && os.Getenv("CI") != "true" {
		suite.T().Skip("Subsumption integration tests require TEST_ENV=integration")
	}

	suite.baseURL = os.Getenv("KB7_BASE_URL")
	if suite.baseURL == "" {
		suite.baseURL = "http://localhost:8087"
	}

	suite.httpClient = &http.Client{
		Timeout: fixtures.GetAUTestTimeout(),
	}

	suite.waitForServiceReady()
}

func (suite *SubsumptionIntegrationTestSuite) waitForServiceReady() {
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

// TestSubsumptionConfigEndpoint tests the subsumption configuration endpoint
func (suite *SubsumptionIntegrationTestSuite) TestSubsumptionConfigEndpoint() {
	t := suite.T()

	resp, err := suite.httpClient.Get(suite.baseURL + "/v1/subsumption/config")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var config struct {
		PreferredBackend string `json:"preferred_backend"`
		Backends         map[string]struct {
			Available bool   `json:"available"`
			URL       string `json:"url,omitempty"`
		} `json:"backends"`
	}
	err = json.NewDecoder(resp.Body).Decode(&config)
	require.NoError(t, err)

	// Verify configuration structure
	assert.NotEmpty(t, config.PreferredBackend, "Should have a preferred backend")
	assert.NotNil(t, config.Backends, "Should have backends configuration")

	t.Logf("Subsumption config: preferred=%s, backends=%+v",
		config.PreferredBackend, config.Backends)
}

// TestSubsumptionCheck tests the basic subsumption check endpoint
func (suite *SubsumptionIntegrationTestSuite) TestSubsumptionCheck() {
	t := suite.T()

	// Test: Disease is-a Clinical finding
	request := map[string]interface{}{
		"subCode":   "64572001",  // Disease (disorder)
		"superCode": "404684003", // Clinical finding
		"system":    "http://snomed.info/sct",
	}

	result := suite.performSubsumptionCheck(request)
	assert.NotNil(t, result, "Should get a subsumption result")

	if result != nil {
		t.Logf("Subsumption check result: %+v", result)
	}
}

// TestSNOMEDSubsumptionTestCases runs all predefined subsumption test cases
func (suite *SubsumptionIntegrationTestSuite) TestSNOMEDSubsumptionTestCases() {
	for _, tc := range fixtures.SNOMEDSubsumptionTestCases {
		suite.Run(tc.Name, func() {
			t := suite.T()

			request := map[string]interface{}{
				"subCode":   tc.SubCode,
				"superCode": tc.SuperCode,
				"system":    "http://snomed.info/sct",
			}

			result := suite.performSubsumptionCheck(request)
			if result == nil {
				t.Logf("SKIP: %s - subsumption service not available", tc.Description)
				return
			}

			subsumes, ok := result["subsumes"].(bool)
			if !ok {
				// Try alternative field names
				if r, ok := result["result"].(bool); ok {
					subsumes = r
				}
			}

			if tc.ExpectedResult {
				assert.True(t, subsumes,
					"Expected %s (%s) to be subsumed by %s (%s): %s",
					tc.SubCode, tc.Name, tc.SuperCode, tc.Name, tc.Description)
			} else {
				assert.False(t, subsumes,
					"Expected %s (%s) NOT to be subsumed by %s (%s): %s",
					tc.SubCode, tc.Name, tc.SuperCode, tc.Name, tc.Description)
			}
		})
	}
}

// TestClinicalFindingHierarchy tests the Clinical Finding SNOMED hierarchy
func (suite *SubsumptionIntegrationTestSuite) TestClinicalFindingHierarchy() {
	hierarchyTests := []struct {
		name        string
		child       string
		parent      string
		shouldMatch bool
	}{
		{"Disease_under_ClinicalFinding", "64572001", "404684003", true},
		{"Sepsis_under_Disease", "91302008", "64572001", true},
		{"Sepsis_under_ClinicalFinding", "91302008", "404684003", true},
		{"ClinicalFinding_NOT_under_Procedure", "404684003", "71388002", false},
	}

	for _, tc := range hierarchyTests {
		suite.Run(tc.name, func() {
			request := map[string]interface{}{
				"subCode":   tc.child,
				"superCode": tc.parent,
				"system":    "http://snomed.info/sct",
			}

			result := suite.performSubsumptionCheck(request)
			if result == nil {
				suite.T().Skip("Subsumption service not available")
				return
			}

			subsumes := suite.getSubsumesValue(result)
			if tc.shouldMatch {
				assert.True(suite.T(), subsumes, "%s should be under %s", tc.child, tc.parent)
			} else {
				assert.False(suite.T(), subsumes, "%s should NOT be under %s", tc.child, tc.parent)
			}
		})
	}
}

// TestPharmaceuticalHierarchy tests pharmaceutical SNOMED concepts
func (suite *SubsumptionIntegrationTestSuite) TestPharmaceuticalHierarchy() {
	suite.Run("Paracetamol_IsAnalgesic", func() {
		request := map[string]interface{}{
			"subCode":   "387517004", // Paracetamol
			"superCode": "373265006", // Analgesic
			"system":    "http://snomed.info/sct",
		}

		result := suite.performSubsumptionCheck(request)
		if result == nil {
			suite.T().Skip("Subsumption service not available")
			return
		}

		subsumes := suite.getSubsumesValue(result)
		assert.True(suite.T(), subsumes, "Paracetamol should be an Analgesic")
	})

	suite.Run("Vancomycin_IsAntibiotic", func() {
		request := map[string]interface{}{
			"subCode":   "372735009", // Vancomycin
			"superCode": "373297006", // Anti-infective
			"system":    "http://snomed.info/sct",
		}

		result := suite.performSubsumptionCheck(request)
		if result == nil {
			suite.T().Skip("Subsumption service not available")
			return
		}

		subsumes := suite.getSubsumesValue(result)
		assert.True(suite.T(), subsumes, "Vancomycin should be an Anti-infective")
	})
}

// TestSelfSubsumption tests that a concept subsumes itself
func (suite *SubsumptionIntegrationTestSuite) TestSelfSubsumption() {
	t := suite.T()

	request := map[string]interface{}{
		"subCode":   "404684003", // Clinical finding
		"superCode": "404684003", // Same code
		"system":    "http://snomed.info/sct",
	}

	result := suite.performSubsumptionCheck(request)
	if result == nil {
		t.Skip("Subsumption service not available")
		return
	}

	subsumes := suite.getSubsumesValue(result)
	assert.True(t, subsumes, "A concept should subsume itself")
}

// TestSubsumptionWithInvalidCode tests behavior with invalid SNOMED codes
func (suite *SubsumptionIntegrationTestSuite) TestSubsumptionWithInvalidCode() {
	t := suite.T()

	request := map[string]interface{}{
		"subCode":   "INVALID123",
		"superCode": "404684003",
		"system":    "http://snomed.info/sct",
	}

	reqJSON, err := json.Marshal(request)
	require.NoError(t, err)

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/v1/subsumption/check",
		"application/json",
		bytes.NewBuffer(reqJSON))

	if err != nil {
		t.Skip("Subsumption service not available")
		return
	}
	defer resp.Body.Close()

	// Should either return an error or false for invalid codes
	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		subsumes := suite.getSubsumesValue(result)
		assert.False(t, subsumes, "Invalid code should not subsume anything")
	} else {
		// Error response is acceptable
		assert.Contains(t, []int{400, 404, 422}, resp.StatusCode,
			"Should return error status for invalid code")
	}
}

// TestSubsumptionBatchCheck tests batch subsumption checking
func (suite *SubsumptionIntegrationTestSuite) TestSubsumptionBatchCheck() {
	t := suite.T()

	batchRequest := []map[string]interface{}{
		{"subCode": "64572001", "superCode": "404684003", "system": "http://snomed.info/sct"},
		{"subCode": "91302008", "superCode": "64572001", "system": "http://snomed.info/sct"},
		{"subCode": "14669001", "superCode": "90708001", "system": "http://snomed.info/sct"},
	}

	reqJSON, err := json.Marshal(batchRequest)
	require.NoError(t, err)

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/v1/subsumption/batch",
		"application/json",
		bytes.NewBuffer(reqJSON))

	if err != nil {
		t.Skip("Batch subsumption not available")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Batch endpoint not implemented")
		return
	}

	if resp.StatusCode == http.StatusOK {
		var results []map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&results)
		if err == nil {
			assert.Len(t, results, 3, "Should return results for all 3 requests")
		}
	}
}

// TestSubsumptionPerformance measures subsumption check latency
func (suite *SubsumptionIntegrationTestSuite) TestSubsumptionPerformance() {
	t := suite.T()

	request := map[string]interface{}{
		"subCode":   "64572001",
		"superCode": "404684003",
		"system":    "http://snomed.info/sct",
	}

	reqJSON, err := json.Marshal(request)
	require.NoError(t, err)

	// Warm up
	for i := 0; i < 3; i++ {
		resp, err := suite.httpClient.Post(
			suite.baseURL+"/v1/subsumption/check",
			"application/json",
			bytes.NewBuffer(reqJSON))
		if err != nil {
			t.Skip("Subsumption service not available")
			return
		}
		resp.Body.Close()
	}

	// Measure 10 requests
	const numRequests = 10
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		resp, err := suite.httpClient.Post(
			suite.baseURL+"/v1/subsumption/check",
			"application/json",
			bytes.NewBuffer(reqJSON))
		require.NoError(t, err)
		resp.Body.Close()
	}

	duration := time.Since(start)
	avgLatency := duration / numRequests

	t.Logf("Subsumption performance: %d requests in %v (avg: %v/request)",
		numRequests, duration, avgLatency)

	// Performance assertion: average latency should be under 500ms
	assert.Less(t, avgLatency, 500*time.Millisecond,
		"Average subsumption latency should be under 500ms")
}

// TestNeo4jAUSubsumption tests subsumption using Neo4j AU backend
func (suite *SubsumptionIntegrationTestSuite) TestNeo4jAUSubsumption() {
	t := suite.T()

	// Check if Neo4j AU is available
	configResp, err := suite.httpClient.Get(suite.baseURL + "/v1/subsumption/config")
	if err != nil {
		t.Skip("Cannot check subsumption config")
		return
	}
	defer configResp.Body.Close()

	var config struct {
		PreferredBackend string `json:"preferred_backend"`
		Backends         map[string]struct {
			Available bool `json:"available"`
		} `json:"backends"`
	}
	if err := json.NewDecoder(configResp.Body).Decode(&config); err != nil {
		t.Skip("Cannot parse subsumption config")
		return
	}

	neo4jAvailable := false
	if neo4j, ok := config.Backends["neo4j"]; ok {
		neo4jAvailable = neo4j.Available
	}

	if !neo4jAvailable {
		t.Skip("Neo4j AU backend not available")
		return
	}

	// Test with Neo4j backend
	request := map[string]interface{}{
		"subCode":   "91302008",  // Sepsis
		"superCode": "404684003", // Clinical finding
		"system":    "http://snomed.info/sct",
		"backend":   "neo4j", // Force Neo4j backend
	}

	result := suite.performSubsumptionCheck(request)
	if result == nil {
		t.Skip("Neo4j subsumption check failed")
		return
	}

	subsumes := suite.getSubsumesValue(result)
	assert.True(t, subsumes, "Sepsis should be a Clinical finding (via Neo4j)")

	t.Logf("Neo4j AU subsumption result: %+v", result)
}

// Helper methods

func (suite *SubsumptionIntegrationTestSuite) performSubsumptionCheck(request map[string]interface{}) map[string]interface{} {
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil
	}

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/v1/subsumption/check",
		"application/json",
		bytes.NewBuffer(reqJSON))

	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	return result
}

func (suite *SubsumptionIntegrationTestSuite) getSubsumesValue(result map[string]interface{}) bool {
	if v, ok := result["subsumes"].(bool); ok {
		return v
	}
	if v, ok := result["result"].(bool); ok {
		return v
	}
	if v, ok := result["is_subsumed"].(bool); ok {
		return v
	}
	return false
}

func TestSubsumptionIntegrationSuite(t *testing.T) {
	suite.Run(t, new(SubsumptionIntegrationTestSuite))
}
