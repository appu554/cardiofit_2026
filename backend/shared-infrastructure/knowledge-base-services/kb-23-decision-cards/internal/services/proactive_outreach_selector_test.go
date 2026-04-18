package services

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutreach_HighRiskStable_Included(t *testing.T) {
	predictions := []PredictedRiskSummary{
		{
			PatientID:               "P-HIGH-STABLE",
			RiskScore:               55,
			RiskTier:                "HIGH",
			RiskSummary:             "Elevated HbA1c trend with rising BP",
			RecommendedAction:       "Schedule proactive outreach call",
			CounterfactualReduction: 12.5,
			ModifiableDriverNames:   []string{"HbA1c", "SystolicBP"},
		},
	}
	paiTiers := map[string]string{"P-HIGH-STABLE": "LOW"}
	contactDays := map[string]int{}

	items := SelectProactiveOutreach(
		predictions, paiTiers, contactDays,
		8, 25, []string{"CRITICAL", "HIGH"}, 14,
	)

	require.Len(t, items, 1)
	assert.Equal(t, "P-HIGH-STABLE", items[0].PatientID)
	assert.Equal(t, 55.0, items[0].RiskScore)
	assert.Equal(t, "HIGH", items[0].RiskTier)
	assert.Equal(t, "Schedule proactive outreach call", items[0].RecommendedAction)
	assert.InDelta(t, 12.5, items[0].CounterfactualReduction, 0.01)
	assert.Equal(t, []string{"HbA1c", "SystolicBP"}, items[0].ModifiableDrivers)
}

func TestOutreach_PAICritical_Excluded(t *testing.T) {
	predictions := []PredictedRiskSummary{
		{
			PatientID:             "P-CRIT",
			RiskScore:             60,
			RiskTier:              "HIGH",
			RiskSummary:           "Acute decompensation risk",
			RecommendedAction:     "Immediate clinical review",
			ModifiableDriverNames: []string{"eGFR"},
		},
	}
	// PAI tier is CRITICAL → already in urgent worklist → excluded.
	paiTiers := map[string]string{"P-CRIT": "CRITICAL"}
	contactDays := map[string]int{}

	items := SelectProactiveOutreach(
		predictions, paiTiers, contactDays,
		8, 25, []string{"CRITICAL", "HIGH"}, 14,
	)

	assert.Empty(t, items, "patient with CRITICAL PAI tier should be excluded from proactive outreach")
}

func TestOutreach_RecentlyContacted_Excluded(t *testing.T) {
	predictions := []PredictedRiskSummary{
		{
			PatientID:         "P-RECENT",
			RiskScore:         50,
			RiskTier:          "HIGH",
			RiskSummary:       "Moderate risk with recent contact",
			RecommendedAction: "Follow-up call",
		},
	}
	paiTiers := map[string]string{"P-RECENT": "LOW"}
	// Contacted 5 days ago — cooldown is 14 days → excluded.
	contactDays := map[string]int{"P-RECENT": 5}

	items := SelectProactiveOutreach(
		predictions, paiTiers, contactDays,
		8, 25, []string{"CRITICAL", "HIGH"}, 14,
	)

	assert.Empty(t, items, "patient contacted 5 days ago should be excluded (14-day cooldown)")
}

func TestOutreach_DailyCap_Enforced(t *testing.T) {
	// Create 12 eligible patients with risk scores 40..51.
	predictions := make([]PredictedRiskSummary, 12)
	for i := range predictions {
		predictions[i] = PredictedRiskSummary{
			PatientID:         fmt.Sprintf("P-%02d", i),
			RiskScore:         float64(40 + i),
			RiskTier:          "HIGH",
			RiskSummary:       "Elevated risk",
			RecommendedAction: "Outreach call",
		}
	}

	paiTiers := map[string]string{}
	contactDays := map[string]int{}

	items := SelectProactiveOutreach(
		predictions, paiTiers, contactDays,
		8, 25, []string{"CRITICAL", "HIGH"}, 14,
	)

	require.Len(t, items, 8, "should cap at maxItems=8")

	// Verify sorted descending — top 8 should be patients with scores 51..44.
	assert.Equal(t, "P-11", items[0].PatientID)
	assert.Equal(t, 51.0, items[0].RiskScore)
	assert.Equal(t, "P-04", items[7].PatientID)
	assert.Equal(t, 44.0, items[7].RiskScore)
}
