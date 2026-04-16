package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"kb-patient-profile/internal/models"
)

func TestClassify_DirectionalWithOverride_Genuine(t *testing.T) {
	overrides := []models.OverrideEvent{
		{EventType: "MEDICATION_START", EventDate: time.Now(), Domain: "cardio", Detail: "started ACE inhibitor"},
	}
	history := []models.ClusterTransitionRecord{
		{PreviousCluster: "STABLE_CONTROLLED", NewCluster: "IMPROVING"},
		{PreviousCluster: "IMPROVING", NewCluster: "WORSENING"},
	}

	result := ClassifyTransition("WORSENING", "UNCONTROLLED", overrides, history, false, false)
	assert.Equal(t, models.ClassificationGenuine, result)
}

func TestClassify_OscillationNoEvent_ProbableFlap(t *testing.T) {
	// History shows A→B and now we see B→A again — oscillation
	history := []models.ClusterTransitionRecord{
		{PreviousCluster: "STABLE_CONTROLLED", NewCluster: "IMPROVING"},
		{PreviousCluster: "IMPROVING", NewCluster: "STABLE_CONTROLLED"},
	}

	result := ClassifyTransition("STABLE_CONTROLLED", "IMPROVING", nil, history, false, false)
	assert.Equal(t, models.ClassificationFlap, result)
}

func TestClassify_InsufficientHistory_Uncertain(t *testing.T) {
	// Empty history — not enough context
	result := ClassifyTransition("STABLE_CONTROLLED", "IMPROVING", nil, nil, false, false)
	assert.Equal(t, models.ClassificationUncertain, result)

	// Single-entry history — still not enough
	history := []models.ClusterTransitionRecord{
		{PreviousCluster: "STABLE_CONTROLLED", NewCluster: "IMPROVING"},
	}
	result = ClassifyTransition("IMPROVING", "WORSENING", nil, history, false, false)
	assert.Equal(t, models.ClassificationUncertain, result)
}

func TestClassify_EngagementCoincident_DataQuality(t *testing.T) {
	history := []models.ClusterTransitionRecord{
		{PreviousCluster: "STABLE_CONTROLLED", NewCluster: "IMPROVING"},
		{PreviousCluster: "IMPROVING", NewCluster: "WORSENING"},
	}

	result := ClassifyTransition("WORSENING", "UNCONTROLLED", nil, history, false, true)
	assert.Equal(t, models.ClassificationUncertain, result)
}

func TestClassify_SeasonalWindow_Uncertain(t *testing.T) {
	history := []models.ClusterTransitionRecord{
		{PreviousCluster: "STABLE_CONTROLLED", NewCluster: "IMPROVING"},
		{PreviousCluster: "IMPROVING", NewCluster: "WORSENING"},
	}

	result := ClassifyTransition("WORSENING", "UNCONTROLLED", nil, history, true, false)
	assert.Equal(t, models.ClassificationUncertain, result)
}

func TestClassify_MedicationOverride_Genuine(t *testing.T) {
	// Even with oscillation history, override event wins
	history := []models.ClusterTransitionRecord{
		{PreviousCluster: "STABLE_CONTROLLED", NewCluster: "IMPROVING"},
		{PreviousCluster: "IMPROVING", NewCluster: "STABLE_CONTROLLED"},
	}
	overrides := []models.OverrideEvent{
		{EventType: "MEDICATION_CLASS_ADDITION", EventDate: time.Now(), Domain: "cardio", Detail: "added beta-blocker"},
	}

	result := ClassifyTransition("STABLE_CONTROLLED", "IMPROVING", overrides, history, false, false)
	assert.Equal(t, models.ClassificationGenuine, result)
}
