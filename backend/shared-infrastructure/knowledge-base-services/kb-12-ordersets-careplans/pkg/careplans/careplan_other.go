// Package careplans provides chronic disease care plan templates
// Other care plans: CKD, Depression, Osteoporosis + Aggregator functions
package careplans

import (
	"time"

	"kb-12-ordersets-careplans/internal/models"
)

// GetOtherCarePlans returns CKD, Depression, and Osteoporosis care plans
func GetOtherCarePlans() []*models.CarePlanTemplate {
	return []*models.CarePlanTemplate{
		createCKDCarePlan(),
		createDepressionCarePlan(),
		createOsteoporosisCarePlan(),
	}
}

// GetAllCarePlans returns all 12 chronic care plans from all categories
func GetAllCarePlans() []*models.CarePlanTemplate {
	allPlans := make([]*models.CarePlanTemplate, 0, 12)
	allPlans = append(allPlans, GetMetabolicCarePlans()...)       // 3: Diabetes, Obesity, Hypothyroidism
	allPlans = append(allPlans, GetCardiovascularCarePlans()...)  // 4: HTN, HF, CAD, AFib
	allPlans = append(allPlans, GetRespiratoryCarePlans()...)     // 2: COPD, Asthma
	allPlans = append(allPlans, GetOtherCarePlans()...)           // 3: CKD, Depression, Osteoporosis
	return allPlans
}

// GetCarePlanCount returns count by category
func GetCarePlanCount() map[string]int {
	return map[string]int{
		"metabolic":      3,
		"cardiovascular": 4,
		"respiratory":    2,
		"renal":          1,
		"mental_health":  1,
		"musculoskeletal": 1,
		"total":          12,
	}
}

