// Package calculator provides clinical score calculators.
package calculator

import (
	"context"
	"fmt"
	"math"
	"time"

	"kb-8-calculator-service/internal/models"
)

// ASCVDCalculator calculates 10-year ASCVD risk using Pooled Cohort Equations.
// Reference: Goff DC Jr, et al. Circulation. 2014;129(25 Suppl 2):S49-73.
// Updated: 2018 AHA/ACC/AACVPR/AAPA/ABC/ACPM/ADA/AGS/APhA/ASPC/NLA/PCNA Guidelines.
//
// The Pooled Cohort Equations estimate 10-year risk of first atherosclerotic
// cardiovascular disease (ASCVD) event: nonfatal MI, CHD death, or fatal/nonfatal stroke.
type ASCVDCalculator struct{}

// NewASCVDCalculator creates a new ASCVD calculator instance.
func NewASCVDCalculator() *ASCVDCalculator {
	return &ASCVDCalculator{}
}

// Calculate computes the 10-year ASCVD risk from the given parameters.
// Returns risk as percentage (0-100%).
func (c *ASCVDCalculator) Calculate(ctx context.Context, params *models.ASCVDParams) (*models.ASCVDResult, error) {
	// Validate required inputs
	if params.AgeYears < 40 || params.AgeYears > 79 {
		return nil, fmt.Errorf("%w: ASCVD calculator valid for ages 40-79", models.ErrInvalidAge)
	}
	if !params.Sex.IsValid() {
		return nil, models.ErrInvalidSex
	}
	if params.TotalCholesterol <= 0 || params.TotalCholesterol > 500 {
		return nil, fmt.Errorf("invalid total cholesterol: must be 1-500 mg/dL")
	}
	if params.HDLCholesterol <= 0 || params.HDLCholesterol > 200 {
		return nil, fmt.Errorf("invalid HDL cholesterol: must be 1-200 mg/dL")
	}
	if params.SystolicBP <= 0 || params.SystolicBP > 300 {
		return nil, fmt.Errorf("invalid systolic BP: must be 1-300 mmHg")
	}

	result := &models.ASCVDResult{
		Provenance: models.Provenance{
			CalculatorType: string(models.CalculatorASCVD),
			Version:        "PCE-2013-Revised-2018",
			Formula:        "Pooled Cohort Equations (Cox proportional hazards)",
			Reference:      "Goff DC Jr, et al. 2013 ACC/AHA Guideline. Circulation. 2014;129(25 Suppl 2):S49-73",
			CalculatedAt:   time.Now().UTC(),
			InputsUsed: []models.InputUsed{
				{Name: "age", Value: params.AgeYears, Unit: "years"},
				{Name: "sex", Value: string(params.Sex)},
				{Name: "race", Value: c.normalizeRace(params.Race)},
				{Name: "total_cholesterol", Value: params.TotalCholesterol, Unit: "mg/dL"},
				{Name: "hdl_cholesterol", Value: params.HDLCholesterol, Unit: "mg/dL"},
				{Name: "systolic_bp", Value: params.SystolicBP, Unit: "mmHg"},
				{Name: "on_bp_treatment", Value: params.OnBPTreatment},
				{Name: "has_diabetes", Value: params.HasDiabetes},
				{Name: "is_smoker", Value: params.IsSmoker},
			},
			DataQuality: models.DataQualityComplete,
		},
	}

	// Calculate risk using Pooled Cohort Equations
	riskPercent := c.calculateRisk(params)
	result.RiskPercent = math.Round(riskPercent*10) / 10 // Round to 1 decimal

	// Determine risk category (per 2018 guidelines)
	result.RiskCategory = c.determineRiskCategory(riskPercent)

	// Generate recommendations
	result.StatinRecommendation = c.generateStatinRecommendation(riskPercent, params)
	result.Interpretation = c.generateInterpretation(result, params)

	result.Provenance.Caveats = []string{
		"Valid for ages 40-79 without prior ASCVD",
		"May overestimate risk in some populations",
		"Race coefficients available for White and African American; other races use White coefficients",
		"Does not account for LDL-C, family history of premature ASCVD, or hsCRP",
		"Lifetime risk considerations important for younger patients",
	}

	return result, nil
}

// normalizeRace normalizes race input to expected values.
func (c *ASCVDCalculator) normalizeRace(race string) string {
	switch race {
	case "african_american", "black", "african-american", "aa":
		return "african_american"
	case "white", "caucasian":
		return "white"
	default:
		return "other" // Will use white coefficients
	}
}

