"""
L1 Completeness Oracle — PyMuPDF rawdict verification of Marker output.

Zero-model, deterministic verification layer that diffs every character in the
PDF text layer (via PyMuPDF rawdict) against Marker's extracted markdown to
detect silent text loss.

Architecture position:  Marker → **Oracle** → Channel 0
Gate logic:             HIGH priority misses = 0
                        (coverage % is a dashboard metric, NOT a gate criterion)
Recovery:               HIGH misses → MergedSpan(contributing_channels=["L1_RECOVERY"])
                        injected into reviewer queue with distinct source label.

Multi-tier matching strategy:
    Tier 1   — Exact substring:  rawdict_block in marker_text
    Tier 2   — Quad check:       ≥50% of 4-word subsequences found
    Tier 2.5 — Word overlap:     ≥70% of content words found (adaptive:
                                 3+ chars for short blocks, 5+ for long)
    Tier 3   — Fuzzy ratio:      rapidfuzz partial_ratio ≥ 80

Clinical signal triage for missed blocks:
    HIGH       — Drug name (Aho-Corasick) or clinical threshold (regex)
    LOW        — No clinical signal (page headers, footers, decorative text)
    IMAGE_TEXT — Block from image region (rawdict found nothing → vision needed)

Usage:
    from l1_completeness_oracle import L1CompletenessOracle

    oracle = L1CompletenessOracle()
    report = oracle.validate("guideline.pdf", marker_markdown_text)

    if not report.gate_passed:
        recovery_spans = oracle.recovery_merged_spans(report, job_id)
        merged_spans.extend(recovery_spans)
"""

import re
import time
from dataclasses import dataclass, field
from typing import Literal, Optional
from uuid import UUID, uuid4

import pymupdf
import ahocorasick
from rapidfuzz import fuzz

from clinical_constants import (
    KDIGO_DRUG_NAMES,
    CLINICAL_THRESHOLD_RE,
    NUMERIC_THRESHOLD_RE,
    RECOMMENDATION_RE,
)


# ═══════════════════════════════════════════════════════════════════════════════
# Data Models
# ═══════════════════════════════════════════════════════════════════════════════

@dataclass
class RawDictBlock:
    """A text block extracted from PyMuPDF rawdict (PDF text layer)."""
    text: str
    page_number: int          # Absolute page number (with offset applied)
    bbox: tuple               # (x0, y0, x1, y1) in PDF points
    font_size: float = 0.0
    is_bold: bool = False
    char_count: int = 0       # Length of text (cached for coverage calc)


@dataclass
class MatchResult:
    """Result of multi-tier block matching."""
    matched: bool
    tier: Literal["exact", "quad", "word_overlap", "fuzzy", "skip", "none"]
    best_score: float         # 0-100 scale


@dataclass
class MissedBlock:
    """A rawdict block that Marker's output does not contain."""
    block: RawDictBlock
    priority: Literal["HIGH", "LOW", "IMAGE_TEXT"]
    reason: str               # Human-readable triage explanation
    best_match_score: float = 0.0
    surrounding_context: str = ""   # Adjacent block text for reviewer context


@dataclass
class CompletenessReport:
    """Result of L1 Oracle validation.

    Gate criterion:   gate_passed = (high_priority_misses == 0)
    Dashboard metrics: char_coverage_pct, block_coverage_pct (for humans, NOT gates)
    """
    total_rawdict_blocks: int
    matched_blocks: int
    missed_blocks: list[MissedBlock] = field(default_factory=list)

    # Dashboard metrics — informational only, NOT gate criteria
    char_coverage_pct: float = 100.0
    block_coverage_pct: float = 100.0
    total_rawdict_chars: int = 0
    matched_chars: int = 0

    # Gate criteria
    high_priority_misses: int = 0
    low_priority_misses: int = 0
    image_text_gaps: int = 0

    elapsed_ms: float = 0.0

    @property
    def gate_passed(self) -> bool:
        """Pipeline gate: passes if and only if zero HIGH-priority misses."""
        return self.high_priority_misses == 0

    def summary(self) -> str:
        gate = "PASS" if self.gate_passed else "FAIL"
        return (
            f"L1 Oracle: {gate} | "
            f"blocks {self.matched_blocks}/{self.total_rawdict_blocks} "
            f"({self.block_coverage_pct:.1f}%) | "
            f"chars {self.matched_chars:,}/{self.total_rawdict_chars:,} "
            f"({self.char_coverage_pct:.1f}%) | "
            f"HIGH={self.high_priority_misses} LOW={self.low_priority_misses} "
            f"IMAGE_TEXT={self.image_text_gaps} | "
            f"{self.elapsed_ms:.0f}ms"
        )


