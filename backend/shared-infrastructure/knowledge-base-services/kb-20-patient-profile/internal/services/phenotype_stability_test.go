package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-patient-profile/internal/models"
)

// defaultTestConfig returns a StabilityConfig suitable for most test cases.
func defaultTestConfig() StabilityConfig {
	return StabilityConfig{
		DwellMinWeeks:          4,
		DwellExtendedWeeks:     8,
		FlapLookbackDays:       90,
		FlapMinOscillations:    2,
		HighMembershipProb:     0.7,
		ModerateMembershipProb: 0.4,
		CGMStartGraceWeeks:     2,
		CGMStopGraceWeeks:      2,
		ConservatismRank: map[string]int{
			"STABLE_CONTROLLED": 1,
			"STABLE_MEDICATED":  2,
			"IMPROVING":         3,
			"WORSENING":         4,
			"UNCONTROLLED":      5,
		},
	}
}

func TestStability_FirstAssignment_Accepted(t *testing.T) {
	engine := NewStabilityEngine()
	input := StabilityInput{
		PatientID:       "P001",
		RawClusterLabel: "STABLE_CONTROLLED",
		MembershipProb:  0.85,
		RunDate:         time.Now(),
		CurrentState:    nil,
		Config:          defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, "P001", decision.PatientID)
	assert.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "STABLE_CONTROLLED", decision.StableClusterLabel)
	assert.Equal(t, models.TransitionTypeInitial, decision.TransitionType)
}

func TestStability_SameCluster_NoChange(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P002",
		RawClusterLabel: "IMPROVING",
		MembershipProb:  0.9,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P002",
			CurrentStableCluster: "IMPROVING",
			StableSince:          now.AddDate(0, -2, 0),
			DwellDays:            60,
			Confidence:           0.9,
		},
		Config: defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "IMPROVING", decision.StableClusterLabel)
	assert.Empty(t, decision.TransitionType, "no transition when cluster unchanged")
}

func TestStability_DifferentCluster_WithinDwell_Held(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P003",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.8,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P003",
			CurrentStableCluster: "IMPROVING",
			StableSince:          now.AddDate(0, 0, -14), // 2 weeks — within 4-week dwell
			DwellDays:            14,
			Confidence:           0.8,
		},
		Config: defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionHoldDwell, decision.Decision)
	assert.Equal(t, "IMPROVING", decision.StableClusterLabel, "stable cluster unchanged during dwell")
}

func TestStability_DifferentCluster_PastDwell_Accepted(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P004",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.8,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P004",
			CurrentStableCluster: "IMPROVING",
			StableSince:          now.AddDate(0, 0, -35), // 5 weeks — past 4-week dwell
			DwellDays:            35,
			Confidence:           0.8,
			PendingRawCluster:    strPtr("WORSENING"),
			PendingSince:         timePtr(now.AddDate(0, 0, -35)),
		},
		Config: defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "WORSENING", decision.StableClusterLabel)
	assert.Equal(t, models.TransitionTypeGenuine, decision.TransitionType)
}

func TestStability_DifferentCluster_WithinDwell_OverrideEvent_Accepted(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P005",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.8,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P005",
			CurrentStableCluster: "IMPROVING",
			StableSince:          now.AddDate(0, 0, -7), // 1 week — within dwell
			DwellDays:            7,
			Confidence:           0.8,
		},
		OverrideEvents: []models.OverrideEvent{
			{EventType: "CKM_STAGE_TRANSITION", EventDate: now.AddDate(0, 0, -3)},
		},
		Config: defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "WORSENING", decision.StableClusterLabel)
}

func TestStability_CKMStageTransition_OverridesDwell(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P006",
		RawClusterLabel: "UNCONTROLLED",
		MembershipProb:  0.75,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P006",
			CurrentStableCluster: "STABLE_MEDICATED",
			StableSince:          now.AddDate(0, 0, -10),
			DwellDays:            10,
			Confidence:           0.8,
		},
		OverrideEvents: []models.OverrideEvent{
			{EventType: "CKM_STAGE_TRANSITION", EventDate: now.AddDate(0, 0, -2), Domain: "renal"},
		},
		Config: defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	require.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "UNCONTROLLED", decision.StableClusterLabel)
	assert.Equal(t, models.TransitionTypeOverride, decision.TransitionType)
	assert.Contains(t, decision.TriggerEvent, "CKM_STAGE_TRANSITION")
}

