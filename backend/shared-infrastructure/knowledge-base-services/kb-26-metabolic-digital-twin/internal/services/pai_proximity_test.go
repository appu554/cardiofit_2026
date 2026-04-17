package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestProximity_EGFRNearThreshold(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		CurrentEGFR: floatPtr(32), // 2 from danger threshold 30
	}

	score := ComputeProximityScore(input, cfg)

	if score < 80 {
		t.Errorf("expected score ≥80 for eGFR 32 (near danger 30), got %.2f", score)
	}
}

func TestProximity_EGFRFarFromThreshold(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		CurrentEGFR: floatPtr(75), // far from danger
	}

	score := ComputeProximityScore(input, cfg)

	if score >= 15 {
		t.Errorf("expected score <15 for eGFR 75 (far from danger), got %.2f", score)
	}
}

func TestProximity_MultipleMetricsNearThreshold(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		CurrentEGFR:      floatPtr(33),  // near danger 30
		CurrentSBP:       floatPtr(172), // near danger 180
		CurrentPotassium: floatPtr(5.4), // past warning 5.5? no, 5.4 < 5.5 so safe. Wait — 5.4 is below warning 5.5 so safe zone for "above" direction.
	}
	// Actually potassium 5.4 < 5.5 warning → safe zone → 0.
	// But eGFR 33 + SBP 172 should still compound to ≥70.
	// eGFR 33: fraction = (45-33)/(45-30) = 12/15 = 0.8, scaled = 0.64, score = 50 + 0.64*50 = 82
	// SBP 172: fraction = (172-160)/(180-160) = 12/20 = 0.6, scaled = 0.36, score = 60 + 0.36*40 = 74.4
	// max = 82, secondary sum = 74.4, final = 82 + 0.2*74.4 = 96.88
	// Let's use potassium 5.7 instead to be in warning zone
	input.CurrentPotassium = floatPtr(5.7)

	score := ComputeProximityScore(input, cfg)

	if score < 70 {
		t.Errorf("expected compound score ≥70 for multiple near-threshold metrics, got %.2f", score)
	}
}

func TestProximity_ExponentialScaling(t *testing.T) {
	cfg := testPAIConfig()

	// eGFR 31.5: 90% of danger distance covered
	// fraction = (45-31.5)/(45-30) = 13.5/15 = 0.9, scaled = 0.9^2 = 0.81
	inputNear := models.PAIDimensionInput{
		CurrentEGFR: floatPtr(31.5),
	}

	// eGFR 37.5: 50% of danger distance covered
	// fraction = (45-37.5)/(45-30) = 7.5/15 = 0.5, scaled = 0.5^2 = 0.25
	inputMid := models.PAIDimensionInput{
		CurrentEGFR: floatPtr(37.5),
	}

	scoreNear := ComputeProximityScore(inputNear, cfg)
	scoreMid := ComputeProximityScore(inputMid, cfg)

	if scoreMid == 0 {
		t.Fatal("mid-range score should be >0 for eGFR 37.5")
	}

	// Exponential scaling means the incremental score above the warning
	// floor grows nonlinearly. With exponent=2: scaled fractions are
	// 0.81 vs 0.25 → ratio of incremental contributions >2×.
	// Score = ScoreAtWarning + scaledFraction * (ScoreAtDanger - ScoreAtWarning)
	// Incremental above warning: near = 0.81*50 = 40.5, mid = 0.25*50 = 12.5
	incrementNear := scoreNear - 50 // 50 is ScoreAtWarning for eGFR
	incrementMid := scoreMid - 50

	if incrementMid <= 0 {
		t.Fatal("mid-range incremental score should be >0")
	}

	ratio := incrementNear / incrementMid
	if ratio <= 2.0 {
		t.Errorf("expected exponential scaling: incremental ratio >2× between 90%% and 50%% danger distance, got %.2f (near=%.2f, mid=%.2f)",
			ratio, scoreNear, scoreMid)
	}
}

func TestProximity_AcuteWeightGain_HFOnly(t *testing.T) {
	cfg := testPAIConfig()

	// HF patient (CKM 4c) with 2.5kg gain
	inputHF := models.PAIDimensionInput{
		CurrentWeight:     floatPtr(82.5),
		PreviousWeight72h: floatPtr(80.0),
		CKMStage:          "4c",
	}

	// Non-HF patient (CKM 2) with same 2.5kg gain
	inputNonHF := models.PAIDimensionInput{
		CurrentWeight:     floatPtr(82.5),
		PreviousWeight72h: floatPtr(80.0),
		CKMStage:          "2",
	}

	scoreHF := ComputeProximityScore(inputHF, cfg)
	scoreNonHF := ComputeProximityScore(inputNonHF, cfg)

	if scoreHF <= scoreNonHF {
		t.Errorf("HF patient (CKM 4c) weight gain score should exceed non-HF: HF=%.2f, nonHF=%.2f",
			scoreHF, scoreNonHF)
	}
	if scoreHF < 40 {
		t.Errorf("expected HF weight gain score ≥40, got %.2f", scoreHF)
	}
}

func TestProximity_NoData_Zero(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{}

	score := ComputeProximityScore(input, cfg)

	if score != 0 {
		t.Errorf("expected score 0 for empty input, got %.2f", score)
	}
}
