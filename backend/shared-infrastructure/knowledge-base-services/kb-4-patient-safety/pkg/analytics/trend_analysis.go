package analytics

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	TrendUp       TrendDirection = "INCREASING"
	TrendDown     TrendDirection = "DECREASING"
	TrendStable   TrendDirection = "STABLE"
	TrendCyclical TrendDirection = "CYCLICAL"
	TrendErratic  TrendDirection = "ERRATIC"
)

// TrendConfidence represents confidence level in trend detection
type TrendConfidence string

const (
	ConfidenceHigh   TrendConfidence = "HIGH"
	ConfidenceMedium TrendConfidence = "MEDIUM"
	ConfidenceLow    TrendConfidence = "LOW"
)

// TrendType represents the type of trend analysis
type TrendType string

const (
	TrendTypeLinear      TrendType = "LINEAR"
	TrendTypeExponential TrendType = "EXPONENTIAL"
	TrendTypeSeasonal    TrendType = "SEASONAL"
	TrendTypeMovingAvg   TrendType = "MOVING_AVERAGE"
)

// StatisticalTrend represents a detected trend
type StatisticalTrend struct {
	TrendID            string                 `json:"trendId"`
	DrugCode           string                 `json:"drugCode"`
	EventType          string                 `json:"eventType"`
	Direction          TrendDirection         `json:"direction"`
	Confidence         TrendConfidence        `json:"confidence"`
	TrendType          TrendType              `json:"trendType"`
	Slope              float64                `json:"slope"`
	RSquared           float64                `json:"rSquared"`
	PValue             float64                `json:"pValue"`
	PercentageChange   float64                `json:"percentageChange"`
	StartPeriod        time.Time              `json:"startPeriod"`
	EndPeriod          time.Time              `json:"endPeriod"`
	DataPoints         int                    `json:"dataPoints"`
	Forecast           []ForecastPoint        `json:"forecast"`
	SeasonalComponents *SeasonalDecomposition `json:"seasonalComponents,omitempty"`
	Anomalies          []TrendAnomaly         `json:"anomalies"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// ForecastPoint represents a forecasted future value
type ForecastPoint struct {
	Timestamp          time.Time `json:"timestamp"`
	PredictedValue     float64   `json:"predictedValue"`
	ConfidenceInterval []float64 `json:"confidenceInterval"`
}

// SeasonalDecomposition represents seasonal trend components
type SeasonalDecomposition struct {
	TrendComponent    []float64 `json:"trendComponent"`
	SeasonalComponent []float64 `json:"seasonalComponent"`
	ResidualComponent []float64 `json:"residualComponent"`
	SeasonalPeriod    int       `json:"seasonalPeriod"`
}

// TrendAnomaly represents an anomaly within a trend
type TrendAnomaly struct {
	Timestamp       time.Time `json:"timestamp"`
	Value           float64   `json:"value"`
	ExpectedValue   float64   `json:"expectedValue"`
	Deviation       float64   `json:"deviation"`
	AnomalyStrength float64   `json:"anomalyStrength"`
}

// TrendRequest represents a trend analysis request
type TrendRequest struct {
	RequestID           string      `json:"requestId"`
	DrugCodes           []string    `json:"drugCodes"`
	EventTypes          []string    `json:"eventTypes"`
	StartTime           time.Time   `json:"startTime"`
	EndTime             time.Time   `json:"endTime"`
	Granularity         string      `json:"granularity"` // DAY, WEEK, MONTH
	TrendTypes          []TrendType `json:"trendTypes"`
	IncludeForecast     bool        `json:"includeForecast"`
	ForecastPeriods     int         `json:"forecastPeriods"`
	ConfidenceThreshold float64     `json:"confidenceThreshold"`
}

// TrendResponse represents the trend analysis response
type TrendResponse struct {
	RequestID         string             `json:"requestId"`
	Trends            []StatisticalTrend `json:"trends"`
	Summary           TrendSummary       `json:"summary"`
	AnalysisTimestamp time.Time          `json:"analysisTimestamp"`
}

// TrendSummary provides summary of trend analysis
type TrendSummary struct {
	TotalTrendsDetected   int       `json:"totalTrendsDetected"`
	IncreasingTrends      int       `json:"increasingTrends"`
	DecreasingTrends      int       `json:"decreasingTrends"`
	StableTrends          int       `json:"stableTrends"`
	HighConfidenceTrends  int       `json:"highConfidenceTrends"`
	AnalysisPeriodDays    int       `json:"analysisPeriodDays"`
	OverallTrendDirection string    `json:"overallTrendDirection"`
	Alerts                []string  `json:"alerts"`
}

// TrendAnalyzer performs time-series trend analysis
type TrendAnalyzer struct {
	mu             sync.RWMutex
	historicalData map[string][]TimeSeriesPoint
	detectedTrends map[string]*StatisticalTrend
}

// TimeSeriesPoint represents a time-series data point
type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
	DrugCode  string
	EventType string
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{
		historicalData: make(map[string][]TimeSeriesPoint),
		detectedTrends: make(map[string]*StatisticalTrend),
	}
}

// AnalyzeTrends performs comprehensive trend analysis
func (ta *TrendAnalyzer) AnalyzeTrends(req *TrendRequest) *TrendResponse {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	response := &TrendResponse{
		RequestID:         req.RequestID,
		Trends:            []StatisticalTrend{},
		AnalysisTimestamp: time.Now(),
	}

	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
		response.RequestID = req.RequestID
	}

	var increasingCount, decreasingCount, stableCount, highConfCount int
	var alerts []string

	for _, drugCode := range req.DrugCodes {
		for _, eventType := range req.EventTypes {
			key := drugCode + ":" + eventType
			data := ta.getDataForKey(key, req.StartTime, req.EndTime)

			if len(data) < 5 {
				continue
			}

			// Analyze based on requested trend types
			for _, trendType := range req.TrendTypes {
				var trend *StatisticalTrend

				switch trendType {
				case TrendTypeLinear:
					trend = ta.analyzeLinearTrend(drugCode, eventType, data, req)
				case TrendTypeMovingAvg:
					trend = ta.analyzeMovingAverageTrend(drugCode, eventType, data, req)
				case TrendTypeSeasonal:
					trend = ta.analyzeSeasonalTrend(drugCode, eventType, data, req)
				case TrendTypeExponential:
					trend = ta.analyzeExponentialTrend(drugCode, eventType, data, req)
				default:
					trend = ta.analyzeLinearTrend(drugCode, eventType, data, req)
				}

				if trend != nil && trend.Confidence != "" {
					// Apply confidence threshold filter
					if isConfidenceSufficient(trend.Confidence, req.ConfidenceThreshold) {
						response.Trends = append(response.Trends, *trend)
						ta.detectedTrends[trend.TrendID] = trend

						// Count by direction
						switch trend.Direction {
						case TrendUp:
							increasingCount++
						case TrendDown:
							decreasingCount++
						case TrendStable:
							stableCount++
						}

						if trend.Confidence == ConfidenceHigh {
							highConfCount++
						}

						// Generate alerts for significant trends
						if trend.Direction == TrendUp && math.Abs(trend.PercentageChange) > 25 {
							alerts = append(alerts, "Significant increase detected for "+drugCode+": "+eventType)
						}
						if trend.Direction == TrendDown && math.Abs(trend.PercentageChange) > 25 {
							alerts = append(alerts, "Significant decrease detected for "+drugCode+": "+eventType)
						}
					}
				}
			}
		}
	}

	// Determine overall direction
	var overallDirection string
	if increasingCount > decreasingCount && increasingCount > stableCount {
		overallDirection = "PREDOMINANTLY_INCREASING"
	} else if decreasingCount > increasingCount && decreasingCount > stableCount {
		overallDirection = "PREDOMINANTLY_DECREASING"
	} else {
		overallDirection = "PREDOMINANTLY_STABLE"
	}

	response.Summary = TrendSummary{
		TotalTrendsDetected:   len(response.Trends),
		IncreasingTrends:      increasingCount,
		DecreasingTrends:      decreasingCount,
		StableTrends:          stableCount,
		HighConfidenceTrends:  highConfCount,
		AnalysisPeriodDays:    int(req.EndTime.Sub(req.StartTime).Hours() / 24),
		OverallTrendDirection: overallDirection,
		Alerts:                alerts,
	}

	return response
}

// analyzeLinearTrend performs linear regression trend analysis
func (ta *TrendAnalyzer) analyzeLinearTrend(drugCode, eventType string, data []TimeSeriesPoint, req *TrendRequest) *StatisticalTrend {
	n := len(data)
	if n < 3 {
		return nil
	}

	// Prepare data for linear regression
	x := make([]float64, n)
	y := make([]float64, n)
	for i, dp := range data {
		x[i] = float64(i)
		y[i] = dp.Value
	}

	// Calculate linear regression coefficients
	slope, intercept, rSquared := linearRegression(x, y)

	// Calculate p-value for slope significance
	pValue := calculateSlopePValue(x, y, slope)

	// Determine direction based on slope
	direction := determineTrendDirection(slope, rSquared)

	// Calculate percentage change
	startValue := intercept
	endValue := intercept + slope*float64(n-1)
	percentageChange := 0.0
	if startValue != 0 {
		percentageChange = ((endValue - startValue) / math.Abs(startValue)) * 100
	}

	// Determine confidence
	confidence := determineConfidence(rSquared, pValue, n)

	// Detect anomalies
	anomalies := detectTrendAnomalies(data, slope, intercept)

	trend := &StatisticalTrend{
		TrendID:          uuid.New().String(),
		DrugCode:         drugCode,
		EventType:        eventType,
		Direction:        direction,
		Confidence:       confidence,
		TrendType:        TrendTypeLinear,
		Slope:            slope,
		RSquared:         rSquared,
		PValue:           pValue,
		PercentageChange: percentageChange,
		StartPeriod:      data[0].Timestamp,
		EndPeriod:        data[n-1].Timestamp,
		DataPoints:       n,
		Anomalies:        anomalies,
		Metadata: map[string]interface{}{
			"intercept":    intercept,
			"meanValue":    calculateMean(extractTSValues(data)),
			"stdDev":       calculateStdDev(extractTSValues(data), calculateMean(extractTSValues(data))),
		},
	}

	// Add forecast if requested
	if req.IncludeForecast && req.ForecastPeriods > 0 {
		trend.Forecast = generateLinearForecast(data, slope, intercept, req.ForecastPeriods, req.Granularity)
	}

	return trend
}

// analyzeMovingAverageTrend performs moving average trend analysis
func (ta *TrendAnalyzer) analyzeMovingAverageTrend(drugCode, eventType string, data []TimeSeriesPoint, req *TrendRequest) *StatisticalTrend {
	n := len(data)
	if n < 7 {
		return nil
	}

	windowSize := 7 // 7-period moving average
	if n < windowSize*2 {
		windowSize = n / 2
	}

	// Calculate moving averages
	maValues := calculateMovingAverage(extractTSValues(data), windowSize)

	// Analyze trend in moving average
	if len(maValues) < 3 {
		return nil
	}

	maX := make([]float64, len(maValues))
	for i := range maX {
		maX[i] = float64(i)
	}

	slope, intercept, rSquared := linearRegression(maX, maValues)
	direction := determineTrendDirection(slope, rSquared)
	confidence := determineConfidence(rSquared, 0.05, len(maValues)) // Using 0.05 as default p-value

	// Calculate percentage change based on smoothed values
	percentageChange := 0.0
	if maValues[0] != 0 {
		percentageChange = ((maValues[len(maValues)-1] - maValues[0]) / math.Abs(maValues[0])) * 100
	}

	return &StatisticalTrend{
		TrendID:          uuid.New().String(),
		DrugCode:         drugCode,
		EventType:        eventType,
		Direction:        direction,
		Confidence:       confidence,
		TrendType:        TrendTypeMovingAvg,
		Slope:            slope,
		RSquared:         rSquared,
		PValue:           0.05, // Placeholder
		PercentageChange: percentageChange,
		StartPeriod:      data[windowSize-1].Timestamp,
		EndPeriod:        data[n-1].Timestamp,
		DataPoints:       len(maValues),
		Metadata: map[string]interface{}{
			"windowSize":      windowSize,
			"intercept":       intercept,
			"smoothedValues":  maValues,
		},
	}
}

// analyzeSeasonalTrend performs seasonal decomposition
func (ta *TrendAnalyzer) analyzeSeasonalTrend(drugCode, eventType string, data []TimeSeriesPoint, req *TrendRequest) *StatisticalTrend {
	n := len(data)
	if n < 14 { // Need at least 2 periods for seasonal analysis
		return nil
	}

	// Detect seasonal period (try common periods: 7, 30, 365 days)
	period := detectSeasonalPeriod(extractTSValues(data))
	if period == 0 {
		period = 7 // Default to weekly
	}

	// Perform seasonal decomposition
	decomposition := seasonalDecompose(extractTSValues(data), period)
	if decomposition == nil {
		return nil
	}

	// Analyze trend component
	trendX := make([]float64, len(decomposition.TrendComponent))
	for i := range trendX {
		trendX[i] = float64(i)
	}

	slope, intercept, rSquared := linearRegression(trendX, decomposition.TrendComponent)
	direction := determineTrendDirection(slope, rSquared)

	percentageChange := 0.0
	if len(decomposition.TrendComponent) > 0 && decomposition.TrendComponent[0] != 0 {
		percentageChange = ((decomposition.TrendComponent[len(decomposition.TrendComponent)-1] - decomposition.TrendComponent[0]) / math.Abs(decomposition.TrendComponent[0])) * 100
	}

	// Calculate seasonal strength
	seasonalStrength := calculateSeasonalStrength(decomposition)

	return &StatisticalTrend{
		TrendID:            uuid.New().String(),
		DrugCode:           drugCode,
		EventType:          eventType,
		Direction:          direction,
		Confidence:         determineConfidence(rSquared, 0.05, len(decomposition.TrendComponent)),
		TrendType:          TrendTypeSeasonal,
		Slope:              slope,
		RSquared:           rSquared,
		PValue:             0.05,
		PercentageChange:   percentageChange,
		StartPeriod:        data[0].Timestamp,
		EndPeriod:          data[n-1].Timestamp,
		DataPoints:         n,
		SeasonalComponents: decomposition,
		Metadata: map[string]interface{}{
			"seasonalPeriod":   period,
			"seasonalStrength": seasonalStrength,
			"intercept":        intercept,
		},
	}
}

// analyzeExponentialTrend performs exponential trend analysis
func (ta *TrendAnalyzer) analyzeExponentialTrend(drugCode, eventType string, data []TimeSeriesPoint, req *TrendRequest) *StatisticalTrend {
	n := len(data)
	if n < 5 {
		return nil
	}

	values := extractTSValues(data)

	// Check for positive values (required for log transform)
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}

	// Shift values if necessary to ensure all positive
	shift := 0.0
	if minVal <= 0 {
		shift = math.Abs(minVal) + 1
	}

	// Transform to log scale
	logValues := make([]float64, n)
	for i, v := range values {
		logValues[i] = math.Log(v + shift)
	}

	x := make([]float64, n)
	for i := range x {
		x[i] = float64(i)
	}

	// Linear regression on log values
	slope, intercept, rSquared := linearRegression(x, logValues)

	// Back-transform coefficients
	expGrowthRate := math.Exp(slope) - 1 // Growth rate per period
	initialValue := math.Exp(intercept) - shift

	direction := TrendStable
	if expGrowthRate > 0.01 {
		direction = TrendUp
	} else if expGrowthRate < -0.01 {
		direction = TrendDown
	}

	percentageChange := expGrowthRate * 100 * float64(n-1) // Total percentage change

	return &StatisticalTrend{
		TrendID:          uuid.New().String(),
		DrugCode:         drugCode,
		EventType:        eventType,
		Direction:        direction,
		Confidence:       determineConfidence(rSquared, 0.05, n),
		TrendType:        TrendTypeExponential,
		Slope:            expGrowthRate,
		RSquared:         rSquared,
		PValue:           0.05,
		PercentageChange: percentageChange,
		StartPeriod:      data[0].Timestamp,
		EndPeriod:        data[n-1].Timestamp,
		DataPoints:       n,
		Metadata: map[string]interface{}{
			"exponentialRate": expGrowthRate,
			"initialValue":    initialValue,
			"shift":           shift,
		},
	}
}

// AddTimeSeriesPoint adds a new data point for trend analysis
func (ta *TrendAnalyzer) AddTimeSeriesPoint(point TimeSeriesPoint) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	key := point.DrugCode + ":" + point.EventType
	ta.historicalData[key] = append(ta.historicalData[key], point)

	// Keep only last 365 days of data per key
	if len(ta.historicalData[key]) > 365 {
		ta.historicalData[key] = ta.historicalData[key][len(ta.historicalData[key])-365:]
	}
}

// getDataForKey retrieves historical data for a key
func (ta *TrendAnalyzer) getDataForKey(key string, start, end time.Time) []TimeSeriesPoint {
	data := ta.historicalData[key]
	var filtered []TimeSeriesPoint
	for _, dp := range data {
		if (dp.Timestamp.Equal(start) || dp.Timestamp.After(start)) &&
			(dp.Timestamp.Equal(end) || dp.Timestamp.Before(end)) {
			filtered = append(filtered, dp)
		}
	}
	return filtered
}

// GetTrend retrieves a detected trend by ID
func (ta *TrendAnalyzer) GetTrend(trendID string) *StatisticalTrend {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	return ta.detectedTrends[trendID]
}

// Helper functions

func extractTSValues(data []TimeSeriesPoint) []float64 {
	values := make([]float64, len(data))
	for i, dp := range data {
		values[i] = dp.Value
	}
	return values
}

func linearRegression(x, y []float64) (slope, intercept, rSquared float64) {
	n := float64(len(x))
	if n == 0 {
		return 0, 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, sumY / n, 0
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n

	// Calculate R-squared
	meanY := sumY / n
	var ssTot, ssRes float64
	for i := range y {
		predicted := slope*x[i] + intercept
		ssTot += (y[i] - meanY) * (y[i] - meanY)
		ssRes += (y[i] - predicted) * (y[i] - predicted)
	}

	if ssTot == 0 {
		rSquared = 1
	} else {
		rSquared = 1 - ssRes/ssTot
	}

	return slope, intercept, rSquared
}

func calculateSlopePValue(x, y []float64, slope float64) float64 {
	n := float64(len(x))
	if n <= 2 {
		return 1
	}

	// Calculate standard error of slope
	meanX := 0.0
	for _, v := range x {
		meanX += v
	}
	meanX /= n

	var ssX, ssRes float64
	_, intercept, _ := linearRegression(x, y)
	for i := range x {
		ssX += (x[i] - meanX) * (x[i] - meanX)
		predicted := slope*x[i] + intercept
		ssRes += (y[i] - predicted) * (y[i] - predicted)
	}

	if ssX == 0 {
		return 1
	}

	mse := ssRes / (n - 2)
	seBeta := math.Sqrt(mse / ssX)

	if seBeta == 0 {
		return 0
	}

	tStat := slope / seBeta
	// Approximate p-value using normal distribution for large samples
	pValue := 2 * (1 - normalCDF(math.Abs(tStat)))

	return pValue
}

func determineTrendDirection(slope, rSquared float64) TrendDirection {
	if rSquared < 0.1 {
		return TrendErratic
	}

	threshold := 0.001 // Minimum slope for trend detection
	if math.Abs(slope) < threshold {
		return TrendStable
	}

	if slope > 0 {
		return TrendUp
	}
	return TrendDown
}

func determineConfidence(rSquared, pValue float64, n int) TrendConfidence {
	// Consider sample size, R-squared, and p-value
	if n < 10 {
		return ConfidenceLow
	}

	if rSquared >= 0.7 && pValue < 0.01 {
		return ConfidenceHigh
	}

	if rSquared >= 0.4 && pValue < 0.05 {
		return ConfidenceMedium
	}

	return ConfidenceLow
}

func isConfidenceSufficient(confidence TrendConfidence, threshold float64) bool {
	switch confidence {
	case ConfidenceHigh:
		return threshold <= 0.9
	case ConfidenceMedium:
		return threshold <= 0.7
	case ConfidenceLow:
		return threshold <= 0.5
	}
	return false
}

func detectTrendAnomalies(data []TimeSeriesPoint, slope, intercept float64) []TrendAnomaly {
	var anomalies []TrendAnomaly

	values := extractTSValues(data)
	residuals := make([]float64, len(values))
	for i, v := range values {
		expected := slope*float64(i) + intercept
		residuals[i] = v - expected
	}

	// Calculate residual statistics
	residMean := calculateMean(residuals)
	residStd := calculateStdDev(residuals, residMean)

	// Detect anomalies (> 2 std deviations)
	for i, r := range residuals {
		if residStd > 0 {
			zScore := (r - residMean) / residStd
			if math.Abs(zScore) > 2 {
				expected := slope*float64(i) + intercept
				anomalies = append(anomalies, TrendAnomaly{
					Timestamp:       data[i].Timestamp,
					Value:           values[i],
					ExpectedValue:   expected,
					Deviation:       r,
					AnomalyStrength: math.Abs(zScore),
				})
			}
		}
	}

	return anomalies
}

func calculateMovingAverage(values []float64, windowSize int) []float64 {
	if len(values) < windowSize {
		return nil
	}

	result := make([]float64, len(values)-windowSize+1)
	for i := 0; i <= len(values)-windowSize; i++ {
		sum := 0.0
		for j := 0; j < windowSize; j++ {
			sum += values[i+j]
		}
		result[i] = sum / float64(windowSize)
	}
	return result
}

func detectSeasonalPeriod(values []float64) int {
	// Use autocorrelation to detect seasonality
	n := len(values)
	if n < 14 {
		return 0
	}

	// Check common periods: 7 (weekly), 30 (monthly), 12 (quarterly if monthly data)
	periods := []int{7, 14, 30, 12}
	bestPeriod := 0
	bestCorrelation := 0.0

	for _, period := range periods {
		if period > n/2 {
			continue
		}

		// Calculate autocorrelation at this lag
		correlation := autocorrelation(values, period)
		if correlation > bestCorrelation && correlation > 0.3 {
			bestCorrelation = correlation
			bestPeriod = period
		}
	}

	return bestPeriod
}

func autocorrelation(values []float64, lag int) float64 {
	n := len(values)
	if lag >= n {
		return 0
	}

	mean := calculateMean(values)
	var numerator, denominator float64

	for i := 0; i < n-lag; i++ {
		numerator += (values[i] - mean) * (values[i+lag] - mean)
	}

	for i := 0; i < n; i++ {
		denominator += (values[i] - mean) * (values[i] - mean)
	}

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func seasonalDecompose(values []float64, period int) *SeasonalDecomposition {
	n := len(values)
	if n < period*2 {
		return nil
	}

	// Calculate trend using centered moving average
	trend := make([]float64, n)
	halfWindow := period / 2
	for i := halfWindow; i < n-halfWindow; i++ {
		sum := 0.0
		count := 0
		for j := i - halfWindow; j <= i+halfWindow; j++ {
			if j >= 0 && j < n {
				sum += values[j]
				count++
			}
		}
		trend[i] = sum / float64(count)
	}

	// Fill edges with linear extrapolation
	for i := 0; i < halfWindow; i++ {
		trend[i] = trend[halfWindow]
	}
	for i := n - halfWindow; i < n; i++ {
		trend[i] = trend[n-halfWindow-1]
	}

	// Calculate detrended series
	detrended := make([]float64, n)
	for i := range values {
		detrended[i] = values[i] - trend[i]
	}

	// Calculate seasonal component (average for each season)
	seasonal := make([]float64, n)
	seasonalAvg := make([]float64, period)
	seasonCounts := make([]int, period)

	for i := range detrended {
		seasonIdx := i % period
		seasonalAvg[seasonIdx] += detrended[i]
		seasonCounts[seasonIdx]++
	}

	for i := range seasonalAvg {
		if seasonCounts[i] > 0 {
			seasonalAvg[i] /= float64(seasonCounts[i])
		}
	}

	// Apply seasonal pattern
	for i := range seasonal {
		seasonal[i] = seasonalAvg[i%period]
	}

	// Calculate residual
	residual := make([]float64, n)
	for i := range values {
		residual[i] = values[i] - trend[i] - seasonal[i]
	}

	return &SeasonalDecomposition{
		TrendComponent:    trend,
		SeasonalComponent: seasonal,
		ResidualComponent: residual,
		SeasonalPeriod:    period,
	}
}

func calculateSeasonalStrength(decomp *SeasonalDecomposition) float64 {
	if decomp == nil {
		return 0
	}

	// Seasonal strength = 1 - Var(residual) / Var(seasonal + residual)
	combined := make([]float64, len(decomp.SeasonalComponent))
	for i := range combined {
		combined[i] = decomp.SeasonalComponent[i] + decomp.ResidualComponent[i]
	}

	varResidual := variance(decomp.ResidualComponent)
	varCombined := variance(combined)

	if varCombined == 0 {
		return 0
	}

	strength := 1 - varResidual/varCombined
	return math.Max(0, strength)
}

func variance(values []float64) float64 {
	mean := calculateMean(values)
	var sum float64
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	if len(values) <= 1 {
		return 0
	}
	return sum / float64(len(values)-1)
}

func generateLinearForecast(data []TimeSeriesPoint, slope, intercept float64, periods int, granularity string) []ForecastPoint {
	if len(data) == 0 || periods == 0 {
		return nil
	}

	n := len(data)
	lastTimestamp := data[n-1].Timestamp
	values := extractTSValues(data)
	residualStd := calculateResidualStd(values, slope, intercept)

	forecasts := make([]ForecastPoint, periods)
	for i := 0; i < periods; i++ {
		futureX := float64(n + i)
		predicted := slope*futureX + intercept

		// Calculate confidence interval (95%)
		margin := 1.96 * residualStd * math.Sqrt(1+1/float64(n)+math.Pow(futureX-float64(n)/2, 2))

		// Determine next timestamp based on granularity
		var nextTime time.Time
		switch granularity {
		case "DAY":
			nextTime = lastTimestamp.AddDate(0, 0, i+1)
		case "WEEK":
			nextTime = lastTimestamp.AddDate(0, 0, (i+1)*7)
		case "MONTH":
			nextTime = lastTimestamp.AddDate(0, i+1, 0)
		default:
			nextTime = lastTimestamp.AddDate(0, 0, i+1)
		}

		forecasts[i] = ForecastPoint{
			Timestamp:          nextTime,
			PredictedValue:     predicted,
			ConfidenceInterval: []float64{predicted - margin, predicted + margin},
		}
	}

	return forecasts
}

func calculateResidualStd(values []float64, slope, intercept float64) float64 {
	var residuals []float64
	for i, v := range values {
		expected := slope*float64(i) + intercept
		residuals = append(residuals, v-expected)
	}
	mean := calculateMean(residuals)
	return calculateStdDev(residuals, mean)
}

// GetAllTrends returns all detected trends (for testing/debugging)
func (ta *TrendAnalyzer) GetAllTrends() []*StatisticalTrend {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	trends := make([]*StatisticalTrend, 0, len(ta.detectedTrends))
	for _, trend := range ta.detectedTrends {
		trends = append(trends, trend)
	}

	// Sort by start period
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].StartPeriod.Before(trends[j].StartPeriod)
	})

	return trends
}
