package analytics

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SignalDetectionAlgorithm represents available detection algorithms
type SignalDetectionAlgorithm string

const (
	AlgorithmSPC      SignalDetectionAlgorithm = "SPC"       // Statistical Process Control
	AlgorithmCUSUM    SignalDetectionAlgorithm = "CUSUM"     // Cumulative Sum
	AlgorithmEWMA     SignalDetectionAlgorithm = "EWMA"      // Exponentially Weighted Moving Average
	AlgorithmMLAnomaly SignalDetectionAlgorithm = "ML_ANOMALY" // Machine Learning Anomaly
)

// SafetySignal represents a detected safety signal
type SafetySignal struct {
	SignalID               string                   `json:"signalId"`
	DrugCode               string                   `json:"drugCode"`
	SignalType             string                   `json:"signalType"`
	SignalStrength         float64                  `json:"signalStrength"`
	PValue                 float64                  `json:"pValue"`
	ConfidenceIntervalLower float64                 `json:"confidenceIntervalLower"`
	ConfidenceIntervalUpper float64                 `json:"confidenceIntervalUpper"`
	ClinicalDescription    string                   `json:"clinicalDescription"`
	AffectedPopulations    []string                 `json:"affectedPopulations"`
	SeverityAssessment     string                   `json:"severityAssessment"`
	FirstDetected          time.Time                `json:"firstDetected"`
	LastUpdated            time.Time                `json:"lastUpdated"`
	OccurrenceCount        int                      `json:"occurrenceCount"`
	DetectionMethod        SignalDetectionAlgorithm `json:"detectionMethod"`
	DetectionParameters    map[string]interface{}   `json:"detectionParameters"`
}

// SignalDetectionRequest represents a signal detection request
type SignalDetectionRequest struct {
	AnalysisID            string                   `json:"analysisId"`
	DrugCodes             []string                 `json:"drugCodes"`
	StartTime             time.Time                `json:"startTime"`
	EndTime               time.Time                `json:"endTime"`
	Algorithm             SignalDetectionAlgorithm `json:"algorithm"`
	ConfidenceThreshold   float64                  `json:"confidenceThreshold"`
	MinimumSampleSize     int                      `json:"minimumSampleSize"`
	IncludeHistoricalBaseline bool                 `json:"includeHistoricalBaseline"`
}

// SignalDetectionResponse represents the detection results
type SignalDetectionResponse struct {
	AnalysisID        string                 `json:"analysisId"`
	DetectedSignals   []SafetySignal         `json:"detectedSignals"`
	Summary           StatisticalSummary     `json:"summary"`
	Warnings          []string               `json:"warnings"`
	AnalysisTimestamp time.Time              `json:"analysisTimestamp"`
}

// StatisticalSummary provides summary statistics
type StatisticalSummary struct {
	TotalEventsAnalyzed   int64     `json:"totalEventsAnalyzed"`
	SignalsDetected       int       `json:"signalsDetected"`
	OverallSignalRate     float64   `json:"overallSignalRate"`
	AnalysisPeriodStart   time.Time `json:"analysisPeriodStart"`
	AnalysisPeriodEnd     time.Time `json:"analysisPeriodEnd"`
	Metrics               []Metric  `json:"metrics"`
}

// Metric represents a statistical metric
type Metric struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
}

// ControlLimits for SPC charts
type ControlLimits struct {
	UpperControlLimit  float64 `json:"upperControlLimit"`
	LowerControlLimit  float64 `json:"lowerControlLimit"`
	UpperWarningLimit  float64 `json:"upperWarningLimit"`
	LowerWarningLimit  float64 `json:"lowerWarningLimit"`
	CenterLine         float64 `json:"centerLine"`
}

// SignalDetector performs statistical signal detection
type SignalDetector struct {
	mu              sync.RWMutex
	historicalData  map[string][]DataPoint
	detectedSignals map[string]*SafetySignal
	controlLimits   map[string]*ControlLimits
}

