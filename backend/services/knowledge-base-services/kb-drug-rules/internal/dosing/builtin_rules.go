// Package dosing provides built-in drug dosing rules
// Reference: FDA-approved prescribing information, clinical guidelines
package dosing

import "strings"

// initializeBuiltInRules creates the 24 built-in drug rules as documented in KB-1 README
func initializeBuiltInRules() map[string]*DrugRule {
	rules := make(map[string]*DrugRule)

	// =========================================================================
	// DIABETES MEDICATIONS (4)
	// =========================================================================

	// Metformin - Most prescribed antidiabetic
	rules["6809"] = &DrugRule{
		RxNormCode:       "6809",
		DrugName:         "Metformin",
		TherapeuticClass: "Biguanide Antidiabetic",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     500,
		MinDailyDose:     500,
		MaxDailyDose:     2000,
		MaxSingleDose:    1000,
		DoseUnit:         "mg",
		Frequency:        "BID",
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 45, MaxEGFR: 60, DoseMultiplier: 1.0, MaxDose: 2000, Notes: "Monitor renal function, hold for contrast"},
			{MinEGFR: 30, MaxEGFR: 44, DoseMultiplier: 0.5, MaxDose: 1000, Notes: "Reduce maximum dose to 1000mg/day"},
			{MinEGFR: 0, MaxEGFR: 29, Contraindicated: true, Notes: "Contraindicated - lactic acidosis risk"},
		},
		MonitoringRequired: []string{"HbA1c", "Renal function", "Vitamin B12"},
	}

	// Empagliflozin - SGLT2 Inhibitor
	rules["1545653"] = &DrugRule{
		RxNormCode:       "1545653",
		DrugName:         "Empagliflozin",
		TherapeuticClass: "SGLT2 Inhibitor",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     10,
		MinDailyDose:     10,
		MaxDailyDose:     25,
		MaxSingleDose:    25,
		DoseUnit:         "mg",
		Frequency:        "QD",
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 45, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Full dose for glycemic control"},
			{MinEGFR: 20, MaxEGFR: 44, MaxDose: 10, Notes: "Limited efficacy for glycemic control; continue for cardiorenal benefits"},
			{MinEGFR: 0, MaxEGFR: 19, Contraindicated: true, Notes: "Do not initiate; may continue if already established"},
		},
		MonitoringRequired: []string{"HbA1c", "Renal function", "Blood pressure", "Volume status"},
	}

	// Liraglutide - GLP-1 Receptor Agonist
	rules["475968"] = &DrugRule{
		RxNormCode:        "475968",
		DrugName:          "Liraglutide",
		TherapeuticClass:  "GLP-1 Receptor Agonist",
		DosingMethod:      DosingMethodTitration,
		StartingDose:      0.6,
		MinDailyDose:      0.6,
		MaxDailyDose:      1.8,
		MaxSingleDose:     1.8,
		DoseUnit:          "mg",
		Frequency:         "QD",
		HasBlackBoxWarning: true, // Thyroid C-cell tumors
		TitrationSteps: []TitrationStep{
			{Step: 1, AfterDays: 0, TargetDose: 0.6, Monitoring: "GI tolerance"},
			{Step: 2, AfterDays: 7, TargetDose: 1.2, Monitoring: "GI tolerance, blood glucose"},
			{Step: 3, AfterDays: 14, TargetDose: 1.8, Monitoring: "Blood glucose, HbA1c"},
		},
		MonitoringRequired: []string{"HbA1c", "Thyroid function (history of MTC)", "GI symptoms"},
	}

	// Insulin Glargine - Long-Acting Insulin
	rules["261551"] = &DrugRule{
		RxNormCode:       "261551",
		DrugName:         "Insulin Glargine",
		TherapeuticClass: "Long-Acting Insulin",
		DosingMethod:     DosingMethodWeightBased,
		DosePerKg:        0.2, // Starting dose 0.1-0.2 units/kg
		MinDailyDose:     10,
		MaxDailyDose:     100,
		DoseUnit:         "units",
		Frequency:        "QD",
		IsHighAlert:      true,
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 30, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Monitor for hypoglycemia"},
			{MinEGFR: 15, MaxEGFR: 29, DoseMultiplier: 0.75, Notes: "Reduced clearance - reduce dose 25%"},
			{MinEGFR: 0, MaxEGFR: 14, DoseMultiplier: 0.5, Notes: "ESRD - reduce dose 50%, monitor closely"},
		},
		MonitoringRequired: []string{"Blood glucose", "HbA1c", "Hypoglycemia symptoms"},
	}

	// =========================================================================
	// CARDIOVASCULAR MEDICATIONS (7)
	// =========================================================================

	// Lisinopril - ACE Inhibitor
	rules["8610"] = &DrugRule{
		RxNormCode:       "8610",
		DrugName:         "Lisinopril",
		TherapeuticClass: "ACE Inhibitor",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     10,
		MinDailyDose:     2.5,
		MaxDailyDose:     40,
		MaxSingleDose:    40,
		DoseUnit:         "mg",
		Frequency:        "QD",
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 30, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Standard dosing"},
			{MinEGFR: 10, MaxEGFR: 29, MaxDose: 20, Notes: "Reduce maximum dose"},
			{MinEGFR: 0, MaxEGFR: 9, MaxDose: 5, Notes: "Use lowest effective dose"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 65, MaxAge: 150, DoseMultiplier: 0.8, MaxDose: 30, Notes: "Start lower in elderly"},
		},
		TitrationSteps: []TitrationStep{
			{Step: 1, AfterDays: 7, IncreaseBy: 5, Monitoring: "BP, creatinine, potassium"},
			{Step: 2, AfterDays: 14, IncreaseBy: 5, Monitoring: "BP, creatinine, potassium"},
			{Step: 3, AfterDays: 21, IncreaseBy: 10, Monitoring: "BP, creatinine, potassium"},
		},
		MonitoringRequired: []string{"Blood pressure", "Serum creatinine", "Potassium"},
	}

	// Losartan - ARB
	rules["52175"] = &DrugRule{
		RxNormCode:       "52175",
		DrugName:         "Losartan",
		TherapeuticClass: "Angiotensin II Receptor Blocker",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     50,
		MinDailyDose:     25,
		MaxDailyDose:     100,
		MaxSingleDose:    100,
		DoseUnit:         "mg",
		Frequency:        "QD",
		HepaticAdjustments: []HepaticAdjustment{
			{ChildPughClass: "A", DoseMultiplier: 1.0, Notes: "No adjustment needed"},
			{ChildPughClass: "B", DoseMultiplier: 0.5, MaxDose: 50, Notes: "Start at 25mg"},
			{ChildPughClass: "C", DoseMultiplier: 0.5, MaxDose: 50, Notes: "Start at 25mg, use with caution"},
		},
		MonitoringRequired: []string{"Blood pressure", "Serum creatinine", "Potassium"},
	}

	// Metoprolol Succinate - Beta Blocker
	rules["866924"] = &DrugRule{
		RxNormCode:       "866924",
		DrugName:         "Metoprolol Succinate",
		TherapeuticClass: "Beta-1 Selective Blocker",
		DosingMethod:     DosingMethodTitration,
		StartingDose:     25,
		MinDailyDose:     12.5,
		MaxDailyDose:     400,
		MaxSingleDose:    200,
		DoseUnit:         "mg",
		Frequency:        "QD",
		TitrationSteps: []TitrationStep{
			{Step: 1, AfterDays: 14, TargetDose: 50, Monitoring: "HR, BP, HF symptoms"},
			{Step: 2, AfterDays: 28, TargetDose: 100, Monitoring: "HR, BP, HF symptoms"},
			{Step: 3, AfterDays: 42, TargetDose: 200, Monitoring: "HR, BP, HF symptoms"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 65, MaxAge: 150, DoseMultiplier: 0.5, Notes: "Start at lower dose, titrate slowly"},
		},
		MonitoringRequired: []string{"Heart rate", "Blood pressure", "Signs of decompensation"},
	}

	// Carvedilol - Non-Selective Beta Blocker with Alpha-1 Block
	rules["20352"] = &DrugRule{
		RxNormCode:       "20352",
		DrugName:         "Carvedilol",
		TherapeuticClass: "Non-Selective Beta/Alpha-1 Blocker",
		DosingMethod:     DosingMethodTitration,
		StartingDose:     6.25,
		MinDailyDose:     6.25,
		MaxDailyDose:     100, // 50mg BID for severe HF
		MaxSingleDose:    50,
		DoseUnit:         "mg",
		Frequency:        "BID",
		TitrationSteps: []TitrationStep{
			{Step: 1, AfterDays: 14, TargetDose: 12.5, Monitoring: "Take with food, monitor for dizziness"},
			{Step: 2, AfterDays: 28, TargetDose: 25, Monitoring: "HR, BP, weight, HF symptoms"},
			{Step: 3, AfterDays: 42, TargetDose: 50, Monitoring: "HR, BP, weight, HF symptoms"},
		},
		HepaticAdjustments: []HepaticAdjustment{
			{ChildPughClass: "C", Contraindicated: true, Notes: "Contraindicated in severe hepatic impairment"},
		},
		MonitoringRequired: []string{"Heart rate", "Blood pressure", "Weight", "Signs of decompensation"},
	}

	// Atorvastatin - HMG-CoA Reductase Inhibitor (Statin)
	rules["83367"] = &DrugRule{
		RxNormCode:       "83367",
		DrugName:         "Atorvastatin",
		TherapeuticClass: "HMG-CoA Reductase Inhibitor",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     20,
		MinDailyDose:     10,
		MaxDailyDose:     80,
		MaxSingleDose:    80,
		DoseUnit:         "mg",
		Frequency:        "QD",
		HepaticAdjustments: []HepaticAdjustment{
			{ChildPughClass: "A", DoseMultiplier: 1.0, Notes: "Use with caution, monitor LFTs"},
			{ChildPughClass: "B", Contraindicated: true, Notes: "Contraindicated"},
			{ChildPughClass: "C", Contraindicated: true, Notes: "Contraindicated"},
		},
		MonitoringRequired: []string{"LDL-C", "Total cholesterol", "LFTs", "CK if symptomatic"},
	}

	// Furosemide - Loop Diuretic
	rules["4603"] = &DrugRule{
		RxNormCode:       "4603",
		DrugName:         "Furosemide",
		TherapeuticClass: "Loop Diuretic",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     40,
		MinDailyDose:     20,
		MaxDailyDose:     600,
		MaxSingleDose:    200,
		DoseUnit:         "mg",
		Frequency:        "QD-BID",
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 60, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Standard dosing"},
			{MinEGFR: 30, MaxEGFR: 59, DoseMultiplier: 1.5, Notes: "Higher doses may be needed"},
			{MinEGFR: 15, MaxEGFR: 29, DoseMultiplier: 2.0, Notes: "Much higher doses often required"},
			{MinEGFR: 0, MaxEGFR: 14, Notes: "IV preferred, very high doses may be needed"},
		},
		MonitoringRequired: []string{"Electrolytes", "Renal function", "Weight", "Urine output"},
	}

	// Spironolactone - Aldosterone Antagonist
	rules["9997"] = &DrugRule{
		RxNormCode:       "9997",
		DrugName:         "Spironolactone",
		TherapeuticClass: "Aldosterone Antagonist",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     25,
		MinDailyDose:     12.5,
		MaxDailyDose:     400,
		MaxSingleDose:    100,
		DoseUnit:         "mg",
		Frequency:        "QD",
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 50, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Monitor potassium"},
			{MinEGFR: 30, MaxEGFR: 49, MaxDose: 25, Notes: "Use with caution, monitor potassium closely"},
			{MinEGFR: 0, MaxEGFR: 29, Contraindicated: true, Notes: "Avoid - hyperkalemia risk"},
		},
		MonitoringRequired: []string{"Potassium", "Renal function", "Blood pressure"},
	}

	// =========================================================================
	// ANTICOAGULANTS (4)
	// =========================================================================

	// Warfarin - Vitamin K Antagonist
	rules["11289"] = &DrugRule{
		RxNormCode:       "11289",
		DrugName:         "Warfarin",
		TherapeuticClass: "Vitamin K Antagonist",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     5, // Highly variable
		MinDailyDose:     1,
		MaxDailyDose:     15,
		MaxSingleDose:    10,
		DoseUnit:         "mg",
		Frequency:        "QD",
		IsHighAlert:      true,
		IsNarrowTI:       true,
		HepaticAdjustments: []HepaticAdjustment{
			{ChildPughClass: "A", DoseMultiplier: 0.75, Notes: "Reduce dose, monitor INR closely"},
			{ChildPughClass: "B", DoseMultiplier: 0.5, Notes: "Significant dose reduction needed"},
			{ChildPughClass: "C", DoseMultiplier: 0.25, Notes: "Very low doses, monitor closely"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 65, MaxAge: 150, DoseMultiplier: 0.8, Notes: "Elderly often require lower doses"},
		},
		MonitoringRequired: []string{"INR (target 2.0-3.0 typically)", "Signs of bleeding", "Diet changes"},
	}

	// Apixaban - Direct Xa Inhibitor
	rules["1364430"] = &DrugRule{
		RxNormCode:       "1364430",
		DrugName:         "Apixaban",
		TherapeuticClass: "Direct Factor Xa Inhibitor",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     5,
		MinDailyDose:     2.5,
		MaxDailyDose:     10, // 5mg BID
		MaxSingleDose:    5,
		DoseUnit:         "mg",
		Frequency:        "BID",
		IsHighAlert:      true,
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 50, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Standard dose 5mg BID"},
			{MinEGFR: 25, MaxEGFR: 49, DoseMultiplier: 1.0, Notes: "Consider 2.5mg BID if age ≥80, weight ≤60kg, or SCr ≥1.5"},
			{MinEGFR: 15, MaxEGFR: 24, MaxDose: 5, Notes: "2.5mg BID recommended"},
			{MinEGFR: 0, MaxEGFR: 14, Notes: "Limited data, use with caution or avoid"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 80, MaxAge: 150, MaxDose: 5, Notes: "2.5mg BID if meets dose reduction criteria"},
		},
		MonitoringRequired: []string{"Renal function", "Signs of bleeding", "Hemoglobin"},
	}

	// Enoxaparin - Low Molecular Weight Heparin
	rules["67108"] = &DrugRule{
		RxNormCode:       "67108",
		DrugName:         "Enoxaparin",
		TherapeuticClass: "Low Molecular Weight Heparin",
		DosingMethod:     DosingMethodWeightBased,
		DosePerKg:        1.0, // 1 mg/kg for treatment
		MinDailyDose:     30,
		MaxDailyDose:     300,
		MaxSingleDose:    150,
		DoseUnit:         "mg",
		Frequency:        "Q12H",
		IsHighAlert:      true,
		UseIdealWeight:   false, // Use actual weight, cap at 150kg
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 30, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Standard dosing"},
			{MinEGFR: 0, MaxEGFR: 29, DoseMultiplier: 0.5, FrequencyChange: "Q24H", Notes: "1mg/kg once daily for CrCl <30"},
		},
		MonitoringRequired: []string{"Anti-Xa levels (if indicated)", "CBC", "Signs of bleeding", "Renal function"},
	}

	// Heparin (Unfractionated)
	rules["5224"] = &DrugRule{
		RxNormCode:       "5224",
		DrugName:         "Heparin",
		TherapeuticClass: "Unfractionated Heparin",
		DosingMethod:     DosingMethodWeightBased,
		DosePerKg:        80,  // 80 units/kg bolus
		MaxDailyDose:     50000,
		MaxSingleDose:    10000,
		DoseUnit:         "units",
		Frequency:        "Continuous infusion",
		IsHighAlert:      true,
		IsNarrowTI:       true,
		UseIdealWeight:   true, // Use actual weight for dosing
		MonitoringRequired: []string{"aPTT (target 1.5-2.5x control)", "Platelet count", "Signs of bleeding", "HIT screening"},
	}

	// =========================================================================
	// ANTIBIOTICS (4)
	// =========================================================================

	// Amoxicillin - Penicillin Antibiotic
	rules["723"] = &DrugRule{
		RxNormCode:       "723",
		DrugName:         "Amoxicillin",
		TherapeuticClass: "Aminopenicillin",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     500,
		MinDailyDose:     750,
		MaxDailyDose:     3000,
		MaxSingleDose:    1000,
		DoseUnit:         "mg",
		Frequency:        "TID",
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 30, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Standard dosing"},
			{MinEGFR: 10, MaxEGFR: 29, FrequencyChange: "Q12H", Notes: "Extend interval"},
			{MinEGFR: 0, MaxEGFR: 9, FrequencyChange: "Q24H", Notes: "Once daily dosing"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 0, MaxAge: 17, Notes: "Pediatric: 25-50 mg/kg/day divided TID"},
		},
	}

	// Vancomycin - Glycopeptide Antibiotic
	rules["11124"] = &DrugRule{
		RxNormCode:       "11124",
		DrugName:         "Vancomycin",
		TherapeuticClass: "Glycopeptide Antibiotic",
		DosingMethod:     DosingMethodWeightBased,
		DosePerKg:        15, // 15-20 mg/kg per dose
		MinDailyDose:     1000,
		MaxDailyDose:     4000,
		MaxSingleDose:    2000,
		DoseUnit:         "mg",
		Frequency:        "Q8-12H",
		IsNarrowTI:       true,
		UseIdealWeight:   true, // Use actual weight
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 90, MaxEGFR: 200, DoseMultiplier: 1.0, FrequencyChange: "Q8H", Notes: "Standard dosing"},
			{MinEGFR: 60, MaxEGFR: 89, DoseMultiplier: 1.0, FrequencyChange: "Q12H", Notes: "Standard dose, extended interval"},
			{MinEGFR: 30, MaxEGFR: 59, DoseMultiplier: 0.75, FrequencyChange: "Q12H", Notes: "Reduce dose"},
			{MinEGFR: 15, MaxEGFR: 29, DoseMultiplier: 0.5, FrequencyChange: "Q24H", Notes: "Once daily"},
			{MinEGFR: 0, MaxEGFR: 14, Notes: "Level-guided dosing, typically Q48-72H"},
		},
		MonitoringRequired: []string{"Trough levels (10-20 mg/L)", "Renal function", "Ototoxicity", "CBC"},
	}

	// Gentamicin - Aminoglycoside
	rules["3058"] = &DrugRule{
		RxNormCode:       "3058",
		DrugName:         "Gentamicin",
		TherapeuticClass: "Aminoglycoside",
		DosingMethod:     DosingMethodWeightBased,
		DosePerKg:        5, // Extended interval: 5-7 mg/kg Q24H
		MinDailyDose:     80,
		MaxDailyDose:     560,
		MaxSingleDose:    560,
		DoseUnit:         "mg",
		Frequency:        "Q24H", // Extended interval dosing
		IsHighAlert:      true,
		IsNarrowTI:       true,
		UseIdealWeight:   true, // Use IBW or AdjBW for obese
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 60, MaxEGFR: 200, DoseMultiplier: 1.0, FrequencyChange: "Q24H", Notes: "Hartford nomogram"},
			{MinEGFR: 40, MaxEGFR: 59, FrequencyChange: "Q36H", Notes: "Extended interval"},
			{MinEGFR: 20, MaxEGFR: 39, FrequencyChange: "Q48H", Notes: "Further extend interval"},
			{MinEGFR: 0, MaxEGFR: 19, Notes: "Level-guided dosing required"},
		},
		MonitoringRequired: []string{"Peak and trough levels", "Renal function", "Ototoxicity assessment", "CBC"},
	}

	// Ciprofloxacin - Fluoroquinolone
	rules["2551"] = &DrugRule{
		RxNormCode:        "2551",
		DrugName:          "Ciprofloxacin",
		TherapeuticClass:  "Fluoroquinolone",
		DosingMethod:      DosingMethodFixed,
		StartingDose:      500,
		MinDailyDose:      500,
		MaxDailyDose:      1500,
		MaxSingleDose:     750,
		DoseUnit:          "mg",
		Frequency:         "BID",
		HasBlackBoxWarning: true, // Tendon rupture, peripheral neuropathy
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 50, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Standard dosing"},
			{MinEGFR: 30, MaxEGFR: 49, MaxDose: 1000, Notes: "250-500mg BID"},
			{MinEGFR: 5, MaxEGFR: 29, MaxDose: 500, Notes: "250-500mg Q18H"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 65, MaxAge: 150, Notes: "Increased risk of tendon rupture - use alternatives if possible"},
		},
		MonitoringRequired: []string{"Signs of tendinitis", "QT interval if risk factors", "Blood glucose in diabetics"},
	}

	// =========================================================================
	// PAIN MEDICATIONS (4)
	// =========================================================================

	// Acetaminophen
	rules["161"] = &DrugRule{
		RxNormCode:       "161",
		DrugName:         "Acetaminophen",
		TherapeuticClass: "Analgesic/Antipyretic",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     650,
		MinDailyDose:     325,
		MaxDailyDose:     4000, // 3000 in elderly or liver disease
		MaxSingleDose:    1000,
		DoseUnit:         "mg",
		Frequency:        "Q4-6H PRN",
		HepaticAdjustments: []HepaticAdjustment{
			{ChildPughClass: "A", MaxDose: 3000, Notes: "Reduce maximum to 3g/day"},
			{ChildPughClass: "B", MaxDose: 2000, Notes: "Maximum 2g/day"},
			{ChildPughClass: "C", Contraindicated: true, Notes: "Avoid or use minimal doses"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 65, MaxAge: 150, MaxDose: 3000, Notes: "Maximum 3g/day in elderly"},
		},
		MonitoringRequired: []string{"Total daily dose from all sources", "LFTs if chronic use"},
	}

	// Ibuprofen - NSAID
	rules["5640"] = &DrugRule{
		RxNormCode:       "5640",
		DrugName:         "Ibuprofen",
		TherapeuticClass: "NSAID",
		DosingMethod:     DosingMethodFixed,
		StartingDose:     400,
		MinDailyDose:     200,
		MaxDailyDose:     3200,
		MaxSingleDose:    800,
		DoseUnit:         "mg",
		Frequency:        "Q6-8H PRN",
		BeersListStatus:  "use_with_caution",
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 60, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Use lowest effective dose for shortest duration"},
			{MinEGFR: 30, MaxEGFR: 59, MaxDose: 1600, Notes: "Reduce maximum dose, monitor renal function"},
			{MinEGFR: 0, MaxEGFR: 29, Contraindicated: true, Notes: "Avoid - significant renal injury risk"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 65, MaxAge: 150, MaxDose: 1200, Notes: "Beers Criteria - increased GI/renal/CV risk"},
		},
		MonitoringRequired: []string{"Renal function", "GI symptoms", "Blood pressure", "Signs of bleeding"},
	}

	// Morphine Sulfate - Opioid
	rules["7052"] = &DrugRule{
		RxNormCode:        "7052",
		DrugName:          "Morphine Sulfate",
		TherapeuticClass:  "Opioid Analgesic",
		DosingMethod:      DosingMethodFixed,
		StartingDose:      15, // Oral immediate release
		MinDailyDose:      15,
		MaxDailyDose:      200, // Variable based on tolerance
		MaxSingleDose:     30,
		DoseUnit:          "mg",
		Frequency:         "Q4H PRN",
		IsHighAlert:       true,
		HasBlackBoxWarning: true, // Addiction, respiratory depression, opioid crisis
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 50, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Standard dosing with monitoring"},
			{MinEGFR: 30, MaxEGFR: 49, DoseMultiplier: 0.75, Notes: "Reduce dose 25%, extend interval"},
			{MinEGFR: 15, MaxEGFR: 29, DoseMultiplier: 0.5, Notes: "Reduce dose 50%, monitor closely"},
			{MinEGFR: 0, MaxEGFR: 14, DoseMultiplier: 0.25, Notes: "Avoid if possible - M6G accumulation"},
		},
		HepaticAdjustments: []HepaticAdjustment{
			{ChildPughClass: "A", DoseMultiplier: 0.75, Notes: "Reduce dose by 25%"},
			{ChildPughClass: "B", DoseMultiplier: 0.5, Notes: "Reduce dose by 50%"},
			{ChildPughClass: "C", DoseMultiplier: 0.25, Notes: "Use with extreme caution"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 65, MaxAge: 150, DoseMultiplier: 0.5, Notes: "Start at 50% dose in elderly"},
		},
		MonitoringRequired: []string{"Pain level", "Respiratory rate", "Sedation level", "Bowel function", "Signs of misuse"},
	}

	// Oxycodone - Opioid
	rules["7804"] = &DrugRule{
		RxNormCode:        "7804",
		DrugName:          "Oxycodone",
		TherapeuticClass:  "Opioid Analgesic",
		DosingMethod:      DosingMethodFixed,
		StartingDose:      5,
		MinDailyDose:      5,
		MaxDailyDose:      160, // Variable based on tolerance
		MaxSingleDose:     20,
		DoseUnit:          "mg",
		Frequency:         "Q4-6H PRN",
		IsHighAlert:       true,
		HasBlackBoxWarning: true, // Addiction, respiratory depression, opioid crisis
		RenalAdjustments: []RenalAdjustment{
			{MinEGFR: 60, MaxEGFR: 200, DoseMultiplier: 1.0, Notes: "Standard dosing"},
			{MinEGFR: 30, MaxEGFR: 59, DoseMultiplier: 0.75, Notes: "Start at lower dose"},
			{MinEGFR: 0, MaxEGFR: 29, DoseMultiplier: 0.5, Notes: "Reduce dose 50%"},
		},
		HepaticAdjustments: []HepaticAdjustment{
			{ChildPughClass: "A", DoseMultiplier: 0.67, Notes: "Start at 1/3 to 1/2 usual dose"},
			{ChildPughClass: "B", DoseMultiplier: 0.5, Notes: "Start at 1/3 to 1/2 usual dose"},
			{ChildPughClass: "C", DoseMultiplier: 0.33, Notes: "Use with extreme caution"},
		},
		AgeAdjustments: []AgeAdjustment{
			{MinAge: 65, MaxAge: 150, DoseMultiplier: 0.5, Notes: "Start at 50% dose in elderly"},
		},
		MonitoringRequired: []string{"Pain level", "Respiratory rate", "Sedation level", "Bowel function", "Signs of misuse"},
	}

	return rules
}

// GetRxNormCodeByName returns the RxNorm code for a drug name (case-insensitive)
func GetRxNormCodeByName(name string) string {
	nameToCode := map[string]string{
		// Diabetes
		"metformin":      "6809",
		"empagliflozin":  "1545653",
		"liraglutide":    "475968",
		"insulin glargine": "261551",

		// Cardiovascular
		"lisinopril":     "8610",
		"losartan":       "52175",
		"metoprolol":     "866924",
		"carvedilol":     "20352",
		"atorvastatin":   "83367",
		"furosemide":     "4603",
		"spironolactone": "9997",

		// Anticoagulants
		"warfarin":       "11289",
		"apixaban":       "1364430",
		"enoxaparin":     "67108",
		"heparin":        "5224",

		// Antibiotics
		"amoxicillin":    "723",
		"vancomycin":     "11124",
		"gentamicin":     "3058",
		"ciprofloxacin":  "2551",

		// Pain
		"acetaminophen":  "161",
		"ibuprofen":      "5640",
		"morphine":       "7052",
		"oxycodone":      "7804",
	}

	return nameToCode[strings.ToLower(name)]
}
