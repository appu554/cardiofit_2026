// Package ordersets provides procedure order set templates for GI procedures
package ordersets

import (
	"kb-12-ordersets-careplans/internal/models"
)

// GetColonoscopyPrepOrderSet returns colonoscopy preparation order set
// Guidelines: ASGE Bowel Preparation Guidelines, ACG Clinical Guidelines
func GetColonoscopyPrepOrderSet() *models.OrderSetTemplate {
	orders := []models.Order{
		// Pre-procedure Instructions (Day Before)
		{
			OrderID:      "COLON-PREP-001",
			OrderType:    models.OrderTypeDiet,
			Category:     "diet_preparation",
			Name:         "Clear Liquid Diet",
			Description:  "Day before colonoscopy diet",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Clear liquids only starting morning before procedure. No red, orange, or purple liquids. Include: water, clear broth, apple juice, white grape juice, Gatorade (yellow/green), black coffee/tea, Jello (not red/purple).",
		},
		{
			OrderID:      "COLON-PREP-002",
			OrderType:    models.OrderTypeNursing,
			Category:     "diet_preparation",
			Name:         "NPO After Midnight",
			Description:  "Nothing by mouth before procedure",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "NPO after midnight. Clear liquids may be taken with prep medications up to 4 hours before procedure.",
		},

		// Bowel Preparation Options
		{
			OrderID:      "COLON-PREP-003",
			OrderType:    models.OrderTypeMedication,
			Category:     "bowel_preparation",
			Name:         "GoLYTELY (PEG-ELS) Split Dose",
			Description:  "Polyethylene glycol electrolyte solution - split dose preferred",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			DrugCode:     "142442",
			DrugName:     "Polyethylene Glycol 3350 with Electrolytes",
			Dose:         "4 liters total",
			Route:        "PO",
			Instructions: "Split-dose: 2L evening before + 2L morning of procedure (complete 4-6 hours before). Drink 8oz every 10-15 minutes. Continue until stools clear.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "142442", Display: "Polyethylene Glycol 3350/Electrolytes"},
			},
			Notes: "Split-dose regimen improves bowel preparation quality. Morning dose must complete ≥4 hours before procedure.",
		},
		{
			OrderID:      "COLON-PREP-004",
			OrderType:    models.OrderTypeMedication,
			Category:     "bowel_preparation",
			Name:         "MiraLAX + Gatorade Alternative",
			Description:  "Low-volume prep alternative for patient preference",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			DrugCode:     "616675",
			DrugName:     "Polyethylene Glycol 3350",
			Dose:         "238g (bottle)",
			Route:        "PO",
			Instructions: "Mix entire bottle (238g) MiraLAX in 64oz Gatorade (not red/purple). Drink 8oz every 15 min evening before. Take 4 Dulcolax tablets before starting MiraLAX.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "616675", Display: "Polyethylene Glycol 3350"},
			},
			Notes: "Off-label but commonly used. Good for patients who cannot tolerate large volume prep.",
		},
		{
			OrderID:      "COLON-PREP-005",
			OrderType:    models.OrderTypeMedication,
			Category:     "bowel_preparation",
			Name:         "SUPREP (Sodium Sulfate/Potassium/Magnesium)",
			Description:  "Low-volume split-dose prep",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			DrugCode:     "994248",
			DrugName:     "Sodium Sulfate/Potassium Sulfate/Magnesium Sulfate",
			Dose:         "2 bottles",
			Route:        "PO",
			Instructions: "Evening before: Mix bottle 1 with 16oz water, drink, then 32oz water over 1 hour. Morning of: Repeat with bottle 2 (≥5 hours before procedure).",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "994248", Display: "SUPREP"},
			},
			Conditions: []models.OrderCondition{
				{Type: "unless", Field: "renal_impairment", Operator: "eq", Value: "true"},
				{Type: "unless", Field: "heart_failure", Operator: "eq", Value: "true"},
			},
			Notes: "Lower volume (32oz vs 4L). Avoid in CKD, CHF, or electrolyte abnormalities.",
		},

		// Adjunctive Medications
		{
			OrderID:      "COLON-PREP-006",
			OrderType:    models.OrderTypeMedication,
			Category:     "adjunctive",
			Name:         "Ondansetron for Nausea",
			Description:  "Antiemetic to prevent prep-induced nausea",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     true,
			DrugCode:     "7847",
			DrugName:     "Ondansetron",
			Dose:         "4-8 mg",
			DoseValue:    4,
			DoseUnit:     "mg",
			Route:        "PO",
			Frequency:    "PRN",
			PRN:          true,
			PRNReason:    "nausea during prep",
			Instructions: "May take 30 minutes before starting prep or PRN nausea. Max 24mg/day.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "7847", Display: "Ondansetron"},
			},
		},
		{
			OrderID:      "COLON-PREP-007",
			OrderType:    models.OrderTypeMedication,
			Category:     "adjunctive",
			Name:         "Simethicone",
			Description:  "Anti-foaming agent to reduce bubbles",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     true,
			DrugCode:     "9679",
			DrugName:     "Simethicone",
			Dose:         "80-160 mg",
			DoseValue:    80,
			DoseUnit:     "mg",
			Route:        "PO",
			Instructions: "Take with final dose of prep to reduce bubbles during procedure.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "9679", Display: "Simethicone"},
			},
		},

		// Medication Management
		{
			OrderID:      "COLON-MED-001",
			OrderType:    models.OrderTypeNursing,
			Category:     "medication_management",
			Name:         "Hold Anticoagulants",
			Description:  "Anticoagulation management for polypectomy",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			Instructions: "If polypectomy planned: Hold warfarin 5 days (bridge if high risk), DOACs 48-72h, aspirin may continue unless high-risk polypectomy planned.",
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "anticoagulant_use", Operator: "eq", Value: "true"},
			},
		},
		{
			OrderID:      "COLON-MED-002",
			OrderType:    models.OrderTypeNursing,
			Category:     "medication_management",
			Name:         "Hold Iron Supplements",
			Description:  "Iron can cause dark stool and obscure visualization",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Stop iron supplements 5-7 days before colonoscopy.",
		},
		{
			OrderID:      "COLON-MED-003",
			OrderType:    models.OrderTypeNursing,
			Category:     "medication_management",
			Name:         "Diabetes Medication Management",
			Description:  "Adjust diabetes medications during prep",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			Instructions: "Day before: Half dose of long-acting insulin. Day of: Hold all oral diabetes meds and insulin until after procedure. Monitor glucose q4h.",
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "diabetes", Operator: "eq", Value: "true"},
			},
		},

		// Pre-procedure Labs (if needed)
		{
			OrderID:      "COLON-LAB-001",
			OrderType:    models.OrderTypeLab,
			Category:     "pre_procedure_labs",
			Name:         "INR Check",
			Description:  "Coagulation check if on warfarin",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			LabCode:      "34714-6",
			Specimen:     "blood",
			Timing:       "day_before",
			Codes: []models.CodeReference{
				{System: "loinc", Code: "34714-6", Display: "INR"},
			},
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "warfarin_use", Operator: "eq", Value: "true"},
			},
			Notes: "Goal INR <1.5 for therapeutic polypectomy.",
		},

		// Day of Procedure
		{
			OrderID:      "COLON-DAY-001",
			OrderType:    models.OrderTypeNursing,
			Category:     "day_of_procedure",
			Name:         "Procedure Day Instructions",
			Description:  "Morning of colonoscopy checklist",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "1) Complete morning prep dose 4-6h before. 2) Take essential medications with sips of water. 3) Arrange driver. 4) Leave jewelry at home. 5) Wear comfortable clothing.",
		},
		{
			OrderID:      "COLON-DAY-002",
			OrderType:    models.OrderTypeNursing,
			Category:     "day_of_procedure",
			Name:         "Escort Requirement",
			Description:  "Must have responsible adult for discharge",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Patient must have responsible adult to drive home. Cannot take taxi/Uber alone. No driving or operating machinery for 24 hours after sedation.",
		},

		// Sedation Orders
		{
			OrderID:      "COLON-SED-001",
			OrderType:    models.OrderTypeMedication,
			Category:     "sedation",
			Name:         "Moderate Sedation Protocol",
			Description:  "Standard colonoscopy sedation",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Fentanyl 50-100mcg IV + Midazolam 1-2mg IV. Titrate to moderate sedation. Monitor SpO2, BP, HR continuously.",
		},
		{
			OrderID:      "COLON-SED-002",
			OrderType:    models.OrderTypeMedication,
			Category:     "sedation",
			Name:         "Propofol Sedation (MAC)",
			Description:  "Deep sedation by anesthesia",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			DrugCode:     "8782",
			DrugName:     "Propofol",
			Instructions: "Administer by qualified anesthesia provider. For patients with sedation failure, complex procedures, or patient/physician preference.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "8782", Display: "Propofol"},
			},
		},

		// Recovery Orders
		{
			OrderID:      "COLON-REC-001",
			OrderType:    models.OrderTypeMonitoring,
			Category:     "recovery",
			Name:         "Post-Procedure Monitoring",
			Description:  "Recovery room observation",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Monitor VS q15min until stable. Assess level of consciousness, pain, nausea. Discharge when Aldrete score ≥9 and ambulating.",
		},
		{
			OrderID:      "COLON-REC-002",
			OrderType:    models.OrderTypeDiet,
			Category:     "recovery",
			Name:         "Post-Procedure Diet",
			Description:  "Resume oral intake",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Resume regular diet as tolerated. Start with clear liquids, advance as tolerated. Avoid alcohol for 24 hours.",
		},
		{
			OrderID:      "COLON-REC-003",
			OrderType:    models.OrderTypeNursing,
			Category:     "recovery",
			Name:         "Post-Polypectomy Instructions",
			Description:  "Care after polyp removal",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			Instructions: "Avoid aspirin/NSAIDs for 7 days. Watch for: severe abdominal pain, blood in stool >1 tablespoon, fever >101°F. Call if any concerns.",
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "polypectomy_performed", Operator: "eq", Value: "true"},
			},
		},

		// Discharge Instructions
		{
			OrderID:      "COLON-DC-001",
			OrderType:    models.OrderTypeEducation,
			Category:     "discharge",
			Name:         "Discharge Instructions",
			Description:  "Post-colonoscopy patient education",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "1) No driving for 24h. 2) Resume normal diet. 3) Mild bloating/cramping normal. 4) Follow up for pathology results. 5) Emergency signs: severe pain, heavy bleeding, fever.",
		},
	}

	return &models.OrderSetTemplate{
		TemplateID:      "PROC-COLON-001",
		Category:        models.CategoryProcedure,
		Name:            "Colonoscopy Preparation",
		Version:         "2024.1",
		GuidelineSource: "ASGE Bowel Preparation Guidelines, ACG Clinical Guidelines on Colonoscopy Surveillance",
		Description:     "Complete colonoscopy preparation order set including split-dose bowel prep, medication management, sedation, and recovery protocols",
		Orders:          orders,
		Active:          true,
	}
}

