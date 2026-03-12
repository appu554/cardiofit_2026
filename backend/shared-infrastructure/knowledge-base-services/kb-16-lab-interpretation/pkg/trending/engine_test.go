package trending

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"kb-16-lab-interpretation/pkg/types"
)

func TestEngine_Mean(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.mean(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestEngine_StdDev(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

	tests := []struct {
		name     string
		values   []float64
		mean     float64
		expected float64
	}{
		{
			name:     "Standard deviation",
			values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			mean:     3.0,
			expected: 1.5811,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.stdDev(tt.values, tt.mean)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestEngine_Median(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "Odd count",
			values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			expected: 3.0,
		},
		{
			name:     "Even count",
			values:   []float64{1.0, 2.0, 3.0, 4.0},
			expected: 2.5,
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
			name:     "Unsorted input",
			values:   []float64{5.0, 1.0, 3.0},
			expected: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.median(tt.values)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestEngine_LinearRegression(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

	baseTime := time.Now()

	tests := []struct {
		name           string
		points         []types.TrendDataPoint
		expectedSlope  float64
		expectedR2     float64
		slopeTolerance float64
		r2Tolerance    float64
	}{
		{
			name: "Perfect increasing trend",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
			},
			expectedSlope:  1.0, // 1 unit per day
			expectedR2:     1.0,
			slopeTolerance: 0.01,
			r2Tolerance:    0.01,
		},
		{
			name: "Perfect decreasing trend",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 4.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 3.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 1.0},
			},
			expectedSlope:  -1.0, // -1 unit per day
			expectedR2:     1.0,
			slopeTolerance: 0.01,
			r2Tolerance:    0.01,
		},
		{
			name: "Flat trend",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 4.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
			},
			expectedSlope:  0.0,
			expectedR2:     1.0,
			slopeTolerance: 0.01,
			r2Tolerance:    0.01,
		},
		{
			name: "Noisy increasing trend",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.5},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 2.8},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.2},
			},
			expectedSlope:  1.0,  // Approximately 1 per day
			expectedR2:     0.95, // Good fit but not perfect
			slopeTolerance: 0.2,
			r2Tolerance:    0.1,
		},
		{
			name: "Insufficient points",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
			},
			expectedSlope:  0.0,
			expectedR2:     0.0,
			slopeTolerance: 0.01,
			r2Tolerance:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slope, _, r2 := engine.linearRegression(tt.points)
			assert.InDelta(t, tt.expectedSlope, slope, tt.slopeTolerance, "Slope mismatch")
			assert.InDelta(t, tt.expectedR2, r2, tt.r2Tolerance, "R-squared mismatch")
		})
	}
}

func TestEngine_DetectTrajectory(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

	baseTime := time.Now()

	tests := []struct {
		name       string
		points     []types.TrendDataPoint
		expected   types.Trajectory
	}{
		{
			name: "Insufficient points",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 4.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 4.5},
			},
			expected: types.TrajectoryUnknown,
		},
		{
			name: "Stable trend",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 4.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 4.01},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.99},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
			},
			expected: types.TrajectoryStable,
		},
		{
			name: "Worsening trend (increasing values)",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 4.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 4.5},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 5.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 5.5},
				{Timestamp: baseTime.Add(96 * time.Hour), Value: 6.0},
			},
			expected: types.TrajectoryWorsening,
		},
		{
			name: "Volatile trend",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 2.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 8.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 7.0},
				{Timestamp: baseTime.Add(96 * time.Hour), Value: 2.5},
			},
			expected: types.TrajectoryVolatile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.detectTrajectory(tt.points)
			assert.Equal(t, tt.expected, result, "Trajectory mismatch")
		})
	}
}

