package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMedicineUseJSONRoundTrip(t *testing.T) {
	prescriber := uuid.New()
	bpSpec := TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
	bpSpecRaw, _ := json.Marshal(bpSpec)

	in := MedicineUse{
		ID:          uuid.New(),
		ResidentID:  uuid.New(),
		AMTCode:     "12345",
		DisplayName: "Perindopril 5mg",
		Intent: Intent{
			Category:   IntentTherapeutic,
			Indication: "essential hypertension",
		},
		Target: Target{
			Kind: TargetKindBPThreshold,
			Spec: bpSpecRaw,
		},
		StopCriteria: StopCriteria{
			Triggers: []string{StopTriggerAdverseEvent, StopTriggerReviewDue},
		},
		Dose:         "5mg",
		Route:        "ORAL",
		Frequency:    "QD",
		PrescriberID: &prescriber,
		StartedAt:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Status:       MedicineUseStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out MedicineUse
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.ID != in.ID || out.AMTCode != in.AMTCode || out.DisplayName != in.DisplayName {
		t.Errorf("identity round-trip mismatch")
	}
	if out.Intent.Category != IntentTherapeutic {
		t.Errorf("intent.category: got %q", out.Intent.Category)
	}
	if out.Target.Kind != TargetKindBPThreshold {
		t.Errorf("target.kind: got %q", out.Target.Kind)
	}
	if len(out.StopCriteria.Triggers) != 2 {
		t.Errorf("stop_criteria.triggers count: got %d", len(out.StopCriteria.Triggers))
	}
	if out.PrescriberID == nil || *out.PrescriberID != prescriber {
		t.Errorf("prescriber_id round-trip lost")
	}
}

func TestMedicineUseOptionalFields(t *testing.T) {
	in := MedicineUse{
		ID:           uuid.New(),
		ResidentID:   uuid.New(),
		DisplayName:  "Test",
		Intent:       Intent{Category: IntentTherapeutic, Indication: "x"},
		Target:       Target{Kind: TargetKindOpen, Spec: json.RawMessage(`{}`)},
		StopCriteria: StopCriteria{Triggers: []string{}},
		StartedAt:    time.Now(),
		Status:       MedicineUseStatusActive,
	}
	b, _ := json.Marshal(in)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	if _, present := m["amt_code"]; present {
		t.Errorf("amt_code should be omitted when empty")
	}
	if _, present := m["ended_at"]; present {
		t.Errorf("ended_at should be omitted when nil")
	}
	if _, present := m["prescriber_id"]; present {
		t.Errorf("prescriber_id should be omitted when nil")
	}
}

func TestTargetSpecOpaqueMarshalling(t *testing.T) {
	// Target.Spec is json.RawMessage; the model layer treats it opaquely.
	// This test pins down that Target.Spec is preserved byte-for-byte through
	// a round-trip — critical for the cross-KB JSONB contract.
	inputSpec := json.RawMessage(`{"systolic_max":140,"diastolic_max":90}`)
	in := Target{Kind: TargetKindBPThreshold, Spec: inputSpec}
	b, _ := json.Marshal(in)
	var out Target
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(out.Spec) != string(inputSpec) {
		t.Errorf("spec opacity broken: got %s want %s", string(out.Spec), string(inputSpec))
	}
}
