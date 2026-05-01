"""
Channel B: Drug Dictionary using Aho-Corasick.

O(n) multi-pattern string matching with word boundary enforcement.
Builds a dictionary from:
1. KNOWN_DRUG_INGREDIENTS (from gliner/extractor.py)
2. DRUG_CLASSES (from gliner/extractor.py)
3. KNOWN_CLASS_ABBREVIATIONS (from gliner/extractor.py)
4. CQL valuesets (RenalCommon.cql, T2DMGuidelines.cql)

Word boundary enforcement prevents "ARB" matching inside "garbanzo".

Pipeline Position:
    Channel A (GuidelineTree) -> Channel B (THIS, parallel with C-F)
"""

from __future__ import annotations

import json
import time
from pathlib import Path
from typing import Optional

import re

from .models import ChannelOutput, GuidelineTree, RawSpan
from .postprocessors import extend_parenthetical
from .provenance import (
    ChannelProvenance,
    _normalise_bbox,
    _normalise_confidence,
    _normalise_page_number,
)
from .v5_flags import is_v5_enabled


def _channel_b_model_version() -> str:
    """Channel B model version, pinned to ChannelBDrugDict.VERSION if present."""
    try:
        return f"aho-corasick@{ChannelBDrugDict.VERSION}"
    except Exception:
        return "aho-corasick@v1.0"


def _channel_b_provenance(
    bbox,
    page_number,
    confidence,
    profile,
    notes: Optional[str] = None,
) -> Optional[ChannelProvenance]:
    """Build a ChannelProvenance entry for Channel B (drug-dictionary Aho-Corasick).

    Returns None when V5_BBOX_PROVENANCE is off or bbox is missing. Bbox is
    normally inherited from the parent block in MonkeyOCR's L1 output.
    """
    if not is_v5_enabled("bbox_provenance", profile):
        return None
    bb = _normalise_bbox(bbox)
    if bb is None:
        return None
    return ChannelProvenance(
        channel_id="B",
        bbox=bb,
        page_number=_normalise_page_number(page_number),
        confidence=_normalise_confidence(confidence),
        model_version=_channel_b_model_version(),
        notes=notes,
    )


