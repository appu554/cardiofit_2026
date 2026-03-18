package services

import (
	"testing"
)

func TestProtocolFlow_PRP_FullCycle(t *testing.T) {
	registry := NewProtocolRegistry()

	// Step 1: Check entry eligibility
	eligible, reason := registry.CheckEntry("M3-PRP", map[string]float64{
		"protein_gap": 25,
		"egfr":        65,
	}, map[string]bool{})
	if !eligible {
		t.Fatalf("PRP should be eligible, got: %s", reason)
	}

	// Step 2: Evaluate Phase 1 → Phase 2 transition
	eval := TransitionEvaluation{
		ProtocolID:       "M3-PRP",
		CurrentPhase:     "STABILIZATION",
		DaysInPhase:      16,
		ProteinAdherence: 0.70,
	}
	decision := EvaluatePRPTransition(eval)
	if decision.Action != "ADVANCE" || decision.NextPhase != "RESTORATION" {
		t.Errorf("expected ADVANCE to RESTORATION, got %s to %s", decision.Action, decision.NextPhase)
	}

	// Step 3: Evaluate Phase 2 → Phase 3 transition
	eval2 := TransitionEvaluation{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "RESTORATION",
		DaysInPhase:       30,
		ProteinAdherence:  0.60,
		ExerciseAdherence: 0.55,
	}
	decision2 := EvaluatePRPTransition(eval2)
	if decision2.Action != "ADVANCE" || decision2.NextPhase != "OPTIMIZATION" {
		t.Errorf("expected ADVANCE to OPTIMIZATION, got %s to %s", decision2.Action, decision2.NextPhase)
	}
}

func TestProtocolFlow_VFRP_AutoActivatesPRP(t *testing.T) {
	registry := NewProtocolRegistry()

	// VFRP eligible
	eligible, _ := registry.CheckEntry("M3-VFRP", map[string]float64{
		"waist_cm": 95,
		"bmi":      26,
	}, map[string]bool{})
	if !eligible {
		t.Fatal("VFRP should be eligible for waist 95cm")
	}

	// When VFRP activates and protein_gap >= 20, PRP should also be eligible
	prpEligible, _ := registry.CheckEntry("M3-PRP", map[string]float64{
		"protein_gap": 22,
		"egfr":        70,
	}, map[string]bool{})
	if !prpEligible {
		t.Fatal("PRP should be auto-eligible when VFRP activates with protein gap >= 20")
	}
}

func TestProtocolFlow_ConcurrentExecution_SharedExercise(t *testing.T) {
	// Verify both protocols can be concurrent
	registry := NewProtocolRegistry()
	prp, _ := registry.GetTemplate("M3-PRP")
	vfrp, _ := registry.GetTemplate("M3-VFRP")

	prpCanConcur := false
	for _, c := range prp.ConcurrentWith {
		if c == "M3-VFRP" {
			prpCanConcur = true
		}
	}
	vfrpCanConcur := false
	for _, c := range vfrp.ConcurrentWith {
		if c == "M3-PRP" {
			vfrpCanConcur = true
		}
	}

	if !prpCanConcur || !vfrpCanConcur {
		t.Error("PRP and VFRP must be marked as concurrent with each other")
	}
}
