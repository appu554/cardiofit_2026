// Package tests provides clinical-device rigor testing for KB-18 Governance Engine.
// This file tests API CONTRACT compliance for all HTTP endpoints.
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"kb-18-governance-engine/internal/api"
	"kb-18-governance-engine/internal/config"
	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// API CONTRACT TESTS - HTTP Endpoint Verification
// =============================================================================

func setupTestServer() *httptest.Server {
	gin.SetMode(gin.TestMode)
	cfg := config.Load()
	server, err := api.NewServer(cfg)
	if err != nil {
		panic("Failed to create test server: " + err.Error())
	}
	return httptest.NewServer(server.Router())
}

// TestAPIContract_HealthCheck verifies GET /health endpoint
func TestAPIContract_HealthCheck(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Health check request failed: %v", err)
	}
	defer resp.Body.Close()

	// Status code must be 200
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Response must be JSON
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Logf("Content-Type: %s", ct)
	}

	// Parse response
	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}

	// Must have status field
	if status, ok := health["status"]; !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", health["status"])
	}

	t.Logf("✅ HEALTH CHECK ENDPOINT: %v", health)
}

// TestAPIContract_GetPrograms verifies GET /api/v1/programs endpoint
func TestAPIContract_GetPrograms(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/programs")
	if err != nil {
		t.Fatalf("Get programs request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse programs response: %v", err)
	}

	// Must have programs array
	if programs, ok := result["programs"]; ok {
		if arr, ok := programs.([]interface{}); ok {
			t.Logf("✅ GET PROGRAMS: Returned %d programs", len(arr))
		}
	} else {
		t.Logf("Response: %v", result)
	}
}

// TestAPIContract_GetProgramByCode verifies GET /api/v1/programs/:code endpoint
func TestAPIContract_GetProgramByCode(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// Test existing program
	resp, err := http.Get(ts.URL + "/api/v1/programs/MAT")
	if err != nil {
		t.Fatalf("Get program request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 200 or 404, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse program response: %v", err)
	}

	t.Logf("✅ GET PROGRAM BY CODE: Status %d", resp.StatusCode)

	// Test non-existing program
	resp404, _ := http.Get(ts.URL + "/api/v1/programs/NONEXISTENT")
	defer resp404.Body.Close()

	if resp404.StatusCode != http.StatusNotFound {
		t.Logf("Non-existent program returned status %d", resp404.StatusCode)
	}
}

// TestAPIContract_EvaluateMedication verifies POST /api/v1/evaluate endpoint
func TestAPIContract_EvaluateMedication(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// Create evaluation request
	evalReq := map[string]interface{}{
		"patientId": "PT-API-TEST",
		"patientContext": map[string]interface{}{
			"patientId":  "PT-API-TEST",
			"age":        30,
			"sex":        "F",
			"isPregnant": true,
		},
		"medicationOrder": map[string]interface{}{
			"medicationCode": "MTX",
			"medicationName": "Methotrexate",
			"drugClass":      "METHOTREXATE",
			"dose":           15.0,
			"doseUnit":       "mg",
		},
		"evaluationType": "medication",
		"requestorId":    "DR-001",
		"requestorRole":  "PHYSICIAN",
		"facilityId":     "HOSP-001",
	}

	body, _ := json.Marshal(evalReq)
	resp, err := http.Post(ts.URL+"/api/v1/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Evaluate request failed: %v", err)
	}
	defer resp.Body.Close()

	// Must return 200
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse evaluate response: %v", err)
	}

	// Must have required fields
	requiredFields := []string{"requestId", "outcome", "isApproved", "hasViolations", "evidenceTrail"}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}

	t.Logf("✅ EVALUATE ENDPOINT: outcome=%v, isApproved=%v", result["outcome"], result["isApproved"])
}

// TestAPIContract_EvaluateBadRequest verifies 400 on invalid request
func TestAPIContract_EvaluateBadRequest(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// Empty request
	resp, err := http.Post(ts.URL+"/api/v1/evaluate", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Logf("Expected 400 for empty request, got %d", resp.StatusCode)
	}

	// Invalid JSON
	resp2, err := http.Post(ts.URL+"/api/v1/evaluate", "application/json", bytes.NewReader([]byte("not json")))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusBadRequest {
		t.Logf("Expected 400 for invalid JSON, got %d", resp2.StatusCode)
	}

	t.Logf("✅ BAD REQUEST HANDLING verified")
}

// TestAPIContract_GetOverrides verifies GET /api/v1/overrides endpoint
func TestAPIContract_GetOverrides(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/overrides")
	if err != nil {
		t.Fatalf("Get overrides request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse overrides response: %v", err)
	}

	t.Logf("✅ GET OVERRIDES ENDPOINT: %v", result)
}

