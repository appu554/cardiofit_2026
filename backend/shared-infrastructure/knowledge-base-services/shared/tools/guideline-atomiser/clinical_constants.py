"""
Clinical Constants — Single source of truth for drug names, threshold patterns,
and clinical signal triggers.

Previously duplicated across:
  - l1_completeness_oracle.py:53-85 (KDIGO_DRUG_NAMES flat list, 65 items)
  - channel_b_drug_dict.py (DRUG_INGREDIENTS with RxNorm codes)

Now both L1 Oracle and CoverageGuard B1 import from this module.

Usage:
    from clinical_constants import (
        KDIGO_DRUG_NAMES,
        CLINICAL_THRESHOLD_RE,
        NUMERIC_THRESHOLD_RE,
        RECOMMENDATION_RE,
        PRESCRIPTIVE_TRIGGERS,
        PROHIBITIVE_TRIGGERS,
        IMPLICIT_RISK_TRIGGERS,
        POPULATION_QUALIFIERS,
        build_drug_automaton,
    )
"""

import re
from typing import Optional

try:
    import ahocorasick
except ImportError:
    ahocorasick = None  # type: ignore[assignment]


# ═══════════════════════════════════════════════════════════════════════════════
# Drug Dictionary — KDIGO Diabetes-in-CKD focus
# Extend via build_drug_automaton(extra_drugs=[...])
# ═══════════════════════════════════════════════════════════════════════════════

KDIGO_DRUG_NAMES: list[str] = [
    # SGLT2 inhibitors
    "empagliflozin", "dapagliflozin", "canagliflozin", "ertugliflozin",
    # GLP-1 receptor agonists
    "liraglutide", "semaglutide", "dulaglutide", "exenatide", "lixisenatide",
    # DPP-4 inhibitors
    "sitagliptin", "saxagliptin", "linagliptin", "alogliptin", "vildagliptin",
    # Biguanides
    "metformin",
    # Sulfonylureas
    "glipizide", "glyburide", "glimepiride", "gliclazide",
    # Insulins
    "insulin", "glargine", "detemir", "degludec", "lispro", "aspart", "glulisine",
    # Meglitinides
    "repaglinide", "nateglinide",
    # Thiazolidinediones
    "pioglitazone", "rosiglitazone",
    # Alpha-glucosidase inhibitors
    "acarbose", "miglitol",
    # MRAs (kidney-relevant)
    "finerenone", "spironolactone", "eplerenone",
    # ACE inhibitors / ARBs
    "lisinopril", "enalapril", "ramipril", "losartan", "valsartan",
    "irbesartan", "telmisartan", "candesartan", "olmesartan",
    # Diuretics
    "furosemide", "hydrochlorothiazide", "chlorthalidone",
    # Statins
    "atorvastatin", "rosuvastatin", "simvastatin", "pravastatin",
    # Anticoagulants (CKD-relevant)
    "warfarin", "apixaban", "rivaroxaban",
    # ESAs
    "erythropoietin", "darbepoetin", "epoetin",
]

# Drug class names (for population-action warnings)
DRUG_CLASS_NAMES: list[str] = [
    "sglt2 inhibitor", "sglt2i",
    "glp-1 receptor agonist", "glp-1 ra",
    "dpp-4 inhibitor",
    "ace inhibitor", "acei",
    "arb", "angiotensin receptor blocker",
    "mra", "mineralocorticoid receptor antagonist",
    "statin",
    "sulfonylurea",
    "thiazolidinedione", "tzd",
    "insulin",
    "diuretic",
    "esa", "erythropoiesis-stimulating agent",
]


# ═══════════════════════════════════════════════════════════════════════════════
# Clinical Threshold Patterns
# ═══════════════════════════════════════════════════════════════════════════════

# Lab values, vitals, biomarkers with comparator + numeric value
CLINICAL_THRESHOLD_RE = re.compile(
    r'(?:eGFR|GFR|CrCl|HbA1c|A1[Cc]|creatinine|albumin|potassium|sodium|'
    r'blood\s*pressure|BP|SBP|DBP|cholesterol|LDL|HDL|triglyceride[s]?|'
    r'glucose|fasting|BMI|body\s*mass|uACR|ACR|proteinuria|albuminuria|'
    r'hemoglobin|phosph|calcium|bicarbonate|urea|BUN|ferritin|transferrin|'
    r'vitamin\s*D|parathyroid|PTH|CKD.?G[1-5]|stage\s+[1-5])'
    r'\s*[<>≤≥=≈±]\s*\d',
    re.IGNORECASE,
)

