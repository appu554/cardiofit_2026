package calculator

import (
	"context"
	"fmt"
	"math"
	"time"

	"kb-8-calculator-service/internal/models"
)

const (
	egfrVersion   = "CKD-EPI-2021-RaceFree"
	egfrReference = "Inker LA, et al. N Engl J Med. 2021;385(19):1737-1749"
	egfrFormula   = "142 × min(Scr/κ, 1)^α × max(Scr/κ, 1)^-1.200 × 0.9938^Age × 1.012 [if female]"
)

// EGFRCalculator implements the CKD-EPI 2021 race-free eGFR equation.
//
// The CKD-EPI 2021 equation was developed to eliminate race-based
// coefficients while maintaining accuracy. It uses only serum creatinine,
// age, and sex.
//
// Reference: Inker LA, Eneanya ND, Coresh J, et al. New Creatinine- and
// Cystatin C-Based Equations to Estimate GFR without Race. N Engl J Med.
// 2021;385(19):1737-1749. doi:10.1056/NEJMoa2102953
type EGFRCalculator struct{}

// NewEGFRCalculator creates a new eGFR calculator.
func NewEGFRCalculator() *EGFRCalculator {
	return &EGFRCalculator{}
}

// Type returns the calculator type.
func (c *EGFRCalculator) Type() models.CalculatorType {
	return models.CalculatorEGFR
}

// Name returns a human-readable name.
func (c *EGFRCalculator) Name() string {
	return "eGFR (CKD-EPI 2021)"
}

// Version returns the formula version.
func (c *EGFRCalculator) Version() string {
	return egfrVersion
}

// Reference returns the clinical citation.
func (c *EGFRCalculator) Reference() string {
	return egfrReference
}

// Calculate computes eGFR using the CKD-EPI 2021 race-free equation.
//
// Formula:
//   eGFR = 142 × min(Scr/κ, 1)^α × max(Scr/κ, 1)^-1.200 × 0.9938^Age × 1.012 [if female]
//
// Where:
//   - Scr is serum creatinine in mg/dL
//   - κ is 0.7 for females and 0.9 for males
//   - α is -0.241 for females and -0.302 for males
//   - Age is in years
//
// Returns eGFR in mL/min/1.73m²
func (c *EGFRCalculator) Calculate(ctx context.Context, params *models.EGFRParams) (*models.EGFRResult, error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return nil, err
	}

	// Calculate eGFR using CKD-EPI 2021 formula
	egfr := c.calculateEGFR(params.SerumCreatinine, params.AgeYears, params.Sex)

	// Determine CKD stage
	ckdStage := c.determineCKDStage(egfr)

	// Build result
	result := &models.EGFRResult{
		Value:                       math.Round(egfr*10) / 10, // Round to 1 decimal
		Unit:                        "mL/min/1.73m²",
		CKDStage:                    ckdStage,
		CKDStageDisplay:             fmt.Sprintf("%s (%s)", ckdStage, ckdStage.Description()),
		RequiresRenalDoseAdjustment: ckdStage.RequiresDoseAdjustment(),
		Equation:                    egfrVersion,
		Interpretation:              c.buildInterpretation(egfr, ckdStage),
		Provenance:                  c.buildProvenance(params, egfr),
	}

	// Add dose adjustment guidance if needed
	if result.RequiresRenalDoseAdjustment {
		result.DoseAdjustmentGuidance = c.getDoseAdjustmentGuidance(egfr, ckdStage)
	}

	return result, nil
}

// calculateEGFR implements the CKD-EPI 2021 race-free equation.
func (c *EGFRCalculator) calculateEGFR(scr float64, age int, sex models.Sex) float64 {
	var kappa, alpha float64

	// Set sex-specific coefficients
	if sex == models.SexFemale {
		kappa = 0.7
		alpha = -0.241
	} else {
		kappa = 0.9
		alpha = -0.302
	}

	// Calculate Scr/κ ratio
	scrKappa := scr / kappa

	// Calculate eGFR
	var egfr float64
	if scrKappa <= 1 {
		// min(Scr/κ, 1)^α × max(Scr/κ, 1)^-1.200
		// When Scr/κ <= 1: min = Scr/κ, max = 1
		egfr = 142.0 * math.Pow(scrKappa, alpha) * math.Pow(1.0, -1.200) * math.Pow(0.9938, float64(age))
	} else {
		// When Scr/κ > 1: min = 1, max = Scr/κ
		egfr = 142.0 * math.Pow(1.0, alpha) * math.Pow(scrKappa, -1.200) * math.Pow(0.9938, float64(age))
	}

	// Apply sex coefficient
	if sex == models.SexFemale {
		egfr *= 1.012
	}

	return egfr
}

