// Package calculator provides clinical score calculators.
package calculator

import (
	"context"
	"fmt"
	"time"

	"kb-8-calculator-service/internal/models"
)

// HASBLEDCalculator calculates HAS-BLED score for major bleeding risk.
// Reference: Pisters R, et al. Chest. 2010;138(5):1093-1100.
//
// Used to assess bleeding risk in patients on anticoagulation for AF,
// to help balance stroke prevention against bleeding risk.
// Should NOT be used to withhold anticoagulation, but to identify
// modifiable risk factors.
type HASBLEDCalculator struct{}

// NewHASBLEDCalculator creates a new HAS-BLED calculator instance.
func NewHASBLEDCalculator() *HASBLEDCalculator {
	return &HASBLEDCalculator{}
}

// Calculate computes the HAS-BLED score from the given parameters.
// Score range: 0-9
// Score ≥3 indicates high bleeding risk requiring caution and frequent review.
func (c *HASBLEDCalculator) Calculate(ctx context.Context, params *models.HASBLEDParams) (*models.HASBLEDResult, error) {
	result := &models.HASBLEDResult{
		Factors: []models.HASBLEDFactor{},
		Provenance: models.Provenance{
			CalculatorType: string(models.CalculatorHASBLED),
			Version:        "HAS-BLED-2010",
			Formula:        "H(1) + A(1-2) + S(1) + B(1) + L(1) + E(1) + D(1-2)",
			Reference:      "Pisters R, et al. A novel user-friendly score to assess bleeding risk. Chest. 2010;138(5):1093-1100",
			CalculatedAt:   time.Now().UTC(),
			InputsUsed:     []models.InputUsed{},
			DataQuality:    models.DataQualityComplete,
		},
	}

	inputsUsed := []models.InputUsed{}
	total := 0

	// H - Hypertension (uncontrolled, SBP > 160 mmHg) (1 point)
	htnFactor := models.HASBLEDFactor{
		Name:    "Hypertension (uncontrolled, SBP >160)",
		Present: params.HasUncontrolledHypertension,
		Points:  0,
	}
	if params.HasUncontrolledHypertension {
		htnFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "uncontrolled_hypertension", Value: true})
	}
	result.Factors = append(result.Factors, htnFactor)

	// A - Abnormal renal function (1 point)
	// Dialysis, transplant, Cr >2.26 mg/dL or >200 µmol/L
	renalFactor := models.HASBLEDFactor{
		Name:    "Abnormal Renal Function",
		Present: params.HasAbnormalRenalFunction,
		Points:  0,
	}
	if params.HasAbnormalRenalFunction {
		renalFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "abnormal_renal_function", Value: true})
	}
	result.Factors = append(result.Factors, renalFactor)

	// A - Abnormal liver function (1 point)
	// Cirrhosis, Bilirubin >2x ULN, AST/ALT/ALP >3x ULN
	liverFactor := models.HASBLEDFactor{
		Name:    "Abnormal Liver Function",
		Present: params.HasAbnormalLiverFunction,
		Points:  0,
	}
	if params.HasAbnormalLiverFunction {
		liverFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "abnormal_liver_function", Value: true})
	}
	result.Factors = append(result.Factors, liverFactor)

	// S - Stroke history (1 point)
	strokeFactor := models.HASBLEDFactor{
		Name:    "Stroke History",
		Present: params.HasStrokeHistory,
		Points:  0,
	}
	if params.HasStrokeHistory {
		strokeFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "stroke_history", Value: true})
	}
	result.Factors = append(result.Factors, strokeFactor)

	// B - Bleeding history or predisposition (1 point)
	// Prior major bleed, anemia, thrombocytopenia
	bleedingFactor := models.HASBLEDFactor{
		Name:    "Bleeding History/Predisposition",
		Present: params.HasBleedingHistory,
		Points:  0,
	}
	if params.HasBleedingHistory {
		bleedingFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "bleeding_history", Value: true})
	}
	result.Factors = append(result.Factors, bleedingFactor)

	// L - Labile INR (1 point)
	// TTR < 60% (time in therapeutic range)
	inrFactor := models.HASBLEDFactor{
		Name:    "Labile INR (TTR <60%)",
		Present: params.HasLabileINR,
		Points:  0,
	}
	if params.HasLabileINR {
		inrFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "labile_inr", Value: true})
	}
	result.Factors = append(result.Factors, inrFactor)

	// E - Elderly (>65 years) (1 point)
	elderlyFactor := models.HASBLEDFactor{
		Name:    "Elderly (>65 years)",
		Present: params.AgeYears > 65,
		Points:  0,
	}
	if params.AgeYears > 65 {
		elderlyFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "age", Value: params.AgeYears, Unit: "years"})
	}
	result.Factors = append(result.Factors, elderlyFactor)

	// D - Drugs (1 point)
	// Concomitant antiplatelet agents, NSAIDs
	drugsFactor := models.HASBLEDFactor{
		Name:    "Drugs (Antiplatelet/NSAIDs)",
		Present: params.TakingAntiplateletOrNSAID,
		Points:  0,
	}
	if params.TakingAntiplateletOrNSAID {
		drugsFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "antiplatelet_or_nsaid", Value: true})
	}
	result.Factors = append(result.Factors, drugsFactor)

	// D - Alcohol excess (1 point)
	// ≥8 drinks/week
	alcoholFactor := models.HASBLEDFactor{
		Name:    "Alcohol Excess (≥8 drinks/week)",
		Present: params.ExcessiveAlcohol,
		Points:  0,
	}
	if params.ExcessiveAlcohol {
		alcoholFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "excessive_alcohol", Value: true})
	}
	result.Factors = append(result.Factors, alcoholFactor)

	result.Total = total
	result.Provenance.InputsUsed = inputsUsed

	// Determine risk category and recommendations
	result.RiskCategory = c.determineRiskCategory(total)
	result.AnnualBleedingRisk = c.estimateAnnualBleedingRisk(total)
	result.HighRisk = total >= 3
	result.Recommendation = c.generateRecommendation(result)

	result.Provenance.Caveats = []string{
		"HAS-BLED should NOT be used to exclude patients from anticoagulation",
		"Purpose is to identify modifiable risk factors and flag patients for closer monitoring",
		"Labile INR criterion less relevant for DOACs (not warfarin dependent)",
		"High HAS-BLED does not necessarily outweigh stroke risk benefit",
	}

	return result, nil
}

