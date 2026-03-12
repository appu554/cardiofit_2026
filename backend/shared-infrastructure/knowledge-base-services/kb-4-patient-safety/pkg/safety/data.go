package safety

import "sync"

// SafetyDatabase holds all in-memory safety data
type SafetyDatabase struct {
	BlackBoxWarnings      map[string]*BlackBoxWarning
	Contraindications     map[string][]*Contraindication
	DoseLimits            map[string]*DoseLimit
	AgeLimits             map[string]*AgeLimit
	PregnancySafety       map[string]*PregnancySafety
	LactationSafety       map[string]*LactationSafety
	HighAlertMedications  map[string]*HighAlertMedication
	BeersEntries          map[string]*BeersEntry
	AnticholinergicBurden map[string]*AnticholinergicBurden
	LabRequirements       map[string]*LabRequirement
	mu                    sync.RWMutex
}

var (
	db   *SafetyDatabase
	once sync.Once
)

// GetDatabase returns the singleton safety database instance
func GetDatabase() *SafetyDatabase {
	once.Do(func() {
		db = &SafetyDatabase{
			BlackBoxWarnings:      make(map[string]*BlackBoxWarning),
			Contraindications:     make(map[string][]*Contraindication),
			DoseLimits:            make(map[string]*DoseLimit),
			AgeLimits:             make(map[string]*AgeLimit),
			PregnancySafety:       make(map[string]*PregnancySafety),
			LactationSafety:       make(map[string]*LactationSafety),
			HighAlertMedications:  make(map[string]*HighAlertMedication),
			BeersEntries:          make(map[string]*BeersEntry),
			AnticholinergicBurden: make(map[string]*AnticholinergicBurden),
			LabRequirements:       make(map[string]*LabRequirement),
		}
		initializeBlackBoxWarnings(db)
		initializePregnancySafety(db)
		initializeLactationSafety(db)
		initializeHighAlertMedications(db)
		initializeBeersEntries(db)
		initializeAnticholinergicBurden(db)
		initializeDoseLimits(db)
		initializeAgeLimits(db)
		initializeContraindications(db)
		initializeLabRequirements(db)
	})
	return db
}

// initializeBlackBoxWarnings populates FDA black box warning data
func initializeBlackBoxWarnings(db *SafetyDatabase) {
	warnings := []*BlackBoxWarning{
		{
			RxNormCode:     "7804",
			DrugName:       "Oxycodone",
			RiskCategories: []string{"Addiction", "Abuse", "Misuse", "Respiratory Depression", "Neonatal Opioid Withdrawal Syndrome"},
			WarningText:    "Oxycodone exposes patients and other users to risks of opioid addiction, abuse, and misuse. Serious, life-threatening, or fatal respiratory depression may occur. Prolonged use during pregnancy can result in neonatal opioid withdrawal syndrome.",
			HasREMS:        false,
		},
		{
			RxNormCode:     "7052",
			DrugName:       "Morphine",
			RiskCategories: []string{"Addiction", "Abuse", "Misuse", "Respiratory Depression", "Neonatal Opioid Withdrawal Syndrome"},
			WarningText:    "Morphine exposes patients to risks of addiction, abuse, and misuse. Life-threatening respiratory depression may occur with use, especially in opioid-naive patients.",
			HasREMS:        false,
		},
		{
			RxNormCode:     "2551",
			DrugName:       "Ciprofloxacin",
			RiskCategories: []string{"Tendon Rupture", "Tendinitis", "Peripheral Neuropathy", "CNS Effects", "Myasthenia Gravis Exacerbation"},
			WarningText:    "Fluoroquinolones are associated with disabling and potentially irreversible serious adverse reactions including tendinitis and tendon rupture, peripheral neuropathy, and CNS effects.",
			HasREMS:        false,
		},
		{
			RxNormCode:     "36437",
			DrugName:       "Sertraline",
			RiskCategories: []string{"Suicidality", "Suicidal Thinking", "Suicidal Behavior"},
			WarningText:    "Antidepressants increased the risk of suicidal thinking and behavior in children, adolescents, and young adults in short-term studies. Monitor closely for clinical worsening and emergence of suicidal thoughts and behaviors.",
			HasREMS:        false,
		},
		{
			RxNormCode:     "475968",
			DrugName:       "Liraglutide",
			RiskCategories: []string{"Thyroid C-Cell Tumors", "Medullary Thyroid Carcinoma"},
			WarningText:    "Liraglutide causes dose-dependent and treatment-duration-dependent thyroid C-cell tumors in rodents. It is unknown whether liraglutide causes thyroid C-cell tumors, including medullary thyroid carcinoma (MTC), in humans.",
			HasREMS:        false,
		},
		{
			RxNormCode:     "2626",
			DrugName:       "Clozapine",
			RiskCategories: []string{"Severe Neutropenia", "Orthostatic Hypotension", "Bradycardia", "Syncope", "Seizures", "Myocarditis", "Cardiomyopathy"},
			WarningText:    "Severe neutropenia can occur with clozapine use. A Clozapine REMS Program is required. Other serious risks include orthostatic hypotension, bradycardia, syncope, seizures, myocarditis, and cardiomyopathy.",
			HasREMS:        true,
			REMSProgram:    "Clozapine REMS Program",
		},
		{
			RxNormCode:     "6064",
			DrugName:       "Isotretinoin",
			RiskCategories: []string{"Teratogenicity", "Birth Defects", "Fetal Death", "Spontaneous Abortion"},
			WarningText:    "Isotretinoin is highly teratogenic. Severe birth defects, spontaneous abortions, and fetal death have been reported. Must use iPLEDGE REMS program. Female patients must use two forms of contraception.",
			HasREMS:        true,
			REMSProgram:    "iPLEDGE REMS Program",
		},
		{
			RxNormCode:     "11289",
			DrugName:       "Warfarin",
			RiskCategories: []string{"Bleeding", "Hemorrhage", "Fatal Bleeding"},
			WarningText:    "Warfarin can cause major or fatal bleeding. Bleeding is more likely during the starting period and with a higher dose. Regular monitoring of INR is required.",
			HasREMS:        false,
		},
		{
			RxNormCode:     "6851",
			DrugName:       "Methotrexate",
			RiskCategories: []string{"Teratogenicity", "Bone Marrow Suppression", "Hepatotoxicity", "Nephrotoxicity", "Pulmonary Toxicity"},
			WarningText:    "Methotrexate can cause embryo-fetal toxicity, bone marrow suppression, serious hepatotoxicity, nephrotoxicity, and pulmonary toxicity. Should only be used by physicians experienced with antimetabolite therapy.",
			HasREMS:        false,
		},
	}

	for _, w := range warnings {
		db.BlackBoxWarnings[w.RxNormCode] = w
	}
}

