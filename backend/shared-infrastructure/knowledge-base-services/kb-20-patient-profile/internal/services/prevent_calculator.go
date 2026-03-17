package services

import "math"

// ── PREVENT Model Types ──

type PREVENTModelVariant string

const (
	PREVENTModelBase  PREVENTModelVariant = "BASE"
	PREVENTModelHbA1c PREVENTModelVariant = "HBA1C"
	PREVENTModelUACR  PREVENTModelVariant = "UACR"
	PREVENTModelFull  PREVENTModelVariant = "FULL"
)

type Sex string

const (
	SexMale   Sex = "MALE"
	SexFemale Sex = "FEMALE"
)

type PREVENTRiskTier string

const (
	RiskTierLow          PREVENTRiskTier = "LOW"
	RiskTierBorderline   PREVENTRiskTier = "BORDERLINE"
	RiskTierIntermediate PREVENTRiskTier = "INTERMEDIATE"
	RiskTierHigh         PREVENTRiskTier = "HIGH"
)

// PREVENTInput holds the 10 base + 2 optional variables for the PREVENT equations.
type PREVENTInput struct {
	Age              float64
	Sex              Sex
	TotalCholesterol float64 // mg/dL, valid 130-320
	HDLCholesterol   float64 // mg/dL, valid 20-100
	SystolicBP       float64 // mmHg, valid 90-200
	OnBPTreatment    bool
	DiabetesStatus   bool
	CurrentSmoking   bool
	EGFR             float64 // mL/min/1.73m², valid 15-140
	BMI              float64 // kg/m², valid 18.5-39.9

	// Optional — set nil if unavailable
	HbA1c *float64 // %, valid 4.5-15.0
	UACR  *float64 // mg/g, valid 0.1-25000

	ModelVariant PREVENTModelVariant
}

// PREVENTResult holds the 5 outcomes × 2 horizons.
type PREVENTResult struct {
	TenYearTotalCVD float64 `json:"ten_year_total_cvd"`
	TenYearASCVD    float64 `json:"ten_year_ascvd"`
	TenYearHF       float64 `json:"ten_year_hf"`
	TenYearCHD      float64 `json:"ten_year_chd"`
	TenYearStroke   float64 `json:"ten_year_stroke"`

	ThirtyYearTotalCVD float64 `json:"thirty_year_total_cvd,omitempty"`
	ThirtyYearASCVD    float64 `json:"thirty_year_ascvd,omitempty"`
	ThirtyYearHF       float64 `json:"thirty_year_hf,omitempty"`
	ThirtyYearCHD      float64 `json:"thirty_year_chd,omitempty"`
	ThirtyYearStroke   float64 `json:"thirty_year_stroke,omitempty"`

	RiskTier  PREVENTRiskTier     `json:"risk_tier"`
	SBPTarget float64             `json:"sbp_target"`
	ModelUsed PREVENTModelVariant `json:"model_used"`
}

// SelectPREVENTModel determines which model variant to use based on available data.
func SelectPREVENTModel(hba1c, uacr *float64) PREVENTModelVariant {
	hasHbA1c := hba1c != nil
	hasUACR := uacr != nil
	switch {
	case hasHbA1c && hasUACR:
		return PREVENTModelFull
	case hasHbA1c:
		return PREVENTModelHbA1c
	case hasUACR:
		return PREVENTModelUACR
	default:
		return PREVENTModelBase
	}
}

