#!/usr/bin/env python3
"""
Expand LOINC Reference Ranges from OHDSI Vocabulary
====================================================
Extracts clinically important Lab Test LOINC codes from OHDSI vocabulary
and assigns standard reference ranges based on component analysis.

This creates 2000+ LOINC entries by:
1. Reading all Lab Test codes from OHDSI CONCEPT.csv
2. Categorizing by component type (chemistry, hematology, etc.)
3. Assigning standard reference ranges where patterns match
4. Flagging codes without standard ranges for manual review
"""

import csv
import re
import os
from pathlib import Path
from typing import Dict, List, Set, Optional, Tuple
from dataclasses import dataclass, field
from collections import defaultdict

# =============================================================================
# COMPONENT PATTERN MATCHING FOR REFERENCE RANGES
# =============================================================================

# Map common components to standard reference ranges
COMPONENT_RANGES = {
    # Electrolytes
    ("sodium", "na"): {"unit": "mmol/L", "low": 136, "high": 145, "crit_low": 120, "crit_high": 160, "category": "electrolyte"},
    ("potassium", "k"): {"unit": "mmol/L", "low": 3.5, "high": 5.0, "crit_low": 2.5, "crit_high": 6.5, "category": "electrolyte"},
    ("chloride", "cl"): {"unit": "mmol/L", "low": 98, "high": 106, "crit_low": 80, "crit_high": 120, "category": "electrolyte"},
    ("bicarbonate", "hco3", "co2"): {"unit": "mmol/L", "low": 22, "high": 29, "crit_low": 10, "crit_high": 40, "category": "electrolyte"},
    ("calcium",): {"unit": "mg/dL", "low": 8.6, "high": 10.2, "crit_low": 6.0, "crit_high": 14.0, "category": "electrolyte"},
    ("phosphorus", "phosphate"): {"unit": "mg/dL", "low": 2.5, "high": 4.5, "crit_low": 1.0, "crit_high": 9.0, "category": "electrolyte"},
    ("magnesium",): {"unit": "mg/dL", "low": 1.7, "high": 2.2, "crit_low": 1.0, "crit_high": 4.0, "category": "electrolyte"},

    # Renal
    ("creatinine",): {"unit": "mg/dL", "low": 0.7, "high": 1.3, "crit_low": 0.4, "crit_high": 10.0, "category": "renal"},
    ("urea nitrogen", "bun"): {"unit": "mg/dL", "low": 7, "high": 20, "crit_low": 2, "crit_high": 100, "category": "renal"},
    ("glomerular filtration", "egfr"): {"unit": "mL/min/1.73m2", "low": 90, "high": 999, "crit_low": 15, "crit_high": None, "category": "renal"},
    ("cystatin c",): {"unit": "mg/L", "low": 0.6, "high": 1.0, "crit_low": None, "crit_high": 3.0, "category": "renal"},
    ("uric acid",): {"unit": "mg/dL", "low": 2.5, "high": 7.0, "crit_low": None, "crit_high": 12.0, "category": "renal"},

    # Liver
    ("alanine aminotransferase", "alt"): {"unit": "U/L", "low": 7, "high": 56, "crit_low": None, "crit_high": 1000, "category": "hepatic"},
    ("aspartate aminotransferase", "ast"): {"unit": "U/L", "low": 10, "high": 40, "crit_low": None, "crit_high": 1000, "category": "hepatic"},
    ("alkaline phosphatase", "alp"): {"unit": "U/L", "low": 44, "high": 147, "crit_low": None, "crit_high": 1000, "category": "hepatic"},
    ("bilirubin",): {"unit": "mg/dL", "low": 0.1, "high": 1.2, "crit_low": None, "crit_high": 15.0, "category": "hepatic"},
    ("albumin",): {"unit": "g/dL", "low": 3.5, "high": 5.0, "crit_low": 1.5, "crit_high": None, "category": "hepatic"},
    ("gamma glutamyl", "ggt"): {"unit": "U/L", "low": 0, "high": 65, "crit_low": None, "crit_high": 1000, "category": "hepatic"},
    ("protein",): {"unit": "g/dL", "low": 6.0, "high": 8.0, "crit_low": 3.0, "crit_high": 12.0, "category": "hepatic"},
    ("ammonia",): {"unit": "umol/L", "low": 11, "high": 32, "crit_low": None, "crit_high": 100, "category": "hepatic"},
    ("lactate dehydrogenase", "ldh"): {"unit": "U/L", "low": 140, "high": 280, "crit_low": None, "crit_high": 1000, "category": "hepatic"},

    # Coagulation
    ("inr",): {"unit": "", "low": 0.9, "high": 1.1, "crit_low": None, "crit_high": 5.0, "category": "coagulation"},
    ("prothrombin time", "pt"): {"unit": "seconds", "low": 11.0, "high": 13.5, "crit_low": None, "crit_high": 30.0, "category": "coagulation"},
    ("thromboplastin", "ptt", "aptt"): {"unit": "seconds", "low": 25, "high": 35, "crit_low": None, "crit_high": 100, "category": "coagulation"},
    ("fibrinogen",): {"unit": "mg/dL", "low": 200, "high": 400, "crit_low": 100, "crit_high": None, "category": "coagulation"},
    ("d-dimer",): {"unit": "ng/mL", "low": 0, "high": 500, "crit_low": None, "crit_high": None, "category": "coagulation"},
    ("antithrombin",): {"unit": "%", "low": 80, "high": 120, "crit_low": 50, "crit_high": None, "category": "coagulation"},
    ("factor",): {"unit": "%", "low": 50, "high": 150, "crit_low": 20, "crit_high": None, "category": "coagulation"},

    # Hematology
    ("hemoglobin",): {"unit": "g/dL", "low": 12.0, "high": 16.0, "crit_low": 7.0, "crit_high": 20.0, "category": "hematology"},
    ("hematocrit",): {"unit": "%", "low": 36, "high": 48, "crit_low": 20, "crit_high": 60, "category": "hematology"},
    ("platelet",): {"unit": "x10^3/uL", "low": 150, "high": 400, "crit_low": 50, "crit_high": 1000, "category": "hematology"},
    ("leukocyte", "wbc", "white blood"): {"unit": "x10^3/uL", "low": 4.5, "high": 11.0, "crit_low": 2.0, "crit_high": 30.0, "category": "hematology"},
    ("erythrocyte", "rbc", "red blood"): {"unit": "x10^6/uL", "low": 4.0, "high": 5.5, "crit_low": 2.5, "crit_high": 7.0, "category": "hematology"},
    ("mcv", "mean cell volume", "mean corpuscular volume"): {"unit": "fL", "low": 80, "high": 100, "crit_low": 50, "crit_high": 130, "category": "hematology"},
    ("mch",): {"unit": "pg", "low": 27, "high": 33, "crit_low": None, "crit_high": None, "category": "hematology"},
    ("mchc",): {"unit": "g/dL", "low": 32, "high": 36, "crit_low": None, "crit_high": None, "category": "hematology"},
    ("rdw",): {"unit": "%", "low": 11.5, "high": 14.5, "crit_low": None, "crit_high": None, "category": "hematology"},
    ("reticulocyte",): {"unit": "%", "low": 0.5, "high": 2.5, "crit_low": None, "crit_high": None, "category": "hematology"},
    ("neutrophil",): {"unit": "%", "low": 40, "high": 70, "crit_low": None, "crit_high": None, "category": "hematology"},
    ("lymphocyte",): {"unit": "%", "low": 20, "high": 40, "crit_low": None, "crit_high": None, "category": "hematology"},
    ("monocyte",): {"unit": "%", "low": 2, "high": 8, "crit_low": None, "crit_high": None, "category": "hematology"},
    ("eosinophil",): {"unit": "%", "low": 1, "high": 4, "crit_low": None, "crit_high": None, "category": "hematology"},
    ("basophil",): {"unit": "%", "low": 0, "high": 1, "crit_low": None, "crit_high": None, "category": "hematology"},

    # Cardiac
    ("troponin",): {"unit": "ng/mL", "low": None, "high": 0.04, "crit_low": None, "crit_high": 0.5, "category": "cardiac"},
    ("bnp", "natriuretic"): {"unit": "pg/mL", "low": None, "high": 100, "crit_low": None, "crit_high": 2000, "category": "cardiac"},
    ("nt-probnp",): {"unit": "pg/mL", "low": None, "high": 125, "crit_low": None, "crit_high": 5000, "category": "cardiac"},
    ("creatine kinase", "ck"): {"unit": "U/L", "low": 30, "high": 200, "crit_low": None, "crit_high": 1000, "category": "cardiac"},
    ("ck-mb",): {"unit": "ng/mL", "low": None, "high": 5, "crit_low": None, "crit_high": 25, "category": "cardiac"},
    ("myoglobin",): {"unit": "ng/mL", "low": None, "high": 90, "crit_low": None, "crit_high": None, "category": "cardiac"},

    # Metabolic
    ("glucose",): {"unit": "mg/dL", "low": 70, "high": 100, "crit_low": 40, "crit_high": 500, "category": "metabolic"},
    ("hemoglobin a1c", "hba1c", "glycosylated hemoglobin"): {"unit": "%", "low": 4.0, "high": 5.6, "crit_low": 3.0, "crit_high": 15.0, "category": "metabolic"},

    # Thyroid
    ("thyrotropin", "tsh"): {"unit": "mIU/L", "low": 0.4, "high": 4.0, "crit_low": 0.01, "crit_high": 100, "category": "thyroid"},
    ("thyroxine", "t4"): {"unit": "ng/dL", "low": 0.8, "high": 1.8, "crit_low": 0.3, "crit_high": 5.0, "category": "thyroid"},
    ("triiodothyronine", "t3"): {"unit": "pg/mL", "low": 2.3, "high": 4.2, "crit_low": 1.0, "crit_high": 10.0, "category": "thyroid"},

    # Lipids
    ("cholesterol",): {"unit": "mg/dL", "low": 0, "high": 200, "crit_low": None, "crit_high": 400, "category": "lipid"},
    ("triglyceride",): {"unit": "mg/dL", "low": 0, "high": 150, "crit_low": None, "crit_high": 1000, "category": "lipid"},
    ("hdl",): {"unit": "mg/dL", "low": 40, "high": 999, "crit_low": 20, "crit_high": None, "category": "lipid"},
    ("ldl",): {"unit": "mg/dL", "low": 0, "high": 100, "crit_low": None, "crit_high": 300, "category": "lipid"},
    ("vldl",): {"unit": "mg/dL", "low": 0, "high": 30, "crit_low": None, "crit_high": 100, "category": "lipid"},
    ("lipoprotein",): {"unit": "mg/dL", "low": 0, "high": 30, "crit_low": None, "crit_high": 100, "category": "lipid"},
    ("apolipoprotein",): {"unit": "mg/dL", "low": 80, "high": 150, "crit_low": None, "crit_high": None, "category": "lipid"},

    # Inflammatory
    ("c-reactive protein", "crp"): {"unit": "mg/L", "low": 0, "high": 10, "crit_low": None, "crit_high": None, "category": "inflammatory"},
    ("erythrocyte sedimentation", "esr"): {"unit": "mm/hr", "low": 0, "high": 20, "crit_low": None, "crit_high": 100, "category": "inflammatory"},
    ("lactate",): {"unit": "mmol/L", "low": 0.5, "high": 2.0, "crit_low": None, "crit_high": 4.0, "category": "inflammatory"},
    ("procalcitonin",): {"unit": "ng/mL", "low": 0, "high": 0.5, "crit_low": None, "crit_high": 10.0, "category": "inflammatory"},
    ("interleukin",): {"unit": "pg/mL", "low": None, "high": None, "crit_low": None, "crit_high": None, "category": "inflammatory"},
    ("ferritin",): {"unit": "ng/mL", "low": 12, "high": 300, "crit_low": 5, "crit_high": 1000, "category": "inflammatory"},

    # Iron Studies
    ("iron",): {"unit": "mcg/dL", "low": 60, "high": 170, "crit_low": 20, "crit_high": 300, "category": "iron_studies"},
    ("tibc", "iron binding"): {"unit": "mcg/dL", "low": 250, "high": 370, "crit_low": None, "crit_high": None, "category": "iron_studies"},
    ("transferrin",): {"unit": "mg/dL", "low": 200, "high": 400, "crit_low": None, "crit_high": None, "category": "iron_studies"},

    # Vitamins
    ("vitamin b12", "cobalamin"): {"unit": "pg/mL", "low": 200, "high": 900, "crit_low": 100, "crit_high": None, "category": "vitamin"},
    ("folate", "folic acid"): {"unit": "ng/mL", "low": 2.7, "high": 17.0, "crit_low": 1.0, "crit_high": None, "category": "vitamin"},
    ("vitamin d", "25-oh"): {"unit": "ng/mL", "low": 30, "high": 100, "crit_low": 10, "crit_high": 150, "category": "vitamin"},
    ("vitamin a", "retinol"): {"unit": "mcg/dL", "low": 30, "high": 80, "crit_low": 10, "crit_high": 200, "category": "vitamin"},
    ("vitamin e", "tocopherol"): {"unit": "mg/L", "low": 5.5, "high": 17, "crit_low": None, "crit_high": None, "category": "vitamin"},
    ("vitamin c", "ascorbic"): {"unit": "mg/dL", "low": 0.4, "high": 2.0, "crit_low": None, "crit_high": None, "category": "vitamin"},
    ("thiamine", "b1"): {"unit": "nmol/L", "low": 70, "high": 180, "crit_low": None, "crit_high": None, "category": "vitamin"},
    ("riboflavin", "b2"): {"unit": "nmol/L", "low": 5, "high": 50, "crit_low": None, "crit_high": None, "category": "vitamin"},
    ("pyridoxine", "b6"): {"unit": "nmol/L", "low": 20, "high": 120, "crit_low": None, "crit_high": None, "category": "vitamin"},

    # Tumor Markers
    ("carcinoembryonic", "cea"): {"unit": "ng/mL", "low": 0, "high": 3.0, "crit_low": None, "crit_high": None, "category": "tumor_marker"},
    ("prostate specific", "psa"): {"unit": "ng/mL", "low": 0, "high": 4.0, "crit_low": None, "crit_high": None, "category": "tumor_marker"},
    ("alpha-fetoprotein", "afp"): {"unit": "ng/mL", "low": 0, "high": 10, "crit_low": None, "crit_high": None, "category": "tumor_marker"},
    ("ca 125", "cancer antigen 125"): {"unit": "U/mL", "low": 0, "high": 35, "crit_low": None, "crit_high": None, "category": "tumor_marker"},
    ("ca 19-9", "cancer antigen 19-9"): {"unit": "U/mL", "low": 0, "high": 37, "crit_low": None, "crit_high": None, "category": "tumor_marker"},
    ("ca 15-3", "cancer antigen 15-3"): {"unit": "U/mL", "low": 0, "high": 30, "crit_low": None, "crit_high": None, "category": "tumor_marker"},

    # Urinalysis
    ("specific gravity",): {"unit": "", "low": 1.005, "high": 1.030, "crit_low": 1.001, "crit_high": 1.040, "category": "urinalysis"},
    ("urine ph",): {"unit": "", "low": 4.5, "high": 8.0, "crit_low": None, "crit_high": None, "category": "urinalysis"},

    # Blood Gas
    ("ph",): {"unit": "", "low": 7.35, "high": 7.45, "crit_low": 7.2, "crit_high": 7.6, "category": "blood_gas"},
    ("pco2", "carbon dioxide partial pressure"): {"unit": "mmHg", "low": 35, "high": 45, "crit_low": 20, "crit_high": 70, "category": "blood_gas"},
    ("po2", "oxygen partial pressure"): {"unit": "mmHg", "low": 80, "high": 100, "crit_low": 50, "crit_high": None, "category": "blood_gas"},
    ("oxygen saturation",): {"unit": "%", "low": 95, "high": 100, "crit_low": 88, "crit_high": None, "category": "blood_gas"},
    ("base excess",): {"unit": "mmol/L", "low": -2, "high": 2, "crit_low": -10, "crit_high": 10, "category": "blood_gas"},

    # Therapeutic Drugs
    ("digoxin",): {"unit": "ng/mL", "low": 0.8, "high": 2.0, "crit_low": None, "crit_high": 2.5, "category": "therapeutic_drug"},
    ("lithium",): {"unit": "mmol/L", "low": 0.6, "high": 1.2, "crit_low": None, "crit_high": 1.5, "category": "therapeutic_drug"},
    ("phenytoin",): {"unit": "mcg/mL", "low": 10, "high": 20, "crit_low": None, "crit_high": 25, "category": "therapeutic_drug"},
    ("theophylline",): {"unit": "mcg/mL", "low": 10, "high": 20, "crit_low": None, "crit_high": 25, "category": "therapeutic_drug"},
    ("vancomycin",): {"unit": "mcg/mL", "low": 10, "high": 20, "crit_low": None, "crit_high": 30, "category": "therapeutic_drug"},
    ("tacrolimus",): {"unit": "ng/mL", "low": 5, "high": 15, "crit_low": None, "crit_high": 25, "category": "therapeutic_drug"},
    ("cyclosporine",): {"unit": "ng/mL", "low": 100, "high": 400, "crit_low": None, "crit_high": 600, "category": "therapeutic_drug"},
    ("valproic", "valproate"): {"unit": "mcg/mL", "low": 50, "high": 100, "crit_low": None, "crit_high": 150, "category": "therapeutic_drug"},
    ("carbamazepine",): {"unit": "mcg/mL", "low": 4, "high": 12, "crit_low": None, "crit_high": 15, "category": "therapeutic_drug"},
    ("phenobarbital",): {"unit": "mcg/mL", "low": 15, "high": 40, "crit_low": None, "crit_high": 60, "category": "therapeutic_drug"},
    ("gentamicin",): {"unit": "mcg/mL", "low": 4, "high": 10, "crit_low": None, "crit_high": 12, "category": "therapeutic_drug"},
    ("tobramycin",): {"unit": "mcg/mL", "low": 4, "high": 10, "crit_low": None, "crit_high": 12, "category": "therapeutic_drug"},
    ("amikacin",): {"unit": "mcg/mL", "low": 15, "high": 25, "crit_low": None, "crit_high": 35, "category": "therapeutic_drug"},
    ("methotrexate",): {"unit": "umol/L", "low": None, "high": None, "crit_low": None, "crit_high": 10, "category": "therapeutic_drug"},
    ("sirolimus",): {"unit": "ng/mL", "low": 4, "high": 12, "crit_low": None, "crit_high": 20, "category": "therapeutic_drug"},
}


