package unit

import (
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/services"
)

func TestDietQualityScore_Perfect(t *testing.T) {
	input := services.DietQualityInput{
		FiberG: 30, SodiumMg: 1500, PotassiumMg: 3500,
		SaturatedFatPct: 5, AddedSugarG: 15, WholeGrainPct: 80,
		FruitVegServings: 7, ProcessedFoodPct: 5,
	}
	score := services.ComputeDietQuality(input)
	if score < 80 {
		t.Errorf("perfect diet should score > 80, got %f", score)
	}
}

func TestDietQualityScore_Poor(t *testing.T) {
	input := services.DietQualityInput{
		FiberG: 5, SodiumMg: 5000, PotassiumMg: 1000,
		SaturatedFatPct: 15, AddedSugarG: 60, WholeGrainPct: 10,
		FruitVegServings: 1, ProcessedFoodPct: 70,
	}
	score := services.ComputeDietQuality(input)
	if score > 40 {
		t.Errorf("poor diet should score < 40, got %f", score)
	}
}

func TestDietQualityScore_Clamped(t *testing.T) {
	input := services.DietQualityInput{
		FiberG: 50, SodiumMg: 500, PotassiumMg: 5000,
		SaturatedFatPct: 2, AddedSugarG: 5, WholeGrainPct: 100,
		FruitVegServings: 10, ProcessedFoodPct: 0,
	}
	score := services.ComputeDietQuality(input)
	if score > 100 || score < 0 {
		t.Errorf("score should be clamped 0-100, got %f", score)
	}
}
