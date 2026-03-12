"""
Dossier Assembly: Bridge between Pipeline 1 and Pipeline 2.

Groups reviewer-verified spans into per-drug dossiers before passing to L3.
Without this step, L3 would receive a flat bag of verified spans with no drug
association.

Algorithm:
1. SCAN ALL SPANS for drug name mentions (text-based, word-boundary matching)
2. BUILD PER-DRUG INDEX — Each drug → list of spans mentioning it
3. ASSOCIATE ORPHAN SIGNALS — Spans without drug mentions → nearest drug via
   section co-location + proximity tie-breaking
4. BUILD PER-DRUG DOSSIER — Collect associated spans, sections, pages, source text
5. OUTPUT — list[DrugDossier], one per drug found in the verified spans

Drug detection uses text matching against the drug ingredient dictionary
(same source as Channel B). This works regardless of whether Channel B
metadata is present in extraction_context — critical for:
- GCP round-trip (metadata not stored in l2_merged_spans)
- REVIEWER-added spans (contributing_channels=["REVIEWER"])
- In-process Pipeline 1→2 (backward compatible — text matching is a superset)

Pipeline Position:
    Reviewer Approval -> Dossier Assembly (THIS) -> L3 Claude (per-drug)
"""

from __future__ import annotations

import re
from collections import defaultdict
from typing import Optional

from .models import DrugDossier, GuidelineTree, VerifiedSpan


