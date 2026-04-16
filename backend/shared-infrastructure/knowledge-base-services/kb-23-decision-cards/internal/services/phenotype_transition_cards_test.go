package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test 1: A StabilityDecision with Decision=ACCEPT and TransitionType=GENUINE
// should produce a PhenotypeTransitionCard with TemplateID "dc-phenotype-transition-v1".
func TestPhenotypeCards_GenuineTransition_ProducesCard(t *testing.T) {
	decision := PhenotypeStabilityDecision{
		PatientID:          "patient-001",
		RawClusterLabel:    "CLUSTER_B",
		StableClusterLabel: "CLUSTER_B",
		Decision:           "ACCEPT",
		TransitionType:     "GENUINE",
		DomainDriver:       "GLYCAEMIC",
		Confidence:         0.92,
		PreviousCluster:    "CLUSTER_A",
	}

	cards := EvaluatePhenotypeTransition(decision)

	assert.Len(t, cards, 1)
	card := cards[0]
	assert.Equal(t, "dc-phenotype-transition-v1", card.TemplateID)
	assert.Equal(t, "patient-001", card.PatientID)
	assert.Equal(t, "CLUSTER_A", card.PreviousCluster)
	assert.Equal(t, "CLUSTER_B", card.NewCluster)
	assert.Equal(t, "GLYCAEMIC", card.DomainDriver)
	assert.InDelta(t, 0.92, card.Confidence, 0.001)
	assert.False(t, card.SuppressInertia)
}

// Test 2: A StabilityDecision with Decision=HOLD_FLAP should produce a card
// with TemplateID "dc-phenotype-flap-warning-v1".
func TestPhenotypeCards_FlapWarning_ProducesCard(t *testing.T) {
	decision := PhenotypeStabilityDecision{
		PatientID:          "patient-002",
		RawClusterLabel:    "CLUSTER_C",
		StableClusterLabel: "CLUSTER_B",
		Decision:           "HOLD_FLAP",
		DomainDriver:       "RENAL",
		Confidence:         0.55,
		FlapPair:           []string{"CLUSTER_B", "CLUSTER_C"},
	}

	cards := EvaluatePhenotypeTransition(decision)

	assert.Len(t, cards, 1)
	card := cards[0]
	assert.Equal(t, "dc-phenotype-flap-warning-v1", card.TemplateID)
	assert.Equal(t, "patient-002", card.PatientID)
	assert.Equal(t, "CLUSTER_B ↔ CLUSTER_C", card.FlapPair)
	assert.Equal(t, "RENAL", card.DomainDriver)
	assert.InDelta(t, 0.55, card.Confidence, 0.001)
	assert.False(t, card.SuppressInertia)
}

// Test 3: A StabilityDecision with Decision=ACCEPT, TransitionType="" (same cluster,
// no transition), and StableClusterLabel="STABLE_CONTROLLED" should return
// SuppressInertia=true. This tells the inertia detector that this patient is
// phenotypically stable-good and should NOT be flagged for therapeutic inertia.
func TestPhenotypeCards_StableGoodCluster_SuppressesInertia(t *testing.T) {
	decision := PhenotypeStabilityDecision{
		PatientID:          "patient-003",
		RawClusterLabel:    "STABLE_CONTROLLED",
		StableClusterLabel: "STABLE_CONTROLLED",
		Decision:           "ACCEPT",
		TransitionType:     "", // same cluster, no transition
		DomainDriver:       "",
		Confidence:         0.98,
		PreviousCluster:    "STABLE_CONTROLLED",
	}

	cards := EvaluatePhenotypeTransition(decision)

	assert.Len(t, cards, 1)
	card := cards[0]
	assert.True(t, card.SuppressInertia)
	assert.Empty(t, card.TemplateID, "stable-good should not produce a template card")
	assert.Equal(t, "patient-003", card.PatientID)
	assert.Equal(t, "STABLE_CONTROLLED", card.NewCluster)
}

// Test 4: A StabilityDecision with Decision=HOLD_DWELL should produce NO card
// (dwell holds are silent — the engine is just waiting, nothing to report).
func TestPhenotypeCards_HoldDwell_NoCard(t *testing.T) {
	decision := PhenotypeStabilityDecision{
		PatientID:          "patient-004",
		RawClusterLabel:    "CLUSTER_A",
		StableClusterLabel: "CLUSTER_A",
		Decision:           "HOLD_DWELL",
		DomainDriver:       "LIPID",
		Confidence:         0.70,
	}

	cards := EvaluatePhenotypeTransition(decision)

	assert.Empty(t, cards, "HOLD_DWELL should produce no cards")
}
