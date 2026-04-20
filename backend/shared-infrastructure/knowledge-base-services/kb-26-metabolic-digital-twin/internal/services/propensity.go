package services

import (
	"errors"
	"math"
)

// PropensityModel is a minimal logistic regression fit via batch gradient descent.
// Sprint 1 deliberately avoids external ML libraries (gonum, etc.); the numerical
// requirements are modest and this keeps the dependency surface small. Sprint 2's
// Python service will replace this with a calibrated gradient-boosted tree
// (spec §6.1) behind the same propensity interface.
type PropensityModel struct {
	Intercept    float64
	Coefficients []float64
	FeatureKeys  []string
}

const (
	propensityEpochs    = 800
	propensityLearnRate = 0.05
	propensityClipAbs   = 30.0 // clip logits to avoid sigmoid overflow
)

// FitPropensity fits a logistic regression on features X (n×d) with binary labels y.
// featureKeys[i] names column i — Predict expects the same ordering in its input map.
func FitPropensity(X [][]float64, y []bool, featureKeys []string) (*PropensityModel, error) {
	n := len(X)
	if n == 0 || len(y) != n {
		return nil, errors.New("empty or mismatched training set")
	}
	d := len(featureKeys)
	if d == 0 {
		return nil, errors.New("no features")
	}
	w := make([]float64, d)
	var b float64
	for epoch := 0; epoch < propensityEpochs; epoch++ {
		dw := make([]float64, d)
		var db float64
		for i := 0; i < n; i++ {
			z := b
			for j := 0; j < d; j++ {
				z += w[j] * X[i][j]
			}
			if z > propensityClipAbs {
				z = propensityClipAbs
			} else if z < -propensityClipAbs {
				z = -propensityClipAbs
			}
			p := 1.0 / (1.0 + math.Exp(-z))
			var yi float64
			if y[i] {
				yi = 1.0
			}
			gerr := p - yi
			for j := 0; j < d; j++ {
				dw[j] += gerr * X[i][j]
			}
			db += gerr
		}
		inv := propensityLearnRate / float64(n)
		for j := 0; j < d; j++ {
			w[j] -= inv * dw[j]
		}
		b -= inv * db
	}
	return &PropensityModel{Intercept: b, Coefficients: w, FeatureKeys: featureKeys}, nil
}

// Predict returns propensity in [0,1] for the given feature map. Missing keys
// default to 0.0, mirroring the eligibility-predicate zero-default in Task 3.
func (m *PropensityModel) Predict(features map[string]float64) float64 {
	z := m.Intercept
	for i, k := range m.FeatureKeys {
		z += m.Coefficients[i] * features[k]
	}
	if z > propensityClipAbs {
		z = propensityClipAbs
	} else if z < -propensityClipAbs {
		z = -propensityClipAbs
	}
	return 1.0 / (1.0 + math.Exp(-z))
}
