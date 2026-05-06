package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTargetBPThresholdSpecRoundTrip(t *testing.T) {
	in := TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out TargetBPThresholdSpec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.SystolicMax != 140 || out.DiastolicMax != 90 {
		t.Errorf("round-trip: got %+v, want %+v", out, in)
	}
}

func TestTargetCompletionDateSpecRoundTrip(t *testing.T) {
	in := TargetCompletionDateSpec{
		EndDate:      time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		DurationDays: 7,
		Rationale:    "amoxicillin course",
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out TargetCompletionDateSpec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.EndDate.Equal(in.EndDate) || out.DurationDays != 7 || out.Rationale != "amoxicillin course" {
		t.Errorf("round-trip: got %+v, want %+v", out, in)
	}
}

func TestTargetSymptomResolutionSpecRoundTrip(t *testing.T) {
	in := TargetSymptomResolutionSpec{
		TargetSymptom:        "pain",
		MonitoringWindowDays: 14,
		SNOMEDCode:           "22253000",
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out TargetSymptomResolutionSpec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.TargetSymptom != in.TargetSymptom || out.SNOMEDCode != in.SNOMEDCode {
		t.Errorf("round-trip mismatch")
	}
}

func TestTargetHbA1cBandSpecRoundTrip(t *testing.T) {
	in := TargetHbA1cBandSpec{Min: 6.5, Max: 8.0}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out TargetHbA1cBandSpec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Min != 6.5 || out.Max != 8.0 {
		t.Errorf("round-trip: got %+v, want %+v", out, in)
	}
}

func TestTargetOpenSpecRoundTrip(t *testing.T) {
	in := TargetOpenSpec{Rationale: "long-term anticoagulation for AF"}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out TargetOpenSpec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Rationale != in.Rationale {
		t.Errorf("round-trip: got %+v, want %+v", out, in)
	}
}
