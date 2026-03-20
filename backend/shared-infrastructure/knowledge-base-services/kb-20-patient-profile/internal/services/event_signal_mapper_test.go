package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"kb-patient-profile/internal/models"
)

func TestMapLabResult_FBG(t *testing.T) {
	mapper := NewEventSignalMapper()
	payload, _ := json.Marshal(models.LabResultPayload{
		LabType: "FBG", Value: 5.5, Unit: "mmol/L",
		MeasuredAt: time.Now().Format(time.RFC3339),
	})
	entry := models.EventOutboxEntry{
		ID:        uuid.New(),
		EventType: models.EventLabResult,
		PatientID: "p1",
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	result, err := mapper.Map(entry)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for FBG lab")
	}
	if result.Signal != nil && string(result.Signal.SignalType) != "FBG" {
		t.Errorf("expected FBG signal type, got %s", result.Signal.SignalType)
	}
}

func TestMapLabResult_UnknownLabType(t *testing.T) {
	mapper := NewEventSignalMapper()
	payload, _ := json.Marshal(models.LabResultPayload{
		LabType: "UNKNOWN_LAB", Value: 1.0,
	})
	entry := models.EventOutboxEntry{
		ID:        uuid.New(),
		EventType: models.EventLabResult,
		PatientID: "p1",
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	result, err := mapper.Map(entry)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Error("expected nil result for unknown lab type")
	}
}

func TestMapMedicationChange(t *testing.T) {
	mapper := NewEventSignalMapper()
	payload, _ := json.Marshal(models.MedicationChangePayload{
		ChangeType: "ADD", DrugName: "metformin", DrugClass: "BIGUANIDE",
	})
	entry := models.EventOutboxEntry{
		ID:        uuid.New(),
		EventType: models.EventMedicationChange,
		PatientID: "p1",
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	result, err := mapper.Map(entry)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.StateChange == nil {
		t.Fatal("expected state change result for medication change")
	}
	if result.StateChange.ChangeType != "MEDICATION_CHANGE" {
		t.Errorf("expected MEDICATION_CHANGE, got %s", result.StateChange.ChangeType)
	}
}

func TestMapOrthostaticAlert(t *testing.T) {
	mapper := NewEventSignalMapper()
	payload, _ := json.Marshal(models.OrthostaticAlertPayload{
		PatientID: "p1", OrthostaticDrop: -25.0,
	})
	entry := models.EventOutboxEntry{
		ID:        uuid.New(),
		EventType: models.EventOrthostaticAlert,
		PatientID: "p1",
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	result, err := mapper.Map(entry)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Signal == nil {
		t.Fatal("expected signal for orthostatic alert")
	}
	if !result.Signal.Priority {
		t.Error("expected priority flag for orthostatic alert")
	}
}
