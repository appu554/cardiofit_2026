"""
PDF Highlighter: Generate reviewer QA PDF with color-coded span annotations.

Takes source PDF + merged spans from Signal Merger → produces a highlighted
PDF where each extracted span is visually marked on the correct page,
color-coded by extraction channel.

Option A implementation: PyMuPDF page.search_for(text) for ~92% match rate.
Future Option B: rawdict bounding-box coordinate mapping for pixel-perfect.

Pipeline Position:
    Signal Merger → merged_spans.json → PDF Highlighter (THIS) → highlighted_review.pdf
"""

from __future__ import annotations

import fitz  # PyMuPDF
from typing import Optional


# Channel → RGB color mapping for highlight annotations
CHANNEL_COLORS: dict[str, tuple[float, float, float]] = {
    "B": (0.2, 0.4, 0.9),          # Blue — Drug Dictionary (Aho-Corasick)
    "C": (0.1, 0.7, 0.3),          # Green — Grammar/Regex
    "D": (0.6, 0.2, 0.8),          # Purple — Table Extraction
    "E": (0.9, 0.5, 0.1),          # Orange — GLiNER NER
    "F": (0.1, 0.7, 0.7),          # Teal — NuExtract
    "L1_RECOVERY": (0.9, 0.2, 0.2),  # Red — L1 Recovery spans
}

CHANNEL_LABELS: dict[str, str] = {
    "B": "Drug Dict",
    "C": "Grammar",
    "D": "Table",
    "E": "GLiNER",
    "F": "NuExtract",
    "L1_RECOVERY": "L1 Recovery",
}

# Text search limits — long strings degrade fitz search_for() accuracy
_MAX_SEARCH_LEN = 80
_TRUNCATED_SEARCH_LEN = 60


