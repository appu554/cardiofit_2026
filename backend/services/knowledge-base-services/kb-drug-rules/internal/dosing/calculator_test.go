package dosing

import (
	"math"
	"testing"
)

// Helper function to compare floats with tolerance
func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

// ============================================================================
// BSA CALCULATION TESTS
// ============================================================================

func TestCalculateBSA_Mosteller(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name     string
		heightCm float64
		weightKg float64
		expected float64
	}{
		{"Average adult male", 175.0, 75.0, 1.91},
		{"Average adult female", 165.0, 60.0, 1.66},
		{"Pediatric patient", 120.0, 25.0, 0.91},
		{"Obese adult", 180.0, 120.0, 2.45},
		{"Small adult", 150.0, 45.0, 1.37},
		{"Tall thin adult", 190.0, 70.0, 1.93},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateBSA(tt.heightCm, tt.weightKg)
			if !almostEqual(result, tt.expected, 0.05) {
				t.Errorf("CalculateBSA(%v, %v) = %v, want %v (±0.05)",
					tt.heightCm, tt.weightKg, result, tt.expected)
			}
		})
	}
}

func TestCalculateBSA_InvalidInput(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name     string
		heightCm float64
		weightKg float64
	}{
		{"Zero height", 0, 75.0},
		{"Zero weight", 175.0, 0},
		{"Negative height", -175.0, 75.0},
		{"Negative weight", 175.0, -75.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateBSA(tt.heightCm, tt.weightKg)
			if result != 0 {
				t.Errorf("CalculateBSA(%v, %v) = %v, want 0 for invalid input",
					tt.heightCm, tt.weightKg, result)
			}
		})
	}
}

// ============================================================================
// IBW CALCULATION TESTS
// ============================================================================

