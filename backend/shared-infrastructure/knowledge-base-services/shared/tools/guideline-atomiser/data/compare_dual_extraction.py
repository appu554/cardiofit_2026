#!/usr/bin/env python3
"""
V4.1 Dual Extraction Comparison: Marker vs Granite-Docling.

Runs Granite-Docling VlmPipeline on the KDIGO Quick Reference PDF and
compares its structural output against an existing Marker pipeline run.

This script measures the V4.1 hybrid architecture's structural gains:
- How many sections does Granite-Docling detect vs Marker regex?
- How many tables does Granite-Docling find (OTSL) vs Marker (pipe)?
- What structural elements (footnotes, captions, lists) does Granite detect
  that Marker's regex-only path misses entirely?
- What would the alignment confidence be for Channel A?

Usage:
    python compare_dual_extraction.py

    # Or specify paths explicitly:
    python compare_dual_extraction.py \
        --pdf /path/to/KDIGO-Quick-Reference.pdf \
        --marker-job /path/to/job_marker_xxx/
"""

import argparse
import json
import os
import sys
import time
from pathlib import Path

# Handle import paths — need guideline-atomiser/ (parent) for extraction.v4.*
_script_dir = os.path.dirname(os.path.abspath(__file__))
_parent_dir = os.path.dirname(_script_dir)  # guideline-atomiser/
_shared_dir = os.path.dirname(_parent_dir)  # shared/ (has extraction/)
sys.path.insert(0, _parent_dir)
sys.path.insert(0, _shared_dir)
sys.path.insert(0, _script_dir)

# ═══════════════════════════════════════════════════════════════════════════
# CLI
# ═══════════════════════════════════════════════════════════════════════════

parser = argparse.ArgumentParser(description="V4.1 Dual Extraction Comparison")
parser.add_argument(
    "--pdf",
    default=os.path.join(_script_dir, "pdfs", "KDIGO-2022-Diabetes-Guideline-Quick-Reference-Guide.pdf"),
    help="Path to KDIGO PDF",
)
parser.add_argument(
    "--marker-job",
    default=os.path.join(
        _script_dir, "output", "v4",
        "job_marker_d6108962-00f6-44f1-88a7-ece9b6adeccc"
    ),
    help="Path to existing Marker pipeline job directory",
)
args = parser.parse_args()


# ═══════════════════════════════════════════════════════════════════════════
# GRANITE-DOCLING EXTRACTION
# ═══════════════════════════════════════════════════════════════════════════

def run_granite_docling(pdf_path: str) -> dict:
    """Run Granite-Docling VlmPipeline and return structured result."""
    from extraction.v4.granite_docling_extractor import GraniteDoclingExtractor

    print("=" * 70)
    print("GRANITE-DOCLING VlmPipeline EXTRACTION")
    print("=" * 70)
    print(f"  PDF: {os.path.basename(pdf_path)}")
    print(f"  Size: {os.path.getsize(pdf_path) / 1024 / 1024:.1f} MB")
    print()

    extractor = GraniteDoclingExtractor()
    print("  Running VlmPipeline (this may take 1-5 minutes)...")
    start = time.monotonic()
    result = extractor.extract(pdf_path)
    elapsed = time.monotonic() - start

    if result.error:
        print(f"  ERROR: {result.error}")
        return {"error": result.error}

    print(f"  Done in {elapsed:.1f}s ({result.elapsed_ms:.0f}ms internal)")
    print()

    # Summarize
    print(f"  Sections detected: {len(result.sections)}")
    for s in result.sections:
        indent = "    " + "  " * (s.level - 1)
        print(f"{indent}[L{s.level}] {s.doctag_type}: {s.text[:80]}")

    print(f"\n  Tables detected (OTSL): {len(result.tables)}")
    for i, t in enumerate(result.tables):
        headers_str = ", ".join(t.column_headers[:4])
        if len(t.column_headers) > 4:
            headers_str += "..."
        print(f"    Table {i+1} (page {t.page_number}): {len(t.column_headers)} cols, "
              f"{t.row_count} rows — [{headers_str}]")

    print(f"\n  Enrichment elements: {len(result.elements)}")
    by_type = {}
    for e in result.elements:
        by_type.setdefault(e.doctag_type, []).append(e)
    for dtype, items in sorted(by_type.items()):
        print(f"    {dtype}: {len(items)}")
        for item in items[:3]:
            print(f"      - {item.text[:90]}...")

    print(f"\n  Total pages: {result.total_pages}")
    print()

    return {
        "sections": [
            {"text": s.text, "level": s.level, "page": s.page_number, "type": s.doctag_type}
            for s in result.sections
        ],
        "tables": [
            {
                "headers": t.column_headers, "row_count": t.row_count,
                "page": t.page_number, "otsl_length": len(t.raw_otsl),
            }
            for t in result.tables
        ],
        "elements": [
            {"text": e.text, "type": e.doctag_type, "page": e.page_number}
            for e in result.elements
        ],
        "total_pages": result.total_pages,
        "elapsed_s": elapsed,
    }


