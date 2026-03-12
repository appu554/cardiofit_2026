"""
PageBoundaryOracle — Authoritative page boundaries from PDF structure.

Uses PyMuPDF per-page text extraction to determine exactly where each page's
content begins in the merged markdown string.  Replaces the fragile heuristic
in MonkeyOCR's _insert_page_markers() (which uses str.index() on repeating
headers) with formatting-tolerant word-sequence regex matching.

The PDF page structure is the source of truth — zero ambiguity, zero heuristics.

Integration:
    - run_chunked_l1.py: after MonkeyOCR/PyMuPDF extraction, call
      oracle.correct_page_markers(markdown) to fix markers
    - Channel A: page_map built from corrected markers is accurate
    - All downstream channels (B-G) inherit correct page numbers

Matching strategy:
    For each page, extract content blocks via PyMuPDF get_text("dict").
    Find blocks unique to that page (not repeating headers/footers).
    Build a formatting-tolerant regex from the block's words:
        "Recommendation 3.3.1: We suggest" →
        r'Recommendation[\\W_]{0,15}3\\.3\\.1[\\W_]{0,15}We[\\W_]{0,15}suggest'
    This matches across **bold**, # headers, | table pipes, etc.
"""

from __future__ import annotations

import re
from pathlib import Path
from typing import Optional

try:
    import pymupdf
except ImportError:
    import fitz as pymupdf


# ═══════════════════════════════════════════════════════════════════════════════
# Page Boundary Oracle
# ═══════════════════════════════════════════════════════════════════════════════

