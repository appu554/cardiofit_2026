// Package interpretation provides clinical interpretation algorithms
// lactate_clearance.go implements Surviving Sepsis Campaign 2021 lactate monitoring
package interpretation

import (
	"time"
)

// =============================================================================
// LACTATE CLEARANCE (Surviving Sepsis Campaign 2021)
// =============================================================================

// LactateClearance represents lactate clearance calculation and interpretation
type LactateClearance struct {
	// Input Values
	InitialLactate float64   `json:"initialLactate"` // mmol/L
	CurrentLactate float64   `json:"currentLactate"` // mmol/L
	InitialTime    time.Time `json:"initialTime"`
	CurrentTime    time.Time `json:"currentTime"`
	HoursElapsed   float64   `json:"hoursElapsed"`

	// Calculated Values
	ClearancePercent float64 `json:"clearancePercent"` // Total % cleared
	ClearanceRate    float64 `json:"clearanceRate"`    // % per hour
	AbsoluteChange   float64 `json:"absoluteChange"`   // mmol/L change

	// Interpretation
	Interpretation string `json:"interpretation"`
	RiskCategory   string `json:"riskCategory"` // RESPONDING, NOT_RESPONDING, CRITICAL

	// Clinical Recommendations
	Recommendations []string `json:"recommendations"`

	// Thresholds Applied
	TargetClearanceRate float64 `json:"targetClearanceRate"` // % per hour target

	// Governance
	Governance LactateGovernance `json:"governance"`
}

// LactateGovernance tracks clinical authority for lactate interpretation
type LactateGovernance struct {
	GuidelineSource   string `json:"guidelineSource"`
	ClearanceTarget   string `json:"clearanceTarget"`
	EvidenceLevel     string `json:"evidenceLevel"`
	RecommendationRef string `json:"recommendationRef"`
}

// Lactate risk thresholds (Surviving Sepsis Campaign 2021)
const (
	LactateNormalHigh     = 2.0  // mmol/L - upper normal
	LactateElevated       = 4.0  // mmol/L - significantly elevated
	LactateSevere         = 7.0  // mmol/L - severe elevation
	LactateCritical       = 10.0 // mmol/L - critical, high mortality

	// Clearance targets
	LactateTargetClearanceRate = 10.0 // ≥10% per hour in first 6 hours
	LactateMinClearanceRate    = 5.0  // Minimum acceptable clearance
	LactateGoalNormalization   = 6.0  // Goal to normalize within 6-8 hours
)

// Risk categories
const (
	LactateRiskResponding    = "RESPONDING"
	LactateRiskNotResponding = "NOT_RESPONDING"
	LactateRiskCritical      = "CRITICAL"
)

// CalculateLactateClearance computes lactate clearance per Surviving Sepsis 2021
func CalculateLactateClearance(initial, current float64, hours float64) *LactateClearance {
	result := &LactateClearance{
		InitialLactate:      initial,
		CurrentLactate:      current,
		HoursElapsed:        hours,
		AbsoluteChange:      initial - current,
		TargetClearanceRate: LactateTargetClearanceRate,
	}

	// Calculate clearance percentage and rate
	if initial > 0 {
		result.ClearancePercent = ((initial - current) / initial) * 100
		result.ClearanceRate = result.ClearancePercent / hours
	}

	// Determine risk category and interpretation
	result.interpretClearance()

	// Set governance
	result.Governance = LactateGovernance{
		GuidelineSource:   "Surviving Sepsis Campaign 2021",
		ClearanceTarget:   "≥10% per hour, normalize within 6-8 hours",
		EvidenceLevel:     "STRONG (1B)",
		RecommendationRef: "Recommendation 21: Lactate-guided resuscitation",
	}

	return result
}

// CalculateLactateClearanceWithTimes calculates using actual timestamps
func CalculateLactateClearanceWithTimes(initial, current float64, initialTime, currentTime time.Time) *LactateClearance {
	hours := currentTime.Sub(initialTime).Hours()
	result := CalculateLactateClearance(initial, current, hours)
	result.InitialTime = initialTime
	result.CurrentTime = currentTime
	return result
}

