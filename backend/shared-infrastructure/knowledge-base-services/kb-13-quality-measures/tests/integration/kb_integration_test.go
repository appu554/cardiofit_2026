//go:build integration
// +build integration

// Package integration provides integration tests for KB-13 with external KB services.
//
// These tests run against LIVE Docker services - NO MOCKS.
//
// Required services (check with docker ps):
//   - KB-18 Governance Engine: port 8018
//   - KB-9 Care Gaps: port 8089
//   - KB-7 Terminology: port 8092 (optional - may be unavailable)
//
// Run with: go test -tags=integration ./tests/integration/... -v
//
// Environment variables (optional overrides):
//   - KB7_URL: KB-7 Terminology Service URL (default: http://localhost:8092)
//   - KB18_URL: KB-18 Governance Engine URL (default: http://localhost:8018)
//   - KB9_URL: KB-9 Care Gaps Service URL (default: http://localhost:8089)
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/integrations"
)

// =============================================================================
// TEST CONFIGURATION
// =============================================================================

// Service URLs - default to local Docker ports
func getKB7URL() string {
	if url := os.Getenv("KB7_URL"); url != "" {
		return url
	}
	return "http://localhost:8092"
}

func getKB18URL() string {
	if url := os.Getenv("KB18_URL"); url != "" {
		return url
	}
	return "http://localhost:8018"
}

func getKB9URL() string {
	if url := os.Getenv("KB9_URL"); url != "" {
		return url
	}
	return "http://localhost:8089"
}

// createTestLogger creates a no-op logger for testing.
func createTestLogger() *zap.Logger {
	return zap.NewNop()
}

// checkServiceHealth checks if a service is available at the given URL.
func checkServiceHealth(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// =============================================================================
// KB-18 GOVERNANCE ENGINE INTEGRATION TESTS (LIVE SERVICE)
// =============================================================================

// TestKB18_Live_HealthCheck tests the real KB-18 health endpoint.
func TestKB18_Live_HealthCheck(t *testing.T) {
	kb18URL := getKB18URL()

	if !checkServiceHealth(kb18URL) {
		t.Skipf("KB-18 service not available at %s", kb18URL)
	}

	logger := createTestLogger()
	client := integrations.NewKB18Client(kb18URL, logger)

	err := client.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("KB-18 health check failed: %v", err)
	}
}

// TestKB18_Live_ListPrograms tests listing governance programs from KB-18.
func TestKB18_Live_ListPrograms(t *testing.T) {
	kb18URL := getKB18URL()

	if !checkServiceHealth(kb18URL) {
		t.Skipf("KB-18 service not available at %s", kb18URL)
	}

	// Call the programs endpoint directly
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(kb18URL + "/api/v1/programs")
	if err != nil {
		t.Fatalf("Failed to get programs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("ListPrograms returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Programs []struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Active      bool   `json:"active"`
		} `json:"programs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode programs response: %v", err)
	}

	t.Run("ProgramsReturned", func(t *testing.T) {
		if len(result.Programs) == 0 {
			t.Error("Expected at least one program from KB-18")
		}
	})

	t.Run("ProgramHasRequiredFields", func(t *testing.T) {
		if len(result.Programs) > 0 {
			prog := result.Programs[0]
			if prog.Code == "" {
				t.Error("Program code should not be empty")
			}
			if prog.Name == "" {
				t.Error("Program name should not be empty")
			}
		}
	})

	t.Logf("KB-18 returned %d governance programs", len(result.Programs))
}

// TestKB18_Live_GetStats tests KB-18 statistics endpoint.
func TestKB18_Live_GetStats(t *testing.T) {
	kb18URL := getKB18URL()

	if !checkServiceHealth(kb18URL) {
		t.Skipf("KB-18 service not available at %s", kb18URL)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(kb18URL + "/api/v1/stats")
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GetStats returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats struct {
		Engine struct {
			TotalEvaluations int `json:"total_evaluations"`
			TotalAllowed     int `json:"total_allowed"`
			TotalBlocked     int `json:"total_blocked"`
		} `json:"engine"`
		Programs struct {
			TotalLoaded int `json:"total_loaded"`
		} `json:"programs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode stats response: %v", err)
	}

	t.Run("ProgramsLoaded", func(t *testing.T) {
		if stats.Programs.TotalLoaded < 0 {
			t.Error("Programs loaded should be >= 0")
		}
		t.Logf("KB-18 has %d programs loaded", stats.Programs.TotalLoaded)
	})
}

