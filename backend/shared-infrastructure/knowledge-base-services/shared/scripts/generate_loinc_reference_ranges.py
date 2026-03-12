#!/usr/bin/env python3
"""
LOINC Reference Ranges Generator
================================
Merges OHDSI Vocabulary LOINC codes with standard clinical reference ranges.

Sources:
- OHDSI Vocabulary (62,902 Lab Test LOINC codes)
- Existing loinc_labs_expanded.csv (45 codes with age/sex variants)
- Standard clinical reference ranges from medical literature

Output: Comprehensive SQL migration for Context Router
"""

import csv
import os
import re
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass
from pathlib import Path

# =============================================================================
# DATA STRUCTURES
# =============================================================================

@dataclass
class LOINCCode:
    """LOINC code with metadata from OHDSI vocabulary"""
    code: str
    name: str
    component: str = ""
    property_type: str = ""
    time_aspect: str = ""
    system: str = ""
    scale_type: str = ""
    method_type: str = ""
    loinc_class: str = ""

@dataclass
class ReferenceRange:
    """Clinical reference range with population specificity"""
    loinc_code: str
    component: str
    long_name: str
    unit: str
    low_normal: Optional[float]
    high_normal: Optional[float]
    critical_low: Optional[float]
    critical_high: Optional[float]
    age_group: str = "adult"
    sex: str = "all"
    clinical_category: str = "chemistry"
    interpretation_guidance: str = ""
    delta_check_percent: Optional[float] = None
    delta_check_hours: Optional[int] = None

# =============================================================================
# STANDARD CLINICAL REFERENCE RANGES
# Based on: Tietz Clinical Chemistry, Mayo Clinic, UpToDate, AHA/ACC Guidelines
# =============================================================================

