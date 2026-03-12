#!/usr/bin/env python3
"""
Calibration Verification Script — B1 Alignment + A1d Footnote Detection

Runs against existing pipeline output (no CUDA or API key required).
Tests two critical calibration points identified during cross-check:

1. B1 Token-Level Alignment: What % of source tokens are covered by spans?
   Expected: 75-90% on content pages (excl. references/front matter)
   If <75%: extraction incomplete → rationale text with prescribing content missing
   If >95%: alignment too loose → matching noise
   Current KDIGO baseline: ~70% (15 Tier 1 gaps in ACEi/ARB, MRA, ESA rationale)

2. A1d Footnote Detection: How many footnote markers does PyMuPDF detect?
   Expected: 5-15 symbol markers in KDIGO 2022 Diabetes-in-CKD
   (KDIGO uses only †‡§¶ symbols, no numbered superscripts ¹²³)
   If significantly lower: markers encoded as regular text, not superscript

Usage:
    python verify_calibration.py

Reads from the most recent full-guide job in data/output/v4/.
"""

import json
import os
import re
import sys

_script_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _script_dir)
sys.path.insert(0, os.path.join(_script_dir, '..', '..'))

# ═══════════════════════════════════════════════════════════════════════════
# Configuration
# ═══════════════════════════════════════════════════════════════════════════

JOB_DIR = os.path.join(
    _script_dir, "data", "output", "v4",
    "job_marker_dfdb5212-9587-402b-b4df-8ab3fce831a5",
)
PDF_PATH = os.path.join(
    _script_dir, "data", "pdfs",
    "KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf",
)

# B1 thresholds
B1_NGRAM_WINDOW = 6
B1_OVERLAP_THRESHOLD = 0.80
B1_EXPECTED_COVERAGE_MIN = 75.0
B1_EXPECTED_COVERAGE_MAX = 90.0

# A1d expected (KDIGO uses symbol markers only, no numbered superscripts)
A1D_EXPECTED_FOOTNOTES_MIN = 5
A1D_EXPECTED_FOOTNOTES_MAX = 15

FOOTNOTE_MARKERS = {"†", "‡", "*", "§", "¶", "||", "#"}

# Page classification (simplified — front matter + references excluded)
EXCLUDE_PAGES_BEFORE = 5   # Front matter pages (0-4)
EXCLUDE_PAGES_AFTER = 64   # References section starts at page 65 in KDIGO 2022


def _tokenize(text: str) -> list[tuple[str, int, int]]:
    """Tokenize text into (word, start, end) tuples."""
    tokens = []
    for m in re.finditer(r'\b\w+\b', text.lower()):
        tokens.append((m.group(), m.start(), m.end()))
    return tokens


def _extract_words(text: str) -> list[str]:
    """Extract lowercase words from text."""
    return re.findall(r'\b\w+\b', text.lower())


# ═══════════════════════════════════════════════════════════════════════════
# CHECK 1: B1 Token-Level Alignment Coverage
# ═══════════════════════════════════════════════════════════════════════════