class PageBoundaryOracle:
    """Authoritative page boundaries from PDF structure.

    Opens the PDF once, extracts per-page text blocks, then uses
    word-sequence regex to locate each page's content in any markdown
    representation of the same PDF.
    """

    VERSION = "1.0.0"

    # Repeating headers/footers to skip when finding unique anchors.
    # These appear on every page and cause the exact bug we're fixing.
    SKIP_RE = re.compile(
        r'www\.|kidney.international|^\d+\s*$|^chapter\s+\d'
        r'|^S\d+\s*$|^Kidney\s+International',
        re.IGNORECASE,
    )

    # Minimum block length for anchor candidates
    MIN_ANCHOR_CHARS = 20

    def __init__(self, pdf_path: str | Path, page_offset: int = 0):
        """
        Args:
            pdf_path:    Path to the PDF (full or chunk).
            page_offset: Added to 0-based page indices to produce global
                         page numbers.  E.g., for chunk-c (pages 20-30),
                         pass page_offset=20 so page 0 in the chunk PDF
                         becomes page 21 globally.
        """
        self.pdf_path = str(pdf_path)
        self.page_offset = page_offset
        self._page_blocks: dict[int, list[str]] = {}  # page_num → content blocks
        self._all_blocks_flat: dict[int, set[str]] = {}  # for uniqueness checks
        self._load_pages()

    def _load_pages(self) -> None:
        """Extract content blocks per page using PyMuPDF dict mode."""
        doc = pymupdf.open(self.pdf_path)
        num_pages = len(doc)

        for page_idx in range(num_pages):
            page = doc[page_idx]
            page_num = page_idx + 1 + self.page_offset
            page_dict = page.get_text("dict")

            blocks: list[str] = []
            for block in page_dict.get("blocks", []):
                if block.get("type") != 0:  # text blocks only
                    continue
                text_parts = []
                for line in block.get("lines", []):
                    for span in line.get("spans", []):
                        text_parts.append(span.get("text", ""))
                text = " ".join(text_parts).strip()

                if len(text) >= self.MIN_ANCHOR_CHARS and not self.SKIP_RE.search(text):
                    blocks.append(text)

            self._page_blocks[page_num] = blocks

        doc.close()

        # Build flat set per page for uniqueness checking
        for page_num, blocks in self._page_blocks.items():
            other_blocks: set[str] = set()
            for other_num, other_bks in self._page_blocks.items():
                if other_num != page_num:
                    other_blocks.update(other_bks)
            self._all_blocks_flat[page_num] = other_blocks

    # ─── Public API ───────────────────────────────────────────────────────

    def correct_page_markers(self, markdown: str) -> str:
        """Strip existing PAGE markers and re-insert at correct positions.

        Args:
            markdown: Merged markdown string (with possibly wrong markers).

        Returns:
            Markdown with corrected <!-- PAGE N --> markers.
        """
        # Strip all existing PAGE markers
        clean_md = re.sub(r'\n?<!--\s*PAGE\s+\d+\s*-->\n?', '', markdown)

        # Find correct page boundary positions
        page_positions = self._find_all_page_positions(clean_md)

        if not page_positions:
            # Fallback: return original if oracle found nothing
            return markdown

        # Re-insert markers in reverse order (so positions don't shift)
        result = clean_md
        for page_num in sorted(page_positions.keys(), reverse=True):
            pos = page_positions[page_num]
            marker = f"\n<!-- PAGE {page_num} -->\n"
            result = result[:pos] + marker + result[pos:]

        return result

    def build_page_map(self, markdown: str) -> dict[int, int]:
        """Build a page_map (offset → page_number) from oracle analysis.

        This can be used directly by Channel A instead of parsing markers.

        Args:
            markdown: Clean markdown (PAGE markers should already be stripped
                      or will be stripped internally).

        Returns:
            Dict mapping character offsets to page numbers.
        """
        clean_md = re.sub(r'\n?<!--\s*PAGE\s+\d+\s*-->\n?', '', markdown)
        return self._find_all_page_positions(clean_md)

    # ─── Core matching logic ──────────────────────────────────────────────

    def _find_all_page_positions(self, clean_md: str) -> dict[int, int]:
        """Find the start position of each page's content in the markdown.

        Uses monotonically increasing search_from to ensure pages are
        discovered in order (page N+1 always starts after page N).
        """
        page_positions: dict[int, int] = {}
        search_from = 0
        sorted_pages = sorted(self._page_blocks.keys())

        for i, page_num in enumerate(sorted_pages):
            blocks = self._page_blocks[page_num]

            pos = self._find_page_start(
                blocks, page_num, clean_md, search_from,
            )

            if pos >= 0:
                page_positions[page_num] = pos
                search_from = pos + 1
            elif i == 0:
                # First page: default to start of text
                page_positions[page_num] = 0
                search_from = 1

        return page_positions

    def _find_page_start(
        self,
        blocks: list[str],
        page_num: int,
        markdown: str,
        search_from: int,
    ) -> int:
        """Find where this page's content begins in markdown.

        Strategy:
        1. Try UNIQUE blocks first (text not found on any other page)
           → These are the most reliable anchors
        2. Fall back to ANY content block
           → Still better than repeating headers

        Uses word-sequence regex for formatting-tolerant matching.
        """
        other_blocks = self._all_blocks_flat.get(page_num, set())

        # Phase 1: unique blocks (preferred)
        for block_text in blocks:
            if block_text in other_blocks:
                continue  # appears on another page — skip

            pattern = self._make_word_pattern(block_text)
            if pattern:
                m = re.search(pattern, markdown[search_from:])
                if m:
                    return search_from + m.start()

        # Phase 2: any content block (fallback)
        for block_text in blocks:
            pattern = self._make_word_pattern(block_text)
            if pattern:
                m = re.search(pattern, markdown[search_from:])
                if m:
                    return search_from + m.start()

        return -1

    # ─── Regex builder ────────────────────────────────────────────────────

    @staticmethod
    def _make_word_pattern(text: str, max_words: int = 10) -> Optional[str]:
        """Build a regex matching text words with any formatting between them.

        Extracts alphanumeric words (including version numbers like "3.3.1"),
        then joins them with [\\W_]{0,15} which absorbs any markdown
        formatting characters (**, ##, |, etc.) and whitespace.

        Example:
            "Recommendation 3.3.1: We suggest that adults"
            → r'Recommendation[\\W_]{0,15}3\\.3\\.1[\\W_]{0,15}We[\\W_]{0,15}suggest[\\W_]{0,15}that[\\W_]{0,15}adults'

        This matches:
            "**Recommendation 3.3.1:** We suggest that adults"  ✅
            "## Recommendation 3.3.1: We suggest that adults"   ✅
            "| Recommendation 3.3.1 | We suggest that adults"   ✅
        """
        # Extract words, preserving dotted numbers (3.3.1, 6.5)
        words = re.findall(r'[A-Za-z0-9]+(?:\.[0-9]+)*', text)
        if len(words) < 3:
            return None
        words = words[:max_words]
        # Allow up to 15 non-word chars between each word
        return r'[\W_]{0,15}'.join(re.escape(w) for w in words)

    # ─── Table content detection ──────────────────────────────────────────

    def page_has_table_content(self, page_num: int, table_id: str) -> bool:
        """Distinguish table CONTENT pages from table REFERENCE pages.

        KDIGO convention:
            Content:   "Table 32 | Medications..."  (pipe after table ID)
            Reference: "shown in Table 32."         (prose mention)

        Args:
            page_num: 1-based page number
            table_id: e.g. "Table 32" or "Figure 44"

        Returns:
            True if page contains the actual table content (pipe-header),
            False if it only references the table in prose.
        """
        blocks = self._page_blocks.get(page_num, [])
        header_pattern = re.compile(
            re.escape(table_id) + r'\s*\|',
            re.IGNORECASE,
        )
        return any(header_pattern.search(b) for b in blocks)

    # ─── Diagnostics ──────────────────────────────────────────────────────

    def diagnostic_report(self, markdown: str) -> str:
        """Generate a diagnostic report showing page boundary detection.

        Useful for verifying the oracle works correctly on a new PDF.
        """
        clean_md = re.sub(r'\n?<!--\s*PAGE\s+\d+\s*-->\n?', '', markdown)
        positions = self._find_all_page_positions(clean_md)

        lines = [f"PageBoundaryOracle v{self.VERSION} — Diagnostic Report"]
        lines.append(f"Pages in PDF: {len(self._page_blocks)}")
        lines.append(f"Pages located in markdown: {len(positions)}")
        lines.append("")

        sorted_pages = sorted(self._page_blocks.keys())
        for page_num in sorted_pages:
            n_blocks = len(self._page_blocks[page_num])
            if page_num in positions:
                pos = positions[page_num]
                # Show 60 chars of context at the boundary
                context = clean_md[pos:pos + 60].replace('\n', '\\n')
                lines.append(
                    f"  Page {page_num:3d}: offset {pos:6d}, "
                    f"{n_blocks} blocks, "
                    f'context="{context}..."'
                )
            else:
                lines.append(
                    f"  Page {page_num:3d}: NOT FOUND ({n_blocks} blocks)"
                )

        missing = [p for p in sorted_pages if p not in positions]
        if missing:
            lines.append(f"\nMISSING PAGES: {missing}")
        else:
            lines.append(f"\nAll {len(sorted_pages)} pages located.")

        return "\n".join(lines)
