package extraction

import (
	"testing"
)

// =============================================================================
// UNIT NORMALIZER TESTS
// =============================================================================

func TestNewUnitNormalizer(t *testing.T) {
	normalizer := NewUnitNormalizer()

	if normalizer == nil {
		t.Fatal("Expected normalizer to be created")
	}
}

// =============================================================================
// UNIT NORMALIZATION TESTS
// =============================================================================

func TestNormalizeUnit_RenalUnits(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"mL/min lowercase", "ml/min", "mL/min"},
		{"mL/min spaced", "ml / min", "mL/min"},
		{"mL/min with 1.73m2", "ml/min/1.73m2", "mL/min/1.73m²"},
		{"mL/min/1.73 m²", "ml/min/1.73 m²", "mL/min/1.73m²"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeUnit(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
		})
	}
}

func TestNormalizeUnit_DoseUnits(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"mg", "mg", "mg"},
		{"milligrams", "milligrams", "mg"},
		{"mg/kg", "mg/kg", "mg/kg"},
		{"mcg", "mcg", "mcg"},
		{"micrograms", "micrograms", "mcg"},
		{"µg", "µg", "mcg"},
		{"g", "g", "g"},
		{"grams", "grams", "g"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeUnit(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
		})
	}
}

func TestNormalizeUnit_PercentUnits(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"percent symbol", "%", "percent"},
		{"percent word", "percent", "percent"},
		{"pct abbreviation", "pct", "percent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeUnit(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
		})
	}
}

// =============================================================================
// VARIABLE NORMALIZATION TESTS
// =============================================================================

func TestNormalizeVariable_RenalVariables(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"crcl", "crcl", "renal_function.crcl"},
		{"creatinine clearance", "creatinine clearance", "renal_function.crcl"},
		{"clcr", "clcr", "renal_function.crcl"},
		{"egfr", "egfr", "renal_function.egfr"},
		{"gfr", "gfr", "renal_function.gfr"},
		{"renal function", "renal function", "renal_function.category"},
		{"kidney function", "kidney function", "renal_function.category"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeVariable(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
		})
	}
}

func TestNormalizeVariable_HepaticVariables(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"child-pugh", "child-pugh", "hepatic.child_pugh"},
		{"child pugh", "child pugh", "hepatic.child_pugh"},
		{"hepatic impairment", "hepatic impairment", "hepatic.impairment_level"},
		{"liver function", "liver function", "hepatic.function"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeVariable(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
		})
	}
}

func TestNormalizeVariable_PatientVariables(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"age", "age", "patient.age"},
		{"patient age", "patient age", "patient.age"},
		{"weight", "weight", "patient.weight"},
		{"body weight", "body weight", "patient.weight"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeVariable(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
		})
	}
}

// =============================================================================
// CHILD-PUGH NORMALIZATION TESTS
// =============================================================================

func TestNormalizeChildPugh(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name              string
		input             string
		expected          string
		expectConfidence  bool
	}{
		{"Child-Pugh A", "Child-Pugh A", "A", true},
		{"Child Pugh A", "Child Pugh A", "A", true},
		{"class a", "class a", "A", true},
		{"Child-Pugh B", "Child-Pugh B", "B", true},
		{"class b", "class b", "B", true},
		{"Child-Pugh C", "Child-Pugh C", "C", true},
		{"class c", "class c", "C", true},
		{"score 5-6", "score 5-6", "A", true},
		{"score 7-9", "score 7-9", "B", true},
		{"score 10-15", "score 10-15", "C", true},
		{"mild hepatic impairment", "mild hepatic impairment", "A", true},
		{"moderate hepatic impairment", "moderate hepatic impairment", "B", true},
		{"severe hepatic impairment", "severe hepatic impairment", "C", true},
		{"unknown text", "some random text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, confidence := normalizer.NormalizeChildPugh(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
			if tt.expectConfidence && confidence <= 0 {
				t.Errorf("Expected positive confidence for '%s'", tt.input)
			}
			if !tt.expectConfidence && confidence > 0 {
				t.Errorf("Expected zero confidence for '%s', got %f", tt.input, confidence)
			}
		})
	}
}

// =============================================================================
// GFR THRESHOLD PARSING TESTS
// =============================================================================

