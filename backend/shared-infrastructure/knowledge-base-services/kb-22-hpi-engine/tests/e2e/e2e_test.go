//go:build e2e

// Package e2e tests the full KB-22 HPI Engine stack from the outside using
// HTTP requests against a running docker-compose deployment.
//
// Prerequisites:
//   docker-compose -f docker-compose.test.yml up -d
//   go run . &   (or docker-compose up kb22-service)
//
// Run:
//   KB22_BASE_URL=http://localhost:8132 go test -tags=e2e -v ./tests/e2e/...
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

var baseURL string

func TestMain(m *testing.M) {
	baseURL = os.Getenv("KB22_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8132"
	}

	// Wait for service to be ready (up to 60s)
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < 30; i++ {
		resp, err := client.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		if i == 29 {
			fmt.Fprintf(os.Stderr, "SKIP: KB-22 service not reachable at %s\n", baseURL)
			os.Exit(0)
		}
		time.Sleep(2 * time.Second)
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func httpJSON(t *testing.T, method, path string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		t.Fatalf("request build: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	return resp.StatusCode, result
}

// ---------------------------------------------------------------------------
// E2E-01: Health and Readiness
// ---------------------------------------------------------------------------

func TestE2E_HealthCheck(t *testing.T) {
	code, body := httpJSON(t, "GET", "/health", nil)
	if code != 200 {
		t.Fatalf("GET /health returned %d", code)
	}
	checks := body["checks"].(map[string]interface{})
	if checks["database"] != "healthy" {
		t.Errorf("database: %v", checks["database"])
	}
	if checks["redis"] != "healthy" {
		t.Errorf("redis: %v", checks["redis"])
	}
}

func TestE2E_Readiness(t *testing.T) {
	code, _ := httpJSON(t, "GET", "/readiness", nil)
	if code != 200 {
		t.Fatalf("GET /readiness returned %d", code)
	}
}

func TestE2E_Metrics(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET /metrics returned %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	content := string(body)
	for _, metric := range []string{
		"kb22_sessions_started_total",
		"kb22_questions_asked_total",
		"kb22_answer_latency_ms",
	} {
		if !contains(content, metric) {
			t.Errorf("expected metric %q in /metrics output", metric)
		}
	}
}

// ---------------------------------------------------------------------------
// E2E-02: Node listing
// ---------------------------------------------------------------------------

func TestE2E_ListNodes(t *testing.T) {
	code, body := httpJSON(t, "GET", "/api/v1/nodes", nil)
	if code != 200 {
		t.Fatalf("GET /nodes returned %d", code)
	}
	count := body["count"].(float64)
	if count < 3 {
		t.Errorf("expected >=3 nodes, got %v", count)
	}
	t.Logf("loaded %v nodes", count)
}

// ---------------------------------------------------------------------------
// E2E-03: Full session lifecycle
// ---------------------------------------------------------------------------

func TestE2E_FullSessionLifecycle(t *testing.T) {
	patientID := uuid.New()

	// Create session
	code, body := httpJSON(t, "POST", "/api/v1/sessions", map[string]interface{}{
		"patient_id": patientID.String(),
		"node_id":    "P01_CHEST_PAIN",
	})
	if code != 201 {
		t.Fatalf("POST /sessions returned %d: %v", code, body)
	}

	sessionID := body["session_id"].(string)
	status := body["status"].(string)
	if status != "ACTIVE" {
		t.Errorf("expected ACTIVE, got %s", status)
	}

	t.Logf("created session %s", sessionID)

	// Get current question
	currentQ := body["current_question"]
	if currentQ == nil {
		t.Fatal("no current_question in session response")
	}
	questionID := currentQ.(map[string]interface{})["question_id"].(string)

	// Submit 5 answers
	for i := 0; i < 5; i++ {
		answerValue := "NO"
		if i%2 == 0 {
			answerValue = "YES"
		}

		code, ansBody := httpJSON(t, "POST", fmt.Sprintf("/api/v1/sessions/%s/answers", sessionID), map[string]interface{}{
			"question_id":  questionID,
			"answer_value": answerValue,
			"latency_ms":   1000 + i*200,
		})

		if code != 200 {
			t.Fatalf("answer %d: returned %d: %v", i+1, code, ansBody)
		}

		ansStatus := ansBody["status"].(string)
		t.Logf("answer %d: status=%s", i+1, ansStatus)

		if ansStatus == "COMPLETED" || ansStatus == "SAFETY_ESCALATED" || ansStatus == "PARTIAL_ASSESSMENT" {
			t.Logf("session terminated after %d answers", i+1)
			return
		}

		nextQ := ansBody["next_question"]
		if nextQ == nil {
			t.Logf("no more questions after answer %d", i+1)
			break
		}
		questionID = nextQ.(map[string]interface{})["question_id"].(string)
	}

	// Get differential
	code, diffBody := httpJSON(t, "GET", fmt.Sprintf("/api/v1/sessions/%s/differential", sessionID), nil)
	if code != 200 {
		t.Fatalf("GET /differential returned %d: %v", code, diffBody)
	}
	diffs := diffBody["differentials"]
	if diffs == nil {
		t.Error("no differentials returned")
	} else {
		diffList := diffs.([]interface{})
		t.Logf("got %d differentials", len(diffList))
		if len(diffList) > 0 {
			top := diffList[0].(map[string]interface{})
			t.Logf("top differential: %v (posterior=%v)", top["differential_id"], top["posterior_probability"])
		}
	}

	// Get safety flags
	code, safetyBody := httpJSON(t, "GET", fmt.Sprintf("/api/v1/sessions/%s/safety", sessionID), nil)
	if code != 200 {
		t.Fatalf("GET /safety returned %d: %v", code, safetyBody)
	}

	// Get session state
	code, sessionBody := httpJSON(t, "GET", fmt.Sprintf("/api/v1/sessions/%s", sessionID), nil)
	if code != 200 {
		t.Fatalf("GET /sessions/%s returned %d", sessionID, code)
	}
	t.Logf("session state: status=%v questions_asked=%v", sessionBody["status"], sessionBody["questions_asked"])
}

// ---------------------------------------------------------------------------
// E2E-04: Suspend/Resume flow
// ---------------------------------------------------------------------------

func TestE2E_SuspendResume(t *testing.T) {
	patientID := uuid.New()

	code, body := httpJSON(t, "POST", "/api/v1/sessions", map[string]interface{}{
		"patient_id": patientID.String(),
		"node_id":    "P01_CHEST_PAIN",
	})
	if code != 201 {
		t.Fatalf("create returned %d: %v", code, body)
	}
	sessionID := body["session_id"].(string)

	// Suspend
	code, _ = httpJSON(t, "POST", fmt.Sprintf("/api/v1/sessions/%s/suspend", sessionID), nil)
	if code != 200 {
		t.Fatalf("suspend returned %d", code)
	}

	// Resume
	code, resumeBody := httpJSON(t, "POST", fmt.Sprintf("/api/v1/sessions/%s/resume", sessionID), nil)
	if code != 200 {
		t.Fatalf("resume returned %d: %v", code, resumeBody)
	}

	t.Logf("resume response: %v", resumeBody)
}

// ---------------------------------------------------------------------------
// E2E-05: Golden dataset import
// ---------------------------------------------------------------------------

func TestE2E_GoldenDatasetImport(t *testing.T) {
	golden := map[string]interface{}{
		"cases": []map[string]interface{}{
			{
				"node_id":              "P01_CHEST_PAIN",
				"stratum_label":        "DM_HTN_base",
				"confirmed_diagnosis":  "ACS",
				"engine_top_1":         "ACS",
				"engine_top_3":         []string{"ACS", "PE", "STABLE_ANGINA"},
				"question_answers":     map[string]string{"Q001": "YES", "Q002": "YES"},
			},
		},
	}

	code, body := httpJSON(t, "POST", "/api/v1/calibration/import-golden", golden)
	if code != 201 {
		t.Fatalf("import-golden returned %d: %v", code, body)
	}
	t.Logf("golden import result: %v", body)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