func TestStability_FlapDetected_HeldAtConservative(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P007",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.7,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P007",
			CurrentStableCluster: "IMPROVING",
			StableSince:          now.AddDate(0, 0, -40),
			DwellDays:            40,
			Confidence:           0.7,
			IsFlapping:           true,
			FlapCount:            3,
			FlapPair:             []string{"IMPROVING", "WORSENING"},
		},
		Config: defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionHoldFlap, decision.Decision)
	// IMPROVING has rank 3, WORSENING has rank 4 → IMPROVING is more conservative
	assert.Equal(t, "IMPROVING", decision.StableClusterLabel, "should hold at more conservative cluster")
}

func TestStability_FlapDetected_OverrideStillWorks(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P008",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.7,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P008",
			CurrentStableCluster: "IMPROVING",
			StableSince:          now.AddDate(0, 0, -40),
			DwellDays:            40,
			Confidence:           0.7,
			IsFlapping:           true,
			FlapCount:            3,
			FlapPair:             []string{"IMPROVING", "WORSENING"},
		},
		OverrideEvents: []models.OverrideEvent{
			{EventType: "CKM_STAGE_TRANSITION", EventDate: now.AddDate(0, 0, -1)},
		},
		Config: defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "WORSENING", decision.StableClusterLabel)
}

func TestStability_NoiseLabel_HoldPrevious(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P009",
		RawClusterLabel: "-1",
		MembershipProb:  0.1,
		IsNoise:         true,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P009",
			CurrentStableCluster: "STABLE_CONTROLLED",
			StableSince:          now.AddDate(0, -3, 0),
			DwellDays:            90,
			Confidence:           0.9,
		},
		Config: defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionHoldDwell, decision.Decision)
	assert.Equal(t, "STABLE_CONTROLLED", decision.StableClusterLabel, "noise should keep previous stable")
	assert.Contains(t, decision.Reason, "noise label held")
}

func TestStability_LowConfidence_Flagged(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P010",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.3, // below 0.4 threshold
		RunDate:         now,
		CurrentState:    nil, // first assignment
		Config:          defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "WORSENING", decision.StableClusterLabel)
	assert.Less(t, decision.Confidence, 0.4, "confidence should reflect low membership probability")
}

func TestStability_CGMStarted_GracePeriod(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	cgmStart := now.AddDate(0, 0, -10) // 10 days ago — within 2-week grace
	input := StabilityInput{
		PatientID:       "P011",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.8,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P011",
			CurrentStableCluster: "IMPROVING",
			StableSince:          now.AddDate(0, -3, 0),
			DwellDays:            90,
			Confidence:           0.8,
		},
		CGMStartDate: &cgmStart,
		Config:       defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionHoldDwell, decision.Decision)
	assert.Equal(t, "IMPROVING", decision.StableClusterLabel)
	assert.Contains(t, decision.Reason, "CGM data modality grace period")
}

func TestStability_TransitionWithDomainDriver(t *testing.T) {
	engine := NewStabilityEngine()
	now := time.Now()
	input := StabilityInput{
		PatientID:       "P012",
		RawClusterLabel: "WORSENING",
		MembershipProb:  0.85,
		RunDate:         now,
		CurrentState: &models.PatientClusterState{
			PatientID:            "P012",
			CurrentStableCluster: "IMPROVING",
			StableSince:          now.AddDate(0, 0, -35),
			DwellDays:            35,
			Confidence:           0.85,
			PendingRawCluster:    strPtr("WORSENING"),
			PendingSince:         timePtr(now.AddDate(0, 0, -35)),
		},
		DomainDriver: "bp_systolic",
		Config:       defaultTestConfig(),
	}

	decision := engine.Evaluate(input)

	assert.Equal(t, models.DecisionAccept, decision.Decision)
	assert.Equal(t, "WORSENING", decision.StableClusterLabel)
	assert.Equal(t, "bp_systolic", decision.DomainDriver)
}

// --- helpers ---

func strPtr(s string) *string    { return &s }
func timePtr(t time.Time) *time.Time { return &t }