func TestCalculateIBW_Devine(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name     string
		heightCm float64
		gender   string
		expected float64
	}{
		// Male: 50 + 2.3 × (height_in - 60)
		// Verified formula: 182.88cm = 72in, IBW = 50 + 2.3*(72-60) = 77.6kg
		{"Male 6ft (183cm)", 182.88, "M", 77.6},
		{"Male 5ft10 (178cm)", 177.8, "M", 73.0},
		{"Male 5ft6 (168cm)", 167.64, "M", 63.8},

		// Female: 45.5 + 2.3 × (height_in - 60)
		{"Female 5ft6 (168cm)", 167.64, "F", 59.3},
		{"Female 5ft4 (163cm)", 162.56, "F", 54.7},
		{"Female 5ft (152cm)", 152.4, "F", 45.5}, // Minimum for <60 inches

		// Gender variations
		{"Male lowercase", 175.0, "m", 70.5},
		{"Female full word", 165.0, "female", 57.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateIBW(tt.heightCm, tt.gender)
			if !almostEqual(result, tt.expected, 1.0) {
				t.Errorf("CalculateIBW(%v, %v) = %v, want %v (±1.0)",
					tt.heightCm, tt.gender, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// ADJUSTED BODY WEIGHT TESTS
// ============================================================================

func TestCalculateAdjBW(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name      string
		actualKg  float64
		ibwKg     float64
		expected  float64
	}{
		{"Obese patient (120kg, IBW 70)", 120.0, 70.0, 90.0},  // 70 + 0.4*(120-70) = 90
		{"Mildly obese (90kg, IBW 70)", 90.0, 70.0, 78.0},     // 70 + 0.4*(90-70) = 78
		{"Normal weight (70kg, IBW 70)", 70.0, 70.0, 70.0},    // Returns actual
		{"Underweight (55kg, IBW 70)", 55.0, 70.0, 55.0},      // Returns actual (not obese)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateAdjBW(tt.actualKg, tt.ibwKg)
			if !almostEqual(result, tt.expected, 0.1) {
				t.Errorf("CalculateAdjBW(%v, %v) = %v, want %v",
					tt.actualKg, tt.ibwKg, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// CREATININE CLEARANCE TESTS (Cockcroft-Gault)
// ============================================================================

func TestCalculateCrCl_CockcroftGault(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name     string
		age      int
		weightKg float64
		scr      float64 // Serum creatinine mg/dL
		gender   string
		expected float64 // mL/min
	}{
		// Male: [(140 - Age) × Weight] / [72 × SCr]
		{"Young healthy male", 30, 80.0, 1.0, "M", 122.22},
		{"Middle-aged male", 50, 75.0, 1.2, "M", 78.12},
		{"Elderly male", 75, 70.0, 1.5, "M", 42.13},
		{"Male with CKD", 60, 80.0, 2.5, "M", 35.56},

		// Female: multiply by 0.85
		{"Young healthy female", 30, 65.0, 0.8, "F", 105.06},
		{"Middle-aged female", 50, 60.0, 1.0, "F", 63.75},
		// CrCl = [(140-75)*55] / [72*1.2] * 0.85 = (65*55)/(86.4)*0.85 = 35.17
		{"Elderly female", 75, 55.0, 1.2, "F", 35.17},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateCrCl(tt.age, tt.weightKg, tt.scr, tt.gender)
			if !almostEqual(result, tt.expected, 1.0) {
				t.Errorf("CalculateCrCl(%v, %v, %v, %v) = %v, want %v (±1.0)",
					tt.age, tt.weightKg, tt.scr, tt.gender, result, tt.expected)
			}
		})
	}
}

func TestCalculateCrCl_InvalidInput(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name     string
		age      int
		weightKg float64
		scr      float64
		gender   string
	}{
		{"Zero age", 0, 80.0, 1.0, "M"},
		{"Negative age", -30, 80.0, 1.0, "M"},
		{"Zero weight", 30, 0, 1.0, "M"},
		{"Zero creatinine", 30, 80.0, 0, "M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateCrCl(tt.age, tt.weightKg, tt.scr, tt.gender)
			if result != 0 {
				t.Errorf("Expected 0 for invalid input, got %v", result)
			}
		})
	}
}

// ============================================================================
// EGFR TESTS (CKD-EPI 2021)
// ============================================================================

func TestCalculateEGFR_CKDEPI2021(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name     string
		age      int
		scr      float64
		gender   string
		minEGFR  float64
		maxEGFR  float64
	}{
		// CKD-EPI 2021 race-free equation ranges
		{"Young male normal SCr", 30, 1.0, "M", 95.0, 115.0},
		{"Middle-aged male normal", 50, 1.0, "M", 80.0, 100.0},
		{"Elderly male normal", 75, 1.0, "M", 65.0, 85.0},
		{"Male elevated SCr", 50, 1.5, "M", 45.0, 60.0},
		{"Male high SCr (CKD)", 50, 2.5, "M", 25.0, 35.0},

		{"Young female normal", 30, 0.8, "F", 100.0, 130.0},
		{"Middle-aged female normal", 50, 0.9, "F", 70.0, 95.0},
		{"Elderly female normal", 75, 1.0, "F", 55.0, 75.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateEGFR(tt.age, tt.scr, tt.gender)
			if result < tt.minEGFR || result > tt.maxEGFR {
				t.Errorf("CalculateEGFR(%v, %v, %v) = %v, expected range [%v, %v]",
					tt.age, tt.scr, tt.gender, result, tt.minEGFR, tt.maxEGFR)
			}
		})
	}
}

// ============================================================================
// CKD STAGING TESTS
// ============================================================================

func TestGetCKDStage(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		egfr          float64
		expectedStage string
	}{
		{120.0, "G1"},
		{95.0, "G1"},
		{90.0, "G1"},
		{85.0, "G2"},
		{60.0, "G2"},
		{55.0, "G3a"},
		{45.0, "G3a"},
		{40.0, "G3b"},
		{30.0, "G3b"},
		{25.0, "G4"},
		{15.0, "G4"},
		{10.0, "G5"},
		{5.0, "G5"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedStage, func(t *testing.T) {
			stage, _ := calc.GetCKDStage(tt.egfr)
			if stage != tt.expectedStage {
				t.Errorf("GetCKDStage(%v) = %v, want %v", tt.egfr, stage, tt.expectedStage)
			}
		})
	}
}

