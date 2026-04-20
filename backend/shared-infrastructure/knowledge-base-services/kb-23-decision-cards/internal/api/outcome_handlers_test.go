package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"kb-23-decision-cards/internal/models"
)

func newTestGinEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestIngestOutcome_SingleSource_ReturnsResolved(t *testing.T) {
	r := newTestGinEngine()
	srv := &Server{}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	body := models.OutcomeRecord{
		PatientID:       "P-ing-001",
		LifecycleID:     &lifecycleID,
		CohortID:        "hcf_catalyst_chf",
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
		IngestedAt:      time.Now(),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	rec, _ := resp["record"].(map[string]interface{})
	if rec == nil {
		t.Fatalf("response missing 'record' object: %s", w.Body.String())
	}
	if rec["reconciliation"] != "RESOLVED" {
		t.Fatalf("expected reconciliation=RESOLVED, got %v", rec["reconciliation"])
	}
}

func TestIngestOutcome_MissingPatientID_Returns400(t *testing.T) {
	r := newTestGinEngine()
	srv := &Server{}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	body := models.OutcomeRecord{
		OutcomeType: "READMISSION_30D",
		Source:      string(models.OutcomeSourceHospitalDischarge),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing patient_id, got %d", w.Code)
	}
}

func TestIngestOutcome_InvalidJSON_Returns400(t *testing.T) {
	r := newTestGinEngine()
	srv := &Server{}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader([]byte(`{"bad json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", w.Code)
	}
}
