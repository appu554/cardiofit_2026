package services

import (
	"encoding/json"
	"testing"
)

func TestRouteSignal_FBG(t *testing.T) {
	router := NewSignalRouter()
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
	if action != RouteProcessObservation {
		t.Errorf("expected RouteProcessObservation, got %s", action)
	}
}

func TestRouteSignal_MedicationChange(t *testing.T) {
	router := NewSignalRouter()
	envelope := map[string]interface{}{
		"change_type": "MEDICATION_CHANGE",
		"patient_id":  "p1",
		"payload":     json.RawMessage(`{"drug_name":"metformin"}`),
	}
	data, _ := json.Marshal(envelope)
	action, err := router.RouteStateChange(data)
	if err != nil {
		t.Fatal(err)
	}
	if action != RouteUpdateMedTimeline {
		t.Errorf("expected RouteUpdateMedTimeline, got %s", action)
	}
}
