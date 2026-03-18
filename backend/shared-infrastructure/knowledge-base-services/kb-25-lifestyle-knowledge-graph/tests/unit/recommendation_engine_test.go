package unit

import (
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/models"
	"kb-25-lifestyle-knowledge-graph/internal/services"
)

func TestRecommendLifestyle_ProteinIncrease(t *testing.T) {
	req := models.LifestyleRecommendationRequest{
		PatientID: "test-patient-1",
		Target:    "protein_increase",
		Constraints: models.RecommendationConstraints{
			DietType: "vegetarian",
			Region:   "south_india",
		},
	}

	engine := services.NewRecommendationEngine(nil, nil) // nil graph + logger for unit test
	result := engine.GenerateRecommendation(req)

	if result.Target != "protein_increase" {
		t.Errorf("expected target protein_increase, got %s", result.Target)
	}
	if len(result.FoodRecommendations) == 0 {
		t.Error("expected at least 1 food recommendation")
	}
}

func TestRecommendLifestyle_CarbSubstitution(t *testing.T) {
	req := models.LifestyleRecommendationRequest{
		PatientID: "test-patient-2",
		Target:    "carb_substitution",
		Constraints: models.RecommendationConstraints{
			Region: "north_india",
		},
	}

	engine := services.NewRecommendationEngine(nil, nil)
	result := engine.GenerateRecommendation(req)

	if result.Target != "carb_substitution" {
		t.Errorf("expected target carb_substitution, got %s", result.Target)
	}
}
