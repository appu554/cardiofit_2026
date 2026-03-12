// Package trending provides multi-window trend analysis for lab results
package trending

import (
	"math"
	"time"

	"kb-16-lab-interpretation/pkg/types"
)

// =============================================================================
// MULTI-HORIZON PREDICTION ENGINE
// =============================================================================

// PredictionHorizon defines standard prediction time horizons
type PredictionHorizon struct {
	Name    string
	Days    int
	MinData int // Minimum data points required
}

// Standard prediction horizons
var StandardHorizons = []PredictionHorizon{
	{Name: "7d", Days: 7, MinData: 4},
	{Name: "14d", Days: 14, MinData: 5},
	{Name: "30d", Days: 30, MinData: 6},
}

// MultiHorizonPrediction contains predictions across multiple time horizons
type MultiHorizonPrediction struct {
	Predictions    map[string]*EnhancedPrediction `json:"predictions"`
	Acceleration   *AccelerationAnalysis          `json:"acceleration,omitempty"`
	TrendStrength  float64                        `json:"trend_strength"` // 0-1, how reliable the trend is
	BasedOnPoints  int                            `json:"based_on_points"`
	AnalyzedAt     time.Time                      `json:"analyzed_at"`
}

// EnhancedPrediction extends PredictedValue with confidence intervals
type EnhancedPrediction struct {
	Value           float64   `json:"value"`
	LowerBound      float64   `json:"lower_bound"`      // 95% confidence lower
	UpperBound      float64   `json:"upper_bound"`      // 95% confidence upper
	Confidence      float64   `json:"confidence"`       // 0-1 confidence score
	PredictedAt     time.Time `json:"predicted_at"`
	HorizonDays     int       `json:"horizon_days"`
	Method          string    `json:"method"`           // linear, weighted_linear
	UncertaintyGrowth float64 `json:"uncertainty_growth"` // How uncertainty increases per day
}

// AccelerationAnalysis measures if rate of change is accelerating or decelerating
type AccelerationAnalysis struct {
	Acceleration      float64 `json:"acceleration"`      // Second derivative (units/day²)
	IsAccelerating    bool    `json:"is_accelerating"`   // Rate of change increasing
	IsDecelerating    bool    `json:"is_decelerating"`   // Rate of change decreasing
	AccelerationTrend string  `json:"acceleration_trend"` // ACCELERATING, DECELERATING, CONSTANT
	Significance      float64 `json:"significance"`       // Statistical significance 0-1
}

// PredictionEngine provides advanced prediction capabilities
type PredictionEngine struct {
	minPoints      int
	maxExtrapolationDays int
}

// NewPredictionEngine creates a new prediction engine
func NewPredictionEngine() *PredictionEngine {
	return &PredictionEngine{
		minPoints:            4,
		maxExtrapolationDays: 30,
	}
}

// PredictMultiHorizon generates predictions for all standard horizons
func (pe *PredictionEngine) PredictMultiHorizon(points []types.TrendDataPoint) *MultiHorizonPrediction {
	if len(points) < pe.minPoints {
		return nil
	}

	result := &MultiHorizonPrediction{
		Predictions:   make(map[string]*EnhancedPrediction),
		BasedOnPoints: len(points),
		AnalyzedAt:    time.Now(),
	}

	// Calculate regression parameters
	slope, intercept, rSquared := linearRegression(points)
	stdError := pe.calculateStandardError(points, slope, intercept)

	// Calculate trend strength based on R² and sample size
	result.TrendStrength = pe.calculateTrendStrength(rSquared, len(points))

	// Generate predictions for each horizon
	lastPoint := points[len(points)-1]
	firstTime := points[0].Timestamp

	for _, horizon := range StandardHorizons {
		if len(points) >= horizon.MinData && horizon.Days <= pe.maxExtrapolationDays {
			predTime := lastPoint.Timestamp.Add(time.Duration(horizon.Days) * 24 * time.Hour)
			daysFromFirst := predTime.Sub(firstTime).Hours() / 24

			prediction := pe.generateEnhancedPrediction(
				slope, intercept, stdError, rSquared,
				daysFromFirst, horizon, len(points), predTime,
			)
			result.Predictions[horizon.Name] = prediction
		}
	}

	// Calculate acceleration
	result.Acceleration = pe.calculateAcceleration(points)

	return result
}