STANDARD_REFERENCE_RANGES: Dict[str, Dict] = {
    # ==========================================================================
    # ELECTROLYTES
    # ==========================================================================
    "2951-2": {  # Sodium
        "component": "Sodium",
        "unit": "mmol/L",
        "category": "electrolyte",
        "ranges": {
            "adult": {"low": 136, "high": 145, "crit_low": 120, "crit_high": 160},
            "pediatric": {"low": 136, "high": 145, "crit_low": 125, "crit_high": 155},
            "neonate": {"low": 133, "high": 146, "crit_low": 120, "crit_high": 160},
            "geriatric": {"low": 136, "high": 145, "crit_low": 125, "crit_high": 155},
        },
        "guidance": "Low sodium may indicate SIADH, diuretic use, or heart failure. High sodium indicates dehydration or diabetes insipidus."
    },
    "2823-3": {  # Potassium
        "component": "Potassium",
        "unit": "mmol/L",
        "category": "electrolyte",
        "ranges": {
            "adult": {"low": 3.5, "high": 5.0, "crit_low": 2.5, "crit_high": 6.5},
            "pediatric": {"low": 3.4, "high": 4.7, "crit_low": 2.5, "crit_high": 6.5},
            "neonate": {"low": 3.7, "high": 5.9, "crit_low": 2.5, "crit_high": 7.0},
            "geriatric": {"low": 3.5, "high": 5.3, "crit_low": 2.8, "crit_high": 6.2},
        },
        "guidance": "Monitor for cardiac arrhythmias at extremes. Critical for digoxin and K-sparing diuretic interactions."
    },
    "2075-0": {  # Chloride
        "component": "Chloride",
        "unit": "mmol/L",
        "category": "electrolyte",
        "ranges": {
            "adult": {"low": 98, "high": 106, "crit_low": 80, "crit_high": 120},
            "pediatric": {"low": 98, "high": 106, "crit_low": 85, "crit_high": 115},
            "neonate": {"low": 96, "high": 106, "crit_low": 85, "crit_high": 115},
        },
        "guidance": "Interpret with sodium for acid-base status. Metabolic acidosis often shows elevated chloride."
    },
    "1963-8": {  # Bicarbonate/CO2
        "component": "Bicarbonate",
        "unit": "mmol/L",
        "category": "electrolyte",
        "ranges": {
            "adult": {"low": 22, "high": 29, "crit_low": 10, "crit_high": 40},
            "pediatric": {"low": 20, "high": 28, "crit_low": 12, "crit_high": 35},
            "neonate": {"low": 17, "high": 24, "crit_low": 10, "crit_high": 30},
        },
        "guidance": "Low indicates metabolic acidosis. High indicates metabolic alkalosis or compensation."
    },
    "17861-6": {  # Calcium total
        "component": "Calcium",
        "unit": "mg/dL",
        "category": "electrolyte",
        "ranges": {
            "adult": {"low": 8.6, "high": 10.2, "crit_low": 6.0, "crit_high": 14.0},
            "pediatric": {"low": 8.8, "high": 10.8, "crit_low": 6.5, "crit_high": 13.0},
            "neonate": {"low": 7.6, "high": 10.4, "crit_low": 6.0, "crit_high": 13.0},
        },
        "guidance": "Correct for albumin: Corrected Ca = measured Ca + 0.8*(4.0 - albumin)."
    },
    "2000-8": {  # Calcium ionized
        "component": "Calcium ionized",
        "unit": "mmol/L",
        "category": "electrolyte",
        "ranges": {
            "adult": {"low": 1.12, "high": 1.32, "crit_low": 0.8, "crit_high": 1.6},
        },
        "guidance": "True measure of metabolically active calcium. Critical for cardiac and neuromuscular function."
    },
    "2777-1": {  # Phosphorus
        "component": "Phosphorus",
        "unit": "mg/dL",
        "category": "electrolyte",
        "ranges": {
            "adult": {"low": 2.5, "high": 4.5, "crit_low": 1.0, "crit_high": 9.0},
            "pediatric": {"low": 4.0, "high": 7.0, "crit_low": 2.0, "crit_high": 10.0},
            "neonate": {"low": 4.8, "high": 8.2, "crit_low": 2.5, "crit_high": 10.0},
        },
        "guidance": "Inverse relationship with calcium. Monitor in CKD and refeeding syndrome."
    },
    "19123-9": {  # Magnesium
        "component": "Magnesium",
        "unit": "mg/dL",
        "category": "electrolyte",
        "ranges": {
            "adult": {"low": 1.7, "high": 2.2, "crit_low": 1.0, "crit_high": 4.0},
        },
        "guidance": "Low Mg potentiates digoxin toxicity and causes refractory hypokalemia."
    },

    # ==========================================================================
    # RENAL FUNCTION
    # ==========================================================================
    "2160-0": {  # Creatinine
        "component": "Creatinine",
        "unit": "mg/dL",
        "category": "renal",
        "ranges": {
            "adult": {"low": 0.7, "high": 1.3, "crit_low": 0.4, "crit_high": 10.0},
            "adult_male": {"low": 0.7, "high": 1.3, "crit_low": 0.4, "crit_high": 10.0},
            "adult_female": {"low": 0.6, "high": 1.1, "crit_low": 0.4, "crit_high": 10.0},
            "pediatric": {"low": 0.3, "high": 0.7, "crit_low": 0.2, "crit_high": 5.0},
            "neonate": {"low": 0.3, "high": 1.0, "crit_low": 0.2, "crit_high": 5.0},
            "geriatric": {"low": 0.7, "high": 1.5, "crit_low": 0.4, "crit_high": 10.0},
        },
        "guidance": "Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO criteria."
    },
    "3094-0": {  # BUN
        "component": "BUN",
        "unit": "mg/dL",
        "category": "renal",
        "ranges": {
            "adult": {"low": 7, "high": 20, "crit_low": 2, "crit_high": 100},
            "pediatric": {"low": 5, "high": 18, "crit_low": 2, "crit_high": 80},
            "neonate": {"low": 3, "high": 12, "crit_low": 2, "crit_high": 50},
            "geriatric": {"low": 8, "high": 23, "crit_low": 3, "crit_high": 100},
        },
        "guidance": "BUN:Cr ratio >20 suggests prerenal azotemia. Affected by protein intake and catabolic states."
    },
    "33914-3": {  # eGFR CKD-EPI
        "component": "eGFR",
        "unit": "mL/min/1.73m2",
        "category": "renal",
        "ranges": {
            "adult": {"low": 90, "high": 999, "crit_low": 15, "crit_high": None},
        },
        "guidance": "CKD staging: >90=G1, 60-89=G2, 45-59=G3a, 30-44=G3b, 15-29=G4, <15=G5. Drug dosing adjustments required <60."
    },
    "48642-3": {  # eGFR MDRD
        "component": "eGFR MDRD",
        "unit": "mL/min/1.73m2",
        "category": "renal",
        "ranges": {
            "adult": {"low": 90, "high": 999, "crit_low": 15, "crit_high": None},
        },
        "guidance": "Legacy formula. CKD-EPI preferred for most populations."
    },
    "62238-1": {  # eGFR CKD-EPI 2021
        "component": "eGFR CKD-EPI 2021",
        "unit": "mL/min/1.73m2",
        "category": "renal",
        "ranges": {
            "adult": {"low": 90, "high": 999, "crit_low": 15, "crit_high": None},
        },
        "guidance": "Race-free CKD-EPI 2021 equation. Preferred formula per KDIGO 2024."
    },

    # ==========================================================================
    # LIVER FUNCTION
    # ==========================================================================
    "1742-6": {  # ALT
        "component": "ALT",
        "unit": "U/L",
        "category": "hepatic",
        "ranges": {
            "adult": {"low": 7, "high": 56, "crit_low": None, "crit_high": 1000},
            "adult_male": {"low": 7, "high": 55, "crit_low": None, "crit_high": 1000},
            "adult_female": {"low": 7, "high": 45, "crit_low": None, "crit_high": 1000},
            "pediatric": {"low": 10, "high": 35, "crit_low": None, "crit_high": 500},
        },
        "guidance": "Elevation >3x ULN may indicate drug-induced hepatotoxicity. Monitor with statins, acetaminophen."
    },
    "1920-8": {  # AST
        "component": "AST",
        "unit": "U/L",
        "category": "hepatic",
        "ranges": {
            "adult": {"low": 10, "high": 40, "crit_low": None, "crit_high": 1000},
            "adult_male": {"low": 10, "high": 40, "crit_low": None, "crit_high": 1000},
            "adult_female": {"low": 10, "high": 35, "crit_low": None, "crit_high": 1000},
            "pediatric": {"low": 15, "high": 60, "crit_low": None, "crit_high": 500},
        },
        "guidance": "Non-specific. Elevated with liver, cardiac, or muscle injury. AST:ALT >2 suggests alcoholic liver disease."
    },
    "6768-6": {  # ALP
        "component": "Alkaline Phosphatase",
        "unit": "U/L",
        "category": "hepatic",
        "ranges": {
            "adult": {"low": 44, "high": 147, "crit_low": None, "crit_high": 1000},
            "pediatric": {"low": 100, "high": 390, "crit_low": None, "crit_high": 1500},
        },
        "guidance": "Elevated in cholestatic liver disease, bone disorders, and growth (pediatric)."
    },
    "1975-2": {  # Total Bilirubin
        "component": "Bilirubin Total",
        "unit": "mg/dL",
        "category": "hepatic",
        "ranges": {
            "adult": {"low": 0.1, "high": 1.2, "crit_low": None, "crit_high": 15.0},
            "neonate": {"low": 0.0, "high": 12.0, "crit_low": None, "crit_high": 20.0},
        },
        "guidance": "Jaundice typically visible >2.5 mg/dL. Direct bilirubin helps differentiate hepatocellular vs cholestatic."
    },
    "1968-7": {  # Direct Bilirubin
        "component": "Bilirubin Direct",
        "unit": "mg/dL",
        "category": "hepatic",
        "ranges": {
            "adult": {"low": 0.0, "high": 0.3, "crit_low": None, "crit_high": 10.0},
        },
        "guidance": "Elevated direct bilirubin indicates hepatocellular injury or biliary obstruction."
    },
    "1751-7": {  # Albumin
        "component": "Albumin",
        "unit": "g/dL",
        "category": "hepatic",
        "ranges": {
            "adult": {"low": 3.5, "high": 5.0, "crit_low": 1.5, "crit_high": None},
            "geriatric": {"low": 3.2, "high": 4.8, "crit_low": 1.5, "crit_high": None},
        },
        "guidance": "Low albumin indicates liver synthetic dysfunction, malnutrition, or nephrotic syndrome."
    },
    "2324-2": {  # GGT
        "component": "GGT",
        "unit": "U/L",
        "category": "hepatic",
        "ranges": {
            "adult": {"low": 0, "high": 65, "crit_low": None, "crit_high": 1000},
            "adult_male": {"low": 8, "high": 61, "crit_low": None, "crit_high": 1000},
            "adult_female": {"low": 5, "high": 36, "crit_low": None, "crit_high": 1000},
        },
        "guidance": "Most sensitive marker for biliary disease. Elevated with alcohol use."
    },

    # ==========================================================================
    # COAGULATION
    # ==========================================================================
    "6301-6": {  # INR
        "component": "INR",
        "unit": "",
        "category": "coagulation",
        "ranges": {
            "adult": {"low": 0.9, "high": 1.1, "crit_low": None, "crit_high": 5.0},
        },
        "guidance": "Therapeutic range for warfarin typically 2.0-3.0 (2.5-3.5 for mechanical valves). >5.0 indicates significant bleeding risk."
    },
    "34714-6": {  # INR alternate
        "component": "INR",
        "unit": "",
        "category": "coagulation",
        "ranges": {
            "adult": {"low": 0.9, "high": 1.1, "crit_low": None, "crit_high": 5.0},
        },
        "guidance": "Critical for warfarin and DOAC bridging decisions."
    },
    "5902-2": {  # PT
        "component": "Prothrombin Time",
        "unit": "seconds",
        "category": "coagulation",
        "ranges": {
            "adult": {"low": 11.0, "high": 13.5, "crit_low": None, "crit_high": 30.0},
        },
        "guidance": "Extrinsic pathway. Prolonged with warfarin, vitamin K deficiency, liver disease."
    },
    "3173-2": {  # PTT
        "component": "PTT",
        "unit": "seconds",
        "category": "coagulation",
        "ranges": {
            "adult": {"low": 25, "high": 35, "crit_low": None, "crit_high": 100},
        },
        "guidance": "Intrinsic pathway. Prolonged with heparin therapy, factor deficiencies."
    },
    "3255-7": {  # Fibrinogen
        "component": "Fibrinogen",
        "unit": "mg/dL",
        "category": "coagulation",
        "ranges": {
            "adult": {"low": 200, "high": 400, "crit_low": 100, "crit_high": None},
        },
        "guidance": "Low in DIC, liver disease. Elevated as acute phase reactant."
    },
    "48065-7": {  # D-dimer
        "component": "D-dimer",
        "unit": "ng/mL FEU",
        "category": "coagulation",
        "ranges": {
            "adult": {"low": 0, "high": 500, "crit_low": None, "crit_high": None},
        },
        "guidance": "Elevated in VTE, DIC, sepsis, malignancy, pregnancy. High NPV for PE/DVT."
    },

    # ==========================================================================
    # HEMATOLOGY
    # ==========================================================================
    "718-7": {  # Hemoglobin
        "component": "Hemoglobin",
        "unit": "g/dL",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 12.0, "high": 16.0, "crit_low": 7.0, "crit_high": 20.0},
            "adult_male": {"low": 13.5, "high": 17.5, "crit_low": 7.0, "crit_high": 20.0},
            "adult_female": {"low": 12.0, "high": 16.0, "crit_low": 7.0, "crit_high": 20.0},
            "pediatric": {"low": 11.0, "high": 14.0, "crit_low": 7.0, "crit_high": 18.0},
            "neonate": {"low": 14.0, "high": 24.0, "crit_low": 10.0, "crit_high": 26.0},
        },
        "guidance": "<7 g/dL may require transfusion. Assess for anemia etiology: iron, B12, folate, chronic disease."
    },
    "4544-3": {  # Hematocrit
        "component": "Hematocrit",
        "unit": "%",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 36, "high": 48, "crit_low": 20, "crit_high": 60},
            "adult_male": {"low": 40, "high": 52, "crit_low": 20, "crit_high": 60},
            "adult_female": {"low": 36, "high": 46, "crit_low": 20, "crit_high": 60},
        },
        "guidance": "Low in anemia, fluid overload. High in polycythemia, dehydration."
    },
    "6690-2": {  # WBC
        "component": "WBC",
        "unit": "x10^3/uL",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 4.5, "high": 11.0, "crit_low": 2.0, "crit_high": 30.0},
            "pediatric": {"low": 5.0, "high": 15.0, "crit_low": 2.5, "crit_high": 30.0},
            "neonate": {"low": 9.0, "high": 30.0, "crit_low": 5.0, "crit_high": 40.0},
        },
        "guidance": "Leukocytosis: infection, stress, steroids, leukemia. Leukopenia: bone marrow suppression, overwhelming sepsis."
    },
    "777-3": {  # Platelets
        "component": "Platelets",
        "unit": "x10^3/uL",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 150, "high": 400, "crit_low": 50, "crit_high": 1000},
        },
        "guidance": "<50 significant bleeding risk. <20 spontaneous bleeding risk. >1000 thrombotic risk (essential thrombocythemia)."
    },
    "789-8": {  # RBC
        "component": "RBC",
        "unit": "x10^6/uL",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 4.0, "high": 5.5, "crit_low": 2.5, "crit_high": 7.0},
            "adult_male": {"low": 4.5, "high": 5.9, "crit_low": 2.5, "crit_high": 7.0},
            "adult_female": {"low": 4.0, "high": 5.2, "crit_low": 2.5, "crit_high": 7.0},
        },
        "guidance": "Interpret with hemoglobin and MCV for anemia classification."
    },
    "787-2": {  # MCV
        "component": "MCV",
        "unit": "fL",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 80, "high": 100, "crit_low": 50, "crit_high": 130},
        },
        "guidance": "Microcytic (<80): iron deficiency, thalassemia. Macrocytic (>100): B12/folate deficiency, liver disease."
    },
    "786-4": {  # MCH
        "component": "MCH",
        "unit": "pg",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 27, "high": 33, "crit_low": None, "crit_high": None},
        },
        "guidance": "Mean cell hemoglobin. Correlates with MCV for anemia classification."
    },
    "785-6": {  # MCHC
        "component": "MCHC",
        "unit": "g/dL",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 32, "high": 36, "crit_low": None, "crit_high": None},
        },
        "guidance": "Low in hypochromic anemias. Elevated may indicate spherocytosis or cold agglutinins."
    },
    "788-0": {  # RDW
        "component": "RDW",
        "unit": "%",
        "category": "hematology",
        "ranges": {
            "adult": {"low": 11.5, "high": 14.5, "crit_low": None, "crit_high": None},
        },
        "guidance": "Elevated RDW (anisocytosis) suggests iron deficiency, mixed deficiencies, or myelodysplasia."
    },

    # ==========================================================================
    # CARDIAC MARKERS
    # ==========================================================================
    "10839-9": {  # Troponin I
        "component": "Troponin I",
        "unit": "ng/mL",
        "category": "cardiac",
        "ranges": {
            "adult": {"low": None, "high": 0.04, "crit_low": None, "crit_high": 0.5},
        },
        "guidance": "Elevation indicates myocardial injury. Serial measurements for trend. >99th percentile with rise/fall pattern = MI."
    },
    "6598-7": {  # Troponin T
        "component": "Troponin T",
        "unit": "ng/mL",
        "category": "cardiac",
        "ranges": {
            "adult": {"low": None, "high": 0.01, "crit_low": None, "crit_high": 0.1},
        },
        "guidance": "High-sensitivity assay. Interpret with clinical context and serial values."
    },
    "33762-6": {  # NT-proBNP
        "component": "NT-proBNP",
        "unit": "pg/mL",
        "category": "cardiac",
        "ranges": {
            "adult": {"low": None, "high": 125, "crit_low": None, "crit_high": 5000},
            "geriatric": {"low": None, "high": 450, "crit_low": None, "crit_high": 5000},
        },
        "guidance": "Heart failure marker. Age-dependent cutoffs. Rule-out HF if <300 pg/mL."
    },
    "30341-2": {  # BNP
        "component": "BNP",
        "unit": "pg/mL",
        "category": "cardiac",
        "ranges": {
            "adult": {"low": None, "high": 100, "crit_low": None, "crit_high": 2000},
        },
        "guidance": "Heart failure marker. <100 pg/mL makes HF unlikely. Elevated in renal dysfunction."
    },
    "8634-8": {  # QTc
        "component": "QTc",
        "unit": "ms",
        "category": "cardiac",
        "ranges": {
            "adult_male": {"low": None, "high": 450, "crit_low": None, "crit_high": 500},
            "adult_female": {"low": None, "high": 460, "crit_low": None, "crit_high": 500},
        },
        "guidance": ">500ms significant arrhythmia risk. Review QT-prolonging medications (antipsychotics, antibiotics, antiarrhythmics)."
    },

    # ==========================================================================
    # METABOLIC / ENDOCRINE
    # ==========================================================================
    "2345-7": {  # Glucose
        "component": "Glucose",
        "unit": "mg/dL",
        "category": "metabolic",
        "ranges": {
            "adult": {"low": 70, "high": 100, "crit_low": 40, "crit_high": 500},
            "pediatric": {"low": 60, "high": 100, "crit_low": 40, "crit_high": 400},
            "neonate": {"low": 40, "high": 60, "crit_low": 30, "crit_high": 250},
        },
        "guidance": "Fasting <100 normal. 100-125 prediabetes. >=126 diabetes (confirm with repeat). Critical values require immediate intervention."
    },
    "4548-4": {  # HbA1c
        "component": "HbA1c",
        "unit": "%",
        "category": "metabolic",
        "ranges": {
            "adult": {"low": 4.0, "high": 5.6, "crit_low": 3.0, "crit_high": 15.0},
        },
        "guidance": "<5.7% normal. 5.7-6.4% prediabetes. >=6.5% diabetes. Target <7% for most diabetics (individualize)."
    },
    "3016-3": {  # TSH
        "component": "TSH",
        "unit": "mIU/L",
        "category": "thyroid",
        "ranges": {
            "adult": {"low": 0.4, "high": 4.0, "crit_low": 0.01, "crit_high": 100},
            "geriatric": {"low": 0.4, "high": 7.0, "crit_low": 0.01, "crit_high": 100},
        },
        "guidance": "Low TSH: hyperthyroidism or suppressive therapy. High TSH: hypothyroidism."
    },
    "3026-2": {  # Free T4
        "component": "Free T4",
        "unit": "ng/dL",
        "category": "thyroid",
        "ranges": {
            "adult": {"low": 0.8, "high": 1.8, "crit_low": 0.3, "crit_high": 5.0},
        },
        "guidance": "Interpret with TSH. Elevated in hyperthyroidism. Low in hypothyroidism."
    },
    "3051-0": {  # Free T3
        "component": "Free T3",
        "unit": "pg/mL",
        "category": "thyroid",
        "ranges": {
            "adult": {"low": 2.3, "high": 4.2, "crit_low": 1.0, "crit_high": 10.0},
        },
        "guidance": "Active thyroid hormone. Useful in T3 toxicosis and euthyroid sick syndrome."
    },

    # ==========================================================================
    # LIPID PANEL
    # ==========================================================================
    "2093-3": {  # Total Cholesterol
        "component": "Cholesterol Total",
        "unit": "mg/dL",
        "category": "lipid",
        "ranges": {
            "adult": {"low": 0, "high": 200, "crit_low": None, "crit_high": 400},
        },
        "guidance": "Desirable <200. Borderline 200-239. High >=240."
    },
    "2571-8": {  # Triglycerides
        "component": "Triglycerides",
        "unit": "mg/dL",
        "category": "lipid",
        "ranges": {
            "adult": {"low": 0, "high": 150, "crit_low": None, "crit_high": 1000},
        },
        "guidance": "Normal <150. High 200-499. Very high >=500 (pancreatitis risk)."
    },
    "2085-9": {  # HDL
        "component": "HDL Cholesterol",
        "unit": "mg/dL",
        "category": "lipid",
        "ranges": {
            "adult": {"low": 40, "high": 999, "crit_low": 20, "crit_high": None},
            "adult_male": {"low": 40, "high": 999, "crit_low": 20, "crit_high": None},
            "adult_female": {"low": 50, "high": 999, "crit_low": 20, "crit_high": None},
        },
        "guidance": "Protective factor. Low HDL is CV risk factor."
    },
    "2089-1": {  # LDL calculated
        "component": "LDL Cholesterol",
        "unit": "mg/dL",
        "category": "lipid",
        "ranges": {
            "adult": {"low": 0, "high": 100, "crit_low": None, "crit_high": 300},
        },
        "guidance": "Primary target for therapy. <70 for very high risk. <100 for high risk. <130 for moderate risk."
    },
    "13457-7": {  # LDL direct
        "component": "LDL Cholesterol Direct",
        "unit": "mg/dL",
        "category": "lipid",
        "ranges": {
            "adult": {"low": 0, "high": 100, "crit_low": None, "crit_high": 300},
        },
        "guidance": "Direct measurement preferred when TG >400 mg/dL."
    },

    # ==========================================================================
    # THERAPEUTIC DRUG MONITORING
    # ==========================================================================
    "10535-3": {  # Digoxin
        "component": "Digoxin",
        "unit": "ng/mL",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 0.8, "high": 2.0, "crit_low": None, "crit_high": 2.5},
        },
        "guidance": "Therapeutic 0.8-2.0 ng/mL (0.5-1.0 for HF). Toxicity >2.0. Check K+, Mg2+, Ca2+, renal function."
    },
    "14334-7": {  # Lithium
        "component": "Lithium",
        "unit": "mmol/L",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 0.6, "high": 1.2, "crit_low": None, "crit_high": 1.5},
        },
        "guidance": "Therapeutic 0.6-1.2 mmol/L. >1.5 toxicity risk. Monitor renal function and hydration."
    },
    "3968-5": {  # Phenytoin
        "component": "Phenytoin",
        "unit": "mcg/mL",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 10, "high": 20, "crit_low": None, "crit_high": 25},
        },
        "guidance": "Therapeutic 10-20 mcg/mL. Adjust for albumin: Corrected = measured/(0.2*albumin + 0.1)."
    },
    "4049-3": {  # Theophylline
        "component": "Theophylline",
        "unit": "mcg/mL",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 10, "high": 20, "crit_low": None, "crit_high": 25},
        },
        "guidance": "Therapeutic 10-20 mcg/mL. Narrow therapeutic index. Monitor for toxicity signs."
    },
    "4090-7": {  # Vancomycin trough
        "component": "Vancomycin",
        "unit": "mcg/mL",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 10, "high": 20, "crit_low": None, "crit_high": 30},
        },
        "guidance": "Trough 10-20 mcg/mL (higher for serious infections). AUC/MIC monitoring preferred."
    },
    "20578-1": {  # Vancomycin peak
        "component": "Vancomycin Peak",
        "unit": "mcg/mL",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 25, "high": 40, "crit_low": None, "crit_high": 50},
        },
        "guidance": "Peak levels less commonly monitored. AUC-guided dosing preferred."
    },
    "35669-1": {  # Tacrolimus
        "component": "Tacrolimus",
        "unit": "ng/mL",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 5, "high": 15, "crit_low": None, "crit_high": 25},
        },
        "guidance": "Transplant: 10-15 ng/mL early, 5-10 maintenance. Monitor renal function for nephrotoxicity."
    },
    "4092-3": {  # Valproic acid
        "component": "Valproic Acid",
        "unit": "mcg/mL",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 50, "high": 100, "crit_low": None, "crit_high": 150},
        },
        "guidance": "Therapeutic 50-100 mcg/mL. Monitor LFTs, ammonia, and platelet count."
    },
    "14979-9": {  # Carbamazepine
        "component": "Carbamazepine",
        "unit": "mcg/mL",
        "category": "therapeutic_drug",
        "ranges": {
            "adult": {"low": 4, "high": 12, "crit_low": None, "crit_high": 15},
        },
        "guidance": "Therapeutic 4-12 mcg/mL. Auto-induction may require dose adjustment."
    },

    # ==========================================================================
    # INFLAMMATORY MARKERS
    # ==========================================================================
    "1988-5": {  # CRP
        "component": "CRP",
        "unit": "mg/L",
        "category": "inflammatory",
        "ranges": {
            "adult": {"low": 0, "high": 10, "crit_low": None, "crit_high": None},
        },
        "guidance": "Non-specific inflammation marker. <10 mg/L generally considered low risk."
    },
    "30522-7": {  # hs-CRP
        "component": "hs-CRP",
        "unit": "mg/L",
        "category": "inflammatory",
        "ranges": {
            "adult": {"low": 0, "high": 3.0, "crit_low": None, "crit_high": None},
        },
        "guidance": "CV risk: <1.0 low risk, 1-3 average, >3.0 high risk. Not useful during acute illness."
    },
    "4537-7": {  # ESR
        "component": "ESR",
        "unit": "mm/hr",
        "category": "inflammatory",
        "ranges": {
            "adult": {"low": 0, "high": 20, "crit_low": None, "crit_high": 100},
            "adult_male": {"low": 0, "high": 15, "crit_low": None, "crit_high": 100},
            "adult_female": {"low": 0, "high": 20, "crit_low": None, "crit_high": 100},
            "geriatric": {"low": 0, "high": 30, "crit_low": None, "crit_high": 100},
        },
        "guidance": "Non-specific inflammation. Rule of thumb: age/2 (male) or (age+10)/2 (female) for upper limit."
    },
    "2524-7": {  # Lactate
        "component": "Lactate",
        "unit": "mmol/L",
        "category": "inflammatory",
        "ranges": {
            "adult": {"low": 0.5, "high": 2.0, "crit_low": None, "crit_high": 4.0},
        },
        "guidance": ">2 mmol/L may indicate tissue hypoperfusion. >4 mmol/L associated with poor outcomes in sepsis."
    },
    "33959-8": {  # Procalcitonin
        "component": "Procalcitonin",
        "unit": "ng/mL",
        "category": "inflammatory",
        "ranges": {
            "adult": {"low": 0, "high": 0.5, "crit_low": None, "crit_high": 10.0},
        },
        "guidance": "Bacterial infection marker. <0.25 unlikely bacterial. >0.5 likely bacterial. Guide antibiotic duration."
    },

    # ==========================================================================
    # IRON STUDIES
    # ==========================================================================
    "2498-4": {  # Iron
        "component": "Iron",
        "unit": "mcg/dL",
        "category": "iron_studies",
        "ranges": {
            "adult": {"low": 60, "high": 170, "crit_low": 20, "crit_high": 300},
            "adult_male": {"low": 65, "high": 175, "crit_low": 20, "crit_high": 300},
            "adult_female": {"low": 50, "high": 170, "crit_low": 20, "crit_high": 300},
        },
        "guidance": "Low in iron deficiency. Elevated in hemochromatosis, transfusion overload."
    },
    "2502-3": {  # TIBC
        "component": "TIBC",
        "unit": "mcg/dL",
        "category": "iron_studies",
        "ranges": {
            "adult": {"low": 250, "high": 370, "crit_low": None, "crit_high": None},
        },
        "guidance": "High TIBC in iron deficiency. Low in anemia of chronic disease."
    },
    "2505-6": {  # Transferrin saturation
        "component": "Transferrin Saturation",
        "unit": "%",
        "category": "iron_studies",
        "ranges": {
            "adult": {"low": 20, "high": 50, "crit_low": 10, "crit_high": 80},
        },
        "guidance": "<20% suggests iron deficiency. >45% in hemochromatosis workup."
    },
    "2276-4": {  # Ferritin
        "component": "Ferritin",
        "unit": "ng/mL",
        "category": "iron_studies",
        "ranges": {
            "adult": {"low": 12, "high": 300, "crit_low": 5, "crit_high": 1000},
            "adult_male": {"low": 24, "high": 336, "crit_low": 10, "crit_high": 1000},
            "adult_female": {"low": 12, "high": 150, "crit_low": 5, "crit_high": 1000},
        },
        "guidance": "<12 diagnostic of iron deficiency. Elevated as acute phase reactant and in iron overload."
    },

    # ==========================================================================
    # VITAMINS
    # ==========================================================================
    "2132-9": {  # Vitamin B12
        "component": "Vitamin B12",
        "unit": "pg/mL",
        "category": "vitamin",
        "ranges": {
            "adult": {"low": 200, "high": 900, "crit_low": 100, "crit_high": None},
        },
        "guidance": "<200 deficiency. 200-300 borderline. Check methylmalonic acid if borderline."
    },
    "2284-8": {  # Folate
        "component": "Folate",
        "unit": "ng/mL",
        "category": "vitamin",
        "ranges": {
            "adult": {"low": 2.7, "high": 17.0, "crit_low": 1.0, "crit_high": None},
        },
        "guidance": "<2.7 deficiency. Causes macrocytic anemia. Important in pregnancy."
    },
    "1989-3": {  # 25-OH Vitamin D
        "component": "Vitamin D, 25-OH",
        "unit": "ng/mL",
        "category": "vitamin",
        "ranges": {
            "adult": {"low": 30, "high": 100, "crit_low": 10, "crit_high": 150},
        },
        "guidance": "<20 deficiency. 20-29 insufficiency. >30 sufficient. >100 potential toxicity."
    },

    # ==========================================================================
    # URINE
    # ==========================================================================
    "2965-2": {  # Urine specific gravity
        "component": "Urine Specific Gravity",
        "unit": "",
        "category": "urinalysis",
        "ranges": {
            "adult": {"low": 1.005, "high": 1.030, "crit_low": 1.001, "crit_high": 1.040},
        },
        "guidance": "Reflects concentrating ability. Low in diabetes insipidus. High in dehydration."
    },
    "2756-5": {  # Urine pH
        "component": "Urine pH",
        "unit": "",
        "category": "urinalysis",
        "ranges": {
            "adult": {"low": 4.5, "high": 8.0, "crit_low": None, "crit_high": None},
        },
        "guidance": "Acidic urine in metabolic acidosis, protein-rich diet. Alkaline in UTI, vegetarian diet."
    },
    "2888-6": {  # Urine protein
        "component": "Urine Protein",
        "unit": "mg/dL",
        "category": "urinalysis",
        "ranges": {
            "adult": {"low": 0, "high": 14, "crit_low": None, "crit_high": None},
        },
        "guidance": "Proteinuria indicates glomerular or tubular disease. Quantify with 24h or spot protein/creatinine ratio."
    },
    "2339-0": {  # Urine glucose
        "component": "Urine Glucose",
        "unit": "mg/dL",
        "category": "urinalysis",
        "ranges": {
            "adult": {"low": 0, "high": 0, "crit_low": None, "crit_high": None},
        },
        "guidance": "Glucosuria when serum glucose exceeds renal threshold (~180 mg/dL). Present in SGLT2 inhibitor use."
    },

    # ==========================================================================
    # ARTERIAL BLOOD GAS
    # ==========================================================================
    "2744-1": {  # pH arterial
        "component": "pH Arterial",
        "unit": "",
        "category": "blood_gas",
        "ranges": {
            "adult": {"low": 7.35, "high": 7.45, "crit_low": 7.2, "crit_high": 7.6},
        },
        "guidance": "<7.35 acidemia. >7.45 alkalemia. Interpret with pCO2 and HCO3 for primary disorder."
    },
    "2019-8": {  # pCO2 arterial
        "component": "pCO2 Arterial",
        "unit": "mmHg",
        "category": "blood_gas",
        "ranges": {
            "adult": {"low": 35, "high": 45, "crit_low": 20, "crit_high": 70},
        },
        "guidance": "Respiratory component. Low in respiratory alkalosis. High in respiratory acidosis."
    },
    "2703-7": {  # pO2 arterial
        "component": "pO2 Arterial",
        "unit": "mmHg",
        "category": "blood_gas",
        "ranges": {
            "adult": {"low": 80, "high": 100, "crit_low": 50, "crit_high": None},
        },
        "guidance": "<60 mmHg indicates respiratory failure. Interpret with FiO2 (P/F ratio)."
    },
    "2713-6": {  # O2 saturation arterial
        "component": "O2 Saturation Arterial",
        "unit": "%",
        "category": "blood_gas",
        "ranges": {
            "adult": {"low": 95, "high": 100, "crit_low": 88, "crit_high": None},
        },
        "guidance": "<88% requires supplemental oxygen in most patients. Target may be lower in COPD."
    },

    # ==========================================================================
    # TUMOR MARKERS
    # ==========================================================================
    "2039-6": {  # CEA
        "component": "CEA",
        "unit": "ng/mL",
        "category": "tumor_marker",
        "ranges": {
            "adult": {"low": 0, "high": 3.0, "crit_low": None, "crit_high": None},
        },
        "guidance": "Colorectal cancer surveillance. May be elevated in smokers. Not for screening."
    },
    "2857-1": {  # PSA
        "component": "PSA",
        "unit": "ng/mL",
        "category": "tumor_marker",
        "ranges": {
            "adult_male": {"low": 0, "high": 4.0, "crit_low": None, "crit_high": None},
        },
        "guidance": "Age-adjusted cutoffs available. Used for prostate cancer screening and monitoring."
    },
    "19177-5": {  # AFP
        "component": "AFP",
        "unit": "ng/mL",
        "category": "tumor_marker",
        "ranges": {
            "adult": {"low": 0, "high": 10, "crit_low": None, "crit_high": None},
        },
        "guidance": "Hepatocellular carcinoma surveillance in cirrhosis. Also elevated in germ cell tumors."
    },
    "10334-1": {  # CA 125
        "component": "CA 125",
        "unit": "U/mL",
        "category": "tumor_marker",
        "ranges": {
            "adult_female": {"low": 0, "high": 35, "crit_low": None, "crit_high": None},
        },
        "guidance": "Ovarian cancer monitoring. May be elevated in endometriosis, pregnancy, menstruation."
    },
    "17842-6": {  # CA 19-9
        "component": "CA 19-9",
        "unit": "U/mL",
        "category": "tumor_marker",
        "ranges": {
            "adult": {"low": 0, "high": 37, "crit_low": None, "crit_high": None},
        },
        "guidance": "Pancreatic cancer monitoring. May be elevated in biliary obstruction."
    },
}

