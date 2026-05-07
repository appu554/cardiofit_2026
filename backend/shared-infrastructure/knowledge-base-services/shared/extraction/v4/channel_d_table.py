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
    FieldProvenance,
    _normalise_bbox,
    _normalise_confidence,
    _normalise_page_number,
)
from .specialists import (
    SpecialistError,
    SpecialistTimeoutError,
    SpecialistUnavailableError,
    TableSpecialistResult,
)
from .v5_flags import is_v5_enabled


def _full_page_bbox(pdf_path: str, page_number: int) -> Optional[tuple[float, float, float, float]]:
    """Return the full-page bbox in PDF points for ``page_number`` (1-indexed).

    Used as a fallback for the nemotron lane when Channel A's TableBoundary
    lacks a per-table bbox (granite_otsl path currently drops geometry).
    Coarser than a per-table crop but lets Nemotron Parse run; the model can
    locate the table inside the page on its own.

    Returns ``None`` on any failure (PyMuPDF missing, page out of range,
    invalid path) — caller treats that as "skip nemotron, fall through".

    Lazy-imports PyMuPDF for the same reason ``_render_table_crop`` does:
    keep the module-level import graph clean for tests.
    """
    try:
        import fitz  # PyMuPDF
        doc = fitz.open(str(pdf_path))
        try:
            if page_number < 1 or page_number > doc.page_count:
                return None
            page = doc[page_number - 1]
            rect = page.rect
            return (float(rect.x0), float(rect.y0), float(rect.x1), float(rect.y1))
        finally:
            doc.close()
    except Exception:  # noqa: BLE001 — fallback path; never escalate
        return None


class _BoundaryShim:
    """Minimal adapter exposing the attributes ``_decompose_nemotron_parse_table``
    reads from a TableBlock, when the source is actually a TableBoundary.

    ``_decompose_nemotron_parse_table`` reads ``tb.table_index`` and
    ``tb.region_type`` via ``getattr(..., default)``, so the shim only needs to
    surface a stable ``table_index`` (we use the boundary's ``table_id`` hash so
    dedup keys stay deterministic across runs) and a ``region_type`` of
    ``"table"``. Everything else flows through the TableSpecialistResult.

    Defined as a plain class — no dataclass — to avoid pulling in
    field-validation costs when this object is created per-table at runtime.
    """
    __slots__ = ("table_index", "region_type", "page_number")

    def __init__(self, boundary):
        # Use the table_id's stable hash so two runs on the same input produce
        # the same table_index (otherwise dedup keys downstream would churn).
        self.table_index = abs(hash(getattr(boundary, "table_id", ""))) & 0xFFFFFFFF
        self.region_type = "table"
        self.page_number = boundary.page_number


def _bbox_to_tuple(bbox) -> Optional[tuple[float, float, float, float]]:
    """Coerce the various bbox representations used in the pipeline to a 4-tuple.

    ``TableBlock.bbox`` may be a ``BoundingBox`` dataclass (with x0/y0/x1/y1
    attributes), a 4-element list/tuple, or ``None``. Nemotron Parse needs the
    flat 4-tuple form. Returns ``None`` for any shape we can't interpret —
    callers treat that as "skip this lane for this table".
    """
    if bbox is None:
        return None
    if hasattr(bbox, "x0") and hasattr(bbox, "y0"):
        try:
            return (float(bbox.x0), float(bbox.y0), float(bbox.x1), float(bbox.y1))
        except (TypeError, AttributeError):
            return None
    if isinstance(bbox, (list, tuple)) and len(bbox) == 4:
        try:
            return tuple(float(v) for v in bbox)  # type: ignore[return-value]
        except (TypeError, ValueError):
            return None
    return None


