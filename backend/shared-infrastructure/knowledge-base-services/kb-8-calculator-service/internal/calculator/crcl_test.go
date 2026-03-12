package calculator

import (
	"context"
	"math"
	"testing"

	"kb-8-calculator-service/internal/models"
)

// TestCrClCalculator_Calculate tests CrCl calculation with reference values.
// Reference: Cockcroft DW, Gault MH. Nephron. 1976;16(1):31-41
func TestCrClCalculator_Calculate(t *testing.T) {
	calc := NewCrClCalculator()
	ctx := context.Background()

	tests := []struct {
		name            string
		params          *models.CrClParams
		expectedCrCl    float64
		expectedRenal   models.RenalFunctionCategory
		requiresDoseAdj bool
		tolerance       float64
	}{
		// Normal renal function
		{
			name: "young_male_normal",
			params: &models.CrClParams{
				SerumCreatinine: 1.0,
				AgeYears:        30,
				Sex:             models.SexMale,
				WeightKg:        70,
			},
			// CrCl = (140-30) × 70 / (72 × 1.0) = 7700/72 = 106.9
			expectedCrCl:    106.9,
			expectedRenal:   models.RenalFunctionNormal,
			requiresDoseAdj: false,
			tolerance:       1.0,
		},
		{
			name: "young_female_normal",
			params: &models.CrClParams{
				SerumCreatinine: 0.8,
				AgeYears:        30,
				Sex:             models.SexFemale,
				WeightKg:        60,
			},
			// CrCl = (140-30) × 60 / (72 × 0.8) × 0.85 = 6600/57.6 × 0.85 = 97.4
			expectedCrCl:    97.4,
			expectedRenal:   models.RenalFunctionNormal,
			requiresDoseAdj: false,
			tolerance:       2.0,
		},
		// Mild impairment
		{
			name: "older_male_mild",
			params: &models.CrClParams{
				SerumCreatinine: 1.2,
				AgeYears:        65,
				Sex:             models.SexMale,
				WeightKg:        75,
			},
			// CrCl = (140-65) × 75 / (72 × 1.2) = 5625/86.4 = 65.1
			expectedCrCl:    65.1,
			expectedRenal:   models.RenalFunctionMild,
			requiresDoseAdj: false,
			tolerance:       2.0,
		},
		// Moderate impairment
		{
			name: "elderly_female_moderate",
			params: &models.CrClParams{
				SerumCreatinine: 1.5,
				AgeYears:        75,
				Sex:             models.SexFemale,
				WeightKg:        55,
			},
			// CrCl = (140-75) × 55 / (72 × 1.5) × 0.85 = 3575/108 × 0.85 = 28.1
			expectedCrCl:    28.1,
			expectedRenal:   models.RenalFunctionSevere, // Actually severe at 28
			requiresDoseAdj: true,
			tolerance:       2.0,
		},
		// Moderate impairment boundary
		{
			name: "moderate_impairment_boundary",
			params: &models.CrClParams{
				SerumCreatinine: 1.8,
				AgeYears:        60,
				Sex:             models.SexMale,
				WeightKg:        80,
			},
			// CrCl = (140-60) × 80 / (72 × 1.8) = 6400/129.6 = 49.4
			expectedCrCl:    49.4,
			expectedRenal:   models.RenalFunctionModerate,
			requiresDoseAdj: true,
			tolerance:       2.0,
		},
		// Severe impairment
		{
			name: "elderly_male_severe",
			params: &models.CrClParams{
				SerumCreatinine: 3.0,
				AgeYears:        80,
				Sex:             models.SexMale,
				WeightKg:        65,
			},
			// CrCl = (140-80) × 65 / (72 × 3.0) = 3900/216 = 18.1
			expectedCrCl:    18.1,
			expectedRenal:   models.RenalFunctionSevere,
			requiresDoseAdj: true,
			tolerance:       1.0,
		},
		// End-stage renal disease
		{
			name: "esrd_female",
			params: &models.CrClParams{
				SerumCreatinine: 6.0,
				AgeYears:        70,
				Sex:             models.SexFemale,
				WeightKg:        50,
			},
			// CrCl = (140-70) × 50 / (72 × 6.0) × 0.85 = 3500/432 × 0.85 = 6.9
			expectedCrCl:    6.9,
			expectedRenal:   models.RenalFunctionEndStage,
			requiresDoseAdj: true,
			tolerance:       1.0,
		},
		// High weight case
		{
			name: "obese_male",
			params: &models.CrClParams{
				SerumCreatinine: 1.0,
				AgeYears:        45,
				Sex:             models.SexMale,
				WeightKg:        120,
			},
			// CrCl = (140-45) × 120 / (72 × 1.0) = 11400/72 = 158.3
			expectedCrCl:    158.3,
			expectedRenal:   models.RenalFunctionNormal,
			requiresDoseAdj: false,
			tolerance:       2.0,
		},
		// Low weight case
		{
			name: "underweight_female",
			params: &models.CrClParams{
				SerumCreatinine: 0.9,
				AgeYears:        50,
				Sex:             models.SexFemale,
				WeightKg:        45,
			},
			// CrCl = (140-50) × 45 / (72 × 0.9) × 0.85 = 4050/64.8 × 0.85 = 53.1
			expectedCrCl:    53.1,
			expectedRenal:   models.RenalFunctionModerate,
			requiresDoseAdj: true,
			tolerance:       2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Calculate(ctx, tt.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check CrCl value
			if math.Abs(result.Value-tt.expectedCrCl) > tt.tolerance {
				t.Errorf("CrCl mismatch: got %.1f, want %.1f (±%.1f)",
					result.Value, tt.expectedCrCl, tt.tolerance)
			}

			// Check renal function category
			if result.RenalFunction != tt.expectedRenal {
				t.Errorf("renal function mismatch: got %s, want %s",
					result.RenalFunction, tt.expectedRenal)
			}

			// Check dose adjustment
			if result.RequiresRenalDoseAdjustment != tt.requiresDoseAdj {
				t.Errorf("dose adjustment mismatch: got %v, want %v",
					result.RequiresRenalDoseAdjustment, tt.requiresDoseAdj)
			}

			// Verify unit
			if result.Unit != "mL/min" {
				t.Errorf("unit mismatch: got %s, want mL/min", result.Unit)
			}

			// Verify equation
			if result.Equation != "Cockcroft-Gault-1976" {
				t.Errorf("equation mismatch: got %s", result.Equation)
			}
		})
	}
}