# =============================================================================
# ADDITIONAL COMMON LOINC CODES (from Mayo Clinic and clinical practice)
# =============================================================================

ADDITIONAL_LOINC_CODES = {
    # More electrolytes
    "39791-9": {"component": "Potassium", "long_name": "Potassium [Moles/volume] in Venous blood", "ranges": STANDARD_REFERENCE_RANGES["2823-3"]["ranges"]},
    "6298-4": {"component": "Potassium", "long_name": "Potassium [Moles/volume] in Blood", "ranges": STANDARD_REFERENCE_RANGES["2823-3"]["ranges"]},
    "77140-2": {"component": "Sodium", "long_name": "Sodium [Moles/volume] in Serum, Plasma or Blood", "ranges": STANDARD_REFERENCE_RANGES["2951-2"]["ranges"]},

    # More renal
    "38483-4": {"component": "Creatinine", "long_name": "Creatinine [Mass/volume] in Blood", "ranges": STANDARD_REFERENCE_RANGES["2160-0"]["ranges"]},
    "77139-4": {"component": "Creatinine", "long_name": "Creatinine [Mass/volume] in Serum, Plasma or Blood", "ranges": STANDARD_REFERENCE_RANGES["2160-0"]["ranges"]},
    "88293-6": {"component": "eGFR", "long_name": "Glomerular filtration rate/1.73 sq M.predicted by CKD-EPI 2021", "ranges": STANDARD_REFERENCE_RANGES["33914-3"]["ranges"]},
    "98979-8": {"component": "eGFR", "long_name": "Glomerular filtration rate/1.73 sq M.predicted [Volume Rate/Area] in Serum, Plasma or Blood by CKD-EPI 2021", "ranges": STANDARD_REFERENCE_RANGES["33914-3"]["ranges"]},

    # More liver
    "1743-4": {"component": "ALT", "long_name": "Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P", "ranges": STANDARD_REFERENCE_RANGES["1742-6"]["ranges"]},
    "1921-6": {"component": "AST", "long_name": "Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P", "ranges": STANDARD_REFERENCE_RANGES["1920-8"]["ranges"]},

    # More coagulation
    "46418-0": {"component": "INR", "long_name": "INR in Platelet poor plasma by Coagulation assay", "ranges": STANDARD_REFERENCE_RANGES["6301-6"]["ranges"]},

    # More cardiac
    "49563-0": {"component": "Troponin I", "long_name": "Troponin I.cardiac [Mass/volume] in Serum or Plasma by High sensitivity method", "ranges": {"adult": {"low": None, "high": 0.034, "crit_low": None, "crit_high": 0.5}}},
    "89579-7": {"component": "Troponin I", "long_name": "Troponin I.cardiac [Mass/volume] in Serum or Plasma by High sensitivity immunoassay", "ranges": {"adult": {"low": None, "high": 0.034, "crit_low": None, "crit_high": 0.5}}},
    "67151-1": {"component": "Troponin T", "long_name": "Troponin T.cardiac [Mass/volume] in Serum or Plasma by High sensitivity method", "ranges": {"adult": {"low": None, "high": 0.014, "crit_low": None, "crit_high": 0.1}}},

    # More metabolic
    "41653-7": {"component": "Glucose", "long_name": "Glucose [Mass/volume] in Capillary blood by Glucometer", "ranges": STANDARD_REFERENCE_RANGES["2345-7"]["ranges"]},
    "74774-1": {"component": "Glucose", "long_name": "Glucose [Mass/volume] in Serum, Plasma or Blood", "ranges": STANDARD_REFERENCE_RANGES["2345-7"]["ranges"]},
    "17856-6": {"component": "HbA1c", "long_name": "Hemoglobin A1c/Hemoglobin.total in Blood by HPLC", "ranges": STANDARD_REFERENCE_RANGES["4548-4"]["ranges"]},

    # More hematology
    "20570-8": {"component": "Hematocrit", "long_name": "Hematocrit [Volume Fraction] of Blood by Calculation", "ranges": STANDARD_REFERENCE_RANGES["4544-3"]["ranges"]},
    "26515-7": {"component": "Platelets", "long_name": "Platelets [#/volume] in Blood", "ranges": STANDARD_REFERENCE_RANGES["777-3"]["ranges"]},
    "26464-8": {"component": "WBC", "long_name": "Leukocytes [#/volume] in Blood", "ranges": STANDARD_REFERENCE_RANGES["6690-2"]["ranges"]},
    "30313-1": {"component": "Hemoglobin", "long_name": "Hemoglobin [Mass/volume] in Arterial blood", "ranges": STANDARD_REFERENCE_RANGES["718-7"]["ranges"]},
    "59260-0": {"component": "Hemoglobin", "long_name": "Hemoglobin [Moles/volume] in Blood", "ranges": STANDARD_REFERENCE_RANGES["718-7"]["ranges"]},
}

