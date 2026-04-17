package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"kb-23-decision-cards/internal/models"
)

func newTestRouter() *EscalationRouter {
	return NewEscalationRouter(DefaultEscalationProtocolConfig())
}

func TestRouter_RenalContraindication_Safety(t *testing.T) {
	r := newTestRouter()
	result := r.RouteCard(EscalationRouterInput{
		CardDifferentialID: "RENAL_CONTRAINDICATION",
		PatientID:          "P001",
	})
	assert.Equal(t, models.TierSafety, result.Tier)
	assert.False(t, result.Suppressed)
}

func TestRouter_TherapeuticInertia_Urgent(t *testing.T) {
	r := newTestRouter()
	result := r.RouteCard(EscalationRouterInput{
		CardDifferentialID: "THERAPEUTIC_INERTIA",
		PatientID:          "P002",
	})
	assert.Equal(t, models.TierUrgent, result.Tier)
	assert.False(t, result.Suppressed)
}

func TestRouter_HaltGate_AmplifiedToSafety(t *testing.T) {
	r := newTestRouter()
	result := r.RouteCard(EscalationRouterInput{
		CardDifferentialID: "THERAPEUTIC_INERTIA",
		MCUGate:            "HALT",
		PatientID:          "P003",
	})
	// THERAPEUTIC_INERTIA is normally URGENT, but HALT gate amplifies to SAFETY
	assert.Equal(t, models.TierSafety, result.Tier)
	assert.Contains(t, result.Reason, "HALT")
	assert.False(t, result.Suppressed)
}

func TestRouter_PAICritical_EGFRLow_Safety(t *testing.T) {
	r := newTestRouter()
	egfr := 25.0
	result := r.RouteCard(EscalationRouterInput{
		PAITier:   "CRITICAL",
		PAIScore:  0.92,
		EGFR:     &egfr,
		PatientID: "P004",
	})
	// PAI CRITICAL normally maps to IMMEDIATE, but eGFR < 30 amplifies to SAFETY
	assert.Equal(t, models.TierSafety, result.Tier)
	assert.False(t, result.Suppressed)
}

func TestRouter_SustainedElevation_Suppressed(t *testing.T) {
	r := newTestRouter()
	// First computation for HIGH PAI — sustained elevation not yet confirmed
	result := r.RouteCard(EscalationRouterInput{
		PAITier:   "HIGH",
		PAIScore:  0.75,
		PatientID: "P005",
	})
	// MinConsecutive=2, first call → count=1, should be suppressed
	assert.True(t, result.Suppressed)
	assert.Equal(t, "awaiting sustained confirmation", result.SuppressionReason)
}

func TestRouter_SustainedElevation_Bypassed_Safety(t *testing.T) {
	r := newTestRouter()
	egfr := 20.0
	// SAFETY tier is exempt from sustained-elevation gate
	result := r.RouteCard(EscalationRouterInput{
		PAITier:   "CRITICAL",
		PAIScore:  0.95,
		EGFR:     &egfr,
		PatientID: "P006",
	})
	// Amplified to SAFETY → exempt from sustained elevation → not suppressed
	assert.Equal(t, models.TierSafety, result.Tier)
	assert.False(t, result.Suppressed)
}

func TestRouter_Deduplication_24h(t *testing.T) {
	r := newTestRouter()
	input := EscalationRouterInput{
		CardDifferentialID: "RENAL_CONTRAINDICATION",
		PatientID:          "P007",
	}
	// First call: should go through
	result1 := r.RouteCard(input)
	assert.False(t, result1.Suppressed)
	assert.Equal(t, models.TierSafety, result1.Tier)

	// Second call: same patient + same card type within 24h → deduplicated
	result2 := r.RouteCard(input)
	assert.True(t, result2.Suppressed)
	assert.Equal(t, "deduplicated", result2.SuppressionReason)
}
