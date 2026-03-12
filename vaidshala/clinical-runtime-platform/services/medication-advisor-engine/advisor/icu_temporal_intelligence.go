// Package advisor provides ICU Temporal Intelligence.
// This file implements Tier-10 Phase 3: Continuous Monitoring Intelligence.
//
// The Temporal Intelligence Brain provides:
// - Trend analysis for all 8 clinical dimensions
// - Deterioration detection algorithms (NEWS, MEWS integration points)
// - Time-series pattern recognition for medication safety
// - Predictive alerts based on trajectory
// - State transition tracking and analysis
package advisor

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Temporal Intelligence Types
// ============================================================================

// ICUTemporalState represents a time-series of ICU states
type ICUTemporalState struct {
	PatientID       uuid.UUID                `json:"patient_id"`
	EncounterID     uuid.UUID                `json:"encounter_id"`
	CurrentState    *ICUClinicalState        `json:"current_state"`
	HistoricalStates []ICUClinicalState      `json:"historical_states"`
	Trends          DimensionTrends          `json:"trends"`
	Alerts          []TemporalAlert          `json:"alerts"`
	Predictions     []DeteriorationPrediction `json:"predictions"`
	LastAnalyzed    time.Time                `json:"last_analyzed"`
}

// DimensionTrends holds trends for all 8 dimensions
type DimensionTrends struct {
	Hemodynamic   TrendAnalysis `json:"hemodynamic"`
	Respiratory   TrendAnalysis `json:"respiratory"`
	Renal         TrendAnalysis `json:"renal"`
	Hepatic       TrendAnalysis `json:"hepatic"`
	Coagulation   TrendAnalysis `json:"coagulation"`
	Neurological  TrendAnalysis `json:"neurological"`
	FluidBalance  TrendAnalysis `json:"fluid_balance"`
	Infection     TrendAnalysis `json:"infection"`
	Composite     TrendAnalysis `json:"composite"`
}

// TrendAnalysis represents trend data for a dimension
type TrendAnalysis struct {
	Dimension       string           `json:"dimension"`
	Direction       TrendDirection   `json:"direction"`
	Slope           float64          `json:"slope"`           // Rate of change
	Velocity        float64          `json:"velocity"`        // Speed of change
	Acceleration    float64          `json:"acceleration"`    // Change in velocity
	Confidence      float64          `json:"confidence"`      // 0-1 confidence in trend
	DataPoints      int              `json:"data_points"`     // Number of points analyzed
	TimeSpan        time.Duration    `json:"time_span"`       // Duration of analysis
	CurrentValue    float64          `json:"current_value"`
	PreviousValue   float64          `json:"previous_value"`
	MinValue        float64          `json:"min_value"`
	MaxValue        float64          `json:"max_value"`
	Mean            float64          `json:"mean"`
	StdDev          float64          `json:"std_dev"`
	LastUpdated     time.Time        `json:"last_updated"`
	KeyPoints       []TrendKeyPoint  `json:"key_points,omitempty"`
}

// TrendKeyPoint represents a significant point in trend data
type TrendKeyPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	EventType string    `json:"event_type"` // peak, trough, inflection, threshold_crossed
}

