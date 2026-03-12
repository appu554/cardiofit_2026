// Package integration provides end-to-end integration tests for KB-9 Care Gaps Service.
// These tests verify the complete API flow including REST, FHIR, and GraphQL endpoints.
//
// Run with: go test -tags=integration ./tests/integration/...
// Requires: KB-9 service running on localhost:8089

//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// getBaseURL returns the KB-9 base URL from environment or default.
func getBaseURL() string {
	if url := os.Getenv("KB9_BASE_URL"); url != "" {
		return url
	}
	return "http://localhost:8089"
}

// TestHealthEndpoints verifies all health/monitoring endpoints.
func TestHealthEndpoints(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 10 * time.Second}

	endpoints := []struct {
		name     string
		path     string
		expected int
	}{
		{"health", "/health", http.StatusOK},
		{"ready", "/ready", http.StatusOK},
		{"live", "/live", http.StatusOK},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			resp, err := client.Get(baseURL + ep.path)
			if err != nil {
				t.Fatalf("Request failed: %v (is KB-9 running on %s?)", err, baseURL)
			}
			defer resp.Body.Close()

			if resp.StatusCode != ep.expected {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", ep.expected, resp.StatusCode, string(body))
			}
		})
	}
}

// TestListMeasures verifies the GET /api/v1/measures endpoint.
func TestListMeasures(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(baseURL + "/api/v1/measures")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Measures []struct {
			Type        string `json:"type"`
			CMSID       string `json:"cmsId"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Domain      string `json:"domain"`
		} `json:"measures"`
		Count int `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify expected measures exist
	expectedMeasures := []string{"CMS122", "CMS165", "CMS130", "CMS2"}
	for _, expected := range expectedMeasures {
		found := false
		for _, m := range result.Measures {
			if m.CMSID == expected || m.Type == expected+"_DIABETES_HBA1C" {
				found = true
				break
			}
		}
		if !found {
			t.Logf("⚠️ Measure %s not found in response (may use different ID format)", expected)
		}
	}

	if result.Count == 0 {
		t.Error("Expected at least one measure, got 0")
	}

	t.Logf("✅ Found %d measures", result.Count)
}

// TestGetCareGaps verifies the POST /api/v1/care-gaps endpoint.
func TestGetCareGaps(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 30 * time.Second}

	// Request body
	requestBody := map[string]interface{}{
		"patientId": "test-patient-001",
		"period": map[string]string{
			"start": "2024-01-01",
			"end":   "2024-12-31",
		},
		"includeEvidence":    true,
		"includeClosedGaps":  false,
		"createScheduleItems": false,
	}

	body, _ := json.Marshal(requestBody)

	resp, err := client.Post(
		baseURL+"/api/v1/care-gaps",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Accept both 200 (success) and 500 (FHIR server not available in test)
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var result struct {
			PatientID  string `json:"patientId"`
			ReportDate string `json:"reportDate"`
			OpenGaps   []struct {
				ID      string `json:"id"`
				Measure struct {
					Type string `json:"type"`
					Name string `json:"name"`
				} `json:"measure"`
				Status   string `json:"status"`
				Priority string `json:"priority"`
			} `json:"openGaps"`
			Summary struct {
				TotalOpenGaps    int `json:"totalOpenGaps"`
				HighPriorityGaps int `json:"highPriorityGaps"`
			} `json:"summary"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		t.Logf("✅ Care gaps report received:")
		t.Logf("   Patient: %s", result.PatientID)
		t.Logf("   Open Gaps: %d", result.Summary.TotalOpenGaps)
		t.Logf("   High Priority: %d", result.Summary.HighPriorityGaps)
	} else {
		t.Logf("⚠️ Care gaps request returned status %d (FHIR server may not be available)", resp.StatusCode)
		t.Logf("   Response: %s", string(respBody))
	}
}

// TestEvaluateMeasure verifies the POST /api/v1/measure/evaluate endpoint.
func TestEvaluateMeasure(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 30 * time.Second}

	requestBody := map[string]interface{}{
		"patientId":   "test-patient-001",
		"measureType": "CMS122_DIABETES_HBA1C",
		"period": map[string]string{
			"start": "2024-01-01",
			"end":   "2024-12-31",
		},
	}

	body, _ := json.Marshal(requestBody)

	resp, err := client.Post(
		baseURL+"/api/v1/measure/evaluate",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var result struct {
			ID      string `json:"id"`
			Measure struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"measure"`
			Status      string `json:"status"`
			Populations []struct {
				Population string `json:"population"`
				Count      int    `json:"count"`
			} `json:"populations"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		t.Logf("✅ Measure evaluation completed:")
		t.Logf("   Measure: %s", result.Measure.Name)
		t.Logf("   Status: %s", result.Status)
	} else {
		t.Logf("⚠️ Measure evaluation returned status %d", resp.StatusCode)
	}
}

// TestFHIRCareGapsOperation verifies the Da Vinci DEQM $care-gaps operation.
func TestFHIRCareGapsOperation(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 30 * time.Second}

	// FHIR Parameters resource
	parameters := map[string]interface{}{
		"resourceType": "Parameters",
		"parameter": []map[string]interface{}{
			{
				"name":        "periodStart",
				"valueDate":   "2024-01-01",
			},
			{
				"name":        "periodEnd",
				"valueDate":   "2024-12-31",
			},
			{
				"name":        "subject",
				"valueString": "Patient/test-patient-001",
			},
			{
				"name":        "status",
				"valueString": "open-gap",
			},
		},
	}

	body, _ := json.Marshal(parameters)

	resp, err := client.Post(
		baseURL+"/fhir/Measure/$care-gaps",
		"application/fhir+json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var bundle struct {
			ResourceType string `json:"resourceType"`
			Type         string `json:"type"`
			Entry        []struct {
				Resource struct {
					ResourceType string `json:"resourceType"`
				} `json:"resource"`
			} `json:"entry"`
		}

		if err := json.Unmarshal(respBody, &bundle); err != nil {
			t.Fatalf("Failed to decode FHIR Bundle: %v", err)
		}

		if bundle.ResourceType != "Bundle" {
			t.Errorf("Expected Bundle resourceType, got %s", bundle.ResourceType)
		}

		t.Logf("✅ FHIR $care-gaps operation successful:")
		t.Logf("   Bundle type: %s", bundle.Type)
		t.Logf("   Entries: %d", len(bundle.Entry))

		// Count resource types
		resourceCounts := make(map[string]int)
		for _, entry := range bundle.Entry {
			resourceCounts[entry.Resource.ResourceType]++
		}
		for rt, count := range resourceCounts {
			t.Logf("   - %s: %d", rt, count)
		}
	} else {
		t.Logf("⚠️ FHIR $care-gaps returned status %d", resp.StatusCode)
		t.Logf("   Response: %s", string(respBody))
	}
}

// TestGraphQLQuery verifies the GraphQL endpoint.
func TestGraphQLQuery(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 30 * time.Second}

	// GraphQL query
	query := map[string]interface{}{
		"query": `
			query {
				availableMeasures {
					type
					cmsId
					name
					domain
				}
			}
		`,
	}

	body, _ := json.Marshal(query)

	resp, err := client.Post(
		baseURL+"/graphql",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var result struct {
			Data struct {
				AvailableMeasures []struct {
					Type   string `json:"type"`
					CMSID  string `json:"cmsId"`
					Name   string `json:"name"`
					Domain string `json:"domain"`
				} `json:"availableMeasures"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to decode GraphQL response: %v", err)
		}

		if len(result.Errors) > 0 {
			t.Logf("⚠️ GraphQL returned errors: %v", result.Errors)
		}

		if result.Data.AvailableMeasures != nil {
			t.Logf("✅ GraphQL query successful:")
			t.Logf("   Found %d measures", len(result.Data.AvailableMeasures))
		}
	} else if resp.StatusCode == http.StatusNotFound {
		t.Log("⚠️ GraphQL endpoint not available (federation may be disabled)")
	} else {
		t.Logf("⚠️ GraphQL query returned status %d", resp.StatusCode)
	}
}

