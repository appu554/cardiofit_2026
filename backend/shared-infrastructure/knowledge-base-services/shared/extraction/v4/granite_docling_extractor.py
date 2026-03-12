"""
Granite-Docling VlmPipeline Wrapper for V4.1 Hybrid Architecture.

Runs IBM's Granite-Docling 258M VLM on a PDF to produce structural
DocTags output — sections, tables (OTSL), footnotes, captions, lists.

This module is consumed ONLY by Channel A (ChannelAStructuralOracle).
It does NOT produce markdown text — Marker handles that.

Usage:
    from .granite_docling_extractor import GraniteDoclingExtractor

    extractor = GraniteDoclingExtractor()
    result = extractor.extract(pdf_path="/path/to/kdigo.pdf")
    # result.sections, result.tables, result.total_pages, result.raw_items
"""

from __future__ import annotations

import re
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional, Union


# ─── Output Data Models ────────────────────────────────────────────────────

@dataclass
class DocTagSection:
    """A section header detected by Granite-Docling DocTags."""
    text: str                   # heading text (stripped of DocTag markup)
    level: int                  # heading depth (1=top-level chapter)
    page_number: int
    doctag_type: str            # "section_header", "title", etc.


@dataclass
class DocTagTable:
    """A table detected by Granite-Docling in OTSL format."""
    raw_otsl: str               # raw OTSL text with <ched>/<fcel>/<nl> tags
    column_headers: list[str]   # extracted header cell text from <ched> tags
    row_count: int              # number of data rows (excluding header)
    page_number: int


@dataclass
class DocTagElement:
    """Any structural element from DocTags (for metadata enrichment)."""
    text: str
    doctag_type: str            # "footnote", "caption", "list_item", etc.
    page_number: int


@dataclass
class DocTagsResult:
    """Complete structural output from Granite-Docling VlmPipeline."""
    sections: list[DocTagSection]
    tables: list[DocTagTable]
    elements: list[DocTagElement]    # footnotes, captions, lists, etc.
    total_pages: int
    elapsed_ms: float = 0.0
    error: Optional[str] = None


# ─── Extractor ──────────────────────────────────────────────────────────────

