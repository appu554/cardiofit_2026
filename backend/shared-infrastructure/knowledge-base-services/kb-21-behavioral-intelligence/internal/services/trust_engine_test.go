package services

import (
	"math"
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

// ---------------------------------------------------------------------------
// ComputeLoopTrust tests
// ---------------------------------------------------------------------------

func TestComputeLoopTrust_AllOnes(t *testing.T) {
	te := NewTrustEngine()
	score := te.ComputeLoopTrust(1.0, 1.0, 1.0, 1.0)
	if score != 1.0 {
		t.Errorf("all 1.0 inputs: got %.4f, want 1.0", score)
	}
}

func TestComputeLoopTrust_AllZeros(t *testing.T) {
	te := NewTrustEngine()
	score := te.ComputeLoopTrust(0.0, 0.0, 0.0, 0.0)
	if score != 0.0 {
		t.Errorf("all 0.0 inputs: got %.4f, want 0.0", score)
	}
}

func TestComputeLoopTrust_SingleZero_ZerosOut(t *testing.T) {
	te := NewTrustEngine()
	// One zero factor should zero the whole score (multiplicative)
	cases := []struct {
		name string
		a, d, p, ts float64
	}{
		{"zero adherence", 0.0, 1.0, 1.0, 1.0},
		{"zero data quality", 0.80, 0.0, 0.90, 1.0},
		{"zero phenotype", 0.80, 0.75, 0.0, 1.0},
		{"zero temporal", 0.80, 0.75, 0.90, 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			score := te.ComputeLoopTrust(tc.a, tc.d, tc.p, tc.ts)
			if score != 0.0 {
				t.Errorf("got %.4f, want 0.0", score)
			}
		})
	}
}

func TestComputeLoopTrust_TypicalPatient(t *testing.T) {
	te := NewTrustEngine()
	// adherence=0.85, HIGH quality=1.0, STEADY=0.90, STABLE=1.0
	score := te.ComputeLoopTrust(0.85, 1.0, 0.90, 1.0)
	expected := 0.85 * 1.0 * 0.90 * 1.0 // = 0.765
	if math.Abs(score-expected) > 0.001 {
		t.Errorf("typical patient: got %.4f, want %.4f", score, expected)
	}
}

func TestComputeLoopTrust_ClampAboveOne(t *testing.T) {
	te := NewTrustEngine()
	// Even if inputs were >1 (shouldn't happen, but safety)
	score := te.ComputeLoopTrust(1.5, 1.5, 1.0, 1.0)
	if score != 1.0 {
		t.Errorf("clamped above 1.0: got %.4f, want 1.0", score)
	}
}

func TestComputeLoopTrust_ClampBelowZero(t *testing.T) {
	te := NewTrustEngine()
	score := te.ComputeLoopTrust(-0.5, 1.0, 1.0, 1.0)
	if score != 0.0 {
		t.Errorf("clamped below 0.0: got %.4f, want 0.0", score)
	}
}

func TestComputeLoopTrust_DecliningDormantPatient(t *testing.T) {
	te := NewTrustEngine()
	// adherence=0.40, LOW=0.50, DECLINING=0.40, CRITICAL=0.40
	score := te.ComputeLoopTrust(0.40, 0.50, 0.40, 0.40)
	expected := 0.40 * 0.50 * 0.40 * 0.40 // = 0.032
	if math.Abs(score-expected) > 0.001 {
		t.Errorf("declining dormant: got %.4f, want %.4f", score, expected)
	}
}

// ---------------------------------------------------------------------------
// DataQualityWeight tests
// ---------------------------------------------------------------------------