# ═══════════════════════════════════════════════════════════════════════════
# MARKER ANALYSIS (from existing job artifacts)
# ═══════════════════════════════════════════════════════════════════════════

def analyze_marker_job(job_dir: str) -> dict:
    """Load and analyze existing Marker pipeline job artifacts."""
    print("=" * 70)
    print("MARKER PIPELINE ANALYSIS (existing job)")
    print("=" * 70)
    print(f"  Job: {os.path.basename(job_dir)}")
    print()

    with open(os.path.join(job_dir, "job_metadata.json")) as f:
        meta = json.load(f)
    print(f"  Source: {meta['source_pdf']}")
    print(f"  Pipeline: v{meta['pipeline_version']}")
    print(f"  Raw spans: {meta['total_raw_spans']}")
    print(f"  Merged spans: {meta['total_merged_spans']}")

    with open(os.path.join(job_dir, "guideline_tree.json")) as f:
        tree = json.load(f)

    sections = tree["sections"]
    tables = tree["tables"]

    print(f"\n  Sections (regex-parsed): {len(sections)}")
    for s in sections[:15]:
        heading = s["heading"][:80]
        print(f"    [{s['block_type']}] {heading}")
    if len(sections) > 15:
        print(f"    ... and {len(sections) - 15} more")

    print(f"\n  Tables (pipe-detected): {len(tables)}")
    for i, t in enumerate(tables):
        headers_str = ", ".join(t["headers"][:4])
        if len(t["headers"]) > 4:
            headers_str += "..."
        print(f"    Table {i+1} (page {t['page_number']}): {len(t['headers'])} cols, "
              f"{t['row_count']} rows — [{headers_str}]")

    with open(os.path.join(job_dir, "normalized_text.txt")) as f:
        text = f.read()
    print(f"\n  Normalized text: {len(text):,} chars")

    # Extract heading texts for alignment comparison
    heading_texts = [s["heading"] for s in sections]

    print()

    return {
        "sections": sections,
        "tables": tables,
        "total_pages": tree["total_pages"],
        "text_length": len(text),
        "heading_texts": heading_texts,
        "raw_spans": meta["total_raw_spans"],
        "merged_spans": meta["total_merged_spans"],
    }


# ═══════════════════════════════════════════════════════════════════════════
# COMPARISON REPORT
# ═══════════════════════════════════════════════════════════════════════════

