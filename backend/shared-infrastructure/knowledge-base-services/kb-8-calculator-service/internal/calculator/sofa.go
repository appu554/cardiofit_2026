// Package calculator provides clinical score calculators.
package calculator

import (
	"context"
	"fmt"
	"time"

	"kb-8-calculator-service/internal/models"
)

// SOFACalculator calculates SOFA (Sequential Organ Failure Assessment) score.
// Reference: Vincent JL, et al. Intensive Care Med. 1996;22(7):707-10.
// Updated: Third International Consensus Definitions for Sepsis (Sepsis-3), 2016.
type SOFACalculator struct{}

// NewSOFACalculator creates a new SOFA calculator instance.
func NewSOFACalculator() *SOFACalculator {
	return &SOFACalculator{}
}

// Calculate computes the SOFA score from the given parameters.
// Each organ system scores 0-4, total 0-24.
// Higher scores indicate greater severity and mortality risk.
func (c *SOFACalculator) Calculate(ctx context.Context, params *models.SOFAParams) (*models.SOFAResult, error) {
	result := &models.SOFAResult{
		Provenance: models.Provenance{
			CalculatorType: string(models.CalculatorSOFA),
			Version:        "SOFA-1996-Updated",
			Formula:        "Sum of 6 organ system scores (0-4 each): Respiration + Coagulation + Liver + Cardiovascular + CNS + Renal",
			Reference:      "Vincent JL, et al. Intensive Care Med. 1996;22(7):707-10; Singer M, et al. JAMA. 2016;315(8):801-810",
			CalculatedAt:   time.Now().UTC(),
			InputsUsed:     []models.InputUsed{},
			DataQuality:    models.DataQualityComplete,
		},
	}

	var missingData []string
	inputsUsed := []models.InputUsed{}

	// Calculate Respiratory component (PaO2/FiO2 ratio)
	result.Respiration = c.scoreRespiration(params, &inputsUsed, &missingData)

	// Calculate Coagulation component (Platelets)
	result.Coagulation = c.scoreCoagulation(params, &inputsUsed, &missingData)

	// Calculate Liver component (Bilirubin)
	result.Liver = c.scoreLiver(params, &inputsUsed, &missingData)

	// Calculate Cardiovascular component (MAP or vasopressor)
	result.Cardiovascular = c.scoreCardiovascular(params, &inputsUsed, &missingData)

	// Calculate CNS component (GCS)
	result.CNS = c.scoreCNS(params, &inputsUsed, &missingData)

	// Calculate Renal component (Creatinine or urine output)
	result.Renal = c.scoreRenal(params, &inputsUsed, &missingData)

	// Calculate total
	result.Total = result.Respiration.Score + result.Coagulation.Score +
		result.Liver.Score + result.Cardiovascular.Score +
		result.CNS.Score + result.Renal.Score

	// Determine severity and mortality risk
	result.RiskLevel = c.determineRiskLevel(result.Total)
	result.MortalityRisk = c.estimateMortality(result.Total)
	result.Interpretation = c.generateInterpretation(result)

	// Update provenance
	result.Provenance.InputsUsed = inputsUsed
	result.Provenance.MissingData = missingData

	if len(missingData) > 0 {
		if len(missingData) >= 3 {
			result.Provenance.DataQuality = models.DataQualityIncomplete
		} else {
			result.Provenance.DataQuality = models.DataQualityPartial
		}
		result.Provenance.Caveats = append(result.Provenance.Caveats,
			fmt.Sprintf("Missing data for %d organ system(s) - score may underestimate severity", len(missingData)))
	}

	return result, nil
}

// scoreRespiration calculates the respiratory SOFA component.
// Based on PaO2/FiO2 ratio (mmHg).
// Score 0: >= 400
// Score 1: < 400
// Score 2: < 300
// Score 3: < 200 with respiratory support
// Score 4: < 100 with respiratory support
func (c *SOFACalculator) scoreRespiration(params *models.SOFAParams, inputs *[]models.InputUsed, missing *[]string) models.SOFAComponent {
	component := models.SOFAComponent{
		InputUnit: "mmHg",
	}

	if params.PaO2FiO2Ratio == nil {
		component.DataAvailable = false
		component.Score = 0 // Default to 0 when missing
		*missing = append(*missing, "PaO2/FiO2 ratio")
		return component
	}

	component.DataAvailable = true
	component.InputValue = *params.PaO2FiO2Ratio
	*inputs = append(*inputs, models.InputUsed{
		Name:  "pao2_fio2_ratio",
		Value: *params.PaO2FiO2Ratio,
		Unit:  "mmHg",
	})

	ratio := *params.PaO2FiO2Ratio
	onVent := params.OnMechanicalVentilation

	switch {
	case ratio >= 400:
		component.Score = 0
	case ratio >= 300:
		component.Score = 1
	case ratio >= 200:
		component.Score = 2
	case ratio >= 100:
		if onVent {
			component.Score = 3
		} else {
			component.Score = 2 // Without vent, max score is 2
		}
	default: // < 100
		if onVent {
			component.Score = 4
		} else {
			component.Score = 2 // Without vent, max score is 2
		}
	}

	return component
}

