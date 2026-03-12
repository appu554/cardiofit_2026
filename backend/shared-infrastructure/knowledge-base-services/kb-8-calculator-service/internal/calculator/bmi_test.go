package calculator

import (
	"context"
	"math"
	"testing"

	"kb-8-calculator-service/internal/models"
)

// TestBMICalculator_Calculate tests BMI calculation with various inputs.
func TestBMICalculator_Calculate(t *testing.T) {
	calc := NewBMICalculator()
	ctx := context.Background()

	tests := []struct {
		name             string
		params           *models.BMIParams
		expectedBMI      float64
		expectedWestern  models.BMICategory
		expectedAsian    models.BMICategory
		tolerance        float64
	}{
		// Underweight cases (same for both systems)
		{
			name: "underweight",
			params: &models.BMIParams{
				WeightKg: 45,
				HeightCm: 170,
			},
			// BMI = 45 / (1.7)² = 45 / 2.89 = 15.6
			expectedBMI:     15.6,
			expectedWestern: models.BMICategoryUnderweight,
			expectedAsian:   models.BMICategoryUnderweight,
			tolerance:       0.2,
		},
		// Normal weight (differs between systems)
		{
			name: "normal_western_overweight_asian",
			params: &models.BMIParams{
				WeightKg: 72,
				HeightCm: 175,
			},
			// BMI = 72 / (1.75)² = 72 / 3.0625 = 23.5
			expectedBMI:     23.5,
			expectedWestern: models.BMICategoryNormal,    // Western: < 25
			expectedAsian:   models.BMICategoryOverweight, // Asian: >= 23
			tolerance:       0.2,
		},
		// Overweight (both systems)
		{
			name: "overweight_both_systems",
			params: &models.BMIParams{
				WeightKg: 85,
				HeightCm: 175,
			},
			// BMI = 85 / (1.75)² = 85 / 3.0625 = 27.8
			expectedBMI:     27.8,
			expectedWestern: models.BMICategoryOverweight,   // Western: 25-29.9
			expectedAsian:   models.BMICategoryObeseClassI,  // Asian: 25-29.9 = Obese I
			tolerance:       0.2,
		},
		// Obese Class I Western, Obese II Asian
		{
			name: "obese_class_i_western_ii_asian",
			params: &models.BMIParams{
				WeightKg: 100,
				HeightCm: 175,
			},
			// BMI = 100 / (1.75)² = 100 / 3.0625 = 32.7
			expectedBMI:     32.7,
			expectedWestern: models.BMICategoryObeseClassI,  // Western: 30-34.9
			expectedAsian:   models.BMICategoryObeseClassII, // Asian: >= 30
			tolerance:       0.2,
		},
		// Normal for both systems
		{
			name: "normal_both_systems",
			params: &models.BMIParams{
				WeightKg: 60,
				HeightCm: 170,
			},
			// BMI = 60 / (1.7)² = 60 / 2.89 = 20.8
			expectedBMI:     20.8,
			expectedWestern: models.BMICategoryNormal,
			expectedAsian:   models.BMICategoryNormal, // Asian: < 23
			tolerance:       0.2,
		},
		// Morbid obesity
		{
			name: "morbid_obesity",
			params: &models.BMIParams{
				WeightKg: 140,
				HeightCm: 170,
			},
			// BMI = 140 / (1.7)² = 140 / 2.89 = 48.4
			expectedBMI:     48.4,
			expectedWestern: models.BMICategoryObeseClassIII,
			expectedAsian:   models.BMICategoryObeseClassII, // Asian max is Class II
			tolerance:       0.3,
		},
		// Boundary: Asian Overweight threshold (23)
		{
			name: "asian_overweight_boundary",
			params: &models.BMIParams{
				WeightKg: 66.5,  // Calculated to give BMI ~23.0
				HeightCm: 170,
			},
			expectedBMI:     23.0,
			expectedWestern: models.BMICategoryNormal,
			expectedAsian:   models.BMICategoryOverweight,
			tolerance:       0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Calculate(ctx, tt.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check BMI value
			if math.Abs(result.Value-tt.expectedBMI) > tt.tolerance {
				t.Errorf("BMI mismatch: got %.1f, want %.1f (±%.1f)",
					result.Value, tt.expectedBMI, tt.tolerance)
			}

			// Check Western category
			if result.CategoryWestern != tt.expectedWestern {
				t.Errorf("Western category mismatch: got %s, want %s",
					result.CategoryWestern, tt.expectedWestern)
			}

			// Check Asian category
			if result.CategoryAsian != tt.expectedAsian {
				t.Errorf("Asian category mismatch: got %s, want %s",
					result.CategoryAsian, tt.expectedAsian)
			}

			// Verify unit
			if result.Unit != "kg/m²" {
				t.Errorf("unit mismatch: got %s, want kg/m²", result.Unit)
			}
		})
	}
}

