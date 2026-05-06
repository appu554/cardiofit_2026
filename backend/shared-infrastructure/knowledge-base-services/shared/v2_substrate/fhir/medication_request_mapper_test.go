package fhir

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestMedicineUseToMedicationRequestRoundTrip(t *testing.T) {
	bp := models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
	bpRaw, _ := json.Marshal(bp)

	in := models.MedicineUse{
		ID:           uuid.New(),
		ResidentID:   uuid.New(),
		AMTCode:      "12345",
		DisplayName:  "Perindopril 5mg",
		Intent:       models.Intent{Category: models.IntentTherapeutic, Indication: "essential hypertension"},
		Target:       models.Target{Kind: models.TargetKindBPThreshold, Spec: bpRaw},
		StopCriteria: models.StopCriteria{Triggers: []string{models.StopTriggerAdverseEvent}},
		Dose:         "5mg",
		Route:        "ORAL",
		Frequency:    "QD",
		StartedAt:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Status:       models.MedicineUseStatusActive,
	}

	mr, err := MedicineUseToAUMedicationRequest(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if mr["resourceType"] != "MedicationRequest" {
		t.Errorf("resourceType: got %v, want MedicationRequest", mr["resourceType"])
	}

	// Round-trip via JSON
	b, _ := json.Marshal(mr)
	var rt map[string]interface{}
	if err := json.Unmarshal(b, &rt); err != nil {
		t.Fatalf("rt unmarshal: %v", err)
	}

	out, err := AUMedicationRequestToMedicineUse(rt)
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}

	if out.AMTCode != in.AMTCode || out.DisplayName != in.DisplayName {
		t.Errorf("identity round-trip: got AMT=%q display=%q", out.AMTCode, out.DisplayName)
	}
	if out.Intent.Category != in.Intent.Category {
		t.Errorf("intent.category lost: got %q", out.Intent.Category)
	}
	if out.Target.Kind != in.Target.Kind {
		t.Errorf("target.kind lost: got %q", out.Target.Kind)
	}
	if string(out.Target.Spec) != string(in.Target.Spec) {
		t.Errorf("target.spec lost: got %s", string(out.Target.Spec))
	}
	if len(out.StopCriteria.Triggers) != 1 || out.StopCriteria.Triggers[0] != models.StopTriggerAdverseEvent {
		t.Errorf("stop_criteria.triggers lost: got %v", out.StopCriteria.Triggers)
	}
	if out.Status != models.MedicineUseStatusActive {
		t.Errorf("status: got %q", out.Status)
	}
}

func TestMedicineUseToMedicationRequest_RejectsInvalid(t *testing.T) {
	bad := models.MedicineUse{ID: uuid.New(), ResidentID: uuid.New(), DisplayName: ""}
	if _, err := MedicineUseToAUMedicationRequest(bad); err == nil {
		t.Errorf("expected validation rejection for missing DisplayName")
	}
}

func TestAUMedicationRequestToMedicineUse_WrongResourceType(t *testing.T) {
	in := map[string]interface{}{"resourceType": "Patient"}
	if _, err := AUMedicationRequestToMedicineUse(in); err == nil {
		t.Errorf("expected error for resourceType=Patient")
	}
}

func TestMedicineUseToMedicationRequest_AllTargetKindsRoundTrip(t *testing.T) {
	bpSpec, _ := json.Marshal(models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90})
	completionSpec, _ := json.Marshal(models.TargetCompletionDateSpec{
		EndDate: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), DurationDays: 7,
	})
	symptomSpec, _ := json.Marshal(models.TargetSymptomResolutionSpec{
		TargetSymptom: "pain", MonitoringWindowDays: 14,
	})
	hba1cSpec, _ := json.Marshal(models.TargetHbA1cBandSpec{Min: 6.5, Max: 8.0})
	openSpec, _ := json.Marshal(models.TargetOpenSpec{Rationale: "indefinite anticoagulation"})

	cases := []struct {
		name   string
		target models.Target
	}{
		{"BP_threshold", models.Target{Kind: models.TargetKindBPThreshold, Spec: bpSpec}},
		{"completion_date", models.Target{Kind: models.TargetKindCompletionDate, Spec: completionSpec}},
		{"symptom_resolution", models.Target{Kind: models.TargetKindSymptomResolution, Spec: symptomSpec}},
		{"HbA1c_band", models.Target{Kind: models.TargetKindHbA1cBand, Spec: hba1cSpec}},
		{"open", models.Target{Kind: models.TargetKindOpen, Spec: openSpec}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := models.MedicineUse{
				ID: uuid.New(), ResidentID: uuid.New(),
				DisplayName:  "test " + c.name,
				Intent:       models.Intent{Category: models.IntentTherapeutic, Indication: "x"},
				Target:       c.target,
				StopCriteria: models.StopCriteria{Triggers: []string{models.StopTriggerReviewDue}},
				StartedAt:    time.Now(), Status: models.MedicineUseStatusActive,
			}
			mr, err := MedicineUseToAUMedicationRequest(in)
			if err != nil {
				t.Fatalf("egress: %v", err)
			}
			b, err := json.Marshal(mr)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var rt map[string]interface{}
			if err := json.Unmarshal(b, &rt); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			out, err := AUMedicationRequestToMedicineUse(rt)
			if err != nil {
				t.Fatalf("ingress: %v", err)
			}
			if out.Target.Kind != in.Target.Kind {
				t.Errorf("kind: got %q want %q", out.Target.Kind, in.Target.Kind)
			}
			if string(out.Target.Spec) != string(in.Target.Spec) {
				t.Errorf("spec opacity broken for %s: got %s want %s", c.name, string(out.Target.Spec), string(in.Target.Spec))
			}
		})
	}
}

func TestMedicineUseToMedicationRequest_WireFormat(t *testing.T) {
	// Pin the FHIR shape directly (not just via round-trip).
	bp := models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
	bpRaw, _ := json.Marshal(bp)
	in := models.MedicineUse{
		ID:           uuid.New(),
		ResidentID:   uuid.New(),
		AMTCode:      "12345",
		DisplayName:  "Perindopril 5mg",
		Intent:       models.Intent{Category: models.IntentTherapeutic, Indication: "essential hypertension"},
		Target:       models.Target{Kind: models.TargetKindBPThreshold, Spec: bpRaw},
		StopCriteria: models.StopCriteria{Triggers: []string{models.StopTriggerAdverseEvent}},
		StartedAt:    time.Now(), Status: models.MedicineUseStatusActive,
	}
	mr, err := MedicineUseToAUMedicationRequest(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}

	// status field should be lowercase per FHIR
	if mr["status"] != "active" {
		t.Errorf("status: got %v want active", mr["status"])
	}
	// intent extension must be present
	exts, ok := mr["extension"].([]map[string]interface{})
	if !ok {
		t.Fatalf("extension array not present: %T", mr["extension"])
	}
	foundIntent := false
	for _, ext := range exts {
		if ext["url"] == ExtMedicineIntent {
			foundIntent = true
			if _, ok := ext["valueString"]; !ok {
				t.Errorf("intent extension missing valueString")
			}
		}
	}
	if !foundIntent {
		t.Errorf("intent extension not found in wire format")
	}
}
