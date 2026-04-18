package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseTransitionInfo() TransitionInfo {
	return TransitionInfo{
		TransitionID:     "TR-001",
		PatientID:        "PAT-100",
		DischargeDate:    time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		FacilityType:     "HOSPITAL",
		PrimaryDiagnosis: "Heart Failure",
		WindowDays:       30,
	}
}

func TestScheduler_StandardSchedule_4Milestones(t *testing.T) {
	info := baseTransitionInfo()

	milestones := ScheduleMilestones(info, false, false)

	require.Len(t, milestones, 4)

	discharge := info.DischargeDate

	// MEDICATION_RECONCILIATION_48H at +48h
	assert.Equal(t, MilestoneMedReconciliation48H, milestones[0].MilestoneType)
	assert.Equal(t, discharge.Add(48*time.Hour), milestones[0].ScheduledFor)
	assert.True(t, milestones[0].Required)

	// FIRST_FOLLOWUP_7D at +168h
	assert.Equal(t, MilestoneFirstFollowup7D, milestones[1].MilestoneType)
	assert.Equal(t, discharge.Add(168*time.Hour), milestones[1].ScheduledFor)
	assert.True(t, milestones[1].Required)

	// MIDPOINT_REVIEW_14D at +336h
	assert.Equal(t, MilestoneMidpointReview14D, milestones[2].MilestoneType)
	assert.Equal(t, discharge.Add(336*time.Hour), milestones[2].ScheduledFor)
	assert.True(t, milestones[2].Required)

	// EXIT_ASSESSMENT_30D at +720h
	assert.Equal(t, MilestoneExitAssessment30D, milestones[3].MilestoneType)
	assert.Equal(t, discharge.Add(720*time.Hour), milestones[3].ScheduledFor)
	assert.True(t, milestones[3].Required)
}

func TestScheduler_EngagementCheck_Added(t *testing.T) {
	info := baseTransitionInfo()

	milestones := ScheduleMilestones(info, true, false)

	require.Len(t, milestones, 5)

	// 5th milestone is ENGAGEMENT_CHECK_72H at +72h
	engagement := milestones[4]
	assert.Equal(t, MilestoneEngagementCheck72H, engagement.MilestoneType)
	assert.Equal(t, info.DischargeDate.Add(72*time.Hour), engagement.ScheduledFor)
	assert.False(t, engagement.Required)
}

func TestScheduler_SupplyCheck_Added(t *testing.T) {
	info := baseTransitionInfo()

	milestones := ScheduleMilestones(info, false, true)

	require.Len(t, milestones, 5)

	// 5th milestone is MEDICATION_SUPPLY_CHECK at +24h
	supply := milestones[4]
	assert.Equal(t, MilestoneMedSupplyCheck, supply.MilestoneType)
	assert.Equal(t, info.DischargeDate.Add(24*time.Hour), supply.ScheduledFor)
	assert.False(t, supply.Required)
}

func TestScheduler_AssessMilestone_48h_Clean(t *testing.T) {
	result := AssessMilestone48h("CLEAN")

	assert.Equal(t, MilestoneMedReconciliation48H, result.MilestoneType)
	assert.Equal(t, "ROUTINE", result.CardTier)
	assert.Contains(t, result.Findings, "no discrepancies")
	assert.NotEmpty(t, result.SuggestedActions)
}

func TestScheduler_AssessMilestone_48h_HighRisk(t *testing.T) {
	result := AssessMilestone48h("HIGH_RISK_URGENT")

	assert.Equal(t, MilestoneMedReconciliation48H, result.MilestoneType)
	assert.Equal(t, "IMMEDIATE", result.CardTier)
	assert.Contains(t, result.Findings, "High-risk")
	assert.Contains(t, result.SuggestedActions, "Immediate")
}
