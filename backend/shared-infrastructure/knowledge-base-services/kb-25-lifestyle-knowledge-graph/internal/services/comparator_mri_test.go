package services

import (
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/models"
)

func TestPrioritizeByMRIDomain(t *testing.T) {
	options := []models.ComparedOption{
		{Option: models.InterventionOption{Code: "bp_medication"}, Rank: 1},
		{Option: models.InterventionOption{Code: "post_meal_walking"}, Rank: 2},
		{Option: models.InterventionOption{Code: "sleep_hygiene"}, Rank: 3},
	}

	result := PrioritizeByMRIDomain(options, "Glucose Control")

	if result[0].Option.Code != "post_meal_walking" {
		t.Errorf("expected post_meal_walking first, got %s", result[0].Option.Code)
	}
	if !result[0].MRIBoost {
		t.Error("expected MRIBoost=true for post_meal_walking")
	}
}

func TestPrioritizeByMRIDomain_NoDriver(t *testing.T) {
	options := []models.ComparedOption{
		{Option: models.InterventionOption{Code: "bp_medication"}, Rank: 1},
	}
	result := PrioritizeByMRIDomain(options, "")
	if result[0].Rank != 1 {
		t.Error("expected no change when topDriver is empty")
	}
}

func TestPrioritizeByMRIDomain_MultipleMatches(t *testing.T) {
	options := []models.ComparedOption{
		{Option: models.InterventionOption{Code: "bp_medication"}, Rank: 1},
		{Option: models.InterventionOption{Code: "post_meal_walking"}, Rank: 2},
		{Option: models.InterventionOption{Code: "carb_quality"}, Rank: 3},
	}

	result := PrioritizeByMRIDomain(options, "Glucose Control")

	// Both post_meal_walking and carb_quality target Glucose Control
	if !result[0].MRIBoost || !result[1].MRIBoost {
		t.Error("expected both glucose-control options to have MRIBoost=true")
	}
	// Non-matching option should remain last
	if result[2].MRIBoost {
		t.Error("expected bp_medication to not have MRIBoost")
	}
}

func TestPrioritizeByMRIDomain_NoMatches(t *testing.T) {
	options := []models.ComparedOption{
		{Option: models.InterventionOption{Code: "bp_medication"}, Rank: 1},
		{Option: models.InterventionOption{Code: "sodium_reduction"}, Rank: 2},
	}

	result := PrioritizeByMRIDomain(options, "Glucose Control")

	// No match — order should be unchanged
	if result[0].Option.Code != "bp_medication" {
		t.Errorf("expected bp_medication first when no match, got %s", result[0].Option.Code)
	}
	for _, o := range result {
		if o.MRIBoost {
			t.Errorf("expected no MRIBoost flags when no domain matches, got one on %s", o.Option.Code)
		}
	}
}