func TestEngine_CalculateStatistics(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

	baseTime := time.Now()

	tests := []struct {
		name                string
		points              []types.TrendDataPoint
		expectedMean        float64
		expectedMin         float64
		expectedMax         float64
		expectedSampleCount int
	}{
		{
			name: "Basic statistics",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(96 * time.Hour), Value: 5.0},
			},
			expectedMean:        3.0,
			expectedMin:         1.0,
			expectedMax:         5.0,
			expectedSampleCount: 5,
		},
		{
			name:                "Empty points",
			points:              []types.TrendDataPoint{},
			expectedMean:        0.0,
			expectedMin:         0.0,
			expectedMax:         0.0,
			expectedSampleCount: 0,
		},
		{
			name: "Potassium values",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 3.8},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 4.2},
			},
			expectedMean:        4.0,
			expectedMin:         3.8,
			expectedMax:         4.2,
			expectedSampleCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := engine.calculateStatistics(tt.points)
			assert.InDelta(t, tt.expectedMean, stats.Mean, 0.001, "Mean mismatch")
			assert.InDelta(t, tt.expectedMin, stats.Min, 0.001, "Min mismatch")
			assert.InDelta(t, tt.expectedMax, stats.Max, 0.001, "Max mismatch")
			assert.Equal(t, tt.expectedSampleCount, stats.SampleCount, "Sample count mismatch")
		})
	}
}

func TestEngine_CalculateRateOfChange(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

	baseTime := time.Now()

	tests := []struct {
		name     string
		points   []types.TrendDataPoint
		expected float64
		tolerance float64
	}{
		{
			name: "1 unit per day increase",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
			},
			expected: 1.0,
			tolerance: 0.01,
		},
		{
			name: "1 unit per day decrease",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 3.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 1.0},
			},
			expected: -1.0,
			tolerance: 0.01,
		},
		{
			name: "No change",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 4.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 4.0},
			},
			expected: 0.0,
			tolerance: 0.01,
		},
		{
			name: "Single point",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 4.0},
			},
			expected: 0.0,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.calculateRateOfChange(tt.points)
			assert.InDelta(t, tt.expected, result, tt.tolerance, "Rate of change mismatch")
		})
	}
}

func TestEngine_PredictNextValue(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

	baseTime := time.Now()

	tests := []struct {
		name           string
		points         []types.TrendDataPoint
		expectNil      bool
		expectedValue  float64
		valueTolerance float64
	}{
		{
			name: "Insufficient points",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
			},
			expectNil: true,
		},
		{
			name: "Predict increasing trend",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
			},
			expectNil:      false,
			expectedValue:  11.0, // Day 10 (day 3 + 7 prediction days) with slope of 1
			valueTolerance: 0.5,
		},
		{
			name: "Predict flat trend",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 4.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
			},
			expectNil:      false,
			expectedValue:  4.0,
			valueTolerance: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.predictNextValue(tt.points)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.InDelta(t, tt.expectedValue, result.Value, tt.valueTolerance, "Predicted value mismatch")
				assert.Greater(t, result.Confidence, 0.0, "Confidence should be positive")
				assert.Equal(t, len(tt.points), result.BasedOnPoints, "BasedOnPoints mismatch")
			}
		})
	}
}

func TestStandardWindows(t *testing.T) {
	assert.Equal(t, 7, StandardWindows["7d"].Days)
	assert.Equal(t, 2, StandardWindows["7d"].MinPoints)

	assert.Equal(t, 30, StandardWindows["30d"].Days)
	assert.Equal(t, 3, StandardWindows["30d"].MinPoints)

	assert.Equal(t, 90, StandardWindows["90d"].Days)
	assert.Equal(t, 4, StandardWindows["90d"].MinPoints)

	assert.Equal(t, 365, StandardWindows["1yr"].Days)
	assert.Equal(t, 6, StandardWindows["1yr"].MinPoints)
}

// =============================================================================
// PREDICTION ENGINE TESTS
// =============================================================================