// initializePregnancySafety populates pregnancy safety data
func initializePregnancySafety(db *SafetyDatabase) {
	entries := []*PregnancySafety{
		{
			RxNormCode:         "6064",
			DrugName:           "Isotretinoin",
			Category:           PregnancyCategoryX,
			Teratogenic:        true,
			TeratogenicEffects: []string{"Craniofacial abnormalities", "CNS defects", "Cardiac defects", "Thymic abnormalities"},
			Recommendation:     "ABSOLUTELY CONTRAINDICATED. Must use iPLEDGE program with two forms of contraception.",
			AlternativeDrugs:   []string{"Topical retinoids (with caution)", "Benzoyl peroxide", "Topical antibiotics"},
		},
		{
			RxNormCode:         "11289",
			DrugName:           "Warfarin",
			Category:           PregnancyCategoryX,
			Teratogenic:        true,
			TeratogenicEffects: []string{"Nasal hypoplasia", "Stippled epiphyses (warfarin embryopathy)", "CNS abnormalities", "Spontaneous abortion"},
			TrimesterRisks: map[string]string{
				"first":  "Highest teratogenic risk - warfarin embryopathy",
				"second": "CNS abnormalities",
				"third":  "Fetal/neonatal hemorrhage",
			},
			Recommendation:   "CONTRAINDICATED throughout pregnancy. Switch to LMWH or unfractionated heparin.",
			AlternativeDrugs: []string{"Enoxaparin", "Dalteparin", "Unfractionated heparin"},
		},
		{
			RxNormCode:         "6851",
			DrugName:           "Methotrexate",
			Category:           PregnancyCategoryX,
			Teratogenic:        true,
			TeratogenicEffects: []string{"Aminopterin syndrome", "CNS abnormalities", "Limb defects", "Growth restriction"},
			Recommendation:     "CONTRAINDICATED. Discontinue at least 3 months before conception.",
			AlternativeDrugs:   []string{"Sulfasalazine", "Hydroxychloroquine (for RA)"},
		},
		{
			RxNormCode:         "1998",
			DrugName:           "Lisinopril",
			Category:           PregnancyCategoryD,
			Teratogenic:        true,
			TeratogenicEffects: []string{"Renal dysgenesis", "Oligohydramnios", "Fetal hypotension", "Skull hypoplasia"},
			TrimesterRisks: map[string]string{
				"first":  "Lower risk but still avoid",
				"second": "Fetal renal failure, oligohydramnios",
				"third":  "Fetal renal failure, neonatal hypotension",
			},
			Recommendation:   "Discontinue as soon as pregnancy detected. CONTRAINDICATED in 2nd/3rd trimester.",
			AlternativeDrugs: []string{"Labetalol", "Methyldopa", "Nifedipine"},
		},
		{
			RxNormCode:     "7804",
			DrugName:       "Oxycodone",
			Category:       PregnancyCategoryC,
			Teratogenic:    false,
			TrimesterRisks: map[string]string{
				"third": "Neonatal opioid withdrawal syndrome (NOWS)",
			},
			Recommendation: "Use only if benefits outweigh risks. Monitor neonate for withdrawal.",
		},
		{
			RxNormCode:     "36437",
			DrugName:       "Sertraline",
			Category:       PregnancyCategoryC,
			Teratogenic:    false,
			TrimesterRisks: map[string]string{
				"third": "Neonatal adaptation syndrome, persistent pulmonary hypertension",
			},
			Recommendation: "Generally considered acceptable. Weigh risks vs untreated depression.",
		},
	}

	for _, e := range entries {
		db.PregnancySafety[e.RxNormCode] = e
	}
}

