package unit

import (
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/models"
)

func TestMealTemplate_RatiosSumTo100(t *testing.T) {
	tmpl := models.MealTemplate{
		Code:         "PLATE_MODEL_VFR",
		Goal:         "visceral_fat",
		VegetablePct: 50,
		ProteinPct:   25,
		CarbPct:      25,
		FatPct:       0,
	}
	sum := tmpl.VegetablePct + tmpl.ProteinPct + tmpl.CarbPct + tmpl.FatPct
	if sum != 100 {
		t.Errorf("plate model ratios should sum to 100, got %d", sum)
	}
}

func TestMealTemplate_Validate_InvalidSum(t *testing.T) {
	tmpl := models.MealTemplate{
		Code:         "BAD",
		Goal:         "test",
		VegetablePct: 50,
		ProteinPct:   50,
		CarbPct:      50,
	}
	if err := tmpl.Validate(); err == nil {
		t.Error("expected validation error for ratios summing to 150")
	}
}
