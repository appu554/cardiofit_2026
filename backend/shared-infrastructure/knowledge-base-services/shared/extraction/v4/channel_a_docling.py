"""
Channel A: Structural Oracle (V4.2 — Granite-Docling mandatory).

V4.2 combines TWO inputs to build the richest possible GuidelineTree:
1. Marker's normalized markdown (text-of-record, via Channel 0)
2. Granite-Docling VlmPipeline DocTags (structural oracle, from original PDF)

The alignment problem: GuidelineTree offsets must reference Marker's markdown
because Channels B-F operate on that text. DocTags give us semantic types
(section_header, footnote, caption, OTSL tables) but no Marker offsets.
Solution: align DocTags headings to Marker headings via text matching.

V4.2 change: Granite-Docling is MANDATORY. No regex fallback.
- pdf_path MUST be provided
- Granite-Docling import failure raises immediately (no silent degradation)
- Low alignment confidence logs a warning but still uses Granite results

V4.2.1 change: Improved heading alignment for VLM text.
- Text normalization: strips markdown formatting, collapses whitespace, normalizes unicode
- Multi-tier matching: exact → substring → section_number → word_overlap → fuzzy
- Lowered fuzzy threshold from 0.85 → 0.70 (VLM OCR text differs from PDF text layer)
- Debug logging for alignment diagnostics

Pipeline Position:
    Channel 0 (normalized text) -> Channel A (THIS) -> Channels B-F (parallel)
"""

from __future__ import annotations

import difflib
import logging
import re
import time
import unicodedata
from typing import Optional

from .models import GuidelineSection, GuidelineTree, TableBoundary
from .provenance import (
    ChannelProvenance,
    _normalise_bbox,
    _normalise_confidence,
    _normalise_page_number,
)
from .v5_flags import is_v5_enabled

logger = logging.getLogger(__name__)


# Channel A model version for ChannelProvenance.model_version. Pinned to the
# Granite-Docling extractor's VERSION constant so a bump there propagates
# automatically to the audit trail.
def _channel_a_model_version() -> str:
    try:
        from .granite_docling_extractor import GraniteDoclingExtractor
        return f"granite-docling@{GraniteDoclingExtractor.VERSION}"
    except Exception:
        # Defensive: never fail span construction over a version-tag lookup.
        return "granite-docling@unknown"


def _channel_a_provenance(
    bbox: tuple[float, float, float, float] | list[float] | None,
    page_number: int,
    confidence: float,
    profile,
    notes: Optional[str] = None,
) -> Optional[ChannelProvenance]:
    """Build a ChannelProvenance entry for Channel A.

    Returns None when:
      - V5_BBOX_PROVENANCE flag is off (default), OR
      - bbox is None / not a 4-tuple (no fake bboxes — caller is expected
        to leave provenance unset rather than fabricate coordinates).

    Defensive normalisation:
      - x0/y0 clamped to >=0 (some upstream parsers emit -0.5 etc.)
      - x1/y1 clamped to >= x0/y0 to satisfy BoundingBox._check_ordered
      - page_number clamped to >=1
      - confidence clamped to [0.0, 1.0]
    """
    if not is_v5_enabled("bbox_provenance", profile):
        return None
    bb = _normalise_bbox(bbox)
    if bb is None:
        return None
    return ChannelProvenance(
        channel_id="A",
        bbox=bb,
        page_number=_normalise_page_number(page_number),
        confidence=_normalise_confidence(confidence),
        model_version=_channel_a_model_version(),
        notes=notes,
    )