@dataclass
class LOINCEntry:
    code: str
    name: str
    category: str = "chemistry"
    unit: str = ""
    low_normal: Optional[float] = None
    high_normal: Optional[float] = None
    critical_low: Optional[float] = None
    critical_high: Optional[float] = None
    matched_pattern: str = ""


def match_component_pattern(loinc_name: str) -> Optional[Tuple[Dict, str]]:
    """Match LOINC name to component pattern for reference ranges"""
    name_lower = loinc_name.lower()

    for patterns, ranges in COMPONENT_RANGES.items():
        for pattern in patterns:
            if pattern in name_lower:
                return ranges, pattern
    return None


def extract_ohdsi_loinc(vocab_path: str, max_codes: int = 5000) -> List[LOINCEntry]:
    """Extract Lab Test LOINC codes from OHDSI CONCEPT.csv"""
    entries = []
    seen_codes: Set[str] = set()

    with open(vocab_path, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='\t')
        for row in reader:
            if row.get('vocabulary_id') == 'LOINC' and row.get('concept_class_id') == 'Lab Test':
                code = row['concept_code']
                name = row['concept_name']

                # Skip duplicates
                if code in seen_codes:
                    continue
                seen_codes.add(code)

                # Try to match component pattern
                match = match_component_pattern(name)
                if match:
                    ranges, pattern = match
                    entries.append(LOINCEntry(
                        code=code,
                        name=name,
                        category=ranges["category"],
                        unit=ranges["unit"],
                        low_normal=ranges["low"],
                        high_normal=ranges["high"],
                        critical_low=ranges["crit_low"],
                        critical_high=ranges["crit_high"],
                        matched_pattern=pattern
                    ))
                else:
                    # Add without ranges (for reference only)
                    entries.append(LOINCEntry(
                        code=code,
                        name=name,
                        category="laboratory",
                        matched_pattern=""
                    ))

                if len(entries) >= max_codes:
                    break

    return entries