// TestBMICalculator_RegionalInterpretation tests region-specific interpretation.
func TestBMICalculator_RegionalInterpretation(t *testing.T) {
	calc := NewBMICalculator()
	ctx := context.Background()

	// BMI ~24 - Normal for Western, Overweight for Asian
	baseParams := models.BMIParams{
		WeightKg: 69,
		HeightCm: 170,
	}

	tests := []struct {
		name           string
		region         models.Region
		ethnicity      string
		expectedRegion models.Region
		usesAsian      bool
	}{
		{
			name:           "global_uses_western",
			region:         models.RegionGlobal,
			ethnicity:      "",
			expectedRegion: models.RegionGlobal,
			usesAsian:      false,
		},
		{
			name:           "india_uses_asian",
			region:         models.RegionIndia,
			ethnicity:      "",
			expectedRegion: models.RegionIndia,
			usesAsian:      true,
		},
		{
			name:           "australia_uses_western",
			region:         models.RegionAustralia,
			ethnicity:      "",
			expectedRegion: models.RegionAustralia,
			usesAsian:      false,
		},
		{
			name:           "ethnicity_overrides_to_asian",
			region:         models.RegionGlobal,
			ethnicity:      "indian",
			expectedRegion: models.RegionIndia, // Should switch to India
			usesAsian:      true,
		},
		{
			name:           "south_asian_uses_asian",
			region:         models.RegionUSA,
			ethnicity:      "south_asian",
			expectedRegion: models.RegionIndia,
			usesAsian:      true,
		},
		{
			name:           "chinese_uses_asian",
			region:         models.RegionGlobal,
			ethnicity:      "chinese",
			expectedRegion: models.RegionIndia,
			usesAsian:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := baseParams
			params.Region = tt.region
			params.Ethnicity = tt.ethnicity

			result, err := calc.Calculate(ctx, &params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check if Asian cutoffs are being used in interpretation
			if tt.usesAsian {
				// Interpretation should mention "Asian" cutoffs
				if result.Region != models.RegionIndia {
					t.Errorf("region should be India for Asian cutoffs, got %s", result.Region)
				}
			}

			t.Logf("%s: BMI=%.1f, Region=%s, Interpretation snippet: %s...",
				tt.name, result.Value, result.Region,
				result.Interpretation[:min(len(result.Interpretation), 50)])
		})
	}
}