// createCKDCarePlan creates a comprehensive CKD management care plan
// Based on KDIGO 2024 Guidelines (Kidney Disease: Improving Global Outcomes)
func createCKDCarePlan() *models.CarePlanTemplate {
	now := time.Now()

	goals := []models.CarePlanGoal{
		{
			GoalID:      "CKD-GOAL-001",
			Description: "Slow CKD progression",
			Category:    "disease_modification",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "CKD-TGT-001", Metric: "eGFR_decline", Value: "< 5 mL/min/1.73m²/year", Timeframe: "annual"},
				{TargetID: "CKD-TGT-002", Metric: "UACR", Value: "≥ 30% reduction or < 30 mg/g", Timeframe: "6 months"},
			},
			Status: "active",
		},
		{
			GoalID:      "CKD-GOAL-002",
			Description: "Optimize blood pressure control",
			Category:    "cardiovascular_protection",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "CKD-TGT-003", Metric: "systolic_BP", Value: "< 120 mmHg (if tolerated)", Timeframe: "3 months"},
				{TargetID: "CKD-TGT-004", Metric: "diastolic_BP", Value: "< 80 mmHg", Timeframe: "3 months"},
			},
			Status: "active",
		},
		{
			GoalID:      "CKD-GOAL-003",
			Description: "Manage glycemia in diabetic CKD",
			Category:    "metabolic",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "CKD-TGT-005", Metric: "HbA1c", Value: "< 7.0% (individualize 7-8% if hypoglycemia risk)", Timeframe: "3 months"},
			},
			Status:    "conditional",
			Condition: "diabetes_present = true",
		},
		{
			GoalID:      "CKD-GOAL-004",
			Description: "Prevent and manage complications",
			Category:    "complication_prevention",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "CKD-TGT-006", Metric: "hemoglobin", Value: "10-11.5 g/dL (avoid > 13)", Timeframe: "ongoing"},
				{TargetID: "CKD-TGT-007", Metric: "phosphorus", Value: "< 4.5 mg/dL", Timeframe: "ongoing"},
				{TargetID: "CKD-TGT-008", Metric: "bicarbonate", Value: "≥ 22 mEq/L", Timeframe: "ongoing"},
				{TargetID: "CKD-TGT-009", Metric: "potassium", Value: "< 5.5 mEq/L", Timeframe: "ongoing"},
			},
			Status: "active",
		},
		{
			GoalID:      "CKD-GOAL-005",
			Description: "Reduce cardiovascular risk",
			Category:    "cardiovascular",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "CKD-TGT-010", Metric: "LDL_cholesterol", Value: "< 70 mg/dL (high risk)", Timeframe: "6 months"},
				{TargetID: "CKD-TGT-011", Metric: "smoking_status", Value: "quit", Timeframe: "6 months"},
			},
			Status: "active",
		},
	}

	activities := []models.Activity{
		// First-line Disease-Modifying Therapy
		{
			ActivityID:  "CKD-ACT-001",
			Type:        "medication",
			Description: "ACE inhibitor or ARB (first-line for proteinuric CKD)",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":           "Lisinopril",
						"rxnorm":         "29046",
						"dose":           "2.5-40 mg",
						"route":          "oral",
						"frequency":      "once daily",
						"indication":     "albuminuria > 30 mg/g OR diabetic CKD",
						"titration":      "start low, uptitrate every 2-4 weeks",
						"monitoring":     "K+ and creatinine 1-2 weeks after initiation/dose change",
						"hold_criteria":  "K+ > 5.5 or creatinine rise > 30%",
					},
					{
						"name":           "Losartan",
						"rxnorm":         "52175",
						"dose":           "25-100 mg",
						"route":          "oral",
						"frequency":      "once daily",
						"alternative_to": "ACEi if cough/angioedema",
					},
				},
				"evidence": "reduces proteinuria and slows progression",
				"kdigo_recommendation": "Grade 1B",
			},
			Status:    "conditional",
			Condition: "UACR > 30 mg/g OR diabetes_present",
		},
		{
			ActivityID:  "CKD-ACT-002",
			Type:        "medication",
			Description: "SGLT2 inhibitor (major CKD benefit)",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Dapagliflozin",
						"rxnorm":       "1488564",
						"dose":         "10 mg",
						"route":        "oral",
						"frequency":    "once daily",
						"indication":   "CKD with eGFR ≥ 20 (continue if started at higher eGFR)",
						"benefit":      "40% reduction in CKD progression (DAPA-CKD)",
						"note":         "effective regardless of diabetes status",
					},
					{
						"name":         "Empagliflozin",
						"rxnorm":       "1545653",
						"dose":         "10 mg",
						"route":        "oral",
						"frequency":    "once daily",
						"indication":   "CKD with eGFR ≥ 20",
						"benefit":      "EMPA-KIDNEY showed similar benefits",
					},
				},
				"monitoring":     []string{"ketoacidosis symptoms", "genital infections", "volume status"},
				"kdigo_recommendation": "Grade 1A",
				"critical_importance": "START EARLY - benefits occur at higher eGFR",
			},
			Status: "active",
		},
		{
			ActivityID:  "CKD-ACT-003",
			Type:        "medication",
			Description: "Non-steroidal MRA (finerenone) for diabetic CKD",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Finerenone",
						"rxnorm":       "2479702",
						"dose":         "10-20 mg",
						"route":        "oral",
						"frequency":    "once daily",
						"indication":   "diabetic CKD with UACR ≥ 30 mg/g on max tolerated ACEi/ARB",
						"starting_dose": "10 mg if eGFR 25-60, 20 mg if eGFR > 60",
						"monitoring":   "K+ at baseline, 4 weeks, then regularly",
					},
				},
				"benefit": "reduces CV events and CKD progression in T2DM",
				"trials":  "FIDELIO-DKD, FIGARO-DKD",
			},
			Status:    "conditional",
			Condition: "diabetic_CKD AND UACR >= 30 AND on_max_ACEi_ARB",
		},
		{
			ActivityID:  "CKD-ACT-004",
			Type:        "medication",
			Description: "Statin therapy for CV risk reduction",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Atorvastatin",
						"rxnorm":       "83367",
						"dose":         "20-40 mg",
						"route":        "oral",
						"frequency":    "once daily",
						"indication":   "CKD G3a-G5 not on dialysis, age ≥ 50",
						"note":         "no dose adjustment for renal function",
					},
				},
				"kdigo_recommendation": "Grade 1A for age ≥ 50 with CKD",
			},
			Status: "active",
		},
		{
			ActivityID:  "CKD-ACT-005",
			Type:        "medication",
			Description: "Anemia management with ESA (if Hgb < 10)",
			Frequency:   "weekly to monthly",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Epoetin alfa",
						"rxnorm":       "3221",
						"dose":         "50-100 units/kg",
						"route":        "subcutaneous",
						"frequency":    "1-3 times weekly",
						"target":       "Hgb 10-11.5 g/dL (avoid > 13)",
					},
					{
						"name":         "Darbepoetin alfa",
						"rxnorm":       "226906",
						"dose":         "0.45 mcg/kg",
						"route":        "subcutaneous",
						"frequency":    "every 2-4 weeks",
					},
				},
				"iron_first": "ensure iron replete (ferritin > 100, TSAT > 20%) before ESA",
				"monitoring": "Hgb monthly until stable, then every 3 months",
			},
			Status:    "conditional",
			Condition: "hemoglobin < 10 g/dL AND iron_replete",
		},
		{
			ActivityID:  "CKD-ACT-006",
			Type:        "medication",
			Description: "Phosphate binder (for hyperphosphatemia in CKD 4-5)",
			Frequency:   "with meals",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Sevelamer carbonate",
						"rxnorm":       "406378",
						"dose":         "800-1600 mg",
						"route":        "oral",
						"frequency":    "with each meal",
						"indication":   "phosphorus > 4.5 mg/dL",
						"benefit":      "non-calcium based, no vascular calcification",
					},
				},
			},
			Status:    "conditional",
			Condition: "CKD_stage >= 4 AND phosphorus > 4.5",
		},
		{
			ActivityID:  "CKD-ACT-007",
			Type:        "medication",
			Description: "Sodium bicarbonate for metabolic acidosis",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Sodium bicarbonate",
						"rxnorm":       "9863",
						"dose":         "650-1950 mg",
						"route":        "oral",
						"frequency":    "2-3 times daily",
						"indication":   "serum bicarbonate < 22 mEq/L",
						"target":       "maintain bicarbonate ≥ 22",
					},
				},
				"benefit": "may slow CKD progression and prevent muscle wasting",
			},
			Status:    "conditional",
			Condition: "bicarbonate < 22 mEq/L",
		},
		// Lifestyle
		{
			ActivityID:  "CKD-ACT-008",
			Type:        "lifestyle",
			Description: "Dietary modifications",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"sodium":     "< 2 g/day (reduces BP and proteinuria)",
				"protein":    "0.8 g/kg/day for CKD 3-5 not on dialysis (avoid very low protein)",
				"potassium":  "individualize based on levels, typically limit if K > 5.0",
				"phosphorus": "limit if elevated (dairy, processed foods, colas)",
				"fluids":     "usually no restriction unless volume overloaded",
				"refer":      "renal dietitian for comprehensive counseling",
			},
			Status: "active",
		},
		{
			ActivityID:  "CKD-ACT-009",
			Type:        "lifestyle",
			Description: "Physical activity",
			Frequency:   "regular",
			Details: map[string]interface{}{
				"recommendation": "30 minutes moderate activity 5 days/week",
				"benefits":       []string{"BP control", "glycemic control", "cardiovascular fitness"},
				"caution":        "avoid extreme exertion, stay hydrated",
			},
			Status: "active",
		},
		{
			ActivityID:  "CKD-ACT-010",
			Type:        "lifestyle",
			Description: "Smoking cessation",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"importance": "smoking accelerates CKD progression and CV risk",
				"support":    "pharmacotherapy and behavioral counseling",
			},
			Status:    "conditional",
			Condition: "current_smoker = true",
		},
		// Education
		{
			ActivityID:  "CKD-ACT-011",
			Type:        "education",
			Description: "CKD self-management education",
			Frequency:   "initial and reinforcement",
			Details: map[string]interface{}{
				"topics": []string{
					"understanding CKD stages and progression",
					"medication importance and adherence",
					"dietary restrictions and food label reading",
					"recognizing complications (edema, fatigue, confusion)",
					"avoiding nephrotoxins (NSAIDs, contrast, certain antibiotics)",
					"preparation for RRT if advanced CKD",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "CKD-ACT-012",
			Type:        "education",
			Description: "Nephrotoxin avoidance",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"avoid": []string{
					"NSAIDs (ibuprofen, naproxen, etc.)",
					"IV contrast if possible (if needed: hydration protocol)",
					"Aminoglycosides (dose adjust or avoid)",
					"Gadolinium (if eGFR < 30)",
					"Herbal supplements with unknown effects",
				},
				"use_with_caution": []string{"metformin at low eGFR", "certain antibiotics"},
			},
			Status: "active",
		},
		// Appointments
		{
			ActivityID:  "CKD-ACT-013",
			Type:        "appointment",
			Description: "Nephrology follow-up",
			Frequency:   "based on CKD stage",
			Details: map[string]interface{}{
				"CKD_3a":   "every 6-12 months",
				"CKD_3b":   "every 3-6 months",
				"CKD_4":    "every 3 months",
				"CKD_5":    "every 1-3 months",
				"rrt_prep": "discuss RRT options when eGFR < 20, refer for access/listing when eGFR < 15",
			},
			Status: "active",
		},
		{
			ActivityID:  "CKD-ACT-014",
			Type:        "appointment",
			Description: "Dietitian consultation",
			Frequency:   "initial + as needed",
			Details: map[string]interface{}{
				"indication": "all CKD patients for medical nutrition therapy",
				"focus":      "sodium, protein, potassium, phosphorus management",
			},
			Status: "active",
		},
		{
			ActivityID:  "CKD-ACT-015",
			Type:        "appointment",
			Description: "Vascular access planning (if approaching dialysis)",
			Frequency:   "when eGFR < 20-25",
			Details: map[string]interface{}{
				"referral":        "vascular surgery for AV fistula evaluation",
				"timing":          "fistula should be placed 6 months before anticipated dialysis",
				"arm_preservation": "avoid blood draws and IVs in non-dominant arm",
			},
			Status:    "conditional",
			Condition: "eGFR < 25 AND progression_expected",
		},
	}

	monitoring := []models.MonitoringItem{
		{ItemID: "CKD-MON-001", Parameter: "eGFR (CKD-EPI)", Frequency: "every 3-6 months (by stage)", Target: "< 5 mL/min/year decline", AlertThreshold: "> 25% decline in 1 year"},
		{ItemID: "CKD-MON-002", Parameter: "UACR", Frequency: "every 3-6 months", Target: "< 30 mg/g or ≥30% reduction", AlertThreshold: "> 300 mg/g or increasing"},
		{ItemID: "CKD-MON-003", Parameter: "Blood pressure", Frequency: "each visit + home monitoring", Target: "< 120/80 mmHg", AlertThreshold: "> 140/90 or symptomatic hypotension"},
		{ItemID: "CKD-MON-004", Parameter: "Potassium", Frequency: "every 1-3 months", Target: "< 5.0 mEq/L", AlertThreshold: "> 5.5 mEq/L"},
		{ItemID: "CKD-MON-005", Parameter: "Bicarbonate", Frequency: "every 3-6 months", Target: "≥ 22 mEq/L", AlertThreshold: "< 20 mEq/L"},
		{ItemID: "CKD-MON-006", Parameter: "Hemoglobin", Frequency: "every 3 months (CKD 3b-5)", Target: "10-11.5 g/dL", AlertThreshold: "< 9 or > 13 g/dL"},
		{ItemID: "CKD-MON-007", Parameter: "Ferritin/TSAT", Frequency: "every 3-6 months if anemic", Target: "Ferritin 100-500, TSAT 20-50%", AlertThreshold: "Ferritin < 100 or TSAT < 20%"},
		{ItemID: "CKD-MON-008", Parameter: "Phosphorus", Frequency: "every 3-6 months (CKD 4-5)", Target: "< 4.5 mg/dL", AlertThreshold: "> 5.5 mg/dL"},
		{ItemID: "CKD-MON-009", Parameter: "Calcium", Frequency: "every 3-6 months", Target: "8.5-10.5 mg/dL", AlertThreshold: "< 8.0 or > 10.5 mg/dL"},
		{ItemID: "CKD-MON-010", Parameter: "PTH", Frequency: "every 6-12 months (CKD 4-5)", Target: "within 2-9x upper normal", AlertThreshold: "> 9x upper normal"},
		{ItemID: "CKD-MON-011", Parameter: "Vitamin D", Frequency: "annually", Target: "> 30 ng/mL", AlertThreshold: "< 20 ng/mL"},
		{ItemID: "CKD-MON-012", Parameter: "HbA1c (if diabetic)", Frequency: "every 3-6 months", Target: "< 7.0% (individualize)", AlertThreshold: "> 9.0%"},
		{ItemID: "CKD-MON-013", Parameter: "Lipid panel", Frequency: "annually", Target: "LDL < 70 mg/dL", AlertThreshold: "LDL > 100 off statin"},
	}

	return &models.CarePlanTemplate{
		TemplateID:   "CP-REN-001",
		Name:         "Chronic Kidney Disease Care Plan",
		Description:  "Comprehensive chronic care plan for CKD management based on KDIGO 2024 Guidelines",
		Version:      "2024.1",
		Category:     "renal",
		Subcategory:  "ckd",
		ConditionRef: &models.ClinicalCondition{Code: "N18.9", System: "ICD-10", Display: "Chronic kidney disease, unspecified"},
		Goals:        goals,
		Activities:   activities,
		Monitoring:   monitoring,
		Duration:     "ongoing",
		ReviewPeriod: "3-6 months based on stage",
		Guidelines: []models.GuidelineReference{
			{GuidelineID: "KDIGO-2024", Name: "KDIGO 2024 CKD Guideline Update", Source: "kdigo.org", URL: "https://kdigo.org/guidelines/ckd-evaluation-and-management/"},
			{GuidelineID: "KDIGO-2022-DM", Name: "KDIGO 2022 Diabetes in CKD Guideline", Source: "kdigo.org", Year: 2022},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "active",
	}
}

