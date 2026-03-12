"""
Noise Archetype Taxonomy: 8-class noise classification for spans.

Classifies extracted spans into specific noise archetypes to enable
targeted filtering and feature engineering for the ML tiering pipeline.

The 8 archetypes were derived from manual analysis of the golden dataset
(6,326 reviewer-labelled spans), where single-channel spans had 75.4%
noise rate.  Each archetype captures a distinct mechanism by which
non-clinical text enters the extraction pipeline.

Pipeline Position:
    Signal Merger -> Noise Archetype Classification -> Tiering Classifier
    (also used as a feature in enrich_span_features for ML classifiers)
"""

from __future__ import annotations

import re
from enum import Enum
from typing import Optional


class NoiseArchetype(str, Enum):
    """The 8 noise archetypes observed in guideline extraction pipelines.

    Each archetype represents a distinct failure mode where non-clinical
    text is captured as a span.
    """

    STANDALONE_DRUG_NAME = "standalone_drug_name"
    """Bare drug/class name without any prescriptive context.
    Examples: "metformin", "SGLT2i" appearing in a legend or abbreviation table.
    Golden dataset: ARB 0/70, MRA 0/66, ACEi 1/79 confirmed as single-channel."""

    REC_PP_LABEL = "rec_pp_label"
    """Recommendation/practice-point label without content.
    Examples: "Recommendation 1.1.1", "Practice Point 3.2", "Grade A".
    These are structural labels, not clinical directives."""

    STANDALONE_LAB_NAME = "standalone_lab_name"
    """Bare lab/biomarker name without threshold or clinical context.
    Examples: "eGFR", "HbA1c", "UACR" appearing alone in headers or legends."""

    ACTION_VERB_FRAGMENT = "action_verb_fragment"
    """Isolated action verb or verb phrase without a clinical object.
    Examples: "Consider", "We suggest", "Monitor" without what to monitor."""

    DOSE_FRAGMENT = "dose_fragment"
    """Numeric dose value without drug context.
    Examples: "10 mg", "2.5–5 mg", "100 units" appearing in table cells."""

    ABBREVIATION_LEGEND = "abbreviation_legend"
    """Entry from an abbreviation/acronym legend table.
    Examples: "SGLT2i = sodium-glucose cotransporter 2 inhibitor",
    "ACEi, angiotensin-converting enzyme inhibitor"."""

    HTML_ARTIFACT = "html_artifact"
    """Residual HTML/XML markup or entity that survived normalization.
    Examples: "&gt;", "&le;", "<br>", "&#8805;", "class=\"table-cell\""."""

    DECONTEXTUALIZED_THRESHOLD = "decontextualized_threshold"
    """Numeric threshold without the parameter it refers to.
    Examples: "> 30", "≥ 7%", "< 60" appearing without "eGFR", "HbA1c", etc."""


# ═══════════════════════════════════════════════════════════════════════
# Detection patterns for each archetype
# ═══════════════════════════════════════════════════════════════════════

# Bare lab/biomarker names (standalone, no threshold)
_LAB_NAMES = frozenset({
    "egfr", "hba1c", "uacr", "acr", "gfr", "creatinine", "bun",
    "potassium", "sodium", "albumin", "hemoglobin", "glucose",
    "ldl", "hdl", "triglycerides", "cholesterol", "bmi",
    "blood pressure", "systolic", "diastolic", "sbp", "dbp",
})

# Recommendation/practice-point label patterns
_REC_LABEL_RE = re.compile(
    r'^(?:'
    r'(?:recommendation|practice\s+point|rec|pp|table|figure|box|chapter)'
    r'\s*\d[\d.]*'
    r'|grade\s+[A-D1-4]'
    r'|level\s+of\s+evidence\s*[A-D1-4]?'
    r'|evidence\s+grade\s*[A-D1-4]?'
    r')\s*\.?\s*$',
    re.IGNORECASE,
)

