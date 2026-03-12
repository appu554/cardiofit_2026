package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

// ---------------------------------------------------------------------------
// classifyDataQuality tests
// ---------------------------------------------------------------------------

func TestClassifyDataQuality_HighAbove80Percent(t *testing.T) {
	got := classifyDataQuality(80, 100)
	if got != models.DataQualityHigh {
		t.Errorf("80/100: got %q, want HIGH", got)
	}
}

func TestClassifyDataQuality_HighExact80(t *testing.T) {
	got := classifyDataQuality(8, 10)
	if got != models.DataQualityHigh {
		t.Errorf("8/10: got %q, want HIGH", got)
	}
}

func TestClassifyDataQuality_Moderate(t *testing.T) {
	got := classifyDataQuality(50, 100)
	if got != models.DataQualityModerate {
		t.Errorf("50/100: got %q, want MODERATE", got)
	}
}

func TestClassifyDataQuality_ModerateJustBelow80(t *testing.T) {
	got := classifyDataQuality(79, 100)
	if got != models.DataQualityModerate {
		t.Errorf("79/100: got %q, want MODERATE", got)
	}
}

func TestClassifyDataQuality_Low(t *testing.T) {
	got := classifyDataQuality(49, 100)
	if got != models.DataQualityLow {
		t.Errorf("49/100: got %q, want LOW", got)
	}
}

func TestClassifyDataQuality_LowZeroResponded(t *testing.T) {
	got := classifyDataQuality(0, 100)
	if got != models.DataQualityLow {
		t.Errorf("0/100: got %q, want LOW", got)
	}
}

func TestClassifyDataQuality_ZeroTotal(t *testing.T) {
	got := classifyDataQuality(0, 0)
	if got != models.DataQualityLow {
		t.Errorf("0/0: got %q, want LOW (zero total defaults to LOW)", got)
	}
}

// ---------------------------------------------------------------------------
// dataQualityWeight tests (adherence service helper)
// ---------------------------------------------------------------------------

func TestDataQualityWeightHelper_AllLevels(t *testing.T) {
	cases := []struct {
		quality  models.DataQuality
		expected float64
	}{
		{models.DataQualityHigh, 1.0},
		{models.DataQualityModerate, 0.7},
		{models.DataQualityLow, 0.4},
		{models.DataQuality("UNKNOWN"), 0.4},
	}
	for _, tc := range cases {
		t.Run(string(tc.quality), func(t *testing.T) {
			got := dataQualityWeight(tc.quality)
			if got != tc.expected {
				t.Errorf("dataQualityWeight(%q) = %.2f, want %.2f", tc.quality, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// barrierToReason tests
// ---------------------------------------------------------------------------

func TestBarrierToReason_AllMappings(t *testing.T) {
	cases := []struct {
		barrier  models.BarrierCode
		expected models.AdherenceReason
	}{
		{models.BarrierCost, models.ReasonCost},
		{models.BarrierSideEffects, models.ReasonSideEffect},
		{models.BarrierForgetfulness, models.ReasonForgot},
		{models.BarrierAccess, models.ReasonSupply},
		{models.BarrierCultural, models.ReasonUnknown},     // no explicit mapping
		{models.BarrierFasting, models.ReasonUnknown},       // no explicit mapping
		{models.BarrierKnowledge, models.ReasonUnknown},     // no explicit mapping
		{models.BarrierPolypharmacy, models.ReasonUnknown},  // no explicit mapping
		{models.BarrierCode(""), models.ReasonUnknown},      // empty
	}
	for _, tc := range cases {
		t.Run(string(tc.barrier), func(t *testing.T) {
			got := barrierToReason(tc.barrier)
			if got != tc.expected {
				t.Errorf("barrierToReason(%q) = %q, want %q", tc.barrier, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// trendSeverity tests
// ---------------------------------------------------------------------------

func TestTrendSeverity_Ordering(t *testing.T) {
	cases := []struct {
		trend    models.AdherenceTrend
		expected int
	}{
		{models.TrendImproving, 0},
		{models.TrendStable, 1},
		{models.TrendDeclining, 2},
		{models.TrendCritical, 3},
		{models.AdherenceTrend("UNKNOWN"), 1}, // default matches STABLE
	}
	for _, tc := range cases {
		t.Run(string(tc.trend), func(t *testing.T) {
			got := trendSeverity(tc.trend)
			if got != tc.expected {
				t.Errorf("trendSeverity(%q) = %d, want %d", tc.trend, got, tc.expected)
			}
		})
	}
}

func TestTrendSeverity_StrictlyIncreasing(t *testing.T) {
	trends := []models.AdherenceTrend{
		models.TrendImproving,
		models.TrendStable,
		models.TrendDeclining,
		models.TrendCritical,
	}
	for i := 1; i < len(trends); i++ {
		prev := trendSeverity(trends[i-1])
		curr := trendSeverity(trends[i])
		if curr <= prev {
			t.Errorf("trendSeverity(%q)=%d not > trendSeverity(%q)=%d",
				trends[i], curr, trends[i-1], prev)
		}
	}
}
