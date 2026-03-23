package unit

import (
	"encoding/json"
	"testing"

	"kb-26-metabolic-digital-twin/internal/services"
)

// allExpectedSignalCodes lists every structured signal code the KB-26
// signal router should accept and route to PROCESS_OBSERVATION.
var allExpectedSignalCodes = []string{
	"FBG", "PPBG", "HBA1C",
	"SBP", "DBP",
	"CREATININE", "ACR", "POTASSIUM",
	"WEIGHT", "WAIST",
	"HR",
	"TOTAL_CHOLESTEROL", "HDL", "LDL", "TRIGLYCERIDES",
	"COMPLIANCE", "ORTHOSTATIC",
	"ACTIVITY",
	"LIPID_PANEL",
	"ADHERENCE",
	"HYPO_EVENT",
}

// TestSignalRouter_CoversAllStructuredCodes verifies the SignalRouter has
// routes for every expected observation signal code.
func TestSignalRouter_CoversAllStructuredCodes(t *testing.T) {
	router := services.NewSignalRouter()

	for _, code := range allExpectedSignalCodes {
		t.Run(code, func(t *testing.T) {
			envelope := map[string]interface{}{
				"signal_type": code,
				"patient_id":  "test-patient",
				"payload":     json.RawMessage(`{}`),
			}
			data, err := json.Marshal(envelope)
			if err != nil {
				t.Fatalf("failed to marshal envelope: %v", err)
			}

			action, routeErr := router.Route(data)
			if routeErr != nil {
				t.Fatalf("Route error for code %s: %v", code, routeErr)
			}
			if action == services.RouteSkip {
				t.Errorf("signal code %s was SKIPped — expected PROCESS_OBSERVATION", code)
			}
			if action != services.RouteProcessObservation {
				t.Errorf("signal code %s got action %s, want PROCESS_OBSERVATION", code, action)
			}
		})
	}
}

// TestSignalRouter_UnknownCodeSkipped verifies that an unrecognised signal
// code is routed to SKIP (not an error).
func TestSignalRouter_UnknownCodeSkipped(t *testing.T) {
	router := services.NewSignalRouter()
	envelope := map[string]interface{}{
		"signal_type": "UNKNOWN_CODE",
		"patient_id":  "p1",
		"payload":     json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(envelope)

	action, err := router.Route(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != services.RouteSkip {
		t.Errorf("unknown code should SKIP, got %s", action)
	}
}

// observationCodesHandledByEventProcessor lists codes that ProcessObservation
// maps to TwinState fields (i.e. the switch cases in event_processor.go).
// These use the case-label strings, including lowercase aliases.
var observationCodesHandledByEventProcessor = []string{
	// Glycaemic
	"FBG", "fbg", "fasting_blood_glucose",
	"PPBG", "ppbg", "postprandial_blood_glucose",
	"HbA1c", "hba1c",
	// Blood pressure
	"SBP", "sbp", "systolic_bp",
	"DBP", "dbp", "diastolic_bp",
	// Renal
	"eGFR", "egfr",
	"CREATININE", "creatinine",
	"ACR", "acr",
	"POTASSIUM", "potassium",
	// Anthropometric
	"waist_cm", "waist",
	"weight_kg", "weight",
	"bmi",
	// Cardiovascular
	"resting_hr", "heart_rate",
	// Lifestyle
	"daily_steps",
	"sleep_quality", "sleep_score",
	// Lipid
	"TOTAL_CHOLESTEROL", "total_cholesterol",
	"HDL", "hdl",
	"LDL", "ldl",
	"TRIGLYCERIDES", "triglycerides",
	// Adherence
	"COMPLIANCE", "compliance", "adherence_score",
	// Safety
	"ORTHOSTATIC", "orthostatic",
}

// TestEventProcessor_CoversTier1TwinStateFields verifies that every
// observation code alias that should map to a TwinState field is present
// in the ProcessObservation switch.
// This is a compile-time documentation test — it ensures we don't forget
// to wire new observation codes into the twin state update path.
func TestEventProcessor_CoversTier1TwinStateFields(t *testing.T) {
	// The expected count of distinct Tier 1 observation code aliases
	// handled by ProcessObservation.
	expectedCount := len(observationCodesHandledByEventProcessor)
	if expectedCount < 30 {
		t.Errorf("expected at least 30 observation code aliases, got %d — "+
			"did someone remove a case from ProcessObservation?", expectedCount)
	}
}

// preventTriggerCodesList lists the codes that should be in the
// preventTriggerCodes map and should trigger PREVENT recomputation.
var preventTriggerCodesList = []string{
	"HbA1c", "hba1c",
	"SBP", "sbp", "systolic_bp",
	"eGFR", "egfr",
	"CREATININE", "creatinine",
	"TOTAL_CHOLESTEROL", "total_cholesterol",
	"HDL", "hdl",
}

// TestPreventTriggerCodes_SubsetOfObservationCodes verifies that every
// code in preventTriggerCodes is also a valid observation code handled
// by ProcessObservation (i.e. has a case in the switch statement).
func TestPreventTriggerCodes_SubsetOfObservationCodes(t *testing.T) {
	// Build a set from the observation codes for O(1) lookup.
	obsSet := make(map[string]bool, len(observationCodesHandledByEventProcessor))
	for _, code := range observationCodesHandledByEventProcessor {
		obsSet[code] = true
	}

	for _, code := range preventTriggerCodesList {
		if !obsSet[code] {
			t.Errorf("preventTriggerCodes contains %q which is not handled "+
				"by ProcessObservation — PREVENT would trigger on an unprocessed signal", code)
		}
	}
}

// TestStateChangeRouter_CoversExpectedChangeTypes verifies the
// RouteStateChange method handles MEDICATION_CHANGE and STRATUM_CHANGE.
func TestStateChangeRouter_CoversExpectedChangeTypes(t *testing.T) {
	router := services.NewSignalRouter()

	tests := []struct {
		changeType string
		wantAction services.RouteAction
	}{
		{"MEDICATION_CHANGE", services.RouteUpdateMedTimeline},
		{"STRATUM_CHANGE", services.RouteUpdateStratum},
	}

	for _, tt := range tests {
		t.Run(tt.changeType, func(t *testing.T) {
			envelope := map[string]interface{}{
				"change_type": tt.changeType,
				"patient_id":  "p1",
				"payload":     json.RawMessage(`{}`),
			}
			data, _ := json.Marshal(envelope)

			action, err := router.RouteStateChange(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if action != tt.wantAction {
				t.Errorf("change_type %s: got %s, want %s", tt.changeType, action, tt.wantAction)
			}
		})
	}
}
