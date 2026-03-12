#!/usr/bin/env python3
"""
Vaidshala Phase 4: Deterministic Table Extraction Pipeline

Extracts COR/LOE recommendation tables from PDF clinical practice guidelines.
LLM used ONLY for genuinely ambiguous cells (<5% of content).

Philosophy: Deterministic First, LLM as Last Resort

Usage:
    python table_extractor.py <pdf_path> <guideline_name>
    python table_extractor.py --test  # Run with sample data

Requirements:
    pip install pdfplumber
"""

import re
import json
import sys
from pathlib import Path
from typing import List, Dict, Optional, Tuple, Any
from dataclasses import dataclass, asdict, field
from datetime import datetime
from enum import Enum


class ClassOfRecommendation(Enum):
    """ACC/AHA Class of Recommendation taxonomy."""
    CLASS_I = "I"           # Strong recommendation, benefit >>> risk
    CLASS_IIA = "IIa"       # Moderate recommendation, benefit >> risk
    CLASS_IIB = "IIb"       # Weak recommendation, benefit >= risk
    CLASS_III_HARM = "III-Harm"      # Strong against, harm > benefit
    CLASS_III_NO_BENEFIT = "III-NoBenefit"  # Strong against, no benefit
    CLASS_III = "III"       # Generic Class III (needs context)
    UNKNOWN = "UNKNOWN"


class LevelOfEvidence(Enum):
    """ACC/AHA Level of Evidence taxonomy."""
    LOE_A = "A"             # High quality: Multiple RCTs or meta-analyses
    LOE_B_R = "B-R"         # Moderate: Single RCT or non-randomized
    LOE_B_NR = "B-NR"       # Moderate: Non-randomized studies
    LOE_B = "B"             # Generic B (needs context)
    LOE_C_LD = "C-LD"       # Limited data
    LOE_C_EO = "C-EO"       # Expert opinion
    LOE_C = "C"             # Generic C (needs context)
    UNKNOWN = "UNKNOWN"


class TemporalConstraintType(Enum):
    """Types of temporal constraints in clinical guidelines."""
    DEADLINE = "DEADLINE"           # "within 1 hour"
    RECURRING = "RECURRING"         # "every 4 hours"
    SEQUENCE_BEFORE = "BEFORE"      # "before antibiotics"
    SEQUENCE_AFTER = "AFTER"        # "after blood cultures"
    URGENT = "URGENT"               # "immediately", "STAT"
    RANGE = "RANGE"                 # "2-4 hours"
    DURATION = "DURATION"           # "for 7 days"


@dataclass
class TemporalConstraint:
    """Extracted temporal constraint."""
    constraint_type: str
    value: str
    unit: Optional[str] = None
    reference_event: Optional[str] = None
    raw_text: str = ""


@dataclass
class RecommendationRow:
    """Single extracted recommendation from guideline table."""
    cor: Optional[str]
    loe: Optional[str]
    recommendation_text: str
    temporal_constraints: List[Dict] = field(default_factory=list)
    needs_llm_review: bool = False
    confidence: float = 0.0
    source_page: int = 0
    source_guideline: str = ""
    extraction_method: str = "DETERMINISTIC"
    raw_row_data: List[str] = field(default_factory=list)