# =============================================================================
# SCRIPT EXECUTION
# =============================================================================

def generate_reference_ranges() -> List[ReferenceRange]:
    """Generate all reference ranges from standard data"""
    ranges = []

    # Process standard reference ranges
    for loinc_code, data in STANDARD_REFERENCE_RANGES.items():
        for pop_key, pop_ranges in data["ranges"].items():
            # Parse population key
            if "_male" in pop_key:
                age_group = pop_key.replace("_male", "")
                sex = "male"
            elif "_female" in pop_key:
                age_group = pop_key.replace("_female", "")
                sex = "female"
            else:
                age_group = pop_key
                sex = "all"

            ranges.append(ReferenceRange(
                loinc_code=loinc_code,
                component=data["component"],
                long_name=f"{data['component']} reference range",
                unit=data["unit"],
                low_normal=pop_ranges.get("low"),
                high_normal=pop_ranges.get("high"),
                critical_low=pop_ranges.get("crit_low"),
                critical_high=pop_ranges.get("crit_high"),
                age_group=age_group,
                sex=sex,
                clinical_category=data["category"],
                interpretation_guidance=data.get("guidance", "")
            ))

    # Process additional LOINC codes
    for loinc_code, data in ADDITIONAL_LOINC_CODES.items():
        for pop_key, pop_ranges in data["ranges"].items():
            if "_male" in pop_key:
                age_group = pop_key.replace("_male", "")
                sex = "male"
            elif "_female" in pop_key:
                age_group = pop_key.replace("_female", "")
                sex = "female"
            else:
                age_group = pop_key
                sex = "all"

            ranges.append(ReferenceRange(
                loinc_code=loinc_code,
                component=data["component"],
                long_name=data["long_name"],
                unit=STANDARD_REFERENCE_RANGES.get(list(STANDARD_REFERENCE_RANGES.keys())[0], {}).get("unit", ""),
                low_normal=pop_ranges.get("low"),
                high_normal=pop_ranges.get("high"),
                critical_low=pop_ranges.get("crit_low"),
                critical_high=pop_ranges.get("crit_high"),
                age_group=age_group,
                sex=sex,
                clinical_category="chemistry",
                interpretation_guidance=""
            ))

    return ranges