func TestDataQualityWeight_AllLevels(t *testing.T) {
	te := NewTrustEngine()
	cases := []struct {
		quality  models.DataQuality
		expected float64
	}{
		{models.DataQualityHigh, 1.0},
		{models.DataQualityModerate, 0.75},
		{models.DataQualityLow, 0.50},
		{models.DataQuality("UNKNOWN"), 0.50}, // default fallback
	}
	for _, tc := range cases {
		t.Run(string(tc.quality), func(t *testing.T) {
			w := te.DataQualityWeight(tc.quality)
			if w != tc.expected {
				t.Errorf("DataQualityWeight(%q) = %.2f, want %.2f", tc.quality, w, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// PhenotypeWeight tests
// ---------------------------------------------------------------------------

func TestPhenotypeWeight_AllPhenotypes(t *testing.T) {
	te := NewTrustEngine()
	cases := []struct {
		phenotype models.BehavioralPhenotype
		expected  float64
	}{
		{models.PhenotypeChampion, 1.0},
		{models.PhenotypeSteady, 0.90},
		{models.PhenotypeSporadic, 0.65},
		{models.PhenotypeDeclining, 0.40},
		{models.PhenotypeDormant, 0.10},
		{models.PhenotypeChurned, 0.0},
		{models.BehavioralPhenotype("UNKNOWN"), 0.50}, // default fallback
	}
	for _, tc := range cases {
		t.Run(string(tc.phenotype), func(t *testing.T) {
			w := te.PhenotypeWeight(tc.phenotype)
			if w != tc.expected {
				t.Errorf("PhenotypeWeight(%q) = %.2f, want %.2f", tc.phenotype, w, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TemporalStability tests
// ---------------------------------------------------------------------------

func TestTemporalStability_AllTrends(t *testing.T) {
	te := NewTrustEngine()
	cases := []struct {
		trend    models.AdherenceTrend
		expected float64
	}{
		{models.TrendStable, 1.0},
		{models.TrendImproving, 1.0},
		{models.TrendDeclining, 0.70},
		{models.TrendCritical, 0.40},
		{models.AdherenceTrend(""), 1.0}, // default fallback
	}
	for _, tc := range cases {
		t.Run(string(tc.trend), func(t *testing.T) {
			s := te.TemporalStability(tc.trend)
			if s != tc.expected {
				t.Errorf("TemporalStability(%q) = %.2f, want %.2f", tc.trend, s, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RecommendAuthority tests
// ---------------------------------------------------------------------------

func TestRecommendAuthority_ThresholdBoundaries(t *testing.T) {
	te := NewTrustEngine()
	cases := []struct {
		score    float64
		expected string
	}{
		{1.0, "AUTO"},
		{0.75, "AUTO"},
		{0.749, "ASSISTED"},
		{0.55, "ASSISTED"},
		{0.549, "CONFIRM"},
		{0.35, "CONFIRM"},
		{0.349, "DISABLED"},
		{0.20, "DISABLED"},
		{0.0, "DISABLED"},
	}
	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			auth := te.RecommendAuthority(tc.score)
			if auth != tc.expected {
				t.Errorf("RecommendAuthority(%.3f) = %q, want %q", tc.score, auth, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// End-to-end composite trust → authority mapping
// ---------------------------------------------------------------------------

func TestLoopTrust_ChampionFullAuto(t *testing.T) {
	te := NewTrustEngine()
	// CHAMPION patient: adh=0.95, HIGH quality, CHAMPION phenotype, IMPROVING trend
	score := te.ComputeLoopTrust(
		0.95,
		te.DataQualityWeight(models.DataQualityHigh),
		te.PhenotypeWeight(models.PhenotypeChampion),
		te.TemporalStability(models.TrendImproving),
	)
	auth := te.RecommendAuthority(score)
	if auth != "AUTO" {
		t.Errorf("CHAMPION patient: trust=%.3f, authority=%q, want AUTO", score, auth)
	}
}

func TestLoopTrust_ChurnedDisabled(t *testing.T) {
	te := NewTrustEngine()
	// CHURNED patient: phenotype weight = 0.0 → score = 0 → DISABLED
	score := te.ComputeLoopTrust(
		0.50,
		te.DataQualityWeight(models.DataQualityModerate),
		te.PhenotypeWeight(models.PhenotypeChurned),
		te.TemporalStability(models.TrendStable),
	)
	if score != 0.0 {
		t.Errorf("CHURNED patient: trust=%.4f, want 0.0 (phenotype weight is 0)", score)
	}
	auth := te.RecommendAuthority(score)
	if auth != "DISABLED" {
		t.Errorf("CHURNED patient: authority=%q, want DISABLED", auth)
	}
}

func TestLoopTrust_SporadicDeclineConfirm(t *testing.T) {
	te := NewTrustEngine()
	// SPORADIC + DECLINING: moderate trust → CONFIRM authority
	score := te.ComputeLoopTrust(
		0.60,
		te.DataQualityWeight(models.DataQualityModerate),
		te.PhenotypeWeight(models.PhenotypeSporadic),
		te.TemporalStability(models.TrendDeclining),
	)
	// 0.60 * 0.75 * 0.65 * 0.70 = 0.20475
	auth := te.RecommendAuthority(score)
	if auth != "DISABLED" {
		t.Errorf("SPORADIC+DECLINING: trust=%.4f, authority=%q, want DISABLED", score, auth)
	}
}
