package services

import "testing"

func TestEvaluateMAINTAINTransition_Consolidation_AdvanceToIndependence(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:          "M3-MAINTAIN",
		CurrentPhase:        "CONSOLIDATION",
		DaysInPhase:         95,
		MRIScore:            43,
		MRISustainedDays:    30,
		AdherencePct:        0.55,
		ConsecutiveCheckins: 5,
	}
	d := EvaluateMAINTAINTransition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("action = %q, want ADVANCE", d.Action)
	}
	if d.NextPhase != "INDEPENDENCE" {
		t.Errorf("next_phase = %q, want INDEPENDENCE", d.NextPhase)
	}
}

func TestEvaluateMAINTAINTransition_Consolidation_Hold(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:   "M3-MAINTAIN",
		CurrentPhase: "CONSOLIDATION",
		DaysInPhase:  45,
		MRIScore:     52,
	}
	d := EvaluateMAINTAINTransition(eval)
	if d.Action != "HOLD" {
		t.Errorf("action = %q, want HOLD", d.Action)
	}
}

func TestEvaluateMAINTAINTransition_Independence_AdvanceToStability(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "M3-MAINTAIN",
		CurrentPhase:     "INDEPENDENCE",
		DaysInPhase:      95,
		MRIScore:         38,
		MRISustainedDays: 60,
		NoRelapseDays:    65,
	}
	d := EvaluateMAINTAINTransition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("action = %q, want ADVANCE", d.Action)
	}
	if d.NextPhase != "STABILITY" {
		t.Errorf("next_phase = %q, want STABILITY", d.NextPhase)
	}
}

func TestEvaluateMAINTAINTransition_Stability_AdvanceToPartnership(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:            "M3-MAINTAIN",
		CurrentPhase:          "STABILITY",
		DaysInPhase:           95,
		HbA1cAtTarget:         true,
		HbA1cAtTargetReadings: 2,
		YearReviewComplete:    true,
		PhysicianGradApproval: true,
	}
	d := EvaluateMAINTAINTransition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("action = %q, want ADVANCE", d.Action)
	}
	if d.NextPhase != "PARTNERSHIP" {
		t.Errorf("next_phase = %q, want PARTNERSHIP", d.NextPhase)
	}
}

func TestEvaluateMAINTAINTransition_Partnership_Hold(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:   "M3-MAINTAIN",
		CurrentPhase: "PARTNERSHIP",
		DaysInPhase:  365,
	}
	d := EvaluateMAINTAINTransition(eval)
	if d.Action != "HOLD" {
		t.Errorf("PARTNERSHIP is indefinite — action = %q, want HOLD", d.Action)
	}
}
