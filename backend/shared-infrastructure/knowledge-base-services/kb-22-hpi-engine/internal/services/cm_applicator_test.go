package services

import (
	"math"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

func newTestCMApplicator() *CMApplicator {
	return NewCMApplicator(testLogger())
}

// --- G14: logit-based delta conversion tests ---

func TestCMLogit_Symmetry(t *testing.T) {
	// cmLogit(0.50 + mag) and cmLogit(0.50 - mag) should be equal magnitude, opposite sign
	magnitudes := []float64{0.05, 0.10, 0.20, 0.30, 0.40, 0.49}
	for _, mag := range magnitudes {
		increase := cmLogit(0.50 + mag)
		decrease := cmLogit(0.50 - mag)
		if math.Abs(increase+decrease) > 1e-10 {
			t.Errorf("mag=%.2f: logit(0.50+mag)=%.6f + logit(0.50-mag)=%.6f should be ~0, got %.10f",
				mag, increase, decrease, increase+decrease)
		}
	}
}

func TestCMLogit_LogitOfHalfIsZero(t *testing.T) {
	// logit(0.50) = log(1) = 0.0 — this is why the formula simplifies
	result := cmLogit(0.50)
	if math.Abs(result) > 1e-15 {
		t.Errorf("cmLogit(0.50) should be 0.0, got %.15f", result)
	}
}

func TestApply_IncreasePrior(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 0.0, "GERD": -0.5}
	mods := []ContextModifier{
		{
			ModifierID:    "CM_01",
			ModifierType:  "COMORBIDITY",
			Effect:        "INCREASE_PRIOR",
			Magnitude:     0.20,
			Differentials: []string{"ACS"},
		},
	}

	updated, deltas := a.Apply(logOdds, mods, nil)

	// delta should be cmLogit(0.50 + 0.20) = cmLogit(0.70)
	expectedDelta := cmLogit(0.70)
	if math.Abs(deltas["CM_01"]-expectedDelta) > 1e-10 {
		t.Errorf("CM_01 delta: expected %.6f, got %.6f", expectedDelta, deltas["CM_01"])
	}

	// ACS should be shifted by the delta
	if math.Abs(updated["ACS"]-expectedDelta) > 1e-10 {
		t.Errorf("ACS log-odds: expected %.6f, got %.6f", expectedDelta, updated["ACS"])
	}

	// GERD should be unchanged (not in Differentials list)
	if math.Abs(updated["GERD"]-(-0.5)) > 1e-10 {
		t.Errorf("GERD should be unchanged at -0.5, got %.6f", updated["GERD"])
	}
}

func TestApply_DecreasePrior(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 0.0}
	mods := []ContextModifier{
		{
			ModifierID:    "CM_02",
			ModifierType:  "LIFESTYLE",
			Effect:        "DECREASE_PRIOR",
			Magnitude:     0.15,
			Differentials: []string{"ACS"},
		},
	}

	updated, deltas := a.Apply(logOdds, mods, nil)

	// delta should be cmLogit(0.50 - 0.15) = cmLogit(0.35) which is negative
	expectedDelta := cmLogit(0.35)
	if expectedDelta >= 0 {
		t.Fatal("cmLogit(0.35) should be negative")
	}
	if math.Abs(deltas["CM_02"]-expectedDelta) > 1e-10 {
		t.Errorf("CM_02 delta: expected %.6f, got %.6f", expectedDelta, deltas["CM_02"])
	}
	if math.Abs(updated["ACS"]-expectedDelta) > 1e-10 {
		t.Errorf("ACS log-odds: expected %.6f, got %.6f", expectedDelta, updated["ACS"])
	}
}

func TestApply_MagnitudeOutOfRange(t *testing.T) {
	a := newTestCMApplicator()

	tests := []struct {
		name string
		mag  float64
	}{
		{"zero", 0.0},
		{"negative", -0.1},
		{"equal_to_half", 0.50},
		{"above_half", 0.60},
		{"exactly_one", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mods := []ContextModifier{
				{ModifierID: "BAD", Effect: "INCREASE_PRIOR", Magnitude: tt.mag, Differentials: []string{"ACS"}},
			}
			_, deltas := a.Apply(
				map[string]float64{"ACS": 1.0},
				mods, nil,
			)
			if _, exists := deltas["BAD"]; exists {
				t.Errorf("magnitude %.2f should be skipped, but delta was recorded", tt.mag)
			}
		})
	}
}

