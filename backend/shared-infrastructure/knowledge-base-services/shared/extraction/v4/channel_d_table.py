"""
Channel D: Table Decomposer (V4.1 Dual-Source).

Consumes TableBoundary objects from Channel A and decomposes each
table into one RawSpan per cell, preserving verbatim provenance.

V4.1: Tables can come from two sources:
- "marker_pipe": Marker markdown pipe tables → split by | delimiters (V4 behavior)
- "granite_otsl": Granite-Docling OTSL tables → parse <ched>/<fcel>/<ecel>/<nl> tags

Each cell becomes a RawSpan with table-aware metadata including:
- row_index, col_index
- col_header (from the header row)
- row_drug (from column 0 of the same row, for drug-signal association)
- table_source (for V4.1 provenance tracking)

Pipeline Position:
    Channel A (GuidelineTree) -> Channel D (THIS, parallel with B/C/E/F)
    Requires: Channel A tables (TableBoundary objects)
"""

from __future__ import annotations

import re
import time
from typing import Optional

from .models import ChannelOutput, GuidelineTree, RawSpan, TableBoundary
from .provenance import (
    ChannelProvenance,
    _normalise_bbox,
    _normalise_confidence,
    _normalise_page_number,
)
from .v5_flags import is_v5_enabled


def _channel_d_model_version(table_source: Optional[str] = None) -> str:
    """Channel D model version. Differentiates docling-OTSL vs marker-pipe paths.

    Mirrors Channel A's both-saw vs marker-only branching: tables sourced from
    Granite-Docling OTSL get one tag; Marker-only pipe tables get another.
    """
    if table_source == "marker_pipe":
        return "pipe-table@v1.0"
    if table_source == "granite_otsl":
        return "docling-otsl@v1.0"
    return "table@v1.0"


def _channel_d_provenance(
    bbox,
    page_number,
    confidence,
    profile,
    notes: Optional[str] = None,
    table_source: Optional[str] = None,
) -> Optional[ChannelProvenance]:
    """Build a ChannelProvenance entry for Channel D (table decomposer).

    Returns None when V5_BBOX_PROVENANCE is off or bbox is missing. The
    ``table_source`` argument selects between docling-OTSL and marker-pipe
    model_version tags.
    """
    if not is_v5_enabled("bbox_provenance", profile):
        return None
    bb = _normalise_bbox(bbox)
    if bb is None:
        return None
    return ChannelProvenance(
        channel_id="D",
        bbox=bb,
        page_number=_normalise_page_number(page_number),
        confidence=_normalise_confidence(confidence),
        model_version=_channel_d_model_version(table_source),
        notes=notes,
    )