class GuidelineTableExtractor:
    """
    Extract COR/LOE tables from PDF guidelines.
    Deterministic regex patterns - LLM only for ambiguous cases.

    Supports:
    - ACC/AHA format (Class I, IIa, IIb, III)
    - GRADE format (Strong, Weak, Conditional)
    - ESC format (similar to ACC/AHA)
    - WHO format (Strong, Conditional)
    """

    # ACC/AHA Class of Recommendation patterns (order matters - specific first)
    COR_PATTERNS = [
        # GRADE/WHO format (MUST be before ACC/AHA to avoid "Strong" matching "I")
        (r'\bConditional\s+(?:for\b|recommendation)', ClassOfRecommendation.CLASS_IIA),
        (r'\bConditional\s+against\b', ClassOfRecommendation.CLASS_III_NO_BENEFIT),
        (r'\bStrong\s+(?:for\b|recommendation)', ClassOfRecommendation.CLASS_I),
        (r'\bStrong\s+against\b', ClassOfRecommendation.CLASS_III_HARM),
        (r'\bWeak\s+(?:for\b|recommendation)', ClassOfRecommendation.CLASS_IIB),

        # Class III with qualifiers (most specific)
        (r'(?:Class\s*)?III\s*[-:]\s*(?:Harm|harm|HARM)', ClassOfRecommendation.CLASS_III_HARM),
        (r'(?:Class\s*)?III\s*[-:]\s*(?:No\s*Benefit|no\s*benefit|NB)', ClassOfRecommendation.CLASS_III_NO_BENEFIT),
        (r'(?:Class\s*)?III\s*\(\s*(?:Harm|harm)\s*\)', ClassOfRecommendation.CLASS_III_HARM),
        (r'(?:Class\s*)?III\s*\(\s*(?:No\s*Benefit|NB)\s*\)', ClassOfRecommendation.CLASS_III_NO_BENEFIT),

        # Class IIa and IIb (before generic II)
        (r'(?:Class\s*)?IIa(?![b-z])', ClassOfRecommendation.CLASS_IIA),
        (r'(?:Class\s*)?IIb(?![a-z])', ClassOfRecommendation.CLASS_IIB),
        (r'(?:Class\s*)?2a(?![b-z])', ClassOfRecommendation.CLASS_IIA),
        (r'(?:Class\s*)?2b(?![a-z])', ClassOfRecommendation.CLASS_IIB),

        # Class III generic
        (r'(?:Class\s*)?III(?![IVab\d])', ClassOfRecommendation.CLASS_III),
        (r'(?:Class\s*)?3(?![ab\d])', ClassOfRecommendation.CLASS_III),

        # Class I (most permissive - last)
        (r'(?:Class\s*)?I(?![IVab\d])', ClassOfRecommendation.CLASS_I),
        (r'(?:Class\s*)?1(?![ab\d])', ClassOfRecommendation.CLASS_I),
    ]

    # Level of Evidence patterns (order matters - specific first)
    LOE_PATTERNS = [
        # Specific B and C subtypes first
        (r'(?:LOE\s*|Level\s*(?:of\s*Evidence\s*)?)?B\s*[-:]?\s*R(?:andomized)?', LevelOfEvidence.LOE_B_R),
        (r'(?:LOE\s*|Level\s*(?:of\s*Evidence\s*)?)?B\s*[-:]?\s*NR', LevelOfEvidence.LOE_B_NR),
        (r'(?:LOE\s*|Level\s*(?:of\s*Evidence\s*)?)?C\s*[-:]?\s*LD', LevelOfEvidence.LOE_C_LD),
        (r'(?:LOE\s*|Level\s*(?:of\s*Evidence\s*)?)?C\s*[-:]?\s*EO', LevelOfEvidence.LOE_C_EO),

        # Generic levels
        (r'(?:LOE\s*|Level\s*(?:of\s*Evidence\s*)?)?A(?![a-zA-Z-])', LevelOfEvidence.LOE_A),
        (r'(?:LOE\s*|Level\s*(?:of\s*Evidence\s*)?)?B(?![a-zA-Z-])', LevelOfEvidence.LOE_B),
        (r'(?:LOE\s*|Level\s*(?:of\s*Evidence\s*)?)?C(?![a-zA-Z-])', LevelOfEvidence.LOE_C),

        # GRADE certainty (Very Low MUST come before Low)
        (r'\bVery\s+[Ll]ow\s*(?:quality|certainty)\b', LevelOfEvidence.LOE_C_EO),
        (r'\bHigh\s*(?:quality|certainty)\b', LevelOfEvidence.LOE_A),
        (r'\bModerate\s*(?:quality|certainty)\b', LevelOfEvidence.LOE_B),
        (r'\bLow\s*(?:quality|certainty)\b', LevelOfEvidence.LOE_C_LD),
    ]

    # Temporal constraint patterns
    TEMPORAL_PATTERNS = [
        # Deadlines: "within X hours/minutes/days"
        (r'within\s+(\d+(?:\.\d+)?)\s*(hours?|hrs?|minutes?|mins?|days?|d)\b',
         TemporalConstraintType.DEADLINE),

        # Recurring: "every X hours/days"
        (r'(?:every|q)\s*(\d+(?:\.\d+)?)\s*(hours?|hrs?|h|days?|d|weeks?|wks?|months?)\b',
         TemporalConstraintType.RECURRING),

        # Sequence before: "before X", "prior to X"
        (r'(?:before|prior\s+to)\s+([a-zA-Z][a-zA-Z\s]{2,30}?)(?:\.|,|;|$)',
         TemporalConstraintType.SEQUENCE_BEFORE),

        # Sequence after: "after X", "following X"
        (r'(?:after|following)\s+([a-zA-Z][a-zA-Z\s]{2,30}?)(?:\.|,|;|$)',
         TemporalConstraintType.SEQUENCE_AFTER),

        # Urgent: "immediately", "STAT", "as soon as possible"
        (r'\b(immediately|STAT|stat|as\s+soon\s+as\s+possible|ASAP|asap|emergent(?:ly)?)\b',
         TemporalConstraintType.URGENT),

        # Range: "2-4 hours", "1 to 3 days"
        (r'(\d+(?:\.\d+)?)\s*(?:to|-)\s*(\d+(?:\.\d+)?)\s*(hours?|hrs?|days?|d|weeks?)\b',
         TemporalConstraintType.RANGE),

        # Duration: "for X days/weeks"
        (r'(?:for|x)\s+(\d+(?:\.\d+)?)\s*(days?|d|weeks?|wks?|months?)\b',
         TemporalConstraintType.DURATION),
    ]

    # Table header indicators - for multi-column recommendation tables
    TABLE_HEADER_KEYWORDS = [
        'class', 'cor', 'recommendation', 'loe', 'level', 'evidence',
        'grade', 'strength', 'quality', 'certainty'
    ]

    # GRADE-style single-column table header (like SSC)
    GRADE_TABLE_KEYWORDS = ['recommendation', 'recommendations']

    def __init__(self, guideline_source: str):
        """
        Initialize extractor for a specific guideline.

        Args:
            guideline_source: Identifier for the guideline (e.g., "ACC-AHA-HF-2022")
        """
        self.guideline_source = guideline_source
        self.extraction_stats = {
            "total_tables": 0,
            "recommendation_tables": 0,
            "total_rows": 0,
            "successful_extractions": 0,
            "needs_llm_review": 0,
            "temporal_constraints_found": 0
        }

    def extract_from_pdf(self, pdf_path: str) -> List[RecommendationRow]:
        """
        Extract recommendation tables from PDF.

        Args:
            pdf_path: Path to PDF file

        Returns:
            List of extracted RecommendationRow objects
        """
        try:
            import pdfplumber
        except ImportError:
            raise ImportError(
                "pdfplumber is required for PDF extraction. "
                "Install with: pip install pdfplumber"
            )

        recommendations = []
        pdf_path = Path(pdf_path)

        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF not found: {pdf_path}")

        with pdfplumber.open(pdf_path) as pdf:
            for page_num, page in enumerate(pdf.pages, 1):
                tables = page.extract_tables()
                self.extraction_stats["total_tables"] += len(tables)

                table_found = False
                for table in tables:
                    if self._is_recommendation_table(table):
                        self.extraction_stats["recommendation_tables"] += 1
                        rows = self._parse_table(table, page_num)
                        recommendations.extend(rows)
                        table_found = True

                # If no structured tables found, try page text extraction for GRADE
                if not table_found:
                    page_text = page.extract_text()
                    if page_text:
                        page_recs = self._extract_grade_from_text(page_text, page_num)
                        recommendations.extend(page_recs)

        return recommendations

    def _extract_grade_from_text(self, text: str, page_num: int) -> List[RecommendationRow]:
        """
        Extract GRADE-format recommendations from page text.

        Handles patterns like:
        - "5. For patients with sepsis... Weak recommendation, low-quality evidence"
        - "Strong recommendation, high certainty: We recommend..."
        """
        recommendations = []

        # Split into potential recommendation blocks
        # Look for numbered recommendations or GRADE indicators
        grade_pattern = r'((?:Strong|Weak|Conditional)\s+recommendation[,\s]+(?:high|moderate|low|very\s+low)[\-\s]*(?:quality|certainty)\s+evidence)'
        bps_pattern = r'(Best\s+Practice\s+Statement)'

        # Find all GRADE indicators
        grade_matches = list(re.finditer(grade_pattern, text, re.IGNORECASE))
        bps_matches = list(re.finditer(bps_pattern, text, re.IGNORECASE))

        all_matches = [(m.start(), m.group(), 'GRADE') for m in grade_matches]
        all_matches.extend([(m.start(), m.group(), 'BPS') for m in bps_matches])
        all_matches.sort(key=lambda x: x[0])

        for i, (pos, indicator, ind_type) in enumerate(all_matches):
            # Find surrounding context (recommendation text)
            # Look backwards for the recommendation
            start = max(0, pos - 500)
            end = min(len(text), pos + len(indicator) + 100)
            context = text[start:end]

            # Extract COR/LOE from the indicator
            cor, cor_conf = self._match_cor(indicator)
            loe, loe_conf = self._match_loe(indicator)

            # Handle Best Practice Statement
            if ind_type == 'BPS':
                cor = ClassOfRecommendation.CLASS_I
                cor_conf = 0.90

            # Extract temporal constraints from context
            temporal = self._extract_temporal_constraints(context)

            # Clean the recommendation text
            rec_text = self._clean_recommendation_text(context)

            if len(rec_text) < 30:
                continue

            # Truncate if too long
            if len(rec_text) > 500:
                rec_text = rec_text[:500] + "..."

            recommendations.append(RecommendationRow(
                cor=cor.value if cor else None,
                loe=loe.value if loe else None,
                recommendation_text=rec_text,
                temporal_constraints=[asdict(t) for t in temporal],
                needs_llm_review=cor is None,
                confidence=min(cor_conf, loe_conf) if cor and loe else 0.6,
                source_page=page_num,
                source_guideline=self.guideline_source,
                extraction_method="GRADE_TEXT",
                raw_row_data=[indicator]
            ))

        return recommendations

    def extract_from_text(self, text_content: str, page_num: int = 1) -> List[RecommendationRow]:
        """
        Extract recommendations from plain text (for testing without PDF).

        Args:
            text_content: Raw text containing recommendation tables
            page_num: Page number for reference

        Returns:
            List of extracted RecommendationRow objects
        """
        recommendations = []

        # Split into lines and look for recommendation patterns
        lines = text_content.strip().split('\n')

        for line in lines:
            line = line.strip()
            if not line:
                continue

            # Try to parse as a recommendation line
            parsed = self._parse_text_line(line, page_num)
            if parsed:
                recommendations.append(parsed)

        return recommendations

    def _is_recommendation_table(self, table: List[List[str]]) -> bool:
        """
        Detect if table is a recommendation table by headers.

        Args:
            table: 2D list of table cells

        Returns:
            True if this appears to be a recommendation table
        """
        if not table or not table[0]:
            return False

        # Combine header row text
        header_text = ' '.join(
            str(cell).lower() for cell in table[0] if cell
        )

        # Check for GRADE-style single column tables (like SSC)
        for keyword in self.GRADE_TABLE_KEYWORDS:
            if keyword in header_text:
                # Check if any row contains GRADE strength indicators
                for row in table[1:3]:  # Check first few rows
                    row_text = ' '.join(str(c).lower() for c in row if c)
                    if any(term in row_text for term in [
                        'strong', 'weak', 'conditional', 'recommend',
                        'suggest', 'best practice'
                    ]):
                        return True

        # Count matching keywords for multi-column tables
        matches = sum(
            1 for keyword in self.TABLE_HEADER_KEYWORDS
            if keyword in header_text
        )

        # Need at least 2 matching keywords
        return matches >= 2

    def _is_grade_table(self, table: List[List[str]]) -> bool:
        """Check if table uses GRADE format (single column with embedded evidence)."""
        if not table or not table[0]:
            return False
        header_text = ' '.join(str(c).lower() for c in table[0] if c)
        return any(kw in header_text for kw in self.GRADE_TABLE_KEYWORDS)

    def _parse_table(self, table: List[List[str]], page_num: int) -> List[RecommendationRow]:
        """
        Parse recommendation table rows.

        Args:
            table: 2D list of table cells
            page_num: Source page number

        Returns:
            List of parsed RecommendationRow objects
        """
        rows = []
        is_grade = self._is_grade_table(table)

        # Skip header row
        for row in table[1:]:
            if not row:
                continue

            # Skip empty rows
            if all(cell is None or str(cell).strip() == '' for cell in row):
                continue

            self.extraction_stats["total_rows"] += 1

            # Use GRADE parsing for single-column GRADE tables
            if is_grade:
                parsed = self._parse_grade_row(row, page_num)
            else:
                parsed = self._parse_row(row, page_num)

            if parsed:
                rows.append(parsed)

                if parsed.needs_llm_review:
                    self.extraction_stats["needs_llm_review"] += 1
                else:
                    self.extraction_stats["successful_extractions"] += 1

                self.extraction_stats["temporal_constraints_found"] += len(
                    parsed.temporal_constraints
                )

        return rows

    def _parse_grade_row(self, row: List[str], page_num: int) -> Optional[RecommendationRow]:
        """
        Parse a GRADE-format recommendation row (SSC, WHO style).

        GRADE format has recommendation text with embedded strength/quality:
        "We recommend X... Strong recommendation, high-quality evidence"
        """
        # Combine all cells into single text
        text = ' '.join(str(c).strip() for c in row if c)

        if not text or len(text) < 20:
            return None

        # Extract GRADE strength and quality from text
        cor, cor_confidence = self._match_cor(text)
        loe, loe_confidence = self._match_loe(text)

        # Extract temporal constraints
        temporal = self._extract_temporal_constraints(text)

        # Clean the recommendation text
        rec_text = self._clean_recommendation_text(text)

        # Determine if LLM review needed
        needs_llm = cor is None or loe is None

        if cor and loe:
            confidence = min(cor_confidence, loe_confidence)
        elif cor or loe:
            confidence = max(cor_confidence, loe_confidence) * 0.7
        else:
            confidence = 0.3

        return RecommendationRow(
            cor=cor.value if cor else None,
            loe=loe.value if loe else None,
            recommendation_text=rec_text,
            temporal_constraints=[asdict(t) for t in temporal],
            needs_llm_review=needs_llm,
            confidence=confidence,
            source_page=page_num,
            source_guideline=self.guideline_source,
            extraction_method="GRADE_TABLE",
            raw_row_data=[text]
        )

    def _parse_row(self, row: List[str], page_num: int) -> Optional[RecommendationRow]:
        """
        Parse a single recommendation row.

        Args:
            row: List of cell values
            page_num: Source page number

        Returns:
            Parsed RecommendationRow or None if unparseable
        """
        if len(row) < 2:
            return None

        # Clean row data
        clean_row = [str(cell).strip() if cell else '' for cell in row]

        # Try to extract COR from first column
        cor_text = clean_row[0]
        cor, cor_confidence = self._match_cor(cor_text)

        # Try to extract LOE from second column (or first if COR not found)
        loe_col_idx = 1 if len(clean_row) > 1 else 0
        loe_text = clean_row[loe_col_idx]
        loe, loe_confidence = self._match_loe(loe_text)

        # If COR not in first column, check if first column is LOE
        if cor is None and loe is None:
            # Maybe COR and LOE are in same cell
            combined = f"{cor_text} {loe_text}"
            cor, cor_confidence = self._match_cor(combined)
            loe, loe_confidence = self._match_loe(combined)

        # Recommendation text from remaining columns
        rec_start_idx = 2 if len(clean_row) > 2 else 1
        rec_text = ' '.join(clean_row[rec_start_idx:]).strip()

        # If no rec text, use last column
        if not rec_text and clean_row:
            rec_text = clean_row[-1]

        # Skip if no meaningful recommendation text
        if not rec_text or len(rec_text) < 10:
            return None

        # Extract temporal constraints from recommendation text
        temporal = self._extract_temporal_constraints(rec_text)

        # Determine if LLM review needed
        needs_llm = (cor is None or loe is None) and bool(rec_text)

        # Calculate confidence
        if cor and loe:
            confidence = min(cor_confidence, loe_confidence)
        elif cor or loe:
            confidence = max(cor_confidence, loe_confidence) * 0.7
        else:
            confidence = 0.3

        return RecommendationRow(
            cor=cor.value if cor else None,
            loe=loe.value if loe else None,
            recommendation_text=rec_text,
            temporal_constraints=[asdict(t) for t in temporal],
            needs_llm_review=needs_llm,
            confidence=confidence,
            source_page=page_num,
            source_guideline=self.guideline_source,
            extraction_method="DETERMINISTIC" if not needs_llm else "NEEDS_LLM",
            raw_row_data=clean_row
        )

    def _parse_text_line(self, line: str, page_num: int) -> Optional[RecommendationRow]:
        """
        Parse a single text line for recommendation content.

        Args:
            line: Text line to parse
            page_num: Source page number

        Returns:
            Parsed RecommendationRow or None
        """
        # Try to match COR and LOE in the line
        cor, cor_confidence = self._match_cor(line)
        loe, loe_confidence = self._match_loe(line)

        # Extract temporal constraints
        temporal = self._extract_temporal_constraints(line)

        # Clean recommendation text (remove COR/LOE indicators)
        rec_text = self._clean_recommendation_text(line)

        if not rec_text or len(rec_text) < 10:
            return None

        needs_llm = cor is None or loe is None
        confidence = min(cor_confidence, loe_confidence) if cor and loe else 0.5

        return RecommendationRow(
            cor=cor.value if cor else None,
            loe=loe.value if loe else None,
            recommendation_text=rec_text,
            temporal_constraints=[asdict(t) for t in temporal],
            needs_llm_review=needs_llm,
            confidence=confidence,
            source_page=page_num,
            source_guideline=self.guideline_source,
            extraction_method="DETERMINISTIC_TEXT",
            raw_row_data=[line]
        )

    def _match_cor(self, text: str) -> Tuple[Optional[ClassOfRecommendation], float]:
        """
        Match Class of Recommendation pattern.

        Args:
            text: Text to match against

        Returns:
            Tuple of (matched COR enum, confidence score)
        """
        text = text.strip()

        for pattern, cor_value in self.COR_PATTERNS:
            if re.search(pattern, text, re.IGNORECASE):
                return cor_value, 0.95

        return None, 0.0

    def _match_loe(self, text: str) -> Tuple[Optional[LevelOfEvidence], float]:
        """
        Match Level of Evidence pattern.

        Args:
            text: Text to match against

        Returns:
            Tuple of (matched LOE enum, confidence score)
        """
        text = text.strip()

        for pattern, loe_value in self.LOE_PATTERNS:
            if re.search(pattern, text, re.IGNORECASE):
                return loe_value, 0.95

        return None, 0.0

    def _extract_temporal_constraints(self, text: str) -> List[TemporalConstraint]:
        """
        Extract temporal constraints from recommendation text.

        Args:
            text: Recommendation text

        Returns:
            List of TemporalConstraint objects
        """
        constraints = []

        for pattern, constraint_type in self.TEMPORAL_PATTERNS:
            matches = re.finditer(pattern, text, re.IGNORECASE)

            for match in matches:
                groups = match.groups()

                if constraint_type == TemporalConstraintType.DEADLINE:
                    constraints.append(TemporalConstraint(
                        constraint_type=constraint_type.value,
                        value=groups[0],
                        unit=groups[1],
                        raw_text=match.group(0)
                    ))

                elif constraint_type == TemporalConstraintType.RECURRING:
                    constraints.append(TemporalConstraint(
                        constraint_type=constraint_type.value,
                        value=groups[0],
                        unit=groups[1],
                        raw_text=match.group(0)
                    ))

                elif constraint_type in (TemporalConstraintType.SEQUENCE_BEFORE,
                                         TemporalConstraintType.SEQUENCE_AFTER):
                    constraints.append(TemporalConstraint(
                        constraint_type=constraint_type.value,
                        value=groups[0].strip(),
                        reference_event=groups[0].strip(),
                        raw_text=match.group(0)
                    ))

                elif constraint_type == TemporalConstraintType.URGENT:
                    constraints.append(TemporalConstraint(
                        constraint_type=constraint_type.value,
                        value=groups[0],
                        raw_text=match.group(0)
                    ))

                elif constraint_type == TemporalConstraintType.RANGE:
                    constraints.append(TemporalConstraint(
                        constraint_type=constraint_type.value,
                        value=f"{groups[0]}-{groups[1]}",
                        unit=groups[2],
                        raw_text=match.group(0)
                    ))

                elif constraint_type == TemporalConstraintType.DURATION:
                    constraints.append(TemporalConstraint(
                        constraint_type=constraint_type.value,
                        value=groups[0],
                        unit=groups[1],
                        raw_text=match.group(0)
                    ))

        return constraints

    def _clean_recommendation_text(self, text: str) -> str:
        """
        Clean recommendation text by removing COR/LOE indicators.

        Args:
            text: Raw text

        Returns:
            Cleaned text
        """
        # Remove common COR/LOE patterns
        patterns_to_remove = [
            r'Class\s*(?:I+[ab]?|III)\s*[-:,]?\s*',
            r'LOE\s*[-:]?\s*[ABC](?:-[A-Z]{1,2})?\s*[-:,]?\s*',
            r'Level\s*(?:of\s*Evidence)?\s*[-:]?\s*[ABC](?:-[A-Z]{1,2})?\s*[-:,]?\s*',
            r'COR\s*[-:]?\s*(?:I+[ab]?|III)\s*[-:,]?\s*',
        ]

        cleaned = text
        for pattern in patterns_to_remove:
            cleaned = re.sub(pattern, '', cleaned, flags=re.IGNORECASE)

        return cleaned.strip()

    def get_stats(self) -> Dict[str, Any]:
        """Get extraction statistics."""
        stats = self.extraction_stats.copy()

        if stats["total_rows"] > 0:
            stats["success_rate"] = (
                stats["successful_extractions"] / stats["total_rows"] * 100
            )
            stats["llm_review_rate"] = (
                stats["needs_llm_review"] / stats["total_rows"] * 100
            )
        else:
            stats["success_rate"] = 0
            stats["llm_review_rate"] = 0

        return stats


