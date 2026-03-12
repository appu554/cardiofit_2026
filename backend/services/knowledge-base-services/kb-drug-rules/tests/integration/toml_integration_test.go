package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"kb-drug-rules/internal/models"
)

// TOMLIntegrationTestSuite tests the complete TOML workflow
type TOMLIntegrationTestSuite struct {
	suite.Suite
	server *httptest.Server
	client *http.Client
}

func (suite *TOMLIntegrationTestSuite) SetupSuite() {
	// Initialize test server with all TOML enhancements
	suite.server = httptest.NewServer(setupTestRouter())
	suite.client = &http.Client{Timeout: 30 * time.Second}
}

func (suite *TOMLIntegrationTestSuite) TearDownSuite() {
	suite.server.Close()
}

// TestCompleteWorkflow tests the entire TOML workflow end-to-end
func (suite *TOMLIntegrationTestSuite) TestCompleteWorkflow() {
	// Step 1: Validate TOML content
	tomlContent := `
[meta]
drug_id = "metformin_integration_test"
name = "Metformin Integration Test"
version = "1.0.0"
clinical_reviewer = "Dr. Integration Test"
therapeutic_class = "Antidiabetic"

[indications]
primary = "Type 2 Diabetes Mellitus"
secondary = ["Polycystic Ovary Syndrome", "Prediabetes"]

[dose_calculation]
base_dose_mg = 500.0
max_daily_dose_mg = 2550.0
titration_interval_days = 7

[safety_verification]
contraindications = ["Severe renal impairment", "Metabolic acidosis"]
monitoring_requirements = ["Renal function", "Vitamin B12 levels"]

[drug_interactions]
major = ["Contrast agents", "Alcohol"]
moderate = ["Furosemide", "Nifedipine"]
`

	// Test validation
	validationResp := suite.validateTOML(tomlContent)
	assert.True(suite.T(), validationResp.IsValid, "TOML validation should pass")
	assert.NotEmpty(suite.T(), validationResp.ConvertedJSON, "Should have converted JSON")

	// Step 2: Test format conversion
	conversionResp := suite.convertFormat(tomlContent, "toml", "json")
	assert.Equal(suite.T(), "json", conversionResp.TargetFormat)
	assert.NotEmpty(suite.T(), conversionResp.ConvertedContent)

	// Step 3: Hotload the TOML rule
	hotloadResp := suite.hotloadTOML(tomlContent, "metformin_integration_test", "1.0.0")
	assert.Equal(suite.T(), "metformin_integration_test", hotloadResp["drug_id"])
	assert.Equal(suite.T(), "1.0.0", hotloadResp["version"])

	// Step 4: Retrieve the rule and verify it was stored correctly
	retrievedRule := suite.getDrugRule("metformin_integration_test")
	assert.Equal(suite.T(), "metformin_integration_test", retrievedRule.DrugID)
	assert.Equal(suite.T(), "toml", retrievedRule.OriginalFormat)
	assert.NotNil(suite.T(), retrievedRule.TOMLContent)

	// Step 5: Update with a new version
	updatedTOML := `
[meta]
drug_id = "metformin_integration_test"
name = "Metformin Integration Test Updated"
version = "1.1.0"
clinical_reviewer = "Dr. Integration Test"
therapeutic_class = "Antidiabetic"

[indications]
primary = "Type 2 Diabetes Mellitus"
secondary = ["Polycystic Ovary Syndrome", "Prediabetes", "Insulin Resistance"]

[dose_calculation]
base_dose_mg = 750.0
max_daily_dose_mg = 2550.0
titration_interval_days = 7

[safety_verification]
contraindications = ["Severe renal impairment", "Metabolic acidosis", "Heart failure"]
monitoring_requirements = ["Renal function", "Vitamin B12 levels", "Liver function"]
`

	// Hotload updated version
	suite.hotloadTOML(updatedTOML, "metformin_integration_test", "1.1.0")

	// Step 6: Test version history
	versionHistory := suite.getVersionHistory("metformin_integration_test")
	assert.GreaterOrEqual(suite.T(), len(versionHistory.Versions), 2, "Should have at least 2 versions")

	// Step 7: Test rollback functionality
	rollbackResp := suite.rollbackVersion("metformin_integration_test", "1.0.0", "Integration test rollback")
	assert.True(suite.T(), rollbackResp.Success, "Rollback should succeed")

	// Step 8: Verify rollback worked
	currentRule := suite.getDrugRule("metformin_integration_test")
	// The rollback creates a new version, so we check the content matches the original
	var currentContent map[string]interface{}
	json.Unmarshal(currentRule.JSONContent, &currentContent)
	
	meta := currentContent["meta"].(map[string]interface{})
	doseCalc := currentContent["dose_calculation"].(map[string]interface{})
	
	assert.Equal(suite.T(), 500.0, doseCalc["base_dose_mg"], "Dose should be rolled back to original value")
}

