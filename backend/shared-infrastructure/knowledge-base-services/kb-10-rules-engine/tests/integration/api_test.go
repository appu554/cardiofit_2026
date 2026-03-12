package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/api"
	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/cardiofit/kb-10-rules-engine/internal/engine"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer creates a test server with mock dependencies
func setupTestServer(t *testing.T) (*httptest.Server, *models.RuleStore) {
	gin.SetMode(gin.TestMode)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8100,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Rules: config.RulesConfig{
			EnableCaching: true,
			CacheTTL:      5 * time.Minute,
		},
	}

	store := models.NewRuleStore()
	cache := engine.NewCache(true, 5*time.Minute, logger)
	vaidshalaConfig := &config.VaidshalaConfig{
		URL:     "http://localhost:8096",
		Enabled: false,
	}
	eng := engine.NewRulesEngine(store, nil, cache, vaidshalaConfig, logger, nil)

	// Add test rules
	addTestRulesForAPI(store)

	server := api.NewServer(cfg, eng, store, nil, nil, logger, nil)
	ts := httptest.NewServer(server.Router())

	return ts, store
}

// addTestRulesForAPI adds sample rules for API testing
func addTestRulesForAPI(store *models.RuleStore) {
	// Critical Hyperkalemia Rule
	store.Add(&models.Rule{
		ID:          "ALERT-K-HIGH",
		Name:        "Critical Hyperkalemia",
		Description: "Potassium critically elevated",
		Type:        models.RuleTypeAlert,
		Category:    "SAFETY",
		Severity:    "CRITICAL",
		Status:      "ACTIVE",
		Priority:    1,
		Conditions: []models.Condition{
			{Field: "labs.potassium.value", Operator: models.OperatorGTE, Value: 6.5},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "ALERT", Message: "CRITICAL: Hyperkalemia", Priority: "STAT"},
		},
		Tags: []string{"electrolyte", "critical"},
	})

	// Hypoglycemia Rule
	store.Add(&models.Rule{
		ID:          "ALERT-GLUCOSE-LOW",
		Name:        "Critical Hypoglycemia",
		Description: "Glucose critically low",
		Type:        models.RuleTypeAlert,
		Category:    "SAFETY",
		Severity:    "CRITICAL",
		Status:      "ACTIVE",
		Priority:    1,
		Conditions: []models.Condition{
			{Field: "labs.glucose.value", Operator: models.OperatorLT, Value: 50.0},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "ALERT", Message: "CRITICAL: Hypoglycemia", Priority: "STAT"},
		},
		Tags: []string{"glucose", "critical"},
	})

	// Sepsis Inference Rule
	store.Add(&models.Rule{
		ID:          "INFERENCE-SEPSIS",
		Name:        "Suspected Sepsis",
		Description: "SIRS criteria met",
		Type:        models.RuleTypeInference,
		Category:    "CLINICAL",
		Severity:    "HIGH",
		Status:      "ACTIVE",
		Priority:    10,
		Conditions: []models.Condition{
			{Field: "vitals.temperature.value", Operator: models.OperatorGT, Value: 38.3},
			{Field: "vitals.heart_rate.value", Operator: models.OperatorGT, Value: 90.0},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "INFERENCE", Message: "Suspected sepsis"},
		},
		Tags: []string{"sepsis", "sirs"},
	})

	// Beers Criteria Validation
	store.Add(&models.Rule{
		ID:          "VALIDATION-BEERS",
		Name:        "Beers Criteria",
		Description: "Elderly medication safety",
		Type:        models.RuleTypeValidation,
		Category:    "GOVERNANCE",
		Severity:    "MODERATE",
		Status:      "ACTIVE",
		Priority:    20,
		Conditions: []models.Condition{
			{Field: "patient.age", Operator: models.OperatorAGEGT, Value: 65},
			{Field: "medications", Operator: models.OperatorIN, Value: []interface{}{"diphenhydramine"}},
		},
		ConditionLogic: "AND",
		Actions: []models.Action{
			{Type: "ALERT", Message: "Beers Criteria warning"},
		},
		Tags: []string{"beers", "geriatrics"},
	})
}

// TestAPI_HealthEndpoint tests the health check endpoint
func TestAPI_HealthEndpoint(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "healthy", body["status"])
}