# Numeric threshold with units (catches bare thresholds like "< 30 mL/min")
NUMERIC_THRESHOLD_RE = re.compile(
    r'[<>≤≥]\s*\d+\.?\d*\s*'
    r'(?:mg|mL|mmol|µmol|g/d[Ll]|g|%|mm\s*Hg|kg|units?|mEq|IU|ng)',
    re.IGNORECASE,
)

# Numeric value extractor for CoverageGuard C1 integrity checking
# Captures: (comparator, value, unit)
# Uses explicit alternation of valid clinical comparators — NOT a character
# class like [<>≤≥]+ which greedily matches garbage across HTML tag boundaries
# (e.g., ">300 mg/g<br><3 mg/mmol" → matches "><3" as comparator+value).
NUMERIC_TUPLE_RE = re.compile(
    r'(≤|≥|<=|>=|<|>)\s*(\d+\.?\d*)\s*'
    r'(mg/d[Ll]|mL/min(?:/1\.73\s*m²)?|mmol/[Ll]|µmol/[Ll]|'
    r'g/d[Ll]|%|mm\s*Hg|kg|mg|mEq/[Ll]|IU|ng/m[Ll]|'
    r'mL|mmol|µmol|g)?',
    re.IGNORECASE,
)

# Bare clinical value with unit (no comparator required).
# Catches "2 g", "4.8 mmol/L", "30 mL/min" — values that appear in
# prescribing text without an explicit comparator prefix.
# Used by B3 branch heuristic to count extracted thresholds where the
# comparator was stripped during extraction (source: "<2 g/day" → span: "2 g").
BARE_CLINICAL_VALUE_RE = re.compile(
    r'\b(\d+\.?\d*)\s*'
    r'(mg/d[Ll]|mL/min(?:/1\.73\s*m²)?|mmol/[Ll]|µmol/[Ll]|'
    r'g/d[Ll]|mm\s*Hg|mEq/[Ll]|ng/m[Ll]|mg/g|'
    r'mmol|µmol|mg|mL|g|%|kg)\b',
    re.IGNORECASE,
)

# Range pattern: "5-10 mg" or "30 to 45 mL/min"
NUMERIC_RANGE_RE = re.compile(
    r'(\d+\.?\d*)\s*[-–—]\s*(\d+\.?\d*)\s*'
    r'(mg/d[Ll]|mL/min(?:/1\.73\s*m²)?|mmol/[Ll]|%|mm\s*Hg|kg|mg)?',
    re.IGNORECASE,
)


# ═══════════════════════════════════════════════════════════════════════════════
# Recommendation / Practice Point markers
# ═══════════════════════════════════════════════════════════════════════════════

RECOMMENDATION_RE = re.compile(
    r'(?:recommendation|practice\s+point|we\s+(?:recommend|suggest)|'
    r'grade\s+[12][A-D]|level\s+[A-D]|evidence\s+quality|'
    r'strength\s+of\s+recommendation|quality\s+of\s+evidence)',
    re.IGNORECASE,
)

# CoverageGuard A1: Specific inventory patterns
RECOMMENDATION_ID_RE = re.compile(
    r'Recommendation\s+(\d+\.\d+\.\d+).*?\(([12][A-D])\)',
    re.IGNORECASE,
)

PRACTICE_POINT_ID_RE = re.compile(
    r'Practice\s+Point\s+(\d+\.\d+(?:\.\d+)?)',
    re.IGNORECASE,
)

RESEARCH_REC_ID_RE = re.compile(
    r'Research\s+[Rr]ecommendation\s+(\d+\.\d+\.\d+)',
    re.IGNORECASE,
)

TABLE_REF_RE = re.compile(r'Table\s+(\d+)[.:]', re.IGNORECASE)
FIGURE_REF_RE = re.compile(r'Figure\s+(\d+)[.:]', re.IGNORECASE)


# ═══════════════════════════════════════════════════════════════════════════════
# CoverageGuard B1: Clinical Signal Triggers (three categories)
# ═══════════════════════════════════════════════════════════════════════════════

PRESCRIPTIVE_TRIGGERS: list[str] = [
    "recommend", "suggest", "should", "consider", "initiate", "prescribe",
    "start", "titrate", "administer", "use", "preferred",
]

PROHIBITIVE_TRIGGERS: list[str] = [
    "contraindicated", "avoid", "do not", "discontinue", "stop",
    "withhold", "not recommended", "should not", "must not",
    "prohibited", "black box",
]