class KB15Formatter:
    """
    Format extracted recommendations for KB-15 Evidence Engine.
    """

    def __init__(self, guideline_metadata: Dict[str, str]):
        """
        Initialize formatter with guideline metadata.

        Args:
            guideline_metadata: Dict with keys like 'title', 'organization', 'year', 'doi'
        """
        self.metadata = guideline_metadata

    def format(self, recommendations: List[RecommendationRow]) -> List[Dict]:
        """
        Convert recommendations to KB-15 format.

        Args:
            recommendations: List of RecommendationRow objects

        Returns:
            List of KB-15 formatted dictionaries
        """
        kb15_entries = []

        for i, rec in enumerate(recommendations, 1):
            entry = {
                "recommendation_id": f"{rec.source_guideline}-{i:04d}",
                "evidence_envelope": {
                    "class_of_recommendation": rec.cor,
                    "cor_display": self._get_cor_display(rec.cor),
                    "level_of_evidence": rec.loe,
                    "loe_display": self._get_loe_display(rec.loe),
                    "extraction_confidence": rec.confidence,
                    "extraction_method": rec.extraction_method,
                    "requires_sme_review": rec.needs_llm_review
                },
                "recommendation": {
                    "text": rec.recommendation_text,
                    "temporal_constraints": rec.temporal_constraints
                },
                "provenance": {
                    "guideline_source": self.metadata,
                    "source_page": rec.source_page,
                    "extraction_timestamp": datetime.utcnow().isoformat() + "Z",
                    "raw_data": rec.raw_row_data
                },
                "governance": {
                    "status": "DRAFT" if rec.needs_llm_review else "PENDING_REVIEW",
                    "sme_approved": False,
                    "activation_ready": False
                }
            }
            kb15_entries.append(entry)

        return kb15_entries

    def _get_cor_display(self, cor: Optional[str]) -> str:
        """Get human-readable COR display text."""
        displays = {
            "I": "Class I (Strong): Benefit >>> Risk",
            "IIa": "Class IIa (Moderate): Benefit >> Risk",
            "IIb": "Class IIb (Weak): Benefit >= Risk",
            "III": "Class III: Risk >= Benefit",
            "III-Harm": "Class III (Harm): Risk > Benefit",
            "III-NoBenefit": "Class III (No Benefit): No proven benefit"
        }
        return displays.get(cor, "Unknown")

    def _get_loe_display(self, loe: Optional[str]) -> str:
        """Get human-readable LOE display text."""
        displays = {
            "A": "Level A: Multiple RCTs or meta-analyses",
            "B-R": "Level B-R: Single RCT or randomized studies",
            "B-NR": "Level B-NR: Non-randomized studies",
            "B": "Level B: Moderate quality evidence",
            "C-LD": "Level C-LD: Limited data",
            "C-EO": "Level C-EO: Expert opinion",
            "C": "Level C: Lower quality evidence"
        }
        return displays.get(loe, "Unknown")


