package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestManager builds an EscalationManager with noop logger, nil DB,
// nil audit, and the supplied channels.
func newTestManager(channels map[string]NotificationChannel) *EscalationManager {
	cfg := DefaultEscalationProtocolConfig()
	router := NewEscalationRouter(cfg)
	tracker := NewAcknowledgmentTracker(cfg)
	log := zap.NewNop()
	return NewEscalationManager(router, tracker, channels, nil, nil, log)
}

func makeCard(differentialID string, patientID uuid.UUID) *models.DecisionCard {
	return &models.DecisionCard{
		CardID:                uuid.New(),
		PatientID:             patientID,
		PrimaryDifferentialID: differentialID,
		MCUGate:               models.MCUGate("MODIFY"),
		ClinicianSummary:      "Test clinician summary for card",
		MCUGateRationale:      "Gate rationale text",
	}
}

// ---------------------------------------------------------------------------
// Test 1: SAFETY tier card creates an escalation event
// ---------------------------------------------------------------------------

func TestManager_HandleCard_SafetyTier_CreatesEscalation(t *testing.T) {
	mgr := newTestManager(nil)
	patientID := uuid.New()
	card := makeCard("RENAL_CONTRAINDICATION", patientID)

	event := mgr.HandleCardCreated(card, "CRITICAL", 0.92)

	require.NotNil(t, event, "SAFETY card must produce an event")
	assert.Equal(t, "SAFETY", event.EscalationTier)
	assert.Equal(t, string(models.StatePending), event.CurrentState)
	assert.Equal(t, 1, event.EscalationLevel)
	assert.NotNil(t, event.TimeoutAt, "SAFETY tier must have a timeout")

	// Timeout should be ~30 min from now.
	expectedTimeout := time.Now().Add(30 * time.Minute)
	diff := event.TimeoutAt.Sub(expectedTimeout)
	assert.True(t, diff > -2*time.Second && diff < 2*time.Second,
		"timeout should be ~30 min from now, got diff=%v", diff)
}

// ---------------------------------------------------------------------------
// Test 2: ROUTINE tier card creates an event (tracked but no active channels)
// ---------------------------------------------------------------------------

func TestManager_HandleCard_RoutineTier_CreatesEvent(t *testing.T) {
	mgr := newTestManager(nil)
	patientID := uuid.New()
	card := makeCard("ADHERENCE_GAP", patientID)

	event := mgr.HandleCardCreated(card, "LOW", 0.3)

	require.NotNil(t, event, "ROUTINE card must produce an event")
	assert.Equal(t, "ROUTINE", event.EscalationTier)
	assert.Equal(t, string(models.StatePending), event.CurrentState)
	assert.Nil(t, event.TimeoutAt, "ROUTINE tier has no timeout (0 min)")
}

// ---------------------------------------------------------------------------
// Test 3: INFORMATIONAL tier returns nil (no event created)
// ---------------------------------------------------------------------------

func TestManager_HandleCard_Informational_ReturnsNil(t *testing.T) {
	mgr := newTestManager(nil)
	patientID := uuid.New()
	// Use a differential ID not in the routing table -> falls to PAI routing.
	// PAI tier "MINIMAL" maps to INFORMATIONAL.
	card := makeCard("SOME_UNKNOWN_DIFFERENTIAL", patientID)

	event := mgr.HandleCardCreated(card, "MINIMAL", 0.05)

	assert.Nil(t, event, "INFORMATIONAL tier should return nil (no event)")
}

// ---------------------------------------------------------------------------
// Test 4: Deduplication — second identical card returns nil
// ---------------------------------------------------------------------------

func TestManager_HandleCard_Deduplicated_ReturnsNil(t *testing.T) {
	mgr := newTestManager(nil)
	patientID := uuid.New()

	card1 := makeCard("RENAL_CONTRAINDICATION", patientID)
	event1 := mgr.HandleCardCreated(card1, "CRITICAL", 0.92)
	require.NotNil(t, event1, "first call must produce an event")

	card2 := makeCard("RENAL_CONTRAINDICATION", patientID)
	event2 := mgr.HandleCardCreated(card2, "CRITICAL", 0.92)
	assert.Nil(t, event2, "second call within dedup window must return nil")
}

// ---------------------------------------------------------------------------
// Test 5: SAFETY card with NoopChannel dispatches and records delivery
// ---------------------------------------------------------------------------

func TestManager_HandleCard_DispatchesChannels(t *testing.T) {
	log := zap.NewNop()
	noop := NewNoopChannel(log)
	channels := map[string]NotificationChannel{
		"push":     noop,
		"sms":      noop,
		"whatsapp": noop,
	}
	mgr := newTestManager(channels)
	patientID := uuid.New()
	card := makeCard("RENAL_CONTRAINDICATION", patientID)

	event := mgr.HandleCardCreated(card, "CRITICAL", 0.92)

	require.NotNil(t, event)
	assert.Equal(t, "SAFETY", event.EscalationTier)
	// SAFETY dispatches to push, sms, whatsapp simultaneously.
	// Each Send returns SENT, so RecordDelivery is called for each.
	assert.Greater(t, event.DeliveryAttempts, 0,
		"at least one delivery attempt should be recorded")
	// With all 3 noop channels registered, we expect 3 delivery attempts.
	assert.Equal(t, 3, event.DeliveryAttempts,
		"SAFETY tier dispatches to all 3 channels simultaneously")
	// State should be DELIVERED after successful dispatch.
	assert.Equal(t, string(models.StateDelivered), event.CurrentState)
}