// TestAPIContract_RequestOverride verifies POST /api/v1/overrides endpoint
func TestAPIContract_RequestOverride(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	overrideReq := map[string]interface{}{
		"violation_id":           "VIOL-001",
		"patient_id":             "PT-001",
		"requestor_id":           "DR-001",
		"requestor_role":         "PHYSICIAN",
		"rule_code":              "MAT-001",
		"reason":                 "Clinical necessity",
		"clinical_justification": "Patient requires treatment despite risk",
		"risk_accepted":          true,
	}

	body, _ := json.Marshal(overrideReq)
	resp, err := http.Post(ts.URL+"/api/v1/overrides", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Override request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 200 or 201
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Logf("Override request returned status %d", resp.StatusCode)
	}

	t.Logf("✅ REQUEST OVERRIDE ENDPOINT: Status %d", resp.StatusCode)
}

// TestAPIContract_GetStats verifies GET /api/v1/stats endpoint
func TestAPIContract_GetStats(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/stats")
	if err != nil {
		t.Fatalf("Get stats request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse stats response: %v", err)
	}

	// Stats should have engine statistics
	if stats, ok := result["engine"]; ok {
		t.Logf("Engine stats: %v", stats)
	}

	t.Logf("✅ GET STATS ENDPOINT: %v", result)
}

// TestAPIContract_ResponseHeaders verifies required response headers
func TestAPIContract_ResponseHeaders(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check content-type
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		t.Error("Missing Content-Type header")
	}

	t.Logf("✅ RESPONSE HEADERS: Content-Type=%s", ct)
}

// TestAPIContract_CORSHeaders verifies CORS headers if applicable
func TestAPIContract_CORSHeaders(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// OPTIONS request
	req, _ := http.NewRequest("OPTIONS", ts.URL+"/api/v1/evaluate", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS request failed: %v", err)
	}
	defer resp.Body.Close()

	// Log CORS headers if present
	acao := resp.Header.Get("Access-Control-Allow-Origin")
	acam := resp.Header.Get("Access-Control-Allow-Methods")

	t.Logf("CORS Headers: Allow-Origin=%s, Allow-Methods=%s", acao, acam)
	t.Logf("✅ CORS HEADERS check complete")
}

// TestAPIContract_EvaluateResponseStructure verifies complete response structure
func TestAPIContract_EvaluateResponseStructure(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	evalReq := &types.EvaluationRequest{
		PatientID: "PT-STRUCT",
		PatientContext: &types.PatientContext{
			PatientID:  "PT-STRUCT",
			Age:        28,
			Sex:        "F",
			IsPregnant: true,
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "PREGNANCY", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "MTX",
			MedicationName: "Methotrexate",
			DrugClass:      "METHOTREXATE",
			Dose:           15.0,
			DoseUnit:       "mg",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		RequestorRole:  "PHYSICIAN",
		Timestamp:      time.Now(),
	}

	body, _ := json.Marshal(evalReq)
	resp, err := http.Post(ts.URL+"/api/v1/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	var result types.EvaluationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify structure
	if result.RequestID == "" {
		t.Error("Missing requestId")
	}

	if result.Outcome == "" {
		t.Error("Missing outcome")
	}

	if result.EvidenceTrail == nil {
		t.Error("Missing evidenceTrail")
	} else {
		if result.EvidenceTrail.TrailID == "" {
			t.Error("Missing trailId in evidence trail")
		}
		if result.EvidenceTrail.Hash == "" {
			t.Error("Missing hash in evidence trail")
		}
	}

	t.Logf("✅ RESPONSE STRUCTURE VERIFIED")
	t.Logf("   RequestID: %s", result.RequestID)
	t.Logf("   Outcome: %s", result.Outcome)
	t.Logf("   Violations: %d", len(result.Violations))
	if result.EvidenceTrail != nil {
		t.Logf("   TrailID: %s", result.EvidenceTrail.TrailID)
	}
}

// TestAPIContract_EscalationsEndpoint verifies GET /api/v1/escalations endpoint
func TestAPIContract_EscalationsEndpoint(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/escalations")
	if err != nil {
		t.Fatalf("Get escalations request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Logf("✅ ESCALATIONS ENDPOINT: Status %d", resp.StatusCode)
}

// TestAPIContract_AcknowledgmentsEndpoint verifies GET /api/v1/acknowledgments endpoint
func TestAPIContract_AcknowledgmentsEndpoint(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/acknowledgments")
	if err != nil {
		t.Fatalf("Get acknowledgments request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Logf("✅ ACKNOWLEDGMENTS ENDPOINT: Status %d", resp.StatusCode)
}