// TemporalAlert represents an alert based on temporal patterns
type TemporalAlert struct {
	ID             uuid.UUID          `json:"id"`
	AlertType      TemporalAlertType  `json:"alert_type"`
	Dimension      string             `json:"dimension"`
	Severity       AlertSeverity      `json:"severity"`
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	TriggerTrend   TrendDirection     `json:"trigger_trend"`
	CurrentValue   float64            `json:"current_value"`
	ProjectedValue float64            `json:"projected_value,omitempty"`
	TimeToThreshold *time.Duration    `json:"time_to_threshold,omitempty"`
	Recommendation string             `json:"recommendation"`
	MedicationImpact []string         `json:"medication_impact,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

// TemporalAlertType categorizes temporal alerts
type TemporalAlertType string

const (
	AlertRapidDeterioration TemporalAlertType = "RAPID_DETERIORATION"
	AlertTrendingTowardCritical TemporalAlertType = "TRENDING_TOWARD_CRITICAL"
	AlertUnstableVitals TemporalAlertType = "UNSTABLE_VITALS"
	AlertFailingToImprove TemporalAlertType = "FAILING_TO_IMPROVE"
	AlertAcuteChange TemporalAlertType = "ACUTE_CHANGE"
	AlertThresholdApproaching TemporalAlertType = "THRESHOLD_APPROACHING"
	AlertPatternAnomaly TemporalAlertType = "PATTERN_ANOMALY"
)

// DeteriorationPrediction represents a predicted deterioration
type DeteriorationPrediction struct {
	ID             uuid.UUID       `json:"id"`
	Dimension      string          `json:"dimension"`
	PredictedEvent string          `json:"predicted_event"`
	Probability    float64         `json:"probability"`
	TimeHorizon    time.Duration   `json:"time_horizon"`
	CurrentTrajectory string       `json:"current_trajectory"`
	RiskFactors    []string        `json:"risk_factors"`
	MedicationRisks []MedTrendRisk `json:"medication_risks,omitempty"`
	PreventiveActions []string     `json:"preventive_actions,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// MedTrendRisk represents medication risk based on trends
type MedTrendRisk struct {
	Medication     ClinicalCode    `json:"medication"`
	RiskType       string          `json:"risk_type"`
	RiskLevel      RiskLevel       `json:"risk_level"`
	TrendImpact    string          `json:"trend_impact"`
	Recommendation string          `json:"recommendation"`
}

// ============================================================================
// Temporal Intelligence Engine
// ============================================================================

// ICUTemporalEngine analyzes ICU state trends and predicts deterioration
type ICUTemporalEngine struct {
	analysisWindow   time.Duration  // How far back to analyze
	predictionWindow time.Duration  // How far ahead to predict
	minDataPoints    int            // Minimum points for trend analysis
	thresholds       TemporalThresholds
}

// TemporalThresholds defines thresholds for temporal alerts
type TemporalThresholds struct {
	// Rapid change thresholds (per hour)
	MAPDropRateAlarm      float64 // mmHg/hr
	SpO2DropRateAlarm     float64 // %/hr
	CreatRiseRateAlarm    float64 // mg/dL/hr
	LactateRiseRateAlarm  float64 // mmol/L/hr
	GCSDropRateAlarm      float64 // points/hr
	TempRiseRateAlarm     float64 // °C/hr

	// Velocity thresholds (acceleration)
	AccelerationThreshold float64 // Rate of rate change

	// Stability thresholds (coefficient of variation)
	CVThresholdUnstable   float64 // CV > this = unstable
	CVThresholdCritical   float64 // CV > this = critically unstable
}

// NewICUTemporalEngine creates a new temporal analysis engine
func NewICUTemporalEngine() *ICUTemporalEngine {
	return &ICUTemporalEngine{
		analysisWindow:   24 * time.Hour,
		predictionWindow: 6 * time.Hour,
		minDataPoints:    3,
		thresholds: TemporalThresholds{
			MAPDropRateAlarm:      10.0,  // 10 mmHg/hr drop
			SpO2DropRateAlarm:     3.0,   // 3%/hr drop
			CreatRiseRateAlarm:    0.3,   // 0.3 mg/dL/hr rise
			LactateRiseRateAlarm:  0.5,   // 0.5 mmol/L/hr rise
			GCSDropRateAlarm:      2.0,   // 2 points/hr drop
			TempRiseRateAlarm:     1.0,   // 1°C/hr rise
			AccelerationThreshold: 0.5,   // Significant acceleration
			CVThresholdUnstable:   0.15,  // 15% CV
			CVThresholdCritical:   0.25,  // 25% CV
		},
	}
}

// AnalyzeTemporalState performs comprehensive temporal analysis
func (e *ICUTemporalEngine) AnalyzeTemporalState(temporal *ICUTemporalState) *TemporalAnalysisResult {
	if temporal.CurrentState == nil || len(temporal.HistoricalStates) < e.minDataPoints {
		return &TemporalAnalysisResult{
			Sufficient: false,
			Message:    fmt.Sprintf("Insufficient data points. Need %d, have %d", e.minDataPoints, len(temporal.HistoricalStates)),
		}
	}

	result := &TemporalAnalysisResult{
		Sufficient:      true,
		AnalyzedAt:      time.Now(),
		DataPointCount:  len(temporal.HistoricalStates) + 1,
		AnalysisWindow:  e.analysisWindow,
	}

	// Sort historical states by time
	states := append(temporal.HistoricalStates, *temporal.CurrentState)
	sort.Slice(states, func(i, j int) bool {
		return states[i].CapturedAt.Before(states[j].CapturedAt)
	})

	// Analyze each dimension
	result.Trends.Hemodynamic = e.analyzeHemodynamicTrend(states)
	result.Trends.Respiratory = e.analyzeRespiratoryTrend(states)
	result.Trends.Renal = e.analyzeRenalTrend(states)
	result.Trends.Hepatic = e.analyzeHepaticTrend(states)
	result.Trends.Coagulation = e.analyzeCoagulationTrend(states)
	result.Trends.Neurological = e.analyzeNeurologicalTrend(states)
	result.Trends.FluidBalance = e.analyzeFluidBalanceTrend(states)
	result.Trends.Infection = e.analyzeInfectionTrend(states)
	result.Trends.Composite = e.analyzeCompositeTrend(states)

	// Generate temporal alerts
	result.Alerts = e.generateTemporalAlerts(result.Trends)

	// Generate deterioration predictions
	result.Predictions = e.generatePredictions(result.Trends, temporal.CurrentState)

	// Calculate overall trajectory
	result.OverallTrajectory = e.calculateOverallTrajectory(result.Trends)

	// Update temporal state
	temporal.Trends = result.Trends
	temporal.Alerts = result.Alerts
	temporal.Predictions = result.Predictions
	temporal.LastAnalyzed = result.AnalyzedAt

	return result
}

// TemporalAnalysisResult contains the full analysis output
type TemporalAnalysisResult struct {
	Sufficient        bool                     `json:"sufficient"`
	Message           string                   `json:"message,omitempty"`
	AnalyzedAt        time.Time                `json:"analyzed_at"`
	DataPointCount    int                      `json:"data_point_count"`
	AnalysisWindow    time.Duration            `json:"analysis_window"`
	Trends            DimensionTrends          `json:"trends"`
	Alerts            []TemporalAlert          `json:"alerts"`
	Predictions       []DeteriorationPrediction `json:"predictions"`
	OverallTrajectory TrajectoryAssessment     `json:"overall_trajectory"`
}

// TrajectoryAssessment summarizes overall patient trajectory
type TrajectoryAssessment struct {
	Direction        TrendDirection `json:"direction"`
	Stability        string         `json:"stability"` // stable, unstable, critical
	DeteriorationRisk float64       `json:"deterioration_risk"` // 0-1
	PrimaryRiskDimension string     `json:"primary_risk_dimension"`
	TimeToIntervention *time.Duration `json:"time_to_intervention,omitempty"`
	Recommendation    string         `json:"recommendation"`
}

// ============================================================================
// Dimension-Specific Trend Analyzers
// ============================================================================

func (e *ICUTemporalEngine) analyzeHemodynamicTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		values[i] = s.Hemodynamic.MAP
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("hemodynamic", values, timestamps)

	// Add hemodynamic-specific key points
	analysis.KeyPoints = e.findKeyPoints(values, timestamps, 65.0, 90.0) // MAP thresholds

	return analysis
}