class ChannelAStructuralOracle:
    """Granite-Docling structural oracle with Marker text alignment.

    Uses Granite-Docling VlmPipeline (258M) for STRUCTURAL understanding:
    - Section hierarchy from DocTags nesting
    - Table detection via OTSL format
    - Footnote, caption, list type discrimination

    Uses Marker's normalized markdown as TEXT OF RECORD:
    - All offsets reference positions in Marker's markdown
    - Channels B-F operate on Marker's text

    V4.2: Granite-Docling is MANDATORY. No regex fallback.
    - pdf_path required (raises ValueError if missing)
    - Import/model errors propagate (no silent degradation)
    - Low alignment confidence warns but still returns Granite results
    """

    VERSION = "4.2.2"
    ALIGNMENT_THRESHOLD = 0.80  # minimum heading match ratio

    def __init__(
        self,
        subordinate_headings: Optional[list[str]] = None,
        chapter_reset_headings: Optional[list[str]] = None,
        profile=None,
    ) -> None:
        """Initialize with optional profile-driven heading sets.

        Args:
            subordinate_headings: Lowercased headings to reparent under the
                preceding numbered section. None = use KDIGO defaults.
                Empty list = skip reparenting entirely.
            chapter_reset_headings: Lowercased headings that reset the
                numbered-section tracker. None = use KDIGO defaults.
        """
        if subordinate_headings is not None:
            self._subordinate_headings = frozenset(subordinate_headings)
        else:
            self._subordinate_headings = self._DEFAULT_SUBORDINATE_HEADINGS

        if chapter_reset_headings is not None:
            self._chapter_reset_headings = frozenset(chapter_reset_headings)
        else:
            self._chapter_reset_headings = self._DEFAULT_CHAPTER_RESET_HEADINGS

        # V5: profile is used by _channel_a_provenance() to consult v5_features.
        # Always allowed to be None (V4 callers don't pass it).
        self.profile = profile

    # ── Regex patterns (shared by both oracle and fallback paths) ──────────

    HEADING_RE = re.compile(r'^(#{1,6})\s+(.+)$', re.MULTILINE)
    TABLE_ROW_RE = re.compile(r'^\|(.+)\|$', re.MULTILINE)
    TABLE_SEP_RE = re.compile(r'^\|[-:\s|]+\|$', re.MULTILINE)
    # V4.2.1: Detect both Marker page-break format {N}--- and HTML comment format
    PAGE_MARKER_RE = re.compile(
        r'(?:^\{(\d+)\}\s*-{3,})|(?:<!--\s*PAGE\s+(\d+)\s*-->)',
        re.IGNORECASE | re.MULTILINE,
    )
    RECOMMENDATION_RE = re.compile(
        r'Recommendation\s+(\d+(?:\.\d+)+)', re.IGNORECASE
    )
    SECTION_NUMBER_RE = re.compile(r'(\d+(?:\.\d+)*)')

    # V4.2.1: Markdown stripping for heading normalization
    _MD_FORMATTING_RE = re.compile(r'\*{1,2}|_{1,2}')
    # Section number pattern for tier-3 alignment (requires at least one dot)
    _SECTION_NUM_RE = re.compile(r'\b(\d+(?:\.\d+)+)\b')
    # Fuzzy threshold for heading alignment (lowered from 0.85 for VLM text)
    _FUZZY_HEADING_THRESHOLD = 0.70
    # Word overlap threshold for tier-4 alignment
    _WORD_OVERLAP_THRESHOLD = 0.60

    @staticmethod
    def _normalize_heading_text(text: str) -> str:
        """Normalize heading text for cross-backend comparison.

        Strips markdown formatting, normalizes unicode, collapses whitespace.
        Used to compare Granite-Docling VLM text against Marker ATX headings.
        """
        # Strip markdown bold/italic markers (**bold**, *italic*, __bold__, _italic_)
        t = ChannelAStructuralOracle._MD_FORMATTING_RE.sub('', text)
        # Normalize unicode (NFC — canonical decomposition then composition)
        t = unicodedata.normalize('NFC', t)
        # Collapse all whitespace (including \n, \t) to single space
        t = re.sub(r'\s+', ' ', t)
        return t.strip()

    # ── Main Entry Point ──────────────────────────────────────────────────

    def parse(
        self,
        text: str,
        pdf_path: Optional[str] = None,
    ) -> GuidelineTree:
        """Parse structure using Granite-Docling structural oracle (mandatory).

        V4.2: Granite-Docling is required. No regex fallback.

        Args:
            text: Normalized markdown text (Channel 0 output from Marker)
            pdf_path: Original PDF path for Granite-Docling processing.
                      REQUIRED — raises ValueError if not provided.

        Returns:
            GuidelineTree with sections, tables, page count, and alignment metadata

        Raises:
            ValueError: If pdf_path is not provided
            ImportError: If Granite-Docling (VlmPipeline) is not installed
            RuntimeError: If Granite-Docling extraction fails
        """
        # Reset section ID counter for each new document
        self._section_id_counter = {}
        self._reparent_log: list[dict] = []

        if not pdf_path:
            raise ValueError(
                "Channel A (V4.2): pdf_path is REQUIRED for Granite-Docling "
                "structural oracle. No regex fallback available."
            )

        tree = self._parse_with_granite_docling(text, pdf_path)

        if tree.alignment_confidence < self.ALIGNMENT_THRESHOLD:
            logger.warning(
                "Channel A: Granite-Docling alignment confidence %.0f%% is below "
                "threshold %.0f%%. Using Granite results anyway (no regex fallback).",
                tree.alignment_confidence * 100,
                self.ALIGNMENT_THRESHOLD * 100,
            )

        return tree

    # ═══════════════════════════════════════════════════════════════════════
    # GRANITE-DOCLING STRUCTURAL ORACLE PATH
    # ═══════════════════════════════════════════════════════════════════════

    def _parse_with_granite_docling(
        self, text: str, pdf_path: str
    ) -> GuidelineTree:
        """Run Granite-Docling on PDF, align DocTags to Marker text."""
        from .granite_docling_extractor import GraniteDoclingExtractor

        extractor = GraniteDoclingExtractor()
        doctags = extractor.extract(pdf_path)

        if doctags.error:
            raise RuntimeError(f"Granite-Docling failed: {doctags.error}")

        # Build page map from Marker text (for offset→page lookups)
        page_map = self._build_page_map(text)
        total_pages = max(
            max(page_map.values()) if page_map else 1,
            doctags.total_pages,
        )

        # Step 1: Align DocTags headings to Marker headings
        aligned_sections, alignment_confidence = self._align_sections(
            doctags, text, page_map
        )

        # Step 2: Align tables (Marker pipe tables + Granite OTSL tables)
        tables = self._align_tables(doctags, text, page_map, aligned_sections)

        # Step 3: Enrich section block_types from DocTags elements
        self._enrich_section_types(aligned_sections, doctags)

        # Step 4: Build hierarchy from flat sections
        tree_sections = self._build_hierarchy(aligned_sections)

        # Step 5: V4.2.1 Fix #1b — Reparent KDIGO subordinate headings
        tree_sections = self._reparent_kdigo_subordinates(tree_sections)

        # Step 6: Expand end_offsets AFTER reparenting (must be this order)
        self._expand_end_offsets(tree_sections)

        # Step 7: Validate tree structure post-reparenting
        self._validate_kdigo_tree(tree_sections)

        return GuidelineTree(
            sections=tree_sections,
            tables=tables,
            total_pages=total_pages,
            alignment_confidence=alignment_confidence,
            structural_source="granite_doctags",
            page_map=page_map,
        )

    # ── Section Alignment ─────────────────────────────────────────────────

    def _align_sections(
        self, doctags, text: str, page_map: dict[int, int]
    ) -> tuple[list[GuidelineSection], float]:
        """Align DocTags section headers to Marker ATX headings.

        Returns:
            (aligned_sections, confidence_ratio)
        """
        # Get Marker headings with offsets
        marker_headings = list(self.HEADING_RE.finditer(text))

        if not marker_headings and not doctags.sections:
            return [GuidelineSection(
                section_id="1", heading="Document",
                start_offset=0, end_offset=len(text),
                page_number=1, block_type="paragraph", level=1, children=[],
            )], 1.0

        # If no DocTags sections, fall back to Marker headings only
        if not doctags.sections:
            return self._sections_from_marker(text, page_map), 0.0

        # Match DocTags headings to Marker headings by text similarity
        matched = 0
        sections: list[GuidelineSection] = []
        used_marker_indices: set[int] = set()

        logger.debug(
            "Alignment: %d DocTags sections vs %d Marker ATX headings",
            len(doctags.sections), len(marker_headings),
        )

        for doctag_section in doctags.sections:
            best_idx, best_match, tier = self._find_best_marker_heading(
                doctag_section.text, marker_headings, used_marker_indices
            )

            if best_match and best_idx is not None:
                logger.debug(
                    "  MATCHED [tier=%s] Granite=%r → Marker=%r",
                    tier, doctag_section.text[:60],
                    best_match.group(2).strip()[:60],
                )
                # Successfully aligned
                match = best_match
                level = len(match.group(1))
                heading_text = match.group(2).strip()
                start_offset = match.start()

                # Find end offset (next heading or end of text)
                end_offset = self._find_heading_end(
                    best_idx, marker_headings, len(text)
                )

                page_number = self._get_page_for_offset(start_offset, page_map)
                section_id = self._extract_section_id(heading_text, level)
                block_type = doctag_section.doctag_type or self._classify_block_type(
                    heading_text, level
                )

                sections.append(GuidelineSection(
                    section_id=section_id,
                    heading=heading_text,
                    start_offset=start_offset,
                    end_offset=end_offset,
                    page_number=page_number,
                    block_type=block_type,
                    level=level,
                    children=[],
                    provenance=_channel_a_provenance(
                        bbox=None,  # DocTagSection has no bbox today; helper no-ops.
                        page_number=page_number,
                        confidence=1.0,
                        profile=self.profile,
                        notes=f"docling-aligned: {heading_text[:40]}",
                    ),
                ))
                used_marker_indices.add(best_idx)
                matched += 1
            else:
                logger.debug(
                    "  MISSED Granite=%r (no Marker match found)",
                    doctag_section.text[:80],
                )

        # Add any Marker headings that weren't matched to DocTags
        for i, match in enumerate(marker_headings):
            if i not in used_marker_indices:
                level = len(match.group(1))
                heading_text = match.group(2).strip()
                start_offset = match.start()
                end_offset = self._find_heading_end(
                    i, marker_headings, len(text)
                )
                page_number = self._get_page_for_offset(start_offset, page_map)
                section_id = self._extract_section_id(heading_text, level)
                block_type = self._classify_block_type(heading_text, level)

                sections.append(GuidelineSection(
                    section_id=section_id,
                    heading=heading_text,
                    start_offset=start_offset,
                    end_offset=end_offset,
                    page_number=page_number,
                    block_type=block_type,
                    level=level,
                    children=[],
                    provenance=_channel_a_provenance(
                        bbox=None,  # Marker-only heading, no docling bbox.
                        page_number=page_number,
                        confidence=0.5,  # lower: Marker saw it, docling did not
                        profile=self.profile,
                        notes=f"marker-only: {heading_text[:40]}",
                    ),
                ))

        # Sort by start_offset for hierarchy building
        sections.sort(key=lambda s: s.start_offset)

        # Alignment confidence = fraction of DocTags headings matched
        total_doctag_sections = len(doctags.sections)
        confidence = matched / total_doctag_sections if total_doctag_sections > 0 else 0.0

        logger.info(
            "Channel A alignment: %d/%d DocTags matched (%.0f%%), "
            "%d unmatched Marker headings added",
            matched, total_doctag_sections, confidence * 100,
            len(marker_headings) - len(used_marker_indices),
        )

        return sections, confidence

    def _find_best_marker_heading(
        self,
        doctag_text: str,
        marker_headings: list,
        used_indices: set[int],
    ) -> tuple[Optional[int], Optional[re.Match], str]:
        """Find the Marker heading that best matches a DocTags heading text.

        V4.2.1: Multi-tier matching with text normalization to handle
        differences between Granite-Docling VLM output and Marker text.

        Tiers (checked in order, first match wins):
          1. Exact match on normalized text (stripped markdown, collapsed whitespace)
          2. Substring containment on normalized text
          3. Section number match (e.g., "2.1.2" appears in both headings)
          4. Word overlap >= 60% of the shorter heading's words
          5. Fuzzy match (SequenceMatcher ratio >= 0.70)

        Returns:
            (index, regex_match, tier_name) — tier_name is for logging.
            (None, None, "none") if no match found.
        """
        norm_doctag = self._normalize_heading_text(doctag_text)
        norm_doctag_lower = norm_doctag.lower()
        doctag_words = set(norm_doctag_lower.split())

        best_ratio = 0.0
        best_idx = None
        best_match = None

        for i, match in enumerate(marker_headings):
            if i in used_indices:
                continue

            marker_text = match.group(2).strip()
            norm_marker = self._normalize_heading_text(marker_text)
            norm_marker_lower = norm_marker.lower()

            # Tier 1: Exact match on normalized text
            if norm_doctag_lower == norm_marker_lower:
                return i, match, "exact"

            # Tier 2: Substring containment on normalized text
            if (norm_doctag_lower in norm_marker_lower
                    or norm_marker_lower in norm_doctag_lower):
                return i, match, "substring"

            # Tier 3: Section number match (e.g., "2.1.2" in both)
            doctag_nums = self._SECTION_NUM_RE.findall(norm_doctag)
            if doctag_nums:
                marker_nums = self._SECTION_NUM_RE.findall(norm_marker)
                if any(n in marker_nums for n in doctag_nums):
                    return i, match, "section_num"

            # Tier 4: Word overlap >= 60% of shorter heading
            marker_words = set(norm_marker_lower.split())
            if len(doctag_words) >= 2 and len(marker_words) >= 2:
                overlap = len(doctag_words & marker_words)
                shorter = min(len(doctag_words), len(marker_words))
                if shorter > 0 and overlap / shorter >= self._WORD_OVERLAP_THRESHOLD:
                    return i, match, "word_overlap"

            # Tier 5: Fuzzy match (accumulate best across all candidates)
            ratio = difflib.SequenceMatcher(
                None, norm_doctag_lower, norm_marker_lower,
            ).ratio()

            if ratio > best_ratio and ratio >= self._FUZZY_HEADING_THRESHOLD:
                best_ratio = ratio
                best_idx = i
                best_match = match

        if best_match is not None:
            return best_idx, best_match, f"fuzzy({best_ratio:.2f})"

        return None, None, "none"

    def _find_heading_end(
        self, idx: int, headings: list, text_length: int
    ) -> int:
        """Find end offset for a heading (start of next heading or end of text)."""
        if idx + 1 < len(headings):
            return headings[idx + 1].start()
        return text_length

    # ── Table Alignment ───────────────────────────────────────────────────

    def _align_tables(
        self,
        doctags,
        text: str,
        page_map: dict[int, int],
        sections: list[GuidelineSection],
    ) -> list[TableBoundary]:
        """Align Granite-Docling tables to Marker markdown.

        Three outcomes per table:
        1. Marker has matching pipe table -> use Marker offsets (source="marker_pipe")
        2. Marker missed the table -> store OTSL text (source="granite_otsl")
        3. No match in either -> skip (logged as warning)
        """
        tables: list[TableBoundary] = []
        table_id = 0

        # First, extract all Marker pipe tables
        marker_tables = self._extract_marker_pipe_tables(text, page_map, sections)

        # Track which Marker tables get matched
        matched_marker_tables: set[int] = set()

        # For each Granite-Docling table, try to find a matching Marker table
        for otsl_table in doctags.tables:
            match_idx = self._find_matching_pipe_table(
                otsl_table.column_headers, marker_tables, matched_marker_tables
            )

            if match_idx is not None:
                # Marker has this table — use Marker offsets, enrich with OTSL
                mt = marker_tables[match_idx]
                tables.append(TableBoundary(
                    table_id=f"table_{table_id}",
                    section_id=mt.section_id,
                    start_offset=mt.start_offset,
                    end_offset=mt.end_offset,
                    headers=mt.headers,
                    row_count=mt.row_count,
                    page_number=mt.page_number,
                    source="marker_pipe",
                    otsl_text=otsl_table.raw_otsl,  # store OTSL as enrichment
                    provenance=_channel_a_provenance(
                        bbox=None,
                        page_number=mt.page_number,
                        confidence=1.0,
                        profile=self.profile,
                        notes=f"docling+marker table_{table_id}",
                    ),
                ))
                matched_marker_tables.add(match_idx)
            else:
                # Marker missed this table — store as OTSL-only
                parent_section_id = self._find_parent_section_by_page(
                    otsl_table.page_number, sections
                )
                tables.append(TableBoundary(
                    table_id=f"table_{table_id}",
                    section_id=parent_section_id,
                    start_offset=-1,
                    end_offset=-1,
                    headers=otsl_table.column_headers,
                    row_count=otsl_table.row_count,
                    page_number=otsl_table.page_number,
                    source="granite_otsl",
                    otsl_text=otsl_table.raw_otsl,
                    provenance=_channel_a_provenance(
                        bbox=None,
                        page_number=otsl_table.page_number,
                        confidence=0.8,  # docling-only table
                        profile=self.profile,
                        notes=f"granite_otsl table_{table_id}",
                    ),
                ))

            table_id += 1

        # Add any Marker tables that weren't matched to Granite-Docling
        for i, mt in enumerate(marker_tables):
            if i not in matched_marker_tables:
                tables.append(TableBoundary(
                    table_id=f"table_{table_id}",
                    section_id=mt.section_id,
                    start_offset=mt.start_offset,
                    end_offset=mt.end_offset,
                    headers=mt.headers,
                    row_count=mt.row_count,
                    page_number=mt.page_number,
                    source="marker_pipe",
                    provenance=_channel_a_provenance(
                        bbox=None,
                        page_number=mt.page_number,
                        confidence=0.5,  # marker-only, docling missed it
                        profile=self.profile,
                        notes=f"marker-only table_{table_id}",
                    ),
                ))
                table_id += 1

        return tables

    def _find_matching_pipe_table(
        self,
        otsl_headers: list[str],
        marker_tables: list[TableBoundary],
        used_indices: set[int],
    ) -> Optional[int]:
        """Find a Marker pipe table matching OTSL column headers."""
        if not otsl_headers:
            return None

        for i, mt in enumerate(marker_tables):
            if i in used_indices:
                continue

            if not mt.headers:
                continue

            # Exact header match
            if mt.headers == otsl_headers:
                return i

            # Fuzzy header match: at least 60% of headers match
            matches = sum(
                1 for oh in otsl_headers
                if any(
                    oh.strip().lower() == mh.strip().lower()
                    for mh in mt.headers
                )
            )
            if matches / max(len(otsl_headers), 1) >= 0.6:
                return i

        return None

    def _find_parent_section_by_page(
        self, page_number: int, sections: list[GuidelineSection]
    ) -> str:
        """Find parent section_id for a given page number."""
        for section in reversed(sections):
            if section.page_number <= page_number:
                return section.section_id
        return sections[0].section_id if sections else "unknown"

    # ── Section Type Enrichment ───────────────────────────────────────────

    def _enrich_section_types(
        self, sections: list[GuidelineSection], doctags
    ) -> None:
        """Enrich section block_types from DocTags elements.

        DocTags provide footnote, caption, ordered_list, unordered_list types
        that Marker's markdown doesn't distinguish. This is metadata-only
        enrichment — no offset recalculation needed.
        """
        for element in doctags.elements:
            # Find the section on the same page containing this element
            for section in sections:
                if section.page_number == element.page_number:
                    # Check if element text appears in the section heading
                    if element.text and element.text[:30] in section.heading:
                        section.block_type = element.doctag_type
                        break

    # ═══════════════════════════════════════════════════════════════════════
    # REGEX PATH (V4.0 legacy — DEPRECATED in V4.2, kept for utilities)
    # Not called by parse() anymore. Shared utilities below are still used
    # by the Granite-Docling path (_sections_from_marker, _build_page_map, etc.)
    # ═══════════════════════════════════════════════════════════════════════

    def _parse_markdown_regex(self, text: str) -> GuidelineTree:
        """Regex-based heading parser for born-digital PDFs.

        V4.0 original path, re-enabled in V4.2.5 as --structure regex
        fast path when Granite-Docling is impractical on CPU.
        """
        self._section_id_counter = {}
        self._reparent_log: list[dict] = []

        page_map = self._build_page_map(text)
        flat_sections = self._sections_from_marker(text, page_map)
        tables = self._extract_marker_pipe_tables(text, page_map, flat_sections)
        tree_sections = self._build_hierarchy(flat_sections)
        tree_sections = self._reparent_kdigo_subordinates(tree_sections)
        self._validate_kdigo_tree(tree_sections)
        self._expand_end_offsets(tree_sections)

        total_pages = max(page_map.values()) if page_map else 1

        tree = GuidelineTree(
            sections=tree_sections,
            tables=tables,
            total_pages=total_pages,
            alignment_confidence=1.0,  # regex is self-consistent
            structural_source="regex",
            page_map=page_map,
        )
        return tree

    # ═══════════════════════════════════════════════════════════════════════
    # SHARED UTILITIES (used by both oracle and fallback paths)
    # ═══════════════════════════════════════════════════════════════════════

    def _build_page_map(self, text: str) -> dict[int, int]:
        """Build a mapping of character offset -> page number.

        V4.2.1: Detects both Marker's {N}--- format and HTML <!-- PAGE N -->
        format. The page number in {N}--- is the original PDF page index
        (typically 0-based from Marker), which we convert to 1-based.
        """
        page_map: dict[int, int] = {}

        for match in self.PAGE_MARKER_RE.finditer(text):
            # Group 1 = {N}--- format, Group 2 = <!-- PAGE N --> format
            raw_num = match.group(1) or match.group(2)
            page_num = int(raw_num)
            # Marker uses 0-based page indices; convert to 1-based
            if match.group(1) is not None and page_num == 0:
                page_num = 1
            elif match.group(1) is not None:
                page_num += 1
            page_map[match.start()] = page_num

        if not page_map:
            page_map[0] = 1

        return page_map

    def _get_page_for_offset(
        self, offset: int, page_map: dict[int, int]
    ) -> int:
        """Get the page number for a given character offset."""
        page = 1
        for marker_offset, page_num in sorted(page_map.items()):
            if marker_offset <= offset:
                page = page_num
            else:
                break
        return page

    def _sections_from_marker(
        self, text: str, page_map: dict[int, int]
    ) -> list[GuidelineSection]:
        """Extract flat list of sections from Marker ATX headings."""
        sections: list[GuidelineSection] = []

        headings = list(self.HEADING_RE.finditer(text))

        for i, match in enumerate(headings):
            level = len(match.group(1))
            heading_text = match.group(2).strip()
            start_offset = match.start()

            if i + 1 < len(headings):
                end_offset = headings[i + 1].start()
            else:
                end_offset = len(text)

            page_number = self._get_page_for_offset(start_offset, page_map)
            section_id = self._extract_section_id(heading_text, level)
            block_type = self._classify_block_type(heading_text, level)

            sections.append(GuidelineSection(
                section_id=section_id,
                heading=heading_text,
                start_offset=start_offset,
                end_offset=end_offset,
                page_number=page_number,
                block_type=block_type,
                level=level,
                children=[],
                provenance=_channel_a_provenance(
                    bbox=None,
                    page_number=page_number,
                    confidence=0.5,
                    profile=self.profile,
                    notes=f"marker-fallback: {heading_text[:40]}",
                ),
            ))

        if not sections:
            sections.append(GuidelineSection(
                section_id="1",
                heading="Document",
                start_offset=0,
                end_offset=len(text),
                page_number=1,
                block_type="paragraph",
                level=1,
                children=[],
            ))

        return sections

    # Counter for ensuring unique section IDs when headings collide
    _section_id_counter: dict[str, int] = {}

    def _extract_section_id(self, heading_text: str, level: int) -> str:
        """Extract a section ID from heading text."""
        rec_match = self.RECOMMENDATION_RE.search(heading_text)
        if rec_match:
            return rec_match.group(1)

        num_match = self.SECTION_NUMBER_RE.search(heading_text)
        if num_match:
            return num_match.group(1)

        cleaned = heading_text
        cleaned = re.sub(r'\*{1,2}([^*]+)\*{1,2}', r'\1', cleaned)
        cleaned = cleaned.strip()
        sanitized = re.sub(r'[^a-zA-Z0-9_]', '_', cleaned[:50])
        sanitized = re.sub(r'_+', '_', sanitized).strip('_')
        base_id = sanitized or f"section_L{level}"

        if base_id not in self._section_id_counter:
            self._section_id_counter[base_id] = 0
            return base_id
        else:
            self._section_id_counter[base_id] += 1
            return f"{base_id}_{self._section_id_counter[base_id]}"

    def _classify_block_type(self, heading_text: str, level: int) -> str:
        """Classify a section's block type based on heading content."""
        heading_lower = heading_text.lower()

        if self.RECOMMENDATION_RE.search(heading_text):
            return "recommendation"
        if heading_lower.startswith("chapter") or level == 1:
            return "heading"
        if heading_lower.startswith("table") or heading_lower.startswith("figure"):
            return "heading"

        return "heading"

    def _extract_marker_pipe_tables(
        self,
        text: str,
        page_map: dict[int, int],
        sections: list[GuidelineSection],
    ) -> list[TableBoundary]:
        """Extract table boundaries from markdown pipe tables."""
        tables: list[TableBoundary] = []
        table_id = 0

        lines = text.split('\n')
        line_starts: list[int] = []
        offset = 0
        for line in lines:
            line_starts.append(offset)
            offset += len(line) + 1

        i = 0
        while i < len(lines):
            line = lines[i].strip()

            if line.startswith('|') and line.endswith('|'):
                table_start = line_starts[i]
                headers: list[str] = []
                row_count = 0
                has_separator = False

                cells = [c.strip() for c in line.split('|')[1:-1]]
                if cells:
                    headers = cells

                j = i + 1
                while j < len(lines):
                    next_line = lines[j].strip()
                    if not (next_line.startswith('|') and next_line.endswith('|')):
                        break
                    if self.TABLE_SEP_RE.match(next_line):
                        has_separator = True
                    else:
                        row_count += 1
                    j += 1

                if has_separator and row_count > 0:
                    table_end = line_starts[j - 1] + len(lines[j - 1])
                    page = self._get_page_for_offset(table_start, page_map)

                    parent_section_id = self._find_parent_section_id(
                        table_start, sections
                    )

                    tables.append(TableBoundary(
                        table_id=f"table_{table_id}",
                        section_id=parent_section_id,
                        start_offset=table_start,
                        end_offset=table_end,
                        headers=headers,
                        row_count=row_count,
                        page_number=page,
                        source="marker_pipe",
                        provenance=_channel_a_provenance(
                            bbox=None,
                            page_number=page,
                            confidence=0.5,
                            profile=self.profile,
                            notes=f"marker-pipe table_{table_id}",
                        ),
                    ))
                    table_id += 1

                i = j
            else:
                i += 1

        return tables

    def _find_parent_section_id(
        self, offset: int, sections: list[GuidelineSection]
    ) -> str:
        """Find the section_id of the section containing the given offset."""
        for section in reversed(sections):
            if section.start_offset <= offset < section.end_offset:
                return section.section_id
        return sections[0].section_id if sections else "unknown"

    # Matches pure numeric section IDs like "2", "2.2", "4.1.2"
    _NUMERIC_SECTION_ID_RE = re.compile(r'^\d+(?:\.\d+)*$')

    def _build_hierarchy(
        self, flat_sections: list[GuidelineSection]
    ) -> list[GuidelineSection]:
        """Build parent-child relationships using hybrid depth.

        V4.2.1: Uses a hybrid depth strategy:
        - Numbered sections (section_id like "2.2", "4.1.2"): depth = number of
          dot-separated parts. This is authoritative for clinical guidelines where
          the numbering defines the canonical hierarchy.
        - Non-numbered sections (section_id like "Research_recommendations"):
          depth = ATX heading level (section.level from Marker ## / ###).

        This prevents mis-nesting like "2.2 Glycemic targets" (major section)
        becoming a child of "Research recommendations" (content-level heading)
        when Marker assigns incorrect ATX levels to numbered headings.
        """
        if not flat_sections:
            return []

        root_sections: list[GuidelineSection] = []
        stack: list[GuidelineSection] = []

        for section in flat_sections:
            depth = self._effective_depth(section)

            while stack and self._effective_depth(stack[-1]) >= depth:
                stack.pop()

            if stack:
                stack[-1].children.append(section)
            else:
                root_sections.append(section)

            stack.append(section)

        # V4.2.1: _expand_end_offsets extracted from here so reparenting
        # can run between hierarchy build and offset expansion.
        # Callers must call _expand_end_offsets() after any reparenting.

        return root_sections

    def _effective_depth(self, section: GuidelineSection) -> int:
        """Compute hierarchy depth for a section.

        Numbered sections: depth = count of dot-separated parts
            "2" → 1, "2.2" → 2, "2.1.2" → 3
        Non-numbered sections: depth = ATX heading level
            ## → 2, ### → 3
        """
        if self._NUMERIC_SECTION_ID_RE.match(section.section_id):
            return section.section_id.count('.') + 1
        return section.level

    def _expand_end_offsets(self, sections: list[GuidelineSection]) -> None:
        """Recursively expand each section's end_offset to cover all children."""
        for section in sections:
            if section.children:
                self._expand_end_offsets(section.children)
                max_child_end = max(c.end_offset for c in section.children)
                if max_child_end > section.end_offset:
                    section.end_offset = max_child_end

    # ── V4.2.2: Profile-driven Subordinate Heading Reparenting ────────────

    # Default KDIGO subordinate heading vocabulary (backward-compatible).
    # Overridden by GuidelineProfile.subordinate_headings when provided.
    _DEFAULT_SUBORDINATE_HEADINGS = frozenset({
        "key information",
        "balance of benefits and harms",
        "certainty of evidence",
        "certainty of the evidence",
        "values and preferences",
        "resource use and costs",
        "considerations for implementation",
        "rationale",
        "rationale and evidence",
        "evidence base",
    })

    _DEFAULT_CHAPTER_RESET_HEADINGS = frozenset({
        "research recommendations",
        "practice implications",
    })

    def _reparent_kdigo_subordinates(
        self, root_sections: list[GuidelineSection]
    ) -> list[GuidelineSection]:
        """Reparent subordinate headings under their governing numbered section.

        V4.2.2: Now profile-driven via self._subordinate_headings and
        self._chapter_reset_headings (set in __init__ from GuidelineProfile).
        If _subordinate_headings is empty, the reparenting pass is skipped.

        Three-pass approach:
        1. Collect: DFS walk producing (section, container_list) pairs in document order
        2. Identify: Track last_numbered; queue subordinates for reparenting;
           chapter-level headings reset the tracker to None
        3. Execute: Remove from old container, append to last_numbered.children

        Uses _normalize_heading_text() to strip markdown bold before vocabulary matching.
        """
        # Skip reparenting entirely if no subordinate headings are configured
        if not self._subordinate_headings:
            logger.info("Channel A V4.2.2: No subordinate headings configured — skipping reparenting")
            return root_sections

        # Pass 1: Collect all (section, container) pairs in document order via DFS
        ordered: list[tuple[GuidelineSection, list[GuidelineSection]]] = []
        self._collect_sections_dfs(root_sections, root_sections, ordered)

        # Pass 2: Identify moves
        last_numbered: Optional[GuidelineSection] = None
        moves: list[tuple[GuidelineSection, list[GuidelineSection], GuidelineSection]] = []

        for section, container in ordered:
            normalized = self._normalize_heading_text(section.heading).lower()

            # Chapter-level reset headings
            if normalized in self._chapter_reset_headings:
                last_numbered = None
                continue

            # Numbered section — update tracker
            if self._NUMERIC_SECTION_ID_RE.match(section.section_id):
                last_numbered = section
                continue

            # Subordinate heading — queue for reparenting if we have a numbered parent
            if normalized in self._subordinate_headings and last_numbered is not None:
                moves.append((section, container, last_numbered))

        # Pass 3: Execute moves (reverse order to preserve list indices)
        for section, old_container, new_parent in reversed(moves):
            if section in old_container:
                old_container.remove(section)
                new_parent.children.append(section)
                self._reparent_log.append({
                    "type": "reparent",
                    "section_id": section.section_id,
                    "heading": section.heading,
                    "from_container": "root" if old_container is root_sections else "children",
                    "to_parent_id": new_parent.section_id,
                    "to_parent_heading": new_parent.heading,
                })

        if moves:
            logger.info(
                "Channel A V4.2.2: Reparented %d subordinate heading(s)",
                len(moves),
            )

        return root_sections

    def _collect_sections_dfs(
        self,
        sections: list[GuidelineSection],
        container: list[GuidelineSection],
        result: list[tuple[GuidelineSection, list[GuidelineSection]]],
    ) -> None:
        """DFS walk collecting (section, container_list) pairs in document order."""
        for section in list(sections):  # copy to allow mutation during pass 3
            result.append((section, container))
            if section.children:
                self._collect_sections_dfs(
                    section.children, section.children, result
                )

    def _validate_kdigo_tree(
        self, root_sections: list[GuidelineSection]
    ) -> None:
        """Validate tree structure post-reparenting. Logs warnings, does not raise.

        Invariants:
        1. KEY_INFO_PARENT: No subordinate heading at root without a numbered parent
        2. OFFSET_CONTINUITY: No child's range exceeds its parent's range
        """
        warnings: list[dict] = []

        # Invariant 1: No orphaned subordinate headings at root
        for section in root_sections:
            normalized = self._normalize_heading_text(section.heading).lower()
            if normalized in self._subordinate_headings:
                warning = {
                    "type": "validation_warning",
                    "invariant": "KEY_INFO_PARENT",
                    "section_id": section.section_id,
                    "heading": section.heading,
                    "message": (
                        f"Subordinate heading '{normalized}' remains at root "
                        f"(no preceding numbered section found for reparenting)"
                    ),
                }
                warnings.append(warning)
                logger.warning("Channel A validation: %s", warning["message"])

        # Invariant 2: Offset continuity — children within parent range
        self._check_offset_continuity(root_sections, warnings)

        self._reparent_log.extend(warnings)

    def _check_offset_continuity(
        self,
        sections: list[GuidelineSection],
        warnings: list[dict],
    ) -> None:
        """Recursively check that no child's range exceeds its parent's range."""
        for section in sections:
            for child in section.children:
                if child.start_offset < section.start_offset or child.end_offset > section.end_offset:
                    warning = {
                        "type": "validation_warning",
                        "invariant": "OFFSET_CONTINUITY",
                        "parent_id": section.section_id,
                        "child_id": child.section_id,
                        "message": (
                            f"Child '{child.section_id}' [{child.start_offset}-{child.end_offset}] "
                            f"exceeds parent '{section.section_id}' [{section.start_offset}-{section.end_offset}]"
                        ),
                    }
                    warnings.append(warning)
                    logger.warning("Channel A validation: %s", warning["message"])
            if section.children:
                self._check_offset_continuity(section.children, warnings)

    def classify_block_types(
        self, text: str, tree: GuidelineTree
    ) -> dict[int, str]:
        """Classify every character offset's block type.

        Returns a mapping of offset -> block_type for non-heading content
        within each section. Used by Channel F to filter prose-only blocks.
        """
        block_types: dict[int, str] = {}
        lines = text.split('\n')
        offset = 0

        for line in lines:
            line_stripped = line.strip()

            table = tree.find_table_for_offset(offset)
            if table is not None:
                block_types[offset] = "table_cell"
            elif line_stripped.startswith('- ') or line_stripped.startswith('* '):
                block_types[offset] = "list_item"
            elif line_stripped.startswith('#'):
                block_types[offset] = "heading"
            elif len(line_stripped) > 0:
                block_types[offset] = "paragraph"

            offset += len(line) + 1

        return block_types


# Short alias for pipeline imports (backward compatible)
ChannelA = ChannelAStructuralOracle
