package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// Test 1: SAFETY escalation wins over PAI HIGH + active cards
// ---------------------------------------------------------------------------

func TestAggregator_SafetyEscalation_FirstPriority(t *testing.T) {
	state := PatientClinicalState{
		PatientID:         "PAT-001",
		PatientName:       "Alice Smith",
		PatientAge:        68,
		PAIScore:          0.82,
		PAITier:           "HIGH",
		PAITrend:          "RISING",
		DominantDimension: "RENAL",
		ActiveEscalations: []EscalationInfo{
			{Tier: "SAFETY", State: "PENDING", PrimaryReason: "Hyperkalaemia K+ 6.8", SuggestedAction: "Withhold RAAS inhibitor"},
			{Tier: "URGENT", State: "PENDING", PrimaryReason: "eGFR declining", SuggestedAction: "Review renal panel"},
		},
		ActiveCards: []CardInfo{
			{CardID: "C-1", ClinicianSummary: "Renal dose gate", MCUGate: "HOLD", SafetyTier: "CRITICAL"},
			{CardID: "C-2", ClinicianSummary: "BP target review", MCUGate: "MODIFY", SafetyTier: "URGENT"},
		},
	}

	item := AggregateWorklistItem(state)

	require.NotNil(t, item)
	assert.Equal(t, "PAT-001", item.PatientID)
	assert.Equal(t, "Alice Smith", item.PatientName)
	assert.Equal(t, models.WorklistTierCritical, item.UrgencyTier)
	assert.Equal(t, models.WorklistTriggerEscalation, item.TriggeringSource)
	assert.Equal(t, "Hyperkalaemia K+ 6.8", item.PrimaryReason)
	assert.Equal(t, "Withhold RAAS inhibitor", item.SuggestedAction)
	assert.Equal(t, "Within 4 hours", item.SuggestedTimeframe)
	assert.Equal(t, "SAFETY", item.EscalationTier)
	assert.Equal(t, models.ResolutionPending, item.ResolutionState)
}

// ---------------------------------------------------------------------------
// Test 2: PAI CRITICAL (no escalation) → CRITICAL tier via PAI_CHANGE
// ---------------------------------------------------------------------------

func TestAggregator_PAICritical_SecondPriority(t *testing.T) {
	state := PatientClinicalState{
		PatientID:          "PAT-002",
		PatientName:        "Bob Jones",
		PatientAge:         72,
		PAIScore:           0.95,
		PAITier:            "CRITICAL",
		PAITrend:           "RISING",
		DominantDimension:  "GLYCAEMIC",
		PAIPrimaryReason:   "HbA1c rising above 9%",
		PAISuggestedAction: "Intensify glycaemic therapy",
		ActiveCards: []CardInfo{
			{CardID: "C-10", ClinicianSummary: "Glycaemic escalation", MCUGate: "MODIFY", SafetyTier: "URGENT"},
			{CardID: "C-11", ClinicianSummary: "Weight management", MCUGate: "ADVISE", SafetyTier: "ROUTINE"},
			{CardID: "C-12", ClinicianSummary: "Lipid review", MCUGate: "ADVISE", SafetyTier: "ROUTINE"},
		},
	}

	item := AggregateWorklistItem(state)

	require.NotNil(t, item)
	assert.Equal(t, models.WorklistTierCritical, item.UrgencyTier)
	assert.Equal(t, models.WorklistTriggerPAIChange, item.TriggeringSource)
	assert.Equal(t, "HbA1c rising above 9%", item.PrimaryReason)
	assert.Equal(t, "Intensify glycaemic therapy", item.SuggestedAction)
	assert.Equal(t, "Within 4 hours", item.SuggestedTimeframe)
}

// ---------------------------------------------------------------------------
// Test 3: Acute event (HIGH severity), no escalation, PAI MODERATE
// ---------------------------------------------------------------------------