# ═══════════════════════════════════════════════════════════════════════════════
# L1 Completeness Oracle
# ═══════════════════════════════════════════════════════════════════════════════

class L1CompletenessOracle:
    """
    Zero-model verification: diffs PyMuPDF rawdict (ground truth from PDF text
    layer) against Marker output to detect silent text loss.

    Position in pipeline: Marker → Oracle → Channel 0
    Gate:                 HIGH priority misses = 0
    Recovery:             HIGH misses → MergedSpan with channel="L1_RECOVERY"
    """

    VERSION = "1.1.0"

    # Minimum block length to consider (skip page numbers, single chars)
    MIN_BLOCK_CHARS = 8

    # Fuzzy match threshold (0-100 rapidfuzz scale)
    FUZZY_THRESHOLD = 80

    # Quad check pass threshold (fraction of 4-word quads that must match)
    QUAD_PASS_FRACTION = 0.5

    def __init__(self, drug_names: Optional[list] = None):
        """
        Args:
            drug_names: Additional drug names for Aho-Corasick triage.
                        Merged with built-in KDIGO drug list.
        """
        all_drugs = list(KDIGO_DRUG_NAMES)
        if drug_names:
            all_drugs.extend(drug_names)

        self._automaton = ahocorasick.Automaton()
        for drug in all_drugs:
            key = drug.lower().strip()
            if key:
                self._automaton.add_word(key, key)
        self._automaton.make_automaton()

    # ─── Public API ───────────────────────────────────────────────────────

    def validate(
        self,
        pdf_path: str,
        marker_text: str,
        page_offset: int = 0,
    ) -> CompletenessReport:
        """
        Compare PyMuPDF rawdict against Marker output.

        Args:
            pdf_path:    Path to the PDF file (or page subset).
            marker_text: Complete markdown output from Marker.
            page_offset: Absolute page number of the first page in the PDF.
                         E.g., if processing pages 58-61 extracted from a
                         128-page guide, pass page_offset=58.

        Returns:
            CompletenessReport with gate status and missed blocks.
        """
        t0 = time.perf_counter()

        # Step 1: Extract every text block from PDF text layer
        rawdict_blocks = self._extract_rawdict_blocks(pdf_path, page_offset)

        # Step 2: Normalize marker text once for all comparisons
        marker_normalized = self._normalize(marker_text)

        # Step 3: Three-tier match each rawdict block against marker text
        matched_count = 0
        matched_chars = 0
        total_chars = 0
        missed: list[MissedBlock] = []

        for i, block in enumerate(rawdict_blocks):
            total_chars += block.char_count

            result = self._match_block(block.text, marker_normalized)

            if result.matched:
                matched_count += 1
                matched_chars += block.char_count
            else:
                priority, reason = self._triage_miss(block)

                # Build surrounding_context from adjacent same-page blocks
                ctx_parts = []
                if i > 0 and rawdict_blocks[i - 1].page_number == block.page_number:
                    ctx_parts.append(f"[BEFORE] {rawdict_blocks[i - 1].text[:200]}")
                if i + 1 < len(rawdict_blocks) and rawdict_blocks[i + 1].page_number == block.page_number:
                    ctx_parts.append(f"[AFTER] {rawdict_blocks[i + 1].text[:200]}")

                missed.append(MissedBlock(
                    block=block,
                    priority=priority,
                    reason=reason,
                    best_match_score=result.best_score,
                    surrounding_context="\n".join(ctx_parts),
                ))

        elapsed_ms = (time.perf_counter() - t0) * 1000

        high = sum(1 for m in missed if m.priority == "HIGH")
        low = sum(1 for m in missed if m.priority == "LOW")
        image = sum(1 for m in missed if m.priority == "IMAGE_TEXT")

        return CompletenessReport(
            total_rawdict_blocks=len(rawdict_blocks),
            matched_blocks=matched_count,
            missed_blocks=missed,
            char_coverage_pct=(matched_chars / total_chars * 100) if total_chars > 0 else 100.0,
            block_coverage_pct=(matched_count / len(rawdict_blocks) * 100) if rawdict_blocks else 100.0,
            total_rawdict_chars=total_chars,
            matched_chars=matched_chars,
            high_priority_misses=high,
            low_priority_misses=low,
            image_text_gaps=image,
            elapsed_ms=elapsed_ms,
        )

    def recovery_merged_spans(
        self,
        report: CompletenessReport,
        job_id: UUID,
    ) -> list:
        """
        Convert HIGH-priority missed blocks into MergedSpan objects tagged
        as L1_RECOVERY for injection into the reviewer queue.

        These spans did NOT go through Channel 0 normalization or any
        extraction channel. They are raw PDF text that Marker silently dropped.
        The reviewer must see them labeled distinctly (yellow highlight, not blue).

        Returns:
            List of MergedSpan objects (imported from extraction.v4.models).
        """
        from extraction.v4.models import MergedSpan

        recovery_spans = []
        for miss in report.missed_blocks:
            if miss.priority != "HIGH":
                continue

            span = MergedSpan(
                id=uuid4(),
                job_id=job_id,
                text=miss.block.text,
                start=-1,       # Sentinel: not an offset in Marker text
                end=-1,         # (this text is absent from Marker output)
                contributing_channels=["L1_RECOVERY"],
                channel_confidences={"L1_RECOVERY": 1.0},
                merged_confidence=1.0,  # Deterministic source — perfect confidence
                has_disagreement=True,  # Flag for reviewer attention
                disagreement_detail=(
                    f"L1_RECOVERY: Marker silently dropped this text. "
                    f"Source: PyMuPDF rawdict page {miss.block.page_number}. "
                    f"Triage: {miss.reason}"
                ),
                page_number=miss.block.page_number,
                section_id=None,
                table_id=None,
                bbox=list(miss.block.bbox),
                surrounding_context=miss.surrounding_context or None,
                review_status="PENDING",
            )
            recovery_spans.append(span)

        return recovery_spans

    # ─── PyMuPDF rawdict extraction ───────────────────────────────────────

    def _extract_rawdict_blocks(
        self,
        pdf_path: str,
        page_offset: int,
    ) -> list[RawDictBlock]:
        """
        Extract text blocks from PDF text layer using PyMuPDF dict.

        Uses "dict" mode (not "rawdict") because many clinical PDFs use
        CMap-encoded custom fonts (e.g., AdvOT3bbb1fa6.B in KDIGO guides).
        "rawdict" returns raw byte sequences *before* CMap decoding — empty
        strings for these fonts. "dict" applies CMap translation and gives
        proper Unicode text. Both modes are deterministic, zero-model, and
        read the PDF text layer directly. Fast (<0.1s per page).

        Extracts at the BLOCK level (PyMuPDF visual grouping). Large blocks
        (>200 chars) are handled by the word-overlap matching tier which is
        tolerant of Marker's paragraph reformatting.
        """
        blocks = []
        doc = pymupdf.open(pdf_path)

        for page_idx in range(len(doc)):
            page = doc[page_idx]
            rawdict = page.get_text("dict")

            for block in rawdict.get("blocks", []):
                # type 0 = text block, type 1 = image block
                if block.get("type") != 0:
                    continue

                # Reconstruct block text from spans
                block_text_parts = []
                max_font_size = 0.0
                is_bold = False

                for line in block.get("lines", []):
                    line_text = ""
                    for span in line.get("spans", []):
                        line_text += span.get("text", "")
                        font_size = span.get("size", 0.0)
                        if font_size > max_font_size:
                            max_font_size = font_size
                        # PyMuPDF flags: bit 4 (16) = bold
                        if span.get("flags", 0) & 16:
                            is_bold = True
                    block_text_parts.append(line_text)

                # Join lines with space
                block_text = " ".join(block_text_parts).strip()

                if len(block_text) < self.MIN_BLOCK_CHARS:
                    continue

                bbox = block.get("bbox", (0, 0, 0, 0))
                blocks.append(RawDictBlock(
                    text=block_text,
                    page_number=page_offset + page_idx,
                    bbox=tuple(bbox),
                    font_size=max_font_size,
                    is_bold=is_bold,
                    char_count=len(block_text),
                ))

        doc.close()
        return blocks

    # ─── Text normalization ───────────────────────────────────────────────

    @staticmethod
    def _normalize(text: str) -> str:
        """Normalize text for comparison.

        Minimal normalization: lowercase, rejoin hyphenated line breaks,
        collapse whitespace.  Applied identically to both rawdict and
        Marker text.

        Deliberately does NOT strip HTML tags or markdown formatting.
        Empirical testing on KDIGO pages 58-61 showed that HTML stripping
        counterintuitively hurts matching (60/78 → 54/78) because it
        changes word-to-word distances in Marker text, breaking quad
        matching.  The tags are harmless noise that the quad and fuzzy
        tiers already tolerate.
        """
        t = text.lower()
        # Rejoin hyphenated line breaks: "con- trol" → "control"
        # Common in clinical PDFs where Marker preserves line-break hyphens
        t = re.sub(r'(\w)- +(\w)', r'\1\2', t)
        # Collapse all whitespace (newlines, tabs, multiple spaces)
        t = re.sub(r'\s+', ' ', t)
        return t.strip()

    # ─── Three-tier matching ──────────────────────────────────────────────

    # Word-overlap tier: minimum content word length and pass threshold
    WORD_OVERLAP_MIN_WORD_LEN = 5
    WORD_OVERLAP_PASS_FRACTION = 0.70
    WORD_OVERLAP_BLOCK_THRESHOLD = 15  # chars — catches table cells, bullet lists + large blocks

    def _match_block(self, block_text: str, marker_normalized: str) -> MatchResult:
        """
        Multi-tier matching strategy:
            Tier 1   — Exact substring match (handles 80%+ of blocks)
            Tier 2   — 4-word quad check (catches reformatted text)
            Tier 2.5 — Word-overlap for long blocks (catches paragraph reflows)
            Tier 3   — Fuzzy partial_ratio (catches minor alterations)

        Tier 2.5 exists because PyMuPDF groups entire page regions into single
        blocks that can span 1500+ chars across multiple Marker paragraphs.
        Quad check fails on these because word order diverges across reformatted
        sections.  Word-overlap is order-independent: checks if ≥70% of unique
        content words (5+ chars) from the block appear anywhere in Marker text.
        """
        normalized_block = self._normalize(block_text)

        # Too short to match meaningfully — assume present
        if len(normalized_block) < 4:
            return MatchResult(matched=True, tier="skip", best_score=100.0)

        # Tier 1: Exact substring
        if normalized_block in marker_normalized:
            return MatchResult(matched=True, tier="exact", best_score=100.0)

        # Tier 2: 4-word quad check
        words = normalized_block.split()
        if len(words) >= 4:
            quads_found = 0
            quads_total = len(words) - 3
            for i in range(quads_total):
                quad = " ".join(words[i : i + 4])
                if quad in marker_normalized:
                    quads_found += 1

            if quads_total > 0 and (quads_found / quads_total) >= self.QUAD_PASS_FRACTION:
                score = (quads_found / quads_total) * 100
                return MatchResult(matched=True, tier="quad", best_score=score)

        # Tier 2.5: Word-overlap check (order-independent, formatting-agnostic)
        # Catches three failure patterns that break quad matching:
        # 1. Bullet lists: "• Metformin • SGLT2..." where • breaks every quad
        # 2. Large blocks spanning multiple Marker paragraphs (reordered text)
        # 3. Short table cells: "HbA1c < 6.5% < 8.0%" reformatted with pipes
        # Adaptive min word length: shorter blocks use shorter words (3+ chars)
        # because clinical values like "6.5%" and "HbA1c" are <5 chars.
        if len(normalized_block) > self.WORD_OVERLAP_BLOCK_THRESHOLD:
            # Short blocks (<100 chars): use 3-char words (clinical values)
            # Long blocks (≥100 chars): use 5-char words (filter noise)
            min_wl = 3 if len(normalized_block) < 100 else self.WORD_OVERLAP_MIN_WORD_LEN
            content_words = set(w for w in words if len(w) >= min_wl)
            if len(content_words) >= 3:  # Need enough words for meaningful check
                found = sum(1 for w in content_words if w in marker_normalized)
                overlap = found / len(content_words)
                if overlap >= self.WORD_OVERLAP_PASS_FRACTION:
                    return MatchResult(
                        matched=True,
                        tier="word_overlap",
                        best_score=overlap * 100,
                    )

        # Tier 3: Fuzzy match (rapidfuzz)
        # partial_ratio finds the best matching substring — ideal for
        # matching a short block against a long document.
        score = fuzz.partial_ratio(normalized_block, marker_normalized)

        if score >= self.FUZZY_THRESHOLD:
            return MatchResult(matched=True, tier="fuzzy", best_score=score)

        return MatchResult(matched=False, tier="none", best_score=score)

    # ─── Clinical signal triage ───────────────────────────────────────────

    def _triage_miss(
        self,
        block: RawDictBlock,
    ) -> tuple[Literal["HIGH", "LOW", "IMAGE_TEXT"], str]:
        """
        Classify a missed block by clinical priority.

        HIGH  — Contains drug name or clinical threshold → patient safety risk
        LOW   — No clinical signal (decorative, headers, copyright)
        IMAGE_TEXT — Reserved for blocks where rawdict found nothing
                     (image-embedded text, needs Granite-Docling vision)
        """
        text_lower = block.text.lower()

        # Check 1: Drug names via Aho-Corasick
        drug_matches = set()
        for end_idx, drug in self._automaton.iter(text_lower):
            drug_matches.add(drug)

        if drug_matches:
            drugs_str = ", ".join(sorted(drug_matches))
            return (
                "HIGH",
                f"Drug name(s) in missed text: {drugs_str} | "
                f"\"{block.text[:100]}\"",
            )

        # Check 2: Clinical thresholds (lab values, biomarkers with comparators)
        if CLINICAL_THRESHOLD_RE.search(block.text):
            match = CLINICAL_THRESHOLD_RE.search(block.text)
            return (
                "HIGH",
                f"Clinical threshold in missed text: \"{match.group()[:60]}\" | "
                f"\"{block.text[:100]}\"",
            )

        # Check 3: Numeric thresholds with units
        if NUMERIC_THRESHOLD_RE.search(block.text):
            match = NUMERIC_THRESHOLD_RE.search(block.text)
            return (
                "HIGH",
                f"Numeric threshold in missed text: \"{match.group()[:60]}\" | "
                f"\"{block.text[:100]}\"",
            )

        # Check 4: Recommendation / Practice Point markers
        if RECOMMENDATION_RE.search(block.text):
            return (
                "HIGH",
                f"Guideline recommendation in missed text: "
                f"\"{block.text[:100]}\"",
            )

        # No clinical signal detected
        return (
            "LOW",
            f"No clinical signal: \"{block.text[:100]}\"",
        )