def check_b1_alignment():
    """Measure what % of source content tokens are covered by merged spans.

    Uses inverted index approach (same as coverage_guard.py _b1_token_residual):
    - Build word → [positions] map from source tokens (O(n))
    - For each span: find rarest word, look up anchor positions, check overlap
    - Complexity: O(spans × avg_candidates) instead of O(spans × tokens)
    """
    from collections import defaultdict
    import time

    print("=" * 70)
    print("CHECK 1: B1 Token-Level Alignment Coverage (Inverted Index)")
    print("=" * 70)

    # Load artifacts
    text_path = os.path.join(JOB_DIR, "normalized_text.txt")
    spans_path = os.path.join(JOB_DIR, "merged_spans.json")

    if not os.path.isfile(text_path) or not os.path.isfile(spans_path):
        print("  ERROR: Missing artifacts. Run pipeline first.")
        return None

    with open(text_path, "r") as f:
        normalized_text = f.read()

    with open(spans_path, "r") as f:
        merged_spans = json.load(f)

    print(f"  Source text: {len(normalized_text):,} chars")
    print(f"  Merged spans: {len(merged_spans):,}")

    # Tokenize source
    source_tokens = _tokenize(normalized_text)
    print(f"  Source tokens: {len(source_tokens):,}")

    t0 = time.time()

    # Build coverage bitmap
    covered = [False] * len(source_tokens)

    # Build inverted index: word → list of token positions
    word_index: dict[str, list[int]] = defaultdict(list)
    for idx, (word, _, _) in enumerate(source_tokens):
        word_index[word].append(idx)

    print(f"  Inverted index: {len(word_index):,} unique words")

    # Match each span against source using inverted index
    for span in merged_spans:
        span_text = span.get("reviewer_text") or span.get("text", "")
        span_words = _extract_words(span_text)

        if len(span_words) < 2:
            # Short spans: try substring match
            span_lower = span_text.lower().strip()
            if len(span_lower) >= 5:
                # Simple substring: mark tokens whose text appears in span
                for idx, (tok, _, _) in enumerate(source_tokens):
                    if tok in span_lower and not covered[idx]:
                        covered[idx] = True
            continue

        span_word_set = set(span_words)

        # Pick rarest word as anchor (fewest source occurrences)
        rarest_word = min(span_word_set, key=lambda w: len(word_index.get(w, [])))
        anchor_positions = word_index.get(rarest_word, [])

        if not anchor_positions:
            continue

        # For each anchor position, check if surrounding region matches
        window = min(len(span_words), 8)
        half_window = window // 2

        for anchor_pos in anchor_positions:
            start = max(0, anchor_pos - half_window)
            end = min(len(source_tokens), anchor_pos + len(span_words) + half_window)
            region_words = {source_tokens[j][0] for j in range(start, end)}

            overlap = len(span_word_set & region_words) / len(span_word_set)
            if overlap >= 0.5:  # Confirmed passage match
                # Mark entire region as covered (passage-level coverage)
                for j in range(start, end):
                    covered[j] = True

    elapsed = time.time() - t0
    print(f"  Alignment time: {elapsed:.1f}s")

    # Calculate coverage (content pages only)
    # Detect page markers: try multiple formats
    # Format 1: <!-- PAGE N -->  (MonkeyOCR output)
    # Format 2: {N}---...        (Marker v1.10 output)
    page_starts = {}
    for pat in [r'<!-- PAGE (\d+) -->', r'\{(\d+)\}[-]{10,}']:
        for m in re.finditer(pat, normalized_text):
            page_num = int(m.group(1))
            page_starts[m.start()] = page_num
        if page_starts:
            break  # Use first format that matches

    # Classify tokens by page
    content_covered = 0
    content_total = 0
    all_covered = sum(covered)

    if page_starts:
        sorted_page_offsets = sorted(page_starts.items())
        for idx, (tok_word, tok_start, tok_end) in enumerate(source_tokens):
            # Find which page this token belongs to
            page_num = 1
            for offset, pnum in sorted_page_offsets:
                if tok_start >= offset:
                    page_num = pnum
                else:
                    break

            if EXCLUDE_PAGES_BEFORE <= page_num <= EXCLUDE_PAGES_AFTER:
                content_total += 1
                if covered[idx]:
                    content_covered += 1
    else:
        # No page markers — use all tokens
        content_total = len(source_tokens)
        content_covered = all_covered

    overall_pct = (all_covered / len(source_tokens)) * 100 if source_tokens else 0
    content_pct = (content_covered / content_total) * 100 if content_total else 0

    print(f"\n  Results:")
    print(f"    Overall coverage: {all_covered:,}/{len(source_tokens):,} tokens = {overall_pct:.1f}%")
    print(f"    Content-page coverage: {content_covered:,}/{content_total:,} tokens = {content_pct:.1f}%")

    if content_pct < B1_EXPECTED_COVERAGE_MIN - 0.05:  # Allow float rounding
        print(f"\n  ⚠️  BELOW EXPECTED RANGE ({B1_EXPECTED_COVERAGE_MIN}-{B1_EXPECTED_COVERAGE_MAX}%)")
        print(f"    Alignment may be too strict. Consider:")
        print(f"    - Reducing B1_OVERLAP_THRESHOLD from {B1_OVERLAP_THRESHOLD}")
        print(f"    - Increasing B1_NGRAM_WINDOW from {B1_NGRAM_WINDOW}")
    elif content_pct > B1_EXPECTED_COVERAGE_MAX:
        print(f"\n  ⚠️  ABOVE EXPECTED RANGE ({B1_EXPECTED_COVERAGE_MIN}-{B1_EXPECTED_COVERAGE_MAX}%)")
        print(f"    Alignment may be too loose — matching noise.")
    else:
        print(f"\n  ✅ Within expected range ({B1_EXPECTED_COVERAGE_MIN}-{B1_EXPECTED_COVERAGE_MAX}%)")

    return content_pct


# ═══════════════════════════════════════════════════════════════════════════
# CHECK 2: A1d Footnote Detection via PyMuPDF Superscript Flags
# ═══════════════════════════════════════════════════════════════════════════

