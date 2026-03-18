package models

// LifestyleRecommendationRequest is the input for /recommend-lifestyle.
type LifestyleRecommendationRequest struct {
	PatientID   string                    `json:"patient_id" binding:"required"`
	Target      string                    `json:"target" binding:"required"`
	Constraints RecommendationConstraints `json:"constraints"`
	Adherence   *AdherenceInput           `json:"adherence,omitempty"`
}

// RecommendationConstraints filters recommendations by patient context.
type RecommendationConstraints struct {
	DietType       string   `json:"diet_type,omitempty"`
	Region         string   `json:"region,omitempty"`
	Allergens      []string `json:"allergens,omitempty"`
	CookingAbility string   `json:"cooking_ability,omitempty"`
	CostTier       string   `json:"cost_tier,omitempty"`
}

// AdherenceInput carries recent adherence signals from KB-21.
type AdherenceInput struct {
	ProteinAdherencePct float64 `json:"protein_adherence_pct,omitempty"`
	MealQualityScore    float64 `json:"meal_quality_score,omitempty"`
	DailySteps          int     `json:"daily_steps,omitempty"`
	ExerciseSessions    int     `json:"exercise_sessions_per_week,omitempty"`
}

// LifestyleRecommendationResult is the output of the recommendation engine.
type LifestyleRecommendationResult struct {
	PatientID               string                   `json:"patient_id"`
	Target                  string                   `json:"target"`
	FoodRecommendations     []FoodRecommendation     `json:"food_recommendations,omitempty"`
	ExerciseRecommendations []ExerciseRecommendation `json:"exercise_recommendations,omitempty"`
	MealTemplate            *MealTemplate            `json:"meal_template,omitempty"`
}

// FoodRecommendation is a single food item recommendation.
type FoodRecommendation struct {
	FoodCode  string  `json:"food_code"`
	FoodName  string  `json:"food_name"`
	ServingG  float64 `json:"serving_g"`
	ProteinG  float64 `json:"protein_g"`
	LeucineG  float64 `json:"leucine_g,omitempty"`
	Rationale string  `json:"rationale"`
}

// ExerciseRecommendation is a single exercise prescription recommendation.
type ExerciseRecommendation struct {
	ExerciseCode string  `json:"exercise_code"`
	ExerciseName string  `json:"exercise_name"`
	DurationMin  int     `json:"duration_min"`
	FreqPerWeek  int     `json:"freq_per_week"`
	METValue     float64 `json:"met_value"`
	SafetyTier   string  `json:"safety_tier"`
}