// scoreCoagulation calculates the coagulation SOFA component.
// Based on Platelet count (×10³/µL).
// Score 0: >= 150
// Score 1: < 150
// Score 2: < 100
// Score 3: < 50
// Score 4: < 20
func (c *SOFACalculator) scoreCoagulation(params *models.SOFAParams, inputs *[]models.InputUsed, missing *[]string) models.SOFAComponent {
	component := models.SOFAComponent{
		InputUnit: "×10³/µL",
	}

	if params.Platelets == nil {
		component.DataAvailable = false
		component.Score = 0
		*missing = append(*missing, "Platelets")
		return component
	}

	component.DataAvailable = true
	component.InputValue = *params.Platelets
	*inputs = append(*inputs, models.InputUsed{
		Name:  "platelets",
		Value: *params.Platelets,
		Unit:  "×10³/µL",
	})

	platelets := *params.Platelets
	switch {
	case platelets >= 150:
		component.Score = 0
	case platelets >= 100:
		component.Score = 1
	case platelets >= 50:
		component.Score = 2
	case platelets >= 20:
		component.Score = 3
	default:
		component.Score = 4
	}

	return component
}

// scoreLiver calculates the liver SOFA component.
// Based on Bilirubin (mg/dL).
// Score 0: < 1.2
// Score 1: 1.2-1.9
// Score 2: 2.0-5.9
// Score 3: 6.0-11.9
// Score 4: >= 12.0
func (c *SOFACalculator) scoreLiver(params *models.SOFAParams, inputs *[]models.InputUsed, missing *[]string) models.SOFAComponent {
	component := models.SOFAComponent{
		InputUnit: "mg/dL",
	}

	if params.Bilirubin == nil {
		component.DataAvailable = false
		component.Score = 0
		*missing = append(*missing, "Bilirubin")
		return component
	}

	component.DataAvailable = true
	component.InputValue = *params.Bilirubin
	*inputs = append(*inputs, models.InputUsed{
		Name:  "bilirubin",
		Value: *params.Bilirubin,
		Unit:  "mg/dL",
	})

	bili := *params.Bilirubin
	switch {
	case bili < 1.2:
		component.Score = 0
	case bili < 2.0:
		component.Score = 1
	case bili < 6.0:
		component.Score = 2
	case bili < 12.0:
		component.Score = 3
	default:
		component.Score = 4
	}

	return component
}