def check_a1d_footnotes():
    """Count footnote markers detected via PyMuPDF superscript flags."""
    print("\n" + "=" * 70)
    print("CHECK 2: A1d Footnote Detection (PyMuPDF Superscript)")
    print("=" * 70)

    if not os.path.isfile(PDF_PATH):
        print(f"  ERROR: PDF not found at {PDF_PATH}")
        return None

    try:
        import pymupdf
    except ImportError:
        print("  ERROR: pymupdf not installed")
        return None

    doc = pymupdf.open(PDF_PATH)
    print(f"  PDF: {os.path.basename(PDF_PATH)}")
    print(f"  Pages: {len(doc)}")

    # Scan for superscript footnote markers
    superscript_markers = []
    inline_markers = []
    table_page_markers = {}  # page -> list of markers

    for page_num in range(len(doc)):
        page = doc[page_num]
        rawdict = page.get_text("rawdict")

        for block in rawdict.get("blocks", []):
            if block.get("type") != 0:  # text block only
                continue

            for line in block.get("lines", []):
                for span in line.get("spans", []):
                    text = span.get("text", "").strip()
                    flags = span.get("flags", 0)
                    font_size = span.get("size", 12.0)

                    # Method 1: PyMuPDF superscript flag (bit 0)
                    is_superscript = bool(flags & 1)

                    # Method 2: Small font size (< 7pt typically superscript)
                    is_small = font_size < 7.0

                    if (is_superscript or is_small) and text:
                        # Check if this is a known footnote marker
                        clean = text.strip()
                        if clean in FOOTNOTE_MARKERS or (
                            len(clean) <= 3 and any(c in FOOTNOTE_MARKERS for c in clean)
                        ):
                            superscript_markers.append({
                                "text": clean,
                                "page": page_num + 1,
                                "flags": flags,
                                "size": font_size,
                                "method": "superscript_flag" if is_superscript else "small_font",
                                "bbox": span.get("bbox", []),
                            })

                    # Method 3: Inline marker characters (not flagged)
                    elif text in FOOTNOTE_MARKERS:
                        inline_markers.append({
                            "text": text,
                            "page": page_num + 1,
                            "flags": flags,
                            "size": font_size,
                            "method": "inline_text",
                        })

    doc.close()

    # Deduplicate by (text, page)
    seen = set()
    unique_markers = []
    for m in superscript_markers:
        key = (m["text"], m["page"])
        if key not in seen:
            seen.add(key)
            unique_markers.append(m)

    unique_inline = []
    for m in inline_markers:
        key = (m["text"], m["page"])
        if key not in seen:
            seen.add(key)
            unique_inline.append(m)

    total = len(unique_markers) + len(unique_inline)

    print(f"\n  Results:")
    print(f"    Superscript-flagged markers: {len(unique_markers)}")
    print(f"    Inline text markers: {len(unique_inline)}")
    print(f"    Total unique markers: {total}")

    if unique_markers:
        print(f"\n  Superscript markers by page:")
        by_page = {}
        for m in unique_markers:
            by_page.setdefault(m["page"], []).append(m["text"])
        for pg in sorted(by_page.keys()):
            print(f"    Page {pg}: {', '.join(by_page[pg])}")

    if unique_inline:
        print(f"\n  Inline markers by page:")
        by_page = {}
        for m in unique_inline:
            by_page.setdefault(m["page"], []).append(m["text"])
        for pg in sorted(by_page.keys())[:10]:
            print(f"    Page {pg}: {', '.join(by_page[pg])}")
        if len(by_page) > 10:
            print(f"    ... and {len(by_page) - 10} more pages")

    if total < A1D_EXPECTED_FOOTNOTES_MIN:
        print(f"\n  ⚠️  BELOW EXPECTED ({A1D_EXPECTED_FOOTNOTES_MIN}-{A1D_EXPECTED_FOOTNOTES_MAX})")
        print(f"    Some markers may be encoded as regular text.")
        print(f"    A1d may need to also scan for inline markers, not just superscript.")
    elif total > A1D_EXPECTED_FOOTNOTES_MAX:
        print(f"\n  ⚠️  ABOVE EXPECTED ({A1D_EXPECTED_FOOTNOTES_MIN}-{A1D_EXPECTED_FOOTNOTES_MAX})")
        print(f"    May be over-counting — check for false positives.")
    else:
        print(f"\n  ✅ Within expected range ({A1D_EXPECTED_FOOTNOTES_MIN}-{A1D_EXPECTED_FOOTNOTES_MAX})")

    return total


# ═══════════════════════════════════════════════════════════════════════════
# MAIN
# ═══════════════════════════════════════════════════════════════════════════

if __name__ == "__main__":
    print("CoverageGuard Calibration Verification")
    print(f"Job: {os.path.basename(JOB_DIR)}")
    print(f"PDF: {os.path.basename(PDF_PATH)}")
    print()

    b1_pct = check_b1_alignment()
    a1d_count = check_a1d_footnotes()

    print("\n" + "=" * 70)
    print("SUMMARY")
    print("=" * 70)
    if b1_pct is not None:
        status = "✅" if B1_EXPECTED_COVERAGE_MIN <= b1_pct <= B1_EXPECTED_COVERAGE_MAX else "⚠️"
        print(f"  {status} B1 Content Coverage: {b1_pct:.1f}% (expected {B1_EXPECTED_COVERAGE_MIN}-{B1_EXPECTED_COVERAGE_MAX}%)")
    if a1d_count is not None:
        status = "✅" if A1D_EXPECTED_FOOTNOTES_MIN <= a1d_count <= A1D_EXPECTED_FOOTNOTES_MAX else "⚠️"
        print(f"  {status} A1d Footnote Markers: {a1d_count} (expected {A1D_EXPECTED_FOOTNOTES_MIN}-{A1D_EXPECTED_FOOTNOTES_MAX})")
    print()
