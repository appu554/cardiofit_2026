// Package calculator provides clinical score calculators.
package calculator

import (
	"context"
	"fmt"
	"time"

	"kb-8-calculator-service/internal/models"
)

// CHA2DS2VAScCalculator calculates CHA₂DS₂-VASc score for stroke risk in AF.
// Reference: Lip GYH, et al. Chest. 2010;137(2):263-272.
//
// Used to estimate annual stroke risk in patients with atrial fibrillation
// and guide anticoagulation therapy decisions.
type CHA2DS2VAScCalculator struct{}

// NewCHA2DS2VAScCalculator creates a new CHA₂DS₂-VASc calculator instance.
func NewCHA2DS2VAScCalculator() *CHA2DS2VAScCalculator {
	return &CHA2DS2VAScCalculator{}
}

// Calculate computes the CHA₂DS₂-VASc score from the given parameters.
// Score range: 0-9
// Higher scores indicate greater annual stroke risk.
func (c *CHA2DS2VAScCalculator) Calculate(ctx context.Context, params *models.CHA2DS2VAScParams) (*models.CHA2DS2VAScResult, error) {
	// Validate required inputs
	if params.AgeYears <= 0 || params.AgeYears > 120 {
		return nil, models.ErrInvalidAge
	}
	if !params.Sex.IsValid() {
		return nil, models.ErrInvalidSex
	}

	result := &models.CHA2DS2VAScResult{
		Factors: []models.CHA2DS2VAScFactor{},
		Provenance: models.Provenance{
			CalculatorType: string(models.CalculatorCHA2DS2VASc),
			Version:        "CHA2DS2-VASc-2010",
			Formula:        "C(1) + H(1) + A₂(2) + D(1) + S₂(2) + V(1) + A(1) + Sc(1)",
			Reference:      "Lip GYH, et al. Refining clinical risk stratification for stroke in AF. Chest. 2010;137(2):263-272",
			CalculatedAt:   time.Now().UTC(),
			InputsUsed:     []models.InputUsed{},
			DataQuality:    models.DataQualityComplete,
		},
	}

	inputsUsed := []models.InputUsed{
		{Name: "age", Value: params.AgeYears, Unit: "years"},
		{Name: "sex", Value: string(params.Sex)},
	}

	total := 0

	// C - Congestive Heart Failure (1 point)
	chfFactor := models.CHA2DS2VAScFactor{
		Name:    "Congestive Heart Failure",
		Present: params.HasCongestiveHeartFailure,
		Points:  0,
	}
	if params.HasCongestiveHeartFailure {
		chfFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "congestive_heart_failure", Value: true})
	}
	result.Factors = append(result.Factors, chfFactor)

	// H - Hypertension (1 point)
	htnFactor := models.CHA2DS2VAScFactor{
		Name:    "Hypertension",
		Present: params.HasHypertension,
		Points:  0,
	}
	if params.HasHypertension {
		htnFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "hypertension", Value: true})
	}
	result.Factors = append(result.Factors, htnFactor)

	// A₂ - Age ≥ 75 years (2 points)
	age75Factor := models.CHA2DS2VAScFactor{
		Name:    "Age ≥75 years",
		Present: params.AgeYears >= 75,
		Points:  0,
	}
	if params.AgeYears >= 75 {
		age75Factor.Points = 2
		total += 2
	}
	result.Factors = append(result.Factors, age75Factor)

	// D - Diabetes Mellitus (1 point)
	dmFactor := models.CHA2DS2VAScFactor{
		Name:    "Diabetes Mellitus",
		Present: params.HasDiabetes,
		Points:  0,
	}
	if params.HasDiabetes {
		dmFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "diabetes", Value: true})
	}
	result.Factors = append(result.Factors, dmFactor)

	// S₂ - Stroke/TIA/Thromboembolism (2 points)
	strokeFactor := models.CHA2DS2VAScFactor{
		Name:    "Prior Stroke/TIA/Thromboembolism",
		Present: params.HasStrokeTIA,
		Points:  0,
	}
	if params.HasStrokeTIA {
		strokeFactor.Points = 2
		total += 2
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "prior_stroke_tia", Value: true})
	}
	result.Factors = append(result.Factors, strokeFactor)

	// V - Vascular disease (MI, PAD, aortic plaque) (1 point)
	vascFactor := models.CHA2DS2VAScFactor{
		Name:    "Vascular Disease (MI, PAD, Aortic Plaque)",
		Present: params.HasVascularDisease,
		Points:  0,
	}
	if params.HasVascularDisease {
		vascFactor.Points = 1
		total++
		inputsUsed = append(inputsUsed, models.InputUsed{Name: "vascular_disease", Value: true})
	}
	result.Factors = append(result.Factors, vascFactor)

	// A - Age 65-74 years (1 point) - only if not already ≥75
	age65Factor := models.CHA2DS2VAScFactor{
		Name:    "Age 65-74 years",
		Present: params.AgeYears >= 65 && params.AgeYears < 75,
		Points:  0,
	}
	if params.AgeYears >= 65 && params.AgeYears < 75 {
		age65Factor.Points = 1
		total++
	}
	result.Factors = append(result.Factors, age65Factor)

	// Sc - Sex category (Female = 1 point)
	sexFactor := models.CHA2DS2VAScFactor{
		Name:    "Female Sex",
		Present: params.Sex == models.SexFemale,
		Points:  0,
	}
	if params.Sex == models.SexFemale {
		sexFactor.Points = 1
		total++
	}
	result.Factors = append(result.Factors, sexFactor)

	result.Total = total
	result.Provenance.InputsUsed = inputsUsed

	// Determine risk category and recommendations
	result.RiskCategory = c.determineRiskCategory(total, params.Sex)
	result.AnnualStrokeRisk = c.estimateAnnualStrokeRisk(total)
	result.AnticoagulationRecommended = c.shouldRecommendAnticoagulation(total, params.Sex)
	result.Recommendation = c.generateRecommendation(total, params.Sex)

	result.Provenance.Caveats = []string{
		"Score applies to non-valvular atrial fibrillation",
		"Female sex alone (score of 1) does not warrant anticoagulation without other risk factors",
		"Clinical judgment should consider individual bleeding risk (HAS-BLED)",
		"Score validated for oral anticoagulation decisions",
	}

	return result, nil
}

