#!/usr/bin/env python3
"""
CoverageGuard E2E — Full run with B2 Adversarial Audit + D1 Dual-LLM enabled.

Loads job artifacts from existing pipeline output, loads API key from .env,
and runs CoverageGuard with all 4 domains including LLM layers.

Usage:
    python run_coverageguard_e2e.py
"""

import json
import os
import sys
import time
from dataclasses import field
from datetime import datetime

# ── Path setup ──────────────────────────────────────────────────────────────
_script_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _script_dir)
sys.path.insert(0, os.path.join(_script_dir, '..', '..'))

# ── Load .env BEFORE any imports that check env vars ────────────────────────
from dotenv import load_dotenv
env_path = os.path.join(_script_dir, ".env")
load_dotenv(env_path)
print(f"Loaded .env from: {env_path}")
api_key = os.environ.get("ANTHROPIC_API_KEY", "")
print(f"ANTHROPIC_API_KEY: {'SET (' + str(len(api_key)) + ' chars)' if api_key else 'NOT SET'}")

# ── Configuration ───────────────────────────────────────────────────────────
JOB_DIR = os.path.join(
    _script_dir, "data", "output", "v4",
    "job_marker_dfdb5212-9587-402b-b4df-8ab3fce831a5",
)
PDF_PATH = os.path.join(
    _script_dir, "data", "pdfs",
    "KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf",
)


# ── Deserializers for dataclass models ──────────────────────────────────────
def _deserialize_section(d: dict):
    """Recursively deserialize GuidelineSection from JSON dict."""
    from extraction.v4.models import GuidelineSection
    children = [_deserialize_section(c) for c in d.get("children", [])]
    return GuidelineSection(
        section_id=d["section_id"],
        heading=d["heading"],
        start_offset=d["start_offset"],
        end_offset=d["end_offset"],
        page_number=d["page_number"],
        block_type=d["block_type"],
        level=d.get("level", 1),
        children=children,
    )


def _deserialize_table(d: dict):
    """Deserialize TableBoundary from JSON dict."""
    from extraction.v4.models import TableBoundary
    return TableBoundary(
        table_id=d["table_id"],
        section_id=d["section_id"],
        start_offset=d["start_offset"],
        end_offset=d["end_offset"],
        headers=d.get("headers", []),
        row_count=d.get("row_count", 0),
        page_number=d.get("page_number", 0),
        source=d.get("source", "marker_pipe"),
        otsl_text=d.get("otsl_text"),
    )


def _deserialize_tree(d: dict):
    """Deserialize GuidelineTree from JSON dict."""
    from extraction.v4.models import GuidelineTree
    sections = [_deserialize_section(s) for s in d.get("sections", [])]
    tables = [_deserialize_table(t) for t in d.get("tables", [])]
    return GuidelineTree(
        sections=sections,
        tables=tables,
        total_pages=d.get("total_pages", 0),
        alignment_confidence=d.get("alignment_confidence", 1.0),
        structural_source=d.get("structural_source", "granite_doctags"),
    )


def _build_oracle_stub(metadata: dict):
    """Build CompletenessReport stub from job_metadata.json l1_oracle fields."""
    from l1_completeness_oracle import CompletenessReport
    oracle = metadata.get("l1_oracle", {})
    return CompletenessReport(
        total_rawdict_blocks=oracle.get("total_rawdict_blocks", 0),
        matched_blocks=oracle.get("matched_blocks", 0),
        char_coverage_pct=oracle.get("char_coverage_pct", 0.0),
        block_coverage_pct=oracle.get("block_coverage_pct", 0.0),
        high_priority_misses=oracle.get("high_priority_misses", 0),
        low_priority_misses=oracle.get("low_priority_misses", 0),
        image_text_gaps=oracle.get("image_text_gaps", 0),
        elapsed_ms=oracle.get("elapsed_ms", 0.0),
    )


