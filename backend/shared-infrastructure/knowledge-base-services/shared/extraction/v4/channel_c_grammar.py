"""
Channel C: Grammar/Regex Pattern Extractor (Drug-Agnostic).

Deterministic pattern matching for clinical signals that are NOT drug names.
All Channel C output is drug-agnostic — a dose "10 mg daily" or threshold
"eGFR < 30" is extracted without knowing which drug it belongs to.
Drug-signal association happens later in Dossier Assembly via section co-location.

Pattern sources harvested from gliner/extractor.py _apply_clinical_rules()
plus additional patterns for recommendation IDs, dose values, and LOINC codes.

Pipeline Position:
    Channel A (GuidelineTree) -> Channel C (THIS, parallel with B/D/E/F)
"""

from __future__ import annotations

import re
import time
from typing import Optional

from .models import ChannelOutput, GuidelineTree, RawSpan
from .postprocessors import extend_parenthetical
from .provenance import (
    ChannelProvenance,
    _normalise_bbox,
    _normalise_confidence,
    _normalise_page_number,
)
from .v5_flags import is_v5_enabled


def _channel_c_model_version() -> str:
    """Channel C model version, pinned to ChannelCGrammar.VERSION if present."""
    try:
        return f"regex@{ChannelCGrammar.VERSION}"
    except Exception:
        return "regex@v1.0"


def _channel_c_provenance(
    bbox,
    page_number,
    confidence,
    profile,
    notes: Optional[str] = None,
) -> Optional[ChannelProvenance]:
    """Build a ChannelProvenance entry for Channel C (grammar/regex patterns).

    Returns None when V5_BBOX_PROVENANCE is off or bbox is missing. Confidence
    typically comes from the per-pattern weight in the profile's extra_patterns.
    """
    if not is_v5_enabled("bbox_provenance", profile):
        return None
    bb = _normalise_bbox(bbox)
    if bb is None:
        return None
    return ChannelProvenance(
        channel_id="C",
        bbox=bb,
        page_number=_normalise_page_number(page_number),
        confidence=_normalise_confidence(confidence),
        model_version=_channel_c_model_version(),
        notes=notes,
    )