// TestBatchOperations tests batch loading functionality
func (suite *TOMLIntegrationTestSuite) TestBatchOperations() {
	rules := []map[string]interface{}{
		{
			"drug_id":           "batch_test_1",
			"version":           "1.0.0",
			"toml_content":      suite.generateTestTOML("batch_test_1", "Batch Test Drug 1"),
			"signed_by":         "test_signer",
			"clinical_reviewer": "Dr. Batch Test",
			"regions":           []string{"US"},
			"tags":              []string{"test", "batch"},
		},
		{
			"drug_id":           "batch_test_2",
			"version":           "1.0.0",
			"toml_content":      suite.generateTestTOML("batch_test_2", "Batch Test Drug 2"),
			"signed_by":         "test_signer",
			"clinical_reviewer": "Dr. Batch Test",
			"regions":           []string{"EU"},
			"tags":              []string{"test", "batch"},
		},
	}

	batchResp := suite.batchLoadRules(rules, "batch_test_user")
	assert.Equal(suite.T(), 2, batchResp.TotalRules)
	assert.Equal(suite.T(), 2, batchResp.SuccessfulRules)
	assert.Equal(suite.T(), 0, batchResp.FailedRules)
}

// TestErrorHandling tests various error scenarios
func (suite *TOMLIntegrationTestSuite) TestErrorHandling() {
	// Test invalid TOML syntax
	invalidTOML := `
[meta
drug_id = "invalid_test"
name = "Invalid Test"
`
	validationResp := suite.validateTOML(invalidTOML)
	assert.False(suite.T(), validationResp.IsValid, "Invalid TOML should fail validation")
	assert.NotEmpty(suite.T(), validationResp.Errors, "Should have validation errors")

	// Test missing required fields
	incompleteTOML := `
[meta]
drug_id = "incomplete_test"
# Missing required fields
`
	validationResp = suite.validateTOML(incompleteTOML)
	assert.False(suite.T(), validationResp.IsValid, "Incomplete TOML should fail validation")

	// Test rollback to non-existent version
	rollbackResp := suite.rollbackVersion("nonexistent_drug", "1.0.0", "Test rollback")
	assert.False(suite.T(), rollbackResp.Success, "Rollback to non-existent version should fail")
}

// TestPerformance tests performance requirements
func (suite *TOMLIntegrationTestSuite) TestPerformance() {
	tomlContent := suite.generateTestTOML("performance_test", "Performance Test Drug")

	// Test validation performance
	start := time.Now()
	for i := 0; i < 100; i++ {
		suite.validateTOML(tomlContent)
	}
	avgValidationTime := time.Since(start) / 100
	assert.Less(suite.T(), avgValidationTime, 100*time.Millisecond, "Average validation time should be < 100ms")

	// Test conversion performance
	start = time.Now()
	for i := 0; i < 100; i++ {
		suite.convertFormat(tomlContent, "toml", "json")
	}
	avgConversionTime := time.Since(start) / 100
	assert.Less(suite.T(), avgConversionTime, 50*time.Millisecond, "Average conversion time should be < 50ms")
}