# ── Main ────────────────────────────────────────────────────────────────────
def main():
    t_start = time.perf_counter()

    print()
    print("=" * 72)
    print("  CoverageGuard E2E — Full Run (B2 + D1 LLM Layers ENABLED)")
    print("=" * 72)
    print()

    # 1. Load artifacts
    print("Loading job artifacts...")

    with open(os.path.join(JOB_DIR, "normalized_text.txt"), "r") as f:
        normalized_text = f.read()
    print(f"  normalized_text: {len(normalized_text):,} chars")

    with open(os.path.join(JOB_DIR, "merged_spans.json"), "r") as f:
        raw_spans = json.load(f)
    from extraction.v4.models import MergedSpan
    merged_spans = [MergedSpan.model_validate(s) for s in raw_spans]
    print(f"  merged_spans: {len(merged_spans):,}")

    with open(os.path.join(JOB_DIR, "guideline_tree.json"), "r") as f:
        tree_data = json.load(f)
    tree = _deserialize_tree(tree_data)
    print(f"  guideline_tree: {len(tree.sections)} sections, {len(tree.tables)} tables")

    with open(os.path.join(JOB_DIR, "job_metadata.json"), "r") as f:
        metadata = json.load(f)
    oracle_report = _build_oracle_stub(metadata)
    job_id = metadata.get("job_id", "unknown")
    print(f"  oracle_report: gate_passed={oracle_report.gate_passed}, "
          f"high_misses={oracle_report.high_priority_misses}")
    print(f"  job_id: {job_id}")
    print(f"  PDF: {os.path.basename(PDF_PATH)}")
    print()

    # 2. Initialize CoverageGuard
    enable_b2 = os.environ.get("COVERAGEGUARD_B2", "1") == "1"
    mode_label = "B2+D1 ENABLED" if enable_b2 else "DETERMINISTIC ONLY"
    print(f"Initializing CoverageGuard ({mode_label})...")
    from coverage_guard import CoverageGuard
    guard = CoverageGuard(enable_b2=enable_b2)
    print(f"  B2 enabled: {guard._enable_b2}")
    print(f"  Anthropic client: {'YES' if guard._client else 'NO'}")
    print()

    # 3. Run validation
    print("=" * 72)
    print("  Running CoverageGuard.validate() — all 4 domains")
    print("=" * 72)
    print()

    report = guard.validate(
        merged_spans=merged_spans,
        tree=tree,
        normalized_text=normalized_text,
        pdf_path=PDF_PATH,
        oracle_report=oracle_report,
        job_id=job_id,
        guideline_document="KDIGO 2022 Diabetes in CKD",
    )

    elapsed = time.perf_counter() - t_start

    # 4. Display results
    print()
    print("=" * 72)
    print("  COVERAGEGUARD E2E RESULTS")
    print("=" * 72)
    print()

    verdict_icon = "PASS" if report.gate_verdict == "PASS" else "BLOCK"
    print(f"  Gate Verdict:     {verdict_icon}")
    print(f"  Total BLOCKs:     {report.total_block_count}")
    print(f"  Total Warnings:   {report.total_warning_count}")
    print(f"  Elapsed:          {elapsed:.1f}s")
    print()

    # Domain A
    print("  DOMAIN A — Structural Completeness")
    print(f"    Inventory expected: {report.inventory_expected}")
    print(f"    Inventory actual:   {report.inventory_actual}")
    missing_elems = [e for e in report.inventory_elements if e.coverage_status == "MISSING"]
    covered_elems = [e for e in report.inventory_elements if e.coverage_status == "COVERED"]
    print(f"    Elements: {len(covered_elems)} COVERED, {len(missing_elems)} MISSING")
    if missing_elems:
        for e in missing_elems[:10]:
            print(f"      MISSING: [{e.element_type}] {e.element_id} (pg {e.page_number})")
        if len(missing_elems) > 10:
            print(f"      ... and {len(missing_elems) - 10} more")
    if report.footnote_bindings:
        bound = sum(1 for fb in report.footnote_bindings if fb.bound_to_span)
        unbound = len(report.footnote_bindings) - bound
        print(f"    Footnotes: {len(report.footnote_bindings)} detected, {bound} bound, {unbound} unbound")
    if report.density_warnings:
        print(f"    Density warnings: {len(report.density_warnings)}")
    print()

    # Domain B
    print("  DOMAIN B — Content Exhaustiveness")
    print(f"    Tier 1 residual count: {report.tier1_residual_count}")
    if report.residual_fragments:
        tier1 = [r for r in report.residual_fragments if r.tier == "TIER_1"]
        tier2 = [r for r in report.residual_fragments if r.tier == "TIER_2"]
        noise = [r for r in report.residual_fragments if r.tier == "NOISE"]
        print(f"    Residual fragments: {len(tier1)} TIER_1, {len(tier2)} TIER_2, {len(noise)} NOISE")
        if tier1:
            print(f"    Tier 1 residuals:")
            for r in tier1[:5]:
                snippet = r.text[:80].replace('\n', ' ')
                print(f"      pg{r.page_number}: [{r.trigger_category}] {snippet}...")
            if len(tier1) > 5:
                print(f"      ... and {len(tier1) - 5} more")
    if report.adversarial_audit_delta is not None and report.adversarial_audit_delta > 0:
        print(f"    B2 Adversarial delta: {report.adversarial_audit_delta} assertions not in spans")
    elif report.adversarial_audit_delta is not None:
        print(f"    B2 Adversarial delta: {report.adversarial_audit_delta} (all assertions covered)")
    else:
        print(f"    B2 Adversarial audit: SKIPPED (no API key or disabled)")
    if report.branch_comparisons:
        branch_blocks = [b for b in report.branch_comparisons if b.action == "BLOCK"]
        print(f"    B3 Branch checks: {len(report.branch_comparisons)} ({len(branch_blocks)} BLOCK)")
    if report.population_action_warnings:
        print(f"    Population-action warnings: {len(report.population_action_warnings)}")
    print()

    # Domain C
    print("  DOMAIN C — Integrity Verification")
    if report.numeric_mismatches:
        block_nm = [m for m in report.numeric_mismatches if m.action == "BLOCK"]
        accept_nm = [m for m in report.numeric_mismatches if m.action == "ACCEPT"]
        print(f"    Numeric mismatches: {len(report.numeric_mismatches)} ({len(block_nm)} BLOCK, {len(accept_nm)} ACCEPT)")
        if block_nm:
            for m in block_nm[:5]:
                print(f"      BLOCK: span={m.span_id[:12]}.. src='{m.source_value}' ext='{m.extracted_value}' [{m.mismatch_type}]")
    else:
        print(f"    Numeric mismatches: 0")
    if report.l1_recovery_escalations:
        print(f"    L1 Recovery escalations: {len(report.l1_recovery_escalations)}")
    if report.corroboration_details:
        low_corr = [c for c in report.corroboration_details if c.action == "BLOCK"]
        print(f"    Corroboration checks: {len(report.corroboration_details)} ({len(low_corr)} BLOCK)")
    print()

    # Domain D
    print("  DOMAIN D — Systemic Meta-Validation")
    if report.dual_llm_agreement_pct is not None:
        print(f"    D1 Dual-LLM agreement: {report.dual_llm_agreement_pct:.1f}%")
    else:
        print(f"    D1 Dual-LLM: SKIPPED")
    if report.validator_health:
        print(f"    Validator health metrics:")
        for k, v in report.validator_health.items():
            print(f"      {k}: {v}")
    print()

    # Gate Blockers
    if report.gate_blockers:
        print("  GATE BLOCKERS (fix in this order):")
        for blocker in report.gate_blockers:
            print(f"    [{blocker.fix_priority}] {blocker.gate_name}: {blocker.blocker_count} issues")
            for detail in blocker.details[:3]:
                print(f"        -> {detail}")
            if len(blocker.details) > 3:
                print(f"        ... and {len(blocker.details) - 3} more")
        print()

    # 5. Save report
    report_path = os.path.join(JOB_DIR, "coverage_guard_report.json")
    report_dict = report.model_dump(mode="json")
    with open(report_path, "w") as f:
        json.dump(report_dict, f, indent=2, default=str)
    print(f"  Report saved: {report_path}")
    print(f"  Report size: {os.path.getsize(report_path):,} bytes")
    print()
    print(f"  Total elapsed: {elapsed:.1f}s")
    print()


if __name__ == "__main__":
    main()
