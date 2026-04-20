package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestGetAttributionByPatient_NoDB_ReturnsEmptyList(t *testing.T) {
	r := newTestEngine()
	srv := &Server{}
	r.GET("/attribution/:patientId", srv.getAttributionByPatient)

	req := httptest.NewRequest(http.MethodGet, "/attribution/P-nobody", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	verdicts, _ := resp["verdicts"].([]interface{})
	if len(verdicts) != 0 {
		t.Fatalf("expected empty verdicts with no DB, got %d", len(verdicts))
	}
	if resp["total"].(float64) != 0 {
		t.Fatalf("expected total=0, got %v", resp["total"])
	}
}

func TestGetAttributionByPatient_EmptyPatientID_Returns400(t *testing.T) {
	r := newTestEngine()
	srv := &Server{}
	r.GET("/attribution/:patientId", srv.getAttributionByPatient)

	// A URL-encoded single-space param → trimmed to empty → handler must reject with 400.
	req := httptest.NewRequest(http.MethodGet, "/attribution/%20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty patient_id, got %d", w.Code)
	}
}

func TestGetAttributionByPatient_LimitQueryParam_IsHonoured(t *testing.T) {
	r := newTestEngine()
	srv := &Server{}
	r.GET("/attribution/:patientId", srv.getAttributionByPatient)

	req := httptest.NewRequest(http.MethodGet, "/attribution/P-001?limit=25", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["limit"].(float64) != 25 {
		t.Fatalf("expected limit=25 in response, got %v", resp["limit"])
	}
}

func TestRunAttribution_NilLedger_Returns503(t *testing.T) {
	r := newTestEngine()
	srv := &Server{} // ledger deliberately nil
	r.POST("/attribution/run", srv.runAttribution)

	body := map[string]interface{}{
		"TreatmentStrategy": "INTERVENTION_TAKEN",
		"PreAlertRiskTier":  "HIGH",
		"PreAlertRiskScore": 62.0,
		"HorizonDays":       30,
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/attribution/run", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 with nil ledger, got %d: %s", w.Code, w.Body.String())
	}
}