// scoreCardiovascular calculates the cardiovascular SOFA component.
// Based on MAP or vasopressor requirements.
// Score 0: MAP >= 70 mmHg
// Score 1: MAP < 70 mmHg
// Score 2: Dopamine <= 5 or dobutamine (any dose)
// Score 3: Dopamine > 5 or epinephrine <= 0.1 or norepinephrine <= 0.1
// Score 4: Dopamine > 15 or epinephrine > 0.1 or norepinephrine > 0.1
func (c *SOFACalculator) scoreCardiovascular(params *models.SOFAParams, inputs *[]models.InputUsed, missing *[]string) models.SOFAComponent {
	component := models.SOFAComponent{
		InputUnit: "mmHg or µg/kg/min",
	}

	// Check for vasopressor use first (highest priority for scoring)
	if params.NorepinephrineDose != nil && *params.NorepinephrineDose > 0 {
		component.DataAvailable = true
		component.InputValue = fmt.Sprintf("Norepinephrine %.2f µg/kg/min", *params.NorepinephrineDose)
		*inputs = append(*inputs, models.InputUsed{
			Name:  "norepinephrine_dose",
			Value: *params.NorepinephrineDose,
			Unit:  "µg/kg/min",
		})
		if *params.NorepinephrineDose > 0.1 {
			component.Score = 4
		} else {
			component.Score = 3
		}
		return component
	}

	if params.EpinephrineDose != nil && *params.EpinephrineDose > 0 {
		component.DataAvailable = true
		component.InputValue = fmt.Sprintf("Epinephrine %.2f µg/kg/min", *params.EpinephrineDose)
		*inputs = append(*inputs, models.InputUsed{
			Name:  "epinephrine_dose",
			Value: *params.EpinephrineDose,
			Unit:  "µg/kg/min",
		})
		if *params.EpinephrineDose > 0.1 {
			component.Score = 4
		} else {
			component.Score = 3
		}
		return component
	}

	if params.DopamineDose != nil && *params.DopamineDose > 0 {
		component.DataAvailable = true
		component.InputValue = fmt.Sprintf("Dopamine %.1f µg/kg/min", *params.DopamineDose)
		*inputs = append(*inputs, models.InputUsed{
			Name:  "dopamine_dose",
			Value: *params.DopamineDose,
			Unit:  "µg/kg/min",
		})
		switch {
		case *params.DopamineDose > 15:
			component.Score = 4
		case *params.DopamineDose > 5:
			component.Score = 3
		default:
			component.Score = 2
		}
		return component
	}

	if params.DobutamineDose != nil && *params.DobutamineDose > 0 {
		component.DataAvailable = true
		component.InputValue = fmt.Sprintf("Dobutamine %.1f µg/kg/min", *params.DobutamineDose)
		*inputs = append(*inputs, models.InputUsed{
			Name:  "dobutamine_dose",
			Value: *params.DobutamineDose,
			Unit:  "µg/kg/min",
		})
		component.Score = 2
		return component
	}

	// Fall back to MAP
	if params.MAP != nil {
		component.DataAvailable = true
		component.InputValue = *params.MAP
		component.InputUnit = "mmHg"
		*inputs = append(*inputs, models.InputUsed{
			Name:  "mean_arterial_pressure",
			Value: *params.MAP,
			Unit:  "mmHg",
		})
		if *params.MAP >= 70 {
			component.Score = 0
		} else {
			component.Score = 1
		}
		return component
	}

	component.DataAvailable = false
	component.Score = 0
	*missing = append(*missing, "MAP or vasopressor data")
	return component
}

// scoreCNS calculates the CNS SOFA component.
// Based on Glasgow Coma Scale.
// Score 0: GCS 15
// Score 1: GCS 13-14
// Score 2: GCS 10-12
// Score 3: GCS 6-9
// Score 4: GCS < 6
func (c *SOFACalculator) scoreCNS(params *models.SOFAParams, inputs *[]models.InputUsed, missing *[]string) models.SOFAComponent {
	component := models.SOFAComponent{
		InputUnit: "GCS points",
	}

	if params.GlasgowComaScale == nil {
		component.DataAvailable = false
		component.Score = 0
		*missing = append(*missing, "Glasgow Coma Scale")
		return component
	}

	component.DataAvailable = true
	component.InputValue = *params.GlasgowComaScale
	*inputs = append(*inputs, models.InputUsed{
		Name:  "glasgow_coma_scale",
		Value: *params.GlasgowComaScale,
		Unit:  "points",
	})

	gcs := *params.GlasgowComaScale
	switch {
	case gcs >= 15:
		component.Score = 0
	case gcs >= 13:
		component.Score = 1
	case gcs >= 10:
		component.Score = 2
	case gcs >= 6:
		component.Score = 3
	default:
		component.Score = 4
	}

	return component
}

// scoreRenal calculates the renal SOFA component.
// Based on Creatinine (mg/dL) or urine output (mL/day).
// Score 0: Cr < 1.2
// Score 1: Cr 1.2-1.9
// Score 2: Cr 2.0-3.4
// Score 3: Cr 3.5-4.9 or UO < 500 mL/day
// Score 4: Cr >= 5.0 or UO < 200 mL/day
func (c *SOFACalculator) scoreRenal(params *models.SOFAParams, inputs *[]models.InputUsed, missing *[]string) models.SOFAComponent {
	component := models.SOFAComponent{}

	// Prefer urine output for higher scores if available
	var crScore, uoScore int = -1, -1

	if params.Creatinine != nil {
		component.DataAvailable = true
		component.InputValue = *params.Creatinine
		component.InputUnit = "mg/dL"
		*inputs = append(*inputs, models.InputUsed{
			Name:  "creatinine",
			Value: *params.Creatinine,
			Unit:  "mg/dL",
		})

		cr := *params.Creatinine
		switch {
		case cr < 1.2:
			crScore = 0
		case cr < 2.0:
			crScore = 1
		case cr < 3.5:
			crScore = 2
		case cr < 5.0:
			crScore = 3
		default:
			crScore = 4
		}
	}

	if params.UrineOutput != nil {
		component.DataAvailable = true
		*inputs = append(*inputs, models.InputUsed{
			Name:  "urine_output",
			Value: *params.UrineOutput,
			Unit:  "mL/day",
		})

		uo := *params.UrineOutput
		switch {
		case uo >= 500:
			uoScore = 0 // Not specifically scored, but normal
		case uo >= 200:
			uoScore = 3
		default:
			uoScore = 4
		}
	}

	if !component.DataAvailable {
		component.Score = 0
		*missing = append(*missing, "Creatinine or urine output")
		return component
	}

	// Use the higher of the two scores
	if crScore >= uoScore {
		component.Score = crScore
		if params.Creatinine != nil {
			component.InputValue = *params.Creatinine
			component.InputUnit = "mg/dL"
		}
	} else {
		component.Score = uoScore
		if params.UrineOutput != nil {
			component.InputValue = *params.UrineOutput
			component.InputUnit = "mL/day"
		}
	}

	return component
}