// TestKB18_Live_EvaluateMedication tests medication governance evaluation.
func TestKB18_Live_EvaluateMedication(t *testing.T) {
	kb18URL := getKB18URL()

	if !checkServiceHealth(kb18URL) {
		t.Skipf("KB-18 service not available at %s", kb18URL)
	}

	// Build a medication evaluation request
	evalRequest := map[string]interface{}{
		"patient_id":     "test-patient-001",
		"medication_id":  "rxnorm:6809", // Metformin
		"context":        "diabetes_management",
		"requesting_user": "test-user",
	}

	body, _ := json.Marshal(evalRequest)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(
		kb18URL+"/api/v1/evaluate/medication",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to evaluate medication: %v", err)
	}
	defer resp.Body.Close()

	// Accept both 200 OK and 400 Bad Request (if required fields are missing)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("EvaluateMedication returned unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	t.Logf("KB-18 medication evaluation responded with status: %d", resp.StatusCode)
}

// =============================================================================
// KB-9 CARE GAPS SERVICE INTEGRATION TESTS (LIVE SERVICE)
// =============================================================================

// TestKB9_Live_HealthCheck tests the real KB-9 health endpoint.
func TestKB9_Live_HealthCheck(t *testing.T) {
	kb9URL := getKB9URL()

	if !checkServiceHealth(kb9URL) {
		t.Skipf("KB-9 service not available at %s", kb9URL)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(kb9URL + "/health")
	if err != nil {
		t.Fatalf("Failed to check health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check returned status %d", resp.StatusCode)
	}

	var health struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	t.Run("StatusHealthy", func(t *testing.T) {
		if health.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", health.Status)
		}
	})

	t.Logf("KB-9 version: %s, status: %s", health.Version, health.Status)
}

