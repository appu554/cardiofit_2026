package calculator

import (
	"context"
	"fmt"
	"math"
	"time"

	"kb-8-calculator-service/internal/models"
)

const (
	crclVersion   = "Cockcroft-Gault-1976"
	crclReference = "Cockcroft DW, Gault MH. Nephron. 1976;16(1):31-41"
	crclFormula   = "CrCl = [(140 - age) × weight] / (72 × SCr) × 0.85 [if female]"
)

// CrClCalculator implements the Cockcroft-Gault creatinine clearance equation.
//
// The Cockcroft-Gault equation estimates creatinine clearance (CrCl) using
// actual body weight. It is widely used for drug dosing as many drug labels
// reference CrCl rather than eGFR.
//
// Note: CrCl (mL/min) is NOT normalized to body surface area, unlike eGFR
// (mL/min/1.73m²). For obese patients, consider using ideal body weight
// or adjusted body weight.
//
// Reference: Cockcroft DW, Gault MH. Prediction of creatinine clearance
// from serum creatinine. Nephron. 1976;16(1):31-41.
type CrClCalculator struct{}

// NewCrClCalculator creates a new CrCl calculator.
func NewCrClCalculator() *CrClCalculator {
	return &CrClCalculator{}
}

// Type returns the calculator type.
func (c *CrClCalculator) Type() models.CalculatorType {
	return models.CalculatorCrCl
}

// Name returns a human-readable name.
func (c *CrClCalculator) Name() string {
	return "CrCl (Cockcroft-Gault)"
}

// Version returns the formula version.
func (c *CrClCalculator) Version() string {
	return crclVersion
}

// Reference returns the clinical citation.
func (c *CrClCalculator) Reference() string {
	return crclReference
}

// Calculate computes CrCl using the Cockcroft-Gault equation.
//
// Formula:
//   CrCl = [(140 - age) × weight] / (72 × SCr)
//   If female: multiply result by 0.85
//
// Where:
//   - age is in years
//   - weight is actual body weight in kg
//   - SCr is serum creatinine in mg/dL
//
// Returns CrCl in mL/min (NOT normalized to BSA)
func (c *CrClCalculator) Calculate(ctx context.Context, params *models.CrClParams) (*models.CrClResult, error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return nil, err
	}

	// Calculate CrCl using Cockcroft-Gault formula
	crcl := c.calculateCrCl(params.AgeYears, params.Sex, params.WeightKg, params.SerumCreatinine)

	// Determine renal function category
	renalFunction := c.categorizeRenalFunction(crcl)

	// Build result
	result := &models.CrClResult{
		Value:                       math.Round(crcl*10) / 10, // Round to 1 decimal
		Unit:                        "mL/min",
		RenalFunction:               renalFunction,
		RequiresRenalDoseAdjustment: renalFunction.RequiresDoseAdjustment(),
		Equation:                    crclVersion,
		Interpretation:              c.buildInterpretation(crcl),
		Provenance:                  c.buildProvenance(params, crcl),
	}

	// Add dose adjustment guidance if needed
	if result.RequiresRenalDoseAdjustment {
		result.DoseAdjustmentGuidance = c.getDoseAdjustmentGuidance(crcl)
	}

	return result, nil
}

// calculateCrCl implements the Cockcroft-Gault equation.
func (c *CrClCalculator) calculateCrCl(age int, sex models.Sex, weight, scr float64) float64 {
	// Base calculation: (140 - age) × weight / (72 × SCr)
	crcl := ((140.0 - float64(age)) * weight) / (72.0 * scr)

	// Apply sex coefficient
	if sex == models.SexFemale {
		crcl *= 0.85
	}

	// Ensure non-negative result
	if crcl < 0 {
		crcl = 0
	}

	return crcl
}

// categorizeRenalFunction determines renal function category based on CrCl.
func (c *CrClCalculator) categorizeRenalFunction(crcl float64) models.RenalFunctionCategory {
	switch {
	case crcl >= 90:
		return models.RenalFunctionNormal
	case crcl >= 60:
		return models.RenalFunctionMild
	case crcl >= 30:
		return models.RenalFunctionModerate
	case crcl >= 15:
		return models.RenalFunctionSevere
	default:
		return models.RenalFunctionEndStage
	}
}

// buildInterpretation creates a clinical interpretation string.
func (c *CrClCalculator) buildInterpretation(crcl float64) string {
	roundedCrCl := math.Round(crcl*10) / 10

	switch {
	case crcl >= 90:
		return fmt.Sprintf("CrCl %.1f mL/min - Normal renal function. Standard dosing appropriate.", roundedCrCl)
	case crcl >= 60:
		return fmt.Sprintf("CrCl %.1f mL/min - Mild impairment. Most drugs can be used at standard doses.", roundedCrCl)
	case crcl >= 30:
		return fmt.Sprintf("CrCl %.1f mL/min - Moderate impairment. Dose adjustment required for many drugs.", roundedCrCl)
	case crcl >= 15:
		return fmt.Sprintf("CrCl %.1f mL/min - Severe impairment. Significant dose reduction required.", roundedCrCl)
	default:
		return fmt.Sprintf("CrCl %.1f mL/min - End-stage renal disease. Many drugs contraindicated.", roundedCrCl)
	}
}

// getDoseAdjustmentGuidance provides dosing recommendations based on CrCl.
func (c *CrClCalculator) getDoseAdjustmentGuidance(crcl float64) string {
	switch {
	case crcl >= 30 && crcl < 50:
		return "Moderate impairment (CrCl 30-49): Many drugs require 50% dose or extended intervals."
	case crcl >= 15 && crcl < 30:
		return "Severe impairment (CrCl 15-29): Significant dose reduction (25-50% of normal). Avoid nephrotoxins."
	case crcl < 15:
		return "ESRD (CrCl <15): Consult nephrology. Many medications contraindicated or require dialysis adjustment."
	default:
		return "Consider dose adjustment for renally-cleared medications."
	}
}

// buildProvenance creates the SaMD provenance record.
func (c *CrClCalculator) buildProvenance(params *models.CrClParams, crcl float64) models.Provenance {
	caveats := []string{
		"Uses actual body weight - may overestimate in obesity",
		"Not validated for patients with rapidly changing kidney function",
		"Consider ideal body weight for obese patients (>130% IBW)",
		"May underestimate in elderly patients with low muscle mass",
	}

	// Add obesity warning if BMI suggests overweight
	if params.WeightKg > 100 {
		caveats = append(caveats, "Patient weight >100kg: Consider using adjusted body weight for drug dosing")
	}

	return models.Provenance{
		CalculatorType: string(models.CalculatorCrCl),
		Version:        crclVersion,
		Formula:        crclFormula,
		Reference:      crclReference,
		CalculatedAt:   time.Now().UTC(),
		InputsUsed: []models.InputUsed{
			{Name: "serum_creatinine", Value: params.SerumCreatinine, Unit: "mg/dL", Source: "lab_result"},
			{Name: "age", Value: params.AgeYears, Unit: "years", Source: "demographics"},
			{Name: "sex", Value: string(params.Sex), Source: "demographics"},
			{Name: "weight", Value: params.WeightKg, Unit: "kg", Source: "vitals"},
		},
		DataQuality: models.DataQualityComplete,
		Caveats:     caveats,
	}
}
