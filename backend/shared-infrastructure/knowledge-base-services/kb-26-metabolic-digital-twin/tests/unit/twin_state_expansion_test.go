package unit

import (
	"math"
	"testing"

	"kb-26-metabolic-digital-twin/internal/services"
)

const egfrEpsilon = 0.5 // eGFR tolerance for float comparison

func TestComputeEGFR_CKDEPI2021_Male(t *testing.T) {
	// Male, age 55, creatinine 1.0 mg/dL
	// CKD-EPI 2021: 142 × min(1.0/0.9,1)^(-0.302) × max(1.0/0.9,1)^(-1.200) × 0.9938^55 × 1.0
	result := services.ComputeEGFR_CKDEPI2021(1.0, "M", 55)
	expected := 88.88
	if math.Abs(result-expected) > egfrEpsilon {
		t.Errorf("eGFR male/55/Cr=1.0: got %.2f, want %.2f ±%.1f", result, expected, egfrEpsilon)
	}
}

func TestComputeEGFR_CKDEPI2021_Female(t *testing.T) {
	// Female, age 55, creatinine 0.8 mg/dL
	// CKD-EPI 2021: 142 × min(0.8/0.7,1)^(-0.241) × max(0.8/0.7,1)^(-1.200) × 0.9938^55 × 1.012
	result := services.ComputeEGFR_CKDEPI2021(0.8, "F", 55)
	expected := 86.96
	if math.Abs(result-expected) > egfrEpsilon {
		t.Errorf("eGFR female/55/Cr=0.8: got %.2f, want %.2f ±%.1f", result, expected, egfrEpsilon)
	}
}

func TestComputeEGFR_CKDEPI2021_HighCreatinine(t *testing.T) {
	// Male, age 65, creatinine 2.0 mg/dL (impaired kidney function)
	// CKD-EPI 2021: 142 × min(2.0/0.9,1)^(-0.302) × max(2.0/0.9,1)^(-1.200) × 0.9938^65 × 1.0
	result := services.ComputeEGFR_CKDEPI2021(2.0, "M", 65)
	expected := 36.36
	if math.Abs(result-expected) > egfrEpsilon {
		t.Errorf("eGFR male/65/Cr=2.0: got %.2f, want %.2f ±%.1f", result, expected, egfrEpsilon)
	}
}

func TestComputeEGFR_CKDEPI2021_Young(t *testing.T) {
	// Male, age 30, creatinine 0.9 mg/dL (healthy young)
	// CKD-EPI 2021: 142 × min(0.9/0.9,1)^(-0.302) × max(0.9/0.9,1)^(-1.200) × 0.9938^30 × 1.0
	result := services.ComputeEGFR_CKDEPI2021(0.9, "M", 30)
	expected := 117.83
	if math.Abs(result-expected) > egfrEpsilon {
		t.Errorf("eGFR male/30/Cr=0.9: got %.2f, want %.2f ±%.1f", result, expected, egfrEpsilon)
	}
}

func TestComputeEGFR_CKDEPI2021_SexDifference(t *testing.T) {
	// Same creatinine and age — kappa/alpha differ by sex so results differ
	male := services.ComputeEGFR_CKDEPI2021(1.0, "M", 50)
	female := services.ComputeEGFR_CKDEPI2021(1.0, "F", 50)
	expectedMale := 91.69
	expectedFemale := 68.63
	if math.Abs(male-expectedMale) > egfrEpsilon {
		t.Errorf("eGFR male/50/Cr=1.0: got %.2f, want %.2f ±%.1f", male, expectedMale, egfrEpsilon)
	}
	if math.Abs(female-expectedFemale) > egfrEpsilon {
		t.Errorf("eGFR female/50/Cr=1.0: got %.2f, want %.2f ±%.1f", female, expectedFemale, egfrEpsilon)
	}
	if math.Abs(male-female) < 1 {
		t.Errorf("Expected sex difference: male=%.2f, female=%.2f", male, female)
	}
}

func TestComputeEGFR_CKDEPI2021_ZeroCreatinine(t *testing.T) {
	result := services.ComputeEGFR_CKDEPI2021(0, "M", 55)
	if result != 0 {
		t.Errorf("eGFR with Cr=0 should be 0, got %f", result)
	}
}

func TestComputeEGFR_CKDEPI2021_NegativeCreatinine(t *testing.T) {
	result := services.ComputeEGFR_CKDEPI2021(-1.5, "F", 40)
	if result != 0 {
		t.Errorf("eGFR with Cr=-1.5 should be 0, got %f", result)
	}
}