// ComputePREVENT implements the AHA PREVENT equations (Khan et al., Circulation 2024).
// Sex-specific coefficients with piecewise linear splines and competing risk adjustment.
//
// IMPORTANT: The coefficients below are placeholders structured to match the
// published equation form. Before clinical deployment, these MUST be replaced
// with the exact coefficients from:
//   - Khan et al., Circulation 2024;149:430-449, Supplemental Tables S1-S24
//   - Cross-validated against the R 'preventr' package
//
// The equation form is:
//
//	risk = 1 - S0(t)^exp(Σ βi·xi - mean_linear_predictor)
//
// where S0(t) is the baseline survival, βi are the coefficients, and xi are
// the centered/splined input values.
func ComputePREVENT(input PREVENTInput) PREVENTResult {
	// Auto-select model if not specified
	if input.ModelVariant == "" {
		input.ModelVariant = SelectPREVENTModel(input.HbA1c, input.UACR)
	}

	coeffs := getCoefficients(input.Sex, input.ModelVariant)

	// Compute linear predictor from input variables
	lp := computeLinearPredictor(input, coeffs)

	// Compute 10-year risks for all 5 outcomes using cause-specific hazards
	result := PREVENTResult{
		ModelUsed: input.ModelVariant,
	}

	result.TenYearTotalCVD = computeRisk(lp.totalCVD, coeffs.baselineSurvival10yr.totalCVD, coeffs.meanLP.totalCVD)
	result.TenYearASCVD = computeRisk(lp.ascvd, coeffs.baselineSurvival10yr.ascvd, coeffs.meanLP.ascvd)
	result.TenYearHF = computeRisk(lp.hf, coeffs.baselineSurvival10yr.hf, coeffs.meanLP.hf)
	result.TenYearCHD = computeRisk(lp.chd, coeffs.baselineSurvival10yr.chd, coeffs.meanLP.chd)
	result.TenYearStroke = computeRisk(lp.stroke, coeffs.baselineSurvival10yr.stroke, coeffs.meanLP.stroke)

	// 30-year risks (available for age 30-59)
	if input.Age >= 30 && input.Age <= 59 {
		result.ThirtyYearTotalCVD = computeRisk(lp.totalCVD, coeffs.baselineSurvival30yr.totalCVD, coeffs.meanLP.totalCVD)
		result.ThirtyYearASCVD = computeRisk(lp.ascvd, coeffs.baselineSurvival30yr.ascvd, coeffs.meanLP.ascvd)
		result.ThirtyYearHF = computeRisk(lp.hf, coeffs.baselineSurvival30yr.hf, coeffs.meanLP.hf)
		result.ThirtyYearCHD = computeRisk(lp.chd, coeffs.baselineSurvival30yr.chd, coeffs.meanLP.chd)
		result.ThirtyYearStroke = computeRisk(lp.stroke, coeffs.baselineSurvival30yr.stroke, coeffs.meanLP.stroke)
	}

	// Risk tier and SBP target
	result.RiskTier = ClassifyRiskTier(result.TenYearTotalCVD)

	egfr := input.EGFR
	acr := 0.0
	if input.UACR != nil {
		acr = *input.UACR
	}
	// Default intensive threshold 0.075 (INTERMEDIATE tier boundary).
	// In production, this is loaded from prevent_config.yaml `intensive_target_threshold`.
	// ComputePREVENT is a pure function — the caller passes the threshold.
	intensiveThreshold := 0.075
	result.SBPTarget = DetermineSBPTarget(result.RiskTier, result.TenYearTotalCVD, egfr, acr, intensiveThreshold)

	return result
}

// ClassifyRiskTier maps 10-year total CVD risk to the 4-tier AHA classification.
func ClassifyRiskTier(tenYearCVD float64) PREVENTRiskTier {
	switch {
	case tenYearCVD >= 0.20:
		return RiskTierHigh
	case tenYearCVD >= 0.075:
		return RiskTierIntermediate
	case tenYearCVD >= 0.05:
		return RiskTierBorderline
	default:
		return RiskTierLow
	}
}

// DetermineSBPTarget returns 120 or 130 mmHg per ADA 2026 + BPROAD/ESPRIT evidence.
// SBP <120 if: 10yr CVD ≥ intensiveThreshold (default 0.075 = INTERMEDIATE) OR eGFR <60 OR ACR ≥300.
// The intensiveThreshold is loaded from prevent_config.yaml `intensive_target_threshold`.
// This makes the tier cutoff clinician-configurable without code changes.
func DetermineSBPTarget(tier PREVENTRiskTier, tenYearCVD, egfr, acr, intensiveThreshold float64) float64 {
	// Risk-based intensive target: use the numeric 10yr CVD risk against the
	// configurable threshold rather than comparing tier strings. This means
	// changing the threshold in YAML automatically adjusts which patients
	// get intensive targeting — no tier-string coupling.
	if tenYearCVD >= intensiveThreshold {
		return 120
	}
	if egfr < 60 {
		return 120
	}
	if acr >= 300 {
		return 120
	}
	return 130
}

// ApplySouthAsianBMICalibration applies the BMI+offset calibration for South Asian
// patients with BMI 23-30 (where PREVENT underestimates risk). BMI ≥30 uses raw value.
func ApplySouthAsianBMICalibration(bmi, offset float64) float64 {
	if bmi >= 23 && bmi < 30 {
		return bmi + offset
	}
	return bmi
}

// ── Internal equation machinery ──

type outcomeSet struct {
	totalCVD, ascvd, hf, chd, stroke float64
}

type linearPredictors struct {
	totalCVD, ascvd, hf, chd, stroke float64
}

type preventCoefficients struct {
	// Per-variable coefficients (sex-specific, model-variant-specific)
	age, totalChol, hdlChol, sbp, egfr, bmi float64
	bpTreatment, diabetes, smoking           float64
	hba1c, uacr                              float64 // optional model add-ons

	// Piecewise linear spline knots
	sbpKnot, egfrKnot, bmiKnot float64

	// Baseline survival and mean linear predictor per outcome
	baselineSurvival10yr outcomeSet
	baselineSurvival30yr outcomeSet
	meanLP               outcomeSet
}

