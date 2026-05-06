package models

import (
	"encoding/json"
	"testing"
)

func TestStopCriteriaReviewSpecRoundTrip(t *testing.T) {
	in := StopCriteriaReviewSpec{ReviewAfterDays: 30, ReviewOwner: "ACOP"}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out StopCriteriaReviewSpec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ReviewAfterDays != 30 || out.ReviewOwner != "ACOP" {
		t.Errorf("round-trip: got %+v want %+v", out, in)
	}
}

func TestStopCriteriaThresholdSpecRoundTrip(t *testing.T) {
	in := StopCriteriaThresholdSpec{
		ObservationKind: "vital",
		LOINCCode:       "8867-4",
		Operator:        "<",
		Value:           50,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out StopCriteriaThresholdSpec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ObservationKind != "vital" || out.Operator != "<" || out.Value != 50 {
		t.Errorf("round-trip: got %+v want %+v", out, in)
	}
}

func TestStopCriteriaThresholdSpecOmitsEmptyOptional(t *testing.T) {
	in := StopCriteriaThresholdSpec{
		ObservationKind: "vital",
		LOINCCode:       "8867-4",
		Operator:        "<",
		Value:           50,
		// SNOMEDCode left empty — must be omitted from JSON
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, present := raw["snomed_code"]; present {
		t.Errorf("expected snomed_code to be omitted when empty; got map %+v", raw)
	}
}