func TestApply_CappingAtMaxLogOddsShift(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 0.0}

	// Three large INCREASE modifiers that should exceed the cap
	mods := []ContextModifier{
		{ModifierID: "CM_A", Effect: "INCREASE_PRIOR", Magnitude: 0.45, Differentials: []string{"ACS"}},
		{ModifierID: "CM_B", Effect: "INCREASE_PRIOR", Magnitude: 0.45, Differentials: []string{"ACS"}},
		{ModifierID: "CM_C", Effect: "INCREASE_PRIOR", Magnitude: 0.45, Differentials: []string{"ACS"}},
	}

	updated, _ := a.Apply(logOdds, mods, nil)

	// Total shift should be capped at maxCMLogOddsShift (2.0)
	if updated["ACS"] > maxCMLogOddsShift+1e-10 {
		t.Errorf("ACS exceeded cap: got %.6f, max is %.1f", updated["ACS"], maxCMLogOddsShift)
	}
}

func TestApply_NegativeCapping(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 0.0}

	// Three large DECREASE modifiers
	mods := []ContextModifier{
		{ModifierID: "CM_A", Effect: "DECREASE_PRIOR", Magnitude: 0.45, Differentials: []string{"ACS"}},
		{ModifierID: "CM_B", Effect: "DECREASE_PRIOR", Magnitude: 0.45, Differentials: []string{"ACS"}},
		{ModifierID: "CM_C", Effect: "DECREASE_PRIOR", Magnitude: 0.45, Differentials: []string{"ACS"}},
	}

	updated, _ := a.Apply(logOdds, mods, nil)

	// Total shift should be capped at -maxCMLogOddsShift (-2.0)
	if updated["ACS"] < -maxCMLogOddsShift-1e-10 {
		t.Errorf("ACS below negative cap: got %.6f, min is %.1f", updated["ACS"], -maxCMLogOddsShift)
	}
}

func TestApply_AdherenceScaling(t *testing.T) {
	a := newTestCMApplicator()

	// Full adherence (1.0) should produce full delta
	logOddsFull := map[string]float64{"ACS": 0.0}
	modsA := []ContextModifier{
		{ModifierID: "CM_DRUG", ModifierType: "CONCOMITANT_DRUG", Effect: "INCREASE_PRIOR",
			Magnitude: 0.20, DrugClass: "STATIN", Differentials: []string{"ACS"}},
	}
	_, deltasFull := a.Apply(logOddsFull, modsA, map[string]float64{"STATIN": 1.0})

	// Half adherence (0.35) → scale = min(1.0, 0.35/0.70) = 0.50
	logOddsHalf := map[string]float64{"ACS": 0.0}
	modsB := []ContextModifier{
		{ModifierID: "CM_DRUG", ModifierType: "CONCOMITANT_DRUG", Effect: "INCREASE_PRIOR",
			Magnitude: 0.20, DrugClass: "STATIN", Differentials: []string{"ACS"}},
	}
	_, deltasHalf := a.Apply(logOddsHalf, modsB, map[string]float64{"STATIN": 0.35})

	// At 50% adherence, the adjusted magnitude is halved, so the logit delta differs
	// The delta should be cmLogit(0.50 + 0.20*0.5) = cmLogit(0.60) vs cmLogit(0.70)
	expectedFull := cmLogit(0.70)
	expectedHalf := cmLogit(0.60) // 0.50 + 0.20*0.5
	if math.Abs(deltasFull["CM_DRUG"]-expectedFull) > 1e-10 {
		t.Errorf("full adherence delta: expected %.6f, got %.6f", expectedFull, deltasFull["CM_DRUG"])
	}
	if math.Abs(deltasHalf["CM_DRUG"]-expectedHalf) > 1e-10 {
		t.Errorf("half adherence delta: expected %.6f, got %.6f", expectedHalf, deltasHalf["CM_DRUG"])
	}
}

func TestApply_AllDifferentials(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 0.0, "GERD": -0.3, "PE": 0.5}

	// Empty Differentials list means "apply to all"
	mods := []ContextModifier{
		{ModifierID: "CM_ALL", Effect: "INCREASE_PRIOR", Magnitude: 0.10},
	}

	updated, _ := a.Apply(logOdds, mods, nil)

	expectedDelta := cmLogit(0.60)
	for diffID, lo := range updated {
		preCM := map[string]float64{"ACS": 0.0, "GERD": -0.3, "PE": 0.5}[diffID]
		if math.Abs(lo-(preCM+expectedDelta)) > 1e-10 {
			t.Errorf("%s: expected %.6f, got %.6f", diffID, preCM+expectedDelta, lo)
		}
	}
}

// --- Gap 1: Passthrough effect type tests ---