class ChannelDTableDecomposer:
    """Decompose tables from either pipe format or OTSL format.

    Also exported as ``ChannelD`` for pipeline convenience.

    V4.1: Dual-source table decomposition.
    - marker_pipe: text[table.start:table.end] → split by | delimiters
    - granite_otsl: table.otsl_text → parse <ched>/<rhed>/<fcel>/<ecel>/<lcel> tags
    """

    VERSION = "4.2.0"
    CONFIDENCE_PIPE = 0.95    # Marker pipe tables
    CONFIDENCE_OTSL = 0.92    # Granite OTSL tables (slightly lower — alignment uncertainty)

    def extract(
        self,
        text: str,
        tree: GuidelineTree,
        l1_tables: Optional[list] = None,
        profile=None,
    ) -> ChannelOutput:
        """Decompose all tables in the tree into cell-level RawSpans.

        Args:
            text: Normalized text (Channel 0 output)
            tree: GuidelineTree from Channel A (contains TableBoundary objects)
            l1_tables: Optional list of Docling TableBlock objects (V5 path)
            profile: Optional V5 feature flag profile

        Returns:
            ChannelOutput with one RawSpan per table cell
        """
        start_time = time.monotonic()
        spans: list[RawSpan] = []
        tables_pipe = 0
        tables_otsl = 0
        tables_docling = 0
        suspicious_tables = 0
        table_specialist_used = False

        # V5 #1 Table Specialist: use Docling TableBlock as primary when flag on
        if l1_tables and is_v5_enabled("table_specialist", profile):
            table_specialist_used = True
            for tb in l1_tables:
                docling_spans = self._decompose_docling_table(tb)
                tables_docling += 1
                spans.extend(docling_spans)
            # Still run OTSL/pipe for any tree.tables NOT covered by l1_tables
            covered_indices = {getattr(tb, "table_index", None) for tb in l1_tables}
            for table in tree.tables:
                if getattr(table, "table_index", None) in covered_indices:
                    continue
                if table.source == "granite_otsl" and table.otsl_text:
                    table_spans = self._decompose_otsl_table(table)
                    tables_otsl += 1
                else:
                    table_spans = self._decompose_pipe_table(text, table, tree)
                    tables_pipe += 1
                if self._is_suspicious(table, table_spans):
                    suspicious_tables += 1
                spans.extend(table_spans)
        else:
            # V4 path: unchanged
            for table in tree.tables:
                if table.source == "granite_otsl" and table.otsl_text:
                    table_spans = self._decompose_otsl_table(table)
                    tables_otsl += 1
                else:
                    table_spans = self._decompose_pipe_table(text, table, tree)
                    tables_pipe += 1
                if self._is_suspicious(table, table_spans):
                    suspicious_tables += 1
                spans.extend(table_spans)

        # V4.2: Extract footnote definitions from figure/table captions
        footnote_spans = self._extract_caption_footnotes(text, tree)
        spans.extend(footnote_spans)

        elapsed_ms = (time.monotonic() - start_time) * 1000

        return ChannelOutput(
            channel="D",
            spans=spans,
            metadata={
                "tables_processed": len(tree.tables) + tables_docling,
                "tables_pipe": tables_pipe,
                "tables_otsl": tables_otsl,
                "tables_docling": tables_docling,
                "cells_extracted": len(spans) - len(footnote_spans),
                "caption_footnotes": len(footnote_spans),
                "suspicious_tables": suspicious_tables,
                "table_specialist_used": table_specialist_used,
            },
            elapsed_ms=elapsed_ms,
        )

    # ═══════════════════════════════════════════════════════════════════════
    # PIPE TABLE DECOMPOSER (V4 behavior, preserved)
    # ═══════════════════════════════════════════════════════════════════════

    def _decompose_pipe_table(
        self, text: str, table: TableBoundary, tree: GuidelineTree = None,
    ) -> list[RawSpan]:
        """Decompose a markdown pipe table into cell-level RawSpans."""
        if table.start_offset < 0 or table.end_offset < 0:
            return []  # OTSL table with no Marker offset — skip pipe path

        table_text = text[table.start_offset:table.end_offset]
        lines = table_text.split('\n')

        spans: list[RawSpan] = []
        headers: list[str] = []
        data_rows: list[tuple[int, list[str]]] = []
        line_offsets: list[int] = []

        # Calculate line start offsets within the full text
        offset = table.start_offset
        for line in lines:
            line_offsets.append(offset)
            offset += len(line) + 1

        # Parse table rows
        header_parsed = False
        separator_seen = False

        for i, line in enumerate(lines):
            stripped = line.strip()
            if not (stripped.startswith('|') and stripped.endswith('|')):
                continue

            cells = [c.strip() for c in stripped.split('|')[1:-1]]

            if all(self._is_separator_cell(c) for c in cells):
                separator_seen = True
                continue

            if not header_parsed and not separator_seen:
                headers = cells
                header_parsed = True
                continue

            if separator_seen:
                data_rows.append((i, cells))

        # Emit one RawSpan per data cell
        for row_idx, (line_idx, cells) in enumerate(data_rows):
            row_drug = cells[0] if cells else None

            for col_idx, cell_text in enumerate(cells):
                if not cell_text.strip():
                    continue

                line_text = lines[line_idx]
                cell_offset = self._find_cell_offset(
                    line_text, cell_text, col_idx
                )
                real_start = line_offsets[line_idx] + cell_offset
                real_end = real_start + len(cell_text)

                col_header = headers[col_idx] if col_idx < len(headers) else None

                # V4.2.2: direct offset lookup for correct page
                cell_page = tree.get_page_for_offset(real_start) if tree and tree.page_map else table.page_number
                spans.append(RawSpan(
                    channel="D",
                    text=cell_text,
                    start=real_start,
                    end=real_end,
                    confidence=self.CONFIDENCE_PIPE,
                    page_number=cell_page,
                    section_id=table.section_id,
                    table_id=table.table_id,
                    source_block_type="table_cell",
                    channel_metadata={
                        "row_index": row_idx,
                        "col_index": col_idx,
                        "col_header": col_header,
                        "row_drug": row_drug,
                        "table_source": "marker_pipe",
                    },
                ))

        return spans

    # ═══════════════════════════════════════════════════════════════════════
    # OTSL TABLE DECOMPOSER (V4.1 NEW)
    # ═══════════════════════════════════════════════════════════════════════

    def _decompose_otsl_table(self, table: TableBoundary) -> list[RawSpan]:
        """Decompose OTSL table from Granite-Docling DocTags.

        OTSL tokens:
        - <ched>: column header cell
        - <rhed>: row header cell
        - <fcel>: filled cell (has content)
        - <ecel>: empty cell
        - <lcel>: left-merged cell (spans from previous column)
        - <nl>: row delimiter

        Each non-empty cell becomes one RawSpan.
        """
        otsl_text = table.otsl_text
        if not otsl_text:
            return []

        rows = otsl_text.split("<nl>")
        headers: list[str] = []
        spans: list[RawSpan] = []

        for row_idx, row in enumerate(rows):
            if not row.strip():
                continue

            # Extract cells with their types
            cells = re.findall(
                r'<(ched|rhed|fcel|ecel|lcel)>(.*?)</\1>', row
            )

            if not cells:
                continue

            if row_idx == 0:
                # Header row — extract column headers from <ched> tags
                headers = [
                    cell_text.strip()
                    for cell_type, cell_text in cells
                    if cell_type == "ched"
                ]
                # Also emit header cells as spans for completeness
                for col_idx, (cell_type, cell_text) in enumerate(cells):
                    if cell_type != "ched" or not cell_text.strip():
                        continue
                    spans.append(self._otsl_cell_to_span(
                        table, cell_text.strip(), row_idx, col_idx,
                        cell_type, headers, cells,
                    ))
                continue

            # Data rows
            for col_idx, (cell_type, cell_text) in enumerate(cells):
                if cell_type in ("ecel", "lcel"):
                    continue  # skip empty and merged cells

                cell_text = cell_text.strip()
                if not cell_text:
                    continue

                spans.append(self._otsl_cell_to_span(
                    table, cell_text, row_idx, col_idx,
                    cell_type, headers, cells,
                ))

        return spans

    def _otsl_cell_to_span(
        self,
        table: TableBoundary,
        cell_text: str,
        row_idx: int,
        col_idx: int,
        cell_type: str,
        headers: list[str],
        row_cells: list[tuple[str, str]],
    ) -> RawSpan:
        """Convert an OTSL cell into a RawSpan."""
        col_header = headers[col_idx] if col_idx < len(headers) else ""
        row_drug = self._get_otsl_row_drug(row_cells)

        return RawSpan(
            channel="D",
            text=cell_text,
            start=-1,              # not in Marker text space
            end=-1,
            confidence=self.CONFIDENCE_OTSL,
            page_number=table.page_number,
            section_id=table.section_id,
            table_id=table.table_id,
            source_block_type="table_cell",
            channel_metadata={
                "row_index": row_idx,
                "col_index": col_idx,
                "col_header": col_header,
                "row_drug": row_drug,
                "table_source": "granite_otsl",
                "cell_type": cell_type,
            },
        )

    # ═══════════════════════════════════════════════════════════════════════
    # V5 #1 TABLE SPECIALIST: Docling TableBlock path
    # ═══════════════════════════════════════════════════════════════════════

    CONFIDENCE_DOCLING = 0.97

    def _decompose_docling_table(self, tb) -> list[RawSpan]:
        """Decompose a Docling TableBlock (from marker_extractor) into cell RawSpans.

        TableBlock.headers: list[str] — column headers
        TableBlock.rows: list[list[str]] — data rows
        Each non-empty cell becomes one RawSpan with row/col/header metadata.
        """
        spans: list[RawSpan] = []
        headers = tb.headers or []
        bbox_raw = None
        if tb.bbox is not None:
            try:
                bbox_raw = [
                    float(tb.bbox.x0),
                    float(tb.bbox.y0),
                    float(tb.bbox.x1),
                    float(tb.bbox.y1),
                ]
            except (AttributeError, TypeError):
                bbox_raw = None

        for row_idx, row in enumerate(tb.rows):
            row_drug = None
            for cell in row:
                if cell and cell.strip():
                    row_drug = cell.strip()
                    break

            for col_idx, cell_text in enumerate(row):
                if not cell_text or not cell_text.strip():
                    continue

                col_header = headers[col_idx] if col_idx < len(headers) else None

                spans.append(RawSpan(
                    channel="D",
                    text=cell_text.strip(),
                    start=-1,
                    end=-1,
                    confidence=self.CONFIDENCE_DOCLING,
                    page_number=tb.page_number,
                    source_block_type="table_cell",
                    bbox=bbox_raw,
                    channel_metadata={
                        "row_index": row_idx,
                        "col_index": col_idx,
                        "col_header": col_header,
                        "row_drug": row_drug,
                        "table_source": "docling_tableblock",
                        "table_index": getattr(tb, "table_index", 0),
                    },
                ))

        return spans

    def _get_otsl_row_drug(
        self, row_cells: list[tuple[str, str]]
    ) -> Optional[str]:
        """Get the drug name from column 0 of an OTSL row."""
        for cell_type, cell_text in row_cells:
            if cell_type in ("ecel", "lcel"):
                continue
            text = cell_text.strip()
            if text:
                return text
        return None

    # ═══════════════════════════════════════════════════════════════════════
    # V4.2: CAPTION FOOTNOTE EXTRACTION (Group C — upstream fix)
    #
    # Figure/table captions in KDIGO guidelines contain footnote definitions
    # marked with \*, +, †, ‡, §, ¶.  These carry Tier 1 clinical content
    # (perioperative SGLT2i management, sick day protocols) that must be
    # captured as spans for CoverageGuard A1d footnote binding.
    #
    # Strategy: Find long captions (>200 chars starting with "Figure N |"
    # or "Table N |"), locate footnote markers within them, and extract
    # each definition as a separate span.
    # ═══════════════════════════════════════════════════════════════════════

    # Matches long figure/table caption lines
    _CAPTION_RE = re.compile(
        r'^((?:Figure|Table)\s+\d+\s*\|.{200,})$',
        re.MULTILINE,
    )

    # Unambiguous footnote markers (always mean "footnote")
    _UNAMB_MARKER_RE = re.compile(r'[†‡§¶]')

    # Ambiguous markers (\* or +) — only count when preceded by ". "
    _AMB_MARKER_RE = re.compile(r'(?<=\.)\s*(?:\\\*|\+)')

    # Attribution text that ends the footnote region
    _ATTRIBUTION_RE = re.compile(
        r'\.\s*(?:Adapted|Copyright|Reprinted|Modified|From)\s+(?:from\s+)?[A-Z]'
    )

    def _extract_caption_footnotes(
        self, text: str, tree: GuidelineTree,
    ) -> list[RawSpan]:
        """Extract footnote definitions from figure/table captions.

        Finds long caption lines, locates footnote markers within them,
        and creates one RawSpan per footnote definition.  Each span covers
        the text from one marker to the next (or to the attribution text
        that closes the caption).
        """
        spans: list[RawSpan] = []

        for cap_match in self._CAPTION_RE.finditer(text):
            caption = cap_match.group(1)
            caption_start = cap_match.start()

            # Collect all marker positions (character index AFTER the marker)
            marker_positions: list[int] = []

            for m in self._UNAMB_MARKER_RE.finditer(caption):
                marker_positions.append(m.end())

            for m in self._AMB_MARKER_RE.finditer(caption):
                marker_positions.append(m.end())

            if not marker_positions:
                continue

            marker_positions.sort()

            # Find end of footnote region (attribution text)
            attr_match = self._ATTRIBUTION_RE.search(caption)
            footnote_region_end = (
                attr_match.start() + 1  # include the period
                if attr_match
                else len(caption)
            )

            # Extract text between consecutive markers
            for i, pos in enumerate(marker_positions):
                end = (
                    marker_positions[i + 1]
                    if i + 1 < len(marker_positions)
                    else footnote_region_end
                )

                # Walk backward from the next marker to find the preceding
                # marker character itself, so we don't include it in the text
                if i + 1 < len(marker_positions):
                    # Find the marker char just before marker_positions[i+1]
                    for step_back in range(1, 6):
                        check_pos = marker_positions[i + 1] - step_back
                        if check_pos >= 0 and caption[check_pos] in '†‡§¶*+\\':
                            end = check_pos
                            break

                def_text = caption[pos:end].strip()
                # Remove leading punctuation/whitespace artifacts
                def_text = re.sub(r'^[\s.;,]+', '', def_text)

                if len(def_text) < 20:
                    continue

                real_start = caption_start + pos
                real_end = caption_start + pos + len(def_text)

                section = tree.find_section_for_offset(real_start)
                section_id = section.section_id if section else None
                # V4.2.2: direct offset lookup for correct page
                page = tree.get_page_for_offset(real_start) if tree.page_map else (section.page_number if section else None)

                spans.append(RawSpan(
                    channel="D",
                    text=def_text,
                    start=real_start,
                    end=real_end,
                    confidence=0.90,
                    page_number=page,
                    section_id=section_id,
                    source_block_type="table_footnote",
                    channel_metadata={
                        "table_source": "caption_footnote",
                    },
                ))

        return spans

    # ═══════════════════════════════════════════════════════════════════════
    # QUALITY HEURISTICS
    # ═══════════════════════════════════════════════════════════════════════

    def _is_suspicious(
        self, table: TableBoundary, spans: list[RawSpan]
    ) -> bool:
        """Flag tables where decomposition might have failed.

        Suspicion heuristics:
        - Zero cells extracted from a non-empty table
        - All cells in one column (likely parsing failure)
        - Inconsistent column counts across rows
        """
        if len(spans) == 0 and table.row_count > 0:
            return True

        if not spans:
            return False

        col_indices = set()
        for span in spans:
            col_indices.add(span.channel_metadata.get("col_index", 0))

        # All cells in one column with multiple rows → likely parsing failure
        if len(col_indices) == 1 and table.row_count > 2:
            return True

        return False

    # ═══════════════════════════════════════════════════════════════════════
    # SHARED UTILITIES
    # ═══════════════════════════════════════════════════════════════════════

    def _is_separator_cell(self, cell: str) -> bool:
        """Check if a cell is a separator (e.g., '---', ':---:', '---:')."""
        cleaned = cell.strip().replace('-', '').replace(':', '').replace(' ', '')
        return len(cleaned) == 0 and len(cell.strip()) > 0

    def _find_cell_offset(
        self, line: str, cell_text: str, col_idx: int
    ) -> int:
        """Find the character offset of a cell's text within its table line."""
        pos = 0
        col = -1

        for i, char in enumerate(line):
            if char == '|':
                col += 1
                if col == col_idx + 1:
                    segment = line[pos:i]
                    idx = segment.find(cell_text)
                    if idx >= 0:
                        return pos + idx
                pos = i + 1

        idx = line.find(cell_text)
        return idx if idx >= 0 else 0


# Short alias for pipeline imports
ChannelD = ChannelDTableDecomposer