class GraniteDoclingExtractor:
    """Run Granite-Docling 258M VlmPipeline on a PDF for structural analysis.

    Returns DocTagsResult with sections, OTSL tables, and typed elements.
    Does NOT produce markdown — Marker is the text-of-record backend.

    Requires:
        pip install docling  (with VlmPipeline support)
        The granite-docling model weights (~258M, auto-downloaded by Docling)
    """

    VERSION = "4.1.0"

    # DocTag types that represent section headings
    HEADING_TYPES = {"section_header", "title", "subtitle"}

    # DocTag types for tables
    TABLE_TYPES = {"table"}

    # DocTag types for metadata enrichment (footnotes, captions, lists)
    ENRICHMENT_TYPES = {
        "footnote", "caption", "figure_caption",
        "ordered_list", "unordered_list", "list_item",
        "page_header", "page_footer",
    }

    def extract(
        self,
        pdf_path: Union[str, Path],
    ) -> DocTagsResult:
        """Extract structural DocTags from a PDF using Granite-Docling VlmPipeline.

        Args:
            pdf_path: Path to the PDF file

        Returns:
            DocTagsResult with sections, tables, and typed elements
        """
        pdf_path = Path(pdf_path)
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF not found: {pdf_path}")

        start_time = time.monotonic()

        try:
            doc = self._run_vlm_pipeline(pdf_path)
            result = self._parse_document(doc)
            result.elapsed_ms = (time.monotonic() - start_time) * 1000
            return result
        except Exception as e:
            elapsed = (time.monotonic() - start_time) * 1000
            return DocTagsResult(
                sections=[],
                tables=[],
                elements=[],
                total_pages=0,
                elapsed_ms=elapsed,
                error=str(e),
            )

    def _run_vlm_pipeline(self, pdf_path: Path):
        """Run the VlmPipeline on the PDF and return a DoclingDocument."""
        from docling.document_converter import DocumentConverter, PdfFormatOption
        from docling.datamodel.base_models import InputFormat

        # Import VlmPipeline — this activates the Granite-Docling 258M model
        try:
            from docling.pipeline.vlm_pipeline import VlmPipeline
            from docling.datamodel.pipeline_options import VlmPipelineOptions
        except ImportError:
            raise ImportError(
                "VlmPipeline not available. Ensure docling is installed with "
                "VLM support: pip install 'docling[vlm]' or similar. "
                "The granite-docling model weights will be auto-downloaded."
            )

        converter = DocumentConverter(
            format_options={
                InputFormat.PDF: PdfFormatOption(
                    pipeline_cls=VlmPipeline,
                    pipeline_options=VlmPipelineOptions(),
                )
            }
        )

        conv_result = converter.convert(str(pdf_path))
        return conv_result.document

    def _parse_document(self, doc) -> DocTagsResult:
        """Parse a DoclingDocument into structured DocTagsResult."""
        sections: list[DocTagSection] = []
        tables: list[DocTagTable] = []
        elements: list[DocTagElement] = []
        total_pages = 1

        # Track page count
        if hasattr(doc, "pages") and doc.pages:
            total_pages = len(doc.pages)

        for item, level in doc.iterate_items():
            item_type = self._get_item_type(item)
            page_number = self._get_page_number(item)

            if page_number > total_pages:
                total_pages = page_number

            # Section headers
            if item_type in self.HEADING_TYPES:
                text = self._get_item_text(item)
                if text:
                    sections.append(DocTagSection(
                        text=text,
                        level=level if level > 0 else 1,
                        page_number=page_number,
                        doctag_type=item_type,
                    ))

            # Tables (OTSL format)
            elif item_type in self.TABLE_TYPES:
                otsl_table = self._parse_table_item(item, page_number)
                if otsl_table:
                    tables.append(otsl_table)

            # Enrichment elements (footnotes, captions, lists)
            elif item_type in self.ENRICHMENT_TYPES:
                text = self._get_item_text(item)
                if text:
                    elements.append(DocTagElement(
                        text=text,
                        doctag_type=item_type,
                        page_number=page_number,
                    ))

        return DocTagsResult(
            sections=sections,
            tables=tables,
            elements=elements,
            total_pages=total_pages,
        )

    def _get_item_type(self, item) -> str:
        """Get the DocTag type name for an item."""
        # Docling items have a 'label' attribute or class-based type
        label = getattr(item, "label", None)
        if label:
            # DocItemLabel enum → string
            return str(label).lower().replace("docitemlabel.", "")

        # Fallback: use class name
        cls_name = type(item).__name__.lower()
        if "table" in cls_name:
            return "table"
        if "heading" in cls_name or "section" in cls_name:
            return "section_header"
        return cls_name

    def _get_item_text(self, item) -> str:
        """Extract text content from a Docling item."""
        # Try common text access patterns
        if hasattr(item, "text"):
            return str(item.text).strip()
        if hasattr(item, "export_to_markdown"):
            try:
                return item.export_to_markdown().strip()
            except Exception:
                pass
        return ""

    def _get_page_number(self, item) -> int:
        """Get page number from an item's provenance."""
        prov = getattr(item, "prov", None)
        if prov and len(prov) > 0:
            page_no = getattr(prov[0], "page_no", None)
            if page_no:
                return int(page_no)
        return 1

    def _parse_table_item(self, item, page_number: int) -> Optional[DocTagTable]:
        """Parse a table item into DocTagTable with OTSL text.

        Tries multiple extraction strategies:
        1. Direct OTSL export (if available in newer Docling versions)
        2. Markdown table export → convert to OTSL-like format
        3. DataFrame export → construct cell representation
        """
        # Strategy 1: Check for direct DocTags/OTSL text
        doctags_text = getattr(item, "text", None)
        if doctags_text and "<" in doctags_text and ">" in doctags_text:
            # Looks like DocTags/OTSL markup
            return self._parse_otsl_text(doctags_text, page_number)

        # Strategy 2: Try export_to_markdown for pipe table
        try:
            md_text = item.export_to_markdown()
            if md_text and "|" in md_text:
                return self._pipe_table_to_otsl(md_text, page_number)
        except Exception:
            pass

        # Strategy 3: Try export_to_dataframe
        try:
            if hasattr(item, "export_to_dataframe"):
                df = item.export_to_dataframe()
                return self._dataframe_to_otsl(df, page_number)
        except Exception:
            pass

        return None

    def _parse_otsl_text(self, otsl_text: str, page_number: int) -> DocTagTable:
        """Parse raw OTSL text into a DocTagTable."""
        # Extract column headers from <ched> tags
        headers = re.findall(r"<ched>(.*?)</ched>", otsl_text)

        # Count data rows (number of <nl> delimiters minus header row)
        rows = otsl_text.split("<nl>")
        # First row is typically headers
        data_row_count = max(0, len(rows) - 1)

        return DocTagTable(
            raw_otsl=otsl_text,
            column_headers=[h.strip() for h in headers],
            row_count=data_row_count,
            page_number=page_number,
        )

    def _pipe_table_to_otsl(self, md_text: str, page_number: int) -> DocTagTable:
        """Convert a markdown pipe table to OTSL-like format."""
        lines = [l.strip() for l in md_text.strip().split("\n") if l.strip()]
        otsl_parts = []
        headers = []
        data_rows = 0

        for i, line in enumerate(lines):
            if not (line.startswith("|") and line.endswith("|")):
                continue

            cells = [c.strip() for c in line.split("|")[1:-1]]

            # Skip separator lines
            if all(self._is_separator(c) for c in cells):
                continue

            if not headers:
                # Header row
                headers = cells
                row_otsl = "".join(f"<ched>{c}</ched>" for c in cells)
            else:
                # Data row
                data_rows += 1
                row_otsl = "".join(f"<fcel>{c}</fcel>" for c in cells)

            otsl_parts.append(row_otsl)

        raw_otsl = "<nl>".join(otsl_parts)

        return DocTagTable(
            raw_otsl=raw_otsl,
            column_headers=headers,
            row_count=data_rows,
            page_number=page_number,
        )

    def _dataframe_to_otsl(self, df, page_number: int) -> DocTagTable:
        """Convert a pandas DataFrame to OTSL format."""
        headers = [str(h) for h in df.columns]
        header_otsl = "".join(f"<ched>{h}</ched>" for h in headers)

        row_parts = [header_otsl]
        for _, row in df.iterrows():
            cells = [str(v).strip() for v in row]
            row_otsl = "".join(
                f"<fcel>{c}</fcel>" if c else "<ecel></ecel>"
                for c in cells
            )
            row_parts.append(row_otsl)

        raw_otsl = "<nl>".join(row_parts)

        return DocTagTable(
            raw_otsl=raw_otsl,
            column_headers=headers,
            row_count=len(df),
            page_number=page_number,
        )

    @staticmethod
    def _is_separator(cell: str) -> bool:
        """Check if a cell is a markdown table separator."""
        cleaned = cell.replace("-", "").replace(":", "").replace(" ", "")
        return len(cleaned) == 0 and len(cell.strip()) > 0