// interpretClearance sets interpretation and recommendations based on clearance
func (lc *LactateClearance) interpretClearance() {
	// Assess based on time period (first 6 hours vs later)
	isEarlyPhase := lc.HoursElapsed <= 6

	// Critical absolute threshold check first
	if lc.CurrentLactate >= LactateCritical {
		lc.RiskCategory = LactateRiskCritical
		lc.Interpretation = "Critical lactate elevation (≥10 mmol/L) - high mortality risk. Aggressive resuscitation required."
		lc.Recommendations = []string{
			"Reassess volume status immediately",
			"Consider vasopressor support if MAP <65 despite fluids",
			"Source control evaluation",
			"Consider ECMO/advanced support consultation",
			"ICU escalation if not already in ICU",
		}
		return
	}

	if lc.CurrentLactate >= LactateSevere {
		lc.RiskCategory = LactateRiskCritical
		lc.Interpretation = "Severe lactate elevation (≥7 mmol/L) - reassess resuscitation strategy."
		lc.Recommendations = []string{
			"Reassess fluid responsiveness",
			"Optimize vasopressor therapy",
			"Evaluate for ongoing source of sepsis",
			"Serial lactate q2-4h",
		}
		return
	}

	// Clearance rate assessment for early resuscitation phase
	if isEarlyPhase {
		if lc.ClearanceRate >= LactateTargetClearanceRate {
			lc.RiskCategory = LactateRiskResponding
			lc.Interpretation = "Adequate lactate clearance (≥10%/hr). Patient responding to resuscitation."
			lc.Recommendations = []string{
				"Continue current resuscitation strategy",
				"Serial lactate monitoring q2-4h",
				"Reassess when lactate normalizes",
			}
		} else if lc.ClearanceRate >= LactateMinClearanceRate {
			lc.RiskCategory = LactateRiskNotResponding
			lc.Interpretation = "Suboptimal lactate clearance (5-10%/hr). Reassess resuscitation adequacy."
			lc.Recommendations = []string{
				"Reassess volume status and fluid responsiveness",
				"Consider passive leg raise or fluid challenge",
				"Evaluate for ongoing source of sepsis",
				"Check ScvO2 if central line available",
				"Repeat lactate in 2 hours",
			}
		} else if lc.ClearancePercent < 0 {
			// Lactate is rising
			lc.RiskCategory = LactateRiskCritical
			lc.Interpretation = "Lactate RISING - patient deteriorating. Urgent reassessment required."
			lc.Recommendations = []string{
				"Immediate reassessment of resuscitation",
				"Consider undrained source of infection",
				"Reassess vasopressor adequacy",
				"Consider mesenteric ischemia/bowel pathology",
				"Urgent source imaging if not done",
			}
		} else {
			lc.RiskCategory = LactateRiskNotResponding
			lc.Interpretation = "Poor lactate clearance (<5%/hr). Consider escalation of care."
			lc.Recommendations = []string{
				"Aggressive fluid resuscitation reassessment",
				"Consider inotropic support",
				"Evaluate for cardiogenic component",
				"Source control reassessment",
				"Consider nephrology if renal replacement needed",
			}
		}
	} else {
		// Later phase (>6 hours)
		if lc.CurrentLactate <= LactateNormalHigh {
			lc.RiskCategory = LactateRiskResponding
			lc.Interpretation = "Lactate normalized. Continue monitoring."
			lc.Recommendations = []string{
				"Lactate normalized - continue supportive care",
				"Reassess lactate if clinical deterioration",
				"De-escalation of monitoring as appropriate",
			}
		} else if lc.CurrentLactate <= LactateElevated && lc.ClearanceRate > 0 {
			lc.RiskCategory = LactateRiskNotResponding
			lc.Interpretation = "Lactate elevated but clearing. Continue current therapy."
			lc.Recommendations = []string{
				"Continue current resuscitation strategy",
				"Serial lactate monitoring q4-6h",
				"Assess for persistent source",
			}
		} else {
			lc.RiskCategory = LactateRiskCritical
			lc.Interpretation = "Persistent lactate elevation beyond 6 hours. Poor prognosis indicator."
			lc.Recommendations = []string{
				"Reassess all aspects of resuscitation",
				"Consider occult source of sepsis",
				"Goals of care discussion if appropriate",
				"Consider advanced hemodynamic monitoring",
			}
		}
	}

	// Append absolute threshold warnings
	if lc.CurrentLactate > LactateElevated {
		lc.Interpretation += " Lactate remains above 4.0 mmol/L target."
	}
}

// =============================================================================
// LACTATE TREND ANALYSIS
// =============================================================================

// LactateTrendPoint represents a single lactate measurement in a trend
type LactateTrendPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Hours     float64   `json:"hours"` // Hours from initial measurement
}

// LactateTrend represents a series of lactate measurements
type LactateTrend struct {
	Points           []LactateTrendPoint `json:"points"`
	InitialValue     float64             `json:"initialValue"`
	LatestValue      float64             `json:"latestValue"`
	PeakValue        float64             `json:"peakValue"`
	TimeToNormal     *float64            `json:"timeToNormal,omitempty"` // Hours to reach <2.0
	OverallClearance float64             `json:"overallClearance"`       // Total % cleared
	Trajectory       string              `json:"trajectory"`             // IMPROVING, WORSENING, PLATEAU
}

