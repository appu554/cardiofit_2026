"""
Channel 0: Text Normalizer for Docling Output.

Runs BEFORE all extraction channels. Fixes Docling output corruption:
1. Unicode ligature corruption (fi -> fi, fl -> fl)
2. Symbol errors (double-dagger -> >=, dagger -> +)
3. OCR letter substitutions (rn -> m, l <-> 1, O <-> 0)
4. Unit normalization (mL/min/1.73m2 variants)
5. Drug name corrections (common OCR misreads)
6. Clinical term normalization (eGFR, HbA1c, CrCl)
7. Whitespace normalization

Sources harvested from:
- marker_extractor.py: ClinicalOCRPostProcessor (100+ regex patterns)
- gliner/extractor.py: OCR_CORRECTIONS dict (simple replacements)
- Real Docling output: KDIGO-2022-Diabetes-CKD-Docling-Output.md

Pipeline Position:
    L1 (Docling) -> Channel 0 (THIS) -> Channel A -> Channels B-F
"""

import re
from typing import Dict, List, Tuple


class Channel0Normalizer:
    """Normalize Docling output text before extraction channels run.

    IMPORTANT: All corrections must be idempotent (safe to run multiple times).
    """

    VERSION = "4.0.0"

    # =========================================================================
    # Unicode ligature map (Docling-specific)
    # =========================================================================

    LIGATURE_MAP: Dict[str, str] = {
        "\ufb01": "fi",         # fi ligature (common in: finerenone, first, define)
        "\ufb02": "fl",         # fl ligature (common in: fluid, flow, influence)
        "\ufb03": "ffi",        # ffi ligature
        "\ufb04": "ffl",        # ffl ligature
        "/uniFB01": "fi",       # Docling raw unicode escape
        "/uniFB02": "fl",       # Docling raw unicode escape
    }

    # =========================================================================
    # Symbol correction map
    # =========================================================================

    SYMBOL_MAP: Dict[str, str] = {
        "\u2021": "\u2265",     # double-dagger -> >= (common Docling artifact)
        "\u2020": "+",          # dagger -> + (footnote artifact)
        "\u00b1": "\u00b1",     # plus-minus stays (identity, for completeness)
        "\u2264": "\u2264",     # <= stays
        "\u2265": "\u2265",     # >= stays
    }

    # =========================================================================
    # OCR simple replacement corrections
    # Harvested from gliner/extractor.py OCR_CORRECTIONS
    # =========================================================================

    OCR_SIMPLE_CORRECTIONS: Dict[str, str] = {
        # Letter substitutions (rn -> m pattern)
        "rnetformin": "metformin",
        "Rnetformin": "Metformin",
        "rnL/min": "mL/min",
        "1.73rn": "1.73m",
        "rng/dL": "mg/dL",
        "rnEq/L": "mEq/L",
        "rnmol/L": "mmol/L",
        # Case fixes for clinical abbreviations
        "CrCI": "CrCl",
        "HbAlc": "HbA1c",
        "HbA 1c": "HbA1c",
        "Hb A1c": "HbA1c",
        # Number-letter confusion in eGFR thresholds
        "eGFR3O": "eGFR 30",
        "eGFR2O": "eGFR 20",
        "eGFR6O": "eGFR 60",
        # Header typos
        "OUICK": "QUICK",
        "Ouick": "Quick",
        "ouick": "quick",
        # Common OCR misreads
        "eGER": "eGFR",
        "EGFR": "eGFR",
    }

    # =========================================================================
    # Regex-based corrections (compiled once)
    # Harvested from marker_extractor.py ClinicalOCRPostProcessor
    # =========================================================================

    UNIT_PATTERNS: List[Tuple[str, str]] = [
        # eGFR unit variants
        (r'mL/min/1\.73\s*m[²2A]', 'mL/min/1.73m\u00b2'),
        (r'mL/min/1\.73\s*m\s*2', 'mL/min/1.73m\u00b2'),
        (r'ml/min/1\.73\s*m[²2]', 'mL/min/1.73m\u00b2'),
        (r'mL/min/1[,.]73m[²2]', 'mL/min/1.73m\u00b2'),
        (r'm[lL]/min/1\.73\s*m\^2', 'mL/min/1.73m\u00b2'),
        (r'mL/min per 1\.73\s*m[²2]', 'mL/min/1.73m\u00b2'),
        # V4.2.2: PDF text layer variants — "per" instead of "/", space before "2"
        (r'm[lL]/min\s+per\s+1\.73\s*m\s*[²2]', 'mL/min/1.73m\u00b2'),
        (r'm[lL]/min\s+per\s+1\.73\s*m\s+2(?!\d)', 'mL/min/1.73m\u00b2'),
        (r'ml/min/1\.73\s*m\s+2(?!\d)', 'mL/min/1.73m\u00b2'),
        # Other units
        (r'mg/d[lL]', 'mg/dL'),
        (r'mEq/[lL]', 'mEq/L'),
        (r'mmol/[lL]', 'mmol/L'),
        (r'\u00b5g/m[lL]', '\u00b5g/mL'),
        (r'mcg/m[lL]', 'mcg/mL'),
        (r'mg/m[²2]', 'mg/m\u00b2'),
    ]

    CLINICAL_TERM_PATTERNS: List[Tuple[str, str]] = [
        # eGFR normalization
        (r'\be[- ]?GFR\b', 'eGFR'),
        (r'\begfr\b', 'eGFR'),
        (r'\beGER\b', 'eGFR'),
        (r'\beGF R\b', 'eGFR'),
        # CrCl
        (r'\bCrCI\b', 'CrCl'),
        (r'\bCRC[lL]\b', 'CrCl'),
        # HbA1c
        (r'\bHbA[1l]c\b', 'HbA1c'),
        (r'\bHBA1C\b', 'HbA1c'),
        (r'\bHgbA1c\b', 'HbA1c'),
        # Kidney disease
        (r'\bESKD\b', 'ESKD'),
        (r'\bESRD\b', 'ESRD'),
        (r'\bAKI\b', 'AKI'),
    ]

    DRUG_NAME_PATTERNS: List[Tuple[str, str]] = [
        # SGLT2 inhibitors (l/1 confusion)
        (r'\b[Dd]apag[1l]if[1l]ozin\b', 'dapagliflozin'),
        (r'\b[Ee]mpag[1l]if[1l]ozin\b', 'empagliflozin'),
        (r'\b[Cc]anag[1l]if[1l]ozin\b', 'canagliflozin'),
        (r'\b[Ee]rtug[1l]if[1l]ozin\b', 'ertugliflozin'),
        (r'\b[Ss]otag[1l]if[1l]ozin\b', 'sotagliflozin'),
        # Metformin
        (r'\b[Mm]etf[o0]rmin\b', 'metformin'),
        (r'\b[Mm]etfonnin\b', 'metformin'),
        # Finerenone
        (r'\b[Ff]ineren[o0]ne\b', 'finerenone'),
        (r'\b[Ff]ineranone\b', 'finerenone'),
        # ACE inhibitors
        (r'\b[Ll]isin[o0]pri[1l]\b', 'lisinopril'),
        (r'\b[Ee]na[1l]apri[1l]\b', 'enalapril'),
        (r'\b[Rr]amipri[1l]\b', 'ramipril'),
        # ARBs
        (r'\b[Ll][o0]sartan\b', 'losartan'),
        (r'\b[Vv]a[1l]sartan\b', 'valsartan'),
        # GLP-1 agonists
        (r'\b[Ss]emag[1l]utide\b', 'semaglutide'),
        (r'\b[Ll]irag[1l]utide\b', 'liraglutide'),
        (r'\b[Dd]u[1l]ag[1l]utide\b', 'dulaglutide'),
        # MRAs
        (r'\b[Ss]pir[o0]n[o0][1l]act[o0]ne\b', 'spironolactone'),
        (r'\b[Ee]p[1l]eren[o0]ne\b', 'eplerenone'),
    ]

    NUMBER_PATTERNS: List[Tuple[str, str]] = [
        # O -> 0 in eGFR thresholds
        (r'\beGFR\s*<\s*3O\b', 'eGFR < 30'),
        (r'\beGFR\s*<\s*2O\b', 'eGFR < 20'),
        (r'\beGFR\s*<\s*6O\b', 'eGFR < 60'),
        (r'\beGFR\s+3O\s*[-\u2013]\s*44\b', 'eGFR 30-44'),
        (r'\beGFR\s+3O\s*[-\u2013]\s*45\b', 'eGFR 30-45'),
        (r'\beGFR\s+45\s*[-\u2013]\s*6O\b', 'eGFR 45-60'),
        # Potassium thresholds
        (r'\bK\+?\s*>\s*5[.,]5\b', 'K+ > 5.5'),
        (r'\bK\+?\s*>\s*6[.,]O\b', 'K+ > 6.0'),
        (r'\bpotassium\s*>\s*5[.,]5\b', 'potassium > 5.5'),
    ]

    SUPERSCRIPT_PATTERNS: List[Tuple[str, str]] = [
        (r'm\^2', 'm\u00b2'),
        (r'm2(?!\d)', 'm\u00b2'),
    ]

    def __init__(self) -> None:
        """Initialize normalizer with compiled regex patterns."""
        self._compiled_patterns: List[Tuple[re.Pattern, str]] = []
        self._compile_all_patterns()

    def _compile_all_patterns(self) -> None:
        """Compile all regex patterns once for reuse."""
        all_patterns = (
            self.UNIT_PATTERNS
            + self.CLINICAL_TERM_PATTERNS
            + self.DRUG_NAME_PATTERNS
            + self.NUMBER_PATTERNS
            + self.SUPERSCRIPT_PATTERNS
        )
        for pattern_str, replacement in all_patterns:
            try:
                compiled = re.compile(pattern_str, re.IGNORECASE)
                self._compiled_patterns.append((compiled, replacement))
            except re.error:
                # Skip invalid patterns silently
                pass

    def normalize(self, text: str) -> Tuple[str, dict]:
        """Normalize text and return (normalized_text, metadata).

        The metadata dict contains:
        - fix_count: Total number of fixes applied
        - ligature_fixes: Count of ligature replacements
        - symbol_fixes: Count of symbol replacements
        - ocr_fixes: Count of simple OCR corrections
        - regex_fixes: Count of regex-based corrections
        - whitespace_fixes: Count of whitespace normalizations

        Args:
            text: Raw Docling output text

        Returns:
            Tuple of (normalized_text, metadata dict with fix counts)
        """
        meta = {
            "fix_count": 0,
            "ligature_fixes": 0,
            "symbol_fixes": 0,
            "ocr_fixes": 0,
            "regex_fixes": 0,
            "whitespace_fixes": 0,
        }

        # Step 1: Ligature replacement
        text, count = self._fix_ligatures(text)
        meta["ligature_fixes"] = count
        meta["fix_count"] += count

        # Step 2: Symbol correction
        text, count = self._fix_symbols(text)
        meta["symbol_fixes"] = count
        meta["fix_count"] += count

        # Step 3: Simple OCR corrections (exact string match)
        text, count = self._fix_ocr_simple(text)
        meta["ocr_fixes"] = count
        meta["fix_count"] += count

        # Step 4: Regex-based corrections (units, terms, drugs, numbers)
        text, count = self._fix_regex_patterns(text)
        meta["regex_fixes"] = count
        meta["fix_count"] += count

        # Step 5: Whitespace normalization
        text, count = self._fix_whitespace(text)
        meta["whitespace_fixes"] = count
        meta["fix_count"] += count

        return text, meta

    def _fix_ligatures(self, text: str) -> Tuple[str, int]:
        """Replace Unicode ligatures with ASCII equivalents."""
        count = 0
        for ligature, replacement in self.LIGATURE_MAP.items():
            occurrences = text.count(ligature)
            if occurrences > 0:
                text = text.replace(ligature, replacement)
                count += occurrences
        return text, count

    def _fix_symbols(self, text: str) -> Tuple[str, int]:
        """Replace misinterpreted symbols."""
        count = 0
        for symbol, replacement in self.SYMBOL_MAP.items():
            if symbol != replacement:  # skip identity mappings
                occurrences = text.count(symbol)
                if occurrences > 0:
                    text = text.replace(symbol, replacement)
                    count += occurrences
        return text, count

    def _fix_ocr_simple(self, text: str) -> Tuple[str, int]:
        """Apply simple string-replacement OCR corrections."""
        count = 0
        for wrong, right in self.OCR_SIMPLE_CORRECTIONS.items():
            occurrences = text.count(wrong)
            if occurrences > 0:
                text = text.replace(wrong, right)
                count += occurrences
        return text, count

    def _fix_regex_patterns(self, text: str) -> Tuple[str, int]:
        """Apply compiled regex patterns for units, terms, drugs, numbers.

        Only counts a substitution as a fix if the text actually changed,
        preventing false positives when a pattern matches already-correct text.
        """
        count = 0
        for pattern, replacement in self._compiled_patterns:
            new_text = pattern.sub(replacement, text)
            if new_text != text:
                # Count actual character differences, not just match count
                count += 1
                text = new_text
        return text, count

    def _fix_whitespace(self, text: str) -> Tuple[str, int]:
        """Normalize whitespace without breaking markdown structure."""
        count = 0

        # Replace multiple spaces (but not at line start for markdown indent)
        lines = text.split('\n')
        fixed_lines = []
        for line in lines:
            # Preserve leading whitespace (markdown indent), fix mid-line multispaces
            stripped = line.lstrip()
            indent = line[:len(line) - len(stripped)]
            mid_fixed = re.sub(r'(?<=\S)  +(?=\S)', ' ', stripped)
            if mid_fixed != stripped:
                count += 1
            fixed_lines.append(indent + mid_fixed)
        text = '\n'.join(fixed_lines)

        # Remove trailing whitespace per line
        lines = text.split('\n')
        cleaned = []
        for line in lines:
            trimmed = line.rstrip()
            if trimmed != line:
                count += 1
            cleaned.append(trimmed)
        text = '\n'.join(cleaned)

        # Normalize multiple blank lines to at most 2 (preserves markdown section breaks)
        new_text = re.sub(r'\n{4,}', '\n\n\n', text)
        if new_text != text:
            count += 1
            text = new_text

        return text, count