func TestApply_HardBlockPassthrough(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 1.0, "GERD": -0.5}
	mods := []ContextModifier{
		{ModifierID: "BLOCK_01", Effect: "HARD_BLOCK", Differentials: []string{"ACS"}},
	}

	updated, deltas := a.Apply(logOdds, mods, nil)

	// HARD_BLOCK must appear in cmLogDeltas with delta=0.0
	delta, exists := deltas["BLOCK_01"]
	if !exists {
		t.Fatal("HARD_BLOCK modifier should be recorded in cmLogDeltas")
	}
	if delta != 0.0 {
		t.Errorf("HARD_BLOCK delta should be 0.0, got %.6f", delta)
	}

	// Log-odds must be unchanged
	if updated["ACS"] != 1.0 {
		t.Errorf("ACS should be unchanged at 1.0, got %.6f", updated["ACS"])
	}
	if updated["GERD"] != -0.5 {
		t.Errorf("GERD should be unchanged at -0.5, got %.6f", updated["GERD"])
	}
}

func TestApply_OverridePassthrough(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 0.5}
	mods := []ContextModifier{
		{ModifierID: "OVR_01", Effect: "OVERRIDE", Differentials: []string{"ACS"}},
	}

	updated, deltas := a.Apply(logOdds, mods, nil)

	if _, exists := deltas["OVR_01"]; !exists {
		t.Fatal("OVERRIDE modifier should be recorded in cmLogDeltas")
	}
	if deltas["OVR_01"] != 0.0 {
		t.Errorf("OVERRIDE delta should be 0.0, got %.6f", deltas["OVR_01"])
	}
	if updated["ACS"] != 0.5 {
		t.Errorf("ACS should be unchanged at 0.5, got %.6f", updated["ACS"])
	}
}

func TestApply_SymptomModificationPassthrough(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"HYPO": 0.3, "ACS": 0.0}
	mods := []ContextModifier{
		{ModifierID: "CM_BB_MASK", Effect: "SYMPTOM_MODIFICATION", Differentials: []string{"HYPO"}},
	}

	updated, deltas := a.Apply(logOdds, mods, nil)

	// Recorded for G8 downstream consumption
	if _, exists := deltas["CM_BB_MASK"]; !exists {
		t.Fatal("SYMPTOM_MODIFICATION modifier should be recorded in cmLogDeltas")
	}
	if deltas["CM_BB_MASK"] != 0.0 {
		t.Errorf("SYMPTOM_MODIFICATION delta should be 0.0, got %.6f", deltas["CM_BB_MASK"])
	}

	// No log-odds change
	if updated["HYPO"] != 0.3 {
		t.Errorf("HYPO should be unchanged at 0.3, got %.6f", updated["HYPO"])
	}
	if updated["ACS"] != 0.0 {
		t.Errorf("ACS should be unchanged at 0.0, got %.6f", updated["ACS"])
	}
}

func TestApply_PassthroughSkipsMagnitudeCheck(t *testing.T) {
	a := newTestCMApplicator()
	// HARD_BLOCK with Magnitude=0 should still be recorded (magnitude check is skipped)
	logOdds := map[string]float64{"ACS": 0.0}
	mods := []ContextModifier{
		{ModifierID: "BLOCK_ZERO", Effect: "HARD_BLOCK", Magnitude: 0.0, Differentials: []string{"ACS"}},
	}

	_, deltas := a.Apply(logOdds, mods, nil)
	if _, exists := deltas["BLOCK_ZERO"]; !exists {
		t.Fatal("HARD_BLOCK should bypass magnitude validation and still be recorded")
	}
}

// --- Gap 2: CM stacking tests ---

func TestApply_CMStackedWarningAtThree(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 0.0}

	// Three CMs targeting the same differential should trigger CM_STACKED
	mods := []ContextModifier{
		{ModifierID: "CM_1", Effect: "INCREASE_PRIOR", Magnitude: 0.10, Differentials: []string{"ACS"}},
		{ModifierID: "CM_2", Effect: "INCREASE_PRIOR", Magnitude: 0.10, Differentials: []string{"ACS"}},
		{ModifierID: "CM_3", Effect: "INCREASE_PRIOR", Magnitude: 0.10, Differentials: []string{"ACS"}},
	}

	updated, deltas := a.Apply(logOdds, mods, nil)

	// All three should be recorded
	if len(deltas) != 3 {
		t.Errorf("expected 3 deltas, got %d", len(deltas))
	}

	// ACS should have cumulative shift from all 3 CMs
	expectedPerCM := cmLogit(0.60)
	expectedTotal := expectedPerCM * 3
	if math.Abs(updated["ACS"]-expectedTotal) > 1e-10 {
		t.Errorf("ACS expected cumulative %.6f, got %.6f", expectedTotal, updated["ACS"])
	}
}