func (e *ICUTemporalEngine) analyzeRespiratoryTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		values[i] = s.Respiratory.SpO2
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("respiratory", values, timestamps)
	analysis.KeyPoints = e.findKeyPoints(values, timestamps, 88.0, 95.0) // SpO2 thresholds

	return analysis
}

func (e *ICUTemporalEngine) analyzeRenalTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		values[i] = s.Renal.Creatinine
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("renal", values, timestamps)
	// For creatinine, higher is worse so invert direction interpretation
	if analysis.Direction == TrendImproving && analysis.Slope > 0 {
		analysis.Direction = TrendDeteriorating
	} else if analysis.Direction == TrendDeteriorating && analysis.Slope < 0 {
		analysis.Direction = TrendImproving
	}

	return analysis
}

func (e *ICUTemporalEngine) analyzeHepaticTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		values[i] = s.Hepatic.TotalBilirubin
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("hepatic", values, timestamps)
	// Higher bilirubin = worse
	if analysis.Direction == TrendImproving && analysis.Slope > 0 {
		analysis.Direction = TrendDeteriorating
	} else if analysis.Direction == TrendDeteriorating && analysis.Slope < 0 {
		analysis.Direction = TrendImproving
	}

	return analysis
}

func (e *ICUTemporalEngine) analyzeCoagulationTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		values[i] = s.Coagulation.INR
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("coagulation", values, timestamps)
	return analysis
}

func (e *ICUTemporalEngine) analyzeNeurologicalTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		values[i] = float64(s.Neurological.GCS)
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("neurological", values, timestamps)
	return analysis
}

func (e *ICUTemporalEngine) analyzeFluidBalanceTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		values[i] = s.FluidBalance.NetBalance24h
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("fluid_balance", values, timestamps)
	return analysis
}

func (e *ICUTemporalEngine) analyzeInfectionTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		// Use WBC as primary infection marker
		values[i] = s.Infection.WBC
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("infection", values, timestamps)
	return analysis
}

func (e *ICUTemporalEngine) analyzeCompositeTrend(states []ICUClinicalState) TrendAnalysis {
	values := make([]float64, len(states))
	timestamps := make([]time.Time, len(states))

	for i, s := range states {
		values[i] = s.ICUAcuityScore
		timestamps[i] = s.CapturedAt
	}

	analysis := e.calculateTrendStats("composite", values, timestamps)
	return analysis
}

// ============================================================================
// Statistical Trend Calculations
// ============================================================================