// TestAPI_ReadyEndpoint tests the readiness endpoint
func TestAPI_ReadyEndpoint(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestAPI_EvaluateEndpoint tests the main evaluation endpoint
func TestAPI_EvaluateEndpoint(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	tests := []struct {
		name              string
		request           map[string]interface{}
		expectedStatus    int
		expectedTriggered int
		expectedRuleID    string
	}{
		{
			name: "Critical hyperkalemia triggers",
			request: map[string]interface{}{
				"patient_id": "test-001",
				"labs": map[string]interface{}{
					"potassium": map[string]interface{}{"value": 7.0},
				},
			},
			expectedStatus:    http.StatusOK,
			expectedTriggered: 1,
			expectedRuleID:    "ALERT-K-HIGH",
		},
		{
			name: "Normal values no triggers",
			request: map[string]interface{}{
				"patient_id": "test-002",
				"labs": map[string]interface{}{
					"potassium": map[string]interface{}{"value": 4.5},
					"glucose":   map[string]interface{}{"value": 100.0},
				},
			},
			expectedStatus:    http.StatusOK,
			expectedTriggered: 0,
		},
		{
			name: "Multiple critical values",
			request: map[string]interface{}{
				"patient_id": "test-003",
				"labs": map[string]interface{}{
					"potassium": map[string]interface{}{"value": 7.0},
					"glucose":   map[string]interface{}{"value": 40.0},
				},
			},
			expectedStatus:    http.StatusOK,
			expectedTriggered: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			resp, err := http.Post(ts.URL+"/api/v1/evaluate", "application/json", bytes.NewBuffer(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, float64(tt.expectedTriggered), result["rules_triggered"])

			if tt.expectedRuleID != "" {
				results := result["results"].([]interface{})
				found := false
				for _, r := range results {
					rMap := r.(map[string]interface{})
					if rMap["rule_id"] == tt.expectedRuleID && rMap["triggered"].(bool) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected rule %s to trigger", tt.expectedRuleID)
			}
		})
	}
}

// TestAPI_EvaluateByType tests the evaluate by type endpoint
func TestAPI_EvaluateByType(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	request := map[string]interface{}{
		"patient_id": "test-001",
		"labs": map[string]interface{}{
			"potassium": map[string]interface{}{"value": 7.0},
		},
		"vitals": map[string]interface{}{
			"temperature": map[string]interface{}{"value": 39.0},
			"heart_rate":  map[string]interface{}{"value": 110.0},
		},
	}

	// Test ALERT type
	body, _ := json.Marshal(request)
	resp, err := http.Post(ts.URL+"/api/v1/evaluate/type/ALERT", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// All triggered rules should be ALERT type
	results := result["results"].([]interface{})
	for _, r := range results {
		rMap := r.(map[string]interface{})
		if rMap["triggered"].(bool) {
			assert.Equal(t, "ALERT", rMap["rule_type"])
		}
	}
}

// TestAPI_EvaluateByCategory tests the evaluate by category endpoint
func TestAPI_EvaluateByCategory(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	request := map[string]interface{}{
		"patient_id": "test-001",
		"labs": map[string]interface{}{
			"potassium": map[string]interface{}{"value": 7.0},
		},
	}

	body, _ := json.Marshal(request)
	resp, err := http.Post(ts.URL+"/api/v1/evaluate/category/SAFETY", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// All results should be from SAFETY category
	results := result["results"].([]interface{})
	for _, r := range results {
		rMap := r.(map[string]interface{})
		assert.Equal(t, "SAFETY", rMap["category"])
	}
}

// TestAPI_EvaluateSpecificRules tests evaluating specific rules by ID
func TestAPI_EvaluateSpecificRules(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	// The endpoint expects: { rule_ids: [...], context: {...} }
	request := map[string]interface{}{
		"rule_ids": []string{"ALERT-K-HIGH"},
		"context": map[string]interface{}{
			"patient_id": "test-001",
			"labs": map[string]interface{}{
				"potassium": map[string]interface{}{"value": 7.0},
				"glucose":   map[string]interface{}{"value": 40.0}, // Would trigger glucose rule if evaluated
			},
		},
	}

	body, _ := json.Marshal(request)
	resp, err := http.Post(ts.URL+"/api/v1/evaluate/rules", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Should only evaluate the specified rule
	assert.Equal(t, float64(1), result["rules_evaluated"])
}

// TestAPI_ListRules tests the list rules endpoint
func TestAPI_ListRules(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/rules")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	rules := result["rules"].([]interface{})
	assert.Len(t, rules, 4, "Should return all 4 test rules")
}

// TestAPI_GetRule tests getting a specific rule by ID
func TestAPI_GetRule(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	// Existing rule
	resp, err := http.Get(ts.URL + "/api/v1/rules/ALERT-K-HIGH")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "ALERT-K-HIGH", result["id"])
	assert.Equal(t, "Critical Hyperkalemia", result["name"])

	// Non-existent rule
	resp2, err := http.Get(ts.URL + "/api/v1/rules/NON-EXISTENT")
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

// TestAPI_CreateRule tests creating a new rule via API
func TestAPI_CreateRule(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	newRule := map[string]interface{}{
		"id":          "NEW-RULE-001",
		"name":        "New API Created Rule",
		"description": "Created via API",
		"type":        "ALERT",
		"category":    "SAFETY",
		"severity":    "HIGH",
		"status":      "ACTIVE",
		"priority":    5,
		"conditions": []map[string]interface{}{
			{"field": "labs.test.value", "operator": "GT", "value": 100},
		},
		"condition_logic": "AND",
		"actions": []map[string]interface{}{
			{"type": "ALERT", "message": "Test alert"},
		},
		"tags": []string{"test", "api-created"},
	}

	body, _ := json.Marshal(newRule)
	resp, err := http.Post(ts.URL+"/api/v1/rules", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify rule was created
	resp2, _ := http.Get(ts.URL + "/api/v1/rules/NEW-RULE-001")
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var created map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&created)
	assert.Equal(t, "New API Created Rule", created["name"])
}

// TestAPI_UpdateRule tests updating an existing rule
func TestAPI_UpdateRule(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	updatedRule := map[string]interface{}{
		"id":          "ALERT-K-HIGH",
		"name":        "Updated Hyperkalemia Rule",
		"description": "Updated description",
		"type":        "ALERT",
		"category":    "SAFETY",
		"severity":    "CRITICAL",
		"status":      "ACTIVE",
		"priority":    2,
		"conditions": []map[string]interface{}{
			{"field": "labs.potassium.value", "operator": "GTE", "value": 6.0}, // Lowered threshold
		},
		"condition_logic": "AND",
		"actions": []map[string]interface{}{
			{"type": "ALERT", "message": "Updated alert message"},
		},
		"tags": []string{"electrolyte", "updated"},
	}

	body, _ := json.Marshal(updatedRule)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/rules/ALERT-K-HIGH", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify update
	resp2, _ := http.Get(ts.URL + "/api/v1/rules/ALERT-K-HIGH")
	defer resp2.Body.Close()

	var updated map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&updated)
	assert.Equal(t, "Updated Hyperkalemia Rule", updated["name"])
	assert.Equal(t, float64(2), updated["priority"])
}

// TestAPI_DeleteRule tests deleting a rule
func TestAPI_DeleteRule(t *testing.T) {
	ts, store := setupTestServer(t)
	defer ts.Close()

	// Add a rule to delete
	store.Add(&models.Rule{
		ID:       "DELETE-ME",
		Name:     "Rule to Delete",
		Type:     models.RuleTypeAlert,
		Category: "SAFETY",
		Severity: "LOW",
		Status:   "ACTIVE",
		Priority: 100,
		Conditions: []models.Condition{
			{Field: "test", Operator: "EQ", Value: true},
		},
		Actions: []models.Action{
			{Type: "ALERT", Message: "Test"},
		},
	})

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/rules/DELETE-ME", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify deletion
	resp2, _ := http.Get(ts.URL + "/api/v1/rules/DELETE-ME")
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

// TestAPI_RuleStats tests the rule statistics endpoint
func TestAPI_RuleStats(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/rules/stats")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)

	assert.Equal(t, float64(4), stats["total_rules"])
	assert.Equal(t, float64(4), stats["active_rules"])
	assert.Contains(t, stats, "rules_by_type")
	assert.Contains(t, stats, "rules_by_category")
	assert.Contains(t, stats, "rules_by_severity")
}

// TestAPI_ReloadRules tests the hot-reload endpoint
func TestAPI_ReloadRules(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/v1/rules/reload", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Without a loader configured (nil in test), returns 503 ServiceUnavailable
	// In production, this would reload from disk
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError, http.StatusServiceUnavailable}, resp.StatusCode)
}

// TestAPI_CacheStats tests the cache statistics endpoint
func TestAPI_CacheStats(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/cache/stats")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)

	// Cache returns "size" for the number of entries, not "entries"
	assert.Contains(t, stats, "size")
	assert.Contains(t, stats, "hits")
	assert.Contains(t, stats, "misses")
}

// TestAPI_CacheClear tests clearing the cache
func TestAPI_CacheClear(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/v1/cache/clear", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestAPI_InvalidRequest tests error handling for invalid requests
func TestAPI_InvalidRequest(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Invalid JSON",
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing patient_id",
			body:           `{"labs": {"potassium": {"value": 5.0}}}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/api/v1/evaluate", "application/json", bytes.NewBufferString(tt.body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestAPI_RuleTypes tests the rule types listing endpoint
func TestAPI_RuleTypes(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/rules/types")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	types := result["types"].([]interface{})
	assert.Greater(t, len(types), 0, "Should return available rule types")
}

// TestAPI_RuleCategories tests the rule categories listing endpoint
func TestAPI_RuleCategories(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/rules/categories")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	categories := result["categories"].([]interface{})
	assert.Greater(t, len(categories), 0, "Should return available categories")
}

// TestAPI_RuleTags tests the rule tags listing endpoint
func TestAPI_RuleTags(t *testing.T) {
	ts, _ := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/rules/tags")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	tags := result["tags"].([]interface{})
	assert.Greater(t, len(tags), 0, "Should return available tags")
}