// initializeLactationSafety populates lactation safety data
func initializeLactationSafety(db *SafetyDatabase) {
	entries := []*LactationSafety{
		{
			RxNormCode:        "6064",
			DrugName:          "Isotretinoin",
			Risk:              LactationContraindicated,
			ExcretedInMilk:    true,
			InfantEffects:     []string{"Unknown effects on nursing infant"},
			Recommendation:    "CONTRAINDICATED during breastfeeding.",
		},
		{
			RxNormCode:        "6851",
			DrugName:          "Methotrexate",
			Risk:              LactationContraindicated,
			ExcretedInMilk:    true,
			InfantEffects:     []string{"Immunosuppression", "Neutropenia", "Growth suppression"},
			Recommendation:    "CONTRAINDICATED. Discontinue breastfeeding or do not use.",
		},
		{
			RxNormCode:        "7804",
			DrugName:          "Oxycodone",
			Risk:              LactationUseWithCaution,
			ExcretedInMilk:    true,
			MilkPlasmaRatio:   "3.4",
			InfantDosePercent: 8.0,
			InfantEffects:     []string{"Sedation", "Respiratory depression", "Poor feeding"},
			Recommendation:    "Use lowest effective dose for shortest duration. Monitor infant closely.",
		},
		{
			RxNormCode:        "36437",
			DrugName:          "Sertraline",
			Risk:              LactationCompatible,
			ExcretedInMilk:    true,
			MilkPlasmaRatio:   "1.8",
			InfantDosePercent: 0.5,
			InfantEffects:     []string{"Usually well tolerated"},
			Recommendation:    "Compatible with breastfeeding. Preferred SSRI for lactation.",
		},
		{
			RxNormCode:        "11289",
			DrugName:          "Warfarin",
			Risk:              LactationCompatible,
			ExcretedInMilk:    false,
			InfantDosePercent: 0.0,
			Recommendation:    "Compatible. Minimal to no excretion in breast milk.",
		},
	}

	for _, e := range entries {
		db.LactationSafety[e.RxNormCode] = e
	}
}

