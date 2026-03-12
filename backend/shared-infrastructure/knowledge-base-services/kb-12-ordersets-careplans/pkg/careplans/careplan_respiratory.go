// Package careplans provides chronic disease care plan templates
// Respiratory care plans: COPD, Asthma
package careplans

import (
	"time"

	"kb-12-ordersets-careplans/internal/models"
)

// GetRespiratoryCarePlans returns all respiratory chronic care plans
func GetRespiratoryCarePlans() []*models.CarePlanTemplate {
	return []*models.CarePlanTemplate{
		createCOPDCarePlan(),
		createAsthmaCarePlan(),
	}
}

// createCOPDCarePlan creates a comprehensive COPD management care plan
// Based on GOLD 2024 Guidelines (Global Initiative for Chronic Obstructive Lung Disease)
func createCOPDCarePlan() *models.CarePlanTemplate {
	now := time.Now()

	goals := []models.CarePlanGoal{
		{
			GoalID:      "COPD-GOAL-001",
			Description: "Reduce exacerbation frequency",
			Category:    "disease_control",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "COPD-TGT-001", Metric: "exacerbations_per_year", Value: "< 2 moderate/severe", Timeframe: "12 months"},
				{TargetID: "COPD-TGT-002", Metric: "hospitalizations", Value: "0 COPD admissions", Timeframe: "12 months"},
			},
			Status: "active",
		},
		{
			GoalID:      "COPD-GOAL-002",
			Description: "Improve symptoms and functional status",
			Category:    "symptom_control",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "COPD-TGT-003", Metric: "CAT_score", Value: "< 10 or ≥2 point improvement", Timeframe: "6 months"},
				{TargetID: "COPD-TGT-004", Metric: "mMRC_dyspnea", Value: "≤ 1 or improvement by 1 grade", Timeframe: "6 months"},
				{TargetID: "COPD-TGT-005", Metric: "6MWD", Value: "≥ 30m improvement", Timeframe: "12 months"},
			},
			Status: "active",
		},
		{
			GoalID:      "COPD-GOAL-003",
			Description: "Smoking cessation (if applicable)",
			Category:    "lifestyle",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "COPD-TGT-006", Metric: "smoking_status", Value: "quit", Timeframe: "6 months"},
				{TargetID: "COPD-TGT-007", Metric: "exhaled_CO", Value: "< 10 ppm", Timeframe: "3 months"},
			},
			Status: "active",
		},
		{
			GoalID:      "COPD-GOAL-004",
			Description: "Optimize lung function preservation",
			Category:    "physiologic",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "COPD-TGT-008", Metric: "FEV1_decline", Value: "< 40 mL/year", Timeframe: "annual"},
				{TargetID: "COPD-TGT-009", Metric: "oxygen_saturation", Value: "≥ 90% on room air or stable O2 requirement", Timeframe: "ongoing"},
			},
			Status: "active",
		},
		{
			GoalID:      "COPD-GOAL-005",
			Description: "Maintain physical activity",
			Category:    "functional",
			Priority:    "low",
			Targets: []models.GoalTarget{
				{TargetID: "COPD-TGT-010", Metric: "daily_steps", Value: "≥ 5000 steps/day", Timeframe: "6 months"},
				{TargetID: "COPD-TGT-011", Metric: "pulmonary_rehab", Value: "complete 8-week program", Timeframe: "3 months"},
			},
			Status: "active",
		},
	}

	activities := []models.Activity{
		// Maintenance Medications - GOLD Group B/E
		{
			ActivityID:  "COPD-ACT-001",
			Type:        "medication",
			Description: "Long-acting bronchodilator therapy (LABA + LAMA)",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Umeclidinium-Vilanterol",
						"rxnorm":       "1487515",
						"dose":         "62.5/25 mcg",
						"route":        "inhalation",
						"frequency":    "once daily",
						"indication":   "maintenance COPD therapy - LAMA/LABA combination",
						"alternatives": []string{"Tiotropium-Olodaterol", "Glycopyrrolate-Formoterol"},
					},
				},
				"gold_recommendation": "First-line for Group B (high symptoms) and Group E (exacerbations)",
				"evidence_level":      "Grade A",
			},
			Status: "active",
		},
		{
			ActivityID:  "COPD-ACT-002",
			Type:        "medication",
			Description: "Inhaled corticosteroid (if eosinophils ≥300 or frequent exacerbations)",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Fluticasone Furoate-Umeclidinium-Vilanterol",
						"rxnorm":       "1945035",
						"dose":         "100/62.5/25 mcg",
						"route":        "inhalation",
						"frequency":    "once daily",
						"indication":   "COPD with eosinophils ≥300 or ≥2 exacerbations/year",
						"ics_criteria": "blood eosinophils ≥300 cells/μL or ≥100 with ≥2 exacerbations",
					},
				},
				"monitoring":     "pneumonia risk, oral candidiasis",
				"evidence_level": "Grade A",
			},
			Status:    "conditional",
			Condition: "blood_eosinophils >= 300 OR exacerbations_last_year >= 2",
		},
		{
			ActivityID:  "COPD-ACT-003",
			Type:        "medication",
			Description: "Rescue bronchodilator",
			Frequency:   "as needed",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":           "Albuterol MDI",
						"rxnorm":         "197318",
						"dose":           "90 mcg/puff, 1-2 puffs",
						"route":          "inhalation",
						"frequency":      "every 4-6 hours as needed",
						"max_per_day":    "12 puffs",
						"concern_trigger": "> 2 canisters/month suggests poor control",
					},
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "COPD-ACT-004",
			Type:        "medication",
			Description: "Macrolide prophylaxis (for frequent exacerbators)",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Azithromycin",
						"rxnorm":       "18631",
						"dose":         "250 mg",
						"route":        "oral",
						"frequency":    "daily or 3 times weekly",
						"duration":     "1 year, then reassess",
						"indication":   "former smoker with ≥1 exacerbation despite optimal therapy",
						"exclusions":   []string{"hearing impairment", "resting tachycardia", "QTc prolongation"},
					},
				},
				"monitoring":     "hearing assessment, ECG for QTc, LFTs",
				"evidence_level": "Grade A for exacerbation reduction",
			},
			Status:    "conditional",
			Condition: "exacerbations >= 1/year AND former_smoker AND optimal_inhaler_therapy",
		},
		{
			ActivityID:  "COPD-ACT-005",
			Type:        "medication",
			Description: "Smoking cessation pharmacotherapy",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"options": []map[string]interface{}{
					{
						"name":       "Varenicline",
						"rxnorm":     "637190",
						"dose":       "1 mg twice daily",
						"duration":   "12 weeks, may extend to 24 weeks",
						"start_date": "1-2 weeks before quit date",
					},
					{
						"name":     "Nicotine patch",
						"rxnorm":   "198029",
						"dose":     "21 mg/day → 14 mg/day → 7 mg/day",
						"duration": "8-12 weeks taper",
					},
					{
						"name":     "Bupropion SR",
						"rxnorm":   "42347",
						"dose":     "150 mg twice daily",
						"duration": "7-12 weeks",
					},
				},
				"combination_therapy": "patch + short-acting NRT or varenicline most effective",
			},
			Status:    "conditional",
			Condition: "current_smoker = true",
		},
		// Lifestyle Interventions
		{
			ActivityID:  "COPD-ACT-006",
			Type:        "lifestyle",
			Description: "Pulmonary rehabilitation program",
			Frequency:   "2-3 sessions per week",
			Details: map[string]interface{}{
				"components": []string{
					"supervised exercise training",
					"breathing techniques",
					"energy conservation",
					"nutritional counseling",
					"psychosocial support",
				},
				"duration":     "6-12 weeks minimum",
				"setting":      "outpatient or home-based",
				"benefits":     "improves dyspnea, exercise capacity, quality of life, reduces hospitalizations",
				"referral":     "all symptomatic patients (CAT ≥10 or mMRC ≥2)",
				"evidence":     "Grade A - strongest evidence for symptom improvement",
				"maintenance":  "ongoing exercise program post-rehab",
			},
			Status: "active",
		},
		{
			ActivityID:  "COPD-ACT-007",
			Type:        "lifestyle",
			Description: "Physical activity prescription",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"recommendation": "30 minutes moderate activity most days",
				"types":          []string{"walking", "cycling", "swimming", "tai chi"},
				"monitoring":     "pulse oximetry during exertion if indicated",
				"adaptations":    "use bronchodilator 15-30 min before exercise",
			},
			Status: "active",
		},
		{
			ActivityID:  "COPD-ACT-008",
			Type:        "lifestyle",
			Description: "Nutrition optimization",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"goals": []string{
					"maintain healthy BMI (avoid both underweight and obesity)",
					"adequate protein intake for muscle mass",
					"small frequent meals if dyspnea with eating",
				},
				"supplements": "vitamin D if deficient (common in COPD)",
				"cachexia":    "high-calorie supplements, consider appetite stimulants",
			},
			Status: "active",
		},
		// Education
		{
			ActivityID:  "COPD-ACT-009",
			Type:        "education",
			Description: "Inhaler technique training",
			Frequency:   "each visit",
			Details: map[string]interface{}{
				"components": []string{
					"demonstrate proper technique",
					"check technique at every visit",
					"spacer use with MDIs",
					"device selection matching patient capability",
				},
				"critical": "incorrect technique is most common cause of poor control",
			},
			Status: "active",
		},
		{
			ActivityID:  "COPD-ACT-010",
			Type:        "education",
			Description: "Action plan for exacerbations",
			Frequency:   "annual review",
			Details: map[string]interface{}{
				"components": []string{
					"recognize worsening symptoms (increased dyspnea, sputum, purulence)",
					"when to increase rescue inhaler",
					"when to start prednisone/antibiotics (prescribe standby)",
					"when to seek emergency care",
				},
				"standby_medications": []string{
					"Prednisone 40mg daily x 5 days",
					"Antibiotic (amoxicillin-clavulanate or azithromycin)",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "COPD-ACT-011",
			Type:        "education",
			Description: "Oxygen therapy education (if applicable)",
			Frequency:   "as needed",
			Details: map[string]interface{}{
				"topics": []string{
					"proper oxygen flow rate",
					"hours of use (≥15 hours/day for survival benefit)",
					"equipment care and safety",
					"travel with oxygen",
				},
			},
			Status:    "conditional",
			Condition: "supplemental_oxygen_prescribed = true",
		},
		// Appointments
		{
			ActivityID:  "COPD-ACT-012",
			Type:        "appointment",
			Description: "Pulmonologist follow-up",
			Frequency:   "every 3-6 months",
			Details: map[string]interface{}{
				"stable":       "every 6 months",
				"after_exacerbation": "within 2-4 weeks",
				"review_items": []string{
					"symptom assessment (CAT, mMRC)",
					"exacerbation history",
					"inhaler technique check",
					"medication adjustment",
					"vaccination status",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "COPD-ACT-013",
			Type:        "appointment",
			Description: "Primary care coordination",
			Frequency:   "annual",
			Details: map[string]interface{}{
				"focus": "comorbidity management, preventive care, comprehensive care coordination",
			},
			Status: "active",
		},
	}

	monitoring := []models.MonitoringItem{
		{ItemID: "COPD-MON-001", Parameter: "CAT score", Frequency: "each visit", Target: "< 10", AlertThreshold: "> 20 or increase ≥ 2"},
		{ItemID: "COPD-MON-002", Parameter: "mMRC dyspnea scale", Frequency: "each visit", Target: "≤ 1", AlertThreshold: "≥ 3"},
		{ItemID: "COPD-MON-003", Parameter: "Exacerbation count", Frequency: "ongoing", Target: "< 2/year", AlertThreshold: "any moderate/severe"},
		{ItemID: "COPD-MON-004", Parameter: "Spirometry (FEV1)", Frequency: "annual", Target: "decline < 40 mL/year", AlertThreshold: "> 10% decline"},
		{ItemID: "COPD-MON-005", Parameter: "Pulse oximetry", Frequency: "each visit", Target: "≥ 92%", AlertThreshold: "< 88%"},
		{ItemID: "COPD-MON-006", Parameter: "Blood eosinophils", Frequency: "annual", Target: "guide ICS use", AlertThreshold: "≥ 300 cells/μL"},
		{ItemID: "COPD-MON-007", Parameter: "6-minute walk distance", Frequency: "annual", Target: "stable or improving", AlertThreshold: "> 30m decline"},
		{ItemID: "COPD-MON-008", Parameter: "Body weight/BMI", Frequency: "each visit", Target: "stable", AlertThreshold: "> 5% unintentional weight loss"},
		{ItemID: "COPD-MON-009", Parameter: "Rescue inhaler use", Frequency: "each visit", Target: "< 1 canister/month", AlertThreshold: "> 2 canisters/month"},
		{ItemID: "COPD-MON-010", Parameter: "Influenza vaccine", Frequency: "annual", Target: "administered", AlertThreshold: "overdue"},
		{ItemID: "COPD-MON-011", Parameter: "Pneumococcal vaccine", Frequency: "per guidelines", Target: "up to date", AlertThreshold: "overdue"},
		{ItemID: "COPD-MON-012", Parameter: "COVID-19 vaccine", Frequency: "per guidelines", Target: "up to date", AlertThreshold: "overdue"},
	}

	return &models.CarePlanTemplate{
		TemplateID:   "CP-RESP-001",
		Name:         "COPD Chronic Care Plan",
		Description:  "Comprehensive chronic care plan for COPD management based on GOLD 2024 Guidelines",
		Version:      "2024.1",
		Category:     "respiratory",
		Subcategory:  "copd",
		ConditionRef: &models.ClinicalCondition{Code: "J44.9", System: "ICD-10", Display: "Chronic obstructive pulmonary disease, unspecified"},
		Goals:        goals,
		Activities:   activities,
		Monitoring:   monitoring,
		Duration:     "ongoing",
		ReviewPeriod: "6 months",
		Guidelines: []models.GuidelineReference{
			{GuidelineID: "GOLD-2024", Name: "Global Initiative for Chronic Obstructive Lung Disease 2024", Source: "goldcopd.org", URL: "https://goldcopd.org/2024-gold-report/"},
			{GuidelineID: "ATS-ERS-2023", Name: "ATS/ERS COPD Pharmacotherapy Guidelines", Source: "AJRCCM", Year: 2023},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "active",
	}
}

// createAsthmaCarePlan creates a comprehensive asthma management care plan
// Based on GINA 2024 Guidelines (Global Initiative for Asthma)
func createAsthmaCarePlan() *models.CarePlanTemplate {
	now := time.Now()

	goals := []models.CarePlanGoal{
		{
			GoalID:      "ASTH-GOAL-001",
			Description: "Achieve and maintain asthma control",
			Category:    "disease_control",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "ASTH-TGT-001", Metric: "ACT_score", Value: "≥ 20 (well-controlled)", Timeframe: "3 months"},
				{TargetID: "ASTH-TGT-002", Metric: "daytime_symptoms", Value: "≤ 2 days/week", Timeframe: "4 weeks"},
				{TargetID: "ASTH-TGT-003", Metric: "nighttime_awakening", Value: "none", Timeframe: "4 weeks"},
				{TargetID: "ASTH-TGT-004", Metric: "activity_limitation", Value: "none", Timeframe: "4 weeks"},
			},
			Status: "active",
		},
		{
			GoalID:      "ASTH-GOAL-002",
			Description: "Minimize exacerbations and oral corticosteroid use",
			Category:    "exacerbation_prevention",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{TargetID: "ASTH-TGT-005", Metric: "exacerbations_per_year", Value: "0-1", Timeframe: "12 months"},
				{TargetID: "ASTH-TGT-006", Metric: "oral_steroid_bursts", Value: "0", Timeframe: "12 months"},
				{TargetID: "ASTH-TGT-007", Metric: "ED_visits_hospitalizations", Value: "0", Timeframe: "12 months"},
			},
			Status: "active",
		},
		{
			GoalID:      "ASTH-GOAL-003",
			Description: "Optimize lung function",
			Category:    "physiologic",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{TargetID: "ASTH-TGT-008", Metric: "FEV1_percent_predicted", Value: "≥ 80% (or personal best)", Timeframe: "6 months"},
				{TargetID: "ASTH-TGT-009", Metric: "FEV1_FVC_ratio", Value: "≥ 0.75", Timeframe: "6 months"},
				{TargetID: "ASTH-TGT-010", Metric: "peak_flow_variability", Value: "< 20%", Timeframe: "ongoing"},
			},
			Status: "active",
		},
		{
			GoalID:      "ASTH-GOAL-004",
			Description: "Minimize medication side effects",
			Category:    "safety",
			Priority:    "low",
			Targets: []models.GoalTarget{
				{TargetID: "ASTH-TGT-011", Metric: "ICS_dose", Value: "lowest effective dose", Timeframe: "ongoing"},
				{TargetID: "ASTH-TGT-012", Metric: "SABA_use", Value: "≤ 2 days/week", Timeframe: "4 weeks"},
			},
			Status: "active",
		},
	}

	activities := []models.Activity{
		// Controller Medications - GINA Step-based
		{
			ActivityID:  "ASTH-ACT-001",
			Type:        "medication",
			Description: "ICS-formoterol as anti-inflammatory reliever (AIR) - GINA Track 1 preferred",
			Frequency:   "as needed or maintenance + as needed (MART)",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Budesonide-Formoterol",
						"rxnorm":       "896188",
						"dose":         "80/4.5 or 160/4.5 mcg",
						"route":        "inhalation",
						"frequency":    "as-needed for Steps 1-2, MART for Steps 3-5",
						"max_per_day":  "12 puffs (160/4.5) or 8 puffs for MART",
						"indication":   "GINA preferred controller - reduces exacerbations vs SABA-only",
						"evidence":     "reduces severe exacerbations by 60% vs SABA-only",
					},
				},
				"gina_track":      "Track 1 (preferred)",
				"evidence_level":  "Grade A",
			},
			Status: "active",
		},
		{
			ActivityID:  "ASTH-ACT-002",
			Type:        "medication",
			Description: "Alternative: Low-dose ICS + SABA as needed (GINA Track 2)",
			Frequency:   "daily ICS + PRN SABA",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Fluticasone propionate",
						"rxnorm":       "896018",
						"dose":         "88-220 mcg",
						"route":        "inhalation",
						"frequency":    "twice daily",
						"indication":   "maintenance controller - traditional approach",
					},
					{
						"name":         "Albuterol",
						"rxnorm":       "197318",
						"dose":         "90 mcg/puff, 1-2 puffs",
						"route":        "inhalation",
						"frequency":    "as needed for rescue",
						"max_per_day":  "12 puffs",
					},
				},
				"gina_track":     "Track 2 (alternative)",
				"consideration":  "use when ICS-formoterol not available or not tolerated",
			},
			Status:    "conditional",
			Condition: "track_1_not_available OR track_1_intolerance",
		},
		{
			ActivityID:  "ASTH-ACT-003",
			Type:        "medication",
			Description: "Step-up therapy: Medium-dose ICS-LABA",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Fluticasone-Salmeterol",
						"rxnorm":       "896006",
						"dose":         "250/50 mcg",
						"route":        "inhalation",
						"frequency":    "twice daily",
						"indication":   "GINA Step 4 - uncontrolled on low-dose ICS-LABA",
					},
				},
				"step_criteria": "symptoms ≥ 2 days/week or night symptoms, or using SABA > 2x/week",
			},
			Status:    "conditional",
			Condition: "asthma_control = uncontrolled AND current_step >= 3",
		},
		{
			ActivityID:  "ASTH-ACT-004",
			Type:        "medication",
			Description: "Add-on therapy: LAMA for severe asthma",
			Frequency:   "daily",
			Details: map[string]interface{}{
				"medications": []map[string]interface{}{
					{
						"name":         "Tiotropium Respimat",
						"rxnorm":       "1116632",
						"dose":         "2.5 mcg",
						"route":        "inhalation",
						"frequency":    "2 puffs once daily",
						"indication":   "add-on for patients uncontrolled on medium/high-dose ICS-LABA",
						"evidence":     "reduces exacerbations, improves FEV1",
					},
				},
			},
			Status:    "conditional",
			Condition: "current_step >= 4 AND uncontrolled on ICS_LABA",
		},
		{
			ActivityID:  "ASTH-ACT-005",
			Type:        "medication",
			Description: "Biologic therapy for severe eosinophilic asthma",
			Frequency:   "every 2-8 weeks depending on agent",
			Details: map[string]interface{}{
				"options": []map[string]interface{}{
					{
						"name":         "Dupilumab",
						"rxnorm":       "1923328",
						"dose":         "200 mg or 300 mg",
						"frequency":    "every 2 weeks",
						"indication":   "eosinophils ≥150 or FeNO ≥25, or oral steroid dependent",
						"monitoring":   "eosinophil counts, FeNO",
					},
					{
						"name":         "Mepolizumab",
						"rxnorm":       "1657973",
						"dose":         "100 mg SC",
						"frequency":    "every 4 weeks",
						"indication":   "blood eosinophils ≥150 cells/μL",
					},
					{
						"name":         "Benralizumab",
						"rxnorm":       "1923334",
						"dose":         "30 mg SC",
						"frequency":    "every 4 weeks x3, then every 8 weeks",
						"indication":   "blood eosinophils ≥150 cells/μL",
					},
					{
						"name":         "Tezepelumab",
						"rxnorm":       "2549102",
						"dose":         "210 mg SC",
						"frequency":    "every 4 weeks",
						"indication":   "severe asthma regardless of phenotype",
					},
				},
				"specialist_required": true,
				"phenotyping":         "obtain eosinophils, IgE, FeNO, allergic workup before selection",
			},
			Status:    "conditional",
			Condition: "severe_asthma AND (eosinophils >= 150 OR FeNO >= 25 OR steroid_dependent)",
		},
		// Trigger Management
		{
			ActivityID:  "ASTH-ACT-006",
			Type:        "lifestyle",
			Description: "Allergen avoidance and environmental control",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"measures": []string{
					"dust mite covers for bedding",
					"HEPA air filters",
					"remove carpeting if possible",
					"control humidity < 50%",
					"pet dander management",
					"mold remediation",
					"avoid outdoor triggers (pollen, pollution)",
				},
				"testing": "consider allergy testing to identify specific triggers",
			},
			Status: "active",
		},
		{
			ActivityID:  "ASTH-ACT-007",
			Type:        "lifestyle",
			Description: "Smoking cessation and secondhand smoke avoidance",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"importance": "smoking reduces ICS efficacy and worsens outcomes",
				"resources":  "provide pharmacotherapy and counseling referral",
			},
			Status: "active",
		},
		{
			ActivityID:  "ASTH-ACT-008",
			Type:        "lifestyle",
			Description: "Physical activity with proper management",
			Frequency:   "regular",
			Details: map[string]interface{}{
				"recommendation": "regular exercise is beneficial for asthma",
				"pretreatment":   "ICS-formoterol before exercise if EIB symptoms",
				"warm_up":        "adequate warm-up reduces exercise-induced symptoms",
			},
			Status: "active",
		},
		{
			ActivityID:  "ASTH-ACT-009",
			Type:        "lifestyle",
			Description: "Weight management (if overweight/obese)",
			Frequency:   "ongoing",
			Details: map[string]interface{}{
				"benefit":         "5-10% weight loss improves asthma control",
				"recommendation":  "Mediterranean diet may have anti-inflammatory benefits",
			},
			Status:    "conditional",
			Condition: "BMI >= 25",
		},
		// Education
		{
			ActivityID:  "ASTH-ACT-010",
			Type:        "education",
			Description: "Written asthma action plan",
			Frequency:   "annual update or with any change",
			Details: map[string]interface{}{
				"zones": []map[string]string{
					{"zone": "green", "description": "well-controlled, continue maintenance"},
					{"zone": "yellow", "description": "worsening, increase controller, consider prednisone"},
					{"zone": "red", "description": "severe symptoms, take rescue, start prednisone, seek emergency care"},
				},
				"includes": []string{
					"peak flow or symptom-based zones",
					"maintenance medications",
					"reliever use instructions",
					"when to increase treatment",
					"emergency contacts",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "ASTH-ACT-011",
			Type:        "education",
			Description: "Inhaler technique demonstration and assessment",
			Frequency:   "each visit",
			Details: map[string]interface{}{
				"critical_points": []string{
					"proper priming",
					"slow deep inhalation for MDI",
					"breath-hold for 10 seconds",
					"spacer use with MDI",
					"rinse mouth after ICS",
				},
				"assessment": "teach-back method at every visit",
			},
			Status: "active",
		},
		{
			ActivityID:  "ASTH-ACT-012",
			Type:        "education",
			Description: "Trigger identification and avoidance strategies",
			Frequency:   "initial and ongoing",
			Details: map[string]interface{}{
				"common_triggers": []string{"allergens", "infections", "exercise", "weather", "irritants", "NSAIDs/aspirin", "stress"},
				"diary":           "symptom diary to identify patterns",
			},
			Status: "active",
		},
		// Appointments
		{
			ActivityID:  "ASTH-ACT-013",
			Type:        "appointment",
			Description: "Regular asthma review",
			Frequency:   "every 1-6 months based on control",
			Details: map[string]interface{}{
				"uncontrolled":        "every 2-6 weeks until controlled",
				"well_controlled":     "every 3-6 months",
				"after_exacerbation":  "within 1 week",
				"review_components": []string{
					"symptom control (ACT score)",
					"exacerbation history",
					"inhaler technique check",
					"adherence assessment",
					"side effects",
					"lung function",
					"comorbidity management",
					"action plan review",
				},
			},
			Status: "active",
		},
		{
			ActivityID:  "ASTH-ACT-014",
			Type:        "appointment",
			Description: "Allergist/Immunologist referral",
			Frequency:   "as needed",
			Details: map[string]interface{}{
				"indications": []string{
					"allergic component suspected",
					"candidate for allergen immunotherapy",
					"severe/difficult-to-control asthma",
					"occupational asthma evaluation",
				},
			},
			Status:    "conditional",
			Condition: "allergic_asthma OR severe_asthma OR occupational_exposure",
		},
	}

	monitoring := []models.MonitoringItem{
		{ItemID: "ASTH-MON-001", Parameter: "ACT score (Asthma Control Test)", Frequency: "each visit", Target: "≥ 20", AlertThreshold: "< 16 (very poorly controlled)"},
		{ItemID: "ASTH-MON-002", Parameter: "Peak flow monitoring", Frequency: "daily (if variable) or PRN", Target: "> 80% personal best", AlertThreshold: "< 50% personal best"},
		{ItemID: "ASTH-MON-003", Parameter: "Exacerbation count", Frequency: "ongoing", Target: "0 per year", AlertThreshold: "any requiring oral steroids or ED visit"},
		{ItemID: "ASTH-MON-004", Parameter: "SABA use frequency", Frequency: "each visit", Target: "≤ 2 days/week", AlertThreshold: "> 2 canisters/month"},
		{ItemID: "ASTH-MON-005", Parameter: "Spirometry (FEV1, FVC)", Frequency: "annual, before/after treatment changes", Target: "FEV1 ≥ 80% predicted", AlertThreshold: "> 12% decline from best"},
		{ItemID: "ASTH-MON-006", Parameter: "Fractional exhaled NO (FeNO)", Frequency: "baseline, during step-up/down", Target: "< 25 ppb (adults)", AlertThreshold: "> 50 ppb indicates eosinophilic inflammation"},
		{ItemID: "ASTH-MON-007", Parameter: "Blood eosinophils", Frequency: "annual, for biologic candidacy", Target: "< 150 cells/μL", AlertThreshold: "≥ 300 cells/μL (consider biologic)"},
		{ItemID: "ASTH-MON-008", Parameter: "IgE level", Frequency: "once if allergic asthma", Target: "guide omalizumab dosing", AlertThreshold: "elevated with allergic symptoms"},
		{ItemID: "ASTH-MON-009", Parameter: "Oral corticosteroid courses", Frequency: "ongoing", Target: "0 per year", AlertThreshold: "≥ 2 courses/year"},
		{ItemID: "ASTH-MON-010", Parameter: "Inhaler technique", Frequency: "each visit", Target: "correct technique", AlertThreshold: "critical errors identified"},
		{ItemID: "ASTH-MON-011", Parameter: "Medication adherence", Frequency: "each visit", Target: "≥ 80%", AlertThreshold: "< 50%"},
		{ItemID: "ASTH-MON-012", Parameter: "Flu vaccine status", Frequency: "annual", Target: "administered", AlertThreshold: "overdue"},
	}

	return &models.CarePlanTemplate{
		TemplateID:   "CP-RESP-002",
		Name:         "Asthma Chronic Care Plan",
		Description:  "Comprehensive chronic care plan for asthma management based on GINA 2024 Guidelines",
		Version:      "2024.1",
		Category:     "respiratory",
		Subcategory:  "asthma",
		ConditionRef: &models.ClinicalCondition{Code: "J45.909", System: "ICD-10", Display: "Unspecified asthma, uncomplicated"},
		Goals:        goals,
		Activities:   activities,
		Monitoring:   monitoring,
		Duration:     "ongoing",
		ReviewPeriod: "3-6 months based on control",
		Guidelines: []models.GuidelineReference{
			{GuidelineID: "GINA-2024", Name: "Global Initiative for Asthma 2024 Report", Source: "ginasthma.org", URL: "https://ginasthma.org/gina-reports/"},
			{GuidelineID: "NAEPP-2020", Name: "NAEPP Expert Panel Report 4", Source: "NHLBI", Year: 2020},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "active",
	}
}
