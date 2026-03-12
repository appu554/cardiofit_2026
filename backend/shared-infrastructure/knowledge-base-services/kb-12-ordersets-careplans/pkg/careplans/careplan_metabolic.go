// Package careplans provides chronic care plan templates
package careplans

import (
	"kb-12-ordersets-careplans/internal/models"
)

// GetDiabetesType2CarePlan returns Type 2 Diabetes chronic care plan
// Guidelines: ADA Standards of Medical Care in Diabetes 2024
func GetDiabetesType2CarePlan() *models.CarePlanTemplate {
	goals := []models.Goal{
		{
			GoalID:      "DM-GOAL-001",
			Description: "Achieve glycemic control with individualized A1c target",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "HbA1c", Code: "4548-4", TargetValue: "<7.0% (individualized)", DueDate: "3_months"},
				{Measure: "Fasting Glucose", Code: "1558-6", TargetValue: "80-130 mg/dL"},
				{Measure: "Post-meal Glucose", Code: "1521-4", TargetValue: "<180 mg/dL"},
			},
			Addresses: []models.CodeReference{
				{System: "icd10", Code: "E11.9", Display: "Type 2 diabetes mellitus without complications"},
			},
		},
		{
			GoalID:      "DM-GOAL-002",
			Description: "Achieve blood pressure control",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Systolic BP", Code: "8480-6", TargetValue: "<130 mmHg"},
				{Measure: "Diastolic BP", Code: "8462-4", TargetValue: "<80 mmHg"},
			},
		},
		{
			GoalID:      "DM-GOAL-003",
			Description: "Achieve lipid control for cardiovascular protection",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "LDL Cholesterol", Code: "2089-1", TargetValue: "<70 mg/dL if high ASCVD risk"},
				{Measure: "Triglycerides", Code: "2571-8", TargetValue: "<150 mg/dL"},
			},
		},
		{
			GoalID:      "DM-GOAL-004",
			Description: "Achieve and maintain healthy weight",
			Category:    "behavioral",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{Measure: "Weight Loss", TargetValue: "5-10% of body weight if overweight"},
				{Measure: "BMI", Code: "39156-5", TargetValue: "<25 kg/m² if feasible"},
			},
		},
		{
			GoalID:      "DM-GOAL-005",
			Description: "Prevent diabetes complications through regular screening",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Annual eye exam", TargetValue: "Completed"},
				{Measure: "Annual foot exam", TargetValue: "Completed"},
				{Measure: "Annual UACR", Code: "14959-1", TargetValue: "<30 mg/g"},
			},
		},
		{
			GoalID:      "DM-GOAL-006",
			Description: "Patient demonstrates diabetes self-management skills",
			Category:    "educational",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "DSMES completion", TargetValue: "Completed"},
				{Measure: "Self-monitoring adherence", TargetValue: ">80%"},
			},
		},
	}

	activities := []models.Activity{
		// Medications
		{
			ActivityID:   "DM-ACT-001",
			ActivityType: "medication",
			Description:  "Metformin - First-line therapy for T2DM",
			Detail: models.ActivityDetail{
				DrugCode:     "6809",
				DrugName:     "Metformin",
				Dose:         "500-2000 mg daily",
				Route:        "PO",
				Frequency:    "with meals (divided BID if >1000mg)",
				Instructions: "Start 500mg with dinner, titrate weekly by 500mg to goal/tolerance. Max 2550mg/day. Hold if eGFR <30.",
			},
			GoalReferences: []string{"DM-GOAL-001"},
		},
		{
			ActivityID:   "DM-ACT-002",
			ActivityType: "medication",
			Description:  "SGLT2 Inhibitor - Cardio/renal protection",
			Detail: models.ActivityDetail{
				DrugCode:     "1545149",
				DrugName:     "Empagliflozin or Dapagliflozin",
				Dose:         "10-25 mg daily",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "Add if ASCVD, HF, or CKD. Monitor for DKA, UTI, genital mycotic infections. Hold for surgery.",
			},
			GoalReferences: []string{"DM-GOAL-001", "DM-GOAL-003"},
		},
		{
			ActivityID:   "DM-ACT-003",
			ActivityType: "medication",
			Description:  "GLP-1 Receptor Agonist - Weight loss and CV benefit",
			Detail: models.ActivityDetail{
				DrugCode:     "475968",
				DrugName:     "Semaglutide or Dulaglutide",
				Dose:         "Per titration schedule",
				Route:        "SubQ",
				Frequency:    "weekly",
				Instructions: "Consider for patients with obesity, ASCVD, or needing weight loss. Titrate slowly to minimize GI side effects.",
			},
			GoalReferences: []string{"DM-GOAL-001", "DM-GOAL-004"},
		},
		{
			ActivityID:   "DM-ACT-004",
			ActivityType: "medication",
			Description:  "Statin therapy for cardiovascular protection",
			Detail: models.ActivityDetail{
				DrugCode:     "617318",
				DrugName:     "Atorvastatin or Rosuvastatin",
				Dose:         "High-intensity if ASCVD or high risk",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "All diabetics 40-75yo should be on at least moderate-intensity statin. High-intensity if ASCVD or 10-year risk >20%.",
			},
			GoalReferences: []string{"DM-GOAL-003"},
		},
		{
			ActivityID:   "DM-ACT-005",
			ActivityType: "medication",
			Description:  "ACE Inhibitor or ARB for renal protection",
			Detail: models.ActivityDetail{
				DrugCode:     "29046",
				DrugName:     "Lisinopril or Losartan",
				Dose:         "Per BP and albuminuria",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "Start if HTN or albuminuria >30 mg/g. Titrate to max tolerated. Monitor K+ and creatinine.",
			},
			GoalReferences: []string{"DM-GOAL-002", "DM-GOAL-005"},
		},

		// Lifestyle
		{
			ActivityID:   "DM-ACT-010",
			ActivityType: "lifestyle",
			Description:  "Medical Nutrition Therapy",
			Detail: models.ActivityDetail{
				Intervention: "Dietary modification",
				Target:       "Carbohydrate-aware eating pattern",
				Instructions: "Refer to RD for individualized meal plan. Focus on consistent carb intake, portion control, and Mediterranean or DASH pattern.",
			},
			GoalReferences: []string{"DM-GOAL-001", "DM-GOAL-004"},
		},
		{
			ActivityID:   "DM-ACT-011",
			ActivityType: "lifestyle",
			Description:  "Physical Activity Program",
			Detail: models.ActivityDetail{
				Intervention: "Exercise prescription",
				Target:       "150 min/week moderate activity",
				Instructions: "Combination of aerobic (150 min/week) and resistance training (2-3x/week). Start slowly if sedentary. Consider cardiac clearance if high risk.",
			},
			GoalReferences: []string{"DM-GOAL-001", "DM-GOAL-004"},
		},
		{
			ActivityID:   "DM-ACT-012",
			ActivityType: "lifestyle",
			Description:  "Smoking Cessation",
			Detail: models.ActivityDetail{
				Intervention: "Tobacco cessation counseling and pharmacotherapy",
				Instructions: "Assess at every visit. Offer NRT, varenicline, or bupropion. Refer to tobacco quitline.",
			},
			GoalReferences: []string{"DM-GOAL-003", "DM-GOAL-005"},
		},

		// Education
		{
			ActivityID:   "DM-ACT-020",
			ActivityType: "education",
			Description:  "Diabetes Self-Management Education and Support (DSMES)",
			Detail: models.ActivityDetail{
				Topic:        "Comprehensive diabetes education",
				Format:       "Group or individual sessions",
				Materials:    "ADA educational materials",
				Instructions: "Refer at diagnosis, annually, when complications arise, and during transitions. AADE7 Self-Care Behaviors focus.",
			},
			GoalReferences: []string{"DM-GOAL-006"},
		},
		{
			ActivityID:   "DM-ACT-021",
			ActivityType: "education",
			Description:  "Self-Monitoring of Blood Glucose (SMBG)",
			Detail: models.ActivityDetail{
				Topic:        "Glucose monitoring technique and pattern management",
				Instructions: "Teach meter use, when to check, how to record, and pattern recognition. Consider CGM for insulin users.",
			},
			GoalReferences: []string{"DM-GOAL-001", "DM-GOAL-006"},
		},
		{
			ActivityID:   "DM-ACT-022",
			ActivityType: "education",
			Description:  "Hypoglycemia Recognition and Treatment",
			Detail: models.ActivityDetail{
				Topic:        "Hypoglycemia management",
				Instructions: "Teach 15-15 rule. Review hypoglycemia causes, symptoms, and prevention. Prescribe glucagon if on insulin or SU.",
			},
			GoalReferences: []string{"DM-GOAL-006"},
		},

		// Appointments
		{
			ActivityID:   "DM-ACT-030",
			ActivityType: "appointment",
			Description:  "Quarterly Diabetes Follow-up Visit",
			Detail: models.ActivityDetail{
				ServiceType:  "Office visit",
				Provider:     "PCP or Endocrinology",
				Duration:     "30 minutes",
				Instructions: "Review glucose logs, A1c, medication adherence, lifestyle, and complications screening.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "quarterly",
				Frequency: 1,
				Interval:  "quarter",
			},
			GoalReferences: []string{"DM-GOAL-001", "DM-GOAL-006"},
		},
		{
			ActivityID:   "DM-ACT-031",
			ActivityType: "appointment",
			Description:  "Annual Dilated Eye Exam",
			Detail: models.ActivityDetail{
				ServiceType:  "Dilated fundoscopic exam",
				Provider:     "Ophthalmology or Optometry",
				Instructions: "Screen for diabetic retinopathy. More frequent if retinopathy present.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "annually",
				Frequency: 1,
				Interval:  "year",
			},
			GoalReferences: []string{"DM-GOAL-005"},
		},
		{
			ActivityID:   "DM-ACT-032",
			ActivityType: "appointment",
			Description:  "Annual Foot Exam",
			Detail: models.ActivityDetail{
				ServiceType:  "Comprehensive foot exam",
				Provider:     "PCP or Podiatry",
				Instructions: "Monofilament, tuning fork, visual inspection. More frequent if neuropathy or PAD.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "annually",
				Frequency: 1,
				Interval:  "year",
			},
			GoalReferences: []string{"DM-GOAL-005"},
		},
	}

	monitoringItems := []models.MonitoringItem{
		{ItemID: "DM-MON-001", Name: "HbA1c", LabCode: "4548-4", Frequency: "quarterly", NormalRange: "<7%", AlertRange: ">9%", Instructions: "Check q3mo until stable, then q6mo"},
		{ItemID: "DM-MON-002", Name: "Fasting Lipid Panel", LabCode: "57698-3", Frequency: "annually", NormalRange: "LDL<100, TG<150", AlertRange: "LDL>190, TG>500"},
		{ItemID: "DM-MON-003", Name: "Comprehensive Metabolic Panel", LabCode: "24323-8", Frequency: "annually", Instructions: "Assess renal function, electrolytes"},
		{ItemID: "DM-MON-004", Name: "Urine Albumin-to-Creatinine Ratio", LabCode: "14959-1", Frequency: "annually", NormalRange: "<30 mg/g", AlertRange: ">300 mg/g"},
		{ItemID: "DM-MON-005", Name: "Blood Pressure", Frequency: "every_visit", NormalRange: "<130/80", AlertRange: ">140/90"},
		{ItemID: "DM-MON-006", Name: "Weight/BMI", LabCode: "39156-5", Frequency: "every_visit", Instructions: "Track trend over time"},
		{ItemID: "DM-MON-007", Name: "Hepatic Panel", LabCode: "24325-3", Frequency: "baseline_and_prn", Instructions: "Baseline before starting statin, then as needed"},
	}

	return &models.CarePlanTemplate{
		PlanID:          "CARE-DM2-001",
		Condition:       "Type 2 Diabetes Mellitus",
		Name:            "Type 2 Diabetes Comprehensive Care Plan",
		GuidelineSource: "ADA Standards of Medical Care in Diabetes 2024",
		Goals:           goals,
		Activities:      activities,
		MonitoringItems: monitoringItems,
		Active:          true,
	}
}

