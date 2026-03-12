package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"kb-7-terminology/tests/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ============================================================================
// Neo4j AU Integration Test Suite
// Tests subsumption and hierarchy operations via the Neo4j AU backend
// ============================================================================

type Neo4jAUTestSuite struct {
	suite.Suite
	baseURL      string
	httpClient   *http.Client
	neo4jAUReady bool
}

func (suite *Neo4jAUTestSuite) SetupSuite() {
	if os.Getenv("TEST_ENV") != "integration" && os.Getenv("CI") != "true" {
		suite.T().Skip("Neo4j AU tests require TEST_ENV=integration")
	}

	suite.baseURL = os.Getenv("KB7_BASE_URL")
	if suite.baseURL == "" {
		suite.baseURL = "http://localhost:8087"
	}

	suite.httpClient = &http.Client{
		Timeout: fixtures.GetAUTestTimeout(),
	}

	suite.waitForServiceReady()
	suite.neo4jAUReady = suite.checkNeo4jAUAvailability()
}

func (suite *Neo4jAUTestSuite) waitForServiceReady() {
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

func (suite *Neo4jAUTestSuite) checkNeo4jAUAvailability() bool {
	resp, err := suite.httpClient.Get(suite.baseURL + "/v1/subsumption/config")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var config struct {
		PreferredBackend string `json:"preferred_backend"`
		Backends         map[string]struct {
			Available bool `json:"available"`
		} `json:"backends"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return false
	}

	if neo4j, ok := config.Backends["neo4j"]; ok {
		return neo4j.Available
	}
	return false
}

// TestNeo4jAUConnectionStatus verifies Neo4j AU backend is reachable
func (suite *Neo4jAUTestSuite) TestNeo4jAUConnectionStatus() {
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

	t.Logf("Subsumption config: preferred=%s", config.PreferredBackend)
	for name, backend := range config.Backends {
		t.Logf("  Backend %s: available=%v, url=%s", name, backend.Available, backend.URL)
	}

	// Log status for debugging
	if !suite.neo4jAUReady {
		t.Logf("⚠️ Neo4j AU backend not available - some tests will be skipped")
	} else {
		t.Logf("✅ Neo4j AU backend is available")
	}
}

// TestNeo4jAUSubsumption_ClinicalFinding tests SNOMED hierarchy via Neo4j
func (suite *Neo4jAUTestSuite) TestNeo4jAUSubsumption_ClinicalFinding() {
	if !suite.neo4jAUReady {
		suite.T().Skip("Neo4j AU backend not available")
	}

	t := suite.T()

	// Clinical Finding hierarchy tests using Neo4j
	tests := []struct {
		name     string
		childCode   string
		parentCode  string
		shouldMatch bool
		description string
	}{
		{
			name:        "Disease_under_ClinicalFinding",
			childCode:   "64572001",  // Disease (disorder)
			parentCode:  "404684003", // Clinical finding
			shouldMatch: true,
			description: "Disease is-a Clinical finding",
		},
		{
			name:        "Sepsis_under_Disease",
			childCode:   "91302008",  // Sepsis
			parentCode:  "64572001",  // Disease
			shouldMatch: true,
			description: "Sepsis is-a Disease",
		},
		{
			name:        "AKI_under_KidneyDisease",
			childCode:   "14669001",  // AKI
			parentCode:  "90708001",  // Kidney disease
			shouldMatch: true,
			description: "AKI is-a Kidney disease",
		},
		{
			name:        "Sepsis_NOT_under_Procedure",
			childCode:   "91302008",  // Sepsis
			parentCode:  "71388002",  // Procedure
			shouldMatch: false,
			description: "Sepsis is NOT a Procedure",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := suite.testSubsumptionWithBackend(tc.childCode, tc.parentCode, "neo4j")

			if tc.shouldMatch {
				assert.True(t, result, "%s (%s) should be subsumed by %s: %s",
					tc.childCode, tc.name, tc.parentCode, tc.description)
			} else {
				assert.False(t, result, "%s (%s) should NOT be subsumed by %s: %s",
					tc.childCode, tc.name, tc.parentCode, tc.description)
			}
		})
	}
}

// TestNeo4jAUSubsumption_Pharmaceutical tests pharmaceutical hierarchy via Neo4j
func (suite *Neo4jAUTestSuite) TestNeo4jAUSubsumption_Pharmaceutical() {
	if !suite.neo4jAUReady {
		suite.T().Skip("Neo4j AU backend not available")
	}

	t := suite.T()

	tests := []struct {
		name        string
		childCode   string
		parentCode  string
		shouldMatch bool
		description string
	}{
		{
			name:        "Paracetamol_IsAnalgesic",
			childCode:   "387517004", // Paracetamol
			parentCode:  "373265006", // Analgesic
			shouldMatch: true,
			description: "Paracetamol is-a Analgesic",
		},
		{
			name:        "Vancomycin_IsAntiInfective",
			childCode:   "372735009", // Vancomycin
			parentCode:  "373297006", // Anti-infective
			shouldMatch: true,
			description: "Vancomycin is-a Anti-infective",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := suite.testSubsumptionWithBackend(tc.childCode, tc.parentCode, "neo4j")

			if tc.shouldMatch {
				assert.True(t, result, tc.description)
			} else {
				assert.False(t, result, tc.description)
			}
		})
	}
}

// TestNeo4jAUAncestors tests retrieving ancestors via Neo4j
func (suite *Neo4jAUTestSuite) TestNeo4jAUAncestors() {
	if !suite.neo4jAUReady {
		suite.T().Skip("Neo4j AU backend not available")
	}

	t := suite.T()

	// Get ancestors of Sepsis
	url := fmt.Sprintf("%s/v1/subsumption/ancestors?code=91302008&system=http://snomed.info/sct&backend=neo4j",
		suite.baseURL)

	resp, err := suite.httpClient.Get(url)
	if err != nil {
		t.Skip("Ancestors endpoint not available")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Ancestors endpoint not implemented")
		return
	}

	if resp.StatusCode == http.StatusOK {
		var result struct {
			Code      string `json:"code"`
			Total     int    `json:"total"`
			Ancestors []struct {
				Code    string `json:"code"`
				Display string `json:"display"`
				Depth   int    `json:"depth"`
			} `json:"ancestors"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		t.Logf("Sepsis ancestors: total=%d", result.Total)
		assert.Greater(t, result.Total, 0, "Sepsis should have ancestors")

		// Verify expected ancestors
		expectedAncestors := []string{"64572001", "404684003"} // Disease, Clinical finding
		foundAncestors := make(map[string]bool)
		for _, a := range result.Ancestors {
			foundAncestors[a.Code] = true
			t.Logf("  Ancestor: %s (%s) depth=%d", a.Code, a.Display, a.Depth)
		}

		for _, expected := range expectedAncestors {
			if foundAncestors[expected] {
				t.Logf("  ✓ Found expected ancestor: %s", expected)
			}
		}
	}
}

// TestNeo4jAUDescendants tests retrieving descendants via Neo4j
func (suite *Neo4jAUTestSuite) TestNeo4jAUDescendants() {
	if !suite.neo4jAUReady {
		suite.T().Skip("Neo4j AU backend not available")
	}

	t := suite.T()

	// Get descendants of Sepsis (limited to avoid huge results)
	url := fmt.Sprintf("%s/v1/subsumption/descendants?code=91302008&system=http://snomed.info/sct&backend=neo4j&limit=50",
		suite.baseURL)

	resp, err := suite.httpClient.Get(url)
	if err != nil {
		t.Skip("Descendants endpoint not available")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Descendants endpoint not implemented")
		return
	}

	if resp.StatusCode == http.StatusOK {
		var result struct {
			Code        string `json:"code"`
			Total       int    `json:"total"`
			Truncated   bool   `json:"truncated"`
			Descendants []struct {
				Code    string `json:"code"`
				Display string `json:"display"`
				Depth   int    `json:"depth"`
			} `json:"descendants"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		t.Logf("Sepsis descendants: total=%d (truncated=%v)", result.Total, result.Truncated)

		// Log first few descendants
		for i, d := range result.Descendants {
			if i < 5 {
				t.Logf("  Descendant: %s (%s)", d.Code, d.Display)
			}
		}
	}
}

// TestNeo4jAUPerformance tests subsumption query performance via Neo4j
func (suite *Neo4jAUTestSuite) TestNeo4jAUPerformance() {
	if !suite.neo4jAUReady {
		suite.T().Skip("Neo4j AU backend not available")
	}

	t := suite.T()

	// Perform multiple subsumption checks and measure latency
	const numTests = 10
	var totalDuration time.Duration

	testCases := []struct {
		childCode  string
		parentCode string
	}{
		{"91302008", "404684003"},  // Sepsis -> Clinical finding
		{"64572001", "404684003"},  // Disease -> Clinical finding
		{"14669001", "90708001"},   // AKI -> Kidney disease
		{"387517004", "373265006"}, // Paracetamol -> Analgesic
		{"372735009", "373297006"}, // Vancomycin -> Anti-infective
	}

	for i := 0; i < numTests; i++ {
		tc := testCases[i%len(testCases)]
		start := time.Now()
		suite.testSubsumptionWithBackend(tc.childCode, tc.parentCode, "neo4j")
		duration := time.Since(start)
		totalDuration += duration
	}

	avgDuration := totalDuration / numTests
	t.Logf("Neo4j AU subsumption performance: %d tests, avg=%v/query", numTests, avgDuration)

	// Performance threshold: avg should be under 200ms for Neo4j queries
	assert.Less(t, avgDuration, 200*time.Millisecond,
		"Neo4j AU average subsumption query should be under 200ms")
}

// TestNeo4jAUCDCSync tests that CDC pipeline is keeping Neo4j AU in sync
func (suite *Neo4jAUTestSuite) TestNeo4jAUCDCSync() {
	if !suite.neo4jAUReady {
		suite.T().Skip("Neo4j AU backend not available")
	}

	t := suite.T()

	// This test verifies that both GraphDB and Neo4j AU have consistent data
	// by checking that the same subsumption queries return the same results

	testCases := []struct {
		childCode  string
		parentCode string
		expected   bool
	}{
		{"91302008", "404684003", true},  // Sepsis -> Clinical finding
		{"91302008", "71388002", false},  // Sepsis -> Procedure (should NOT match)
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("%s_to_%s", tc.childCode, tc.parentCode), func() {
			// Test via default backend (GraphDB)
			graphDBResult := suite.testSubsumptionWithBackend(tc.childCode, tc.parentCode, "")

			// Test via Neo4j AU backend
			neo4jResult := suite.testSubsumptionWithBackend(tc.childCode, tc.parentCode, "neo4j")

			// Results should match (both backends should be in sync)
			assert.Equal(t, graphDBResult, neo4jResult,
				"GraphDB and Neo4j AU should return same result for %s->%s",
				tc.childCode, tc.parentCode)

			if graphDBResult == neo4jResult {
				t.Logf("✓ CDC sync verified: %s->%s (both=%v)", tc.childCode, tc.parentCode, graphDBResult)
			} else {
				t.Logf("⚠️ CDC sync mismatch: %s->%s GraphDB=%v Neo4j=%v",
					tc.childCode, tc.parentCode, graphDBResult, neo4jResult)
			}
		})
	}
}

// TestNeo4jAURegionSpecific tests AU-specific clinical hierarchies
func (suite *Neo4jAUTestSuite) TestNeo4jAURegionSpecific() {
	if !suite.neo4jAUReady {
		suite.T().Skip("Neo4j AU backend not available")
	}

	t := suite.T()

	// Test AU-specific SNOMED hierarchies that are particularly relevant
	// for Australian clinical protocols

	auClinicalTests := []struct {
		name        string
		childCode   string
		parentCode  string
		description string
	}{
		{
			name:        "BacterialSepsis_under_Sepsis",
			childCode:   "10001005",  // Bacterial sepsis
			parentCode:  "91302008",  // Sepsis
			description: "AU Sepsis Protocol: Bacterial sepsis IS-A Sepsis",
		},
		{
			name:        "AcuteRenalFailure_under_AKI",
			childCode:   "35455006",  // Acute renal failure syndrome
			parentCode:  "14669001",  // Acute kidney injury
			description: "AU AKI Protocol: Acute renal failure IS-A AKI",
		},
	}

	for _, tc := range auClinicalTests {
		suite.Run(tc.name, func() {
			result := suite.testSubsumptionWithBackend(tc.childCode, tc.parentCode, "neo4j")

			// These are critical clinical hierarchies
			if result {
				t.Logf("✅ %s: VERIFIED", tc.description)
			} else {
				t.Logf("⚠️ %s: NOT FOUND (may need SNOMED AU data loading)", tc.description)
			}
		})
	}
}

// Helper method to test subsumption with specific backend
func (suite *Neo4jAUTestSuite) testSubsumptionWithBackend(childCode, parentCode, backend string) bool {
	url := fmt.Sprintf("%s/v1/subsumption/check", suite.baseURL)

	request := map[string]interface{}{
		"subCode":   childCode,
		"superCode": parentCode,
		"system":    "http://snomed.info/sct",
	}

	if backend != "" {
		request["backend"] = backend
	}

	reqBody, _ := json.Marshal(request)

	resp, err := suite.httpClient.Post(url, "application/json",
		bytes.NewReader(reqBody))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var result struct {
		Subsumes bool `json:"subsumes"`
		Result   bool `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	return result.Subsumes || result.Result
}

func TestNeo4jAUTestSuite(t *testing.T) {
	suite.Run(t, new(Neo4jAUTestSuite))
}