class KB3TemporalFormatter:
    """
    Format temporal constraints for KB-3 Temporal Brain.
    """

    def format(self, recommendations: List[RecommendationRow]) -> List[Dict]:
        """
        Extract and format temporal constraints for KB-3.

        Args:
            recommendations: List of RecommendationRow objects

        Returns:
            List of KB-3 temporal constraint records
        """
        kb3_entries = []

        for rec in recommendations:
            for constraint in rec.temporal_constraints:
                entry = {
                    "constraint_id": f"{rec.source_guideline}-TC-{len(kb3_entries)+1:04d}",
                    "recommendation_id": f"{rec.source_guideline}",
                    "constraint_type": constraint.get("constraint_type"),
                    "value": constraint.get("value"),
                    "unit": constraint.get("unit"),
                    "reference_event": constraint.get("reference_event"),
                    "iso8601_duration": self._to_iso8601(
                        constraint.get("value"),
                        constraint.get("unit")
                    ),
                    "source_text": constraint.get("raw_text"),
                    "guideline_source": rec.source_guideline
                }
                kb3_entries.append(entry)

        return kb3_entries

    def _to_iso8601(self, value: Optional[str], unit: Optional[str]) -> Optional[str]:
        """Convert value/unit to ISO 8601 duration."""
        if not value or not unit:
            return None

        try:
            num = float(value)
        except (ValueError, TypeError):
            return None

        unit_map = {
            'minute': 'M', 'minutes': 'M', 'min': 'M', 'mins': 'M',
            'hour': 'H', 'hours': 'H', 'hr': 'H', 'hrs': 'H', 'h': 'H',
            'day': 'D', 'days': 'D', 'd': 'D',
            'week': 'W', 'weeks': 'W', 'wk': 'W', 'wks': 'W',
            'month': 'M', 'months': 'M'
        }

        iso_unit = unit_map.get(unit.lower())
        if not iso_unit:
            return None

        if iso_unit in ('M', 'H'):
            return f"PT{int(num)}{iso_unit}"
        else:
            return f"P{int(num)}{iso_unit}"