// generateEnhancedPrediction creates a prediction with confidence intervals
func (pe *PredictionEngine) generateEnhancedPrediction(
	slope, intercept, stdError, rSquared, daysFromFirst float64,
	horizon PredictionHorizon, sampleSize int, predTime time.Time,
) *EnhancedPrediction {

	predictedValue := intercept + (slope * daysFromFirst)

	// Calculate prediction interval (95% confidence)
	// Uses t-distribution approximation for small samples
	tValue := pe.tCriticalValue(sampleSize - 2) // df = n - 2

	// Uncertainty grows with extrapolation distance
	extrapolationFactor := math.Sqrt(1 + 1/float64(sampleSize) +
		math.Pow(daysFromFirst-float64(sampleSize)/2, 2)/float64(sampleSize))

	marginOfError := tValue * stdError * extrapolationFactor

	// Apply minimum uncertainty floor (1% of predicted value or 0.01, whichever is larger)
	// This handles perfectly linear data where stdError is 0
	minMargin := math.Max(math.Abs(predictedValue)*0.01, 0.01)
	if marginOfError < minMargin {
		marginOfError = minMargin * float64(horizon.Days) / 7.0 // Scale with horizon
	}

	// Uncertainty growth per day (for visualization)
	uncertaintyGrowth := (tValue * stdError) / float64(horizon.Days)

	// Confidence score based on R², sample size, and extrapolation distance
	confidence := rSquared * math.Min(1.0, float64(sampleSize)/10.0) *
		(1 - float64(horizon.Days)/60.0) // Decrease confidence for longer horizons

	return &EnhancedPrediction{
		Value:             predictedValue,
		LowerBound:        predictedValue - marginOfError,
		UpperBound:        predictedValue + marginOfError,
		Confidence:        math.Max(0, math.Min(1, confidence)),
		PredictedAt:       predTime,
		HorizonDays:       horizon.Days,
		Method:            "linear",
		UncertaintyGrowth: uncertaintyGrowth,
	}
}

// calculateStandardError computes residual standard error for predictions
func (pe *PredictionEngine) calculateStandardError(points []types.TrendDataPoint, slope, intercept float64) float64 {
	if len(points) < 3 {
		return 0
	}

	firstTime := points[0].Timestamp
	sumSquaredResiduals := 0.0

	for _, p := range points {
		days := p.Timestamp.Sub(firstTime).Hours() / 24
		predicted := intercept + (slope * days)
		residual := p.Value - predicted
		sumSquaredResiduals += residual * residual
	}

	// Standard error with degrees of freedom correction
	return math.Sqrt(sumSquaredResiduals / float64(len(points)-2))
}

// calculateTrendStrength computes how reliable the trend is (0-1)
func (pe *PredictionEngine) calculateTrendStrength(rSquared float64, sampleSize int) float64 {
	// Adjust R² for sample size
	adjustedRSquared := 1 - (1-rSquared)*float64(sampleSize-1)/float64(sampleSize-2)

	// Combine with sample size factor
	sizeFactor := math.Min(1.0, float64(sampleSize)/8.0)

	return math.Max(0, math.Min(1, adjustedRSquared*sizeFactor))
}

// calculateAcceleration determines if the rate of change is accelerating
func (pe *PredictionEngine) calculateAcceleration(points []types.TrendDataPoint) *AccelerationAnalysis {
	if len(points) < 5 {
		return nil
	}

	// Split data into two halves and compare slopes
	mid := len(points) / 2
	firstHalf := points[:mid]
	secondHalf := points[mid:]

	slope1, _, r1 := linearRegression(firstHalf)
	slope2, _, r2 := linearRegression(secondHalf)

	// Calculate acceleration (change in slope)
	firstDuration := firstHalf[len(firstHalf)-1].Timestamp.Sub(firstHalf[0].Timestamp).Hours() / 24
	secondDuration := secondHalf[len(secondHalf)-1].Timestamp.Sub(secondHalf[0].Timestamp).Hours() / 24

	totalDuration := firstDuration + secondDuration
	if totalDuration == 0 {
		return nil
	}

	acceleration := (slope2 - slope1) / (totalDuration / 2)

	// Significance based on both segments having good fit
	significance := (r1 + r2) / 2

	// Determine trend
	accelerationThreshold := math.Abs(slope1) * 0.1 // 10% change is significant

	var trend string
	isAccelerating := false
	isDecelerating := false

	if math.Abs(acceleration) < accelerationThreshold {
		trend = "CONSTANT"
	} else if acceleration > 0 {
		trend = "ACCELERATING"
		isAccelerating = true
	} else {
		trend = "DECELERATING"
		isDecelerating = true
	}

	return &AccelerationAnalysis{
		Acceleration:      acceleration,
		IsAccelerating:    isAccelerating,
		IsDecelerating:    isDecelerating,
		AccelerationTrend: trend,
		Significance:      significance,
	}
}

// tCriticalValue returns approximate t-distribution critical value for 95% confidence
func (pe *PredictionEngine) tCriticalValue(df int) float64 {
	if df < 1 {
		return 12.71 // df = 1
	}

	// Approximate t-values for common degrees of freedom
	tValues := map[int]float64{
		1: 12.71, 2: 4.30, 3: 3.18, 4: 2.78, 5: 2.57,
		6: 2.45, 7: 2.36, 8: 2.31, 9: 2.26, 10: 2.23,
		15: 2.13, 20: 2.09, 25: 2.06, 30: 2.04,
	}

	if val, ok := tValues[df]; ok {
		return val
	}

	// For large df, approach z-value of 1.96
	if df > 30 {
		return 1.96
	}

	// Interpolate for intermediate values
	lower := 1
	for k := range tValues {
		if k <= df && k > lower {
			lower = k
		}
	}
	return tValues[lower]
}

// =============================================================================
// WEIGHTED PREDICTION (RECENT DATA WEIGHTED HIGHER)
// =============================================================================

