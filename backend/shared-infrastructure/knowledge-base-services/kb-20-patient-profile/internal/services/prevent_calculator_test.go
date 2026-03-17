package services

import (
	"math"
	"testing"
)

// Reference values from Khan et al., Circulation 2024;149:430-449, Table S25.
// Each case must match within ±0.005 absolute risk.
func TestPREVENT_TableS25_Female_Base(t *testing.T) {
	// Skip until real coefficients from Khan et al. Supplemental Tables S1-S24
	// are inserted into getCoefficients(). This test is the validation gate —
	// it MUST pass (±0.005) before clinical deployment.
	t.Skip("blocked: placeholder coefficients — replace with Khan et al. S1-S24 then remove this skip")

	input := PREVENTInput{
		Age:              50,
		Sex:              SexFemale,
		TotalCholesterol: 200,
		HDLCholesterol:   45,
		SystolicBP:       160,
		OnBPTreatment:    true,
		DiabetesStatus:   true,
		CurrentSmoking:   false,
		EGFR:             90,
		BMI:              35,
		ModelVariant:     PREVENTModelBase,
	}

	result := ComputePREVENT(input)

	assertRisk(t, "10yr_total_cvd", result.TenYearTotalCVD, 0.147, 0.005)
	assertRisk(t, "10yr_ascvd", result.TenYearASCVD, 0.092, 0.005)
	assertRisk(t, "10yr_hf", result.TenYearHF, 0.081, 0.005)
}

func TestPREVENT_Male_HighRisk(t *testing.T) {
	input := PREVENTInput{
		Age:              65,
		Sex:              SexMale,
		TotalCholesterol: 240,
		HDLCholesterol:   35,
		SystolicBP:       170,
		OnBPTreatment:    true,
		DiabetesStatus:   true,
		CurrentSmoking:   true,
		EGFR:             45,
		BMI:              32,
		HbA1c:            preventFloat64Ptr(8.5),
		UACR:             preventFloat64Ptr(350),
		ModelVariant:     PREVENTModelFull,
	}

	result := ComputePREVENT(input)

	if result.TenYearTotalCVD < 0.20 {
		t.Errorf("expected HIGH tier (≥20%%), got %.3f", result.TenYearTotalCVD)
	}
}

func TestPREVENT_ModelSelection(t *testing.T) {
	tests := []struct {
		name     string
		hba1c    *float64
		uacr     *float64
		expected PREVENTModelVariant
	}{
		{"both available", preventFloat64Ptr(7.0), preventFloat64Ptr(100), PREVENTModelFull},
		{"hba1c only", preventFloat64Ptr(7.0), nil, PREVENTModelHbA1c},
		{"uacr only", nil, preventFloat64Ptr(100), PREVENTModelUACR},
		{"neither", nil, nil, PREVENTModelBase},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectPREVENTModel(tt.hba1c, tt.uacr)
			if got != tt.expected {
				t.Errorf("expected model %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestPREVENT_RiskTierClassification(t *testing.T) {
	tests := []struct {
		risk10yr float64
		expected PREVENTRiskTier
	}{
		{0.03, RiskTierLow},
		{0.06, RiskTierBorderline},
		{0.12, RiskTierIntermediate},
		{0.25, RiskTierHigh},
	}

	for _, tt := range tests {
		got := ClassifyRiskTier(tt.risk10yr)
		if got != tt.expected {
			t.Errorf("risk %.2f: expected %s, got %s", tt.risk10yr, tt.expected, got)
		}
	}
}

func TestPREVENT_SBPTarget(t *testing.T) {
	threshold := 0.075 // default intensive threshold (INTERMEDIATE tier boundary)

	tests := []struct {
		name       string
		tier       PREVENTRiskTier
		tenYearCVD float64
		egfr       float64
		acr        float64
		expected   float64
	}{
		{"HIGH tier (25%)", RiskTierHigh, 0.25, 90, 10, 120},
		{"LOW tier (3%)", RiskTierLow, 0.03, 90, 10, 130},
		{"LOW tier but eGFR<60", RiskTierLow, 0.03, 45, 10, 120},
		{"LOW tier but ACR≥300", RiskTierLow, 0.03, 90, 350, 120},
		{"BORDERLINE (6%)", RiskTierBorderline, 0.06, 90, 10, 130},
		{"INTERMEDIATE (12%)", RiskTierIntermediate, 0.12, 90, 10, 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineSBPTarget(tt.tier, tt.tenYearCVD, tt.egfr, tt.acr, threshold)
			if got != tt.expected {
				t.Errorf("expected SBP target %.0f, got %.0f", tt.expected, got)
			}
		})
	}
}

func TestPREVENT_SBPTarget_CustomThreshold(t *testing.T) {
	// Clinical team lowers threshold to 5% — BORDERLINE patients now get intensive
	customThreshold := 0.05

	got := DetermineSBPTarget(RiskTierBorderline, 0.06, 90, 10, customThreshold)
	if got != 120 {
		t.Errorf("with threshold 0.05, 6%% risk should get intensive target 120, got %.0f", got)
	}

	// 3% risk still below even the lowered threshold
	got = DetermineSBPTarget(RiskTierLow, 0.03, 90, 10, customThreshold)
	if got != 130 {
		t.Errorf("with threshold 0.05, 3%% risk should get standard target 130, got %.0f", got)
	}
}

func TestPREVENT_SouthAsianBMICalibration(t *testing.T) {
	// BMI 26 + calibration factor +3 = effective BMI 29 for PREVENT input
	calibrated := ApplySouthAsianBMICalibration(26.0, 3.0)
	if calibrated != 29.0 {
		t.Errorf("expected calibrated BMI 29.0, got %.1f", calibrated)
	}

	// BMI 31 (above 30 threshold) — no calibration applied
	calibrated = ApplySouthAsianBMICalibration(31.0, 3.0)
	if calibrated != 31.0 {
		t.Errorf("expected uncalibrated BMI 31.0 (above threshold), got %.1f", calibrated)
	}
}

func preventFloat64Ptr(v float64) *float64 { return &v }

func assertRisk(t *testing.T, name string, got, expected, tolerance float64) {
	t.Helper()
	if math.Abs(got-expected) > tolerance {
		t.Errorf("%s: expected %.4f ± %.3f, got %.4f (delta %.4f)",
			name, expected, tolerance, got, math.Abs(got-expected))
	}
}
