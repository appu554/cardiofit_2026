package asha

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestHandleBatchSubmit_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	handler := NewHandler(nil, nil, logger)

	sub := TabletSubmission{
		DeviceID:    "ASHA-TAB-001",
		AshaID:      uuid.New(),
		PatientID:   uuid.New(),
		TenantID:    uuid.New(),
		CollectedAt: time.Now().UTC(),
		SyncSeqNo:   1,
		IsOffline:   false,
		Slots: []SlotEntry{
			{SlotName: "fbg", Domain: "glycemic", Value: 145.0, Unit: "mg/dL", Method: "glucometer"},
			{SlotName: "sbp", Domain: "cardiac", Value: 138.0, Unit: "mmHg", Method: "bp_monitor"},
		},
	}

	body, _ := json.Marshal(sub)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/channel/asha/submit", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleBatchSubmit(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Results []SubmissionResult `json:"results"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	for _, r := range resp.Results {
		if r.Status != "ACCEPTED" {
			t.Errorf("expected ACCEPTED for %s, got %s", r.SlotName, r.Status)
		}
	}
}

func TestHandleBatchSubmit_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	handler := NewHandler(nil, nil, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/channel/asha/submit",
		bytes.NewReader([]byte(`{"device_id":""}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleBatchSubmit(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTabletSubmission_GPSLocation(t *testing.T) {
	sub := TabletSubmission{
		GPSLocation: &GPSLocation{
			Latitude:  19.0760,
			Longitude: 72.8777,
			Accuracy:  10.5,
		},
	}
	if sub.GPSLocation.Latitude != 19.0760 {
		t.Errorf("expected latitude 19.0760, got %f", sub.GPSLocation.Latitude)
	}
}