// WeightedPrediction generates a prediction giving more weight to recent data
func (pe *PredictionEngine) WeightedPrediction(points []types.TrendDataPoint, horizon PredictionHorizon) *EnhancedPrediction {
	if len(points) < horizon.MinData {
		return nil
	}

	// Apply exponential weighting (more recent = higher weight)
	lastTime := points[len(points)-1].Timestamp
	decayRate := 0.1 // Weight halves every 7 days

	slope, intercept, rSquared := pe.weightedLinearRegression(points, lastTime, decayRate)
	stdError := pe.calculateWeightedStandardError(points, slope, intercept, lastTime, decayRate)

	firstTime := points[0].Timestamp
	predTime := lastTime.Add(time.Duration(horizon.Days) * 24 * time.Hour)
	daysFromFirst := predTime.Sub(firstTime).Hours() / 24

	prediction := pe.generateEnhancedPrediction(
		slope, intercept, stdError, rSquared,
		daysFromFirst, horizon, len(points), predTime,
	)
	prediction.Method = "weighted_linear"

	return prediction
}

// weightedLinearRegression performs weighted least squares regression
func (pe *PredictionEngine) weightedLinearRegression(points []types.TrendDataPoint, referenceTime time.Time, decayRate float64) (slope, intercept, rSquared float64) {
	if len(points) < 2 {
		return 0, 0, 0
	}

	firstTime := points[0].Timestamp

	sumW, sumWX, sumWY, sumWXY, sumWX2, sumWY2 := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0

	for _, p := range points {
		days := p.Timestamp.Sub(firstTime).Hours() / 24
		daysFromRef := referenceTime.Sub(p.Timestamp).Hours() / 24
		weight := math.Exp(-decayRate * daysFromRef)

		sumW += weight
		sumWX += weight * days
		sumWY += weight * p.Value
		sumWXY += weight * days * p.Value
		sumWX2 += weight * days * days
		sumWY2 += weight * p.Value * p.Value
	}

	denominator := sumW*sumWX2 - sumWX*sumWX
	if denominator == 0 {
		return 0, sumWY / sumW, 0
	}

	slope = (sumW*sumWXY - sumWX*sumWY) / denominator
	intercept = (sumWY - slope*sumWX) / sumW

	// Calculate weighted R²
	meanY := sumWY / sumW
	ssTot := sumWY2 - sumW*meanY*meanY

	ssRes := 0.0
	for _, p := range points {
		days := p.Timestamp.Sub(firstTime).Hours() / 24
		daysFromRef := referenceTime.Sub(p.Timestamp).Hours() / 24
		weight := math.Exp(-decayRate * daysFromRef)
		predicted := intercept + slope*days
		ssRes += weight * (p.Value - predicted) * (p.Value - predicted)
	}

	if ssTot == 0 {
		rSquared = 1.0
	} else {
		rSquared = 1.0 - (ssRes / ssTot)
	}

	return slope, intercept, rSquared
}

// calculateWeightedStandardError computes weighted residual standard error
func (pe *PredictionEngine) calculateWeightedStandardError(points []types.TrendDataPoint, slope, intercept float64, referenceTime time.Time, decayRate float64) float64 {
	if len(points) < 3 {
		return 0
	}

	firstTime := points[0].Timestamp
	sumWeightedSquaredResiduals := 0.0
	sumWeights := 0.0

	for _, p := range points {
		days := p.Timestamp.Sub(firstTime).Hours() / 24
		daysFromRef := referenceTime.Sub(p.Timestamp).Hours() / 24
		weight := math.Exp(-decayRate * daysFromRef)

		predicted := intercept + (slope * days)
		residual := p.Value - predicted
		sumWeightedSquaredResiduals += weight * residual * residual
		sumWeights += weight
	}

	// Effective degrees of freedom for weighted regression
	effectiveN := sumWeights * sumWeights / sumWeights // Simplified approximation

	return math.Sqrt(sumWeightedSquaredResiduals / (effectiveN - 2))
}

// =============================================================================
// UTILITY FUNCTION
// =============================================================================

// linearRegression is a standalone helper that matches engine.go signature
func linearRegression(points []types.TrendDataPoint) (slope, intercept, rSquared float64) {
	n := float64(len(points))
	if n < 2 {
		return 0, 0, 0
	}

	firstTime := points[0].Timestamp
	xs := make([]float64, len(points))
	ys := make([]float64, len(points))

	for i, p := range points {
		xs[i] = p.Timestamp.Sub(firstTime).Hours() / 24
		ys[i] = p.Value
	}

	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumX2 += xs[i] * xs[i]
		sumY2 += ys[i] * ys[i]
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, sumY / n, 0
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n

	meanY := sumY / n
	ssTot := 0.0
	ssRes := 0.0
	for i := range ys {
		predicted := intercept + slope*xs[i]
		ssTot += (ys[i] - meanY) * (ys[i] - meanY)
		ssRes += (ys[i] - predicted) * (ys[i] - predicted)
	}

	if ssTot == 0 {
		rSquared = 1.0
	} else {
		rSquared = 1.0 - (ssRes / ssTot)
	}

	return slope, intercept, rSquared
}