// initializeHighAlertMedications populates ISMP high-alert medication data
func initializeHighAlertMedications(db *SafetyDatabase) {
	meds := []*HighAlertMedication{
		{
			RxNormCode:   "11289",
			DrugName:     "Warfarin",
			Category:     HighAlertAnticoagulants,
			Requirements: []string{"Double-check dosing", "INR monitoring", "Patient education", "Drug interaction screening"},
			Safeguards:   []string{"Independent double-check", "Standard concentrations", "INR within range"},
			DoubleCheck:  true,
			SmartPump:    false,
		},
		{
			RxNormCode:   "67108",
			DrugName:     "Enoxaparin",
			Category:     HighAlertAnticoagulants,
			Requirements: []string{"Double-check dosing", "Weight-based dosing", "Renal function check", "Anti-Xa monitoring if indicated"},
			Safeguards:   []string{"Independent double-check", "Dose rounding protocol", "Renal adjustment"},
			DoubleCheck:  true,
			SmartPump:    false,
		},
		{
			RxNormCode:   "5224",
			DrugName:     "Heparin",
			Category:     HighAlertAnticoagulants,
			Requirements: []string{"Double-check dosing", "Smart pump required for IV", "Weight-based dosing", "aPTT monitoring"},
			Safeguards:   []string{"Independent double-check", "Standard concentrations only", "Protocol-based dosing"},
			DoubleCheck:  true,
			SmartPump:    true,
		},
		{
			RxNormCode:   "274783",
			DrugName:     "Insulin Glargine",
			Category:     HighAlertInsulin,
			Requirements: []string{"Double-check dosing", "Never give IV", "Clear labeling", "Glucose monitoring"},
			Safeguards:   []string{"Independent double-check", "Storage separation from regular insulin", "TALL-man lettering"},
			DoubleCheck:  true,
			SmartPump:    false,
			TallManName:  "insulin glarGINE",
		},
		{
			RxNormCode:   "5856",
			DrugName:     "Insulin Regular",
			Category:     HighAlertInsulin,
			Requirements: []string{"Double-check dosing", "Smart pump for IV", "Glucose monitoring", "Clear labeling"},
			Safeguards:   []string{"Independent double-check", "Standard concentrations", "Sliding scale protocols"},
			DoubleCheck:  true,
			SmartPump:    true,
		},
		{
			RxNormCode:   "7052",
			DrugName:     "Morphine",
			Category:     HighAlertOpioids,
			Requirements: []string{"Double-check dosing", "Smart pump for PCA/infusions", "Respiratory monitoring", "Sedation assessment"},
			Safeguards:   []string{"Independent double-check", "Standard concentrations", "Dose limits in pump"},
			DoubleCheck:  true,
			SmartPump:    true,
		},
		{
			RxNormCode:   "7804",
			DrugName:     "Oxycodone",
			Category:     HighAlertOpioids,
			Requirements: []string{"PMP check", "Naloxone co-prescription", "Opioid agreement", "Respiratory monitoring"},
			Safeguards:   []string{"Independent double-check", "Dose limits", "Periodic review"},
			DoubleCheck:  true,
			SmartPump:    false,
		},
		{
			RxNormCode:   "8591",
			DrugName:     "Potassium Chloride",
			Category:     HighAlertElectrolytes,
			Requirements: []string{"Double-check dosing", "Smart pump required", "Cardiac monitoring if IV push", "Rate limits"},
			Safeguards:   []string{"Independent double-check", "Premixed solutions only", "No concentrated KCl on units"},
			DoubleCheck:  true,
			SmartPump:    true,
		},
	}

	for _, m := range meds {
		db.HighAlertMedications[m.RxNormCode] = m
	}
}

// initializeBeersEntries populates Beers Criteria data for geriatric patients
func initializeBeersEntries(db *SafetyDatabase) {
	entries := []*BeersEntry{
		{
			RxNormCode:               "3498",
			DrugName:                 "Diphenhydramine",
			DrugClass:                "First-generation antihistamine",
			Recommendation:           BeersAvoid,
			Rationale:                "Highly anticholinergic; clearance reduced with advanced age. Tolerance develops when used as hypnotic. Risk of confusion, dry mouth, constipation, urinary retention.",
			QualityOfEvidence:        "Moderate",
			StrengthOfRecommendation: "Strong",
			AlternativeDrugs:         []string{"Second-generation antihistamines (cetirizine, loratadine)", "Melatonin for sleep"},
		},
		{
			RxNormCode:               "596",
			DrugName:                 "Alprazolam",
			DrugClass:                "Benzodiazepine",
			Recommendation:           BeersAvoid,
			Rationale:                "Older adults have increased sensitivity. Increased risk of cognitive impairment, delirium, falls, fractures, and motor vehicle crashes.",
			QualityOfEvidence:        "Moderate",
			StrengthOfRecommendation: "Strong",
			Conditions:               []string{"Falls", "Delirium", "Cognitive impairment"},
			AlternativeDrugs:         []string{"SSRIs for anxiety", "Buspirone", "CBT"},
		},
		{
			RxNormCode:               "5640",
			DrugName:                 "Ibuprofen",
			DrugClass:                "NSAID",
			Recommendation:           BeersAvoid,
			Rationale:                "Increases risk of GI bleeding/peptic ulcer disease, AKI, fluid retention, and cardiovascular events.",
			QualityOfEvidence:        "Moderate",
			StrengthOfRecommendation: "Strong",
			Conditions:               []string{"History of GI bleeding", "CKD Stage 4+", "Heart failure"},
			AlternativeDrugs:         []string{"Acetaminophen", "Topical NSAIDs", "Topical capsaicin"},
		},
		{
			RxNormCode:               "7676",
			DrugName:                 "Oxybutynin",
			DrugClass:                "Antimuscarinic (urinary)",
			Recommendation:           BeersAvoid,
			Rationale:                "Strongly anticholinergic. Associated with cognitive decline and dementia risk in older adults.",
			QualityOfEvidence:        "Moderate",
			StrengthOfRecommendation: "Strong",
			AlternativeDrugs:         []string{"Mirabegron", "Behavioral interventions", "Pelvic floor exercises"},
		},
		{
			RxNormCode:               "4815",
			DrugName:                 "Glyburide",
			DrugClass:                "Sulfonylurea",
			Recommendation:           BeersAvoid,
			Rationale:                "Higher risk of severe prolonged hypoglycemia in older adults due to long half-life.",
			QualityOfEvidence:        "High",
			StrengthOfRecommendation: "Strong",
			AlternativeDrugs:         []string{"Glipizide", "Glimepiride", "Metformin", "DPP-4 inhibitors"},
		},
		{
			RxNormCode:               "704",
			DrugName:                 "Amitriptyline",
			DrugClass:                "Tricyclic antidepressant",
			Recommendation:           BeersAvoid,
			Rationale:                "Highly anticholinergic, sedating, causes orthostatic hypotension. Risk of falls and cognitive impairment.",
			QualityOfEvidence:        "High",
			StrengthOfRecommendation: "Strong",
			AlternativeDrugs:         []string{"SSRIs", "SNRIs", "Mirtazapine"},
		},
	}

	for _, e := range entries {
		db.BeersEntries[e.RxNormCode] = e
	}
}

