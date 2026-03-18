package services

import (
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

// EstimateInsulinSensitivity computes IS via HOMA-IR (if insulin available) or trajectory fallback.
func EstimateInsulinSensitivity(fbg, fastingInsulin float64, insulinAvailable bool) models.EstimatedVariable {
	if insulinAvailable && fastingInsulin > 0 {
		homaIR := (fbg * fastingInsulin) / 405.0
		class := "NORMAL"
		switch {
		case homaIR > 3:
			class = "SEVERE_RESISTANCE"
		case homaIR > 2:
			class = "MODERATE_RESISTANCE"
		case homaIR > 1:
			class = "MILD_RESISTANCE"
		}
		return models.EstimatedVariable{
			Value:          1.0 / homaIR,
			Classification: class,
			Confidence:     0.75,
			Method:         "HOMA_IR",
		}
	}

	class := "UNKNOWN"
	value := 0.5
	conf := 0.30
	if fbg > 130 {
		class = "LOW"
		value = 0.3
	} else if fbg > 100 {
		class = "MODERATE"
		value = 0.5
	} else {
		class = "NORMAL"
		value = 0.8
	}
	return models.EstimatedVariable{Value: value, Classification: class, Confidence: conf, Method: "TRAJECTORY_FALLBACK"}
}

// EstimateHepaticGlucoseOutput classifies HGO from dawn phenomenon and FBG/PPBG.
func EstimateHepaticGlucoseOutput(dawnPhenomenon bool, fbg, ppbg float64) models.EstimatedVariable {
	ratio := 1.0
	if ppbg > 0 {
		ratio = fbg / ppbg
	}

	class := "NORMAL"
	value := 0.5
	conf := 0.50

	if dawnPhenomenon && ratio > 1.1 {
		class = "HIGH"
		value = 0.8
		conf = 0.80
	} else if dawnPhenomenon || ratio > 1.0 {
		class = "MODERATE"
		value = 0.6
		conf = 0.65
	} else {
		class = "LOW"
		value = 0.3
		conf = 0.60
	}

	return models.EstimatedVariable{Value: value, Classification: class, Confidence: conf, Method: "DAWN_FBG_PPBG_RATIO"}
}

// EstimateMuscleMassProxy computes a composite 0-1 score.
func EstimateMuscleMassProxy(weightKg, proteinGPerKg, dailySteps float64, gripStrengthNormal bool) models.EstimatedVariable {
	score := 0.0
	score += clamp(proteinGPerKg/1.2, 0, 1) * 0.3
	score += clamp(dailySteps/8000, 0, 1) * 0.3
	bmiProxy := weightKg / (1.7 * 1.7)
	bmiScore := 1.0 - math.Abs(bmiProxy-22)/15
	score += clamp(bmiScore, 0, 1) * 0.2
	if gripStrengthNormal {
		score += 0.2
	}

	class := "LOW"
	switch {
	case score > 0.7:
		class = "HIGH"
	case score > 0.4:
		class = "MODERATE"
	}

	return models.EstimatedVariable{Value: clamp(score, 0, 1), Classification: class, Confidence: 0.45, Method: "COMPOSITE"}
}
