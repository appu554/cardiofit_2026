package services

import (
	"math"
	"testing"
)

func TestExpressionEvaluator_Arithmetic(t *testing.T) {
	ev := NewExpressionEvaluator()
	fields := map[string]float64{
		"sbp_nocturnal_mean": 130,
		"sbp_daytime_mean":   140,
	}
	result, err := ev.EvaluateNumeric("(sbp_nocturnal_mean - sbp_daytime_mean) / sbp_daytime_mean", fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := (130.0 - 140.0) / 140.0
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestExpressionEvaluator_Comparison(t *testing.T) {
	ev := NewExpressionEvaluator()

	// dipping_ratio > 0.0 with dipping_ratio = -0.05 → false
	falseResult, err := ev.EvaluateBool("dipping_ratio > 0.0", map[string]float64{"dipping_ratio": -0.05})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if falseResult != false {
		t.Errorf("expected false, got %v", falseResult)
	}

	// dipping_ratio > 0.0 with dipping_ratio = 0.05 → true
	trueResult, err := ev.EvaluateBool("dipping_ratio > 0.0", map[string]float64{"dipping_ratio": 0.05})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trueResult != true {
		t.Errorf("expected true, got %v", trueResult)
	}
}

func TestExpressionEvaluator_LogicalAND(t *testing.T) {
	ev := NewExpressionEvaluator()
	fields := map[string]float64{
		"rate_of_change": -0.10,
		"twin_state.IS":  0.25,
	}
	result, err := ev.EvaluateBool("rate_of_change < -0.08 AND twin_state.IS < 0.30", fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}

	// AND where one side is false
	fields2 := map[string]float64{
		"rate_of_change": -0.05,
		"twin_state.IS":  0.25,
	}
	result2, err := ev.EvaluateBool("rate_of_change < -0.08 AND twin_state.IS < 0.30", fields2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != false {
		t.Errorf("expected false, got %v", result2)
	}
}

func TestExpressionEvaluator_LogicalOR(t *testing.T) {
	ev := NewExpressionEvaluator()

	// a > 1 OR b > 1 with a=0.5, b=2.0 → true
	result, err := ev.EvaluateBool("a > 1 OR b > 1", map[string]float64{"a": 0.5, "b": 2.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}

	// both false
	result2, err := ev.EvaluateBool("a > 1 OR b > 1", map[string]float64{"a": 0.5, "b": 0.5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != false {
		t.Errorf("expected false, got %v", result2)
	}
}

func TestExpressionEvaluator_RejectsDisallowed(t *testing.T) {
	ev := NewExpressionEvaluator()

	// Non-whitelisted function call
	_, err := ev.EvaluateNumeric("os.Exit(1)", map[string]float64{})
	if err == nil {
		t.Error("expected error for os.Exit(1), got nil")
	}

	// import keyword
	_, err2 := ev.EvaluateNumeric("import fmt", map[string]float64{})
	if err2 == nil {
		t.Error("expected error for 'import fmt', got nil")
	}

	// another non-whitelisted function
	_, err3 := ev.EvaluateNumeric("sqrt(4)", map[string]float64{})
	if err3 == nil {
		t.Error("expected error for non-whitelisted sqrt(), got nil")
	}
}

func TestExpressionEvaluator_FieldSubstitution(t *testing.T) {
	ev := NewExpressionEvaluator()
	fields := map[string]float64{
		"sbp_home_mean":  150,
		"bp_target_sbp":  130,
	}
	result, err := ev.EvaluateNumeric("sbp_home_mean - bp_target_sbp", fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result-20.0) > 1e-9 {
		t.Errorf("expected 20.0, got %v", result)
	}
}

func TestExpressionEvaluator_BuiltinNormalize(t *testing.T) {
	ev := NewExpressionEvaluator()

	// normalize(hba1c, 6, 12) with hba1c=9 → (9-6)/(12-6) = 0.5
	result, err := ev.EvaluateNumeric("normalize(hba1c, 6, 12)", map[string]float64{"hba1c": 9})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result-0.5) > 1e-9 {
		t.Errorf("expected 0.5, got %v", result)
	}

	// Clamps to [0, 1]: normalize(hba1c, 6, 12) with hba1c=13 → 1.0
	result2, err := ev.EvaluateNumeric("normalize(hba1c, 6, 12)", map[string]float64{"hba1c": 13})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result2-1.0) > 1e-9 {
		t.Errorf("expected 1.0 (clamped), got %v", result2)
	}

	// Clamps below 0: normalize(hba1c, 6, 12) with hba1c=4 → 0.0
	result3, err := ev.EvaluateNumeric("normalize(hba1c, 6, 12)", map[string]float64{"hba1c": 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result3-0.0) > 1e-9 {
		t.Errorf("expected 0.0 (clamped), got %v", result3)
	}

	// normalize in a compound formula like MD node uses
	fields := map[string]float64{
		"hba1c": 9,
		"fbg":   180,
		"pm04":  0.6,
		"pm05":  0.4,
	}
	result4, err := ev.EvaluateNumeric("0.35*normalize(hba1c,6,12) + 0.30*normalize(fbg,90,250) + 0.20*pm04 + 0.15*pm05", fields)
	if err != nil {
		t.Fatalf("unexpected error for compound formula: %v", err)
	}
	// 0.35*(9-6)/(12-6) + 0.30*(180-90)/(250-90) + 0.20*0.6 + 0.15*0.4
	// = 0.35*0.5 + 0.30*0.5625 + 0.12 + 0.06
	// = 0.175 + 0.16875 + 0.12 + 0.06 = 0.52375
	expected := 0.35*0.5 + 0.30*(180.0-90.0)/(250.0-90.0) + 0.20*0.6 + 0.15*0.4
	if math.Abs(result4-expected) > 1e-9 {
		t.Errorf("expected %v, got %v", expected, result4)
	}
}

func TestExpressionEvaluator_BuiltinAbs(t *testing.T) {
	ev := NewExpressionEvaluator()

	// abs(slope) with slope=-0.05 → 0.05
	result, err := ev.EvaluateNumeric("abs(slope)", map[string]float64{"slope": -0.05})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result-0.05) > 1e-9 {
		t.Errorf("expected 0.05, got %v", result)
	}

	// abs of positive stays positive
	result2, err := ev.EvaluateNumeric("abs(x)", map[string]float64{"x": 3.14})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result2-3.14) > 1e-9 {
		t.Errorf("expected 3.14, got %v", result2)
	}
}

func TestExpressionEvaluator_OperatorPrecedence(t *testing.T) {
	ev := NewExpressionEvaluator()

	// 2 + 3 * 4 = 14 (not 20)
	result, err := ev.EvaluateNumeric("2 + 3 * 4", map[string]float64{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result-14.0) > 1e-9 {
		t.Errorf("expected 14.0, got %v", result)
	}

	// (2 + 3) * 4 = 20
	result2, err := ev.EvaluateNumeric("(2 + 3) * 4", map[string]float64{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result2-20.0) > 1e-9 {
		t.Errorf("expected 20.0, got %v", result2)
	}
}

func TestExpressionEvaluator_ComparisonOperators(t *testing.T) {
	ev := NewExpressionEvaluator()
	fields := map[string]float64{"x": 5.0}

	tests := []struct {
		expr     string
		expected bool
	}{
		{"x >= 5", true},
		{"x >= 6", false},
		{"x <= 5", true},
		{"x <= 4", false},
		{"x == 5", true},
		{"x == 6", false},
		{"x != 5", false},
		{"x != 6", true},
		{"x < 6", true},
		{"x < 5", false},
		{"x > 4", true},
		{"x > 5", false},
	}

	for _, tt := range tests {
		result, err := ev.EvaluateBool(tt.expr, fields)
		if err != nil {
			t.Errorf("expr %q: unexpected error: %v", tt.expr, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("expr %q: expected %v, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestExpressionEvaluator_DottedFieldNames(t *testing.T) {
	ev := NewExpressionEvaluator()
	fields := map[string]float64{
		"twin_state.IS":       0.25,
		"twin_state.severity": 0.8,
	}

	result, err := ev.EvaluateBool("twin_state.IS < 0.30 AND twin_state.severity > 0.5", fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestExpressionEvaluator_UnaryMinus(t *testing.T) {
	ev := NewExpressionEvaluator()

	result, err := ev.EvaluateNumeric("-5 + 3", map[string]float64{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(result-(-2.0)) > 1e-9 {
		t.Errorf("expected -2.0, got %v", result)
	}
}

func TestExpressionEvaluator_MissingField(t *testing.T) {
	ev := NewExpressionEvaluator()

	_, err := ev.EvaluateNumeric("nonexistent_field + 1", map[string]float64{})
	if err == nil {
		t.Error("expected error for missing field, got nil")
	}
}

func TestExpressionEvaluator_DivisionByZero(t *testing.T) {
	ev := NewExpressionEvaluator()

	_, err := ev.EvaluateNumeric("5 / 0", map[string]float64{})
	if err == nil {
		t.Error("expected error for division by zero, got nil")
	}
}

func TestExpressionEvaluator_RejectsGoKeywords(t *testing.T) {
	ev := NewExpressionEvaluator()

	keywords := []string{"func", "go", "return"}
	for _, kw := range keywords {
		_, err := ev.EvaluateNumeric(kw+" something", map[string]float64{})
		if err == nil {
			t.Errorf("expected error for keyword %q, got nil", kw)
		}
	}
}