IMPLICIT_RISK_TRIGGERS: list[str] = [
    "insufficient evidence",
    "not studied",
    "uncertain safety",
    "risk outweighs",
    "higher mortality",
    "increased risk",
    "no benefit demonstrated",
    "failed to show",
    "limited data",
    "caution",
    "not established",
    "safety concern",
    "adverse effect",
    "serious adverse",
]

# Connectors for CoverageGuard B3: Branch completeness
BRANCH_CONNECTORS: list[str] = [
    "and", "or", "unless", "except", "but", "however",
    "if", "when", "provided that", "alternatively",
]

EXCEPTION_KEYWORDS: list[str] = [
    "except", "unless", "excluding", "contraindicated in",
    "not applicable to", "does not apply",
]


# ═══════════════════════════════════════════════════════════════════════════════
# CoverageGuard B1: Population Qualifiers (for same-page Tier 2 warning)
# ═══════════════════════════════════════════════════════════════════════════════

POPULATION_QUALIFIERS: list[str] = [
    "egfr", "gfr", "crcl",
    "ckd stage", "ckd g",
    "type 1 diabetes", "type 2 diabetes", "t1d", "t2d",
    "dialysis", "hemodialysis", "peritoneal dialysis",
    "transplant", "kidney transplant",
    "pregnancy", "pregnant", "breastfeeding",
    "pediatric", "children", "elderly", "older adults",
    "age", "years",
    "albuminuria", "proteinuria", "uacr", "acr",
    "heart failure", "hfref", "hfpef",
    "cardiovascular disease", "cvd", "ascvd",
]


# ═══════════════════════════════════════════════════════════════════════════════
# Footnote marker characters (CoverageGuard A1d)
# ═══════════════════════════════════════════════════════════════════════════════

FOOTNOTE_MARKERS: set[str] = {"†", "‡", "*", "§", "¶", "||", "#"}


# ═══════════════════════════════════════════════════════════════════════════════
# CoverageGuard B1: Abbreviation ↔ Expansion Map
# Bidirectional mapping for inverted-index alignment.
# When source says "ARB" but span says "angiotensin receptor blocker"
# (or vice versa), the index must resolve both directions.
# ═══════════════════════════════════════════════════════════════════════════════

CLINICAL_ABBREVIATION_MAP: dict[str, list[str]] = {
    "arb": ["angiotensin", "receptor", "blocker"],
    "mra": ["mineralocorticoid", "receptor", "antagonist"],
    "acei": ["ace", "inhibitor"],
    "sglt2i": ["sglt2", "inhibitor"],
    "esa": ["erythropoiesis-stimulating", "agent"],
    "glp-1": ["glucagon-like", "peptide"],
    "tzd": ["thiazolidinedione"],
    "dpp-4": ["dipeptidyl", "peptidase"],
}

# Pre-built reverse map: expansion word → abbreviation
# e.g., "angiotensin" → "arb", "mineralocorticoid" → "mra"
CLINICAL_EXPANSION_TO_ABBREV: dict[str, str] = {}
for _abbrev, _expansion_words in CLINICAL_ABBREVIATION_MAP.items():
    for _word in _expansion_words:
        CLINICAL_EXPANSION_TO_ABBREV[_word] = _abbrev


# ═══════════════════════════════════════════════════════════════════════════════
# Aho-Corasick Automaton Builder
# ═══════════════════════════════════════════════════════════════════════════════

def build_drug_automaton(
    extra_drugs: Optional[list[str]] = None,
    include_classes: bool = True,
) -> "ahocorasick.Automaton":
    """
    Build an Aho-Corasick automaton for O(n) multi-pattern drug name matching.

    Args:
        extra_drugs: Additional drug names to include
        include_classes: Also match drug class names (e.g., "SGLT2 inhibitor")

    Returns:
        Compiled ahocorasick.Automaton ready for .iter(text)
    """
    if ahocorasick is None:
        raise ImportError(
            "pyahocorasick is required for drug name matching. "
            "Install with: pip install pyahocorasick"
        )

    A = ahocorasick.Automaton()

    all_names = list(KDIGO_DRUG_NAMES)
    if include_classes:
        all_names.extend(DRUG_CLASS_NAMES)
    if extra_drugs:
        all_names.extend(extra_drugs)

    for idx, drug in enumerate(all_names):
        A.add_word(drug.lower(), (idx, drug.lower()))

    A.make_automaton()
    return A
