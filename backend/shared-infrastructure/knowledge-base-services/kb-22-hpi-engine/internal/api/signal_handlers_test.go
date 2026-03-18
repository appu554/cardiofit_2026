package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/services"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// minimalSignalGroup creates a SignalHandlerGroup with only the loaders
// populated (other fields nil). Suitable for testing node listing endpoints
// which do not require DB or engine access.
func minimalSignalGroup(monDir, deterDir string) *SignalHandlerGroup {
	log := zap.NewNop()
	monLoader := services.NewMonitoringNodeLoader(monDir, log)
	_ = monLoader.Load()
	deterLoader := services.NewDeteriorationNodeLoader(deterDir, log)
	_ = deterLoader.Load()
	return &SignalHandlerGroup{
		monitoringLoader:    monLoader,
		deteriorationLoader: deterLoader,
		log:                 log,
	}
}

func writeTestYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writeTestYAML: %v", err)
	}
}

const minimalMonitoringNodeYAML = `
node_id: PM-01
version: "1.0.0"
type: MONITORING
title_en: "Test PM Node"
title_hi: "टेस्ट पीएम नोड"
required_inputs:
  - field: fbg
    source: KB-20
    unit: mg/dL
    min_observations: 1
    lookback_days: 7
classifications:
  - category: NORMAL
    condition: "fbg < 100"
    severity: INFO
    mcu_gate_suggestion: NONE
insufficient_data:
  action: SKIP
  note_en: "Insufficient data"
`

const minimalDeteriorationNodeYAML = `
node_id: MD-01
version: "1.0.0"
type: DETERIORATION
title_en: "Test MD Node"
title_hi: "टेस्ट एमडी नोड"
trigger_on:
  - event: "OBSERVATION:FBG"
required_inputs:
  - field: egfr
    source: KB-26
    unit: mL/min
    min_observations: 1
    lookback_days: 90
trajectory:
  method: LINEAR_REGRESSION
  window_days: 90
  min_data_points: 3
  rate_unit: mL/min/month
  data_source: KB-26
thresholds:
  - signal: MILD_EGFR_DECLINE
    condition: "egfr < 60"
    severity: MILD
    trajectory: STABLE
    mcu_gate_suggestion: FLAG_FOR_REVIEW
insufficient_data:
  action: SKIP
  note_en: "Insufficient data"
`

// ---------------------------------------------------------------------------
// Event ingestion handler tests
// ---------------------------------------------------------------------------

func TestHandleObservation_Returns202(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.POST("/signals/events/observation", g.handleObservation)

	body := ObservationEvent{
		PatientID:       "patient-001",
		ObservationCode: "FBG",
		Value:           110.5,
		Unit:            "mg/dL",
		StratumLabel:    "T2DM",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/signals/events/observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestHandleObservation_InvalidJSON_Returns400(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.POST("/signals/events/observation", g.handleObservation)

	req := httptest.NewRequest(http.MethodPost, "/signals/events/observation",
		bytes.NewReader([]byte(`{not valid json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}
}

func TestHandleObservation_MissingRequiredField_Returns400(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.POST("/signals/events/observation", g.handleObservation)

	// patient_id is required but missing
	body := map[string]interface{}{
		"observation_code": "FBG",
		"value":            110.5,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/signals/events/observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}
}

func TestHandleTwinStateUpdate_Returns202(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.POST("/signals/events/twin-state-update", g.handleTwinStateUpdate)

	body := TwinStateUpdateEvent{
		PatientID:    "patient-001",
		StratumLabel: "T2DM",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/signals/events/twin-state-update", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %d", w.Code)
	}
}

func TestHandleCheckinResponse_Returns202(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.POST("/signals/events/checkin-response", g.handleCheckinResponse)

	body := CheckinResponseEvent{
		PatientID:    "patient-001",
		PromptID:     "PROMPT-01",
		Response:     3.0,
		StratumLabel: "T2DM",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/signals/events/checkin-response", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Node listing handler tests
// ---------------------------------------------------------------------------

func TestHandleListMonitoringNodes_Returns200(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()
	writeTestYAML(t, monDir, "pm01.yaml", minimalMonitoringNodeYAML)

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.GET("/signals/nodes/monitoring", g.handleListMonitoringNodes)

	req := httptest.NewRequest(http.MethodGet, "/signals/nodes/monitoring", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", w.Code, w.Body.String())
	}

	var list []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("response is not a JSON array: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 node in response, got %d", len(list))
	}
}

func TestHandleListDeteriorationNodes_Returns200(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()
	writeTestYAML(t, deterDir, "md01.yaml", minimalDeteriorationNodeYAML)

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.GET("/signals/nodes/deterioration", g.handleListDeteriorationNodes)

	req := httptest.NewRequest(http.MethodGet, "/signals/nodes/deterioration", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", w.Code, w.Body.String())
	}

	var list []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("response is not a JSON array: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 node in response, got %d", len(list))
	}
}

func TestHandleGetMonitoringNode_Returns200(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()
	writeTestYAML(t, monDir, "pm01.yaml", minimalMonitoringNodeYAML)

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.GET("/signals/nodes/monitoring/:nodeId", g.handleGetMonitoringNode)

	req := httptest.NewRequest(http.MethodGet, "/signals/nodes/monitoring/PM-01", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", w.Code, w.Body.String())
	}

	var node map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &node); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if node["node_id"] != "PM-01" {
		t.Errorf("expected node_id PM-01, got %v", node["node_id"])
	}
}

func TestHandleGetMonitoringNode_NotFound_Returns404(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.GET("/signals/nodes/monitoring/:nodeId", g.handleGetMonitoringNode)

	req := httptest.NewRequest(http.MethodGet, "/signals/nodes/monitoring/PM-DOES-NOT-EXIST", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestHandleGetDeteriorationNode_Returns200(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()
	writeTestYAML(t, deterDir, "md01.yaml", minimalDeteriorationNodeYAML)

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.GET("/signals/nodes/deterioration/:nodeId", g.handleGetDeteriorationNode)

	req := httptest.NewRequest(http.MethodGet, "/signals/nodes/deterioration/MD-01", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", w.Code, w.Body.String())
	}

	var node map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &node); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if node["node_id"] != "MD-01" {
		t.Errorf("expected node_id MD-01, got %v", node["node_id"])
	}
}

func TestHandleGetDeteriorationNode_NotFound_Returns404(t *testing.T) {
	monDir := t.TempDir()
	deterDir := t.TempDir()

	g := minimalSignalGroup(monDir, deterDir)

	router := gin.New()
	router.GET("/signals/nodes/deterioration/:nodeId", g.handleGetDeteriorationNode)

	req := httptest.NewRequest(http.MethodGet, "/signals/nodes/deterioration/MD-MISSING", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", w.Code)
	}
}