func (e *ICUTemporalEngine) calculateTrendStats(dimension string, values []float64, timestamps []time.Time) TrendAnalysis {
	n := len(values)
	if n < 2 {
		return TrendAnalysis{
			Dimension:  dimension,
			Direction:  TrendUnknown,
			Confidence: 0,
			DataPoints: n,
		}
	}

	// Basic statistics
	mean := calculateMean(values)
	stdDev := calculateStdDev(values, mean)
	minVal, maxVal := findMinMax(values)

	// Calculate slope using linear regression
	hours := make([]float64, n)
	baseTime := timestamps[0]
	for i, t := range timestamps {
		hours[i] = t.Sub(baseTime).Hours()
	}
	slope, _ := linearRegression(hours, values)

	// Calculate velocity (recent slope vs historical slope)
	velocity := 0.0
	if n >= 4 {
		midpoint := n / 2
		earlySlope, _ := linearRegression(hours[:midpoint], values[:midpoint])
		recentSlope, _ := linearRegression(hours[midpoint:], values[midpoint:])
		velocity = recentSlope - earlySlope
	}

	// Calculate acceleration (rate of velocity change)
	acceleration := 0.0
	if n >= 6 {
		// Three-way split for acceleration
		third := n / 3
		slope1, _ := linearRegression(hours[:third], values[:third])
		slope2, _ := linearRegression(hours[third:2*third], values[third:2*third])
		slope3, _ := linearRegression(hours[2*third:], values[2*third:])
		velocity1 := slope2 - slope1
		velocity2 := slope3 - slope2
		acceleration = velocity2 - velocity1
	}

	// Determine trend direction
	direction := e.determineTrendDirection(slope, velocity, stdDev)

	// Calculate confidence based on data quality
	cv := 0.0
	if mean != 0 {
		cv = stdDev / math.Abs(mean)
	}
	confidence := 1.0 - math.Min(cv, 1.0) // Higher CV = lower confidence

	timeSpan := time.Duration(0)
	if n > 1 {
		timeSpan = timestamps[n-1].Sub(timestamps[0])
	}

	return TrendAnalysis{
		Dimension:     dimension,
		Direction:     direction,
		Slope:         slope,
		Velocity:      velocity,
		Acceleration:  acceleration,
		Confidence:    confidence,
		DataPoints:    n,
		TimeSpan:      timeSpan,
		CurrentValue:  values[n-1],
		PreviousValue: values[n-2],
		MinValue:      minVal,
		MaxValue:      maxVal,
		Mean:          mean,
		StdDev:        stdDev,
		LastUpdated:   timestamps[n-1],
	}
}

func (e *ICUTemporalEngine) determineTrendDirection(slope, velocity, stdDev float64) TrendDirection {
	// Use slope magnitude relative to standard deviation
	slopeThreshold := stdDev * 0.1 // 10% of std dev per hour

	if math.Abs(slope) < slopeThreshold {
		return TrendStable
	}

	// Check for critical deterioration (rapid negative trend with acceleration)
	if slope < -slopeThreshold*3 || velocity < -slopeThreshold*2 {
		return TrendCritical
	}

	if slope > slopeThreshold {
		return TrendImproving
	}

	return TrendDeteriorating
}

func (e *ICUTemporalEngine) findKeyPoints(values []float64, timestamps []time.Time, lowThreshold, highThreshold float64) []TrendKeyPoint {
	keyPoints := []TrendKeyPoint{}

	for i := 1; i < len(values)-1; i++ {
		// Local peak
		if values[i] > values[i-1] && values[i] > values[i+1] {
			keyPoints = append(keyPoints, TrendKeyPoint{
				Timestamp: timestamps[i],
				Value:     values[i],
				EventType: "peak",
			})
		}

		// Local trough
		if values[i] < values[i-1] && values[i] < values[i+1] {
			keyPoints = append(keyPoints, TrendKeyPoint{
				Timestamp: timestamps[i],
				Value:     values[i],
				EventType: "trough",
			})
		}

		// Threshold crossings
		if i > 0 {
			// Crossed below low threshold
			if values[i-1] >= lowThreshold && values[i] < lowThreshold {
				keyPoints = append(keyPoints, TrendKeyPoint{
					Timestamp: timestamps[i],
					Value:     values[i],
					EventType: "threshold_crossed_low",
				})
			}
			// Crossed above high threshold
			if values[i-1] <= highThreshold && values[i] > highThreshold {
				keyPoints = append(keyPoints, TrendKeyPoint{
					Timestamp: timestamps[i],
					Value:     values[i],
					EventType: "threshold_crossed_high",
				})
			}
		}
	}

	return keyPoints
}

// ============================================================================
// Alert and Prediction Generation
// ============================================================================