// GetUpperEndoscopyOrderSet returns EGD (esophagogastroduodenoscopy) order set
// Guidelines: ASGE Standards of Practice, ACG Guidelines
func GetUpperEndoscopyOrderSet() *models.OrderSetTemplate {
	orders := []models.Order{
		// Pre-procedure Preparation
		{
			OrderID:      "EGD-PREP-001",
			OrderType:    models.OrderTypeNursing,
			Category:     "preparation",
			Name:         "NPO Instructions",
			Description:  "Nothing by mouth before EGD",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "NPO for solids 8 hours before procedure. Clear liquids allowed up to 2 hours before. Sips of water with medications up to 2 hours before.",
		},
		{
			OrderID:      "EGD-PREP-002",
			OrderType:    models.OrderTypeNursing,
			Category:     "preparation",
			Name:         "Consent Verification",
			Description:  "Verify informed consent",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Verify signed informed consent for EGD with possible biopsy. Discuss risks: bleeding, perforation, aspiration, reaction to sedation.",
		},

		// Medication Management
		{
			OrderID:      "EGD-MED-001",
			OrderType:    models.OrderTypeNursing,
			Category:     "medication_management",
			Name:         "Hold Blood Thinners",
			Description:  "Anticoagulation management",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			Instructions: "For diagnostic EGD: Aspirin/P2Y12 may continue. For therapeutic: Hold warfarin 5 days, DOACs 48-72h. Discuss with GI if high bleeding risk.",
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "anticoagulant_use", Operator: "eq", Value: "true"},
			},
		},
		{
			OrderID:      "EGD-MED-002",
			OrderType:    models.OrderTypeMedication,
			Category:     "medication_management",
			Name:         "Hold PPI Before H. pylori Testing",
			Description:  "Avoid false negatives on H. pylori testing",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			Instructions: "If H. pylori testing planned: Hold PPI x 2 weeks, H2 blockers x 2 days, antibiotics x 4 weeks before.",
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "h_pylori_testing", Operator: "eq", Value: "true"},
			},
		},

		// Pre-procedure Labs
		{
			OrderID:      "EGD-LAB-001",
			OrderType:    models.OrderTypeLab,
			Category:     "labs",
			Name:         "Type and Screen",
			Description:  "Blood typing for GI bleed workup",
			Priority:     models.PriorityUrgent,
			Required:     false,
			Selected:     false,
			LabCode:      "882-1",
			Specimen:     "blood",
			Timing:       "on_admission",
			Codes: []models.CodeReference{
				{System: "loinc", Code: "882-1", Display: "ABO and Rh group"},
			},
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "indication", Operator: "eq", Value: "gi_bleed"},
			},
		},
		{
			OrderID:      "EGD-LAB-002",
			OrderType:    models.OrderTypeLab,
			Category:     "labs",
			Name:         "CBC and Coags",
			Description:  "Pre-procedure bleeding risk labs",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			LabCode:      "58410-2",
			LabPanel:     "CBC, PT/INR",
			Specimen:     "blood",
			Timing:       "within_7_days",
			Codes: []models.CodeReference{
				{System: "loinc", Code: "58410-2", Display: "CBC"},
				{System: "loinc", Code: "34714-6", Display: "INR"},
			},
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "therapeutic_procedure", Operator: "eq", Value: "true"},
			},
		},

		// Sedation
		{
			OrderID:      "EGD-SED-001",
			OrderType:    models.OrderTypeMedication,
			Category:     "sedation",
			Name:         "Moderate Sedation",
			Description:  "Standard EGD sedation protocol",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Fentanyl 25-50mcg IV + Midazolam 0.5-2mg IV. Titrate for comfort. May add 25-50mcg fentanyl PRN. Monitor continuously.",
		},
		{
			OrderID:      "EGD-SED-002",
			OrderType:    models.OrderTypeMedication,
			Category:     "sedation",
			Name:         "Propofol MAC",
			Description:  "Deep sedation for complex cases",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			DrugCode:     "8782",
			DrugName:     "Propofol",
			Instructions: "Anesthesia-administered propofol for: prior sedation failure, anticipated long procedure, patient anxiety, or airway concerns.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "8782", Display: "Propofol"},
			},
		},

		// Topical Anesthesia
		{
			OrderID:      "EGD-TOP-001",
			OrderType:    models.OrderTypeMedication,
			Category:     "anesthesia",
			Name:         "Cetacaine Spray",
			Description:  "Topical anesthesia for throat",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			DrugCode:     "105799",
			DrugName:     "Benzocaine/Tetracaine/Aminobenzoate",
			Instructions: "Spray oropharynx before intubation. Avoid in methemoglobinemia risk. NPO for 1 hour after due to aspiration risk.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "105799", Display: "Cetacaine"},
			},
			Notes: "Use sparingly - risk of methemoglobinemia with excessive use.",
		},
		{
			OrderID:      "EGD-TOP-002",
			OrderType:    models.OrderTypeMedication,
			Category:     "anesthesia",
			Name:         "Viscous Lidocaine Alternative",
			Description:  "Lidocaine gargle for throat",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			DrugCode:     "6387",
			DrugName:     "Lidocaine 2% Viscous",
			Dose:         "15 mL",
			Route:        "swish and spit",
			Instructions: "Gargle and spit. Do not swallow. Alternative if Cetacaine contraindicated.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "6387", Display: "Lidocaine Viscous"},
			},
		},

		// Procedure-specific Medications
		{
			OrderID:      "EGD-PROC-001",
			OrderType:    models.OrderTypeMedication,
			Category:     "procedure",
			Name:         "Glucagon for Spasm",
			Description:  "Smooth muscle relaxant",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     true,
			DrugCode:     "4833",
			DrugName:     "Glucagon",
			Dose:         "0.25-0.5 mg",
			DoseValue:    0.5,
			DoseUnit:     "mg",
			Route:        "IV",
			PRN:          true,
			PRNReason:    "esophageal or gastric spasm",
			Instructions: "Give for spasm preventing scope advancement. May repeat x1.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "4833", Display: "Glucagon"},
			},
		},
		{
			OrderID:      "EGD-PROC-002",
			OrderType:    models.OrderTypeMedication,
			Category:     "procedure",
			Name:         "Epinephrine Injection",
			Description:  "For hemostasis of bleeding lesions",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     false,
			DrugCode:     "3992",
			DrugName:     "Epinephrine",
			Dose:         "1:10,000 dilution",
			Route:        "endoscopic injection",
			Instructions: "Inject in 0.5-1mL aliquots around bleeding lesion. Max 10mL total. Always combine with second hemostasis modality.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "3992", Display: "Epinephrine"},
			},
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "indication", Operator: "eq", Value: "gi_bleed"},
			},
		},

		// GI Bleed-Specific Orders
		{
			OrderID:      "EGD-BLEED-001",
			OrderType:    models.OrderTypeMedication,
			Category:     "gi_bleed",
			Name:         "IV PPI Bolus and Drip",
			Description:  "High-dose PPI for ulcer bleeding",
			Priority:     models.PrioritySTAT,
			Required:     false,
			Selected:     false,
			DrugCode:     "40790",
			DrugName:     "Pantoprazole",
			Dose:         "80 mg bolus, then 8 mg/hr",
			Route:        "IV",
			Duration:     "72_hours",
			Instructions: "For high-risk ulcer stigmata post-hemostasis. Continue 72h, then PO PPI BID.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "40790", Display: "Pantoprazole"},
			},
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "high_risk_stigmata", Operator: "eq", Value: "true"},
			},
		},
		{
			OrderID:      "EGD-BLEED-002",
			OrderType:    models.OrderTypeMedication,
			Category:     "gi_bleed",
			Name:         "Erythromycin Pre-EGD",
			Description:  "Prokinetic to clear blood from stomach",
			Priority:     models.PriorityUrgent,
			Required:     false,
			Selected:     false,
			DrugCode:     "4031",
			DrugName:     "Erythromycin",
			Dose:         "250 mg",
			DoseValue:    250,
			DoseUnit:     "mg",
			Route:        "IV",
			Frequency:    "once",
			Instructions: "Give 30-90 minutes before EGD for active upper GI bleed. Improves visualization by clearing gastric contents.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "4031", Display: "Erythromycin"},
			},
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "indication", Operator: "eq", Value: "acute_gi_bleed"},
			},
			Notes: "Evidence supports improved visualization. Avoid in QT prolongation.",
		},

		// Monitoring
		{
			OrderID:      "EGD-MON-001",
			OrderType:    models.OrderTypeMonitoring,
			Category:     "monitoring",
			Name:         "Continuous Monitoring",
			Description:  "Procedure monitoring requirements",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Continuous: SpO2, ECG, ETCO2 (if deep sedation). Intermittent: BP q5min. Document q5min during procedure.",
		},

		// Recovery
		{
			OrderID:      "EGD-REC-001",
			OrderType:    models.OrderTypeMonitoring,
			Category:     "recovery",
			Name:         "Post-procedure Recovery",
			Description:  "Recovery room monitoring",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Monitor VS q15min until stable. Assess LOC, gag reflex, pain. NPO until gag reflex returns (~1 hour). Discharge when Aldrete ≥9.",
		},
		{
			OrderID:      "EGD-REC-002",
			OrderType:    models.OrderTypeDiet,
			Category:     "recovery",
			Name:         "Resume Oral Intake",
			Description:  "Diet after EGD",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "Start with sips of water after gag reflex returns. If tolerated, clear liquids for 1 hour, then advance to regular diet. Avoid hot liquids for 1 hour.",
		},

		// Discharge
		{
			OrderID:      "EGD-DC-001",
			OrderType:    models.OrderTypeEducation,
			Category:     "discharge",
			Name:         "Discharge Instructions",
			Description:  "Post-EGD education",
			Priority:     models.PriorityRoutine,
			Required:     true,
			Selected:     true,
			Instructions: "1) No driving 24h. 2) Sore throat normal 24-48h. 3) Mild bloating expected. 4) Return for: severe pain, bleeding, fever, difficulty swallowing. 5) Follow up for biopsy results.",
		},
		{
			OrderID:      "EGD-DC-002",
			OrderType:    models.OrderTypeMedication,
			Category:     "discharge",
			Name:         "PPI Therapy",
			Description:  "Acid suppression post-EGD",
			Priority:     models.PriorityRoutine,
			Required:     false,
			Selected:     true,
			DrugCode:     "40790",
			DrugName:     "Pantoprazole",
			Dose:         "40 mg",
			DoseValue:    40,
			DoseUnit:     "mg",
			Route:        "PO",
			Frequency:    "daily",
			Duration:     "4-8_weeks",
			Instructions: "Take 30 minutes before breakfast. Duration based on findings: 4 weeks for mild gastritis, 8 weeks for ulcer.",
			Codes: []models.CodeReference{
				{System: "rxnorm", Code: "40790", Display: "Pantoprazole"},
			},
			Conditions: []models.OrderCondition{
				{Type: "if", Field: "findings", Operator: "eq", Value: "gastritis_or_ulcer"},
			},
		},
	}

	return &models.OrderSetTemplate{
		TemplateID:      "PROC-EGD-001",
		Category:        models.CategoryProcedure,
		Name:            "Upper Endoscopy (EGD)",
		Version:         "2024.1",
		GuidelineSource: "ASGE Standards of Practice, ACG Clinical Guidelines",
		Description:     "Complete esophagogastroduodenoscopy (EGD) order set including preparation, sedation, GI bleed management, and recovery protocols",
		Orders:          orders,
		Active:          true,
	}
}

// GetGIProcedureOrderSets returns all GI procedure order sets
func GetGIProcedureOrderSets() []*models.OrderSetTemplate {
	return []*models.OrderSetTemplate{
		GetColonoscopyPrepOrderSet(),
		GetUpperEndoscopyOrderSet(),
	}
}
