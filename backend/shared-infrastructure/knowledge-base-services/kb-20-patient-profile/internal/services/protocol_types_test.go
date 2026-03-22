package services

import "testing"

func TestDrugStep_HasRequiredFields(t *testing.T) {
	step := DrugStep{
		StepOrder:         1,
		DrugClass:         "biguanide",
		DrugName:          "metformin",
		StartingDoseMg:    500,
		DoseIncrementMg:   500,
		MaxDoseMg:         2000,
		FrequencyPerDay:   2,
		EscalationTrigger: "hba1c_above_target_12wk",
		ChannelBGuards:    []string{"B-01", "B-02"},
		ChannelCGuards:    []string{"PG-04", "PG-07"},
	}
	if step.DrugClass == "" {
		t.Error("DrugClass must not be empty")
	}
	if step.MaxDoseMg < step.StartingDoseMg {
		t.Error("MaxDoseMg must be >= StartingDoseMg")
	}
}

func TestTargetRange_SelectByArchetype(t *testing.T) {
	tr := TargetRange{
		DefaultLow:  0,
		DefaultHigh: 7.0,
		IndividualisedTargets: []IndividualisedTarget{
			{Archetype: "ElderlyFrail", High: 8.0, Rationale: "ADA 2026 Sec 6: relaxed target for frail elderly"},
			{Archetype: "CKDProgressor", High: 7.0, Rationale: "Standard target with renal monitoring"},
		},
	}
	target := tr.TargetFor("ElderlyFrail")
	if target.High != 8.0 {
		t.Errorf("expected 8.0 for ElderlyFrail, got %f", target.High)
	}
	target = tr.TargetFor("GoodResponder")
	if target.High != 7.0 {
		t.Errorf("expected default 7.0 for unknown archetype, got %f", target.High)
	}
}

func TestSuccessMode_Constants(t *testing.T) {
	if SuccessModeAll != "ALL" {
		t.Error("SuccessModeAll must be ALL")
	}
	if SuccessModeNever != "NEVER" {
		t.Error("SuccessModeNever must be NEVER")
	}
	if SuccessModeCardOnly != "CARD_ONLY" {
		t.Error("SuccessModeCardOnly must be CARD_ONLY")
	}
}