// determineRiskCategory categorizes bleeding risk based on score.
func (c *HASBLEDCalculator) determineRiskCategory(score int) models.RiskLevel {
	switch {
	case score <= 1:
		return models.RiskLevelLow
	case score == 2:
		return models.RiskLevelModerate
	case score == 3:
		return models.RiskLevelModerateHigh
	case score == 4:
		return models.RiskLevelHigh
	default:
		return models.RiskLevelVeryHigh
	}
}

// estimateAnnualBleedingRisk returns estimated annual major bleeding rate.
// Based on Pisters et al. validation data.
func (c *HASBLEDCalculator) estimateAnnualBleedingRisk(score int) string {
	rates := map[int]string{
		0: "1.13%",
		1: "1.02%",
		2: "1.88%",
		3: "3.74%",
		4: "8.70%",
		5: "12.50%",
	}
	if rate, ok := rates[score]; ok {
		return rate
	}
	return ">12%"
}

// generateRecommendation provides clinical recommendations.
func (c *HASBLEDCalculator) generateRecommendation(result *models.HASBLEDResult) string {
	var rec string
	var modifiable []string

	// Identify modifiable risk factors
	for _, f := range result.Factors {
		if f.Present {
			switch f.Name {
			case "Hypertension (uncontrolled, SBP >160)":
				modifiable = append(modifiable, "optimize blood pressure control")
			case "Labile INR (TTR <60%)":
				modifiable = append(modifiable, "improve INR control or switch to DOAC")
			case "Drugs (Antiplatelet/NSAIDs)":
				modifiable = append(modifiable, "review necessity of concomitant antiplatelet/NSAIDs")
			case "Alcohol Excess (≥8 drinks/week)":
				modifiable = append(modifiable, "recommend alcohol reduction")
			}
		}
	}

	if result.HighRisk {
		rec = fmt.Sprintf("HIGH BLEEDING RISK (HAS-BLED %d). Annual major bleeding risk ~%s. ",
			result.Total, result.AnnualBleedingRisk)
		rec += "Requires careful risk-benefit analysis and closer monitoring. "
		rec += "However, high HAS-BLED alone should NOT exclude patient from anticoagulation if stroke risk warrants treatment. "
	} else if result.RiskCategory == models.RiskLevelModerate {
		rec = fmt.Sprintf("MODERATE BLEEDING RISK (HAS-BLED %d). Annual major bleeding risk ~%s. ",
			result.Total, result.AnnualBleedingRisk)
		rec += "Standard monitoring appropriate. "
	} else {
		rec = fmt.Sprintf("LOW BLEEDING RISK (HAS-BLED %d). Annual major bleeding risk ~%s. ",
			result.Total, result.AnnualBleedingRisk)
		rec += "Favorable safety profile for anticoagulation. "
	}

	if len(modifiable) > 0 {
		rec += "Modifiable factors to address: " + formatFactorList(modifiable) + ". "
	}

	rec += "Compare with CHA₂DS₂-VASc to balance stroke prevention against bleeding risk."

	return rec
}

// formatFactorList formats a list of factors for display.
func formatFactorList(factors []string) string {
	if len(factors) == 0 {
		return "none identified"
	}
	if len(factors) == 1 {
		return factors[0]
	}
	if len(factors) == 2 {
		return factors[0] + " and " + factors[1]
	}
	result := ""
	for i, f := range factors {
		if i == len(factors)-1 {
			result += "and " + f
		} else {
			result += f + ", "
		}
	}
	return result
}