// GetObesityCarePlan returns Obesity/Weight Management care plan
// Guidelines: Obesity Medicine Association, AGA Obesity Guidelines
func GetObesityCarePlan() *models.CarePlanTemplate {
	goals := []models.Goal{
		{
			GoalID:      "OBE-GOAL-001",
			Description: "Achieve clinically meaningful weight loss",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Weight Loss", TargetValue: "≥5% body weight at 6 months", DueDate: "6_months"},
				{Measure: "Sustained Loss", TargetValue: "≥10% body weight at 12 months", DueDate: "12_months"},
			},
			Addresses: []models.CodeReference{
				{System: "icd10", Code: "E66.9", Display: "Obesity, unspecified"},
			},
		},
		{
			GoalID:      "OBE-GOAL-002",
			Description: "Improve obesity-related comorbidities",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Blood Pressure", TargetValue: "Improved or at goal"},
				{Measure: "Glycemic Control", TargetValue: "Improved or at goal"},
				{Measure: "Lipid Profile", TargetValue: "Improved or at goal"},
			},
		},
		{
			GoalID:      "OBE-GOAL-003",
			Description: "Establish sustainable healthy eating habits",
			Category:    "behavioral",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Caloric Deficit", TargetValue: "500-750 kcal/day deficit"},
				{Measure: "Dietary Quality", TargetValue: "Mediterranean or DASH pattern"},
			},
		},
		{
			GoalID:      "OBE-GOAL-004",
			Description: "Achieve recommended physical activity level",
			Category:    "behavioral",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{Measure: "Aerobic Activity", TargetValue: "≥150 min/week moderate or ≥75 min vigorous"},
				{Measure: "Resistance Training", TargetValue: "2x/week"},
			},
		},
		{
			GoalID:      "OBE-GOAL-005",
			Description: "Address psychological factors affecting weight",
			Category:    "behavioral",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{Measure: "Emotional Eating", TargetValue: "Identified and addressed"},
				{Measure: "Sleep Quality", TargetValue: "7-9 hours/night"},
			},
		},
	}

	activities := []models.Activity{
		// Lifestyle Interventions
		{
			ActivityID:   "OBE-ACT-001",
			ActivityType: "lifestyle",
			Description:  "Intensive Behavioral Therapy (IBT) for Obesity",
			Detail: models.ActivityDetail{
				Intervention: "High-intensity lifestyle counseling",
				Target:       "≥14 sessions in first 6 months",
				Instructions: "Weekly sessions for first 3 months, then biweekly. Focus on diet, activity, and behavioral strategies.",
			},
			GoalReferences: []string{"OBE-GOAL-001", "OBE-GOAL-003"},
		},
		{
			ActivityID:   "OBE-ACT-002",
			ActivityType: "lifestyle",
			Description:  "Calorie-Restricted Diet",
			Detail: models.ActivityDetail{
				Intervention: "Reduced calorie eating plan",
				Target:       "1200-1500 kcal/day women, 1500-1800 kcal/day men",
				Instructions: "Work with RD to develop individualized plan. Focus on nutrient density, portion control, and food tracking.",
			},
			GoalReferences: []string{"OBE-GOAL-001", "OBE-GOAL-003"},
		},
		{
			ActivityID:   "OBE-ACT-003",
			ActivityType: "lifestyle",
			Description:  "Progressive Physical Activity Program",
			Detail: models.ActivityDetail{
				Intervention: "Structured exercise prescription",
				Target:       "Progress to 200-300 min/week for weight maintenance",
				Instructions: "Start with 10 min walks, gradually increase. Include strength training. May need cardiac clearance.",
			},
			GoalReferences: []string{"OBE-GOAL-001", "OBE-GOAL-004"},
		},

		// Pharmacotherapy
		{
			ActivityID:   "OBE-ACT-010",
			ActivityType: "medication",
			Description:  "GLP-1 Receptor Agonist for Weight Loss",
			Detail: models.ActivityDetail{
				DrugCode:     "2169305",
				DrugName:     "Semaglutide (Wegovy) or Tirzepatide (Zepbound)",
				Dose:         "Per titration schedule",
				Route:        "SubQ",
				Frequency:    "weekly",
				Instructions: "Consider if BMI ≥30 or ≥27 with comorbidity. Titrate slowly. Monitor for GI side effects, pancreatitis.",
			},
			GoalReferences: []string{"OBE-GOAL-001"},
		},
		{
			ActivityID:   "OBE-ACT-011",
			ActivityType: "medication",
			Description:  "Orlistat (if GLP-1 not appropriate)",
			Detail: models.ActivityDetail{
				DrugCode:     "37925",
				DrugName:     "Orlistat",
				Dose:         "120 mg TID with fatty meals",
				Route:        "PO",
				Instructions: "Take with meals containing fat. May cause GI side effects. Supplement fat-soluble vitamins.",
			},
			GoalReferences: []string{"OBE-GOAL-001"},
		},

		// Monitoring and Support
		{
			ActivityID:   "OBE-ACT-020",
			ActivityType: "monitoring",
			Description:  "Weight and Waist Circumference Tracking",
			Detail: models.ActivityDetail{
				Interval:    "weekly_self_monitoring",
				Instructions: "Daily weigh-ins recommended. Monthly waist circumference. Log in app or journal.",
			},
			GoalReferences: []string{"OBE-GOAL-001"},
		},
		{
			ActivityID:   "OBE-ACT-021",
			ActivityType: "monitoring",
			Description:  "Food and Activity Diary",
			Detail: models.ActivityDetail{
				Intervention: "Self-monitoring with food/activity log",
				Instructions: "Use app (MyFitnessPal, Lose It) or paper diary. Record all intake and activity. Review with provider.",
			},
			GoalReferences: []string{"OBE-GOAL-003", "OBE-GOAL-004"},
		},

		// Appointments
		{
			ActivityID:   "OBE-ACT-030",
			ActivityType: "appointment",
			Description:  "Monthly Weight Management Visit",
			Detail: models.ActivityDetail{
				ServiceType:  "Weight management follow-up",
				Duration:     "20-30 minutes",
				Instructions: "Review weight trend, food/activity logs, barriers, medication tolerance. Adjust plan as needed.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "monthly",
				Frequency: 1,
				Interval:  "month",
			},
			GoalReferences: []string{"OBE-GOAL-001"},
		},
		{
			ActivityID:   "OBE-ACT-031",
			ActivityType: "appointment",
			Description:  "Registered Dietitian Consultation",
			Detail: models.ActivityDetail{
				ServiceType: "MNT for obesity",
				Provider:    "Registered Dietitian",
				Instructions: "Initial comprehensive assessment, then follow-up as needed. Covered by Medicare for obesity.",
			},
			GoalReferences: []string{"OBE-GOAL-003"},
		},
		{
			ActivityID:   "OBE-ACT-032",
			ActivityType: "appointment",
			Description:  "Bariatric Surgery Evaluation (if appropriate)",
			Detail: models.ActivityDetail{
				ServiceType:  "Surgical consultation",
				Provider:     "Bariatric Surgery",
				Instructions: "Consider referral if BMI ≥40 or ≥35 with comorbidities and failed lifestyle/medical therapy.",
			},
			GoalReferences: []string{"OBE-GOAL-001"},
		},

		// Psychological Support
		{
			ActivityID:   "OBE-ACT-040",
			ActivityType: "education",
			Description:  "Behavioral Counseling for Eating Behaviors",
			Detail: models.ActivityDetail{
				Topic:        "Cognitive behavioral therapy for weight management",
				Format:       "Individual or group sessions",
				Instructions: "Address emotional eating, binge eating, and unhealthy relationships with food. Screen for eating disorders.",
			},
			GoalReferences: []string{"OBE-GOAL-005"},
		},
	}

	monitoringItems := []models.MonitoringItem{
		{ItemID: "OBE-MON-001", Name: "Weight", Frequency: "every_visit", Instructions: "Track % weight change from baseline"},
		{ItemID: "OBE-MON-002", Name: "BMI", LabCode: "39156-5", Frequency: "every_visit"},
		{ItemID: "OBE-MON-003", Name: "Waist Circumference", Frequency: "quarterly", AlertRange: ">40in men, >35in women"},
		{ItemID: "OBE-MON-004", Name: "Blood Pressure", Frequency: "every_visit", NormalRange: "<130/80"},
		{ItemID: "OBE-MON-005", Name: "Fasting Glucose or A1c", LabCode: "4548-4", Frequency: "annually", AlertRange: "A1c ≥5.7%"},
		{ItemID: "OBE-MON-006", Name: "Lipid Panel", LabCode: "57698-3", Frequency: "annually"},
		{ItemID: "OBE-MON-007", Name: "Hepatic Panel", LabCode: "24325-3", Frequency: "annually", Instructions: "Screen for NAFLD"},
	}

	return &models.CarePlanTemplate{
		PlanID:          "CARE-OBE-001",
		Condition:       "Obesity",
		Name:            "Obesity/Weight Management Care Plan",
		GuidelineSource: "Obesity Medicine Association Guidelines, AGA Clinical Practice Guidelines 2022",
		Goals:           goals,
		Activities:      activities,
		MonitoringItems: monitoringItems,
		Active:          true,
	}
}

