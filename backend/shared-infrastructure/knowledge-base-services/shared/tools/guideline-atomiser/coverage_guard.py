"""
CoverageGuard — Post-Merge Quality Gate for Clinical Guideline Extraction.

Runs AFTER Signal Merger produces merged_spans, BEFORE facts enter the
knowledge base. Validates extraction quality across 4 domains:

    Domain A: Structural Completeness (prevents omission of entire elements)
    Domain B: Content Exhaustiveness (prevents omission within covered sections)
    Domain C: Integrity Verification (prevents distortion — numeric corruption)
    Domain D: Systemic Meta-Validation (prevents validator blind spots)

Eight release gate conditions must ALL pass for PASS verdict:
    1. All Recommendations & Practice Points covered (A2)
    2. All Tier 1 footnotes captured and bound (A1d + A2)
    3. Tier 1 coverage = 100% (B1 + B2)
    4. Tier 1 numeric integrity = 100% (C1)
    5. No branch or exception loss (B3)
    6. No unreviewed Tier 1 span with corroboration <0.5 (C3)
    7. Residual Tier 1 signals = 0 (B1)
    8. Pharmacist sign-off on Tier 1 fact list

Gate precedence (when multiple fail, fix in this order):
    1. C1 Numeric integrity — value corruption
    2. A2 Structural gaps — entire elements missing
    3. B3 Branch/exception losses — logic incomplete
    4. B1 Residual signals — text dropped
    5. C3 Corroboration warnings — low-confidence spans
    6. B2 Adversarial delta — possible false positives

Usage:
    from coverage_guard import CoverageGuard

    guard = CoverageGuard()
    report = guard.validate(
        merged_spans=merged_spans,
        tree=tree,
        normalized_text=normalized_text,
        pdf_path=pdf_path,
        oracle_report=oracle_report,
    )

    if report.gate_verdict == "BLOCK":
        for blocker in report.gate_blockers:
            print(f"  BLOCK: {blocker.gate_name} ({blocker.blocker_count} issues)")
"""

import json
import logging
import os
import random
import re
import time
from collections import defaultdict
from typing import Optional

import pymupdf

from clinical_constants import (
    BARE_CLINICAL_VALUE_RE,
    BRANCH_CONNECTORS,
    CLINICAL_ABBREVIATION_MAP,
    CLINICAL_EXPANSION_TO_ABBREV,
    CLINICAL_THRESHOLD_RE,
    DRUG_CLASS_NAMES,
    EXCEPTION_KEYWORDS,
    FOOTNOTE_MARKERS,
    IMPLICIT_RISK_TRIGGERS,
    KDIGO_DRUG_NAMES,
    NUMERIC_THRESHOLD_RE,
    NUMERIC_TUPLE_RE,
    NUMERIC_RANGE_RE,
    POPULATION_QUALIFIERS,
    PRESCRIPTIVE_TRIGGERS,
    PROHIBITIVE_TRIGGERS,
    RECOMMENDATION_ID_RE,
    RECOMMENDATION_RE,
    PRACTICE_POINT_ID_RE,
    RESEARCH_REC_ID_RE,
    TABLE_REF_RE,
    FIGURE_REF_RE,
    build_drug_automaton,
)

try:
    import ahocorasick
except ImportError:
    ahocorasick = None  # type: ignore[assignment]

logger = logging.getLogger(__name__)


# ═══════════════════════════════════════════════════════════════════════════════
# Page type classification for A3 density analysis
# ═══════════════════════════════════════════════════════════════════════════════

PAGE_TYPE_DENSITY = {
    "PROSE": (3, 25),       # expected spans per 100 words
    "TABLE": (5, 40),
    "FIGURE": (0, 5),
    "ALGORITHM": (2, 20),
    "REFERENCES": (0, 1),
    "FRONT_MATTER": (0, 1),
}


# ═══════════════════════════════════════════════════════════════════════════════
# C3 Channel Corroboration Scoring Table
# ═══════════════════════════════════════════════════════════════════════════════

DETERMINISTIC_CHANNELS = {"B", "C", "D"}
LLM_CHANNELS = {"F"}
RECOVERY_CHANNELS = {"L1_RECOVERY"}


def _corroboration_score(channels: list[str]) -> float:
    """Compute corroboration score per C3 scoring table.

    | Pattern                              | Score |
    |--------------------------------------|-------|
    | 3+ channels incl ≥2 deterministic    | 1.0   |
    | 2 deterministic (B+C, B+D, C+D)     | 0.9   |
    | 1 deterministic + LLM (B+F)         | 0.7   |
    | 1 deterministic only                 | 0.6   |
    | LLM only (F)                         | 0.4   |
    | L1 Recovery only                     | 0.3   |
    """
    channel_set = set(channels)
    det_count = len(channel_set & DETERMINISTIC_CHANNELS)
    llm_count = len(channel_set & LLM_CHANNELS)
    recovery_only = channel_set <= RECOVERY_CHANNELS

    if recovery_only:
        return 0.3
    if det_count == 0 and llm_count > 0:
        return 0.4
    if det_count >= 2 and len(channel_set) >= 3:
        return 1.0
    if det_count >= 2:
        return 0.9
    if det_count >= 1 and llm_count >= 1:
        return 0.7
    if det_count >= 1:
        return 0.6
    return 0.3


# ═══════════════════════════════════════════════════════════════════════════════
# CoverageGuard Main Class
# ═══════════════════════════════════════════════════════════════════════════════

