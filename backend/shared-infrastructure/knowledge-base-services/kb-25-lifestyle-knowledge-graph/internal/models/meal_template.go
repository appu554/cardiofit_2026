package models

import "fmt"

// MealTemplate defines macronutrient ratios by metabolic goal.
// Used by VFRP Phase 2 plate model and PRP meal recommendations.
type MealTemplate struct {
	Code               string  `json:"code"`
	Goal               string  `json:"goal"` // glycemic_control | visceral_fat | renal_protection | balanced
	VegetablePct       int     `json:"vegetable_pct"`
	ProteinPct         int     `json:"protein_pct"`
	CarbPct            int     `json:"carb_pct"`
	FatPct             int     `json:"fat_pct"`
	MaxGIPerMeal       float64 `json:"max_gi_per_meal"`
	MinFiberPerMealG   float64 `json:"min_fiber_per_meal_g"`
	MinProteinPerMealG float64 `json:"min_protein_per_meal_g"`
}

// Validate checks that the macronutrient ratios sum to 100%.
func (m *MealTemplate) Validate() error {
	sum := m.VegetablePct + m.ProteinPct + m.CarbPct + m.FatPct
	if sum != 100 {
		return fmt.Errorf("macronutrient ratios must sum to 100%%, got %d%%", sum)
	}
	return nil
}
