// Package trending provides multi-window trend analysis for lab results
package trending

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"kb-16-lab-interpretation/pkg/store"
	"kb-16-lab-interpretation/pkg/types"
)

// WindowConfig defines a trending window configuration
type WindowConfig struct {
	Name      string
	Days      int
	MinPoints int
	UseCase   string
}

// Standard trending windows
var StandardWindows = map[string]WindowConfig{
	"7d":  {Name: "7 days", Days: 7, MinPoints: 2, UseCase: "Acute changes"},
	"30d": {Name: "30 days", Days: 30, MinPoints: 3, UseCase: "Medication effects"},
	"90d": {Name: "90 days", Days: 90, MinPoints: 4, UseCase: "Chronic monitoring"},
	"1yr": {Name: "1 year", Days: 365, MinPoints: 6, UseCase: "Disease progression"},
}

// Engine performs trend analysis on lab results
type Engine struct {
	resultStore       *store.ResultStore
	predictionEngine  *PredictionEngine
	log               *logrus.Entry
}

// NewEngine creates a new trending engine
func NewEngine(resultStore *store.ResultStore, log *logrus.Entry) *Engine {
	return &Engine{
		resultStore:      resultStore,
		predictionEngine: NewPredictionEngine(),
		log:              log.WithField("component", "trending_engine"),
	}
}

// AnalyzeTrend performs trend analysis for a specific test
func (e *Engine) AnalyzeTrend(ctx context.Context, patientID, code string, windowDays int) (*types.TrendAnalysis, error) {
	// Get results within window
	results, err := e.resultStore.GetByPatientAndCode(ctx, patientID, code, windowDays)
	if err != nil {
		return nil, err
	}

	if len(results) < 2 {
		return nil, nil // Not enough data points for trend analysis
	}

	// Convert to data points
	points := e.toDataPoints(results)

	// Calculate statistics
	stats := e.calculateStatistics(points)

	// Detect trajectory
	trajectory := e.detectTrajectory(points)

	// Calculate rate of change
	rateOfChange := e.calculateRateOfChange(points)

	// Generate prediction if enough data
	var prediction *types.PredictedValue
	if len(points) >= 4 {
		prediction = e.predictNextValue(points)
	}

	return &types.TrendAnalysis{
		TestCode:      code,
		PatientID:     patientID,
		WindowDays:    windowDays,
		DataPoints:    points,
		Trajectory:    trajectory,
		RateOfChange:  rateOfChange,
		Statistics:    stats,
		Prediction:    prediction,
		AnalyzedAt:    time.Now(),
		DataPointCount: len(points),
	}, nil
}

// AnalyzeMultiWindow performs trend analysis across all standard windows
func (e *Engine) AnalyzeMultiWindow(ctx context.Context, patientID, code string) (map[string]*types.TrendAnalysis, error) {
	results := make(map[string]*types.TrendAnalysis)

	for windowKey, config := range StandardWindows {
		trend, err := e.AnalyzeTrend(ctx, patientID, code, config.Days)
		if err != nil {
			e.log.WithError(err).WithField("window", windowKey).Warn("Failed to analyze window")
			continue
		}
		if trend != nil && len(trend.DataPoints) >= config.MinPoints {
			results[windowKey] = trend
		}
	}

	return results, nil
}

// GetAllTrends returns trends for all tests for a patient
func (e *Engine) GetAllTrends(ctx context.Context, patientID string, windowDays int) (map[string]*types.TrendAnalysis, error) {
	// Get all unique codes for patient
	codes, err := e.resultStore.GetDistinctCodes(ctx, patientID)
	if err != nil {
		return nil, err
	}

	results := make(map[string]*types.TrendAnalysis)

	for _, code := range codes {
		trend, err := e.AnalyzeTrend(ctx, patientID, code, windowDays)
		if err != nil {
			e.log.WithError(err).WithField("code", code).Warn("Failed to analyze trend")
			continue
		}
		if trend != nil && len(trend.DataPoints) >= 2 {
			results[code] = trend
		}
	}

	return results, nil
}

// toDataPoints converts lab results to trend data points
func (e *Engine) toDataPoints(results []types.LabResult) []types.TrendDataPoint {
	points := make([]types.TrendDataPoint, 0, len(results))

	for _, r := range results {
		if r.ValueNumeric != nil {
			points = append(points, types.TrendDataPoint{
				Timestamp: r.CollectedAt,
				Value:     *r.ValueNumeric,
				ResultID:  r.ID.String(),
			})
		}
	}

	// Sort by timestamp ascending
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	return points
}

