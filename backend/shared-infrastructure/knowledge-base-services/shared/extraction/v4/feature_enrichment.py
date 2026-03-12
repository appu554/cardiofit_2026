"""
Feature Enrichment: Compute ML features from MergedSpan + context.

Produces a flat feature dict suitable for scikit-learn / XGBoost ingestion.
Used in two modes:
1. Offline batch enrichment (scripts/enrich_golden_dataset.py) — populate
   the golden dataset Parquet with features for model training.
2. Online inference (classifiers/) — compute features at pipeline runtime
   for the noise gate, tier assigner, and safety criticality detector.

Pipeline Position:
    Signal Merger -> Feature Enrichment (THIS) -> Classifiers -> Tiering
"""

from __future__ import annotations

import re
from typing import Optional

from .noise_archetypes import classify_noise_archetype
from .channel_b_drug_dict import ChannelBDrugDict


# ═══════════════════════════════════════════════════════════════════════
# Pre-compiled patterns for feature extraction
# ═══════════════════════════════════════════════════════════════════════

_UNIT_RE = re.compile(
    r'mg|mL|mmol|kg|%|mmHg|mcg|µg|IU|units?|mEq|g/d[Ll]|mg/d[Ll]',
    re.IGNORECASE,
)
_COMPARISON_RE = re.compile(r'[<>]=?|≥|≤|greater\s+than|less\s+than', re.IGNORECASE)
_NUMBER_RE = re.compile(r'\d')
_SAFETY_KEYWORDS_RE = re.compile(
    r'contraindicated|avoid|black\s*box|maximum\s+dose|do\s+not\s+use|'
    r'not\s+recommended|discontinue|withhold|prohibited',
    re.IGNORECASE,
)

# Clinical verbs (same set as Channel B context gate)
_CLINICAL_VERBS = frozenset({
    "recommend", "prescribe", "administer", "contraindicated", "avoid",
    "discontinue", "initiate", "titrate", "monitor", "reduce",
    "increase", "decrease", "adjust", "consider", "suggest",
    "preferred", "alternative", "indicated", "effective",
})

# Build lowered drug name set from Channel B's class constants
_ALL_DRUG_NAMES_LOWER = frozenset(
    k.lower() for k in ChannelBDrugDict.DRUG_INGREDIENTS
)
_ALL_CLASS_NAMES_LOWER = frozenset(
    k.lower() for k in ChannelBDrugDict.DRUG_CLASSES
)
_BARE_CLASS_ABBREVS = frozenset(
    a.lower() for a in {
        "ARB", "MRA", "ACEi", "ACEI", "ACE", "CCB", "BB",
        "NSAID", "NSAIDs", "SGLT2", "DPP4", "GLP1",
    }
)


# ═══════════════════════════════════════════════════════════════════════
# Main enrichment function
# ═══════════════════════════════════════════════════════════════════════