func (e *ICUTemporalEngine) generateTemporalAlerts(trends DimensionTrends) []TemporalAlert {
	alerts := []TemporalAlert{}

	// Check hemodynamic trends
	if trends.Hemodynamic.Direction == TrendDeteriorating || trends.Hemodynamic.Direction == TrendCritical {
		if trends.Hemodynamic.Slope < -e.thresholds.MAPDropRateAlarm {
			alerts = append(alerts, TemporalAlert{
				ID:            uuid.New(),
				AlertType:     AlertRapidDeterioration,
				Dimension:     "hemodynamic",
				Severity:      AlertCritical,
				Title:         "Rapid MAP Decline",
				Description:   fmt.Sprintf("MAP dropping at %.1f mmHg/hr", -trends.Hemodynamic.Slope),
				TriggerTrend:  trends.Hemodynamic.Direction,
				CurrentValue:  trends.Hemodynamic.CurrentValue,
				Recommendation: "Assess volume status, consider vasopressor escalation",
				MedicationImpact: []string{"Hold antihypertensives", "Consider vasopressor increase"},
				CreatedAt:     time.Now(),
			})
		}
	}

	// Check respiratory trends
	if trends.Respiratory.Direction == TrendDeteriorating || trends.Respiratory.Direction == TrendCritical {
		if trends.Respiratory.Slope < -e.thresholds.SpO2DropRateAlarm {
			alerts = append(alerts, TemporalAlert{
				ID:            uuid.New(),
				AlertType:     AlertRapidDeterioration,
				Dimension:     "respiratory",
				Severity:      AlertCritical,
				Title:         "Rapid SpO2 Decline",
				Description:   fmt.Sprintf("SpO2 dropping at %.1f%%/hr", -trends.Respiratory.Slope),
				TriggerTrend:  trends.Respiratory.Direction,
				CurrentValue:  trends.Respiratory.CurrentValue,
				Recommendation: "Increase FiO2, assess airway, consider intubation if trending < 88%",
				MedicationImpact: []string{"Hold sedatives", "Prepare for intubation meds if needed"},
				CreatedAt:     time.Now(),
			})
		}
	}

	// Check renal trends (rising creatinine)
	if trends.Renal.Direction == TrendDeteriorating {
		if trends.Renal.Slope > e.thresholds.CreatRiseRateAlarm {
			alerts = append(alerts, TemporalAlert{
				ID:            uuid.New(),
				AlertType:     AlertTrendingTowardCritical,
				Dimension:     "renal",
				Severity:      AlertUrgent,
				Title:         "Rising Creatinine Trend",
				Description:   fmt.Sprintf("Creatinine rising at %.2f mg/dL/hr", trends.Renal.Slope),
				TriggerTrend:  trends.Renal.Direction,
				CurrentValue:  trends.Renal.CurrentValue,
				Recommendation: "Review nephrotoxic medications, optimize volume status",
				MedicationImpact: []string{"Avoid NSAIDs", "Renally dose all medications", "Consider contrast avoidance"},
				CreatedAt:     time.Now(),
			})
		}
	}

	// Check neurological trends (falling GCS)
	if trends.Neurological.Direction == TrendDeteriorating || trends.Neurological.Direction == TrendCritical {
		if trends.Neurological.Slope < -e.thresholds.GCSDropRateAlarm {
			alerts = append(alerts, TemporalAlert{
				ID:            uuid.New(),
				AlertType:     AlertAcuteChange,
				Dimension:     "neurological",
				Severity:      AlertCritical,
				Title:         "Acute GCS Decline",
				Description:   fmt.Sprintf("GCS dropping at %.1f points/hr", -trends.Neurological.Slope),
				TriggerTrend:  trends.Neurological.Direction,
				CurrentValue:  trends.Neurological.CurrentValue,
				Recommendation: "Stat neuro assessment, head CT if indicated, secure airway if GCS ≤ 8",
				MedicationImpact: []string{"Hold sedatives for assessment", "Prepare airway medications"},
				CreatedAt:     time.Now(),
			})
		}
	}

	// Check infection trends (rising temperature or WBC with lactate)
	if trends.Infection.Direction == TrendDeteriorating {
		alerts = append(alerts, TemporalAlert{
			ID:            uuid.New(),
			AlertType:     AlertTrendingTowardCritical,
			Dimension:     "infection",
			Severity:      AlertWarning,
			Title:         "Worsening Infection Markers",
			Description:   "Infection markers trending worse",
			TriggerTrend:  trends.Infection.Direction,
			CurrentValue:  trends.Infection.CurrentValue,
			Recommendation: "Review antibiotic coverage, consider broadening if deteriorating",
			MedicationImpact: []string{"Ensure adequate antibiotic dosing", "Consider source control"},
			CreatedAt:     time.Now(),
		})
	}

	// Check composite acuity
	if trends.Composite.Direction == TrendCritical {
		alerts = append(alerts, TemporalAlert{
			ID:            uuid.New(),
			AlertType:     AlertRapidDeterioration,
			Dimension:     "composite",
			Severity:      AlertCritical,
			Title:         "Multi-Organ Deterioration",
			Description:   "Overall ICU acuity rapidly worsening",
			TriggerTrend:  trends.Composite.Direction,
			CurrentValue:  trends.Composite.CurrentValue,
			Recommendation: "Urgent ICU team bedside assessment. Consider goals of care discussion.",
			MedicationImpact: []string{"Review all medications", "Pharmacy consultation required"},
			CreatedAt:     time.Now(),
		})
	}

	// Check for instability (high coefficient of variation)
	for _, t := range []TrendAnalysis{trends.Hemodynamic, trends.Respiratory, trends.Neurological} {
		cv := 0.0
		if t.Mean != 0 {
			cv = t.StdDev / math.Abs(t.Mean)
		}
		if cv > e.thresholds.CVThresholdCritical {
			alerts = append(alerts, TemporalAlert{
				ID:            uuid.New(),
				AlertType:     AlertUnstableVitals,
				Dimension:     t.Dimension,
				Severity:      AlertUrgent,
				Title:         fmt.Sprintf("Critically Unstable %s", t.Dimension),
				Description:   fmt.Sprintf("High variability (CV: %.1f%%) indicates instability", cv*100),
				TriggerTrend:  t.Direction,
				CurrentValue:  t.CurrentValue,
				Recommendation: "Frequent reassessment required. Avoid PRN medications.",
				CreatedAt:     time.Now(),
			})
		}
	}

	return alerts
}