// calculateStatistics computes statistical measures for the data points
func (e *Engine) calculateStatistics(points []types.TrendDataPoint) types.TrendStatistics {
	if len(points) == 0 {
		return types.TrendStatistics{}
	}

	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = p.Value
	}

	// Calculate basic statistics
	mean := e.mean(values)
	stdDev := e.stdDev(values, mean)
	min, max := e.minMax(values)
	median := e.median(values)

	// Calculate coefficient of variation
	cv := 0.0
	if mean != 0 {
		cv = (stdDev / mean) * 100
	}

	return types.TrendStatistics{
		Mean:                  mean,
		StdDev:                stdDev,
		Min:                   min,
		Max:                   max,
		Median:                median,
		CoefficientOfVariation: cv,
		SampleCount:           len(points),
	}
}

// detectTrajectory determines the trend direction
func (e *Engine) detectTrajectory(points []types.TrendDataPoint) types.Trajectory {
	if len(points) < 3 {
		return types.TrajectoryUnknown
	}

	// Calculate linear regression
	slope, _, rSquared := e.linearRegression(points)

	// Calculate coefficient of variation for volatility
	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = p.Value
	}
	mean := e.mean(values)
	stdDev := e.stdDev(values, mean)
	cv := 0.0
	if mean != 0 {
		cv = stdDev / mean
	}

	// Volatility threshold
	if cv > 0.3 {
		return types.TrajectoryVolatile
	}

	// Trend thresholds
	slopeThreshold := 0.01 * mean // 1% of mean per day

	if rSquared < 0.3 {
		// Poor fit - likely stable or volatile
		if cv < 0.1 {
			return types.TrajectoryStable
		}
		return types.TrajectoryVolatile
	}

	if slope > slopeThreshold {
		return types.TrajectoryWorsening // For most labs, increasing trend may be concerning
	}
	if slope < -slopeThreshold {
		return types.TrajectoryImproving // Decreasing trend may indicate improvement
	}

	return types.TrajectoryStable
}

// calculateRateOfChange computes the rate of change per day
func (e *Engine) calculateRateOfChange(points []types.TrendDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	// Use linear regression slope as rate of change
	slope, _, _ := e.linearRegression(points)
	return slope
}

// predictNextValue estimates the next value based on trend
func (e *Engine) predictNextValue(points []types.TrendDataPoint) *types.PredictedValue {
	if len(points) < 4 {
		return nil
	}

	slope, intercept, rSquared := e.linearRegression(points)

	// Predict 7 days ahead
	lastPoint := points[len(points)-1]
	predictionTime := lastPoint.Timestamp.Add(7 * 24 * time.Hour)

	// Days since first point
	firstTime := points[0].Timestamp
	daysFromFirst := predictionTime.Sub(firstTime).Hours() / 24

	predictedValue := intercept + (slope * daysFromFirst)

	// Calculate confidence based on R-squared and sample size
	confidence := rSquared * math.Min(1.0, float64(len(points))/10.0)

	return &types.PredictedValue{
		Value:          predictedValue,
		PredictedAt:    predictionTime,
		Confidence:     confidence,
		BasedOnPoints:  len(points),
	}
}

// linearRegression calculates slope, intercept, and R-squared
func (e *Engine) linearRegression(points []types.TrendDataPoint) (slope, intercept, rSquared float64) {
	n := float64(len(points))
	if n < 2 {
		return 0, 0, 0
	}

	// Convert timestamps to days from first point
	firstTime := points[0].Timestamp
	xs := make([]float64, len(points))
	ys := make([]float64, len(points))

	for i, p := range points {
		xs[i] = p.Timestamp.Sub(firstTime).Hours() / 24 // Days
		ys[i] = p.Value
	}

	// Calculate means
	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumX2 += xs[i] * xs[i]
		sumY2 += ys[i] * ys[i]
	}

	// Calculate slope and intercept
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, sumY / n, 0
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n

	// Calculate R-squared
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

// Statistical helper functions

func (e *Engine) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (e *Engine) stdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

func (e *Engine) minMax(values []float64) (min, max float64) {
	if len(values) == 0 {
		return 0, 0
	}
	min, max = values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return
}

func (e *Engine) median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// =============================================================================
// ENHANCED TREND ANALYSIS (With Clinical Context & Multi-Horizon Predictions)
// =============================================================================