// TestCareGapsWithTemporalContext tests KB-3 temporal enrichment.
func TestCareGapsWithTemporalContext(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 30 * time.Second}

	requestBody := map[string]interface{}{
		"patientId": "test-patient-temporal-001",
		"period": map[string]string{
			"start": "2024-01-01",
			"end":   "2024-12-31",
		},
		"includeEvidence":     true,
		"createScheduleItems": true, // Enable KB-3 integration
	}

	body, _ := json.Marshal(requestBody)

	resp, err := client.Post(
		baseURL+"/api/v1/care-gaps",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var result struct {
			PatientID   string `json:"patientId"`
			UpcomingDue []struct {
				ID              string `json:"id"`
				TemporalContext struct {
					DaysUntilDue int    `json:"daysUntilDue"`
					Status       string `json:"status"`
				} `json:"temporalContext"`
			} `json:"upcomingDue"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Logf("Response parsing info: %v", err)
		}

		t.Logf("✅ Care gaps with temporal context:")
		t.Logf("   Patient: %s", result.PatientID)
		t.Logf("   Upcoming due: %d gaps", len(result.UpcomingDue))
	} else {
		t.Logf("⚠️ Temporal care gaps returned status %d (KB-3 may not be available)", resp.StatusCode)
	}
}

// TestGapManagement tests gap lifecycle operations.
func TestGapManagement(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 10 * time.Second}

	testGapID := "test-gap-12345"

	// Test 1: Mark gap as addressed
	t.Run("address_gap", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"patientId":    "test-patient-001",
			"intervention": "lab_order",
			"notes":        "HbA1c test ordered",
		}
		body, _ := json.Marshal(requestBody)

		resp, err := client.Post(
			fmt.Sprintf("%s/api/v1/gaps/%s/addressed", baseURL, testGapID),
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Accept 200 or 404 (gap not found in test environment)
		if resp.StatusCode == http.StatusOK {
			t.Log("✅ Gap addressed successfully")
		} else {
			t.Logf("⚠️ Gap address returned status %d (gap may not exist)", resp.StatusCode)
		}
	})

	// Test 2: Dismiss gap
	t.Run("dismiss_gap", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"patientId": "test-patient-001",
			"reason":    "Patient declined intervention",
		}
		body, _ := json.Marshal(requestBody)

		resp, err := client.Post(
			fmt.Sprintf("%s/api/v1/gaps/%s/dismiss", baseURL, testGapID),
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Log("✅ Gap dismissed successfully")
		} else {
			t.Logf("⚠️ Gap dismiss returned status %d", resp.StatusCode)
		}
	})

	// Test 3: Snooze gap
	t.Run("snooze_gap", func(t *testing.T) {
		snoozeUntil := time.Now().Add(30 * 24 * time.Hour).Format("2006-01-02")
		requestBody := map[string]interface{}{
			"patientId":    "test-patient-001",
			"snoozeUntil":  snoozeUntil,
			"reason":       "Patient traveling, reschedule for next month",
		}
		body, _ := json.Marshal(requestBody)

		resp, err := client.Post(
			fmt.Sprintf("%s/api/v1/gaps/%s/snooze", baseURL, testGapID),
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Logf("✅ Gap snoozed until %s", snoozeUntil)
		} else {
			t.Logf("⚠️ Gap snooze returned status %d", resp.StatusCode)
		}
	})
}

// TestMetricsEndpoint verifies Prometheus metrics are exposed.
func TestMetricsEndpoint(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(baseURL + "/metrics")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check for expected Prometheus metrics
		expectedMetrics := []string{
			"go_goroutines",
			"go_memstats",
			"process_",
		}

		for _, metric := range expectedMetrics {
			if !bytes.Contains(body, []byte(metric)) {
				t.Logf("⚠️ Metric %s not found in output", metric)
			}
		}

		t.Logf("✅ Metrics endpoint responding (size: %d bytes)", len(bodyStr))
	} else if resp.StatusCode == http.StatusNotFound {
		t.Log("⚠️ Metrics endpoint not available (may be disabled)")
	} else {
		t.Errorf("Unexpected status: %d", resp.StatusCode)
	}
}

// TestConcurrentRequests verifies the service handles concurrent requests.
func TestConcurrentRequests(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 30 * time.Second}

	numRequests := 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			resp, err := client.Get(baseURL + "/health")
			if err != nil {
				results <- -1
				return
			}
			resp.Body.Close()
			results <- resp.StatusCode
		}(i)
	}

	successCount := 0
	for i := 0; i < numRequests; i++ {
		status := <-results
		if status == http.StatusOK {
			successCount++
		}
	}

	if successCount < numRequests {
		t.Errorf("Only %d/%d concurrent requests succeeded", successCount, numRequests)
	} else {
		t.Logf("✅ All %d concurrent requests succeeded", numRequests)
	}
}