func TestAggregator_AcuteEvent_FifthPriority(t *testing.T) {
	state := PatientClinicalState{
		PatientID:          "PAT-003",
		PatientName:        "Carol Davis",
		PatientAge:         55,
		PAIScore:           0.55,
		PAITier:            "MODERATE",
		PAITrend:           "STABLE",
		DominantDimension:  "CARDIAC",
		HasAcuteEvent:      true,
		AcuteEventType:     "MI",
		AcuteEventSeverity: "HIGH",
	}

	item := AggregateWorklistItem(state)

	require.NotNil(t, item)
	assert.Equal(t, models.WorklistTierHigh, item.UrgencyTier)
	assert.Equal(t, models.WorklistTriggerAcuteEvent, item.TriggeringSource)
	assert.Contains(t, item.PrimaryReason, "MI")
	assert.Equal(t, "Today", item.SuggestedTimeframe)
}

// ---------------------------------------------------------------------------
// Test 4: One item per patient — 4 cards + 2 escalations → exactly 1 item
// ---------------------------------------------------------------------------

func TestAggregator_OneItemPerPatient(t *testing.T) {
	state := PatientClinicalState{
		PatientID:         "PAT-004",
		PatientName:       "Dan Wilson",
		PatientAge:        60,
		PAIScore:          0.70,
		PAITier:           "MODERATE",
		PAITrend:          "RISING",
		DominantDimension: "RENAL",
		ActiveEscalations: []EscalationInfo{
			{Tier: "URGENT", State: "PENDING", PrimaryReason: "eGFR drop", SuggestedAction: "Renal panel"},
			{Tier: "URGENT", State: "DELIVERED", PrimaryReason: "Proteinuria", SuggestedAction: "ACR check"},
		},
		ActiveCards: []CardInfo{
			{CardID: "C-20", ClinicianSummary: "Card A", MCUGate: "HOLD", SafetyTier: "URGENT"},
			{CardID: "C-21", ClinicianSummary: "Card B", MCUGate: "MODIFY", SafetyTier: "ROUTINE"},
			{CardID: "C-22", ClinicianSummary: "Card C", MCUGate: "ADVISE", SafetyTier: "ROUTINE"},
			{CardID: "C-23", ClinicianSummary: "Card D", MCUGate: "ADVISE", SafetyTier: "ROUTINE"},
		},
	}

	item := AggregateWorklistItem(state)

	// Exactly one item, not 6 (4 cards + 2 escalations).
	require.NotNil(t, item)
	assert.Equal(t, "PAT-004", item.PatientID)
	// The function returns a single *WorklistItem, so "one item" is structural.
	// The winning priority should be URGENT escalation → HIGH tier.
	assert.Equal(t, models.WorklistTierHigh, item.UrgencyTier)
}

// ---------------------------------------------------------------------------
// Test 5: Routine patient — nothing actionable → nil
// ---------------------------------------------------------------------------

func TestAggregator_RoutinePatient_NoItem(t *testing.T) {
	state := PatientClinicalState{
		PatientID:         "PAT-005",
		PatientName:       "Eve Brown",
		PatientAge:        45,
		PAIScore:          0.20,
		PAITier:           "LOW",
		PAITrend:          "STABLE",
		DominantDimension: "NONE",
	}

	item := AggregateWorklistItem(state)

	assert.Nil(t, item)
}

// ---------------------------------------------------------------------------
// Test 6: Transition patient adds POST_DISCHARGE context tag
// ---------------------------------------------------------------------------

func TestAggregator_TransitionPatient_ContextTag(t *testing.T) {
	state := PatientClinicalState{
		PatientID:           "PAT-006",
		PatientName:         "Frank Lee",
		PatientAge:          63,
		PAIScore:            0.60,
		PAITier:             "MODERATE",
		PAITrend:            "RISING",
		DominantDimension:   "CARDIAC",
		HasActiveTransition: true,
		TransitionDays:      5,
		PAIPrimaryReason:    "Post-discharge BP lability",
		PAISuggestedAction:  "Review antihypertensive regimen",
	}

	item := AggregateWorklistItem(state)

	require.NotNil(t, item)
	assert.Contains(t, item.ContextTags, "POST_DISCHARGE")
	assert.Contains(t, item.ContextTags, "RISING_PAI")
	// Active transition with pending milestone → MODERATE tier
	assert.Equal(t, models.WorklistTierModerate, item.UrgencyTier)
	assert.Equal(t, models.WorklistTriggerTransitionMilestone, item.TriggeringSource)
}
