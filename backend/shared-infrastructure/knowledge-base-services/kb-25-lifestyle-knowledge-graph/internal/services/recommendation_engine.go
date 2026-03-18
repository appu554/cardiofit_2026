package services

import (
	"kb-25-lifestyle-knowledge-graph/internal/graph"
	"kb-25-lifestyle-knowledge-graph/internal/models"

	"go.uber.org/zap"
)

// RecommendationEngine generates lifestyle recommendations based on
// patient constraints, adherence data, and the causal knowledge graph.
type RecommendationEngine struct {
	graphClient graph.GraphClient
	logger      *zap.Logger
}

// NewRecommendationEngine creates a new engine. Both graphClient and logger
// may be nil for unit-testing convenience.
func NewRecommendationEngine(graphClient graph.GraphClient, logger *zap.Logger) *RecommendationEngine {
	return &RecommendationEngine{graphClient: graphClient, logger: logger}
}

// GenerateRecommendation dispatches on Target to produce food/exercise recs.
func (r *RecommendationEngine) GenerateRecommendation(req models.LifestyleRecommendationRequest) *models.LifestyleRecommendationResult {
	result := &models.LifestyleRecommendationResult{
		PatientID: req.PatientID,
		Target:    req.Target,
	}

	switch req.Target {
	case "protein_increase":
		result.FoodRecommendations = r.proteinFoods(req.Constraints)
		result.ExerciseRecommendations = r.baseExercise()
	case "carb_substitution":
		result.FoodRecommendations = r.carbSubstitutions(req.Constraints)
	case "exercise_progression":
		result.ExerciseRecommendations = r.exerciseProgression(req.Adherence)
	case "sodium_reduction":
		result.FoodRecommendations = r.sodiumReductions(req.Constraints)
	}

	return result
}

func (r *RecommendationEngine) proteinFoods(c models.RecommendationConstraints) []models.FoodRecommendation {
	foods := []models.FoodRecommendation{
		{FoodCode: "egg_whole", FoodName: "Whole egg (boiled)", ServingG: 100, ProteinG: 13.0, LeucineG: 1.09, Rationale: "High-quality protein with complete amino acid profile"},
		{FoodCode: "moong_dal", FoodName: "Moong dal (cooked)", ServingG: 150, ProteinG: 12.0, LeucineG: 1.23, Rationale: "Plant protein staple with good leucine content"},
		{FoodCode: "dahi", FoodName: "Dahi / Curd (plain)", ServingG: 200, ProteinG: 7.0, LeucineG: 0.90, Rationale: "Probiotic benefit + moderate protein"},
	}

	if c.DietType == "vegan" {
		foods = foods[1:] // remove egg
	}

	return foods
}

func (r *RecommendationEngine) carbSubstitutions(c models.RecommendationConstraints) []models.FoodRecommendation {
	subs := []models.FoodRecommendation{
		{FoodCode: "brown_rice", FoodName: "Brown rice", ServingG: 150, Rationale: "GI 68 vs white rice GI 73 — lower postprandial spike"},
		{FoodCode: "ragi", FoodName: "Ragi / Finger millet", ServingG: 100, Rationale: "GI 54, high fiber (3.6g/100g), high calcium"},
	}

	if c.Region == "north_india" {
		subs = append(subs, models.FoodRecommendation{
			FoodCode:  "whole_wheat_atta",
			FoodName:  "Whole wheat atta (replace maida)",
			ServingG:  100,
			Rationale: "GI 49 vs maida GI 71 — significantly lower glycemic response",
		})
	}

	return subs
}

func (r *RecommendationEngine) baseExercise() []models.ExerciseRecommendation {
	return []models.ExerciseRecommendation{
		{ExerciseCode: "EX_POST_MEAL_WALK", ExerciseName: "Post-meal walk", DurationMin: 10, FreqPerWeek: 14, METValue: 3.5, SafetyTier: "T1_SAFE"},
	}
}

func (r *RecommendationEngine) exerciseProgression(adherence *models.AdherenceInput) []models.ExerciseRecommendation {
	recs := []models.ExerciseRecommendation{
		{ExerciseCode: "EX_BRISK_WALK", ExerciseName: "Brisk walking", DurationMin: 30, FreqPerWeek: 5, METValue: 4.3, SafetyTier: "T1_SAFE"},
	}

	if adherence != nil && adherence.ExerciseSessions >= 2 {
		recs = append(recs, models.ExerciseRecommendation{
			ExerciseCode: "EX_RESISTANCE_LIGHT", ExerciseName: "Light resistance training", DurationMin: 20, FreqPerWeek: 3, METValue: 3.5, SafetyTier: "T2_CONDITIONAL",
		})
	}

	return recs
}

func (r *RecommendationEngine) sodiumReductions(c models.RecommendationConstraints) []models.FoodRecommendation {
	return []models.FoodRecommendation{
		{FoodCode: "reduce_pickle", FoodName: "Reduce pickle / achar intake", Rationale: "Pickles contain 3000-5000mg sodium per 100g"},
		{FoodCode: "reduce_papad", FoodName: "Reduce papad", Rationale: "Single papad contains 400-600mg sodium"},
	}
}
