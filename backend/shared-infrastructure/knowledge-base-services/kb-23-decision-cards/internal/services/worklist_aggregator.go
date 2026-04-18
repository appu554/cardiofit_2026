package services

import (
	"fmt"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// Input types — aggregated patient clinical state
// ---------------------------------------------------------------------------

// PatientClinicalState is the aggregated clinical snapshot for one patient,
// used as input to the worklist aggregator.
type PatientClinicalState struct {
	PatientID          string
	PatientName        string
	PatientAge         int
	PAIScore           float64
	PAITier            string
	PAITrend           string
	DominantDimension  string
	PAIPrimaryReason   string
	PAISuggestedAction string
	ActiveEscalations  []EscalationInfo
	ActiveCards        []CardInfo
	HasActiveTransition bool
	TransitionDays     int
	LastClinicianDays  int
	HasAcuteEvent      bool
	AcuteEventType     string
	AcuteEventSeverity string
}

// EscalationInfo is a lightweight escalation summary for aggregation.
type EscalationInfo struct {
	Tier            string // SAFETY, IMMEDIATE, URGENT, ROUTINE
	State           string // PENDING, DELIVERED, ACKNOWLEDGED
	PrimaryReason   string
	SuggestedAction string
}

// CardInfo is a lightweight card summary for aggregation.
type CardInfo struct {
	CardID           string
	ClinicianSummary string
	MCUGate          string
	SafetyTier       string
}

// ---------------------------------------------------------------------------
// Timeframe mapping
// ---------------------------------------------------------------------------

var timeframeByTier = map[string]string{
	models.WorklistTierCritical: "Within 4 hours",
	models.WorklistTierHigh:     "Today",
	models.WorklistTierModerate: "Within 24 hours",
	models.WorklistTierLow:      "This week",
}

// ---------------------------------------------------------------------------
// AggregateWorklistItem applies the "one most important thing" rule.
// Returns nil if the patient doesn't need worklist attention.
// ---------------------------------------------------------------------------

func AggregateWorklistItem(state PatientClinicalState) *models.WorklistItem {
	item := &models.WorklistItem{
		PatientID:   state.PatientID,
		PatientName: state.PatientName,
		PatientAge:  state.PatientAge,
		PAIScore:    state.PAIScore,
		PAITrend:    state.PAITrend,
	}

	triggered := false

	// Priority cascade — first match wins.

	// 1. SAFETY escalation (PENDING or DELIVERED)
	if esc, ok := findEscalation(state.ActiveEscalations, "SAFETY"); ok {
		item.UrgencyTier = models.WorklistTierCritical
		item.TriggeringSource = models.WorklistTriggerEscalation
		item.EscalationTier = esc.Tier
		item.PrimaryReason = esc.PrimaryReason
		item.SuggestedAction = esc.SuggestedAction
		triggered = true
	}

	// 2. IMMEDIATE escalation
	if !triggered {
		if esc, ok := findEscalation(state.ActiveEscalations, "IMMEDIATE"); ok {
			item.UrgencyTier = models.WorklistTierCritical
			item.TriggeringSource = models.WorklistTriggerEscalation
			item.EscalationTier = esc.Tier
			item.PrimaryReason = esc.PrimaryReason
			item.SuggestedAction = esc.SuggestedAction
			triggered = true
		}
	}

	// 3. PAI CRITICAL
	if !triggered && state.PAITier == "CRITICAL" {
		item.UrgencyTier = models.WorklistTierCritical
		item.TriggeringSource = models.WorklistTriggerPAIChange
		item.PrimaryReason = state.PAIPrimaryReason
		item.SuggestedAction = state.PAISuggestedAction
		triggered = true
	}

	// 4. URGENT escalation
	if !triggered {
		if esc, ok := findEscalation(state.ActiveEscalations, "URGENT"); ok {
			item.UrgencyTier = models.WorklistTierHigh
			item.TriggeringSource = models.WorklistTriggerEscalation
			item.EscalationTier = esc.Tier
			item.PrimaryReason = esc.PrimaryReason
			item.SuggestedAction = esc.SuggestedAction
			triggered = true
		}
	}

	// 5. Acute event, HIGH severity
	if !triggered && state.HasAcuteEvent && state.AcuteEventSeverity == "HIGH" {
		item.UrgencyTier = models.WorklistTierHigh
		item.TriggeringSource = models.WorklistTriggerAcuteEvent
		item.PrimaryReason = fmt.Sprintf("Acute %s event — HIGH severity", state.AcuteEventType)
		item.SuggestedAction = "Review acute event and adjust plan"
		triggered = true
	}

	// 6. Active transition with pending milestone
	if !triggered && state.HasActiveTransition {
		item.UrgencyTier = models.WorklistTierModerate
		item.TriggeringSource = models.WorklistTriggerTransitionMilestone
		item.PrimaryReason = state.PAIPrimaryReason
		item.SuggestedAction = state.PAISuggestedAction
		if item.PrimaryReason == "" {
			item.PrimaryReason = fmt.Sprintf("Post-discharge day %d — milestone pending", state.TransitionDays)
		}
		if item.SuggestedAction == "" {
			item.SuggestedAction = "Review transition milestones"
		}
		triggered = true
	}

	// 7. PAI HIGH
	if !triggered && state.PAITier == "HIGH" {
		item.UrgencyTier = models.WorklistTierHigh
		item.TriggeringSource = models.WorklistTriggerPAIChange
		item.PrimaryReason = state.PAIPrimaryReason
		item.SuggestedAction = state.PAISuggestedAction
		triggered = true
	}

	// 8. Active unacknowledged card with non-SAFE gate
	if !triggered {
		if card, ok := findActionableCard(state.ActiveCards); ok {
			item.UrgencyTier = models.WorklistTierModerate
			item.TriggeringSource = models.WorklistTriggerCard
			item.PrimaryReason = card.ClinicianSummary
			item.SuggestedAction = fmt.Sprintf("Review card %s — gate %s", card.CardID, card.MCUGate)
			triggered = true
		}
	}

	// 9. Attention gap (LastClinicianDays > 30)
	if !triggered && state.LastClinicianDays > 30 {
		item.UrgencyTier = models.WorklistTierLow
		item.TriggeringSource = models.WorklistTriggerAttentionGap
		item.PrimaryReason = fmt.Sprintf("No clinician contact for %d days", state.LastClinicianDays)
		item.SuggestedAction = "Schedule follow-up"
		triggered = true
	}

	// 10. Nothing actionable
	if !triggered {
		return nil
	}

	// Set timeframe from tier.
	item.SuggestedTimeframe = timeframeByTier[item.UrgencyTier]

	// Collect context tags.
	item.ContextTags = buildContextTags(state)

	// Collect underlying card IDs.
	for _, c := range state.ActiveCards {
		item.UnderlyingCardIDs = append(item.UnderlyingCardIDs, c.CardID)
	}

	item.DominantDimension = state.DominantDimension
	item.LastClinicianDays = state.LastClinicianDays
	item.ResolutionState = models.ResolutionPending
	item.ComputedAt = time.Now()

	return item
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// findEscalation returns the first escalation matching tier that is not
// ACKNOWLEDGED (i.e. still pending or delivered).
func findEscalation(escs []EscalationInfo, tier string) (EscalationInfo, bool) {
	for _, e := range escs {
		if e.Tier == tier && e.State != "ACKNOWLEDGED" {
			return e, true
		}
	}
	return EscalationInfo{}, false
}

// findActionableCard returns the first card with a non-SAFE MCU gate.
func findActionableCard(cards []CardInfo) (CardInfo, bool) {
	for _, c := range cards {
		if c.MCUGate != "SAFE" && c.MCUGate != "" {
			return c, true
		}
	}
	return CardInfo{}, false
}

// buildContextTags derives context tags from the patient state.
func buildContextTags(state PatientClinicalState) []string {
	var tags []string
	if state.HasActiveTransition {
		tags = append(tags, "POST_DISCHARGE")
	}
	if state.PAITrend == "RISING" {
		tags = append(tags, "RISING_PAI")
	}
	// Check for any unacknowledged escalation.
	for _, e := range state.ActiveEscalations {
		if e.State != "ACKNOWLEDGED" {
			tags = append(tags, "UNACKNOWLEDGED")
			break
		}
	}
	return tags
}
