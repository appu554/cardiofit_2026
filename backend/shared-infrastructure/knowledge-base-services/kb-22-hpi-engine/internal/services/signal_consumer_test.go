package services

import (
	"encoding/json"
	"testing"
)

func TestKB22RouteSignal_FBG_ToObservation(t *testing.T) {
	router := NewKB22SignalRouter()
	envelope := map[string]interface{}{
		"signal_type": "FBG",
		"patient_id":  "p1",
		"payload":     json.RawMessage(`{"lab_type":"FBG","value":5.5}`),
	}
	data, _ := json.Marshal(envelope)
	action, err := router.Route(data)
	if err != nil {
		t.Fatal(err)
	}
	if action != KB22RouteObservation {
		t.Errorf("expected KB22RouteObservation, got %s", action)
	}
}

func TestKB22RouteSignal_Symptom_ToHPI(t *testing.T) {
	router := NewKB22SignalRouter()
	envelope := map[string]interface{}{
		"signal_type": "SYMPTOM",
		"patient_id":  "p1",
		"payload":     json.RawMessage(`{"symptom":"headache"}`),
	}
	data, _ := json.Marshal(envelope)
	action, err := router.Route(data)
	if err != nil {
		t.Fatal(err)
	}
	if action != KB22RouteHPISession {
		t.Errorf("expected KB22RouteHPISession, got %s", action)
	}
}