// ============================================================================
// BMI TESTS
// ============================================================================

func TestCalculateBMI(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name     string
		heightCm float64
		weightKg float64
		expected float64
	}{
		{"Normal BMI", 175.0, 70.0, 22.9},
		{"Underweight", 175.0, 50.0, 16.3},
		{"Overweight", 175.0, 85.0, 27.8},
		{"Obese", 175.0, 100.0, 32.7},
		{"Morbidly obese", 170.0, 130.0, 45.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateBMI(tt.heightCm, tt.weightKg)
			if !almostEqual(result, tt.expected, 0.2) {
				t.Errorf("CalculateBMI(%v, %v) = %v, want %v",
					tt.heightCm, tt.weightKg, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// CALCULATE ALL PARAMETERS TESTS
// ============================================================================

func TestCalculateAllParameters(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name   string
		params PatientParameters
		checks func(*CalculatedParameters) bool
	}{
		{
			name: "Normal adult male",
			params: PatientParameters{
				Age:             45,
				Gender:          "M",
				WeightKg:        80.0,
				HeightCm:        175.0,
				SerumCreatinine: 1.0,
			},
			checks: func(cp *CalculatedParameters) bool {
				return cp.BSA > 1.8 && cp.BSA < 2.1 &&
					cp.IBW > 65 && cp.IBW < 80 &&
					cp.EGFR > 80 && cp.EGFR < 110 &&
					cp.CrCl > 80 && cp.CrCl < 130 &&
					!cp.IsPediatric && !cp.IsGeriatric
			},
		},
		{
			name: "Geriatric female with CKD",
			params: PatientParameters{
				Age:             75,
				Gender:          "F",
				WeightKg:        60.0,
				HeightCm:        160.0,
				SerumCreatinine: 1.5,
			},
			checks: func(cp *CalculatedParameters) bool {
				return cp.IsGeriatric &&
					!cp.IsPediatric &&
					cp.EGFR < 60 // CKD
			},
		},
		{
			name: "Obese patient",
			params: PatientParameters{
				Age:             50,
				Gender:          "M",
				WeightKg:        120.0,
				HeightCm:        175.0,
				SerumCreatinine: 1.0,
			},
			checks: func(cp *CalculatedParameters) bool {
				// For 120kg patient with IBW ~71kg:
				// AdjBW = IBW + 0.4*(120-IBW) ≈ 90kg
				// So AdjBW < 120 (actual weight) and AdjBW > 71 (IBW)
				return cp.IsObese &&
					cp.BMI > 30 &&
					cp.AdjBW < 120.0 && // Less than actual weight
					cp.AdjBW > cp.IBW
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.CalculateAllParameters(tt.params)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !tt.checks(result) {
				t.Errorf("CalculateAllParameters failed checks. Result: %+v", result)
			}
		})
	}
}

func TestCalculateAllParameters_InvalidInput(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name   string
		params PatientParameters
	}{
		{
			name: "Zero weight",
			params: PatientParameters{
				Age:      30,
				Gender:   "M",
				WeightKg: 0,
				HeightCm: 175.0,
			},
		},
		{
			name: "Zero height",
			params: PatientParameters{
				Age:      30,
				Gender:   "M",
				WeightKg: 80.0,
				HeightCm: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := calc.CalculateAllParameters(tt.params)
			if err == nil {
				t.Error("Expected error for invalid input, got nil")
			}
		})
	}
}

// ============================================================================
// CHILD-PUGH CLASSIFICATION TESTS
// ============================================================================

func TestGetChildPughClassFromScore(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		score    int
		expected string
	}{
		{5, "A"},
		{6, "A"},
		{7, "B"},
		{8, "B"},
		{9, "B"},
		{10, "C"},
		{11, "C"},
		{15, "C"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			class, _ := calc.GetChildPughClassFromScore(tt.score)
			if class != tt.expected {
				t.Errorf("GetChildPughClassFromScore(%v) = %v, want %v",
					tt.score, class, tt.expected)
			}
		})
	}
}