// createDepressionCarePlan creates a comprehensive depression management care plan
// Based on APA 2019/2024 Guidelines and VA/DoD CPG
func createDepressionCarePlan() *models.CarePlanTemplate {
	now := time.Now()

	goals := []models.CarePlanGoal{
		{
			GoalID:      "DEP-GOAL-001",
			Description: "Achieve remission of depressive symptoms",
			Category:    "symptom_control",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "DEP-TGT-001", Metric: "PHQ9_score", Value: "< 5 (remission)", Timeframe: "12 weeks"},
				{TargetID: "DEP-TGT-002", Metric: "symptom_response", Value: "≥ 50% reduction in PHQ-9", Timeframe: "6 weeks"},
			},
			Status: "active",
		},
		{
			GoalID:      "DEP-GOAL-002",
			Description: "Restore functional capacity",
			Category:    "functional",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "DEP-TGT-003", Metric: "work_productivity", Value: "return to baseline", Timeframe: "12 weeks"},
				{TargetID: "DEP-TGT-004", Metric: "social_engagement", Value: "≥ 3 activities/week", Timeframe: "8 weeks"},
				{TargetID: "DEP-TGT-005", Metric: "self_care", Value: "independent ADLs", Timeframe: "4 weeks"},
			},
			Status: "active",
		},
		{
			GoalID:      "DEP-GOAL-003",
			Description: "Ensure safety and prevent self-harm",
			Category:    "safety",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "DEP-TGT-006", Metric: "suicidal_ideation", Value: "none", Timeframe: "ongoing"},
				{TargetID: "DEP-TGT-007", Metric: "safety_plan", Value: "completed and accessible", Timeframe: "first visit"},
			},
			Status: "active",
		},
		{
			GoalID:      "DEP-GOAL-004",
			Description: "Prevent relapse",
			Category:    "maintenance",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "DEP-TGT-008", Metric: "relapse_episodes", Value: "0 in maintenance phase", Timeframe: "12 months"},
				{TargetID: "DEP-TGT-009", Metric: "medication_adherence", Value: "≥ 80%", Timeframe: "ongoing"},
			},
			Status: "active",
		},
	}

	activities := []models.Activity{
		// Pharmacotherapy
		{
			ActivityID:  "DEP-ACT-001",
			Type:        "medication",
			Description: "First-line antidepressant therapy (SSRI)",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":       "Sertraline",
						"rxnorm":     "36437",
						"dose":       "50-200 mg",
						"route":      "oral",
						"frequency":  "once daily",
						"start_dose": "50 mg (25 mg in elderly)",
						"titration":  "increase by 50 mg every 2-4 weeks to effect",
						"benefits":   "well-tolerated, good evidence base, safe in cardiac",
					},
					{
						"name":       "Escitalopram",
						"rxnorm":     "321988",
						"dose":       "10-20 mg",
						"route":      "oral",
						"frequency":  "once daily",
						"start_dose": "10 mg (5 mg in elderly)",
						"max_dose":   "20 mg (10 mg if age > 65 due to QT)",
						"benefits":   "most selective SSRI, fewer drug interactions",
					},
				},
				"response_timeline": "2-4 weeks for initial response, 6-8 weeks for full effect",
				"duration":          "minimum 6-12 months after remission, longer if recurrent",
				"evidence":          "Grade A recommendation for moderate-severe MDD",
			},
			Status: "active",
		},
		{
			ActivityID:  "DEP-ACT-002",
			Type:        "medication",
			Description: "Alternative first-line: SNRI",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":       "Duloxetine",
						"rxnorm":     "72625",
						"dose":       "60-120 mg",
						"route":      "oral",
						"frequency":  "once daily",
						"indication": "depression with pain component, fibromyalgia, neuropathy",
						"benefits":   "dual mechanism, helps pain symptoms",
					},
					{
						"name":       "Venlafaxine XR",
						"rxnorm":     "39786",
						"dose":       "75-225 mg",
						"route":      "oral",
						"frequency":  "once daily",
						"monitoring": "blood pressure at higher doses",
					},
				},
			},
			Status:    "conditional",
			Condition: "SSRI_inadequate_response OR pain_comorbidity",
		},
		{
			ActivityID:  "DEP-ACT-003",
			Type:        "medication",
			Description: "Augmentation strategies for partial response",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"options": []map[string]interface{}{
					{
						"name":       "Aripiprazole",
						"rxnorm":     "89013",
						"dose":       "2-15 mg",
						"indication": "atypical antipsychotic augmentation",
						"evidence":   "Grade A for SSRI augmentation",
					},
					{
						"name":       "Bupropion",
						"rxnorm":     "42347",
						"dose":       "150-300 mg",
						"indication": "add to SSRI for partial response or fatigue",
						"benefit":    "no sexual dysfunction, may help with energy",
					},
					{
						"name":       "Lithium",
						"rxnorm":     "42351",
						"dose":       "300-900 mg",
						"monitoring": "lithium levels, renal function, thyroid",
						"indication": "augmentation for treatment-resistant",
					},
				},
				"timing": "consider after 4-6 weeks of adequate dose without full response",
			},
			Status:    "conditional",
			Condition: "partial_response_to_monotherapy",
		},
		// Psychotherapy
		{
			ActivityID:  "DEP-ACT-004",
			Type:        "therapy",
			Description: "Cognitive Behavioral Therapy (CBT)",
			Frequency:   "weekly for 12-16 sessions",
			Details: map[string]interface{}{
				"components": []string{
					"cognitive restructuring",
					"behavioral activation",
					"problem-solving skills",
					"relapse prevention",
				},
				"evidence": "Grade A - as effective as medication for mild-moderate",
				"format":   "individual or group, in-person or telehealth",
				"combined": "combination with medication most effective for moderate-severe",
			},
			Status: "active",
		},
		{
			ActivityID:  "DEP-ACT-005",
			Type:        "therapy",
			Description: "Interpersonal Therapy (IPT)",
			Frequency:   "weekly for 12-16 sessions",
			Details: map[string]interface{}{
				"focus_areas": []string{"grief", "role transitions", "interpersonal disputes", "interpersonal deficits"},
				"indication":  "depression with significant relational component",
				"evidence":    "Grade A for acute depression",
			},
			Status:    "conditional",
			Condition: "interpersonal_issues_prominent OR patient_preference",
		},
		{
			ActivityID:  "DEP-ACT-006",
			Type:        "therapy",
			Description: "Behavioral Activation",
			Frequency:   "weekly sessions + daily practice",
			Details: map[string]interface{}{
				"components": []string{
					"activity monitoring",
					"scheduling pleasant activities",
					"graded task assignment",
					"reducing avoidance behaviors",
				},
				"self_help": "can be guided self-help or therapist-delivered",
				"evidence":  "as effective as CBT for depression",
			},
			Status: "active",
		},
		// Lifestyle
		{
			ActivityID:  "DEP-ACT-007",
			Type:        "lifestyle",
			Description: "Physical activity prescription",
			Frequency:   "most days",
			Details: map[string]interface{}{
				"recommendation": "30-45 minutes moderate aerobic activity 3-5 days/week",
				"evidence":       "equivalent to antidepressant for mild-moderate depression",
				"types":          []string{"walking", "jogging", "swimming", "cycling", "yoga"},
				"start_small":    "begin with 10-15 minutes and gradually increase",
			},
			Status: "active",
		},
		{
			ActivityID:  "DEP-ACT-008",
			Type:        "lifestyle",
			Description: "Sleep hygiene optimization",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"recommendations": []string{
					"consistent sleep/wake times",
					"limit screen time before bed",
					"avoid caffeine after noon",
					"create cool, dark sleep environment",
					"limit alcohol (disrupts sleep architecture)",
				},
				"treat_insomnia": "CBT-I if persistent insomnia",
			},
			Status: "active",
		},
		{
			ActivityID:  "DEP-ACT-009",
			Type:        "lifestyle",
			Description: "Social connection and support",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"activities": []string{
					"maintain regular social contact",
					"identify supportive relationships",
					"consider support groups",
					"limit isolation behaviors",
				},
			},
			Status: "active",
		},
		// Safety
		{
			ActivityID:  "DEP-ACT-010",
			Type:        "safety",
			Description: "Safety planning (if any suicidal ideation)",
			Frequency:   "initial and update as needed",
			Details: map[string]interface{}{
				"components": []string{
					"warning signs identification",
					"internal coping strategies",
					"social contacts for distraction",
					"family/friends who can help",
					"professional resources and crisis lines",
					"means restriction (lethal means counseling)",
				},
				"crisis_resources": []string{
					"988 Suicide & Crisis Lifeline",
					"Crisis Text Line: text HOME to 741741",
					"Emergency department if imminent risk",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "DEP-ACT-011",
			Type:        "safety",
			Description: "Lethal means counseling",
			Frequency:   "at risk assessment",
			Details: map[string]interface{}{
				"actions": []string{
					"discuss access to firearms, medications, other means",
					"temporary removal or securing of firearms",
					"medication quantity limitations",
					"involve family in safety planning",
				},
			},
			Status:    "conditional",
			Condition: "suicidal_ideation_present OR history_of_attempt",
		},
		// Education
		{
			ActivityID:  "DEP-ACT-012",
			Type:        "education",
			Description: "Depression psychoeducation",
			Frequency:   "initial and reinforcement",
			Details: map[string]interface{}{
				"topics": []string{
					"depression as a medical illness",
					"expected treatment timeline",
					"importance of medication adherence",
					"recognizing early warning signs",
					"when to seek help urgently",
				},
				"resources": "provide written materials and reliable websites",
			},
			Status: "active",
		},
		// Appointments
		{
			ActivityID:  "DEP-ACT-013",
			Type:        "appointment",
			Description: "Psychiatric/prescriber follow-up",
			Frequency:   "varies by phase",
			Details: map[string]interface{}{
				"acute_phase":       "every 1-2 weeks until stable",
				"continuation":      "every 2-4 weeks",
				"maintenance":       "every 1-3 months",
				"assess_at_visits": []string{
					"symptom severity (PHQ-9)",
					"suicidality",
					"medication side effects",
					"functioning",
					"adherence",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "DEP-ACT-014",
			Type:        "appointment",
			Description: "Therapy sessions",
			Frequency:   "weekly during acute phase",
			Details: map[string]interface{}{
				"duration":    "typically 12-16 sessions for acute episode",
				"maintenance": "may continue less frequently for relapse prevention",
			},
			Status: "active",
		},
	}

	monitoring := []models.MonitoringItem{
		{ItemID: "DEP-MON-001", Parameter: "PHQ-9 score", Frequency: "each visit", Target: "< 5 (remission)", AlertThreshold: "≥ 15 (moderately severe) or increase ≥ 5 points"},
		{ItemID: "DEP-MON-002", Parameter: "Suicidal ideation (PHQ-9 Q9)", Frequency: "each visit", Target: "0", AlertThreshold: "any positive response"},
		{ItemID: "DEP-MON-003", Parameter: "Columbia Suicide Severity Rating", Frequency: "if SI present", Target: "no active ideation", AlertThreshold: "active SI with plan or intent"},
		{ItemID: "DEP-MON-004", Parameter: "GAD-7 (comorbid anxiety)", Frequency: "baseline and periodic", Target: "< 5", AlertThreshold: "≥ 10"},
		{ItemID: "DEP-MON-005", Parameter: "Medication adherence", Frequency: "each visit", Target: "≥ 80%", AlertThreshold: "< 50%"},
		{ItemID: "DEP-MON-006", Parameter: "Side effects", Frequency: "each visit", Target: "tolerable", AlertThreshold: "intolerable or dangerous (serotonin syndrome, hyponatremia)"},
		{ItemID: "DEP-MON-007", Parameter: "Functional status", Frequency: "each visit", Target: "improving/stable", AlertThreshold: "declining work/social function"},
		{ItemID: "DEP-MON-008", Parameter: "Sleep quality", Frequency: "each visit", Target: "adequate and restorative", AlertThreshold: "persistent insomnia or hypersomnia"},
		{ItemID: "DEP-MON-009", Parameter: "Substance use", Frequency: "periodically", Target: "no harmful use", AlertThreshold: "new or increased use"},
		{ItemID: "DEP-MON-010", Parameter: "Weight", Frequency: "monthly initially", Target: "stable", AlertThreshold: "> 5% change"},
	}

	return &models.CarePlanTemplate{
		TemplateID:   "CP-MH-001",
		Name:         "Major Depression Care Plan",
		Description:  "Comprehensive chronic care plan for Major Depressive Disorder based on APA and VA/DoD Guidelines",
		Version:      "2024.1",
		Category:     "mental_health",
		Subcategory:  "depression",
		ConditionRef: &models.ClinicalCondition{Code: "F32.9", System: "ICD-10", Display: "Major depressive disorder, single episode, unspecified"},
		Goals:        goals,
		Activities:   activities,
		Monitoring:   monitoring,
		Duration:     "ongoing with phases (acute, continuation, maintenance)",
		ReviewPeriod: "2-4 weeks acute, 1-3 months maintenance",
		Guidelines: []models.GuidelineReference{
			{GuidelineID: "APA-2019", Name: "APA Practice Guideline for Treatment of Depression", Source: "psychiatry.org", Year: 2019},
			{GuidelineID: "VA-DOD-2022", Name: "VA/DoD Clinical Practice Guideline for MDD", Source: "healthquality.va.gov", Year: 2022},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "active",
	}
}

// createOsteoporosisCarePlan creates a comprehensive osteoporosis management care plan
// Based on AACE/ACE 2020 and Endocrine Society Guidelines
func createOsteoporosisCarePlan() *models.CarePlanTemplate {
	now := time.Now()

	goals := []models.CarePlanGoal{
		{
			GoalID:      "OST-GOAL-001",
			Description: "Prevent fragility fractures",
			Category:    "fracture_prevention",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "OST-TGT-001", Metric: "new_fractures", Value: "0", Timeframe: "ongoing"},
				{TargetID: "OST-TGT-002", Metric: "FRAX_major", Value: "< 20% or decreasing", Timeframe: "3 years"},
			},
			Status: "active",
		},
		{
			GoalID:      "ASTH-GOAL-002",
			Description: "Improve or stabilize bone mineral density",
			Category:    "bone_health",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "OST-TGT-003", Metric: "BMD_T_score", Value: "improvement ≥ 3-5% or stable", Timeframe: "2 years"},
				{TargetID: "OST-TGT-004", Metric: "spine_BMD", Value: "T-score > -2.5 or improving", Timeframe: "3-5 years"},
			},
			Status: "active",
		},
		{
			GoalID:      "OST-GOAL-003",
			Description: "Reduce fall risk",
			Category:    "fall_prevention",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "OST-TGT-005", Metric: "falls_per_year", Value: "0", Timeframe: "12 months"},
				{TargetID: "OST-TGT-006", Metric: "balance_assessment", Value: "Timed Up and Go < 12 sec", Timeframe: "6 months"},
			},
			Status: "active",
		},
		{
			GoalID:      "OST-GOAL-004",
			Description: "Optimize calcium and vitamin D status",
			Category:    "nutrition",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "OST-TGT-007", Metric: "vitamin_D_level", Value: "30-50 ng/mL", Timeframe: "3 months"},
				{TargetID: "OST-TGT-008", Metric: "calcium_intake", Value: "1000-1200 mg/day", Timeframe: "ongoing"},
			},
			Status: "active",
		},
	}

	activities := []models.Activity{
		// Pharmacotherapy - Antiresorptive
		{
			ActivityID:  "OST-ACT-001",
			Type:        "medication",
			Description: "First-line bisphosphonate therapy",
			Frequency:   "weekly or monthly",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Alendronate",
						"rxnorm":       "1094",
						"dose":         "70 mg",
						"route":        "oral",
						"frequency":    "once weekly",
						"instructions": []string{
							"take first thing in AM with 8 oz water",
							"remain upright for 30 minutes",
							"no food/drink/other meds for 30 min",
						},
						"duration":      "5 years typical, then reassess (drug holiday possible)",
						"evidence":      "reduces hip and vertebral fractures by 50%",
					},
					{
						"name":         "Risedronate",
						"rxnorm":       "35894",
						"dose":         "35 mg weekly or 150 mg monthly",
						"route":        "oral",
						"frequency":    "weekly or monthly",
						"alternative":  "for those intolerant of alendronate",
					},
				},
				"contraindications": []string{
					"esophageal abnormalities",
					"inability to remain upright",
					"GFR < 30-35 mL/min",
					"hypocalcemia",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "OST-ACT-002",
			Type:        "medication",
			Description: "IV bisphosphonate (if oral not tolerated)",
			Frequency:   "annually",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":          "Zoledronic acid",
						"rxnorm":        "77651",
						"dose":          "5 mg IV",
						"frequency":     "once yearly",
						"administration": "15-minute infusion, ensure hydrated",
						"indication":    "oral bisphosphonate intolerance, poor GI absorption",
						"benefit":       "highest BMD gains of bisphosphonates",
					},
				},
				"monitoring":     "renal function before each dose, vitamin D replete",
				"side_effects":   "acute phase reaction (flu-like) common with first dose",
			},
			Status:    "conditional",
			Condition: "oral_bisphosphonate_intolerant OR poor_adherence",
		},
		{
			ActivityID:  "OST-ACT-003",
			Type:        "medication",
			Description: "Denosumab (RANK-L inhibitor)",
			Frequency:   "every 6 months",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":       "Denosumab",
						"rxnorm":     "997223",
						"dose":       "60 mg SC",
						"frequency":  "every 6 months",
						"indication": []string{
							"bisphosphonate intolerant/contraindicated",
							"renal impairment (no GFR restriction)",
							"very high fracture risk",
						},
						"critical":   "CANNOT be stopped abruptly - rebound bone loss",
						"transition": "must transition to bisphosphonate after stopping",
					},
				},
				"monitoring": "calcium levels, vitamin D, dental health",
			},
			Status:    "conditional",
			Condition: "bisphosphonate_contraindicated OR very_high_risk",
		},
		{
			ActivityID:  "OST-ACT-004",
			Type:        "medication",
			Description: "Anabolic therapy for very high risk or treatment failure",
			Frequency:   "daily for 12-24 months",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":       "Romosozumab",
						"rxnorm":     "2105816",
						"dose":       "210 mg SC",
						"frequency":  "monthly for 12 months",
						"indication": "very high fracture risk, building phase",
						"warning":    "black box for CV events - avoid if recent MI/stroke",
						"benefit":    "builds new bone rapidly, reduces fractures quickly",
					},
					{
						"name":       "Teriparatide",
						"rxnorm":     "187832",
						"dose":       "20 mcg SC",
						"frequency":  "daily for up to 24 months",
						"indication": "severe osteoporosis, glucocorticoid-induced",
						"lifetime":   "2-year maximum lifetime use",
					},
					{
						"name":       "Abaloparatide",
						"rxnorm":     "1858928",
						"dose":       "80 mcg SC",
						"frequency":  "daily for up to 24 months",
						"benefit":    "similar to teriparatide with faster effect",
					},
				},
				"sequence": "ALWAYS follow anabolic with antiresorptive to maintain gains",
			},
			Status:    "conditional",
			Condition: "very_high_risk OR multiple_fractures OR treatment_failure",
		},
		// Supplements
		{
			ActivityID:  "OST-ACT-005",
			Type:        "medication",
			Description: "Calcium supplementation (if dietary intake insufficient)",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":       "Calcium carbonate",
						"rxnorm":     "1980",
						"dose":       "500-600 mg elemental calcium",
						"route":      "oral",
						"frequency":  "with meals, divided doses",
						"maximum":    "1000-1200 mg total daily (diet + supplement)",
						"note":       "carbonate requires acid, take with food",
					},
					{
						"name":       "Calcium citrate",
						"rxnorm":     "11280",
						"dose":       "500-600 mg elemental calcium",
						"frequency":  "can be taken without food",
						"indication": "achlorhydria, PPI use, GI malabsorption",
					},
				},
				"preference": "dietary calcium preferred over supplements",
				"caution":    "excess calcium may increase CV risk - avoid > 1200 mg/day",
			},
			Status:    "conditional",
			Condition: "dietary_calcium < 1000 mg/day",
		},
		{
			ActivityID:  "OST-ACT-006",
			Type:        "medication",
			Description: "Vitamin D supplementation",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":       "Cholecalciferol (D3)",
						"rxnorm":     "11253",
						"dose":       "800-2000 IU daily (higher if deficient)",
						"route":      "oral",
						"target":     "25-OH vitamin D 30-50 ng/mL",
						"loading":    "50,000 IU weekly x 8-12 weeks if < 20 ng/mL",
					},
				},
				"monitoring": "check level at 3 months, then annually",
			},
			Status: "active",
		},
		// Lifestyle
		{
			ActivityID:  "OST-ACT-007",
			Type:        "lifestyle",
			Description: "Weight-bearing and resistance exercise",
			Frequency:   "regular",
			Details: map[string]interface{}{
				"recommendations": []string{
					"weight-bearing exercise 30 min most days (walking, jogging, stairs)",
					"resistance training 2-3x/week (builds muscle, stimulates bone)",
					"balance exercises daily (reduces falls)",
					"avoid high-impact if severe osteoporosis or prior fracture",
				},
				"benefit": "improves bone density, muscle strength, and balance",
			},
			Status: "active",
		},
		{
			ActivityID:  "OST-ACT-008",
			Type:        "lifestyle",
			Description: "Fall prevention strategies",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"home_modifications": []string{
					"remove throw rugs and clutter",
					"install grab bars in bathroom",
					"ensure adequate lighting",
					"use non-slip mats",
					"handrails on stairs",
				},
				"personal_measures": []string{
					"appropriate footwear",
					"assistive devices if needed",
					"vision correction",
					"review medications causing dizziness",
				},
				"assessment": "annual fall risk assessment, PT referral if high risk",
			},
			Status: "active",
		},
		{
			ActivityID:  "OST-ACT-009",
			Type:        "lifestyle",
			Description: "Lifestyle modifications",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"smoking": "cessation critical - smoking accelerates bone loss",
				"alcohol": "limit to ≤ 2 drinks/day",
				"caffeine": "moderate intake (> 4 cups coffee may affect calcium)",
			},
			Status: "active",
		},
		{
			ActivityID:  "OST-ACT-010",
			Type:        "lifestyle",
			Description: "Dietary calcium optimization",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"target":  "1000-1200 mg/day from diet",
				"sources": []string{
					"dairy (milk, yogurt, cheese)",
					"calcium-fortified foods",
					"leafy greens (kale, broccoli)",
					"canned fish with bones",
					"tofu",
				},
				"dietitian": "refer if dietary assessment needed",
			},
			Status: "active",
		},
		// Education
		{
			ActivityID:  "OST-ACT-011",
			Type:        "education",
			Description: "Osteoporosis education and self-management",
			Frequency:   "initial and reinforcement",
			Details: map[string]interface{}{
				"topics": []string{
					"understanding bone health and osteoporosis",
					"medication importance and proper administration",
					"fall prevention strategies",
					"safe movement and posture",
					"fracture signs and when to seek care",
					"spine protection (avoid forward bending with load)",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "OST-ACT-012",
			Type:        "education",
			Description: "Medication-specific counseling",
			Frequency:   "with each prescription",
			Details: map[string]interface{}{
				"bisphosphonate": "proper timing, remain upright, report GI symptoms",
				"denosumab":      "never miss a dose, transition plan if stopping",
				"anabolic":       "injection technique, storage, refrigeration",
				"rare_risks":     "atypical femur fracture and ONJ - symptoms to report",
			},
			Status: "active",
		},
		// Appointments
		{
			ActivityID:  "OST-ACT-013",
			Type:        "appointment",
			Description: "Bone health monitoring visits",
			Frequency:   "every 1-2 years",
			Details: map[string]interface{}{
				"components": []string{
					"medication adherence and tolerance",
					"fall assessment",
					"new fracture symptoms",
					"height measurement (≥1 inch loss suggests vertebral fracture)",
					"review of risk factors",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "OST-ACT-014",
			Type:        "appointment",
			Description: "Dental examination",
			Frequency:   "every 6 months",
			Details: map[string]interface{}{
				"importance": "ONJ prevention - complete dental work before starting bisphosphonate/denosumab",
				"ongoing":    "regular dental care, avoid invasive procedures if possible on therapy",
			},
			Status: "active",
		},
		{
			ActivityID:  "OST-ACT-015",
			Type:        "appointment",
			Description: "Physical therapy referral",
			Frequency:   "as needed",
			Details: map[string]interface{}{
				"indications": []string{
					"fall history",
					"balance impairment",
					"post-fracture rehabilitation",
					"exercise prescription",
				},
			},
			Status:    "conditional",
			Condition: "fall_history OR balance_impairment OR post_fracture",
		},
	}

	monitoring := []models.MonitoringItem{
		{ItemID: "OST-MON-001", Parameter: "DXA bone density", Frequency: "every 2 years (or per treatment response)", Target: "stable or improving T-score", AlertThreshold: "T-score decline > 5%"},
		{ItemID: "OST-MON-002", Parameter: "Height measurement", Frequency: "annually", Target: "stable", AlertThreshold: "≥ 2 cm (0.8 in) loss"},
		{ItemID: "OST-MON-003", Parameter: "New fracture assessment", Frequency: "each visit", Target: "none", AlertThreshold: "any new fragility fracture"},
		{ItemID: "OST-MON-004", Parameter: "25-OH Vitamin D", Frequency: "baseline, 3 months, then annually", Target: "30-50 ng/mL", AlertThreshold: "< 20 ng/mL"},
		{ItemID: "OST-MON-005", Parameter: "Serum calcium", Frequency: "annually", Target: "8.5-10.5 mg/dL", AlertThreshold: "< 8.5 or > 10.5 mg/dL"},
		{ItemID: "OST-MON-006", Parameter: "Renal function (if on bisphosphonate)", Frequency: "annually", Target: "eGFR > 35", AlertThreshold: "eGFR < 30"},
		{ItemID: "OST-MON-007", Parameter: "Fall history", Frequency: "each visit", Target: "0 falls", AlertThreshold: "any fall"},
		{ItemID: "OST-MON-008", Parameter: "FRAX score", Frequency: "baseline, every 2-3 years", Target: "decreasing or stable", AlertThreshold: "10-year major fracture risk > 20%"},
		{ItemID: "OST-MON-009", Parameter: "Medication adherence", Frequency: "each visit", Target: "≥ 80%", AlertThreshold: "< 50%"},
		{ItemID: "OST-MON-010", Parameter: "Bone turnover markers (CTX, P1NP)", Frequency: "optional - 3-6 months after starting therapy", Target: "decrease from baseline", AlertThreshold: "no suppression on antiresorptive"},
		{ItemID: "OST-MON-011", Parameter: "Thigh/groin pain (AFF screening)", Frequency: "each visit if on long-term bisphosphonate", Target: "none", AlertThreshold: "new thigh pain"},
		{ItemID: "OST-MON-012", Parameter: "Dental health", Frequency: "every 6 months", Target: "good oral health", AlertThreshold: "planned invasive dental procedure"},
	}

	return &models.CarePlanTemplate{
		TemplateID:   "CP-MSK-001",
		Name:         "Osteoporosis Care Plan",
		Description:  "Comprehensive chronic care plan for osteoporosis management based on AACE/ACE and Endocrine Society Guidelines",
		Version:      "2024.1",
		Category:     "musculoskeletal",
		Subcategory:  "osteoporosis",
		ConditionRef: &models.ClinicalCondition{Code: "M81.0", System: "ICD-10", Display: "Age-related osteoporosis without current pathological fracture"},
		Goals:        goals,
		Activities:   activities,
		Monitoring:   monitoring,
		Duration:     "ongoing with treatment cycles",
		ReviewPeriod: "6-12 months",
		Guidelines: []models.GuidelineReference{
			{GuidelineID: "AACE-2020", Name: "AACE/ACE Clinical Practice Guidelines for Osteoporosis", Source: "aace.com", Year: 2020},
			{GuidelineID: "ENDO-2020", Name: "Endocrine Society Osteoporosis in Men Guidelines", Source: "endocrine.org", Year: 2020},
			{GuidelineID: "NOF-2021", Name: "National Osteoporosis Foundation Clinician's Guide", Source: "nof.org", Year: 2021},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "active",
	}
}
