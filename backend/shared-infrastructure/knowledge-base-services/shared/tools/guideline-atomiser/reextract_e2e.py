#!/usr/bin/env python3
"""
Re-extraction E2E — Re-run Channels C+D with upstream fixes, re-merge, re-validate.

Loads existing job artifacts (normalized_text, tree, B/E/F raw spans),
re-runs Channels C and D (which have Group A-D upstream fixes),
re-merges through Signal Merger, and runs CoverageGuard.

Faster than full pipeline — skips L1 parsing, Channel 0, Channel A.
"""

import json
import os
import sys
import time
from uuid import UUID, uuid4

# ── Path setup
_script_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _script_dir)
sys.path.insert(0, os.path.join(_script_dir, '..', '..'))

# ── Load .env
from dotenv import load_dotenv
load_dotenv(os.path.join(_script_dir, ".env"))

# ── Configuration
JOB_DIR = os.path.join(
    _script_dir, "data", "output", "v4",
    "job_marker_dfdb5212-9587-402b-b4df-8ab3fce831a5",
)
PDF_PATH = os.path.join(
    _script_dir, "data", "pdfs",
    "KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf",
)


def _deserialize_section(d):
    from extraction.v4.models import GuidelineSection
    children = [_deserialize_section(c) for c in d.get("children", [])]
    return GuidelineSection(
        section_id=d["section_id"], heading=d["heading"],
        start_offset=d["start_offset"], end_offset=d["end_offset"],
        page_number=d["page_number"], block_type=d["block_type"],
        level=d.get("level", 1), children=children,
    )


def _deserialize_table(d):
    from extraction.v4.models import TableBoundary
    return TableBoundary(
        table_id=d["table_id"], section_id=d["section_id"],
        start_offset=d["start_offset"], end_offset=d["end_offset"],
        headers=d.get("headers", []), row_count=d.get("row_count", 0),
        page_number=d.get("page_number", 0), source=d.get("source", "marker_pipe"),
        otsl_text=d.get("otsl_text"),
    )


def _deserialize_tree(d):
    from extraction.v4.models import GuidelineTree
    raw_pm = d.get("page_map", {})
    page_map = {int(k): v for k, v in raw_pm.items()} if raw_pm else {}
    return GuidelineTree(
        sections=[_deserialize_section(s) for s in d.get("sections", [])],
        tables=[_deserialize_table(t) for t in d.get("tables", [])],
        total_pages=d.get("total_pages", 0),
        alignment_confidence=d.get("alignment_confidence", 1.0),
        structural_source=d.get("structural_source", "granite_doctags"),
        page_map=page_map,
    )


def _build_oracle_stub(metadata):
    from l1_completeness_oracle import CompletenessReport
    o = metadata.get("l1_oracle", {})
    return CompletenessReport(
        total_rawdict_blocks=o.get("total_rawdict_blocks", 0),
        matched_blocks=o.get("matched_blocks", 0),
        char_coverage_pct=o.get("char_coverage_pct", 0.0),
        block_coverage_pct=o.get("block_coverage_pct", 0.0),
        high_priority_misses=o.get("high_priority_misses", 0),
        low_priority_misses=o.get("low_priority_misses", 0),
        image_text_gaps=o.get("image_text_gaps", 0),
        elapsed_ms=o.get("elapsed_ms", 0.0),
    )


def _raw_spans_to_channel_output(channel, span_dicts):
    from extraction.v4.models import ChannelOutput, RawSpan
    spans = []
    for sd in span_dicts:
        try:
            spans.append(RawSpan(
                channel=sd.get("channel", channel),
                text=sd.get("text", ""),
                start=sd.get("start", 0),
                end=sd.get("end", 0),
                confidence=sd.get("confidence", 0.5),
                page_number=sd.get("page_number"),
                section_id=sd.get("section_id"),
                table_id=sd.get("table_id"),
                source_block_type=sd.get("source_block_type"),
                channel_metadata=sd.get("channel_metadata", {}),
            ))
        except Exception:
            # Skip spans with invalid field values (e.g. unknown source_block_type)
            pass
    return ChannelOutput(channel=channel, spans=spans)


