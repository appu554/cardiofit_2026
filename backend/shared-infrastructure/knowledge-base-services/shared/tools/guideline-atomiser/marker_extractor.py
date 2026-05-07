"""
L1: PDF Parsing with Marker v1.10 - Full Provenance Preservation.

This module provides PDF extraction using Marker with complete provenance
tracking including byte ranges, page coordinates, and confidence scores.

Key Principle: Every extracted text must trace back to its exact source location
for clinical audit and regulatory compliance.

Usage:
    from marker_extractor import MarkerExtractor, extract_pdf_with_provenance

    extractor = MarkerExtractor()
    result = extractor.extract(pdf_path="kdigo_2022_diabetes.pdf")

    # Access structured content with provenance
    for block in result.blocks:
        print(f"Page {block.page_number}: {block.text[:50]}...")
        print(f"  Confidence: {block.confidence}")
        print(f"  Bbox: {block.bbox}")
"""

import json
import re
from pathlib import Path
from dataclasses import dataclass, field
from typing import Optional, Literal, Dict, List, Tuple, Union
from datetime import datetime, timezone
import hashlib


@dataclass
class BoundingBox:
    """Page coordinates for a text block."""
    x0: float  # Left edge
    y0: float  # Top edge
    x1: float  # Right edge
    y1: float  # Bottom edge
    page_width: float = 612.0  # Standard letter width
    page_height: float = 792.0  # Standard letter height

    def to_dict(self) -> dict:
        return {
            "x0": self.x0,
            "y0": self.y0,
            "x1": self.x1,
            "y1": self.y1,
            "page_width": self.page_width,
            "page_height": self.page_height,
        }


@dataclass
class TextBlock:
    """A block of extracted text with full provenance."""
    text: str
    page_number: int
    block_type: Literal["text", "table", "heading", "list", "code", "image_caption"]
    bbox: Optional[BoundingBox] = None
    confidence: float = 1.0  # OCR confidence if applicable
    byte_range_start: Optional[int] = None
    byte_range_end: Optional[int] = None
    font_info: Optional[dict] = None
    is_bold: bool = False
    is_italic: bool = False
    heading_level: Optional[int] = None  # 1-6 for headings
    table_data: Optional[list[list[str]]] = None  # For table blocks
    seed: Optional[int] = None  # For reproducibility
    # V5 MonkeyOCR: DoclayoutYOLO region type for specialist routing downstream
    region_type: Optional[str] = None

    def to_dict(self) -> dict:
        result = {
            "text": self.text,
            "page_number": self.page_number,
            "block_type": self.block_type,
            "confidence": self.confidence,
            "is_bold": self.is_bold,
            "is_italic": self.is_italic,
        }
        if self.bbox:
            result["bbox"] = self.bbox.to_dict()
        if self.byte_range_start is not None:
            result["byte_range"] = {
                "start": self.byte_range_start,
                "end": self.byte_range_end,
            }
        if self.font_info:
            result["font_info"] = self.font_info
        if self.heading_level:
            result["heading_level"] = self.heading_level
        if self.table_data:
            result["table_data"] = self.table_data
        if self.seed is not None:
            result["seed"] = self.seed
        return result


@dataclass
class TableBlock:
    """Specialized structure for extracted tables."""
    headers: list[str]
    rows: list[list[str]]
    page_number: int
    bbox: Optional[BoundingBox] = None
    confidence: float = 1.0
    caption: Optional[str] = None
    table_index: int = 0  # Index within the document
    # V5 MonkeyOCR: per-cell data with individual bboxes from Qwen2.5-VL spans.
    # Each entry: {"text", "row_idx", "col_idx", "bbox": [x0,y0,x1,y1], "confidence"}
    cell_data: Optional[list[dict]] = None
    # DoclayoutYOLO region type — used by Channel D to route to specialist paths
    region_type: str = "table"

    def to_markdown(self) -> str:
        """Convert table to markdown format."""
        if not self.headers:
            return ""

        lines = []
        # Header row
        lines.append("| " + " | ".join(self.headers) + " |")
        # Separator
        lines.append("| " + " | ".join(["---"] * len(self.headers)) + " |")
        # Data rows
        for row in self.rows:
            # Pad row if needed
            padded = row + [""] * (len(self.headers) - len(row))
            lines.append("| " + " | ".join(padded[:len(self.headers)]) + " |")

        return "\n".join(lines)

    def to_dict(self) -> dict:
        return {
            "headers": self.headers,
            "rows": self.rows,
            "page_number": self.page_number,
            "bbox": self.bbox.to_dict() if self.bbox else None,
            "confidence": self.confidence,
            "caption": self.caption,
            "table_index": self.table_index,
            "markdown": self.to_markdown(),
        }


@dataclass
class ExtractionProvenance:
    """Complete provenance information for an extraction."""
    source_file: str
    source_hash: str  # SHA-256 of source file
    extraction_timestamp: str
    extractor_version: str
    marker_version: str
    seed: int  # For reproducibility
    total_pages: int
    extraction_params: dict = field(default_factory=dict)

    def to_dict(self) -> dict:
        return {
            "source_file": self.source_file,
            "source_hash": self.source_hash,
            "extraction_timestamp": self.extraction_timestamp,
            "extractor_version": self.extractor_version,
            "marker_version": self.marker_version,
            "seed": self.seed,
            "total_pages": self.total_pages,
            "extraction_params": self.extraction_params,
        }


