package models

type Food struct {
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	NameLocal   string             `json:"name_local,omitempty"`
	Region      string             `json:"region"`
	DietType    string             `json:"diet_type"`
	Category    string             `json:"category"`
	FoodGroup   string             `json:"food_group"`
	Nutrients   map[string]float64 `json:"nutrients"`
	GI          float64            `json:"gi"`
	GL          float64            `json:"gl"`
	ServingSize float64            `json:"serving_size_g"`
	Fiber       float64            `json:"fiber_g"`
	Sodium      float64            `json:"sodium_mg"`
	Potassium   float64            `json:"potassium_mg"`
}

// LeucinePerServing returns leucine grams for a given serving weight in grams.
// Returns 0 if leucine data is not available in the nutrient map.
// MPS threshold: >= 2.5g leucine per meal for Grade A MPS stimulation.
func (f *Food) LeucinePerServing(servingGrams float64) float64 {
	leucinePer100g, ok := f.Nutrients["leucine"]
	if !ok || f.ServingSize == 0 {
		return 0
	}
	return leucinePer100g * servingGrams / 100.0
}

// MPSThresholdG is the leucine threshold for Grade A muscle protein synthesis stimulation.
const MPSThresholdG = 2.5