def enrich_span_features(
    text: str,
    start: int,
    end: int,
    contributing_channels: list[str],
    channel_confidences: dict[str, float],
    merged_confidence: float,
    has_disagreement: bool,
    full_text: str,
    section_id: Optional[str] = None,
    page_number: Optional[int] = None,
) -> dict:
    """Compute a flat feature dict for a single span.

    Designed for both MergedSpan objects and raw DB rows — accepts
    primitive arguments rather than a model instance to avoid import
    coupling with Pydantic models.

    Args:
        text: The span text.
        start: Start character offset in full_text.
        end: End character offset in full_text.
        contributing_channels: List of channel IDs (e.g. ["B", "C"]).
        channel_confidences: Per-channel confidence scores.
        merged_confidence: Signal Merger confidence (0.0-1.0).
        has_disagreement: Whether contributing channels disagree on text.
        full_text: Full normalized guideline text (for context window).
        section_id: Optional section ID from the guideline tree.
        page_number: Optional page number in source PDF.

    Returns:
        dict of feature_name -> feature_value (all numeric or boolean,
        ready for DataFrame / XGBoost ingestion).
    """
    stripped = text.strip()
    stripped_lower = stripped.lower()
    n_channels = len(contributing_channels)

    # ── Text features ────────────────────────────────────────────────
    text_length = len(stripped)
    word_count = len(stripped.split())
    has_number = bool(_NUMBER_RE.search(stripped))
    has_unit = bool(_UNIT_RE.search(stripped))
    has_comparison = bool(_COMPARISON_RE.search(stripped))
    has_safety_keyword = bool(_SAFETY_KEYWORDS_RE.search(stripped))

    # ── Channel features ─────────────────────────────────────────────
    channel_set = set(contributing_channels)
    has_channel_B = "B" in channel_set
    has_channel_C = "C" in channel_set
    has_channel_D = "D" in channel_set
    has_channel_E = "E" in channel_set
    has_channel_F = "F" in channel_set
    has_channel_G = "G" in channel_set
    has_channel_H = "H" in channel_set

    # ── Context features (200-char window) ───────────────────────────
    context_window = 200
    ctx_start = max(0, start - context_window)
    ctx_end = min(len(full_text), end + context_window)
    context_lower = full_text[ctx_start:ctx_end].lower()

    clinical_verb_count = sum(
        1 for verb in _CLINICAL_VERBS if verb in context_lower
    )
    # Density: clinical verbs per 100 chars of context
    context_len = max(len(context_lower), 1)
    clinical_term_density = clinical_verb_count / context_len * 100

    # ── Section features ─────────────────────────────────────────────
    section_depth = _section_depth(section_id)

    # ── Drug/clinical features ───────────────────────────────────────
    has_drug_name = any(drug in stripped_lower for drug in _ALL_DRUG_NAMES_LOWER)
    has_drug_class = any(cls in stripped_lower for cls in _ALL_CLASS_NAMES_LOWER)
    is_bare_abbreviation = stripped_lower in _BARE_CLASS_ABBREVS

    # ── Noise archetype ──────────────────────────────────────────────
    noise_archetype = classify_noise_archetype(stripped)
    # Encode as categorical int for ML (None → 0, each archetype → 1-8)
    _ARCHETYPE_MAP = {
        None: 0,
        "standalone_drug_name": 1,
        "rec_pp_label": 2,
        "standalone_lab_name": 3,
        "action_verb_fragment": 4,
        "dose_fragment": 5,
        "abbreviation_legend": 6,
        "html_artifact": 7,
        "decontextualized_threshold": 8,
    }
    noise_archetype_code = _ARCHETYPE_MAP.get(noise_archetype, 0)

    return {
        # Text features
        "text_length": text_length,
        "word_count": word_count,
        "has_number": has_number,
        "has_unit": has_unit,
        "has_comparison": has_comparison,
        "has_safety_keyword": has_safety_keyword,

        # Channel features
        "n_channels": n_channels,
        "has_channel_B": has_channel_B,
        "has_channel_C": has_channel_C,
        "has_channel_D": has_channel_D,
        "has_channel_E": has_channel_E,
        "has_channel_F": has_channel_F,
        "has_channel_G": has_channel_G,
        "has_channel_H": has_channel_H,
        "has_disagreement": has_disagreement,
        "merged_confidence": merged_confidence,

        # Context features
        "clinical_verb_count": clinical_verb_count,
        "clinical_term_density": clinical_term_density,

        # Section features
        "section_depth": section_depth,

        # Drug/clinical features
        "has_drug_name": has_drug_name,
        "has_drug_class": has_drug_class,
        "is_bare_abbreviation": is_bare_abbreviation,

        # Noise archetype (encoded)
        "noise_archetype_code": noise_archetype_code,
        "noise_archetype": noise_archetype,  # string label for analysis

        # Page features
        "page_number": page_number or 0,
    }


def _section_depth(section_id: Optional[str]) -> int:
    """Compute section nesting depth from dotted section_id.

    "4.1.1" → 3, "4" → 1, None → 0
    """
    if not section_id:
        return 0
    return section_id.count(".") + 1