// determineCKDStage classifies eGFR into CKD stages per KDIGO 2012 guidelines.
func (c *EGFRCalculator) determineCKDStage(egfr float64) models.CKDStage {
	switch {
	case egfr >= 90:
		return models.CKDStageG1
	case egfr >= 60:
		return models.CKDStageG2
	case egfr >= 45:
		return models.CKDStageG3a
	case egfr >= 30:
		return models.CKDStageG3b
	case egfr >= 15:
		return models.CKDStageG4
	default:
		return models.CKDStageG5
	}
}

// buildInterpretation creates a clinical interpretation string.
func (c *EGFRCalculator) buildInterpretation(egfr float64, stage models.CKDStage) string {
	roundedEGFR := math.Round(egfr*10) / 10

	switch stage {
	case models.CKDStageG1:
		return fmt.Sprintf("eGFR %.1f mL/min/1.73m² - Normal kidney function. No renal dose adjustment required.", roundedEGFR)
	case models.CKDStageG2:
		return fmt.Sprintf("eGFR %.1f mL/min/1.73m² - Mildly decreased kidney function. Monitor for progression.", roundedEGFR)
	case models.CKDStageG3a:
		return fmt.Sprintf("eGFR %.1f mL/min/1.73m² - Mild-moderate CKD (G3a). Review medications for renal dose adjustment.", roundedEGFR)
	case models.CKDStageG3b:
		return fmt.Sprintf("eGFR %.1f mL/min/1.73m² - Moderate-severe CKD (G3b). Renal dose adjustment required for many medications.", roundedEGFR)
	case models.CKDStageG4:
		return fmt.Sprintf("eGFR %.1f mL/min/1.73m² - Severe CKD (G4). Significant renal dose adjustment required. Consider nephrology referral.", roundedEGFR)
	case models.CKDStageG5:
		return fmt.Sprintf("eGFR %.1f mL/min/1.73m² - Kidney failure (G5). Many medications contraindicated. Nephrology consultation essential.", roundedEGFR)
	default:
		return fmt.Sprintf("eGFR %.1f mL/min/1.73m²", roundedEGFR)
	}
}

// getDoseAdjustmentGuidance provides dosing recommendations based on eGFR.
func (c *EGFRCalculator) getDoseAdjustmentGuidance(egfr float64, stage models.CKDStage) string {
	switch stage {
	case models.CKDStageG3a:
		return "Mild impairment: Review drug dosing. Some medications may require 25% dose reduction."
	case models.CKDStageG3b:
		return "Moderate impairment: Dose adjustment required for renally-cleared drugs. Typical reduction 25-50%."
	case models.CKDStageG4:
		return "Severe impairment: Significant dose reduction required (50-75%). Avoid nephrotoxic medications."
	case models.CKDStageG5:
		return "Kidney failure: Many medications contraindicated. Consult pharmacy/nephrology for safe alternatives."
	default:
		return ""
	}
}

// buildProvenance creates the SaMD provenance record.
func (c *EGFRCalculator) buildProvenance(params *models.EGFRParams, egfr float64) models.Provenance {
	return models.Provenance{
		CalculatorType: string(models.CalculatorEGFR),
		Version:        egfrVersion,
		Formula:        egfrFormula,
		Reference:      egfrReference,
		CalculatedAt:   time.Now().UTC(),
		InputsUsed: []models.InputUsed{
			{Name: "serum_creatinine", Value: params.SerumCreatinine, Unit: "mg/dL", Source: "lab_result"},
			{Name: "age", Value: params.AgeYears, Unit: "years", Source: "demographics"},
			{Name: "sex", Value: string(params.Sex), Source: "demographics"},
		},
		DataQuality: models.DataQualityComplete,
		Caveats: []string{
			"Race-free equation - does not use race as a variable",
			"Assumes stable kidney function (not valid during AKI)",
			"May be less accurate with extreme muscle mass (cachexia, bodybuilders)",
			"Validated for ages 18-90; use with caution outside this range",
		},
	}
}
