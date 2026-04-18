package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExit_CleanTrajectory_Successful(t *testing.T) {
	input := TransitionOutcomeInput{
		TransitionID:                "TR-001",
		WindowDays:                  30,
		DaysSinceDischarge:          30,
		WasReadmitted:               false,
		PAIHistory:                  []float64{45, 40, 38, 35},
		PAITierAtExit:               "LOW",
		EscalationCount:             2,
		ReadingCount:                20,
		PreAdmissionReadingsPerWeek: 7,
		ReconciliationResolved:      true,
	}

	result := ComputeTransitionOutcome(input)

	assert.Equal(t, "SUCCESSFUL", result.OutcomeCategory)
	assert.Equal(t, "LOW", result.FinalPAITier)
	assert.Equal(t, "RESOLVED", result.ReconciliationOutcome)
	assert.Equal(t, 2, result.EscalationsTriggeredCount)
	// Engagement: 20 readings / (30/7) weeks ≈ 4.67
	assert.InDelta(t, 4.67, result.EngagementMetric, 0.1)
}

func TestExit_Readmission_Day14(t *testing.T) {
	readmitDate := time.Date(2026, 4, 15, 8, 0, 0, 0, time.UTC)
	input := TransitionOutcomeInput{
		TransitionID:       "TR-002",
		WindowDays:         30,
		DaysSinceDischarge: 14,
		WasReadmitted:      true,
		ReadmissionDate:    &readmitDate,
		ReadmissionReason:  "Acute HF exacerbation",
		PAIHistory:         []float64{50, 65, 72, 85},
		PAITierAtExit:      "CRITICAL",
		EscalationCount:    4,
		ReadingCount:       10,
	}

	result := ComputeTransitionOutcome(input)

	assert.Equal(t, "READMITTED", result.OutcomeCategory)
	assert.Equal(t, &readmitDate, result.ReadmissionDate)
	assert.Equal(t, "Acute HF exacerbation", result.ReadmissionReason)
}

func TestExit_PAIHigh_Throughout_Deteriorated(t *testing.T) {
	input := TransitionOutcomeInput{
		TransitionID:       "TR-003",
		WindowDays:         30,
		DaysSinceDischarge: 30,
		WasReadmitted:      false,
		PAIHistory:         []float64{70, 78, 82, 85}, // max 85 > 80
		PAITierAtExit:      "HIGH",
		EscalationCount:    5,
		ReadingCount:       15,
		ReconciliationResolved: true,
	}

	result := ComputeTransitionOutcome(input)

	assert.Equal(t, "DETERIORATED", result.OutcomeCategory)
	assert.Equal(t, "HIGH", result.FinalPAITier)
}

func TestExit_NoReadings14Days_Disengaged(t *testing.T) {
	input := TransitionOutcomeInput{
		TransitionID:       "TR-004",
		WindowDays:         30,
		DaysSinceDischarge: 21,
		WasReadmitted:      false,
		PAIHistory:         []float64{},
		PAITierAtExit:      "UNKNOWN",
		EscalationCount:    0,
		ReadingCount:       0,
		ReconciliationResolved: false,
	}

	result := ComputeTransitionOutcome(input)

	assert.Equal(t, "DISENGAGED", result.OutcomeCategory)
	assert.Equal(t, "UNRESOLVED", result.ReconciliationOutcome)
	assert.Equal(t, 0.0, result.EngagementMetric)
}