def main():
    t_start = time.perf_counter()

    print()
    print("=" * 72)
    print("  RE-EXTRACTION E2E — Ch C+D Re-run + Re-merge + CoverageGuard")
    print("=" * 72)
    print()

    # ── 1. Load existing artifacts ──
    print("1. Loading existing job artifacts...")

    with open(os.path.join(JOB_DIR, "normalized_text.txt")) as f:
        normalized_text = f.read()
    print(f"   normalized_text: {len(normalized_text):,} chars")

    with open(os.path.join(JOB_DIR, "guideline_tree.json")) as f:
        tree = _deserialize_tree(json.load(f))
    print(f"   guideline_tree: {len(tree.sections)} sections, {len(tree.tables)} tables")

    with open(os.path.join(JOB_DIR, "raw_spans.json")) as f:
        raw_spans_data = json.load(f)

    with open(os.path.join(JOB_DIR, "job_metadata.json")) as f:
        metadata = json.load(f)
    oracle_report = _build_oracle_stub(metadata)
    job_id = uuid4()
    print(f"   job_id (new): {job_id}")
    print()

    # ── 2. Existing B, E, F channels (unchanged) ──
    print("2. Loading existing channel outputs (B, E, F — unchanged)...")
    b_output = _raw_spans_to_channel_output("B", raw_spans_data["B"]["spans"])
    e_output = _raw_spans_to_channel_output("E", raw_spans_data["E"]["spans"])
    f_output = _raw_spans_to_channel_output("F", raw_spans_data["F"]["spans"])
    print(f"   B: {len(b_output.spans)}  E: {len(e_output.spans)}  F: {len(f_output.spans)}")
    print()

    # ── 3. RE-RUN Channel C (v4.3.0 — Groups A+B+D) ──
    print("3. RE-RUNNING Channel C (v4.3.0 — Groups A+B+D fixes)...")
    from extraction.v4.channel_c_grammar import ChannelC
    channel_c = ChannelC()
    c_output = channel_c.extract(normalized_text, tree)
    c_old = len(raw_spans_data["C"]["spans"])
    c_new = len(c_output.spans)
    c_meta = c_output.metadata or {}
    print(f"   Version: {channel_c.VERSION}")
    print(f"   OLD spans: {c_old}  →  NEW spans: {c_new}  (delta: {c_new - c_old:+d})")
    print(f"   Exception clauses: {c_meta.get('exception_clauses', 0)}")
    print(f"   Bound thresholds: {c_meta.get('bound_thresholds', 0)}")

    cats = {}
    for s in c_output.spans:
        cat = s.channel_metadata.get("pattern", "unknown")
        cats[cat] = cats.get(cat, 0) + 1
    for cat, count in sorted(cats.items(), key=lambda x: -x[1]):
        print(f"     {cat}: {count}")
    print()

    # ── 4. RE-RUN Channel D (v4.2.0 — Group C footnotes) ──
    print("4. RE-RUNNING Channel D (v4.2.0 — Group C footnote fix)...")
    from extraction.v4.channel_d_table import ChannelD
    channel_d = ChannelD()
    d_output = channel_d.extract(normalized_text, tree)
    d_old = len(raw_spans_data["D"]["spans"])
    d_new = len(d_output.spans)
    d_meta = d_output.metadata or {}
    print(f"   Version: {channel_d.VERSION}")
    print(f"   OLD spans: {d_old}  →  NEW spans: {d_new}  (delta: {d_new - d_old:+d})")
    print(f"   Caption footnotes: {d_meta.get('caption_footnotes', 0)}")
    print()

    # ── 5. SIGNAL MERGER ──
    print("5. SIGNAL MERGER — re-merging all channels...")
    from extraction.v4.signal_merger import SignalMerger

    channel_outputs = [b_output, c_output, d_output, e_output, f_output]
    total_raw = sum(len(co.spans) for co in channel_outputs)
    print(f"   Total raw spans: {total_raw}")

    merger = SignalMerger()
    merged_spans = merger.merge(job_id, channel_outputs, tree)

    # Re-inject L1_RECOVERY spans from old merged_spans
    from extraction.v4.models import MergedSpan
    with open(os.path.join(JOB_DIR, "merged_spans.json")) as f:
        old_merged_data = json.load(f)
    old_merged_count = len(old_merged_data)

    recovery_count = 0
    for s in old_merged_data:
        if "L1_RECOVERY" in s.get("contributing_channels", []):
            try:
                merged_spans.append(MergedSpan.model_validate(s))
                recovery_count += 1
            except Exception:
                pass

    multi_ch = sum(1 for s in merged_spans if len(s.contributing_channels) > 1)
    print(f"   OLD merged: {old_merged_count}  →  NEW merged: {len(merged_spans)}")
    print(f"   L1_RECOVERY re-injected: {recovery_count}")
    print(f"   Multi-channel corroborated: {multi_ch}")
    print()

    # ── 6. COVERAGEGUARD ──
    print("=" * 72)
    print("  COVERAGEGUARD — Re-validation with upstream fixes")
    print("=" * 72)
    print()

    from coverage_guard import CoverageGuard
    guard = CoverageGuard(enable_b2=False)

    report = guard.validate(
        merged_spans=merged_spans,
        tree=tree,
        normalized_text=normalized_text,
        pdf_path=PDF_PATH,
        oracle_report=oracle_report,
        job_id=str(job_id),
        guideline_document="KDIGO 2022 Diabetes in CKD",
    )

    elapsed = time.perf_counter() - t_start

    # ── 7. RESULTS ──
    print()
    print("=" * 72)
    print("  RESULTS — Before vs After Upstream Fixes")
    print("=" * 72)
    print()

    prev_path = os.path.join(JOB_DIR, "coverage_guard_report.json")
    if os.path.exists(prev_path):
        with open(prev_path) as f:
            prev = json.load(f)
        pb = prev.get("total_block_count", "?")
        pw = prev.get("total_warning_count", "?")
        print(f"  PREVIOUS:  {pb} BLOCKs, {pw} warnings")
        print(f"  THIS RUN:  {report.total_block_count} BLOCKs, {report.total_warning_count} warnings")
        if isinstance(pb, int):
            print(f"  DELTA:     {report.total_block_count - pb:+d} BLOCKs, "
                  f"{report.total_warning_count - int(pw):+d} warnings")
        print()

    verdict = "PASS" if report.gate_verdict == "PASS" else "BLOCK"
    print(f"  Gate Verdict:   {verdict}")
    print(f"  Total BLOCKs:   {report.total_block_count}")
    print(f"  Total Warnings: {report.total_warning_count}")
    print(f"  Elapsed:        {elapsed:.1f}s")
    print()

    # Domain A
    print("  DOMAIN A — Structural Completeness")
    missing = [e for e in report.inventory_elements if e.coverage_status == "MISSING"]
    covered = [e for e in report.inventory_elements if e.coverage_status == "COVERED"]
    print(f"    {len(covered)} COVERED, {len(missing)} MISSING")
    if report.footnote_bindings:
        bound = sum(1 for fb in report.footnote_bindings if fb.bound_to_span)
        print(f"    Footnotes: {len(report.footnote_bindings)} detected, {bound} bound")
    print()

    # Domain B
    print("  DOMAIN B — Content Exhaustiveness")
    print(f"    Tier 1 residuals: {report.tier1_residual_count}")
    if report.branch_comparisons:
        bb = [b for b in report.branch_comparisons if b.action == "BLOCK"]
        bp = [b for b in report.branch_comparisons if b.action != "BLOCK"]
        print(f"    B3 Branch: {len(report.branch_comparisons)} checks ({len(bb)} BLOCK, {len(bp)} PASS)")
        for b in bb[:8]:
            lost = ", ".join(b.exception_keywords_lost) if b.exception_keywords_lost else "none"
            print(f"      BLOCK sec={b.section_id}: thresh {b.source_threshold_count}→{b.extracted_threshold_count}, "
                  f"lost=[{lost}]")
    print()

    # Domain C
    print("  DOMAIN C — Integrity Verification")
    if report.numeric_mismatches:
        bnm = [m for m in report.numeric_mismatches if m.action == "BLOCK"]
        print(f"    Numeric mismatches: {len(report.numeric_mismatches)} ({len(bnm)} BLOCK)")
    else:
        print(f"    Numeric mismatches: 0")
    if report.corroboration_details:
        lc = [c for c in report.corroboration_details if c.action == "BLOCK"]
        print(f"    Corroboration: {len(report.corroboration_details)} ({len(lc)} BLOCK)")
    print()

    # Gate blockers
    if report.gate_blockers:
        print("  GATE BLOCKERS:")
        for blocker in report.gate_blockers:
            print(f"    [{blocker.fix_priority}] {blocker.gate_name}: {blocker.blocker_count}")
            for d in blocker.details[:5]:
                print(f"        -> {d}")
            if len(blocker.details) > 5:
                print(f"        ... +{len(blocker.details) - 5} more")
        print()

    # Save
    rpt_path = os.path.join(JOB_DIR, "coverage_guard_report_v2.json")
    with open(rpt_path, "w") as f:
        json.dump(report.model_dump(mode="json"), f, indent=2, default=str)
    print(f"  Report saved: {os.path.basename(rpt_path)}")
    print()


if __name__ == "__main__":
    main()
