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
