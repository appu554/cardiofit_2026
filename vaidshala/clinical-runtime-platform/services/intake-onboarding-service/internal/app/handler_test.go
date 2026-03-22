package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestFillSlotRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		body    FillSlotRequest
		wantErr bool
	}{
		{
			name: "valid request",
			body: FillSlotRequest{
				SlotName:       "fbg",
				Value:          json.RawMessage(`178`),
				ExtractionMode: "BUTTON",
				Confidence:     1.0,
				SourceChannel:  "APP",
			},
			wantErr: false,
		},
		{
			name: "missing slot_name",
			body: FillSlotRequest{
				Value:          json.RawMessage(`178`),
				ExtractionMode: "BUTTON",
				Confidence:     1.0,
				SourceChannel:  "APP",
			},
			wantErr: true,
		},
		{
			name: "unknown slot_name",
			body: FillSlotRequest{
				SlotName:       "nonexistent_slot",
				Value:          json.RawMessage(`178`),
				ExtractionMode: "BUTTON",
				Confidence:     1.0,
				SourceChannel:  "APP",
			},
			wantErr: true,
		},
		{
			name: "missing value",
			body: FillSlotRequest{
				SlotName:       "fbg",
				ExtractionMode: "BUTTON",
				Confidence:     1.0,
				SourceChannel:  "APP",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.body.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFillSlotResponse_Structure(t *testing.T) {
	resp := FillSlotResponse{
		Status:         "ok",
		SlotName:       "fbg",
		FHIRResourceID: "obs-123",
		SafetyResult: &SafetyResultResponse{
			HardStops: []RuleResultResponse{},
			SoftFlags: []RuleResultResponse{
				{RuleID: "SF-01", Reason: "Elderly"},
			},
		},
		NextNode: &NextNodeResponse{
			NodeID: "glycemic",
			Slots:  []string{"hba1c", "ppbg"},
		},
		Progress: ProgressResponse{
			Filled:   5,
			Total:    50,
			Percent:  10.0,
			Complete: false,
		},
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(raw, &decoded)
	if decoded["status"] != "ok" {
		t.Errorf("expected status=ok")
	}
	if decoded["slot_name"] != "fbg" {
		t.Errorf("expected slot_name=fbg")
	}
}

func TestFillSlotHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	h := &Handler{logger: testLogger()}
	router.POST("/fhir/Encounter/:id/$fill-slot", h.HandleFillSlot)

	encID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/fhir/Encounter/"+encID+"/$fill-slot", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