func (e *ICUTemporalEngine) generatePredictions(trends DimensionTrends, current *ICUClinicalState) []DeteriorationPrediction {
	predictions := []DeteriorationPrediction{}

	// Predict hemodynamic deterioration
	if trends.Hemodynamic.Direction == TrendDeteriorating && trends.Hemodynamic.Slope < 0 {
		hoursToThreshold := (trends.Hemodynamic.CurrentValue - 60) / (-trends.Hemodynamic.Slope)
		if hoursToThreshold > 0 && hoursToThreshold < 6 {
			predictions = append(predictions, DeteriorationPrediction{
				ID:               uuid.New(),
				Dimension:        "hemodynamic",
				PredictedEvent:   "MAP < 60 mmHg (Severe Hypotension)",
				Probability:      calculateProbability(hoursToThreshold, trends.Hemodynamic.Confidence),
				TimeHorizon:      time.Duration(hoursToThreshold) * time.Hour,
				CurrentTrajectory: fmt.Sprintf("MAP %.1f → 60 mmHg in ~%.1f hours", trends.Hemodynamic.CurrentValue, hoursToThreshold),
				RiskFactors:       []string{"Trending MAP decline", "Vasopressor dependence"},
				PreventiveActions: []string{"Consider vasopressor escalation", "Volume resuscitation if appropriate"},
				CreatedAt:        time.Now(),
			})
		}
	}

	// Predict respiratory failure
	if trends.Respiratory.Direction == TrendDeteriorating && trends.Respiratory.Slope < 0 {
		hoursToThreshold := (trends.Respiratory.CurrentValue - 85) / (-trends.Respiratory.Slope)
		if hoursToThreshold > 0 && hoursToThreshold < 6 {
			predictions = append(predictions, DeteriorationPrediction{
				ID:               uuid.New(),
				Dimension:        "respiratory",
				PredictedEvent:   "SpO2 < 85% (Severe Hypoxemia)",
				Probability:      calculateProbability(hoursToThreshold, trends.Respiratory.Confidence),
				TimeHorizon:      time.Duration(hoursToThreshold) * time.Hour,
				CurrentTrajectory: fmt.Sprintf("SpO2 %.1f%% → 85%% in ~%.1f hours", trends.Respiratory.CurrentValue, hoursToThreshold),
				RiskFactors:       []string{"Declining oxygenation", "High FiO2 requirements"},
				PreventiveActions: []string{"Increase respiratory support", "Prepare for intubation"},
				MedicationRisks: []MedTrendRisk{
					{
						Medication:     ClinicalCode{Display: "Sedatives"},
						RiskType:       "Respiratory depression",
						RiskLevel:      RiskHigh,
						TrendImpact:    "Will accelerate SpO2 decline",
						Recommendation: "Avoid or use minimal sedation",
					},
				},
				CreatedAt: time.Now(),
			})
		}
	}

	// Predict AKI progression
	if trends.Renal.Direction == TrendDeteriorating && trends.Renal.Slope > 0 {
		hoursToThreshold := (4.0 - trends.Renal.CurrentValue) / trends.Renal.Slope // Creat > 4 threshold
		if hoursToThreshold > 0 && hoursToThreshold < 24 && trends.Renal.CurrentValue < 4.0 {
			predictions = append(predictions, DeteriorationPrediction{
				ID:               uuid.New(),
				Dimension:        "renal",
				PredictedEvent:   "Creatinine > 4 mg/dL (Severe AKI)",
				Probability:      calculateProbability(hoursToThreshold/4, trends.Renal.Confidence), // Slower events
				TimeHorizon:      time.Duration(hoursToThreshold) * time.Hour,
				CurrentTrajectory: fmt.Sprintf("Creatinine %.2f → 4.0 in ~%.1f hours", trends.Renal.CurrentValue, hoursToThreshold),
				RiskFactors:       []string{"Rising creatinine trend", "Nephrotoxic exposure"},
				PreventiveActions: []string{"Nephrology consult", "CRRT preparation"},
				MedicationRisks: []MedTrendRisk{
					{
						Medication:     ClinicalCode{Display: "Aminoglycosides"},
						RiskType:       "Nephrotoxicity",
						RiskLevel:      RiskCritical,
						TrendImpact:    "Will accelerate renal decline",
						Recommendation: "Avoid or switch to alternative",
					},
					{
						Medication:     ClinicalCode{Display: "NSAIDs"},
						RiskType:       "Nephrotoxicity",
						RiskLevel:      RiskCritical,
						TrendImpact:    "Will accelerate renal decline",
						Recommendation: "Contraindicated with trending AKI",
					},
				},
				CreatedAt: time.Now(),
			})
		}
	}

	return predictions
}

