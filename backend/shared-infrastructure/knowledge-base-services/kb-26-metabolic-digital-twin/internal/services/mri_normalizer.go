package services

// India-specific population parameters for MRI z-score normalization.
// Sources: ICMR-NIN, NFHS-5, ADA/KDIGO/ESC guidelines.
// All parameters are risk-direction: higher value = higher z = higher risk.
//
// Spec §4.1: "Each signal is normalized to a risk z-score where 0 = population mean,
// positive = worse than mean, negative = better than mean."

// Population parameters: {mean, sd}
var (
	// Glucose domain
	paramsFBG        = [2]float64{95, 15}  // Indian population mean FBG ~95 mg/dL
	paramsPPBG       = [2]float64{130, 25} // Higher due to carb-heavy diet
	paramsHbA1cTrend = [2]float64{0, 0.2}  // Stable = 0, SD = 0.2%/quarter

	// Body composition (male)
	paramsWaistM      = [2]float64{85, 10} // IDF South Asian M threshold 90
	paramsWaistF      = [2]float64{76, 8}  // IDF South Asian F threshold 80
	paramsWeightTrend = [2]float64{0, 0.5} // Stable = 0 kg/month
	paramsMuscle      = [2]float64{12, 3}  // STS 30s count, EWGSOP2+AWGS

	// Cardiovascular
	paramsSBP      = [2]float64{125, 12} // Standard
	paramsSBPTrend = [2]float64{0, 5}    // Stable = 0 mmHg/4wk

	// Behavioral (inverted: higher value = LOWER risk)
	paramsSteps   = [2]float64{5000, 2500} // Indian urban avg ~3500
	paramsProtein = [2]float64{0.8, 0.2}   // ICMR RDA 0.8-1.0 g/kg/day
)

// NormalizeZScore computes a standard z-score: (value - mean) / sd.
func NormalizeZScore(value, mean, sd float64) float64 {
	if sd <= 0 {
		return 0
	}
	return (value - mean) / sd
}

// NormalizeTrend normalizes a trend value where positive = worsening.
func NormalizeTrend(trendValue, mean, sd float64) float64 {
	return NormalizeZScore(trendValue, mean, sd)
}

// --- Signal-specific normalizers (spec §4.1) ---

// NormalizeFBG converts FBG (mg/dL) to risk z-score.
func NormalizeFBG(fbg float64) float64 {
	return NormalizeZScore(fbg, paramsFBG[0], paramsFBG[1])
}

// NormalizePPBG converts PPBG (mg/dL) to risk z-score.
func NormalizePPBG(ppbg float64) float64 {
	return NormalizeZScore(ppbg, paramsPPBG[0], paramsPPBG[1])
}

// NormalizeHbA1cTrend converts HbA1c trend (%/quarter) to risk z-score.
func NormalizeHbA1cTrend(trend float64) float64 {
	return NormalizeTrend(trend, paramsHbA1cTrend[0], paramsHbA1cTrend[1])
}

// NormalizeWaistSexSpecific converts waist (cm) to risk z-score using sex-specific params.
func NormalizeWaistSexSpecific(waist float64, sex string) float64 {
	if sex == "F" || sex == "female" {
		return NormalizeZScore(waist, paramsWaistF[0], paramsWaistF[1])
	}
	return NormalizeZScore(waist, paramsWaistM[0], paramsWaistM[1])
}

// NormalizeWeightTrend converts weight trend (kg/month) to risk z-score.
func NormalizeWeightTrend(trend float64) float64 {
	return NormalizeTrend(trend, paramsWeightTrend[0], paramsWeightTrend[1])
}

// NormalizeWeightTrendBMI converts weight trend to risk z-score with BMI awareness.
// Spec Table 2: "Weight loss in BMI <22 is penalized, not rewarded (LS-15 alignment)."
// For BMI >= 22: positive trend (gaining) = positive z (bad), negative trend (losing) = negative z (good).
// For BMI < 22: polarity is INVERTED — losing weight when underweight is penalized.
func NormalizeWeightTrendBMI(trendKgPerMonth float64, bmi float64) float64 {
	z := NormalizeTrend(trendKgPerMonth, paramsWeightTrend[0], paramsWeightTrend[1])
	if bmi > 0 && bmi < 22.0 {
		return -z // invert: weight loss becomes risk, weight gain becomes benefit
	}
	return z
}

// NormalizeMuscleFunction converts 30s sit-to-stand count to risk z-score.
// INVERTED: higher count = lower risk (negative z).
func NormalizeMuscleFunction(stsCount float64) float64 {
	return -NormalizeZScore(stsCount, paramsMuscle[0], paramsMuscle[1])
}

// NormalizeSBP converts SBP (mmHg) to risk z-score.
func NormalizeSBP(sbp float64) float64 {
	return NormalizeZScore(sbp, paramsSBP[0], paramsSBP[1])
}

// NormalizeSBPTrend converts SBP trend (mmHg/4 weeks) to risk z-score.
func NormalizeSBPTrend(trend float64) float64 {
	return NormalizeTrend(trend, paramsSBPTrend[0], paramsSBPTrend[1])
}

// DippingToZScore converts BP dipping classification to z-score.
// Spec §4.1: Dipper=-1, Non-dipper=0, Reverse=+2
func DippingToZScore(class string) float64 {
	switch class {
	case "DIPPER":
		return -1.0
	case "NON_DIPPER":
		return 0.0
	case "REVERSE_DIPPER":
		return 2.0
	default:
		return 0.0
	}
}

// NormalizeSteps converts daily steps to risk z-score.
// INVERTED: higher steps = lower risk (negative z).
func NormalizeSteps(steps float64) float64 {
	return -NormalizeZScore(steps, paramsSteps[0], paramsSteps[1])
}

// NormalizeProtein converts protein intake (g/kg/day) to risk z-score.
// INVERTED: higher protein = lower risk (negative z).
func NormalizeProtein(gPerKgDay float64) float64 {
	return -NormalizeZScore(gPerKgDay, paramsProtein[0], paramsProtein[1])
}

// ComputeSleepZScore converts a 0-1 sleep quality score to a risk z-score.
// 1.0 (good) → z ≈ -1 (lower risk), 0.5 → z = 0, 0.0 (poor) → z ≈ +2 (higher risk).
func ComputeSleepZScore(quality float64) float64 {
	// Linear map: quality 1.0 → z=-1, quality 0.5 → z=0, quality 0.0 → z=+2
	return 2.0 - 3.0*clamp(quality, 0, 1)
}
