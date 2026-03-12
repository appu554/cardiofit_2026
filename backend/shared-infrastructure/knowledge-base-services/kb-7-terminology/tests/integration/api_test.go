package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"kb-7-terminology/internal/config"
	"kb-7-terminology/internal/models"
	"kb-7-terminology/tests/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// APIIntegrationTestSuite provides real API integration testing with running service
type APIIntegrationTestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
	serverCmd  *exec.Cmd
	config     *config.Config
}

// SetupSuite runs once before all tests in the suite
func (suite *APIIntegrationTestSuite) SetupSuite() {
	// Check if we're in test environment
	if os.Getenv("TEST_ENV") != "docker" {
		suite.T().Skip("API integration tests require TEST_ENV=docker")
	}

	// Configure test environment
	suite.baseURL = "http://localhost:8085" // Different port for testing
	suite.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Set environment variables for the service
	os.Setenv("DATABASE_URL", fixtures.GetTestDatabaseURL())
	os.Setenv("REDIS_URL", fixtures.GetTestRedisURL())
	os.Setenv("PORT", "8085")
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("LOG_LEVEL", "6") // Error level only

	// Start the service in the background
	suite.startService()

	// Wait for service to be ready
	suite.waitForServiceReady()

	// Load test data
	suite.loadTestData()
}

// TearDownSuite runs once after all tests in the suite
func (suite *APIIntegrationTestSuite) TearDownSuite() {
	// Stop the service
	if suite.serverCmd != nil && suite.serverCmd.Process != nil {
		suite.serverCmd.Process.Kill()
		suite.serverCmd.Wait()
	}

	// Clean environment variables
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("PORT")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("LOG_LEVEL")
}

// startService starts the terminology service in background
func (suite *APIIntegrationTestSuite) startService() {
	// Build the service first
	buildCmd := exec.Command("go", "build", "-o", "kb-7-terminology-test", "./cmd/server")
	buildCmd.Dir = "../../" // Go back to service root
	err := buildCmd.Run()
	require.NoError(suite.T(), err, "Failed to build service for testing")

	// Start the service
	suite.serverCmd = exec.Command("./kb-7-terminology-test")
	suite.serverCmd.Dir = "../../"
	
	// Capture output for debugging
	suite.serverCmd.Stdout = os.Stdout
	suite.serverCmd.Stderr = os.Stderr

	err = suite.serverCmd.Start()
	require.NoError(suite.T(), err, "Failed to start service for testing")
}

// waitForServiceReady waits for the service to be ready to accept requests
func (suite *APIIntegrationTestSuite) waitForServiceReady() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			suite.T().Fatal("Service did not become ready within timeout")
		case <-ticker.C:
			resp, err := suite.httpClient.Get(suite.baseURL + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// loadTestData loads minimal test data for API testing
func (suite *APIIntegrationTestSuite) loadTestData() {
	// Load test terminology system via API
	system := fixtures.TestTerminologySystem
	systemJSON, err := json.Marshal(system)
	require.NoError(suite.T(), err)

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/api/v1/terminology-systems",
		"application/json",
		bytes.NewBuffer(systemJSON))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Accept either 201 (created) or 409 (already exists)
	assert.Contains(suite.T(), []int{201, 409}, resp.StatusCode)

	// Load test concepts via API
	for _, concept := range fixtures.TestConcepts {
		conceptJSON, err := json.Marshal(concept)
		require.NoError(suite.T(), err)

		resp, err := suite.httpClient.Post(
			suite.baseURL+"/api/v1/concepts",
			"application/json",
			bytes.NewBuffer(conceptJSON))
		require.NoError(suite.T(), err)
		resp.Body.Close()
		
		// Accept either 201 (created) or 409 (already exists)
		assert.Contains(suite.T(), []int{201, 409}, resp.StatusCode)
	}
}

