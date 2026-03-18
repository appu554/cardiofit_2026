// E-04: Meal Composition Template nodes for M3 protocol templates

// VFRP Plate Model: 50% vegetables, 25% protein, 25% carbohydrate
CREATE (:MealTemplate {code: 'PLATE_MODEL_VFR', goal: 'visceral_fat', vegetable_pct: 50, protein_pct: 25, carb_pct: 25, fat_pct: 0, max_gi_per_meal: 55, min_fiber_per_meal_g: 8, min_protein_per_meal_g: 15});

// PRP Protein Restoration: 30% vegetables, 35% protein, 25% carbohydrate, 10% fat
CREATE (:MealTemplate {code: 'PLATE_MODEL_PRP', goal: 'protein_restoration', vegetable_pct: 30, protein_pct: 35, carb_pct: 25, fat_pct: 10, max_gi_per_meal: 60, min_fiber_per_meal_g: 5, min_protein_per_meal_g: 20});

// CKD Protective: 40% vegetables, 20% protein, 30% carbohydrate, 10% fat (with sodium cap)
CREATE (:MealTemplate {code: 'PLATE_MODEL_CKD', goal: 'renal_protection', vegetable_pct: 40, protein_pct: 20, carb_pct: 30, fat_pct: 10, max_gi_per_meal: 55, min_fiber_per_meal_g: 6, min_protein_per_meal_g: 12});

// Prediabetes: 45% vegetables, 25% protein, 25% carbohydrate, 5% fat
CREATE (:MealTemplate {code: 'PLATE_MODEL_PREDIABETES', goal: 'glycemic_control', vegetable_pct: 45, protein_pct: 25, carb_pct: 25, fat_pct: 5, max_gi_per_meal: 50, min_fiber_per_meal_g: 10, min_protein_per_meal_g: 15});
