package config

// =============================================================================
// PHASE-A FORMULARY: HIGH-RISK DRUG COVERAGE
// =============================================================================
// CTO/CMO Decision: Start with ~200 drugs that represent 80-90% of clinical risk.
// These are drugs where dosing errors cause ICU admissions, deaths, and litigation.
//
// Selection criteria:
// - High-alert (ISMP list)
// - Renal cleared (CKD prevalence)
// - Pregnancy-sensitive (maternal mortality)
// - Pediatric weight-based dosing
// - ICU critical medications
// - Chemotherapy (narrow therapeutic index)
// - Anti-infectives (resistance + renal dosing)
//
// ONLY expand beyond Phase-A after these 200 are manually validated.
// =============================================================================

// PhaseAFormulary contains the authoritative list of high-risk drugs
// that must be ingested, validated, and approved before scaling.
var PhaseAFormulary = PhaseAConfig{
	Version:     "1.0.0",
	Description: "Phase-A High-Risk Drug Formulary for Clinical Safety",
	Categories:  phaseACategories,
	DrugNames:   getAllPhaseADrugs(),
	TotalTarget: 200,
}

// PhaseAConfig defines the Phase-A formulary structure
type PhaseAConfig struct {
	Version     string
	Description string
	Categories  []DrugCategory
	DrugNames   []string
	TotalTarget int
}

// DrugCategory represents a clinical category of drugs
type DrugCategory struct {
	Name        string
	RiskLevel   string // CRITICAL, HIGH, STANDARD
	Description string
	Drugs       []string
	TargetCount int
}