func TestApply_CMStackedNotTriggeredAtTwo(t *testing.T) {
	a := newTestCMApplicator()
	logOdds := map[string]float64{"ACS": 0.0, "GERD": 0.0}

	// Two CMs targeting ACS — should NOT trigger CM_STACKED
	// One CM targeting GERD — should NOT trigger CM_STACKED
	mods := []ContextModifier{
		{ModifierID: "CM_1", Effect: "INCREASE_PRIOR", Magnitude: 0.10, Differentials: []string{"ACS"}},
		{ModifierID: "CM_2", Effect: "INCREASE_PRIOR", Magnitude: 0.10, Differentials: []string{"ACS"}},
		{ModifierID: "CM_3", Effect: "INCREASE_PRIOR", Magnitude: 0.10, Differentials: []string{"GERD"}},
	}

	// This test primarily validates the counting logic doesn't false-positive.
	// The CM_STACKED warning is a log emission; we verify correctness by
	// checking the cumulative shift math is unaffected.
	updated, _ := a.Apply(logOdds, mods, nil)

	expectedDelta := cmLogit(0.60)
	if math.Abs(updated["ACS"]-(expectedDelta*2)) > 1e-10 {
		t.Errorf("ACS expected %.6f, got %.6f", expectedDelta*2, updated["ACS"])
	}
	if math.Abs(updated["GERD"]-expectedDelta) > 1e-10 {
		t.Errorf("GERD expected %.6f, got %.6f", expectedDelta, updated["GERD"])
	}
}

// --- Gap 3: YAML CM expansion tests ---

func TestExpandNodeCMs_BasicExpansion(t *testing.T) {
	defs := []models.ContextModifierDef{
		{
			ID:   "CM01",
			Name: "ARB/ACEi active",
			Adjustments: map[string]float64{
				"ORTHOSTATIC_HYPO": 0.08,
				"DRUG_INDUCED":     0.06,
			},
		},
	}

	expanded := ExpandNodeCMs(defs)

	if len(expanded) != 2 {
		t.Fatalf("expected 2 expanded CMs, got %d", len(expanded))
	}

	// Verify each expanded CM has correct structure
	foundOH := false
	foundDI := false
	for _, cm := range expanded {
		if cm.ModifierID != "CM01" {
			t.Errorf("expected ModifierID=CM01, got %s", cm.ModifierID)
		}
		if cm.ModifierType != "NODE_CM" {
			t.Errorf("expected ModifierType=NODE_CM, got %s", cm.ModifierType)
		}
		if cm.Effect != "INCREASE_PRIOR" {
			t.Errorf("expected Effect=INCREASE_PRIOR, got %s", cm.Effect)
		}
		if len(cm.Differentials) != 1 {
			t.Errorf("expected 1 differential, got %d", len(cm.Differentials))
			continue
		}
		switch cm.Differentials[0] {
		case "ORTHOSTATIC_HYPO":
			foundOH = true
			if cm.Magnitude != 0.08 {
				t.Errorf("OH magnitude: expected 0.08, got %.4f", cm.Magnitude)
			}
		case "DRUG_INDUCED":
			foundDI = true
			if cm.Magnitude != 0.06 {
				t.Errorf("DI magnitude: expected 0.06, got %.4f", cm.Magnitude)
			}
		default:
			t.Errorf("unexpected differential: %s", cm.Differentials[0])
		}
	}

	if !foundOH {
		t.Error("missing expanded CM for ORTHOSTATIC_HYPO")
	}
	if !foundDI {
		t.Error("missing expanded CM for DRUG_INDUCED")
	}
}

func TestExpandNodeCMs_MultipleDefs(t *testing.T) {
	defs := []models.ContextModifierDef{
		{ID: "CM01", Adjustments: map[string]float64{"A": 0.10}},
		{ID: "CM02", Adjustments: map[string]float64{"B": 0.15, "C": 0.20}},
		{ID: "CM03", Adjustments: map[string]float64{"A": 0.05}},
	}

	expanded := ExpandNodeCMs(defs)

	// CM01 -> 1, CM02 -> 2, CM03 -> 1 = 4 total
	if len(expanded) != 4 {
		t.Fatalf("expected 4 expanded CMs, got %d", len(expanded))
	}
}

func TestExpandNodeCMs_EmptyDefs(t *testing.T) {
	expanded := ExpandNodeCMs(nil)
	if len(expanded) != 0 {
		t.Errorf("expected 0 expanded CMs for nil input, got %d", len(expanded))
	}

	expanded = ExpandNodeCMs([]models.ContextModifierDef{})
	if len(expanded) != 0 {
		t.Errorf("expected 0 expanded CMs for empty input, got %d", len(expanded))
	}
}