def compare(marker_data: dict, granite_data: dict):
    """Generate side-by-side comparison report."""
    print("=" * 70)
    print("COMPARISON REPORT: Marker (regex) vs Granite-Docling (DocTags)")
    print("=" * 70)
    print()

    # Section comparison
    m_sections = len(marker_data["sections"])
    g_sections = len(granite_data["sections"])
    print("SECTIONS")
    print(f"  Marker (regex):      {m_sections}")
    print(f"  Granite (DocTags):   {g_sections}")
    diff = g_sections - m_sections
    print(f"  Delta:               {'+' if diff >= 0 else ''}{diff}")
    print()

    # Table comparison
    m_tables = len(marker_data["tables"])
    g_tables = len(granite_data["tables"])
    print("TABLES")
    print(f"  Marker (pipe):       {m_tables}")
    print(f"  Granite (OTSL):      {g_tables}")
    diff = g_tables - m_tables
    print(f"  Delta:               {'+' if diff >= 0 else ''}{diff}")
    print()

    # Enrichment elements (Granite-only)
    g_elements = len(granite_data.get("elements", []))
    print("ENRICHMENT ELEMENTS (Granite-only)")
    if granite_data.get("elements"):
        by_type = {}
        for e in granite_data["elements"]:
            by_type.setdefault(e["type"], 0)
            by_type[e["type"]] += 1
        for dtype, count in sorted(by_type.items()):
            print(f"  {dtype}: {count}")
    else:
        print("  (none)")
    print(f"  Total: {g_elements} (Marker has 0 — no enrichment extraction)")
    print()

    # Heading alignment simulation
    print("HEADING ALIGNMENT SIMULATION")
    print("  (Simulating what Channel A V4.1 would do)")
    print()

    from difflib import SequenceMatcher

    marker_headings = marker_data["heading_texts"]
    granite_headings = [s["text"] for s in granite_data["sections"]]

    aligned = 0
    unaligned_granite = []

    for g_heading in granite_headings:
        best_ratio = 0
        best_match = None
        for m_heading in marker_headings:
            # Clean markdown artifacts for fair comparison
            m_clean = m_heading.replace("**", "").strip()
            g_clean = g_heading.strip()
            ratio = SequenceMatcher(None, g_clean.lower(), m_clean.lower()).ratio()
            if ratio > best_ratio:
                best_ratio = ratio
                best_match = m_heading
        if best_ratio >= 0.85:
            aligned += 1
            print(f"  ALIGNED ({best_ratio:.0%}): '{g_heading[:60]}' <-> '{best_match[:60]}'")
        else:
            unaligned_granite.append((g_heading, best_match, best_ratio))

    if unaligned_granite:
        print()
        print("  UNALIGNED Granite headings:")
        for g_heading, closest, ratio in unaligned_granite:
            print(f"    '{g_heading[:70]}' (best: {ratio:.0%} -> '{closest[:50] if closest else 'none'}')")

    total = len(granite_headings) if granite_headings else 1
    alignment_conf = aligned / total
    print()
    print(f"  Alignment confidence: {alignment_conf:.0%} ({aligned}/{total})")
    threshold = 0.80
    if alignment_conf >= threshold:
        print(f"  PASS (>= {threshold:.0%}) — Channel A would use Granite DocTags as oracle")
    else:
        print(f"  BELOW THRESHOLD (< {threshold:.0%}) — Channel A would fall back to regex")
    print()

    # V4.1 hybrid value assessment
    print("V4.1 HYBRID ARCHITECTURE VALUE ASSESSMENT")
    print()
    print("  What Granite-Docling adds that Marker regex misses:")

    gains = []
    if g_sections > m_sections:
        gains.append(f"  + {g_sections - m_sections} additional sections detected")
    if g_tables > m_tables:
        gains.append(f"  + {g_tables - m_tables} additional tables (OTSL) detected")
    if g_elements > 0:
        gains.append(f"  + {g_elements} enrichment elements (footnotes, captions, lists)")
    gains.append(f"  + Semantic heading types (section_header vs title vs subtitle)")
    gains.append(f"  + Heading hierarchy levels (L1, L2, L3) vs flat regex")

    for g in gains:
        print(g)

    print()
    print("  What Marker provides that Granite-Docling cannot:")
    print(f"  + Clean markdown text ({marker_data['text_length']:,} chars) — text-of-record for B/C/E/F")
    print(f"  + Character-level offsets for every span")
    print(f"  + Pipe table text for precise cell offset tracking")
    print()

    # Save report
    report = {
        "comparison_date": time.strftime("%Y-%m-%dT%H:%M:%S"),
        "pdf": os.path.basename(args.pdf),
        "marker": {
            "sections": m_sections,
            "tables": m_tables,
            "text_chars": marker_data["text_length"],
            "raw_spans": marker_data["raw_spans"],
            "merged_spans": marker_data["merged_spans"],
        },
        "granite_docling": {
            "sections": g_sections,
            "tables": g_tables,
            "enrichment_elements": g_elements,
            "elapsed_s": granite_data.get("elapsed_s", 0),
        },
        "alignment": {
            "aligned_headings": aligned,
            "total_granite_headings": total,
            "confidence": round(alignment_conf, 3),
            "passes_threshold": alignment_conf >= threshold,
        },
    }

    report_path = os.path.join(_script_dir, "output", "v4_1_dual_comparison.json")
    os.makedirs(os.path.dirname(report_path), exist_ok=True)
    with open(report_path, "w") as f:
        json.dump(report, f, indent=2)
    print(f"  Report saved: {report_path}")
    print()


# ═══════════════════════════════════════════════════════════════════════════
# MAIN
# ═══════════════════════════════════════════════════════════════════════════

def main():
    pdf_path = args.pdf
    marker_job = args.marker_job

    if not os.path.exists(pdf_path):
        print(f"ERROR: PDF not found: {pdf_path}")
        sys.exit(1)

    if not os.path.isdir(marker_job):
        print(f"ERROR: Marker job not found: {marker_job}")
        print("Run Pipeline 1 with --l1 marker first, or specify --marker-job")
        sys.exit(1)

    print()
    print("V4.1 DUAL EXTRACTION COMPARISON")
    print(f"  PDF: {os.path.basename(pdf_path)}")
    print(f"  Marker job: {os.path.basename(marker_job)}")
    print()

    # Step 1: Analyze existing Marker job
    marker_data = analyze_marker_job(marker_job)

    # Step 2: Run Granite-Docling
    granite_data = run_granite_docling(pdf_path)

    if "error" in granite_data:
        print(f"Cannot compare — Granite-Docling failed: {granite_data['error']}")
        sys.exit(1)

    # Step 3: Compare
    compare(marker_data, granite_data)


if __name__ == "__main__":
    main()