// determineRiskLevel categorizes the total SOFA score into risk levels.
func (c *SOFACalculator) determineRiskLevel(total int) models.RiskLevel {
	switch {
	case total == 0:
		return models.RiskLevelLow
	case total <= 5:
		return models.RiskLevelLowModerate
	case total <= 9:
		return models.RiskLevelModerate
	case total <= 12:
		return models.RiskLevelModerateHigh
	case total <= 14:
		return models.RiskLevelHigh
	default:
		return models.RiskLevelCritical
	}
}

// estimateMortality returns estimated ICU mortality based on SOFA score.
// Based on Vincent et al. and subsequent validation studies.
func (c *SOFACalculator) estimateMortality(total int) string {
	switch {
	case total == 0:
		return "<5%"
	case total <= 2:
		return "~5%"
	case total <= 5:
		return "~10%"
	case total <= 9:
		return "15-20%"
	case total <= 12:
		return "40-50%"
	case total <= 14:
		return "50-60%"
	default: // >= 15
		return ">80%"
	}
}

// generateInterpretation creates a clinical interpretation of the SOFA result.
func (c *SOFACalculator) generateInterpretation(result *models.SOFAResult) string {
	var dysfunctionalOrgans []string

	if result.Respiration.Score >= 2 {
		dysfunctionalOrgans = append(dysfunctionalOrgans, "respiratory")
	}
	if result.Coagulation.Score >= 2 {
		dysfunctionalOrgans = append(dysfunctionalOrgans, "coagulation")
	}
	if result.Liver.Score >= 2 {
		dysfunctionalOrgans = append(dysfunctionalOrgans, "hepatic")
	}
	if result.Cardiovascular.Score >= 2 {
		dysfunctionalOrgans = append(dysfunctionalOrgans, "cardiovascular")
	}
	if result.CNS.Score >= 2 {
		dysfunctionalOrgans = append(dysfunctionalOrgans, "neurological")
	}
	if result.Renal.Score >= 2 {
		dysfunctionalOrgans = append(dysfunctionalOrgans, "renal")
	}

	var interpretation string

	switch result.RiskLevel {
	case models.RiskLevelLow:
		interpretation = "SOFA 0 - No organ dysfunction detected."
	case models.RiskLevelLowModerate:
		interpretation = fmt.Sprintf("SOFA %d - Minimal organ dysfunction.", result.Total)
	case models.RiskLevelModerate:
		interpretation = fmt.Sprintf("SOFA %d - Moderate organ dysfunction.", result.Total)
	case models.RiskLevelModerateHigh:
		interpretation = fmt.Sprintf("SOFA %d - Significant organ dysfunction.", result.Total)
	case models.RiskLevelHigh:
		interpretation = fmt.Sprintf("SOFA %d - Severe organ dysfunction. ICU-level care recommended.", result.Total)
	case models.RiskLevelCritical:
		interpretation = fmt.Sprintf("SOFA %d - Critical organ dysfunction. Mortality risk >80%%.", result.Total)
	}

	if len(dysfunctionalOrgans) > 0 {
		interpretation += fmt.Sprintf(" Affected systems: %s.", formatOrganList(dysfunctionalOrgans))
	}

	interpretation += fmt.Sprintf(" Estimated ICU mortality: %s.", result.MortalityRisk)

	// Add Sepsis-3 note if score >= 2
	if result.Total >= 2 {
		interpretation += " Per Sepsis-3 criteria, SOFA ≥2 from baseline indicates acute organ dysfunction."
	}

	return interpretation
}

// formatOrganList formats a list of organs for display.
func formatOrganList(organs []string) string {
	if len(organs) == 0 {
		return "none"
	}
	if len(organs) == 1 {
		return organs[0]
	}
	if len(organs) == 2 {
		return organs[0] + " and " + organs[1]
	}
	result := ""
	for i, organ := range organs {
		if i == len(organs)-1 {
			result += "and " + organ
		} else {
			result += organ + ", "
		}
	}
	return result
}
