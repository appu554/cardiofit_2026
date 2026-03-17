package unit

import (
	"math"
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/models"
	"kb-25-lifestyle-knowledge-graph/internal/services"
)

const floatTolerance = 1e-9

func TestComputeNetEffect_SingleHop(t *testing.T) {
	chain := models.CausalChain{
		Components: []models.ChainComponent{
			{
				Effect: models.EffectDescriptor{
					EffectSize:    -12.0,
					EffectUnit:    "mg/dL",
					EvidenceGrade: "A",
				},
			},
		},
	}

	net := services.ComputeNetEffect(chain.Components)
	if net.EffectSize != -12.0 {
		t.Errorf("expected -12.0, got %f", net.EffectSize)
	}
	if net.EvidenceGrade != "A" {
		t.Errorf("expected grade A, got %s", net.EvidenceGrade)
	}
}

func TestComputeNetEffect_MultiHop(t *testing.T) {
	chain := models.CausalChain{
		Components: []models.ChainComponent{
			{Effect: models.EffectDescriptor{EffectSize: 0.35, EvidenceGrade: "A"}},
			{Effect: models.EffectDescriptor{EffectSize: 0.20, EvidenceGrade: "A"}},
			{Effect: models.EffectDescriptor{EffectSize: -12.0, EvidenceGrade: "B"}},
		},
	}

	net := services.ComputeNetEffect(chain.Components)
	expected := 0.35 * 0.20 * -12.0
	if math.Abs(net.EffectSize-expected) > floatTolerance {
		t.Errorf("expected %f, got %f", expected, net.EffectSize)
	}
	if net.EvidenceGrade != "B" {
		t.Errorf("expected grade B (weakest link), got %s", net.EvidenceGrade)
	}
}

func TestWeakestGrade(t *testing.T) {
	tests := []struct {
		grades   []string
		expected string
	}{
		{[]string{"A", "A", "A"}, "A"},
		{[]string{"A", "B", "A"}, "B"},
		{[]string{"A", "C", "B"}, "C"},
		{[]string{"D"}, "D"},
	}

	for _, tc := range tests {
		got := services.WeakestGrade(tc.grades)
		if got != tc.expected {
			t.Errorf("WeakestGrade(%v) = %s, want %s", tc.grades, got, tc.expected)
		}
	}
}
