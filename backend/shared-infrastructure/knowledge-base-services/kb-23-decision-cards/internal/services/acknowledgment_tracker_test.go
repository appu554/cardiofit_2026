package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kb-23-decision-cards/internal/models"
)

func newTestTracker() *AcknowledgmentTracker {
	return NewAcknowledgmentTracker(DefaultEscalationProtocolConfig())
}

func TestTracker_RecordDelivery_SetsT1(t *testing.T) {
	tracker := newTestTracker()
	event := &models.EscalationEvent{
		CurrentState:   string(models.StatePending),
		EscalationTier: "URGENT",
	}

	tracker.RecordDelivery(event, "PUSH", "msg-001")

	assert.Equal(t, string(models.StateDelivered), event.CurrentState)
	require.NotNil(t, event.DeliveredAt)
	assert.WithinDuration(t, time.Now(), *event.DeliveredAt, 2*time.Second)
	assert.Equal(t, 1, event.DeliveryAttempts)
}

func TestTracker_RecordAcknowledgment_SetsT2(t *testing.T) {
	tracker := newTestTracker()
	event := &models.EscalationEvent{
		CurrentState:   string(models.StateDelivered),
		EscalationTier: "URGENT",
	}

	tracker.RecordAcknowledgment(event, "DR-SMITH")

	assert.Equal(t, string(models.StateAcknowledged), event.CurrentState)
	require.NotNil(t, event.AcknowledgedAt)
	assert.WithinDuration(t, time.Now(), *event.AcknowledgedAt, 2*time.Second)
	assert.Equal(t, "DR-SMITH", event.AcknowledgedBy)
}

func TestTracker_RecordAction_SetsT3(t *testing.T) {
	tracker := newTestTracker()
	event := &models.EscalationEvent{
		CurrentState:   string(models.StateAcknowledged),
		EscalationTier: "URGENT",
	}

	tracker.RecordAction(event, "DOSE_ADJUST", "Reduced metformin to 500mg")

	assert.Equal(t, string(models.StateActed), event.CurrentState)
	require.NotNil(t, event.ActedAt)
	assert.WithinDuration(t, time.Now(), *event.ActedAt, 2*time.Second)
	assert.Equal(t, "DOSE_ADJUST", event.ActionType)
	assert.Equal(t, "Reduced metformin to 500mg", event.ActionDetail)
}

func TestTracker_Timeout_Level1_EscalatesToLevel2(t *testing.T) {
	tracker := newTestTracker()
	pastTimeout := time.Now().Add(-10 * time.Minute)
	event := &models.EscalationEvent{
		CurrentState:    string(models.StateDelivered),
		EscalationTier:  "SAFETY",
		EscalationLevel: 1,
		TimeoutAt:       &pastTimeout,
	}

	result := tracker.CheckTimeout(event, time.Now())

	assert.True(t, result.ShouldEscalate)
	assert.False(t, result.ShouldExpire)
	assert.Equal(t, 2, result.NextLevel)
}

func TestTracker_Timeout_MaxLevel_Expires(t *testing.T) {
	tracker := newTestTracker()
	pastTimeout := time.Now().Add(-10 * time.Minute)
	event := &models.EscalationEvent{
		CurrentState:    string(models.StateDelivered),
		EscalationTier:  "SAFETY",
		EscalationLevel: 3, // max for SAFETY
		TimeoutAt:       &pastTimeout,
	}

	result := tracker.CheckTimeout(event, time.Now())

	assert.False(t, result.ShouldEscalate)
	assert.True(t, result.ShouldExpire)
}

func TestTracker_DeEscalation_CancelsUrgent(t *testing.T) {
	tracker := newTestTracker()
	event := &models.EscalationEvent{
		CurrentState:   string(models.StatePending),
		EscalationTier: "URGENT",
	}

	resolved := tracker.HandlePAIImprovement(event, "ROUTINE")

	assert.True(t, resolved)
	assert.Equal(t, string(models.StateResolved), event.CurrentState)
	assert.Equal(t, "CONDITION_IMPROVED", event.ResolutionReason)
	require.NotNil(t, event.ResolvedAt)
}

func TestTracker_DeEscalation_SafetyNotCancelled(t *testing.T) {
	tracker := newTestTracker()
	event := &models.EscalationEvent{
		CurrentState:   string(models.StateDelivered),
		EscalationTier: "SAFETY",
	}

	resolved := tracker.HandlePAIImprovement(event, "ROUTINE")

	assert.False(t, resolved)
	assert.Equal(t, string(models.StateDelivered), event.CurrentState)
}
