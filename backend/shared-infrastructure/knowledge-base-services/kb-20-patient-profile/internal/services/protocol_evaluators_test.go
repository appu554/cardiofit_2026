package services

import "testing"

// ---------------------------------------------------------------------------
// GLYC-1 Evaluator Tests
// ---------------------------------------------------------------------------

func TestEvaluateGLYC1_Monotherapy_AdvanceToCombination(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "GLYC-1",
		CurrentPhase:     "MONOTHERAPY",
		DaysInPhase:      84, // 12 weeks
		HbA1cAboveTarget: true,
	}
	d := EvaluateGLYC1Transition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", d.Action)
	}
	if d.NextPhase != "COMBINATION" {
		t.Errorf("expected COMBINATION, got %s", d.NextPhase)
	}
}

func TestEvaluateGLYC1_Monotherapy_HoldBelow12Weeks(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "GLYC-1",
		CurrentPhase:     "MONOTHERAPY",
		DaysInPhase:      56, // 8 weeks
		HbA1cAboveTarget: true,
	}
	d := EvaluateGLYC1Transition(eval)
	if d.Action != "HOLD" {
		t.Errorf("expected HOLD at 8 weeks, got %s", d.Action)
	}
}

func TestEvaluateGLYC1_Combination_AdvanceToOptimization(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "GLYC-1",
		CurrentPhase:     "COMBINATION",
		DaysInPhase:      168, // 24 weeks
		HbA1cAboveTarget: false,
	}
	d := EvaluateGLYC1Transition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", d.Action)
	}
	if d.NextPhase != "OPTIMIZATION" {
		t.Errorf("expected OPTIMIZATION, got %s", d.NextPhase)
	}
}

func TestEvaluateGLYC1_SafetyAbort(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:   "GLYC-1",
		CurrentPhase: "MONOTHERAPY",
		DaysInPhase:  30,
		SafetyFlags:  true,
	}
	d := EvaluateGLYC1Transition(eval)
	if d.Action != "ABORT" {
		t.Errorf("expected ABORT on safety flag, got %s", d.Action)
	}
}

// ---------------------------------------------------------------------------
// HTN-1 Evaluator Tests
// ---------------------------------------------------------------------------

func TestEvaluateHTN1_Monotherapy_AdvanceToDualTherapy(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:     "HTN-1",
		CurrentPhase:   "MONOTHERAPY",
		DaysInPhase:    28,
		SBPAboveTarget: true,
	}
	d := EvaluateHTN1Transition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", d.Action)
	}
	if d.NextPhase != "DUAL_THERAPY" {
		t.Errorf("expected DUAL_THERAPY, got %s", d.NextPhase)
	}
}

func TestEvaluateHTN1_DualTherapy_AdvanceToTriple(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:     "HTN-1",
		CurrentPhase:   "DUAL_THERAPY",
		DaysInPhase:    28,
		SBPAboveTarget: true,
	}
	d := EvaluateHTN1Transition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", d.Action)
	}
	if d.NextPhase != "TRIPLE_THERAPY" {
		t.Errorf("expected TRIPLE_THERAPY, got %s", d.NextPhase)
	}
}

func TestEvaluateHTN1_ResistantHTN_Escalate(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:     "HTN-1",
		CurrentPhase:   "RESISTANT_HTN",
		DaysInPhase:    28,
		SBPAboveTarget: true,
	}
	d := EvaluateHTN1Transition(eval)
	if d.Action != "ESCALATE" {
		t.Errorf("expected ESCALATE for resistant HTN on 4 agents, got %s", d.Action)
	}
}

func TestEvaluateHTN1_SafetyAbort(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:   "HTN-1",
		CurrentPhase: "MONOTHERAPY",
		DaysInPhase:  10,
		SafetyFlags:  true,
	}
	d := EvaluateHTN1Transition(eval)
	if d.Action != "ABORT" {
		t.Errorf("expected ABORT, got %s", d.Action)
	}
}

// ---------------------------------------------------------------------------
// RENAL-1 Evaluator Tests
// ---------------------------------------------------------------------------

func TestEvaluateRENAL1_RAAS_AdvanceToSGLT2i(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:      "RENAL-1",
		CurrentPhase:    "RAAS_OPTIMISATION",
		DaysInPhase:     28,
		ACRNotImproving: true,
	}
	d := EvaluateRENAL1Transition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", d.Action)
	}
	if d.NextPhase != "SGLT2I_ADDITION" {
		t.Errorf("expected SGLT2I_ADDITION, got %s", d.NextPhase)
	}
}

