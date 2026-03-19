package services

import (
	"math"
	"testing"
)

func TestNormalizeZScore_PopulationMean(t *testing.T) {
	// Value at population mean → z ≈ 0
	z := NormalizeZScore(95, 95, 15)
	if math.Abs(z) > 0.001 {
		t.Errorf("expected z≈0 at population mean, got %f", z)
	}
}

func TestNormalizeZScore_AboveMean(t *testing.T) {
	// Value above mean → positive z (worse)
	z := NormalizeZScore(125, 95, 15)
	if z <= 0 {
		t.Errorf("expected positive z above mean, got %f", z)
	}
}

func TestNormalizeZScore_BelowMean(t *testing.T) {
	// Value below mean → negative z (better for risk signals)
	z := NormalizeZScore(80, 95, 15)
	if z >= 0 {
		t.Errorf("expected negative z below mean, got %f", z)
	}
}

func TestNormalizeFBG_Optimal(t *testing.T) {
	z := NormalizeFBG(85)
	if z > -0.5 {
		t.Errorf("FBG 85 (optimal) should have z < -0.5, got %f", z)
	}
}

func TestNormalizeFBG_HighRisk(t *testing.T) {
	z := NormalizeFBG(150)
	if z < 2.0 {
		t.Errorf("FBG 150 (high risk) should have z > 2.0, got %f", z)
	}
}

func TestNormalizeWaist_Male_SouthAsianThreshold(t *testing.T) {
	// IDF South Asian threshold for males is 90cm
	z := NormalizeWaistSexSpecific(90, "M")
	// At threshold → should be moderately positive
	if z < 0 {
		t.Errorf("waist 90cm (male threshold) should have z > 0, got %f", z)
	}
}

func TestNormalizeWaist_Female_SouthAsianThreshold(t *testing.T) {
	// IDF South Asian threshold for females is 80cm
	z := NormalizeWaistSexSpecific(80, "F")
	if z < 0 {
		t.Errorf("waist 80cm (female threshold) should have z > 0, got %f", z)
	}
}

func TestNormalizeSteps_Sedentary(t *testing.T) {
	// 2000 steps = very sedentary → high positive z
	z := NormalizeSteps(2000)
	if z < 1.0 {
		t.Errorf("2000 steps should have z > 1.0, got %f", z)
	}
}

func TestNormalizeSteps_Active(t *testing.T) {
	// 10000 steps = active → negative z
	z := NormalizeSteps(10000)
	if z > -0.5 {
		t.Errorf("10000 steps should have z < -0.5, got %f", z)
	}
}

func TestDippingToZScore(t *testing.T) {
	tests := []struct {
		class    string
		wantMin  float64
		wantMax  float64
	}{
		{"DIPPER", -1.5, -0.5},
		{"NON_DIPPER", -0.1, 0.5},
		{"REVERSE_DIPPER", 1.5, 2.5},
	}
	for _, tc := range tests {
		z := DippingToZScore(tc.class)
		if z < tc.wantMin || z > tc.wantMax {
			t.Errorf("DippingToZScore(%s) = %f, want [%f, %f]", tc.class, z, tc.wantMin, tc.wantMax)
		}
	}
}

func TestNormalizeTrend_Improving(t *testing.T) {
	// HbA1c improving by 0.3%/quarter → negative z (good)
	z := NormalizeTrend(-0.3, 0, 0.2)
	if z > -1.0 {
		t.Errorf("improving HbA1c trend should have z < -1.0, got %f", z)
	}
}

func TestNormalizeTrend_Worsening(t *testing.T) {
	// HbA1c worsening by 0.5%/quarter → positive z (bad)
	z := NormalizeTrend(0.5, 0, 0.2)
	if z < 2.0 {
		t.Errorf("worsening HbA1c trend should have z > 2.0, got %f", z)
	}
}

func TestComputeSleepZScore(t *testing.T) {
	// Good sleep = negative z (lower risk)
	z := ComputeSleepZScore(1.0)
	if z > 0 {
		t.Errorf("expected negative z for good sleep, got %f", z)
	}
	// Poor sleep = positive z (higher risk)
	z = ComputeSleepZScore(0.0)
	if z < 0 {
		t.Errorf("expected positive z for poor sleep, got %f", z)
	}
}

func TestNormalizeWeightTrendBMIAware(t *testing.T) {
	// Normal BMI (25): weight loss = good (negative z)
	z := NormalizeWeightTrendBMI(-0.5, 25.0)
	if z >= 0 {
		t.Errorf("normal BMI weight loss should be negative z, got %f", z)
	}

	// Low BMI (<22): weight loss = BAD (positive z, penalized)
	z = NormalizeWeightTrendBMI(-0.5, 20.0)
	if z <= 0 {
		t.Errorf("low BMI weight loss should be penalized (positive z), got %f", z)
	}

	// Low BMI (<22): weight gain = good (negative z)
	z = NormalizeWeightTrendBMI(0.5, 20.0)
	if z >= 0 {
		t.Errorf("low BMI weight gain should be beneficial (negative z), got %f", z)
	}

	// Normal BMI: weight gain = bad (positive z)
	z = NormalizeWeightTrendBMI(0.5, 25.0)
	if z <= 0 {
		t.Errorf("normal BMI weight gain should be positive z, got %f", z)
	}
}