func (e *ICUTemporalEngine) calculateOverallTrajectory(trends DimensionTrends) TrajectoryAssessment {
	// Count deteriorating dimensions
	deteriorating := 0
	critical := 0
	primaryRisk := ""
	worstSlope := 0.0

	dimensions := []TrendAnalysis{
		trends.Hemodynamic, trends.Respiratory, trends.Renal,
		trends.Hepatic, trends.Coagulation, trends.Neurological,
		trends.FluidBalance, trends.Infection,
	}

	for _, t := range dimensions {
		if t.Direction == TrendDeteriorating {
			deteriorating++
			if math.Abs(t.Slope) > worstSlope {
				worstSlope = math.Abs(t.Slope)
				primaryRisk = t.Dimension
			}
		}
		if t.Direction == TrendCritical {
			critical++
			primaryRisk = t.Dimension
		}
	}

	// Determine overall direction
	direction := TrendStable
	if critical > 0 {
		direction = TrendCritical
	} else if deteriorating >= 3 {
		direction = TrendDeteriorating
	} else if trends.Composite.Direction == TrendImproving {
		direction = TrendImproving
	}

	// Determine stability
	stability := "stable"
	if critical > 0 || deteriorating >= 4 {
		stability = "critical"
	} else if deteriorating >= 2 {
		stability = "unstable"
	}

	// Calculate deterioration risk
	risk := float64(deteriorating) * 0.15
	if critical > 0 {
		risk += 0.4
	}
	if trends.Composite.Acceleration < -0.5 {
		risk += 0.2
	}
	if risk > 1.0 {
		risk = 1.0
	}

	// Determine recommendation
	var recommendation string
	switch {
	case critical > 0:
		recommendation = "CRITICAL: Immediate bedside assessment. Consider escalation of care and goals discussion."
	case deteriorating >= 3:
		recommendation = "URGENT: Multi-system deterioration. ICU team review and intervention planning required."
	case deteriorating >= 1:
		recommendation = "CAUTION: Single-system decline. Close monitoring and proactive intervention recommended."
	default:
		recommendation = "STABLE: Continue current management with routine monitoring."
	}

	return TrajectoryAssessment{
		Direction:            direction,
		Stability:            stability,
		DeteriorationRisk:    risk,
		PrimaryRiskDimension: primaryRisk,
		Recommendation:       recommendation,
	}
}

// ============================================================================
// Medication-Trend Safety Integration
// ============================================================================

// EvaluateMedicationWithTrends evaluates medication safety considering trends
func (e *ICUTemporalEngine) EvaluateMedicationWithTrends(
	medication ClinicalCode,
	temporal *ICUTemporalState,
	rulesEngine *ICUSafetyRulesEngine,
) *TemporalMedEvaluation {
	eval := &TemporalMedEvaluation{
		Medication:    medication,
		EvaluatedAt:   time.Now(),
		StaticSafe:    true,
		TrendSafe:     true,
		TrendWarnings: []string{},
	}

	if temporal.CurrentState == nil {
		return eval
	}

	// First, run static rules
	staticViolations := rulesEngine.EvaluateMedication(medication, temporal.CurrentState)
	if len(staticViolations) > 0 {
		eval.StaticSafe = false
		eval.StaticViolations = staticViolations
	}

	// Then, evaluate against trends
	trendRisks := e.evaluateMedAgainstTrends(medication, temporal.Trends)
	eval.TrendRisks = trendRisks

	for _, risk := range trendRisks {
		if risk.RiskLevel == RiskCritical || risk.RiskLevel == RiskHigh {
			eval.TrendSafe = false
			eval.TrendWarnings = append(eval.TrendWarnings, risk.TrendImpact)
		}
	}

	// Overall safety
	eval.OverallSafe = eval.StaticSafe && eval.TrendSafe

	return eval
}

