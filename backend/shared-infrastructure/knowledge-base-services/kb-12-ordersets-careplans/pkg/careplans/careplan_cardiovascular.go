// Package careplans provides chronic care plan templates for cardiovascular conditions
package careplans

import (
	"kb-12-ordersets-careplans/internal/models"
)

// GetHypertensionCarePlan returns Hypertension care plan
// Guidelines: ACC/AHA Hypertension Guidelines 2017
func GetHypertensionCarePlan() *models.CarePlanTemplate {
	goals := []models.Goal{
		{
			GoalID:      "HTN-GOAL-001",
			Description: "Achieve blood pressure goal",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Systolic BP", Code: "8480-6", TargetValue: "<130 mmHg (or <140 if low risk)"},
				{Measure: "Diastolic BP", Code: "8462-4", TargetValue: "<80 mmHg"},
				{Measure: "Home BP", TargetValue: "<130/80 mmHg"},
			},
			Addresses: []models.CodeReference{
				{System: "icd10", Code: "I10", Display: "Essential hypertension"},
			},
		},
		{
			GoalID:      "HTN-GOAL-002",
			Description: "Reduce cardiovascular risk",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "10-Year ASCVD Risk", TargetValue: "Reduced by ≥25%"},
				{Measure: "LDL Cholesterol", Code: "2089-1", TargetValue: "<100 mg/dL or per risk category"},
			},
		},
		{
			GoalID:      "HTN-GOAL-003",
			Description: "Implement lifestyle modifications",
			Category:    "behavioral",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Sodium Intake", TargetValue: "<2300 mg/day (ideal <1500 mg)"},
				{Measure: "Weight", TargetValue: "Achieve healthy BMI or lose 5-10%"},
				{Measure: "Physical Activity", TargetValue: "≥150 min/week moderate activity"},
			},
		},
		{
			GoalID:      "HTN-GOAL-004",
			Description: "Prevent hypertensive target organ damage",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "eGFR", TargetValue: "Stable or improved"},
				{Measure: "UACR", Code: "14959-1", TargetValue: "<30 mg/g"},
				{Measure: "LVH", TargetValue: "Regression or prevention"},
			},
		},
		{
			GoalID:      "HTN-GOAL-005",
			Description: "Patient performs accurate home BP monitoring",
			Category:    "educational",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{Measure: "Home BP Log", TargetValue: "Maintained and reviewed"},
				{Measure: "Proper Technique", TargetValue: "Demonstrates correct technique"},
			},
		},
	}

	activities := []models.Activity{
		// First-line Medications
		{
			ActivityID:   "HTN-ACT-001",
			ActivityType: "medication",
			Description:  "ACE Inhibitor or ARB (if diabetes, CKD, or HF)",
			Detail: models.ActivityDetail{
				DrugCode:     "29046",
				DrugName:     "Lisinopril (ACEi) or Losartan (ARB)",
				Dose:         "Start low, titrate to goal or max",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "First-line for DM, CKD, proteinuria, HF. Monitor K+ and Cr 1-2 weeks after starting.",
			},
			GoalReferences: []string{"HTN-GOAL-001", "HTN-GOAL-004"},
		},
		{
			ActivityID:   "HTN-ACT-002",
			ActivityType: "medication",
			Description:  "Calcium Channel Blocker",
			Detail: models.ActivityDetail{
				DrugCode:     "17767",
				DrugName:     "Amlodipine",
				Dose:         "2.5-10 mg daily",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "First-line option. Good for Black patients. May cause peripheral edema.",
			},
			GoalReferences: []string{"HTN-GOAL-001"},
		},
		{
			ActivityID:   "HTN-ACT-003",
			ActivityType: "medication",
			Description:  "Thiazide Diuretic",
			Detail: models.ActivityDetail{
				DrugCode:     "5487",
				DrugName:     "Chlorthalidone or HCTZ",
				Dose:         "12.5-25 mg daily",
				Route:        "PO",
				Frequency:    "daily (morning)",
				Instructions: "First-line option. Chlorthalidone preferred. Monitor electrolytes, glucose, uric acid.",
			},
			GoalReferences: []string{"HTN-GOAL-001"},
		},

		// Lifestyle
		{
			ActivityID:   "HTN-ACT-010",
			ActivityType: "lifestyle",
			Description:  "DASH Diet",
			Detail: models.ActivityDetail{
				Intervention: "DASH dietary pattern",
				Target:       "Rich in fruits, vegetables, whole grains, low-fat dairy",
				Instructions: "Reduces BP 8-14 mmHg. Limit sodium, saturated fat, red meat. Consider RD referral.",
			},
			GoalReferences: []string{"HTN-GOAL-001", "HTN-GOAL-003"},
		},
		{
			ActivityID:   "HTN-ACT-011",
			ActivityType: "lifestyle",
			Description:  "Sodium Restriction",
			Detail: models.ActivityDetail{
				Intervention: "Dietary sodium reduction",
				Target:       "<2300 mg/day, ideally <1500 mg/day",
				Instructions: "Avoid processed foods, read labels, cook at home. Can reduce BP 5-6 mmHg.",
			},
			GoalReferences: []string{"HTN-GOAL-001", "HTN-GOAL-003"},
		},
		{
			ActivityID:   "HTN-ACT-012",
			ActivityType: "lifestyle",
			Description:  "Regular Aerobic Exercise",
			Detail: models.ActivityDetail{
				Intervention: "Structured exercise program",
				Target:       "≥150 min/week moderate or 75 min vigorous",
				Instructions: "Walking, swimming, cycling. Start slowly if sedentary. Can reduce BP 5-8 mmHg.",
			},
			GoalReferences: []string{"HTN-GOAL-001", "HTN-GOAL-003"},
		},
		{
			ActivityID:   "HTN-ACT-013",
			ActivityType: "lifestyle",
			Description:  "Alcohol Moderation",
			Detail: models.ActivityDetail{
				Intervention: "Limit alcohol intake",
				Target:       "≤2 drinks/day men, ≤1 drink/day women",
				Instructions: "Excess alcohol raises BP and blunts medication effect.",
			},
			GoalReferences: []string{"HTN-GOAL-001", "HTN-GOAL-003"},
		},

		// Monitoring
		{
			ActivityID:   "HTN-ACT-020",
			ActivityType: "monitoring",
			Description:  "Home Blood Pressure Monitoring",
			Detail: models.ActivityDetail{
				Intervention: "Self-measured BP (SMBP)",
				Interval:    "twice daily (AM and PM)",
				Instructions: "Validated upper-arm cuff. Morning before meds, evening before dinner. Rest 5 min. Take 2 readings, 1 min apart. Log results.",
			},
			GoalReferences: []string{"HTN-GOAL-001", "HTN-GOAL-005"},
		},

		// Education
		{
			ActivityID:   "HTN-ACT-030",
			ActivityType: "education",
			Description:  "Hypertension Self-Management Education",
			Detail: models.ActivityDetail{
				Topic:        "Understanding hypertension and treatment",
				Instructions: "Explain silent nature of HTN, importance of adherence, lifestyle impact, and complications if untreated.",
			},
			GoalReferences: []string{"HTN-GOAL-005"},
		},
		{
			ActivityID:   "HTN-ACT-031",
			ActivityType: "education",
			Description:  "Home BP Monitoring Technique",
			Detail: models.ActivityDetail{
				Topic:        "Proper BP measurement technique",
				Instructions: "Teach cuff placement, body position, timing, and when to notify provider of high readings.",
			},
			GoalReferences: []string{"HTN-GOAL-005"},
		},

		// Appointments
		{
			ActivityID:   "HTN-ACT-040",
			ActivityType: "appointment",
			Description:  "Monthly BP Follow-up During Titration",
			Detail: models.ActivityDetail{
				ServiceType: "Office or telehealth visit",
				Duration:    "15 minutes",
				Instructions: "Review home BP log, assess symptoms, titrate medications until at goal.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "monthly",
				Frequency: 1,
				Interval:  "month",
			},
			GoalReferences: []string{"HTN-GOAL-001"},
		},
		{
			ActivityID:   "HTN-ACT-041",
			ActivityType: "appointment",
			Description:  "Quarterly Follow-up Once Stable",
			Detail: models.ActivityDetail{
				ServiceType:  "Office visit",
				Duration:     "15-20 minutes",
				Instructions: "Once at goal, visit every 3-6 months. Review home BP, lifestyle, medication adherence.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "quarterly",
				Frequency: 1,
				Interval:  "quarter",
			},
			GoalReferences: []string{"HTN-GOAL-001", "HTN-GOAL-002"},
		},
	}

	monitoringItems := []models.MonitoringItem{
		{ItemID: "HTN-MON-001", Name: "Blood Pressure", Frequency: "every_visit", NormalRange: "<130/80", AlertRange: "≥180/120 (hypertensive crisis)"},
		{ItemID: "HTN-MON-002", Name: "Basic Metabolic Panel", LabCode: "51990-0", Frequency: "annually_and_after_med_changes", Instructions: "Monitor K+, Cr with ACEi/ARB/diuretics"},
		{ItemID: "HTN-MON-003", Name: "Lipid Panel", LabCode: "57698-3", Frequency: "annually"},
		{ItemID: "HTN-MON-004", Name: "Urine Albumin-to-Creatinine Ratio", LabCode: "14959-1", Frequency: "annually", NormalRange: "<30 mg/g"},
		{ItemID: "HTN-MON-005", Name: "ECG", Frequency: "baseline_and_as_needed", Instructions: "Assess for LVH, arrhythmias"},
		{ItemID: "HTN-MON-006", Name: "Fasting Glucose or A1c", Frequency: "annually", Instructions: "Screen for diabetes"},
	}

	return &models.CarePlanTemplate{
		PlanID:          "CARE-HTN-001",
		Condition:       "Essential Hypertension",
		Name:            "Hypertension Management Care Plan",
		GuidelineSource: "ACC/AHA Hypertension Guidelines 2017",
		Goals:           goals,
		Activities:      activities,
		MonitoringItems: monitoringItems,
		Active:          true,
	}
}

