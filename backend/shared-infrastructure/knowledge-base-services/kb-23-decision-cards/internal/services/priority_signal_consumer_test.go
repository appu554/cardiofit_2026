package services

import (
	"encoding/json"
	"testing"
)

func TestPrioritySignalRouter_Route(t *testing.T) {
	router := &PrioritySignalRouter{}

	tests := []struct {
		name     string
		input    priorityEnvelope
		expected PriorityRouteAction
		wantErr  bool
	}{
		{
			name:     "HYPO_EVENT routes to RouteHypo",
			input:    priorityEnvelope{SignalType: "HYPO_EVENT", PatientID: "p1"},
			expected: RouteHypo,
		},
		{
			name:     "ORTHOSTATIC routes to RouteOrthostatic",
			input:    priorityEnvelope{SignalType: "ORTHOSTATIC", PatientID: "p1"},
			expected: RouteOrthostatic,
		},
		{
			name:     "POTASSIUM with priority flag routes to RoutePotassium",
			input:    priorityEnvelope{SignalType: "POTASSIUM", PatientID: "p1", Priority: true},
			expected: RoutePotassium,
		},
		{
			name:     "POTASSIUM without priority flag routes to skip",
			input:    priorityEnvelope{SignalType: "POTASSIUM", PatientID: "p1", Priority: false},
			expected: RoutePrioritySkip,
		},
		{
			name:     "ADVERSE_EVENT routes to RouteAdverseEvent",
			input:    priorityEnvelope{SignalType: "ADVERSE_EVENT", PatientID: "p1"},
			expected: RouteAdverseEvent,
		},
		{
			name:     "HOSPITALISATION routes to RouteHospitalisation",
			input:    priorityEnvelope{SignalType: "HOSPITALISATION", PatientID: "p1"},
			expected: RouteHospitalisation,
		},
		{
			// Phase 6 P6-6 Decision 9: CKM stage transitions route to
			// RouteCKMTransition; the handler internally filters for 4c.
			name:     "CKM_STAGE_TRANSITION routes to RouteCKMTransition",
			input:    priorityEnvelope{SignalType: "CKM_STAGE_TRANSITION", PatientID: "p1"},
			expected: RouteCKMTransition,
		},
		{
			name:     "unknown signal type routes to skip",
			input:    priorityEnvelope{SignalType: "UNKNOWN_SIGNAL", PatientID: "p1"},
			expected: RoutePrioritySkip,
		},
		{
			name:     "empty signal type routes to skip",
			input:    priorityEnvelope{SignalType: "", PatientID: "p1"},
			expected: RoutePrioritySkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal input: %v", err)
			}

			action, err := router.Route(data)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if action != tt.expected {
				t.Errorf("got action %q, want %q", action, tt.expected)
			}
		})
	}
}

func TestPrioritySignalRouter_Route_InvalidJSON(t *testing.T) {
	router := &PrioritySignalRouter{}

	_, err := router.Route([]byte("not valid json"))
	if err == nil {
		t.Error("expected error for invalid JSON but got nil")
	}
}

func TestPrioritySignalRouter_Route_WithPayload(t *testing.T) {
	router := &PrioritySignalRouter{}

	// Verify that the router correctly handles messages with payload data
	env := priorityEnvelope{
		SignalType: "ORTHOSTATIC",
		PatientID:  "patient-123",
		Priority:   true,
		Payload:    json.RawMessage(`{"sbp_drop": 25, "dbp_drop": 10, "standing_sbp": 90}`),
	}

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	action, err := router.Route(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != RouteOrthostatic {
		t.Errorf("got action %q, want %q", action, RouteOrthostatic)
	}
}
