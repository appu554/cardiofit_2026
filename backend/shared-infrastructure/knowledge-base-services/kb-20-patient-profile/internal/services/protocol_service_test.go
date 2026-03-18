package services

import (
	"testing"
)

func TestProtocolService_EvaluateTransition_PRP_Phase1Ready(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "STABILIZATION",
		DaysInPhase:       15,
		ProteinAdherence:  0.65,
		ExerciseAdherence: 0.0,
		SafetyFlags:       false,
	}

	decision := EvaluatePRPTransition(eval)
	if decision.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE for day 15 + 65%% adherence, got %s", decision.Action)
	}
	if decision.NextPhase != "RESTORATION" {
		t.Errorf("expected RESTORATION, got %s", decision.NextPhase)
	}
}

func TestProtocolService_EvaluateTransition_PRP_Phase1Hold(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "M3-PRP",
		CurrentPhase:     "STABILIZATION",
		DaysInPhase:      15,
		ProteinAdherence: 0.45,
		SafetyFlags:      false,
	}

	decision := EvaluatePRPTransition(eval)
	if decision.Action != "HOLD" {
		t.Errorf("expected HOLD for 45%% adherence, got %s", decision.Action)
	}
}

func TestProtocolService_EvaluateTransition_PRP_Phase1Abort(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:   "M3-PRP",
		CurrentPhase: "STABILIZATION",
		DaysInPhase:  15,
		SafetyFlags:  true,
	}

	decision := EvaluatePRPTransition(eval)
	if decision.Action != "ABORT" {
		t.Errorf("expected ABORT for safety flag, got %s", decision.Action)
	}
}

func TestProtocolService_EvaluateTransition_VFRP_Phase2Ready(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:           "M3-VFRP",
		CurrentPhase:         "FAT_MOBILIZATION",
		DaysInPhase:          43,
		ExerciseAdherence:    0.55,
		MealQualityScore:     65,
		MealQualityImproving: true,
		SafetyFlags:          false,
	}

	decision := EvaluateVFRPTransition(eval)
	if decision.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", decision.Action)
	}
}

func TestProtocolService_EvaluateTransition_VFRP_Phase1Abort_ExcessiveWeightLoss(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "M3-VFRP",
		CurrentPhase:     "METABOLIC_STABILIZATION",
		DaysInPhase:      10,
		MealQualityScore: 60,
		WeightLossKg:     3.5,
		BMI:              23.0,
		SafetyFlags:      false,
	}

	decision := EvaluateVFRPTransition(eval)
	if decision.Action != "ABORT" {
		t.Errorf("expected ABORT for weight loss 3.5kg with BMI 23, got %s", decision.Action)
	}
}

func TestProtocolService_EvaluateTransition_VFRP_WeightLoss_HighBMI_NoAbort(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "M3-VFRP",
		CurrentPhase:     "METABOLIC_STABILIZATION",
		DaysInPhase:      10,
		MealQualityScore: 60,
		WeightLossKg:     3.5,
		BMI:              28.0,
		SafetyFlags:      false,
	}

	decision := EvaluateVFRPTransition(eval)
	if decision.Action == "ABORT" {
		t.Error("should NOT abort for weight loss when BMI >= 24")
	}
}