// GetHeartFailureCarePlan returns Heart Failure (HFrEF) care plan
// Guidelines: ACC/AHA/HFSA Heart Failure Guidelines 2022
func GetHeartFailureCarePlan() *models.CarePlanTemplate {
	goals := []models.Goal{
		{
			GoalID:      "HF-GOAL-001",
			Description: "Optimize guideline-directed medical therapy (GDMT)",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "GDMT Achievement", TargetValue: "On all 4 pillars at target doses"},
				{Measure: "ACEi/ARB/ARNI", TargetValue: "At target dose"},
				{Measure: "Beta-blocker", TargetValue: "At target dose"},
				{Measure: "MRA", TargetValue: "On therapy if EF≤35%"},
				{Measure: "SGLT2i", TargetValue: "On therapy"},
			},
			Addresses: []models.CodeReference{
				{System: "icd10", Code: "I50.9", Display: "Heart failure, unspecified"},
			},
		},
		{
			GoalID:      "HF-GOAL-002",
			Description: "Improve symptoms and functional capacity",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "NYHA Class", TargetValue: "Improve by ≥1 class"},
				{Measure: "6-Minute Walk Distance", TargetValue: "Improve by ≥50 meters"},
				{Measure: "Quality of Life (KCCQ)", TargetValue: "Improve by ≥5 points"},
			},
		},
		{
			GoalID:      "HF-GOAL-003",
			Description: "Reduce hospitalizations",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "30-Day Readmission", TargetValue: "Avoided"},
				{Measure: "Annual HF Hospitalizations", TargetValue: "Reduced by ≥50%"},
			},
		},
		{
			GoalID:      "HF-GOAL-004",
			Description: "Maintain euvolemic state",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Daily Weight", TargetValue: "Stable (within 2-3 lbs of dry weight)"},
				{Measure: "Edema", TargetValue: "Absent or minimal"},
				{Measure: "Orthopnea/PND", TargetValue: "Absent"},
			},
		},
		{
			GoalID:      "HF-GOAL-005",
			Description: "Patient demonstrates HF self-care",
			Category:    "educational",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Daily Weights", TargetValue: "Performed and logged"},
				{Measure: "Sodium Restriction", TargetValue: "<2000 mg/day"},
				{Measure: "Medication Adherence", TargetValue: ">90%"},
			},
		},
	}

	activities := []models.Activity{
		// GDMT - 4 Pillars
		{
			ActivityID:   "HF-ACT-001",
			ActivityType: "medication",
			Description:  "ARNI or ACEi/ARB",
			Detail: models.ActivityDetail{
				DrugCode:     "1656349", // Sacubitril/valsartan
				DrugName:     "Sacubitril/Valsartan (ARNI) preferred, or ACEi/ARB",
				Dose:         "Titrate to target (sacubitril/valsartan 97/103 mg BID)",
				Route:        "PO",
				Frequency:    "twice daily (ARNI) or daily (ACEi/ARB)",
				Instructions: "ARNI preferred if tolerating ACEi/ARB. 36-hour washout between ACEi and ARNI. Monitor K+, Cr, BP.",
			},
			GoalReferences: []string{"HF-GOAL-001", "HF-GOAL-002"},
		},
		{
			ActivityID:   "HF-ACT-002",
			ActivityType: "medication",
			Description:  "Evidence-Based Beta-Blocker",
			Detail: models.ActivityDetail{
				DrugCode:     "20352",
				DrugName:     "Carvedilol, Metoprolol Succinate, or Bisoprolol",
				Dose:         "Titrate to target (carvedilol 25mg BID, metoprolol 200mg daily)",
				Route:        "PO",
				Frequency:    "daily or twice daily",
				Instructions: "Start low, increase slowly (every 2 weeks). Only use evidence-based BB. May temporarily worsen symptoms.",
			},
			GoalReferences: []string{"HF-GOAL-001", "HF-GOAL-002"},
		},
		{
			ActivityID:   "HF-ACT-003",
			ActivityType: "medication",
			Description:  "Mineralocorticoid Receptor Antagonist (MRA)",
			Detail: models.ActivityDetail{
				DrugCode:     "9997",
				DrugName:     "Spironolactone or Eplerenone",
				Dose:         "Spironolactone 25-50mg or Eplerenone 25-50mg daily",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "Add if EF≤35% and NYHA II-IV. Contraindicated if K+>5.0 or CrCl<30. Monitor K+ closely.",
			},
			GoalReferences: []string{"HF-GOAL-001", "HF-GOAL-002"},
		},
		{
			ActivityID:   "HF-ACT-004",
			ActivityType: "medication",
			Description:  "SGLT2 Inhibitor",
			Detail: models.ActivityDetail{
				DrugCode:     "1545149",
				DrugName:     "Dapagliflozin or Empagliflozin",
				Dose:         "10 mg daily",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "New standard of care for HFrEF and HFpEF. Benefits regardless of diabetes status. Monitor for UTI, genital infections.",
			},
			GoalReferences: []string{"HF-GOAL-001", "HF-GOAL-002"},
		},

		// Diuretics
		{
			ActivityID:   "HF-ACT-005",
			ActivityType: "medication",
			Description:  "Loop Diuretic for Volume Management",
			Detail: models.ActivityDetail{
				DrugCode:     "4603",
				DrugName:     "Furosemide, Torsemide, or Bumetanide",
				Dose:         "Individualized for euvolemia",
				Route:        "PO",
				Frequency:    "daily or BID",
				Instructions: "Lowest dose to maintain euvolemia. Teach flexible diuretic protocol for weight gain. Monitor electrolytes.",
			},
			GoalReferences: []string{"HF-GOAL-004"},
		},

		// Lifestyle
		{
			ActivityID:   "HF-ACT-010",
			ActivityType: "lifestyle",
			Description:  "Sodium Restriction",
			Detail: models.ActivityDetail{
				Intervention: "Low sodium diet",
				Target:       "<2000 mg sodium/day",
				Instructions: "Avoid processed foods, read labels. May consider RD referral. Stricter restriction if volume-overloaded.",
			},
			GoalReferences: []string{"HF-GOAL-004", "HF-GOAL-005"},
		},
		{
			ActivityID:   "HF-ACT-011",
			ActivityType: "lifestyle",
			Description:  "Fluid Restriction (if hyponatremic)",
			Detail: models.ActivityDetail{
				Intervention: "Fluid intake limitation",
				Target:       "1.5-2 L/day if Na<130",
				Instructions: "Individualized based on sodium level and volume status. Not always necessary if euvolemic.",
			},
			GoalReferences: []string{"HF-GOAL-004"},
		},
		{
			ActivityID:   "HF-ACT-012",
			ActivityType: "lifestyle",
			Description:  "Cardiac Rehabilitation",
			Detail: models.ActivityDetail{
				Intervention: "Structured exercise program",
				Target:       "Complete 36-session program",
				Instructions: "Refer all stable HF patients. Improves functional capacity and quality of life. Covered by Medicare.",
			},
			GoalReferences: []string{"HF-GOAL-002"},
		},

		// Monitoring
		{
			ActivityID:   "HF-ACT-020",
			ActivityType: "monitoring",
			Description:  "Daily Weight Monitoring",
			Detail: models.ActivityDetail{
				Intervention: "Daily AM weights",
				Instructions: "Same scale, same time (after voiding, before eating). Call if gain >3 lbs in 1 day or >5 lbs in 1 week.",
			},
			GoalReferences: []string{"HF-GOAL-004", "HF-GOAL-005"},
		},
		{
			ActivityID:   "HF-ACT-021",
			ActivityType: "monitoring",
			Description:  "Symptom Diary",
			Detail: models.ActivityDetail{
				Intervention: "Daily symptom tracking",
				Instructions: "Note dyspnea level, swelling, orthopnea, exercise tolerance. Bring to visits.",
			},
			GoalReferences: []string{"HF-GOAL-002", "HF-GOAL-005"},
		},

		// Education
		{
			ActivityID:   "HF-ACT-030",
			ActivityType: "education",
			Description:  "Heart Failure Self-Care Education",
			Detail: models.ActivityDetail{
				Topic:        "Comprehensive HF education",
				Format:       "Individual or group sessions",
				Instructions: "Teach: daily weights, sodium restriction, medication adherence, symptom recognition, when to call. Consider HF nurse or DSMES.",
			},
			GoalReferences: []string{"HF-GOAL-005"},
		},

		// Appointments
		{
			ActivityID:   "HF-ACT-040",
			ActivityType: "appointment",
			Description:  "Post-Discharge Follow-up",
			Detail: models.ActivityDetail{
				ServiceType:  "Office visit",
				Duration:     "30 minutes",
				Instructions: "Within 7 days of discharge. Assess volume status, titrate medications, reinforce education. Critical for preventing readmission.",
			},
			GoalReferences: []string{"HF-GOAL-003"},
		},
		{
			ActivityID:   "HF-ACT-041",
			ActivityType: "appointment",
			Description:  "Cardiology Follow-up",
			Detail: models.ActivityDetail{
				ServiceType: "Cardiology visit",
				Provider:    "Heart Failure Specialist",
				Duration:    "30 minutes",
				Instructions: "Every 3-6 months when stable. More frequent during GDMT titration or if decompensated.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "quarterly",
				Frequency: 1,
				Interval:  "quarter",
			},
			GoalReferences: []string{"HF-GOAL-001", "HF-GOAL-002"},
		},

		// Advanced Therapies Evaluation
		{
			ActivityID:   "HF-ACT-050",
			ActivityType: "appointment",
			Description:  "Device Therapy Evaluation",
			Detail: models.ActivityDetail{
				ServiceType:  "Electrophysiology consultation",
				Instructions: "Evaluate for ICD (primary prevention if EF≤35%) and CRT (if EF≤35%, LBBB, QRS≥150ms). Recheck EF after GDMT optimization.",
			},
			GoalReferences: []string{"HF-GOAL-001"},
		},
	}

	monitoringItems := []models.MonitoringItem{
		{ItemID: "HF-MON-001", Name: "Daily Weight", Frequency: "daily", AlertRange: ">3 lbs/day or >5 lbs/week gain"},
		{ItemID: "HF-MON-002", Name: "Blood Pressure", Frequency: "every_visit", NormalRange: "Stable, not hypotensive"},
		{ItemID: "HF-MON-003", Name: "BMP (K+, Cr, BUN)", LabCode: "51990-0", Frequency: "every_1-2_weeks_during_titration_then_quarterly", AlertRange: "K+>5.5, Cr increase>30%"},
		{ItemID: "HF-MON-004", Name: "BNP or NT-proBNP", LabCode: "30934-4", Frequency: "at_visits_and_prn", Instructions: "Track trend, not absolute value"},
		{ItemID: "HF-MON-005", Name: "Echocardiogram", Frequency: "baseline_and_annually_or_with_change", Instructions: "Reassess EF 3-6 months after GDMT initiation"},
		{ItemID: "HF-MON-006", Name: "Iron Studies", LabCode: "2498-4", Frequency: "annually", NormalRange: "Ferritin>100, TSAT>20%", Instructions: "Consider IV iron if deficient"},
		{ItemID: "HF-MON-007", Name: "CBC", LabCode: "58410-2", Frequency: "annually", Instructions: "Check for anemia"},
	}

	return &models.CarePlanTemplate{
		PlanID:          "CARE-HF-001",
		Condition:       "Heart Failure with Reduced Ejection Fraction",
		Name:            "Heart Failure (HFrEF) Care Plan",
		GuidelineSource: "ACC/AHA/HFSA Heart Failure Guidelines 2022",
		Goals:           goals,
		Activities:      activities,
		MonitoringItems: monitoringItems,
		Active:          true,
	}
}