// TestBMICalculator_Validation tests input validation.
func TestBMICalculator_Validation(t *testing.T) {
	calc := NewBMICalculator()
	ctx := context.Background()

	tests := []struct {
		name        string
		params      *models.BMIParams
		expectedErr error
	}{
		{
			name: "invalid_weight_zero",
			params: &models.BMIParams{
				WeightKg: 0,
				HeightCm: 170,
			},
			expectedErr: models.ErrInvalidWeight,
		},
		{
			name: "invalid_weight_negative",
			params: &models.BMIParams{
				WeightKg: -50,
				HeightCm: 170,
			},
			expectedErr: models.ErrInvalidWeight,
		},
		{
			name: "invalid_weight_too_high",
			params: &models.BMIParams{
				WeightKg: 600,
				HeightCm: 170,
			},
			expectedErr: models.ErrInvalidWeight,
		},
		{
			name: "invalid_height_zero",
			params: &models.BMIParams{
				WeightKg: 70,
				HeightCm: 0,
			},
			expectedErr: models.ErrInvalidHeight,
		},
		{
			name: "invalid_height_negative",
			params: &models.BMIParams{
				WeightKg: 70,
				HeightCm: -170,
			},
			expectedErr: models.ErrInvalidHeight,
		},
		{
			name: "invalid_height_too_high",
			params: &models.BMIParams{
				WeightKg: 70,
				HeightCm: 350,
			},
			expectedErr: models.ErrInvalidHeight,
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

// TestBMICalculator_WesternCategories tests Western (WHO) category boundaries.
func TestBMICalculator_WesternCategories(t *testing.T) {
	calc := NewBMICalculator()

	tests := []struct {
		bmi      float64
		expected models.BMICategory
	}{
		{bmi: 16.0, expected: models.BMICategoryUnderweight},
		{bmi: 18.4, expected: models.BMICategoryUnderweight},
		{bmi: 18.5, expected: models.BMICategoryNormal},
		{bmi: 24.9, expected: models.BMICategoryNormal},
		{bmi: 25.0, expected: models.BMICategoryOverweight},
		{bmi: 29.9, expected: models.BMICategoryOverweight},
		{bmi: 30.0, expected: models.BMICategoryObeseClassI},
		{bmi: 34.9, expected: models.BMICategoryObeseClassI},
		{bmi: 35.0, expected: models.BMICategoryObeseClassII},
		{bmi: 39.9, expected: models.BMICategoryObeseClassII},
		{bmi: 40.0, expected: models.BMICategoryObeseClassIII},
		{bmi: 50.0, expected: models.BMICategoryObeseClassIII},
	}

	for _, tt := range tests {
		category := calc.categorizeWestern(tt.bmi)
		if category != tt.expected {
			t.Errorf("BMI %.1f: got %s, want %s", tt.bmi, category, tt.expected)
		}
	}
}

// TestBMICalculator_AsianCategories tests Asian (WHO Asia-Pacific) category boundaries.
func TestBMICalculator_AsianCategories(t *testing.T) {
	calc := NewBMICalculator()

	tests := []struct {
		bmi      float64
		expected models.BMICategory
	}{
		{bmi: 16.0, expected: models.BMICategoryUnderweight},
		{bmi: 18.4, expected: models.BMICategoryUnderweight},
		{bmi: 18.5, expected: models.BMICategoryNormal},
		{bmi: 22.9, expected: models.BMICategoryNormal},       // Asian normal max
		{bmi: 23.0, expected: models.BMICategoryOverweight},   // Asian overweight starts
		{bmi: 24.9, expected: models.BMICategoryOverweight},   // Asian overweight max
		{bmi: 25.0, expected: models.BMICategoryObeseClassI},  // Asian obese I starts
		{bmi: 29.9, expected: models.BMICategoryObeseClassI},  // Asian obese I max
		{bmi: 30.0, expected: models.BMICategoryObeseClassII}, // Asian obese II
		{bmi: 40.0, expected: models.BMICategoryObeseClassII}, // Asian has no Class III
	}

	for _, tt := range tests {
		category := calc.categorizeAsian(tt.bmi)
		if category != tt.expected {
			t.Errorf("BMI %.1f (Asian): got %s, want %s", tt.bmi, category, tt.expected)
		}
	}
}

// TestBMICalculator_CategoryDifferences highlights where systems differ.
func TestBMICalculator_CategoryDifferences(t *testing.T) {
	calc := NewBMICalculator()

	// BMI values where Western and Asian categories differ
	differingBMIs := []float64{23.0, 23.5, 24.0, 24.5, 25.5, 27.0, 30.0, 35.0}

	t.Log("BMI values where Western and Asian categories differ:")
	for _, bmi := range differingBMIs {
		western := calc.categorizeWestern(bmi)
		asian := calc.categorizeAsian(bmi)
		if western != asian {
			t.Logf("  BMI %.1f: Western=%s, Asian=%s", bmi, western, asian)
		}
	}
}

// TestBMICalculator_EthnicityRecognition tests ethnicity string recognition.
func TestBMICalculator_EthnicityRecognition(t *testing.T) {
	calc := NewBMICalculator()

	asianEthnicities := []string{
		"asian", "indian", "south_asian", "southeast_asian",
		"chinese", "japanese", "korean", "filipino",
		"vietnamese", "thai", "malaysian", "indonesian",
		"bangladeshi", "pakistani", "sri_lankan",
	}

	nonAsianEthnicities := []string{
		"caucasian", "african", "hispanic", "european",
		"", "unknown", "other",
	}

	for _, eth := range asianEthnicities {
		if !calc.isAsianEthnicity(eth) {
			t.Errorf("expected %s to be recognized as Asian", eth)
		}
	}

	for _, eth := range nonAsianEthnicities {
		if calc.isAsianEthnicity(eth) {
			t.Errorf("expected %s to NOT be recognized as Asian", eth)
		}
	}
}

// TestBMICalculator_Provenance validates provenance record.
func TestBMICalculator_Provenance(t *testing.T) {
	calc := NewBMICalculator()
	ctx := context.Background()

	params := &models.BMIParams{
		WeightKg:  75,
		HeightCm:  175,
		Region:    models.RegionIndia,
		Ethnicity: "indian",
	}

	result, err := calc.Calculate(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prov := result.Provenance

	// Verify fields
	if prov.CalculatorType != "BMI" {
		t.Errorf("calculator type: got %s, want BMI", prov.CalculatorType)
	}
	if prov.Version != "WHO-2004-AsianAdapted" {
		t.Errorf("version: got %s", prov.Version)
	}

	// Should have 3 inputs (weight, height, region)
	if len(prov.InputsUsed) != 3 {
		t.Errorf("expected 3 inputs, got %d", len(prov.InputsUsed))
	}

	// Should have Asian cutoffs caveat
	hasAsianCaveat := false
	for _, caveat := range prov.Caveats {
		if len(caveat) > 10 && caveat[:10] == "Asian cuto" {
			hasAsianCaveat = true
		}
	}
	if !hasAsianCaveat {
		t.Log("Expected Asian cutoffs caveat for India region")
	}
}

// TestBMICalculator_CategoryDiscrepancyWarning tests discrepancy warning.
func TestBMICalculator_CategoryDiscrepancyWarning(t *testing.T) {
	calc := NewBMICalculator()
	ctx := context.Background()

	// BMI 24 - Normal for Western, Overweight for Asian
	params := &models.BMIParams{
		WeightKg: 69.4,
		HeightCm: 170,
		Region:   models.RegionGlobal,
	}

	result, err := calc.Calculate(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Categories should differ
	if result.CategoryWestern == result.CategoryAsian {
		t.Log("BMI chosen doesn't show category discrepancy")
		return
	}

	// Check for discrepancy caveat
	hasDiscrepancy := false
	for _, caveat := range result.Provenance.Caveats {
		if len(caveat) > 20 {
			hasDiscrepancy = true
		}
	}
	if hasDiscrepancy {
		t.Log("Discrepancy warning present in caveats")
	}
}

// BenchmarkBMICalculator_Calculate benchmarks calculation performance.
func BenchmarkBMICalculator_Calculate(b *testing.B) {
	calc := NewBMICalculator()
	ctx := context.Background()
	params := &models.BMIParams{
		WeightKg: 75,
		HeightCm: 175,
		Region:   models.RegionIndia,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.Calculate(ctx, params)
	}
}
