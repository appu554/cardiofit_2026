"""
TableBoundaryOracle — Coordinate-based table reconstruction from PDF structure.

Uses PyMuPDF per-page block coordinates to detect and reconstruct tables,
producing structured TableRule spans instead of decontextualized cell fragments.

Solves two problems:
    1. Table CONTENT pages vs REFERENCE pages — "Table 32 |" (content with
       pipe-header) vs "shown in Table 32." (prose mention)
    2. Cell fragment noise — individual cells like "150 mg" get rejected as
       NOISE by the tiering classifier. Structured rules carry full context:
       drug + condition + value + source table.

Four table reconstruction patterns (KDIGO-specific):
    Pattern 1: Pipe-grid tables (Table 32, Figure 44)
        → Simple markdown pipe tables with | delimiters
    Pattern 2: Multi-block column tables (Table 27, Table 31)
        → Drug names in left column, conditions in right column(s)
    Pattern 3: Sub-tables with footnotes (Figure 43)
        → Vertically stacked sub-tables separated by repeated headers
    Pattern 4: Coordinate-grid extraction (generic fallback)
        → x-clustering for columns, y-clustering for rows

Integration:
    - Channel D: augments existing cell spans with structured TableRule spans
    - Signal Merger: handles "table_rule" source_block_type
    - Tiering Classifier: table_rule spans bypass noise gate (Tier 1 floor)

Pipeline Position:
    PDF → TableBoundaryOracle → TableRule spans (structured)
    PDF → Channel A → Channel D → cell spans (fragments)
    Both feed into Signal Merger → merged output
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

try:
    import pymupdf
except ImportError:
    import fitz as pymupdf


# ═══════════════════════════════════════════════════════════════════════════════
# Data Models
# ═══════════════════════════════════════════════════════════════════════════════

@dataclass
class TableRule:
    """A structured rule extracted from a table cell with full context.

    Unlike a raw cell fragment ("150 mg"), a TableRule carries:
    - The drug name (from row header or column 0)
    - The condition (from column header)
    - The value (cell content)
    - The source table ID and page number
    """
    drug: str                          # e.g., "Apixaban"
    condition: str                     # e.g., "eCrCl 31-50"
    value: str                         # e.g., "2.5 mg b.i.d."
    table_id: str                      # e.g., "figure_43" or "table_32"
    page_number: int
    sub_table: Optional[str] = None    # e.g., "a" for Figure 43a (RCT-supported)
    footnote: Optional[str] = None     # Associated footnote text
    confidence: float = 0.95           # High — coordinate-based, not heuristic
    bbox: Optional[tuple] = None       # (x0, y0, x1, y1) of the cell


@dataclass
class TableDetection:
    """Result of table detection on a single page."""
    page_number: int
    table_ids: list[str]               # Table/Figure IDs found as content
    reference_ids: list[str]           # Table/Figure IDs found as references only
    rules: list[TableRule] = field(default_factory=list)
    block_count: int = 0


# ═══════════════════════════════════════════════════════════════════════════════
# Table Boundary Oracle
# ═══════════════════════════════════════════════════════════════════════════════

class TableBoundaryOracle:
    """Coordinate-based table reconstruction from PDF structure.

    Opens the PDF once, scans each page for table content, and produces
    structured TableRule spans that carry full drug+condition+value context.
    """

    VERSION = "1.1.1"

    # Matches "Table N |" or "Figure N |" (content indicator)
    TABLE_CONTENT_RE = re.compile(
        r'((?:Table|Figure)\s+\d+[a-z]?)\s*\|',
        re.IGNORECASE,
    )

    # Matches "Table N" or "Figure N" in prose (reference indicator)
    TABLE_REF_RE = re.compile(
        r'(?:in|see|from|shown\s+in|refer\s+to|listed\s+in)\s+'
        r'((?:Table|Figure)\s+\d+[a-z]?)',
        re.IGNORECASE,
    )

    # eCrCl header pattern for sub-table detection (Figure 43).
    # Anchored to start-of-string so footnote mentions like "(eCrCl)."
    # are excluded — only actual column headers match.
    ECRCL_HEADER_RE = re.compile(r'^e(?:Cr|GFR|CrCl)', re.IGNORECASE)

    def __init__(self, pdf_path: str | Path, page_offset: int = 0):
        self.pdf_path = str(pdf_path)
        self.page_offset = page_offset
        self._page_data: dict[int, dict] = {}  # page_num → {"text": str, "blocks": list}
        self._load_pages()

    def _load_pages(self) -> None:
        """Extract text and block data per page using PyMuPDF."""
        doc = pymupdf.open(self.pdf_path)
        for page_idx in range(len(doc)):
            page = doc[page_idx]
            page_num = page_idx + 1 + self.page_offset
            page_text = page.get_text("text")
            page_dict = page.get_text("dict")

            self._page_data[page_num] = {
                "text": page_text,
                "blocks": page_dict.get("blocks", []),
            }
        doc.close()

    # ─── Public API ───────────────────────────────────────────────────────

    def detect_all_tables(self) -> dict[int, TableDetection]:
        """Scan all pages and produce table detections.

        Returns:
            Dict mapping page_number → TableDetection
        """
        results = {}
        for page_num, data in self._page_data.items():
            detection = self._detect_page_tables(page_num, data)
            if detection.table_ids or detection.reference_ids:
                results[page_num] = detection
        return results

    def extract_all_rules(self) -> list[TableRule]:
        """Extract structured TableRule spans from all detected table pages.

        Returns:
            List of TableRule objects with drug+condition+value context.
        """
        all_rules = []
        detections = self.detect_all_tables()

        for page_num, detection in detections.items():
            if not detection.table_ids:
                continue  # Skip reference-only pages

            data = self._page_data[page_num]
            rules = self._extract_rules_from_page(
                page_num, data, detection.table_ids,
            )
            all_rules.extend(rules)

        return all_rules

    def page_has_table_content(self, page_num: int, table_id: str) -> bool:
        """Check if page has actual table content (not just a reference).

        Args:
            page_num: 1-based page number
            table_id: e.g., "Table 32" or "Figure 44"

        Returns:
            True if page contains table content (pipe-header pattern).
        """
        data = self._page_data.get(page_num)
        if not data:
            return False
        pattern = re.compile(re.escape(table_id) + r'\s*\|', re.IGNORECASE)
        return bool(pattern.search(data["text"]))

    # ─── Detection logic ──────────────────────────────────────────────────

    def _detect_page_tables(
        self, page_num: int, data: dict,
    ) -> TableDetection:
        """Classify a page's table mentions as CONTENT or REFERENCE."""
        text = data["text"]
        blocks = data["blocks"]

        content_ids = []
        reference_ids = []

        # Find table CONTENT (pipe-header pattern)
        for m in self.TABLE_CONTENT_RE.finditer(text):
            table_id = m.group(1).strip()
            if table_id not in content_ids:
                content_ids.append(table_id)

        # Find table REFERENCES (prose mentions)
        for m in self.TABLE_REF_RE.finditer(text):
            table_id = m.group(1).strip()
            if table_id not in reference_ids and table_id not in content_ids:
                reference_ids.append(table_id)

        return TableDetection(
            page_number=page_num,
            table_ids=content_ids,
            reference_ids=reference_ids,
            block_count=len([b for b in blocks if b.get("type") == 0]),
        )

    # ─── Rule extraction (coordinate-based) ───────────────────────────────

    def _extract_rules_from_page(
        self,
        page_num: int,
        data: dict,
        table_ids: list[str],
    ) -> list[TableRule]:
        """Extract structured rules from a table content page.

        Strategy:
        1. Get all text blocks with bounding boxes
        2. Cluster by y-coordinate (rows) and x-coordinate (columns)
        3. Assign drug names from leftmost column
        4. Assign conditions from header row
        5. Produce one TableRule per non-empty cell
        """
        blocks = data["blocks"]
        text_blocks = self._extract_text_blocks(blocks)

        if not text_blocks:
            return []

        # Check for sub-tables (Pattern 3: Figure 43 with stacked eCrCl headers)
        sub_table_splits = self._detect_sub_tables(text_blocks)

        rules = []
        table_id = table_ids[0] if table_ids else f"page_{page_num}"

        if sub_table_splits:
            # Pattern 3: Split into sub-tables and extract each
            for sub_idx, (start_y, end_y, sub_blocks) in enumerate(sub_table_splits):
                sub_label = chr(ord('a') + sub_idx)  # a, b, c, ...
                sub_rules = self._extract_grid_rules(
                    sub_blocks, table_id, page_num, sub_table=sub_label,
                )
                rules.extend(sub_rules)
        else:
            # Pattern 1/2/4: Single table extraction
            rules = self._extract_grid_rules(
                text_blocks, table_id, page_num,
            )

        return rules

    # Minimum x-gap (in PDF points) between adjacent spans to treat them
    # as separate columns.  Figure 43 column gaps range from 17pt
    # (Warfarin→Apixaban) to 45pt (Edoxaban→Rivaroxaban).  Normal
    # intra-word spacing is <5pt.  15pt safely separates all columns
    # without splitting within-cell text.
    COLUMN_GAP_THRESHOLD = 15.0

    # Known drug-column header patterns — when the header row contains
    # drug names and column 0 is a condition (e.g., eCrCl bands), the
    # drug/condition fields need to be swapped.
    CONDITION_ROW_HEADER_RE = re.compile(
        r'eCrCl|eGFR|CrCl|GFR\b|creatinine\s+clearance',
        re.IGNORECASE,
    )

    def _extract_text_blocks(
        self, blocks: list[dict],
    ) -> list[dict]:
        """Extract text cells with coordinates from PyMuPDF span data.

        **V1.1 change — span-level extraction with x-gap column splitting.**

        PyMuPDF "blocks" can merge an entire table row into one element,
        destroying column boundaries.  This method instead:

        1. Iterates every span inside every line inside every block.
        2. Groups consecutive spans that are spatially close on the x-axis
           (gap < COLUMN_GAP_THRESHOLD) into a single "cell".
        3. When a gap exceeding the threshold is found, the accumulated
           spans are flushed as one cell and a new cell begins.

        This correctly separates "2.5 mg PO b.i.d." (Apixaban column,
        x≈227) from "Unknown" (Dabigatran column, x≈317) even though
        PyMuPDF groups them into the same block.

        Returns list of {text, x0, y0, x1, y1, font_size, is_bold}.
        """
        # Collect every span with its coordinates
        raw_spans: list[dict] = []
        for block in blocks:
            if block.get("type") != 0:
                continue
            for line in block.get("lines", []):
                for span in line.get("spans", []):
                    text = span.get("text", "").strip()
                    if not text:
                        continue
                    sbbox = span.get("bbox", (0, 0, 0, 0))
                    raw_spans.append({
                        "text": text,
                        "x0": sbbox[0],
                        "y0": sbbox[1],
                        "x1": sbbox[2],
                        "y1": sbbox[3],
                        "font_size": span.get("size", 0),
                        "is_bold": bool(span.get("flags", 0) & 16),
                    })

        if not raw_spans:
            return []

        # Sort spans top-to-bottom then left-to-right.
        # CRITICAL: Round y to nearest integer before sorting.
        # PyMuPDF reports y-coordinates as floats that can differ by
        # fractions of a point for spans on the same visual line
        # (e.g., 135.9 vs 136.0).  Without rounding, a strict (y, x)
        # sort can interleave columns: (135.9, x=470) sorts before
        # (136.0, x=317), making the gap check go negative and merging
        # unrelated columns.
        raw_spans.sort(key=lambda s: (round(s["y0"]), s["x0"]))

        # Group spans into cells by y-proximity (same row) and x-gap
        cells: list[dict] = []
        current: dict | None = None

        for span in raw_spans:
            if current is None:
                # Start a new cell
                current = {
                    "texts": [span["text"]],
                    "x0": span["x0"],
                    "y0": span["y0"],
                    "x1": span["x1"],
                    "y1": span["y1"],
                    "font_size": span["font_size"],
                    "is_bold": span["is_bold"],
                }
                continue

            same_row = abs(span["y0"] - current["y0"]) <= 8.0
            small_gap = (span["x0"] - current["x1"]) < self.COLUMN_GAP_THRESHOLD

            if same_row and small_gap:
                # Extend current cell
                current["texts"].append(span["text"])
                current["x1"] = max(current["x1"], span["x1"])
                current["y1"] = max(current["y1"], span["y1"])
                if span["font_size"] > current["font_size"]:
                    current["font_size"] = span["font_size"]
                if span["is_bold"]:
                    current["is_bold"] = True
            else:
                # Flush current cell, start new one
                current["text"] = " ".join(current.pop("texts"))
                cells.append(current)
                current = {
                    "texts": [span["text"]],
                    "x0": span["x0"],
                    "y0": span["y0"],
                    "x1": span["x1"],
                    "y1": span["y1"],
                    "font_size": span["font_size"],
                    "is_bold": span["is_bold"],
                }

        # Flush last cell
        if current is not None:
            current["text"] = " ".join(current.pop("texts"))
            cells.append(current)

        # Filter out tiny/empty cells
        return [c for c in cells if len(c["text"]) >= 2]

    # Maximum y-gap between consecutive table rows before we consider
    # the table data to have ended (caption/footnotes follow).
    # Calibrated from Figure 43: max intra-table gap = ~14pt,
    # table-to-caption gap = ~20pt.  18pt cleanly separates them.
    MAX_TABLE_ROW_GAP = 18.0

    def _detect_sub_tables(
        self, blocks: list[dict],
    ) -> list[tuple[float, float, list[dict]]]:
        """Detect vertically stacked sub-tables separated by repeated headers.

        Pattern 3 (Figure 43): Two sub-tables (43a for RCT-supported doses,
        43b for PK-only doses) separated by a repeated eCrCl header row.

        Returns:
            List of (start_y, end_y, sub_blocks) tuples, or empty list
            if no sub-tables detected.
        """
        # Find cells whose text starts with eCrCl-like header patterns.
        # The ^anchor ensures footnote mentions like "(eCrCl)." are excluded.
        header_indices = []
        for i, block in enumerate(blocks):
            if self.ECRCL_HEADER_RE.search(block["text"]):
                header_indices.append(i)

        if len(header_indices) < 2:
            return []  # No sub-table pattern

        # Split blocks at each header index, truncating each sub-table
        # at the first large y-gap (which marks the end of table data
        # and start of caption/footnote text).
        splits = []
        for j in range(len(header_indices)):
            start_idx = header_indices[j]
            end_idx = (
                header_indices[j + 1]
                if j + 1 < len(header_indices)
                else len(blocks)
            )
            raw_sub = blocks[start_idx:end_idx]
            # Truncate at first large y-gap within the sub-table
            sub_blocks = self._truncate_at_gap(raw_sub)
            if sub_blocks:
                start_y = sub_blocks[0]["y0"]
                end_y = sub_blocks[-1]["y1"]
                splits.append((start_y, end_y, sub_blocks))

        return splits if len(splits) >= 2 else []

    def _truncate_at_gap(self, blocks: list[dict]) -> list[dict]:
        """Truncate a block list at the first large vertical gap.

        Tables have consistent row spacing (~15-18pt). A gap >25pt
        indicates the transition from table data to caption or footnotes.
        """
        if len(blocks) <= 1:
            return blocks

        sorted_by_y = sorted(blocks, key=lambda b: b["y0"])
        result = [sorted_by_y[0]]
        for block in sorted_by_y[1:]:
            gap = block["y0"] - result[-1]["y1"]
            if gap > self.MAX_TABLE_ROW_GAP:
                break  # End of table data
            result.append(block)
        return result

    # Maximum distance (PDF points) from a column center for a cell
    # to be considered part of that column.
    MAX_COLUMN_DISTANCE = 25.0

    def _extract_grid_rules(
        self,
        blocks: list[dict],
        table_id: str,
        page_num: int,
        sub_table: Optional[str] = None,
    ) -> list[TableRule]:
        """Extract rules from a grid of cells using column-anchored assignment.

        Unlike simple position-index mapping, this method:
        1. Defines column positions from the header row
        2. Assigns every cell to its nearest column by x-coordinate
        3. Within each column, merges multi-line cells by concatenation
        4. Rebuilds data rows from column-assigned cells

        This correctly handles multi-line cells (e.g., "150 mg b.i.d. or
        / 110 mg b.i.d." in the Dabigatran column) that create duplicate
        x-positions when naively sorted.
        """
        if not blocks:
            return []

        # ── Step 1: Identify header row ──────────────────────────────
        rows = self._cluster_by_y(blocks, tolerance=8.0)
        if len(rows) < 2:
            return []

        header_row = rows[0]
        header_row.sort(key=lambda b: b["x0"])
        headers = [b["text"].strip() for b in header_row]
        col_centers = [b["x0"] for b in header_row]
        n_cols = len(col_centers)

        if n_cols < 2:
            return []

        # ── Step 2: Assign non-header cells to columns ───────────────
        # column_cells[col_idx] = list of (y0, text, bbox) sorted by y
        column_cells: dict[int, list[tuple[float, str, tuple]]] = {
            i: [] for i in range(n_cols)
        }

        for row in rows[1:]:
            for cell in row:
                col_idx = self._nearest_column(cell["x0"], col_centers)
                if col_idx is None:
                    continue  # Cell too far from any column — skip
                column_cells[col_idx].append((
                    cell["y0"],
                    cell["text"].strip(),
                    (cell["x0"], cell["y0"], cell["x1"], cell["y1"]),
                ))

        # ── Step 3: Within each column, merge vertically adjacent ────
        # cells into single values.  Two cells within 12pt vertically
        # in the same column are continuations (e.g., "150 mg b.i.d. or"
        # at y=136 + "110 mg b.i.d." at y=145 → one merged value).
        #
        # The 12pt threshold is calibrated from Figure 43 geometry:
        #   Multi-line continuation gaps: 9-10pt → merge ✅
        #   Cross-row gaps: 14pt minimum → stay separate ✅
        MERGE_Y_THRESHOLD = 12.0
        merged_cols: dict[int, list[tuple[float, str, tuple]]] = {}
        for col_idx, cells in column_cells.items():
            cells.sort(key=lambda c: c[0])  # sort by y

            # V1.1.1: Deduplicate cells at the same y-position with identical text.
            # PyMuPDF can extract the same span twice from overlapping block/line
            # boundaries, producing e.g. two "could be considered" at y=428.
            # Without dedup, the merge step concatenates them:
            #   "could be considered" + " " + "could be considered"
            deduped: list[tuple[float, str, tuple]] = []
            for y0, text, bbox in cells:
                if deduped and abs(y0 - deduped[-1][0]) < 2.0 and text == deduped[-1][1]:
                    continue  # Skip exact duplicate at same y-position
                deduped.append((y0, text, bbox))

            merged: list[tuple[float, str, tuple]] = []
            for y0, text, bbox in deduped:
                if merged and (y0 - merged[-1][0]) < MERGE_Y_THRESHOLD:
                    prev_y, prev_text, prev_bbox = merged[-1]
                    # V1.1.1: Skip if text is identical to what's already
                    # merged (PyMuPDF duplicate span at different y-offset).
                    if text == prev_text:
                        continue
                    # Also skip if text is already a suffix of prev_text
                    if prev_text.endswith(text):
                        continue
                    # Merge with previous — append text
                    merged[-1] = (
                        prev_y,
                        prev_text + " " + text,
                        (prev_bbox[0], prev_bbox[1],
                         max(prev_bbox[2], bbox[2]),
                         max(prev_bbox[3], bbox[3])),
                    )
                else:
                    merged.append((y0, text, bbox))
            merged_cols[col_idx] = merged

        # ── Step 4: Rebuild data rows from column-merged cells ───────
        # Collect all unique y-positions across columns, cluster into
        # row bands, then for each band pick the cell from each column.
        all_ys = []
        for col_idx, cells in merged_cols.items():
            for y0, text, bbox in cells:
                all_ys.append(y0)

        if not all_ys:
            return []

        # Cluster y-positions into row bands
        row_bands = self._cluster_y_values(sorted(set(all_ys)), tolerance=12.0)

        # ── Step 5: Detect drug/condition orientation ────────────────
        condition_in_rows = bool(
            self.CONDITION_ROW_HEADER_RE.search(headers[0])
        )

        # ── Step 6: Produce rules ────────────────────────────────────
        rules = []
        for band_y in row_bands:
            # Find cells from each column that fall in this band
            row_data: dict[int, tuple[str, tuple]] = {}
            for col_idx, cells in merged_cols.items():
                for y0, text, bbox in cells:
                    if abs(y0 - band_y) <= 12.0:
                        row_data[col_idx] = (text, bbox)
                        break

            # Column 0 = row label
            if 0 not in row_data:
                continue
            row_label = row_data[0][0]
            if not row_label or len(row_label) < 2:
                continue

            # Skip rows where the row label is too long — these are
            # caption/footnote text that leaked through the sub-table
            # boundary detection.  Real table row labels (eCrCl bands,
            # drug names) are always concise (< 50 chars).
            if len(row_label) > 50:
                continue

            # Columns 1..n = values
            for col_idx in range(1, n_cols):
                if col_idx not in row_data:
                    continue

                value, bbox = row_data[col_idx]
                if not value:
                    continue

                col_header = headers[col_idx] if col_idx < len(headers) else ""

                # Strip footnote superscripts from values
                value_clean = re.sub(r'\s+[a-h],[a-h]$', '', value)
                value_clean = re.sub(r'\s+[a-h]$', '', value_clean)

                if condition_in_rows:
                    drug = col_header    # Header = drug names
                    condition = row_label  # Column 0 = eCrCl band
                else:
                    drug = row_label     # Column 0 = drug names
                    condition = col_header  # Header = conditions

                rules.append(TableRule(
                    drug=drug,
                    condition=condition,
                    value=value_clean,
                    table_id=table_id.lower().replace(" ", "_"),
                    page_number=page_num,
                    sub_table=sub_table,
                    confidence=0.95,
                    bbox=bbox,
                ))

        return rules

    def _nearest_column(
        self, x0: float, col_centers: list[float],
    ) -> Optional[int]:
        """Find the nearest column index for a cell's x-position.

        Returns None if the cell is too far from any column center,
        which filters out caption/footnote text that spans the full
        page width instead of aligning to table columns.
        """
        min_dist = float("inf")
        best_col = None
        for i, center in enumerate(col_centers):
            dist = abs(x0 - center)
            if dist < min_dist:
                min_dist = dist
                best_col = i
        if min_dist > self.MAX_COLUMN_DISTANCE:
            return None
        return best_col

    @staticmethod
    def _cluster_y_values(
        y_values: list[float], tolerance: float = 12.0,
    ) -> list[float]:
        """Cluster a sorted list of y-values into row band representatives.

        Returns one representative y per band (the first value in each
        cluster), used to define row boundaries for the grid.
        """
        if not y_values:
            return []
        bands = [y_values[0]]
        for y in y_values[1:]:
            if y - bands[-1] > tolerance:
                bands.append(y)
        return bands

    def _cluster_by_y(
        self, blocks: list[dict], tolerance: float = 8.0,
    ) -> list[list[dict]]:
        """Cluster blocks into rows by y-coordinate proximity.

        Blocks within `tolerance` PDF points of each other vertically
        are grouped into the same row. Rows are sorted top-to-bottom.
        """
        if not blocks:
            return []

        sorted_blocks = sorted(blocks, key=lambda b: b["y0"])
        rows: list[list[dict]] = [[sorted_blocks[0]]]

        for block in sorted_blocks[1:]:
            last_row = rows[-1]
            last_y = sum(b["y0"] for b in last_row) / len(last_row)
            if abs(block["y0"] - last_y) <= tolerance:
                last_row.append(block)
            else:
                rows.append([block])

        return rows

    # ─── NOAC Dosing Completeness Check ───────────────────────────────────

    @staticmethod
    def validate_noac_completeness(rules: list[TableRule]) -> list[str]:
        """Verify Figure 43 extracted all expected drug-eCrCl combinations.

        This is a clinical safety check — missing a drug-dose-eCrCl combination
        means the dosing engine could fail silently for a patient query.

        Returns:
            List of missing drug-eCrCl combinations (empty = all present).
        """
        expected_drugs = {"Apixaban", "Dabigatran", "Edoxaban", "Rivaroxaban"}
        expected_bands_a = {">95", "51-95", "31-50"}  # RCT-supported bands

        # Collect found combinations for figure_43 rules.
        # Normalize en-dashes (–) to hyphens (-) for matching, since
        # PyMuPDF extracts Unicode en-dashes from the PDF.
        found_combos: set[tuple[str, str]] = set()
        for rule in rules:
            if "figure_43" in rule.table_id or "43" in rule.table_id:
                norm_condition = rule.condition.replace("\u2013", "-")
                found_combos.add((rule.drug, norm_condition))

        if not found_combos:
            return []  # No Figure 43 rules found — skip check

        missing = []
        for drug in expected_drugs:
            for band in expected_bands_a:
                if not any(
                    drug.lower() in combo[0].lower()
                    and band in combo[1]
                    for combo in found_combos
                ):
                    missing.append(f"{drug} at eCrCl {band}")

        return missing

    # ─── Diagnostics ──────────────────────────────────────────────────────

    def diagnostic_report(self) -> str:
        """Generate a diagnostic report of all table detections."""
        detections = self.detect_all_tables()
        rules = self.extract_all_rules()

        lines = [f"TableBoundaryOracle v{self.VERSION} — Diagnostic Report"]
        lines.append(f"Pages scanned: {len(self._page_data)}")
        lines.append(f"Pages with tables: {len(detections)}")
        lines.append(f"Total rules extracted: {len(rules)}")
        lines.append("")

        for page_num in sorted(detections.keys()):
            det = detections[page_num]
            content = ", ".join(det.table_ids) if det.table_ids else "none"
            refs = ", ".join(det.reference_ids) if det.reference_ids else "none"
            page_rules = [r for r in rules if r.page_number == page_num]
            lines.append(
                f"  Page {page_num:3d}: "
                f"CONTENT=[{content}] "
                f"REF=[{refs}] "
                f"rules={len(page_rules)}"
            )

        # NOAC completeness check
        noac_missing = self.validate_noac_completeness(rules)
        if noac_missing:
            lines.append(f"\nNOAC COMPLETENESS: FAIL — missing {len(noac_missing)} combinations:")
            for m in noac_missing:
                lines.append(f"    {m}")
        elif any("43" in r.table_id for r in rules):
            lines.append(f"\nNOAC COMPLETENESS: PASS — all expected combinations found.")

        return "\n".join(lines)