// initializeAnticholinergicBurden populates anticholinergic burden score data
func initializeAnticholinergicBurden(db *SafetyDatabase) {
	entries := []*AnticholinergicBurden{
		// ACB Score 3 (High)
		{RxNormCode: "3498", DrugName: "Diphenhydramine", ACBScore: 3, RiskLevel: "High", Effects: []string{"Confusion", "Delirium", "Cognitive impairment", "Falls"}},
		{RxNormCode: "7676", DrugName: "Oxybutynin", ACBScore: 3, RiskLevel: "High", Effects: []string{"Cognitive decline", "Confusion", "Constipation"}},
		{RxNormCode: "704", DrugName: "Amitriptyline", ACBScore: 3, RiskLevel: "High", Effects: []string{"Sedation", "Confusion", "Dry mouth", "Urinary retention"}},
		{RxNormCode: "2597", DrugName: "Chlorpheniramine", ACBScore: 3, RiskLevel: "High", Effects: []string{"Sedation", "Confusion", "Dry mouth"}},
		{RxNormCode: "9009", DrugName: "Promethazine", ACBScore: 3, RiskLevel: "High", Effects: []string{"Sedation", "Confusion", "Extrapyramidal effects"}},
		{RxNormCode: "5093", DrugName: "Hydroxyzine", ACBScore: 3, RiskLevel: "High", Effects: []string{"Sedation", "Confusion", "Dry mouth"}},
		{RxNormCode: "2183", DrugName: "Clomipramine", ACBScore: 3, RiskLevel: "High", Effects: []string{"Sedation", "Confusion", "Cardiac effects"}},
		{RxNormCode: "3638", DrugName: "Doxepin", ACBScore: 3, RiskLevel: "High", Effects: []string{"Sedation", "Confusion", "Orthostatic hypotension"}},

		// ACB Score 2 (Moderate)
		{RxNormCode: "24677", DrugName: "Cyclobenzaprine", ACBScore: 2, RiskLevel: "Moderate", Effects: []string{"Sedation", "Dry mouth"}},
		{RxNormCode: "2626", DrugName: "Clozapine", ACBScore: 2, RiskLevel: "Moderate", Effects: []string{"Sedation", "Constipation", "Sialorrhea"}},
		{RxNormCode: "7531", DrugName: "Olanzapine", ACBScore: 2, RiskLevel: "Moderate", Effects: []string{"Sedation", "Weight gain"}},
		{RxNormCode: "8123", DrugName: "Paroxetine", ACBScore: 2, RiskLevel: "Moderate", Effects: []string{"Sedation", "Dry mouth"}},
		{RxNormCode: "10689", DrugName: "Tolterodine", ACBScore: 2, RiskLevel: "Moderate", Effects: []string{"Dry mouth", "Constipation"}},

		// ACB Score 1 (Low)
		{RxNormCode: "4603", DrugName: "Furosemide", ACBScore: 1, RiskLevel: "Low", Effects: []string{"Minimal anticholinergic effects"}},
		{RxNormCode: "6918", DrugName: "Metoprolol", ACBScore: 1, RiskLevel: "Low", Effects: []string{"Minimal anticholinergic effects"}},
		{RxNormCode: "10582", DrugName: "Trazodone", ACBScore: 1, RiskLevel: "Low", Effects: []string{"Sedation"}},
		{RxNormCode: "5691", DrugName: "Isosorbide", ACBScore: 1, RiskLevel: "Low", Effects: []string{"Minimal anticholinergic effects"}},
		{RxNormCode: "3393", DrugName: "Digoxin", ACBScore: 1, RiskLevel: "Low", Effects: []string{"Minimal anticholinergic effects"}},
		{RxNormCode: "8787", DrugName: "Prednisone", ACBScore: 1, RiskLevel: "Low", Effects: []string{"Minimal anticholinergic effects"}},
	}

	for _, e := range entries {
		db.AnticholinergicBurden[e.RxNormCode] = e
	}
}