func TestPredictionEngine_PredictMultiHorizon(t *testing.T) {
	pe := NewPredictionEngine()
	baseTime := time.Now()

	tests := []struct {
		name            string
		points          []types.TrendDataPoint
		expectNil       bool
		expectHorizons  []string
		minConfidence   float64
	}{
		{
			name:      "Insufficient points",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
			},
			expectNil: true,
		},
		{
			name: "Perfect linear trend - all horizons",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(96 * time.Hour), Value: 5.0},
				{Timestamp: baseTime.Add(120 * time.Hour), Value: 6.0},
			},
			expectNil:      false,
			expectHorizons: []string{"7d", "14d", "30d"},
			minConfidence:  0.5,
		},
		{
			name: "4 points - only 7d horizon",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
			},
			expectNil:      false,
			expectHorizons: []string{"7d"},
			minConfidence:  0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pe.PredictMultiHorizon(tt.points)

			if tt.expectNil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, len(tt.points), result.BasedOnPoints)

			// Check expected horizons
			for _, horizon := range tt.expectHorizons {
				pred, exists := result.Predictions[horizon]
				assert.True(t, exists, "Expected horizon %s not found", horizon)
				if exists {
					assert.Greater(t, pred.Confidence, 0.0, "Confidence should be positive")
					assert.LessOrEqual(t, pred.Confidence, 1.0, "Confidence should be <= 1.0")
					assert.Less(t, pred.LowerBound, pred.Value, "Lower bound should be < value")
					assert.Greater(t, pred.UpperBound, pred.Value, "Upper bound should be > value")
				}
			}

			// Check trend strength
			assert.GreaterOrEqual(t, result.TrendStrength, 0.0)
			assert.LessOrEqual(t, result.TrendStrength, 1.0)
		})
	}
}

func TestPredictionEngine_ConfidenceIntervals(t *testing.T) {
	pe := NewPredictionEngine()
	baseTime := time.Now()

	// Create perfect linear data
	points := []types.TrendDataPoint{
		{Timestamp: baseTime, Value: 1.0},
		{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
		{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
		{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
		{Timestamp: baseTime.Add(96 * time.Hour), Value: 5.0},
		{Timestamp: baseTime.Add(120 * time.Hour), Value: 6.0},
	}

	result := pe.PredictMultiHorizon(points)
	assert.NotNil(t, result)

	// For perfect linear data, confidence intervals should be narrow
	pred7d := result.Predictions["7d"]
	pred30d := result.Predictions["30d"]

	assert.NotNil(t, pred7d)
	assert.NotNil(t, pred30d)

	// 30d interval should be wider than 7d (more uncertainty with longer horizon)
	interval7d := pred7d.UpperBound - pred7d.LowerBound
	interval30d := pred30d.UpperBound - pred30d.LowerBound
	assert.Greater(t, interval30d, interval7d, "30d interval should be wider than 7d")

	// 30d confidence should be lower than 7d
	assert.Less(t, pred30d.Confidence, pred7d.Confidence, "30d confidence should be lower")
}

func TestPredictionEngine_AccelerationAnalysis(t *testing.T) {
	pe := NewPredictionEngine()
	baseTime := time.Now()

	tests := []struct {
		name                string
		points              []types.TrendDataPoint
		expectNil           bool
		expectedAccelerating bool
		expectedDecelerating bool
	}{
		{
			name: "Insufficient points for acceleration",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
			},
			expectNil: true,
		},
		{
			name: "Constant rate (linear trend)",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 2.0},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 3.0},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(96 * time.Hour), Value: 5.0},
				{Timestamp: baseTime.Add(120 * time.Hour), Value: 6.0},
			},
			expectNil:            false,
			expectedAccelerating: false,
			expectedDecelerating: false,
		},
		{
			name: "Accelerating trend (quadratic-like)",
			points: []types.TrendDataPoint{
				{Timestamp: baseTime, Value: 1.0},
				{Timestamp: baseTime.Add(24 * time.Hour), Value: 1.5},
				{Timestamp: baseTime.Add(48 * time.Hour), Value: 2.5},
				{Timestamp: baseTime.Add(72 * time.Hour), Value: 4.0},
				{Timestamp: baseTime.Add(96 * time.Hour), Value: 6.0},
				{Timestamp: baseTime.Add(120 * time.Hour), Value: 8.5},
			},
			expectNil:            false,
			expectedAccelerating: true,
			expectedDecelerating: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pe.PredictMultiHorizon(tt.points)

			if tt.expectNil {
				if result != nil {
					assert.Nil(t, result.Acceleration)
				}
				return
			}

			assert.NotNil(t, result)
			assert.NotNil(t, result.Acceleration)

			if tt.expectedAccelerating {
				assert.True(t, result.Acceleration.IsAccelerating, "Expected accelerating trend")
			}
			if tt.expectedDecelerating {
				assert.True(t, result.Acceleration.IsDecelerating, "Expected decelerating trend")
			}
		})
	}
}

// =============================================================================
// CLINICAL CONTEXT TESTS
// =============================================================================