// GetCADCarePlan returns Coronary Artery Disease secondary prevention care plan
// Guidelines: ACC/AHA Chronic Coronary Disease Guidelines 2023
func GetCADCarePlan() *models.CarePlanTemplate {
	goals := []models.Goal{
		{
			GoalID:      "CAD-GOAL-001",
			Description: "Prevent cardiovascular events through aggressive risk factor control",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "LDL Cholesterol", Code: "2089-1", TargetValue: "<70 mg/dL (or >50% reduction)"},
				{Measure: "Blood Pressure", TargetValue: "<130/80 mmHg"},
				{Measure: "A1c (if diabetic)", Code: "4548-4", TargetValue: "<7%"},
			},
			Addresses: []models.CodeReference{
				{System: "icd10", Code: "I25.10", Display: "Atherosclerotic heart disease"},
			},
		},
		{
			GoalID:      "CAD-GOAL-002",
			Description: "Maintain optimal antiplatelet and anticoagulation therapy",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Aspirin", TargetValue: "On therapy unless contraindicated"},
				{Measure: "DAPT (post-PCI)", TargetValue: "Per indicated duration"},
			},
		},
		{
			GoalID:      "CAD-GOAL-003",
			Description: "Control anginal symptoms",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Angina Frequency", TargetValue: "None or minimal"},
				{Measure: "Exercise Tolerance", TargetValue: "Able to perform ADLs"},
			},
		},
		{
			GoalID:      "CAD-GOAL-004",
			Description: "Implement heart-healthy lifestyle",
			Category:    "behavioral",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Smoking Status", TargetValue: "Non-smoker"},
				{Measure: "Physical Activity", TargetValue: "≥150 min/week moderate activity"},
				{Measure: "Mediterranean Diet", TargetValue: "Adherent"},
			},
		},
	}

	activities := []models.Activity{
		// Medications
		{
			ActivityID:   "CAD-ACT-001",
			ActivityType: "medication",
			Description:  "Aspirin",
			Detail: models.ActivityDetail{
				DrugCode:     "1191",
				DrugName:     "Aspirin",
				Dose:         "81 mg daily",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "Continue indefinitely unless contraindicated. If GI intolerance, add PPI.",
			},
			GoalReferences: []string{"CAD-GOAL-002"},
		},
		{
			ActivityID:   "CAD-ACT-002",
			ActivityType: "medication",
			Description:  "P2Y12 Inhibitor (if post-PCI)",
			Detail: models.ActivityDetail{
				DrugCode:     "32968",
				DrugName:     "Clopidogrel, Ticagrelor, or Prasugrel",
				Dose:         "Per drug (clopidogrel 75mg, ticagrelor 90mg BID)",
				Route:        "PO",
				Frequency:    "daily or BID",
				Instructions: "DAPT duration: 12 months post-ACS, 6 months post-elective PCI (can shorten to 3 months if high bleeding risk). Consider extended DAPT if tolerated and low bleeding risk.",
			},
			GoalReferences: []string{"CAD-GOAL-002"},
		},
		{
			ActivityID:   "CAD-ACT-003",
			ActivityType: "medication",
			Description:  "High-Intensity Statin",
			Detail: models.ActivityDetail{
				DrugCode:     "617318",
				DrugName:     "Atorvastatin 40-80mg or Rosuvastatin 20-40mg",
				Dose:         "Maximum tolerated dose",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "Reduce LDL by ≥50%. If not at goal on max statin, add ezetimibe. If still not at goal, consider PCSK9 inhibitor.",
			},
			GoalReferences: []string{"CAD-GOAL-001"},
		},
		{
			ActivityID:   "CAD-ACT-004",
			ActivityType: "medication",
			Description:  "Beta-Blocker",
			Detail: models.ActivityDetail{
				DrugCode:     "6918",
				DrugName:     "Metoprolol, Carvedilol, or Bisoprolol",
				Dose:         "Per indication",
				Route:        "PO",
				Frequency:    "daily or BID",
				Instructions: "Indicated if prior MI, LV dysfunction, or angina. Target HR 55-60 for angina.",
			},
			GoalReferences: []string{"CAD-GOAL-003"},
		},
		{
			ActivityID:   "CAD-ACT-005",
			ActivityType: "medication",
			Description:  "ACE Inhibitor or ARB",
			Detail: models.ActivityDetail{
				DrugCode:     "29046",
				DrugName:     "Lisinopril, Ramipril, or equivalent",
				Dose:         "Titrate to target",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "Indicated if HTN, DM, CKD, or EF≤40%. Consider for all CAD patients.",
			},
			GoalReferences: []string{"CAD-GOAL-001"},
		},
		{
			ActivityID:   "CAD-ACT-006",
			ActivityType: "medication",
			Description:  "Nitroglycerin PRN",
			Detail: models.ActivityDetail{
				DrugCode:     "7417",
				DrugName:     "Nitroglycerin SL",
				Dose:         "0.4 mg",
				Route:        "SL",
				PRN:          true,
				PRNReason:    "chest pain",
				Instructions: "Take at onset of chest pain. May repeat x2 at 5 min intervals. If no relief after 3 doses, call 911. Replace every 6 months.",
			},
			GoalReferences: []string{"CAD-GOAL-003"},
		},

		// Lifestyle
		{
			ActivityID:   "CAD-ACT-010",
			ActivityType: "lifestyle",
			Description:  "Cardiac Rehabilitation",
			Detail: models.ActivityDetail{
				Intervention: "Comprehensive cardiac rehab program",
				Target:       "Complete 36 sessions",
				Instructions: "Refer after MI, PCI, CABG, or stable angina. Improves outcomes. Medicare-covered.",
			},
			GoalReferences: []string{"CAD-GOAL-003", "CAD-GOAL-004"},
		},
		{
			ActivityID:   "CAD-ACT-011",
			ActivityType: "lifestyle",
			Description:  "Mediterranean Diet",
			Detail: models.ActivityDetail{
				Intervention: "Heart-healthy dietary pattern",
				Target:       "Mediterranean or DASH pattern",
				Instructions: "Emphasize: olive oil, nuts, fish, fruits, vegetables, whole grains. Limit: red meat, processed foods, saturated fat.",
			},
			GoalReferences: []string{"CAD-GOAL-001", "CAD-GOAL-004"},
		},
		{
			ActivityID:   "CAD-ACT-012",
			ActivityType: "lifestyle",
			Description:  "Smoking Cessation",
			Detail: models.ActivityDetail{
				Intervention: "Complete smoking cessation",
				Target:       "Tobacco-free",
				Instructions: "Most important modifiable risk factor. Offer counseling + pharmacotherapy (NRT, varenicline, bupropion) every visit.",
			},
			GoalReferences: []string{"CAD-GOAL-004"},
		},

		// Appointments
		{
			ActivityID:   "CAD-ACT-020",
			ActivityType: "appointment",
			Description:  "Cardiology Follow-up",
			Detail: models.ActivityDetail{
				ServiceType:  "Cardiology visit",
				Duration:     "20-30 minutes",
				Instructions: "Review symptoms, medications, risk factors. Stress test if symptoms change. Echo if EF concern.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "annually",
				Frequency: 1,
				Interval:  "year",
			},
			GoalReferences: []string{"CAD-GOAL-001", "CAD-GOAL-003"},
		},
	}

	monitoringItems := []models.MonitoringItem{
		{ItemID: "CAD-MON-001", Name: "Lipid Panel", LabCode: "57698-3", Frequency: "annually_or_after_therapy_change", NormalRange: "LDL<70", AlertRange: "LDL>100"},
		{ItemID: "CAD-MON-002", Name: "Blood Pressure", Frequency: "every_visit", NormalRange: "<130/80"},
		{ItemID: "CAD-MON-003", Name: "A1c (if diabetic)", LabCode: "4548-4", Frequency: "quarterly_if_dm"},
		{ItemID: "CAD-MON-004", Name: "Creatinine/eGFR", LabCode: "33914-3", Frequency: "annually"},
		{ItemID: "CAD-MON-005", Name: "Stress Test", Frequency: "as_needed_for_symptoms", Instructions: "Not routine in asymptomatic patients"},
		{ItemID: "CAD-MON-006", Name: "Echo", Frequency: "baseline_and_prn", Instructions: "If concern for LV dysfunction"},
	}

	return &models.CarePlanTemplate{
		PlanID:          "CARE-CAD-001",
		Condition:       "Coronary Artery Disease",
		Name:            "CAD Secondary Prevention Care Plan",
		GuidelineSource: "ACC/AHA Chronic Coronary Disease Guidelines 2023",
		Goals:           goals,
		Activities:      activities,
		MonitoringItems: monitoringItems,
		Active:          true,
	}
}

