package services

import (
	"math"
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

// ---------------------------------------------------------------------------
// classifyResponse tests (CorrelationService method, but logic is pure once
// we provide the inputs directly)
// ---------------------------------------------------------------------------

func newCorrelationService() *CorrelationService {
	return &CorrelationService{minEventsForCorr: 10}
}

func TestClassifyResponse_Concordant(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.5 // HbA1c dropped by 0.5%
	got := cs.classifyResponse(0.80, models.TrendStable, &delta, 15)
	if got != models.ResponseConcordant {
		t.Errorf("high adherence + improving: got %q, want CONCORDANT", got)
	}
}

func TestClassifyResponse_Discordant(t *testing.T) {
	cs := newCorrelationService()
	delta := 0.1 // HbA1c went UP despite high adherence
	got := cs.classifyResponse(0.85, models.TrendStable, &delta, 20)
	if got != models.ResponseDiscordant {
		t.Errorf("high adherence + not improving: got %q, want DISCORDANT", got)
	}
}

func TestClassifyResponse_Discordant_FlatDelta(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.2 // Not enough improvement (needs < -0.3)
	got := cs.classifyResponse(0.75, models.TrendStable, &delta, 12)
	if got != models.ResponseDiscordant {
		t.Errorf("high adherence + flat delta (-0.2): got %q, want DISCORDANT", got)
	}
}

func TestClassifyResponse_BehavioralGap(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.5
	got := cs.classifyResponse(0.50, models.TrendDeclining, &delta, 15)
	if got != models.ResponseBehavioral {
		t.Errorf("low adherence: got %q, want BEHAVIORAL_GAP", got)
	}
}

func TestClassifyResponse_BehavioralGap_NoImprovement(t *testing.T) {
	cs := newCorrelationService()
	delta := 0.3
	got := cs.classifyResponse(0.40, models.TrendCritical, &delta, 15)
	if got != models.ResponseBehavioral {
		t.Errorf("low adherence + worsening: got %q, want BEHAVIORAL_GAP", got)
	}
}

func TestClassifyResponse_InsufficientData(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.5
	got := cs.classifyResponse(0.80, models.TrendStable, &delta, 5) // below minEventsForCorr=10
	if got != models.ResponseInsufficient {
		t.Errorf("insufficient events: got %q, want INSUFFICIENT_DATA", got)
	}
}

func TestClassifyResponse_NilDelta_HighAdherence(t *testing.T) {
	cs := newCorrelationService()
	got := cs.classifyResponse(0.80, models.TrendStable, nil, 15)
	// nil delta → improving=false → high adherence + not improving = DISCORDANT
	if got != models.ResponseDiscordant {
		t.Errorf("nil delta, high adherence: got %q, want DISCORDANT", got)
	}
}

func TestClassifyResponse_BoundaryAdherence(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.5
	// Boundary at 0.70
	got70 := cs.classifyResponse(0.70, models.TrendStable, &delta, 15)
	if got70 != models.ResponseConcordant {
		t.Errorf("adh=0.70 (boundary): got %q, want CONCORDANT", got70)
	}
	got69 := cs.classifyResponse(0.69, models.TrendStable, &delta, 15)
	if got69 != models.ResponseBehavioral {
		t.Errorf("adh=0.69 (below boundary): got %q, want BEHAVIORAL_GAP", got69)
	}
}

// ---------------------------------------------------------------------------
// computeCorrelationStrength tests
// ---------------------------------------------------------------------------

func TestCorrelationStrength_NilDelta(t *testing.T) {
	cs := newCorrelationService()
	got := cs.computeCorrelationStrength(0.80, nil)
	if got != 0 {
		t.Errorf("nil delta: got %.4f, want 0", got)
	}
}

func TestCorrelationStrength_HighAdherenceStrongDrop(t *testing.T) {
	cs := newCorrelationService()
	delta := -2.0 // Large HbA1c drop
	got := cs.computeCorrelationStrength(0.90, &delta)
	// 0.90 * |−2.0| / 2.0 = 0.90 — clamped to [0,1]
	if math.Abs(got-0.90) > 0.01 {
		t.Errorf("high adherence + strong drop: got %.4f, want ~0.90", got)
	}
}

func TestCorrelationStrength_LowAdherenceSmallDrop(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.3
	got := cs.computeCorrelationStrength(0.40, &delta)
	// 0.40 * 0.3 / 2.0 = 0.06
	if math.Abs(got-0.06) > 0.01 {
		t.Errorf("low adherence + small drop: got %.4f, want ~0.06", got)
	}
}

func TestCorrelationStrength_ClampedToOne(t *testing.T) {
	cs := newCorrelationService()
	delta := -5.0 // Extreme delta
	got := cs.computeCorrelationStrength(1.0, &delta)
	// 1.0 * 5.0 / 2.0 = 2.5 → clamped to 1.0
	if got != 1.0 {
		t.Errorf("clamped: got %.4f, want 1.0", got)
	}
}

// ---------------------------------------------------------------------------
// classifyConfidence tests
// ---------------------------------------------------------------------------

func TestClassifyConfidence_AllLevels(t *testing.T) {
	cs := newCorrelationService()
	cases := []struct {
		eventCount int
		expected   string
	}{
		{30, "HIGH"},
		{50, "HIGH"},
		{15, "MODERATE"},
		{29, "MODERATE"},
		{14, "LOW"},
		{0, "LOW"},
	}
	for _, tc := range cases {
		got := cs.classifyConfidence(tc.eventCount)
		if got != tc.expected {
			t.Errorf("classifyConfidence(%d) = %q, want %q", tc.eventCount, got, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// isCelebrationEligible tests
// ---------------------------------------------------------------------------

func TestCelebrationEligible_Concordant_HighAdherence_LargeDrop(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.5
	corr := models.OutcomeCorrelation{
		TreatmentResponseClass: models.ResponseConcordant,
		MeanAdherenceScore:     0.80,
		HbA1cDelta:             &delta,
	}
	if !cs.isCelebrationEligible(corr) {
		t.Error("should be celebration eligible")
	}
}

func TestCelebrationEligible_NotConcordant(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.5
	corr := models.OutcomeCorrelation{
		TreatmentResponseClass: models.ResponseDiscordant,
		MeanAdherenceScore:     0.80,
		HbA1cDelta:             &delta,
	}
	if cs.isCelebrationEligible(corr) {
		t.Error("DISCORDANT should NOT be celebration eligible")
	}
}

func TestCelebrationEligible_LowAdherence(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.5
	corr := models.OutcomeCorrelation{
		TreatmentResponseClass: models.ResponseConcordant,
		MeanAdherenceScore:     0.70, // below 0.75 threshold
		HbA1cDelta:             &delta,
	}
	if cs.isCelebrationEligible(corr) {
		t.Error("adherence 0.70 < 0.75 should NOT be eligible")
	}
}

func TestCelebrationEligible_SmallDelta(t *testing.T) {
	cs := newCorrelationService()
	delta := -0.2 // not < -0.3
	corr := models.OutcomeCorrelation{
		TreatmentResponseClass: models.ResponseConcordant,
		MeanAdherenceScore:     0.80,
		HbA1cDelta:             &delta,
	}
	if cs.isCelebrationEligible(corr) {
		t.Error("delta -0.2 (not < -0.3) should NOT be eligible")
	}
}

func TestCelebrationEligible_NilDelta(t *testing.T) {
	cs := newCorrelationService()
	corr := models.OutcomeCorrelation{
		TreatmentResponseClass: models.ResponseConcordant,
		MeanAdherenceScore:     0.90,
		HbA1cDelta:             nil,
	}
	if cs.isCelebrationEligible(corr) {
		t.Error("nil delta should NOT be eligible")
	}
}

// ---------------------------------------------------------------------------
// dominantTrend tests
// ---------------------------------------------------------------------------

func TestDominantTrend_CriticalWins(t *testing.T) {
	cs := newCorrelationService()
	states := []models.AdherenceState{
		{AdherenceTrend: models.TrendStable},
		{AdherenceTrend: models.TrendCritical},
		{AdherenceTrend: models.TrendImproving},
	}
	got := cs.dominantTrend(states)
	if got != models.TrendCritical {
		t.Errorf("got %q, want CRITICAL", got)
	}
}

func TestDominantTrend_DecliningWins(t *testing.T) {
	cs := newCorrelationService()
	states := []models.AdherenceState{
		{AdherenceTrend: models.TrendStable},
		{AdherenceTrend: models.TrendDeclining},
		{AdherenceTrend: models.TrendImproving},
	}
	got := cs.dominantTrend(states)
	if got != models.TrendDeclining {
		t.Errorf("got %q, want DECLINING", got)
	}
}

func TestDominantTrend_ImprovingWins(t *testing.T) {
	cs := newCorrelationService()
	states := []models.AdherenceState{
		{AdherenceTrend: models.TrendStable},
		{AdherenceTrend: models.TrendImproving},
	}
	got := cs.dominantTrend(states)
	if got != models.TrendImproving {
		t.Errorf("got %q, want IMPROVING", got)
	}
}

func TestDominantTrend_AllStable(t *testing.T) {
	cs := newCorrelationService()
	states := []models.AdherenceState{
		{AdherenceTrend: models.TrendStable},
		{AdherenceTrend: models.TrendStable},
	}
	got := cs.dominantTrend(states)
	if got != models.TrendStable {
		t.Errorf("got %q, want STABLE", got)
	}
}

func TestDominantTrend_Empty(t *testing.T) {
	cs := newCorrelationService()
	got := cs.dominantTrend(nil)
	if got != models.TrendStable {
		t.Errorf("empty: got %q, want STABLE (default)", got)
	}
}

// ---------------------------------------------------------------------------
// generateCelebrationMessage tests
// ---------------------------------------------------------------------------

func TestCelebrationMessage_NilDelta(t *testing.T) {
	cs := newCorrelationService()
	corr := models.OutcomeCorrelation{HbA1cDelta: nil}
	msg := cs.generateCelebrationMessage(corr)
	if msg != "" {
		t.Errorf("nil delta: got %q, want empty", msg)
	}
}

func TestCelebrationMessage_NonNilDelta(t *testing.T) {
	cs := newCorrelationService()
	delta := -1.2
	corr := models.OutcomeCorrelation{HbA1cDelta: &delta}
	msg := cs.generateCelebrationMessage(corr)
	if msg == "" {
		t.Error("non-nil delta should produce a celebration message")
	}
	// Should contain the absolute value "1.2"
	if len(msg) < 10 {
		t.Error("message too short")
	}
}