func TestParseGFRThreshold(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name        string
		input       string
		expectRange bool
		expectMin   float64
		expectMax   float64
		expectValue float64
		expectErr   bool
	}{
		{
			name:        "CrCl less than",
			input:       "CrCl < 30",
			expectRange: false,
			expectValue: 30,
		},
		{
			name:        "CrCl greater than",
			input:       "CrCl > 60",
			expectRange: false,
			expectValue: 60,
		},
		{
			name:        "eGFR greater or equal",
			input:       "eGFR >= 60",
			expectRange: false,
			expectValue: 60,
		},
		{
			name:        "GFR range with dash",
			input:       "GFR 30-60",
			expectRange: true,
			expectMin:   30,
			expectMax:   60,
		},
		{
			name:        "CrCl range with 'to'",
			input:       "CrCl 30 to 60",
			expectRange: true,
			expectMin:   30,
			expectMax:   60,
		},
		{
			name:      "No GFR found",
			input:     "some random text",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizer.ParseGFRThreshold(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectRange {
				if result.MinValue == nil || *result.MinValue != tt.expectMin {
					t.Errorf("Expected min value %f, got %v", tt.expectMin, result.MinValue)
				}
				if result.MaxValue == nil || *result.MaxValue != tt.expectMax {
					t.Errorf("Expected max value %f, got %v", tt.expectMax, result.MaxValue)
				}
			} else {
				if result.NumericValue == nil || *result.NumericValue != tt.expectValue {
					t.Errorf("Expected value %f, got %v", tt.expectValue, result.NumericValue)
				}
			}

			// All GFR values should have mL/min unit
			if result.Unit != "mL/min" {
				t.Errorf("Expected unit 'mL/min', got '%s'", result.Unit)
			}
		})
	}
}

func TestParseGFRRange(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name      string
		input     string
		expectMin float64
		expectMax float64
		expectErr bool
	}{
		{"Range with dash", "30-60", 30, 60, false},
		{"Range with 'to'", "30 to 60", 30, 60, false},
		{"Range with en-dash", "15–30", 15, 30, false},
		{"Decimals", "29.5-59.5", 29.5, 59.5, false},
		{"No range", "just text", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max, err := normalizer.ParseGFRRange(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if min != tt.expectMin {
				t.Errorf("Expected min %f, got %f", tt.expectMin, min)
			}
			if max != tt.expectMax {
				t.Errorf("Expected max %f, got %f", tt.expectMax, max)
			}
		})
	}
}

// =============================================================================
// RENAL CATEGORY TESTS
// =============================================================================

func TestGFRToCategory(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name     string
		gfr      float64
		expected RenalCategory
	}{
		{"Normal GFR 95", 95, RenalNormal},
		{"Normal GFR 90", 90, RenalNormal},
		{"Mild impairment 75", 75, RenalMild},
		{"Mild impairment 60", 60, RenalMild},
		{"Moderate impairment 45", 45, RenalModerate},
		{"Moderate impairment 30", 30, RenalModerate},
		{"Severe impairment 20", 20, RenalSevere},
		{"Severe impairment 15", 15, RenalSevere},
		{"Kidney failure 10", 10, RenalKidneyFailure},
		{"Kidney failure 5", 5, RenalKidneyFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.GFRToCategory(tt.gfr)
			if result != tt.expected {
				t.Errorf("Expected %s for GFR %f, got %s", tt.expected, tt.gfr, result)
			}
		})
	}
}

func TestCategoryToGFRRange(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		category  RenalCategory
		expectMin float64
		expectMax float64
	}{
		{RenalNormal, 90, 999},
		{RenalMild, 60, 89},
		{RenalModerate, 30, 59},
		{RenalSevere, 15, 29},
		{RenalKidneyFailure, 0, 14},
		{RenalESRD, 0, 14},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			min, max := normalizer.CategoryToGFRRange(tt.category)
			if min != tt.expectMin {
				t.Errorf("Expected min %f, got %f", tt.expectMin, min)
			}
			if max != tt.expectMax {
				t.Errorf("Expected max %f, got %f", tt.expectMax, max)
			}
		})
	}
}

func TestParseRenalCategory(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name             string
		input            string
		expected         RenalCategory
		expectConfidence bool
	}{
		{"Normal renal function", "normal renal function", RenalNormal, true},
		{"Mild renal impairment", "mild renal impairment", RenalMild, true},
		{"Moderate renal impairment", "moderate renal impairment", RenalModerate, true},
		{"Severe renal impairment", "severe renal impairment", RenalSevere, true},
		{"Dialysis", "on dialysis", RenalDialysis, true},
		{"ESRD", "ESRD", RenalESRD, true},
		{"End-stage renal disease", "end-stage renal disease", RenalESRD, true},
		{"Unknown text", "some random text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, confidence := normalizer.ParseRenalCategory(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
			if tt.expectConfidence && confidence <= 0 {
				t.Errorf("Expected positive confidence for '%s'", tt.input)
			}
			if !tt.expectConfidence && confidence > 0 {
				t.Errorf("Expected zero confidence for '%s', got %f", tt.input, confidence)
			}
		})
	}
}