class DossierAssembler:
    """Assemble per-drug dossiers from reviewer-verified spans.

    Drug anchor detection is text-based (word-boundary regex matching against
    the drug ingredient + drug class dictionaries). This is the PRIMARY
    mechanism — it works for all span sources (Channel B, REVIEWER, GCP
    round-trip). Channel B extraction_context metadata, when present, is
    used as optional enrichment for RxNorm candidates.

    Signal association uses:
    1. Drug name detected in span text (direct)
    2. Section co-location with proximity tie-breaking (200 char threshold)
    """

    VERSION = "4.3.0"  # Class-vs-member dedup: drop class when members detected
    PROXIMITY_THRESHOLD = 200  # characters for tie-breaking

    # ── Drug Dictionary (from Channel B) ────────────────────────────────

    # Drug ingredients: lowercase name → RxNorm CUI
    _DRUG_INGREDIENTS: dict[str, str] = {
        # SGLT2 inhibitors
        "metformin": "860975", "dapagliflozin": "1488564",
        "empagliflozin": "1545653", "canagliflozin": "1373458",
        "ertugliflozin": "1992672", "sotagliflozin": "2169285",
        # MRAs
        "finerenone": "2555902", "spironolactone": "9997", "eplerenone": "298869",
        # ACE inhibitors
        "lisinopril": "29046", "enalapril": "3827", "ramipril": "35296",
        "perindopril": "54552",
        # ARBs
        "losartan": "52175", "valsartan": "69749", "irbesartan": "83818",
        "candesartan": "214354", "telmisartan": "73494", "olmesartan": "321064",
        # GLP-1 RAs
        "semaglutide": "1991302", "liraglutide": "475968",
        "dulaglutide": "1551291", "exenatide": "60548",
        # DPP-4 inhibitors
        "sitagliptin": "593411", "linagliptin": "1100699",
        "saxagliptin": "857974", "alogliptin": "1368001",
        # TZDs
        "pioglitazone": "33738", "rosiglitazone": "84108",
        # Sulfonylureas
        "glyburide": "4815", "glipizide": "4821", "glimepiride": "25789",
        # Insulin
        "insulin": "5856",
        # Diuretics
        "furosemide": "4603", "bumetanide": "1808",
        "hydrochlorothiazide": "5487", "chlorthalidone": "2409",
        "indapamide": "5764",
        # CCBs
        "amlodipine": "17767", "nifedipine": "7417",
        "diltiazem": "3443", "verapamil": "11170",
        # Beta blockers
        "atenolol": "1202", "metoprolol": "6918",
        "carvedilol": "20352", "bisoprolol": "19484",
        # Statins
        "atorvastatin": "83367", "rosuvastatin": "301542",
    }

    # Drug classes: lowercase name → canonical class label
    _DRUG_CLASSES: dict[str, str] = {
        "sglt2 inhibitor": "SGLT2i", "sglt2 inhibitors": "SGLT2i",
        "sglt2i": "SGLT2i", "sglt-2 inhibitor": "SGLT2i",
        "ace inhibitor": "ACEi", "ace inhibitors": "ACEi",
        "acei": "ACEi", "ace-i": "ACEi",
        "angiotensin receptor blocker": "ARB",
        "angiotensin receptor blockers": "ARB", "arb": "ARB", "arbs": "ARB",
        "glp-1 receptor agonist": "GLP-1 RA", "glp-1 receptor agonists": "GLP-1 RA",
        "glp-1 ra": "GLP-1 RA", "glp1-ra": "GLP-1 RA",
        "glp-1 agonist": "GLP-1 RA", "glp-1 agonists": "GLP-1 RA",
        "mineralocorticoid receptor antagonist": "MRA",
        "mineralocorticoid receptor antagonists": "MRA",
        "mra": "MRA", "mras": "MRA", "ns-mra": "nsMRA",
        "dpp-4 inhibitor": "DPP-4i", "dpp-4 inhibitors": "DPP-4i",
        "dpp4i": "DPP-4i", "dpp-4i": "DPP-4i",
        "sulfonylurea": "Sulfonylurea", "sulfonylureas": "Sulfonylurea",
        "thiazolidinedione": "TZD", "thiazolidinediones": "TZD",
        "rasi": "RASi", "ras inhibitor": "RASi", "ras inhibitors": "RASi",
        "beta blocker": "Beta-blocker", "beta blockers": "Beta-blocker",
        "beta-blocker": "Beta-blocker", "beta-blockers": "Beta-blocker",
        "calcium channel blocker": "CCB", "calcium channel blockers": "CCB",
        "diuretic": "Diuretic", "diuretics": "Diuretic",
        "loop diuretic": "Loop diuretic", "loop diuretics": "Loop diuretic",
        "statin": "Statin", "statins": "Statin",
        "nsaid": "NSAID", "nsaids": "NSAID",
    }

    # Canonical class label → set of ingredient names belonging to that class.
    # Used for dedup: when BOTH a class AND its member drugs are detected,
    # the class dossier is dropped (member drugs have specific dosing data).
    _CLASS_TO_MEMBERS: dict[str, set[str]] = {
        "SGLT2i": {"dapagliflozin", "empagliflozin", "canagliflozin",
                    "ertugliflozin", "sotagliflozin"},
        "ACEi": {"lisinopril", "enalapril", "ramipril", "perindopril"},
        "ARB": {"losartan", "valsartan", "irbesartan", "candesartan",
                "telmisartan", "olmesartan"},
        "GLP-1 RA": {"semaglutide", "liraglutide", "dulaglutide", "exenatide"},
        "MRA": {"spironolactone", "eplerenone"},
        "nsMRA": {"finerenone"},
        "DPP-4i": {"sitagliptin", "linagliptin", "saxagliptin", "alogliptin"},
        "Sulfonylurea": {"glyburide", "glipizide", "glimepiride"},
        "TZD": {"pioglitazone", "rosiglitazone"},
        "RASi": {"lisinopril", "enalapril", "ramipril", "perindopril",
                 "losartan", "valsartan", "irbesartan", "candesartan",
                 "telmisartan", "olmesartan"},
        "Beta-blocker": {"atenolol", "metoprolol", "carvedilol", "bisoprolol"},
        "CCB": {"amlodipine", "nifedipine", "diltiazem", "verapamil"},
        "Diuretic": {"furosemide", "bumetanide", "hydrochlorothiazide",
                     "chlorthalidone", "indapamide"},
        "Loop diuretic": {"furosemide", "bumetanide"},
        "Statin": {"atorvastatin", "rosuvastatin"},
    }

    # Pre-compiled word-boundary patterns for each drug name (case-insensitive).
    # Sorted longest-first so "dpp-4 inhibitors" matches before "dpp-4".
    _ALL_DRUG_NAMES: list[tuple[str, str, re.Pattern]] = sorted(
        [
            (name, rxnorm, re.compile(rf"\b{re.escape(name)}\b", re.IGNORECASE))
            for name, rxnorm in _DRUG_INGREDIENTS.items()
        ] + [
            (name, cls_label, re.compile(rf"\b{re.escape(name)}\b", re.IGNORECASE))
            for name, cls_label in _DRUG_CLASSES.items()
        ],
        key=lambda x: -len(x[0]),  # longest match first
    )

    # ── Public API ──────────────────────────────────────────────────────

    def assemble(
        self,
        verified_spans: list[VerifiedSpan],
        tree: GuidelineTree,
        text: str,
    ) -> list[DrugDossier]:
        """Assemble per-drug dossiers from verified spans.

        Args:
            verified_spans: Reviewer-approved spans (CONFIRMED/EDITED/ADDED)
            tree: GuidelineTree from Channel A
            text: Full normalized text

        Returns:
            List of DrugDossier objects, one per drug found
        """
        # Step 1: Scan ALL spans for drug mentions
        span_drugs = self._detect_drugs_in_spans(verified_spans)

        # Step 2: Separate drug-mentioning spans from orphan signals
        drug_spans: dict[str, list[VerifiedSpan]] = defaultdict(list)
        orphan_spans: list[VerifiedSpan] = []
        drug_rxnorm: dict[str, str] = {}  # drug_name → RxNorm CUI

        for span in verified_spans:
            drugs_found = span_drugs.get(id(span), [])
            if drugs_found:
                for drug_name, rxnorm_or_class in drugs_found:
                    # Canonicalize: drug classes → canonical label (ARB, MRA, SGLT2i)
                    #               drug ingredients → lowercase name (metformin)
                    if rxnorm_or_class and not rxnorm_or_class.isdigit():
                        # Class match: "arbs"→"ARB", "mra"→"MRA", etc.
                        canonical = rxnorm_or_class
                    else:
                        # Ingredient match: "metformin"→"metformin"
                        canonical = drug_name.lower()
                    drug_spans[canonical].append(span)
                    # Store RxNorm if it's a CUI (numeric), not a class label
                    if rxnorm_or_class and rxnorm_or_class.isdigit():
                        drug_rxnorm[canonical] = rxnorm_or_class
            else:
                orphan_spans.append(span)

        if not drug_spans:
            return []

        # Step 2b: Class-vs-member dedup
        # When BOTH a class (SGLT2i) AND its individual drugs (dapagliflozin,
        # empagliflozin) are detected, drop the class dossier to avoid:
        #   - Double L3 extraction (class + N individuals = wasted API calls)
        #   - Logical collision in L5 CQL (overlapping defines)
        #   - L4 validation noise (classes have no RxCUI)
        # Class spans are redistributed to member drugs so coverage isn't lost.
        classes_to_remove = []
        for class_label, members in self._CLASS_TO_MEMBERS.items():
            if class_label not in drug_spans:
                continue
            # Check if ANY member drug is also detected
            detected_members = [m for m in members if m in drug_spans]
            if detected_members:
                classes_to_remove.append((class_label, detected_members))

        for class_label, detected_members in classes_to_remove:
            class_spans_list = drug_spans.pop(class_label)
            # Redistribute class-only spans to member drugs
            for span in class_spans_list:
                for member in detected_members:
                    if span not in drug_spans[member]:
                        drug_spans[member].append(span)
            drug_rxnorm.pop(class_label, None)

        if not drug_spans:
            return []

        # Step 3: Associate orphan spans with drugs via section proximity
        drug_anchors = [
            (name, spans[0]) for name, spans in drug_spans.items()
        ]
        for orphan in orphan_spans:
            associated = self._associate_signal(orphan, drug_anchors, tree)
            for drug_name in associated:
                drug_spans[drug_name].append(orphan)

        # Step 4: Build per-drug dossiers
        dossiers: list[DrugDossier] = []
        for drug_name, spans in drug_spans.items():
            dossier = self._build_dossier(
                drug_name, spans, drug_rxnorm.get(drug_name), tree, text,
            )
            dossiers.append(dossier)

        return dossiers

    # ── Drug Detection ──────────────────────────────────────────────────

    def _detect_drugs_in_spans(
        self, spans: list[VerifiedSpan],
    ) -> dict[int, list[tuple[str, str]]]:
        """Scan every span's text for drug name mentions.

        Returns:
            Dict mapping span id(span) → list of (drug_name, rxnorm_or_class_label).
            A span can mention multiple drugs (e.g., "metformin and dapagliflozin").
        """
        result: dict[int, list[tuple[str, str]]] = {}

        for span in spans:
            text_lower = span.text.lower()
            drugs_in_span: list[tuple[str, str]] = []
            matched_positions: set[tuple[int, int]] = set()

            for drug_name, rxnorm_or_class, pattern in self._ALL_DRUG_NAMES:
                for m in pattern.finditer(text_lower):
                    pos = (m.start(), m.end())
                    # Avoid overlapping matches (longer patterns matched first)
                    if any(
                        pos[0] < ep and pos[1] > sp
                        for sp, ep in matched_positions
                    ):
                        continue
                    matched_positions.add(pos)
                    drugs_in_span.append((drug_name, rxnorm_or_class))

            if drugs_in_span:
                result[id(span)] = drugs_in_span

        return result

    # ── Signal Association ──────────────────────────────────────────────

    def _associate_signal(
        self,
        signal: VerifiedSpan,
        drug_anchors: list[tuple[str, VerifiedSpan]],
        tree: GuidelineTree,
    ) -> list[str]:
        """Associate an orphan signal span with one or more drugs.

        Priority:
        1. Same section + proximity tie-breaking
        2. Nearest drug globally (if no drugs in same section)
        """
        signal_section = signal.section_id
        same_section_drugs: list[tuple[str, int]] = []

        for drug_name, anchor in drug_anchors:
            if anchor.section_id == signal_section and signal_section is not None:
                distance = abs(signal.start - anchor.start)
                same_section_drugs.append((drug_name, distance))

        if not same_section_drugs:
            # No drugs in same section — find nearest drug globally
            nearest_drug = None
            nearest_dist = float('inf')
            for drug_name, anchor in drug_anchors:
                dist = abs(signal.start - anchor.start)
                if dist < nearest_dist:
                    nearest_dist = dist
                    nearest_drug = drug_name
            return [nearest_drug] if nearest_drug else []

        if len(same_section_drugs) == 1:
            return [same_section_drugs[0][0]]

        # Multiple drugs in section — use proximity tie-breaking
        same_section_drugs.sort(key=lambda x: x[1])
        closest_drug, closest_dist = same_section_drugs[0]

        if closest_dist <= self.PROXIMITY_THRESHOLD:
            return [closest_drug]

        # Not within proximity of any single drug → associate with all
        return [drug for drug, _ in same_section_drugs]

    # ── Dossier Construction ────────────────────────────────────────────

    def _build_dossier(
        self,
        drug_name: str,
        spans: list[VerifiedSpan],
        rxnorm_candidate: Optional[str],
        tree: GuidelineTree,
        text: str,
    ) -> DrugDossier:
        """Build a DrugDossier for a single drug."""
        # Deduplicate spans by (start, end, text)
        seen: set[tuple[int, int, str]] = set()
        unique_spans: list[VerifiedSpan] = []
        for span in spans:
            key = (span.start, span.end, span.text)
            if key not in seen:
                seen.add(key)
                unique_spans.append(span)

        # Collect source sections and pages
        source_sections = sorted({
            s.section_id for s in unique_spans if s.section_id
        })
        source_pages = sorted({
            s.page_number for s in unique_spans if s.page_number
        })

        # RxNorm: use text-detected CUI, or Channel B metadata if available
        rxnorm = rxnorm_candidate
        if rxnorm is None:
            # Check extraction_context as optional enrichment
            for span in unique_spans:
                ctx_rxnorm = span.extraction_context.get("channel_B_rxnorm_candidate")
                if ctx_rxnorm:
                    rxnorm = ctx_rxnorm
                    break

        # Extract source text from enclosing sections
        source_text = self._get_source_text(source_sections, tree, text)

        # Build signal summary from contributing channels
        signal_summary = self._summarize_signals(unique_spans)

        return DrugDossier(
            drug_name=drug_name,
            rxnorm_candidate=rxnorm,
            verified_spans=unique_spans,
            source_sections=source_sections,
            source_pages=source_pages,
            source_text=source_text,
            signal_summary=signal_summary,
        )

    # ── Helpers ─────────────────────────────────────────────────────────

    def _get_source_text(
        self,
        section_ids: list[str],
        tree: GuidelineTree,
        text: str,
    ) -> str:
        """Extract text from the specified sections."""
        all_sections = self._flatten_sections(tree.sections)
        section_texts = []

        for section in all_sections:
            if section.section_id in section_ids:
                section_texts.append(
                    text[section.start_offset:section.end_offset]
                )

        return "\n\n".join(section_texts)

    def _flatten_sections(self, sections):
        """Flatten nested sections into a flat list."""
        result = []
        for s in sections:
            result.append(s)
            result.extend(self._flatten_sections(s.children))
        return result

    def _summarize_signals(self, spans: list[VerifiedSpan]) -> dict:
        """Build a signal summary from verified spans.

        Uses contributing_channels (always available) rather than
        extraction_context metadata (may not survive GCP round-trip).
        """
        summary: dict[str, int] = defaultdict(int)

        for span in spans:
            channels = set(span.contributing_channels)
            if "B" in channels:
                summary["drug_match"] += 1
            elif "C" in channels:
                summary["clinical_pattern"] += 1
            elif "D" in channels:
                summary["table_cell"] += 1
            elif "E" in channels or "F" in channels:
                summary["proposition"] += 1
            elif "REVIEWER" in channels:
                summary["reviewer_added"] += 1
            elif "L1_RECOVERY" in channels:
                summary["l1_recovery"] += 1
            else:
                summary["other"] += 1

        return dict(summary)
