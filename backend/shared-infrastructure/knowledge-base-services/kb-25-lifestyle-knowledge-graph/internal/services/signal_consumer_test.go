package services

import (
	"encoding/json"
	"testing"
)

func TestSignalRouter_Route(t *testing.T) {
	router := NewSignalRouter()

	tests := []struct {
		name       string
		input      signalEnvelope
		wantAction RouteAction
		wantErr    bool
	}{
		{
			name:       "MEAL_LOG routes to MATCH_FOOD",
			input:      signalEnvelope{SignalType: "MEAL_LOG", PatientID: "p1", Payload: json.RawMessage(`{}`)},
			wantAction: RouteMatchFood,
		},
		{
			name:       "ACTIVITY routes to MATCH_EXERCISE",
			input:      signalEnvelope{SignalType: "ACTIVITY", PatientID: "p2", Payload: json.RawMessage(`{}`)},
			wantAction: RouteMatchExercise,
		},
		{
			name:       "WEIGHT routes to UPDATE_WEIGHT",
			input:      signalEnvelope{SignalType: "WEIGHT", PatientID: "p3", Payload: json.RawMessage(`{}`)},
			wantAction: RouteUpdateWeight,
		},
		{
			name:       "WAIST routes to UPDATE_WAIST",
			input:      signalEnvelope{SignalType: "WAIST", PatientID: "p4", Payload: json.RawMessage(`{}`)},
			wantAction: RouteUpdateWaist,
		},
		{
			name:       "unknown signal routes to SKIP",
			input:      signalEnvelope{SignalType: "BLOOD_PRESSURE", PatientID: "p5", Payload: json.RawMessage(`{}`)},
			wantAction: RouteSkip,
		},
		{
			name:       "empty signal type routes to SKIP",
			input:      signalEnvelope{SignalType: "", PatientID: "p6", Payload: json.RawMessage(`{}`)},
			wantAction: RouteSkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal test input: %v", err)
			}

			action, patientID, _, routeErr := router.Route(data)
			if tt.wantErr {
				if routeErr == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if routeErr != nil {
				t.Fatalf("unexpected error: %v", routeErr)
			}
			if action != tt.wantAction {
				t.Errorf("action = %q, want %q", action, tt.wantAction)
			}
			if patientID != tt.input.PatientID {
				t.Errorf("patientID = %q, want %q", patientID, tt.input.PatientID)
			}
		})
	}
}

func TestSignalRouter_Route_InvalidJSON(t *testing.T) {
	router := NewSignalRouter()

	_, _, _, err := router.Route([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