func TestLabContextDatabase_Coverage(t *testing.T) {
	// Test that common lab codes have context
	expectedCodes := []string{
		"2160-0",  // Creatinine
		"2823-3",  // Potassium
		"718-7",   // Hemoglobin
		"2345-7",  // Glucose
		"1742-6",  // ALT
		"30934-4", // BNP
	}

	for _, code := range expectedCodes {
		ctx, found := GetLabContext(code)
		assert.True(t, found, "Expected context for code %s", code)
		assert.NotNil(t, ctx)
		assert.NotEmpty(t, ctx.Name)
		assert.NotEmpty(t, ctx.Direction)
		assert.Greater(t, ctx.ClinicalSignificance, 0.0)
	}
}

func TestInterpretTrajectory_ContextAware(t *testing.T) {
	tests := []struct {
		name               string
		labCode            string
		trajectory         types.Trajectory
		slope              float64
		expectedMeaning    string
	}{
		{
			name:            "Creatinine rising (bad)",
			labCode:         "2160-0", // Creatinine - lower is better
			trajectory:      types.TrajectoryWorsening,
			slope:           0.1,
			expectedMeaning: "WORSENING",
		},
		{
			name:            "Creatinine falling (good)",
			labCode:         "2160-0",
			trajectory:      types.TrajectoryImproving,
			slope:           -0.1,
			expectedMeaning: "IMPROVING",
		},
		{
			name:            "Hemoglobin rising (good)",
			labCode:         "718-7", // Hemoglobin - higher is better
			trajectory:      types.TrajectoryImproving,
			slope:           0.5,
			expectedMeaning: "IMPROVING",
		},
		{
			name:            "Hemoglobin falling (bad)",
			labCode:         "718-7",
			trajectory:      types.TrajectoryWorsening,
			slope:           -0.5,
			expectedMeaning: "WORSENING",
		},
		{
			name:            "Potassium (mid-optimal) trending",
			labCode:         "2823-3", // Potassium - mid is optimal
			trajectory:      types.TrajectoryWorsening,
			slope:           0.1,
			expectedMeaning: "MONITOR",
		},
		{
			name:            "Volatile trend",
			labCode:         "2345-7", // Glucose
			trajectory:      types.TrajectoryVolatile,
			slope:           0.0,
			expectedMeaning: "CONCERNING",
		},
		{
			name:            "Unknown lab code (default behavior)",
			labCode:         "UNKNOWN-CODE",
			trajectory:      types.TrajectoryWorsening,
			slope:           0.1,
			expectedMeaning: "WORSENING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := InterpretTrajectory(tt.labCode, tt.trajectory, tt.slope)
			assert.Equal(t, tt.expectedMeaning, interp.ClinicalMeaning)
			assert.NotEmpty(t, interp.Urgency)
			assert.NotEmpty(t, interp.Explanation)
		})
	}
}