// phaseACategories defines all Phase-A drug categories with clinical rationale
var phaseACategories = []DrugCategory{
	// ==========================================================================
	// CRITICAL RISK: Small error → death
	// ==========================================================================
	{
		Name:        "Anticoagulants",
		RiskLevel:   "CRITICAL",
		Description: "Top legal risk - bleeding/clotting deaths. Every anticoagulant error is a lawsuit.",
		TargetCount: 10,
		Drugs: []string{
			"warfarin",
			"heparin",
			"enoxaparin",
			"dabigatran",
			"apixaban",
			"rivaroxaban",
			"fondaparinux",
			"edoxaban",
			"dalteparin",
			"tinzaparin",
		},
	},
	{
		Name:        "Insulins & Diabetes",
		RiskLevel:   "CRITICAL",
		Description: "Hypoglycemia kills. Insulin errors are top 5 ISMP high-alert.",
		TargetCount: 15,
		Drugs: []string{
			// Insulins
			"insulin regular",
			"insulin nph",
			"insulin glargine",
			"insulin lispro",
			"insulin aspart",
			"insulin detemir",
			"insulin degludec",
			"insulin glulisine",
			// Oral diabetes
			"metformin",
			"glimepiride",
			"gliclazide",
			"glipizide",
			"pioglitazone",
			"sitagliptin",
			"empagliflozin",
			"dapagliflozin",
			"liraglutide",
			"semaglutide",
		},
	},
	{
		Name:        "Opioids & Pain",
		RiskLevel:   "CRITICAL",
		Description: "Respiratory depression kills. Opioid overdose is preventable.",
		TargetCount: 12,
		Drugs: []string{
			"morphine",
			"fentanyl",
			"oxycodone",
			"hydromorphone",
			"methadone",
			"tramadol",
			"codeine",
			"pethidine",
			"buprenorphine",
			"naloxone",
			"tapentadol",
			"oxycontin",
		},
	},
	{
		Name:        "ICU Vasopressors & Inotropes",
		RiskLevel:   "CRITICAL",
		Description: "Hemodynamic collapse. Wrong dose = cardiac arrest.",
		TargetCount: 12,
		Drugs: []string{
			"norepinephrine",
			"noradrenaline",
			"epinephrine",
			"adrenaline",
			"dopamine",
			"dobutamine",
			"vasopressin",
			"phenylephrine",
			"milrinone",
			"amiodarone",
			"adenosine",
			"atropine",
		},
	},
	{
		Name:        "Oncology & Chemotherapy",
		RiskLevel:   "CRITICAL",
		Description: "Narrow therapeutic index. BSA-based dosing with max caps.",
		TargetCount: 15,
		Drugs: []string{
			"methotrexate",
			"cyclophosphamide",
			"cisplatin",
			"carboplatin",
			"doxorubicin",
			"vincristine",
			"fluorouracil",
			"paclitaxel",
			"docetaxel",
			"imatinib",
			"rituximab",
			"trastuzumab",
			"tamoxifen",
			"letrozole",
			"anastrozole",
		},
	},

	// ==========================================================================
	// HIGH RISK: Narrow TI, black box, renal adjustment critical
	// ==========================================================================
	{
		Name:        "Antibiotics - Aminoglycosides",
		RiskLevel:   "HIGH",
		Description: "Nephrotoxicity + ototoxicity. Renal dosing mandatory.",
		TargetCount: 5,
		Drugs: []string{
			"gentamicin",
			"amikacin",
			"tobramycin",
			"streptomycin",
			"neomycin",
		},
	},
	{
		Name:        "Antibiotics - Beta-lactams",
		RiskLevel:   "HIGH",
		Description: "India ICU backbone. Renal adjustment + resistance patterns.",
		TargetCount: 15,
		Drugs: []string{
			"amoxicillin",
			"amoxicillin-clavulanate",
			"ampicillin",
			"ampicillin-sulbactam",
			"piperacillin-tazobactam",
			"ceftriaxone",
			"ceftazidime",
			"cefepime",
			"cefuroxime",
			"cephalexin",
			"meropenem",
			"imipenem-cilastatin",
			"ertapenem",
			"aztreonam",
			"cefixime",
		},
	},
	{
		Name:        "Antibiotics - Glycopeptides & Others",
		RiskLevel:   "HIGH",
		Description: "Last-line agents. Renal dosing + TDM required.",
		TargetCount: 10,
		Drugs: []string{
			"vancomycin",
			"teicoplanin",
			"linezolid",
			"daptomycin",
			"colistin",
			"tigecycline",
			"clindamycin",
			"metronidazole",
			"trimethoprim-sulfamethoxazole",
			"nitrofurantoin",
		},
	},
	{
		Name:        "Antibiotics - Fluoroquinolones",
		RiskLevel:   "HIGH",
		Description: "QT prolongation + tendon rupture. Renal dosing.",
		TargetCount: 5,
		Drugs: []string{
			"ciprofloxacin",
			"levofloxacin",
			"moxifloxacin",
			"ofloxacin",
			"norfloxacin",
		},
	},
	{
		Name:        "Antibiotics - Macrolides & Others",
		RiskLevel:   "STANDARD",
		Description: "Common outpatient use. Drug interactions.",
		TargetCount: 5,
		Drugs: []string{
			"azithromycin",
			"clarithromycin",
			"erythromycin",
			"doxycycline",
			"rifampicin",
		},
	},
	{
		Name:        "Cardiac - Antiarrhythmics",
		RiskLevel:   "HIGH",
		Description: "Proarrhythmic risk. Narrow therapeutic index.",
		TargetCount: 8,
		Drugs: []string{
			"amiodarone",
			"sotalol",
			"flecainide",
			"propafenone",
			"digoxin",
			"diltiazem",
			"verapamil",
			"lidocaine",
		},
	},
	{
		Name:        "Cardiac - Antihypertensives",
		RiskLevel:   "STANDARD",
		Description: "Common use. Renal dosing for ACE-I/ARBs.",
		TargetCount: 15,
		Drugs: []string{
			"amlodipine",
			"nifedipine",
			"metoprolol",
			"atenolol",
			"carvedilol",
			"bisoprolol",
			"labetalol",
			"propranolol",
			"lisinopril",
			"enalapril",
			"ramipril",
			"losartan",
			"telmisartan",
			"valsartan",
			"hydrochlorothiazide",
			"furosemide",
			"spironolactone",
			"torsemide",
		},
	},
	{
		Name:        "Maternal-Fetal Risk",
		RiskLevel:   "HIGH",
		Description: "Pregnancy categories. Maternal mortality prevention.",
		TargetCount: 12,
		Drugs: []string{
			"oxytocin",
			"magnesium sulfate",
			"methyldopa",
			"labetalol",
			"nifedipine",
			"misoprostol",
			"betamethasone",
			"dexamethasone",
			"terbutaline",
			"carboprost",
			"dinoprostone",
			"tranexamic acid",
		},
	},
	{
		Name:        "Renal & Transplant",
		RiskLevel:   "HIGH",
		Description: "Immunosuppressants. Narrow TI + TDM required.",
		TargetCount: 10,
		Drugs: []string{
			"tacrolimus",
			"cyclosporine",
			"mycophenolate",
			"azathioprine",
			"prednisone",
			"prednisolone",
			"methylprednisolone",
			"erythropoietin",
			"darbepoetin",
			"calcitriol",
			"sevelamer",
		},
	},
	{
		Name:        "Psychiatry & Neurology",
		RiskLevel:   "HIGH",
		Description: "Beers list + ACB risk. Narrow TI for lithium/antiepileptics.",
		TargetCount: 15,
		Drugs: []string{
			// Antipsychotics
			"haloperidol",
			"risperidone",
			"quetiapine",
			"olanzapine",
			"aripiprazole",
			// Mood stabilizers
			"lithium",
			"valproate",
			"carbamazepine",
			"lamotrigine",
			"levetiracetam",
			"phenytoin",
			// Antidepressants
			"sertraline",
			"fluoxetine",
			"escitalopram",
			"amitriptyline",
		},
	},
	{
		Name:        "ICU Sedation & Anesthesia",
		RiskLevel:   "HIGH",
		Description: "Respiratory depression. Weight-based dosing.",
		TargetCount: 8,
		Drugs: []string{
			"propofol",
			"midazolam",
			"lorazepam",
			"diazepam",
			"ketamine",
			"dexmedetomidine",
			"rocuronium",
			"succinylcholine",
		},
	},
	{
		Name:        "Pediatric Essentials",
		RiskLevel:   "HIGH",
		Description: "Weight-based killers. Decimal point errors.",
		TargetCount: 10,
		Drugs: []string{
			"paracetamol",
			"acetaminophen",
			"ibuprofen",
			"amoxicillin",
			"gentamicin",
			"ceftriaxone",
			"morphine",
			"insulin",
			"salbutamol",
			"phenobarbital",
		},
	},
	{
		Name:        "Respiratory",
		RiskLevel:   "STANDARD",
		Description: "Inhaler dosing. Theophylline narrow TI.",
		TargetCount: 8,
		Drugs: []string{
			"salbutamol",
			"albuterol",
			"ipratropium",
			"budesonide",
			"fluticasone",
			"formoterol",
			"salmeterol",
			"theophylline",
			"montelukast",
		},
	},
	{
		Name:        "Anticonvulsants",
		RiskLevel:   "HIGH",
		Description: "Narrow TI. TDM required for phenytoin, carbamazepine.",
		TargetCount: 8,
		Drugs: []string{
			"phenytoin",
			"carbamazepine",
			"valproate",
			"levetiracetam",
			"lamotrigine",
			"phenobarbital",
			"topiramate",
			"gabapentin",
		},
	},
	{
		Name:        "Antifungals",
		RiskLevel:   "HIGH",
		Description: "Nephrotoxicity + drug interactions.",
		TargetCount: 6,
		Drugs: []string{
			"fluconazole",
			"voriconazole",
			"itraconazole",
			"amphotericin b",
			"caspofungin",
			"micafungin",
		},
	},
	{
		Name:        "Antivirals",
		RiskLevel:   "STANDARD",
		Description: "HIV + hepatitis. Drug interactions.",
		TargetCount: 5,
		Drugs: []string{
			"acyclovir",
			"oseltamivir",
			"tenofovir",
			"sofosbuvir",
			"remdesivir",
		},
	},
	{
		Name:        "GI & Antiemetics",
		RiskLevel:   "STANDARD",
		Description: "Common use. QT prolongation for some.",
		TargetCount: 6,
		Drugs: []string{
			"omeprazole",
			"pantoprazole",
			"ondansetron",
			"metoclopramide",
			"ranitidine",
			"sucralfate",
		},
	},
}