// TestHealthEndpoint tests the health check endpoint
func (suite *APIIntegrationTestSuite) TestHealthEndpoint() {
	t := suite.T()

	resp, err := suite.httpClient.Get(suite.baseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var healthResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	require.NoError(t, err)

	assert.Equal(t, "kb-7-terminology", healthResponse["service"])
	assert.Equal(t, "healthy", healthResponse["status"])
	assert.Contains(t, healthResponse, "checks")
	assert.Contains(t, healthResponse, "timestamp")
}

// TestGetTerminologySystem tests retrieving a terminology system
func (suite *APIIntegrationTestSuite) TestGetTerminologySystem() {
	t := suite.T()

	systemID := fixtures.TestTerminologySystem.ID
	resp, err := suite.httpClient.Get(suite.baseURL + "/api/v1/terminology-systems/" + systemID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var system models.TerminologySystem
	err = json.NewDecoder(resp.Body).Decode(&system)
	require.NoError(t, err)

	assert.Equal(t, fixtures.TestTerminologySystem.ID, system.ID)
	assert.Equal(t, fixtures.TestTerminologySystem.SystemURI, system.SystemURI)
	assert.Equal(t, fixtures.TestTerminologySystem.SystemName, system.SystemName)
	assert.Equal(t, fixtures.TestTerminologySystem.Status, system.Status)
}

// TestGetTerminologySystemNotFound tests retrieving a non-existent system
func (suite *APIIntegrationTestSuite) TestGetTerminologySystemNotFound() {
	t := suite.T()

	resp, err := suite.httpClient.Get(suite.baseURL + "/api/v1/terminology-systems/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestLookupConcept tests concept lookup by code and system
func (suite *APIIntegrationTestSuite) TestLookupConcept() {
	t := suite.T()

	systemURI := fixtures.TestTerminologySystem.SystemURI
	code := fixtures.TestConcepts[0].Code

	url := fmt.Sprintf("%s/api/v1/concepts/lookup?system=%s&code=%s", 
		suite.baseURL, systemURI, code)
	
	resp, err := suite.httpClient.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var lookupResult models.LookupResult
	err = json.NewDecoder(resp.Body).Decode(&lookupResult)
	require.NoError(t, err)

	assert.Equal(t, fixtures.TestConcepts[0].Code, lookupResult.Concept.Code)
	assert.Equal(t, fixtures.TestConcepts[0].Display, lookupResult.Concept.Display)
	assert.Equal(t, fixtures.TestConcepts[0].SystemID, lookupResult.Concept.SystemID)
}

// TestLookupConceptNotFound tests lookup of non-existent concept
func (suite *APIIntegrationTestSuite) TestLookupConceptNotFound() {
	t := suite.T()

	systemURI := fixtures.TestTerminologySystem.SystemURI
	code := "NONEXISTENT123"

	url := fmt.Sprintf("%s/api/v1/concepts/lookup?system=%s&code=%s", 
		suite.baseURL, systemURI, code)
	
	resp, err := suite.httpClient.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestValidateCode tests code validation endpoint
func (suite *APIIntegrationTestSuite) TestValidateCode() {
	t := suite.T()

	// Test valid code
	validationRequest := map[string]interface{}{
		"code":   fixtures.TestConcepts[0].Code,
		"system": fixtures.TestTerminologySystem.SystemURI,
	}
	
	reqJSON, err := json.Marshal(validationRequest)
	require.NoError(t, err)

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/api/v1/codes/validate",
		"application/json",
		bytes.NewBuffer(reqJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var validationResult models.ValidationResult
	err = json.NewDecoder(resp.Body).Decode(&validationResult)
	require.NoError(t, err)

	assert.True(t, validationResult.Valid)
	assert.Equal(t, fixtures.TestConcepts[0].Code, validationResult.Code)
	assert.Equal(t, fixtures.TestTerminologySystem.SystemURI, validationResult.System)
	assert.Equal(t, fixtures.TestConcepts[0].Display, validationResult.Display)
	assert.Equal(t, "information", validationResult.Severity)
}

// TestValidateInvalidCode tests validation of invalid code
func (suite *APIIntegrationTestSuite) TestValidateInvalidCode() {
	t := suite.T()

	validationRequest := map[string]interface{}{
		"code":   "INVALID123",
		"system": fixtures.TestTerminologySystem.SystemURI,
	}
	
	reqJSON, err := json.Marshal(validationRequest)
	require.NoError(t, err)

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/api/v1/codes/validate",
		"application/json",
		bytes.NewBuffer(reqJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var validationResult models.ValidationResult
	err = json.NewDecoder(resp.Body).Decode(&validationResult)
	require.NoError(t, err)

	assert.False(t, validationResult.Valid)
	assert.Equal(t, "INVALID123", validationResult.Code)
	assert.Equal(t, fixtures.TestTerminologySystem.SystemURI, validationResult.System)
	assert.Equal(t, "error", validationResult.Severity)
	assert.Contains(t, validationResult.Message, "not found")
}

// TestSearchConcepts tests concept search functionality
func (suite *APIIntegrationTestSuite) TestSearchConcepts() {
	t := suite.T()

	searchQuery := map[string]interface{}{
		"query":     "Test Concept",
		"system":    fixtures.TestTerminologySystem.SystemURI,
		"count":     10,
		"offset":    0,
	}
	
	queryJSON, err := json.Marshal(searchQuery)
	require.NoError(t, err)

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/api/v1/concepts/search",
		"application/json",
		bytes.NewBuffer(queryJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var searchResults models.SearchResult
	err = json.NewDecoder(resp.Body).Decode(&searchResults)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(searchResults.Concepts), 1)
	assert.GreaterOrEqual(t, searchResults.Total, int64(1))

	// Verify first result contains expected data
	firstResult := searchResults.Concepts[0]
	assert.Contains(t, firstResult.Display, "Test Concept")
}

// TestConceptHierarchy tests hierarchical concept relationships
func (suite *APIIntegrationTestSuite) TestConceptHierarchy() {
	t := suite.T()

	// Get parent concept
	parentCode := fixtures.TestConcepts[0].Code
	systemURI := fixtures.TestTerminologySystem.SystemURI

	url := fmt.Sprintf("%s/api/v1/concepts/lookup?system=%s&code=%s&include_hierarchy=true", 
		suite.baseURL, systemURI, parentCode)
	
	resp, err := suite.httpClient.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var lookupResult models.LookupResult
	err = json.NewDecoder(resp.Body).Decode(&lookupResult)
	require.NoError(t, err)

	// Verify hierarchy information
	assert.Len(t, lookupResult.Concept.ParentCodes, 0) // Parent has no parents
	assert.Len(t, lookupResult.Concept.ChildCodes, 2) // Parent has 2 children
	assert.Contains(t, lookupResult.Concept.ChildCodes, "TEST002")
	assert.Contains(t, lookupResult.Concept.ChildCodes, "TEST003")
}

// TestAPIRateLimiting tests basic rate limiting behavior
func (suite *APIIntegrationTestSuite) TestAPIRateLimiting() {
	t := suite.T()

	// Make multiple rapid requests to test rate limiting
	const numRequests = 20
	var successCount, rateLimitCount int

	for i := 0; i < numRequests; i++ {
		resp, err := suite.httpClient.Get(suite.baseURL + "/health")
		require.NoError(t, err)
		
		if resp.StatusCode == http.StatusOK {
			successCount++
		} else if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitCount++
		}
		
		resp.Body.Close()
		time.Sleep(10 * time.Millisecond) // Small delay between requests
	}

	// At least some requests should succeed
	assert.GreaterOrEqual(t, successCount, 10, "At least 10 requests should succeed")
	
	// Log the rate limiting behavior for analysis
	t.Logf("Successful requests: %d, Rate limited: %d", successCount, rateLimitCount)
}

// TestAPIContentNegotiation tests content type handling
func (suite *APIIntegrationTestSuite) TestAPIContentNegotiation() {
	t := suite.T()

	// Test with different Accept headers
	testCases := []struct {
		acceptHeader     string
		expectedResponse string
	}{
		{"application/json", "application/json"},
		{"application/fhir+json", "application/fhir+json"},
		{"*/*", "application/json"}, // Default to JSON
	}

	for _, tc := range testCases {
		req, err := http.NewRequest("GET", suite.baseURL+"/health", nil)
		require.NoError(t, err)
		req.Header.Set("Accept", tc.acceptHeader)

		resp, err := suite.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), tc.expectedResponse)
	}
}

// TestAPICORSHeaders tests CORS header handling
func (suite *APIIntegrationTestSuite) TestAPICORSHeaders() {
	t := suite.T()

	// Test preflight request
	req, err := http.NewRequest("OPTIONS", suite.baseURL+"/api/v1/concepts/lookup", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := suite.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify CORS headers are present
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Headers"))
}

// TestAPIErrorHandling tests error response formatting
func (suite *APIIntegrationTestSuite) TestAPIErrorHandling() {
	t := suite.T()

	// Test malformed JSON request
	malformedJSON := `{"code": "TEST001", "system":}`
	
	resp, err := suite.httpClient.Post(
		suite.baseURL+"/api/v1/codes/validate",
		"application/json",
		bytes.NewBufferString(malformedJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errorResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)

	assert.Contains(t, errorResponse, "error")
	assert.Contains(t, errorResponse, "message")
	assert.Contains(t, errorResponse, "timestamp")
}

// Run the API integration test suite
func TestAPIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(APIIntegrationTestSuite))
}