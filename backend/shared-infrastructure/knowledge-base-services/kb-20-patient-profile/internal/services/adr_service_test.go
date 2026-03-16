package services

import (
	"encoding/json"
	"testing"
)

func TestMergePartialCMRule_PipelineAddsConditionToExistingDeltaP(t *testing.T) {
	existing := `{"target_differential":"OH","delta_p":0.20,"effect_type":"INCREASE_PRIOR"}`
	incoming := `{"condition":"egfr < 45","safety_flag_text_en":"eGFR threshold from KDIGO"}`

	result := mergePartialCMRule(existing, incoming)

	var merged map[string]interface{}
	if err := json.Unmarshal([]byte(result), &merged); err != nil {
		t.Fatalf("failed to parse merged result: %v", err)
	}

	// existing fields preserved
	if merged["delta_p"] != 0.20 {
		t.Errorf("delta_p = %v, want 0.20", merged["delta_p"])
	}
	if merged["target_differential"] != "OH" {
		t.Errorf("target_differential = %v, want OH", merged["target_differential"])
	}

	// incoming field added
	if merged["condition"] != "egfr < 45" {
		t.Errorf("condition = %v, want 'egfr < 45'", merged["condition"])
	}
	if merged["safety_flag_text_en"] != "eGFR threshold from KDIGO" {
		t.Errorf("safety_flag_text_en missing")
	}
}

func TestMergePartialCMRule_NeverOverwritesExistingKeys(t *testing.T) {
	existing := `{"delta_p":0.20,"condition":"med_class==ARB"}`
	incoming := `{"delta_p":0.35,"condition":"egfr < 45"}`

	result := mergePartialCMRule(existing, incoming)

	var merged map[string]interface{}
	json.Unmarshal([]byte(result), &merged)

	// Existing keys must NOT be overwritten
	if merged["delta_p"] != 0.20 {
		t.Errorf("delta_p was overwritten: got %v, want 0.20", merged["delta_p"])
	}
	if merged["condition"] != "med_class==ARB" {
		t.Errorf("condition was overwritten: got %v, want 'med_class==ARB'", merged["condition"])
	}
}

func TestMergePartialCMRule_EmptyExistingGetsIncoming(t *testing.T) {
	existing := `{}`
	incoming := `{"delta_p":0.25,"condition":"med_class==SGLT2i"}`

	result := mergePartialCMRule(existing, incoming)

	var merged map[string]interface{}
	json.Unmarshal([]byte(result), &merged)

	if merged["delta_p"] != 0.25 {
		t.Errorf("delta_p = %v, want 0.25", merged["delta_p"])
	}
}
