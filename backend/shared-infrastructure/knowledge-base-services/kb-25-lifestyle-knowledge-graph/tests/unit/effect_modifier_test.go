package unit

import (
	"math"
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/clients"
	"kb-25-lifestyle-knowledge-graph/internal/models"
	"kb-25-lifestyle-knowledge-graph/internal/services"
)

func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestComputeModifiedEffect_AgeOver65(t *testing.T) {
	base := models.EffectDescriptor{EffectSize: -12.0, EffectUnit: "mg/dL"}
	patient := &clients.PatientSnapshot{Age: 70}
	modifiers := []models.ModifierRef{
		{ContextCode: "AGE_GT_65", Multiplier: 0.75, Condition: "age > 65"},
	}

	result := services.ComputeModifiedEffect(base, patient, modifiers)
	expected := -12.0 * 0.75
	if result.EffectSize != expected {
		t.Errorf("expected %f, got %f", expected, result.EffectSize)
	}
}

func TestComputeModifiedEffect_NoModifiersApply(t *testing.T) {
	base := models.EffectDescriptor{EffectSize: -12.0, EffectUnit: "mg/dL"}
	patient := &clients.PatientSnapshot{Age: 40}
	modifiers := []models.ModifierRef{
		{ContextCode: "AGE_GT_65", Multiplier: 0.75, Condition: "age > 65"},
	}

	result := services.ComputeModifiedEffect(base, patient, modifiers)
	if result.EffectSize != -12.0 {
		t.Errorf("expected -12.0 (unmodified), got %f", result.EffectSize)
	}
}

func TestComputeModifiedEffect_CKDReduces(t *testing.T) {
	base := models.EffectDescriptor{EffectSize: -10.0, EffectUnit: "mg/dL"}
	patient := &clients.PatientSnapshot{EGFR: 25}
	modifiers := []models.ModifierRef{
		{ContextCode: "CKD_STAGE_45", Multiplier: 0.50, Condition: "eGFR < 30"},
	}

	result := services.ComputeModifiedEffect(base, patient, modifiers)
	expected := -10.0 * 0.50
	if result.EffectSize != expected {
		t.Errorf("expected %f, got %f", expected, result.EffectSize)
	}
}

func TestAdherenceAdjust(t *testing.T) {
	effect := -12.0
	adherence := 0.70
	result := services.AdherenceAdjust(effect, adherence)
	expected := -12.0 * 0.70
	if !floatEquals(result, expected) {
		t.Errorf("expected %f, got %f", expected, result)
	}
}
