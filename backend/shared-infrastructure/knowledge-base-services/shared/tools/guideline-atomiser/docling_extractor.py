"""
L1: PDF Parsing with Docling - Structural Document Understanding.

Alternative L1 extractor using IBM's Docling library for PDF parsing.
Docling provides deep document structure understanding including:
- Layout analysis with bounding boxes
- Table structure recognition (TableFormer)
- Reading order detection
- Hierarchical section structure

Both Marker and Docling produce markdown output consumed by Channel 0+A.
This module conforms to the same ExtractionResult interface as marker_extractor.py
so the pipeline can swap L1 backends via --l1 flag.

Usage:
    from docling_extractor import DoclingExtractor

    extractor = DoclingExtractor()
    result = extractor.extract(pdf_path="kdigo_2022_diabetes.pdf")
    # result.markdown, result.blocks, result.tables, result.provenance
"""

import hashlib
import re
from pathlib import Path
from datetime import datetime, timezone
from typing import Optional, Union

# Reuse the same data models as MarkerExtractor for interface compatibility
from marker_extractor import (
    BoundingBox,
    TextBlock,
    TableBlock,
    ExtractionProvenance,
    ExtractionResult,
    ClinicalOCRPostProcessor,
)


class DoclingExtractor:
    """
    L1 PDF Extractor using IBM Docling.

    Docling provides structural document understanding with:
    - Deep layout analysis (headings, paragraphs, lists, tables, figures)
    - TableFormer for accurate table structure recovery
    - Reading order detection across complex layouts
    - OCR integration for scanned documents
    - Export to Markdown, JSON, or DoclingDocument

    Interface-compatible with MarkerExtractor — same ExtractionResult output.
    """

    VERSION = "1.0.0"

    def __init__(
        self,
        enable_ocr: bool = True,
        enable_table_structure: bool = True,
        enable_ocr_postprocessing: bool = True,
        ocr_postprocessor: Optional[ClinicalOCRPostProcessor] = None,
    ):
        """
        Initialize the Docling extractor.

        Args:
            enable_ocr: Enable OCR for scanned documents
            enable_table_structure: Enable TableFormer for table structure
            enable_ocr_postprocessing: Enable clinical OCR error correction
            ocr_postprocessor: Custom OCR post-processor (uses default if None)
        """
        self.enable_ocr = enable_ocr
        self.enable_table_structure = enable_table_structure
        self.enable_ocr_postprocessing = enable_ocr_postprocessing
        self._docling_version = self._get_docling_version()

        if enable_ocr_postprocessing:
            self.ocr_postprocessor = ocr_postprocessor or ClinicalOCRPostProcessor()
        else:
            self.ocr_postprocessor = None

    def _get_docling_version(self) -> str:
        """Get the installed Docling version."""
        try:
            import docling
            return getattr(docling, "__version__", "unknown")
        except ImportError:
            return "not_installed"

    def _compute_file_hash(self, file_path: Path) -> str:
        """Compute SHA-256 hash of the source file."""
        sha256 = hashlib.sha256()
        with open(file_path, "rb") as f:
            for chunk in iter(lambda: f.read(8192), b""):
                sha256.update(chunk)
        return sha256.hexdigest()

    def extract(
        self,
        pdf_path: Union[str, Path],
        page_range: Optional[tuple[int, int]] = None,
    ) -> ExtractionResult:
        """
        Extract content from a PDF using Docling.

        Args:
            pdf_path: Path to the PDF file
            page_range: Optional (start, end) page range (1-indexed, inclusive)

        Returns:
            ExtractionResult with blocks, tables, markdown, and provenance
        """
        pdf_path = Path(pdf_path)
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF not found: {pdf_path}")

        return self._extract_with_docling(pdf_path, page_range)

    def _extract_with_docling(
        self,
        pdf_path: Path,
        page_range: Optional[tuple[int, int]] = None,
    ) -> ExtractionResult:
        """Extract using Docling library."""
        from docling.document_converter import DocumentConverter, PdfFormatOption
        from docling.datamodel.pipeline_options import PdfPipelineOptions
        from docling.datamodel.base_models import InputFormat

        # Configure pipeline options
        pipeline_options = PdfPipelineOptions()
        pipeline_options.do_ocr = self.enable_ocr
        pipeline_options.do_table_structure = self.enable_table_structure

        # Create converter with PDF options
        converter = DocumentConverter(
            format_options={
                InputFormat.PDF: PdfFormatOption(pipeline_options=pipeline_options)
            }
        )

        # Convert PDF
        conv_result = converter.convert(str(pdf_path))
        doc = conv_result.document

        # Extract page dimensions for V5 bbox provenance fallback.
        # Docling NER channels operate on text offsets, not geometry.
        # Storing page sizes lets the signal merger synthesize page-level
        # bboxes (x0=0, y0=0, x1=w, y1=h) when block-level coords are absent.
        page_sizes: dict[int, tuple[float, float]] = {}
        try:
            for page_no, page in (doc.pages or {}).items():
                sz = getattr(page, "size", None)
                if sz is not None:
                    w = getattr(sz, "width", 0.0) or 0.0
                    h = getattr(sz, "height", 0.0) or 0.0
                    if w > 0 and h > 0:
                        page_sizes[int(page_no)] = (float(w), float(h))
        except Exception:
            pass  # page_sizes stays empty — fallback bbox won't be available

        # Export to markdown
        markdown_text = doc.export_to_markdown()

        # Insert page markers for Channel A compatibility
        # Docling's markdown may not include page markers, so we add them
        markdown_text = self._inject_page_markers(markdown_text, doc)

        # Parse markdown into blocks with page-level bboxes for V5 provenance
        blocks = self._parse_markdown_to_blocks(markdown_text, page_sizes=page_sizes)
        tables = self._extract_tables_from_doc(doc)

        # Apply OCR post-processing
        ocr_correction_summary = None
        if self.ocr_postprocessor:
            blocks, block_summary = self.ocr_postprocessor.process_blocks(blocks)
            markdown_text, md_summary = self.ocr_postprocessor.process_markdown(
                markdown_text
            )
            ocr_correction_summary = {
                "blocks": block_summary,
                "markdown": md_summary,
                "total_corrections": (
                    block_summary["total_corrections"]
                    + md_summary["total_corrections"]
                ),
            }

        # Determine total pages
        total_pages = 1
        if hasattr(doc, "pages") and doc.pages:
            total_pages = len(doc.pages)
        elif blocks:
            total_pages = max((b.page_number for b in blocks), default=1)

        # Build provenance
        extraction_params = {
            "enable_ocr": self.enable_ocr,
            "enable_table_structure": self.enable_table_structure,
            "page_range": list(page_range) if page_range else None,
            "ocr_postprocessing_enabled": self.enable_ocr_postprocessing,
            "l1_backend": "docling",
        }
        if ocr_correction_summary:
            extraction_params["ocr_corrections"] = {
                "total": ocr_correction_summary["total_corrections"],
                "by_type": ocr_correction_summary["blocks"].get(
                    "correction_types", {}
                ),
            }

        provenance = ExtractionProvenance(
            source_file=str(pdf_path),
            source_hash=self._compute_file_hash(pdf_path),
            extraction_timestamp=datetime.now(timezone.utc).isoformat(),
            extractor_version=self.VERSION,
            marker_version=f"docling-{self._docling_version}",
            seed=0,  # Docling doesn't use seed
            total_pages=total_pages,
            extraction_params=extraction_params,
        )

        return ExtractionResult(
            blocks=blocks,
            tables=tables,
            markdown=markdown_text,
            provenance=provenance,
        )

    def _inject_page_markers(self, markdown_text: str, doc) -> str:
        """Inject <!-- PAGE N --> markers into markdown for Channel A.

        Docling's DoclingDocument tracks page provenance per element.
        We insert page markers at transitions for downstream compatibility.
        """
        # If Docling already includes page markers, skip
        if "<!-- PAGE " in markdown_text:
            return markdown_text

        # Try to get page info from document elements
        try:
            lines = markdown_text.split("\n")
            result_lines = [f"\n<!-- PAGE 1 -->\n"]
            current_page = 1

            # Docling items have provenance with page numbers
            # Walk through document items to find page transitions
            page_offsets = set()
            for item, _level in doc.iterate_items():
                prov = getattr(item, "prov", None)
                if prov and len(prov) > 0:
                    page_no = getattr(prov[0], "page_no", None)
                    if page_no and page_no != current_page:
                        page_offsets.add(page_no)
                        current_page = page_no

            # If we found page transitions, inject markers
            # Simple heuristic: inject at roughly equal intervals
            if page_offsets:
                max_page = max(page_offsets)
                lines_per_page = max(1, len(lines) // max_page)
                current_page = 1
                for i, line in enumerate(lines):
                    expected_page = min(max_page, (i // lines_per_page) + 1)
                    if expected_page > current_page:
                        current_page = expected_page
                        result_lines.append(f"\n<!-- PAGE {current_page} -->\n")
                    result_lines.append(line)
                return "\n".join(result_lines)

        except Exception:
            pass

        # Fallback: just wrap entire content as page 1
        return f"<!-- PAGE 1 -->\n{markdown_text}"

    def _parse_markdown_to_blocks(
        self,
        markdown_text: str,
        page_sizes: dict[int, tuple[float, float]] | None = None,
    ) -> list[TextBlock]:
        """Parse Docling markdown output into structured blocks.

        page_sizes: optional dict of page_number → (width, height) in PDF
        points. When provided, each TextBlock receives a page-level bbox
        (x0=0, y0=0, x1=w, y1=h) so that V5 bbox provenance can attribute
        NER spans to their source page even when block-level geometry is
        unavailable from this L1 backend.
        """
        blocks = []
        current_page = 1
        byte_offset = 0

        for line in markdown_text.split("\n"):
            if not line.strip():
                byte_offset += len(line) + 1
                continue

            # Detect page markers
            if line.strip().startswith("<!-- PAGE"):
                try:
                    page_match = re.search(r"PAGE\s+(\d+)", line)
                    if page_match:
                        current_page = int(page_match.group(1))
                except (ValueError, AttributeError):
                    pass
                byte_offset += len(line) + 1
                continue

            # Classify block type
            block_type = "text"
            heading_level = None
            is_bold = False
            is_italic = False

            stripped = line.strip()
            if stripped.startswith("#"):
                block_type = "heading"
                hashes = stripped.split()[0] if stripped.split() else ""
                heading_level = len(hashes) if all(c == "#" for c in hashes) else None
            elif stripped.startswith("- ") or stripped.startswith("* "):
                block_type = "list"
            elif stripped.startswith("```"):
                block_type = "code"
            elif stripped.startswith("|") and stripped.endswith("|"):
                block_type = "table"
            elif stripped.startswith("**") and stripped.endswith("**"):
                is_bold = True
            elif stripped.startswith("*") and stripped.endswith("*"):
                is_italic = True

            # Page-level bbox for V5 provenance: covers full page area.
            # Exact block geometry is unavailable for text-based NER channels;
            # this records WHICH PAGE the span belongs to, not its sub-page pos.
            page_bbox: BoundingBox | None = None
            if page_sizes:
                dims = page_sizes.get(current_page)
                if dims:
                    page_bbox = BoundingBox(x0=0.0, y0=0.0, x1=dims[0], y1=dims[1])

            blocks.append(
                TextBlock(
                    text=line,
                    page_number=current_page,
                    block_type=block_type,
                    byte_range_start=byte_offset,
                    byte_range_end=byte_offset + len(line),
                    heading_level=heading_level,
                    is_bold=is_bold,
                    is_italic=is_italic,
                    bbox=page_bbox,
                )
            )

            byte_offset += len(line) + 1

        return blocks

    def _extract_tables_from_doc(self, doc) -> list[TableBlock]:
        """Extract tables from Docling document structure."""
        tables = []
        table_index = 0

        try:
            for item, _level in doc.iterate_items():
                # Check if this is a table item
                item_type = type(item).__name__
                if item_type == "TableItem" or hasattr(item, "export_to_dataframe"):
                    try:
                        # Get table data via dataframe
                        df = item.export_to_dataframe()
                        headers = list(df.columns)
                        rows = df.values.tolist()

                        # Get page number from provenance
                        page_number = 1
                        prov = getattr(item, "prov", None)
                        if prov and len(prov) > 0:
                            page_number = getattr(prov[0], "page_no", 1) or 1

                        # Get bounding box if available
                        bbox = None
                        if prov and len(prov) > 0:
                            bbox_data = getattr(prov[0], "bbox", None)
                            if bbox_data:
                                bbox = BoundingBox(
                                    x0=getattr(bbox_data, "l", 0),
                                    y0=getattr(bbox_data, "t", 0),
                                    x1=getattr(bbox_data, "r", 0),
                                    y1=getattr(bbox_data, "b", 0),
                                )

                        tables.append(
                            TableBlock(
                                headers=[str(h) for h in headers],
                                rows=[[str(c) for c in row] for row in rows],
                                page_number=page_number,
                                bbox=bbox,
                                table_index=table_index,
                            )
                        )
                        table_index += 1

                    except Exception:
                        pass  # Skip tables that fail to parse
        except Exception:
            pass  # Graceful degradation if doc iteration fails

        return tables
