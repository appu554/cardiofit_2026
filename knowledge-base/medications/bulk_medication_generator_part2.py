#!/usr/bin/env python3
"""
Bulk Medication Generator Part 2 - Cardiovascular, Analgesics, Sedatives, Insulin, Anticonvulsants, Respiratory
Continuation of medication database expansion (medications 15-83)
"""

# CARDIOVASCULAR MEDICATIONS (16 medications - items 15-30)
CARDIOVASCULAR_MEDICATIONS = [
    {
        "medicationId": "MED-EPI-001",
        "genericName": "Epinephrine",
        "brandNames": ["Adrenalin"],
        "rxNormCode": "3992",
        "ndcCode": "0517-1000",
        "atcCode": "C01CA24",
        "category": "cardiovascular",
        "subcategory": "vasopressors",
        "classification": {
            "therapeuticClass": "Vasopressor",
            "pharmacologicClass": "Alpha and beta adrenergic agonist",
            "chemicalClass": "Catecholamine",
            "category": "Cardiovascular",
            "subcategories": ["Vasopressor", "Cardiac arrest", "Anaphylaxis"],
            "highAlert": True,
            "blackBoxWarning": False
        },
        "adultDosing": {
            "standard": {
                "dose": "0.05-2 mcg/min continuous infusion OR 1 mg IV push for arrest",
                "route": "IV",
                "frequency": "Continuous or bolus per ACLS",
                "duration": "Until effect",
                "maxDailyDose": "Variable - titrate to effect",
                "infusionDuration": "Continuous via central line preferred"
            },
            "indicationBased": {
                "cardiac_arrest": {"dose": "1 mg IV/IO push", "frequency": "every 3-5 minutes"},
                "anaphylaxis": {"dose": "0.3-0.5 mg IM (1:1000)", "frequency": "may repeat every 5-15 min"},
                "shock": {"dose": "0.05-2 mcg/min", "frequency": "titrate to MAP ≥65"}
            },
            "renalAdjustment": {
                "creatinineClearanceMethod": "Not applicable",
                "requiresDialysisAdjustment": False,
                "adjustments": {}
            }
        },
        "contraindications": {
            "absolute": ["None in cardiac arrest"],
            "relative": ["Coronary artery disease", "Severe hypertension"],
            "allergies": ["sulfite"],
            "diseaseStates": []
        },
        "adverseEffects": {
            "common": {"tachycardia": "Common", "hypertension": "Common", "anxiety": "Common"},
            "serious": {"arrhythmias": "Variable", "myocardial ischemia": "Variable", "extravasation necrosis": "If peripheral"},
            "blackBoxWarnings": [],
            "monitoring": "Continuous cardiac monitoring, BP, extravasation"
        },
        "pregnancyLactation": {
            "fdaCategory": "C",
            "pregnancyRisk": "Moderate",
            "lactationRisk": "Compatible",
            "pregnancyGuidance": "May reduce uterine blood flow",
            "breastfeedingGuidance": "Compatible - life-saving drug",
            "infantRiskCategory": "L1"
        }
    },
    {
        "medicationId": "MED-DOPA-001",
        "genericName": "Dopamine",
        "brandNames": ["Intropin"],
        "rxNormCode": "3616",
        "ndcCode": "0409-2501",
        "atcCode": "C01CA04",
        "category": "cardiovascular",
        "subcategory": "vasopressors",
        "classification": {
            "therapeuticClass": "Inotrope/Vasopressor",
            "pharmacologicClass": "Catecholamine",
            "chemicalClass": "Catecholamine",
            "category": "Cardiovascular",
            "subcategories": ["Vasopressor", "Inotrope"],
            "highAlert": True,
            "blackBoxWarning": True
        },
        "adultDosing": {
            "standard": {
                "dose": "2-20 mcg/kg/min",
                "route": "IV continuous infusion",
                "frequency": "Continuous",
                "duration": "Until hemodynamic stability",
                "maxDailyDose": "50 mcg/kg/min (rarely)",
                "infusionDuration": "Continuous via central line preferred"
            },
            "indicationBased": {
                "shock": {"dose": "5-20 mcg/kg/min", "frequency": "titrate to effect"},
                "renal_perfusion": {"dose": "2-5 mcg/kg/min", "frequency": "renal dose"}
            },
            "renalAdjustment": {
                "creatinineClearanceMethod": "Not applicable",
                "requiresDialysisAdjustment": False,
                "adjustments": {}
            }
        },
        "contraindications": {
            "absolute": ["Pheochromocytoma", "Uncorrected tachyarrhythmias"],
            "relative": ["Recent MAOI use"],
            "allergies": ["sulfite"],
            "diseaseStates": []
        },
        "adverseEffects": {
            "common": {"tachycardia": "Dose-dependent", "arrhythmias": "Variable"},
            "serious": {"extravasation necrosis": "If peripheral", "myocardial ischemia": "High doses"},
            "blackBoxWarnings": ["Extravasation can cause tissue necrosis"],
            "monitoring": "Continuous BP, HR, MAP, ECG, IV site"
        },
        "pregnancyLactation": {
            "fdaCategory": "C",
            "pregnancyRisk": "Moderate",
            "lactationRisk": "Unknown",
            "pregnancyGuidance": "May reduce uterine blood flow",
            "breastfeedingGuidance": "Unknown excretion",
            "infantRiskCategory": "L3"
        }
    },
    {
        "medicationId": "MED-VASO-001",
        "genericName": "Vasopressin",
        "brandNames": ["Pitressin"],
        "rxNormCode": "11137",
        "ndcCode": "0517-6410",
        "atcCode": "H01BA01",
        "category": "cardiovascular",
        "subcategory": "vasopressors",
        "classification": {
            "therapeuticClass": "Vasopressor",
            "pharmacologicClass": "Vasopressin analog",
            "chemicalClass": "Polypeptide hormone",
            "category": "Cardiovascular",
            "subcategories": ["Vasopressor", "Cardiac arrest"],
            "highAlert": True,
            "blackBoxWarning": False
        },
        "adultDosing": {
            "standard": {
                "dose": "0.03-0.04 units/min",
                "route": "IV continuous infusion",
                "frequency": "Continuous",
                "duration": "Until hemodynamic stability",
                "maxDailyDose": "0.04 units/min",
                "infusionDuration": "Continuous"
            },
            "indicationBased": {
                "septic_shock": {"dose": "0.03 units/min", "frequency": "add to norepinephrine"},
                "cardiac_arrest": {"dose": "40 units IV push", "frequency": "single dose alternative to epi"},
                "gi_bleeding": {"dose": "0.2-0.4 units/min", "frequency": "up to 0.9 units/min"}
            },
            "renalAdjustment": {
                "creatinineClearanceMethod": "Not applicable",
                "requiresDialysisAdjustment": False,
                "adjustments": {}
            }
        },
        "contraindications": {
            "absolute": ["None in cardiac arrest"],
            "relative": ["Coronary artery disease", "Peripheral vascular disease"],
            "allergies": [],
            "diseaseStates": []
        },
        "adverseEffects": {
            "common": {"decreased cardiac output": "Variable", "arrhythmias": "Variable"},
            "serious": {"myocardial ischemia": "Variable", "peripheral ischemia": "Variable", "SIADH": "Rare"},
            "blackBoxWarnings": [],
            "monitoring": "Continuous cardiac monitoring, BP, urine output, ECG"
        },
        "pregnancyLactation": {
            "fdaCategory": "C",
            "pregnancyRisk": "Moderate",
            "lactationRisk": "Unknown",
            "pregnancyGuidance": "May cause uterine contractions",
            "breastfeedingGuidance": "Unknown excretion",
            "infantRiskCategory": "L3"
        }
    },
    {
        "medicationId": "MED-PHEN-001",
        "genericName": "Phenylephrine",
        "brandNames": ["Neo-Synephrine"],
        "rxNormCode": "8163",
        "ndcCode": "0409-6454",
        "atcCode": "C01CA06",
        "category": "cardiovascular",
        "subcategory": "vasopressors",
        "classification": {
            "therapeuticClass": "Vasopressor",
            "pharmacologicClass": "Alpha-1 adrenergic agonist",
            "chemicalClass": "Sympathomimetic",
            "category": "Cardiovascular",
            "subcategories": ["Vasopressor"],
            "highAlert": True,
            "blackBoxWarning": False
        },
        "adultDosing": {
            "standard": {
                "dose": "0.5-1.4 mcg/kg/min",
                "route": "IV continuous infusion",
                "frequency": "Continuous",
                "duration": "Until hemodynamic stability",
                "maxDailyDose": "Variable - titrate to effect",
                "infusionDuration": "Continuous"
            },
            "indicationBased": {
                "hypotension": {"dose": "40-180 mcg/min", "frequency": "titrate to MAP goal"},
                "spinal_anesthesia_hypotension": {"dose": "100-200 mcg IV bolus", "frequency": "as needed"}
            },
            "renalAdjustment": {
                "creatinineClearanceMethod": "Not applicable",
                "requiresDialysisAdjustment": False,
                "adjustments": {}
            }
        },
        "contraindications": {
            "absolute": ["Severe hypertension"],
            "relative": ["Coronary artery disease", "Bradycardia"],
            "allergies": ["sulfite"],
            "diseaseStates": []
        },
        "adverseEffects": {
            "common": {"bradycardia": "Reflex", "hypertension": "Variable"},
            "serious": {"extravasation necrosis": "If peripheral", "arrhythmias": "Rare"},
            "blackBoxWarnings": [],
            "monitoring": "Continuous BP, HR, IV site"
        },
        "pregnancyLactation": {
            "fdaCategory": "C",
            "pregnancyRisk": "Moderate",
            "lactationRisk": "Unknown",
            "pregnancyGuidance": "May reduce uterine blood flow",
            "breastfeedingGuidance": "Unknown excretion",
            "infantRiskCategory": "L3"
        }
    },
    {
        "medicationId": "MED-DOBU-001",
        "genericName": "Dobutamine",
        "brandNames": ["Dobutrex"],
        "rxNormCode": "3616",
        "ndcCode": "0409-1245",
        "atcCode": "C01CA07",
        "category": "cardiovascular",
        "subcategory": "inotropes",
        "classification": {
            "therapeuticClass": "Inotrope",
            "pharmacologicClass": "Beta-1 adrenergic agonist",
            "chemicalClass": "Synthetic catecholamine",
            "category": "Cardiovascular",
            "subcategories": ["Inotrope", "Heart failure"],
            "highAlert": True,
            "blackBoxWarning": False
        },
        "adultDosing": {
            "standard": {
                "dose": "2.5-20 mcg/kg/min",
                "route": "IV continuous infusion",
                "frequency": "Continuous",
                "duration": "Variable",
                "maxDailyDose": "40 mcg/kg/min (rarely)",
                "infusionDuration": "Continuous"
            },
            "indicationBased": {
                "cardiogenic_shock": {"dose": "2.5-20 mcg/kg/min", "frequency": "titrate to cardiac output"},
                "stress_echo": {"dose": "5-40 mcg/kg/min", "frequency": "incremental increase"}
            },
            "renalAdjustment": {
                "creatinineClearanceMethod": "Not applicable",
                "requiresDialysisAdjustment": False,
                "adjustments": {}
            }
        },
        "contraindications": {
            "absolute": ["Idiopathic hypertrophic subaortic stenosis"],
            "relative": ["Atrial fibrillation", "Hypovolemia"],
            "allergies": ["sulfite"],
            "diseaseStates": []
        },
        "adverseEffects": {
            "common": {"tachycardia": "28%", "increased BP": "Variable"},
            "serious": {"arrhythmias": "5%", "myocardial ischemia": "Variable"},
            "blackBoxWarnings": [],
            "monitoring": "Continuous BP, HR, ECG, cardiac output"
        },
        "pregnancyLactation": {
            "fdaCategory": "B",
            "pregnancyRisk": "Low",
            "lactationRisk": "Unknown",
            "pregnancyGuidance": "Compatible with pregnancy in critical situations",
            "breastfeedingGuidance": "Unknown excretion",
            "infantRiskCategory": "L3"
        }
    },
    {
        "medicationId": "MED-MILR-001",
        "genericName": "Milrinone",
        "brandNames": ["Primacor"],
        "rxNormCode": "30131",
        "ndcCode": "0143-9864",
        "atcCode": "C01CE02",
        "category": "cardiovascular",
        "subcategory": "inotropes",
        "classification": {
            "therapeuticClass": "Inotrope/Vasodilator",
            "pharmacologicClass": "Phosphodiesterase-3 inhibitor",
            "chemicalClass": "Bipyridine derivative",
            "category": "Cardiovascular",
            "subcategories": ["Inotrope", "Heart failure"],
            "highAlert": True,
            "blackBoxWarning": False
        },
        "adultDosing": {
            "standard": {
                "dose": "0.375-0.75 mcg/kg/min",
                "route": "IV continuous infusion",
                "frequency": "Continuous",
                "duration": "Variable",
                "maxDailyDose": "0.75 mcg/kg/min",
                "infusionDuration": "Loading: 50 mcg/kg over 10 min, then continuous"
            },
            "indicationBased": {
                "acute_heart_failure": {"dose": "0.375-0.75 mcg/kg/min", "frequency": "with or without loading dose"}
            },
            "renalAdjustment": {
                "creatinineClearanceMethod": "Cockcroft-Gault",
                "requiresDialysisAdjustment": False,
                "adjustments": {
                    "50": {"crClRange": "50 mL/min", "adjustedDose": "0.43 mcg/kg/min"},
                    "40": {"crClRange": "40 mL/min", "adjustedDose": "0.38 mcg/kg/min"},
                    "30": {"crClRange": "30 mL/min", "adjustedDose": "0.33 mcg/kg/min"},
                    "20": {"crClRange": "20 mL/min", "adjustedDose": "0.28 mcg/kg/min"}
                }
            }
        },
        "contraindications": {
            "absolute": ["Hypersensitivity to milrinone"],
            "relative": ["Severe aortic or pulmonary stenosis", "Hypertrophic cardiomyopathy"],
            "allergies": [],
            "diseaseStates": []
        },
        "adverseEffects": {
            "common": {"ventricular arrhythmias": "12%", "hypotension": "3%", "headache": "3%"},
            "serious": {"sustained ventricular tachycardia": "Rare", "thrombocytopenia": "Rare"},
            "blackBoxWarnings": [],
            "monitoring": "Continuous BP, HR, ECG, cardiac output, platelets"
        },
        "pregnancyLactation": {
            "fdaCategory": "C",
            "pregnancyRisk": "Moderate",
            "lactationRisk": "Unknown",
            "pregnancyGuidance": "Use if benefit outweighs risk",
            "breastfeedingGuidance": "Unknown excretion",
            "infantRiskCategory": "L3"
        }
    }
]

# Continue with additional cardiovascular medications in next part...