func TestEvaluateRENAL1_EGFRDecline_Escalate(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:   "RENAL-1",
		CurrentPhase: "SGLT2I_ADDITION",
		DaysInPhase:  14,
		EGFRDelta:    7.0, // >5 triggers escalation
	}
	d := EvaluateRENAL1Transition(eval)
	if d.Action != "ESCALATE" {
		t.Errorf("expected ESCALATE for eGFR decline >5, got %s", d.Action)
	}
}

func TestEvaluateRENAL1_Finerenone_AdvanceToMonitoring(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:   "RENAL-1",
		CurrentPhase: "FINERENONE_ADDITION",
		DaysInPhase:  56,
	}
	d := EvaluateRENAL1Transition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", d.Action)
	}
	if d.NextPhase != "MONITORING" {
		t.Errorf("expected MONITORING, got %s", d.NextPhase)
	}
}

// ---------------------------------------------------------------------------
// DEPRESC-1 Evaluator Tests
// ---------------------------------------------------------------------------

func TestEvaluateDEPRESC1_Assessment_AdvanceToStepdown(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:   "DEPRESC-1",
		CurrentPhase: "ASSESSMENT",
		DaysInPhase:  7,
	}
	d := EvaluateDEPRESC1Transition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", d.Action)
	}
	if d.NextPhase != "STEPDOWN" {
		t.Errorf("expected STEPDOWN, got %s", d.NextPhase)
	}
}

func TestEvaluateDEPRESC1_Stepdown_EscalateOnHighHbA1c(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "DEPRESC-1",
		CurrentPhase:     "STEPDOWN",
		DaysInPhase:      30,
		HbA1cAboveTarget: true,
	}
	d := EvaluateDEPRESC1Transition(eval)
	if d.Action != "ESCALATE" {
		t.Errorf("expected ESCALATE when HbA1c rises during stepdown, got %s", d.Action)
	}
}

func TestEvaluateDEPRESC1_Stepdown_AdvanceToMonitoring(t *testing.T) {
	eval := TransitionEvaluation{
		ProtocolID:       "DEPRESC-1",
		CurrentPhase:     "STEPDOWN",
		DaysInPhase:      56,
		HbA1cAboveTarget: false,
	}
	d := EvaluateDEPRESC1Transition(eval)
	if d.Action != "ADVANCE" {
		t.Errorf("expected ADVANCE, got %s", d.Action)
	}
	if d.NextPhase != "MONITORING" {
		t.Errorf("expected MONITORING, got %s", d.NextPhase)
	}
}

// ---------------------------------------------------------------------------
// EvaluateAndTransition Integration Tests
// ---------------------------------------------------------------------------

func TestEvaluateAndTransition_GLYC1_PublishesEscalationEvent(t *testing.T) {
	spy := &spyEventBus{}
	svc := &ProtocolService{eventBus: spy}

	eval := TransitionEvaluation{
		ProtocolID:       "GLYC-1",
		CurrentPhase:     "MONOTHERAPY",
		DaysInPhase:      112, // exceeded 16 weeks — timeout escalation
		HbA1cAboveTarget: false,
	}

	decision, err := svc.EvaluateAndTransition("patient-med-1", eval)
	if decision.Action != "ESCALATE" {
		t.Fatalf("expected ESCALATE, got %s", decision.Action)
	}
	if err == nil {
		t.Fatal("expected non-nil error for ESCALATE")
	}
}

func TestEvaluateAndTransition_LIPID1_CardOnly_NoTransition(t *testing.T) {
	spy := &spyEventBus{}
	svc := &ProtocolService{eventBus: spy}

	eval := TransitionEvaluation{
		ProtocolID:   "LIPID-1",
		CurrentPhase: "ASSESSMENT",
		DaysInPhase:  90,
	}

	decision, err := svc.EvaluateAndTransition("patient-lipid-1", eval)
	if decision.Action != "HOLD" {
		t.Errorf("expected HOLD for LIPID-1 card-only, got %s", decision.Action)
	}
	if err != nil {
		t.Errorf("expected nil error for LIPID-1 HOLD, got %v", err)
	}
}