def generate_expanded_sql(entries: List[LOINCEntry], output_file: str):
    """Generate SQL for expanded LOINC reference ranges"""
    with_ranges = [e for e in entries if e.low_normal is not None or e.high_normal is not None]

    with open(output_file, 'w') as f:
        f.write("-- =============================================================================\n")
        f.write("-- EXPANDED LOINC REFERENCE RANGES FROM OHDSI VOCABULARY\n")
        f.write(f"-- Total entries: {len(with_ranges)} with reference ranges\n")
        f.write(f"-- Total LOINC codes reviewed: {len(entries)}\n")
        f.write("-- Source: OHDSI Vocabulary + Standard Clinical Reference Ranges\n")
        f.write("-- =============================================================================\n\n")
        f.write("BEGIN;\n\n")

        for entry in with_ranges:
            low = entry.low_normal if entry.low_normal is not None else "NULL"
            high = entry.high_normal if entry.high_normal is not None else "NULL"
            crit_low = entry.critical_low if entry.critical_low is not None else "NULL"
            crit_high = entry.critical_high if entry.critical_high is not None else "NULL"

            # Escape quotes
            name = entry.name.replace("'", "''")

            f.write(f"""INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('{entry.code}', '{entry.matched_pattern.title()}', '{name}', '{entry.unit}', {low}, {high}, {crit_low}, {crit_high}, 'adult', 'all', '{entry.category}', 'Auto-generated from OHDSI vocabulary pattern matching.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

""")

        f.write("COMMIT;\n")

    return len(with_ranges)