// calculateRisk computes the actual 10-year ASCVD risk percentage.
// Uses sex and race-specific coefficients from the Pooled Cohort Equations.
func (c *ASCVDCalculator) calculateRisk(params *models.ASCVDParams) float64 {
	isAA := c.normalizeRace(params.Race) == "african_american"
	isFemale := params.Sex == models.SexFemale

	// Get appropriate coefficients
	var coef coefficients
	switch {
	case isFemale && isAA:
		coef = aaFemaleCoef
	case isFemale && !isAA:
		coef = whiteFemaleCoef
	case !isFemale && isAA:
		coef = aaMaleCoef
	default:
		coef = whiteMaleCoef
	}

	// Calculate individual terms (all natural log transforms)
	lnAge := math.Log(float64(params.AgeYears))
	lnTC := math.Log(params.TotalCholesterol)
	lnHDL := math.Log(params.HDLCholesterol)
	lnSBP := math.Log(params.SystolicBP)

	// Build the sum of terms
	sum := coef.lnAge * lnAge
	sum += coef.lnAgeSquared * lnAge * lnAge
	sum += coef.lnTC * lnTC
	sum += coef.lnAgeLnTC * lnAge * lnTC
	sum += coef.lnHDL * lnHDL
	sum += coef.lnAgeLnHDL * lnAge * lnHDL

	// Blood pressure terms depend on treatment status
	if params.OnBPTreatment {
		sum += coef.lnTreatedSBP * lnSBP
		sum += coef.lnAgeLnTreatedSBP * lnAge * lnSBP
	} else {
		sum += coef.lnUntreatedSBP * lnSBP
		sum += coef.lnAgeLnUntreatedSBP * lnAge * lnSBP
	}

	// Smoking term
	if params.IsSmoker {
		sum += coef.currentSmoker
		sum += coef.lnAgeCurrentSmoker * lnAge
	}

	// Diabetes term
	if params.HasDiabetes {
		sum += coef.diabetes
	}

	// Calculate 10-year risk
	// Risk = 1 - S0^exp(sum - meanCoef)
	risk := 1 - math.Pow(coef.baselineSurvival, math.Exp(sum-coef.meanCoef))

	// Convert to percentage and clamp to reasonable range
	riskPercent := risk * 100
	if riskPercent < 0 {
		riskPercent = 0
	}
	if riskPercent > 100 {
		riskPercent = 100
	}

	return riskPercent
}

// determineRiskCategory categorizes risk based on 2018 AHA/ACC guidelines.
func (c *ASCVDCalculator) determineRiskCategory(riskPercent float64) models.RiskLevel {
	switch {
	case riskPercent < 5:
		return models.RiskLevelLow
	case riskPercent < 7.5:
		return models.RiskLevelLowModerate // "Borderline" in guidelines
	case riskPercent < 20:
		return models.RiskLevelModerate // "Intermediate"
	default:
		return models.RiskLevelHigh
	}
}

// generateStatinRecommendation provides statin therapy guidance.
func (c *ASCVDCalculator) generateStatinRecommendation(riskPercent float64, params *models.ASCVDParams) string {
	// Check for diabetes (special consideration)
	if params.HasDiabetes {
		if riskPercent >= 20 {
			return "HIGH-INTENSITY statin therapy recommended. Diabetes with 10-year ASCVD risk ≥20%."
		}
		if params.AgeYears >= 40 {
			return "MODERATE-INTENSITY statin therapy recommended for diabetic patients age 40-75. Consider high-intensity if multiple risk factors."
		}
	}

	switch {
	case riskPercent < 5:
		return "Statin therapy generally not recommended. Focus on lifestyle modifications. Consider risk-enhancing factors."

	case riskPercent < 7.5:
		return "BORDERLINE RISK. Statin therapy may be considered if risk-enhancing factors present (family history, elevated LDL-C ≥160, metabolic syndrome, CKD, hsCRP ≥2)."

	case riskPercent < 20:
		return "INTERMEDIATE RISK. If risk discussion favors statin therapy, initiate MODERATE-INTENSITY statin. Consider high-intensity if LDL-C ≥190 or risk-enhancing factors."

	default: // >= 20%
		return "HIGH RISK. HIGH-INTENSITY statin therapy recommended to achieve ≥50% LDL-C reduction. Target LDL-C <70 mg/dL if additional risk factors."
	}
}