// GetHypothyroidismCarePlan returns Hypothyroidism care plan
// Guidelines: ATA Guidelines for Treatment of Hypothyroidism
func GetHypothyroidismCarePlan() *models.CarePlanTemplate {
	goals := []models.Goal{
		{
			GoalID:      "HYPO-GOAL-001",
			Description: "Achieve euthyroid state with normalized TSH",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "TSH", Code: "3016-3", TargetValue: "0.5-2.5 mIU/L (individualized)", DueDate: "3_months"},
				{Measure: "Free T4", Code: "3024-7", TargetValue: "Normal range"},
			},
			Addresses: []models.CodeReference{
				{System: "icd10", Code: "E03.9", Display: "Hypothyroidism, unspecified"},
			},
		},
		{
			GoalID:      "HYPO-GOAL-002",
			Description: "Resolve hypothyroid symptoms",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Fatigue", TargetValue: "Resolved or improved"},
				{Measure: "Weight", TargetValue: "Stable or improving"},
				{Measure: "Cognitive Function", TargetValue: "Improved"},
			},
		},
		{
			GoalID:      "HYPO-GOAL-003",
			Description: "Optimize cardiovascular risk factors",
			Category:    "clinical",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{Measure: "LDL Cholesterol", Code: "2089-1", TargetValue: "Improved with treatment"},
			},
		},
		{
			GoalID:      "HYPO-GOAL-004",
			Description: "Patient demonstrates understanding of thyroid condition",
			Category:    "educational",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{Measure: "Medication Adherence", TargetValue: ">90%"},
				{Measure: "Proper Medication Timing", TargetValue: "Demonstrates understanding"},
			},
		},
	}

	activities := []models.Activity{
		// Medications
		{
			ActivityID:   "HYPO-ACT-001",
			ActivityType: "medication",
			Description:  "Levothyroxine Replacement Therapy",
			Detail: models.ActivityDetail{
				DrugCode:     "10582",
				DrugName:     "Levothyroxine",
				Dose:         "1.6 mcg/kg/day (full replacement), start lower in elderly/cardiac",
				Route:        "PO",
				Frequency:    "once daily, morning on empty stomach",
				Instructions: "Take 30-60 min before breakfast or 3-4 hours after dinner. Avoid calcium, iron, and antacids within 4 hours. Same brand preferred for consistency.",
			},
			GoalReferences: []string{"HYPO-GOAL-001", "HYPO-GOAL-002"},
		},
		{
			ActivityID:   "HYPO-ACT-002",
			ActivityType: "medication",
			Description:  "Alternative: Liothyronine (T3) if needed",
			Detail: models.ActivityDetail{
				DrugCode:     "10609",
				DrugName:     "Liothyronine",
				Dose:         "5-25 mcg daily in divided doses",
				Route:        "PO",
				Frequency:    "BID-TID",
				Instructions: "Only consider if symptomatic on adequate LT4 with normal TSH. Not first-line. Avoid in cardiac disease.",
			},
			GoalReferences: []string{"HYPO-GOAL-001", "HYPO-GOAL-002"},
		},

		// Education
		{
			ActivityID:   "HYPO-ACT-010",
			ActivityType: "education",
			Description:  "Thyroid Disease Education",
			Detail: models.ActivityDetail{
				Topic:        "Understanding hypothyroidism and treatment",
				Instructions: "Explain chronic nature, importance of adherence, drug interactions, and when to seek care.",
			},
			GoalReferences: []string{"HYPO-GOAL-004"},
		},
		{
			ActivityID:   "HYPO-ACT-011",
			ActivityType: "education",
			Description:  "Medication Timing and Interactions",
			Detail: models.ActivityDetail{
				Topic:        "Proper levothyroxine administration",
				Instructions: "Empty stomach, avoid interfering substances (calcium, iron, PPI, soy). Consistency in timing.",
			},
			GoalReferences: []string{"HYPO-GOAL-001", "HYPO-GOAL-004"},
		},

		// Appointments
		{
			ActivityID:   "HYPO-ACT-020",
			ActivityType: "appointment",
			Description:  "TSH Follow-up (Initial Titration)",
			Detail: models.ActivityDetail{
				ServiceType:  "Lab and office visit",
				Duration:     "15 minutes",
				Instructions: "Check TSH 6-8 weeks after starting or changing dose. Adjust by 12.5-25 mcg increments.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "monthly",
				Frequency: 2, // Every 2 months during titration
				Interval:  "month",
			},
			GoalReferences: []string{"HYPO-GOAL-001"},
		},
		{
			ActivityID:   "HYPO-ACT-021",
			ActivityType: "appointment",
			Description:  "Annual Thyroid Review",
			Detail: models.ActivityDetail{
				ServiceType:  "Annual follow-up",
				Duration:     "20 minutes",
				Instructions: "Once stable, annual TSH. Assess symptoms, weight, cardiac status. May need dose adjustment with weight change or aging.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "annually",
				Frequency: 1,
				Interval:  "year",
			},
			GoalReferences: []string{"HYPO-GOAL-001", "HYPO-GOAL-002"},
		},
	}

	monitoringItems := []models.MonitoringItem{
		{ItemID: "HYPO-MON-001", Name: "TSH", LabCode: "3016-3", Frequency: "6-8 weeks during titration, then annually", NormalRange: "0.5-4.0 mIU/L", AlertRange: "<0.1 or >10 mIU/L"},
		{ItemID: "HYPO-MON-002", Name: "Free T4", LabCode: "3024-7", Frequency: "with TSH if abnormal", NormalRange: "0.8-1.8 ng/dL"},
		{ItemID: "HYPO-MON-003", Name: "Lipid Panel", LabCode: "57698-3", Frequency: "baseline and 3 months after euthyroid", Instructions: "Hypothyroidism can cause hyperlipidemia"},
		{ItemID: "HYPO-MON-004", Name: "Weight", Frequency: "every_visit", Instructions: "Monitor for changes suggesting over/under replacement"},
		{ItemID: "HYPO-MON-005", Name: "Heart Rate", Frequency: "every_visit", AlertRange: ">100 may indicate over-replacement"},
	}

	return &models.CarePlanTemplate{
		PlanID:          "CARE-HYPO-001",
		Condition:       "Hypothyroidism",
		Name:            "Hypothyroidism Management Care Plan",
		GuidelineSource: "ATA Guidelines for Treatment of Hypothyroidism 2014",
		Goals:           goals,
		Activities:      activities,
		MonitoringItems: monitoringItems,
		Active:          true,
	}
}

// GetMetabolicCarePlans returns all metabolic chronic care plans
func GetMetabolicCarePlans() []*models.CarePlanTemplate {
	return []*models.CarePlanTemplate{
		GetDiabetesType2CarePlan(),
		GetObesityCarePlan(),
		GetHypothyroidismCarePlan(),
	}
}
