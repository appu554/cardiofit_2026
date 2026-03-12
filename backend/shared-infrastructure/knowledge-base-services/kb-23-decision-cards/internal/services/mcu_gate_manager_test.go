package services

import (
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

func newGateManager() *MCUGateManager {
	return NewMCUGateManager(testConfig(), zap.NewNop())
}

// ---------------------------------------------------------------------------
// EvaluateGate tests
// ---------------------------------------------------------------------------

func TestEvaluateGate_DefaultGate(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules:      nil, // no rules
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, rationale, notes := mgr.EvaluateGate(tmpl, models.TierProbable, ctx)

	if gate != models.GateSafe {
		t.Errorf("gate = %q, want %q", gate, models.GateSafe)
	}
	if rationale != "template default gate" {
		t.Errorf("rationale = %q, want %q", rationale, "template default gate")
	}
	if notes != "" {
		t.Errorf("adjustmentNotes = %q, want empty", notes)
	}
}

func TestEvaluateGate_FirstMatchingRuleWins(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "TIER_FIRM", Gate: models.GateHalt, Rationale: "rule-1"},
			{Condition: "TIER_PROBABLE", Gate: models.GateModify, Rationale: "rule-2", AdjustmentNotes: "adj-2"},
			{Condition: "ALWAYS", Gate: models.GatePause, Rationale: "rule-3"},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, rationale, notes := mgr.EvaluateGate(tmpl, models.TierProbable, ctx)

	if gate != models.GateModify {
		t.Errorf("gate = %q, want %q (second rule should match)", gate, models.GateModify)
	}
	if rationale != "rule-2" {
		t.Errorf("rationale = %q, want %q", rationale, "rule-2")
	}
	if notes != "adj-2" {
		t.Errorf("adjustmentNotes = %q, want %q", notes, "adj-2")
	}
}

func TestEvaluateGate_V06StressOverride_EscalatesToPause(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
	}
	ctx := &PatientContext{PatientID: "p1", IsAcuteIll: true}

	gate, rationale, _ := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if gate != models.GatePause {
		t.Errorf("gate = %q, want %q (V-06 should escalate SAFE to PAUSE)", gate, models.GatePause)
	}
	if rationale != "V-06: stress hyperglycaemia -- acute illness, medication intensification paused" {
		t.Errorf("rationale = %q, want V-06 message", rationale)
	}
}

func TestEvaluateGate_V06NoOp_AlreadyMoreRestrictive(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateHalt,
	}
	ctx := &PatientContext{PatientID: "p1", IsAcuteIll: true}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if gate != models.GateHalt {
		t.Errorf("gate = %q, want %q (HALT already more restrictive than PAUSE)", gate, models.GateHalt)
	}
}

func TestEvaluateGate_N05Enforcement_EmptyNotes(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "ALWAYS", Gate: models.GateModify, Rationale: "modify-rule", AdjustmentNotes: ""},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, _, notes := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if gate != models.GateModify {
		t.Errorf("gate = %q, want %q", gate, models.GateModify)
	}
	if notes != "MODIFY_GATE: titration adjustment required -- see recommendations" {
		t.Errorf("adjustmentNotes = %q, want N-05 auto-filled message", notes)
	}
}

func TestEvaluateGate_N05WithExistingNotes_Preserved(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "ALWAYS", Gate: models.GateModify, Rationale: "modify-rule", AdjustmentNotes: "reduce 20%"},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	_, _, notes := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if notes != "reduce 20%" {
		t.Errorf("adjustmentNotes = %q, want %q (existing notes should be preserved)", notes, "reduce 20%")
	}
}

func TestEvaluateGate_NilPatientContext(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "TIER_FIRM", Gate: models.GatePause, Rationale: "firm-rule"},
		},
	}

	gate, rationale, _ := mgr.EvaluateGate(tmpl, models.TierFirm, nil)

	if gate != models.GatePause {
		t.Errorf("gate = %q, want %q", gate, models.GatePause)
	}
	if rationale != "firm-rule" {
		t.Errorf("rationale = %q, want %q", rationale, "firm-rule")
	}
}

// ---------------------------------------------------------------------------
// evaluateCondition tests (indirect via EvaluateGate)
// ---------------------------------------------------------------------------

func TestCondition_TierFirm_Matches(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "TIER_FIRM", Gate: models.GateHalt, Rationale: "firm-matched"},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if gate != models.GateHalt {
		t.Errorf("TIER_FIRM with FIRM tier: gate = %q, want %q", gate, models.GateHalt)
	}
}

