package services

import (
	"time"

	"kb-23-decision-cards/internal/models"
)

// AcknowledgmentTracker manages the T0→T3 lifecycle of escalation events:
// PENDING → DELIVERED → ACKNOWLEDGED → ACTED, with timeout escalation
// and PAI-improvement de-escalation.
type AcknowledgmentTracker struct {
	maxEscalationLevels map[string]int // tier → max escalation levels
}

// TimeoutResult is the outcome of a timeout check on an escalation event.
type TimeoutResult struct {
	ShouldEscalate bool
	ShouldExpire   bool
	NextLevel      int
}

// NewAcknowledgmentTracker creates a tracker with tier-specific max escalation levels.
func NewAcknowledgmentTracker(config *EscalationProtocolConfig) *AcknowledgmentTracker {
	maxLevels := map[string]int{
		"SAFETY":    3,
		"IMMEDIATE": 2,
		"URGENT":    1,
		"ROUTINE":   0,
	}
	return &AcknowledgmentTracker{maxEscalationLevels: maxLevels}
}

// RecordDelivery marks T1: the event was delivered to a clinician via the given channel.
func (t *AcknowledgmentTracker) RecordDelivery(event *models.EscalationEvent, channel string, messageID string) {
	now := time.Now()
	event.DeliveredAt = &now
	event.CurrentState = string(models.StateDelivered)
	event.DeliveryAttempts++
}

// RecordAcknowledgment marks T2: the clinician acknowledged the notification.
func (t *AcknowledgmentTracker) RecordAcknowledgment(event *models.EscalationEvent, clinicianID string) {
	now := time.Now()
	event.AcknowledgedAt = &now
	event.AcknowledgedBy = clinicianID
	event.CurrentState = string(models.StateAcknowledged)
}

// RecordAction marks T3: the clinician took a specific action in response.
func (t *AcknowledgmentTracker) RecordAction(event *models.EscalationEvent, actionType, actionDetail string) {
	now := time.Now()
	event.ActedAt = &now
	event.ActionType = actionType
	event.ActionDetail = actionDetail
	event.CurrentState = string(models.StateActed)
}

// CheckTimeout evaluates whether the event has timed out and should be escalated or expired.
func (t *AcknowledgmentTracker) CheckTimeout(event *models.EscalationEvent, now time.Time) TimeoutResult {
	// No timeout configured or not yet reached
	if event.TimeoutAt == nil || now.Before(*event.TimeoutAt) {
		return TimeoutResult{}
	}

	// Only timeout events that are still pending or delivered
	state := models.EscalationState(event.CurrentState)
	if state != models.StatePending && state != models.StateDelivered {
		return TimeoutResult{}
	}

	// Look up max levels for this tier
	maxLevel, ok := t.maxEscalationLevels[event.EscalationTier]
	if !ok {
		maxLevel = 0
	}

	if event.EscalationLevel >= maxLevel {
		return TimeoutResult{ShouldExpire: true}
	}

	return TimeoutResult{
		ShouldEscalate: true,
		NextLevel:      event.EscalationLevel + 1,
	}
}

// HandlePAIImprovement cancels non-SAFETY escalations when the patient's condition improves.
// Returns true if the event was resolved, false otherwise.
func (t *AcknowledgmentTracker) HandlePAIImprovement(event *models.EscalationEvent, newPAITier string) bool {
	// SAFETY events always require explicit acknowledgment
	if event.EscalationTier == "SAFETY" {
		return false
	}

	// Only cancel events that are still pending or delivered
	state := models.EscalationState(event.CurrentState)
	if state != models.StatePending && state != models.StateDelivered {
		return false
	}

	// Check if the new tier is lower priority than the event's tier
	currentPriority := tierPriority(models.EscalationTier(event.EscalationTier))
	newPriority := tierPriority(models.EscalationTier(newPAITier))

	if newPriority < currentPriority {
		now := time.Now()
		event.CurrentState = string(models.StateResolved)
		event.ResolvedAt = &now
		event.ResolutionReason = "CONDITION_IMPROVED"
		return true
	}

	return false
}