@dataclass
class ExtractionResult:
    """Complete extraction result with provenance."""
    blocks: list[TextBlock]
    tables: list[TableBlock]
    markdown: str  # Full document as markdown
    provenance: ExtractionProvenance

    def to_dict(self) -> dict:
        return {
            "blocks": [b.to_dict() for b in self.blocks],
            "tables": [t.to_dict() for t in self.tables],
            "markdown": self.markdown,
            "provenance": self.provenance.to_dict(),
        }

    def to_json(self, indent: int = 2) -> str:
        return json.dumps(self.to_dict(), indent=indent)

    def get_page_content(self, page_number: int) -> str:
        """Get all text content from a specific page."""
        return "\n\n".join(
            block.text for block in self.blocks
            if block.page_number == page_number
        )

    def get_tables_by_page(self, page_number: int) -> list[TableBlock]:
        """Get all tables from a specific page."""
        return [t for t in self.tables if t.page_number == page_number]

    def find_section(self, heading_text: str) -> list[TextBlock]:
        """Find blocks belonging to a specific section by heading."""
        result = []
        in_section = False
        section_level = None

        for block in self.blocks:
            if block.block_type == "heading":
                if heading_text.lower() in block.text.lower():
                    in_section = True
                    section_level = block.heading_level or 1
                    result.append(block)
                elif in_section and block.heading_level and block.heading_level <= section_level:
                    # Hit a heading at same or higher level, section ends
                    break
            elif in_section:
                result.append(block)

        return result