func TestCondition_TierFirm_NoMatch(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "TIER_FIRM", Gate: models.GateHalt, Rationale: "firm-matched"},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierProbable, ctx)

	if gate != models.GateSafe {
		t.Errorf("TIER_FIRM with PROBABLE tier: gate = %q, want %q (default)", gate, models.GateSafe)
	}
}

func TestCondition_AcuteIllness_Matches(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "ACUTE_ILLNESS", Gate: models.GateHalt, Rationale: "acute-matched"},
		},
	}
	ctx := &PatientContext{PatientID: "p1", IsAcuteIll: true}

	gate, rationale, _ := mgr.EvaluateGate(tmpl, models.TierProbable, ctx)

	// Rule matches HALT, but V-06 only escalates to PAUSE -- HALT is already
	// more restrictive so it stays HALT.
	if gate != models.GateHalt {
		t.Errorf("ACUTE_ILLNESS: gate = %q, want %q", gate, models.GateHalt)
	}
	if rationale != "acute-matched" {
		t.Errorf("rationale = %q, want %q", rationale, "acute-matched")
	}
}

func TestCondition_EGFRLow_Matches(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "EGFR_LOW", Gate: models.GatePause, Rationale: "egfr-low"},
		},
	}
	ctx := &PatientContext{PatientID: "p1", EGFRValue: 25}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if gate != models.GatePause {
		t.Errorf("EGFR_LOW with eGFR=25: gate = %q, want %q", gate, models.GatePause)
	}
}

func TestCondition_EGFRLow_NoMatch(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "EGFR_LOW", Gate: models.GatePause, Rationale: "egfr-low"},
		},
	}
	ctx := &PatientContext{PatientID: "p1", EGFRValue: 35}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if gate != models.GateSafe {
		t.Errorf("EGFR_LOW with eGFR=35: gate = %q, want %q (default)", gate, models.GateSafe)
	}
}

func TestCondition_STEMIConfirmed_Matches(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		DifferentialID: "ACS_STEMI",
		GateRules: []models.GateRule{
			{Condition: "stemi_confirmed", Gate: models.GateHalt, Rationale: "stemi-confirmed"},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if gate != models.GateHalt {
		t.Errorf("stemi_confirmed with FIRM+ACS_STEMI: gate = %q, want %q", gate, models.GateHalt)
	}
}

func TestCondition_STEMIConfirmed_NoMatch_WrongTier(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		DifferentialID: "ACS_STEMI",
		GateRules: []models.GateRule{
			{Condition: "stemi_confirmed", Gate: models.GateHalt, Rationale: "stemi-confirmed"},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierProbable, ctx)

	if gate != models.GateSafe {
		t.Errorf("stemi_confirmed with PROBABLE tier: gate = %q, want %q (default)", gate, models.GateSafe)
	}
}

func TestCondition_NSTEMIHighRisk_Matches(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		DifferentialID: "ACS_NSTEMI",
		GateRules: []models.GateRule{
			{Condition: "nstemi_high_risk", Gate: models.GatePause, Rationale: "nstemi-high"},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierProbable, ctx)

	if gate != models.GatePause {
		t.Errorf("nstemi_high_risk with PROBABLE+ACS_NSTEMI: gate = %q, want %q", gate, models.GatePause)
	}
}

func TestCondition_HypoglycaemiaRecent_Matches(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "hypoglycaemia_recent", Gate: models.GatePause, Rationale: "hypo-recent"},
		},
	}
	ctx := &PatientContext{PatientID: "p1", HasRecentHypoglycaemia: true}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierFirm, ctx)

	if gate != models.GatePause {
		t.Errorf("hypoglycaemia_recent: gate = %q, want %q", gate, models.GatePause)
	}
}

func TestCondition_Always_Matches(t *testing.T) {
	mgr := newGateManager()
	tmpl := &models.CardTemplate{
		MCUGateDefault: models.GateSafe,
		GateRules: []models.GateRule{
			{Condition: "ALWAYS", Gate: models.GateModify, Rationale: "always-rule", AdjustmentNotes: "always-adj"},
		},
	}
	ctx := &PatientContext{PatientID: "p1"}

	gate, _, _ := mgr.EvaluateGate(tmpl, models.TierUncertain, ctx)

	if gate != models.GateModify {
		t.Errorf("ALWAYS condition: gate = %q, want %q", gate, models.GateModify)
	}
}