// GetAFibCarePlan returns Atrial Fibrillation care plan
// Guidelines: ACC/AHA/HRS Atrial Fibrillation Guidelines 2023
func GetAFibCarePlan() *models.CarePlanTemplate {
	goals := []models.Goal{
		{
			GoalID:      "AF-GOAL-001",
			Description: "Prevent stroke with appropriate anticoagulation",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "CHA2DS2-VASc Score", TargetValue: "Calculated and documented"},
				{Measure: "Anticoagulation", TargetValue: "On therapy if score ≥2 (men) or ≥3 (women)"},
				{Measure: "Bleeding Risk", TargetValue: "HAS-BLED assessed"},
			},
			Addresses: []models.CodeReference{
				{System: "icd10", Code: "I48.91", Display: "Unspecified atrial fibrillation"},
			},
		},
		{
			GoalID:      "AF-GOAL-002",
			Description: "Achieve rate control or rhythm control",
			Category:    "clinical",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Resting Heart Rate", TargetValue: "<110 bpm (lenient) or <80 bpm (strict)"},
				{Measure: "Symptoms", TargetValue: "Minimized (EHRA score ≤IIa)"},
			},
		},
		{
			GoalID:      "AF-GOAL-003",
			Description: "Manage modifiable risk factors",
			Category:    "behavioral",
			Priority:    "medium",
			Targets: []models.GoalTarget{
				{Measure: "Blood Pressure", TargetValue: "<130/80 mmHg"},
				{Measure: "BMI", TargetValue: "Reduce by ≥10% if obese"},
				{Measure: "Alcohol", TargetValue: "Minimize or abstain"},
				{Measure: "Sleep Apnea", TargetValue: "Treated if present"},
			},
		},
		{
			GoalID:      "AF-GOAL-004",
			Description: "Patient understands AF management",
			Category:    "educational",
			Priority:    "high",
			Targets: []models.GoalTarget{
				{Measure: "Stroke Prevention", TargetValue: "Understands importance of anticoagulation"},
				{Measure: "Pulse Monitoring", TargetValue: "Can detect irregular pulse"},
			},
		},
	}

	activities := []models.Activity{
		// Anticoagulation
		{
			ActivityID:   "AF-ACT-001",
			ActivityType: "medication",
			Description:  "Direct Oral Anticoagulant (DOAC)",
			Detail: models.ActivityDetail{
				DrugCode:     "1364430", // Apixaban
				DrugName:     "Apixaban, Rivaroxaban, Dabigatran, or Edoxaban",
				Dose:         "Per drug and renal function",
				Route:        "PO",
				Frequency:    "daily or BID per drug",
				Instructions: "DOACs preferred over warfarin for most patients. Adjust dose for age, weight, renal function. Apixaban: 5mg BID (2.5mg BID if 2 of: age≥80, weight≤60kg, Cr≥1.5).",
			},
			GoalReferences: []string{"AF-GOAL-001"},
		},
		{
			ActivityID:   "AF-ACT-002",
			ActivityType: "medication",
			Description:  "Warfarin (if DOAC contraindicated)",
			Detail: models.ActivityDetail{
				DrugCode:     "11289",
				DrugName:     "Warfarin",
				Dose:         "Per INR, target 2.0-3.0",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "Use if mechanical valve, moderate-severe mitral stenosis, or DOAC contraindication. Maintain TTR >70%. Consider anticoag clinic.",
			},
			GoalReferences: []string{"AF-GOAL-001"},
		},

		// Rate Control
		{
			ActivityID:   "AF-ACT-010",
			ActivityType: "medication",
			Description:  "Beta-Blocker for Rate Control",
			Detail: models.ActivityDetail{
				DrugCode:     "6918",
				DrugName:     "Metoprolol or Carvedilol",
				Dose:         "Titrate to HR goal",
				Route:        "PO",
				Frequency:    "daily or BID",
				Instructions: "First-line for rate control. Target resting HR <110 (lenient) or <80 (strict) if symptomatic.",
			},
			GoalReferences: []string{"AF-GOAL-002"},
		},
		{
			ActivityID:   "AF-ACT-011",
			ActivityType: "medication",
			Description:  "Diltiazem or Verapamil (if BB contraindicated)",
			Detail: models.ActivityDetail{
				DrugCode:     "3443",
				DrugName:     "Diltiazem or Verapamil",
				Dose:         "Titrate to HR goal",
				Route:        "PO",
				Frequency:    "daily (extended-release) or TID",
				Instructions: "Alternative if beta-blocker contraindicated. Avoid in HFrEF.",
			},
			GoalReferences: []string{"AF-GOAL-002"},
		},
		{
			ActivityID:   "AF-ACT-012",
			ActivityType: "medication",
			Description:  "Digoxin (adjunctive rate control)",
			Detail: models.ActivityDetail{
				DrugCode:     "3407",
				DrugName:     "Digoxin",
				Dose:         "0.125-0.25 mg daily",
				Route:        "PO",
				Frequency:    "daily",
				Instructions: "Add-on if HR not controlled with BB or CCB. Useful in HF. Target level 0.5-0.9 ng/mL. Reduce dose in renal impairment.",
			},
			GoalReferences: []string{"AF-GOAL-002"},
		},

		// Rhythm Control
		{
			ActivityID:   "AF-ACT-020",
			ActivityType: "medication",
			Description:  "Antiarrhythmic for Rhythm Control (if chosen)",
			Detail: models.ActivityDetail{
				DrugCode:     "703",
				DrugName:     "Amiodarone, Flecainide, Propafenone, Sotalol, or Dofetilide",
				Instructions: "If rhythm control strategy preferred. Selection based on structural heart disease. Flecainide/propafenone only if no CAD/HF. Amiodarone if HF. Dofetilide requires inpatient initiation.",
			},
			GoalReferences: []string{"AF-GOAL-002"},
		},
		{
			ActivityID:   "AF-ACT-021",
			ActivityType: "appointment",
			Description:  "Catheter Ablation Evaluation",
			Detail: models.ActivityDetail{
				ServiceType:  "Electrophysiology consultation",
				Instructions: "Consider referral if: symptomatic despite rate control, young patient, prefer rhythm control, HFrEF. First-line option for some patients.",
			},
			GoalReferences: []string{"AF-GOAL-002"},
		},

		// Lifestyle
		{
			ActivityID:   "AF-ACT-030",
			ActivityType: "lifestyle",
			Description:  "Weight Loss",
			Detail: models.ActivityDetail{
				Intervention: "Weight management program",
				Target:       "≥10% weight loss if obese",
				Instructions: "Weight loss reduces AF burden. May achieve AF remission in some patients. LEGACY trial evidence.",
			},
			GoalReferences: []string{"AF-GOAL-003"},
		},
		{
			ActivityID:   "AF-ACT-031",
			ActivityType: "lifestyle",
			Description:  "Alcohol Reduction",
			Detail: models.ActivityDetail{
				Intervention: "Minimize alcohol consumption",
				Target:       "Abstinence or <1 drink/day",
				Instructions: "Alcohol is a trigger for AF. Binge drinking increases risk of 'holiday heart'.",
			},
			GoalReferences: []string{"AF-GOAL-003"},
		},
		{
			ActivityID:   "AF-ACT-032",
			ActivityType: "appointment",
			Description:  "Sleep Apnea Screening",
			Detail: models.ActivityDetail{
				ServiceType:  "Sleep study",
				Instructions: "Screen all AF patients for OSA. Treat with CPAP if present. Untreated OSA increases AF recurrence.",
			},
			GoalReferences: []string{"AF-GOAL-003"},
		},

		// Education
		{
			ActivityID:   "AF-ACT-040",
			ActivityType: "education",
			Description:  "AF Education and Stroke Prevention",
			Detail: models.ActivityDetail{
				Topic:        "Understanding atrial fibrillation",
				Instructions: "Explain: stroke risk, importance of anticoagulation adherence, symptoms to report, pulse monitoring. Shared decision-making for rate vs rhythm control.",
			},
			GoalReferences: []string{"AF-GOAL-004"},
		},
		{
			ActivityID:   "AF-ACT-041",
			ActivityType: "education",
			Description:  "Pulse Self-Monitoring",
			Detail: models.ActivityDetail{
				Topic:        "Detecting irregular pulse",
				Instructions: "Teach radial pulse check. Consider smartwatch or AliveCor for rhythm monitoring. Report prolonged rapid or irregular pulse.",
			},
			GoalReferences: []string{"AF-GOAL-002", "AF-GOAL-004"},
		},

		// Appointments
		{
			ActivityID:   "AF-ACT-050",
			ActivityType: "appointment",
			Description:  "Cardiology/EP Follow-up",
			Detail: models.ActivityDetail{
				ServiceType:  "Cardiology visit",
				Duration:     "20-30 minutes",
				Instructions: "Every 3-6 months initially, annually when stable. Assess symptoms, HR, anticoag adherence, bleeding. ECG at each visit.",
			},
			Recurrence: &models.RecurrencePattern{
				Type:      "quarterly",
				Frequency: 1,
				Interval:  "quarter",
			},
			GoalReferences: []string{"AF-GOAL-001", "AF-GOAL-002"},
		},
	}

	monitoringItems := []models.MonitoringItem{
		{ItemID: "AF-MON-001", Name: "Heart Rate", Frequency: "every_visit", NormalRange: "<110 resting", AlertRange: ">120 persistent"},
		{ItemID: "AF-MON-002", Name: "ECG", Frequency: "every_visit", Instructions: "Document rhythm, rate, QTc"},
		{ItemID: "AF-MON-003", Name: "Renal Function (CrCl)", LabCode: "33914-3", Frequency: "annually_and_with_doac", Instructions: "Adjust DOAC dose if decline"},
		{ItemID: "AF-MON-004", Name: "INR (if on warfarin)", LabCode: "34714-6", Frequency: "weekly_initially_then_monthly", NormalRange: "2.0-3.0"},
		{ItemID: "AF-MON-005", Name: "CBC", LabCode: "58410-2", Frequency: "annually", Instructions: "Monitor for anemia/bleeding"},
		{ItemID: "AF-MON-006", Name: "Liver Function", LabCode: "24325-3", Frequency: "baseline_and_annually", Instructions: "Baseline before antiarrhythmics"},
		{ItemID: "AF-MON-007", Name: "TSH", LabCode: "3016-3", Frequency: "baseline_and_annually", Instructions: "Hyperthyroidism can cause AF; amiodarone affects thyroid"},
	}

	return &models.CarePlanTemplate{
		PlanID:          "CARE-AF-001",
		Condition:       "Atrial Fibrillation",
		Name:            "Atrial Fibrillation Care Plan",
		GuidelineSource: "ACC/AHA/HRS Atrial Fibrillation Guidelines 2023",
		Goals:           goals,
		Activities:      activities,
		MonitoringItems: monitoringItems,
		Active:          true,
	}
}

// GetCardiovascularCarePlans returns all cardiovascular chronic care plans
func GetCardiovascularCarePlans() []*models.CarePlanTemplate {
	return []*models.CarePlanTemplate{
		GetHypertensionCarePlan(),
		GetHeartFailureCarePlan(),
		GetCADCarePlan(),
		GetAFibCarePlan(),
	}
}
