package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"kb-patient-profile/internal/models"
	"kb-patient-profile/internal/services"
)

// defaultConfig returns a StabilityConfig with the standard production values.
func defaultConfig() services.StabilityConfig {
	return services.StabilityConfig{
		DwellMinWeeks:          4,
		DwellExtendedWeeks:     8,
		FlapLookbackDays:       90,
		FlapMinOscillations:    2,
		HighMembershipProb:     0.7,
		ModerateMembershipProb: 0.4,
		CGMStartGraceWeeks:     2,
		CGMStopGraceWeeks:      4,
		ConservatismRank: map[string]int{
			"STABLE_CONTROLLED":     1,
			"STABLE_MEDICATED":      2,
			"PROGRESSIVE_GLYCAEMIC": 3,
			"CARDIORENAL_COMPLEX":   4,
			"HIGH_RISK_UNSTABLE":    5,
			"NOISE":                 6,
		},
	}
}

// TestPhenotypeHandler_StabilityHold verifies the stability engine holds a
// cluster change when dwell time is insufficient. This is the critical path:
// if the Python pipeline assigns WORSENING but the patient has only been in
// STABLE_CONTROLLED for 7 days (well under the 56-day extended dwell), the
// handler must keep STABLE_CONTROLLED.
func TestPhenotypeHandler_StabilityHold(t *testing.T) {
	engine := services.NewStabilityEngine()

	input := services.StabilityInput{
		PatientID:       "test-001",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.8,
		RunDate:         time.Now(),
		CurrentState: &models.PatientClusterState{
			PatientID:            "test-001",
			CurrentStableCluster: "STABLE_CONTROLLED",
			DwellDays:            7,
			Confidence:           0.9,
		},
		Config: defaultConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionHoldDwell, decision.Decision)
	assert.Equal(t, "STABLE_CONTROLLED", decision.StableClusterLabel)
	assert.Equal(t, "WORSENING", decision.RawClusterLabel)
}

// TestPhenotypeHandler_StabilityAccept verifies the stability engine accepts a
// cluster transition when dwell time has been met. The patient has been pending
// in WORSENING for 35 days (past the 28-day standard dwell for rank 3+).
func TestPhenotypeHandler_StabilityAccept(t *testing.T) {
	engine := services.NewStabilityEngine()
	now := time.Now()

	input := services.StabilityInput{
		PatientID:       "test-002",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.85,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "test-002",
			CurrentStableCluster: "PROGRESSIVE_GLYCAEMIC",
			DwellDays:            35,
			Confidence:           0.8,
			PendingRawCluster:    strPtr("WORSENING"),
			PendingSince:         timePtr(now.AddDate(0, 0, -35)),
		},
		Config: defaultConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "WORSENING", decision.StableClusterLabel)
	assert.Equal(t, models.TransitionTypeGenuine, decision.TransitionType)
}

func strPtr(s string) *string       { return &s }
func timePtr(t time.Time) *time.Time { return &t }
