package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// spyEventPublisher records Publish calls for verification.
type spyEventPublisher struct {
	calls []publishCall
}

type publishCall struct {
	eventType string
	patientID string
	payload   interface{}
}

func (s *spyEventPublisher) Publish(eventType, patientID string, payload interface{}) {
	s.calls = append(s.calls, publishCall{eventType, patientID, payload})
}

// newTestSignalServer creates a minimal Server with only the fields needed
// for signal handler tests: an EventPublisher spy and a logger.
func newTestSignalServer() (*Server, *spyEventPublisher) {
	gin.SetMode(gin.TestMode)
	spy := &spyEventPublisher{}
	logger, _ := zap.NewDevelopment()
	s := &Server{
		Router:   gin.New(),
		eventBus: spy,
		logger:   logger,
	}
	// Register only signal routes for testing
	patient := s.Router.Group("/api/v1/patient")
	signals := patient.Group("/:id/signals")
	{
		signals.POST("/meal", s.submitMealSignal)
		signals.POST("/activity", s.submitActivitySignal)
		signals.POST("/waist", s.submitWaistSignal)
		signals.POST("/adherence", s.submitAdherenceSignal)
		signals.POST("/symptom", s.submitSymptomSignal)
		signals.POST("/adverse-event", s.submitAdverseEventSignal)
		signals.POST("/resolution", s.submitResolutionSignal)
		signals.POST("/hospitalisation", s.submitHospitalisationSignal)
	}
	return s, spy
}

func TestSignalEndpoints_HappyPath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		body          map[string]interface{}
		wantSignal    string
		wantPatientID string
	}{
		{
			name: "meal signal",
			path: "/api/v1/patient/p-001/signals/meal",
			body: map[string]interface{}{
				"meal_type":   "breakfast",
				"carbs_g":     45.0,
				"measured_at": "2026-03-20T08:00:00Z",
			},
			wantSignal:    "MEAL_LOG",
			wantPatientID: "p-001",
		},
		{
			name: "activity signal",
			path: "/api/v1/patient/p-002/signals/activity",
			body: map[string]interface{}{
				"step_count":  8500,
				"intensity":   "moderate",
				"measured_at": "2026-03-20T07:30:00Z",
			},
			wantSignal:    "ACTIVITY_LOG",
			wantPatientID: "p-002",
		},
		{
			name: "waist signal",
			path: "/api/v1/patient/p-003/signals/waist",
			body: map[string]interface{}{
				"value_cm":    92.5,
				"measured_at": "2026-03-20T09:00:00Z",
			},
			wantSignal:    "WAIST_MEASUREMENT",
			wantPatientID: "p-003",
		},
		{
			name: "adherence signal",
			path: "/api/v1/patient/p-004/signals/adherence",
			body: map[string]interface{}{
				"drug_class":  "ACEi",
				"taken":       true,
				"measured_at": "2026-03-20T10:00:00Z",
			},
			wantSignal:    "ADHERENCE_REPORT",
			wantPatientID: "p-004",
		},
		{
			name: "symptom signal",
			path: "/api/v1/patient/p-005/signals/symptom",
			body: map[string]interface{}{
				"symptom_code": "dizziness",
				"severity":     5,
				"measured_at":  "2026-03-20T11:00:00Z",
			},
			wantSignal:    "SYMPTOM_REPORT",
			wantPatientID: "p-005",
		},
		{
			name: "adverse event signal",
			path: "/api/v1/patient/p-006/signals/adverse-event",
			body: map[string]interface{}{
				"drug_class": "SGLT2i",
				"event_type": "gi_upset",
				"severity":   "moderate",
				"onset_at":   "2026-03-19T14:00:00Z",
			},
			wantSignal:    "ADVERSE_EVENT",
			wantPatientID: "p-006",
		},
		{
			name: "resolution signal",
			path: "/api/v1/patient/p-007/signals/resolution",
			body: map[string]interface{}{
				"original_event_type": "SYMPTOM",
				"resolution":          "resolved",
				"resolved_at":         "2026-03-20T12:00:00Z",
			},
			wantSignal:    "RESOLUTION_REPORT",
			wantPatientID: "p-007",
		},
		{
			name: "hospitalisation signal",
			path: "/api/v1/patient/p-008/signals/hospitalisation",
			body: map[string]interface{}{
				"reason":      "acute kidney injury",
				"facility":    "City Hospital",
				"admitted_at": "2026-03-18T06:00:00Z",
			},
			wantSignal:    "HOSPITALISATION",
			wantPatientID: "p-008",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, spy := newTestSignalServer()

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			s.Router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
			}

			// Verify EventBus was called
			if len(spy.calls) != 1 {
				t.Fatalf("expected 1 Publish call, got %d", len(spy.calls))
			}
			if spy.calls[0].eventType != tt.wantSignal {
				t.Errorf("expected event type %q, got %q", tt.wantSignal, spy.calls[0].eventType)
			}
			if spy.calls[0].patientID != tt.wantPatientID {
				t.Errorf("expected patient ID %q, got %q", tt.wantPatientID, spy.calls[0].patientID)
			}

			// Verify response body
			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if resp["success"] != true {
				t.Error("expected success=true in response")
			}
			if resp["signal"] != tt.wantSignal {
				t.Errorf("expected signal=%q in response, got %q", tt.wantSignal, resp["signal"])
			}
		})
	}
}

func TestSignalEndpoints_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		path string
		body map[string]interface{}
	}{
		{
			name: "meal missing meal_type",
			path: "/api/v1/patient/p-001/signals/meal",
			body: map[string]interface{}{
				"measured_at": "2026-03-20T08:00:00Z",
			},
		},
		{
			name: "waist missing value_cm",
			path: "/api/v1/patient/p-001/signals/waist",
			body: map[string]interface{}{
				"measured_at": "2026-03-20T08:00:00Z",
			},
		},
		{
			name: "adherence missing drug_class",
			path: "/api/v1/patient/p-001/signals/adherence",
			body: map[string]interface{}{
				"taken":       true,
				"measured_at": "2026-03-20T08:00:00Z",
			},
		},
		{
			name: "symptom missing severity",
			path: "/api/v1/patient/p-001/signals/symptom",
			body: map[string]interface{}{
				"symptom_code": "headache",
				"measured_at":  "2026-03-20T08:00:00Z",
			},
		},
		{
			name: "adverse event missing drug_class",
			path: "/api/v1/patient/p-001/signals/adverse-event",
			body: map[string]interface{}{
				"event_type": "rash",
				"severity":   "mild",
				"onset_at":   "2026-03-20T08:00:00Z",
			},
		},
		{
			name: "resolution missing resolution field",
			path: "/api/v1/patient/p-001/signals/resolution",
			body: map[string]interface{}{
				"original_event_type": "SYMPTOM",
				"resolved_at":         "2026-03-20T08:00:00Z",
			},
		},
		{
			name: "hospitalisation missing reason",
			path: "/api/v1/patient/p-001/signals/hospitalisation",
			body: map[string]interface{}{
				"admitted_at": "2026-03-20T08:00:00Z",
			},
		},
		{
			name: "invalid JSON",
			path: "/api/v1/patient/p-001/signals/meal",
			body: nil, // will be sent as "null"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, spy := newTestSignalServer()

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			s.Router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}

			// EventBus should NOT have been called
			if len(spy.calls) != 0 {
				t.Errorf("expected 0 Publish calls on bad request, got %d", len(spy.calls))
			}
		})
	}
}