// DataPoint represents a time-series data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
	DrugCode  string
	EventType string
}

// NewSignalDetector creates a new signal detector
func NewSignalDetector() *SignalDetector {
	return &SignalDetector{
		historicalData:  make(map[string][]DataPoint),
		detectedSignals: make(map[string]*SafetySignal),
		controlLimits:   make(map[string]*ControlLimits),
	}
}

// DetectSignals performs signal detection using the specified algorithm
func (sd *SignalDetector) DetectSignals(req *SignalDetectionRequest) *SignalDetectionResponse {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	response := &SignalDetectionResponse{
		AnalysisID:        req.AnalysisID,
		DetectedSignals:   []SafetySignal{},
		Warnings:          []string{},
		AnalysisTimestamp: time.Now(),
	}

	if req.AnalysisID == "" {
		req.AnalysisID = uuid.New().String()
		response.AnalysisID = req.AnalysisID
	}

	var totalEvents int64
	for _, drugCode := range req.DrugCodes {
		data := sd.getDataForDrug(drugCode, req.StartTime, req.EndTime)
		totalEvents += int64(len(data))

		if len(data) < req.MinimumSampleSize {
			response.Warnings = append(response.Warnings,
				"Insufficient data for "+drugCode+": "+string(rune(len(data)))+" samples (minimum: "+string(rune(req.MinimumSampleSize))+")")
			continue
		}

		var signals []SafetySignal
		switch req.Algorithm {
		case AlgorithmSPC:
			signals = sd.detectSPC(drugCode, data, req.ConfidenceThreshold)
		case AlgorithmCUSUM:
			signals = sd.detectCUSUM(drugCode, data, req.ConfidenceThreshold)
		case AlgorithmEWMA:
			signals = sd.detectEWMA(drugCode, data, req.ConfidenceThreshold)
		case AlgorithmMLAnomaly:
			signals = sd.detectMLAnomaly(drugCode, data, req.ConfidenceThreshold)
		default:
			signals = sd.detectSPC(drugCode, data, req.ConfidenceThreshold) // Default to SPC
		}

		response.DetectedSignals = append(response.DetectedSignals, signals...)
	}

	// Calculate summary
	response.Summary = StatisticalSummary{
		TotalEventsAnalyzed: totalEvents,
		SignalsDetected:     len(response.DetectedSignals),
		OverallSignalRate: func() float64 {
			if totalEvents == 0 {
				return 0
			}
			return float64(len(response.DetectedSignals)) / float64(totalEvents)
		}(),
		AnalysisPeriodStart: req.StartTime,
		AnalysisPeriodEnd:   req.EndTime,
		Metrics: []Metric{
			{Name: "total_drugs_analyzed", Value: float64(len(req.DrugCodes)), Unit: "count", Description: "Number of drugs analyzed"},
			{Name: "analysis_duration_ms", Value: float64(time.Since(response.AnalysisTimestamp).Milliseconds()), Unit: "ms", Description: "Time to complete analysis"},
		},
	}

	return response
}