// initializeDoseLimits populates maximum dose information
func initializeDoseLimits(db *SafetyDatabase) {
	limits := []*DoseLimit{
		{
			RxNormCode:        "7804",
			DrugName:          "Oxycodone",
			MaxSingleDose:     30,
			MaxSingleDoseUnit: "mg",
			MaxDailyDose:      120,
			MaxDailyDoseUnit:  "mg",
			GeriatricMaxDose:  15,
			RenalAdjustment:   "Reduce dose by 50% if CrCl < 30 mL/min",
			HepaticAdjustment: "Start with 1/3 to 1/2 usual dose in severe impairment",
		},
		{
			RxNormCode:        "7052",
			DrugName:          "Morphine",
			MaxSingleDose:     30,
			MaxSingleDoseUnit: "mg",
			MaxDailyDose:      200,
			MaxDailyDoseUnit:  "mg",
			GeriatricMaxDose:  15,
			RenalAdjustment:   "Avoid in ESRD; active metabolite accumulation",
			HepaticAdjustment: "Reduce dose in cirrhosis",
		},
		{
			RxNormCode:        "5640",
			DrugName:          "Ibuprofen",
			MaxSingleDose:     800,
			MaxSingleDoseUnit: "mg",
			MaxDailyDose:      3200,
			MaxDailyDoseUnit:  "mg",
			GeriatricMaxDose:  1200,
			RenalAdjustment:   "Avoid if CrCl < 30 mL/min",
		},
		{
			RxNormCode:        "161",
			DrugName:          "Acetaminophen",
			MaxSingleDose:     1000,
			MaxSingleDoseUnit: "mg",
			MaxDailyDose:      4000,
			MaxDailyDoseUnit:  "mg",
			GeriatricMaxDose:  3000,
			HepaticAdjustment: "Max 2000 mg/day in chronic liver disease",
		},
		{
			RxNormCode:        "8591",
			DrugName:          "Potassium Chloride",
			MaxSingleDose:     40,
			MaxSingleDoseUnit: "mEq",
			MaxDailyDose:      100,
			MaxDailyDoseUnit:  "mEq",
			RenalAdjustment:   "Reduce in renal impairment; monitor K+ closely",
		},
		{
			RxNormCode:        "6851",
			DrugName:          "Methotrexate",
			MaxSingleDose:     25,
			MaxSingleDoseUnit: "mg",
			MaxDailyDose:      25,
			MaxDailyDoseUnit:  "mg",
			RenalAdjustment:   "Contraindicated if CrCl < 30 mL/min for RA doses",
		},
	}

	for _, l := range limits {
		db.DoseLimits[l.RxNormCode] = l
	}
}

// initializeAgeLimits populates age restriction data
func initializeAgeLimits(db *SafetyDatabase) {
	limits := []*AgeLimit{
		{
			RxNormCode:  "2551",
			DrugName:    "Ciprofloxacin",
			MinAgeYears: 18,
			Rationale:   "Not recommended in pediatrics due to risk of cartilage damage in weight-bearing joints. May use in specific serious infections if no alternative.",
			Severity:    SeverityHigh,
		},
		{
			RxNormCode:  "36437",
			DrugName:    "Sertraline",
			MinAgeYears: 6,
			Rationale:   "Approved for OCD in children 6+. Monitor closely for suicidality in patients under 25.",
			Severity:    SeverityModerate,
		},
		{
			RxNormCode:  "6064",
			DrugName:    "Isotretinoin",
			MinAgeYears: 12,
			Rationale:   "Not recommended under 12 years. iPLEDGE requirements apply to all patients of childbearing potential.",
			Severity:    SeverityHigh,
		},
		{
			RxNormCode:  "596",
			DrugName:    "Alprazolam",
			MinAgeYears: 18,
			Rationale:   "Safety and efficacy not established in pediatric patients. Avoid in elderly (Beers Criteria).",
			Severity:    SeverityHigh,
		},
		{
			RxNormCode:  "7804",
			DrugName:    "Oxycodone",
			MinAgeYears: 11,
			Rationale:   "Extended-release formulations approved for opioid-tolerant patients 11+ weighing at least 20kg.",
			Severity:    SeverityCritical,
		},
	}

	for _, l := range limits {
		db.AgeLimits[l.RxNormCode] = l
	}
}