class ClinicalOCRPostProcessor:
    """
    Post-processor to fix common OCR errors in clinical text.

    OCR often misreads clinical terms, drug names, units, and numbers.
    This post-processor applies curated corrections specific to medical/clinical
    guidelines like KDIGO, ADA, and FDA SPL labels.

    Key correction categories:
    1. Medical units (mL/min/1.73m², mg/dL, mEq/L)
    2. Clinical terms (eGFR, HbA1c, UACR)
    3. Drug names (common misspellings)
    4. Number/letter confusion (0/O, 1/l/I, 5/S)
    5. Superscript/subscript normalization (m², CO₂)
    """

    # ═══════════════════════════════════════════════════════════════════════
    # OCR CORRECTION PATTERNS
    # Format: (pattern, replacement, description)
    # ═══════════════════════════════════════════════════════════════════════

    # Medical unit corrections
    UNIT_CORRECTIONS: List[Tuple[str, str, str]] = [
        # eGFR units - common OCR errors
        (r'mL/min/1\.73\s*m[²2A]', 'mL/min/1.73m²', 'eGFR unit normalization'),
        (r'mL/min/1\.73\s*m\s*2', 'mL/min/1.73m²', 'eGFR unit with space'),
        (r'ml/min/1\.73\s*m[²2]', 'mL/min/1.73m²', 'lowercase mL fix'),
        (r'mL/min/1[,.]73m[²2]', 'mL/min/1.73m²', 'comma vs decimal'),
        (r'm[lL]/min/1\.73\s*m\^2', 'mL/min/1.73m²', 'caret superscript'),
        (r'mL/min per 1\.73\s*m[²2]', 'mL/min/1.73m²', 'per variant'),

        # Other common units
        (r'mg/d[lL]', 'mg/dL', 'milligram per deciliter'),
        (r'mEq/[lL]', 'mEq/L', 'milliequivalent per liter'),
        (r'mmol/[lL]', 'mmol/L', 'millimole per liter'),
        (r'µg/m[lL]', 'µg/mL', 'microgram per milliliter'),
        (r'mcg/m[lL]', 'mcg/mL', 'microgram per milliliter alt'),
        (r'mg/m[²2]', 'mg/m²', 'BSA-based dosing'),
        (r'mg/kg/d(ay)?', 'mg/kg/day', 'daily weight-based dosing'),
    ]

    # Clinical term corrections
    CLINICAL_TERM_CORRECTIONS: List[Tuple[str, str, str]] = [
        # eGFR variants
        (r'\be[- ]?GFR\b', 'eGFR', 'eGFR normalization'),
        (r'\bEGFR\b', 'eGFR', 'EGFR to eGFR'),
        (r'\begfr\b', 'eGFR', 'lowercase egfr'),
        (r'\beGER\b', 'eGFR', 'OCR misread F as E'),
        (r'\beGF R\b', 'eGFR', 'space in eGFR'),
        (r'\bGFR\b(?!\s*<|\s*>|\s*≤|\s*≥|\s*=)', 'eGFR', 'GFR without comparison'),

        # CrCl variants
        (r'\bCrCl\b', 'CrCl', 'creatinine clearance'),
        (r'\bCrCI\b', 'CrCl', 'OCR misread l as I'),
        (r'\bCRC[lL]\b', 'CrCl', 'case variations'),

        # HbA1c variants
        (r'\bHbA[1l]c\b', 'HbA1c', 'HbA1c normalization'),
        (r'\bHBA1C\b', 'HbA1c', 'uppercase variant'),
        (r'\bA[1l]C\b', 'A1C', 'A1C normalization'),
        (r'\bHgbA1c\b', 'HbA1c', 'hemoglobin variant'),

        # UACR
        (r'\bUACR\b', 'UACR', 'urine albumin-creatinine ratio'),
        (r'\bACR\b', 'ACR', 'albumin-creatinine ratio'),

        # Blood pressure
        (r'\bBP\b', 'BP', 'blood pressure'),
        (r'\bSBP\b', 'SBP', 'systolic BP'),
        (r'\bDBP\b', 'DBP', 'diastolic BP'),

        # Heart conditions
        (r'\bHFrEF\b', 'HFrEF', 'HF reduced ejection fraction'),
        (r'\bHFpEF\b', 'HFpEF', 'HF preserved ejection fraction'),
        (r'\bHFmrEF\b', 'HFmrEF', 'HF mid-range EF'),

        # Kidney disease stages
        (r'\bCKD\s+stage\s+([1-5])\b', r'CKD stage \1', 'CKD stage'),
        (r'\bCKD\s*([1-5])\b', r'CKD stage \1', 'CKD stage shorthand'),
        (r'\bESKD\b', 'ESKD', 'end-stage kidney disease'),
        (r'\bESRD\b', 'ESRD', 'end-stage renal disease'),
        (r'\bAKI\b', 'AKI', 'acute kidney injury'),
    ]

    # Drug name corrections (common OCR errors)
    DRUG_NAME_CORRECTIONS: List[Tuple[str, str, str]] = [
        # SGLT2 inhibitors
        (r'\b[Dd]apag[1l]if[1l]ozin\b', 'dapagliflozin', 'dapagliflozin l/1 fix'),
        (r'\b[Ee]mpag[1l]if[1l]ozin\b', 'empagliflozin', 'empagliflozin l/1 fix'),
        (r'\b[Cc]anag[1l]if[1l]ozin\b', 'canagliflozin', 'canagliflozin l/1 fix'),
        (r'\b[Ee]rtug[1l]if[1l]ozin\b', 'ertugliflozin', 'ertugliflozin l/1 fix'),
        (r'\b[Ss]otag[1l]if[1l]ozin\b', 'sotagliflozin', 'sotagliflozin l/1 fix'),

        # Metformin
        (r'\b[Mm]etf[o0]rmin\b', 'metformin', 'metformin o/0 fix'),
        (r'\b[Mm]etfonnin\b', 'metformin', 'metformin OCR r→n'),

        # Finerenone
        (r'\b[Ff]ineren[o0]ne\b', 'finerenone', 'finerenone fix'),
        (r'\b[Ff]ineranone\b', 'finerenone', 'finerenone vowel fix'),

        # ACE inhibitors
        (r'\b[Ll]isin[o0]pri[1l]\b', 'lisinopril', 'lisinopril fixes'),
        (r'\b[Ee]na[1l]apri[1l]\b', 'enalapril', 'enalapril fixes'),
        (r'\b[Rr]amipri[1l]\b', 'ramipril', 'ramipril l/1 fix'),

        # ARBs
        (r'\b[Ll][o0]sartan\b', 'losartan', 'losartan o/0 fix'),
        (r'\b[Vv]a[1l]sartan\b', 'valsartan', 'valsartan l/1 fix'),
        (r'\b[Ii]rbesartan\b', 'irbesartan', 'irbesartan'),

        # GLP-1 agonists
        (r'\b[Ss]emag[1l]utide\b', 'semaglutide', 'semaglutide l/1 fix'),
        (r'\b[Ll]irag[1l]utide\b', 'liraglutide', 'liraglutide l/1 fix'),
        (r'\b[Dd]u[1l]ag[1l]utide\b', 'dulaglutide', 'dulaglutide l/1 fix'),

        # MRAs
        (r'\b[Ss]pir[o0]n[o0][1l]act[o0]ne\b', 'spironolactone', 'spironolactone fix'),
        (r'\b[Ee]p[1l]eren[o0]ne\b', 'eplerenone', 'eplerenone l/1 fix'),
    ]

    # Number/letter confusion patterns
    NUMBER_CORRECTIONS: List[Tuple[str, str, str]] = [
        # Common number confusions in thresholds
        (r'\beGFR\s*<\s*3O\b', 'eGFR < 30', 'O→0 in eGFR < 30'),
        (r'\beGFR\s*<\s*2O\b', 'eGFR < 20', 'O→0 in eGFR < 20'),
        (r'\beGFR\s*<\s*45\b', 'eGFR < 45', 'eGFR < 45'),  # Confirm correct
        (r'\beGFR\s*<\s*6O\b', 'eGFR < 60', 'O→0 in eGFR < 60'),
        (r'\beGFR\s+3O\s*-\s*44\b', 'eGFR 30-44', 'O→0 in range'),
        (r'\beGFR\s+3O\s*-\s*45\b', 'eGFR 30-45', 'O→0 in range'),
        (r'\beGFR\s+45\s*-\s*6O\b', 'eGFR 45-60', 'O→0 in range'),

        # Potassium thresholds
        (r'\bK\+?\s*>\s*5[.,]5\b', 'K+ > 5.5', 'potassium threshold'),
        (r'\bK\+?\s*>\s*6[.,]O\b', 'K+ > 6.0', 'O→0 in potassium'),
        (r'\bpotassium\s*>\s*5[.,]5\b', 'potassium > 5.5', 'potassium threshold'),
    ]

    # Superscript/subscript normalization
    SUPERSCRIPT_CORRECTIONS: List[Tuple[str, str, str]] = [
        (r'm\^2', 'm²', 'caret to superscript 2'),
        (r'm2(?!\d)', 'm²', 'm2 to m²'),
        (r'CO2', 'CO₂', 'CO2 subscript'),
        (r'H2O', 'H₂O', 'H2O subscript'),
        (r'O2', 'O₂', 'O2 subscript'),
    ]

    # ═══════════════════════════════════════════════════════════════════════
    # LaTeX STRIPPING — MonkeyOCR produces LaTeX notation for math/symbols
    # Must run BEFORE the regex-based OCR corrections above.
    # ═══════════════════════════════════════════════════════════════════════

    _LATEX_SUPERSCRIPT_RE = re.compile(r'\$\^\{([^}]*)\}\$')
    _LATEX_SUBSCRIPT_RE = re.compile(r'\$_\{([^}]*)\}\$')
    _LATEX_GEQ_RE = re.compile(r'\$\\geq\$')
    _LATEX_LEQ_RE = re.compile(r'\$\\leq\$')
    _LATEX_TIMES_RE = re.compile(r'\$\\times\$')
    _LATEX_PM_RE = re.compile(r'\$\\pm\$')
    _LATEX_APPROX_RE = re.compile(r'\$\\approx\$')
    _LATEX_INLINE_RE = re.compile(r'\$([^$]+)\$')

    _SUBSCRIPT_MAP = {
        "0": "\u2080", "1": "\u2081", "2": "\u2082", "3": "\u2083",
        "4": "\u2084", "5": "\u2085", "6": "\u2086", "7": "\u2087",
        "8": "\u2088", "9": "\u2089",
    }
    _SUPERSCRIPT_MAP = {
        "0": "\u2070", "1": "\u00B9", "2": "\u00B2", "3": "\u00B3",
        "4": "\u2074", "5": "\u2075", "6": "\u2076", "7": "\u2077",
        "8": "\u2078", "9": "\u2079",
    }

    @classmethod
    def strip_latex(cls, text: str) -> str:
        """
        Strip LaTeX notation and convert to Unicode equivalents.

        Designed for MonkeyOCR output which uses LaTeX for superscripts,
        subscripts, and comparators.  Must run BEFORE the regex-based OCR
        correction patterns so that ``$\\geq$30`` becomes ``≥30`` before
        the eGFR threshold pattern tries to match.

        Conversions:
          $^{N}$     → superscript Unicode or plain N
          $_{N}$     → subscript Unicode ($_{2}$ → ₂)
          $\\geq$    → ≥
          $\\leq$    → ≤
          $\\times$  → ×
          $\\pm$     → ±
          $\\approx$ → ≈
          1.73m$^{2}$ → 1.73m²
        """
        if "$" not in text:
            return text

        text = cls._LATEX_GEQ_RE.sub("≥", text)
        text = cls._LATEX_LEQ_RE.sub("≤", text)
        text = cls._LATEX_TIMES_RE.sub("×", text)
        text = cls._LATEX_PM_RE.sub("±", text)
        text = cls._LATEX_APPROX_RE.sub("≈", text)

        def _replace_superscript(m):
            inner = m.group(1)
            if len(inner) == 1 and inner in cls._SUPERSCRIPT_MAP:
                return cls._SUPERSCRIPT_MAP[inner]
            return inner

        text = cls._LATEX_SUPERSCRIPT_RE.sub(_replace_superscript, text)

        def _replace_subscript(m):
            inner = m.group(1)
            return "".join(cls._SUBSCRIPT_MAP.get(ch, ch) for ch in inner)

        text = cls._LATEX_SUBSCRIPT_RE.sub(_replace_subscript, text)

        text = cls._LATEX_INLINE_RE.sub(lambda m: m.group(1), text)
        return text

    def __init__(
        self,
        enable_unit_fixes: bool = True,
        enable_term_fixes: bool = True,
        enable_drug_fixes: bool = True,
        enable_number_fixes: bool = True,
        enable_superscript_fixes: bool = True,
        enable_latex_stripping: bool = True,
        custom_corrections: Optional[List[Tuple[str, str, str]]] = None,
    ):
        """
        Initialize the OCR post-processor.

        Args:
            enable_unit_fixes: Fix medical unit OCR errors
            enable_term_fixes: Fix clinical term OCR errors
            enable_drug_fixes: Fix drug name OCR errors
            enable_number_fixes: Fix number/letter confusion
            enable_superscript_fixes: Normalize superscripts/subscripts
            enable_latex_stripping: Strip LaTeX notation (MonkeyOCR output)
            custom_corrections: Additional custom corrections (pattern, replacement, desc)
        """
        self.enable_unit_fixes = enable_unit_fixes
        self.enable_term_fixes = enable_term_fixes
        self.enable_drug_fixes = enable_drug_fixes
        self.enable_number_fixes = enable_number_fixes
        self.enable_superscript_fixes = enable_superscript_fixes
        self.enable_latex_stripping = enable_latex_stripping
        self.custom_corrections = custom_corrections or []

        # Compile all regex patterns for efficiency
        self._compiled_patterns: List[Tuple[re.Pattern, str, str]] = []
        self._compile_patterns()

    def _compile_patterns(self):
        """Compile all enabled correction patterns."""
        all_corrections = []

        if self.enable_unit_fixes:
            all_corrections.extend(self.UNIT_CORRECTIONS)
        if self.enable_term_fixes:
            all_corrections.extend(self.CLINICAL_TERM_CORRECTIONS)
        if self.enable_drug_fixes:
            all_corrections.extend(self.DRUG_NAME_CORRECTIONS)
        if self.enable_number_fixes:
            all_corrections.extend(self.NUMBER_CORRECTIONS)
        if self.enable_superscript_fixes:
            all_corrections.extend(self.SUPERSCRIPT_CORRECTIONS)

        all_corrections.extend(self.custom_corrections)

        for pattern, replacement, desc in all_corrections:
            try:
                compiled = re.compile(pattern, re.IGNORECASE)
                self._compiled_patterns.append((compiled, replacement, desc))
            except re.error as e:
                print(f"⚠️ Invalid regex pattern '{pattern}': {e}")

    def fix_text(self, text: str) -> Tuple[str, List[Dict]]:
        """
        Apply all OCR corrections to a text string.

        Args:
            text: Input text to correct

        Returns:
            Tuple of (corrected_text, list of corrections applied)
        """
        corrections_applied = []

        # Apply LaTeX stripping first (MonkeyOCR produces $\geq$, $^{2}$, etc.)
        if self.enable_latex_stripping:
            stripped = self.strip_latex(text)
            if stripped != text:
                corrections_applied.append({
                    "original": "[LaTeX notation]",
                    "corrected": "[Unicode equivalents]",
                    "description": "LaTeX stripping (MonkeyOCR)",
                    "position": 0,
                })
                text = stripped

        for pattern, replacement, desc in self._compiled_patterns:
            matches = list(pattern.finditer(text))
            if matches:
                for match in matches:
                    corrections_applied.append({
                        "original": match.group(),
                        "corrected": pattern.sub(replacement, match.group()),
                        "description": desc,
                        "position": match.start(),
                    })
                text = pattern.sub(replacement, text)

        return text, corrections_applied

    def process_blocks(
        self,
        blocks: List['TextBlock'],
    ) -> Tuple[List['TextBlock'], Dict]:
        """
        Apply OCR corrections to a list of TextBlocks.

        Args:
            blocks: List of TextBlock objects to correct

        Returns:
            Tuple of (corrected_blocks, correction_summary)
        """
        total_corrections = 0
        correction_details = []

        corrected_blocks = []
        for block in blocks:
            corrected_text, corrections = self.fix_text(block.text)

            if corrections:
                total_corrections += len(corrections)
                correction_details.extend([
                    {**c, "page": block.page_number, "block_type": block.block_type}
                    for c in corrections
                ])

            # Create new block with corrected text
            corrected_block = TextBlock(
                text=corrected_text,
                page_number=block.page_number,
                block_type=block.block_type,
                bbox=block.bbox,
                confidence=block.confidence,
                byte_range_start=block.byte_range_start,
                byte_range_end=block.byte_range_end,
                font_info=block.font_info,
                is_bold=block.is_bold,
                is_italic=block.is_italic,
                heading_level=block.heading_level,
                table_data=block.table_data,
                seed=block.seed,
            )
            corrected_blocks.append(corrected_block)

        summary = {
            "total_corrections": total_corrections,
            "blocks_affected": len(set(c["page"] for c in correction_details)),
            "correction_types": {},
            "details": correction_details,
        }

        # Count correction types
        for c in correction_details:
            desc = c["description"]
            summary["correction_types"][desc] = summary["correction_types"].get(desc, 0) + 1

        return corrected_blocks, summary

    def process_markdown(self, markdown: str) -> Tuple[str, Dict]:
        """
        Apply OCR corrections to markdown text.

        Args:
            markdown: Markdown text to correct

        Returns:
            Tuple of (corrected_markdown, correction_summary)
        """
        corrected, corrections = self.fix_text(markdown)

        summary = {
            "total_corrections": len(corrections),
            "correction_types": {},
            "details": corrections,
        }

        for c in corrections:
            desc = c["description"]
            summary["correction_types"][desc] = summary["correction_types"].get(desc, 0) + 1

        return corrected, summary


