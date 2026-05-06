package fhir

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestObservationToAUObservation_VitalRoundTrip(t *testing.T) {
	val := 142.0
	in := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		LOINCCode:  "8480-6",
		Kind:       models.ObservationKindVital,
		Value:      &val,
		Unit:       "mmHg",
		ObservedAt: time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
	}
	fhir, err := ObservationToAUObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if fhir["resourceType"] != "Observation" {
		t.Errorf("resourceType: got %v want Observation", fhir["resourceType"])
	}
	b, _ := json.Marshal(fhir)
	var rt map[string]interface{}
	_ = json.Unmarshal(b, &rt)
	out, err := AUObservationToObservation(rt)
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.LOINCCode != in.LOINCCode || out.Kind != in.Kind {
		t.Errorf("round-trip drift: got %+v want %+v", out, in)
	}
	if out.Value == nil || *out.Value != *in.Value {
		t.Errorf("Value lost: got %v want %v", out.Value, in.Value)
	}
}

func TestObservationToAUObservation_LabRoundTrip(t *testing.T) {
	val := 7.4
	in := models.Observation{
		ID: uuid.New(), ResidentID: uuid.New(),
		LOINCCode: "4548-4", Kind: models.ObservationKindLab,
		Value: &val, Unit: "%", ObservedAt: time.Now().UTC(),
	}
	fhir, err := ObservationToAUObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	b, _ := json.Marshal(fhir)
	var rt map[string]interface{}
	_ = json.Unmarshal(b, &rt)
	out, err := AUObservationToObservation(rt)
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.Kind != models.ObservationKindLab {
		t.Errorf("Kind lost: got %q", out.Kind)
	}
}

func TestObservationToAUObservation_BehaviouralValueText(t *testing.T) {
	in := models.Observation{
		ID: uuid.New(), ResidentID: uuid.New(),
		Kind: models.ObservationKindBehavioural,
		ValueText: "agitation episode 14:30",
		ObservedAt: time.Now().UTC(),
	}
	fhir, err := ObservationToAUObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	b, _ := json.Marshal(fhir)
	var rt map[string]interface{}
	_ = json.Unmarshal(b, &rt)
	out, err := AUObservationToObservation(rt)
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.Value != nil {
		t.Errorf("Value should be nil for behavioural; got %v", *out.Value)
	}
	if out.ValueText != in.ValueText {
		t.Errorf("ValueText lost: got %q want %q", out.ValueText, in.ValueText)
	}
}

func TestObservationToAUObservation_MobilityRoundTrip(t *testing.T) {
	val := 4.0
	in := models.Observation{
		ID: uuid.New(), ResidentID: uuid.New(),
		Kind: models.ObservationKindMobility,
		Value: &val, Unit: "score",
		ObservedAt: time.Now().UTC(),
	}
	fhir, err := ObservationToAUObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	b, _ := json.Marshal(fhir)
	var rt map[string]interface{}
	_ = json.Unmarshal(b, &rt)
	out, _ := AUObservationToObservation(rt)
	if out.Kind != models.ObservationKindMobility {
		t.Errorf("Kind lost: got %q", out.Kind)
	}
}

func TestObservationToAUObservation_WeightRoundTrip(t *testing.T) {
	val := 78.4
	in := models.Observation{
		ID: uuid.New(), ResidentID: uuid.New(),
		Kind: models.ObservationKindWeight,
		Value: &val, Unit: "kg",
		ObservedAt: time.Now().UTC(),
	}
	fhir, err := ObservationToAUObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	b, _ := json.Marshal(fhir)
	var rt map[string]interface{}
	_ = json.Unmarshal(b, &rt)
	out, _ := AUObservationToObservation(rt)
	if out.Value == nil || *out.Value != val {
		t.Errorf("weight Value lost: got %v want %v", out.Value, val)
	}
}

func TestObservationToAUObservation_RejectsInvalid(t *testing.T) {
	bad := models.Observation{ID: uuid.New(), ResidentID: uuid.New(), Kind: "behavioral" /* US spelling */, ObservedAt: time.Now()}
	if _, err := ObservationToAUObservation(bad); err == nil {
		t.Errorf("expected egress validation error for invalid kind; got nil")
	}
}

func TestAUObservationToObservation_WrongResourceType(t *testing.T) {
	payload := map[string]interface{}{"resourceType": "Patient", "id": uuid.NewString()}
	if _, err := AUObservationToObservation(payload); err == nil {
		t.Errorf("expected error for resourceType=Patient; got nil")
	}
}

func TestObservationToAUObservation_WireFormatHasKindExtension(t *testing.T) {
	val := 7.0
	in := models.Observation{
		ID: uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		ResidentID: uuid.MustParse("99999999-8888-7777-6666-555555555555"),
		Kind: models.ObservationKindLab, LOINCCode: "4548-4",
		Value: &val, Unit: "%", ObservedAt: time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
	}
	fhir, err := ObservationToAUObservation(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	b, _ := json.Marshal(fhir)
	s := string(b)
	if !strings.Contains(s, ExtObservationKind) {
		t.Errorf("wire format missing ExtObservationKind URI; got: %s", s)
	}
	if !strings.Contains(s, `"resourceType":"Observation"`) {
		t.Errorf("wire format missing resourceType; got: %s", s)
	}
	if !strings.Contains(s, `"loinc_code"`) && !strings.Contains(s, `"4548-4"`) {
		t.Errorf("wire format missing LOINC code; got: %s", s)
	}
}