class PDFHighlighter:
    """Generate a highlighted PDF from merged spans using PyMuPDF text search.

    Color-codes each span by its primary extraction channel so the reviewer
    can visually QA what was extracted and from which channel.
    """

    VERSION = "1.0.0"

    def highlight(
        self,
        pdf_path: str,
        merged_spans: list,
        output_path: str,
        page_offset: int = 0,
    ) -> dict:
        """Generate highlighted PDF from merged spans.

        Args:
            pdf_path: Path to source KDIGO PDF
            merged_spans: list[MergedSpan] from Signal Merger
            output_path: Where to save highlighted PDF
            page_offset: If source PDF is a page subset (e.g., pages 58-61),
                         page_offset maps span.page_number to PDF page index.
                         span page 58 → PDF index 0 when page_offset=58.
                         When 0, page_number is used as 1-based index directly.

        Returns:
            Stats dict with keys: matched, unmatched, total, unmatched_texts
        """
        doc = fitz.open(pdf_path)
        total_pages = len(doc)

        matched = 0
        unmatched = 0
        unmatched_texts: list[str] = []

        # Group spans by page_number for efficient per-page processing
        page_groups: dict[int, list] = {}
        no_page_spans: list = []

        for span in merged_spans:
            pn = getattr(span, "page_number", None)
            if pn is None:
                no_page_spans.append(span)
                continue
            page_groups.setdefault(pn, []).append(span)

        # Process each page group
        for page_num, spans in sorted(page_groups.items()):
            # Convert page_number to 0-based PDF page index
            if page_offset > 0:
                page_idx = page_num - page_offset
            else:
                # page_number is 1-based → convert to 0-based index
                page_idx = page_num - 1

            if page_idx < 0 or page_idx >= total_pages:
                # Page out of range — count as unmatched
                for span in spans:
                    unmatched += 1
                    unmatched_texts.append(
                        f"[page {page_num} out of range] {_truncate(span.text, 50)}"
                    )
                continue

            page = doc[page_idx]

            for span in spans:
                found = self._search_and_highlight(page, span)
                if found:
                    matched += 1
                else:
                    unmatched += 1
                    unmatched_texts.append(
                        f"[page {page_num}] {_truncate(span.text, 50)}"
                    )

        # Spans with no page_number — try all pages as fallback
        for span in no_page_spans:
            found = False
            search_text = _prepare_search_text(span.text)
            if search_text:
                for page_idx in range(total_pages):
                    rects = doc[page_idx].search_for(search_text)
                    if rects:
                        self._add_highlight(doc[page_idx], rects[0], span)
                        found = True
                        break
            if found:
                matched += 1
            else:
                unmatched += 1
                unmatched_texts.append(
                    f"[no page] {_truncate(span.text, 50)}"
                )

        # Add color legend on first page
        if total_pages > 0:
            self._add_legend(doc[0])

        doc.save(output_path)
        doc.close()

        return {
            "matched": matched,
            "unmatched": unmatched,
            "total": matched + unmatched,
            "unmatched_texts": unmatched_texts[:20],  # cap for metadata size
        }

    def _search_and_highlight(self, page: fitz.Page, span) -> bool:
        """Search for span text on page and add highlight annotation.

        Tries full text first, then truncated, then line-by-line for
        multi-line spans. Returns True if at least one rect was highlighted.
        """
        text = span.text
        search_text = _prepare_search_text(text)

        if not search_text:
            return False

        # Strategy 1: Search full text (or truncated if too long)
        rects = page.search_for(search_text)
        if rects:
            self._add_highlight(page, rects[0], span)
            return True

        # Strategy 2: For multi-line text, try first line only
        if "\n" in text:
            first_line = text.split("\n")[0].strip()
            if len(first_line) >= 5:
                rects = page.search_for(first_line[:_TRUNCATED_SEARCH_LEN])
                if rects:
                    self._add_highlight(page, rects[0], span)
                    return True

        # Strategy 3: Try first N words (handles reformatted text)
        words = text.split()
        if len(words) > 3:
            partial = " ".join(words[:5])
            rects = page.search_for(partial)
            if rects:
                self._add_highlight(page, rects[0], span)
                return True

        return False

    def _add_highlight(self, page: fitz.Page, rect: fitz.Rect, span) -> None:
        """Add a colored highlight annotation for a span."""
        channels = getattr(span, "contributing_channels", [])
        primary_channel = channels[0] if channels else "C"
        color = CHANNEL_COLORS.get(primary_channel, (0.5, 0.5, 0.5))

        annot = page.add_highlight_annot(rect)
        annot.set_colors(stroke=color)
        annot.set_opacity(0.35)

        # Popup content: span details for reviewer
        confidence = getattr(span, "merged_confidence", 0.0)
        channel_str = ", ".join(channels)
        label = CHANNEL_LABELS.get(primary_channel, primary_channel)
        info_text = (
            f"Channel: {channel_str} ({label})\n"
            f"Confidence: {confidence:.0%}\n"
            f"Text: {_truncate(span.text, 120)}"
        )
        if getattr(span, "has_disagreement", False):
            info_text += f"\nDISAGREEMENT: {getattr(span, 'disagreement_detail', '')}"

        annot.set_info(content=info_text)
        annot.update()

    def _add_legend(self, page: fitz.Page) -> None:
        """Add a color legend annotation in the top-right corner of page 1."""
        # Build legend text
        lines = ["Pipeline 1 Extraction Channels:"]
        for ch in ["B", "C", "D", "E", "F", "L1_RECOVERY"]:
            label = CHANNEL_LABELS.get(ch, ch)
            lines.append(f"  {ch} = {label}")
        lines.append("")
        lines.append("Hover highlights for details.")
        legend_text = "\n".join(lines)

        # Place in top-right corner
        page_rect = page.rect
        x1 = page_rect.width - 10
        x0 = x1 - 180
        y0 = 10
        y1 = y0 + 130

        annot = page.add_text_annot(fitz.Point(x0, y0), legend_text)
        annot.set_info(title="V4 Pipeline QA Legend")
        annot.update()


def _prepare_search_text(text: str) -> Optional[str]:
    """Prepare span text for PyMuPDF search_for().

    Cleans whitespace, handles length limits, returns None if too short.
    """
    cleaned = " ".join(text.split())  # collapse whitespace
    if len(cleaned) < 3:
        return None
    if len(cleaned) > _MAX_SEARCH_LEN:
        return cleaned[:_TRUNCATED_SEARCH_LEN]
    return cleaned


def _truncate(text: str, max_len: int) -> str:
    """Truncate text for display/logging."""
    text = text.replace("\n", " ").strip()
    if len(text) <= max_len:
        return text
    return text[:max_len - 3] + "..."