// detectSPC performs Statistical Process Control detection
func (sd *SignalDetector) detectSPC(drugCode string, data []DataPoint, confidenceThreshold float64) []SafetySignal {
	if len(data) < 2 {
		return nil
	}

	values := extractValues(data)
	mean := calculateMean(values)
	stdDev := calculateStdDev(values, mean)

	// Calculate control limits (3-sigma rule)
	limits := &ControlLimits{
		CenterLine:         mean,
		UpperControlLimit:  mean + 3*stdDev,
		LowerControlLimit:  mean - 3*stdDev,
		UpperWarningLimit:  mean + 2*stdDev,
		LowerWarningLimit:  mean - 2*stdDev,
	}
	sd.controlLimits[drugCode] = limits

	var signals []SafetySignal

	// Check for points outside control limits
	for _, dp := range data {
		if dp.Value > limits.UpperControlLimit || dp.Value < limits.LowerControlLimit {
			zScore := (dp.Value - mean) / stdDev
			pValue := 2 * (1 - normalCDF(math.Abs(zScore)))

			if pValue <= (1 - confidenceThreshold) {
				signal := SafetySignal{
					SignalID:               uuid.New().String(),
					DrugCode:               drugCode,
					SignalType:             "OUT_OF_CONTROL",
					SignalStrength:         math.Abs(zScore),
					PValue:                 pValue,
					ConfidenceIntervalLower: dp.Value - 1.96*stdDev/math.Sqrt(float64(len(data))),
					ConfidenceIntervalUpper: dp.Value + 1.96*stdDev/math.Sqrt(float64(len(data))),
					ClinicalDescription:    "Value exceeds statistical control limits",
					AffectedPopulations:    []string{"General"},
					SeverityAssessment:     assessSeverity(zScore),
					FirstDetected:          dp.Timestamp,
					LastUpdated:            time.Now(),
					OccurrenceCount:        1,
					DetectionMethod:        AlgorithmSPC,
					DetectionParameters: map[string]interface{}{
						"mean":   mean,
						"stdDev": stdDev,
						"UCL":    limits.UpperControlLimit,
						"LCL":    limits.LowerControlLimit,
					},
				}
				signals = append(signals, signal)
			}
		}
	}

	// Check for runs (7 consecutive points on one side of center line)
	signals = append(signals, sd.detectRuns(drugCode, data, mean)...)

	// Check for trends (7 consecutive increasing or decreasing points)
	signals = append(signals, sd.detectTrends(drugCode, data)...)

	return signals
}

// detectCUSUM performs Cumulative Sum detection
func (sd *SignalDetector) detectCUSUM(drugCode string, data []DataPoint, confidenceThreshold float64) []SafetySignal {
	if len(data) < 5 {
		return nil
	}

	values := extractValues(data)
	targetMean := calculateMean(values[:len(values)/2]) // Use first half as baseline
	stdDev := calculateStdDev(values[:len(values)/2], targetMean)

	// CUSUM parameters
	k := 0.5 * stdDev // Slack parameter (typically 0.5 sigma)
	h := 5.0 * stdDev // Decision threshold (typically 4-5 sigma)

	var signals []SafetySignal
	cusumPlus := 0.0
	cusumMinus := 0.0

	for i, dp := range data {
		// Update CUSUM statistics
		cusumPlus = math.Max(0, cusumPlus+(dp.Value-targetMean-k))
		cusumMinus = math.Max(0, cusumMinus-(dp.Value-targetMean)+k)

		// Check for shift detection
		if cusumPlus > h {
			signal := SafetySignal{
				SignalID:            uuid.New().String(),
				DrugCode:            drugCode,
				SignalType:          "CUSUM_SHIFT_UP",
				SignalStrength:      cusumPlus / h,
				PValue:              estimateCUSUMPValue(cusumPlus, h),
				ClinicalDescription: "Upward shift detected in process mean",
				AffectedPopulations: []string{"General"},
				SeverityAssessment:  assessCUSUMSeverity(cusumPlus, h),
				FirstDetected:       data[i].Timestamp,
				LastUpdated:         time.Now(),
				OccurrenceCount:     1,
				DetectionMethod:     AlgorithmCUSUM,
				DetectionParameters: map[string]interface{}{
					"cusumValue": cusumPlus,
					"threshold":  h,
					"targetMean": targetMean,
					"k":          k,
				},
			}
			signals = append(signals, signal)
			cusumPlus = 0 // Reset after detection
		}

		if cusumMinus > h {
			signal := SafetySignal{
				SignalID:            uuid.New().String(),
				DrugCode:            drugCode,
				SignalType:          "CUSUM_SHIFT_DOWN",
				SignalStrength:      cusumMinus / h,
				PValue:              estimateCUSUMPValue(cusumMinus, h),
				ClinicalDescription: "Downward shift detected in process mean",
				AffectedPopulations: []string{"General"},
				SeverityAssessment:  assessCUSUMSeverity(cusumMinus, h),
				FirstDetected:       data[i].Timestamp,
				LastUpdated:         time.Now(),
				OccurrenceCount:     1,
				DetectionMethod:     AlgorithmCUSUM,
				DetectionParameters: map[string]interface{}{
					"cusumValue": cusumMinus,
					"threshold":  h,
					"targetMean": targetMean,
					"k":          k,
				},
			}
			signals = append(signals, signal)
			cusumMinus = 0 // Reset after detection
		}
	}

	return signals
}

