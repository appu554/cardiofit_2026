package pattern_detection

import "testing"

// ---------------------------------------------------------------------------
// Plan-verbatim tests (Task 10)
// ---------------------------------------------------------------------------

func TestBias_FlagsHighDisparityRatio(t *testing.T) {
	stratified := map[string]float64{
		"65-74": 0.60,
		"75-84": 0.55,
		"85+":   0.30, // disparity vs 65-74 = 2.0x
	}
	if !DetectBiasDisparity(stratified, 1.5) {
		t.Errorf("expected disparity flag")
	}
}

func TestBias_DoesNotFlagUniform(t *testing.T) {
	stratified := map[string]float64{"a": 0.5, "b": 0.55, "c": 0.52}
	if DetectBiasDisparity(stratified, 1.5) {
		t.Errorf("uniform stratification should not flag")
	}
}

// ---------------------------------------------------------------------------
// Augmentations
// ---------------------------------------------------------------------------

// TestBias_LessThanTwoStrataReturnsFalse guards against the degenerate case
// of a single-stratum input. With only one data point there is no comparative
// basis for disparity detection.
func TestBias_LessThanTwoStrataReturnsFalse(t *testing.T) {
	single := map[string]float64{"all": 0.60}
	if DetectBiasDisparity(single, 1.5) {
		t.Errorf("single-stratum input should return false (no disparity possible)")
	}
	empty := map[string]float64{}
	if DetectBiasDisparity(empty, 1.5) {
		t.Errorf("empty stratum map should return false")
	}
}

// TestBias_ZeroMinNonzeroMaxFlags exercises the divide-by-zero special case
// documented in DetectBiasDisparity: when minV ≤ 0 and maxV > 0 the ratio is
// effectively infinite, which always exceeds any finite threshold.
func TestBias_ZeroMinNonzeroMaxFlags(t *testing.T) {
	stratified := map[string]float64{
		"group-a": 0.60,
		"group-b": 0.0, // zero — infinite disparity against group-a
	}
	if !DetectBiasDisparity(stratified, 1.5) {
		t.Errorf("non-zero max against zero min should flag (infinite disparity)")
	}
}

// TestBias_AllZerosDoesNotFlag confirms that uniformly null outcomes (all
// strata at zero) are not flagged. Zero-against-zero is not disparity — it
// indicates uniform absence of signal, not differential treatment.
func TestBias_AllZerosDoesNotFlag(t *testing.T) {
	stratified := map[string]float64{"x": 0.0, "y": 0.0, "z": 0.0}
	if DetectBiasDisparity(stratified, 1.5) {
		t.Errorf("all-zero stratification should not flag (uniform null)")
	}
}