def main():
    # Find OHDSI vocabulary path
    ohdsi_paths = list(Path("/Users/apoorvabk/Downloads").glob("vocabulary_download_v5_*/CONCEPT.csv"))

    if not ohdsi_paths:
        print("ERROR: OHDSI vocabulary not found!")
        return

    vocab_path = str(ohdsi_paths[0])
    print(f"Using OHDSI vocabulary: {vocab_path}")

    # Extract LOINC codes
    print("\nExtracting LOINC Lab Test codes from OHDSI...")
    entries = extract_ohdsi_loinc(vocab_path, max_codes=10000)
    print(f"Total LOINC codes extracted: {len(entries)}")

    # Count with ranges
    with_ranges = [e for e in entries if e.low_normal is not None or e.high_normal is not None]
    print(f"Codes with matched reference ranges: {len(with_ranges)}")

    # Count by category
    from collections import Counter
    categories = Counter(e.category for e in with_ranges)
    print("\nBy category:")
    for cat, count in sorted(categories.items(), key=lambda x: -x[1]):
        print(f"  {cat}: {count}")

    # Generate SQL
    output_dir = Path(__file__).parent.parent / "migrations"
    output_file = output_dir / "006_expanded_loinc_reference_ranges.sql"

    count = generate_expanded_sql(entries, str(output_file))
    print(f"\nSQL written to: {output_file}")
    print(f"Total INSERT statements: {count}")


if __name__ == "__main__":
    main()
