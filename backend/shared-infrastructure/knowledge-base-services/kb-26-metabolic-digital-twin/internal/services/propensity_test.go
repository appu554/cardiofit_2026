package services

import (
	"math"
	"testing"
)

func TestPropensityModel_Fit_PredictsKnownSeparableData(t *testing.T) {
	// Synthetic data where treated when feature > 0 with some noise.
	X := [][]float64{
		{1.0}, {1.2}, {0.9}, {1.5}, {0.8}, {1.1},
		{-1.0}, {-0.5}, {-1.2}, {-0.9}, {-1.5}, {-0.7},
	}
	y := []bool{true, true, true, true, true, true, false, false, false, false, false, false}
	m, err := FitPropensity(X, y, []string{"x"})
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	if p := m.Predict(map[string]float64{"x": 1.0}); p < 0.7 {
		t.Fatalf("want high propensity at x=1.0, got %.3f", p)
	}
	if p := m.Predict(map[string]float64{"x": -1.0}); p > 0.3 {
		t.Fatalf("want low propensity at x=-1.0, got %.3f", p)
	}
}

func TestPropensityModel_Predict_AlwaysIn01(t *testing.T) {
	X := [][]float64{{0}, {1}, {2}, {3}}
	y := []bool{false, false, true, true}
	m, err := FitPropensity(X, y, []string{"x"})
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	for x := -10.0; x <= 10.0; x += 0.5 {
		p := m.Predict(map[string]float64{"x": x})
		if p < 0 || p > 1 || math.IsNaN(p) {
			t.Fatalf("propensity out of [0,1]: %.3f at x=%.1f", p, x)
		}
	}
	// Extreme values must also stay in [0,1] via logit clip.
	for _, extreme := range []float64{-1e9, -1e6, 1e6, 1e9} {
		p := m.Predict(map[string]float64{"x": extreme})
		if p < 0 || p > 1 || math.IsNaN(p) {
			t.Fatalf("propensity out of [0,1] at extreme x=%.0e: %.6f", extreme, p)
		}
	}
}

func TestPropensityModel_Fit_RejectsEmptyTrainingSet(t *testing.T) {
	if _, err := FitPropensity(nil, nil, []string{"x"}); err == nil {
		t.Fatal("expected error for empty training set")
	}
}

func TestPropensityModel_Fit_RejectsMismatchedLengths(t *testing.T) {
	X := [][]float64{{1.0}, {2.0}, {3.0}, {4.0}, {5.0}}
	y := []bool{true, false, true} // 3 labels for 5 rows
	if _, err := FitPropensity(X, y, []string{"x"}); err == nil {
		t.Fatal("expected error for mismatched X and y lengths")
	}
}

func TestPropensityModel_Fit_RejectsFeatureKeyColumnMismatch(t *testing.T) {
	X := [][]float64{{1.0, 2.0}, {3.0, 4.0}} // 2 columns
	y := []bool{true, false}
	if _, err := FitPropensity(X, y, []string{"x"}); err == nil {
		t.Fatal("expected error when featureKeys count (1) doesn't match column count (2)")
	}
}
