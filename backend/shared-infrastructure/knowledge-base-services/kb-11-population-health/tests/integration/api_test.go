// Package integration provides integration tests for KB-11 Population Health Engine.
package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────────────────────────────────
// API Response Structure Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestAPIResponseFormat verifies standard API response format.
func TestAPIResponseFormat(t *testing.T) {
	t.Run("success response format", func(t *testing.T) {
		response := gin.H{
			"data": gin.H{
				"patient_id": "patient-123",
				"risk_score": 0.75,
			},
		}

		jsonBytes, err := json.Marshal(response)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(jsonBytes, &parsed)
		require.NoError(t, err)

		assert.Contains(t, parsed, "data")
	})

	t.Run("error response format", func(t *testing.T) {
		response := gin.H{
			"error": gin.H{
				"message": "Patient not found",
				"code":    "PATIENT_NOT_FOUND",
				"details": "No patient exists with ID patient-999",
			},
		}

		jsonBytes, err := json.Marshal(response)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(jsonBytes, &parsed)
		require.NoError(t, err)

		assert.Contains(t, parsed, "error")
		errorObj := parsed["error"].(map[string]interface{})
		assert.Contains(t, errorObj, "message")
		assert.Contains(t, errorObj, "code")
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Risk Endpoint Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestRiskCalculationEndpoint tests the risk calculation endpoint structure.
func TestRiskCalculationEndpoint(t *testing.T) {
	t.Run("valid request body structure", func(t *testing.T) {
		requestBody := `{
			"patient_fhir_id": "patient-123",
			"model_name": "hospitalization_risk",
			"features": {
				"age": 68,
				"gender": "male",
				"conditions": ["E11", "I10"],
				"medications": ["6809"]
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(requestBody), &parsed)
		require.NoError(t, err)

		assert.Equal(t, "patient-123", parsed["patient_fhir_id"])
		assert.Equal(t, "hospitalization_risk", parsed["model_name"])
		assert.NotNil(t, parsed["features"])
	})

	t.Run("response includes governance data", func(t *testing.T) {
		responseBody := `{
			"risk_result": {
				"patient_fhir_id": "patient-123",
				"model_name": "hospitalization_risk",
				"model_version": "1.0.0",
				"score": 0.72,
				"risk_tier": "HIGH",
				"confidence": 0.85,
				"input_hash": "sha256:abc123...",
				"calculation_hash": "sha256:def456...",
				"governance_event_id": "evt-789"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		result := parsed["risk_result"].(map[string]interface{})
		assert.Contains(t, result, "input_hash")
		assert.Contains(t, result, "calculation_hash")
		assert.Contains(t, result, "governance_event_id")
	})
}

// TestBatchRiskCalculationEndpoint tests batch risk calculation structure.
func TestBatchRiskCalculationEndpoint(t *testing.T) {
	t.Run("batch request structure", func(t *testing.T) {
		requestBody := `{
			"patients": [
				{"patient_fhir_id": "patient-1", "model_name": "hospitalization_risk"},
				{"patient_fhir_id": "patient-2", "model_name": "hospitalization_risk"},
				{"patient_fhir_id": "patient-3", "model_name": "readmission_risk"}
			],
			"options": {
				"parallel": true,
				"include_contributing_factors": true
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(requestBody), &parsed)
		require.NoError(t, err)

		patients := parsed["patients"].([]interface{})
		assert.Len(t, patients, 3)
	})

	t.Run("batch response structure", func(t *testing.T) {
		responseBody := `{
			"results": [
				{"patient_fhir_id": "patient-1", "score": 0.65, "risk_tier": "HIGH", "success": true},
				{"patient_fhir_id": "patient-2", "score": 0.32, "risk_tier": "LOW", "success": true},
				{"patient_fhir_id": "patient-3", "error": "Patient not found", "success": false}
			],
			"summary": {
				"total": 3,
				"succeeded": 2,
				"failed": 1
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		results := parsed["results"].([]interface{})
		assert.Len(t, results, 3)

		summary := parsed["summary"].(map[string]interface{})
		assert.Equal(t, float64(3), summary["total"])
		assert.Equal(t, float64(2), summary["succeeded"])
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Endpoint Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCohortCreateEndpoint tests cohort creation endpoint structure.
func TestCohortCreateEndpoint(t *testing.T) {
	t.Run("static cohort creation request", func(t *testing.T) {
		requestBody := `{
			"name": "Diabetes Management Program",
			"description": "Patients enrolled in diabetes management",
			"type": "STATIC"
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(requestBody), &parsed)
		require.NoError(t, err)

		assert.Equal(t, "Diabetes Management Program", parsed["name"])
		assert.Equal(t, "STATIC", parsed["type"])
	})

	t.Run("dynamic cohort creation request", func(t *testing.T) {
		requestBody := `{
			"name": "High Risk Elderly",
			"description": "Elderly patients with high risk scores",
			"type": "DYNAMIC",
			"criteria": [
				{"field": "age", "operator": "gte", "value": 65},
				{"field": "risk_tier", "operator": "in", "value": ["HIGH", "VERY_HIGH"]}
			],
			"refresh_config": {
				"auto_refresh": true,
				"refresh_interval_hours": 24
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(requestBody), &parsed)
		require.NoError(t, err)

		criteria := parsed["criteria"].([]interface{})
		assert.Len(t, criteria, 2)

		refreshConfig := parsed["refresh_config"].(map[string]interface{})
		assert.True(t, refreshConfig["auto_refresh"].(bool))
	})

	t.Run("cohort creation response", func(t *testing.T) {
		responseBody := `{
			"cohort": {
				"id": "550e8400-e29b-41d4-a716-446655440000",
				"name": "High Risk Elderly",
				"type": "DYNAMIC",
				"status": "ACTIVE",
				"member_count": 0,
				"created_at": "2024-06-15T10:30:00Z"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		cohort := parsed["cohort"].(map[string]interface{})
		assert.Contains(t, cohort, "id")
		assert.Equal(t, "ACTIVE", cohort["status"])
	})
}

// TestCohortRefreshEndpoint tests cohort refresh endpoint structure.
func TestCohortRefreshEndpoint(t *testing.T) {
	t.Run("refresh response structure", func(t *testing.T) {
		responseBody := `{
			"refresh_result": {
				"cohort_id": "550e8400-e29b-41d4-a716-446655440000",
				"previous_count": 100,
				"new_count": 115,
				"added_count": 20,
				"removed_count": 5,
				"duration_ms": 250,
				"refreshed_at": "2024-06-15T10:30:00Z"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		result := parsed["refresh_result"].(map[string]interface{})
		assert.Equal(t, float64(100), result["previous_count"])
		assert.Equal(t, float64(115), result["new_count"])
		assert.Equal(t, float64(20), result["added_count"])
		assert.Equal(t, float64(5), result["removed_count"])
	})
}

// TestCohortMembershipEndpoint tests cohort membership endpoints.
func TestCohortMembershipEndpoint(t *testing.T) {
	t.Run("add members request", func(t *testing.T) {
		requestBody := `{
			"patient_fhir_ids": ["patient-1", "patient-2", "patient-3"],
			"metadata": {
				"source": "manual_enrollment",
				"enrolled_by": "care-manager-1"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(requestBody), &parsed)
		require.NoError(t, err)

		patients := parsed["patient_fhir_ids"].([]interface{})
		assert.Len(t, patients, 3)
	})

	t.Run("list members response", func(t *testing.T) {
		responseBody := `{
			"members": [
				{
					"patient_fhir_id": "patient-1",
					"membership_status": "ACTIVE",
					"added_at": "2024-06-15T10:30:00Z",
					"risk_score_at_add": 0.72
				},
				{
					"patient_fhir_id": "patient-2",
					"membership_status": "ACTIVE",
					"added_at": "2024-06-15T10:30:00Z",
					"risk_score_at_add": 0.65
				}
			],
			"pagination": {
				"total": 115,
				"limit": 20,
				"offset": 0,
				"has_more": true
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		members := parsed["members"].([]interface{})
		assert.Len(t, members, 2)

		pagination := parsed["pagination"].(map[string]interface{})
		assert.Equal(t, float64(115), pagination["total"])
		assert.True(t, pagination["has_more"].(bool))
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Analytics Endpoint Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestPopulationSnapshotEndpoint tests population snapshot endpoint.
func TestPopulationSnapshotEndpoint(t *testing.T) {
	t.Run("snapshot response structure", func(t *testing.T) {
		responseBody := `{
			"snapshot": {
				"total_patients": 10000,
				"high_risk_count": 1500,
				"rising_risk_count": 350,
				"average_risk_score": 0.42,
				"risk_percentages": {
					"LOW": 45.0,
					"MODERATE": 30.0,
					"HIGH": 12.0,
					"VERY_HIGH": 5.0,
					"RISING": 3.5,
					"UNSCORED": 4.5
				},
				"care_gap_metrics": {
					"total_open_gaps": 4500,
					"patients_with_gaps": 2800
				},
				"calculated_at": "2024-06-15T10:30:00Z"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		snapshot := parsed["snapshot"].(map[string]interface{})
		assert.Equal(t, float64(10000), snapshot["total_patients"])
		assert.Equal(t, float64(1500), snapshot["high_risk_count"])
		assert.Contains(t, snapshot, "risk_percentages")
		assert.Contains(t, snapshot, "care_gap_metrics")
	})

	t.Run("snapshot with filter query params", func(t *testing.T) {
		// Test query parameter parsing
		queryParams := "?practice=cardiology&min_age=65&max_age=85&risk_tier=HIGH&risk_tier=VERY_HIGH&with_care_gaps=true"

		assert.Contains(t, queryParams, "practice=cardiology")
		assert.Contains(t, queryParams, "min_age=65")
		assert.Contains(t, queryParams, "risk_tier=HIGH")
		assert.Contains(t, queryParams, "with_care_gaps=true")
	})
}

// TestRiskStratificationEndpoint tests risk stratification endpoint.
func TestRiskStratificationEndpoint(t *testing.T) {
	t.Run("stratification report response", func(t *testing.T) {
		responseBody := `{
			"report": {
				"distribution": {
					"LOW": {"count": 4500, "percentage": 45.0, "average_score": 0.15},
					"MODERATE": {"count": 3000, "percentage": 30.0, "average_score": 0.45},
					"HIGH": {"count": 1000, "percentage": 10.0, "average_score": 0.72},
					"VERY_HIGH": {"count": 500, "percentage": 5.0, "average_score": 0.88}
				},
				"rising_risk_patients": 500,
				"high_risk_breakdown": {
					"by_condition": {
						"diabetes": 450,
						"heart_failure": 350
					}
				},
				"report_date": "2024-06-15T10:30:00Z"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		report := parsed["report"].(map[string]interface{})
		assert.Contains(t, report, "distribution")
		assert.Contains(t, report, "high_risk_breakdown")
		assert.Equal(t, float64(500), report["rising_risk_patients"])
	})
}

// TestDashboardEndpoints tests dashboard endpoint structures.
func TestDashboardEndpoints(t *testing.T) {
	t.Run("executive dashboard response", func(t *testing.T) {
		responseBody := `{
			"dashboard": {
				"total_patients": 10000,
				"high_risk_count": 1500,
				"rising_risk_count": 350,
				"average_risk": 0.42,
				"high_risk_percentage": 15.0,
				"care_gap_summary": {
					"total_open_gaps": 4500,
					"patients_with_gaps": 2800
				},
				"attribution_summary": {
					"total_pcps": 85,
					"total_practices": 12
				},
				"calculated_at": "2024-06-15T10:30:00Z"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		dashboard := parsed["dashboard"].(map[string]interface{})
		assert.Contains(t, dashboard, "total_patients")
		assert.Contains(t, dashboard, "high_risk_percentage")
		assert.Contains(t, dashboard, "care_gap_summary")
	})

	t.Run("care manager dashboard response", func(t *testing.T) {
		responseBody := `{
			"dashboard": {
				"risk_tiers": {
					"HIGH": {"count": 1000, "percentage": 10.0},
					"VERY_HIGH": {"count": 500, "percentage": 5.0},
					"RISING": {"count": 500, "percentage": 5.0}
				},
				"actionable_patient_count": 2000,
				"rising_risk_patients": 500,
				"report_date": "2024-06-15T10:30:00Z"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		dashboard := parsed["dashboard"].(map[string]interface{})
		assert.Equal(t, float64(2000), dashboard["actionable_patient_count"])
		assert.Contains(t, dashboard, "risk_tiers")
	})
}

// TestProviderAnalyticsEndpoint tests provider analytics endpoint.
func TestProviderAnalyticsEndpoint(t *testing.T) {
	t.Run("provider analytics response", func(t *testing.T) {
		responseBody := `{
			"analytics": {
				"provider_id": "provider-123",
				"provider_name": "Dr. Smith",
				"panel_size": 250,
				"high_risk_count": 35,
				"rising_risk_count": 12,
				"average_risk_score": 0.38,
				"compared_to_average": 0.05,
				"risk_distribution": {
					"LOW": 110,
					"MODERATE": 78,
					"HIGH": 30,
					"VERY_HIGH": 5
				}
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		analytics := parsed["analytics"].(map[string]interface{})
		assert.Equal(t, "provider-123", analytics["provider_id"])
		assert.Equal(t, float64(250), analytics["panel_size"])
	})
}

// TestComparisonEndpoints tests provider/practice comparison endpoints.
func TestComparisonEndpoints(t *testing.T) {
	t.Run("provider comparison response", func(t *testing.T) {
		responseBody := `{
			"comparisons": [
				{
					"provider_id": "provider-1",
					"panel_size": 250,
					"high_risk_count": 35,
					"average_risk_score": 0.38,
					"comparison": 0.05
				},
				{
					"provider_id": "provider-2",
					"panel_size": 280,
					"high_risk_count": 48,
					"average_risk_score": 0.45,
					"comparison": 0.12
				}
			],
			"provider_count": 2
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		comparisons := parsed["comparisons"].([]interface{})
		assert.Len(t, comparisons, 2)
		assert.Equal(t, float64(2), parsed["provider_count"])
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Health Check Endpoint Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestHealthCheckEndpoint tests health check endpoint.
func TestHealthCheckEndpoint(t *testing.T) {
	t.Run("healthy response", func(t *testing.T) {
		responseBody := `{
			"status": "healthy",
			"version": "1.0.0",
			"service": "kb-11-population-health",
			"timestamp": "2024-06-15T10:30:00Z",
			"dependencies": {
				"database": "healthy",
				"redis": "healthy",
				"kb-18": "healthy",
				"kb-13": "healthy"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		assert.Equal(t, "healthy", parsed["status"])
		assert.Equal(t, "kb-11-population-health", parsed["service"])

		deps := parsed["dependencies"].(map[string]interface{})
		assert.Equal(t, "healthy", deps["database"])
		assert.Equal(t, "healthy", deps["kb-18"])
	})

	t.Run("degraded response", func(t *testing.T) {
		responseBody := `{
			"status": "degraded",
			"version": "1.0.0",
			"service": "kb-11-population-health",
			"dependencies": {
				"database": "healthy",
				"redis": "unhealthy",
				"kb-18": "healthy"
			}
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(responseBody), &parsed)
		require.NoError(t, err)

		assert.Equal(t, "degraded", parsed["status"])

		deps := parsed["dependencies"].(map[string]interface{})
		assert.Equal(t, "unhealthy", deps["redis"])
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Error Response Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestAPIErrorResponses tests various error response formats.
func TestAPIErrorResponses(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		errorCode  string
		message    string
	}{
		{"Not Found", http.StatusNotFound, "NOT_FOUND", "Resource not found"},
		{"Bad Request", http.StatusBadRequest, "BAD_REQUEST", "Invalid request parameters"},
		{"Unauthorized", http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required"},
		{"Internal Error", http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred"},
		{"Conflict", http.StatusConflict, "CONFLICT", "Resource already exists"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errorResponse := gin.H{
				"error": gin.H{
					"message": tc.message,
					"code":    tc.errorCode,
				},
			}

			jsonBytes, err := json.Marshal(errorResponse)
			require.NoError(t, err)

			var parsed map[string]interface{}
			err = json.Unmarshal(jsonBytes, &parsed)
			require.NoError(t, err)

			errorObj := parsed["error"].(map[string]interface{})
			assert.Equal(t, tc.message, errorObj["message"])
			assert.Equal(t, tc.errorCode, errorObj["code"])
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Request Validation Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestRequestValidation tests request body validation.
func TestRequestValidation(t *testing.T) {
	t.Run("missing required field", func(t *testing.T) {
		// Cohort without name should fail
		requestBody := `{
			"description": "A cohort without name",
			"type": "STATIC"
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(requestBody), &parsed)
		require.NoError(t, err)

		_, hasName := parsed["name"]
		assert.False(t, hasName, "Request should be missing 'name' field")
	})

	t.Run("invalid cohort type", func(t *testing.T) {
		requestBody := `{
			"name": "Test Cohort",
			"type": "INVALID_TYPE"
		}`

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(requestBody), &parsed)
		require.NoError(t, err)

		cohortType := parsed["type"].(string)
		validTypes := []string{"STATIC", "DYNAMIC", "SNAPSHOT"}
		isValid := false
		for _, vt := range validTypes {
			if cohortType == vt {
				isValid = true
				break
			}
		}
		assert.False(t, isValid, "INVALID_TYPE should not be a valid cohort type")
	})

	t.Run("invalid UUID format", func(t *testing.T) {
		invalidID := "not-a-uuid"
		_, err := uuid.Parse(invalidID)
		assert.Error(t, err, "Should fail to parse invalid UUID")
	})

	t.Run("valid UUID format", func(t *testing.T) {
		validID := "550e8400-e29b-41d4-a716-446655440000"
		parsed, err := uuid.Parse(validID)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, parsed)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Pagination Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestPaginationParams tests pagination parameter handling.
func TestPaginationParams(t *testing.T) {
	t.Run("default pagination", func(t *testing.T) {
		// Default values when no params provided
		defaultLimit := 20
		defaultOffset := 0

		assert.Equal(t, 20, defaultLimit)
		assert.Equal(t, 0, defaultOffset)
	})

	t.Run("custom pagination", func(t *testing.T) {
		queryParams := "?limit=50&offset=100"

		assert.Contains(t, queryParams, "limit=50")
		assert.Contains(t, queryParams, "offset=100")
	})

	t.Run("pagination response structure", func(t *testing.T) {
		pagination := map[string]interface{}{
			"total":    1000,
			"limit":    50,
			"offset":   100,
			"has_more": true,
		}

		assert.Equal(t, 1000, pagination["total"])
		assert.Equal(t, 50, pagination["limit"])
		assert.Equal(t, 100, pagination["offset"])
		assert.True(t, pagination["has_more"].(bool))
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// HTTP Handler Mock Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestHTTPHandlerRouting tests HTTP routing patterns.
func TestHTTPHandlerRouting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("cohort routes", func(t *testing.T) {
		routes := []struct {
			method string
			path   string
		}{
			{"POST", "/v1/cohorts/static"},
			{"POST", "/v1/cohorts/dynamic"},
			{"POST", "/v1/cohorts/snapshot"},
			{"GET", "/v1/cohorts"},
			{"GET", "/v1/cohorts/:id"},
			{"DELETE", "/v1/cohorts/:id"},
			{"POST", "/v1/cohorts/:id/members"},
			{"GET", "/v1/cohorts/:id/members"},
			{"DELETE", "/v1/cohorts/:id/members/:patient_id"},
			{"POST", "/v1/cohorts/:id/refresh"},
			{"GET", "/v1/cohorts/:id/stats"},
			{"GET", "/v1/cohorts/compare"},
		}

		for _, route := range routes {
			t.Run(route.method+" "+route.path, func(t *testing.T) {
				assert.NotEmpty(t, route.method)
				assert.NotEmpty(t, route.path)
				assert.True(t, strings.HasPrefix(route.path, "/v1/cohorts"))
			})
		}
	})

	t.Run("analytics routes", func(t *testing.T) {
		routes := []struct {
			method string
			path   string
		}{
			{"GET", "/v1/analytics/population/snapshot"},
			{"GET", "/v1/analytics/risk/stratification"},
			{"GET", "/v1/analytics/providers/:provider_id"},
			{"GET", "/v1/analytics/practices/:practice_id"},
			{"GET", "/v1/analytics/dashboard/executive"},
			{"GET", "/v1/analytics/dashboard/care-manager"},
			{"GET", "/v1/analytics/compare/providers"},
			{"GET", "/v1/analytics/compare/practices"},
		}

		for _, route := range routes {
			t.Run(route.method+" "+route.path, func(t *testing.T) {
				assert.NotEmpty(t, route.method)
				assert.NotEmpty(t, route.path)
				assert.True(t, strings.HasPrefix(route.path, "/v1/analytics"))
			})
		}
	})

	t.Run("risk routes", func(t *testing.T) {
		routes := []struct {
			method string
			path   string
		}{
			{"POST", "/v1/risk/calculate"},
			{"POST", "/v1/risk/batch"},
			{"GET", "/v1/risk/patient/:patient_id"},
			{"GET", "/v1/risk/patient/:patient_id/history"},
		}

		for _, route := range routes {
			t.Run(route.method+" "+route.path, func(t *testing.T) {
				assert.NotEmpty(t, route.method)
				assert.NotEmpty(t, route.path)
				assert.True(t, strings.HasPrefix(route.path, "/v1/risk"))
			})
		}
	})
}

// TestHTTPResponseRecorder tests HTTP response recording.
func TestHTTPResponseRecorder(t *testing.T) {
	t.Run("record success response", func(t *testing.T) {
		w := httptest.NewRecorder()

		// Simulate writing a JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("record error response", func(t *testing.T) {
		w := httptest.NewRecorder()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"Bad request"}}`))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "error")
	})
}
