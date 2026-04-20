package models

import (
	"testing"
)

func TestInterventionDefinition_Validate_RejectsMissingID(t *testing.T) {
	def := InterventionDefinition{
		CohortID: "hcf_catalyst_chf",
		Category: string(CategoryFollowUp),
		Name:     "Nurse phone follow-up",
	}
	if err := def.Validate(); err == nil {
		t.Fatal("expected validation error for missing ID")
	}
}

func TestInterventionDefinition_Validate_RejectsUnknownCategory(t *testing.T) {
	def := InterventionDefinition{
		ID:       "nurse_phone_48h",
		CohortID: "hcf_catalyst_chf",
		Category: "NOT_A_REAL_CATEGORY",
		Name:     "Nurse phone follow-up",
	}
	if err := def.Validate(); err == nil {
		t.Fatal("expected validation error for unknown category")
	}
}

func TestInterventionDefinition_Validate_AcceptsWellFormed(t *testing.T) {
	def := InterventionDefinition{
		ID:               "nurse_phone_48h",
		CohortID:         "hcf_catalyst_chf",
		Category:         string(CategoryFollowUp),
		Name:             "Nurse phone follow-up",
		CoolDownHours:    48,
		ResourceCost:     1.0,
		FeatureSignature: []string{"age", "ef_last", "nt_probnp_trend_7d"},
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