// TemporalMedEvaluation combines static and trend-based evaluation
type TemporalMedEvaluation struct {
	Medication       ClinicalCode        `json:"medication"`
	EvaluatedAt      time.Time           `json:"evaluated_at"`
	StaticSafe       bool                `json:"static_safe"`
	TrendSafe        bool                `json:"trend_safe"`
	OverallSafe      bool                `json:"overall_safe"`
	StaticViolations []ICURuleViolation  `json:"static_violations,omitempty"`
	TrendRisks       []MedTrendRisk      `json:"trend_risks,omitempty"`
	TrendWarnings    []string            `json:"trend_warnings,omitempty"`
}

func (e *ICUTemporalEngine) evaluateMedAgainstTrends(med ClinicalCode, trends DimensionTrends) []MedTrendRisk {
	risks := []MedTrendRisk{}
	medName := strings.ToLower(med.Display)

	// Hemodynamic trend risks
	if trends.Hemodynamic.Direction == TrendDeteriorating || trends.Hemodynamic.Direction == TrendCritical {
		if containsAny(medName, []string{"metoprolol", "atenolol", "propranolol", "carvedilol"}) {
			risks = append(risks, MedTrendRisk{
				Medication:     med,
				RiskType:       "Hemodynamic worsening",
				RiskLevel:      RiskHigh,
				TrendImpact:    "Beta-blocker will accelerate MAP decline",
				Recommendation: "Hold until hemodynamics stabilize",
			})
		}
		if containsAny(medName, []string{"amlodipine", "nifedipine", "diltiazem"}) {
			risks = append(risks, MedTrendRisk{
				Medication:     med,
				RiskType:       "Vasodilation during hypotension",
				RiskLevel:      RiskCritical,
				TrendImpact:    "Calcium channel blocker contraindicated with declining MAP",
				Recommendation: "Contraindicated - will worsen hypotension",
			})
		}
	}

	// Respiratory trend risks
	if trends.Respiratory.Direction == TrendDeteriorating || trends.Respiratory.Direction == TrendCritical {
		if containsAny(medName, []string{"morphine", "fentanyl", "hydromorphone", "oxycodone"}) {
			risks = append(risks, MedTrendRisk{
				Medication:     med,
				RiskType:       "Respiratory depression",
				RiskLevel:      RiskHigh,
				TrendImpact:    "Opioid will accelerate SpO2 decline",
				Recommendation: "Use lowest effective dose with continuous SpO2 monitoring",
			})
		}
		if containsAny(medName, []string{"midazolam", "lorazepam", "diazepam", "propofol"}) {
			risks = append(risks, MedTrendRisk{
				Medication:     med,
				RiskType:       "Sedation during respiratory decline",
				RiskLevel:      RiskHigh,
				TrendImpact:    "Sedative increases respiratory failure risk",
				Recommendation: "Ensure secured airway before administration",
			})
		}
	}

	// Renal trend risks
	if trends.Renal.Direction == TrendDeteriorating {
		if containsAny(medName, []string{"gentamicin", "tobramycin", "amikacin"}) {
			risks = append(risks, MedTrendRisk{
				Medication:     med,
				RiskType:       "Nephrotoxicity acceleration",
				RiskLevel:      RiskCritical,
				TrendImpact:    "Aminoglycoside will accelerate AKI progression",
				Recommendation: "Switch to non-nephrotoxic alternative",
			})
		}
		if containsAny(medName, []string{"ibuprofen", "ketorolac", "naproxen"}) {
			risks = append(risks, MedTrendRisk{
				Medication:     med,
				RiskType:       "NSAID in AKI progression",
				RiskLevel:      RiskCritical,
				TrendImpact:    "NSAID will dramatically worsen renal function",
				Recommendation: "Absolutely contraindicated with rising creatinine",
			})
		}
	}

	// Neurological trend risks
	if trends.Neurological.Direction == TrendDeteriorating && trends.Neurological.CurrentValue <= 10 {
		if containsAny(medName, []string{"midazolam", "propofol", "dexmedetomidine"}) {
			risks = append(risks, MedTrendRisk{
				Medication:     med,
				RiskType:       "Sedation obscuring neuro assessment",
				RiskLevel:      RiskHigh,
				TrendImpact:    "Sedative will mask further neurological decline",
				Recommendation: "Hold for neuro assessment unless intubated",
			})
		}
	}

	return risks
}

// ============================================================================
// Helper Functions
// ============================================================================

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

func findMinMax(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

func linearRegression(x, y []float64) (slope, intercept float64) {
	n := float64(len(x))
	if n < 2 {
		return 0, 0
	}

	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, sumY / n
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n
	return
}

func calculateProbability(hoursToEvent, confidence float64) float64 {
	// Closer events have higher probability
	baseProbability := 1.0 / (1.0 + hoursToEvent/2) // Sigmoid-like
	return baseProbability * confidence
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