func TestIsClinicallySignificant(t *testing.T) {
	tests := []struct {
		name     string
		labCode  string
		change   float64
		expected bool
	}{
		{
			name:     "Creatinine significant change",
			labCode:  "2160-0",
			change:   0.3, // MinClinicalChange is 0.2
			expected: true,
		},
		{
			name:     "Creatinine insignificant change",
			labCode:  "2160-0",
			change:   0.1, // Below 0.2 threshold
			expected: false,
		},
		{
			name:     "Hemoglobin significant change",
			labCode:  "718-7",
			change:   0.6, // MinClinicalChange is 0.5
			expected: true,
		},
		{
			name:     "Unknown code - any change significant",
			labCode:  "UNKNOWN",
			change:   0.1,
			expected: true, // Default behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsClinicallySignificant(tt.labCode, tt.change)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetVolatilityThreshold(t *testing.T) {
	// Known lab codes should have their specific thresholds
	creatinineThreshold := GetVolatilityThreshold("2160-0")
	assert.Equal(t, 0.25, creatinineThreshold) // 25% for creatinine

	potassiumThreshold := GetVolatilityThreshold("2823-3")
	assert.Equal(t, 0.15, potassiumThreshold) // 15% for potassium

	// Unknown code should return default
	unknownThreshold := GetVolatilityThreshold("UNKNOWN")
	assert.Equal(t, 0.3, unknownThreshold) // 30% default
}

// =============================================================================
// ENHANCED ANALYSIS TESTS
// =============================================================================

func TestEngine_PopulateTrendWindows(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	engine := NewEngine(nil, log)

	now := time.Now()

	// Create analysis with data points spanning 100 days
	analysis := &types.TrendAnalysis{
		TestCode:   "2160-0",
		PatientID:  "test-patient",
		DataPoints: []types.TrendDataPoint{},
	}

	// Add points over 100 days
	for i := 0; i <= 100; i += 5 {
		analysis.DataPoints = append(analysis.DataPoints, types.TrendDataPoint{
			Timestamp: now.AddDate(0, 0, -100+i),
			Value:     1.0 + float64(i)*0.01, // Gradually increasing
		})
	}

	// Populate windows
	engine.PopulateTrendWindows(analysis)

	// Should have multiple windows populated
	assert.NotNil(t, analysis.Windows)
	assert.Greater(t, len(analysis.Windows), 0, "Should have at least one window")

	// Check that each window has proper statistics
	for windowKey, window := range analysis.Windows {
		t.Run("Window_"+windowKey, func(t *testing.T) {
			assert.NotEmpty(t, window.Name)
			assert.Greater(t, window.Days, 0)
			assert.Greater(t, len(window.DataPoints), 0)
			assert.NotNil(t, window.Statistics)
			assert.Greater(t, window.Statistics.Count, 0)
			assert.NotEmpty(t, window.Trend) // Should be "increasing", "decreasing", or "stable"
		})
	}
}

func TestTrendDirectionConstants(t *testing.T) {
	// Verify direction constants are properly defined
	assert.Equal(t, TrendDirection("HIGHER_BETTER"), DirectionHigherBetter)
	assert.Equal(t, TrendDirection("LOWER_BETTER"), DirectionLowerBetter)
	assert.Equal(t, TrendDirection("MID_OPTIMAL"), DirectionMidOptimal)
	assert.Equal(t, TrendDirection("CONTEXTUAL"), DirectionContextual)
}

func TestLabContextDatabase_DirectionMappings(t *testing.T) {
	// Verify correct direction assignments for key lab tests
	tests := []struct {
		code              string
		expectedDirection TrendDirection
		description       string
	}{
		{"2160-0", DirectionLowerBetter, "Creatinine - lower is better"},
		{"718-7", DirectionHigherBetter, "Hemoglobin - higher is better"},
		{"2823-3", DirectionMidOptimal, "Potassium - mid is optimal"},
		{"6690-2", DirectionContextual, "WBC - context dependent"},
		{"2085-9", DirectionHigherBetter, "HDL - higher is better"},
		{"13457-7", DirectionLowerBetter, "LDL - lower is better"},
		{"1751-7", DirectionHigherBetter, "Albumin - higher is better"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			ctx, found := GetLabContext(tt.code)
			assert.True(t, found, "Context should exist for %s", tt.code)
			assert.Equal(t, tt.expectedDirection, ctx.Direction, "Direction mismatch for %s", tt.code)
		})
	}
}

func TestWeightedPrediction(t *testing.T) {
	pe := NewPredictionEngine()
	baseTime := time.Now()

	// Create data with recent trend change
	points := []types.TrendDataPoint{
		{Timestamp: baseTime.AddDate(0, 0, -30), Value: 5.0},
		{Timestamp: baseTime.AddDate(0, 0, -25), Value: 5.1},
		{Timestamp: baseTime.AddDate(0, 0, -20), Value: 5.2},
		// Recent upturn
		{Timestamp: baseTime.AddDate(0, 0, -10), Value: 6.0},
		{Timestamp: baseTime.AddDate(0, 0, -5), Value: 6.5},
		{Timestamp: baseTime, Value: 7.0},
	}

	horizon := PredictionHorizon{Name: "7d", Days: 7, MinData: 4}

	// Weighted prediction should give more importance to recent trend
	prediction := pe.WeightedPrediction(points, horizon)

	assert.NotNil(t, prediction)
	assert.Equal(t, "weighted_linear", prediction.Method)
	assert.Greater(t, prediction.Value, 7.0, "Prediction should extrapolate the recent upward trend")
	assert.Greater(t, prediction.Confidence, 0.0)
	assert.Less(t, prediction.LowerBound, prediction.Value)
	assert.Greater(t, prediction.UpperBound, prediction.Value)
}