// detectEWMA performs Exponentially Weighted Moving Average detection
func (sd *SignalDetector) detectEWMA(drugCode string, data []DataPoint, confidenceThreshold float64) []SafetySignal {
	if len(data) < 3 {
		return nil
	}

	lambda := 0.2 // Smoothing parameter (typically 0.1-0.3)
	values := extractValues(data)
	mean := calculateMean(values)
	stdDev := calculateStdDev(values, mean)

	var signals []SafetySignal
	ewma := mean // Initialize EWMA with process mean

	for i, dp := range data {
		// Update EWMA
		ewma = lambda*dp.Value + (1-lambda)*ewma

		// Calculate control limits (they narrow over time)
		L := 3.0 // Control limit multiplier
		sigmaEWMA := stdDev * math.Sqrt(lambda/(2-lambda)*(1-math.Pow(1-lambda, 2*float64(i+1))))
		ucl := mean + L*sigmaEWMA
		lcl := mean - L*sigmaEWMA

		// Check for out of control
		if ewma > ucl || ewma < lcl {
			signal := SafetySignal{
				SignalID:               uuid.New().String(),
				DrugCode:               drugCode,
				SignalType:             "EWMA_OUT_OF_CONTROL",
				SignalStrength:         math.Abs(ewma-mean) / sigmaEWMA,
				PValue:                 2 * (1 - normalCDF(math.Abs(ewma-mean)/sigmaEWMA)),
				ConfidenceIntervalLower: ewma - 1.96*sigmaEWMA,
				ConfidenceIntervalUpper: ewma + 1.96*sigmaEWMA,
				ClinicalDescription:    "EWMA indicates process shift",
				AffectedPopulations:    []string{"General"},
				SeverityAssessment:     assessSeverity(math.Abs(ewma-mean) / sigmaEWMA),
				FirstDetected:          dp.Timestamp,
				LastUpdated:            time.Now(),
				OccurrenceCount:        1,
				DetectionMethod:        AlgorithmEWMA,
				DetectionParameters: map[string]interface{}{
					"ewmaValue": ewma,
					"lambda":    lambda,
					"ucl":       ucl,
					"lcl":       lcl,
					"mean":      mean,
				},
			}
			signals = append(signals, signal)
		}
	}

	return signals
}

