package integration

import (
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

// ValueSetIntegrationTestSuite tests all 49 value sets and their operations
type ValueSetIntegrationTestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
}

// SetupSuite runs once before all tests in the suite
func (suite *ValueSetIntegrationTestSuite) SetupSuite() {
	// Check for integration test environment
	if os.Getenv("TEST_ENV") != "integration" && os.Getenv("CI") != "true" {
		suite.T().Skip("Value set integration tests require TEST_ENV=integration")
	}

	suite.baseURL = os.Getenv("KB7_BASE_URL")
	if suite.baseURL == "" {
		suite.baseURL = "http://localhost:8087"
	}

	suite.httpClient = &http.Client{
		Timeout: fixtures.GetAUTestTimeout(),
	}

	// Wait for service to be ready
	suite.waitForServiceReady()

	// Ensure value sets are seeded
	suite.seedValueSets()
}

// waitForServiceReady waits for the KB-7 service to be available
func (suite *ValueSetIntegrationTestSuite) waitForServiceReady() {
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
func (suite *ValueSetIntegrationTestSuite) seedValueSets() {
	resp, err := suite.httpClient.Post(suite.baseURL+"/v1/rules/seed", "application/json", nil)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	// Accept both 200 (already seeded) and 201 (newly seeded)
	assert.Contains(suite.T(), []int{200, 201}, resp.StatusCode)
}

// TestValueSetCount verifies all 49 value sets are present
func (suite *ValueSetIntegrationTestSuite) TestValueSetCount() {
	t := suite.T()

	resp, err := suite.httpClient.Get(suite.baseURL + "/v1/valuesets/builtin/count")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var countResult struct {
		BuiltinCount int `json:"builtin_count"`
	}
	err = json.NewDecoder(resp.Body).Decode(&countResult)
	require.NoError(t, err)

	assert.Equal(t, fixtures.ExpectedValueSetCounts.Total, countResult.BuiltinCount,
		"Expected %d value sets, got %d", fixtures.ExpectedValueSetCounts.Total, countResult.BuiltinCount)
}

// TestAllValueSetsExist verifies each of the 49 value sets is present
func (suite *ValueSetIntegrationTestSuite) TestAllValueSetsExist() {
	t := suite.T()

	resp, err := suite.httpClient.Get(suite.baseURL + "/v1/rules/valuesets?limit=100")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResult struct {
		ValueSets []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"value_sets"`
		Total int `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&listResult)
	require.NoError(t, err)

	// Create a map of existing value sets for easy lookup
	existingVS := make(map[string]bool)
	for _, vs := range listResult.ValueSets {
		existingVS[vs.Name] = true
		existingVS[vs.ID] = true
	}

	// Check each expected value set exists
	for _, expectedName := range fixtures.AllValueSetNames {
		found := existingVS[expectedName]
		assert.True(t, found, "Value set '%s' should exist", expectedName)
	}
}

// TestFHIRAdministrativeValueSets verifies all 18 FHIR R4 administrative value sets
func (suite *ValueSetIntegrationTestSuite) TestFHIRAdministrativeValueSets() {
	t := suite.T()

	fhirAdminValueSets := []string{
		"AdministrativeGender",
		"AddressType",
		"AddressUse",
		"ContactPointSystem",
		"ContactPointUse",
		"IdentifierUse",
		"NameUse",
		"PublicationStatus",
		"NarrativeStatus",
		"QuantityComparator",
		"ResourceTypes",
		"Languages",
		"MaritalStatus",
		"ContactRelationship",
		"AllergyIntoleranceCategory",
		"AllergyIntoleranceCriticality",
		"AllergyIntoleranceSeverity",
		"AllergyIntoleranceType",
	}

	for _, vsName := range fhirAdminValueSets {
		suite.Run(fmt.Sprintf("FHIR_%s", vsName), func() {
			suite.verifyValueSetExists(vsName)
		})
	}

	// Verify count matches expected
	assert.Equal(t, fixtures.ExpectedValueSetCounts.FHIRAdmin, len(fhirAdminValueSets),
		"FHIR Administrative value set count mismatch")
}

// TestAUSpecificValueSets verifies all 6 AU-specific clinical value sets
func (suite *ValueSetIntegrationTestSuite) TestAUSpecificValueSets() {
	t := suite.T()

	for _, vsName := range fixtures.AUSpecificValueSets {
		suite.Run(fmt.Sprintf("AU_%s", vsName), func() {
			suite.verifyValueSetExists(vsName)
			suite.verifyValueSetHasConcepts(vsName)
		})
	}

	assert.Equal(t, fixtures.ExpectedValueSetCounts.AUSpecific, len(fixtures.AUSpecificValueSets),
		"AU-specific value set count mismatch")
}

// TestClinicalConditionValueSets verifies condition-related value sets
func (suite *ValueSetIntegrationTestSuite) TestClinicalConditionValueSets() {
	conditionValueSets := []string{
		"InfectionSource",
		"Hypertension",
		"AtrialFibrillation",
		"IschemicStroke",
		"ActiveBleeding",
		"RespiratoryFailure",
		"DiabetesMellitus",
		"HeartFailure",
		"RenalConditions",
		"CardiacConditions",
	}

	for _, vsName := range conditionValueSets {
		suite.Run(fmt.Sprintf("Condition_%s", vsName), func() {
			suite.verifyValueSetExists(vsName)
		})
	}
}

// TestMedicationValueSets verifies medication-related value sets
func (suite *ValueSetIntegrationTestSuite) TestMedicationValueSets() {
	medicationValueSets := []string{
		"BroadSpectrumAntibiotics",
		"Anticoagulants",
		"ACEInhibitors",
		"NSAIDs",
		"BetaBlockers",
		"Statins",
	}

	for _, vsName := range medicationValueSets {
		suite.Run(fmt.Sprintf("Medication_%s", vsName), func() {
			suite.verifyValueSetExists(vsName)
		})
	}
}

// TestLabValueSets verifies laboratory-related value sets
func (suite *ValueSetIntegrationTestSuite) TestLabValueSets() {
	labValueSets := []string{
		"LabTroponin",
		"LabINR",
		"LabEGFR",
		"LabBloodCulture",
		"LabBNP",
		"LabCreatinine",
		"LabLactate",
	}

	for _, vsName := range labValueSets {
		suite.Run(fmt.Sprintf("Lab_%s", vsName), func() {
			suite.verifyValueSetExists(vsName)
		})
	}
}

// TestProcedureValueSets verifies procedure-related value sets
func (suite *ValueSetIntegrationTestSuite) TestProcedureValueSets() {
	procedureValueSets := []string{
		"ProcDialysis",
		"ProcCTContrast",
	}

	for _, vsName := range procedureValueSets {
		suite.Run(fmt.Sprintf("Procedure_%s", vsName), func() {
			suite.verifyValueSetExists(vsName)
		})
	}
}

// TestValueSetMembership tests code membership in value sets
func (suite *ValueSetIntegrationTestSuite) TestValueSetMembership() {
	for _, tc := range fixtures.ValueSetMembershipTestCases {
		suite.Run(tc.Description, func() {
			result := suite.checkValueSetContains(tc.ValueSetName, tc.Code, tc.System)
			assert.Equal(suite.T(), tc.ExpectedMember, result,
				"Membership mismatch for code %s in value set %s", tc.Code, tc.ValueSetName)
		})
	}
}

// TestAdministrativeGenderValueSet validates AdministrativeGender value set contents
func (suite *ValueSetIntegrationTestSuite) TestAdministrativeGenderValueSet() {
	t := suite.T()

	expectedCodes := []struct {
		Code    string
		Display string
	}{
		{"male", "Male"},
		{"female", "Female"},
		{"other", "Other"},
		{"unknown", "Unknown"},
	}

	for _, expected := range expectedCodes {
		contains := suite.checkValueSetContains("AdministrativeGender", expected.Code, "http://hl7.org/fhir/administrative-gender")
		assert.True(t, contains, "AdministrativeGender should contain code '%s'", expected.Code)
	}
}

// TestAUSepsisConditionsValueSet validates AU Sepsis Conditions value set
func (suite *ValueSetIntegrationTestSuite) TestAUSepsisConditionsValueSet() {
	t := suite.T()

	// SNOMED CT codes that should be in AUSepsisConditions
	sepsisCodesExpected := []string{
		"91302008",  // Sepsis
		"76571007",  // Septic shock
		"10001005",  // Bacterial sepsis
	}

	for _, code := range sepsisCodesExpected {
		contains := suite.checkValueSetContains("AUSepsisConditions", code, "http://snomed.info/sct")
		assert.True(t, contains, "AUSepsisConditions should contain SNOMED code '%s'", code)
	}
}

// TestAUAKIConditionsValueSet validates AU AKI Conditions value set
func (suite *ValueSetIntegrationTestSuite) TestAUAKIConditionsValueSet() {
	t := suite.T()

	// SNOMED CT codes that should be in AUAKIConditions
	akiCodesExpected := []string{
		"14669001", // Acute kidney injury
		"35455006", // Acute renal failure syndrome
	}

	for _, code := range akiCodesExpected {
		contains := suite.checkValueSetContains("AUAKIConditions", code, "http://snomed.info/sct")
		assert.True(t, contains, "AUAKIConditions should contain SNOMED code '%s'", code)
	}
}

// TestValueSetCategorization verifies value sets are properly categorized
func (suite *ValueSetIntegrationTestSuite) TestValueSetCategorization() {
	t := suite.T()

	categories := make(map[string]int)
	for _, category := range fixtures.ValueSetCategoryMap {
		categories[category]++
	}

	// Verify category distribution
	assert.Greater(t, categories["administrative"], 0, "Should have administrative value sets")
	assert.Greater(t, categories["au-clinical"], 0, "Should have AU clinical value sets")
	assert.Greater(t, categories["conditions"], 0, "Should have condition value sets")
	assert.Greater(t, categories["medications"], 0, "Should have medication value sets")
	assert.Greater(t, categories["labs"], 0, "Should have lab value sets")
	assert.Greater(t, categories["procedures"], 0, "Should have procedure value sets")

	t.Logf("Value set category distribution: %+v", categories)
}

// TestValueSetExpansion tests that value sets can be expanded
func (suite *ValueSetIntegrationTestSuite) TestValueSetExpansion() {
	// Test a few representative value sets for expansion
	testValueSets := []string{
		"AdministrativeGender",
		"AUSepsisConditions",
		"BroadSpectrumAntibiotics",
	}

	for _, vsName := range testValueSets {
		suite.Run(fmt.Sprintf("Expand_%s", vsName), func() {
			expansion := suite.getValueSetExpansion(vsName)
			assert.NotNil(suite.T(), expansion, "Should be able to expand value set %s", vsName)
			if expansion != nil {
				assert.Greater(suite.T(), len(expansion), 0, "Expansion should contain concepts")
			}
		})
	}
}

// Helper methods

// verifyValueSetExists checks that a value set exists by name
func (suite *ValueSetIntegrationTestSuite) verifyValueSetExists(vsName string) {
	t := suite.T()

	resp, err := suite.httpClient.Get(suite.baseURL + "/v1/rules/valuesets?limit=100")
	require.NoError(t, err)
	defer resp.Body.Close()

	var listResult struct {
		ValueSets []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"value_sets"`
	}
	err = json.NewDecoder(resp.Body).Decode(&listResult)
	require.NoError(t, err)

	found := false
	for _, vs := range listResult.ValueSets {
		if vs.Name == vsName || vs.ID == vsName {
			found = true
			break
		}
	}

	assert.True(t, found, "Value set '%s' should exist", vsName)
}

// verifyValueSetHasConcepts checks that a value set has at least one concept
func (suite *ValueSetIntegrationTestSuite) verifyValueSetHasConcepts(vsName string) {
	t := suite.T()

	expansion := suite.getValueSetExpansion(vsName)
	assert.NotNil(t, expansion, "Value set '%s' should have an expansion", vsName)
	if expansion != nil {
		assert.Greater(t, len(expansion), 0, "Value set '%s' should have concepts", vsName)
	}
}

// checkValueSetContains checks if a code is in a value set
func (suite *ValueSetIntegrationTestSuite) checkValueSetContains(vsName, code, system string) bool {
	url := fmt.Sprintf("%s/v1/rules/valuesets/%s/contains?code=%s&system=%s",
		suite.baseURL, vsName, code, system)

	resp, err := suite.httpClient.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var result struct {
		Contains bool `json:"contains"`
		Result   bool `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	return result.Contains || result.Result
}

// getValueSetExpansion retrieves the expansion of a value set
func (suite *ValueSetIntegrationTestSuite) getValueSetExpansion(vsName string) []map[string]interface{} {
	url := fmt.Sprintf("%s/v1/rules/valuesets/%s", suite.baseURL, vsName)

	resp, err := suite.httpClient.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result struct {
		Expansion struct {
			Contains []map[string]interface{} `json:"contains"`
		} `json:"expansion"`
		Concepts []map[string]interface{} `json:"concepts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	if len(result.Expansion.Contains) > 0 {
		return result.Expansion.Contains
	}
	return result.Concepts
}

// Run the value set integration test suite
func TestValueSetIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ValueSetIntegrationTestSuite))
}