// =============================================================================
// DOSE PARSING TESTS
// =============================================================================

func TestParseDose(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name        string
		input       string
		expectValue float64
		expectUnit  string
		expectErr   bool
	}{
		{"Simple mg", "500 mg", 500, "mg", false},
		{"Decimal mg", "2.5 mg", 2.5, "mg", false},
		{"mcg dose", "100 mcg", 100, "mcg", false},
		{"mL dose", "5 mL", 5, "mL", false},
		{"No dose", "take with food", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizer.ParseDose(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.NumericValue == nil || *result.NumericValue != tt.expectValue {
				t.Errorf("Expected value %f, got %v", tt.expectValue, result.NumericValue)
			}
			if result.Unit != tt.expectUnit {
				t.Errorf("Expected unit '%s', got '%s'", tt.expectUnit, result.Unit)
			}
		})
	}
}

func TestParsePercentage(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name        string
		input       string
		expectValue float64
		expectErr   bool
	}{
		{"Simple percent", "50%", 50, false},
		{"Decimal percent", "12.5%", 12.5, false},
		{"Percent with space", "75 %", 75, false},
		{"No percent", "fifty", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizer.ParsePercentage(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if *result != tt.expectValue {
				t.Errorf("Expected %f, got %f", tt.expectValue, *result)
			}
		})
	}
}

// =============================================================================
// FREQUENCY PARSING TESTS
// =============================================================================

func TestParseFrequency(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"once daily", "once daily", "daily"},
		{"qd", "qd", "daily"},
		{"daily", "daily", "daily"},
		{"twice daily", "twice daily", "BID"},
		{"bid", "bid", "BID"},
		{"three times daily", "three times daily", "TID"},
		{"tid", "tid", "TID"},
		{"four times daily", "four times daily", "QID"},
		{"qid", "qid", "QID"},
		{"q12h", "q12h", "Q12H"},
		{"every 12 hours", "every 12 hours", "every 12 hours"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.ParseFrequency(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s' for input '%s', got '%s'", tt.expected, tt.input, result)
			}
		})
	}
}

// =============================================================================
// NORMALIZED VALUE TESTS
// =============================================================================

func TestNormalizedValue_IsRange(t *testing.T) {
	min := 30.0
	max := 60.0
	value := 45.0

	rangeVal := &NormalizedValue{
		MinValue: &min,
		MaxValue: &max,
	}

	singleVal := &NormalizedValue{
		NumericValue: &value,
	}

	if !rangeVal.IsRange() {
		t.Error("Expected IsRange() to return true for range value")
	}

	if singleVal.IsRange() {
		t.Error("Expected IsRange() to return false for single value")
	}
}

// =============================================================================
// COMPREHENSIVE NORMALIZATION TESTS
// =============================================================================

func TestNormalizeConditionText(t *testing.T) {
	normalizer := NewUnitNormalizer()

	tests := []struct {
		name      string
		input     string
		expectVar string
		expectErr bool
	}{
		{"GFR threshold", "CrCl < 30", "renal_function.crcl", false},
		{"Child-Pugh class", "Child-Pugh B", "hepatic.child_pugh", false},
		{"Renal category", "mild renal impairment", "renal_function.category", false},
		{"Dose value", "500 mg", "dose", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizer.NormalizeConditionText(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Variable != tt.expectVar {
				t.Errorf("Expected variable '%s', got '%s'", tt.expectVar, result.Variable)
			}
		})
	}
}

// =============================================================================
// BENCHMARKS
// =============================================================================

func BenchmarkNormalizeUnit(b *testing.B) {
	normalizer := NewUnitNormalizer()

	units := []string{"mL/min", "mg", "mcg", "percent", "mg/kg"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, u := range units {
			normalizer.NormalizeUnit(u)
		}
	}
}

func BenchmarkNormalizeVariable(b *testing.B) {
	normalizer := NewUnitNormalizer()

	vars := []string{"crcl", "egfr", "child-pugh", "age", "weight"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range vars {
			normalizer.NormalizeVariable(v)
		}
	}
}

func BenchmarkParseGFRThreshold(b *testing.B) {
	normalizer := NewUnitNormalizer()

	thresholds := []string{"CrCl < 30", "eGFR >= 60", "GFR 30-60"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range thresholds {
			normalizer.ParseGFRThreshold(t)
		}
	}
}

func BenchmarkNormalizeChildPugh(b *testing.B) {
	normalizer := NewUnitNormalizer()

	inputs := []string{"Child-Pugh A", "class B", "severe hepatic impairment"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			normalizer.NormalizeChildPugh(input)
		}
	}
}