class ChannelBDrugDict:
    """Aho-Corasick drug dictionary matcher with word boundary enforcement.

    Also exported as ``ChannelB`` for pipeline convenience.

    Uses the ahocorasick library for O(n+m) multi-pattern matching where
    n = text length and m = total pattern length. After each match, a
    word boundary check prevents partial-word matches.
    """

    VERSION = "4.1.0"
    CONFIDENCE = 1.0  # Dictionary matches are deterministic

    # Default context window size for clinical context gate (chars each direction)
    CONTEXT_WINDOW_SIZE = 200

    # Clinical verbs that indicate a drug mention is in a prescriptive/clinical context
    _CLINICAL_VERBS = frozenset({
        "recommend", "prescribe", "administer", "contraindicated", "avoid",
        "discontinue", "initiate", "titrate", "monitor", "reduce",
        "increase", "decrease", "adjust", "consider", "suggest",
        "preferred", "alternative", "indicated", "effective",
    })

    # =========================================================================
    # Drug dictionaries harvested from existing codebase
    # =========================================================================

    # From gliner/extractor.py KNOWN_DRUG_INGREDIENTS
    DRUG_INGREDIENTS: dict[str, Optional[str]] = {
        # SGLT2 inhibitors
        "metformin": "860975",
        "dapagliflozin": "1488564",
        "empagliflozin": "1545653",
        "canagliflozin": "1373458",
        "ertugliflozin": "1992672",
        "sotagliflozin": "2169285",
        # MRAs
        "finerenone": "2555902",
        "spironolactone": "9997",
        "eplerenone": "298869",
        # ACE inhibitors
        "lisinopril": "29046",
        "enalapril": "3827",
        "ramipril": "35296",
        "perindopril": "54552",
        # ARBs
        "losartan": "52175",
        "valsartan": "69749",
        "irbesartan": "83818",
        "candesartan": "214354",
        "telmisartan": "73494",
        "olmesartan": "321064",
        # GLP-1 RAs
        "semaglutide": "1991302",
        "liraglutide": "475968",
        "dulaglutide": "1551291",
        "exenatide": "60548",
        # DPP-4 inhibitors
        "sitagliptin": "593411",
        "linagliptin": "1100699",
        "saxagliptin": "857974",
        "alogliptin": "1368001",
        # TZDs
        "pioglitazone": "33738",
        "rosiglitazone": "84108",
        # Sulfonylureas
        "glyburide": "4815",
        "glipizide": "4821",
        "glimepiride": "25789",
        # Insulin
        "insulin": "5856",
        # Diuretics
        "furosemide": "4603",
        "bumetanide": "1808",
        "hydrochlorothiazide": "5487",
        "chlorthalidone": "2409",
        "indapamide": "5764",
        # Calcium channel blockers
        "amlodipine": "17767",
        "nifedipine": "7417",
        "diltiazem": "3443",
        "verapamil": "11170",
        # Beta blockers
        "atenolol": "1202",
        "metoprolol": "6918",
        "carvedilol": "20352",
        "bisoprolol": "19484",
        # Statins
        "atorvastatin": "83367",
        "rosuvastatin": "301542",
    }

    # From gliner/extractor.py DRUG_CLASSES (canonicalized)
    DRUG_CLASSES: dict[str, str] = {
        "sglt2 inhibitor": "SGLT2i",
        "sglt2 inhibitors": "SGLT2i",
        "sglt2i": "SGLT2i",
        "sglt-2 inhibitor": "SGLT2i",
        "sglt-2 inhibitors": "SGLT2i",
        "sglt-2i": "SGLT2i",
        "ace inhibitor": "ACEi",
        "ace inhibitors": "ACEi",
        "acei": "ACEi",
        "ace-i": "ACEi",
        "angiotensin receptor blocker": "ARB",
        "angiotensin receptor blockers": "ARB",
        "arb": "ARB",
        "arbs": "ARB",
        "glp-1 agonist": "GLP-1 RA",
        "glp-1 agonists": "GLP-1 RA",
        "glp-1 receptor agonist": "GLP-1 RA",
        "glp-1 receptor agonists": "GLP-1 RA",
        "glp-1 ra": "GLP-1 RA",
        "glp1-ra": "GLP-1 RA",
        "mineralocorticoid receptor antagonist": "MRA",
        "mineralocorticoid receptor antagonists": "MRA",
        "mra": "MRA",
        "mras": "MRA",
        "ns-mra": "nsMRA",
        "dpp-4 inhibitor": "DPP-4i",
        "dpp-4 inhibitors": "DPP-4i",
        "dpp4i": "DPP-4i",
        "dpp-4i": "DPP-4i",
        "sulfonylurea": "Sulfonylurea",
        "sulfonylureas": "Sulfonylurea",
        "biguanide": "Biguanide",
        "biguanides": "Biguanide",
        "thiazolidinedione": "TZD",
        "thiazolidinediones": "TZD",
        "tzd": "TZD",
        "tzds": "TZD",
        "rasi": "RASi",
        "ras inhibitor": "RASi",
        "ras inhibitors": "RASi",
        "ras-i": "RASi",
        "beta blocker": "Beta-blocker",
        "beta blockers": "Beta-blocker",
        "beta-blocker": "Beta-blocker",
        "beta-blockers": "Beta-blocker",
        "calcium channel blocker": "CCB",
        "calcium channel blockers": "CCB",
        "diuretic": "Diuretic",
        "diuretics": "Diuretic",
        "loop diuretic": "Loop diuretic",
        "loop diuretics": "Loop diuretic",
        "thiazide": "Thiazide",
        "thiazides": "Thiazide",
        "nsaid": "NSAID",
        "nsaids": "NSAID",
        "statin": "Statin",
        "statins": "Statin",
    }

    def __init__(
        self,
        extra_ingredients: Optional[dict[str, Optional[str]]] = None,
        extra_classes: Optional[dict[str, str]] = None,
        context_window_size: Optional[int] = None,
    ) -> None:
        """Initialize the drug dictionary and build the Aho-Corasick automaton.

        Args:
            extra_ingredients: Additional drug ingredients to merge into the
                automaton beyond the base DRUG_INGREDIENTS dict.  Format:
                ``{"drug_name": "rxnorm_code"}`` or ``{"drug_name": None}``.
                Provided by GuidelineProfile.extra_drug_ingredients.
            extra_classes: Additional drug class variants to merge.  Format:
                ``{"variant_text": "CanonicalClassName"}``.
                Provided by GuidelineProfile.extra_drug_classes.
            context_window_size: Number of characters to scan in each direction
                for clinical context validation.  Defaults to CONTEXT_WINDOW_SIZE (200).
                Provided by GuidelineProfile.context_window_size.
        """
        self._extra_ingredients = extra_ingredients or {}
        self._extra_classes = extra_classes or {}
        self._context_window_size = context_window_size or self.CONTEXT_WINDOW_SIZE
        self._automaton = None
        self._build_automaton()
        # Build a lowered set of all known drug names for context scanning
        self._all_drug_names_lower = frozenset(
            k.lower() for k in self.DRUG_INGREDIENTS
        ) | frozenset(
            k.lower() for k in self._extra_ingredients
        )

    def _build_automaton(self) -> None:
        """Build Aho-Corasick automaton from all drug names and classes."""
        try:
            import ahocorasick
        except ImportError:
            raise ImportError(
                "ahocorasick-python is required for Channel B. "
                "Install with: pip install ahocorasick-python"
            )

        self._automaton = ahocorasick.Automaton()

        # Add base drug ingredients
        for drug_name, rxnorm in self.DRUG_INGREDIENTS.items():
            key = drug_name.lower()
            self._automaton.add_word(key, (key, "exact", rxnorm))

        # Add profile-specific extra ingredients (guideline supplements)
        for drug_name, rxnorm in self._extra_ingredients.items():
            key = drug_name.lower()
            if key not in self.DRUG_INGREDIENTS:  # don't overwrite base
                self._automaton.add_word(key, (key, "exact", rxnorm))

        # Add drug classes (lower-cased for case-insensitive matching)
        for class_variant, canonical in self.DRUG_CLASSES.items():
            key = class_variant.lower()
            if key not in self.DRUG_INGREDIENTS:  # avoid overwriting ingredients
                self._automaton.add_word(key, (key, "class", canonical))

        # Add profile-specific extra drug classes
        for class_variant, canonical in self._extra_classes.items():
            key = class_variant.lower()
            if key not in self.DRUG_INGREDIENTS and key not in self.DRUG_CLASSES:
                self._automaton.add_word(key, (key, "class", canonical))

        self._automaton.make_automaton()

    def extract(
        self,
        text: str,
        tree: GuidelineTree,
    ) -> ChannelOutput:
        """Run Aho-Corasick matching on normalized text.

        Args:
            text: Normalized text (Channel 0 output)
            tree: GuidelineTree from Channel A

        Returns:
            ChannelOutput with drug RawSpans
        """
        start_time = time.monotonic()
        spans: list[RawSpan] = []
        text_lower = text.lower()
        context_gate_rejections = 0

        if self._automaton is None:
            return ChannelOutput(
                channel="B",
                spans=[],
                error="Aho-Corasick automaton not initialized",
            )

        seen_positions: set[tuple[int, int]] = set()

        for end_idx, (pattern, match_type, meta_value) in self._automaton.iter(text_lower):
            start_idx = end_idx - len(pattern) + 1
            end_idx_exclusive = end_idx + 1

            # Word boundary check
            if not self._is_word_boundary(text, start_idx, end_idx_exclusive):
                continue

            # Context gate: reject matches in non-clinical zones
            if not self._is_clinical_context(text, start_idx, end_idx_exclusive, tree):
                context_gate_rejections += 1
                continue

            # Deduplicate overlapping matches at same position
            pos_key = (start_idx, end_idx_exclusive)
            if pos_key in seen_positions:
                continue
            seen_positions.add(pos_key)

            # Parenthetical extension: capture trailing qualifiers
            start_idx, end_idx_exclusive = extend_parenthetical(
                text, start_idx, end_idx_exclusive
            )

            # Use original-case text from source
            matched_text = text[start_idx:end_idx_exclusive]

            # Find section and page
            # V4.2.2: page from direct offset lookup, not section heading page
            section = tree.find_section_for_offset(start_idx)
            section_id = section.section_id if section else None
            page = tree.get_page_for_offset(start_idx)

            # Build channel metadata
            channel_metadata: dict = {"match_type": match_type}
            if match_type == "exact":
                channel_metadata["rxnorm_candidate"] = meta_value
            elif match_type == "class":
                channel_metadata["canonical_class"] = meta_value

            spans.append(RawSpan(
                channel="B",
                text=matched_text,
                start=start_idx,
                end=end_idx_exclusive,
                confidence=self.CONFIDENCE,
                page_number=page,
                section_id=section_id,
                source_block_type="paragraph",
                channel_metadata=channel_metadata,
            ))

        elapsed_ms = (time.monotonic() - start_time) * 1000

        return ChannelOutput(
            channel="B",
            spans=spans,
            metadata={
                "dictionary_size": len(self.DRUG_INGREDIENTS) + len(self.DRUG_CLASSES),
                "matches_found": len(spans),
                "context_gate_rejections": context_gate_rejections,
            },
            elapsed_ms=elapsed_ms,
        )

    def _is_word_boundary(self, text: str, start: int, end: int) -> bool:
        """Reject matches that aren't at word boundaries.

        Prevents "ARB" matching inside "garbanzo" or "carb".
        """
        if start > 0 and text[start - 1].isalnum():
            return False
        if end < len(text) and text[end].isalnum():
            return False
        return True

    # =========================================================================
    # V4.1: CONTEXT GATE — reject matches in non-clinical zones
    #
    # The Aho-Corasick automaton matches drug names everywhere in the text.
    # Many of these matches are in reference sections, author lists, or
    # citation brackets where the drug name is not a clinical directive.
    # The context gate uses lightweight heuristics to reject these.
    # =========================================================================

    # Reference section headings that indicate non-clinical text
    _REFERENCE_HEADING_RE = re.compile(
        r'^\s*(?:references?|bibliography|works?\s+cited|'
        r'supplementary|acknowledgments?|disclosures?|'
        r'conflicts?\s+of\s+interest|funding|author\s+contributions?)\s*$',
        re.IGNORECASE | re.MULTILINE,
    )

    # Citation brackets: [Smith et al., 2020] or [1-3] or [14,15]
    _CITATION_BRACKET_RE = re.compile(
        r'\[(?:[A-Z][a-z]+\s+et\s+al\.|'  # [Smith et al.]
        r'\d{1,3}(?:\s*[-–,]\s*\d{1,3})*'  # [1-3] or [14,15]
        r')\]',
    )

    def _is_clinical_context(
        self,
        text: str,
        start: int,
        end: int,
        tree: "GuidelineTree",
    ) -> bool:
        """Check whether a match is in a clinical context worth extracting.

        Uses a configurable context window (default 200 chars) for all
        surrounding-text checks.  Rejects matches in:
        1. Reference/bibliography sections (after reference heading)
        2. Inside citation brackets [Smith et al., 2020]
        3. Author list zones (contiguous proper noun sequences)
        4. Clinically barren zones (no clinical verbs or co-occurring drugs
           within the context window)

        Returns True if the match should be KEPT (clinical context).
        Returns False if the match should be REJECTED (non-clinical).
        """
        win = self._context_window_size

        # --- Check 1: Reference section ---
        # Use the tree to determine if this offset is in a reference section
        section = tree.find_section_for_offset(start)
        if section and section.heading:
            heading_lower = section.heading.lower().strip()
            if any(kw in heading_lower for kw in (
                "reference", "bibliography", "works cited",
                "supplementary", "disclosure", "acknowledgment",
                "conflicts of interest", "funding",
                "author contribution",
            )):
                return False

        # --- Check 2: Inside citation brackets ---
        # Look backward for an opening bracket within context window
        search_start = max(0, start - win)
        search_end = min(len(text), end + 20)
        before_text = text[search_start:search_end]
        for m in self._CITATION_BRACKET_RE.finditer(before_text):
            bracket_start = search_start + m.start()
            bracket_end = search_start + m.end()
            if bracket_start <= start and bracket_end >= end:
                return False

        # --- Check 3: Author list heuristic ---
        # If surrounded by capitalized words (Proper, Noun, et al.), likely author list
        ctx_start = max(0, start - win)
        ctx_end = min(len(text), end + win)
        context = text[ctx_start:ctx_end]

        # Count capitalized words in context (excluding the match itself)
        before_match = text[ctx_start:start]
        after_match = text[end:ctx_end]
        cap_words_before = len(re.findall(r'\b[A-Z][a-z]+\b', before_match))
        cap_words_after = len(re.findall(r'\b[A-Z][a-z]+\b', after_match))
        total_words_before = len(before_match.split())
        total_words_after = len(after_match.split())

        # If >70% of surrounding words are capitalized proper nouns,
        # this is likely an author list (permissive threshold)
        total_words = total_words_before + total_words_after
        cap_words = cap_words_before + cap_words_after
        if total_words >= 4 and cap_words / total_words > 0.70:
            # Additional check: "et al." nearby confirms author list
            if "et al" in context.lower():
                return False

        # --- Check 4: Clinical term density ---
        # Reject drug matches in text that has zero clinical signal within
        # the context window (no clinical verbs and no co-occurring drugs).
        # This filters out drug names appearing in legends, abbreviation
        # lists, table-of-contents entries, and other non-prescriptive text.
        context_lower = context.lower()
        matched_text_lower = text[start:end].lower().strip()

        has_clinical_verb = any(
            verb in context_lower for verb in self._CLINICAL_VERBS
        )
        # Check for co-occurring drug names (different from the matched one)
        has_cooccurring_drug = any(
            drug in context_lower and drug != matched_text_lower
            for drug in self._all_drug_names_lower
        )

        if not has_clinical_verb and not has_cooccurring_drug:
            # Safety check: keep the match if it's in a clearly clinical section
            # (section heading contains clinical keywords)
            if section and section.heading:
                heading_lc = section.heading.lower()
                if any(kw in heading_lc for kw in (
                    "treatment", "therapy", "management", "dosing",
                    "pharmacol", "medication", "drug", "recommendation",
                )):
                    return True
            return False

        return True


# Short alias for pipeline imports
ChannelB = ChannelBDrugDict