// detectMLAnomaly performs simple ML-based anomaly detection
func (sd *SignalDetector) detectMLAnomaly(drugCode string, data []DataPoint, confidenceThreshold float64) []SafetySignal {
	if len(data) < 10 {
		return nil
	}

	// Use IQR-based anomaly detection (robust to outliers)
	values := extractValues(data)
	sort.Float64s(values)

	q1 := percentile(values, 25)
	q3 := percentile(values, 75)
	iqr := q3 - q1

	lowerFence := q1 - 1.5*iqr
	upperFence := q3 + 1.5*iqr

	// Severe outliers (3*IQR)
	lowerSevereFence := q1 - 3*iqr
	upperSevereFence := q3 + 3*iqr

	var signals []SafetySignal
	for _, dp := range data {
		if dp.Value < lowerSevereFence || dp.Value > upperSevereFence {
			// Severe outlier
			signal := SafetySignal{
				SignalID:            uuid.New().String(),
				DrugCode:            drugCode,
				SignalType:          "SEVERE_OUTLIER",
				SignalStrength:      math.Abs(dp.Value-((q1+q3)/2)) / iqr,
				PValue:              0.001, // Approximate
				ClinicalDescription: "Severe outlier detected (>3*IQR)",
				AffectedPopulations: []string{"General"},
				SeverityAssessment:  "CRITICAL",
				FirstDetected:       dp.Timestamp,
				LastUpdated:         time.Now(),
				OccurrenceCount:     1,
				DetectionMethod:     AlgorithmMLAnomaly,
				DetectionParameters: map[string]interface{}{
					"q1":         q1,
					"q3":         q3,
					"iqr":        iqr,
					"lowerFence": lowerSevereFence,
					"upperFence": upperSevereFence,
				},
			}
			signals = append(signals, signal)
		} else if dp.Value < lowerFence || dp.Value > upperFence {
			// Mild outlier
			signal := SafetySignal{
				SignalID:            uuid.New().String(),
				DrugCode:            drugCode,
				SignalType:          "MILD_OUTLIER",
				SignalStrength:      math.Abs(dp.Value-((q1+q3)/2)) / iqr,
				PValue:              0.05, // Approximate
				ClinicalDescription: "Mild outlier detected (>1.5*IQR)",
				AffectedPopulations: []string{"General"},
				SeverityAssessment:  "MODERATE",
				FirstDetected:       dp.Timestamp,
				LastUpdated:         time.Now(),
				OccurrenceCount:     1,
				DetectionMethod:     AlgorithmMLAnomaly,
				DetectionParameters: map[string]interface{}{
					"q1":         q1,
					"q3":         q3,
					"iqr":        iqr,
					"lowerFence": lowerFence,
					"upperFence": upperFence,
				},
			}
			signals = append(signals, signal)
		}
	}

	return signals
}

// detectRuns detects runs in SPC
func (sd *SignalDetector) detectRuns(drugCode string, data []DataPoint, centerLine float64) []SafetySignal {
	var signals []SafetySignal
	runLength := 7 // Western Electric rule

	aboveCount := 0
	belowCount := 0
	runStart := 0

	for i, dp := range data {
		if dp.Value > centerLine {
			aboveCount++
			belowCount = 0
			if aboveCount == 1 {
				runStart = i
			}
		} else {
			belowCount++
			aboveCount = 0
			if belowCount == 1 {
				runStart = i
			}
		}

		if aboveCount >= runLength || belowCount >= runLength {
			signal := SafetySignal{
				SignalID:            uuid.New().String(),
				DrugCode:            drugCode,
				SignalType:          "RUN_DETECTED",
				SignalStrength:      float64(max(aboveCount, belowCount)) / float64(runLength),
				PValue:              math.Pow(0.5, float64(runLength)), // Probability of run by chance
				ClinicalDescription: "Run detected - consecutive points on one side of center line",
				SeverityAssessment:  "MODERATE",
				FirstDetected:       data[runStart].Timestamp,
				LastUpdated:         time.Now(),
				OccurrenceCount:     max(aboveCount, belowCount),
				DetectionMethod:     AlgorithmSPC,
				DetectionParameters: map[string]interface{}{
					"runLength":  max(aboveCount, belowCount),
					"direction":  boolToString(aboveCount > belowCount, "above", "below"),
					"centerLine": centerLine,
				},
			}
			signals = append(signals, signal)

			// Reset counters
			aboveCount = 0
			belowCount = 0
		}
	}

	return signals
}