// EnhancedTrendAnalysis extends TrendAnalysis with clinical intelligence
type EnhancedTrendAnalysis struct {
	types.TrendAnalysis

	// Clinical context interpretation
	ClinicalInterpretation TrajectoryInterpretation `json:"clinical_interpretation"`

	// Multi-horizon predictions with confidence intervals
	MultiHorizonPrediction *MultiHorizonPrediction `json:"multi_horizon_prediction,omitempty"`

	// Lab-specific context
	LabContext *LabTrendContext `json:"lab_context,omitempty"`

	// Is the change clinically significant?
	IsClinicallySignificant bool `json:"is_clinically_significant"`

	// Trend strength indicator (0-1)
	TrendStrength float64 `json:"trend_strength"`
}

// AnalyzeTrendEnhanced performs comprehensive trend analysis with clinical context
func (e *Engine) AnalyzeTrendEnhanced(ctx context.Context, patientID, code string, windowDays int) (*EnhancedTrendAnalysis, error) {
	// Get base trend analysis
	baseTrend, err := e.AnalyzeTrend(ctx, patientID, code, windowDays)
	if err != nil {
		return nil, err
	}
	if baseTrend == nil {
		return nil, nil
	}

	// Get clinical context for this lab
	labContext, _ := GetLabContext(code)

	// Create enhanced analysis
	enhanced := &EnhancedTrendAnalysis{
		TrendAnalysis: *baseTrend,
		LabContext:    labContext,
	}

	// Clinical interpretation based on context
	enhanced.ClinicalInterpretation = InterpretTrajectory(code, baseTrend.Trajectory, baseTrend.RateOfChange)

	// Multi-horizon predictions
	if len(baseTrend.DataPoints) >= 4 {
		enhanced.MultiHorizonPrediction = e.predictionEngine.PredictMultiHorizon(baseTrend.DataPoints)
		if enhanced.MultiHorizonPrediction != nil {
			enhanced.TrendStrength = enhanced.MultiHorizonPrediction.TrendStrength
		}
	}

	// Determine clinical significance
	if len(baseTrend.DataPoints) >= 2 {
		first := baseTrend.DataPoints[0].Value
		last := baseTrend.DataPoints[len(baseTrend.DataPoints)-1].Value
		change := math.Abs(last - first)
		enhanced.IsClinicallySignificant = IsClinicallySignificant(code, change)
	}

	return enhanced, nil
}

// AnalyzeMultiWindowEnhanced performs enhanced analysis across all windows
func (e *Engine) AnalyzeMultiWindowEnhanced(ctx context.Context, patientID, code string) (*MultiWindowEnhancedAnalysis, error) {
	result := &MultiWindowEnhancedAnalysis{
		PatientID:  patientID,
		TestCode:   code,
		Windows:    make(map[string]*EnhancedTrendAnalysis),
		AnalyzedAt: time.Now(),
	}

	// Get lab context
	labContext, hasContext := GetLabContext(code)
	if hasContext {
		result.LabContext = labContext
	}

	// Analyze each window
	for windowKey, config := range StandardWindows {
		enhanced, err := e.AnalyzeTrendEnhanced(ctx, patientID, code, config.Days)
		if err != nil {
			e.log.WithError(err).WithField("window", windowKey).Warn("Failed to analyze window")
			continue
		}
		if enhanced != nil && len(enhanced.DataPoints) >= config.MinPoints {
			result.Windows[windowKey] = enhanced
		}
	}

	// Determine overall trajectory across windows
	result.OverallTrajectory = e.determineOverallTrajectory(result.Windows)

	// Synthesize clinical assessment
	result.ClinicalSummary = e.synthesizeClinicalSummary(result)

	return result, nil
}

// MultiWindowEnhancedAnalysis contains enhanced analysis across all time windows
type MultiWindowEnhancedAnalysis struct {
	PatientID          string                            `json:"patient_id"`
	TestCode           string                            `json:"test_code"`
	Windows            map[string]*EnhancedTrendAnalysis `json:"windows"`
	OverallTrajectory  types.Trajectory                  `json:"overall_trajectory"`
	LabContext         *LabTrendContext                  `json:"lab_context,omitempty"`
	ClinicalSummary    *ClinicalTrendSummary             `json:"clinical_summary"`
	AnalyzedAt         time.Time                         `json:"analyzed_at"`
}

