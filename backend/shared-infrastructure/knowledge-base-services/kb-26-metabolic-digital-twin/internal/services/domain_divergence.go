package services

import (
	"fmt"
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

// detectDivergences finds pairs of domains moving in opposite directions.
// Method on TrajectoryEngine so it can read divergence thresholds from config.
func (e *TrajectoryEngine) detectDivergences(slopes map[models.MHRIDomain]models.DomainSlope) []models.DivergencePattern {
	var divergences []models.DivergencePattern
	domains := models.AllMHRIDomains

	improvingThreshold := e.thresholds.Divergence.MinImprovingSlope
	decliningThreshold := e.thresholds.Divergence.MinDecliningSlope
	minRate := e.thresholds.Divergence.MinDivergenceRate

	for i := 0; i < len(domains); i++ {
		for j := i + 1; j < len(domains); j++ {
			slopeA := slopes[domains[i]]
			slopeB := slopes[domains[j]]

			var improving, declining models.DomainSlope
			if slopeA.SlopePerDay > improvingThreshold && slopeB.SlopePerDay < decliningThreshold {
				improving = slopeA
				improving.Domain = domains[i]
				declining = slopeB
				declining.Domain = domains[j]
			} else if slopeB.SlopePerDay > improvingThreshold && slopeA.SlopePerDay < decliningThreshold {
				improving = slopeB
				improving.Domain = domains[j]
				declining = slopeA
				declining.Domain = domains[i]
			} else {
				continue
			}

			divergenceRate := math.Abs(improving.SlopePerDay) + math.Abs(declining.SlopePerDay)
			if divergenceRate < minRate {
				continue
			}

			divergences = append(divergences, models.DivergencePattern{
				ImprovingDomain: improving.Domain,
				DecliningDomain: declining.Domain,
				ImprovingSlope:  improving.SlopePerDay,
				DecliningSlope:  declining.SlopePerDay,
				DivergenceRate:  roundTo3(divergenceRate),
				ClinicalConcern: fmt.Sprintf("%s improving while %s declining — therapeutic attention may be misdirected",
					improving.Domain, declining.Domain),
				PossibleMechanism: inferDivergenceMechanism(improving.Domain, declining.Domain),
			})
		}
	}

	return divergences
}

// inferDivergenceMechanism remains a free function — no config dependency.
func inferDivergenceMechanism(improving, declining models.MHRIDomain) string {
	key := string(improving) + "_" + string(declining)
	mechanisms := map[string]string{
		"GLUCOSE_CARDIO": "Glycaemic therapy may lack hemodynamic benefit, or antihypertensive review needed. " +
			"Consider SGLT2i (dual glucose + BP benefit) or adding dedicated antihypertensive.",
		"CARDIO_GLUCOSE": "BP medications may be worsening glycaemic control (e.g., thiazide raising glucose, " +
			"beta-blocker masking hypoglycaemia). Review cross-domain drug effects.",
		"GLUCOSE_BEHAVIORAL": "Glycaemic markers improving on medication but patient disengaging from self-management. " +
			"Improvement may not sustain without behavioral re-engagement.",
		"BEHAVIORAL_GLUCOSE": "Patient engaged and self-managing but glycaemic control worsening — suggests " +
			"medication inadequacy rather than adherence problem. Intensify pharmacotherapy.",
		"CARDIO_BEHAVIORAL": "BP improving but engagement declining — may indicate medication working but patient " +
			"developing complacency. Monitor for future adherence-related BP rebound.",
		"BEHAVIORAL_CARDIO": "Patient engaged but cardiovascular metrics worsening despite adherence — " +
			"suggests medication resistance, secondary hypertension workup, or emerging cardiac pathology.",
		"GLUCOSE_BODY_COMP": "Glycaemic control improving while body composition worsening — " +
			"check for insulin-driven weight gain or thiazolidinedione fluid retention.",
		"BODY_COMP_GLUCOSE": "Body composition improving (weight loss) but glucose worsening — " +
			"paradoxical in T2DM. Investigate: stress hyperglycaemia, steroid use, pancreatic pathology.",
		"CARDIO_BODY_COMP": "Cardiovascular metrics improving while body composition declining — " +
			"may indicate effective medication but dietary non-adherence.",
		"BODY_COMP_CARDIO": "Weight management improving but CV worsening — " +
			"consider sleep apnea, endocrine causes of hypertension, or medication interaction.",
	}

	if m, ok := mechanisms[key]; ok {
		return m
	}
	return "Domain divergence detected — clinical review recommended to identify cause and adjust therapy."
}
