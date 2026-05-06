package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestObservationJSONRoundTripVital(t *testing.T) {
	val := 142.0
	src := uuid.New()
	in := Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		LOINCCode:  "8480-6",
		Kind:       ObservationKindVital,
		Value:      &val,
		Unit:       "mmHg",
		ObservedAt: time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
		SourceID:   &src,
		CreatedAt:  time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Observation
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID != in.ID || out.Kind != in.Kind || out.LOINCCode != in.LOINCCode {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
	if out.Value == nil || *out.Value != *in.Value {
		t.Errorf("Value pointer round-trip lost: got %v want %v", out.Value, in.Value)
	}
	if out.SourceID == nil || *out.SourceID != src {
		t.Errorf("SourceID round-trip lost: got %v want %v", out.SourceID, src)
	}
}

func TestObservationJSONRoundTripBehaviouralValueText(t *testing.T) {
	in := Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       ObservationKindBehavioural,
		Value:      nil, // intentionally nil — ValueText carries the data
		ValueText:  "agitation episode 14:30, paced corridor 22 minutes",
		ObservedAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), `"value":`) && !strings.Contains(string(b), `"value":null`) {
		// omitempty on *float64 nil should drop the key entirely
		t.Errorf("value should be omitted when nil, got: %s", string(b))
	}
	var out Observation
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Value != nil {
		t.Errorf("Value should remain nil after round-trip, got %v", *out.Value)
	}
	if out.ValueText != in.ValueText {
		t.Errorf("ValueText round-trip lost: got %q want %q", out.ValueText, in.ValueText)
	}
}

func TestObservationDeltaRoundTrip(t *testing.T) {
	val := 8.2
	in := Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       ObservationKindLab,
		LOINCCode:  "4548-4",
		Value:      &val,
		Unit:       "%",
		ObservedAt: time.Now().UTC(),
		Delta: &Delta{
			BaselineValue:   7.0,
			DeviationStdDev: 2.4,
			DirectionalFlag: DeltaFlagSeverelyElevated,
			ComputedAt:      time.Now().UTC(),
		},
		CreatedAt: time.Now().UTC(),
	}
	b, _ := json.Marshal(in)
	var out Observation
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Delta == nil {
		t.Fatalf("Delta lost in round-trip")
	}
	if out.Delta.DirectionalFlag != DeltaFlagSeverelyElevated {
		t.Errorf("Delta.DirectionalFlag: got %q want %q", out.Delta.DirectionalFlag, DeltaFlagSeverelyElevated)
	}
	if out.Delta.BaselineValue != 7.0 || out.Delta.DeviationStdDev != 2.4 {
		t.Errorf("Delta numeric fields drifted: %+v", out.Delta)
	}
}

func TestObservationOmitsEmptyOptionalFields(t *testing.T) {
	in := Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       ObservationKindWeight,
		ValueText:  "78.4",
		ObservedAt: time.Now().UTC(),
	}
	b, _ := json.Marshal(in)
	s := string(b)
	for _, k := range []string{`"loinc_code"`, `"snomed_code"`, `"unit"`, `"source_id"`, `"delta"`} {
		if strings.Contains(s, k) {
			t.Errorf("expected %s to be omitted, got: %s", k, s)
		}
	}
}