// ClinicalTrendSummary provides a synthesized clinical assessment
type ClinicalTrendSummary struct {
	ShortTermTrend     string `json:"short_term_trend"`      // 7d assessment
	MediumTermTrend    string `json:"medium_term_trend"`     // 30d assessment
	LongTermTrend      string `json:"long_term_trend"`       // 90d-1yr assessment
	OverallAssessment  string `json:"overall_assessment"`
	ClinicalUrgency    string `json:"clinical_urgency"`
	KeyInsights        []string `json:"key_insights"`
	RecommendedActions []string `json:"recommended_actions"`
}

// determineOverallTrajectory synthesizes trajectories across windows
func (e *Engine) determineOverallTrajectory(windows map[string]*EnhancedTrendAnalysis) types.Trajectory {
	if len(windows) == 0 {
		return types.TrajectoryUnknown
	}

	// Count trajectories with window weighting (recent windows matter more)
	trajectoryScores := make(map[types.Trajectory]float64)
	windowWeights := map[string]float64{
		"7d":  1.0,  // Most recent, highest weight
		"30d": 0.8,
		"90d": 0.5,
		"1yr": 0.3,
	}

	for windowKey, analysis := range windows {
		weight := windowWeights[windowKey]
		if weight == 0 {
			weight = 0.5
		}
		trajectoryScores[analysis.Trajectory] += weight
	}

	// Find dominant trajectory
	var dominantTrajectory types.Trajectory
	var maxScore float64
	for traj, score := range trajectoryScores {
		if score > maxScore {
			maxScore = score
			dominantTrajectory = traj
		}
	}

	return dominantTrajectory
}

// synthesizeClinicalSummary creates a comprehensive clinical summary
func (e *Engine) synthesizeClinicalSummary(analysis *MultiWindowEnhancedAnalysis) *ClinicalTrendSummary {
	summary := &ClinicalTrendSummary{
		KeyInsights:        make([]string, 0),
		RecommendedActions: make([]string, 0),
	}

	// Assess each time horizon
	if w, ok := analysis.Windows["7d"]; ok {
		summary.ShortTermTrend = string(w.Trajectory)
		if w.ClinicalInterpretation.Urgency == "URGENT" || w.ClinicalInterpretation.Urgency == "ATTENTION" {
			summary.KeyInsights = append(summary.KeyInsights, "Short-term: "+w.ClinicalInterpretation.Explanation)
		}
	} else {
		summary.ShortTermTrend = "INSUFFICIENT_DATA"
	}

	if w, ok := analysis.Windows["30d"]; ok {
		summary.MediumTermTrend = string(w.Trajectory)
		if w.IsClinicallySignificant {
			summary.KeyInsights = append(summary.KeyInsights, "30-day trend shows clinically significant change")
		}
	} else {
		summary.MediumTermTrend = "INSUFFICIENT_DATA"
	}

	// Long-term: prefer 90d, fall back to 1yr
	if w, ok := analysis.Windows["90d"]; ok {
		summary.LongTermTrend = string(w.Trajectory)
	} else if w, ok := analysis.Windows["1yr"]; ok {
		summary.LongTermTrend = string(w.Trajectory)
	} else {
		summary.LongTermTrend = "INSUFFICIENT_DATA"
	}

	// Overall assessment based on trajectory
	summary.OverallAssessment = e.assessOverallStatus(analysis)
	summary.ClinicalUrgency = e.determineClinicalUrgency(analysis)

	// Generate recommended actions
	summary.RecommendedActions = e.generateRecommendedActions(analysis)

	return summary
}

// assessOverallStatus provides an overall status assessment
func (e *Engine) assessOverallStatus(analysis *MultiWindowEnhancedAnalysis) string {
	switch analysis.OverallTrajectory {
	case types.TrajectoryImproving:
		return "Values are trending favorably. Continue current management."
	case types.TrajectoryWorsening:
		return "Values are trending in a concerning direction. Evaluation recommended."
	case types.TrajectoryStable:
		return "Values remain stable. Continue routine monitoring."
	case types.TrajectoryVolatile:
		return "High variability observed. Investigate underlying causes."
	default:
		return "Insufficient data for comprehensive assessment."
	}
}