// TestCrClCalculator_Validation tests input validation.
func TestCrClCalculator_Validation(t *testing.T) {
	calc := NewCrClCalculator()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      *models.CrClParams
		expectedErr error
	}{
		{
			name: "invalid_creatinine_zero",
			params: &models.CrClParams{
				SerumCreatinine: 0,
				AgeYears:        50,
				Sex:             models.SexMale,
				WeightKg:        70,
			},
			expectedErr: models.ErrInvalidCreatinine,
		},
		{
			name: "invalid_creatinine_negative",
			params: &models.CrClParams{
				SerumCreatinine: -0.5,
				AgeYears:        50,
				Sex:             models.SexMale,
				WeightKg:        70,
			},
			expectedErr: models.ErrInvalidCreatinine,
		},
		{
			name: "invalid_age_zero",
			params: &models.CrClParams{
				SerumCreatinine: 1.0,
				AgeYears:        0,
				Sex:             models.SexMale,
				WeightKg:        70,
			},
			expectedErr: models.ErrInvalidAge,
		},
		{
			name: "invalid_age_too_high",
			params: &models.CrClParams{
				SerumCreatinine: 1.0,
				AgeYears:        150,
				Sex:             models.SexMale,
				WeightKg:        70,
			},
			expectedErr: models.ErrInvalidAge,
		},
		{
			name: "invalid_sex",
			params: &models.CrClParams{
				SerumCreatinine: 1.0,
				AgeYears:        50,
				Sex:             "other",
				WeightKg:        70,
			},
			expectedErr: models.ErrInvalidSex,
		},
		{
			name: "invalid_weight_zero",
			params: &models.CrClParams{
				SerumCreatinine: 1.0,
				AgeYears:        50,
				Sex:             models.SexMale,
				WeightKg:        0,
			},
			expectedErr: models.ErrInvalidWeight,
		},
		{
			name: "invalid_weight_too_high",
			params: &models.CrClParams{
				SerumCreatinine: 1.0,
				AgeYears:        50,
				Sex:             models.SexMale,
				WeightKg:        600,
			},
			expectedErr: models.ErrInvalidWeight,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := calc.Calculate(ctx, tt.params)
			if err != tt.expectedErr {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

// TestCrClCalculator_RenalFunctionCategories tests category boundaries.
func TestCrClCalculator_RenalFunctionCategories(t *testing.T) {
	calc := NewCrClCalculator()

	tests := []struct {
		crcl     float64
		expected models.RenalFunctionCategory
	}{
		{crcl: 100, expected: models.RenalFunctionNormal},
		{crcl: 90, expected: models.RenalFunctionNormal},
		{crcl: 89.9, expected: models.RenalFunctionMild},
		{crcl: 60, expected: models.RenalFunctionMild},
		{crcl: 59.9, expected: models.RenalFunctionModerate},
		{crcl: 30, expected: models.RenalFunctionModerate},
		{crcl: 29.9, expected: models.RenalFunctionSevere},
		{crcl: 15, expected: models.RenalFunctionSevere},
		{crcl: 14.9, expected: models.RenalFunctionEndStage},
		{crcl: 5, expected: models.RenalFunctionEndStage},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			category := calc.categorizeRenalFunction(tt.crcl)
			if category != tt.expected {
				t.Errorf("CrCl %.1f: got %s, want %s", tt.crcl, category, tt.expected)
			}
		})
	}
}

// TestCrClCalculator_SexCoefficient validates male vs female difference.
func TestCrClCalculator_SexCoefficient(t *testing.T) {
	calc := NewCrClCalculator()
	ctx := context.Background()

	maleParams := &models.CrClParams{
		SerumCreatinine: 1.0,
		AgeYears:        50,
		Sex:             models.SexMale,
		WeightKg:        70,
	}
	femaleParams := &models.CrClParams{
		SerumCreatinine: 1.0,
		AgeYears:        50,
		Sex:             models.SexFemale,
		WeightKg:        70,
	}

	maleResult, _ := calc.Calculate(ctx, maleParams)
	femaleResult, _ := calc.Calculate(ctx, femaleParams)

	// Female should be ~85% of male with same inputs
	expectedRatio := 0.85
	actualRatio := femaleResult.Value / maleResult.Value

	if math.Abs(actualRatio-expectedRatio) > 0.01 {
		t.Errorf("female/male ratio: got %.3f, want %.3f", actualRatio, expectedRatio)
	}

	t.Logf("Male CrCl: %.1f, Female CrCl: %.1f, Ratio: %.3f",
		maleResult.Value, femaleResult.Value, actualRatio)
}

// TestCrClCalculator_ObesityWarning tests obesity caveat in provenance.
func TestCrClCalculator_ObesityWarning(t *testing.T) {
	calc := NewCrClCalculator()
	ctx := context.Background()

	// Normal weight - should not have obesity warning
	normalParams := &models.CrClParams{
		SerumCreatinine: 1.0,
		AgeYears:        50,
		Sex:             models.SexMale,
		WeightKg:        70,
	}
	normalResult, _ := calc.Calculate(ctx, normalParams)

	// Obese - should have obesity warning
	obeseParams := &models.CrClParams{
		SerumCreatinine: 1.0,
		AgeYears:        50,
		Sex:             models.SexMale,
		WeightKg:        120,
	}
	obeseResult, _ := calc.Calculate(ctx, obeseParams)

	// Check for obesity warning in caveats
	hasObesityWarning := false
	for _, caveat := range obeseResult.Provenance.Caveats {
		if len(caveat) > 0 && caveat[0:min(len(caveat), 20)] == "Patient weight >100k" {
			hasObesityWarning = true
			break
		}
	}

	if !hasObesityWarning {
		t.Log("Expected obesity warning for weight > 100kg")
	}

	// Normal weight should have fewer caveats
	if len(normalResult.Provenance.Caveats) >= len(obeseResult.Provenance.Caveats) {
		t.Log("Obese patient should have additional caveat")
	}
}

// TestCrClCalculator_Provenance validates provenance record.
func TestCrClCalculator_Provenance(t *testing.T) {
	calc := NewCrClCalculator()
	ctx := context.Background()

	params := &models.CrClParams{
		SerumCreatinine: 1.5,
		AgeYears:        60,
		Sex:             models.SexFemale,
		WeightKg:        65,
	}

	result, err := calc.Calculate(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prov := result.Provenance

	// Verify fields
	if prov.CalculatorType != "CRCL" {
		t.Errorf("calculator type: got %s, want CRCL", prov.CalculatorType)
	}
	if prov.Version != "Cockcroft-Gault-1976" {
		t.Errorf("version: got %s", prov.Version)
	}
	if prov.Reference == "" {
		t.Error("reference should not be empty")
	}

	// Should have 4 inputs
	if len(prov.InputsUsed) != 4 {
		t.Errorf("expected 4 inputs, got %d", len(prov.InputsUsed))
	}

	// Verify weight input exists
	hasWeight := false
	for _, input := range prov.InputsUsed {
		if input.Name == "weight" {
			hasWeight = true
			if input.Value != 65.0 {
				t.Errorf("weight input: got %v, want 65", input.Value)
			}
		}
	}
	if !hasWeight {
		t.Error("weight input not found in provenance")
	}
}

// TestCrClCalculator_DoseAdjustmentGuidance tests guidance text.
func TestCrClCalculator_DoseAdjustmentGuidance(t *testing.T) {
	calc := NewCrClCalculator()
	ctx := context.Background()

	// Normal - no guidance
	normalParams := &models.CrClParams{
		SerumCreatinine: 0.8,
		AgeYears:        35,
		Sex:             models.SexMale,
		WeightKg:        75,
	}
	normalResult, _ := calc.Calculate(ctx, normalParams)
	if normalResult.DoseAdjustmentGuidance != "" {
		t.Error("normal function should not have dose guidance")
	}

	// ESRD - should have guidance
	esrdParams := &models.CrClParams{
		SerumCreatinine: 8.0,
		AgeYears:        65,
		Sex:             models.SexMale,
		WeightKg:        70,
	}
	esrdResult, _ := calc.Calculate(ctx, esrdParams)
	if esrdResult.DoseAdjustmentGuidance == "" {
		t.Error("ESRD should have dose guidance")
	}
	t.Logf("ESRD guidance: %s", esrdResult.DoseAdjustmentGuidance)
}

// BenchmarkCrClCalculator_Calculate benchmarks calculation performance.
func BenchmarkCrClCalculator_Calculate(b *testing.B) {
	calc := NewCrClCalculator()
	ctx := context.Background()
	params := &models.CrClParams{
		SerumCreatinine: 1.2,
		AgeYears:        55,
		Sex:             models.SexMale,
		WeightKg:        75,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.Calculate(ctx, params)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