// AnalyzeLactateTrend analyzes a series of lactate measurements
func AnalyzeLactateTrend(points []LactateTrendPoint) *LactateTrend {
	if len(points) == 0 {
		return nil
	}

	trend := &LactateTrend{
		Points:       points,
		InitialValue: points[0].Value,
		LatestValue:  points[len(points)-1].Value,
		PeakValue:    points[0].Value,
	}

	// Find peak and check for normalization
	for i, p := range points {
		if p.Value > trend.PeakValue {
			trend.PeakValue = p.Value
		}
		if p.Value <= LactateNormalHigh && trend.TimeToNormal == nil {
			hours := p.Hours
			trend.TimeToNormal = &hours
		}
		_ = i
	}

	// Calculate overall clearance from peak
	if trend.PeakValue > 0 {
		trend.OverallClearance = ((trend.PeakValue - trend.LatestValue) / trend.PeakValue) * 100
	}

	// Determine trajectory (comparing last 2 points)
	if len(points) >= 2 {
		last := points[len(points)-1].Value
		prev := points[len(points)-2].Value
		change := ((prev - last) / prev) * 100

		if change >= 5 { // >5% decrease
			trend.Trajectory = "IMPROVING"
		} else if change <= -5 { // >5% increase
			trend.Trajectory = "WORSENING"
		} else {
			trend.Trajectory = "PLATEAU"
		}
	} else {
		trend.Trajectory = "UNKNOWN"
	}

	return trend
}

// =============================================================================
// SEPSIS MARKER COMBINATIONS
// =============================================================================

// SepsisMarkerPanel combines lactate with other sepsis indicators
type SepsisMarkerPanel struct {
	Lactate       *LactateClearance `json:"lactate,omitempty"`
	Procalcitonin *float64          `json:"procalcitonin,omitempty"` // ng/mL
	CRP           *float64          `json:"crp,omitempty"`           // mg/L
	WBC           *float64          `json:"wbc,omitempty"`           // x10^9/L

	// qSOFA components (if available)
	SBP             *int  `json:"sbp,omitempty"`             // mmHg
	RespRate        *int  `json:"respRate,omitempty"`        // breaths/min
	AlteredMentaton bool  `json:"alteredMentation,omitempty"`
	QSOFAScore      *int  `json:"qsofaScore,omitempty"`

	// Interpretation
	OverallRisk     string   `json:"overallRisk"`     // LOW, MODERATE, HIGH, CRITICAL
	Interpretation  string   `json:"interpretation"`
	Recommendations []string `json:"recommendations"`
}

// CalculateQSOFA calculates qSOFA score from components
func (smp *SepsisMarkerPanel) CalculateQSOFA() {
	if smp.SBP == nil && smp.RespRate == nil {
		return
	}

	score := 0
	if smp.SBP != nil && *smp.SBP <= 100 {
		score++
	}
	if smp.RespRate != nil && *smp.RespRate >= 22 {
		score++
	}
	if smp.AlteredMentaton {
		score++
	}
	smp.QSOFAScore = &score
}

// EvaluateSepsisPanel provides integrated sepsis marker interpretation
func EvaluateSepsisPanel(panel *SepsisMarkerPanel) {
	panel.CalculateQSOFA()

	criticalCount := 0
	elevatedCount := 0

	// Assess each marker
	if panel.Lactate != nil {
		if panel.Lactate.RiskCategory == LactateRiskCritical {
			criticalCount++
		} else if panel.Lactate.CurrentLactate > LactateNormalHigh {
			elevatedCount++
		}
	}

	if panel.Procalcitonin != nil && *panel.Procalcitonin >= 2.0 {
		elevatedCount++
	}

	if panel.QSOFAScore != nil && *panel.QSOFAScore >= 2 {
		elevatedCount++
	}

	// Overall risk assessment
	if criticalCount > 0 || (elevatedCount >= 2 && panel.Lactate != nil && panel.Lactate.CurrentLactate > LactateElevated) {
		panel.OverallRisk = "CRITICAL"
		panel.Interpretation = "Multiple markers indicate severe sepsis/septic shock."
		panel.Recommendations = []string{
			"Initiate sepsis bundle immediately",
			"Blood cultures before antibiotics",
			"Broad spectrum antibiotics within 1 hour",
			"30 mL/kg crystalloid for hypotension or lactate ≥4",
			"Vasopressors if hypotensive after fluids",
		}
	} else if elevatedCount >= 2 {
		panel.OverallRisk = "HIGH"
		panel.Interpretation = "Multiple elevated sepsis markers - high suspicion for sepsis."
		panel.Recommendations = []string{
			"Initiate sepsis workup",
			"Consider empiric antibiotics",
			"Close monitoring for deterioration",
		}
	} else if elevatedCount == 1 {
		panel.OverallRisk = "MODERATE"
		panel.Interpretation = "Single elevated marker - monitor closely."
		panel.Recommendations = []string{
			"Serial lactate monitoring",
			"Assess for infection source",
			"Clinical correlation required",
		}
	} else {
		panel.OverallRisk = "LOW"
		panel.Interpretation = "Sepsis markers within acceptable range."
		panel.Recommendations = []string{
			"Continue monitoring as clinically indicated",
		}
	}
}
