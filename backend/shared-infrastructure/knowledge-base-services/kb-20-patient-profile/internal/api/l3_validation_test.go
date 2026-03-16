package api

import (
	"testing"
)

func TestValidateL3_RejectsMissingE1(t *testing.T) {
	profiles := []map[string]interface{}{
		{"drug_class": "", "reaction": "", "source": "PIPELINE"},
	}
	errors := validateL3Payload(profiles)
	if len(errors) == 0 {
		t.Fatal("expected E1_MISSING error for empty drug_class+reaction")
	}
	if errors[0].Code != "E1_MISSING" {
		t.Errorf("got code %s, want E1_MISSING", errors[0].Code)
	}
}

func TestValidateL3_RejectsDeltaPOutOfRange(t *testing.T) {
	profiles := []map[string]interface{}{
		{
			"drug_class": "ARB",
			"reaction":   "Dizziness",
			"source":     "MANUAL_CURATED",
			"context_modifier_rule": `{"delta_p": 0.55, "target_differential": "OH"}`,
		},
	}
	errors := validateL3Payload(profiles)
	if len(errors) == 0 {
		t.Fatal("expected DELTA_P_OUT_OF_RANGE error for delta_p=0.55")
	}
	if errors[0].Code != "DELTA_P_OUT_OF_RANGE" {
		t.Errorf("got code %s, want DELTA_P_OUT_OF_RANGE", errors[0].Code)
	}
}

func TestValidateL3_AcceptsNullDeltaPForHardBlock(t *testing.T) {
	profiles := []map[string]interface{}{
		{
			"drug_class": "PDE5i",
			"reaction":   "Severe hypotension",
			"source":     "MANUAL_CURATED",
			"context_modifier_rule": `{"effect_type": "HARD_BLOCK", "condition": "med_class==PDE5i"}`,
		},
	}
	errors := validateL3Payload(profiles)
	if len(errors) != 0 {
		t.Errorf("expected no errors for HARD_BLOCK without delta_p, got %d: %v", len(errors), errors)
	}
}

func TestValidateL3_RejectsInvalidSource(t *testing.T) {
	profiles := []map[string]interface{}{
		{
			"drug_class": "ARB",
			"reaction":   "Dizziness",
			"source":     "UNKNOWN_SOURCE",
		},
	}
	errors := validateL3Payload(profiles)
	if len(errors) == 0 {
		t.Fatal("expected INVALID_SOURCE error")
	}
	if errors[0].Code != "INVALID_SOURCE" {
		t.Errorf("got code %s, want INVALID_SOURCE", errors[0].Code)
	}
}
