package unit

import (
	"testing"

	"kb-25-lifestyle-knowledge-graph/internal/models"
)

func TestFood_LeucinePerServing_Adequate(t *testing.T) {
	food := models.Food{
		Code:        "egg_whole",
		Nutrients:   map[string]float64{"protein": 13.0, "leucine": 1.1},
		ServingSize: 100,
	}
	// 2 eggs (200g) = 2.2g leucine → adequate for MPS (threshold 2.5g per meal)
	leucine := food.LeucinePerServing(200)
	if leucine < 2.0 {
		t.Errorf("expected leucine >= 2.0g for 200g eggs, got %.2f", leucine)
	}
}

func TestFood_LeucinePerServing_NoLeucineData(t *testing.T) {
	food := models.Food{
		Code:        "brown_rice",
		Nutrients:   map[string]float64{"protein": 2.6},
		ServingSize: 100,
	}
	leucine := food.LeucinePerServing(150)
	if leucine != 0 {
		t.Errorf("expected 0 leucine when not in nutrients, got %.2f", leucine)
	}
}
