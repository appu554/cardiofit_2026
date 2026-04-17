package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ── helpers ──────────────────────────────────────────────────────────────────
// floatPtr is declared in target_status_test.go (same package).

func paiStringPtr(s string) *string { return &s }
func paiIntPtr(i int) *int          { return &i }

func testPAIConfig() *PAIConfig {
	return &PAIConfig{
		VelocityWeight:  0.30,
		ProximityWeight: 0.25,
		BehavioralWeight: 0.20,
		ContextWeight:   0.15,
		AttentionWeight: 0.10,
		// Velocity thresholds
		SevereDeclineSlope:            -2.0,
		ModerateDeclineSlope:          -1.0,
		MildDeclineSlope:              -0.3,
		StableSlope:                   0.3,
		AcceleratingDeclineMultiplier: 1.5,
		DeceleratingDeclineMultiplier: 0.7,
		ConcordantBonus:               15,
		PerAdditionalDomain:           5,
		ConfounderDampeningEnabled:    false,
		MaxVelocityDuringSeason:       60,
		// Proximity
		ProximityExponent: 2.0,
		// Behavioral
		BehavioralCessationDays:    5,
		BehavioralReducedThreshold: 0.50,
		BehavioralSlightlyReduced:  0.25,
		BehavioralCompoundBoth:     95,
		// Context
		ContextCKMStageBase: map[string]float64{
			"0": 0, "1": 5, "2": 10, "3": 20, "3a": 25, "3b": 35,
			"4": 50, "4a": 55, "4b": 60, "4c": 65,
		},
		ContextPostDischarge30d:    25,
		ContextAcuteIllness:        20,
		ContextRecentHypo:          15,
		ContextActiveSteroid:       10,
		ContextPolypharmacyElderly: 15,
		ContextPolypharmacyAge:     75,
		ContextPolypharmacyMeds:    5,
		ContextNYHAAmplifier: map[string]float64{
			"I": 1.0, "II": 1.1, "III": 1.3, "IV": 1.5,
		},
		ContextMaxScore: 100,
		// Attention
		AttentionCriticalDays: 90,
		AttentionHighDays:     60,
		AttentionModerateDays: 30,
		AttentionAdequateDays: 14,
		AttentionPerCard:      10,
		AttentionPerDayOldest: 3,
		AttentionCardCap:      50,
		// Tiers
		CriticalThreshold: 80,
		HighThreshold:     60,
		ModerateThreshold: 40,
		LowThreshold:      20,
		SignificantDelta:   10,
	}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestVelocity_SevereDecline(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		MHRICompositeSlope:      floatPtr(-2.5),
		ConcordantDeterioration: true,
		DomainsDeterioriating:   3,
	}

	score := ComputeVelocityScore(input, cfg)

	// slope -2.5 ≤ severe(-2.0) → base 100
	// concordant with 3 domains → +15 + (3-2)*5 = +20 → 120 clamped to 100
	// expect ≥85
	if score < 85 {
		t.Errorf("expected score ≥85 for severe decline + concordant 3 domains, got %.2f", score)
	}
	if score > 100 {
		t.Errorf("expected score ≤100 (clamped), got %.2f", score)
	}
}

func TestVelocity_AcceleratingDecline_Amplified(t *testing.T) {
	cfg := testPAIConfig()

	baseInput := models.PAIDimensionInput{
		MHRICompositeSlope: floatPtr(-1.5),
		SecondDerivative:   paiStringPtr("STABLE"),
	}
	accelInput := models.PAIDimensionInput{
		MHRICompositeSlope: floatPtr(-1.5),
		SecondDerivative:   paiStringPtr("ACCELERATING_DECLINE"),
	}

	baseScore := ComputeVelocityScore(baseInput, cfg)
	accelScore := ComputeVelocityScore(accelInput, cfg)

	if accelScore <= baseScore {
		t.Errorf("ACCELERATING_DECLINE should amplify score: accel=%.2f, base=%.2f", accelScore, baseScore)
	}
}

func TestVelocity_Improving_LowScore(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		MHRICompositeSlope: floatPtr(1.5),
	}

	score := ComputeVelocityScore(input, cfg)

	// slope +1.5 > stable(+0.3) → base 0, no bonuses → 0
	if score >= 15 {
		t.Errorf("expected score <15 for improving slope +1.5, got %.2f", score)
	}
}

func TestVelocity_Stable_ModerateScore(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		MHRICompositeSlope: floatPtr(-0.1),
	}

	score := ComputeVelocityScore(input, cfg)

	// slope -0.1 is between mild(-0.3) and stable(+0.3)
	// scaleLinear(-0.1, -0.3, 0.3, 30, 0) → ratio = 0.2/0.6 = 0.333 → 30 + 0.333*(-30) = 20
	if score < 0 || score > 30 {
		t.Errorf("expected score in [0, 30] for near-stable slope -0.1, got %.2f", score)
	}
}

func TestVelocity_NoData_Zero(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		MHRICompositeSlope: nil,
	}

	score := ComputeVelocityScore(input, cfg)

	if score != 0 {
		t.Errorf("expected score 0 for nil MHRICompositeSlope, got %.2f", score)
	}
}

func TestVelocity_SeasonalDampening(t *testing.T) {
	cfg := testPAIConfig()
	cfg.ConfounderDampeningEnabled = true

	input := models.PAIDimensionInput{
		MHRICompositeSlope: floatPtr(-1.8),
		SeasonalWindow:     true,
	}

	score := ComputeVelocityScore(input, cfg)

	// slope -1.8 between severe(-2.0) and moderate(-1.0)
	// scaleLinear(-1.8, -2.0, -1.0, 100, 60) → ratio = 0.2/1.0 = 0.2 → 100 + 0.2*(-40) = 92
	// seasonal cap at 60
	if score > 60 {
		t.Errorf("expected score ≤60 with seasonal dampening, got %.2f", score)
	}
	if score <= 0 {
		t.Errorf("expected score >0 for declining slope, got %.2f", score)
	}
}