// initializeContraindications populates contraindication data
func initializeContraindications(db *SafetyDatabase) {
	contraindications := []*Contraindication{
		{
			RxNormCode:            "1998",
			DrugName:              "Lisinopril",
			ConditionCodes:        []string{"T78.3", "D84.1"},
			ConditionDescriptions: []string{"Angioedema", "Hereditary angioedema"},
			Type:                  "absolute",
			Severity:              SeverityCritical,
			ClinicalRationale:     "ACE inhibitors can cause life-threatening angioedema, especially in patients with history.",
		},
		{
			RxNormCode:            "5640",
			DrugName:              "Ibuprofen",
			ConditionCodes:        []string{"K25", "K26", "K27", "K28"},
			ConditionDescriptions: []string{"Gastric ulcer", "Duodenal ulcer", "Peptic ulcer", "GI bleeding"},
			Type:                  "relative",
			Severity:              SeverityHigh,
			ClinicalRationale:     "NSAIDs increase risk of GI bleeding and can exacerbate existing ulcers.",
			AlternativeConsiderations: "Consider acetaminophen or topical NSAIDs if needed.",
		},
		{
			RxNormCode:            "5640",
			DrugName:              "Ibuprofen",
			ConditionCodes:        []string{"N18.4", "N18.5", "N18.6"},
			ConditionDescriptions: []string{"CKD Stage 4", "CKD Stage 5", "ESRD"},
			Type:                  "absolute",
			Severity:              SeverityCritical,
			ClinicalRationale:     "NSAIDs cause renal vasoconstriction and can precipitate acute kidney injury.",
		},
		{
			RxNormCode:            "6851",
			DrugName:              "Methotrexate",
			ConditionCodes:        []string{"B20", "D80"},
			ConditionDescriptions: []string{"HIV disease", "Immunodeficiency"},
			Type:                  "relative",
			Severity:              SeverityHigh,
			ClinicalRationale:     "Methotrexate causes immunosuppression, increasing infection risk.",
		},
		{
			RxNormCode:            "11289",
			DrugName:              "Warfarin",
			ConditionCodes:        []string{"I60", "I61", "I62"},
			ConditionDescriptions: []string{"Subarachnoid hemorrhage", "Intracerebral hemorrhage", "Intracranial hemorrhage"},
			Type:                  "absolute",
			Severity:              SeverityCritical,
			ClinicalRationale:     "Active bleeding is an absolute contraindication to anticoagulation.",
		},
		{
			RxNormCode:            "2626",
			DrugName:              "Clozapine",
			ConditionCodes:        []string{"D70.9"},
			ConditionDescriptions: []string{"Neutropenia", "Agranulocytosis"},
			Type:                  "absolute",
			Severity:              SeverityCritical,
			ClinicalRationale:     "Clozapine can cause severe neutropenia. Prior drug-induced neutropenia is contraindication.",
		},
	}

	for _, c := range contraindications {
		if db.Contraindications[c.RxNormCode] == nil {
			db.Contraindications[c.RxNormCode] = []*Contraindication{}
		}
		db.Contraindications[c.RxNormCode] = append(db.Contraindications[c.RxNormCode], c)
	}
}

// initializeLabRequirements populates required laboratory monitoring data
func initializeLabRequirements(db *SafetyDatabase) {
	requirements := []*LabRequirement{
		{
			RxNormCode:       "11289",
			DrugName:         "Warfarin",
			RequiredLabs:     []string{"INR", "PT"},
			LabCodes:         []string{"6301-6", "5902-2"},
			Frequency:        "Weekly until stable, then monthly",
			BaselineRequired: true,
			Rationale:        "INR monitoring required to maintain therapeutic range and prevent bleeding/clotting.",
		},
		{
			RxNormCode:       "6851",
			DrugName:         "Methotrexate",
			RequiredLabs:     []string{"CBC with differential", "LFTs", "Creatinine", "Albumin"},
			LabCodes:         []string{"57021-8", "24325-3", "2160-0", "1751-7"},
			Frequency:        "Weekly for 4 weeks, then monthly",
			BaselineRequired: true,
			Rationale:        "Monitor for bone marrow suppression, hepatotoxicity, and nephrotoxicity.",
		},
		{
			RxNormCode:       "2626",
			DrugName:         "Clozapine",
			RequiredLabs:     []string{"ANC (Absolute Neutrophil Count)"},
			LabCodes:         []string{"751-8"},
			Frequency:        "Weekly for 6 months, then every 2 weeks for 6 months, then monthly",
			BaselineRequired: true,
			Rationale:        "REMS requirement. Monitor for agranulocytosis.",
		},
		{
			RxNormCode:       "1998",
			DrugName:         "Lisinopril",
			RequiredLabs:     []string{"Potassium", "Creatinine", "BUN"},
			LabCodes:         []string{"2823-3", "2160-0", "3094-0"},
			Frequency:        "Within 2 weeks of initiation, then every 6-12 months",
			BaselineRequired: true,
			Rationale:        "Monitor for hyperkalemia and changes in renal function.",
		},
		{
			RxNormCode:       "4815",
			DrugName:         "Glyburide",
			RequiredLabs:     []string{"HbA1c", "Fasting glucose", "Renal function"},
			LabCodes:         []string{"4548-4", "1558-6", "2160-0"},
			Frequency:        "HbA1c every 3 months, renal function annually",
			BaselineRequired: true,
			Rationale:        "Monitor glycemic control and renal function for dose adjustments.",
		},
		{
			RxNormCode:       "8591",
			DrugName:         "Potassium Chloride",
			RequiredLabs:     []string{"Potassium", "Magnesium"},
			LabCodes:         []string{"2823-3", "2601-3"},
			Frequency:        "Within 48-72 hours of high-dose supplementation",
			BaselineRequired: true,
			Rationale:        "Prevent hyperkalemia, especially in patients with renal impairment.",
		},
	}

	for _, r := range requirements {
		db.LabRequirements[r.RxNormCode] = r
	}
}

