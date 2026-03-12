// Package calculator provides clinical score calculators.
package calculator

import (
	"context"
	"fmt"
	"time"

	"kb-8-calculator-service/internal/models"
)

// QSOFACalculator calculates qSOFA (quick SOFA) score for sepsis screening.
// Reference: Seymour CW, et al. JAMA. 2016;315(8):762-774. (Sepsis-3)
//
// qSOFA is a bedside screening tool that does NOT require lab tests.
// Score ≥ 2 indicates elevated risk of poor outcomes and should prompt:
// 1. Investigation for organ dysfunction (full SOFA)
// 2. Increased monitoring
// 3. Consider ICU-level care
type QSOFACalculator struct{}

// NewQSOFACalculator creates a new qSOFA calculator instance.
func NewQSOFACalculator() *QSOFACalculator {
	return &QSOFACalculator{}
}

// Calculate computes the qSOFA score from the given parameters.
// Each criterion scores 0 or 1, total 0-3.
// Score ≥ 2 is considered positive for sepsis risk.
func (c *QSOFACalculator) Calculate(ctx context.Context, params *models.QSOFAParams) (*models.QSOFAResult, error) {
	result := &models.QSOFAResult{
		Provenance: models.Provenance{
			CalculatorType: string(models.CalculatorQSOFA),
			Version:        "qSOFA-Sepsis3-2016",
			Formula:        "Sum of 3 criteria (0-1 each): Respiratory Rate ≥22 + Altered Mentation (GCS<15) + Systolic BP ≤100",
			Reference:      "Seymour CW, et al. Assessment of Clinical Criteria for Sepsis (Sepsis-3). JAMA. 2016;315(8):762-774",
			CalculatedAt:   time.Now().UTC(),
			InputsUsed:     []models.InputUsed{},
			DataQuality:    models.DataQualityComplete,
		},
	}

	var missingData []string
	inputsUsed := []models.InputUsed{}

	// Evaluate Respiratory Rate criterion (≥22 breaths/min)
	result.RespiratoryRateCriteria = c.evaluateRespiratoryRate(params, &inputsUsed, &missingData)

	// Evaluate Altered Mentation criterion (GCS < 15)
	result.AlteredMentationCriteria = c.evaluateAlteredMentation(params, &inputsUsed, &missingData)

	// Evaluate Systolic BP criterion (≤100 mmHg)
	result.SystolicBPCriteria = c.evaluateSystolicBP(params, &inputsUsed, &missingData)

	// Calculate total
	result.Total = 0
	if result.RespiratoryRateCriteria.Met {
		result.Total++
	}
	if result.AlteredMentationCriteria.Met {
		result.Total++
	}
	if result.SystolicBPCriteria.Met {
		result.Total++
	}

	// Determine if positive (≥2)
	result.Positive = result.Total >= 2

	// Set risk level
	if result.Positive {
		result.RiskLevel = models.RiskLevelHigh
	} else if result.Total == 1 {
		result.RiskLevel = models.RiskLevelModerate
	} else {
		result.RiskLevel = models.RiskLevelLow
	}

	// Generate interpretation and recommendation
	result.Interpretation = c.generateInterpretation(result, len(missingData))
	result.Recommendation = c.generateRecommendation(result)

	// Update provenance
	result.Provenance.InputsUsed = inputsUsed
	result.Provenance.MissingData = missingData

	if len(missingData) > 0 {
		if len(missingData) == 3 {
			result.Provenance.DataQuality = models.DataQualityIncomplete
		} else {
			result.Provenance.DataQuality = models.DataQualityPartial
		}
		result.Provenance.Caveats = append(result.Provenance.Caveats,
			fmt.Sprintf("Missing %d of 3 criteria - score may not fully reflect sepsis risk", len(missingData)))
	}

	result.Provenance.Caveats = append(result.Provenance.Caveats,
		"qSOFA is a screening tool, not diagnostic for sepsis",
		"Negative qSOFA does not rule out sepsis - clinical judgment required",
		"Positive qSOFA warrants full SOFA assessment for organ dysfunction")

	return result, nil
}

// evaluateRespiratoryRate checks if respiratory rate ≥ 22/min.
func (c *QSOFACalculator) evaluateRespiratoryRate(params *models.QSOFAParams, inputs *[]models.InputUsed, missing *[]string) models.QSOFACriterion {
	criterion := models.QSOFACriterion{
		Threshold: "≥22 breaths/min",
	}

	if params.RespiratoryRate == nil {
		criterion.DataAvailable = false
		criterion.Met = false
		*missing = append(*missing, "Respiratory rate")
		return criterion
	}

	criterion.DataAvailable = true
	criterion.Value = *params.RespiratoryRate
	criterion.Met = *params.RespiratoryRate >= 22

	*inputs = append(*inputs, models.InputUsed{
		Name:  "respiratory_rate",
		Value: *params.RespiratoryRate,
		Unit:  "breaths/min",
	})

	return criterion
}

