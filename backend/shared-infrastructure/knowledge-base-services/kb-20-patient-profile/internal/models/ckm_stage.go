package models

const (
	CKMStage0 = 0 // No CKM risk factors
	CKMStage1 = 1 // Excess adiposity, dyslipidemia, or metabolic syndrome
	CKMStage2 = 2 // Metabolic risk + moderate-high risk CKD or T2DM
	CKMStage3 = 3 // Subclinical CVD or high predicted ASCVD risk
	CKMStage4 = 4 // Clinical CVD event
)

// ComputeCKMStage classifies a patient into AHA CKM Stage 0-4.
// Per DD#6 §6, staging is deterministic — no ML, no LLM.
func ComputeCKMStage(p PatientProfile) int {
	// Stage 4: clinical CVD trumps everything
	if p.HasClinicalCVD {
		return CKMStage4
	}

	// Stage 3: subclinical CVD markers or high ASCVD risk
	if p.ASCVDRisk10y != nil && *p.ASCVDRisk10y >= 20.0 {
		return CKMStage3
	}
	if p.UACR != nil && *p.UACR >= 300.0 {
		return CKMStage3
	}
	if p.EGFR != nil && *p.EGFR < 30.0 {
		return CKMStage3
	}

	// Stage 2: T2DM or moderate-to-high CKD with metabolic risk
	hasT2DM := p.HbA1c != nil && *p.HbA1c >= 6.5
	if p.DiabetesYears != nil && *p.DiabetesYears > 0 {
		hasT2DM = true
	}
	moderateCKD := (p.EGFR != nil && *p.EGFR < 60.0) || (p.UACR != nil && *p.UACR >= 30.0)
	if hasT2DM || moderateCKD {
		return CKMStage2
	}

	// Stage 1: excess adiposity, dyslipidemia, or metabolic syndrome markers
	// Note: BMI is a value type (float64), not a pointer — use > 0 guard for "present"
	excessAdipose := (p.BMI >= 25.0) || (p.WaistToHeightRatio != nil && *p.WaistToHeightRatio >= 0.5)
	dyslipidemia := p.TGHDLRatio != nil && *p.TGHDLRatio >= 3.5
	preDiabetes := p.HbA1c != nil && *p.HbA1c >= 5.7 && *p.HbA1c < 6.5
	if excessAdipose || dyslipidemia || preDiabetes {
		return CKMStage1
	}

	return CKMStage0
}
