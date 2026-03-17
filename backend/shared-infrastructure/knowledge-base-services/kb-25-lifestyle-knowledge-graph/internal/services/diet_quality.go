package services

import "math"

type DietQualityInput struct {
	FiberG           float64
	SodiumMg         float64
	PotassiumMg      float64
	SaturatedFatPct  float64
	AddedSugarG      float64
	WholeGrainPct    float64
	FruitVegServings float64
	ProcessedFoodPct float64
}

func ComputeDietQuality(input DietQualityInput) float64 {
	score := 0.0
	score += math.Min(input.FiberG/25.0, 1.0) * 12.5
	if input.SodiumMg <= 2300 {
		score += 12.5
	} else {
		score += math.Max(0, 12.5*(1.0-(input.SodiumMg-2300)/2700))
	}
	score += math.Min(input.PotassiumMg/3500.0, 1.0) * 12.5
	if input.SaturatedFatPct <= 7 {
		score += 12.5
	} else {
		score += math.Max(0, 12.5*(1.0-(input.SaturatedFatPct-7)/13))
	}
	if input.AddedSugarG <= 25 {
		score += 12.5
	} else {
		score += math.Max(0, 12.5*(1.0-(input.AddedSugarG-25)/35))
	}
	score += math.Min(input.WholeGrainPct/50.0, 1.0) * 12.5
	score += math.Min(input.FruitVegServings/5.0, 1.0) * 12.5
	if input.ProcessedFoodPct <= 20 {
		score += 12.5
	} else {
		score += math.Max(0, 12.5*(1.0-(input.ProcessedFoodPct-20)/60))
	}
	return math.Max(0, math.Min(100, score))
}