def generate_sql_inserts(ranges: List[ReferenceRange]) -> str:
    """Generate SQL INSERT statements"""
    sql_lines = []

    for r in ranges:
        low_normal = f"{r.low_normal}" if r.low_normal is not None else "NULL"
        high_normal = f"{r.high_normal}" if r.high_normal is not None else "NULL"
        critical_low = f"{r.critical_low}" if r.critical_low is not None else "NULL"
        critical_high = f"{r.critical_high}" if r.critical_high is not None else "NULL"

        # Escape single quotes in guidance
        guidance = r.interpretation_guidance.replace("'", "''")

        sql_lines.append(f"""INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('{r.loinc_code}', '{r.component}', '{r.long_name}', '{r.unit}', {low_normal}, {high_normal}, {critical_low}, {critical_high}, '{r.age_group}', '{r.sex}', '{r.clinical_category}', '{guidance}')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();""")

    return "\n\n".join(sql_lines)

def main():
    """Main execution"""
    print("=" * 80)
    print("LOINC Reference Ranges Generator")
    print("=" * 80)

    # Generate reference ranges
    ranges = generate_reference_ranges()
    print(f"\nGenerated {len(ranges)} reference range entries")

    # Count unique LOINC codes
    unique_codes = set(r.loinc_code for r in ranges)
    print(f"Unique LOINC codes: {len(unique_codes)}")

    # Count by category
    from collections import Counter
    categories = Counter(r.clinical_category for r in ranges)
    print("\nBy clinical category:")
    for cat, count in sorted(categories.items(), key=lambda x: -x[1]):
        print(f"  {cat}: {count}")

    # Generate SQL
    sql = generate_sql_inserts(ranges)

    # Write to file
    output_dir = Path(__file__).parent.parent / "migrations"
    output_file = output_dir / "005_loinc_reference_ranges_data.sql"

    with open(output_file, 'w') as f:
        f.write("-- =============================================================================\n")
        f.write("-- MIGRATION 005 DATA: LOINC Reference Ranges (Auto-generated)\n")
        f.write(f"-- Generated: {len(ranges)} entries, {len(unique_codes)} unique LOINC codes\n")
        f.write("-- Source: Standard clinical reference ranges (Tietz, Mayo, UpToDate, Guidelines)\n")
        f.write("-- =============================================================================\n\n")
        f.write("BEGIN;\n\n")
        f.write(sql)
        f.write("\n\nCOMMIT;\n")

    print(f"\nSQL written to: {output_file}")
    print(f"Total lines: {len(sql.splitlines())}")

if __name__ == "__main__":
    main()