// getCoefficients returns sex-specific, model-variant-specific PREVENT coefficients.
// PLACEHOLDER: These must be replaced with published coefficients from Khan et al.
// Supplemental Tables before clinical use.
func getCoefficients(sex Sex, model PREVENTModelVariant) preventCoefficients {
	// TODO: Replace with actual coefficients from Khan et al. Supplemental Tables S1-S24.
	// The structure is correct; the values are illustrative.
	// Cross-validate against R 'preventr' package before deployment.
	if sex == SexFemale {
		return preventCoefficients{
			age: 0.064, totalChol: 0.002, hdlChol: -0.010, sbp: 0.017,
			egfr: -0.008, bmi: 0.014, bpTreatment: 0.229, diabetes: 0.519,
			smoking: 0.459, hba1c: 0.098, uacr: 0.0004,
			sbpKnot: 110, egfrKnot: 60, bmiKnot: 30,
			baselineSurvival10yr: outcomeSet{0.965, 0.978, 0.987, 0.985, 0.991},
			baselineSurvival30yr: outcomeSet{0.890, 0.925, 0.950, 0.945, 0.965},
			meanLP:               outcomeSet{2.15, 1.82, 1.45, 1.55, 1.20},
		}
	}
	return preventCoefficients{
		age: 0.072, totalChol: 0.002, hdlChol: -0.012, sbp: 0.018,
		egfr: -0.009, bmi: 0.012, bpTreatment: 0.254, diabetes: 0.478,
		smoking: 0.501, hba1c: 0.105, uacr: 0.0005,
		sbpKnot: 110, egfrKnot: 60, bmiKnot: 30,
		baselineSurvival10yr: outcomeSet{0.945, 0.960, 0.975, 0.970, 0.982},
		baselineSurvival30yr: outcomeSet{0.860, 0.900, 0.930, 0.920, 0.950},
		meanLP:               outcomeSet{2.55, 2.20, 1.75, 1.90, 1.40},
	}
}

func computeLinearPredictor(input PREVENTInput, c preventCoefficients) linearPredictors {
	// Piecewise linear splines
	sbpTerm := piecewiseLinear(input.SystolicBP, c.sbpKnot, c.sbp)
	egfrTerm := piecewiseLinear(input.EGFR, c.egfrKnot, c.egfr)
	bmiTerm := piecewiseLinear(input.BMI, c.bmiKnot, c.bmi)

	lp := c.age*input.Age +
		c.totalChol*input.TotalCholesterol/10.0 + // per 10 mg/dL
		c.hdlChol*input.HDLCholesterol/10.0 +
		sbpTerm + egfrTerm + bmiTerm

	if input.OnBPTreatment {
		lp += c.bpTreatment
	}
	if input.DiabetesStatus {
		lp += c.diabetes
	}
	if input.CurrentSmoking {
		lp += c.smoking
	}

	// Optional add-on variables
	if input.HbA1c != nil && (input.ModelVariant == PREVENTModelHbA1c || input.ModelVariant == PREVENTModelFull) {
		lp += c.hba1c * (*input.HbA1c)
	}
	if input.UACR != nil && (input.ModelVariant == PREVENTModelUACR || input.ModelVariant == PREVENTModelFull) {
		lp += c.uacr * math.Log(*input.UACR+1)
	}

	// ARCHITECTURAL NOTE: The real PREVENT equations (Khan et al. Supplemental
	// Tables S1-S24) define SEPARATE coefficient sets per outcome (total CVD,
	// ASCVD, HF, CHD, stroke) × sex × model variant = up to 40 coefficient
	// vectors. The placeholder below uses a single LP scaled by fixed multipliers
	// as a structural scaffold. When replacing coefficients:
	//   1. Change preventCoefficients to hold per-outcome coefficient maps
	//   2. Compute each LP independently: lp_ascvd = Σ β_ascvd_i · x_i
	//   3. Each outcome gets its own baselineSurvival and meanLP
	// The linearPredictors return type already supports this — only this
	// function body and getCoefficients() need to change.
	return linearPredictors{
		totalCVD: lp,
		ascvd:    lp * 0.92, // placeholder scaling — replace with independent LP
		hf:       lp * 0.85, // placeholder scaling — replace with independent LP
		chd:      lp * 0.88, // placeholder scaling — replace with independent LP
		stroke:   lp * 0.78, // placeholder scaling — replace with independent LP
	}
}

// piecewiseLinear implements a hinge function at the knot point.
// Below the knot: contribution is zero. Above the knot: linear in (value - knot).
// PLACEHOLDER: The real PREVENT equations use restricted cubic splines with
// 5 knots per continuous predictor. This hinge is a structural scaffold —
// replace with the published spline form when inserting real coefficients.
func piecewiseLinear(value, knot, coeff float64) float64 {
	if value <= knot {
		return 0
	}
	return coeff * (value - knot)
}

// computeRisk applies the Cox proportional hazards formula.
// Defensive clamps: for valid S0 ∈ (0,1) and finite LP, the result is always
// in [0,1]. The clamps guard against floating-point edge cases or invalid
// placeholder coefficients — they should never trigger with real coefficients.
func computeRisk(lp, baselineSurvival, meanLP float64) float64 {
	risk := 1 - math.Pow(baselineSurvival, math.Exp(lp-meanLP))
	if risk < 0 {
		return 0
	}
	if risk > 1 {
		return 1
	}
	return risk
}
