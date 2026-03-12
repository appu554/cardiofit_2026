package baseline

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestTracker_Mean(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	tracker := NewTracker(nil, nil, log)

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "Simple mean",
			values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			expected: 3.0,
		},
		{
			name:     "Single value",
			values:   []float64{5.0},
			expected: 5.0,
		},
		{
			name:     "Empty slice",
			values:   []float64{},
			expected: 0.0,
		},
		{
			name:     "Decimal values",
			values:   []float64{1.5, 2.5, 3.5},
			expected: 2.5,
		},
		{
			name:     "Potassium range values",
			values:   []float64{3.8, 4.0, 4.2, 4.1, 3.9},
			expected: 4.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.mean(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestTracker_StdDev(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	tracker := NewTracker(nil, nil, log)

	tests := []struct {
		name     string
		values   []float64
		mean     float64
		expected float64
	}{
		{
			name:     "Standard deviation of 1,2,3,4,5",
			values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			mean:     3.0,
			expected: 1.5811, // sqrt(2.5)
		},
		{
			name:     "All same values",
			values:   []float64{4.0, 4.0, 4.0, 4.0},
			mean:     4.0,
			expected: 0.0,
		},
		{
			name:     "Single value",
			values:   []float64{5.0},
			mean:     5.0,
			expected: 0.0,
		},
		{
			name:     "Empty slice",
			values:   []float64{},
			mean:     0.0,
			expected: 0.0,
		},
		{
			name:     "Potassium normal variation",
			values:   []float64{3.8, 4.0, 4.2, 4.0, 4.0},
			mean:     4.0,
			expected: 0.1414, // Small variation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.stdDev(tt.values, tt.mean)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestTracker_MinMax(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	tracker := NewTracker(nil, nil, log)

	tests := []struct {
		name        string
		values      []float64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "Simple range",
			values:      []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			expectedMin: 1.0,
			expectedMax: 5.0,
		},
		{
			name:        "Single value",
			values:      []float64{5.0},
			expectedMin: 5.0,
			expectedMax: 5.0,
		},
		{
			name:        "Empty slice",
			values:      []float64{},
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
		{
			name:        "Negative values",
			values:      []float64{-3.0, -1.0, 0.0, 2.0},
			expectedMin: -3.0,
			expectedMax: 2.0,
		},
		{
			name:        "Unsorted input",
			values:      []float64{5.0, 1.0, 3.0, 2.0, 4.0},
			expectedMin: 1.0,
			expectedMax: 5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max := tracker.minMax(tt.values)
			assert.Equal(t, tt.expectedMin, min, "Min mismatch")
			assert.Equal(t, tt.expectedMax, max, "Max mismatch")
		})
	}
}

func TestTracker_RemoveOutliers(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	tracker := NewTracker(nil, nil, log)

	tests := []struct {
		name           string
		values         []float64
		expectedCount  int
		shouldContain  []float64
		shouldExclude  []float64
	}{
		{
			name:          "No outliers",
			values:        []float64{3.8, 4.0, 4.1, 3.9, 4.2, 4.0, 3.95, 4.05},
			expectedCount: 8,
			shouldContain: []float64{3.8, 4.2},
		},
		{
			name:          "With clear outlier",
			values:        []float64{4.0, 4.1, 4.0, 3.9, 4.0, 4.1, 10.0}, // 10.0 is outlier
			expectedCount: 6,
			shouldContain: []float64{4.0, 4.1, 3.9},
			shouldExclude: []float64{10.0},
		},
		{
			name:          "Too few values - no filtering",
			values:        []float64{1.0, 5.0, 10.0},
			expectedCount: 3,
		},
		{
			name:          "Potassium with panic value",
			values:        []float64{3.9, 4.0, 4.1, 4.0, 3.8, 4.2, 7.5}, // 7.5 is outlier
			expectedCount: 6,
			shouldExclude: []float64{7.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.removeOutliers(tt.values)
			assert.Equal(t, tt.expectedCount, len(result), "Result count mismatch")

			for _, v := range tt.shouldContain {
				assert.Contains(t, result, v, "Should contain %f", v)
			}

			for _, v := range tt.shouldExclude {
				assert.NotContains(t, result, v, "Should not contain %f", v)
			}
		})
	}
}

func TestTracker_FilterStablePeriod(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	tracker := NewTracker(nil, nil, log)

	tests := []struct {
		name          string
		values        []float64
		expectedCount int
	}{
		{
			name:          "All stable values",
			values:        []float64{4.0, 4.1, 3.9, 4.0, 4.2, 4.0, 3.95, 4.05},
			expectedCount: 8,
		},
		{
			name:          "With acute spike",
			values:        []float64{4.0, 4.1, 3.9, 4.0, 4.2, 8.0, 4.0, 3.95},
			expectedCount: 7, // 8.0 should be filtered
		},
		{
			name:          "Too few values - no filtering",
			values:        []float64{1.0, 5.0, 10.0, 15.0},
			expectedCount: 4,
		},
		{
			name:          "All same values",
			values:        []float64{4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0},
			expectedCount: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.filterStablePeriod(tt.values)
			assert.Equal(t, tt.expectedCount, len(result), "Result count mismatch")
		})
	}
}

func TestZScoreCalculation(t *testing.T) {
	// Test z-score calculation logic used in CompareToBaseline
	tests := []struct {
		name          string
		value         float64
		mean          float64
		stdDev        float64
		expectedZ     float64
		isSignificant bool
	}{
		{
			name:          "Value at mean",
			value:         4.0,
			mean:          4.0,
			stdDev:        0.2,
			expectedZ:     0.0,
			isSignificant: false,
		},
		{
			name:          "Value 1 SD above mean",
			value:         4.2,
			mean:          4.0,
			stdDev:        0.2,
			expectedZ:     1.0,
			isSignificant: false,
		},
		{
			name:          "Value 1.9 SD above mean (not significant)",
			value:         4.38,
			mean:          4.0,
			stdDev:        0.2,
			expectedZ:     1.9,
			isSignificant: false, // Below threshold
		},
		{
			name:          "Value >2 SD above mean (significant)",
			value:         4.5,
			mean:          4.0,
			stdDev:        0.2,
			expectedZ:     2.5,
			isSignificant: true,
		},
		{
			name:          "Value >2 SD below mean (significant)",
			value:         3.4,
			mean:          4.0,
			stdDev:        0.2,
			expectedZ:     -3.0,
			isSignificant: true,
		},
		{
			name:          "Zero stdDev",
			value:         5.0,
			mean:          4.0,
			stdDev:        0.0,
			expectedZ:     0.0,
			isSignificant: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var zScore float64
			if tt.stdDev > 0 {
				zScore = (tt.value - tt.mean) / tt.stdDev
			}

			isSignificant := zScore > 2.0 || zScore < -2.0

			assert.InDelta(t, tt.expectedZ, zScore, 0.001, "Z-score mismatch")
			assert.Equal(t, tt.isSignificant, isSignificant, "Significance mismatch")
		})
	}
}

func TestPercentDeviationCalculation(t *testing.T) {
	// Test percent deviation calculation
	tests := []struct {
		name             string
		value            float64
		mean             float64
		expectedDeviation float64
	}{
		{
			name:             "Value at mean",
			value:            4.0,
			mean:             4.0,
			expectedDeviation: 0.0,
		},
		{
			name:             "10% above mean",
			value:            4.4,
			mean:             4.0,
			expectedDeviation: 10.0,
		},
		{
			name:             "25% below mean",
			value:            3.0,
			mean:             4.0,
			expectedDeviation: -25.0,
		},
		{
			name:             "Zero mean",
			value:            5.0,
			mean:             0.0,
			expectedDeviation: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var deviation float64
			if tt.mean != 0 {
				deviation = ((tt.value - tt.mean) / tt.mean) * 100
			}

			assert.InDelta(t, tt.expectedDeviation, deviation, 0.001, "Percent deviation mismatch")
		})
	}
}