def run_test():
    """Run extraction test with sample data."""

    print("=" * 60)
    print("PHASE 4: Table Extraction Pipeline - Test Run")
    print("=" * 60)

    # Sample recommendation data (simulating extracted table content)
    sample_recommendations = """
Class I, LOE A: In patients with HFrEF, ARNi is recommended to reduce morbidity and mortality. Initiate within 24 hours of admission.
Class IIa, LOE B-R: For patients with sepsis, blood cultures should be obtained before antibiotic administration, provided this does not delay therapy beyond 1 hour.
Class I, LOE A: Administer broad-spectrum antibiotics within 1 hour of sepsis recognition. Blood cultures before antibiotics if possible.
Class IIb, LOE C-LD: Consider repeat lactate measurement every 2-4 hours until lactate normalizes.
Class III (Harm), LOE B-NR: Do not use vasopressors if MAP is above 65 mmHg without evidence of tissue hypoperfusion.
Strong recommendation, High certainty: All pregnant women should receive tetanus toxoid vaccination. Administer during weeks 27-36.
Class I, LOE A: Beta-blocker therapy is recommended for all patients with HFrEF. Titrate to target dose over 2-4 weeks.
"""

    # Initialize extractor
    extractor = GuidelineTableExtractor("TEST-GUIDELINE-2026")

    # Extract from sample text
    recommendations = extractor.extract_from_text(sample_recommendations)

    print(f"\nExtracted {len(recommendations)} recommendations\n")

    # Display results
    for i, rec in enumerate(recommendations, 1):
        print(f"--- Recommendation {i} ---")
        print(f"  COR: {rec.cor}")
        print(f"  LOE: {rec.loe}")
        print(f"  Text: {rec.recommendation_text[:80]}...")
        print(f"  Temporal: {len(rec.temporal_constraints)} constraints")
        for tc in rec.temporal_constraints:
            print(f"    - {tc['constraint_type']}: {tc.get('value', '')} {tc.get('unit', '')}")
        print(f"  Needs LLM: {rec.needs_llm_review}")
        print(f"  Confidence: {rec.confidence:.2f}")
        print()

    # Get stats
    stats = extractor.get_stats()
    print("=" * 60)
    print("EXTRACTION STATISTICS")
    print("=" * 60)
    print(f"  Total rows processed: {stats['total_rows']}")
    print(f"  Successful extractions: {stats['successful_extractions']}")
    print(f"  Needs LLM review: {stats['needs_llm_review']}")
    print(f"  Success rate: {stats.get('success_rate', 0):.1f}%")
    print(f"  LLM exposure: {stats.get('llm_review_rate', 0):.1f}%")
    print(f"  Temporal constraints found: {stats['temporal_constraints_found']}")

    # Format for KB-15
    print("\n" + "=" * 60)
    print("KB-15 OUTPUT (first entry)")
    print("=" * 60)

    formatter = KB15Formatter({
        "title": "Test Clinical Practice Guideline",
        "organization": "Vaidshala Test",
        "year": "2026",
        "doi": "10.1234/test.2026"
    })

    kb15_output = formatter.format(recommendations)
    if kb15_output:
        print(json.dumps(kb15_output[0], indent=2))

    # Format for KB-3
    print("\n" + "=" * 60)
    print("KB-3 TEMPORAL CONSTRAINTS")
    print("=" * 60)

    kb3_formatter = KB3TemporalFormatter()
    kb3_output = kb3_formatter.format(recommendations)

    for constraint in kb3_output[:5]:
        print(f"  {constraint['constraint_type']}: {constraint.get('iso8601_duration', 'N/A')} - {constraint['source_text']}")

    print("\n" + "=" * 60)
    print("TEST COMPLETE")
    print("=" * 60)

    return recommendations, kb15_output, kb3_output