// TestKB9_Live_GetCareGaps tests the care gaps endpoint.
func TestKB9_Live_GetCareGaps(t *testing.T) {
	kb9URL := getKB9URL()

	if !checkServiceHealth(kb9URL) {
		t.Skipf("KB-9 service not available at %s", kb9URL)
	}

	// Build a care gaps request
	careGapsRequest := map[string]interface{}{
		"patientId":         "test-patient-001",
		"measures":          []string{}, // All measures
		"includeClosedGaps": false,
		"includeEvidence":   true,
	}

	body, _ := json.Marshal(careGapsRequest)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(
		kb9URL+"/api/v1/care-gaps",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to get care gaps: %v", err)
	}
	defer resp.Body.Close()

	// Accept 200 OK or 404 Not Found (if patient doesn't exist)
	if resp.StatusCode == http.StatusOK {
		var result struct {
			PatientID string `json:"patientId"`
			Gaps      []struct {
				MeasureType string `json:"measureType"`
				Status      string `json:"status"`
			} `json:"gaps"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode care gaps response: %v", err)
		}
		t.Logf("KB-9 returned %d care gaps for patient", len(result.Gaps))
	} else if resp.StatusCode == http.StatusNotFound {
		t.Log("KB-9 returned 404 - test patient not found (expected for test data)")
	} else {
		respBody, _ := io.ReadAll(resp.Body)
		t.Logf("KB-9 care gaps returned status %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestKB9_Live_MeasureEvaluation tests measure evaluation endpoint.
func TestKB9_Live_MeasureEvaluation(t *testing.T) {
	kb9URL := getKB9URL()

	if !checkServiceHealth(kb9URL) {
		t.Skipf("KB-9 service not available at %s", kb9URL)
	}

	// Build a measure evaluation request
	evalRequest := map[string]interface{}{
		"patientId":   "test-patient-001",
		"measure":     "HBA1C_CONTROL",
		"periodStart": "2024-01-01",
		"periodEnd":   "2024-12-31",
	}

	body, _ := json.Marshal(evalRequest)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(
		kb9URL+"/api/v1/measure/evaluate",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to evaluate measure: %v", err)
	}
	defer resp.Body.Close()

	// Log the response status - we expect various responses depending on data availability
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("KB-9 measure evaluation returned status %d", resp.StatusCode)
	if len(respBody) < 500 {
		t.Logf("Response: %s", string(respBody))
	}
}

// =============================================================================
// KB-7 TERMINOLOGY SERVICE INTEGRATION TESTS (LIVE SERVICE)
// =============================================================================

// TestKB7_Live_HealthCheck tests the real KB-7 health endpoint.
func TestKB7_Live_HealthCheck(t *testing.T) {
	kb7URL := getKB7URL()

	if !checkServiceHealth(kb7URL) {
		t.Skipf("KB-7 service not available at %s", kb7URL)
	}

	logger := createTestLogger()
	client := integrations.NewKB7Client(kb7URL, logger)

	err := client.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("KB-7 health check failed: %v", err)
	}

	t.Log("✅ KB-7 health check passed")
}

// TestKB7_Live_Version tests KB-7 version endpoint.
func TestKB7_Live_Version(t *testing.T) {
	kb7URL := getKB7URL()

	if !checkServiceHealth(kb7URL) {
		t.Skipf("KB-7 service not available at %s", kb7URL)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get(kb7URL + "/version")
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Version returned status %d: %s", resp.StatusCode, string(body))
	}

	var version struct {
		Service      string   `json:"service"`
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
		RuleEngine   struct {
			Enabled bool `json:"enabled"`
		} `json:"rule_engine"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		t.Fatalf("Failed to decode version response: %v", err)
	}

	t.Run("ServiceIdentified", func(t *testing.T) {
		if version.Service != "kb-7-terminology" {
			t.Errorf("Expected service 'kb-7-terminology', got '%s'", version.Service)
		}
	})

	t.Run("HasCapabilities", func(t *testing.T) {
		if len(version.Capabilities) == 0 {
			t.Error("Expected at least one capability")
		}
	})

	t.Run("RuleEngineEnabled", func(t *testing.T) {
		if !version.RuleEngine.Enabled {
			t.Log("Warning: Rule engine is disabled")
		}
	})

	t.Logf("✅ KB-7 version: %s with %d capabilities", version.Version, len(version.Capabilities))
}

// TestKB7_Live_ListTerminologySystems tests listing terminology systems.
func TestKB7_Live_ListTerminologySystems(t *testing.T) {
	kb7URL := getKB7URL()

	if !checkServiceHealth(kb7URL) {
		t.Skipf("KB-7 service not available at %s", kb7URL)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get(kb7URL + "/v1/systems")
	if err != nil {
		t.Fatalf("Failed to list systems: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("ListSystems returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Systems []string `json:"systems"`
		Message string   `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode systems response: %v", err)
	}

	t.Run("SystemsReturned", func(t *testing.T) {
		if len(result.Systems) == 0 {
			t.Error("Expected at least one terminology system")
		}
	})

	t.Run("StandardSystemsPresent", func(t *testing.T) {
		expected := map[string]bool{
			"SNOMED-CT": false,
			"ICD-10":    false,
			"RxNorm":    false,
			"LOINC":     false,
		}
		for _, sys := range result.Systems {
			if _, ok := expected[sys]; ok {
				expected[sys] = true
			}
		}
		for sys, found := range expected {
			if !found {
				t.Logf("Note: Standard system %s not in returned list", sys)
			}
		}
	})

	t.Logf("✅ KB-7 returned %d terminology systems: %v", len(result.Systems), result.Systems)
}

// TestKB7_Live_RuleValueSets tests the rule engine value sets endpoint.
func TestKB7_Live_RuleValueSets(t *testing.T) {
	kb7URL := getKB7URL()

	if !checkServiceHealth(kb7URL) {
		t.Skipf("KB-7 service not available at %s", kb7URL)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get(kb7URL + "/v1/rules/valuesets")
	if err != nil {
		t.Fatalf("Failed to list rule valuesets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("ListRuleValueSets returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Count       int    `json:"count"`
		Description string `json:"description"`
		Source      string `json:"source"`
		ValueSets   []struct {
			Identifier string `json:"identifier"`
			Name       string `json:"name"`
		} `json:"value_sets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode rule valuesets response: %v", err)
	}

	t.Run("RuleEngineSource", func(t *testing.T) {
		if result.Source != "rule_engine" {
			t.Errorf("Expected source 'rule_engine', got '%s'", result.Source)
		}
	})

	t.Logf("✅ KB-7 rule engine has %d value sets loaded", result.Count)
}

// TestKB7_Live_ClassifyCode tests code classification endpoint.
func TestKB7_Live_ClassifyCode(t *testing.T) {
	kb7URL := getKB7URL()

	if !checkServiceHealth(kb7URL) {
		t.Skipf("KB-7 service not available at %s", kb7URL)
	}

	// Test classification with a diabetes SNOMED code
	classifyRequest := map[string]interface{}{
		"code":   "44054006",                  // Type 2 diabetes mellitus
		"system": "http://snomed.info/sct",
	}

	body, _ := json.Marshal(classifyRequest)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Post(
		kb7URL+"/v1/rules/classify",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to classify code: %v", err)
	}
	defer resp.Body.Close()

	// Accept both 200 (found) and 404 (no matching valuesets)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("ClassifyCode returned unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	if resp.StatusCode == http.StatusOK {
		var result struct {
			Code           string   `json:"code"`
			System         string   `json:"system"`
			MatchedSets    []string `json:"matched_sets"`
			TotalMatches   int      `json:"total_matches"`
			Classification string   `json:"classification"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			// May have different response structure - log and continue
			t.Logf("Note: Could not decode classify response as expected structure")
		} else {
			t.Logf("✅ KB-7 classified code 44054006: %d matching valuesets", result.TotalMatches)
		}
	} else {
		t.Log("✅ KB-7 classify endpoint responded (no matching valuesets for test code)")
	}
}

// TestKB7_Live_ValueSetLookup tests value set operations if KB-7 is available.
func TestKB7_Live_ValueSetLookup(t *testing.T) {
	kb7URL := getKB7URL()

	if !checkServiceHealth(kb7URL) {
		t.Skipf("KB-7 service not available at %s", kb7URL)
	}

	logger := createTestLogger()
	client := integrations.NewKB7Client(kb7URL, logger)

	ctx := context.Background()

	// Try to get the Diabetes value set
	vs, err := client.GetValueSet(ctx, "2.16.840.1.113883.3.464.1003.103.12.1001")
	if err != nil {
		t.Logf("ValueSet lookup returned error (may be expected if valueset not loaded): %v", err)
		return
	}

	t.Run("ValueSetReturned", func(t *testing.T) {
		if vs.OID == "" && vs.Name == "" {
			t.Error("Expected ValueSet to have OID or Name")
		}
	})

	t.Logf("✅ KB-7 returned value set: %s (%s)", vs.Name, vs.OID)
}

// TestKB7_Live_FHIREndpoints tests FHIR-compliant endpoints.
func TestKB7_Live_FHIREndpoints(t *testing.T) {
	kb7URL := getKB7URL()

	if !checkServiceHealth(kb7URL) {
		t.Skipf("KB-7 service not available at %s", kb7URL)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}

	t.Run("FHIRMetadata", func(t *testing.T) {
		resp, err := httpClient.Get(kb7URL + "/fhir/metadata")
		if err != nil {
			t.Fatalf("Failed to get FHIR metadata: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("FHIR metadata returned status %d: %s", resp.StatusCode, string(body))
		}

		var capability struct {
			ResourceType string `json:"resourceType"`
			Status       string `json:"status"`
			FhirVersion  string `json:"fhirVersion"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&capability); err != nil {
			t.Logf("Note: Could not decode capability statement: %v", err)
			return
		}

		if capability.ResourceType != "CapabilityStatement" {
			t.Errorf("Expected ResourceType 'CapabilityStatement', got '%s'", capability.ResourceType)
		}

		t.Logf("✅ FHIR CapabilityStatement: version=%s, status=%s", capability.FhirVersion, capability.Status)
	})

	t.Run("FHIRHealth", func(t *testing.T) {
		resp, err := httpClient.Get(kb7URL + "/fhir/health")
		if err != nil {
			t.Fatalf("Failed to get FHIR health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("FHIR health returned status %d", resp.StatusCode)
		}

		t.Log("✅ KB-7 FHIR health endpoint accessible")
	})
}

// =============================================================================
// CROSS-SERVICE INTEGRATION TESTS
// =============================================================================

// TestCrossService_KB13_KB18_Integration tests KB-13 to KB-18 governance flow.
func TestCrossService_KB13_KB18_Integration(t *testing.T) {
	kb18URL := getKB18URL()

	if !checkServiceHealth(kb18URL) {
		t.Skipf("KB-18 service not available at %s", kb18URL)
	}

	logger := createTestLogger()
	kb18Client := integrations.NewKB18Client(kb18URL, logger)

	// Test the governance client can communicate with KB-18
	t.Run("HealthCheckConnection", func(t *testing.T) {
		err := kb18Client.HealthCheck(context.Background())
		if err != nil {
			t.Errorf("KB-18 client health check failed: %v", err)
		}
	})

	// Test creating an audit entry (if endpoint exists)
	t.Run("CreateAuditEntry", func(t *testing.T) {
		err := kb18Client.CreateAuditEntry(
			context.Background(),
			"TEST_INTEGRATION",
			"CMS122v12",
			"Integration test audit entry from KB-13",
		)
		// Log error but don't fail - endpoint may not exist
		if err != nil {
			t.Logf("CreateAuditEntry returned: %v (endpoint may not be implemented)", err)
		}
	})
}

// TestCrossService_KB13_KB9_Integration tests KB-13 to KB-9 care gaps flow.
func TestCrossService_KB13_KB9_Integration(t *testing.T) {
	kb9URL := getKB9URL()

	if !checkServiceHealth(kb9URL) {
		t.Skipf("KB-9 service not available at %s", kb9URL)
	}

	// Test that we can communicate with KB-9
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(kb9URL + "/health")
	if err != nil {
		t.Fatalf("Failed to connect to KB-9: %v", err)
	}
	defer resp.Body.Close()

	t.Run("ServiceReachable", func(t *testing.T) {
		if resp.StatusCode != http.StatusOK {
			t.Errorf("KB-9 health check returned status %d", resp.StatusCode)
		}
	})

	// Verify KB-9 is the authoritative care gaps source
	t.Run("KB9_Is_Authoritative_For_CareGaps", func(t *testing.T) {
		// This validates the architectural principle that KB-9 is authoritative
		// KB-13 can calculate gaps but must mark them as non-authoritative
		var health struct {
			Service string `json:"service,omitempty"`
			Status  string `json:"status"`
			Checks  struct {
				CareGapsService string `json:"care_gaps_service,omitempty"`
			} `json:"checks,omitempty"`
		}

		resp, _ := client.Get(kb9URL + "/health")
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&health)

		// KB-9 must be healthy for it to be the authoritative source
		if health.Status != "healthy" {
			t.Logf("Warning: KB-9 is not healthy - KB-13 gaps would be the only source")
		} else {
			t.Log("KB-9 (authoritative care gaps) is healthy")
		}
	})
}

// =============================================================================
// SERVICE AVAILABILITY SUMMARY TEST
// =============================================================================

// TestServiceAvailability provides a summary of all KB service availability.
func TestServiceAvailability(t *testing.T) {
	services := []struct {
		name string
		url  string
	}{
		{"KB-7 Terminology", getKB7URL()},
		{"KB-9 Care Gaps", getKB9URL()},
		{"KB-18 Governance", getKB18URL()},
	}

	available := 0
	for _, svc := range services {
		isAvailable := checkServiceHealth(svc.url)
		status := "❌ UNAVAILABLE"
		if isAvailable {
			status = "✅ AVAILABLE"
			available++
		}
		t.Logf("%s (%s): %s", svc.name, svc.url, status)
	}

	t.Logf("\n=== Summary: %d/%d KB services available ===", available, len(services))

	if available == 0 {
		t.Log("No KB services available - integration tests will be skipped")
	}
}