// detectTrends detects trends in SPC
func (sd *SignalDetector) detectTrends(drugCode string, data []DataPoint) []SafetySignal {
	var signals []SafetySignal
	trendLength := 7 // Western Electric rule

	increasingCount := 0
	decreasingCount := 0
	trendStart := 0

	for i := 1; i < len(data); i++ {
		if data[i].Value > data[i-1].Value {
			increasingCount++
			if decreasingCount > 0 {
				decreasingCount = 0
				trendStart = i - 1
			}
		} else if data[i].Value < data[i-1].Value {
			decreasingCount++
			if increasingCount > 0 {
				increasingCount = 0
				trendStart = i - 1
			}
		}

		if increasingCount >= trendLength-1 || decreasingCount >= trendLength-1 {
			signal := SafetySignal{
				SignalID:            uuid.New().String(),
				DrugCode:            drugCode,
				SignalType:          "TREND_DETECTED",
				SignalStrength:      float64(max(increasingCount, decreasingCount)) / float64(trendLength),
				PValue:              math.Pow(0.5, float64(trendLength-1)), // Probability of trend by chance
				ClinicalDescription: "Trend detected - consecutive increasing/decreasing points",
				SeverityAssessment:  "HIGH",
				FirstDetected:       data[trendStart].Timestamp,
				LastUpdated:         time.Now(),
				OccurrenceCount:     max(increasingCount, decreasingCount) + 1,
				DetectionMethod:     AlgorithmSPC,
				DetectionParameters: map[string]interface{}{
					"trendLength": max(increasingCount, decreasingCount) + 1,
					"direction":   boolToString(increasingCount > decreasingCount, "increasing", "decreasing"),
				},
			}
			signals = append(signals, signal)

			// Reset counters
			increasingCount = 0
			decreasingCount = 0
		}
	}

	return signals
}

// AddDataPoint adds a new data point for signal detection
func (sd *SignalDetector) AddDataPoint(dp DataPoint) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	key := dp.DrugCode
	sd.historicalData[key] = append(sd.historicalData[key], dp)

	// Keep only last 1000 points per drug
	if len(sd.historicalData[key]) > 1000 {
		sd.historicalData[key] = sd.historicalData[key][len(sd.historicalData[key])-1000:]
	}
}

// getDataForDrug retrieves historical data for a drug
func (sd *SignalDetector) getDataForDrug(drugCode string, start, end time.Time) []DataPoint {
	data := sd.historicalData[drugCode]
	var filtered []DataPoint
	for _, dp := range data {
		if (dp.Timestamp.Equal(start) || dp.Timestamp.After(start)) &&
			(dp.Timestamp.Equal(end) || dp.Timestamp.Before(end)) {
			filtered = append(filtered, dp)
		}
	}
	return filtered
}

// GetControlLimits returns the control limits for a drug
func (sd *SignalDetector) GetControlLimits(drugCode string) *ControlLimits {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.controlLimits[drugCode]
}

// Helper functions

func extractValues(data []DataPoint) []float64 {
	values := make([]float64, len(data))
	for i, dp := range data {
		values[i] = dp.Value
	}
	return values
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		sumSquares += (v - mean) * (v - mean)
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper {
		return sorted[lower]
	}
	return sorted[lower] + (index-float64(lower))*(sorted[upper]-sorted[lower])
}

func normalCDF(x float64) float64 {
	// Approximation of standard normal CDF
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

func assessSeverity(zScore float64) string {
	absZ := math.Abs(zScore)
	switch {
	case absZ >= 4:
		return "CRITICAL"
	case absZ >= 3:
		return "HIGH"
	case absZ >= 2:
		return "MODERATE"
	default:
		return "LOW"
	}
}

func assessCUSUMSeverity(cusum, threshold float64) string {
	ratio := cusum / threshold
	switch {
	case ratio >= 2:
		return "CRITICAL"
	case ratio >= 1.5:
		return "HIGH"
	case ratio >= 1:
		return "MODERATE"
	default:
		return "LOW"
	}
}

func estimateCUSUMPValue(cusum, threshold float64) float64 {
	// Approximate p-value based on CUSUM statistic
	ratio := cusum / threshold
	return math.Exp(-2 * ratio * ratio)
}

func boolToString(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