# Action verb fragments (verb without clinical object)
_ACTION_VERB_RE = re.compile(
    r'^(?:consider|suggest|recommend|monitor|assess|evaluate|review|'
    r'we\s+(?:suggest|recommend)|'
    r'it\s+is\s+(?:suggested|recommended))\s*\.?\s*$',
    re.IGNORECASE,
)

# Dose fragment: numeric dose without drug name context
_DOSE_FRAGMENT_RE = re.compile(
    r'^\s*\d+(?:\.\d+)?(?:\s*[-–]\s*\d+(?:\.\d+)?)?\s*'
    r'(?:mg|mcg|µg|g|ml|mL|units?|IU|mmol|mEq)\s*'
    r'(?:/(?:day|d|kg|L|dL|min))?\s*$',
    re.IGNORECASE,
)

# Abbreviation legend: "ABBR = full name" or "ABBR, full name" pattern
_ABBREV_LEGEND_RE = re.compile(
    r'^[A-Z][A-Za-z0-9-]{1,10}\s*[=,]\s*.{10,}$',
)

# HTML artifact patterns
_HTML_ARTIFACT_RE = re.compile(
    r'&(?:gt|lt|ge|le|amp|nbsp|#\d{2,5}|#x[0-9a-fA-F]{2,4});'
    r'|</?(?:br|div|span|td|tr|th|p|table|img|a)\b'
    r'|class\s*=\s*["\']',
    re.IGNORECASE,
)

# Decontextualized threshold: comparison operator + number, no parameter name
_THRESHOLD_RE = re.compile(
    r'^\s*(?:[<>]=?|≥|≤|&[gl][te];)\s*\d+(?:\.\d+)?\s*'
    r'(?:%|mg|mL|mmol|mmHg|kg|L|dL|min)?\s*$',
)

# Drug-related abbreviations that are commonly noise when standalone
_BARE_DRUG_ABBREVS = frozenset({
    "arb", "mra", "acei", "ace", "ccb", "bb",
    "nsaid", "nsaids", "sglt2", "dpp4", "glp1",
    "sglt2i", "dpp-4i", "glp-1 ra", "rasi",
})


def classify_noise_archetype(text: str) -> Optional[str]:
    """Classify a span's text into a noise archetype, if any.

    Args:
        text: The span text to classify.

    Returns:
        The NoiseArchetype value string if the text matches an archetype,
        or None if the text does not match any noise pattern.
    """
    stripped = text.strip()
    stripped_lower = stripped.lower()

    # Very short text is not classifiable into a specific archetype
    if len(stripped) < 2:
        return None

    # 1. HTML artifacts (check first — these are always noise)
    if _HTML_ARTIFACT_RE.search(stripped):
        return NoiseArchetype.HTML_ARTIFACT

    # 2. Recommendation/practice-point labels
    if _REC_LABEL_RE.match(stripped):
        return NoiseArchetype.REC_PP_LABEL

    # 3. Action verb fragments (no clinical object)
    if _ACTION_VERB_RE.match(stripped):
        return NoiseArchetype.ACTION_VERB_FRAGMENT

    # 4. Decontextualized thresholds (operator + number, no parameter)
    if _THRESHOLD_RE.match(stripped):
        return NoiseArchetype.DECONTEXTUALIZED_THRESHOLD

    # 5. Dose fragments (number + unit, no drug name)
    if _DOSE_FRAGMENT_RE.match(stripped):
        return NoiseArchetype.DOSE_FRAGMENT

    # 6. Abbreviation legend entries
    if _ABBREV_LEGEND_RE.match(stripped) and "=" in stripped:
        return NoiseArchetype.ABBREVIATION_LEGEND

    # 7. Standalone lab/biomarker names
    if stripped_lower in _LAB_NAMES:
        return NoiseArchetype.STANDALONE_LAB_NAME

    # 8. Standalone drug name / bare class abbreviation
    # (only exact matches — partial drug names within longer text are
    # handled by the tiering classifier's context checks)
    if stripped_lower in _BARE_DRUG_ABBREVS:
        return NoiseArchetype.STANDALONE_DRUG_NAME

    return None