// getAllPhaseADrugs returns a flat list of all Phase-A drug names
func getAllPhaseADrugs() []string {
	var drugs []string
	seen := make(map[string]bool)

	for _, category := range phaseACategories {
		for _, drug := range category.Drugs {
			if !seen[drug] {
				seen[drug] = true
				drugs = append(drugs, drug)
			}
		}
	}

	return drugs
}

// GetPhaseADrugCount returns the total count of Phase-A drugs
func GetPhaseADrugCount() int {
	return len(PhaseAFormulary.DrugNames)
}

// IsPhaseADrug checks if a drug name is in the Phase-A formulary
func IsPhaseADrug(drugName string) bool {
	drugNameLower := normalizeForSearch(drugName)
	for _, phaseADrug := range PhaseAFormulary.DrugNames {
		if containsIgnoreCase(drugNameLower, phaseADrug) {
			return true
		}
	}
	return false
}

// GetCategoryForDrug returns the category for a given drug
func GetCategoryForDrug(drugName string) *DrugCategory {
	drugNameLower := normalizeForSearch(drugName)
	for i := range phaseACategories {
		for _, drug := range phaseACategories[i].Drugs {
			if containsIgnoreCase(drugNameLower, drug) {
				return &phaseACategories[i]
			}
		}
	}
	return nil
}

// normalizeForSearch prepares a drug name for matching
func normalizeForSearch(name string) string {
	// Simple lowercase for now - could add more normalization
	result := ""
	for _, c := range name {
		if c >= 'a' && c <= 'z' {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32) // lowercase
		} else if c == ' ' || c == '-' {
			result += " "
		}
	}
	return result
}

// containsIgnoreCase checks if haystack contains needle (case-insensitive)
func containsIgnoreCase(haystack, needle string) bool {
	needleLower := normalizeForSearch(needle)
	return len(needleLower) > 0 && len(haystack) >= len(needleLower) &&
		(haystack == needleLower ||
		 len(haystack) > len(needleLower) &&
		 (haystack[:len(needleLower)] == needleLower ||
		  haystack[len(haystack)-len(needleLower):] == needleLower ||
		  containsSubstring(haystack, needleLower)))
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