// Helper methods

func (suite *TOMLIntegrationTestSuite) validateTOML(content string) models.TOMLValidationResponse {
	request := map[string]interface{}{
		"content": content,
		"format":  "toml",
	}
	
	resp := suite.makeRequest("POST", "/v1/validate-toml", request)
	var response models.TOMLValidationResponse
	json.NewDecoder(resp.Body).Decode(&response)
	return response
}

func (suite *TOMLIntegrationTestSuite) convertFormat(content, sourceFormat, targetFormat string) models.FormatConversionResponse {
	request := map[string]interface{}{
		"content":       content,
		"source_format": sourceFormat,
		"target_format": targetFormat,
	}
	
	resp := suite.makeRequest("POST", "/v1/convert", request)
	var response models.FormatConversionResponse
	json.NewDecoder(resp.Body).Decode(&response)
	return response
}

func (suite *TOMLIntegrationTestSuite) hotloadTOML(content, drugID, version string) map[string]interface{} {
	request := map[string]interface{}{
		"drug_id":           drugID,
		"version":           version,
		"toml_content":      content,
		"signed_by":         "test_signer",
		"clinical_reviewer": "Dr. Test",
		"regions":           []string{"US"},
		"tags":              []string{"test"},
	}
	
	resp := suite.makeRequest("POST", "/v1/hotload-toml", request)
	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	return response
}

func (suite *TOMLIntegrationTestSuite) getDrugRule(drugID string) models.DrugRulePack {
	resp := suite.makeRequest("GET", "/v1/items/"+drugID, nil)
	var response models.DrugRulePack
	json.NewDecoder(resp.Body).Decode(&response)
	return response
}

func (suite *TOMLIntegrationTestSuite) getVersionHistory(drugID string) models.VersionHistoryResponse {
	resp := suite.makeRequest("GET", "/v1/versions/"+drugID+"/history", nil)
	var response models.VersionHistoryResponse
	json.NewDecoder(resp.Body).Decode(&response)
	return response
}

func (suite *TOMLIntegrationTestSuite) rollbackVersion(drugID, targetVersion, reason string) models.RollbackResponse {
	request := map[string]interface{}{
		"drug_id":        drugID,
		"target_version": targetVersion,
		"reason":         reason,
		"user":           "test_user",
	}
	
	resp := suite.makeRequest("POST", "/v1/rollback", request)
	var response models.RollbackResponse
	json.NewDecoder(resp.Body).Decode(&response)
	return response
}

func (suite *TOMLIntegrationTestSuite) batchLoadRules(rules []map[string]interface{}, user string) models.BatchLoadResponse {
	request := map[string]interface{}{
		"rules": rules,
		"user":  user,
	}
	
	resp := suite.makeRequest("POST", "/v1/batch-load", request)
	var response models.BatchLoadResponse
	json.NewDecoder(resp.Body).Decode(&response)
	return response
}

func (suite *TOMLIntegrationTestSuite) makeRequest(method, path string, body interface{}) *http.Response {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonData, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	
	req, _ := http.NewRequest(method, suite.server.URL+path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := suite.client.Do(req)
	assert.NoError(suite.T(), err)
	return resp
}

func (suite *TOMLIntegrationTestSuite) generateTestTOML(drugID, name string) string {
	return fmt.Sprintf(`
[meta]
drug_id = "%s"
name = "%s"
version = "1.0.0"
clinical_reviewer = "Dr. Test"

[dose_calculation]
base_dose_mg = 500.0
max_daily_dose_mg = 2000.0
`, drugID, name)
}

func TestTOMLIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TOMLIntegrationTestSuite))
}

// setupTestRouter would initialize the test router with all TOML enhancements
func setupTestRouter() http.Handler {
	// This would be implemented to set up the test environment
	// with all the TOML support features enabled
	return http.NewServeMux()
}