// generateInterpretation creates a clinical interpretation.
func (c *ASCVDCalculator) generateInterpretation(result *models.ASCVDResult, params *models.ASCVDParams) string {
	riskDesc := ""
	switch result.RiskCategory {
	case models.RiskLevelLow:
		riskDesc = "LOW (<5%)"
	case models.RiskLevelLowModerate:
		riskDesc = "BORDERLINE (5-7.4%)"
	case models.RiskLevelModerate:
		riskDesc = "INTERMEDIATE (7.5-19.9%)"
	case models.RiskLevelHigh:
		riskDesc = "HIGH (≥20%)"
	}

	interpretation := fmt.Sprintf("10-year ASCVD risk: %.1f%% (%s). ", result.RiskPercent, riskDesc)

	// Add risk factor summary
	var riskFactors []string
	if params.HasDiabetes {
		riskFactors = append(riskFactors, "diabetes")
	}
	if params.IsSmoker {
		riskFactors = append(riskFactors, "current smoker")
	}
	if params.OnBPTreatment {
		riskFactors = append(riskFactors, "treated hypertension")
	} else if params.SystolicBP >= 140 {
		riskFactors = append(riskFactors, "elevated BP")
	}
	if params.HDLCholesterol < 40 {
		riskFactors = append(riskFactors, "low HDL")
	}

	if len(riskFactors) > 0 {
		interpretation += fmt.Sprintf("Contributing factors: %s. ", formatRiskFactorList(riskFactors))
	}

	// Age consideration for younger patients
	if params.AgeYears < 50 && result.RiskPercent < 7.5 {
		interpretation += "Note: Younger patients may have low 10-year risk but high lifetime risk - consider lifetime risk assessment. "
	}

	interpretation += "Risk calculation assumes no prior ASCVD events."

	return interpretation
}

// formatRiskFactorList formats a list of risk factors.
func formatRiskFactorList(factors []string) string {
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

// coefficients holds the Pooled Cohort Equation coefficients for a demographic group.
type coefficients struct {
	lnAge                  float64
	lnAgeSquared           float64
	lnTC                   float64
	lnAgeLnTC              float64
	lnHDL                  float64
	lnAgeLnHDL             float64
	lnTreatedSBP           float64
	lnAgeLnTreatedSBP      float64
	lnUntreatedSBP         float64
	lnAgeLnUntreatedSBP    float64
	currentSmoker          float64
	lnAgeCurrentSmoker     float64
	diabetes               float64
	meanCoef               float64
	baselineSurvival       float64
}

// Pooled Cohort Equation coefficients by demographic group
// Source: Goff et al. 2014 (Supplementary Tables)

var whiteFemaleCoef = coefficients{
	lnAge:                  -29.799,
	lnAgeSquared:           4.884,
	lnTC:                   13.540,
	lnAgeLnTC:              -3.114,
	lnHDL:                  -13.578,
	lnAgeLnHDL:             3.149,
	lnTreatedSBP:           2.019,
	lnAgeLnTreatedSBP:      0,
	lnUntreatedSBP:         1.957,
	lnAgeLnUntreatedSBP:    0,
	currentSmoker:          7.574,
	lnAgeCurrentSmoker:     -1.665,
	diabetes:               0.661,
	meanCoef:               -29.18,
	baselineSurvival:       0.9665,
}

var whiteMaleCoef = coefficients{
	lnAge:                  12.344,
	lnAgeSquared:           0,
	lnTC:                   11.853,
	lnAgeLnTC:              -2.664,
	lnHDL:                  -7.990,
	lnAgeLnHDL:             1.769,
	lnTreatedSBP:           1.797,
	lnAgeLnTreatedSBP:      0,
	lnUntreatedSBP:         1.764,
	lnAgeLnUntreatedSBP:    0,
	currentSmoker:          7.837,
	lnAgeCurrentSmoker:     -1.795,
	diabetes:               0.658,
	meanCoef:               61.18,
	baselineSurvival:       0.9144,
}

var aaFemaleCoef = coefficients{
	lnAge:                  17.114,
	lnAgeSquared:           0,
	lnTC:                   0.940,
	lnAgeLnTC:              0,
	lnHDL:                  -18.920,
	lnAgeLnHDL:             4.475,
	lnTreatedSBP:           29.291,
	lnAgeLnTreatedSBP:      -6.432,
	lnUntreatedSBP:         27.820,
	lnAgeLnUntreatedSBP:    -6.087,
	currentSmoker:          0.691,
	lnAgeCurrentSmoker:     0,
	diabetes:               0.874,
	meanCoef:               86.61,
	baselineSurvival:       0.9533,
}

var aaMaleCoef = coefficients{
	lnAge:                  2.469,
	lnAgeSquared:           0,
	lnTC:                   0.302,
	lnAgeLnTC:              0,
	lnHDL:                  -0.307,
	lnAgeLnHDL:             0,
	lnTreatedSBP:           1.916,
	lnAgeLnTreatedSBP:      0,
	lnUntreatedSBP:         1.809,
	lnAgeLnUntreatedSBP:    0,
	currentSmoker:          0.549,
	lnAgeCurrentSmoker:     0,
	diabetes:               0.645,
	meanCoef:               19.54,
	baselineSurvival:       0.8954,
}
