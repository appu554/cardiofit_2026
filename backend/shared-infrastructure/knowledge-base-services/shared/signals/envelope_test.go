package signals

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEnvelope_KafkaTopic_Priority(t *testing.T) {
	env := ClinicalSignalEnvelope{
		EventID:  uuid.New(),
		Priority: true,
	}
	if env.KafkaTopic() != "clinical.priority-events.v1" {
		t.Errorf("expected priority topic, got %s", env.KafkaTopic())
	}
}

func TestEnvelope_KafkaTopic_Standard(t *testing.T) {
	env := ClinicalSignalEnvelope{
		EventID:  uuid.New(),
		Priority: false,
	}
	if env.KafkaTopic() != "clinical.observations.v1" {
		t.Errorf("expected observations topic, got %s", env.KafkaTopic())
	}
}

func TestEnvelope_JSON_RoundTrip(t *testing.T) {
	env := ClinicalSignalEnvelope{
		EventID:    uuid.New(),
		PatientID:  "patient-123",
		SignalType: SignalFBG,
		Priority:   false,
		MeasuredAt: time.Now().UTC(),
		Source:     SourceAppManual,
		Confidence: 1.0,
		LOINCCode:  "1558-6",
		Payload:    json.RawMessage(`{"value":5.5,"unit":"mmol/L"}`),
		CreatedAt:  time.Now().UTC(),
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ClinicalSignalEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.PatientID != env.PatientID {
		t.Error("patient_id mismatch")
	}
	if decoded.SignalType != SignalFBG {
		t.Error("signal_type mismatch")
	}
}

func TestStateChangeEnvelope_JSON(t *testing.T) {
	env := ClinicalStateChangeEnvelope{
		EventID:    uuid.New(),
		PatientID:  "patient-456",
		ChangeType: "MEDICATION_CHANGE",
		Timestamp:  time.Now().UTC(),
		Payload:    json.RawMessage(`{"drug":"metformin"}`),
		CreatedAt:  time.Now().UTC(),
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("empty JSON")
	}
}