// determineRiskCategory categorizes stroke risk based on score.
func (c *CHA2DS2VAScCalculator) determineRiskCategory(score int, sex models.Sex) models.RiskLevel {
	// Adjust for sex - females need score ≥2 for meaningful risk
	if sex == models.SexFemale {
		switch {
		case score <= 1:
			return models.RiskLevelLow
		case score == 2:
			return models.RiskLevelLowModerate
		case score <= 4:
			return models.RiskLevelModerate
		case score <= 6:
			return models.RiskLevelHigh
		default:
			return models.RiskLevelVeryHigh
		}
	}
	// Males
	switch {
	case score == 0:
		return models.RiskLevelLow
	case score == 1:
		return models.RiskLevelLowModerate
	case score <= 3:
		return models.RiskLevelModerate
	case score <= 5:
		return models.RiskLevelHigh
	default:
		return models.RiskLevelVeryHigh
	}
}

// estimateAnnualStrokeRisk returns estimated annual stroke rate percentage.
// Based on Lip et al. validation data.
func (c *CHA2DS2VAScCalculator) estimateAnnualStrokeRisk(score int) string {
	rates := map[int]string{
		0: "0.2%",
		1: "0.6%",
		2: "2.2%",
		3: "3.2%",
		4: "4.8%",
		5: "7.2%",
		6: "9.7%",
		7: "11.2%",
		8: "10.8%",
		9: "12.2%",
	}
	if rate, ok := rates[score]; ok {
		return rate
	}
	return ">12%"
}

// shouldRecommendAnticoagulation determines if OAC is recommended.
// Based on current ESC/AHA guidelines.
func (c *CHA2DS2VAScCalculator) shouldRecommendAnticoagulation(score int, sex models.Sex) bool {
	// ESC 2020 Guidelines:
	// - Males: OAC recommended if score ≥1 (should be considered) or ≥2 (recommended)
	// - Females: Score of 1 (sex alone) doesn't warrant OAC; need score ≥2

	if sex == models.SexFemale {
		// Female with score 1 = sex factor alone - not sufficient
		return score >= 2
	}
	// Male: recommend if score ≥1 (per guidelines, ≥2 is Class I recommendation)
	return score >= 1
}

// generateRecommendation provides clinical recommendations.
func (c *CHA2DS2VAScCalculator) generateRecommendation(score int, sex models.Sex) string {
	// Handle female with score 1 (sex factor alone)
	if sex == models.SexFemale && score == 1 {
		return "CHA₂DS₂-VASc score of 1 in females (sex category alone) does not warrant anticoagulation. No additional risk factors identified. Reassess periodically for development of risk factors."
	}

	switch {
	case score == 0:
		return "Low stroke risk. No anticoagulation recommended. Consider aspirin only if other cardiovascular indications exist. Reassess risk factors periodically."

	case score == 1:
		if sex == models.SexMale {
			return "Low-moderate stroke risk. Oral anticoagulation may be considered (Class IIa). Balance bleeding risk (HAS-BLED) against stroke risk. DOAC preferred over warfarin."
		}
		return "Low stroke risk. Reassess periodically for additional risk factors."

	case score == 2:
		return "Moderate stroke risk. Oral anticoagulation is recommended (Class I). DOAC (dabigatran, rivaroxaban, apixaban, edoxaban) preferred over warfarin unless contraindicated. Assess bleeding risk with HAS-BLED."

	case score >= 3 && score <= 4:
		return "Moderate-high stroke risk. Oral anticoagulation strongly recommended (Class I). DOAC preferred. Assess and modify modifiable bleeding risk factors."

	case score >= 5 && score <= 6:
		return "High stroke risk. Oral anticoagulation essential. Annual stroke risk >7%. DOAC preferred. Close monitoring recommended."

	default: // score >= 7
		return fmt.Sprintf("Very high stroke risk (annual risk >%s). Anticoagulation mandatory unless absolute contraindication. Consider cardiology consultation for rhythm control strategies.", c.estimateAnnualStrokeRisk(score))
	}
}
