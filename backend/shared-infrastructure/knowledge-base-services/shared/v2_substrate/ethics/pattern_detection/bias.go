package pattern_detection

// DetectBiasDisparity identifies statistically significant outcome disparity
// across demographic strata per Guidelines §7.2.
//
// stratified maps stratum label → metric value (e.g. acceptance rate, mean
// appropriateness score). ratioThreshold is the maximum permissible ratio of
// the highest-performing stratum to the lowest-performing stratum before a
// disparity is flagged (default 1.5 per the Guidelines).
//
// The function returns false immediately when fewer than two strata are
// provided — a single-stratum input cannot exhibit disparity by definition.
//
// Division-by-zero / zero-value handling: when the minimum stratum value is
// ≤ 0 the ratio is undefined or infinite. In that case the function returns
// true if and only if the maximum value is > 0, capturing the semantics of
// "any non-zero outcome against a zero outcome is infinite disparity". When
// both max and min are zero the outcomes are uniformly null — no disparity is
// flagged (returns false).
//
// Boundary behaviour: the ratio comparison uses ≥ ratioThreshold, so a ratio
// exactly equal to the threshold DOES flag.
//
// This is a foundation implementation (Phase 1c). Full equity-audit
// stratification by age band, sex, frailty tier, CALD background,
// socioeconomic indicator, facility type, and geography is deferred to
// Phase 2+.
func DetectBiasDisparity(stratified map[string]float64, ratioThreshold float64) bool {
	if len(stratified) < 2 {
		return false
	}
	var maxV, minV float64 = -1, -1
	for _, v := range stratified {
		if maxV < 0 || v > maxV {
			maxV = v
		}
		if minV < 0 || v < minV {
			minV = v
		}
	}
	if minV <= 0 {
		return maxV > 0 // any non-zero against zero is infinite disparity
	}
	return (maxV / minV) >= ratioThreshold
}