class ChannelCGrammar:
    """Regex/grammar pattern extractor for drug-agnostic clinical signals.

    Also exported as ``ChannelC`` for pipeline convenience.

    Extracts:
    - eGFR thresholds (eGFR < 30, eGFR 30-45)
    - Monitoring frequencies (every 3-6 months, Q3-6mo)
    - Lab tests (eGFR, potassium, HbA1c)
    - Contraindication markers (contraindicated, avoid)
    - Recommendation IDs (Recommendation 4.1.1)
    - Dose values (10 mg daily, 500 mg)
    - eGFR ranges (eGFR 30-45 mL/min/1.73m2)
    - Potassium thresholds (K+ > 5.5 mEq/L)
    - LOINC code references
    """

    VERSION = "4.3.0"

    # =========================================================================
    # Pattern definitions with confidence and category
    # (pattern, confidence, category)
    #
    # V4.2.1 global fixes:
    #  1. eGFR operator: handles LaTeX ($\geq$), HTML (&ge;), Unicode (≥),
    #     and ASCII (>=) — prevents dose_value false positives on "30 ml/min".
    #  2. Contraindication "avoid": requires prohibition frame (avoid use/in/
    #     concomitant, should be avoided) — bare "avoid" is too ambiguous in
    #     clinical text where it commonly appears in therapeutic goals.
    #  3. Lab test boundary: rejects matches followed by hyphen — prevents
    #     compound-term false positives (e.g., "sodium-glucose cotransporter").
    # =========================================================================

    # V4.2.1: Universal clinical operator pattern — handles LaTeX, HTML, Unicode, ASCII
    _OP = (
        r'(?:'
        r'\$?\\?(?:leq|geq|le|ge|lt|gt)\$?'  # LaTeX: $\geq$, \leq
        r'|&(?:le|ge|lt|gt);'                  # HTML entities: &ge;
        r'|>=|<='                               # ASCII: >=, <=
        r'|[<>≤≥]'                              # Unicode: ≤ ≥ < >
        r')'
    )

    PATTERNS: list[tuple[str, float, str]] = [
        # eGFR thresholds — V4.2.1: universal operator pattern
        (r'\b(?:e?GFR|CrCl)\s*' + _OP + r'\s*\d+(?:\.\d+)?(?:\s*(?:mL|ml)/min(?:/1\.73\s*m[²2])?)?',
         0.95, "egfr_threshold"),
        (r'\b(?:e?GFR|CrCl)\s+(?:of\s+)?(?:less than|greater than|below|above)\s+\d+(?:\.\d+)?',
         0.95, "egfr_threshold"),
        (r'\b(?:e?GFR|CrCl)\s+\d+\s*[-–]\s*\d+(?:\s*(?:mL|ml)/min(?:/1\.73\s*m[²2])?)?',
         0.95, "egfr_range"),

        # Monitoring frequencies (from extractor.py lines 607-623)
        (r'\b(?:every|each)\s+\d+\s*[-–]\s*\d+\s*(?:months?|weeks?|days?)',
         0.90, "monitoring_frequency"),
        (r'\bQ\d+[-–]?\d*\s*(?:mo|months?|wk|weeks?)',
         0.90, "monitoring_frequency"),
        (r'\bat\s+(?:week|month)\s+\d+',
         0.90, "monitoring_frequency"),
        (r'\bafter\s+\d+\s+(?:weeks?|months?)',
         0.90, "monitoring_frequency"),
        (r'\b(?:annually|quarterly|monthly|weekly|daily)\b',
         0.85, "monitoring_frequency"),
        (r'\bat\s+baseline\b',
         0.90, "monitoring_frequency"),
        (r'\bwithin\s+\d+\s+(?:weeks?|months?)\s+of\s+(?:initiation|starting)',
         0.90, "monitoring_frequency"),

        # Contraindication markers — V4.2.1: require prohibition frame for "avoid"
        # Bare "avoid" is too ambiguous: "to avoid hypoglycemia" = therapeutic goal
        (r'\bcontraindicated\b',
         0.95, "contraindication"),
        (r'\bavoid(?:ed|ing)?\s+(?:use|using|in\s+patients|in\s+those|concomitant|concurrent|coadministration)',
         0.95, "contraindication"),
        (r'\b(?:should|must|is\s+to)\s+be\s+avoided\b',
         0.95, "contraindication"),
        (r'\bdo\s+not\s+(?:use|initiate|start|administer)\b',
         0.95, "contraindication"),
        (r'\b(?:should\s+not\s+be\s+(?:used|done|combined|replaced|prescribed|'
         r'initiated|administered)|not\s+recommended)\b',
         0.95, "contraindication"),

        # Prescriptive counsel/inform directives (V4.3 — Group D upstream)
        (r'\b(?:should|must)\s+(?:be\s+)?(?:counsele?d|informed|advised|educated)'
         r'\s+(?:about|regarding|on|of)\b',
         0.90, "prescriptive_counsel"),
        (r'\b(?:discontinue|stop|hold)\b',
         0.90, "action_marker"),

        # Lab tests — V4.2.1: reject if followed by hyphen (compound term boundary)
        # "sodium-glucose cotransporter" → NOT a lab test; "sodium level" → yes
        (r'\b(?:eGFR|serum\s+creatinine|creatinine|potassium|K\+|sodium|Na\+|'
         r'HbA1c|A1C|fasting\s+glucose|UACR|urine\s+albumin|'
         r'albumin[-\s]to[-\s]creatinine|BUN|blood\s+urea|'
         r'liver\s+function|LFTs?|AST|ALT|hemoglobin|hematocrit)\b(?!-)',
         0.85, "lab_test"),

        # Recommendation IDs — V4.2.2: multi-authority support
        # KDIGO: "Recommendation 3.6.1" / ADA: "Recommendation 9.4"
        (r'\bRecommendation\s+\d+(?:\.\d+)*',
         0.98, "recommendation_id"),

        # Practice Point IDs (KDIGO format — supplementary to section passages)
        (r'\bPractice\s+Point\s+\d+\.\d+(?:\.\d+)*',
         0.98, "practice_point_id"),

        # Dose values — V4.2.1: mL only when NOT followed by /min (flow rate ≠ dose)
        (r'\b\d+(?:\.\d+)?\s*(?:mg|mcg|g|units?)(?:\s*/\s*(?:day|dose|kg))?\b',
         0.85, "dose_value"),
        (r'\b\d+(?:\.\d+)?\s*mL(?!\s*/\s*min)\b',
         0.80, "dose_value"),

        # Potassium thresholds — V4.2.1: universal operator
        (r'\b(?:potassium|K\+)\s*' + _OP + r'\s*\d+(?:\.\d+)?\s*(?:mEq/L|mmol/L)',
         0.95, "potassium_threshold"),

        # LOINC code references
        (r'\b\d{4,5}-\d\b',
         0.80, "loinc_reference"),

        # Evidence/recommendation grades — V4.2.2: multi-authority
        # KDIGO: "Grade 1A", "Level 2C"
        (r'\b(?:Grade|Level)\s+[1-4][A-D]?\b',
         0.90, "evidence_grade"),
        (r'\b(?:strong|weak)\s+(?:recommendation|suggestion)\b',
         0.90, "evidence_grade"),
        # ADA: single-letter grades at end of recommendation text "...A" or "(A)"
        # A = clear RCT evidence, B = supportive, C = poor, E = expert consensus
        (r'\s[ABCE]\s*$',
         0.85, "evidence_grade"),
        (r'\([ABCE]\)\s*$',
         0.90, "evidence_grade"),
    ]

    def __init__(self) -> None:
        """Compile all regex patterns."""
        self._compiled: list[tuple[re.Pattern, float, str]] = []
        for pattern_str, confidence, category in self.PATTERNS:
            try:
                compiled = re.compile(pattern_str, re.IGNORECASE)
                self._compiled.append((compiled, confidence, category))
            except re.error:
                pass  # Skip invalid patterns

    def extract(
        self,
        text: str,
        tree: GuidelineTree,
    ) -> ChannelOutput:
        """Run all regex patterns on normalized text.

        Args:
            text: Normalized text (Channel 0 output)
            tree: GuidelineTree from Channel A

        Returns:
            ChannelOutput with drug-agnostic signal RawSpans
        """
        start_time = time.monotonic()
        spans: list[RawSpan] = []
        occupied: list[tuple[int, int]] = []  # prevent overlapping spans
        filter_counts = {
            "citation_zone": 0,
            "header_zone": 0,
            "table_caption": 0,
        }

        for pattern, confidence, category in self._compiled:
            for match in pattern.finditer(text):
                start = match.start()
                end = match.end()

                # Skip if overlaps with a higher-priority span
                if self._overlaps_existing(start, end, occupied):
                    continue

                # Triple filter: reject non-clinical zones
                if self._is_in_citation_zone(start, end, text, tree):
                    filter_counts["citation_zone"] += 1
                    continue
                if self._is_in_header_zone(start, end, tree):
                    filter_counts["header_zone"] += 1
                    continue
                if self._is_in_table_caption(start, end, text):
                    filter_counts["table_caption"] += 1
                    continue

                # Parenthetical extension
                start, end = extend_parenthetical(text, start, end)

                occupied.append((start, end))

                # Find section and page from tree
                section = tree.find_section_for_offset(start)
                section_id = section.section_id if section else None
                page = tree.get_page_for_offset(start) if tree.page_map else (section.page_number if section else None)

                # Determine source block type
                table = tree.find_table_for_offset(start)
                block_type = "table_cell" if table else "paragraph"

                spans.append(RawSpan(
                    channel="C",
                    text=text[start:end],
                    start=start,
                    end=end,
                    confidence=confidence,
                    page_number=page,
                    section_id=section_id,
                    source_block_type=block_type,
                    channel_metadata={
                        "pattern": category,
                        "regex_match": match.group(),
                    },
                ))

        # V4.3: Extract exception clauses from rec/PP blocks (Group A)
        exception_spans = self._extract_exception_clauses(text, tree)
        spans.extend(exception_spans)

        # V4.3: Bind lab parameters to thresholds in rec/PP blocks (Group B)
        spans = self._bind_thresholds_in_recs(spans, text, tree)

        # V4.3.1: Extract full prescriptive clauses for B1 coverage (Group D2)
        prescriptive_spans = self._extract_prescriptive_clauses(text, tree)
        spans.extend(prescriptive_spans)

        # Sort by offset
        spans.sort(key=lambda s: s.start)

        elapsed_ms = (time.monotonic() - start_time) * 1000

        return ChannelOutput(
            channel="C",
            spans=spans,
            metadata={
                "patterns_compiled": len(self._compiled),
                "matches_found": len(spans),
                "exception_clauses": len(exception_spans),
                "bound_thresholds": sum(
                    1 for s in spans
                    if s.channel_metadata.get("pattern") == "bound_threshold"
                ),
                "prescriptive_clauses": len(prescriptive_spans),
                "filter_rejections": filter_counts,
            },
            elapsed_ms=elapsed_ms,
        )

    def _overlaps_existing(
        self, start: int, end: int, occupied: list[tuple[int, int]]
    ) -> bool:
        """Check if a span overlaps with any already-captured span."""
        for occ_start, occ_end in occupied:
            # Compute overlap
            overlap_start = max(start, occ_start)
            overlap_end = min(end, occ_end)
            if overlap_start < overlap_end:
                overlap_len = overlap_end - overlap_start
                span_len = end - start
                # Reject if >50% overlap
                if overlap_len / span_len > 0.5:
                    return True
        return False

    # =========================================================================
    # V4.1: TRIPLE FILTER — reject matches in non-clinical zones
    #
    # Three lightweight heuristics to reduce noise from citation zones,
    # section headings, and table captions without affecting clinical spans.
    # =========================================================================

    # Inline citation: [1], [1-3], [14,15], [Smith et al., 2020]
    _INLINE_CITE_RE = re.compile(
        r'\[(?:'
        r'[A-Z][a-z]+\s+et\s+al\.'      # [Smith et al.]
        r'|\d{1,3}(?:\s*[-–,]\s*\d{1,3})*'  # [1-3] or [14,15]
        r')\]',
    )

    _REFERENCE_HEADING_KEYWORDS = frozenset({
        "reference", "bibliography", "works cited",
        "supplementary", "disclosure", "acknowledgment",
        "conflicts of interest", "funding",
    })

    def _is_in_citation_zone(
        self, start: int, end: int, text: str, tree: "GuidelineTree",
    ) -> bool:
        """Reject matches inside citation brackets or reference sections.

        Checks two conditions:
        1. Match is enclosed by an inline citation bracket [nn]
        2. Match is in a section whose heading indicates references/bibliography
        """
        # Check inline citation brackets (look for enclosing [])
        search_start = max(0, start - 80)
        search_end = min(len(text), end + 20)
        region = text[search_start:search_end]
        for m in self._INLINE_CITE_RE.finditer(region):
            bracket_start = search_start + m.start()
            bracket_end = search_start + m.end()
            if bracket_start <= start and bracket_end >= end:
                return True

        # Check reference section heading
        section = tree.find_section_for_offset(start)
        if section and section.heading:
            heading_lower = section.heading.lower().strip()
            if any(kw in heading_lower for kw in self._REFERENCE_HEADING_KEYWORDS):
                return True

        return False

    def _is_in_header_zone(
        self, start: int, end: int, tree: "GuidelineTree",
    ) -> bool:
        """Reject matches that fall within a section heading (not body text).

        Section headings like "4.1 SGLT2 inhibitors and renal outcomes" contain
        drug names and clinical terms but are structural, not prescriptive.

        Only applies to sections with block_type="heading" — sections typed as
        "paragraph" or "recommendation" use the heading as metadata, not as
        text occupying those offsets.
        """
        for sec in tree.sections:
            if sec.heading and sec.start_offset is not None and sec.block_type == "heading":
                # Heading spans from section start to start + len(heading) + margin
                heading_end = sec.start_offset + len(sec.heading) + 5
                if sec.start_offset <= start < heading_end and end <= heading_end + 20:
                    return True
        return False

    def _is_in_table_caption(
        self, start: int, end: int, text: str,
    ) -> bool:
        """Reject matches within table caption lines.

        Lines starting with "Table N" are captions, not clinical directives.
        Matches within these lines are structural noise.
        """
        # Find the line containing this match
        line_start = text.rfind("\n", 0, start)
        line_start = line_start + 1 if line_start >= 0 else 0

        line_end = text.find("\n", end)
        if line_end < 0:
            line_end = len(text)

        line = text[line_start:line_end].strip()

        # Table caption pattern: "Table N" at start of line
        if re.match(r'^Table\s+\d+', line, re.IGNORECASE):
            return True

        # Figure caption pattern: "Figure N" at start of line
        if re.match(r'^Figure\s+\d+', line, re.IGNORECASE):
            return True

        return False

    # =========================================================================
    # V4.3: EXCEPTION CLAUSE EXTRACTION (Group A — upstream fix)
    #
    # Rec/PP blocks contain conditional language ("unless", "except",
    # "does not apply") that bounds prescribing rules.  Without these
    # keywords in merged spans, the KB presents rules as unconditional —
    # actively more dangerous than a missing rule.
    #
    # This method scans rec/PP blocks for exception keywords and captures
    # the clause from the keyword through the end of the clause.
    # =========================================================================

    _EXCEPTION_KW_RE = re.compile(
        r'\b(?:unless|except(?:\s+(?:in|for|when|if|that|where|those))?|'
        r'(?:does|do)\s+not\s+apply|excluding|provided\s+that)\b',
        re.IGNORECASE,
    )

    _REC_PP_START_RE = re.compile(
        r'(?:Recommendation|Practice\s+Point)\s+\d+\.\d+(?:\.\d+)*[:\s]',
        re.IGNORECASE,
    )

    _REC_PP_END_RE = re.compile(
        r'\n\s*(?:#{1,4}\s|Recommendation\s+\d|Practice\s+Point\s+\d|'
        r'Rationale|Key\s+information|Evidence|Discussion|Implementation)',
        re.IGNORECASE,
    )

    def _extract_exception_clauses(
        self, text: str, tree: GuidelineTree,
    ) -> list[RawSpan]:
        """Capture exception clauses within recommendation/practice point blocks.

        Scans each rec/PP block for exception keywords (unless, except,
        does not apply, excluding, provided that).  For each keyword found,
        captures from the keyword through the end of the clause (next period,
        semicolon, or block end).

        Only operates within rec/PP blocks — not rationale, discussion, or
        evidence text — to avoid capturing "unless" in non-prescriptive
        contexts.
        """
        spans: list[RawSpan] = []

        for m in self._REC_PP_START_RE.finditer(text):
            remaining = text[m.end():min(m.end() + 3000, len(text))]
            end_match = self._REC_PP_END_RE.search(remaining)
            block_end = (
                m.end() + end_match.start()
                if end_match
                else min(m.end() + 3000, len(text))
            )
            block_text = text[m.start():block_end]

            for kw_match in self._EXCEPTION_KW_RE.finditer(block_text):
                kw_start = kw_match.start()
                # Find clause end: next period, semicolon, or block end
                clause_end = len(block_text)
                for i in range(kw_match.end(), len(block_text)):
                    if block_text[i] in '.;\n':
                        clause_end = i + 1
                        break

                clause_text = block_text[kw_start:clause_end].strip()
                if len(clause_text) < 8:
                    continue

                real_start = m.start() + kw_start
                real_end = m.start() + clause_end

                section = tree.find_section_for_offset(real_start)
                section_id = section.section_id if section else None
                page = tree.get_page_for_offset(real_start) if tree.page_map else (section.page_number if section else None)

                spans.append(RawSpan(
                    channel="C",
                    text=clause_text,
                    start=real_start,
                    end=real_end,
                    confidence=0.90,
                    page_number=page,
                    section_id=section_id,
                    source_block_type="paragraph",
                    channel_metadata={
                        "pattern": "exception_clause",
                        "regex_match": clause_text[:60],
                    },
                ))

        return spans

    # =========================================================================
    # V4.3: THRESHOLD BINDING (Group B — upstream fix)
    #
    # Channels currently extract "potassium" (lab_test) and "≤4.8 mmol/L"
    # (threshold) as separate spans.  They don't cluster in Signal Merger
    # because they don't overlap (gap of 5-30 chars between them).
    #
    # Within rec/PP blocks, this method finds adjacent lab_test + threshold
    # pairs and creates a single bound span covering both.
    # =========================================================================

    # Lab parameters that should be bound to nearby thresholds
    _LAB_PARAM_RE = re.compile(
        r'\b(?:e?GFR|CrCl|potassium|K\+|serum\s+creatinine|creatinine|'
        r'sodium|Na\+|HbA1c|A1C|albumin|UACR|ACR|hemoglobin|'
        r'phosphate|calcium|bicarbonate)\b',
        re.IGNORECASE,
    )

    # Numeric value with clinical units (standalone — no leading parameter)
    _CLINICAL_VALUE_RE = re.compile(
        r'(?:' + _OP + r')?\s*\d+(?:\.\d+)?\s*'
        r'(?:mEq/L|mmol/L|mg/dL|mg/g|µmol/L|g/dL|mg/mmol|'
        r'mL/min(?:/1\.73\s*m[²2])?|%)',
        re.IGNORECASE,
    )

    def _bind_thresholds_in_recs(
        self, spans: list[RawSpan], text: str, tree: GuidelineTree,
    ) -> list[RawSpan]:
        """Bind lab parameters to numeric values within rec/PP blocks.

        When a lab parameter and a clinical value appear within 50 characters
        of each other in the same rec/PP block, creates a combined span
        covering the full binding (e.g. "potassium ≤4.8 mmol/L").

        Returns the original spans PLUS the new bound spans.
        """
        # Build rec/PP ranges
        rec_ranges: list[tuple[int, int]] = []
        for m in self._REC_PP_START_RE.finditer(text):
            remaining = text[m.end():min(m.end() + 3000, len(text))]
            end_match = self._REC_PP_END_RE.search(remaining)
            block_end = (
                m.end() + end_match.start()
                if end_match
                else min(m.end() + 3000, len(text))
            )
            rec_ranges.append((m.start(), block_end))

        if not rec_ranges:
            return spans

        # Index existing spans by category
        THRESHOLD_CATS = {
            "egfr_threshold", "egfr_range", "potassium_threshold", "dose_value",
        }
        lab_spans = [
            s for s in spans
            if s.channel_metadata.get("pattern") == "lab_test"
        ]
        threshold_spans = [
            s for s in spans
            if s.channel_metadata.get("pattern") in THRESHOLD_CATS
        ]

        bound_spans: list[RawSpan] = []
        used_thresholds: set[int] = set()  # index into threshold_spans

        for rec_start, rec_end in rec_ranges:
            block_labs = [
                s for s in lab_spans if rec_start <= s.start < rec_end
            ]
            block_thresholds = [
                (i, s) for i, s in enumerate(threshold_spans)
                if rec_start <= s.start < rec_end
            ]

            for lab in block_labs:
                # Find the nearest unused threshold within 50 chars AFTER
                best_thresh: Optional[RawSpan] = None
                best_idx = -1
                best_gap = 51

                for idx, thresh in block_thresholds:
                    if idx in used_thresholds:
                        continue
                    gap = thresh.start - lab.end
                    if 0 <= gap < best_gap:
                        best_thresh = thresh
                        best_idx = idx
                        best_gap = gap

                if best_thresh is None or best_gap >= 50:
                    continue

                bound_text = text[lab.start:best_thresh.end]
                bound_spans.append(RawSpan(
                    channel="C",
                    text=bound_text,
                    start=lab.start,
                    end=best_thresh.end,
                    confidence=max(lab.confidence, best_thresh.confidence),
                    page_number=lab.page_number,
                    section_id=lab.section_id,
                    source_block_type=lab.source_block_type,
                    channel_metadata={
                        "pattern": "bound_threshold",
                        "lab_param": lab.text,
                        "threshold": best_thresh.text,
                    },
                ))
                used_thresholds.add(best_idx)

        return spans + bound_spans

    # =========================================================================
    # V4.3.1: PRESCRIPTIVE CLAUSE EXTRACTION (Group D2 — B1 residual fix)
    #
    # B1 (Token-Level Residual Analysis) finds Tier 1 residual fragments
    # on pages with prescriptive clinical content that mentions drugs.
    # Group D's PATTERNS create keyword-length spans (~20-40 chars) but
    # B1 expects the FULL prescriptive sentence to be covered.
    #
    # This method finds prescriptive verb frames, extracts the full
    # sentence (from sentence start to sentence end), and only keeps
    # sentences that mention a drug class.  Fires globally — B1 content
    # lives in rationale text, not rec/PP blocks.
    # =========================================================================

    # Prescriptive verb frames indicating clinical directives
    _PRESCRIPTIVE_VERB_RE = re.compile(
        r'\b(?:'
        # Modal + action (should/must + verb)
        r'should\s+(?:not\s+)?(?:be\s+)?'
        r'(?:counsele?d|avoided|used|prescribed|administered|initiated|'
        r'combined|done|monitored|discontinued|stopped|withdrawn|withheld|'
        r'undertaken|drive|driven|increased|decreased|adjusted|reduced|continued)'
        r'|must\s+(?:not\s+)?(?:be\s+)?'
        r'(?:undertaken|exercised|used|prescribed|monitored)'
        # Negated capability
        r'|cannot\s+be\s+(?:confidently\s+)?'
        r'(?:refuted|recommended|replaced|used|combined)'
        # Risk escalation (standalone — no preceding "risk" anchor needed)
        r'|(?:was|were)\s+significantly\s+'
        r'(?:increased|higher|elevated|reduced)'
        # Evidence absence
        r'|(?:no|insufficient)\s+evidence\s+that'
        # Caution mandate
        r'|caution\s+must\s+be'
        r')\b',
        re.IGNORECASE,
    )

    # Drug class names that appear in B1 residual contexts.
    # Kept small and specific to avoid false positives.
    _DRUG_CLASS_RE = re.compile(
        r'\b(?:ACEi?|ACE\s+inhibitor|ARBs?|angiotensin|MRA|'
        r'mineralocorticoid|SGLT2i?|metformin|insulin|GLP-1\s*RA|'
        r'finerenone|empagliflozin|dapagliflozin|canagliflozin|'
        r'semaglutide|liraglutide|RAS\s*blockade|steroidal|nonsteroidal|'
        r'antihypertensive)\b',
        re.IGNORECASE,
    )

    def _extract_prescriptive_clauses(
        self, text: str, tree: GuidelineTree,
    ) -> list[RawSpan]:
        """Extract full prescriptive sentences containing drug mentions.

        Unlike PATTERNS (keyword-length spans), this creates sentence-level
        spans (80-300 chars) to provide B1-sufficient coverage.

        Only fires on sentences that contain BOTH:
        - A prescriptive verb frame (should/must/cannot + clinical action)
        - A drug class mention (ACEi, ARB, MRA, SGLT2i, etc.)

        Fires globally across the document, not scoped to rec/PP blocks,
        because prescriptive content in rationale text needs B1 coverage.
        """
        spans: list[RawSpan] = []
        seen_ranges: set[tuple[int, int]] = set()

        for verb_match in self._PRESCRIPTIVE_VERB_RE.finditer(text):
            # Walk backward to sentence start (previous ". " or paragraph)
            sent_start = verb_match.start()
            search_back = max(0, verb_match.start() - 250)
            for i in range(verb_match.start() - 1, search_back, -1):
                if text[i] == '.' and i + 1 < len(text) and text[i + 1] in ' \n':
                    sent_start = i + 1
                    break
                if text[i] == '\n' and (i == 0 or text[i - 1] == '\n'):
                    sent_start = i + 1
                    break
            else:
                sent_start = search_back

            # Extend backward: if the preceding sentence also mentions a drug,
            # include it to bridge B1 residual gaps between adjacent sentences.
            if sent_start > 0:
                prev_end = sent_start
                prev_start = sent_start
                prev_search = max(0, sent_start - 250)
                for i in range(sent_start - 2, prev_search, -1):
                    if text[i] == '.' and i + 1 < len(text) and text[i + 1] in ' \n':
                        prev_start = i + 1
                        break
                    if text[i] == '\n' and (i == 0 or text[i - 1] == '\n'):
                        prev_start = i + 1
                        break
                else:
                    prev_start = prev_search
                prev_sentence = text[prev_start:prev_end]
                if self._DRUG_CLASS_RE.search(prev_sentence) and len(prev_sentence) < 300:
                    sent_start = prev_start

            # Walk forward to sentence end (next "." followed by space/newline)
            sent_end = verb_match.end()
            search_fwd = min(len(text), verb_match.end() + 300)
            for i in range(verb_match.end(), search_fwd):
                if text[i] == '.' and (i + 1 >= len(text) or text[i + 1] in ' \n'):
                    sent_end = i + 1
                    break
            else:
                sent_end = search_fwd

            sentence = text[sent_start:sent_end].strip()

            # Filter: must contain a drug class mention
            if not self._DRUG_CLASS_RE.search(sentence):
                continue

            # Skip very short or very long sentences
            if len(sentence) < 40 or len(sentence) > 500:
                continue

            # Deduplicate overlapping sentences
            range_key = (sent_start // 50, sent_end // 50)
            if range_key in seen_ranges:
                continue
            seen_ranges.add(range_key)

            section = tree.find_section_for_offset(sent_start)
            section_id = section.section_id if section else None
            page = tree.get_page_for_offset(sent_start) if tree.page_map else (section.page_number if section else None)

            spans.append(RawSpan(
                channel="C",
                text=sentence,
                start=sent_start,
                end=sent_end,
                confidence=0.85,
                page_number=page,
                section_id=section_id,
                source_block_type="paragraph",
                channel_metadata={
                    "pattern": "prescriptive_clause",
                    "verb_frame": verb_match.group()[:40],
                },
            ))

        return spans


# Short alias for pipeline imports
ChannelC = ChannelCGrammar
