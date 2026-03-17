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

// PREVENTInput holds the base + optional variables for the PREVENT equations.
// Non-HDL cholesterol is derived internally: nonHDL = (TC - HDL) / 38.67 [mg/dL → mmol/L].
type PREVENTInput struct {
	Age              float64
	Sex              Sex
	TotalCholesterol float64 // mg/dL, valid 130-320
	HDLCholesterol   float64 // mg/dL, valid 20-100
	SystolicBP       float64 // mmHg, valid 90-200
	OnBPTreatment    bool
	OnStatin         bool    // needed for Statin×nonHDL interaction (C15)
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

// ComputePREVENT implements the AHA PREVENT equations (Khan et al., Circulation 2024;149:430-449).
//
// EQUATION FORM: Logistic regression (NOT Cox proportional hazards).
//
// Coefficients are from the preventr R package (v0.11.0, CRAN), which embeds the
// published values from Khan et al., Supplemental Tables S1-S24.
//
// Key transforms (centering values from Khan et al.):
//   - Age:    centered at 55, scaled by 10 → (Age-55)/10
//   - nonHDL: centered at 3.5 mmol/L
//   - HDL:    centered at 1.3 mmol/L, scaled by 0.3
//   - SBP:    piecewise spline knot at 110 mmHg
//   - BMI:    piecewise spline knot at 30 kg/m², lower segment centered at 25
//   - eGFR:   piecewise spline knot at 60, upper segment centered at 90
//   - HbA1c:  centered at 5.3%, DM-conditional (C28 for DM, C29 for non-DM)
//   - UACR:   natural log transform, ln(UACR)
//
// Missing-data handling: when HbA1c or UACR data is not available, the model adds
// a calibrated missing-data offset (hba1cMissing / uacrMissing) instead of zero.
// SDI (Social Deprivation Index) is always marked missing for non-US populations.
func ComputePREVENT(input PREVENTInput) PREVENTResult {
	if input.ModelVariant == "" {
		input.ModelVariant = SelectPREVENTModel(input.HbA1c, input.UACR)
	}

	// Derive non-HDL and HDL cholesterol in mmol/L
	nonHDLmmol := (input.TotalCholesterol - input.HDLCholesterol) / 38.67
	hdlMmol := input.HDLCholesterol / 38.67

	// Build centered/scaled/splined input vector
	xv := buildTransformedInputs(input, nonHDLmmol, hdlMmol)

	// Get per-outcome coefficient sets for this sex × model variant
	coeffSet := getLogisticCoefficients(input.Sex, input.ModelVariant)

	// Compute 10-year risks
	result := PREVENTResult{
		ModelUsed: input.ModelVariant,
	}

	result.TenYearTotalCVD = logisticRisk(xv, coeffSet.totalCVD10yr)
	result.TenYearASCVD = logisticRisk(xv, coeffSet.ascvd10yr)
	result.TenYearHF = logisticRisk(xv, coeffSet.hf10yr)
	result.TenYearCHD = logisticRisk(xv, coeffSet.chd10yr)
	result.TenYearStroke = logisticRisk(xv, coeffSet.stroke10yr)

	// 30-year risks (available for age 30-59)
	if input.Age >= 30 && input.Age <= 59 {
		result.ThirtyYearTotalCVD = logisticRisk(xv, coeffSet.totalCVD30yr)
		result.ThirtyYearASCVD = logisticRisk(xv, coeffSet.ascvd30yr)
		result.ThirtyYearHF = logisticRisk(xv, coeffSet.hf30yr)
		result.ThirtyYearCHD = logisticRisk(xv, coeffSet.chd30yr)
		result.ThirtyYearStroke = logisticRisk(xv, coeffSet.stroke30yr)
	}

	// Risk tier and SBP target
	result.RiskTier = ClassifyRiskTier(result.TenYearTotalCVD)

	egfr := input.EGFR
	acr := 0.0
	if input.UACR != nil {
		acr = *input.UACR
	}
	intensiveThreshold := 0.075
	result.SBPTarget = DetermineSBPTarget(result.TenYearTotalCVD, egfr, acr, intensiveThreshold)

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
func DetermineSBPTarget(tenYearCVD, egfr, acr, intensiveThreshold float64) float64 {
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

// ── Internal logistic regression machinery ──

// transformedInputs holds the pre-computed, centered/scaled/splined input vector.
type transformedInputs struct {
	// C0-C1: Age (centered at 55, scaled by /10)
	ageScaled   float64 // (Age - 55) / 10
	ageScaledSq float64 // ((Age - 55) / 10)²

	// C2-C3: Lipids (centered)
	nonHDLcent float64 // nonHDL_mmol - 3.5
	hdlScaled  float64 // (HDL_mmol - 1.3) / 0.3

	// C4-C5: SBP piecewise spline (knot at 110 mmHg)
	sbpLower float64 // (min(SBP, 110) - 110) / 20
	sbpUpper float64 // (max(SBP, 110) - 130) / 20

	// C6-C7: Binary predictors
	diabetes float64 // 0/1
	smoking  float64 // 0/1

	// C8-C9: BMI piecewise spline (knot at 30, lower centered at 25)
	bmiLower float64 // (min(BMI, 30) - 25) / 5
	bmiUpper float64 // (max(BMI, 30) - 30) / 5

	// C10-C11: eGFR piecewise spline (knot at 60, upper centered at 90)
	egfrLower float64 // (min(eGFR, 60) - 60) / (-15)
	egfrUpper float64 // (max(eGFR, 60) - 90) / (-15)

	// C12-C13: Treatment binary predictors
	bpTx   float64 // 0/1
	statin float64 // 0/1

	// C14-C15: Treatment × continuous interactions
	bpTxSbpUpper float64 // BPTx × sbpUpper
	statinNonHDL float64 // Statin × nonHDLcent

	// C16-C22: Age interaction terms
	ageNonHDL    float64 // ageScaled × nonHDLcent
	ageHDL       float64 // ageScaled × hdlScaled
	ageSbpUpper  float64 // ageScaled × sbpUpper
	ageDiabetes  float64 // ageScaled × diabetes
	ageSmoking   float64 // ageScaled × smoking
	ageBmiUpper  float64 // ageScaled × bmiUpper
	ageEgfrLower float64 // ageScaled × egfrLower

	// Optional variables
	hba1cCentDM   float64 // (HbA1c - 5.3) if DM and HbA1c available, else 0
	hba1cCentNoDM float64 // (HbA1c - 5.3) if not DM and HbA1c available, else 0
	hasHbA1c      bool
	logUACR       float64 // ln(UACR) if available, else 0
	hasUACR       bool
}

// buildTransformedInputs computes the centered/scaled/splined input vector.
func buildTransformedInputs(input PREVENTInput, nonHDLmmol, hdlMmol float64) transformedInputs {
	ageScaled := (input.Age - 55) / 10.0
	nonHDLcent := nonHDLmmol - 3.5
	hdlScaled := (hdlMmol - 1.3) / 0.3
	sbpLower := (math.Min(input.SystolicBP, 110) - 110) / 20.0
	sbpUpper := (math.Max(input.SystolicBP, 110) - 130) / 20.0
	// BMI: knot at 30, lower segment centered at 25 (FIX: was min(BMI,25))
	bmiLower := (math.Min(input.BMI, 30) - 25) / 5.0
	bmiUpper := (math.Max(input.BMI, 30) - 30) / 5.0
	egfrLower := (math.Min(input.EGFR, 60) - 60) / (-15.0)
	egfrUpper := (math.Max(input.EGFR, 60) - 90) / (-15.0)

	bpTx := boolToFloat(input.OnBPTreatment)
	stat := boolToFloat(input.OnStatin)
	dm := boolToFloat(input.DiabetesStatus)
	smk := boolToFloat(input.CurrentSmoking)

	xv := transformedInputs{
		ageScaled:   ageScaled,
		ageScaledSq: ageScaled * ageScaled,
		nonHDLcent:  nonHDLcent,
		hdlScaled:   hdlScaled,
		sbpLower:    sbpLower,
		sbpUpper:    sbpUpper,
		diabetes:    dm,
		smoking:     smk,
		bmiLower:    bmiLower,
		bmiUpper:    bmiUpper,
		egfrLower:   egfrLower,
		egfrUpper:   egfrUpper,
		bpTx:        bpTx,
		statin:      stat,

		// Treatment × continuous interactions
		bpTxSbpUpper: bpTx * sbpUpper,
		statinNonHDL: stat * nonHDLcent,

		// Age × predictor interactions
		ageNonHDL:    ageScaled * nonHDLcent,
		ageHDL:       ageScaled * hdlScaled,
		ageSbpUpper:  ageScaled * sbpUpper,
		ageDiabetes:  ageScaled * dm,
		ageSmoking:   ageScaled * smk,
		ageBmiUpper:  ageScaled * bmiUpper,
		ageEgfrLower: ageScaled * egfrLower,
	}

	// HbA1c: centered at 5.3, DM-conditional
	if input.HbA1c != nil && (input.ModelVariant == PREVENTModelHbA1c || input.ModelVariant == PREVENTModelFull) {
		xv.hasHbA1c = true
		centered := *input.HbA1c - 5.3
		if input.DiabetesStatus {
			xv.hba1cCentDM = centered
		} else {
			xv.hba1cCentNoDM = centered
		}
	}

	// UACR: natural log (FIX: was log(UACR+1))
	if input.UACR != nil && (input.ModelVariant == PREVENTModelUACR || input.ModelVariant == PREVENTModelFull) {
		xv.logUACR = math.Log(*input.UACR)
		xv.hasUACR = true
	}

	return xv
}

// logisticCoeffs holds the coefficient vector for one outcome × one horizon.
// Coefficients from Khan et al., Circulation 2024;149:430-449, Supplemental Tables S1-S24.
type logisticCoeffs struct {
	intercept float64

	// Base predictors (EBMcalc C-indices vary by model variant; field names are canonical)
	age       float64 // ageScaled
	ageSq     float64 // ageScaledSq (30yr models only)
	nonHDL    float64 // nonHDLcent
	hdl       float64 // hdlScaled
	sbpLower  float64 // SBP lower spline
	sbpUpper  float64 // SBP upper spline
	diabetes  float64
	smoking   float64
	bmiLower  float64 // BMI lower spline
	bmiUpper  float64 // BMI upper spline
	egfrLower float64 // eGFR lower spline
	egfrUpper float64 // eGFR upper spline
	bpTx      float64
	statin    float64

	// Treatment × continuous interactions
	bpTxSBP      float64 // BPTx × sbpUpper
	statinNonHDL float64 // Statin × nonHDLcent

	// Age interaction terms
	ageSBP      float64 // ageScaled × sbpUpper
	ageSmoking  float64 // ageScaled × smoking
	ageDiabetes float64 // ageScaled × diabetes
	ageBMI      float64 // ageScaled × bmiUpper
	ageEGFR     float64 // ageScaled × egfrLower
	ageNonHDL   float64 // ageScaled × nonHDLcent
	ageHDL      float64 // ageScaled × hdlScaled

	// Optional: HbA1c (DM-conditional, centered at 5.3)
	hba1cDM      float64 // coefficient when diabetes present
	hba1cNoDM    float64 // coefficient when diabetes absent
	hba1cMissing float64 // offset when HbA1c data not available

	// Optional: UACR
	logUACR     float64 // coefficient for ln(UACR)
	uacrMissing float64 // offset when UACR data not available

	// SDI missing indicator (always applied for non-US populations)
	sdiMissing float64 // offset for missing SDI
}

// outcomeCoeffSet holds per-outcome coefficient vectors for all 5 outcomes × 2 horizons.
type outcomeCoeffSet struct {
	totalCVD10yr, ascvd10yr, hf10yr, chd10yr, stroke10yr logisticCoeffs
	totalCVD30yr, ascvd30yr, hf30yr, chd30yr, stroke30yr logisticCoeffs
}

// logisticRisk computes risk = sigmoid(logit) = exp(logit) / (1 + exp(logit)).
func logisticRisk(xv transformedInputs, c logisticCoeffs) float64 {
	logit := c.intercept +
		// Base predictors (C0-C13)
		c.age*xv.ageScaled +
		c.ageSq*xv.ageScaledSq +
		c.nonHDL*xv.nonHDLcent +
		c.hdl*xv.hdlScaled +
		c.sbpLower*xv.sbpLower +
		c.sbpUpper*xv.sbpUpper +
		c.diabetes*xv.diabetes +
		c.smoking*xv.smoking +
		c.bmiLower*xv.bmiLower +
		c.bmiUpper*xv.bmiUpper +
		c.egfrLower*xv.egfrLower +
		c.egfrUpper*xv.egfrUpper +
		c.bpTx*xv.bpTx +
		c.statin*xv.statin +
		// Treatment × continuous interactions (C14-C15)
		c.bpTxSBP*xv.bpTxSbpUpper +
		c.statinNonHDL*xv.statinNonHDL +
		// Age interaction terms (C16-C22)
		c.ageSBP*xv.ageSbpUpper +
		c.ageSmoking*xv.ageSmoking +
		c.ageDiabetes*xv.ageDiabetes +
		c.ageBMI*xv.ageBmiUpper +
		c.ageEGFR*xv.ageEgfrLower +
		c.ageNonHDL*xv.ageNonHDL +
		c.ageHDL*xv.ageHDL

	// SDI missing — always added for non-US populations
	logit += c.sdiMissing

	// HbA1c contribution (DM-conditional, centered at 5.3)
	if xv.hasHbA1c {
		logit += c.hba1cDM*xv.hba1cCentDM + c.hba1cNoDM*xv.hba1cCentNoDM
	} else {
		logit += c.hba1cMissing
	}

	// UACR contribution
	if xv.hasUACR {
		logit += c.logUACR * xv.logUACR
	} else {
		logit += c.uacrMissing
	}

	return sigmoid(logit)
}

// sigmoid computes the inverse logit: exp(x) / (1 + exp(x)).
func sigmoid(x float64) float64 {
	if x >= 0 {
		ez := math.Exp(-x)
		return 1.0 / (1.0 + ez)
	}
	ez := math.Exp(x)
	return ez / (1.0 + ez)
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// getLogisticCoefficients returns per-outcome logistic regression coefficients
// from Khan et al., Circulation 2024;149:430-449, Supplemental Tables S1-S24.
// Coefficients extracted from the preventr R package (v0.11.0, CRAN).
func getLogisticCoefficients(sex Sex, model PREVENTModelVariant) outcomeCoeffSet {
	if sex == SexFemale {
		return femaleLogisticCoeffs(model)
	}
	return maleLogisticCoeffs(model)
}

func femaleLogisticCoeffs(model PREVENTModelVariant) outcomeCoeffSet {
	switch model {
	case PREVENTModelBase:
		return outcomeCoeffSet{
			totalCVD10yr: logisticCoeffs{
				intercept: -3.307728,
				age: 0.793933, ageSq: 0,
				nonHDL: 0.030524, hdl: -0.160686,
				sbpLower: -0.239400, sbpUpper: 0.360078,
				diabetes: 0.866760, smoking: 0.536074,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.604592, egfrUpper: 0.043377,
				bpTx: 0.315167, statin: -0.147765,
				bpTxSBP: -0.066361, statinNonHDL: 0.119788,
				ageSBP: -0.094635, ageSmoking: -0.078715, ageDiabetes: -0.270570,
				ageBMI: 0, ageEGFR: -0.163781,
				ageNonHDL: -0.081972, ageHDL: 0.030677,
			},
			ascvd10yr: logisticCoeffs{
				intercept: -3.819975,
				age: 0.719883, ageSq: 0,
				nonHDL: 0.117697, hdl: -0.151185,
				sbpLower: -0.083536, sbpUpper: 0.359285,
				diabetes: 0.834858, smoking: 0.483108,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.486462, egfrUpper: 0.039778,
				bpTx: 0.226531, statin: -0.059237,
				bpTxSBP: -0.039576, statinNonHDL: 0.084442,
				ageSBP: -0.103598, ageSmoking: -0.079114, ageDiabetes: -0.241754,
				ageBMI: 0, ageEGFR: -0.167149,
				ageNonHDL: -0.056784, ageHDL: 0.032569,
			},
			hf10yr: logisticCoeffs{
				intercept: -4.310409,
				age: 0.899823, ageSq: 0,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.455977, sbpUpper: 0.357650,
				diabetes: 1.038346, smoking: 0.583916,
				bmiLower: -0.007229, bmiUpper: 0.299771,
				egfrLower: 0.745164, egfrUpper: 0.055709,
				bpTx: 0.353444, statin: 0,
				bpTxSBP: -0.098151, statinNonHDL: 0,
				ageSBP: -0.094666, ageSmoking: -0.115945, ageDiabetes: -0.358104,
				ageBMI: -0.003878, ageEGFR: -0.188429,
				ageNonHDL: 0, ageHDL: 0,
			},
			chd10yr: logisticCoeffs{
				intercept: -4.608751,
				age: 0.758715, ageSq: 0,
				nonHDL: 0.181095, hdl: -0.201451,
				sbpLower: -0.088183, sbpUpper: 0.354773,
				diabetes: 0.904536, smoking: 0.541092,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.519872, egfrUpper: 0.032593,
				bpTx: 0.201064, statin: -0.036195,
				bpTxSBP: -0.089124, statinNonHDL: 0.075072,
				ageSBP: -0.089809, ageSmoking: -0.078661, ageDiabetes: -0.256904,
				ageBMI: 0, ageEGFR: -0.159751,
				ageNonHDL: -0.068326, ageHDL: 0.048475,
			},
			stroke10yr: logisticCoeffs{
				intercept: -4.409199,
				age: 0.690785, ageSq: 0,
				nonHDL: 0.053428, hdl: -0.105511,
				sbpLower: -0.113078, sbpUpper: 0.366522,
				diabetes: 0.801372, smoking: 0.418704,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.453977, egfrUpper: 0.051509,
				bpTx: 0.249462, statin: -0.079883,
				bpTxSBP: -0.007904, statinNonHDL: 0.083310,
				ageSBP: -0.119121, ageSmoking: -0.099806, ageDiabetes: -0.248055,
				ageBMI: 0, ageEGFR: -0.175907,
				ageNonHDL: -0.040924, ageHDL: 0.016994,
			},
			totalCVD30yr: logisticCoeffs{
				intercept: -1.318827,
				age: 0.550308, ageSq: -0.092837,
				nonHDL: 0.040979, hdl: -0.166331,
				sbpLower: -0.162865, sbpUpper: 0.329950,
				diabetes: 0.679389, smoking: 0.319611,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.185710, egfrUpper: 0.055353,
				bpTx: 0.289400, statin: -0.075688,
				bpTxSBP: -0.056367, statinNonHDL: 0.107102,
				ageSBP: -0.099878, ageSmoking: -0.160786, ageDiabetes: -0.320617,
				ageBMI: 0, ageEGFR: -0.145079,
				ageNonHDL: -0.075144, ageHDL: 0.030179,
			},
			ascvd30yr: logisticCoeffs{
				intercept: -1.974074,
				age: 0.466920, ageSq: -0.089312,
				nonHDL: 0.125690, hdl: -0.154225,
				sbpLower: -0.001809, sbpUpper: 0.322949,
				diabetes: 0.629671, smoking: 0.268292,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.100106, egfrUpper: 0.049966,
				bpTx: 0.187529, statin: 0.015248,
				bpTxSBP: -0.027612, statinNonHDL: 0.073615,
				ageSBP: -0.104610, ageSmoking: -0.153091, ageDiabetes: -0.272779,
				ageBMI: 0, ageEGFR: -0.129915,
				ageNonHDL: -0.052196, ageHDL: 0.031692,
			},
			hf30yr: logisticCoeffs{
				intercept: -2.205379,
				age: 0.625437, ageSq: -0.098304,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.391924, sbpUpper: 0.314229,
				diabetes: 0.833079, smoking: 0.343865,
				bmiLower: 0.059487, bmiUpper: 0.252554,
				egfrLower: 0.298164, egfrUpper: 0.066716,
				bpTx: 0.333921, statin: 0,
				bpTxSBP: -0.089318, statinNonHDL: 0,
				ageSBP: -0.097430, ageSmoking: -0.198299, ageDiabetes: -0.404855,
				ageBMI: -0.003562, ageEGFR: -0.156421,
				ageNonHDL: 0, ageHDL: 0,
			},
			chd30yr: logisticCoeffs{
				intercept: -2.733866,
				age: 0.491242, ageSq: -0.091708,
				nonHDL: 0.187826, hdl: -0.203570,
				sbpLower: -0.003022, sbpUpper: 0.311176,
				diabetes: 0.680325, smoking: 0.321531,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.125261, egfrUpper: 0.041458,
				bpTx: 0.156130, statin: 0.038414,
				bpTxSBP: -0.079553, statinNonHDL: 0.063526,
				ageSBP: -0.087648, ageSmoking: -0.151363, ageDiabetes: -0.280310,
				ageBMI: 0, ageEGFR: -0.113045,
				ageNonHDL: -0.063767, ageHDL: 0.047407,
			},
			stroke30yr: logisticCoeffs{
				intercept: -2.620780,
				age: 0.436698, ageSq: -0.087367,
				nonHDL: 0.058633, hdl: -0.106902,
				sbpLower: -0.031711, sbpUpper: 0.327274,
				diabetes: 0.584173, smoking: 0.204568,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.076581, egfrUpper: 0.060323,
				bpTx: 0.208782, statin: -0.009514,
				bpTxSBP: 0.001444, statinNonHDL: 0.072001,
				ageSBP: -0.117906, ageSmoking: -0.170284, ageDiabetes: -0.271022,
				ageBMI: 0, ageEGFR: -0.132099,
				ageNonHDL: -0.036178, ageHDL: 0.015888,
			},
		}
	case PREVENTModelHbA1c:
		return outcomeCoeffSet{
			totalCVD10yr: logisticCoeffs{
				intercept: -3.306162,
				age: 0.785818, ageSq: 0,
				nonHDL: 0.019444, hdl: -0.152196,
				sbpLower: -0.229668, sbpUpper: 0.346578,
				diabetes: 0.536624, smoking: 0.541168,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.593190, egfrUpper: 0.047246,
				bpTx: 0.315857, statin: -0.153517,
				bpTxSBP: -0.068775, statinNonHDL: 0.105475,
				ageSBP: -0.090597, ageSmoking: -0.080186, ageDiabetes: -0.224186,
				ageBMI: 0, ageEGFR: -0.166729,
				ageNonHDL: -0.076112, ageHDL: 0.030747,
				hba1cDM: 0.133835, hba1cNoDM: 0.162241, hba1cMissing: -0.014250,
			},
			ascvd10yr: logisticCoeffs{
				intercept: -3.838746,
				age: 0.711183, ageSq: 0,
				nonHDL: 0.106797, hdl: -0.142574,
				sbpLower: -0.073682, sbpUpper: 0.348084,
				diabetes: 0.511295, smoking: 0.488029,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.475500, egfrUpper: 0.043813,
				bpTx: 0.225909, statin: -0.064887,
				bpTxSBP: -0.043764, statinNonHDL: 0.069708,
				ageSBP: -0.099644, ageSmoking: -0.080354, ageDiabetes: -0.192434,
				ageBMI: 0, ageEGFR: -0.168259,
				ageNonHDL: -0.050638, ageHDL: 0.032747,
				hba1cDM: 0.133906, hba1cNoDM: 0.159646, hba1cMissing: 0.001568,
			},
			hf10yr: logisticCoeffs{
				intercept: -4.288225,
				age: 0.899739, ageSq: 0,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.442275, sbpUpper: 0.337869,
				diabetes: 0.681284, smoking: 0.588600,
				bmiLower: -0.014866, bmiUpper: 0.295837,
				egfrLower: 0.734470, egfrUpper: 0.059260,
				bpTx: 0.354347, statin: 0,
				bpTxSBP: -0.100214, statinNonHDL: 0,
				ageSBP: -0.087876, ageSmoking: -0.117894, ageDiabetes: -0.303684,
				ageBMI: -0.008345, ageEGFR: -0.191218,
				ageNonHDL: 0, ageHDL: 0,
				hba1cDM: 0.185644, hba1cNoDM: 0.183308, hba1cMissing: -0.014311,
			},
			chd10yr: logisticCoeffs{
				intercept: -4.667457,
				age: 0.755942, ageSq: 0,
				nonHDL: 0.166393, hdl: -0.190500,
				sbpLower: -0.076851, sbpUpper: 0.339647,
				diabetes: 0.563223, smoking: 0.547800,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.510509, egfrUpper: 0.038134,
				bpTx: 0.202309, statin: -0.042593,
				bpTxSBP: -0.095379, statinNonHDL: 0.058737,
				ageSBP: -0.084170, ageSmoking: -0.080707, ageDiabetes: -0.198697,
				ageBMI: 0, ageEGFR: -0.161091,
				ageNonHDL: -0.060766, ageHDL: 0.048577,
				hba1cDM: 0.183274, hba1cNoDM: 0.175546, hba1cMissing: 0.026838,
			},
			stroke10yr: logisticCoeffs{
				intercept: -4.396448,
				age: 0.685496, ageSq: 0,
				nonHDL: 0.043261, hdl: -0.098379,
				sbpLower: -0.101720, sbpUpper: 0.356596,
				diabetes: 0.460956, smoking: 0.419830,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.443652, egfrUpper: 0.054499,
				bpTx: 0.249383, statin: -0.085977,
				bpTxSBP: -0.011456, statinNonHDL: 0.068908,
				ageSBP: -0.115679, ageSmoking: -0.100388, ageDiabetes: -0.202582,
				ageBMI: 0, ageEGFR: -0.176008,
				ageNonHDL: -0.036013, ageHDL: 0.017181,
				hba1cDM: 0.092969, hba1cNoDM: 0.154505, hba1cMissing: -0.025509,
			},
			totalCVD30yr: logisticCoeffs{
				intercept: -1.341059,
				age: 0.534349, ageSq: -0.095231,
				nonHDL: 0.029812, hdl: -0.157845,
				sbpLower: -0.150449, sbpUpper: 0.317337,
				diabetes: 0.431474, smoking: 0.320940,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.177144, egfrUpper: 0.058283,
				bpTx: 0.288895, statin: -0.079589,
				bpTxSBP: -0.060044, statinNonHDL: 0.092060,
				ageSBP: -0.095405, ageSmoking: -0.162394, ageDiabetes: -0.276341,
				ageBMI: 0, ageEGFR: -0.143051,
				ageNonHDL: -0.069611, ageHDL: 0.030881,
				hba1cDM: 0.094054, hba1cNoDM: 0.111649, hba1cMissing: -0.002480,
			},
			ascvd30yr: logisticCoeffs{
				intercept: -2.011533,
				age: 0.455557, ageSq: -0.090350,
				nonHDL: 0.114832, hdl: -0.145875,
				sbpLower: 0.008932, sbpUpper: 0.313903,
				diabetes: 0.386281, smoking: 0.271431,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.093099, egfrUpper: 0.053222,
				bpTx: 0.186218, statin: 0.010696,
				bpTxSBP: -0.032971, statinNonHDL: 0.058361,
				ageSBP: -0.100478, ageSmoking: -0.154186, ageDiabetes: -0.226694,
				ageBMI: 0, ageEGFR: -0.128601,
				ageNonHDL: -0.046327, ageHDL: 0.032472,
				hba1cDM: 0.087583, hba1cNoDM: 0.112642, hba1cMissing: 0.012436,
			},
			hf30yr: logisticCoeffs{
				intercept: -2.193553,
				age: 0.621086, ageSq: -0.100097,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.377370, sbpUpper: 0.295316,
				diabetes: 0.568169, smoking: 0.344914,
				bmiLower: 0.054009, bmiUpper: 0.249767,
				egfrLower: 0.287578, egfrUpper: 0.069201,
				bpTx: 0.333494, statin: 0,
				bpTxSBP: -0.092234, statinNonHDL: 0,
				ageSBP: -0.090788, ageSmoking: -0.200885, ageDiabetes: -0.355465,
				ageBMI: -0.007961, ageEGFR: -0.156803,
				ageNonHDL: 0, ageHDL: 0,
				hba1cDM: 0.144834, hba1cNoDM: 0.127784, hba1cMissing: -0.002259,
			},
			chd30yr: logisticCoeffs{
				intercept: -2.802642,
				age: 0.485319, ageSq: -0.092965,
				nonHDL: 0.173315, hdl: -0.192945,
				sbpLower: 0.009223, sbpUpper: 0.298244,
				diabetes: 0.421029, smoking: 0.325424,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.118347, egfrUpper: 0.046281,
				bpTx: 0.156533, statin: 0.033512,
				bpTxSBP: -0.086614, statinNonHDL: 0.046850,
				ageSBP: -0.081980, ageSmoking: -0.153584, ageDiabetes: -0.226493,
				ageBMI: 0, ageEGFR: -0.112274,
				ageNonHDL: -0.056455, ageHDL: 0.048103,
				hba1cDM: 0.134395, hba1cNoDM: 0.127046, hba1cMissing: 0.037491,
			},
			stroke30yr: logisticCoeffs{
				intercept: -2.618396,
				age: 0.429668, ageSq: -0.088312,
				nonHDL: 0.048732, hdl: -0.100141,
				sbpLower: -0.019995, sbpUpper: 0.319838,
				diabetes: 0.327096, smoking: 0.203954,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.069905, egfrUpper: 0.062555,
				bpTx: 0.208098, statin: -0.014201,
				bpTxSBP: -0.002954, statinNonHDL: 0.057452,
				ageSBP: -0.114502, ageSmoking: -0.170899, ageDiabetes: -0.229892,
				ageBMI: 0, ageEGFR: -0.130743,
				ageNonHDL: -0.031538, ageHDL: 0.016645,
				hba1cDM: 0.045762, hba1cNoDM: 0.105647, hba1cMissing: -0.017081,
			},
		}
	case PREVENTModelUACR:
		return outcomeCoeffSet{
			totalCVD10yr: logisticCoeffs{
				intercept: -3.738341,
				age: 0.796925, ageSq: 0,
				nonHDL: 0.025663, hdl: -0.158811,
				sbpLower: -0.225570, sbpUpper: 0.339665,
				diabetes: 0.804751, smoking: 0.528534,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.480351, egfrUpper: 0.043447,
				bpTx: 0.298521, statin: -0.149779,
				bpTxSBP: -0.074289, statinNonHDL: 0.106756,
				ageSBP: -0.090717, ageSmoking: -0.083056, ageDiabetes: -0.270512,
				ageBMI: 0, ageEGFR: -0.138925,
				ageNonHDL: -0.077813, ageHDL: 0.030677,
				logUACR: 0.179304, uacrMissing: 0.013207,
			},
			ascvd10yr: logisticCoeffs{
				intercept: -4.174614,
				age: 0.720200, ageSq: 0,
				nonHDL: 0.113577, hdl: -0.149351,
				sbpLower: -0.072668, sbpUpper: 0.343626,
				diabetes: 0.777309, smoking: 0.474666,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.382465, egfrUpper: 0.039418,
				bpTx: 0.212518, statin: -0.060305,
				bpTxSBP: -0.046605, statinNonHDL: 0.073312,
				ageSBP: -0.099989, ageSmoking: -0.082694, ageDiabetes: -0.241176,
				ageBMI: 0, ageEGFR: -0.144474,
				ageNonHDL: -0.053426, ageHDL: 0.032569,
				logUACR: 0.150122, uacrMissing: 0.005026,
			},
			hf10yr: logisticCoeffs{
				intercept: -4.841506,
				age: 0.914597, ageSq: 0,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.444135, sbpUpper: 0.326032,
				diabetes: 0.961136, smoking: 0.575579,
				bmiLower: 0.000883, bmiUpper: 0.298896,
				egfrLower: 0.591529, egfrUpper: 0.055682,
				bpTx: 0.331410, statin: 0,
				bpTxSBP: -0.107860, statinNonHDL: 0,
				ageSBP: -0.087523, ageSmoking: -0.122025, ageDiabetes: -0.356859,
				ageBMI: -0.005364, ageEGFR: -0.161039,
				ageNonHDL: 0, ageHDL: 0,
				logUACR: 0.219728, uacrMissing: 0.032667,
			},
			chd10yr: logisticCoeffs{
				intercept: -4.989932,
				age: 0.761493, ageSq: 0,
				nonHDL: 0.176575, hdl: -0.199651,
				sbpLower: -0.076077, sbpUpper: 0.337430,
				diabetes: 0.851544, smoking: 0.532785,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.406759, egfrUpper: 0.032597,
				bpTx: 0.187926, statin: -0.036242,
				bpTxSBP: -0.097686, statinNonHDL: 0.063161,
				ageSBP: -0.085682, ageSmoking: -0.082561, ageDiabetes: -0.255988,
				ageBMI: 0, ageEGFR: -0.135737,
				ageNonHDL: -0.064455, ageHDL: 0.048810,
				logUACR: 0.158833, uacrMissing: 0.032150,
			},
			stroke10yr: logisticCoeffs{
				intercept: -4.722218,
				age: 0.693084, ageSq: 0,
				nonHDL: 0.049782, hdl: -0.103624,
				sbpLower: -0.101898, sbpUpper: 0.351388,
				diabetes: 0.737190, smoking: 0.412470,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.357442, egfrUpper: 0.050545,
				bpTx: 0.233958, statin: -0.082831,
				bpTxSBP: -0.013437, statinNonHDL: 0.073392,
				ageSBP: -0.115719, ageSmoking: -0.101434, ageDiabetes: -0.246956,
				ageBMI: 0, ageEGFR: -0.155055,
				ageNonHDL: -0.037860, ageHDL: 0.016987,
				logUACR: 0.140267, uacrMissing: -0.028275,
			},
			totalCVD30yr: logisticCoeffs{
				intercept: -1.583738,
				age: 0.549177, ageSq: -0.093731,
				nonHDL: 0.035985, hdl: -0.164297,
				sbpLower: -0.148340, sbpUpper: 0.313353,
				diabetes: 0.625377, smoking: 0.314717,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.109466, egfrUpper: 0.055071,
				bpTx: 0.278243, statin: -0.078624,
				bpTxSBP: -0.062895, statinNonHDL: 0.093204,
				ageSBP: -0.095145, ageSmoking: -0.163639, ageDiabetes: -0.316823,
				ageBMI: 0, ageEGFR: -0.126548,
				ageNonHDL: -0.071069, ageHDL: 0.030636,
				logUACR: 0.114225, uacrMissing: -0.005586,
			},
			ascvd30yr: logisticCoeffs{
				intercept: -2.178888,
				age: 0.462967, ageSq: -0.090278,
				nonHDL: 0.121521, hdl: -0.152207,
				sbpLower: 0.009268, sbpUpper: 0.311361,
				diabetes: 0.581256, smoking: 0.263167,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.039173, egfrUpper: 0.049296,
				bpTx: 0.178618, statin: 0.013106,
				bpTxSBP: -0.032514, statinNonHDL: 0.061709,
				ageSBP: -0.100319, ageSmoking: -0.154730, ageDiabetes: -0.268457,
				ageBMI: 0, ageEGFR: -0.113070,
				ageNonHDL: -0.048919, ageHDL: 0.032108,
				logUACR: 0.090347, uacrMissing: -0.014582,
			},
			hf30yr: logisticCoeffs{
				intercept: -2.538952,
				age: 0.631951, ageSq: -0.100928,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.378717, sbpUpper: 0.286339,
				diabetes: 0.763122, smoking: 0.335584,
				bmiLower: 0.067708, bmiUpper: 0.251724,
				egfrLower: 0.194007, egfrUpper: 0.066401,
				bpTx: 0.317144, statin: 0,
				bpTxSBP: -0.097066, statinNonHDL: 0,
				ageSBP: -0.089624, ageSmoking: -0.204204, ageDiabetes: -0.400743,
				ageBMI: -0.005470, ageEGFR: -0.136020,
				ageNonHDL: 0, ageHDL: 0,
				logUACR: 0.148603, uacrMissing: 0.011608,
			},
			chd30yr: logisticCoeffs{
				intercept: -2.958872,
				age: 0.489834, ageSq: -0.092687,
				nonHDL: 0.183351, hdl: -0.201611,
				sbpLower: 0.009111, sbpUpper: 0.298186,
				diabetes: 0.637817, smoking: 0.316201,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.056102, egfrUpper: 0.041201,
				bpTx: 0.148227, statin: 0.037480,
				bpTxSBP: -0.085779, statinNonHDL: 0.050937,
				ageSBP: -0.082956, ageSmoking: -0.153567, ageDiabetes: -0.275971,
				ageBMI: 0, ageEGFR: -0.095718,
				ageNonHDL: -0.059969, ageHDL: 0.048156,
				logUACR: 0.097952, uacrMissing: 0.012861,
			},
			stroke30yr: logisticCoeffs{
				intercept: -2.780053,
				age: 0.435484, ageSq: -0.088154,
				nonHDL: 0.055000, hdl: -0.104846,
				sbpLower: -0.020703, sbpUpper: 0.316529,
				diabetes: 0.530582, smoking: 0.201964,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.021882, egfrUpper: 0.059055,
				bpTx: 0.198286, statin: -0.013321,
				bpTxSBP: -0.001796, statinNonHDL: 0.061445,
				ageSBP: -0.113934, ageSmoking: -0.170055, ageDiabetes: -0.266448,
				ageBMI: 0, ageEGFR: -0.117711,
				ageNonHDL: -0.033190, ageHDL: 0.016286,
				logUACR: 0.080619, uacrMissing: -0.049015,
			},
		}
	case PREVENTModelFull:
		return outcomeCoeffSet{
			totalCVD10yr: logisticCoeffs{
				intercept: -3.860385,
				age: 0.771679, ageSq: 0,
				nonHDL: 0.006211, hdl: -0.154776,
				sbpLower: -0.193312, sbpUpper: 0.307122,
				diabetes: 0.496753, smoking: 0.466605,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.478070, egfrUpper: 0.052908,
				bpTx: 0.303489, statin: -0.155652,
				bpTxSBP: -0.066703, statinNonHDL: 0.106182,
				ageSBP: -0.087519, ageSmoking: -0.067613, ageDiabetes: -0.226710,
				ageBMI: 0, ageEGFR: -0.149323,
				ageNonHDL: -0.074227, ageHDL: 0.028824,
				hba1cDM: 0.129851, hba1cNoDM: 0.141256, hba1cMissing: -0.003166,
				logUACR: 0.164592, uacrMissing: 0.019841,
				sdiMissing: 0.180451,
			},
			ascvd10yr: logisticCoeffs{
				intercept: -4.291503,
				age: 0.702307, ageSq: 0,
				nonHDL: 0.089876, hdl: -0.140732,
				sbpLower: -0.025665, sbpUpper: 0.314511,
				diabetes: 0.479922, smoking: 0.406205,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.384774, egfrUpper: 0.049517,
				bpTx: 0.213386, statin: -0.067855,
				bpTxSBP: -0.045142, statinNonHDL: 0.078819,
				ageSBP: -0.096184, ageSmoking: -0.058647, ageDiabetes: -0.200147,
				ageBMI: 0, ageEGFR: -0.153779,
				ageNonHDL: -0.053599, ageHDL: 0.029176,
				hba1cDM: 0.123192, hba1cNoDM: 0.141057, hba1cMissing: 0.005866,
				logUACR: 0.137182, uacrMissing: 0.006161,
				sdiMissing: 0.158891,
			},
			hf10yr: logisticCoeffs{
				intercept: -4.896524,
				age: 0.884209, ageSq: 0,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.421474, sbpUpper: 0.300292,
				diabetes: 0.617036, smoking: 0.538027,
				bmiLower: -0.019134, bmiUpper: 0.276430,
				egfrLower: 0.597585, egfrUpper: 0.065420,
				bpTx: 0.331361, statin: 0,
				bpTxSBP: -0.100230, statinNonHDL: 0,
				ageSBP: -0.084536, ageSmoking: -0.111135, ageDiabetes: -0.298906,
				ageBMI: 0.000810, ageEGFR: -0.166663,
				ageNonHDL: 0, ageHDL: 0,
				hba1cDM: 0.176668, hba1cNoDM: 0.161491, hba1cMissing: -0.001058,
				logUACR: 0.194814, uacrMissing: 0.039537,
				sdiMissing: 0.181914,
			},
			chd10yr: logisticCoeffs{
				intercept: -5.131505,
				age: 0.754421, ageSq: 0,
				nonHDL: 0.140359, hdl: -0.195470,
				sbpLower: -0.002181, sbpUpper: 0.312775,
				diabetes: 0.523075, smoking: 0.459423,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.417324, egfrUpper: 0.038517,
				bpTx: 0.198164, statin: -0.055740,
				bpTxSBP: -0.094047, statinNonHDL: 0.084489,
				ageSBP: -0.080196, ageSmoking: -0.048474, ageDiabetes: -0.200252,
				ageBMI: 0, ageEGFR: -0.145534,
				ageNonHDL: -0.063011, ageHDL: 0.047345,
				hba1cDM: 0.165663, hba1cNoDM: 0.158638, hba1cMissing: 0.026670,
				logUACR: 0.140628, uacrMissing: 0.032372,
				sdiMissing: 0.160251,
			},
			stroke10yr: logisticCoeffs{
				intercept: -4.800802,
				age: 0.684839, ageSq: 0,
				nonHDL: 0.032725, hdl: -0.095761,
				sbpLower: -0.074740, sbpUpper: 0.323786,
				diabetes: 0.444147, smoking: 0.363695,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.362953, egfrUpper: 0.060315,
				bpTx: 0.225342, statin: -0.088563,
				bpTxSBP: -0.016159, statinNonHDL: 0.072707,
				ageSBP: -0.112146, ageSmoking: -0.070340, ageDiabetes: -0.208971,
				ageBMI: 0, ageEGFR: -0.160500,
				ageNonHDL: -0.038902, ageHDL: 0.014174,
				hba1cDM: 0.089734, hba1cNoDM: 0.135479, hba1cMissing: -0.015845,
				logUACR: 0.128536, uacrMissing: -0.024353,
				sdiMissing: 0.140274,
			},
			totalCVD30yr: logisticCoeffs{
				intercept: -1.748475,
				age: 0.507375, ageSq: -0.098175,
				nonHDL: 0.016230, hdl: -0.161715,
				sbpLower: -0.111124, sbpUpper: 0.282946,
				diabetes: 0.400407, smoking: 0.291870,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.101710, egfrUpper: 0.062264,
				bpTx: 0.287242, statin: -0.076814,
				bpTxSBP: -0.055728, statinNonHDL: 0.091759,
				ageSBP: -0.090775, ageSmoking: -0.137322, ageDiabetes: -0.270212,
				ageBMI: 0, ageEGFR: -0.125586,
				ageNonHDL: -0.067913, ageHDL: 0.029076,
				hba1cDM: 0.092528, hba1cNoDM: 0.097560, hba1cMissing: 0.010171,
				logUACR: 0.102806, uacrMissing: -0.000618,
				sdiMissing: 0.156712,
			},
			ascvd30yr: logisticCoeffs{
				intercept: -2.314066,
				age: 0.438674, ageSq: -0.092196,
				nonHDL: 0.097773, hdl: -0.145352,
				sbpLower: 0.059092, sbpUpper: 0.286286,
				diabetes: 0.366914, smoking: 0.235469,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.035434, egfrUpper: 0.057309,
				bpTx: 0.184008, statin: 0.011750,
				bpTxSBP: -0.033195, statinNonHDL: 0.066431,
				ageSBP: -0.096471, ageSmoking: -0.120405, ageDiabetes: -0.227965,
				ageBMI: 0, ageEGFR: -0.115764,
				ageNonHDL: -0.049283, ageHDL: 0.028889,
				hba1cDM: 0.079471, hba1cNoDM: 0.100262, hba1cMissing: 0.017301,
				logUACR: 0.081074, uacrMissing: -0.014778,
				sdiMissing: 0.130896,
			},
			hf30yr: logisticCoeffs{
				intercept: -2.642208,
				age: 0.592751, ageSq: -0.102875,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.359378, sbpUpper: 0.262856,
				diabetes: 0.511347, smoking: 0.347344,
				bmiLower: 0.056466, bmiUpper: 0.236386,
				egfrLower: 0.197130, egfrUpper: 0.073523,
				bpTx: 0.321939, statin: 0,
				bpTxSBP: -0.088032, statinNonHDL: 0,
				ageSBP: -0.086313, ageSmoking: -0.181405, ageDiabetes: -0.342536,
				ageBMI: 0.003129, ageEGFR: -0.135699,
				ageNonHDL: 0, ageHDL: 0,
				hba1cDM: 0.137834, hba1cNoDM: 0.113883, hba1cMissing: 0.013898,
				logUACR: 0.127331, uacrMissing: 0.016701,
				sdiMissing: 0.148580,
			},
			chd30yr: logisticCoeffs{
				intercept: -3.099189,
				age: 0.474880, ageSq: -0.095073,
				nonHDL: 0.147248, hdl: -0.199352,
				sbpLower: 0.085684, sbpUpper: 0.278139,
				diabetes: 0.393980, smoking: 0.283216,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.058593, egfrUpper: 0.045266,
				bpTx: 0.163443, statin: 0.024292,
				bpTxSBP: -0.083630, statinNonHDL: 0.071497,
				ageSBP: -0.077771, ageSmoking: -0.109245, ageDiabetes: -0.221671,
				ageBMI: 0, ageEGFR: -0.099214,
				ageNonHDL: -0.058662, ageHDL: 0.046904,
				hba1cDM: 0.120011, hba1cNoDM: 0.116242, hba1cMissing: 0.037834,
				logUACR: 0.082756, uacrMissing: 0.011610,
				sdiMissing: 0.128516,
			},
			stroke30yr: logisticCoeffs{
				intercept: -2.858449,
				age: 0.420389, ageSq: -0.090390,
				nonHDL: 0.038309, hdl: -0.098962,
				sbpLower: 0.008802, sbpUpper: 0.294008,
				diabetes: 0.323036, smoking: 0.192530,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.020674, egfrUpper: 0.067264,
				bpTx: 0.193590, statin: -0.011897,
				bpTxSBP: -0.005677, statinNonHDL: 0.059887,
				ageSBP: -0.110299, ageSmoking: -0.129854, ageDiabetes: -0.230591,
				ageBMI: 0, ageEGFR: -0.117207,
				ageNonHDL: -0.034387, ageHDL: 0.013539,
				hba1cDM: 0.045113, hba1cNoDM: 0.092487, hba1cMissing: -0.006835,
				logUACR: 0.072681, uacrMissing: -0.046343,
				sdiMissing: 0.109615,
			},
		}
	default:
		return femaleLogisticCoeffs(PREVENTModelBase)
	}
}

func maleLogisticCoeffs(model PREVENTModelVariant) outcomeCoeffSet {
	switch model {
	case PREVENTModelBase:
		return outcomeCoeffSet{
			totalCVD10yr: logisticCoeffs{
				intercept: -3.031168,
				age: 0.768853, ageSq: 0,
				nonHDL: 0.073617, hdl: -0.095443,
				sbpLower: -0.434735, sbpUpper: 0.336266,
				diabetes: 0.769286, smoking: 0.438687,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.537898, egfrUpper: 0.016483,
				bpTx: 0.288879, statin: -0.133735,
				bpTxSBP: -0.047592, statinNonHDL: 0.150273,
				ageSBP: -0.104948, ageSmoking: -0.089507, ageDiabetes: -0.225195,
				ageBMI: 0, ageEGFR: -0.154370,
				ageNonHDL: -0.051787, ageHDL: 0.019117,
			},
			ascvd10yr: logisticCoeffs{
				intercept: -3.500655,
				age: 0.709985, ageSq: 0,
				nonHDL: 0.165866, hdl: -0.114429,
				sbpLower: -0.283721, sbpUpper: 0.323998,
				diabetes: 0.718960, smoking: 0.395697,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.369007, egfrUpper: 0.020362,
				bpTx: 0.203652, statin: -0.086558,
				bpTxSBP: -0.032292, statinNonHDL: 0.114563,
				ageSBP: -0.092702, ageSmoking: -0.097053, ageDiabetes: -0.201852,
				ageBMI: 0, ageEGFR: -0.121708,
				ageNonHDL: -0.030000, ageHDL: 0.023275,
			},
			hf10yr: logisticCoeffs{
				intercept: -3.946391,
				age: 0.897264, ageSq: 0,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.681147, sbpUpper: 0.363446,
				diabetes: 0.923776, smoking: 0.502374,
				bmiLower: -0.048584, bmiUpper: 0.372693,
				egfrLower: 0.692692, egfrUpper: 0.025183,
				bpTx: 0.298092, statin: 0,
				bpTxSBP: -0.049773, statinNonHDL: 0,
				ageSBP: -0.128920, ageSmoking: -0.140169, ageDiabetes: -0.304092,
				ageBMI: 0.006813, ageEGFR: -0.179778,
				ageNonHDL: 0, ageHDL: 0,
			},
			chd10yr: logisticCoeffs{
				intercept: -4.156753,
				age: 0.742328, ageSq: 0,
				nonHDL: 0.257211, hdl: -0.182037,
				sbpLower: -0.317451, sbpUpper: 0.312778,
				diabetes: 0.748525, smoking: 0.391205,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.376487, egfrUpper: 0.019369,
				bpTx: 0.158820, statin: -0.049455,
				bpTxSBP: -0.057785, statinNonHDL: 0.080977,
				ageSBP: -0.085040, ageSmoking: -0.120640, ageDiabetes: -0.210755,
				ageBMI: 0, ageEGFR: -0.077950,
				ageNonHDL: -0.051787, ageHDL: 0.048903,
			},
			stroke10yr: logisticCoeffs{
				intercept: -4.208810,
				age: 0.722513, ageSq: 0,
				nonHDL: 0.026335, hdl: -0.024896,
				sbpLower: -0.268104, sbpUpper: 0.347463,
				diabetes: 0.684699, smoking: 0.387484,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.387783, egfrUpper: 0.020196,
				bpTx: 0.232963, statin: -0.117893,
				bpTxSBP: 0.012093, statinNonHDL: 0.155739,
				ageSBP: -0.115539, ageSmoking: -0.082413, ageDiabetes: -0.212374,
				ageBMI: 0, ageEGFR: -0.180789,
				ageNonHDL: 0.014193, ageHDL: -0.011175,
			},
			totalCVD30yr: logisticCoeffs{
				intercept: -1.148204,
				age: 0.462731, ageSq: -0.098428,
				nonHDL: 0.083609, hdl: -0.102982,
				sbpLower: -0.214035, sbpUpper: 0.290432,
				diabetes: 0.533128, smoking: 0.214191,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.115556, egfrUpper: 0.060378,
				bpTx: 0.232714, statin: -0.027211,
				bpTxSBP: -0.038449, statinNonHDL: 0.134192,
				ageSBP: -0.110144, ageSmoking: -0.156641, ageDiabetes: -0.258594,
				ageBMI: 0, ageEGFR: -0.116678,
				ageNonHDL: -0.051176, ageHDL: 0.016587,
			},
			ascvd30yr: logisticCoeffs{
				intercept: -1.736444,
				age: 0.399410, ageSq: -0.093748,
				nonHDL: 0.174464, hdl: -0.120203,
				sbpLower: -0.066512, sbpUpper: 0.275304,
				diabetes: 0.479026, smoking: 0.178263,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.021879, egfrUpper: 0.060255,
				bpTx: 0.142118, statin: 0.013600,
				bpTxSBP: -0.021826, statinNonHDL: 0.101315,
				ageSBP: -0.092093, ageSmoking: -0.154881, ageDiabetes: -0.215995,
				ageBMI: 0, ageEGFR: -0.071255,
				ageNonHDL: -0.031262, ageHDL: 0.020673,
			},
			hf30yr: logisticCoeffs{
				intercept: -1.957510,
				age: 0.568154, ageSq: -0.104839,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.476156, sbpUpper: 0.303240,
				diabetes: 0.684034, smoking: 0.265627,
				bmiLower: 0.083311, bmiUpper: 0.269990,
				egfrLower: 0.254180, egfrUpper: 0.063892,
				bpTx: 0.258363, statin: 0,
				bpTxSBP: -0.039194, statinNonHDL: 0,
				ageSBP: -0.126912, ageSmoking: -0.204302, ageDiabetes: -0.327357,
				ageBMI: -0.018283, ageEGFR: -0.134262,
				ageNonHDL: 0, ageHDL: 0,
			},
			chd30yr: logisticCoeffs{
				intercept: -2.376762,
				age: 0.417121, ageSq: -0.094999,
				nonHDL: 0.265191, hdl: -0.187945,
				sbpLower: -0.097175, sbpUpper: 0.258931,
				diabetes: 0.495646, smoking: 0.172884,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.009196, egfrUpper: 0.057815,
				bpTx: 0.093920, statin: 0.050892,
				bpTxSBP: -0.048602, statinNonHDL: 0.066948,
				ageSBP: -0.081223, ageSmoking: -0.174920, ageDiabetes: -0.216315,
				ageBMI: 0, ageEGFR: -0.024147,
				ageNonHDL: -0.053336, ageHDL: 0.046143,
			},
			stroke30yr: logisticCoeffs{
				intercept: -2.458022,
				age: 0.400345, ageSq: -0.093593,
				nonHDL: 0.030942, hdl: -0.028076,
				sbpLower: -0.047704, sbpUpper: 0.292573,
				diabetes: 0.423682, smoking: 0.167524,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.000922, egfrUpper: 0.057522,
				bpTx: 0.168551, statin: -0.020829,
				bpTxSBP: 0.023004, statinNonHDL: 0.141365,
				ageSBP: -0.111847, ageSmoking: -0.133930, ageDiabetes: -0.215295,
				ageBMI: 0, ageEGFR: -0.122508,
				ageNonHDL: 0.014541, ageHDL: -0.014961,
			},
		}
	case PREVENTModelHbA1c:
		return outcomeCoeffSet{
			totalCVD10yr: logisticCoeffs{
				intercept: -3.040901,
				age: 0.769918, ageSq: 0,
				nonHDL: 0.060509, hdl: -0.088853,
				sbpLower: -0.417713, sbpUpper: 0.328866,
				diabetes: 0.475947, smoking: 0.438566,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.533462, egfrUpper: 0.020643,
				bpTx: 0.291752, statin: -0.138331,
				bpTxSBP: -0.048262, statinNonHDL: 0.139380,
				ageSBP: -0.103772, ageSmoking: -0.091584, ageDiabetes: -0.173770,
				ageBMI: 0, ageEGFR: -0.163704,
				ageNonHDL: -0.046350, ageHDL: 0.020593,
				hba1cDM: 0.131590, hba1cNoDM: 0.129519, hba1cMissing: -0.012837,
			},
			ascvd10yr: logisticCoeffs{
				intercept: -3.518350,
				age: 0.706415, ageSq: 0,
				nonHDL: 0.153227, hdl: -0.108217,
				sbpLower: -0.267529, sbpUpper: 0.317381,
				diabetes: 0.432604, smoking: 0.395884,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.366501, egfrUpper: 0.025024,
				bpTx: 0.206116, statin: -0.089999,
				bpTxSBP: -0.033496, statinNonHDL: 0.103417,
				ageSBP: -0.091744, ageSmoking: -0.098089, ageDiabetes: -0.149920,
				ageBMI: 0, ageEGFR: -0.130523,
				ageNonHDL: -0.025541, ageHDL: 0.024754,
				hba1cDM: 0.115716, hba1cNoDM: 0.128830, hba1cMissing: -0.001000,
			},
			hf10yr: logisticCoeffs{
				intercept: -3.961954,
				age: 0.911787, ageSq: 0,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.656807, sbpUpper: 0.352465,
				diabetes: 0.584975, smoking: 0.501401,
				bmiLower: -0.051235, bmiUpper: 0.365294,
				egfrLower: 0.689222, egfrUpper: 0.029238,
				bpTx: 0.303830, statin: 0,
				bpTxSBP: -0.051503, statinNonHDL: 0,
				ageSBP: -0.126234, ageSmoking: -0.139222, ageDiabetes: -0.244951,
				ageBMI: 0.000959, ageEGFR: -0.191711,
				ageNonHDL: 0, ageHDL: 0,
				hba1cDM: 0.165286, hba1cNoDM: 0.150586, hba1cMissing: -0.011344,
			},
			chd10yr: logisticCoeffs{
				intercept: -4.219654,
				age: 0.722364, ageSq: 0,
				nonHDL: 0.244359, hdl: -0.175524,
				sbpLower: -0.302575, sbpUpper: 0.305689,
				diabetes: 0.487484, smoking: 0.394246,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.374801, egfrUpper: 0.025041,
				bpTx: 0.161618, statin: -0.051373,
				bpTxSBP: -0.059131, statinNonHDL: 0.071091,
				ageSBP: -0.083846, ageSmoking: -0.122069, ageDiabetes: -0.163068,
				ageBMI: 0, ageEGFR: -0.085950,
				ageNonHDL: -0.047659, ageHDL: 0.050269,
				hba1cDM: 0.125106, hba1cNoDM: 0.117701, hba1cMissing: 0.028544,
			},
			stroke10yr: logisticCoeffs{
				intercept: -4.179346,
				age: 0.724447, ageSq: 0,
				nonHDL: 0.013429, hdl: -0.020436,
				sbpLower: -0.250696, sbpUpper: 0.340856,
				diabetes: 0.353550, smoking: 0.385490,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.386979, egfrUpper: 0.023766,
				bpTx: 0.236159, statin: -0.124198,
				bpTxSBP: 0.009588, statinNonHDL: 0.142417,
				ageSBP: -0.114358, ageSmoking: -0.082419, ageDiabetes: -0.155981,
				ageBMI: 0, ageEGFR: -0.189090,
				ageNonHDL: 0.018922, ageHDL: -0.009345,
				hba1cDM: 0.107602, hba1cNoDM: 0.140860, hba1cMissing: -0.047094,
			},
			totalCVD30yr: logisticCoeffs{
				intercept: -1.180767,
				age: 0.451987, ageSq: -0.101624,
				nonHDL: 0.070046, hdl: -0.096800,
				sbpLower: -0.192353, sbpUpper: 0.282704,
				diabetes: 0.341715, smoking: 0.210527,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.111329, egfrUpper: 0.064014,
				bpTx: 0.233425, statin: -0.029942,
				bpTxSBP: -0.039320, statinNonHDL: 0.122885,
				ageSBP: -0.108574, ageSmoking: -0.157798, ageDiabetes: -0.220805,
				ageBMI: 0, ageEGFR: -0.117938,
				ageNonHDL: -0.046374, ageHDL: 0.018460,
				hba1cDM: 0.076817, hba1cNoDM: 0.077730, hba1cMissing: 0.009220,
			},
			ascvd30yr: logisticCoeffs{
				intercept: -1.777708,
				age: 0.388327, ageSq: -0.095811,
				nonHDL: 0.161337, hdl: -0.114442,
				sbpLower: -0.047434, sbpUpper: 0.269128,
				diabetes: 0.285977, smoking: 0.175955,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.024290, egfrUpper: 0.064452,
				bpTx: 0.142874, statin: 0.011506,
				bpTxSBP: -0.023330, statinNonHDL: 0.089966,
				ageSBP: -0.090802, ageSmoking: -0.154885, ageDiabetes: -0.177189,
				ageBMI: 0, ageEGFR: -0.073275,
				ageNonHDL: -0.027548, ageHDL: 0.022573,
				hba1cDM: 0.059109, hba1cNoDM: 0.082116, hba1cMissing: 0.017975,
			},
			hf30yr: logisticCoeffs{
				intercept: -1.974999,
				age: 0.570373, ageSq: -0.108454,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.447177, sbpUpper: 0.291015,
				diabetes: 0.450724, smoking: 0.259585,
				bmiLower: 0.085068, bmiUpper: 0.263722,
				egfrLower: 0.245471, egfrUpper: 0.067565,
				bpTx: 0.261199, statin: 0,
				bpTxSBP: -0.040891, statinNonHDL: 0,
				ageSBP: -0.124105, ageSmoking: -0.203231, ageDiabetes: -0.284946,
				ageBMI: -0.023971, ageEGFR: -0.138301,
				ageNonHDL: 0, ageHDL: 0,
				hba1cDM: 0.110118, hba1cNoDM: 0.094920, hba1cMissing: 0.008419,
			},
			chd30yr: logisticCoeffs{
				intercept: -2.484330,
				age: 0.394451, ageSq: -0.095686,
				nonHDL: 0.251937, hdl: -0.181881,
				sbpLower: -0.082920, sbpUpper: 0.253460,
				diabetes: 0.329820, smoking: 0.177338,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.005951, egfrUpper: 0.062562,
				bpTx: 0.096027, statin: 0.048911,
				bpTxSBP: -0.050231, statinNonHDL: 0.057355,
				ageSBP: -0.079667, ageSmoking: -0.174495, ageDiabetes: -0.182284,
				ageBMI: 0, ageEGFR: -0.025946,
				ageNonHDL: -0.049882, ageHDL: 0.047914,
				hba1cDM: 0.069564, hba1cNoDM: 0.070754, hba1cMissing: 0.049160,
			},
			stroke30yr: logisticCoeffs{
				intercept: -2.438154,
				age: 0.397402, ageSq: -0.095342,
				nonHDL: 0.018020, hdl: -0.024253,
				sbpLower: -0.028341, sbpUpper: 0.287064,
				diabetes: 0.189441, smoking: 0.163228,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.002762, egfrUpper: 0.060562,
				bpTx: 0.170037, statin: -0.025615,
				bpTxSBP: 0.020110, statinNonHDL: 0.128168,
				ageSBP: -0.110585, ageSmoking: -0.133394, ageDiabetes: -0.174408,
				ageBMI: 0, ageEGFR: -0.125459,
				ageNonHDL: 0.018552, ageHDL: -0.012734,
				hba1cDM: 0.050054, hba1cNoDM: 0.091773, hba1cMissing: -0.031199,
			},
		}
	case PREVENTModelUACR:
		return outcomeCoeffSet{
			totalCVD10yr: logisticCoeffs{
				intercept: -3.510705,
				age: 0.776865, ageSq: 0,
				nonHDL: 0.065995, hdl: -0.095111,
				sbpLower: -0.420667, sbpUpper: 0.312015,
				diabetes: 0.698521, smoking: 0.431467,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.384136, egfrUpper: 0.009384,
				bpTx: 0.267649, statin: -0.139097,
				bpTxSBP: -0.057931, statinNonHDL: 0.138372,
				ageSBP: -0.102454, ageSmoking: -0.089485, ageDiabetes: -0.223635,
				ageBMI: 0, ageEGFR: -0.132185,
				ageNonHDL: -0.048833, ageHDL: 0.020041,
				logUACR: 0.188797, uacrMissing: 0.091698,
			},
			ascvd10yr: logisticCoeffs{
				intercept: -3.851460,
				age: 0.714172, ageSq: 0,
				nonHDL: 0.160219, hdl: -0.113909,
				sbpLower: -0.271946, sbpUpper: 0.305872,
				diabetes: 0.660063, smoking: 0.388402,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.246632, egfrUpper: 0.015185,
				bpTx: 0.186167, statin: -0.089440,
				bpTxSBP: -0.041188, statinNonHDL: 0.105821,
				ageSBP: -0.091232, ageSmoking: -0.096936, ageDiabetes: -0.200489,
				ageBMI: 0, ageEGFR: -0.102287,
				ageNonHDL: -0.028089, ageHDL: 0.024043,
				logUACR: 0.151007, uacrMissing: 0.055600,
			},
			hf10yr: logisticCoeffs{
				intercept: -4.556907,
				age: 0.911180, ageSq: 0,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.669365, sbpUpper: 0.329008,
				diabetes: 0.837766, smoking: 0.497892,
				bmiLower: -0.042749, bmiUpper: 0.362416,
				egfrLower: 0.507580, egfrUpper: 0.013772,
				bpTx: 0.273996, statin: 0,
				bpTxSBP: -0.064571, statinNonHDL: 0,
				ageSBP: -0.123004, ageSmoking: -0.141032, ageDiabetes: -0.301330,
				ageBMI: 0.002153, ageEGFR: -0.154802,
				ageNonHDL: 0, ageHDL: 0,
				logUACR: 0.230630, uacrMissing: 0.147219,
			},
			chd10yr: logisticCoeffs{
				intercept: -4.505640,
				age: 0.750295, ageSq: 0,
				nonHDL: 0.250938, hdl: -0.181556,
				sbpLower: -0.306272, sbpUpper: 0.295124,
				diabetes: 0.694044, smoking: 0.385377,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.254199, egfrUpper: 0.014138,
				bpTx: 0.143656, statin: -0.051752,
				bpTxSBP: -0.066891, statinNonHDL: 0.072524,
				ageSBP: -0.083481, ageSmoking: -0.119464, ageDiabetes: -0.209361,
				ageBMI: 0, ageEGFR: -0.058564,
				ageNonHDL: -0.050119, ageHDL: 0.049639,
				logUACR: 0.144008, uacrMissing: 0.069125,
			},
			stroke10yr: logisticCoeffs{
				intercept: -4.594023,
				age: 0.731802, ageSq: 0,
				nonHDL: 0.021362, hdl: -0.025297,
				sbpLower: -0.257460, sbpUpper: 0.328418,
				diabetes: 0.616640, smoking: 0.383551,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.267703, egfrUpper: 0.013970,
				bpTx: 0.213571, statin: -0.121991,
				bpTxSBP: 0.003814, statinNonHDL: 0.145425,
				ageSBP: -0.113155, ageSmoking: -0.080403, ageDiabetes: -0.208108,
				ageBMI: 0, ageEGFR: -0.161721,
				ageNonHDL: 0.016520, ageHDL: -0.010184,
				logUACR: 0.162424, uacrMissing: 0.041238,
			},
			totalCVD30yr: logisticCoeffs{
				intercept: -1.398727,
				age: 0.464491, ageSq: -0.099890,
				nonHDL: 0.075761, hdl: -0.103178,
				sbpLower: -0.199071, sbpUpper: 0.271582,
				diabetes: 0.475464, smoking: 0.206967,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.033110, egfrUpper: 0.054047,
				bpTx: 0.218991, statin: -0.033104,
				bpTxSBP: -0.045340, statinNonHDL: 0.121454,
				ageSBP: -0.105932, ageSmoking: -0.156154, ageDiabetes: -0.249286,
				ageBMI: 0, ageEGFR: -0.101243,
				ageNonHDL: -0.048399, ageHDL: 0.017900,
				logUACR: 0.100757, uacrMissing: 0.057246,
			},
			ascvd30yr: logisticCoeffs{
				intercept: -1.873449,
				age: 0.399561, ageSq: -0.094557,
				nonHDL: 0.168669, hdl: -0.120215,
				sbpLower: -0.055556, sbpUpper: 0.263357,
				diabetes: 0.436204, smoking: 0.171623,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.077528, egfrUpper: 0.056124,
				bpTx: 0.131933, statin: 0.010243,
				bpTxSBP: -0.026929, statinNonHDL: 0.092056,
				ageSBP: -0.089335, ageSmoking: -0.154272, ageDiabetes: -0.208147,
				ageBMI: 0, ageEGFR: -0.059725,
				ageNonHDL: -0.029702, ageHDL: 0.021794,
				logUACR: 0.068487, uacrMissing: 0.019396,
			},
			hf30yr: logisticCoeffs{
				intercept: -2.314872,
				age: 0.575024, ageSq: -0.106227,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.463399, sbpUpper: 0.274287,
				diabetes: 0.612208, smoking: 0.261499,
				bmiLower: 0.089546, bmiUpper: 0.263242,
				egfrLower: 0.143047, egfrUpper: 0.053518,
				bpTx: 0.241747, statin: 0,
				bpTxSBP: -0.049857, statinNonHDL: 0,
				ageSBP: -0.119383, ageSmoking: -0.204612, ageDiabetes: -0.316651,
				ageBMI: -0.021688, ageEGFR: -0.116564,
				ageNonHDL: 0, ageHDL: 0,
				logUACR: 0.136645, uacrMissing: 0.107836,
			},
			chd30yr: logisticCoeffs{
				intercept: -2.505263,
				age: 0.420671, ageSq: -0.095916,
				nonHDL: 0.258816, hdl: -0.187972,
				sbpLower: -0.086513, sbpUpper: 0.247711,
				diabetes: 0.458800, smoking: 0.167728,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.065487, egfrUpper: 0.053731,
				bpTx: 0.086180, statin: 0.048272,
				bpTxSBP: -0.053793, statinNonHDL: 0.058057,
				ageSBP: -0.078396, ageSmoking: -0.173351, ageDiabetes: -0.208588,
				ageBMI: 0, ageEGFR: -0.012972,
				ageNonHDL: -0.052005, ageHDL: 0.047197,
				logUACR: 0.060943, uacrMissing: 0.032488,
			},
			stroke30yr: logisticCoeffs{
				intercept: -2.619781,
				age: 0.405896, ageSq: -0.094257,
				nonHDL: 0.025870, hdl: -0.029002,
				sbpLower: -0.037480, sbpUpper: 0.280121,
				diabetes: 0.372918, smoking: 0.164976,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.053820, egfrUpper: 0.052469,
				bpTx: 0.156430, statin: -0.024804,
				bpTxSBP: 0.018991, statinNonHDL: 0.130596,
				ageSBP: -0.108059, ageSmoking: -0.131120, ageDiabetes: -0.204636,
				ageBMI: 0, ageEGFR: -0.112214,
				ageNonHDL: 0.016562, ageHDL: -0.013689,
				logUACR: 0.079376, uacrMissing: 0.002552,
			},
		}
	case PREVENTModelFull:
		return outcomeCoeffSet{
			totalCVD10yr: logisticCoeffs{
				intercept: -3.631387,
				age: 0.784758, ageSq: 0,
				nonHDL: 0.053449, hdl: -0.091128,
				sbpLower: -0.492197, sbpUpper: 0.297241,
				diabetes: 0.452705, smoking: 0.372664,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.388685, egfrUpper: 0.008166,
				bpTx: 0.250805, statin: -0.153848,
				bpTxSBP: -0.047469, statinNonHDL: 0.141538,
				ageSBP: -0.102269, ageSmoking: -0.071587, ageDiabetes: -0.176251,
				ageBMI: 0, ageEGFR: -0.142867,
				ageNonHDL: -0.043645, ageHDL: 0.019955,
				hba1cDM: 0.116570, hba1cNoDM: 0.104830, hba1cMissing: -0.023007,
				logUACR: 0.177285, uacrMissing: 0.109567,
				sdiMissing: 0.144759,
			},
			ascvd10yr: logisticCoeffs{
				intercept: -3.969788,
				age: 0.712874, ageSq: 0,
				nonHDL: 0.146520, hdl: -0.112579,
				sbpLower: -0.338722, sbpUpper: 0.298025,
				diabetes: 0.399583, smoking: 0.337911,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.258260, egfrUpper: 0.014777,
				bpTx: 0.168662, statin: -0.107362,
				bpTxSBP: -0.038104, statinNonHDL: 0.103417,
				ageSBP: -0.089745, ageSmoking: -0.077206, ageDiabetes: -0.149746,
				ageBMI: 0, ageEGFR: -0.119837,
				ageNonHDL: -0.022876, ageHDL: 0.026745,
				hba1cDM: 0.101282, hba1cNoDM: 0.109273, hba1cMissing: -0.011285,
				logUACR: 0.137584, uacrMissing: 0.065294,
				sdiMissing: 0.138849,
			},
			hf10yr: logisticCoeffs{
				intercept: -4.663513,
				age: 0.909570, ageSq: 0,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.676518, sbpUpper: 0.311165,
				diabetes: 0.553505, smoking: 0.432681,
				bmiLower: -0.085429, bmiUpper: 0.355174,
				egfrLower: 0.510224, egfrUpper: 0.015472,
				bpTx: 0.257096, statin: 0,
				bpTxSBP: -0.059118, statinNonHDL: 0,
				ageSBP: -0.121906, ageSmoking: -0.105363, ageDiabetes: -0.243758,
				ageBMI: 0.003791, ageEGFR: -0.166021,
				ageNonHDL: 0, ageHDL: 0,
				hba1cDM: 0.148297, hba1cNoDM: 0.123409, hba1cMissing: -0.023464,
				logUACR: 0.216461, uacrMissing: 0.170281,
				sdiMissing: 0.169463,
			},
			chd10yr: logisticCoeffs{
				intercept: -4.629874,
				age: 0.733585, ageSq: 0,
				nonHDL: 0.240310, hdl: -0.182224,
				sbpLower: -0.414886, sbpUpper: 0.288008,
				diabetes: 0.493728, smoking: 0.332608,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.258684, egfrUpper: 0.012498,
				bpTx: 0.113276, statin: -0.082354,
				bpTxSBP: -0.071879, statinNonHDL: 0.076592,
				ageSBP: -0.081329, ageSmoking: -0.092797, ageDiabetes: -0.167929,
				ageBMI: 0, ageEGFR: -0.067706,
				ageNonHDL: -0.048901, ageHDL: 0.053146,
				hba1cDM: 0.116195, hba1cNoDM: 0.099111, hba1cMissing: 0.021368,
				logUACR: 0.135418, uacrMissing: 0.075949,
				sdiMissing: 0.094206,
			},
			stroke10yr: logisticCoeffs{
				intercept: -4.683048,
				age: 0.718276, ageSq: 0,
				nonHDL: 0.000886, hdl: -0.022068,
				sbpLower: -0.271976, sbpUpper: 0.319957,
				diabetes: 0.272829, smoking: 0.342676,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.290713, egfrUpper: 0.018655,
				bpTx: 0.240036, statin: -0.121894,
				bpTxSBP: 0.004259, statinNonHDL: 0.141273,
				ageSBP: -0.112190, ageSmoking: -0.058033, ageDiabetes: -0.144699,
				ageBMI: 0, ageEGFR: -0.182091,
				ageNonHDL: 0.027588, ageHDL: -0.007999,
				hba1cDM: 0.086831, hba1cNoDM: 0.118754, hba1cMissing: -0.060318,
				logUACR: 0.140494, uacrMissing: 0.053690,
				sdiMissing: 0.189271,
			},
			totalCVD30yr: logisticCoeffs{
				intercept: -1.504558,
				age: 0.442759, ageSq: -0.106411,
				nonHDL: 0.062938, hdl: -0.101543,
				sbpLower: -0.254233, sbpUpper: 0.254968,
				diabetes: 0.333835, smoking: 0.187383,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: 0.024610, egfrUpper: 0.055201,
				bpTx: 0.197973, statin: -0.040771,
				bpTxSBP: -0.036552, statinNonHDL: 0.123282,
				ageSBP: -0.104666, ageSmoking: -0.127791, ageDiabetes: -0.211611,
				ageBMI: 0, ageEGFR: -0.095592,
				ageNonHDL: -0.044133, ageHDL: 0.017787,
				hba1cDM: 0.067620, hba1cNoDM: 0.063409, hba1cMissing: 0.003878,
				logUACR: 0.089460, uacrMissing: 0.071012,
				sdiMissing: 0.089241,
			},
			ascvd30yr: logisticCoeffs{
				intercept: -1.985368,
				age: 0.374357, ageSq: -0.099550,
				nonHDL: 0.154481, hdl: -0.121530,
				sbpLower: -0.108397, sbpUpper: 0.255518,
				diabetes: 0.269700, smoking: 0.162843,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.077507, egfrUpper: 0.058341,
				bpTx: 0.112032, statin: -0.002506,
				bpTxSBP: -0.025612, statinNonHDL: 0.088675,
				ageSBP: -0.086915, ageSmoking: -0.124471, ageDiabetes: -0.165745,
				ageBMI: 0, ageEGFR: -0.062455,
				ageNonHDL: -0.025451, ageHDL: 0.024464,
				hba1cDM: 0.050142, hba1cNoDM: 0.072290, hba1cMissing: 0.011494,
				logUACR: 0.056017, uacrMissing: 0.025224,
				sdiMissing: 0.084570,
			},
			hf30yr: logisticCoeffs{
				intercept: -2.425439,
				age: 0.547883, ageSq: -0.111193,
				nonHDL: 0, hdl: 0,
				sbpLower: -0.454735, sbpUpper: 0.252760,
				diabetes: 0.438538, smoking: 0.239795,
				bmiLower: 0.064093, bmiUpper: 0.264308,
				egfrLower: 0.135459, egfrUpper: 0.057069,
				bpTx: 0.220666, statin: 0,
				bpTxSBP: -0.043677, statinNonHDL: 0,
				ageSBP: -0.116838, ageSmoking: -0.157369, ageDiabetes: -0.273006,
				ageBMI: -0.017500, ageEGFR: -0.112868,
				ageNonHDL: 0, ageHDL: 0,
				hba1cDM: 0.098506, hba1cNoDM: 0.080484, hba1cMissing: 0.002281,
				logUACR: 0.123349, uacrMissing: 0.127480,
				sdiMissing: 0.107678,
			},
			chd30yr: logisticCoeffs{
				intercept: -2.643083,
				age: 0.385830, ageSq: -0.099413,
				nonHDL: 0.247964, hdl: -0.191333,
				sbpLower: -0.183709, sbpUpper: 0.241755,
				diabetes: 0.359997, smoking: 0.158105,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.070796, egfrUpper: 0.054403,
				bpTx: 0.053914, statin: 0.021464,
				bpTxSBP: -0.060352, statinNonHDL: 0.061636,
				ageSBP: -0.075496, ageSmoking: -0.135563, ageDiabetes: -0.177355,
				ageBMI: 0, ageEGFR: -0.007771,
				ageNonHDL: -0.051751, ageHDL: 0.050696,
				hba1cDM: 0.065637, hba1cNoDM: 0.062151, hba1cMissing: 0.045716,
				logUACR: 0.054848, uacrMissing: 0.036299,
				sdiMissing: 0.036964,
			},
			stroke30yr: logisticCoeffs{
				intercept: -2.713952,
				age: 0.376275, ageSq: -0.097472,
				nonHDL: 0.005415, hdl: -0.028769,
				sbpLower: -0.036424, sbpUpper: 0.273376,
				diabetes: 0.128778, smoking: 0.168363,
				bmiLower: 0, bmiUpper: 0,
				egfrLower: -0.038650, egfrUpper: 0.059634,
				bpTx: 0.183380, statin: -0.018885,
				bpTxSBP: 0.018129, statinNonHDL: 0.125821,
				ageSBP: -0.106380, ageSmoking: -0.100203, ageDiabetes: -0.151944,
				ageBMI: 0, ageEGFR: -0.121428,
				ageNonHDL: 0.026783, ageHDL: -0.011618,
				hba1cDM: 0.035332, hba1cNoDM: 0.080409, hba1cMissing: -0.041002,
				logUACR: 0.059955, uacrMissing: 0.012492,
				sdiMissing: 0.134716,
			},
		}
	default:
		return maleLogisticCoeffs(PREVENTModelBase)
	}
}