// determineClinicalUrgency determines the urgency level
func (e *Engine) determineClinicalUrgency(analysis *MultiWindowEnhancedAnalysis) string {
	// Check if any window shows urgent findings
	for _, w := range analysis.Windows {
		if w.ClinicalInterpretation.Urgency == "URGENT" {
			return "URGENT"
		}
	}

	// Check for attention-level findings in recent windows
	if w, ok := analysis.Windows["7d"]; ok {
		if w.ClinicalInterpretation.Urgency == "ATTENTION" {
			return "ATTENTION"
		}
	}

	// Check for worsening in short term
	if w, ok := analysis.Windows["7d"]; ok {
		if w.Trajectory == types.TrajectoryWorsening {
			return "MONITOR"
		}
	}

	return "ROUTINE"
}

// generateRecommendedActions creates actionable recommendations
func (e *Engine) generateRecommendedActions(analysis *MultiWindowEnhancedAnalysis) []string {
	actions := make([]string, 0)

	// Based on overall trajectory
	switch analysis.OverallTrajectory {
	case types.TrajectoryWorsening:
		actions = append(actions, "Review recent clinical events and medication changes")
		actions = append(actions, "Consider more frequent monitoring")
	case types.TrajectoryVolatile:
		actions = append(actions, "Verify specimen collection procedures")
		actions = append(actions, "Assess for acute clinical changes affecting results")
	case types.TrajectoryImproving:
		actions = append(actions, "Continue current treatment approach")
	}

	// Check acceleration in recent window
	if w, ok := analysis.Windows["7d"]; ok {
		if w.MultiHorizonPrediction != nil && w.MultiHorizonPrediction.Acceleration != nil {
			acc := w.MultiHorizonPrediction.Acceleration
			if acc.IsAccelerating && analysis.OverallTrajectory == types.TrajectoryWorsening {
				actions = append(actions, "ALERT: Rate of change is accelerating - prompt evaluation needed")
			}
		}
	}

	// Add lab-specific actions if context available
	if analysis.LabContext != nil {
		if len(analysis.Windows) > 0 {
			for _, w := range analysis.Windows {
				if w.ClinicalInterpretation.RecommendedAction != "" {
					actions = append(actions, w.ClinicalInterpretation.RecommendedAction)
					break // Just add one lab-specific action
				}
			}
		}
	}

	return actions
}

// PopulateTrendWindows fills in the Windows map field of TrendAnalysis
func (e *Engine) PopulateTrendWindows(analysis *types.TrendAnalysis) {
	if analysis == nil || len(analysis.DataPoints) < 2 {
		return
	}

	analysis.Windows = make(map[string]types.TrendWindow)

	for windowKey, config := range StandardWindows {
		// Filter points within this window
		cutoff := time.Now().AddDate(0, 0, -config.Days)
		windowPoints := make([]types.DataPoint, 0)

		for _, dp := range analysis.DataPoints {
			if dp.Timestamp.After(cutoff) {
				windowPoints = append(windowPoints, types.DataPoint{
					Timestamp: dp.Timestamp,
					Value:     dp.Value,
				})
			}
		}

		if len(windowPoints) < config.MinPoints {
			continue
		}

		// Calculate window-specific statistics
		values := make([]float64, len(windowPoints))
		for i, p := range windowPoints {
			values[i] = p.Value
		}

		mean := e.mean(values)
		stdDev := e.stdDev(values, mean)
		minVal, maxVal := e.minMax(values)
		cv := 0.0
		if mean != 0 {
			cv = (stdDev / mean) * 100
		}

		// Calculate window-specific trend
		trendPoints := make([]types.TrendDataPoint, len(windowPoints))
		for i, p := range windowPoints {
			trendPoints[i] = types.TrendDataPoint{
				Timestamp: p.Timestamp,
				Value:     p.Value,
			}
		}
		slope, _, rSquared := e.linearRegression(trendPoints)

		trendDirection := "stable"
		if slope > 0.01*mean {
			trendDirection = "increasing"
		} else if slope < -0.01*mean {
			trendDirection = "decreasing"
		}

		analysis.Windows[windowKey] = types.TrendWindow{
			Name:       config.Name,
			Days:       config.Days,
			DataPoints: windowPoints,
			Statistics: &types.Statistics{
				Mean:             mean,
				StdDev:           stdDev,
				Min:              minVal,
				Max:              maxVal,
				Median:           e.median(values),
				Count:            len(windowPoints),
				CoefficientOfVar: cv,
			},
			Trend:    trendDirection,
			Slope:    slope,
			RSquared: rSquared,
		}
	}
}