def _channel_d_model_version(table_source: Optional[str] = None) -> str:
    """Channel D model version. Differentiates lane paths in audit provenance.

    Each lane writes a distinct ``model_version`` into the per-channel
    provenance so KB-0's audit dashboard can filter by extraction source.
    Without distinct tags, V5 nemotron-derived cells silently mapped to
    the generic ``table@v1.0`` and were indistinguishable from V4 OTSL
    output in the merged spans (post-remediation audit on job bfb11d94).
    """
    if table_source == "marker_pipe":
        return "pipe-table@v1.0"
    if table_source == "granite_otsl":
        return "docling-otsl@v1.0"
    if table_source == "monkeyocr_vlm":
        return "monkeyocr-qwen2.5-vl@v1.0"
    if table_source == "nemotron_parse":
        return "nvidia/NVIDIA-Nemotron-Parse-v1.1-TC@sidecar"
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

    VERSION = "4.4.0"
    CONFIDENCE_PIPE = 0.95          # Marker pipe tables
    CONFIDENCE_OTSL = 0.92          # Granite OTSL tables (slightly lower — alignment uncertainty)
    CONFIDENCE_DOCLING = 0.97       # Docling TableFormer (structured table objects, V5 #1)
    CONFIDENCE_VLM = 0.98           # MonkeyOCR Qwen2.5-VL per-cell OCR (V5 vlm_table_specialist)
    CONFIDENCE_NEMOTRON_PARSE = 0.99  # Nemotron Parse 1.1 — purpose-built table specialist (V5 nemotron_parse)

    # Lazy-created Nemotron Parse client. Cached on the instance so a 100-table
    # document only constructs one HTTP session, not one per call.
    _nemotron_parse_client = None

    def extract(
        self,
        text: str,
        tree: GuidelineTree,
        l1_tables: Optional[list] = None,
        profile=None,
        pdf_path: Optional[str] = None,
    ) -> ChannelOutput:
        """Decompose all tables in the tree into cell-level RawSpans.

        Args:
            text: Normalized text (Channel 0 output)
            tree: GuidelineTree from Channel A (contains TableBoundary objects)
            l1_tables: Optional list of Docling TableBlock objects (V5 path)
            profile: Optional V5 feature flag profile
            pdf_path: Optional path to source PDF — required only for the
                Nemotron Parse lane, which crops table regions for VLM
                inference. Older callers can omit it; the nemotron lane is
                skipped silently when missing.

        Returns:
            ChannelOutput with one RawSpan per table cell
        """
        start_time = time.monotonic()
        spans: list[RawSpan] = []
        tables_pipe = 0
        tables_otsl = 0
        tables_docling = 0
        tables_nemotron = 0
        suspicious_tables = 0
        table_specialist_used = False

        # V5 #1 Table Specialist priority chain (per-table):
        #   nemotron_parse  (cloud OR sidecar — requires bbox + (NVIDIA_API_KEY|NEMOTRON_PARSE_URL))
        #     ↓ on Unavailable / Timeout / Error
        #   vlm_table_specialist  (MonkeyOCR Qwen2.5-VL cell_data — only when l1_tables has it)
        #     ↓ on no cell_data
        #   docling  (Docling TableFormer, text-only, no per-cell bbox)
        #     ↓ on no l1_tables for this region
        #   V4 OTSL/pipe  (final fallback for tree.tables only)
        #
        # The V5 chain runs as long as ``table_specialist`` is on AND we have
        # SOMETHING to dispatch — either l1_tables (richer, has cell_data) or
        # tree.tables with bboxes (lighter, header-only). In an earlier
        # iteration the gate was ``if l1_tables`` only, which silently bypassed
        # the nemotron lane on PDFs where MonkeyOCR didn't classify anything as
        # a table but Channel A's Granite-Docling did. The widened gate here
        # ensures Nemotron Parse gets invoked on every routable table.
        v5_table_specialist = is_v5_enabled("table_specialist", profile)
        nemotron_lane_enabled = (
            v5_table_specialist
            and is_v5_enabled("nemotron_parse", profile)
            and pdf_path is not None
        )
        has_l1_tables = bool(l1_tables)
        # Any tree.tables (with or without bbox) qualifies — the nemotron lane
        # has a full-page fallback for tables that lack per-table geometry,
        # so we no longer require ``provenance.bbox`` on tree.tables to enter
        # the V5 chain.
        has_tree_tables = bool(tree.tables)
        v5_chain_will_run = v5_table_specialist and (has_l1_tables or has_tree_tables)

        if v5_chain_will_run:
            table_specialist_used = True

            # Pass 1: l1_tables (Docling TableBlock objects, may have cell_data)
            if has_l1_tables:
                for tb in l1_tables:
                    lane_spans = None
                    if nemotron_lane_enabled and getattr(tb, "bbox", None) is not None:
                        lane_spans = self._try_nemotron_parse_lane(
                            tb, pdf_path=pdf_path, profile=profile
                        )
                    if lane_spans is not None:
                        tables_nemotron += 1
                        spans.extend(lane_spans)
                        continue
                    if getattr(tb, "cell_data", None) and is_v5_enabled("vlm_table_specialist", profile):
                        # MonkeyOCR Qwen2.5-VL path — per-cell bbox, FieldProvenance populated
                        docling_spans = self._decompose_vlm_structured_table(tb, profile=profile)
                    else:
                        # Docling TableFormer fallback (text-only, no per-cell bbox)
                        docling_spans = self._decompose_docling_table(tb)
                    tables_docling += 1
                    spans.extend(docling_spans)

            # Pass 2: tree.tables not covered by l1_tables. Try nemotron first
            # (when bbox available via Channel A's TableBoundary.provenance), then
            # the V4 OTSL/pipe paths as a final fallback. This is the path that
            # fires when MonkeyOCR L1 doesn't detect tables but Channel A does.
            covered_indices = {
                idx for idx in
                (getattr(tb, "table_index", None) for tb in (l1_tables or []))
                if idx is not None
            }
            for table in tree.tables:
                if getattr(table, "table_index", None) in covered_indices:
                    continue

                lane_spans = None
                if nemotron_lane_enabled:
                    lane_spans = self._try_nemotron_parse_lane_for_boundary(
                        table, pdf_path=pdf_path, profile=profile
                    )
                if lane_spans is not None:
                    tables_nemotron += 1
                    spans.extend(lane_spans)
                    continue

                # Final V4 fallback
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
            # V4 path: unchanged (no V5 specialist, no l1_tables — nothing to route)
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
                "tables_processed": tables_docling + tables_pipe + tables_otsl + tables_nemotron,
                "tables_pipe": tables_pipe,
                "tables_otsl": tables_otsl,
                "tables_docling": tables_docling,
                "tables_nemotron_parse": tables_nemotron,
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

    # ═══════════════════════════════════════════════════════════════════════
    # V5 VLM TABLE SPECIALIST: MonkeyOCR Qwen2.5-VL per-cell path
    # ═══════════════════════════════════════════════════════════════════════

    # ═══════════════════════════════════════════════════════════════════════
    # V5 NEMOTRON PARSE 1.1: cloud table specialist (highest priority lane)
    # ═══════════════════════════════════════════════════════════════════════

    def _try_nemotron_parse_lane(
        self,
        tb,
        pdf_path: str,
        profile=None,
    ) -> Optional[list[RawSpan]]:
        """Attempt Nemotron Parse and return spans on success, ``None`` on fall-through.

        Wraps the lane in fault isolation: any specialist exception is logged
        and turns into a ``None`` return so the caller drops to the next lane
        without aborting the whole table batch. We intentionally do NOT
        re-raise — a single bad table must not take down the document.

        ``SpecialistUnavailableError`` is logged at debug because it's the
        expected state when ``NVIDIA_API_KEY`` is unset; other failures are
        logged at warning so they show up in pipeline logs without spamming.
        """
        try:
            client = self._get_nemotron_parse_client()
            bbox = tb.bbox
            # tb.bbox can be a BoundingBox dataclass (marker_extractor) with
            # .x0/.y0/.x1/.y1 attrs OR a 4-list. Normalise to a tuple.
            bbox_tuple = _bbox_to_tuple(bbox)
            if bbox_tuple is None:
                return None
            result = client.extract_table(
                pdf_path=pdf_path,
                page_number=int(tb.page_number),
                table_bbox_pts=bbox_tuple,
            )
        except SpecialistUnavailableError as e:
            # Expected when the lane is gated off — quiet log, fall through.
            import logging
            logging.getLogger(__name__).debug(
                "channel_d: nemotron_parse unavailable, falling through: %s", e
            )
            return None
        except SpecialistTimeoutError as e:
            import logging
            logging.getLogger(__name__).warning(
                "channel_d: nemotron_parse timeout on page=%s — falling through: %s",
                tb.page_number, e,
            )
            return None
        except SpecialistError as e:
            import logging
            logging.getLogger(__name__).warning(
                "channel_d: nemotron_parse failed on page=%s — falling through: %s",
                tb.page_number, e,
            )
            return None

        if result.is_empty:
            return None

        return self._decompose_nemotron_parse_table(result, tb, profile=profile)

    def _try_nemotron_parse_lane_for_boundary(
        self,
        table: "TableBoundary",
        pdf_path: str,
        profile=None,
    ) -> Optional[list[RawSpan]]:
        """Run Nemotron Parse on a Channel-A ``TableBoundary`` (no l1_tables).

        Same fault-isolation contract as ``_try_nemotron_parse_lane`` — returns
        ``None`` on any specialist exception so the caller can fall through.

        The bbox source is ``table.provenance.bbox`` (a ``BoundingBox`` with
        x0/y0/x1/y1) instead of ``TableBlock.bbox``. We require the provenance
        to be populated; without bbox we have nothing to crop and the lane
        cannot proceed (returns ``None``, falls back to OTSL/pipe).

        Cells are converted to RawSpans via the same ``_decompose_nemotron_parse_table``
        path used for l1_tables, so the audit trail (model_version,
        FieldProvenance, channel_metadata.table_source="nemotron_parse") is
        identical regardless of which lane fed the dispatch.
        """
        prov = getattr(table, "provenance", None)
        bbox = getattr(prov, "bbox", None) if prov is not None else None

        # Full-page fallback: Channel A's granite_otsl path doesn't capture
        # per-table bbox today (DocTagTable has no geometry field — see
        # granite_docling_extractor.DocTagTable). Rather than skipping the
        # nemotron lane on every OTSL table, we fall back to cropping the
        # whole page and letting Nemotron Parse v1.1 detect the table itself.
        # The model is purpose-built for that workload, so the precision cost
        # of a coarser crop is small. The proper upstream fix is to capture
        # bbox in DocTagTable; tracked as a follow-up.
        if bbox is None:
            bbox_tuple = _full_page_bbox(pdf_path, table.page_number)
        else:
            # ChannelProvenance.bbox is a BoundingBox dataclass; flatten to tuple.
            bbox_tuple = _bbox_to_tuple(bbox)
        if bbox_tuple is None:
            return None

        try:
            client = self._get_nemotron_parse_client()
            result = client.extract_table(
                pdf_path=pdf_path,
                page_number=int(table.page_number),
                table_bbox_pts=bbox_tuple,
            )
        except SpecialistUnavailableError as e:
            import logging
            logging.getLogger(__name__).debug(
                "channel_d: nemotron_parse unavailable for tree.table page=%s: %s",
                table.page_number, e,
            )
            return None
        except SpecialistTimeoutError as e:
            import logging
            logging.getLogger(__name__).warning(
                "channel_d: nemotron_parse timeout on tree.table page=%s — falling through: %s",
                table.page_number, e,
            )
            return None
        except SpecialistError as e:
            import logging
            logging.getLogger(__name__).warning(
                "channel_d: nemotron_parse failed on tree.table page=%s — falling through: %s",
                table.page_number, e,
            )
            return None

        if result.is_empty:
            return None

        # Reuse the l1_tables decomposer — it only reads .table_index
        # and .region_type via getattr, which TableBoundary either supplies
        # (table_index doesn't exist on TableBoundary; we synthesise one).
        # Wrap the boundary in a minimal shim so the existing helper works
        # without a second parallel implementation drift risk.
        shim = _BoundaryShim(table)
        return self._decompose_nemotron_parse_table(result, shim, profile=profile)

    def _get_nemotron_parse_client(self):
        """Lazy-create and cache the NIM client.

        Cached on the instance — one HTTP session for the whole document.
        Created lazily so unit tests that never reach this lane don't pay
        the import cost.
        """
        if self._nemotron_parse_client is None:
            # Local import — avoids requiring ``requests`` at module import
            # time for the (common) case where the lane is disabled.
            from .specialists.nemotron_parse import NemotronParseTableSpecialist
            self._nemotron_parse_client = NemotronParseTableSpecialist()
        return self._nemotron_parse_client

    def _decompose_nemotron_parse_table(
        self,
        result: TableSpecialistResult,
        tb,
        profile=None,
    ) -> list[RawSpan]:
        """Convert a ``TableSpecialistResult`` into one RawSpan per body cell.

        Header cells go into ``headers`` (used downstream as ``col_header``
        on each body span), not into spans of their own — same convention as
        ``_decompose_vlm_structured_table``. This keeps the cell-count metric
        ("informational rows") comparable across lanes.

        FieldProvenance is populated when ``bbox_provenance`` is on AND the
        specialist returned a per-cell bbox. The model_version on each
        FieldProvenance carries ``nemo-document-parse-1.1@nim`` so audits can
        distinguish nemotron cells from MonkeyOCR cells in the merged store.
        """
        spans: list[RawSpan] = []

        # Build the header row → headers list.
        # Sorted by col_idx to defend against unordered specialist output.
        header_cells = sorted(
            (c for c in result.cells if c.is_header or c.row_idx == 0),
            key=lambda c: c.col_idx,
        )
        headers: list[str] = []
        if header_cells:
            max_col = max(c.col_idx for c in header_cells)
            headers = [""] * (max_col + 1)
            for hc in header_cells:
                if 0 <= hc.col_idx < len(headers):
                    headers[hc.col_idx] = hc.text.strip()
        has_header_row = bool(headers)

        for cell in result.cells:
            if has_header_row and (cell.is_header or cell.row_idx == 0):
                continue
            cell_text = cell.text.strip()
            if not cell_text:
                continue

            data_row_idx = (cell.row_idx - 1) if has_header_row else cell.row_idx
            col_header = headers[cell.col_idx] if cell.col_idx < len(headers) else None
            confidence = cell.confidence if cell.confidence > 0 else self.CONFIDENCE_NEMOTRON_PARSE

            fp_list: list[FieldProvenance] = []
            if is_v5_enabled("bbox_provenance", profile) and cell.bbox is not None:
                bb = _normalise_bbox(list(cell.bbox))
                if bb is not None:
                    fp_list.append(FieldProvenance(
                        field_name="table_cell",
                        value=cell_text,
                        bbox=bb,
                        page_number=_normalise_page_number(result.page_number),
                        confidence=_normalise_confidence(confidence),
                        channel_id="D",
                    ))

            spans.append(RawSpan(
                channel="D",
                text=cell_text,
                start=-1,
                end=-1,
                confidence=confidence,
                page_number=result.page_number,
                source_block_type="table_cell",
                bbox=list(cell.bbox) if cell.bbox else None,
                field_provenance=fp_list,
                channel_metadata={
                    "row_index": data_row_idx,
                    "col_index": cell.col_idx,
                    "col_header": col_header,
                    "table_source": "nemotron_parse",
                    "table_index": getattr(tb, "table_index", 0),
                    "region_type": getattr(tb, "region_type", "table"),
                    "model_version": result.model_version,
                },
            ))

        return spans

    def _decompose_vlm_structured_table(self, tb, profile=None) -> list[RawSpan]:
        """Decompose a MonkeyOCR VLM-structured table with per-cell bbox.

        Uses ``tb.cell_data`` — per-cell records captured from middle.json's
        Qwen2.5-VL span output — to produce one RawSpan per cell.  Each span
        carries a ``FieldProvenance`` entry so the exact PDF region for the
        extracted value is traceable (Feature #2 per-fact bbox).

        Falls through to ``_decompose_docling_table`` implicitly if cell_data
        is empty (caller already guards with ``getattr(tb, 'cell_data', None)``).
        """
        spans: list[RawSpan] = []
        headers = tb.headers or []

        # Infer header row: cells with row_idx == 0 supply the column header names
        # when tb.headers is empty (MonkeyOCR doesn't always split header vs data).
        if not headers:
            headers = [
                c.get("text", "")
                for c in (tb.cell_data or [])
                if c.get("row_idx", 0) == 0
            ]

        # Determine whether row 0 is a header row (skip it as a data span)
        has_header_row = bool(headers)

        for cell in (tb.cell_data or []):
            cell_text = cell.get("text", "").strip()
            if not cell_text:
                continue

            row_idx = cell.get("row_idx", 0)
            col_idx = cell.get("col_idx", 0)

            # Skip header row — captured in `headers`, not a clinical value span
            if has_header_row and row_idx == 0:
                continue

            data_row_idx = (row_idx - 1) if has_header_row else row_idx
            col_header = headers[col_idx] if col_idx < len(headers) else None
            cell_bbox_raw = cell.get("bbox")
            confidence = float(cell.get("confidence", self.CONFIDENCE_VLM))

            # Build per-cell FieldProvenance for Feature #2 sub-span bbox
            fp_list: list[FieldProvenance] = []
            if is_v5_enabled("bbox_provenance", profile) and cell_bbox_raw:
                bb = _normalise_bbox(cell_bbox_raw)
                if bb is not None:
                    fp_list.append(FieldProvenance(
                        field_name="table_cell",
                        value=cell_text,
                        bbox=bb,
                        page_number=_normalise_page_number(tb.page_number),
                        confidence=_normalise_confidence(confidence),
                        channel_id="D",
                    ))

            spans.append(RawSpan(
                channel="D",
                text=cell_text,
                start=-1,
                end=-1,
                confidence=confidence,
                page_number=tb.page_number,
                source_block_type="table_cell",
                bbox=cell_bbox_raw,
                field_provenance=fp_list,
                channel_metadata={
                    "row_index": data_row_idx,
                    "col_index": col_idx,
                    "col_header": col_header,
                    "table_source": "monkeyocr_vlm",
                    "table_index": getattr(tb, "table_index", 0),
                    "region_type": getattr(tb, "region_type", "table"),
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