class MarkerExtractor:
    """
    L1 PDF Extractor using Marker v1.10.

    Marker provides high-quality PDF extraction with:
    - Accurate table detection and extraction
    - OCR fallback for scanned documents
    - Structural understanding (headings, lists, code blocks)
    - Font and style preservation
    - OCR post-processing for clinical term correction
    """

    VERSION = "1.1.0"  # Updated for OCR post-processor

    def __init__(
        self,
        seed: int = 42,
        enable_ocr: bool = True,
        extract_images: bool = False,
        table_strategy: Literal["auto", "ocr", "native"] = "auto",
        enable_ocr_postprocessing: bool = True,
        ocr_postprocessor: Optional[ClinicalOCRPostProcessor] = None,
    ):
        """
        Initialize the Marker extractor.

        Args:
            seed: Random seed for reproducibility
            enable_ocr: Enable OCR for scanned documents
            extract_images: Extract image captions and alt text
            table_strategy: Strategy for table extraction
            enable_ocr_postprocessing: Enable clinical OCR error correction
            ocr_postprocessor: Custom OCR post-processor (uses default if None)
        """
        self.seed = seed
        self.enable_ocr = enable_ocr
        self.extract_images = extract_images
        self.table_strategy = table_strategy
        self.enable_ocr_postprocessing = enable_ocr_postprocessing
        self._marker_version = self._get_marker_version()

        # Initialize OCR post-processor
        if enable_ocr_postprocessing:
            self.ocr_postprocessor = ocr_postprocessor or ClinicalOCRPostProcessor()
        else:
            self.ocr_postprocessor = None

    def _get_marker_version(self) -> str:
        """Get the installed Marker version."""
        try:
            import marker
            return getattr(marker, "__version__", "unknown")
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
        Extract content from a PDF with full provenance.

        Args:
            pdf_path: Path to the PDF file
            page_range: Optional (start, end) page range (1-indexed, inclusive)

        Returns:
            ExtractionResult with blocks, tables, markdown, and provenance
        """
        pdf_path = Path(pdf_path)
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF not found: {pdf_path}")

        # Use Marker for extraction (models pre-cached in Docker)
        return self._extract_with_marker(pdf_path, page_range)

    def _extract_with_marker(
        self,
        pdf_path: Path,
        page_range: Optional[tuple[int, int]] = None,
    ) -> ExtractionResult:
        """Extract using Marker library v1.10+."""
        from marker.converters.pdf import PdfConverter
        from marker.models import create_model_dict
        from marker.config.parser import ConfigParser
        from marker.output import text_from_rendered

        # Configure Marker v1.10 API
        config = {
            "output_format": "markdown",
            "paginate_output": True,
            "extract_images": self.extract_images,
        }
        config_parser = ConfigParser(config)

        # Create model dict (loads ML models) - REQUIRED for v1.10+
        artifact_dict = create_model_dict()

        # Initialize converter with required artifact_dict
        converter = PdfConverter(
            config=config_parser.generate_config_dict(),
            artifact_dict=artifact_dict,
            processor_list=config_parser.get_processors(),
            renderer=config_parser.get_renderer(),
        )

        # Convert PDF
        rendered = converter(str(pdf_path))
        markdown_text, _, images = text_from_rendered(rendered)

        # Extract metadata from rendered object
        metadata = {}
        if hasattr(rendered, 'metadata'):
            metadata = rendered.metadata if isinstance(rendered.metadata, dict) else {}

        # Parse the markdown into blocks
        blocks = self._parse_markdown_to_blocks(markdown_text, metadata)
        tables = self._extract_tables_from_blocks(blocks)

        # Apply OCR post-processing if enabled
        ocr_correction_summary = None
        if self.ocr_postprocessor:
            blocks, block_summary = self.ocr_postprocessor.process_blocks(blocks)
            markdown_text, md_summary = self.ocr_postprocessor.process_markdown(markdown_text)
            ocr_correction_summary = {
                "blocks": block_summary,
                "markdown": md_summary,
                "total_corrections": block_summary["total_corrections"] + md_summary["total_corrections"],
            }

        # Extract total pages from metadata or count from blocks
        total_pages = 1
        if metadata and "total_pages" in metadata:
            total_pages = metadata["total_pages"]
        elif blocks:
            total_pages = len(set(b.page_number for b in blocks))

        # Build provenance
        extraction_params = {
            "enable_ocr": self.enable_ocr,
            "extract_images": self.extract_images,
            "table_strategy": self.table_strategy,
            "page_range": list(page_range) if page_range else None,
            "ocr_postprocessing_enabled": self.enable_ocr_postprocessing,
        }
        if ocr_correction_summary:
            extraction_params["ocr_corrections"] = {
                "total": ocr_correction_summary["total_corrections"],
                "by_type": ocr_correction_summary["blocks"].get("correction_types", {}),
            }

        provenance = ExtractionProvenance(
            source_file=str(pdf_path),
            source_hash=self._compute_file_hash(pdf_path),
            extraction_timestamp=datetime.now(timezone.utc).isoformat(),
            extractor_version=self.VERSION,
            marker_version=self._marker_version,
            seed=self.seed,
            total_pages=total_pages,
            extraction_params=extraction_params,
        )

        return ExtractionResult(
            blocks=blocks,
            tables=tables,
            markdown=markdown_text,
            provenance=provenance,
        )

    def _extract_with_pymupdf(
        self,
        pdf_path: Path,
        page_range: Optional[tuple[int, int]] = None,
    ) -> ExtractionResult:
        """
        Extract using PyMuPDF (fitz) as lightweight fallback.

        This doesn't require large ML model downloads and provides
        good text extraction for most clinical guidelines.
        """
        try:
            import fitz  # PyMuPDF
        except ImportError:
            print("⚠️  PyMuPDF not installed, falling back to mock extraction")
            return self._extract_mock(pdf_path, page_range)

        doc = fitz.open(pdf_path)
        total_pages = len(doc)

        # Apply page range if specified
        start_page = 0
        end_page = total_pages
        if page_range:
            start_page = max(0, page_range[0] - 1)
            end_page = min(total_pages, page_range[1])

        blocks = []
        tables = []
        markdown_parts = []

        for page_num in range(start_page, end_page):
            page = doc[page_num]
            page_number = page_num + 1  # 1-indexed

            # Get page dimensions
            rect = page.rect
            page_width = rect.width
            page_height = rect.height

            # Extract text with position info
            text_dict = page.get_text("dict")

            markdown_parts.append(f"\n<!-- PAGE {page_number} -->\n")

            for block in text_dict.get("blocks", []):
                if block.get("type") == 0:  # Text block
                    # Get bounding box
                    bbox_coords = block.get("bbox", [0, 0, 0, 0])
                    bbox = BoundingBox(
                        x0=bbox_coords[0],
                        y0=bbox_coords[1],
                        x1=bbox_coords[2],
                        y1=bbox_coords[3],
                        page_width=page_width,
                        page_height=page_height,
                    )

                    # Extract text from lines
                    block_text = ""
                    for line in block.get("lines", []):
                        for span in line.get("spans", []):
                            block_text += span.get("text", "") + " "
                        block_text += "\n"

                    block_text = block_text.strip()
                    if not block_text:
                        continue

                    # Detect block type based on font size/style
                    block_type = "text"
                    heading_level = None
                    is_bold = False

                    # Check if heading based on font size
                    if block.get("lines"):
                        first_line = block["lines"][0]
                        if first_line.get("spans"):
                            first_span = first_line["spans"][0]
                            font_size = first_span.get("size", 12)
                            font_flags = first_span.get("flags", 0)
                            is_bold = bool(font_flags & 2)  # Bold flag

                            if font_size >= 16:
                                block_type = "heading"
                                heading_level = 1
                            elif font_size >= 14:
                                block_type = "heading"
                                heading_level = 2
                            elif font_size >= 12 and is_bold:
                                block_type = "heading"
                                heading_level = 3

                    text_block = TextBlock(
                        text=block_text,
                        page_number=page_number,
                        block_type=block_type,
                        confidence=0.95,  # PyMuPDF extraction is generally accurate
                        bbox=bbox,
                        heading_level=heading_level,
                        is_bold=is_bold,
                    )
                    blocks.append(text_block)

                    # Add to markdown
                    if block_type == "heading" and heading_level:
                        markdown_parts.append(f"\n{'#' * heading_level} {block_text}\n")
                    else:
                        markdown_parts.append(f"{block_text}\n\n")

            # Try to extract tables
            try:
                page_tables = page.find_tables()
                for table_idx, table in enumerate(page_tables):
                    try:
                        table_data = table.extract()
                        if table_data and len(table_data) > 1:  # Has header + data
                            headers = [str(h).strip() if h else "" for h in table_data[0]]
                            rows = []
                            for row in table_data[1:]:
                                rows.append([str(c).strip() if c else "" for c in row])

                            parsed_table = ParsedTable(
                                headers=headers,
                                rows=rows,
                                page_number=page_number,
                                table_index=table_idx,
                            )
                            tables.append(parsed_table)

                            # Add table to markdown
                            markdown_parts.append(f"\n{parsed_table.to_markdown()}\n")
                    except Exception:
                        pass  # Skip problematic tables
            except Exception:
                pass  # Table extraction not available in all PyMuPDF versions

        doc.close()

        markdown_text = "".join(markdown_parts)

        # Apply OCR post-processing if enabled
        ocr_correction_summary = None
        if self.ocr_postprocessor:
            blocks, block_summary = self.ocr_postprocessor.process_blocks(blocks)
            markdown_text, md_summary = self.ocr_postprocessor.process_markdown(markdown_text)
            ocr_correction_summary = {
                "blocks": block_summary,
                "markdown": md_summary,
                "total_corrections": block_summary["total_corrections"] + md_summary["total_corrections"],
            }

        # Build provenance
        extraction_params = {
            "enable_ocr": False,
            "extract_images": False,
            "table_strategy": "pymupdf",
            "page_range": list(page_range) if page_range else None,
            "fallback": "pymupdf",
            "ocr_postprocessing_enabled": self.enable_ocr_postprocessing,
        }
        if ocr_correction_summary:
            extraction_params["ocr_corrections"] = {
                "total": ocr_correction_summary["total_corrections"],
                "by_type": ocr_correction_summary["blocks"].get("correction_types", {}),
            }

        provenance = ExtractionProvenance(
            source_file=str(pdf_path),
            source_hash=self._compute_file_hash(pdf_path),
            extraction_timestamp=datetime.now(timezone.utc).isoformat(),
            extractor_version=self.VERSION,
            marker_version="pymupdf-fallback",
            seed=self.seed,
            total_pages=total_pages,
            extraction_params=extraction_params,
        )

        return ExtractionResult(
            blocks=blocks,
            tables=tables,
            markdown=markdown_text,
            provenance=provenance,
        )

    def _extract_mock(
        self,
        pdf_path: Path,
        page_range: Optional[tuple[int, int]] = None,
    ) -> ExtractionResult:
        """
        Mock extraction for when Marker is not installed.

        This provides a development/testing fallback with proper structure.
        """
        # Create a minimal mock extraction
        blocks = [
            TextBlock(
                text=f"[Mock extraction from {pdf_path.name}]",
                page_number=1,
                block_type="text",
                confidence=0.0,  # Indicate this is mock data
            ),
            TextBlock(
                text="Install marker-pdf>=1.10.0 for actual PDF extraction",
                page_number=1,
                block_type="text",
                confidence=0.0,
            ),
        ]

        provenance = ExtractionProvenance(
            source_file=str(pdf_path),
            source_hash=self._compute_file_hash(pdf_path),
            extraction_timestamp=datetime.now(timezone.utc).isoformat(),
            extractor_version=self.VERSION,
            marker_version="mock",
            seed=self.seed,
            total_pages=1,
            extraction_params={
                "mock": True,
                "reason": "marker-pdf not installed",
            },
        )

        return ExtractionResult(
            blocks=blocks,
            tables=[],
            markdown=f"# Mock Extraction\n\nInstall marker-pdf for actual extraction.",
            provenance=provenance,
        )

    def _parse_markdown_to_blocks(
        self,
        markdown_text: str,
        metadata: dict,
    ) -> list[TextBlock]:
        """Parse markdown output into structured blocks."""
        blocks = []
        current_page = 1
        byte_offset = 0

        for line in markdown_text.split("\n"):
            if not line.strip():
                byte_offset += len(line) + 1
                continue

            # Detect page breaks (Marker uses specific markers)
            if line.startswith("<!-- PAGE "):
                try:
                    current_page = int(line.split()[2])
                except (IndexError, ValueError):
                    pass
                byte_offset += len(line) + 1
                continue

            # Detect block type
            block_type: Literal["text", "table", "heading", "list", "code", "image_caption"] = "text"
            heading_level = None
            is_bold = False
            is_italic = False

            if line.startswith("#"):
                block_type = "heading"
                heading_level = len(line.split()[0])  # Count #s
            elif line.startswith("- ") or line.startswith("* "):
                block_type = "list"
            elif line.startswith("```"):
                block_type = "code"
            elif line.startswith("|"):
                block_type = "table"
            elif line.startswith("**") and line.endswith("**"):
                is_bold = True
            elif line.startswith("*") and line.endswith("*"):
                is_italic = True

            blocks.append(TextBlock(
                text=line,
                page_number=current_page,
                block_type=block_type,
                byte_range_start=byte_offset,
                byte_range_end=byte_offset + len(line),
                heading_level=heading_level,
                is_bold=is_bold,
                is_italic=is_italic,
                seed=self.seed,
            ))

            byte_offset += len(line) + 1

        return blocks

    def _extract_tables_from_blocks(self, blocks: list[TextBlock]) -> list[TableBlock]:
        """Extract table structures from blocks."""
        tables = []
        current_table_rows = []
        current_page = 1
        table_index = 0

        for block in blocks:
            if block.block_type == "table":
                current_page = block.page_number
                # Parse markdown table row
                cells = [cell.strip() for cell in block.text.split("|")[1:-1]]
                if cells and not all(c.startswith("-") for c in cells):
                    current_table_rows.append(cells)
            else:
                if current_table_rows:
                    # End of table, create TableBlock
                    headers = current_table_rows[0] if current_table_rows else []
                    rows = current_table_rows[1:] if len(current_table_rows) > 1 else []

                    tables.append(TableBlock(
                        headers=headers,
                        rows=rows,
                        page_number=current_page,
                        table_index=table_index,
                    ))

                    current_table_rows = []
                    table_index += 1

        # Handle final table if document ends with one
        if current_table_rows:
            headers = current_table_rows[0] if current_table_rows else []
            rows = current_table_rows[1:] if len(current_table_rows) > 1 else []

            tables.append(TableBlock(
                headers=headers,
                rows=rows,
                page_number=current_page,
                table_index=table_index,
            ))

        return tables


def extract_pdf_with_provenance(
    pdf_path: Union[str, Path],
    seed: int = 42,
    page_range: Optional[tuple[int, int]] = None,
) -> ExtractionResult:
    """
    Convenience function for single PDF extraction.

    Args:
        pdf_path: Path to the PDF file
        seed: Random seed for reproducibility
        page_range: Optional (start, end) page range

    Returns:
        ExtractionResult with full provenance
    """
    extractor = MarkerExtractor(seed=seed)
    return extractor.extract(pdf_path, page_range)


# CLI interface
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="L1 PDF Extraction with Marker v1.10"
    )
    parser.add_argument("pdf", type=Path, help="Path to PDF file")
    parser.add_argument(
        "--output", "-o",
        type=Path,
        help="Output JSON file (default: stdout)"
    )
    parser.add_argument(
        "--markdown", "-m",
        type=Path,
        help="Output markdown file"
    )
    parser.add_argument(
        "--seed",
        type=int,
        default=42,
        help="Random seed for reproducibility"
    )
    parser.add_argument(
        "--pages",
        type=str,
        help="Page range (e.g., '1-10')"
    )
    parser.add_argument(
        "--no-ocr-fix",
        action="store_true",
        help="Disable OCR post-processing for clinical terms"
    )
    parser.add_argument(
        "--show-ocr-corrections",
        action="store_true",
        help="Print OCR corrections summary"
    )

    args = parser.parse_args()

    # Parse page range
    page_range = None
    if args.pages:
        start, end = args.pages.split("-")
        page_range = (int(start), int(end))

    # Create extractor with OCR post-processing option
    extractor = MarkerExtractor(
        seed=args.seed,
        enable_ocr_postprocessing=not args.no_ocr_fix,
    )

    # Extract
    result = extractor.extract(args.pdf, page_range=page_range)

    # Show OCR corrections if requested
    if args.show_ocr_corrections:
        ocr_corrections = result.provenance.extraction_params.get("ocr_corrections", {})
        if ocr_corrections:
            print(f"\n🔧 OCR Corrections Applied: {ocr_corrections.get('total', 0)}")
            by_type = ocr_corrections.get("by_type", {})
            for correction_type, count in sorted(by_type.items(), key=lambda x: -x[1]):
                print(f"   - {correction_type}: {count}")
        else:
            print("\n✓ No OCR corrections needed")

    # Output
    if args.output:
        args.output.write_text(result.to_json())
        print(f"Extraction saved to {args.output}")
    else:
        print(result.to_json())

    if args.markdown:
        args.markdown.write_text(result.markdown)
        print(f"Markdown saved to {args.markdown}")