def main():
    """Main entry point."""
    if len(sys.argv) < 2:
        print("Usage: python table_extractor.py <pdf_path> <guideline_name>")
        print("       python table_extractor.py --test")
        sys.exit(1)

    if sys.argv[1] == "--test":
        run_test()
        return

    if len(sys.argv) < 3:
        print("Error: Please provide both PDF path and guideline name")
        sys.exit(1)

    pdf_path = sys.argv[1]
    guideline_name = sys.argv[2]

    print(f"Extracting from: {pdf_path}")
    print(f"Guideline: {guideline_name}")

    extractor = GuidelineTableExtractor(guideline_name)

    try:
        recommendations = extractor.extract_from_pdf(pdf_path)
    except ImportError as e:
        print(f"Error: {e}")
        sys.exit(1)
    except FileNotFoundError as e:
        print(f"Error: {e}")
        sys.exit(1)

    print(f"\nExtracted {len(recommendations)} recommendations")

    # Get and display stats
    stats = extractor.get_stats()
    needs_llm = stats['needs_llm_review']
    total = stats['total_rows']

    if total > 0:
        print(f"Needs LLM review: {needs_llm} ({100*needs_llm/total:.1f}%)")

    # Format and save KB-15 output
    formatter = KB15Formatter({
        "title": guideline_name,
        "organization": "Extracted",
        "year": "2026"
    })

    kb15_entries = formatter.format(recommendations)

    output_file = Path(pdf_path).stem + "_kb15.json"
    with open(output_file, 'w') as f:
        json.dump(kb15_entries, f, indent=2)

    print(f"KB-15 output written to: {output_file}")

    # Format and save KB-3 temporal output
    kb3_formatter = KB3TemporalFormatter()
    kb3_entries = kb3_formatter.format(recommendations)

    temporal_file = Path(pdf_path).stem + "_kb3_temporal.json"
    with open(temporal_file, 'w') as f:
        json.dump(kb3_entries, f, indent=2)

    print(f"KB-3 temporal output written to: {temporal_file}")


if __name__ == "__main__":
    main()