// evaluateAlteredMentation checks for altered mental status.
// Can use either AlteredMentation flag directly or GCS < 15.
func (c *QSOFACalculator) evaluateAlteredMentation(params *models.QSOFAParams, inputs *[]models.InputUsed, missing *[]string) models.QSOFACriterion {
	criterion := models.QSOFACriterion{
		Threshold: "GCS < 15 or altered mentation",
	}

	// Check AlteredMentation flag first (direct input)
	if params.AlteredMentation != nil {
		criterion.DataAvailable = true
		criterion.Value = *params.AlteredMentation
		criterion.Met = *params.AlteredMentation

		*inputs = append(*inputs, models.InputUsed{
			Name:  "altered_mentation",
			Value: *params.AlteredMentation,
		})

		return criterion
	}

	// Fall back to GCS
	if params.GlasgowComaScale != nil {
		criterion.DataAvailable = true
		criterion.Value = *params.GlasgowComaScale
		criterion.Met = *params.GlasgowComaScale < 15

		*inputs = append(*inputs, models.InputUsed{
			Name:  "glasgow_coma_scale",
			Value: *params.GlasgowComaScale,
			Unit:  "points",
		})

		return criterion
	}

	criterion.DataAvailable = false
	criterion.Met = false
	*missing = append(*missing, "Altered mentation/GCS")
	return criterion
}

// evaluateSystolicBP checks if systolic BP ≤ 100 mmHg.
func (c *QSOFACalculator) evaluateSystolicBP(params *models.QSOFAParams, inputs *[]models.InputUsed, missing *[]string) models.QSOFACriterion {
	criterion := models.QSOFACriterion{
		Threshold: "≤100 mmHg",
	}

	if params.SystolicBP == nil {
		criterion.DataAvailable = false
		criterion.Met = false
		*missing = append(*missing, "Systolic blood pressure")
		return criterion
	}

	criterion.DataAvailable = true
	criterion.Value = *params.SystolicBP
	criterion.Met = *params.SystolicBP <= 100

	*inputs = append(*inputs, models.InputUsed{
		Name:  "systolic_blood_pressure",
		Value: *params.SystolicBP,
		Unit:  "mmHg",
	})

	return criterion
}

// generateInterpretation creates a clinical interpretation.
func (c *QSOFACalculator) generateInterpretation(result *models.QSOFAResult, missingCount int) string {
	var interpretation string

	// Build criteria summary
	var metCriteria []string
	if result.RespiratoryRateCriteria.Met && result.RespiratoryRateCriteria.DataAvailable {
		metCriteria = append(metCriteria, fmt.Sprintf("tachypnea (RR %v)", result.RespiratoryRateCriteria.Value))
	}
	if result.AlteredMentationCriteria.Met && result.AlteredMentationCriteria.DataAvailable {
		metCriteria = append(metCriteria, "altered mentation")
	}
	if result.SystolicBPCriteria.Met && result.SystolicBPCriteria.DataAvailable {
		metCriteria = append(metCriteria, fmt.Sprintf("hypotension (SBP %v mmHg)", result.SystolicBPCriteria.Value))
	}

	if result.Positive {
		interpretation = fmt.Sprintf("qSOFA POSITIVE (score %d/3). ", result.Total)
		interpretation += "Elevated risk of poor outcomes in patients with suspected infection. "
		if len(metCriteria) > 0 {
			interpretation += fmt.Sprintf("Criteria met: %s. ", formatCriteriaList(metCriteria))
		}
		interpretation += "Prompt assessment for sepsis and organ dysfunction recommended."
	} else if result.Total == 1 {
		interpretation = fmt.Sprintf("qSOFA score %d/3. ", result.Total)
		interpretation += "Single criterion met - "
		if len(metCriteria) > 0 {
			interpretation += fmt.Sprintf("%s. ", metCriteria[0])
		}
		interpretation += "Monitor closely for clinical deterioration."
	} else {
		interpretation = "qSOFA score 0/3. "
		interpretation += "No qSOFA criteria met at this time. "
		interpretation += "Continue clinical monitoring; qSOFA is a screening tool, not a diagnostic test."
	}

	if missingCount > 0 {
		interpretation += fmt.Sprintf(" Note: %d criteria could not be evaluated due to missing data.", missingCount)
	}

	return interpretation
}

// generateRecommendation provides clinical recommendations.
func (c *QSOFACalculator) generateRecommendation(result *models.QSOFAResult) string {
	if result.Positive {
		return "1) Assess for infection source 2) Calculate full SOFA score 3) Obtain lactate level 4) Consider early antibiotics if infection suspected 5) ICU consultation if unstable"
	}
	if result.Total == 1 {
		return "1) Continue monitoring vital signs 2) Re-assess qSOFA if clinical status changes 3) Maintain high index of suspicion for sepsis if infection suspected"
	}
	return "Continue routine monitoring. Re-evaluate if clinical concern for infection develops."
}

// formatCriteriaList formats a list of criteria for display.
func formatCriteriaList(criteria []string) string {
	if len(criteria) == 0 {
		return "none"
	}
	if len(criteria) == 1 {
		return criteria[0]
	}
	if len(criteria) == 2 {
		return criteria[0] + " and " + criteria[1]
	}
	result := ""
	for i, crit := range criteria {
		if i == len(criteria)-1 {
			result += "and " + crit
		} else {
			result += crit + ", "
		}
	}
	return result
}
