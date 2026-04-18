package services

import "time"

// TransitionOutcomeInput carries the data needed to compute transition outcome.
type TransitionOutcomeInput struct {
	TransitionID                string
	WindowDays                  int
	DaysSinceDischarge          int
	WasReadmitted               bool
	ReadmissionDate             *time.Time
	ReadmissionReason           string
	PAIHistory                  []float64 // PAI scores during window
	PAITierAtExit               string
	EscalationCount             int
	ReadingCount                int     // vital sign readings during window
	PreAdmissionReadingsPerWeek float64
	ReconciliationResolved      bool
}

// TransitionOutcomeResult is the computed outcome.
type TransitionOutcomeResult struct {
	OutcomeCategory           string // SUCCESSFUL, READMITTED, DETERIORATED, DISENGAGED
	ReadmissionDate           *time.Time
	ReadmissionReason         string
	FinalPAITier              string
	ReconciliationOutcome     string
	EngagementMetric          float64 // readings per week during window
	EscalationsTriggeredCount int
}

// ComputeTransitionOutcome determines the outcome category for a care transition.
//
// Logic precedence:
//  1. If WasReadmitted → READMITTED
//  2. If ReadingCount == 0 and DaysSinceDischarge > 14 → DISENGAGED
//  3. If PAI history max > 80 (CRITICAL) or PAITierAtExit in {HIGH, CRITICAL} → DETERIORATED
//  4. Otherwise → SUCCESSFUL
func ComputeTransitionOutcome(input TransitionOutcomeInput) TransitionOutcomeResult {
	// Compute engagement metric: readings per week.
	var engagementMetric float64
	if input.DaysSinceDischarge > 0 {
		weeks := float64(input.DaysSinceDischarge) / 7.0
		engagementMetric = float64(input.ReadingCount) / weeks
	}

	// Determine reconciliation outcome string.
	reconciliationOutcome := "UNRESOLVED"
	if input.ReconciliationResolved {
		reconciliationOutcome = "RESOLVED"
	}

	result := TransitionOutcomeResult{
		ReadmissionDate:           input.ReadmissionDate,
		ReadmissionReason:         input.ReadmissionReason,
		FinalPAITier:              input.PAITierAtExit,
		ReconciliationOutcome:     reconciliationOutcome,
		EngagementMetric:          engagementMetric,
		EscalationsTriggeredCount: input.EscalationCount,
	}

	// 1. Readmission check.
	if input.WasReadmitted {
		result.OutcomeCategory = "READMITTED"
		return result
	}

	// 2. Disengagement check.
	if input.ReadingCount == 0 && input.DaysSinceDischarge > 14 {
		result.OutcomeCategory = "DISENGAGED"
		return result
	}

	// 3. Deterioration check: PAI history max > 80 or exit tier HIGH/CRITICAL.
	if paiMaxAboveCritical(input.PAIHistory) || input.PAITierAtExit == "HIGH" || input.PAITierAtExit == "CRITICAL" {
		result.OutcomeCategory = "DETERIORATED"
		return result
	}

	// 4. Default: successful transition.
	result.OutcomeCategory = "SUCCESSFUL"
	return result
}

// paiMaxAboveCritical returns true if any PAI score exceeds 80.
func paiMaxAboveCritical(history []float64) bool {
	for _, v := range history {
		if v > 80 {
			return true
		}
	}
	return false
}

// TransitionExitActions describes the side effects that must be performed
// when a transition exits. The caller (API handler or batch processor)
// is responsible for executing these actions against the respective services.
type TransitionExitActions struct {
	ResetBaselineToSteadyState    bool   // KB-26: set baseline stage → STEADY_STATE
	DeactivateHeightenedSurveillance bool // KB-26: remove threshold tightening + PAI boost
	ResetPAIContextBoost          bool   // KB-26: remove +15 post-discharge boost
	RestoreEngagementGapThreshold bool   // restore 7d from 72h
	RestoreEscalationTiers        bool   // remove ROUTINE→URGENT amplification
	PatientID                     string
}

// ComputeExitActions returns the set of side effects that must be performed
// when the transition exits, regardless of outcome category. Every transition
// exit requires resetting the heightened surveillance state.
func ComputeExitActions(patientID string) TransitionExitActions {
	return TransitionExitActions{
		ResetBaselineToSteadyState:       true,
		DeactivateHeightenedSurveillance: true,
		ResetPAIContextBoost:             true,
		RestoreEngagementGapThreshold:    true,
		RestoreEscalationTiers:           true,
		PatientID:                        patientID,
	}
}
