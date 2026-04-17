package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

	cards := EvaluatePhenotypeTransition(decision, []string{"metformin", "empagliflozin"})

	assert.Len(t, cards, 1)
	card := cards[0]
	assert.Equal(t, "dc-phenotype-transition-v1", card.TemplateID)
	assert.Equal(t, "patient-001", card.PatientID)
	assert.Equal(t, "CLUSTER_A", card.PreviousCluster)
	assert.Equal(t, "CLUSTER_B", card.NewCluster)
	assert.Equal(t, "GLYCAEMIC", card.DomainDriver)
	assert.InDelta(t, 0.92, card.Confidence, 0.001)
	assert.Contains(t, card.Interpretation, "metformin")
	assert.Contains(t, card.Interpretation, "CLUSTER_A")
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

	cards := EvaluatePhenotypeTransition(decision, nil)

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

	cards := EvaluatePhenotypeTransition(decision, nil)

	assert.Len(t, cards, 1)
	card := cards[0]
	assert.True(t, card.SuppressInertia)
	assert.Empty(t, card.TemplateID, "stable-good should not produce a template card")
	assert.Equal(t, "patient-003", card.PatientID)
	assert.Equal(t, "STABLE_CONTROLLED", card.NewCluster)
}

// Test 4: Phenotype YAML templates load from disk and parse correctly.
func TestPhenotypeCards_TemplateLoadsFromDisk(t *testing.T) {
	templates := []string{
		"../../templates/phenotype/genuine_transition.yaml",
		"../../templates/phenotype/flap_warning.yaml",
	}

	for _, path := range templates {
		data, err := os.ReadFile(path)
		require.NoError(t, err, "template file should exist: %s", path)

		var parsed map[string]interface{}
		err = yaml.Unmarshal(data, &parsed)
		require.NoError(t, err, "template should parse as valid YAML: %s", path)

		assert.NotEmpty(t, parsed["template_id"], "template_id required in %s", path)
		assert.NotEmpty(t, parsed["node_id"], "node_id required in %s", path)
		assert.NotEmpty(t, parsed["differential_id"], "differential_id required in %s", path)
		assert.NotEmpty(t, parsed["fragments"], "fragments required in %s", path)
	}
}

// Test 5: Engagement-coincident transition. When an ACCEPT decision has
// TransitionType=GENUINE and previous != stable, a transition card is
// produced even when no medications are provided (nil patientMeds).
func TestPhenotypeCards_EngagementCoincident_ProducesCard(t *testing.T) {
	decision := PhenotypeStabilityDecision{
		PatientID:          "patient-005",
		RawClusterLabel:    "WORSENING",
		StableClusterLabel: "WORSENING",
		Decision:           "ACCEPT",
		TransitionType:     "GENUINE",
		DomainDriver:       "BEHAVIORAL",
		Confidence:         0.65,
		PreviousCluster:    "STABLE_CONTROLLED",
	}

	cards := EvaluatePhenotypeTransition(decision, nil)

	assert.Len(t, cards, 1)
	card := cards[0]
	assert.Equal(t, "dc-phenotype-transition-v1", card.TemplateID)
	assert.Equal(t, "STABLE_CONTROLLED", card.PreviousCluster)
	assert.Equal(t, "WORSENING", card.NewCluster)
	assert.Contains(t, card.Interpretation, "STABLE_CONTROLLED")
	assert.NotContains(t, card.Interpretation, "regimen", "no meds → no regimen text")
}