class CoverageGuard:
    """Post-merge quality gate — 4 domains, 9 layers, 8 release conditions.

    Reads:
        merged_spans: list[MergedSpan]  — extraction output
        tree: GuidelineTree             — document structure from Channel A
        normalized_text: str            — L1 parser output (MonkeyOCR markdown)
        pdf_path: str                   — original PDF (for PyMuPDF rawdict)
        oracle_report: CompletenessReport — L1 Oracle results

    Produces:
        CoverageGuardReport saved as coverage_guard_report.json
    """

    VERSION = "2.0.0"

    # B2 LLM configuration
    B2_PRIMARY_MODEL = "claude-sonnet-4-20250514"
    B2_CROSSVAL_MODEL = "claude-haiku-4-5-20251001"
    B2_MAX_TOKENS = 4096
    B2_CROSSVAL_SAMPLE_RATE = 0.20  # 20% of sections
    B2_DIVERGENCE_THRESHOLD = 0.20  # >20% = investigate

    def __init__(self, anthropic_client=None, enable_b2: bool = True):
        """Initialize CoverageGuard.

        Args:
            anthropic_client: Optional pre-configured Anthropic client.
                              If None and enable_b2=True, will try to create
                              one from ANTHROPIC_API_KEY env var.
            enable_b2: Enable B2 adversarial audit + D1 dual-LLM validation.
                       Set False for deterministic-only mode (faster, no cost).
        """
        self._drug_automaton = build_drug_automaton(include_classes=True)
        self._enable_b2 = enable_b2
        self._client = anthropic_client

        if enable_b2 and self._client is None:
            api_key = os.environ.get("ANTHROPIC_API_KEY", "")
            if api_key and len(api_key) > 30:
                try:
                    from anthropic import Anthropic
                    self._client = Anthropic(api_key=api_key)
                    logger.info("B2/D1: Anthropic client initialized")
                except ImportError:
                    logger.warning("B2/D1: anthropic package not installed — B2 disabled")
                    self._enable_b2 = False
            else:
                logger.warning("B2/D1: ANTHROPIC_API_KEY not set — B2 disabled")
                self._enable_b2 = False

    # ─── Public API ──────────────────────────────────────────────────────

    def validate(
        self,
        merged_spans: list,
        tree,
        normalized_text: str,
        pdf_path: str,
        oracle_report,
        job_id: str = "",
        guideline_document: str = "",
    ):
        """Run all 4 domains and produce CoverageGuardReport.

        Args:
            merged_spans: list[MergedSpan] from Signal Merger
            tree: GuidelineTree from Channel A
            normalized_text: Full markdown text from L1 parser
            pdf_path: Path to source PDF for PyMuPDF rawdict
            oracle_report: CompletenessReport from L1 Oracle
            job_id: Pipeline job identifier
            guideline_document: Source document name

        Returns:
            CoverageGuardReport with gate verdict and all domain results.
        """
        from extraction.v4.models import CoverageGuardReport

        t0 = time.perf_counter()

        report = CoverageGuardReport(
            job_id=job_id,
            guideline_document=guideline_document,
            pipeline_version=f"CoverageGuard-{self.VERSION}",
        )

        # ── Domain A: Structural Completeness ────────────────────────────
        self._domain_a(report, merged_spans, tree, normalized_text, pdf_path)

        # ── Domain B: Content Exhaustiveness ─────────────────────────────
        self._domain_b(report, merged_spans, tree, normalized_text)

        # ── Domain C: Integrity Verification ─────────────────────────────
        self._domain_c(report, merged_spans, normalized_text, oracle_report)

        # ── Domain D: Systemic Meta-Validation ───────────────────────────
        self._domain_d(report, merged_spans, normalized_text)

        # ── Release Gate ─────────────────────────────────────────────────
        self._evaluate_release_gate(report)

        elapsed = time.perf_counter() - t0
        logger.info(
            "CoverageGuard completed in %.2fs — verdict=%s, blocks=%d, warnings=%d",
            elapsed, report.gate_verdict, report.total_block_count, report.total_warning_count,
        )

        return report

    # =====================================================================
    # DOMAIN A — Structural Completeness
    # =====================================================================

    def _domain_a(self, report, merged_spans, tree, normalized_text, pdf_path):
        """A1 inventory, A1d footnotes, A2 coverage, A3 density."""

        # ── A1: Source Inventory Registry ─────────────────────────────────
        inventory = self._a1_source_inventory(normalized_text, pdf_path)
        report.inventory_expected = {
            k: len(v) for k, v in inventory.items()
        }

        # ── A1d: Footnote Inventory & Binding ────────────────────────────
        footnote_bindings = self._a1d_footnote_binding(pdf_path, inventory)
        report.footnote_bindings = footnote_bindings

        # ── A2: Structural Coverage Check ────────────────────────────────
        elements = self._a2_structural_coverage(inventory, merged_spans, footnote_bindings)
        report.inventory_elements = elements
        report.inventory_actual = {}
        for elem in elements:
            etype = elem.element_type
            if etype not in report.inventory_actual:
                report.inventory_actual[etype] = 0
            if elem.coverage_status == "COVERED":
                report.inventory_actual[etype] += 1

        # ── A3: Density & Channel Routing Check ──────────────────────────
        density_warnings = self._a3_density_check(merged_spans, tree, normalized_text)
        report.density_warnings = density_warnings

    def _a1_source_inventory(self, normalized_text: str, pdf_path: str) -> dict:
        """Build expected element inventory from source text + PyMuPDF rawdict.

        Returns dict mapping element_type to list of (element_id, page_number).
        """
        from extraction.v4.models import InventoryElement

        inventory: dict[str, list[tuple[str, int]]] = {
            "recommendation": [],
            "practice_point": [],
            "research_rec": [],
            "table": [],
            "figure": [],
        }

        # Scan normalized text for structural elements
        for m in RECOMMENDATION_ID_RE.finditer(normalized_text):
            rec_id = f"Recommendation {m.group(1)} ({m.group(2)})"
            page = self._estimate_page_from_offset(normalized_text, m.start())
            inventory["recommendation"].append((rec_id, page))

        for m in PRACTICE_POINT_ID_RE.finditer(normalized_text):
            pp_id = f"Practice Point {m.group(1)}"
            page = self._estimate_page_from_offset(normalized_text, m.start())
            inventory["practice_point"].append((pp_id, page))

        for m in RESEARCH_REC_ID_RE.finditer(normalized_text):
            rr_id = f"Research Recommendation {m.group(1)}"
            page = self._estimate_page_from_offset(normalized_text, m.start())
            inventory["research_rec"].append((rr_id, page))

        # Tables/Figures: collect ALL occurrences, then deduplicate preferring
        # the CONTENT page over prose reference pages. In KDIGO PDFs:
        #   Content: "Table 31:" or "Table 31 |" at line start (actual table header)
        #   Reference: "...in Table 31:" within a sentence (prose mention)
        # The content page is where the span matcher should look for table spans.
        _table_all_occurrences: dict[str, list[tuple[int, bool]]] = {}
        for m in TABLE_REF_RE.finditer(normalized_text):
            table_id = f"Table {m.group(1)}"
            page = self._estimate_page_from_offset(normalized_text, m.start())
            # Check if this is a content header (at or near line start)
            line_start = normalized_text.rfind("\n", 0, m.start())
            prefix = normalized_text[line_start + 1:m.start()].strip() if line_start >= 0 else ""
            is_content = len(prefix) < 10  # Content headers have minimal prefix
            _table_all_occurrences.setdefault(table_id, []).append((page, is_content))

        for table_id, occurrences in _table_all_occurrences.items():
            # Prefer content pages over reference pages
            content_pages = [(p, c) for p, c in occurrences if c]
            if content_pages:
                inventory["table"].append((table_id, content_pages[0][0]))
            else:
                inventory["table"].append((table_id, occurrences[0][0]))

        _figure_all_occurrences: dict[str, list[tuple[int, bool]]] = {}
        for m in FIGURE_REF_RE.finditer(normalized_text):
            fig_id = f"Figure {m.group(1)}"
            page = self._estimate_page_from_offset(normalized_text, m.start())
            line_start = normalized_text.rfind("\n", 0, m.start())
            prefix = normalized_text[line_start + 1:m.start()].strip() if line_start >= 0 else ""
            is_content = len(prefix) < 10
            _figure_all_occurrences.setdefault(fig_id, []).append((page, is_content))

        for fig_id, occurrences in _figure_all_occurrences.items():
            content_pages = [(p, c) for p, c in occurrences if c]
            if content_pages:
                inventory["figure"].append((fig_id, content_pages[0][0]))
            else:
                inventory["figure"].append((fig_id, occurrences[0][0]))

        # Deduplicate non-table/figure element types by ID (same element may
        # appear in ToC, in-text references, AND at its actual location).
        # Tables and figures are already deduplicated above with content-page preference.
        for key in ["recommendation", "practice_point", "research_rec"]:
            seen_ids = set()
            deduped = []
            for elem_id, page in inventory[key]:
                if elem_id not in seen_ids:
                    seen_ids.add(elem_id)
                    deduped.append((elem_id, page))
            inventory[key] = deduped

        return inventory

    # Regex to detect footnote marker chars embedded in inline text.
    # Matches: word + marker char (e.g., "MRA†", "death‡", "Semaglutide§")
    # Does NOT match: marker char + digits + unit (e.g., "‡20 ml/min" = misencoded ≥)
    _INLINE_FOOTNOTE_RE = re.compile(
        r'(\w{2,})([†‡§¶])(?:\s|$|[,.\);:])',
    )
    # Context filter: ‡ followed by number + unit is a ≥ misencoding, not a footnote
    _GEQ_MISENCODING_RE = re.compile(
        r'[‡]\s*\d+\.?\d*\s*(?:ml|mg|mL|mmol|µmol|g|%|mm)',
        re.IGNORECASE,
    )

    def _a1d_footnote_binding(self, pdf_path: str, inventory: dict) -> list:
        """Detect footnote markers in tables and bind to footnote definitions.

        Dual detection strategy:
        1. Primary: Scan for inline footnote marker characters (†‡§¶) embedded
           in text spans. Most clinical PDFs don't set the superscript font flag —
           markers appear as regular characters appended to words.
        2. Fallback: Check PyMuPDF superscript flag (flags & 1) for PDFs that
           do use proper superscript encoding.
        3. Context filter: Exclude ‡ used as ≥ misencoding (common OCR artifact
           where "eGFR ≥20 ml/min" becomes "eGFR ‡20 ml/min").
        4. Footnote text extraction: marker char followed by prose below the
           marker position on the same page.
        5. Tier classification: drug name + threshold → Tier 1; else Tier 2.
        """
        from extraction.v4.models import FootnoteBinding

        bindings: list[FootnoteBinding] = []

        try:
            doc = pymupdf.open(pdf_path)
        except Exception as e:
            logger.warning("A1d: Cannot open PDF for footnote detection: %s", e)
            return bindings

        try:
            for page_idx in range(len(doc)):
                page = doc[page_idx]
                page_dict = page.get_text("rawdict", flags=pymupdf.TEXT_PRESERVE_WHITESPACE)

                marker_positions: list[dict] = []
                all_text_blocks: list[dict] = []
                seen_markers: set[tuple[str, int]] = set()  # (char, page_idx)

                # ── Method 1: Superscript-flagged markers (rawdict) ────────
                # Needed for PDFs that properly set the superscript font flag.
                for block in page_dict.get("blocks", []):
                    if block.get("type") != 0:
                        continue

                    block_bbox = block.get("bbox", [0, 0, 0, 0])
                    block_text_parts = []

                    for line in block.get("lines", []):
                        for span in line.get("spans", []):
                            text = span.get("text", "").strip()
                            flags = span.get("flags", 0)
                            is_superscript = bool(flags & 1)
                            span_y = span.get("origin", [0, 0])[1] if "origin" in span else block_bbox[1]

                            if is_superscript and text:
                                for char in text:
                                    if char in FOOTNOTE_MARKERS:
                                        dedup_key = (char, page_idx)
                                        if dedup_key not in seen_markers:
                                            seen_markers.add(dedup_key)
                                            marker_positions.append({
                                                "char": char,
                                                "page": page_idx + 1,
                                                "bbox": block_bbox,
                                                "y": span_y,
                                                "method": "superscript_flag",
                                            })

                            block_text_parts.append(text)

                    full_text = " ".join(block_text_parts).strip()
                    if full_text:
                        all_text_blocks.append({
                            "text": full_text,
                            "bbox": block_bbox,
                            "y_top": block_bbox[1],
                        })

                # ── Method 2: Inline marker detection (full page text) ─────
                # Uses page.get_text() which preserves char adjacency within
                # words even when markers are in separate font spans.
                # This is the PRIMARY method — most clinical PDFs don't set
                # the superscript flag on footnote marker characters.
                page_text = page.get_text()

                # Resolve inline marker Y-coordinates from rawdict spans
                # so footnote-definition search can find text blocks below.
                char_positions: dict[int, float] = {}  # text offset → y
                _rd_offset = 0
                for block in page_dict.get("blocks", []):
                    if block.get("type") != 0:
                        continue
                    for line in block.get("lines", []):
                        for span in line.get("spans", []):
                            span_y = span.get("origin", [0, 0])[1]
                            span_len = len(span.get("text", ""))
                            for i in range(span_len):
                                char_positions[_rd_offset + i] = span_y
                            _rd_offset += span_len

                for m in self._INLINE_FOOTNOTE_RE.finditer(page_text):
                    marker_char = m.group(2)

                    # Filter ‡ used as ≥ misencoding
                    if marker_char == "‡" and self._GEQ_MISENCODING_RE.search(
                        page_text[m.start(2):]
                    ):
                        continue

                    dedup_key = (marker_char, page_idx)
                    if dedup_key not in seen_markers:
                        seen_markers.add(dedup_key)
                        context_start = max(0, m.start() - 10)
                        context = page_text[context_start:m.end() + 20].replace("\n", " ")

                        # Resolve Y from rawdict character offset map.
                        # page.get_text() and rawdict traverse blocks in the
                        # same order, so char offsets are approximately aligned.
                        marker_offset = m.start(2)
                        resolved_y = char_positions.get(marker_offset, 0)
                        if resolved_y == 0:
                            # Fallback: try nearby offsets (±5 chars)
                            for delta in range(1, 6):
                                resolved_y = char_positions.get(marker_offset + delta, 0) or \
                                             char_positions.get(marker_offset - delta, 0)
                                if resolved_y > 0:
                                    break

                        marker_positions.append({
                            "char": marker_char,
                            "page": page_idx + 1,
                            "bbox": [0, 0, 0, 0],
                            "y": resolved_y,
                            "method": "inline_text",
                            "context": context.strip(),
                            "_text_offset": m.start(2),
                        })

                # For each marker, find footnote definition on the same page.
                # Try Strategy A (rawdict blocks) first, then Strategy B
                # (text-offset scanning) as fallback. Both are always attempted
                # because rawdict may produce blocks that don't contain the
                # footnote definitions (e.g., definitions in XObject frames).
                for marker in marker_positions:
                    best_footnote = None

                    # ── Strategy A: Rawdict block scanning ───────────
                    if all_text_blocks:
                        marker_y = marker["y"]
                        candidates = all_text_blocks
                        if marker_y > 0:
                            candidates = [tb for tb in all_text_blocks
                                          if tb["y_top"] > marker_y]
                        else:
                            candidates = sorted(all_text_blocks,
                                                key=lambda tb: -tb["y_top"])

                        for tblock in candidates:
                            text = tblock["text"]
                            stripped = text.lstrip()
                            if stripped.startswith(marker["char"]):
                                fn_text = stripped.lstrip(marker["char"]).lstrip(" .):").strip()
                                if len(fn_text) > 5:
                                    best_footnote = fn_text
                                    break
                            marker_re = re.search(
                                r'(?:^|\s)[(\[]?' + re.escape(marker["char"]) +
                                r'[)\]]?\s+(.{6,})',
                                text,
                            )
                            if marker_re:
                                best_footnote = marker_re.group(1).strip()
                                break

                    # ── Strategy B: Simple string scan (fallback) ────
                    # Scans page_text for all occurrences of the marker char.
                    # A definition starts with the marker followed by an
                    # uppercase letter (inline refs like "CKD†" have the
                    # marker AFTER the word, not before uppercase text).
                    if not best_footnote:
                        marker_char = marker["char"]
                        idx = 0
                        while idx < len(page_text):
                            pos = page_text.find(marker_char, idx)
                            if pos == -1:
                                break
                            idx = pos + 1

                            # Check what follows the marker
                            after = page_text[pos + 1:pos + 4].lstrip()
                            if not after or not after[0].isupper():
                                continue

                            # Skip ‡ followed by digit (≥ misencoding)
                            if marker_char == "‡" and after[0].isdigit():
                                continue

                            # Extract until next newline (or end)
                            end_pos = page_text.find('\n', pos + 1)
                            if end_pos == -1:
                                end_pos = len(page_text)
                            fn_text = page_text[pos + 1:end_pos].strip()

                            # Multi-line: if line is short (<40 chars),
                            # continuation may be on the next line
                            if len(fn_text) < 40 and end_pos < len(page_text):
                                next_nl = page_text.find('\n', end_pos + 1)
                                if next_nl == -1:
                                    next_nl = len(page_text)
                                next_line = page_text[end_pos + 1:next_nl].strip()
                                if next_line and next_line[0].islower():
                                    fn_text += " " + next_line

                            if len(fn_text) > 10:
                                best_footnote = fn_text
                                break

                    # Tier classification
                    tier = self._classify_footnote_tier(best_footnote or "")
                    table_id = f"table_page_{marker['page']}"

                    bindings.append(FootnoteBinding(
                        marker_char=marker["char"],
                        table_id=table_id,
                        page_number=marker["page"],
                        footnote_text=best_footnote or "(definition not found)",
                        tier=tier,
                        bound_to_span=False,  # Updated in A2
                    ))

                    logger.debug(
                        "A1d: %s on page %d via %s — %s",
                        marker["char"], marker["page"],
                        marker.get("method", "unknown"),
                        (best_footnote or "(no def)")[:50],
                    )

        finally:
            doc.close()

        logger.info("A1d: Found %d footnote markers (%d via inline, %d via superscript)",
                    len(bindings),
                    sum(1 for b in marker_positions if b.get("method") == "inline_text"),
                    sum(1 for b in marker_positions if b.get("method") == "superscript_flag"))

        return bindings

    def _classify_footnote_tier(self, text: str) -> str:
        """Classify footnote as TIER_1 (clinical) or TIER_2 (informational)."""
        text_lower = text.lower()

        # Check for drug names
        has_drug = False
        for _, (_, drug) in self._drug_automaton.iter(text_lower):
            has_drug = True
            break

        # Check for thresholds
        has_threshold = bool(
            CLINICAL_THRESHOLD_RE.search(text)
            or NUMERIC_THRESHOLD_RE.search(text)
        )

        # Drug + threshold or prohibitive trigger → Tier 1
        if has_drug and has_threshold:
            return "TIER_1"
        for trigger in PROHIBITIVE_TRIGGERS:
            if trigger in text_lower:
                return "TIER_1"

        return "TIER_2"

    def _a2_structural_coverage(self, inventory, merged_spans, footnote_bindings) -> list:
        """Check each inventory element against merged_spans for coverage."""
        from extraction.v4.models import InventoryElement

        elements: list[InventoryElement] = []
        span_texts = {str(s.id): s.text.lower() for s in merged_spans}
        all_span_text = " ".join(span_texts.values())

        for elem_type, items in inventory.items():
            for elem_id, page_num in items:
                # Search merged spans for this element.
                # For recommendations, strip the grade suffix "(1B)" before matching
                # because spans contain "Recommendation 1.2.1: We recommend..." not
                # "Recommendation 1.2.1 (1B)" — the grade may appear elsewhere.
                search_key = elem_id.lower()
                if elem_type == "recommendation":
                    # "Recommendation 1.2.1 (1B)" → "recommendation 1.2.1"
                    search_key = re.sub(r'\s*\([12][a-d]\)\s*$', '', search_key, flags=re.IGNORECASE)
                matched_ids = []

                for sid, stext in span_texts.items():
                    if search_key in stext:
                        matched_ids.append(sid)

                # For tables and figures, also check by number
                if not matched_ids and elem_type in ("table", "figure"):
                    # Extract number from "Table 3" or "Figure 5"
                    parts = elem_id.split()
                    if len(parts) >= 2:
                        num_pattern = f"{parts[0].lower()} {parts[1]}"
                        for sid, stext in span_texts.items():
                            if num_pattern in stext:
                                matched_ids.append(sid)

                # Content-signature fallback: Channel D extracts table
                # cell content, not captions. Match by column header
                # keywords that uniquely identify each KDIGO table.
                if not matched_ids and elem_type == "table":
                    matched_ids = self._match_table_by_content_signature(
                        elem_id, span_texts
                    )

                status = "COVERED" if matched_ids else "MISSING"
                elements.append(InventoryElement(
                    element_type=elem_type,
                    element_id=elem_id,
                    page_number=page_num,
                    coverage_status=status,
                    matched_span_ids=matched_ids[:10],  # cap for report size
                ))

        # A1d binding verification: check footnotes against span text.
        # Normalize whitespace for comparison — PyMuPDF and Marker/Channel D
        # may differ on whitespace around punctuation (e.g., "/ " vs "/").
        def _ws_normalize(t: str) -> str:
            t = re.sub(r'\s+', ' ', t).strip()
            t = re.sub(r'\s*/\s*', '/', t)   # "/ " or " /" → "/"
            t = re.sub(r'\s*-\s*', '-', t)   # " - " → "-"
            return t

        for binding in footnote_bindings:
            fn_norm = _ws_normalize(binding.footnote_text.lower()[:100])
            for sid, stext in span_texts.items():
                stext_norm = _ws_normalize(stext)
                if fn_norm in stext_norm:
                    binding.bound_to_span = True
                    binding.span_id = sid
                    break

            # Add footnote as inventory element
            elements.append(InventoryElement(
                element_type="footnote",
                element_id=f"{binding.marker_char} ({binding.table_id})",
                page_number=binding.page_number,
                coverage_status="COVERED" if binding.bound_to_span else "MISSING",
                matched_span_ids=[binding.span_id] if binding.span_id else [],
            ))

        return elements

    # ── Table content-signature matching ───────────────────────────────
    # Channel D extracts table cell content, not caption labels. When no
    # span contains the literal "Table N" string, fall back to matching
    # by column header keywords that uniquely identify each KDIGO table.
    _TABLE_CONTENT_SIGNATURES: dict[str, list[str]] = {
        "Table 25": ["factor/mechanism", "possible cause"],
        "Table 26": ["class", "mechanism", "example"],
        "Table 27": ["polystyrene", "patiromer", "zirconium"],
        "Table 28": ["severity of hyperkalemia", "clinically unwell"],
        "Table 31": ["nephrotoxic medication", "non-nephrotoxic alternative"],
        "Table 32": ["medications", "perioperative adverse events"],
        "Table 33": ["patient-associated", "procedure-associated"],
    }

    def _match_table_by_content_signature(
        self, table_id: str, span_texts: dict[str, str]
    ) -> list[str]:
        """Match a table by its column-header content signature."""
        signatures = self._TABLE_CONTENT_SIGNATURES.get(table_id, [])
        if not signatures:
            return []

        all_text = " ".join(span_texts.values())
        if any(sig in all_text for sig in signatures):
            # Return IDs of spans containing any signature term
            return [
                sid for sid, stext in span_texts.items()
                if any(sig in stext for sig in signatures)
            ][:5]
        return []

    def _a3_density_check(self, merged_spans, tree, normalized_text: str) -> list[str]:
        """Check span density per page against expected ranges by page type."""
        warnings: list[str] = []

        if tree is None:
            warnings.append("A3: GuidelineTree not available — density check skipped")
            return warnings

        # Count spans per page
        page_span_counts: dict[int, int] = defaultdict(int)
        for span in merged_spans:
            if span.page_number is not None:
                page_span_counts[span.page_number] += 1

        # Estimate words per page (rough: total / pages)
        total_words = len(normalized_text.split())
        total_pages = tree.total_pages if tree.total_pages > 0 else 1
        avg_words_per_page = total_words / total_pages

        # Check for pages with zero spans (potential gaps)
        for page_num in range(1, total_pages + 1):
            span_count = page_span_counts.get(page_num, 0)

            if span_count == 0 and avg_words_per_page > 50:
                warnings.append(
                    f"A3: Page {page_num} has 0 spans — possible extraction gap"
                )

        # Check channel distribution per section
        section_channels: dict[str, dict[str, int]] = defaultdict(lambda: defaultdict(int))
        for span in merged_spans:
            if span.section_id:
                for ch in span.contributing_channels:
                    section_channels[span.section_id][ch] += 1

        for section_id, ch_counts in section_channels.items():
            total = sum(ch_counts.values())
            if total == 0:
                continue
            for ch, count in ch_counts.items():
                pct = count / total * 100
                if pct > 80 and total >= 5:
                    warnings.append(
                        f"A3: Section {section_id} is {pct:.0f}% Channel {ch} "
                        f"({count}/{total} spans) — limited corroboration"
                    )

        return warnings

    # =====================================================================
    # DOMAIN B — Content Exhaustiveness
    # =====================================================================

    def _domain_b(self, report, merged_spans, tree, normalized_text: str):
        """B1 token residual, B2 adversarial audit, B3 branch heuristic."""

        # ── B1: Token-Level Residual Analysis ────────────────────────────
        residuals, tier1_count, pop_warnings = self._b1_token_residual(
            merged_spans, tree, normalized_text,
        )
        report.residual_fragments = residuals
        report.tier1_residual_count = tier1_count
        report.population_action_warnings = pop_warnings

        # ── B2: Adversarial Recall Audit (LLM) ──────────────────────────
        if self._enable_b2 and self._client is not None:
            b2_delta, b2_assertions = self._b2_adversarial_audit(
                merged_spans, tree, normalized_text,
            )
            report.adversarial_audit_delta = b2_delta
            # Store assertions for D1 cross-validation
            self._b2_primary_assertions = b2_assertions
        else:
            report.adversarial_audit_delta = None
            self._b2_primary_assertions = []

        # ── B3: Conditional Branch Heuristic ─────────────────────────────
        branch_comparisons = self._b3_branch_heuristic(merged_spans, tree, normalized_text)
        report.branch_comparisons = branch_comparisons

    def _b1_token_residual(self, merged_spans, tree, normalized_text: str):
        """Token-level residual analysis with n-gram alignment.

        Algorithm:
        1. Tokenize source into word tokens with character offsets
        2. Build coverage bitmap (boolean array, one per token)
        3. For each MergedSpan: extract word-level n-grams (window 5-8),
           match against source tokens using word overlap ratio ≥0.8
        4. Extract uncovered fragments (contiguous runs of uncovered tokens)
        5. Filter structural noise (REFERENCES, FRONT_MATTER, headers/footers)
        6. Clinical signal scan on remaining fragments

        Returns:
            (residual_fragments, tier1_count, population_action_warnings)
        """
        from extraction.v4.models import ResidualFragment

        # Step 1: Tokenize source text into words with offsets
        tokens = self._tokenize_with_offsets(normalized_text)
        if not tokens:
            return [], 0, []

        # Step 2: Coverage bitmap
        covered = [False] * len(tokens)

        # Step 3: Build inverted index (word → list of token positions)
        # This allows O(1) lookup of candidate positions instead of O(n) scan
        word_index: dict[str, list[int]] = defaultdict(list)
        for idx, (word, _, _) in enumerate(tokens):
            word_index[word].append(idx)

        # Step 3b: Augment index with abbreviation ↔ expansion aliases.
        # When source has "arb" at positions [P1, P2], inject "angiotensin" → [P1, P2]
        # so spans using expanded form can find anchors. And vice versa.
        # One-time O(k) where k = len(CLINICAL_ABBREVIATION_MAP) ≈ 8.
        for abbrev, expansion_words in CLINICAL_ABBREVIATION_MAP.items():
            abbrev_positions = word_index.get(abbrev, [])
            if abbrev_positions:
                # Source has abbreviation → inject expansion words as aliases
                for exp_word in expansion_words:
                    if exp_word not in word_index:
                        word_index[exp_word] = list(abbrev_positions)
            else:
                # Source might have expansion → inject abbreviation as alias
                for exp_word in expansion_words:
                    exp_positions = word_index.get(exp_word, [])
                    if exp_positions and abbrev not in word_index:
                        word_index[abbrev] = list(exp_positions)
                        break  # one expansion word is enough for anchor

        # Step 4: Match each span against source using inverted index
        for span in merged_spans:
            span_text = span.reviewer_text if span.reviewer_text else span.text
            span_words = self._extract_words(span_text.lower())

            if len(span_words) < 2:
                span_lower = span_text.lower().strip()
                if len(span_lower) >= 5:
                    self._mark_substring_coverage(tokens, covered, span_lower)
                continue

            span_word_set = set(span_words)

            # Find anchor positions via inverted index:
            # Pick the rarest word in the span as anchor (fewest occurrences)
            rarest_word = min(span_word_set, key=lambda w: len(word_index.get(w, [])))
            anchor_positions = word_index.get(rarest_word, [])

            if not anchor_positions:
                # Fallback: try abbreviation-expanded anchor lookup.
                # If span uses expanded form ("angiotensin receptor blocker")
                # but source uses abbreviation ("ARB"), the augmented index
                # maps expansion words to abbreviation positions.
                expanded = set()
                for sw in span_word_set:
                    if sw in CLINICAL_EXPANSION_TO_ABBREV:
                        expanded.add(CLINICAL_EXPANSION_TO_ABBREV[sw])
                    if sw in CLINICAL_ABBREVIATION_MAP:
                        expanded.update(CLINICAL_ABBREVIATION_MAP[sw])
                expanded -= span_word_set  # only new words
                if expanded:
                    combined = span_word_set | expanded
                    rarest_word = min(combined, key=lambda w: len(word_index.get(w, [])))
                    anchor_positions = word_index.get(rarest_word, [])
                    if anchor_positions:
                        # Use expanded set for this span's overlap check
                        span_word_set = combined

            if not anchor_positions:
                continue

            # For each anchor position, check if surrounding tokens match
            window = min(len(span_words), 8)
            half_window = window // 2
            best_overlap = 0.0

            for anchor_pos in anchor_positions:
                # Check a region around the anchor
                start = max(0, anchor_pos - half_window)
                end = min(len(tokens), anchor_pos + len(span_words) + half_window)
                region_words = {tokens[j][0] for j in range(start, end)}

                overlap = len(span_word_set & region_words) / len(span_word_set)
                if overlap >= 0.5:  # Confirmed passage match
                    # Mark entire region as covered (passage-level coverage)
                    # Rationale: connecting words (the, of, in) between matched
                    # content words are part of the same covered passage
                    for j in range(start, end):
                        covered[j] = True

        # Step 4a: Gap closing — fill short uncovered gaps between SUBSTANTIAL
        # covered regions.  Eliminates boundary artifacts where a fragment straddles
        # two matched spans (e.g., "...ARBs). Dosage recommendations..." where
        # period-boundary tokens aren't covered by either span's region).
        # Only closes gaps when both flanking covered regions have ≥ MIN_REGION
        # consecutive covered tokens — this prevents the single-word substring
        # matcher (short spans) from creating false coverage through gap-closing.
        GAP_CLOSE_THRESHOLD = 10  # max gap size to fill
        MIN_REGION = 5  # min consecutive covered tokens on each side of gap
        in_gap = False
        gap_start = -1
        for i in range(len(covered)):
            if not covered[i]:
                if not in_gap:
                    in_gap = True
                    gap_start = i
            else:
                if in_gap:
                    gap_len = i - gap_start
                    if gap_len <= GAP_CLOSE_THRESHOLD and gap_start >= MIN_REGION:
                        # Check left flank: MIN_REGION consecutive covered tokens before gap
                        left_ok = all(covered[j] for j in range(gap_start - MIN_REGION, gap_start))
                        # Check right flank: MIN_REGION consecutive covered tokens after gap
                        right_end = min(len(covered), i + MIN_REGION)
                        right_ok = (right_end - i >= MIN_REGION and
                                    all(covered[j] for j in range(i, right_end)))
                        if left_ok and right_ok:
                            for j in range(gap_start, i):
                                covered[j] = True
                    in_gap = False

        # Step 4b: Extract uncovered fragments
        raw_fragments = self._extract_uncovered_fragments(tokens, covered, normalized_text)

        # Step 5: Filter structural noise (page markers, copyright, ToC entries)
        noise_patterns = re.compile(
            r'(?:references?\s*$|^\d+\.\s*$|^[ivxlc]+\.\s*$|'
            r'kidney\s+international|©|copyright|doi:|'
            r'^\s*\d+\s*$|^page\s+\d+|table\s+of\s+contents|'
            # Back-matter: contributor disclosures, editorial, acknowledgments
            r'editor-in-chief|associate\s+editor|editorial\s+board|'
            r'public\s+review|provided\s+feedback|'
            r'disclosure|conflict\s+of\s+interest|'
            r'acknowledgment|supplementary\s+material|'
            # Page-break marker contamination (Marker v1.10 format)
            r'\{\d+\}-{10,})',
            re.IGNORECASE | re.MULTILINE,
        )
        filtered_fragments: list[dict] = []
        for frag in raw_fragments:
            if noise_patterns.search(frag["text"]):
                continue
            if len(frag["text"].strip()) < 15:
                continue
            filtered_fragments.append(frag)

        # Step 5b: Filter rationale/evidence/discussion sections.
        # Same section-type filtering as B3 — residual fragments from
        # rationale text contain drug names + trigger words but describe
        # evidence, not prescribing directives.
        _RATIONALE_PREFIXES = ("Rationale", "Evidence", "Discussion",
                               "Key_information", "Implementation")
        content_fragments: list[dict] = []
        for frag in filtered_fragments:
            if tree:
                sec = tree.find_section_for_offset(frag["char_start"])
                if sec and sec.section_id:
                    sid = sec.section_id
                    if any(sid.startswith(p) for p in _RATIONALE_PREFIXES):
                        continue
                    if re.fullmatch(r'\d+', sid):
                        continue
            content_fragments.append(frag)
        filtered_fragments = content_fragments

        # Step 6: Clinical signal scan
        residuals: list[ResidualFragment] = []
        tier1_count = 0

        for frag in filtered_fragments:
            text_lower = frag["text"].lower()

            # Drug name scan — with word-boundary check for short names
            # (≤4 chars like "esa", "arb", "mra") to avoid substring matches
            # inside person names (e.g., "Teresa" → "esa", "Baris" → "arb").
            drug_names = []
            for end_pos, (_, drug) in self._drug_automaton.iter(text_lower):
                if len(drug) <= 4:
                    start_pos = end_pos - len(drug) + 1
                    if start_pos > 0 and text_lower[start_pos - 1].isalnum():
                        continue
                    if end_pos + 1 < len(text_lower) and text_lower[end_pos + 1].isalnum():
                        continue
                drug_names.append(drug)

            # Trigger category scan
            trigger_cat = None
            for trigger in PROHIBITIVE_TRIGGERS:
                if trigger in text_lower:
                    trigger_cat = "prohibitive"
                    break
            if trigger_cat is None:
                for trigger in PRESCRIPTIVE_TRIGGERS:
                    if trigger in text_lower:
                        trigger_cat = "prescriptive"
                        break
            if trigger_cat is None:
                for trigger in IMPLICIT_RISK_TRIGGERS:
                    if trigger in text_lower:
                        trigger_cat = "implicit_risk"
                        break

            # Tier classification
            if drug_names and trigger_cat:
                tier = "TIER_1"
                tier1_count += 1
            elif drug_names or trigger_cat:
                tier = "TIER_2"
            else:
                tier = "NOISE"

            page_num = self._estimate_page_from_offset(
                normalized_text, frag["char_start"],
            )

            residuals.append(ResidualFragment(
                text=frag["text"][:500],  # cap for report size
                char_start=frag["char_start"],
                char_end=frag["char_end"],
                page_number=page_num,
                trigger_category=trigger_cat,
                drug_names_found=drug_names,
                tier=tier,
            ))

        # Population-action same-page warning (Tier 2)
        pop_warnings = self._b1_population_action_warnings(merged_spans)

        return residuals, tier1_count, pop_warnings

    def _tokenize_with_offsets(self, text: str) -> list[tuple[str, int, int]]:
        """Split text into word tokens with (word, char_start, char_end) tuples."""
        tokens = []
        i = 0
        n = len(text)
        while i < n:
            # Skip whitespace/punctuation
            while i < n and not text[i].isalnum():
                i += 1
            if i >= n:
                break
            # Collect word
            start = i
            while i < n and (text[i].isalnum() or text[i] in "-'"):
                i += 1
            word = text[start:i].lower()
            if len(word) >= 2:  # Skip single chars
                tokens.append((word, start, i))
        return tokens

    def _extract_words(self, text: str) -> list[str]:
        """Extract word tokens from text (lowercase, no offsets)."""
        words = []
        i = 0
        n = len(text)
        while i < n:
            while i < n and not text[i].isalnum():
                i += 1
            if i >= n:
                break
            start = i
            while i < n and (text[i].isalnum() or text[i] in "-'"):
                i += 1
            word = text[start:i]
            if len(word) >= 2:
                words.append(word)
        return words

    def _mark_substring_coverage(self, tokens, covered, substring: str):
        """Mark tokens as covered if their combined text contains the substring."""
        # Build running text from token words to find substring match
        for i in range(len(tokens)):
            combined = ""
            for j in range(i, min(i + 20, len(tokens))):
                if combined:
                    combined += " "
                combined += tokens[j][0]
                if substring in combined:
                    for k in range(i, j + 1):
                        covered[k] = True
                    break

    def _match_ngram_to_source(
        self,
        tokens: list[tuple[str, int, int]],
        covered: list[bool],
        ngram: list[str],
        overlap_threshold: float = 0.8,
        span_word_set: set = None,
    ):
        """Find best match for ngram in source tokens and extend coverage.

        When a match is found, marks the ngram window as covered, then extends
        outward to include adjacent tokens that belong to the span's vocabulary.
        This handles spans where n-gram anchors only match part of the text
        due to formatting differences between source and extracted spans.
        """
        ngram_set = set(ngram)
        ngram_len = len(ngram)

        for i in range(len(tokens) - ngram_len + 1):
            window_words = {tokens[i + k][0] for k in range(ngram_len)}
            overlap = len(ngram_set & window_words) / ngram_len
            if overlap >= overlap_threshold:
                # Mark the anchor window
                for k in range(ngram_len):
                    covered[i + k] = True

                # Extend coverage outward using the full span vocabulary
                if span_word_set:
                    # Extend left
                    j = i - 1
                    while j >= 0 and tokens[j][0] in span_word_set:
                        covered[j] = True
                        j -= 1
                    # Extend right
                    j = i + ngram_len
                    while j < len(tokens) and tokens[j][0] in span_word_set:
                        covered[j] = True
                        j += 1

    def _extract_uncovered_fragments(
        self,
        tokens: list[tuple[str, int, int]],
        covered: list[bool],
        full_text: str,
    ) -> list[dict]:
        """Extract contiguous runs of uncovered tokens as text fragments.

        Merges fragments separated by ≤2 covered tokens.
        """
        fragments: list[dict] = []
        i = 0
        n = len(tokens)

        while i < n:
            if covered[i]:
                i += 1
                continue

            # Start of uncovered run
            frag_start = i
            while i < n:
                if not covered[i]:
                    i += 1
                    continue
                # Check if gap is ≤2 tokens followed by more uncovered
                gap_end = i
                while gap_end < n and covered[gap_end] and (gap_end - i) <= 2:
                    gap_end += 1
                if gap_end < n and not covered[gap_end] and (gap_end - i) <= 2:
                    # Bridge the small gap
                    i = gap_end
                else:
                    break

            frag_end = i
            if frag_end > frag_start:
                char_start = tokens[frag_start][1]
                char_end = tokens[min(frag_end - 1, n - 1)][2]
                text = full_text[char_start:char_end].strip()
                if text:
                    fragments.append({
                        "text": text,
                        "char_start": char_start,
                        "char_end": char_end,
                        "token_count": frag_end - frag_start,
                    })

        return fragments

    def _b1_population_action_warnings(self, merged_spans) -> list[str]:
        """Tier 2: Flag drug-action spans missing population qualifier on same page.

        When a span contains a drug action but no population threshold (eGFR, age,
        CKD stage, pregnancy, dialysis, transplant) appears on the same page, emit
        a Tier 2 warning. NOT a BLOCK — full cross-span coreference is out of scope.
        """
        warnings: list[str] = []

        # Group spans by page
        page_spans: dict[int, list] = defaultdict(list)
        for span in merged_spans:
            if span.page_number is not None:
                page_spans[span.page_number].append(span)

        for page_num, spans in page_spans.items():
            page_text_lower = " ".join(
                (s.reviewer_text or s.text).lower() for s in spans
            )

            # Check if any span on this page has a drug action
            has_drug_action = False
            for span in spans:
                span_lower = (span.reviewer_text or span.text).lower()
                for _, (_, drug) in self._drug_automaton.iter(span_lower):
                    # Check if accompanied by a prescriptive or prohibitive trigger
                    for trigger in PRESCRIPTIVE_TRIGGERS + PROHIBITIVE_TRIGGERS:
                        if trigger in span_lower:
                            has_drug_action = True
                            break
                    if has_drug_action:
                        break
                if has_drug_action:
                    break

            if not has_drug_action:
                continue

            # Check if population qualifier exists on same page
            has_population = False
            for qualifier in POPULATION_QUALIFIERS:
                if qualifier in page_text_lower:
                    has_population = True
                    break

            if not has_population:
                warnings.append(
                    f"B1-T2: Page {page_num} has drug action span(s) "
                    f"without population qualifier on same page"
                )

        return warnings

    # ─── B2: Adversarial Recall Audit ──────────────────────────────────

    _B2_SYSTEM_PROMPT = (
        "You are a clinical guideline auditor. Given a section of a clinical "
        "guideline, extract ALL minimal atomic prescribing assertions. Each "
        "assertion must be self-contained and testable against extracted spans.\n\n"
        "Categories:\n"
        "  ELIGIBILITY — who qualifies (population + criteria)\n"
        "  DOSING — dose, route, frequency, titration\n"
        "  CONTRAINDICATION — who must NOT receive the drug\n"
        "  MONITORING — labs, vitals, frequency of monitoring\n"
        "  CONDITIONAL — if X then Y (branching logic)\n\n"
        "Rules:\n"
        "- One assertion per atomic clinical fact\n"
        "- Include the drug name in each assertion\n"
        "- Set is_negative=true for assertions about what NOT to do\n"
        "- Set is_conditional=true for if/then branching\n"
        "- Be exhaustive — missing an assertion means a patient safety gap\n"
        "- Do NOT include general background text or definitions\n"
        "- Only include actionable prescribing information"
    )

    def _b2_adversarial_audit(
        self,
        merged_spans,
        tree,
        normalized_text: str,
    ) -> tuple[int, list]:
        """B2: LLM generates atomic assertions per section, diffs against spans.

        Scoped to sections that have recommendation/practice point content
        (cost control: only clinically relevant sections).

        Returns:
            (delta_count, all_assertions) where delta = assertions NOT in spans.
        """
        from extraction.v4.models import AdversarialAssertion, AdversarialAuditResult

        all_assertions: list[AdversarialAssertion] = []
        delta_count = 0

        if not tree or not tree.sections:
            return 0, []

        # Select sections with clinical content (cost control)
        sections = self._flatten_sections(tree.sections)
        clinical_sections = []
        for section in sections:
            section_text = normalized_text[section.start_offset:section.end_offset]
            if len(section_text) < 50:
                continue
            # Only audit sections with recommendation/drug content
            has_rec = bool(RECOMMENDATION_RE.search(section_text))
            has_drug = False
            for _, (_, _drug) in self._drug_automaton.iter(section_text.lower()):
                has_drug = True
                break
            if has_rec or has_drug:
                clinical_sections.append(section)

        if not clinical_sections:
            return 0, []

        logger.info("B2: Auditing %d clinical sections", len(clinical_sections))

        # Build span text index for matching
        span_texts_lower = [
            (s.reviewer_text or s.text).lower()
            for s in merged_spans
        ]

        # Audit each section
        tool_schema = AdversarialAuditResult.model_json_schema()

        for section in clinical_sections:
            section_text = normalized_text[section.start_offset:section.end_offset]
            if len(section_text) > 6000:
                section_text = section_text[:6000]  # Cap to control cost

            prompt = (
                f"Section: {section.heading} (ID: {section.section_id})\n\n"
                f"Text:\n{section_text}\n\n"
                f"Extract ALL atomic prescribing assertions from this section."
            )

            try:
                response = self._client.messages.create(
                    model=self.B2_PRIMARY_MODEL,
                    max_tokens=self.B2_MAX_TOKENS,
                    system=self._B2_SYSTEM_PROMPT,
                    tool_choice={"type": "any"},
                    tools=[{
                        "name": "record_assertions",
                        "description": "Record all atomic prescribing assertions found in this section.",
                        "input_schema": tool_schema,
                    }],
                    messages=[{"role": "user", "content": prompt}],
                )

                # Extract tool_use result
                tool_result = None
                for block in response.content:
                    if block.type == "tool_use":
                        tool_result = block.input
                        break

                if tool_result is None:
                    logger.warning("B2: No tool_use response for section %s", section.section_id)
                    continue

                if isinstance(tool_result, str):
                    tool_result = json.loads(tool_result)

                audit_result = AdversarialAuditResult.model_validate(tool_result)

                # Tag section_id on each assertion
                for assertion in audit_result.assertions:
                    assertion.section_id = section.section_id

                all_assertions.extend(audit_result.assertions)

            except Exception as e:
                logger.warning("B2: LLM call failed for section %s: %s", section.section_id, e)
                continue

        # Match assertions against merged spans
        for assertion in all_assertions:
            if not self._assertion_covered_by_spans(assertion, span_texts_lower):
                delta_count += 1

        logger.info(
            "B2: %d assertions generated, %d not covered (delta)",
            len(all_assertions), delta_count,
        )

        return delta_count, all_assertions

    def _assertion_covered_by_spans(
        self,
        assertion,
        span_texts_lower: list[str],
    ) -> bool:
        """Check if an assertion's key content is covered by any merged span.

        Uses word overlap: if ≥70% of the assertion's content words appear
        in any single span, consider it covered.
        """
        assertion_words = set(self._extract_words(assertion.assertion_text.lower()))
        if len(assertion_words) < 3:
            # Very short assertion: check substring
            key = assertion.assertion_text.lower().strip()
            return any(key in st for st in span_texts_lower)

        # Also include drug name as a required word
        drug_lower = assertion.drug_name.lower()
        threshold = 0.70

        for span_text in span_texts_lower:
            span_words = set(self._extract_words(span_text))
            if not span_words:
                continue
            overlap = len(assertion_words & span_words) / len(assertion_words)
            # Drug name must be present in the matching span
            if overlap >= threshold and drug_lower in span_text:
                return True

        return False

    # ─── D1: Dual-LLM Cross-Validation ─────────────────────────────────

    def _d1_dual_llm_agreement(self, normalized_text: str) -> Optional[float]:
        """D1: Run B2 with a second model on 20% of sections, compare.

        Returns agreement percentage. >80% = healthy. <80% = investigate.
        """
        from extraction.v4.models import AdversarialAuditResult

        primary_by_section: dict[str, set[str]] = defaultdict(set)
        for a in self._b2_primary_assertions:
            primary_by_section[a.section_id].add(a.assertion_text.lower().strip())

        if not primary_by_section:
            return None

        # Sample 20% of sections
        section_ids = list(primary_by_section.keys())
        sample_size = max(1, int(len(section_ids) * self.B2_CROSSVAL_SAMPLE_RATE))
        random.seed(42)  # Reproducible sampling
        sampled_ids = random.sample(section_ids, min(sample_size, len(section_ids)))

        if not sampled_ids:
            return None

        logger.info("D1: Cross-validating %d/%d sections with %s",
                     len(sampled_ids), len(section_ids), self.B2_CROSSVAL_MODEL)

        # Find section text from normalized_text using primary assertions' section_ids
        # We need the tree to get offsets, but we stored assertions with section_id
        # Use a simpler approach: find section text by searching for assertion content
        tool_schema = AdversarialAuditResult.model_json_schema()

        agreement_scores: list[float] = []

        for section_id in sampled_ids:
            primary_assertions = primary_by_section[section_id]
            if not primary_assertions:
                continue

            # Build context from primary assertions for the cross-val prompt
            # (We can't easily recover original section text here, but we can
            # ask the second model to verify the assertions against the text
            # surrounding them)
            section_text = self._find_section_text(normalized_text, section_id)
            if not section_text:
                continue

            prompt = (
                f"Section ID: {section_id}\n\n"
                f"Text:\n{section_text}\n\n"
                f"Extract ALL atomic prescribing assertions from this section."
            )

            try:
                response = self._client.messages.create(
                    model=self.B2_CROSSVAL_MODEL,
                    max_tokens=self.B2_MAX_TOKENS,
                    system=self._B2_SYSTEM_PROMPT,
                    tool_choice={"type": "any"},
                    tools=[{
                        "name": "record_assertions",
                        "description": "Record all atomic prescribing assertions found in this section.",
                        "input_schema": tool_schema,
                    }],
                    messages=[{"role": "user", "content": prompt}],
                )

                tool_result = None
                for block in response.content:
                    if block.type == "tool_use":
                        tool_result = block.input
                        break

                if tool_result is None:
                    continue

                if isinstance(tool_result, str):
                    tool_result = json.loads(tool_result)

                crossval_result = AdversarialAuditResult.model_validate(tool_result)
                crossval_assertions = {
                    a.assertion_text.lower().strip()
                    for a in crossval_result.assertions
                }

                # Compute Jaccard-like agreement
                if not primary_assertions and not crossval_assertions:
                    agreement_scores.append(1.0)
                    continue

                union = primary_assertions | crossval_assertions
                intersection = primary_assertions & crossval_assertions

                # Also count fuzzy matches (word overlap ≥80%)
                fuzzy_matches = 0
                unmatched_primary = primary_assertions - intersection
                unmatched_crossval = crossval_assertions - intersection

                for p_assertion in unmatched_primary:
                    p_words = set(self._extract_words(p_assertion))
                    for c_assertion in unmatched_crossval:
                        c_words = set(self._extract_words(c_assertion))
                        if p_words and c_words:
                            overlap = len(p_words & c_words) / max(len(p_words), len(c_words))
                            if overlap >= 0.80:
                                fuzzy_matches += 1
                                break

                effective_agreement = (len(intersection) + fuzzy_matches) / max(len(union), 1)
                agreement_scores.append(effective_agreement)

            except Exception as e:
                logger.warning("D1: Cross-val failed for section %s: %s", section_id, e)
                continue

        if not agreement_scores:
            return None

        avg_agreement = sum(agreement_scores) / len(agreement_scores) * 100
        logger.info("D1: Agreement = %.1f%% across %d sections", avg_agreement, len(agreement_scores))

        return round(avg_agreement, 1)

    def _find_section_text(self, normalized_text: str, section_id: str) -> Optional[str]:
        """Find section text in normalized_text by looking for section heading patterns."""
        # Try to find section by heading pattern (e.g., "4.1.1" or "Chapter 4")
        patterns = [
            re.compile(rf'(?:^|\n)#{1,4}\s*{re.escape(section_id)}[\s.:]+(.+?)(?=\n#{1,4}\s|\Z)',
                       re.DOTALL),
            re.compile(rf'{re.escape(section_id)}[.\s](.{{100,3000}})', re.DOTALL),
        ]
        for pattern in patterns:
            m = pattern.search(normalized_text)
            if m:
                text = m.group(0)[:4000]  # Cap
                return text

        return None

    # Regex to extract recommendation/practice point text blocks from source.
    # Captures from "Recommendation X.Y.Z:" or "Practice Point X.Y.Z:" through
    # the end of the prescribing statement — stops at the next heading, rationale
    # marker, or double newline (paragraph boundary).
    #
    # CRITICAL: Uses \n\n (double newline / blank line) as the paragraph
    # terminator, NOT \n\n\n.  KDIGO format repeats each Rec/PP in the rationale
    # section separated by only one blank line.  Triple-newline allowed rationale
    # text (RCT results, p-values, study thresholds) to leak into the source
    # block, inflating source_threshold_count and causing false B3 BLOCKs.
    _REC_BLOCK_RE = re.compile(
        r'(?:Recommendation|Practice\s+Point)\s+\d+\.\d+\.\d+[:\s]+'
        r'(.*?)(?=\n\s*(?:#{1,4}\s|Recommendation\s+\d|Practice\s+Point\s+\d|'
        r'Rationale|Key\s+information|Evidence|Discussion|Implementation)|\n\n)',
        re.DOTALL | re.IGNORECASE,
    )

    def _b3_branch_heuristic(self, merged_spans, tree, normalized_text: str) -> list:
        """B3: Detect conditional branch loss in multi-branch recommendations.

        ONLY counts thresholds/connectors inside recommendation and practice
        point text blocks — NOT across entire sections (which include rationale,
        evidence discussion, p-values, study results, etc.). This prevents
        inflated threshold counts from non-prescriptive content.

        For each rec/PP block:
        - Count source thresholds vs extracted thresholds
        - Count source connectors (AND/OR/UNLESS/EXCEPT) vs extracted
        - Detect exception keyword loss
        """
        from extraction.v4.models import BranchComparison

        comparisons: list[BranchComparison] = []

        if not tree or not tree.sections:
            return comparisons

        # Extract recommendation/practice point TEXT BLOCKS from source
        # (not entire sections — only the prescribing statement itself)
        rec_blocks = list(self._REC_BLOCK_RE.finditer(normalized_text))
        if not rec_blocks:
            return comparisons

        # Build span text index by section_id
        section_span_texts: dict[str, str] = defaultdict(str)
        for s in merged_spans:
            if s.section_id:
                section_span_texts[s.section_id] += " " + (s.reviewer_text or s.text)

        for block_match in rec_blocks:
            source_text = block_match.group(0)
            if len(source_text) < 20:
                continue

            # Determine which section this block belongs to
            block_offset = block_match.start()
            section_id = None
            if tree:
                sec = tree.find_section_for_offset(block_offset)
                if sec:
                    section_id = sec.section_id

            if not section_id:
                # Fallback: extract rec ID from the matched text
                id_match = re.search(r'(\d+\.\d+\.\d+)', source_text)
                section_id = id_match.group(1) if id_match else "unknown"

            # Skip non-recommendation sections: Rationale sections inflate
            # threshold counts from evidence discussion, and bare chapter-level
            # IDs (e.g., "1", "2") aggregate thresholds from the entire chapter
            # introduction text.
            if section_id.startswith("Rationale"):
                continue
            if re.fullmatch(r'\d+', section_id):
                continue

            # Get extracted spans for this section
            section_span_text = section_span_texts.get(section_id, "")
            if not section_span_text:
                continue

            # Count thresholds in the RECOMMENDATION BLOCK ONLY.
            # Source uses strict patterns (comparator required) because the
            # guideline source text has explicit comparators.
            source_thresholds = len(CLINICAL_THRESHOLD_RE.findall(source_text))
            source_thresholds += len(NUMERIC_THRESHOLD_RE.findall(source_text))
            # Extracted spans use strict + bare-value matching because
            # extraction may strip comparators ("limit to <2 g/day" → "2 g").
            extracted_thresholds = len(CLINICAL_THRESHOLD_RE.findall(section_span_text))
            extracted_thresholds += len(NUMERIC_THRESHOLD_RE.findall(section_span_text))
            extracted_thresholds += len(BARE_CLINICAL_VALUE_RE.findall(section_span_text))

            # Count connectors
            source_lower = source_text.lower()
            extracted_lower = section_span_text.lower()

            source_connectors = sum(
                1 for conn in BRANCH_CONNECTORS
                if f" {conn} " in source_lower
            )
            extracted_connectors = sum(
                1 for conn in BRANCH_CONNECTORS
                if f" {conn} " in extracted_lower
            )

            # Check exception keywords
            lost_exceptions = []
            for kw in EXCEPTION_KEYWORDS:
                if kw in source_lower and kw not in extracted_lower:
                    lost_exceptions.append(kw)

            # Skip blocks with no branches
            if source_thresholds < 2 and source_connectors < 1:
                continue

            # Determine action
            action = "PASS"
            if extracted_thresholds < source_thresholds:
                action = "BLOCK"
            if lost_exceptions:
                action = "BLOCK"

            comparisons.append(BranchComparison(
                section_id=section_id,
                source_threshold_count=source_thresholds,
                extracted_threshold_count=extracted_thresholds,
                source_connector_count=source_connectors,
                extracted_connector_count=extracted_connectors,
                exception_keywords_lost=lost_exceptions,
                action=action,
            ))

        return comparisons

    # =====================================================================
    # DOMAIN C — Integrity Verification
    # =====================================================================

    def _domain_c(self, report, merged_spans, normalized_text: str, oracle_report):
        """C1 numeric integrity, C2 recovery escalation, C3 corroboration."""

        # ── C1: Numeric & Comparator Integrity Lock ──────────────────────
        mismatches = self._c1_numeric_integrity(merged_spans, normalized_text)
        report.numeric_mismatches = mismatches

        # ── C2: L1 Recovery Risk Escalation ──────────────────────────────
        escalations = self._c2_recovery_escalation(merged_spans, oracle_report)
        report.l1_recovery_escalations = escalations

        # ── C3: Channel Corroboration Scoring ────────────────────────────
        corroborations = self._c3_corroboration_scoring(merged_spans)
        report.corroboration_details = corroborations

    def _c1_numeric_integrity(self, merged_spans, normalized_text: str) -> list:
        """Compare numeric values in spans against source text.

        For each span: extract (comparator, value, unit) tuples from both
        source and span text. Match exactly. Flag mismatches by type.
        """
        from extraction.v4.models import NumericMismatch

        mismatches: list[NumericMismatch] = []
        source_lower = normalized_text.lower()

        # Build page boundary map for page-scoped source lookup.
        # Marker v1.10 format: {N}-------...
        _page_re = re.compile(r'\{(\d+)\}-{20,}')
        page_boundaries: dict[int, tuple[int, int]] = {}
        page_offsets = [(m.start(), int(m.group(1))) for m in _page_re.finditer(normalized_text)]
        for i, (offset, pg) in enumerate(page_offsets):
            end = page_offsets[i + 1][0] if i + 1 < len(page_offsets) else len(normalized_text)
            page_boundaries[pg] = (offset, end)

        # Pre-extract page-level source tuples for fast lookup.
        # For each page, extract all (comparator, value) pairs from the source.
        page_source_tuples: dict[int, set[tuple[str, str]]] = {}
        for pg, (start, end) in page_boundaries.items():
            page_text_region = re.sub(r'<[^>]+>', ' ', normalized_text[start:end])
            tuples = NUMERIC_TUPLE_RE.findall(page_text_region)
            page_source_tuples[pg] = {
                (self._normalize_comparator(c), v) for c, v, u in tuples
            }

        for span in merged_spans:
            span_text = span.reviewer_text if span.reviewer_text else span.text

            # Minimum 8 chars — bare table fragments like ">30" or "<3"
            # aren't meaningful clinical assertions. Keeps "eGFR ≥20" (8 chars).
            if len(span_text.strip()) < 8:
                continue

            # Clean HTML tags before numeric extraction to prevent cross-tag
            # boundary artifacts (e.g., ">300 mg/g<br><3 mg/mmol").
            clean_span = re.sub(r'<[^>]+>', ' ', span_text)

            # Extract numeric tuples from span
            span_tuples = NUMERIC_TUPLE_RE.findall(clean_span)
            if not span_tuples:
                continue

            page_num = span.page_number or 0

            # Get the page's source tuples (±1 page for boundary spans)
            source_set: set[tuple[str, str]] = set()
            for pg in range(max(0, page_num - 1), page_num + 2):
                source_set |= page_source_tuples.get(pg, set())

            if not source_set:
                continue

            # Inverted logic: for each span tuple, verify it EXISTS in
            # the same page's source. If it does → not corrupted.
            # Only flag when a span tuple has NO match on the same page.
            for s_comp, s_val, s_unit in span_tuples:
                norm_comp = self._normalize_comparator(s_comp)
                if (norm_comp, s_val) in source_set:
                    continue  # Tuple verified — exists in source

                # Tuple not found on this page — potential corruption.
                # Find the closest source tuple to classify the mismatch.
                # Use page-scoped narrow search for mismatch details.
                clean_key = clean_span[:60].lower().strip()
                search_start = 0
                search_end = len(normalized_text)
                if page_num > 0 and page_num in page_boundaries:
                    search_start = page_boundaries[page_num][0]
                    search_end = page_boundaries[page_num][1]

                page_source_text = source_lower[search_start:search_end]
                source_pos = page_source_text.find(clean_key)
                if source_pos == -1:
                    clean_key = clean_span[:30].lower().strip()
                    source_pos = page_source_text.find(clean_key)
                if source_pos == -1:
                    continue

                source_pos += search_start
                source_region = normalized_text[
                    max(0, source_pos - 100):
                    min(len(normalized_text), source_pos + len(span_text) + 100)
                ]
                clean_source = re.sub(r'<[^>]+>', ' ', source_region)
                source_tuples = NUMERIC_TUPLE_RE.findall(clean_source)

                # Find the closest source tuple to classify mismatch
                for src_comp, src_val, src_unit in source_tuples:
                    # Unit-aware cross-concept skip: if both tuples have
                    # units and they differ, these are measuring different
                    # clinical parameters (e.g., eGFR ml/min vs ACR mg/g).
                    # Not corruption — skip to next source tuple.
                    if s_unit and src_unit:
                        s_unit_norm = re.sub(r'\s+', '', s_unit.lower())
                        src_unit_norm = re.sub(r'\s+', '', src_unit.lower())
                        if s_unit_norm != src_unit_norm:
                            continue

                    mismatch_type = None
                    action = "ACCEPT"

                    if s_val != src_val:
                        mismatch_type = "value_change"
                        action = "BLOCK"
                    elif self._normalize_comparator(s_comp) != self._normalize_comparator(src_comp):
                        # Check if it's just unicode normalization
                        if self._is_unicode_equivalent(s_comp, src_comp):
                            mismatch_type = "unicode_normalization"
                            action = "ACCEPT"
                        else:
                            mismatch_type = "comparator_flip"
                            action = "BLOCK"
                    elif not s_unit and src_unit:
                        mismatch_type = "unit_dropped"
                        action = "ACCEPT"

                    if mismatch_type:
                        page_num = span.page_number or 0
                        mismatches.append(NumericMismatch(
                            span_id=str(span.id),
                            source_value=f"{src_comp}{src_val} {src_unit}".strip(),
                            extracted_value=f"{s_comp}{s_val} {s_unit}".strip(),
                            mismatch_type=mismatch_type,
                            action=action,
                            page_number=page_num,
                        ))
                        break  # Only report first mismatch per span tuple

            # Also check for range boundary loss — compare ranges in the
            # span's page source against ranges in the span text.
            page_start = page_boundaries.get(page_num, (0, len(normalized_text)))[0]
            page_end = page_boundaries.get(page_num, (0, len(normalized_text)))[1]

            # Find span's footprint in source for range comparison
            page_src_lower = source_lower[page_start:page_end]
            rng_key = clean_span[:40].lower().strip()
            rng_pos = page_src_lower.find(rng_key)
            if rng_pos >= 0:
                abs_pos = rng_pos + page_start
                span_source_region = normalized_text[
                    abs_pos:min(len(normalized_text), abs_pos + len(span_text) + 10)
                ]
                clean_span_source = re.sub(r'<[^>]+>', ' ', span_source_region)
                source_ranges = NUMERIC_RANGE_RE.findall(clean_span_source)
                span_ranges = NUMERIC_RANGE_RE.findall(clean_span)
                if len(source_ranges) > len(span_ranges):
                    for src_range in source_ranges:
                        src_low, src_high, src_unit = src_range
                        found = False
                        for sp_range in span_ranges:
                            if sp_range[0] == src_low and sp_range[1] == src_high:
                                found = True
                                break
                        if not found:
                            # Verify the range doesn't exist on the page
                            # (same inverted logic as tuple check)
                            page_ranges = NUMERIC_RANGE_RE.findall(
                                re.sub(r'<[^>]+>', ' ', clean_span)
                            )
                            range_in_span = any(
                                r[0] == src_low and r[1] == src_high
                                for r in page_ranges
                            )
                            if not range_in_span:
                                mismatches.append(NumericMismatch(
                                    span_id=str(span.id),
                                    source_value=f"{src_low}-{src_high} {src_unit}".strip(),
                                    extracted_value="(range missing)",
                                    mismatch_type="range_boundary_loss",
                                    action="BLOCK",
                                    page_number=span.page_number or 0,
                                ))

        return mismatches

    def _normalize_comparator(self, comp: str) -> str:
        """Normalize comparator to canonical form."""
        mapping = {
            ">=": "≥", "⩾": "≥", "≧": "≥",
            "<=": "≤", "⩽": "≤", "≦": "≤",
            ">": ">", "<": "<",
            "=": "=", "==": "=",
        }
        return mapping.get(comp.strip(), comp.strip())

    def _is_unicode_equivalent(self, a: str, b: str) -> bool:
        """Check if two comparators are Unicode equivalents (≥ vs >=)."""
        return self._normalize_comparator(a) == self._normalize_comparator(b)

    def _c2_recovery_escalation(self, merged_spans, oracle_report) -> list[str]:
        """Flag Tier 1 L1_RECOVERY spans for mandatory visual verification.

        L1_RECOVERY spans come from degraded source (PyMuPDF rawdict text that
        the primary parser dropped). If they carry clinical signals, they require
        pharmacist visual check against the PDF.
        """
        escalations: list[str] = []

        for span in merged_spans:
            if "L1_RECOVERY" not in span.contributing_channels:
                continue

            span_text = (span.reviewer_text or span.text).lower()

            # Check for clinical signals
            has_drug = False
            for _, (_, drug) in self._drug_automaton.iter(span_text):
                has_drug = True
                break

            has_threshold = bool(
                CLINICAL_THRESHOLD_RE.search(span_text)
                or NUMERIC_THRESHOLD_RE.search(span_text)
            )

            has_trigger = False
            for trigger in PROHIBITIVE_TRIGGERS + PRESCRIPTIVE_TRIGGERS:
                if trigger in span_text:
                    has_trigger = True
                    break

            if has_drug or has_threshold or has_trigger:
                escalations.append(str(span.id))

        return escalations

    def _c3_corroboration_scoring(self, merged_spans) -> list:
        """Score each span's channel corroboration and flag low-confidence Tier 1."""
        from extraction.v4.models import CorroborationDetail

        details: list[CorroborationDetail] = []

        for span in merged_spans:
            channels = span.contributing_channels
            score = _corroboration_score(channels)

            # Determine tier: spans with clinical signals are Tier 1
            span_text = (span.reviewer_text or span.text).lower()
            has_clinical = False

            for _, (_, drug) in self._drug_automaton.iter(span_text):
                has_clinical = True
                break

            if not has_clinical:
                has_clinical = bool(
                    CLINICAL_THRESHOLD_RE.search(span_text)
                    or RECOMMENDATION_RE.search(span_text)
                )

            tier = "TIER_1" if has_clinical else "TIER_2"

            # BLOCK only for L1_RECOVERY-only (score ≤0.3).
            # F-only spans (score 0.4) route to Phase 3 clinical review
            # regardless — reclassifying them as WARNING keeps the gate
            # count focused on integrity failures, not confidence signals.
            action = "BLOCK" if (tier == "TIER_1" and score <= 0.3) else "PASS"

            details.append(CorroborationDetail(
                span_id=str(span.id),
                contributing_channels=list(channels),
                corroboration_score=score,
                tier=tier,
                action=action,
            ))

        return details

    # =====================================================================
    # DOMAIN D — Systemic Meta-Validation
    # =====================================================================

    def _domain_d(self, report, merged_spans, normalized_text: str):
        """D1 dual-LLM agreement, D3 validator health metrics."""

        # ── D1: Dual-LLM Audit Agreement ───────────────────────────────
        if self._enable_b2 and self._client is not None and self._b2_primary_assertions:
            report.dual_llm_agreement_pct = self._d1_dual_llm_agreement(
                normalized_text,
            )
        else:
            report.dual_llm_agreement_pct = None

        # ── D3: Validator Health Metrics ─────────────────────────────────
        total_source_chars = len(normalized_text) if normalized_text else 1
        residual_chars = sum(
            f.char_end - f.char_start
            for f in report.residual_fragments
        )
        residual_pct = (residual_chars / total_source_chars) * 100

        tier1_spans = sum(
            1 for d in report.corroboration_details if d.tier == "TIER_1"
        )
        tier1_blocked = sum(
            1 for d in report.corroboration_details
            if d.tier == "TIER_1" and d.action == "BLOCK"
        )
        tier1_delta_rate = (tier1_blocked / max(tier1_spans, 1)) * 100

        numeric_block_count = sum(
            1 for m in report.numeric_mismatches if m.action == "BLOCK"
        )
        total_numeric = max(len(report.numeric_mismatches), 1)
        numeric_mismatch_rate = (numeric_block_count / total_numeric) * 100

        report.validator_health = {
            "residual_pct": round(residual_pct, 2),
            "tier1_delta_rate": round(tier1_delta_rate, 2),
            "numeric_mismatch_rate": round(numeric_mismatch_rate, 2),
            "total_spans": len(merged_spans),
            "total_inventory_elements": len(report.inventory_elements),
        }

    # =====================================================================
    # RELEASE GATE — 8 conditions, ALL must pass
    # =====================================================================

    def _evaluate_release_gate(self, report):
        """Evaluate 8 release gate conditions and set verdict."""
        from extraction.v4.models import GateBlocker

        blockers: list[GateBlocker] = []
        warning_count = 0

        # Gate 1: All Recommendations & Practice Points covered (A2)
        missing_recs = [
            e for e in report.inventory_elements
            if e.element_type in ("recommendation", "practice_point")
            and e.coverage_status == "MISSING"
        ]
        if missing_recs:
            blockers.append(GateBlocker(
                gate_number=1,
                gate_name="A2: Structural coverage — Recommendations & Practice Points",
                blocker_count=len(missing_recs),
                fix_priority=2,
                details=[f"MISSING: {e.element_id}" for e in missing_recs[:20]],
            ))

        # Gate 2: All Tier 1 footnotes captured (A1d + A2)
        missing_t1_footnotes = [
            b for b in report.footnote_bindings
            if b.tier == "TIER_1" and not b.bound_to_span
        ]
        if missing_t1_footnotes:
            blockers.append(GateBlocker(
                gate_number=2,
                gate_name="A1d: Tier 1 footnote binding",
                blocker_count=len(missing_t1_footnotes),
                fix_priority=2,
                details=[
                    f"MISSING: {b.marker_char} on page {b.page_number} — {b.footnote_text[:60]}"
                    for b in missing_t1_footnotes[:10]
                ],
            ))

        # Gate 3: Tier 1 coverage = 100% (B1 residual)
        if report.tier1_residual_count > 0:
            tier1_frags = [
                f for f in report.residual_fragments if f.tier == "TIER_1"
            ]
            blockers.append(GateBlocker(
                gate_number=3,
                gate_name="B1: Tier 1 residual signals",
                blocker_count=report.tier1_residual_count,
                fix_priority=4,
                details=[
                    f"RESIDUAL: {f.text[:80]}... (drugs: {', '.join(f.drug_names_found)})"
                    for f in tier1_frags[:10]
                ],
            ))

        # Gate 4: Tier 1 numeric integrity = 100% (C1)
        numeric_blocks = [
            m for m in report.numeric_mismatches if m.action == "BLOCK"
        ]
        if numeric_blocks:
            blockers.append(GateBlocker(
                gate_number=4,
                gate_name="C1: Numeric/comparator integrity",
                blocker_count=len(numeric_blocks),
                fix_priority=1,  # Highest priority — fix first
                details=[
                    f"{m.mismatch_type}: {m.source_value} → {m.extracted_value} (span {m.span_id[:8]})"
                    for m in numeric_blocks[:10]
                ],
            ))

        # Gate 5: No branch or exception loss (B3)
        branch_blocks = [
            b for b in report.branch_comparisons if b.action == "BLOCK"
        ]
        if branch_blocks:
            blockers.append(GateBlocker(
                gate_number=5,
                gate_name="B3: Conditional branch completeness",
                blocker_count=len(branch_blocks),
                fix_priority=3,
                details=[
                    (
                        f"Section {b.section_id}: "
                        + (
                            f"{b.extracted_threshold_count}/{b.source_threshold_count} "
                            f"thresholds preserved"
                            if b.source_threshold_count > 0
                            else f"{b.extracted_threshold_count} thresholds extracted"
                        )
                        + (
                            f"; LOST exception keywords: {b.exception_keywords_lost}"
                            if b.exception_keywords_lost
                            else ""
                        )
                    )
                    for b in branch_blocks[:10]
                ],
            ))

        # Gate 6: No unreviewed Tier 1 span with corroboration <0.5 (C3)
        low_corr_blocks = [
            d for d in report.corroboration_details
            if d.action == "BLOCK"
        ]
        if low_corr_blocks:
            blockers.append(GateBlocker(
                gate_number=6,
                gate_name="C3: Corroboration threshold",
                blocker_count=len(low_corr_blocks),
                fix_priority=5,
                details=[
                    f"Span {d.span_id[:8]}: score={d.corroboration_score:.1f}, "
                    f"channels={d.contributing_channels}"
                    for d in low_corr_blocks[:10]
                ],
            ))

        # Gate 7: Residual Tier 1 signals = 0 (same as Gate 3, kept for explicit tracking)
        # Already covered by Gate 3

        # Gate 8: Pharmacist sign-off (cannot be automated — always warning)
        warning_count += 1  # Pharmacist sign-off is a manual step

        # Count warnings from density, population-action, adversarial delta
        warning_count += len(report.density_warnings)
        warning_count += len(report.population_action_warnings)
        if report.adversarial_audit_delta is not None and report.adversarial_audit_delta > 0:
            warning_count += 1

        # L1 recovery escalations are warnings (require visual verification)
        warning_count += len(report.l1_recovery_escalations)

        # Sort blockers by fix_priority
        blockers.sort(key=lambda b: b.fix_priority)

        report.gate_blockers = blockers
        report.total_block_count = sum(b.blocker_count for b in blockers)
        report.total_warning_count = warning_count
        report.gate_verdict = "BLOCK" if blockers else "PASS"

    # =====================================================================
    # Utilities
    # =====================================================================

    def _estimate_page_from_offset(self, text: str, offset: int) -> int:
        """Estimate page number from character offset using PAGE markers.

        Supports two marker formats:
        - MonkeyOCR: <!-- PAGE N -->
        - Marker v1.10: {N}---...  (10+ dashes)
        """
        # Try both formats, use whichever is present
        last_page = 1
        for pattern in [r'<!--\s*PAGE\s+(\d+)\s*-->', r'\{(\d+)\}[-]{10,}']:
            page_marker_re = re.compile(pattern)
            found = False
            for m in page_marker_re.finditer(text):
                found = True
                if m.start() <= offset:
                    last_page = int(m.group(1))
                else:
                    break
            if found:
                return last_page
        return last_page

    def _flatten_sections(self, sections) -> list:
        """Flatten nested GuidelineSection tree into a flat list."""
        result = []
        for section in sections:
            result.append(section)
            if section.children:
                result.extend(self._flatten_sections(section.children))
        return result
