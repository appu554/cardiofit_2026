package services

import "kb-patient-profile/internal/models"

// computeGlycaemicScore returns 0-6 glycaemic domain score.
// +2: FBG trend WORSENING
// +2: Glucose CV% > 36%
// +2: HbA1c rise >= 1.5% from previous
func computeGlycaemicScore(fbg *models.FBGTracking, hba1cCurrent, hba1cPrev float64) int {
	score := 0
	if fbg != nil && fbg.Trend == "WORSENING" {
		score += 2
	}
	if fbg != nil && fbg.CV30d > 36.0 {
		score += 2
	}
	if hba1cPrev > 0 && hba1cCurrent-hba1cPrev >= 1.5 {
		score += 2
	}
	if score > 6 {
		score = 6
	}
	return score
}

// computeRenalScore returns 0-6 renal domain score.
// +2: eGFR slope < -5 mL/min/year (rapid decline)
// +2: ACR trend WORSENING or category A3
// +2: Creatinine rise > 20% (unexplained)
func computeRenalScore(egfrSlope float64, acrTrend, acrCategory string, creatRisePct float64) int {
	score := 0
	if egfrSlope < -5.0 {
		score += 2
	}
	if acrTrend == "WORSENING" || acrCategory == "A3" {
		score += 2
	}
	if creatRisePct > 20.0 {
		score += 2
	}
	if score > 6 {
		score = 6
	}
	return score
}

// cdiRiskLevel maps total CDI score to risk level.
func cdiRiskLevel(score int) string {
	switch {
	case score >= 17:
		return "CRITICAL"
	case score >= 13:
		return "HIGH"
	case score >= 7:
		return "MODERATE"
	default:
		return "LOW"
	}
}