// Lookup methods

// GetBlackBoxWarning retrieves black box warning by RxNorm code
func (db *SafetyDatabase) GetBlackBoxWarning(rxnormCode string) *BlackBoxWarning {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.BlackBoxWarnings[rxnormCode]
}

// GetAllBlackBoxWarnings returns all black box warnings
func (db *SafetyDatabase) GetAllBlackBoxWarnings() []*BlackBoxWarning {
	db.mu.RLock()
	defer db.mu.RUnlock()
	result := make([]*BlackBoxWarning, 0, len(db.BlackBoxWarnings))
	for _, w := range db.BlackBoxWarnings {
		result = append(result, w)
	}
	return result
}

// GetContraindications retrieves contraindications by RxNorm code
func (db *SafetyDatabase) GetContraindications(rxnormCode string) []*Contraindication {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.Contraindications[rxnormCode]
}

// GetDoseLimit retrieves dose limits by RxNorm code
func (db *SafetyDatabase) GetDoseLimit(rxnormCode string) *DoseLimit {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.DoseLimits[rxnormCode]
}

// GetAgeLimit retrieves age limits by RxNorm code
func (db *SafetyDatabase) GetAgeLimit(rxnormCode string) *AgeLimit {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.AgeLimits[rxnormCode]
}

// GetPregnancySafety retrieves pregnancy safety info by RxNorm code
func (db *SafetyDatabase) GetPregnancySafety(rxnormCode string) *PregnancySafety {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.PregnancySafety[rxnormCode]
}

// GetLactationSafety retrieves lactation safety info by RxNorm code
func (db *SafetyDatabase) GetLactationSafety(rxnormCode string) *LactationSafety {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.LactationSafety[rxnormCode]
}

// GetHighAlertMedication retrieves high-alert medication info by RxNorm code
func (db *SafetyDatabase) GetHighAlertMedication(rxnormCode string) *HighAlertMedication {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.HighAlertMedications[rxnormCode]
}

// GetAllHighAlertMedications returns all high-alert medications
func (db *SafetyDatabase) GetAllHighAlertMedications() []*HighAlertMedication {
	db.mu.RLock()
	defer db.mu.RUnlock()
	result := make([]*HighAlertMedication, 0, len(db.HighAlertMedications))
	for _, m := range db.HighAlertMedications {
		result = append(result, m)
	}
	return result
}

// GetBeersEntry retrieves Beers Criteria entry by RxNorm code
func (db *SafetyDatabase) GetBeersEntry(rxnormCode string) *BeersEntry {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.BeersEntries[rxnormCode]
}

// GetAnticholinergicBurden retrieves ACB score by RxNorm code
func (db *SafetyDatabase) GetAnticholinergicBurden(rxnormCode string) *AnticholinergicBurden {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.AnticholinergicBurden[rxnormCode]
}

// GetLabRequirement retrieves lab requirements by RxNorm code
func (db *SafetyDatabase) GetLabRequirement(rxnormCode string) *LabRequirement {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.LabRequirements[rxnormCode]
}

// IsHighAlertDrug checks if a drug is on the high-alert list
func (db *SafetyDatabase) IsHighAlertDrug(rxnormCode string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	_, exists := db.HighAlertMedications[rxnormCode]
	return exists
}

// HasBlackBoxWarning checks if a drug has a black box warning
func (db *SafetyDatabase) HasBlackBoxWarning(rxnormCode string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	_, exists := db.BlackBoxWarnings[rxnormCode]
	return exists
}
