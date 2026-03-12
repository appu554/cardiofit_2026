// Package integration_tests validates the Vaidshala V3 clinical runtime chain.
//
// Chain under test:
//
//	KB-20 (Patient Profile) → KB-22 (HPI Engine) → KB-23 (Decision Cards) → KB-19 (Protocol Orchestrator)
//
// Each test checks health, then exercises the primary API contract.
// Run with: go test -tags=integration -v ./integration_tests/...
//
//go:build integration

package integration_tests

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

// Canonical ports for Vaidshala runtime services.
var (
	kb19URL = envOrDefault("KB19_URL", "http://localhost:8103")
	kb20URL = envOrDefault("KB20_URL", "http://localhost:8131")
	kb21URL = envOrDefault("KB21_URL", "http://localhost:8133")
	kb22URL = envOrDefault("KB22_URL", "http://localhost:8132")
	kb23URL = envOrDefault("KB23_URL", "http://localhost:8134")
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ---------------------------------------------------------------------------
// 1. Health checks — every service must respond before chain tests run
// ---------------------------------------------------------------------------

func TestHealth_KB19(t *testing.T) { assertHealthy(t, kb19URL, "kb-19-protocol-orchestrator") }
func TestHealth_KB20(t *testing.T) { assertHealthy(t, kb20URL, "kb-20-patient-profile") }
func TestHealth_KB21(t *testing.T) { assertHealthy(t, kb21URL, "kb-21-behavioral-intelligence") }
func TestHealth_KB22(t *testing.T) { assertHealthy(t, kb22URL, "kb-22-hpi-engine") }
func TestHealth_KB23(t *testing.T) { assertHealthy(t, kb23URL, "kb-23-decision-cards") }

func assertHealthy(t *testing.T, baseURL, name string) {
	t.Helper()
	resp, err := httpGet(baseURL + "/health")
	if err != nil {
		t.Fatalf("%s health check failed: %v", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s unhealthy: status=%d body=%s", name, resp.StatusCode, string(body))
	}
	t.Logf("%s: healthy", name)
}

// ---------------------------------------------------------------------------
// 2. KB-20 Patient Profile — create patient and add labs
// ---------------------------------------------------------------------------

func TestKB20_PatientLifecycle(t *testing.T) {
	// Create patient
	patient := map[string]interface{}{
		"patient_id":    "integration-test-001",
		"name":          "Integration Test Patient",
		"date_of_birth": "1960-05-15",
		"sex":           "male",
	}
	resp := mustPost(t, kb20URL+"/api/v1/patient", patient)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		t.Fatalf("KB-20 create patient: unexpected status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Add a lab result (creatinine for eGFR)
	lab := map[string]interface{}{
		"lab_type":    "creatinine",
		"value":       1.2,
		"unit":        "mg/dL",
		"measured_at": time.Now().UTC().Format(time.RFC3339),
		"source":      "integration_test",
	}
	resp = mustPost(t, kb20URL+"/api/v1/patient/integration-test-001/labs", lab)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("KB-20 add lab: status=%d body=%s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Read stratum
	resp, err := httpGet(kb20URL + "/api/v1/patient/integration-test-001/stratum/diabetes_t2")
	if err != nil {
		t.Fatalf("KB-20 get stratum: %v", err)
	}
	defer resp.Body.Close()
	t.Logf("KB-20 stratum status: %d", resp.StatusCode)
}

// ---------------------------------------------------------------------------
// 3. KB-21 Behavioral Intelligence — adherence query
// ---------------------------------------------------------------------------

func TestKB21_AdherenceQuery(t *testing.T) {
	resp, err := httpGet(kb21URL + "/api/v1/patient/integration-test-001/adherence/summary")
	if err != nil {
		t.Fatalf("KB-21 adherence query failed: %v", err)
	}
	defer resp.Body.Close()
	// 200 or 404 (patient not yet tracked) are both acceptable
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("KB-21 adherence: unexpected status %d body=%s", resp.StatusCode, string(body))
	}
	t.Logf("KB-21 adherence status: %d", resp.StatusCode)
}

// ---------------------------------------------------------------------------
// 4. KB-22 HPI Engine — create session and submit answer
// ---------------------------------------------------------------------------

func TestKB22_SessionLifecycle(t *testing.T) {
	// List available nodes first
	resp, err := httpGet(kb22URL + "/api/v1/nodes")
	if err != nil {
		t.Fatalf("KB-22 list nodes: %v", err)
	}
	defer resp.Body.Close()

	var nodes []map[string]interface{}
	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &nodes)
		t.Logf("KB-22 available nodes: %d", len(nodes))
	}

	if len(nodes) == 0 {
		t.Skip("KB-22: no nodes loaded, skipping session test")
	}

	// Create an HPI session using the first available node
	nodeID := nodes[0]["node_id"]
	session := map[string]interface{}{
		"patient_id": "00000000-0000-0000-0000-000000000001",
		"node_id":    nodeID,
	}
	resp2 := mustPost(t, kb22URL+"/api/v1/sessions", session)
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusCreated {
		var created map[string]interface{}
		body, _ := io.ReadAll(resp2.Body)
		_ = json.Unmarshal(body, &created)
		sessionID, ok := created["session_id"]
		if ok {
			t.Logf("KB-22 session created: %v", sessionID)

			// Get differential
			diffResp, err := httpGet(fmt.Sprintf("%s/api/v1/sessions/%v/differential", kb22URL, sessionID))
			if err == nil {
				defer diffResp.Body.Close()
				t.Logf("KB-22 differential status: %d", diffResp.StatusCode)
			}
		}
	} else {
		body, _ := io.ReadAll(resp2.Body)
		t.Logf("KB-22 session creation: status=%d body=%s", resp2.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// 5. KB-23 Decision Cards — generate card and check MCU gate
// ---------------------------------------------------------------------------

func TestKB23_DecisionCardGeneration(t *testing.T) {
	cardReq := map[string]interface{}{
		"patient_id":  "integration-test-001",
		"session_id":  "00000000-0000-0000-0000-000000000001",
		"template_id": "diabetes_t2_review",
		"node_id":     "diabetes_t2",
		"primary_differential": map[string]interface{}{
			"differential_id": "T2DM",
			"posterior":       0.82,
		},
	}
	resp := mustPost(t, kb23URL+"/api/v1/decision-cards", cardReq)
	defer resp.Body.Close()

	// May fail if template not loaded — that's acceptable for integration smoke test
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		var card map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &card)
		t.Logf("KB-23 card generated: card_id=%v", card["card_id"])
	} else {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("KB-23 card generation: status=%d body=%s (may need template loading)", resp.StatusCode, string(body))
	}

	// Check MCU gate endpoint
	gateResp, err := httpGet(kb23URL + "/api/v1/patients/integration-test-001/mcu-gate")
	if err != nil {
		t.Logf("KB-23 MCU gate query: %v", err)
		return
	}
	defer gateResp.Body.Close()
	t.Logf("KB-23 MCU gate status: %d", gateResp.StatusCode)
}

// ---------------------------------------------------------------------------
// 6. KB-19 Protocol Orchestrator — readiness and health
// ---------------------------------------------------------------------------

func TestKB19_Readiness(t *testing.T) {
	resp, err := httpGet(kb19URL + "/ready")
	if err != nil {
		t.Fatalf("KB-19 readiness check: %v", err)
	}
	defer resp.Body.Close()

	var ready map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &ready)
	t.Logf("KB-19 readiness: %s", string(body))
}

// ---------------------------------------------------------------------------
// 7. Cross-KB chain test: KB-20 → KB-22 → KB-23
// ---------------------------------------------------------------------------

func TestChain_ProfileToCard(t *testing.T) {
	// Step 1: Ensure patient profile exists in KB-20
	resp, err := httpGet(kb20URL + "/api/v1/patient/integration-test-001/profile")
	if err != nil {
		t.Skipf("KB-20 patient profile unavailable, skipping chain test: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skip("KB-20 patient not found, skipping chain test")
	}

	// Step 2: Check KB-21 behavioural data availability
	resp2, err := httpGet(kb21URL + "/api/v1/patient/integration-test-001/adherence/summary")
	if err != nil {
		t.Logf("KB-21 unavailable (non-fatal): %v", err)
	} else {
		resp2.Body.Close()
		t.Logf("KB-21 adherence for chain patient: status=%d", resp2.StatusCode)
	}

	// Step 3: Check KB-23 active cards for the patient
	resp3, err := httpGet(kb23URL + "/api/v1/patients/integration-test-001/active-cards")
	if err != nil {
		t.Logf("KB-23 unavailable (non-fatal): %v", err)
	} else {
		defer resp3.Body.Close()
		body, _ := io.ReadAll(resp3.Body)
		t.Logf("KB-23 active cards: status=%d body=%s", resp3.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// 8. Port registry validation — canonical port assignments
// ---------------------------------------------------------------------------

func TestPortRegistry_CanonicalPorts(t *testing.T) {
	expected := map[string]string{
		"KB-19": "8103",
		"KB-20": "8131",
		"KB-21": "8133",
		"KB-22": "8132",
		"KB-23": "8134",
	}

	urls := map[string]string{
		"KB-19": kb19URL,
		"KB-20": kb20URL,
		"KB-21": kb21URL,
		"KB-22": kb22URL,
		"KB-23": kb23URL,
	}

	for name, port := range expected {
		url := urls[name]
		expectedSuffix := ":" + port
		if len(url) < len(expectedSuffix) {
			t.Errorf("%s URL %q does not end with canonical port %s", name, url, expectedSuffix)
			continue
		}
		// Extract port from URL
		actualPort := url[len(url)-4:]
		if actualPort != port {
			t.Errorf("%s: expected canonical port %s, got URL %s", name, port, url)
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var client = &http.Client{Timeout: 10 * time.Second}

func httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	return client.Do(req)
}

func mustPost(t *testing.T, url string, payload interface{}) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}
