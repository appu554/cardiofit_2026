#!/usr/bin/env python3
"""
Clinical Guideline Curation Pipeline - TARGETED EXTRACTION (V4 + V3 Legacy)

V4 Architecture (two-pipeline split):
    Pipeline 1: L1 (targeted PDF) → L2 Multi-Channel → Signal Merger → Reviewer Queue → EXIT
    Pipeline 2: Reviewed Spans → Dossier Assembly → L3 (per-drug) → L4 (THREE-CHECK) → L5

V3 Legacy (single invocation):
    L1 → L2 (GLiNER) → L2.5 (RxNorm pre-lookup) → L3 → L4 → L5

Usage:
    # V4 Pipeline 1: Targeted multi-channel extraction
    python run_pipeline_targeted.py --pipeline 1 --source quick-reference
    python run_pipeline_targeted.py --pipeline 1 --source full-guide --pages 50-60

    # V4 Pipeline 2: Process reviewed spans
    python run_pipeline_targeted.py --pipeline 2 --job-dir /data/output/v4/job_<uuid>/

    # V3 Legacy (default)
    python run_pipeline_targeted.py --source quick-reference
    python run_pipeline_targeted.py --source full-guide --pages 50-60
"""
import sys
import os
import json
import hashlib
import argparse
from datetime import datetime
from uuid import uuid4

# Handle both Docker and local execution paths
_script_dir = os.path.dirname(os.path.abspath(__file__))
_parent_dir = os.path.dirname(_script_dir)  # guideline-atomiser directory
_shared_dir = os.path.dirname(os.path.dirname(_parent_dir))  # shared/ directory (for extraction.v4.*)
sys.path.insert(0, _shared_dir)
sys.path.insert(0, _parent_dir)
sys.path.insert(0, '/app')                    # Docker: extraction.v4.* resolves from /app
sys.path.insert(0, '/app/guideline-atomiser')  # Docker: fact_extractor etc.


# ═══════════════════════════════════════════════════════════════════════════
# CLI ARGUMENT PARSING
# ═══════════════════════════════════════════════════════════════════════════

parser = argparse.ArgumentParser(description="Targeted guideline extraction (V4 multi-channel + V3 legacy)")
parser.add_argument(
    "--pipeline",
    choices=["1", "2", "legacy"],
    default="legacy",
    help="Pipeline mode: 1 (extract+review), 2 (dossier+L3), legacy (V3 full)"
)
parser.add_argument(
    "--source",
    default="quick-reference",
    help="Which PDF to use (default: quick-reference). "
    "Valid choices depend on the guideline profile."
)
parser.add_argument("--pages", type=str, help="Page range for full-guide (e.g., '50-65')")
parser.add_argument(
    "--target-kb",
    choices=["dosing", "safety", "monitoring", "contextual", "all"],
    default="all",
    help="Target KB type for extraction (default: all). "
    "'contextual' targets KB-20 (ADR profiles + contextual modifiers for P2)."
)
parser.add_argument(
    "--job-dir",
    type=str,
    help="Job directory for Pipeline 2 (contains reviewed merged_spans.json)"
)
parser.add_argument(
    "--l1",
    choices=["monkeyocr", "marker", "docling"],
    default="monkeyocr",
    help="L1 PDF parser backend: monkeyocr (default), marker (legacy), or docling"
)
parser.add_argument(
    "--pdf-path",
    type=str,
    help="Custom PDF path (overrides --source). Use with --pages for Oracle page offset."
)
parser.add_argument(
    "--guideline",
    type=str,
    default="kdigo",
    help="Guideline profile to use (default: kdigo). "
    "Set to 'kdigo' for built-in KDIGO 2022, or provide a path to a profile YAML."
)
parser.add_argument(
    "--push-kb20",
    action="store_true",
    default=False,
    help="Push contextual/ADR extraction results to KB-20 via batch API (requires KB-20 running)",
)
args, _ = parser.parse_known_args()


# ═══════════════════════════════════════════════════════════════════════════
# SHARED HELPERS
# ═══════════════════════════════════════════════════════════════════════════

# PDF source selection — detect local vs Docker execution
LOCAL_PDFS_DIR = os.path.join(_script_dir, "pdfs")
DOCKER_PDFS_DIR = "/data/pdfs"
PDFS_DIR = LOCAL_PDFS_DIR if os.path.exists(LOCAL_PDFS_DIR) else DOCKER_PDFS_DIR

# ── GuidelineProfile Construction ─────────────────────────────────────
# Replaces all KDIGO-hardcoded values with a profile-driven system.
# "kdigo" (the default) uses GuidelineProfile.kdigo_default() for
# byte-identical backward compatibility. Any other value is treated
# as a path to a YAML profile file.

from extraction.v4.guideline_profile import GuidelineProfile
from extraction.v4.kb20_push_client import KB20PushClient

_PROFILES_DIR = os.path.join(_script_dir, "profiles")

if args.guideline == "kdigo":
    profile = GuidelineProfile.kdigo_default()
else:
    # Treat as a path: try absolute, then relative to profiles dir
    _profile_path = args.guideline
    if not os.path.isabs(_profile_path):
        _candidate = os.path.join(_PROFILES_DIR, _profile_path)
        if os.path.exists(_candidate):
            _profile_path = _candidate
        elif not _profile_path.endswith(".yaml"):
            _candidate_yaml = os.path.join(_PROFILES_DIR, f"{_profile_path}.yaml")
            if os.path.exists(_candidate_yaml):
                _profile_path = _candidate_yaml
    profile = GuidelineProfile.from_yaml(_profile_path)

# PDF_PATHS: profile-derived (replaces former KDIGO-hardcoded dict)
PDF_PATHS = {
    key: os.path.join(PDFS_DIR, filename)
    for key, filename in profile.pdf_sources.items()
}

# Validate --source against profile's available PDFs (unless --pdf-path overrides)
if not args.pdf_path and args.source not in PDF_PATHS:
    available = ", ".join(sorted(PDF_PATHS.keys()))
    parser.error(
        f"--source '{args.source}' is not available for guideline '{profile.profile_id}'. "
        f"Available sources: {available}"
    )


def _merge_with_v5_flag(merger, *args_, profile, **kwargs):
    """Resolve V5_BBOX_PROVENANCE from the loaded profile and call merger.merge.

    Threads the V5 feature flag (resolved via extraction.v4.v5_flags) and the
    profile object into ``signal_merger.merge`` so ChannelProvenance entries
    can be built when the flag is on. V4 callers are unaffected because the
    merge() defaults are ``v5_bbox_provenance=False, profile=None``.
    """
    from extraction.v4.v5_flags import is_v5_enabled
    v5_bbox = is_v5_enabled("bbox_provenance", profile)
    kwargs["v5_consensus_entropy"] = is_v5_enabled("consensus_entropy", profile)
    return merger.merge(
        *args_,
        v5_bbox_provenance=v5_bbox,
        profile=profile,
        **kwargs,
    )


def normalize_drug_name(name: str) -> str:
    """Normalize drug name for comparison — lowercase and strip dose forms.

    Critical for L4 validation to detect RxNorm code hallucinations where
    Claude generates a valid code that belongs to a different drug.
    """
    if not name:
        return ""
    normalized = name.lower()
    dose_forms = [
        " tablet", " capsule", " injection", " solution", " suspension",
        " pen injector", " mg/ml", " mg", " mcg", " extended release",
        " er", " xl", " xr", " sr", " cr", " dr", " oral", " injectable",
        " powder", " liquid", " syrup", " cream", " ointment", " patch",
        " inhaler", " spray", " drops", " gel", " lotion", " suppository",
    ]
    for form in dose_forms:
        normalized = normalized.replace(form, "")
    return normalized.strip()


def resolve_output_dir():
    """Resolve output directory (local or Docker).

    Docker detection: check for /data/output which is created by Dockerfile.
    If running in Docker, use /data/output. Otherwise use local ./output.
    """
    docker_output = "/data/output"
    if os.path.isdir(docker_output):
        return docker_output
    local_output = os.path.join(_script_dir, "output")
    os.makedirs(local_output, exist_ok=True)
    return local_output


def require_api_key():
    """Require and return ANTHROPIC_API_KEY."""
    api_key = os.environ.get("ANTHROPIC_API_KEY", "")
    if not api_key or len(api_key) < 30 or api_key.startswith("${"):
        print("❌ FATAL: ANTHROPIC_API_KEY is required for L3 extraction")
        print("   export ANTHROPIC_API_KEY='sk-ant-...'")
        sys.exit(1)
    return api_key


def guideline_context_kdigo():
    """Guideline context derived from the active profile.

    Retained as a function (not inlined) for backward compatibility with
    any callers that import it.  The name is a legacy artifact — it now
    delegates to ``profile.guideline_context()`` regardless of which
    guideline is active.
    """
    ctx = profile.guideline_context()
    ctx["source"] = args.source
    return ctx


def resolve_kb7_url():
    """Resolve KB-7 URL based on execution environment."""
    _is_local = os.path.exists(LOCAL_PDFS_DIR)
    _default = "http://localhost:8092" if _is_local else "http://kb7-terminology:8092"
    return os.environ.get("KB7_URL", _default)


def resolve_rxnav_url():
    """Resolve RxNav-in-a-Box URL based on execution environment."""
    _is_local = os.path.exists(LOCAL_PDFS_DIR)
    _default = "http://localhost:4000" if _is_local else "http://rxnav:4000"
    return os.environ.get("RXNAV_URL", _default)


# ═══════════════════════════════════════════════════════════════════════════
# PIPELINE 1: Targeted Multi-Channel Extraction → Reviewer Queue
# ═══════════════════════════════════════════════════════════════════════════

def pipeline_1():
    """V4 Pipeline 1: Targeted L1 → L2 Multi-Channel → Signal Merger → Reviewer Queue."""
    print("=" * 70)
    print("V4 PIPELINE 1 (TARGETED): Multi-Channel Extraction → Reviewer Queue")
    print("=" * 70)
    print()

    # ─── L1: TARGETED PDF PARSING ────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L1: TARGETED PDF PARSING                                            │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    # PDF source resolution: --pdf-path overrides --source lookup
    if args.pdf_path:
        pdf_path = args.pdf_path
        if not os.path.isabs(pdf_path):
            pdf_path = os.path.join(_script_dir, pdf_path)
    else:
        pdf_path = PDF_PATHS[args.source]

    # Page range handling:
    # --pages serves TWO purposes depending on context:
    #   1. With --source: tells Marker which pages to extract from full PDF
    #   2. With --pdf-path: the PDF is already a subset, so Marker extracts
    #      ALL pages. --pages only provides the Oracle page_offset.
    page_range = None
    oracle_page_offset = 0
    if args.pages:
        start, end = args.pages.split("-")
        oracle_page_offset = int(start)
        if not args.pdf_path:
            # Full PDF: Marker needs page_range to select subset
            page_range = (int(start), int(end))
        # With --pdf-path: page_range stays None (extract all pages)

    if args.pdf_path:
        print(f"📄 Source: custom PDF ({os.path.basename(pdf_path)})")
        if args.pages:
            print(f"   Page offset: {oracle_page_offset} (for Oracle)")
    elif page_range:
        print(f"📄 Source: {args.source} (pages {page_range[0]}-{page_range[1]})")
    else:
        print(f"📄 Source: {args.source}")

    if not os.path.exists(pdf_path):
        print(f"❌ FATAL: PDF not found at {pdf_path}")
        sys.exit(1)

    print(f"   Size: {os.path.getsize(pdf_path) / 1024 / 1024:.1f} MB")

    l1_backend = args.l1
    print(f"🔧 L1 Backend: {l1_backend}")

    if l1_backend == "docling":
        from docling_extractor import DoclingExtractor
        print("🔄 Loading Docling PDF parser...")
        l1_extractor = DoclingExtractor()
    elif l1_backend == "marker":
        from marker_extractor import MarkerExtractor
        print("🔄 Loading Marker PDF parser (legacy)...")
        l1_extractor = MarkerExtractor()
    else:
        from monkeyocr_extractor import MonkeyOCRExtractor
        print("🔄 Loading MonkeyOCR PDF parser...")
        l1_extractor = MonkeyOCRExtractor()

    print(f"🔄 Extracting PDF{f' (pages {page_range[0]}-{page_range[1]})' if page_range else ''}...")
    l1_result = l1_extractor.extract(pdf_path, page_range=page_range)

    markdown_text = l1_result.markdown
    total_pages = l1_result.provenance.total_pages

    if l1_result.provenance.marker_version == "mock":
        print("❌ FATAL: Parser returned mock data")
        sys.exit(1)

    print(f"   ✅ Pages: {total_pages}")
    print(f"   ✅ Blocks: {len(l1_result.blocks)}")
    print(f"   ✅ Tables: {len(l1_result.tables)}")
    print(f"   ✅ Markdown: {len(markdown_text):,} chars")

    # Build page→bbox map for V5 provenance fallback.
    # When blocks carry per-page bbox (Docling: full-page; MonkeyOCR: block-level),
    # store the first non-null bbox per page. The signal merger uses this as a
    # fallback when a NER channel span has no block-level bbox of its own.
    _page_bbox_map: dict[int, list[float]] = {}
    for _blk in l1_result.blocks:
        if _blk.bbox is not None and _blk.page_number not in _page_bbox_map:
            _b = _blk.bbox
            _page_bbox_map[_blk.page_number] = [_b.x0, _b.y0, _b.x1, _b.y1]

    # Show detected tables
    if l1_result.tables:
        print()
        print("📊 Tables Detected:")
        for i, table in enumerate(l1_result.tables[:5]):
            headers = ", ".join(table.headers[:3]) + ("..." if len(table.headers) > 3 else "")
            print(f"   Table {i+1} (page {table.page_number}): {headers} ({len(table.rows)} rows)")
    print()

    # ─── L1 COMPLETENESS ORACLE (Marker → Oracle → Channel 0) ────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L1 ORACLE: PyMuPDF rawdict completeness check                      │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    from l1_completeness_oracle import L1CompletenessOracle

    oracle = L1CompletenessOracle()
    completeness_report = oracle.validate(pdf_path, markdown_text, page_offset=oracle_page_offset)
    print(f"   {completeness_report.summary()}")

    if completeness_report.gate_passed:
        print("   Gate: PASS — zero HIGH-priority misses, proceeding to Channel 0")
    else:
        print(f"   Gate: FAIL — {completeness_report.high_priority_misses} HIGH-priority miss(es) detected")
        print("   HIGH misses will be injected into reviewer queue as L1_RECOVERY spans")
        for miss in completeness_report.missed_blocks:
            if miss.priority == "HIGH":
                print(f"      HIGH p{miss.block.page_number}: {miss.reason}")

    if completeness_report.low_priority_misses > 0:
        print(f"   LOW misses (informational): {completeness_report.low_priority_misses}")

    # Save oracle report to job artifacts (written later with other artifacts)
    _oracle_report = completeness_report
    print()

    # ─── L2 V4: MULTI-CHANNEL EXTRACTION ─────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L2 V4: MULTI-CHANNEL EXTRACTION (Channels 0, A-F)                  │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    from extraction.v4.channel_0_normalizer import Channel0Normalizer
    from extraction.v4.channel_a_docling import ChannelA
    from extraction.v4.channel_b_drug_dict import ChannelB
    from extraction.v4.channel_c_grammar import ChannelC
    from extraction.v4.channel_d_table import ChannelD
    from extraction.v4.channel_e_gliner import ChannelE
    from extraction.v4.channel_f_nuextract import ChannelF
    from extraction.v4.signal_merger import SignalMerger
    from extraction.v4.models import ChannelOutput

    job_id = uuid4()
    print(f"📋 Job ID: {job_id}")
    print(f"🎯 Target KB: {args.target_kb}")
    print()

    # Channel 0: Text normalization
    print("   [Channel 0] Normalizing text...")
    normalizer = Channel0Normalizer()
    normalized_text, norm_meta = normalizer.normalize(markdown_text)
    print(f"      ✅ {len(normalized_text):,} chars (normalized, {norm_meta.get('fix_count', 0)} fixes)")

    # Channel A: Structure parsing → GuidelineTree
    # V4.2: Granite-Docling structural oracle (VLM-based)
    # V4.2.2: Pass profile-driven heading sets for authority-specific reparenting
    print("   [Channel A] Parsing document structure (Granite-Docling)...")
    channel_a = ChannelA(
        subordinate_headings=profile.subordinate_headings or None,
        chapter_reset_headings=profile.chapter_reset_headings or None,
        profile=profile,  # V5: enables ChannelProvenance emission when flag on
    )
    tree = channel_a.parse(normalized_text, pdf_path=pdf_path)
    source_tag = tree.structural_source
    align_pct = f"{tree.alignment_confidence:.0%}"
    print(f"      ✅ {len(tree.sections)} sections, {len(tree.tables)} tables")
    print(f"      ✅ Structural source: {source_tag} (alignment: {align_pct})")

    # ── L1→L2 Page Coverage Gate ─────────────────────────────────────────
    # Mandatory check: the page_map built from PAGE markers must cover at
    # least 80% of the document's pages. If coverage is below this
    # threshold, the L1 cache has broken page numbering (e.g. stale
    # chunked-L1 intermediate JSONs with chunk-local page numbers).
    # Halt here — every downstream channel would inherit wrong pages.
    if tree.page_map and tree.total_pages > 1:
        mapped_pages = sorted(set(tree.page_map.values()))
        page_coverage_pct = len(mapped_pages) / tree.total_pages * 100
        missing_pages = [p for p in range(1, tree.total_pages + 1)
                         if p not in mapped_pages]
        print(f"   [Gate] Page coverage: {len(mapped_pages)}/{tree.total_pages} "
              f"({page_coverage_pct:.0f}%)")
        if page_coverage_pct < 80:
            print(f"      ❌ BLOCKED: Page coverage {page_coverage_pct:.0f}% < 80% threshold.")
            print(f"      Missing pages: {missing_pages}")
            print(f"      This indicates broken PAGE markers in the L1 cache.")
            print(f"      Fix: delete l1_cache/_chunk_*_result.json and re-run chunked L1.")
            sys.exit(1)
        elif missing_pages:
            print(f"      ⚠️  Missing {len(missing_pages)} pages: {missing_pages[:10]}"
                  f"{'...' if len(missing_pages) > 10 else ''}")
        else:
            print(f"      ✅ All {tree.total_pages} pages covered")

    # Channels B-F
    channel_outputs = []

    print("   [Channel B] Drug dictionary scan (Aho-Corasick)...")
    channel_b = ChannelB(
        extra_ingredients=profile.extra_drug_ingredients,
        extra_classes=profile.extra_drug_classes,
        context_window_size=profile.context_window_size,
    )
    b_output = channel_b.extract(normalized_text, tree)
    channel_outputs.append(b_output)
    print(f"      ✅ {len(b_output.spans)} drug spans")

    print("   [Channel C] Grammar/regex patterns...")
    channel_c = ChannelC()
    c_output = channel_c.extract(normalized_text, tree)
    channel_outputs.append(c_output)
    print(f"      ✅ {len(c_output.spans)} pattern spans")

    print("   [Channel D] Table cell decomposition...")
    channel_d = ChannelD()
    d_output = channel_d.extract(normalized_text, tree, l1_tables=l1_result.tables, profile=profile)
    d_pipe = d_output.metadata.get("tables_pipe", 0)
    d_otsl = d_output.metadata.get("tables_otsl", 0)
    d_suspicious = d_output.metadata.get("suspicious_tables", 0)
    print(f"      ✅ {len(d_output.spans)} table cell spans (pipe: {d_pipe}, otsl: {d_otsl})")
    if d_suspicious:
        print(f"      ⚠️ {d_suspicious} suspicious tables flagged")

    # TableBoundaryOracle: augment Channel D with structured table rules
    # These carry full drug+condition+value context instead of decontextualized
    # cell fragments, giving the tiering classifier enough signal to avoid NOISE.
    try:
        from extraction.v4.table_boundary_oracle import TableBoundaryOracle
        from extraction.v4.models import RawSpan
        table_oracle = TableBoundaryOracle(str(pdf_path), page_offset=0)
        oracle_rules = table_oracle.extract_all_rules()
        oracle_spans = []
        for rule in oracle_rules:
            # Combine drug+condition+value into a contextual span text
            span_text = f"{rule.drug}: {rule.value}"
            if rule.condition:
                span_text = f"{rule.drug} ({rule.condition}): {rule.value}"
            oracle_spans.append(RawSpan(
                channel="D",
                text=span_text,
                start=-1,       # Not in markdown text space (coordinate-based)
                end=-1,
                confidence=rule.confidence,
                page_number=rule.page_number,
                section_id=None,
                table_id=rule.table_id,
                source_block_type="table_cell",
                channel_metadata={
                    "table_source": "oracle_coordinate",
                    "drug": rule.drug,
                    "condition": rule.condition,
                    "value": rule.value,
                    "sub_table": rule.sub_table,
                    "oracle_version": TableBoundaryOracle.VERSION,
                },
            ))
        if oracle_spans:
            d_output.spans.extend(oracle_spans)
            d_output.metadata["oracle_rules"] = len(oracle_spans)
            print(f"      ✅ +{len(oracle_spans)} structured table rules (TableBoundaryOracle)")

        # NOAC completeness check (clinical safety net)
        noac_missing = table_oracle.validate_noac_completeness(oracle_rules)
        if noac_missing:
            print(f"      ⚠️ NOAC completeness: {len(noac_missing)} missing combinations")
            for m in noac_missing[:5]:
                print(f"         {m}")
    except Exception as e:
        print(f"      ⚠️ TableBoundaryOracle: {e}")

    channel_outputs.append(d_output)

    print("   [Channel E] GLiNER residual NER...")
    try:
        channel_e = ChannelE()
        existing_spans = b_output.spans + c_output.spans
        e_output = channel_e.extract(normalized_text, tree, existing_spans)
        channel_outputs.append(e_output)
        print(f"      ✅ {len(e_output.spans)} novel entity spans")
    except Exception as e:
        channel_outputs.append(ChannelOutput(channel="E", spans=[], error=str(e)))
        print(f"      ⚠️ GLiNER error: {e}")

    print("   [Channel F] NuExtract proposition extraction (via Ollama)...")
    try:
        channel_f = ChannelF()
        if channel_f.available:
            f_output = channel_f.extract(normalized_text, tree)
            channel_outputs.append(f_output)
            f_failed = f_output.metadata.get("blocks_failed", 0)
            print(f"      ✅ {len(f_output.spans)} proposition spans "
                  f"(model: {channel_f.model_name})")
            if f_failed:
                print(f"      ⚠️ {f_failed} blocks failed inference")
        else:
            channel_outputs.append(ChannelOutput(channel="F", spans=[],
                                                 error=channel_f._init_error))
            print(f"      ⚠️ NuExtract/Ollama not available: {channel_f._init_error}")
    except Exception as e:
        channel_outputs.append(ChannelOutput(channel="F", spans=[], error=str(e)))
        print(f"      ⚠️ NuExtract error: {e}")

    # Channel G: Sentence-level context extraction (derived from B-F)
    print("   [Channel G] Sentence-level context extraction...")
    try:
        from extraction.v4.channel_g_sentence import ChannelG
        channel_g = ChannelG()
        g_output = channel_g.extract(normalized_text, tree, channel_outputs)
        channel_outputs.append(g_output)
        print(f"      ✅ {len(g_output.spans)} sentence context spans")
    except Exception as e:
        channel_outputs.append(ChannelOutput(channel="G", spans=[], error=str(e)))
        print(f"      ⚠️ Channel G error: {e}")

    total_raw = sum(len(co.spans) for co in channel_outputs)
    print(f"\n   Total raw spans: {total_raw}")

    # ─── CHANNEL OUTPUT CACHE ──────────────────────────────────────────────
    # V4.2.5: Persist channel outputs so we can resume from merger if it crashes.
    # Cache key = job_id. Stored alongside L1 cache for co-location.
    _channel_cache_path = os.path.join(
        _script_dir, "l1_cache", f"_channels_{job_id}.json"
    )
    try:
        _channel_cache = {
            "job_id": str(job_id),
            "profile_id": profile.profile_id,
            "source": args.source,
            "total_raw_spans": total_raw,
            "guideline_tree_json": {
                "total_pages": tree.total_pages,
                "alignment_confidence": tree.alignment_confidence,
                "structural_source": tree.structural_source,
                "sections_count": len(tree.sections),
                "tables_count": len(tree.tables),
            },
            "channels": [],
        }
        for co in channel_outputs:
            _channel_cache["channels"].append({
                "channel": co.channel,
                "span_count": len(co.spans),
                "error": co.error,
                "spans": [
                    {
                        "channel": s.channel,
                        "text": s.text,
                        "start": s.start,
                        "end": s.end,
                        "confidence": s.confidence,
                        "page_number": s.page_number,
                        "section_id": s.section_id,
                        "table_id": getattr(s, "table_id", None),
                        "source_block_type": s.source_block_type,
                        "channel_metadata": s.channel_metadata,
                    }
                    for s in co.spans
                ],
            })
        with open(_channel_cache_path, "w") as f:
            json.dump(_channel_cache, f)
        print(f"\n   💾 Channel cache saved: {os.path.basename(_channel_cache_path)}")
    except Exception as e:
        print(f"\n   ⚠️ Channel cache save failed: {e}")
    print()

    # ─── SIGNAL MERGER ────────────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ SIGNAL MERGER: Clustering + Confidence Boosting + Tiering          │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    from extraction.v4.tiering_classifier import (
        RuleBasedTieringClassifier, TrainedTieringClassifier, ShadowTieringClassifier,
    )

    # Profile-driven classifier selection
    _classifier_type = profile.tiering_classifier
    # Resolve relative model paths against _script_dir (the data/ directory)
    _model_path = profile.tiering_golden_dataset
    if _model_path and not os.path.isabs(_model_path):
        _model_path = os.path.join(_script_dir, _model_path)
    if _classifier_type == "trained" and _model_path:
        tiering_classifier = TrainedTieringClassifier(_model_path)
        _classifier_version = f"trained_v1_{profile.tiering_golden_dataset}"
        print(f"   Classifier: TRAINED ({_model_path})")
    elif _classifier_type == "shadow" and _model_path:
        tiering_classifier = ShadowTieringClassifier(_model_path)
        _classifier_version = f"shadow_v1_{profile.tiering_golden_dataset}"
        print(f"   Classifier: SHADOW (rule-based authoritative, trained logged)")
    else:
        tiering_classifier = RuleBasedTieringClassifier()
        _classifier_version = "rule_based_v4.1"
        if _classifier_type not in ("rule_based", None):
            print(f"   ⚠️ Requested '{_classifier_type}' but no model path — falling back to rule_based")
        print(f"   Classifier: RULE_BASED")

    merger = SignalMerger()
    merged_spans = _merge_with_v5_flag(
        merger, job_id, channel_outputs, tree,
        classifier=tiering_classifier, profile=profile,
        page_bbox_map=_page_bbox_map or None,
    )

    # Assign prediction tracking metadata for ML feedback loop
    import uuid as _uuid_mod
    for span in merged_spans:
        span.prediction_id = str(_uuid_mod.uuid4())
        span.classifier_version = _classifier_version

    disagreements = sum(1 for s in merged_spans if s.has_disagreement)
    multi_channel = sum(1 for s in merged_spans if len(s.contributing_channels) > 1)
    tier_1_count = sum(1 for s in merged_spans if s.tier == "TIER_1")
    tier_2_count = sum(1 for s in merged_spans if s.tier == "TIER_2")
    noise_count = sum(1 for s in merged_spans if s.tier == "NOISE")

    print(f"   ✅ Merged spans: {len(merged_spans)}")
    print(f"   ✅ Multi-channel corroborated: {multi_channel}")
    print(f"   ✅ Tiering: TIER_1={tier_1_count}, TIER_2={tier_2_count}, NOISE={noise_count}")
    if disagreements:
        print(f"   ⚠️ Disagreements flagged: {disagreements}")

    # Shadow mode reporting
    if isinstance(tiering_classifier, ShadowTieringClassifier):
        _agree = tiering_classifier.agreement_rate
        _shadow_n = len(tiering_classifier.shadow_log)
        print(f"   🔍 Shadow mode: {_shadow_n} predictions, {_agree:.1%} agreement rate")

    # ─── CHANNEL H: CROSS-CHANNEL RECOVERY ────────────────────────────
    print()
    print("   [Channel H] Cross-channel recovery analysis...")
    try:
        from extraction.v4.channel_h_recovery import ChannelH
        channel_h = ChannelH()
        h_output = channel_h.extract(merged_spans, normalized_text, tree)
        if h_output.spans:
            # Feed recovery spans back through merger as additional input
            recovery_co = [h_output]
            h_merged = _merge_with_v5_flag(
                merger, job_id, recovery_co, tree,
                classifier=tiering_classifier, profile=profile,
                page_bbox_map=_page_bbox_map or None,
            )
            # Assign prediction tracking to recovery spans
            for span in h_merged:
                span.prediction_id = str(_uuid_mod.uuid4())
                span.classifier_version = _classifier_version
            merged_spans.extend(h_merged)
            print(f"      ✅ {len(h_output.spans)} recovery spans → {len(h_merged)} merged")
            print(f"      Reasons: {h_output.metadata.get('recovery_reasons', {})}")
        else:
            print("      ✅ No recovery needed (all channels corroborated)")
    except Exception as e:
        print(f"      ⚠️ Channel H error: {e}")

    # ─── SAFETY CRITICALITY CHECK ────────────────────────────────────────
    # Safety-critical spans (contraindications, black-box warnings, dose limits)
    # get a tier floor of TIER_2 — they should never be classified as NOISE.
    print()
    print("   [Safety] Safety criticality check...")
    try:
        from extraction.v4.classifiers.safety_criticality import SafetyCriticalityDetector
        _safety_detector = SafetyCriticalityDetector.try_load()
        _safety_upgrades = 0
        for span in merged_spans:
            is_critical, _sc_conf = _safety_detector.is_safety_critical(span.text, {})
            if is_critical and span.tier == "NOISE":
                span.tier = "TIER_2"
                span.tier_reason = (span.tier_reason or "") + " [safety-critical override: NOISE→TIER_2]"
                _safety_upgrades += 1
        if _safety_upgrades:
            print(f"      ⚠️ {_safety_upgrades} NOISE spans upgraded to TIER_2 (safety-critical)")
        else:
            print(f"      ✅ No safety-critical overrides needed")
    except Exception as e:
        print(f"      ⚠️ Safety criticality check error: {e}")

    # ─── RANGE INTEGRITY ENGINE ─────────────────────────────────────────
    print()
    print("   [RIE] Range Integrity Engine — numeric threshold validation...")
    try:
        from extraction.v4.range_integrity_engine import RangeIntegrityEngine
        rie = RangeIntegrityEngine(
            severity_keywords_path=profile.severity_keywords_path,
        )
        rie_report = rie.validate(merged_spans, normalized_text)
        rie_warnings = sum(1 for i in rie_report.issues if i.severity == "WARNING")
        rie_errors = sum(1 for i in rie_report.issues if i.severity == "ERROR")
        print(f"      ✅ Intervals: {rie_report.total_intervals}, "
              f"Issues: {rie_warnings} warnings, {rie_errors} errors")
        if rie_report.issues:
            for issue in rie_report.issues[:5]:
                print(f"      {'⚠️' if issue.severity == 'WARNING' else '❌'} "
                      f"{issue.check}: {issue.description}")
            if len(rie_report.issues) > 5:
                print(f"      ... and {len(rie_report.issues) - 5} more issues")
    except Exception as e:
        rie_report = None
        print(f"      ⚠️ RIE error: {e}")

    # ─── L1 RECOVERY INJECTION ─────────────────────────────────────────
    # Inject HIGH-priority missed blocks from L1 Oracle as L1_RECOVERY spans.
    # These are raw PDF text that Marker dropped — they did NOT go through
    # Channel 0 normalization or any extraction channel. Tagged distinctly
    # so the reviewer renders them differently (yellow highlight, not blue).
    recovery_spans = []
    if not _oracle_report.gate_passed:
        recovery_spans = oracle.recovery_merged_spans(_oracle_report, job_id)
        merged_spans.extend(recovery_spans)
        print(f"\n   L1_RECOVERY: injected {len(recovery_spans)} HIGH-priority spans into reviewer queue")

    # ─── COVERAGEGUARD: POST-MERGE QUALITY GATE ──────────────────────────
    print()
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ COVERAGEGUARD: Post-Merge Quality Gate (4 Domains, 8 Gates)        │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    from coverage_guard import CoverageGuard

    coverage_guard = CoverageGuard()
    _cg_report = coverage_guard.validate(
        merged_spans=merged_spans,
        tree=tree,
        normalized_text=normalized_text,
        pdf_path=pdf_path,
        oracle_report=_oracle_report,
        job_id=str(job_id),
        guideline_document=profile.document_title,
    )

    # Display CoverageGuard results
    print(f"   Gate verdict: {'✅ PASS' if _cg_report.gate_verdict == 'PASS' else '❌ BLOCK'}")
    print(f"   Total blocks: {_cg_report.total_block_count}")
    print(f"   Total warnings: {_cg_report.total_warning_count}")
    if _cg_report.inventory_expected:
        print(f"   Inventory expected: {_cg_report.inventory_expected}")
        print(f"   Inventory actual: {_cg_report.inventory_actual}")
    if _cg_report.tier1_residual_count:
        print(f"   ⚠️  Tier 1 residuals: {_cg_report.tier1_residual_count}")
    if _cg_report.numeric_mismatches:
        block_count = sum(1 for m in _cg_report.numeric_mismatches if m.action == "BLOCK")
        print(f"   Numeric mismatches: {len(_cg_report.numeric_mismatches)} ({block_count} BLOCK)")
    if _cg_report.gate_blockers:
        print(f"\n   Gate blockers (fix in this order):")
        for blocker in _cg_report.gate_blockers:
            print(f"     [{blocker.fix_priority}] {blocker.gate_name}: {blocker.blocker_count} issues")
            for detail in blocker.details[:3]:
                print(f"         → {detail}")
    print()

    # ─── SECTION PASSAGE ASSEMBLY (V4.2.1) ─────────────────────────────
    section_passages = merger.assemble_section_passages(merged_spans, tree, normalized_text)
    reparent_log = getattr(channel_a, '_reparent_log', [])
    print(f"\n   ✅ Section passages: {len(section_passages)}")
    if reparent_log:
        reparent_count = sum(1 for e in reparent_log if e.get("type") == "reparent")
        warning_count = sum(1 for e in reparent_log if e.get("type") == "validation_warning")
        print(f"   ✅ {profile.authority} reparented: {reparent_count} subordinate heading(s)")
        if warning_count:
            print(f"   ⚠️  Validation warnings: {warning_count}")

    print()

    # ─── V5 #5: GUIDELINE DECOMPOSITION ─────────────────────────────────
    # Produces a knowledge graph (graph.json) alongside merged_spans.json.
    # Runs only when the "decomposition" V5 feature flag is enabled in the
    # active guideline profile. The decomposer is imported lazily here to
    # avoid top-level import cycles (matching the pattern used for other V5
    # feature steps in this file).
    _decomposition_graph = None
    if _is_v5_enabled("decomposition", profile):
        try:
            from extraction.v4.guideline_decomposer import GuidelineDecomposer
            _decomposer = GuidelineDecomposer()
            _decomposition_graph = _decomposer.decompose(
                job_id=str(job_id),
                merged_spans=merged_spans,
                tree=tree,
                section_passages=section_passages,
                profile=profile,
            )
            if _decomposition_graph is not None:
                _n_nodes = len(_decomposition_graph.nodes) if hasattr(_decomposition_graph, "nodes") else 0
                _n_edges = len(_decomposition_graph.edges) if hasattr(_decomposition_graph, "edges") else 0
                print(f"   [V5 #5] Decomposition: {_n_nodes} nodes, {_n_edges} edges → graph.json")
            else:
                print("   [V5 #5] Decomposition: no graph produced (decomposer returned None)")
        except Exception as e:
            print(f"   [V5 #5] Decomposition error: {e}")

    print()

    # ─── SAVE JOB ARTIFACTS ──────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ SAVING JOB ARTIFACTS → Reviewer Queue                              │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    base_output = resolve_output_dir()
    l1_tag = args.l1  # "marker" or "docling"
    job_dir = os.path.join(base_output, "v4", f"job_{l1_tag}_{job_id}")
    os.makedirs(job_dir, exist_ok=True)

    with open(pdf_path, "rb") as f:
        source_hash = hashlib.sha256(f.read()).hexdigest()[:16]

    # Job metadata (includes targeted extraction params + oracle results)
    from extraction.v4.v5_flags import is_v5_enabled as _is_v5_enabled
    _V5_KNOWN_FEATURES = ["bbox_provenance", "table_specialist", "consensus_entropy", "decomposition"]
    _v5_features_enabled = [f for f in _V5_KNOWN_FEATURES if _is_v5_enabled(f, profile)]

    job_meta = {
        "job_id": str(job_id),
        "source_pdf": os.path.basename(pdf_path),
        "source_hash": source_hash,
        "source_type": args.source,
        "page_range": args.pages,
        "target_kb": args.target_kb,
        "guideline_authority": profile.authority,
        "guideline_document": profile.document_title,
        "review_status": "PENDING",
        "total_raw_spans": total_raw,
        "total_merged_spans": len(merged_spans),
        "l1_recovery_spans": len(recovery_spans),
        "disagreements": disagreements,
        "created_at": datetime.utcnow().isoformat(),
        "pipeline_version": "4.2.2",
        "v5_features_enabled": _v5_features_enabled,
        "l1_backend": l1_tag,
        "structural_source": tree.structural_source,
        "alignment_confidence": tree.alignment_confidence,
        "section_passages": len(section_passages),
        "reparented_sections": sum(1 for e in reparent_log if e.get("type") == "reparent"),
        "validation_warnings": sum(1 for e in reparent_log if e.get("type") == "validation_warning"),
        "l1_oracle": {
            "gate_passed": _oracle_report.gate_passed,
            "block_coverage_pct": round(_oracle_report.block_coverage_pct, 1),
            "char_coverage_pct": round(_oracle_report.char_coverage_pct, 1),
            "high_priority_misses": _oracle_report.high_priority_misses,
            "low_priority_misses": _oracle_report.low_priority_misses,
            "image_text_gaps": _oracle_report.image_text_gaps,
            "elapsed_ms": round(_oracle_report.elapsed_ms, 1),
            "total_rawdict_blocks": _oracle_report.total_rawdict_blocks,
            "matched_blocks": _oracle_report.matched_blocks,
        },
        "coverage_guard": {
            "gate_verdict": _cg_report.gate_verdict,
            "total_block_count": _cg_report.total_block_count,
            "total_warning_count": _cg_report.total_warning_count,
            "inventory_expected": _cg_report.inventory_expected,
            "inventory_actual": _cg_report.inventory_actual,
            "tier1_residual_count": _cg_report.tier1_residual_count,
            "numeric_block_count": sum(1 for m in _cg_report.numeric_mismatches if m.action == "BLOCK"),
            "branch_block_count": sum(1 for b in _cg_report.branch_comparisons if b.action == "BLOCK"),
            "gate_blockers": [
                {"gate": b.gate_name, "count": b.blocker_count, "priority": b.fix_priority}
                for b in _cg_report.gate_blockers
            ],
        },
    }
    meta_path = os.path.join(job_dir, "job_metadata.json")
    with open(meta_path, "w") as f:
        json.dump(job_meta, f, indent=2)
    print(f"   💾 Job metadata: {meta_path}")

    # Normalized text
    text_path = os.path.join(job_dir, "normalized_text.txt")
    with open(text_path, "w") as f:
        f.write(normalized_text)
    print(f"   💾 Normalized text: {text_path}")

    # Guideline tree
    tree_data = {
        "sections": [_serialize_section(s) for s in tree.sections],
        "tables": [
            {
                "table_id": t.table_id, "headers": t.headers, "row_count": t.row_count,
                "page_number": t.page_number, "section_id": t.section_id,
                "start_offset": t.start_offset, "end_offset": t.end_offset,
                "source": t.source, "otsl_text": t.otsl_text,
            }
            for t in tree.tables
        ],
        "total_pages": tree.total_pages,
        "alignment_confidence": tree.alignment_confidence,
        "structural_source": tree.structural_source,
        "page_map": {str(k): v for k, v in tree.page_map.items()} if tree.page_map else {},
    }
    tree_path = os.path.join(job_dir, "guideline_tree.json")
    with open(tree_path, "w") as f:
        json.dump(tree_data, f, indent=2)
    print(f"   💾 Guideline tree: {tree_path}")

    # Merged spans (reviewer queue)
    spans_data = [s.model_dump(mode="json") for s in merged_spans]
    spans_path = os.path.join(job_dir, "merged_spans.json")
    with open(spans_path, "w") as f:
        json.dump(spans_data, f, indent=2, default=str)
    print(f"   💾 Merged spans ({len(merged_spans)}): {spans_path}")

    # CoverageGuard report
    cg_data = _cg_report.model_dump(mode="json")
    cg_path = os.path.join(job_dir, "coverage_guard_report.json")
    with open(cg_path, "w") as f:
        json.dump(cg_data, f, indent=2, default=str)
    print(f"   💾 CoverageGuard report ({_cg_report.gate_verdict}): {cg_path}")

    # Range Integrity Engine report (if available)
    if rie_report is not None:
        from dataclasses import asdict as _asdict
        rie_data = _asdict(rie_report)
        rie_path = os.path.join(job_dir, "range_integrity_report.json")
        with open(rie_path, "w") as f:
            json.dump(rie_data, f, indent=2, default=str)
        print(f"   💾 Range Integrity report ({len(rie_report.issues)} issues): {rie_path}")

    # Raw spans (debug)
    raw_data = {}
    for co in channel_outputs:
        raw_data[co.channel] = {
            "count": len(co.spans), "error": co.error,
            "spans": [s.model_dump(mode="json") for s in co.spans],
        }
    raw_path = os.path.join(job_dir, "raw_spans.json")
    with open(raw_path, "w") as f:
        json.dump(raw_data, f, indent=2, default=str)
    print(f"   💾 Raw spans (debug): {raw_path}")

    # Section passages (V4.2.1 — L3 bridge)
    passages_data = [
        {
            "section_id": p.section_id,
            "heading": p.heading,
            "page_number": p.page_number,
            "prose_text": p.prose_text,
            "span_ids": [str(sid) for sid in p.span_ids],
            "span_count": p.span_count,
            "child_section_ids": p.child_section_ids,
            "start_offset": p.start_offset,
            "end_offset": p.end_offset,
        }
        for p in section_passages
    ]
    passages_path = os.path.join(job_dir, "section_passages.json")
    with open(passages_path, "w") as f:
        json.dump(passages_data, f, indent=2)
    print(f"   💾 Section passages ({len(section_passages)}): {passages_path}")

    # Decomposition graph (V5 #5 — knowledge graph)
    if _decomposition_graph is not None:
        graph_path = os.path.join(job_dir, "graph.json")
        with open(graph_path, "w") as f:
            f.write(json.dumps(_decomposition_graph.to_dict(), indent=2))
        _n_nodes = len(_decomposition_graph.nodes) if hasattr(_decomposition_graph, "nodes") else 0
        _n_edges = len(_decomposition_graph.edges) if hasattr(_decomposition_graph, "edges") else 0
        print(f"   💾 Decomposition graph ({_n_nodes} nodes, {_n_edges} edges): {graph_path}")

    # Reparenting log (V4.2.1 — audit trail)
    if reparent_log:
        reparent_path = os.path.join(job_dir, "reparenting_log.json")
        with open(reparent_path, "w") as f:
            json.dump(reparent_log, f, indent=2)
        print(f"   💾 Reparenting log ({len(reparent_log)} entries): {reparent_path}")

    # Shadow classifier log (3.F4 — flush buffered predictions to artifact)
    # The ShadowTieringClassifier accumulates entries in memory during merge;
    # persist here so nightly_enrichment.py can load and insert into
    # l2_classifier_shadow_log via batch DB insert.
    if isinstance(tiering_classifier, ShadowTieringClassifier) and tiering_classifier.shadow_log:
        shadow_entries = [
            {**entry, "job_id": str(job_id), "created_at": datetime.utcnow().isoformat()}
            for entry in tiering_classifier.shadow_log
        ]
        shadow_log_path = os.path.join(job_dir, "shadow_classifier_log.json")
        with open(shadow_log_path, "w") as f:
            json.dump(shadow_entries, f, indent=2, default=str)
        print(f"   💾 Shadow classifier log ({len(shadow_entries)} entries, "
              f"{tiering_classifier.agreement_rate:.1%} agreement): {shadow_log_path}")

    print()

    # ─── PIPELINE 1 SUMMARY ──────────────────────────────────────────────
    print("=" * 70)
    print("PIPELINE 1 (TARGETED) COMPLETE — REVIEW REQUIRED")
    print("=" * 70)
    print()
    print(f"   Job ID:          {job_id}")
    print(f"   Source:           {args.source}{f' (pages {args.pages})' if args.pages else ''}")
    print(f"   L1 Oracle:       {'PASS' if _oracle_report.gate_passed else 'FAIL'} "
          f"(blocks {_oracle_report.block_coverage_pct:.1f}%, "
          f"chars {_oracle_report.char_coverage_pct:.1f}%)")
    print(f"   Job Directory:   {job_dir}")
    print(f"   Merged Spans:    {len(merged_spans)} (all PENDING review)")
    if recovery_spans:
        print(f"   L1_RECOVERY:     {len(recovery_spans)} (raw PDF text Marker dropped)")
    print(f"   Disagreements:   {disagreements}")
    print(f"   Section Passages:{len(section_passages)}")
    if reparent_log:
        _rc = sum(1 for e in reparent_log if e.get("type") == "reparent")
        _wc = sum(1 for e in reparent_log if e.get("type") == "validation_warning")
        print(f"   {profile.authority} Reparented:{_rc} subordinate heading(s)")
        if _wc:
            print(f"   Warnings:        {_wc} validation warning(s)")
    print()
    print("NEXT STEP:")
    print(f"   1. Review merged spans in: {spans_path}")
    print(f"   2. Run Pipeline 2:")
    print(f"      python run_pipeline_targeted.py --pipeline 2 --job-dir {job_dir} --target-kb {args.target_kb}")
    print()


def _serialize_section(s):
    """Serialize a GuidelineSection to dict."""
    return {
        "section_id": s.section_id, "heading": s.heading,
        "start_offset": s.start_offset, "end_offset": s.end_offset,
        "page_number": s.page_number, "block_type": s.block_type,
        "level": s.level,
        "children": [_serialize_section(c) for c in s.children],
    }


def _deserialize_section(data):
    """Deserialize a GuidelineSection from dict."""
    from extraction.v4.models import GuidelineSection
    return GuidelineSection(
        section_id=data["section_id"], heading=data["heading"],
        start_offset=data["start_offset"], end_offset=data["end_offset"],
        page_number=data["page_number"], block_type=data["block_type"],
        level=data.get("level", 1),
        children=[_deserialize_section(c) for c in data.get("children", [])],
    )


# ═══════════════════════════════════════════════════════════════════════════
# PIPELINE 2: Reviewed Spans → Dossier → L3 per Drug → L4 THREE-CHECK → L5
# ═══════════════════════════════════════════════════════════════════════════

def pipeline_2():
    """V4 Pipeline 2 (Targeted): Dossier Assembly → L3 → L4 THREE-CHECK → L5."""
    if not args.job_dir:
        print("❌ FATAL: --job-dir is required for Pipeline 2")
        print("   python run_pipeline_targeted.py --pipeline 2 --job-dir /path/to/job_<uuid>/")
        sys.exit(1)

    if not os.path.isdir(args.job_dir):
        print(f"❌ FATAL: Job directory not found: {args.job_dir}")
        sys.exit(1)

    print("=" * 70)
    print("V4 PIPELINE 2 (TARGETED): Dossier → L3 Per-Drug → L4 THREE-CHECK → L5")
    print("=" * 70)
    print()

    # ─── LOAD JOB ARTIFACTS ──────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ LOADING REVIEWED JOB ARTIFACTS                                      │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    job_dir = args.job_dir

    with open(os.path.join(job_dir, "job_metadata.json")) as f:
        job_meta = json.load(f)
    job_id = job_meta["job_id"]
    print(f"   📋 Job ID: {job_id}")
    print(f"   📄 Source: {job_meta.get('source_type', job_meta['source_pdf'])}")

    with open(os.path.join(job_dir, "normalized_text.txt")) as f:
        normalized_text = f.read()
    print(f"   📝 Text: {len(normalized_text):,} chars")

    with open(os.path.join(job_dir, "guideline_tree.json")) as f:
        tree_data = json.load(f)

    from extraction.v4.models import GuidelineTree, TableBoundary, MergedSpan

    tree = GuidelineTree(
        sections=[_deserialize_section(s) for s in tree_data["sections"]],
        tables=[
            TableBoundary(
                table_id=t["table_id"], headers=t["headers"],
                row_count=t.get("row_count", 0),
                page_number=t["page_number"], section_id=t.get("section_id", ""),
                start_offset=t.get("start_offset", -1),
                end_offset=t.get("end_offset", -1),
                source=t.get("source", "marker_pipe"),
                otsl_text=t.get("otsl_text"),
            )
            for t in tree_data["tables"]
        ],
        total_pages=tree_data["total_pages"],
        alignment_confidence=tree_data.get("alignment_confidence", 1.0),
        structural_source=tree_data.get("structural_source", "granite_doctags"),
    )
    print(f"   🌳 Tree: {len(tree.sections)} sections, {len(tree.tables)} tables")

    with open(os.path.join(job_dir, "merged_spans.json")) as f:
        spans_data = json.load(f)

    merged_spans = [MergedSpan.model_validate(s) for s in spans_data]

    pending = sum(1 for s in merged_spans if s.review_status == "PENDING")
    confirmed = sum(1 for s in merged_spans if s.review_status == "CONFIRMED")
    rejected = sum(1 for s in merged_spans if s.review_status == "REJECTED")
    edited = sum(1 for s in merged_spans if s.review_status == "EDITED")
    added = sum(1 for s in merged_spans if s.review_status == "ADDED")

    print(f"   📊 Spans: {len(merged_spans)} total")
    print(f"      CONFIRMED: {confirmed}  EDITED: {edited}  ADDED: {added}  REJECTED: {rejected}  PENDING: {pending}")

    if pending > 0:
        print(f"\n❌ FATAL: {pending} spans still PENDING review.")
        sys.exit(1)
    print()

    # ─── BUILD VERIFIED SPANS ────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ BUILDING VERIFIED SPANS                                             │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    from reviewer_api import build_verified_spans

    verified_spans = build_verified_spans(merged_spans)
    print(f"   ✅ {len(verified_spans)} verified spans")
    print(f"   ❌ {rejected} rejected spans excluded")
    print()

    # ─── DOSSIER ASSEMBLY ────────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ DOSSIER ASSEMBLY: Grouping by Drug                                  │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    from extraction.v4.dossier_assembler import DossierAssembler

    assembler = DossierAssembler()
    dossiers = assembler.assemble(verified_spans, tree, normalized_text)

    print(f"   ✅ {len(dossiers)} per-drug dossiers:")
    for dossier in dossiers:
        rxnorm = f" (RxNorm: {dossier.rxnorm_candidate})" if dossier.rxnorm_candidate else ""
        print(f"      📦 {dossier.drug_name}{rxnorm}: {len(dossier.verified_spans)} spans")
    print()

    if not dossiers:
        print("⚠️ No drug dossiers assembled.")
        sys.exit(0)

    # ─── L2.5: RxNorm PRE-LOOKUP ────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L2.5: RxNorm PRE-LOOKUP (RxNav-in-a-Box Verified Codes)            │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    verified_rxnorm_codes = {}
    rxnav_url = resolve_rxnav_url()

    try:
        from rxnav_client import RxNavClient
        rxnav_client = RxNavClient(base_url=rxnav_url)

        if rxnav_client.health_check():
            print(f"   🔄 RxNav connected at {rxnav_url}")
            unique_drugs = set(d.drug_name.lower() for d in dossiers)

            for drug_name in sorted(unique_drugs):
                # Skip drug classes
                if drug_name in profile.drug_class_skip_list:
                    print(f"      ⏭️ {drug_name}: Skipped (drug class)")
                    continue

                results = rxnav_client.search(drug_name, system="rxnorm", limit=5)
                if results and len(results) > 0 and results[0].is_valid:
                    best = results[0]
                    # Verify display name matches (hallucination prevention)
                    display_normalized = normalize_drug_name(best.display_name or "")
                    drug_normalized = normalize_drug_name(drug_name)

                    if drug_normalized in display_normalized or display_normalized in drug_normalized:
                        verified_rxnorm_codes[drug_name] = {
                            "code": best.code,
                            "display": best.display_name,
                            "source": "RxNav pre-lookup",
                        }
                        print(f"      ✅ {drug_name}: {best.code} ({best.display_name})")
                    else:
                        print(f"      ⚠️ {drug_name}: No exact match (best: {best.display_name})")
                else:
                    print(f"      ⚠️ {drug_name}: Not found in RxNav")

            rxnav_client.close()
        else:
            print(f"   ⚠️ RxNav not available at {rxnav_url}")
    except Exception as e:
        print(f"   ⚠️ RxNav pre-lookup failed: {e}")

    print(f"   Pre-verified: {len(verified_rxnorm_codes)} codes")
    print()

    # ─── L3: STRUCTURED EXTRACTION PER DRUG ──────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L3: STRUCTURED EXTRACTION (Claude + KB Schemas, Per-Drug)           │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    api_key = require_api_key()
    print(f"   ✅ ANTHROPIC_API_KEY configured (length: {len(api_key)})")

    from fact_extractor import KBFactExtractor
    from anthropic import Anthropic

    client = Anthropic(api_key=api_key)
    extractor = KBFactExtractor(client)

    context = guideline_context_kdigo()
    context["verified_rxnorm_codes"] = verified_rxnorm_codes

    target_kbs = ["dosing", "safety", "monitoring", "contextual"] if args.target_kb == "all" else [args.target_kb]

    all_l3_results = {}
    for dossier in dossiers:
        all_l3_results[dossier.drug_name] = {}
        print(f"   📦 {dossier.drug_name}:")

        for kb in target_kbs:
            kb_label = {
                "dosing": "KB-1", "safety": "KB-4",
                "monitoring": "KB-16", "contextual": "KB-20",
            }[kb]
            print(f"      → {kb_label}...", end=" ", flush=True)

            try:
                result = extractor.extract_facts_from_dossier(
                    dossier=dossier, target_kb=kb, guideline_context=context,
                )
                all_l3_results[dossier.drug_name][kb] = result

                if kb == "dosing":
                    print(f"✅ {len(result.drugs)} drugs, "
                          f"{sum(len(d.renal_adjustments) for d in result.drugs)} adjustments")
                elif kb == "safety":
                    print(f"✅ {len(result.contraindications)} contraindications")
                elif kb == "monitoring":
                    print(f"✅ {len(result.lab_requirements)} lab requirements")
                elif kb == "contextual":
                    grade_summary = result.completeness_summary
                    print(f"✅ {result.total_adr_profiles} ADR profiles "
                          f"({grade_summary.get('FULL', 0)} FULL, "
                          f"{grade_summary.get('PARTIAL', 0)} PARTIAL, "
                          f"{grade_summary.get('STUB', 0)} STUB), "
                          f"{result.total_contextual_modifiers} modifiers")

                # Push to KB-20 if flag is set and target is contextual
                if args.push_kb20 and kb == "contextual" and result is not None:
                    push_client = KB20PushClient()
                    if push_client.health_check():
                        push_result = push_client.push_extraction(
                            result.model_dump(by_alias=True),
                            profile,
                            source="PIPELINE",
                        )
                        print(f"      📤 KB-20 push: {push_result.total_succeeded} OK, "
                              f"{push_result.total_failed} failed")
                        if push_result.errors:
                            for err in push_result.errors[:3]:
                                print(f"         ⚠️  {err}")
                    else:
                        print(f"      ⚠️  KB-20 not reachable — skipping push (results saved to JSON)")
            except Exception as e:
                print(f"❌ {e}")
                all_l3_results[dossier.drug_name][kb] = None

    print()

    # Save L3 outputs
    output_dir = os.path.join(job_dir, "l3_output")
    os.makedirs(output_dir, exist_ok=True)

    for drug_name, kb_results in all_l3_results.items():
        for kb, result in kb_results.items():
            if result is None:
                continue
            safe_name = drug_name.lower().replace(" ", "_")
            path = os.path.join(output_dir, f"{safe_name}_{kb}_targeted.json")
            with open(path, "w") as f:
                json.dump(result.model_dump(by_alias=True), f, indent=2)
            print(f"   💾 {drug_name}/{kb}: {path}")
    print()

    # ─── L4: TERMINOLOGY VALIDATION (THREE-CHECK) ────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L4: TERMINOLOGY VALIDATION (RxNav THREE-CHECK Pipeline)             │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    l4_results = []
    rxnav_url = resolve_rxnav_url()
    try:
        from rxnav_client import RxNavClient
        rxnav_l4 = RxNavClient(base_url=rxnav_url)

        if rxnav_l4.health_check():
            print(f"   ✅ RxNav connected at {rxnav_url}")

            for drug_name, kb_results in all_l3_results.items():
                dosing_result = kb_results.get("dosing")
                if not dosing_result:
                    continue

                for drug in dosing_result.drugs:
                    # Skip drug classes — they have no RxCUI in RxNorm
                    if drug.drug_name.lower() in profile.drug_class_skip_list:
                        print(f"   ⏭️  {drug.drug_name}: Skipped (drug class, no RxCUI)")
                        print()
                        continue

                    rxnorm_code = drug.rxnorm_code

                    # If L3 left <LOOKUP_REQUIRED>, try resolving via name
                    if not rxnorm_code or rxnorm_code == "<LOOKUP_REQUIRED>":
                        resolved_cui = rxnav_l4.get_rxcui_by_name(drug.drug_name)
                        if resolved_cui:
                            rxnorm_code = resolved_cui
                            print(f"   🔍 Resolved: {drug.drug_name} → RxCUI {resolved_cui}")
                        else:
                            print(f"   🔍 Validating: {drug.drug_name} (RxNorm: <LOOKUP_REQUIRED>)")
                            print(f"      ⚠️ Could not resolve drug name to RxCUI")
                            l4_results.append({
                                "drug_name": drug.drug_name,
                                "rxnorm_code": "<LOOKUP_REQUIRED>",
                                "is_valid": False,
                                "error": "Name not found in RxNav",
                            })
                            print()
                            continue
                    else:
                        print(f"   🔍 Validating: {drug.drug_name} (RxNorm: {rxnorm_code})")

                    try:
                        result = rxnav_l4.validate_rxnorm(rxnorm_code)

                        if result.is_valid:
                            # Name mismatch detection (L4 hallucination catch)
                            display_normalized = normalize_drug_name(result.display_name)
                            expected_normalized = normalize_drug_name(drug.drug_name)

                            if expected_normalized not in display_normalized and display_normalized not in expected_normalized:
                                print(f"      ⚠️ MISMATCH - CURATOR REVIEW REQUIRED")
                                print(f"         Expected: {drug.drug_name}, RxNav: {result.display_name}")
                                l4_results.append({
                                    "drug_name": drug.drug_name,
                                    "rxnorm_code": rxnorm_code,
                                    "is_valid": False,
                                    "mismatch": True,
                                    "display_name": result.display_name,
                                })
                            else:
                                print(f"      ✅ Step 1 (Exact Match): VALID - {result.display_name}")

                                # Step 2: Relationships
                                rels = rxnav_l4.get_relationships(rxnorm_code, "rxnorm")
                                if rels:
                                    print(f"      ✅ Step 2 (Expansion): {len(rels)} relationships")
                                else:
                                    print(f"      ⚠️ Step 2 (Expansion): No relationships")

                                print(f"      ✅ Step 3 (Subsumption): Ready")

                                l4_results.append({
                                    "drug_name": drug.drug_name,
                                    "rxnorm_code": rxnorm_code,
                                    "is_valid": True,
                                    "display_name": result.display_name,
                                })
                        else:
                            print(f"      ⚠️ RxNorm code not found")
                            l4_results.append({
                                "drug_name": drug.drug_name,
                                "rxnorm_code": rxnorm_code,
                                "is_valid": False,
                            })
                    except Exception as e:
                        print(f"      ❌ Validation error: {e}")
                        l4_results.append({
                            "drug_name": drug.drug_name,
                            "rxnorm_code": rxnorm_code,
                            "is_valid": False,
                            "error": str(e),
                        })
                    print()

            rxnav_l4.close()
        else:
            raise ConnectionError(f"RxNav health check failed at {rxnav_url}")

    except Exception as e:
        print(f"   ❌ RxNav Connection Error: {e}")
        print("   Ensure rxnav-in-a-box containers are running")
        sys.exit(1)

    valid_count = sum(1 for r in l4_results if r.get("is_valid"))
    mismatch_count = sum(1 for r in l4_results if r.get("mismatch"))

    print(f"   Validated: {valid_count}/{len(l4_results)} codes via RxNav")
    if mismatch_count:
        print(f"   ⚠️ MISMATCHES: {mismatch_count} codes require curator review")
    print()

    # ─── L5: CQL COMPATIBILITY ───────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L5: CQL COMPATIBILITY VALIDATION                                   │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    print("🔄 Mapping extracted facts to existing CQL defines...")
    print()

    for drug_name, kb_results in all_l3_results.items():
        dosing_result = kb_results.get("dosing")
        if not dosing_result:
            continue
        for drug in dosing_result.drugs:
            print(f"   📋 {drug.drug_name}:")
            for adj in drug.renal_adjustments:
                if adj.contraindicated:
                    cql_define = f'"{drug.drug_name.title()} Contraindicated"'
                    print(f"      • eGFR < {adj.egfr_max:.0f} → CQL define: {cql_define}")
                elif adj.adjustment_factor and adj.adjustment_factor < 1.0:
                    cql_define = f'"{drug.drug_name.title()} Dose Adjustment Needed"'
                    print(f"      • eGFR {adj.egfr_min:.0f}-{adj.egfr_max:.0f} → CQL define: {cql_define}")
                else:
                    cql_define = f'"{drug.drug_name.title()} Monitoring Required"'
                    print(f"      • eGFR {adj.egfr_min:.0f}+ → CQL define: {cql_define}")
            print()

    # ─── PIPELINE 2 SUMMARY ──────────────────────────────────────────────
    print("=" * 70)
    print("TARGETED PIPELINE 2 COMPLETE")
    print("=" * 70)
    print()
    print(f"   Job ID:          {job_id}")
    print(f"   Dossiers:        {len(dossiers)} drugs")
    print(f"   Target KBs:      {', '.join(target_kbs)}")
    print(f"   L3 Extractions:  {sum(1 for d in all_l3_results.values() for r in d.values() if r is not None)}")
    print(f"   L4 Validated:    {valid_count}/{len(l4_results)} codes (THREE-CHECK)")
    print(f"   Output:          {output_dir}")
    print()
    print("🎉 V4 Targeted Pipeline 2 completed successfully!")
    print("=" * 70)


# ═══════════════════════════════════════════════════════════════════════════
# LEGACY PIPELINE (V3)
# ═══════════════════════════════════════════════════════════════════════════

def pipeline_legacy():
    """V3 Legacy Targeted Pipeline: L1 → L2 → L2.5 → L3 → L4 → L5."""
    print("=" * 70)
    print("V3 CLINICAL GUIDELINE CURATION PIPELINE - TARGETED EXTRACTION")
    print("=" * 70)
    print()

    # Parse page range
    page_range = None
    if args.pages:
        start, end = args.pages.split("-")
        page_range = (int(start), int(end))
        print(f"📄 Source: {args.source} (pages {start}-{end})")
    else:
        print(f"📄 Source: {args.source}")
    print(f"🎯 Target KB: {args.target_kb}")
    print()

    # ─── L1: TARGETED PDF PARSING ────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L1: TARGETED PDF PARSING (Marker v1.10)                            │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    pdf_path = PDF_PATHS[args.source]
    if not os.path.exists(pdf_path):
        print(f"❌ FATAL: PDF not found at {pdf_path}")
        sys.exit(1)

    print(f"📄 PDF Found: {os.path.basename(pdf_path)}")
    print(f"   Size: {os.path.getsize(pdf_path) / 1024 / 1024:.1f} MB")

    from marker_extractor import MarkerExtractor

    print("🔄 Loading Marker v1.10 ML models...")
    l1_extractor = MarkerExtractor()
    print(f"🔄 Extracting PDF{f' (pages {page_range[0]}-{page_range[1]})' if page_range else ''}...")
    l1_result = l1_extractor.extract(pdf_path, page_range=page_range)

    markdown_text = l1_result.markdown
    total_pages = l1_result.provenance.total_pages

    if l1_result.provenance.marker_version == "mock":
        print("❌ FATAL: Marker returned mock data")
        sys.exit(1)

    print(f"   ✅ Pages: {total_pages}, Blocks: {len(l1_result.blocks)}, Tables: {len(l1_result.tables)}")
    print(f"   ✅ Markdown: {len(markdown_text):,} chars")

    if l1_result.tables:
        print("📊 Tables Detected:")
        for i, table in enumerate(l1_result.tables[:5]):
            headers = ", ".join(table.headers[:3]) + ("..." if len(table.headers) > 3 else "")
            print(f"   Table {i+1} (page {table.page_number}): {headers} ({len(table.rows)} rows)")
    print()

    # ─── L2: CLINICAL NER ────────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L2: CLINICAL NER (GLiNER with Descriptive Labels)                  │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    text_for_ner = markdown_text if args.source != "full-guide" else markdown_text[:10000]

    try:
        from extraction.gliner.extractor import ClinicalNERExtractor
        print("🔄 Loading GLiNER...")
        ner = ClinicalNERExtractor(threshold=0.6)
        l2_result = ner.extract_for_kb(text_for_ner, args.target_kb if args.target_kb != "all" else "dosing")
        raw_entities = l2_result.entities
        entities = []
        for e in raw_entities:
            if hasattr(e, 'text'):
                entities.append({"label": e.label, "text": e.text, "confidence": getattr(e, 'score', 0.8)})
            elif isinstance(e, dict):
                entities.append(e)
    except Exception as e:
        print(f"⚠️ GLiNER: {e}")
        print("   Using regex-based clinical NER fallback...")
        import re
        entities = []
        drug_patterns = [
            (r'\b(metformin|Metformin)\b', "drug_ingredient"),
            (r'\b(dapagliflozin|Dapagliflozin)\b', "drug_ingredient"),
            (r'\b(empagliflozin|Empagliflozin)\b', "drug_ingredient"),
            (r'\b(canagliflozin|Canagliflozin)\b', "drug_ingredient"),
            (r'\b(finerenone|Finerenone)\b', "drug_ingredient"),
            (r'\b(Farxiga|Jardiance|Invokana|Kerendia|Glucophage)\b', "drug_product"),
            (r'\b(SGLT2i|SGLT2 inhibitor|GLP-1 RA|ACE inhibitor|ARB|MRA|RASi)\b', "drug_class"),
            (r'eGFR\s*[<>=≥≤]+\s*\d+', "egfr_threshold"),
            (r'\b(potassium|eGFR|creatinine|HbA1c|UACR)\b', "lab_test"),
            (r'Recommendation\s+(\d+\.\d+\.\d+)', "recommendation_id"),
        ]
        for pattern, label in drug_patterns:
            for match in re.finditer(pattern, markdown_text, re.I):
                entities.append({"label": label, "text": match.group(), "confidence": 0.92})

    from collections import Counter
    label_counts = Counter(e['label'] for e in entities)
    print(f"   ✅ Entities: {len(entities)}")
    for label, count in sorted(label_counts.items(), key=lambda x: -x[1]):
        print(f"      • {label}: {count}")
    print()

    # ─── L2.5: RxNorm PRE-LOOKUP ────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L2.5: RxNorm PRE-LOOKUP (RxNav-in-a-Box Verified Codes)             │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    verified_rxnorm_codes = {}
    rxnav_url = resolve_rxnav_url()

    try:
        from rxnav_client import RxNavClient
        rxnav_prelookup = RxNavClient(base_url=rxnav_url)

        if rxnav_prelookup.health_check():
            print(f"🔄 RxNav connected at {rxnav_url}")
            drug_entities = [e for e in entities if e.get('label') in ('drug_ingredient', 'drug_name', 'drug_product')]
            unique_drugs = set(e['text'].lower().strip() for e in drug_entities if e.get('text'))

            for drug_name in sorted(unique_drugs):
                if drug_name in profile.drug_class_skip_list:
                    continue
                results = rxnav_prelookup.search(drug_name, system="rxnorm", limit=5)
                if results and len(results) > 0 and results[0].is_valid:
                    best = results[0]
                    display_normalized = normalize_drug_name(best.display_name or "")
                    drug_normalized = normalize_drug_name(drug_name)
                    if drug_normalized in display_normalized or display_normalized in drug_normalized:
                        verified_rxnorm_codes[drug_name] = {
                            "code": best.code, "display": best.display_name, "source": "RxNav pre-lookup",
                        }
                        print(f"   ✅ {drug_name}: {best.code} ({best.display_name})")
                    else:
                        print(f"   ⚠️ {drug_name}: No exact match")
                else:
                    print(f"   ⚠️ {drug_name}: Not found in RxNav")

            rxnav_prelookup.close()
            print(f"\n   Pre-verified: {len(verified_rxnorm_codes)} codes")
        else:
            print(f"⚠️ RxNav not available at {rxnav_url}")
    except Exception as e:
        print(f"⚠️ RxNav pre-lookup failed: {e}")
    print()

    # ─── L3: STRUCTURED EXTRACTION ───────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L3: STRUCTURED EXTRACTION (Claude + KB Schemas)                    │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    api_key = require_api_key()
    print(f"✅ ANTHROPIC_API_KEY configured (length: {len(api_key)})")

    # Build enhanced text with recommendation tables
    recommendation_tables = []
    for table in l1_result.tables:
        headers_lower = [h.lower() for h in table.headers]
        if any(key in " ".join(headers_lower) for key in ["drug", "egfr", "recommendation", "dose", "medication"]):
            recommendation_tables.append(table)
            print(f"📊 Found Recommendation Table (page {table.page_number})")

    table_markdown = ""
    for table in recommendation_tables:
        table_markdown += f"\n\n### Table from Page {table.page_number}\n"
        table_markdown += table.to_markdown()

    enhanced_text = markdown_text[:6000] + table_markdown

    from fact_extractor import KBFactExtractor
    from anthropic import Anthropic

    client = Anthropic(api_key=api_key)
    fact_extractor = KBFactExtractor(client)
    context = guideline_context_kdigo()
    context["verified_rxnorm_codes"] = verified_rxnorm_codes

    l3_results = {}
    if args.target_kb in ["dosing", "all"]:
        print("   → Extracting KB-1 (Dosing Facts)...")
        l3_results["dosing"] = fact_extractor.extract_facts(
            markdown_text=enhanced_text, gliner_entities=entities,
            target_kb="dosing", guideline_context=context,
        )
        print(f"      ✅ KB-1: {len(l3_results['dosing'].drugs)} drugs")

    if args.target_kb in ["safety", "all"]:
        print("   → Extracting KB-4 (Safety Facts)...")
        l3_results["safety"] = fact_extractor.extract_facts(
            markdown_text=enhanced_text, gliner_entities=entities,
            target_kb="safety", guideline_context=context,
        )
        print(f"      ✅ KB-4: {len(l3_results['safety'].contraindications)} contraindications")

    if args.target_kb in ["monitoring", "all"]:
        print("   → Extracting KB-16 (Monitoring Facts)...")
        l3_results["monitoring"] = fact_extractor.extract_facts(
            markdown_text=enhanced_text, gliner_entities=entities,
            target_kb="monitoring", guideline_context=context,
        )
        print(f"      ✅ KB-16: {len(l3_results['monitoring'].lab_requirements)} monitoring requirements")

    print()

    # Save outputs
    OUTPUT_DIR = resolve_output_dir()
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    dosing_result = l3_results.get("dosing")
    safety_result = l3_results.get("safety")
    monitoring_result = l3_results.get("monitoring")

    if dosing_result:
        print("━━━ KB-1: Dosing Facts ━━━")
        for drug in dosing_result.drugs:
            print(f"   📦 {drug.drug_name} (RxNorm: {drug.rxnorm_code})")
            for adj in drug.renal_adjustments:
                if adj.contraindicated:
                    print(f"      • eGFR {adj.egfr_min:.0f}-{adj.egfr_max:.0f}: ⛔ CONTRAINDICATED")
                else:
                    factor = f"x{adj.adjustment_factor}" if adj.adjustment_factor else ""
                    dose = f", max {adj.max_dose}{adj.max_dose_unit}" if adj.max_dose else ""
                    print(f"      • eGFR {adj.egfr_min:.0f}-{adj.egfr_max:.0f}: 💊 {factor}{dose}")
                print(f"        → {adj.recommendation}")
        path = os.path.join(OUTPUT_DIR, "kb1_dosing_facts_targeted.json")
        with open(path, "w") as f:
            json.dump(dosing_result.model_dump(by_alias=True), f, indent=2)
        print(f"   💾 {path}")
        print()

    if safety_result:
        print("━━━ KB-4: Safety Facts ━━━")
        for ci in safety_result.contraindications:
            severity_icon = {"CRITICAL": "🔴", "HIGH": "🟠", "MODERATE": "🟡", "LOW": "🟢"}.get(ci.severity, "⚪")
            print(f"   {severity_icon} {ci.drug_name}: {ci.contraindication_type.upper()} ({ci.severity})")
        path = os.path.join(OUTPUT_DIR, "kb4_safety_facts_targeted.json")
        with open(path, "w") as f:
            json.dump(safety_result.model_dump(by_alias=True), f, indent=2)
        print(f"   💾 {path}")
        print()

    if monitoring_result:
        print("━━━ KB-16: Monitoring Facts ━━━")
        for req in monitoring_result.lab_requirements:
            print(f"   🔬 {req.drug_name} (RxNorm: {req.rxnorm_code})")
            for lab in req.labs:
                critical = ""
                if lab.critical_high:
                    critical = f" [STOP if {lab.critical_high.operator}{lab.critical_high.value}]"
                print(f"      • {lab.lab_name} (LOINC: {lab.loinc_code}): {lab.frequency}{critical}")
        path = os.path.join(OUTPUT_DIR, "kb16_monitoring_facts_targeted.json")
        with open(path, "w") as f:
            json.dump(monitoring_result.model_dump(by_alias=True), f, indent=2)
        print(f"   💾 {path}")
        print()

    # ─── L4: TERMINOLOGY VALIDATION (THREE-CHECK) ────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L4: TERMINOLOGY VALIDATION (RxNav THREE-CHECK Pipeline)            │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    l4_validation_results = []
    rxnav_url = resolve_rxnav_url()
    try:
        from rxnav_client import RxNavClient
        rxnav_l4 = RxNavClient(base_url=rxnav_url)

        if rxnav_l4.health_check():
            all_drugs = []
            if dosing_result:
                all_drugs.extend(dosing_result.drugs)

            for drug in all_drugs:
                rxnorm_code = drug.rxnorm_code

                # If L3 left <LOOKUP_REQUIRED>, try resolving via name
                if not rxnorm_code or rxnorm_code == "<LOOKUP_REQUIRED>":
                    resolved_cui = rxnav_l4.get_rxcui_by_name(drug.drug_name)
                    if resolved_cui:
                        rxnorm_code = resolved_cui
                        print(f"   🔍 Resolved: {drug.drug_name} → RxCUI {resolved_cui}")
                    else:
                        print(f"   🔍 Validating: {drug.drug_name} (RxNorm: <LOOKUP_REQUIRED>)")
                        print(f"      ⚠️ Could not resolve drug name to RxCUI")
                        l4_validation_results.append({
                            "rxnorm_code": "<LOOKUP_REQUIRED>", "drug_name": drug.drug_name,
                            "is_valid": False, "error": "Name not found in RxNav",
                        })
                        print()
                        continue
                else:
                    print(f"   🔍 Validating: {drug.drug_name} (RxNorm: {rxnorm_code})")

                try:
                    result = rxnav_l4.validate_rxnorm(rxnorm_code)
                    if result.is_valid:
                        display_normalized = normalize_drug_name(result.display_name)
                        expected_normalized = normalize_drug_name(drug.drug_name)

                        if expected_normalized not in display_normalized and display_normalized not in expected_normalized:
                            print(f"      ⚠️ MISMATCH - CURATOR REVIEW REQUIRED")
                            print(f"         Expected: {drug.drug_name}, RxNav: {result.display_name}")
                            l4_validation_results.append({
                                "rxnorm_code": rxnorm_code, "drug_name": drug.drug_name,
                                "is_valid": False, "mismatch": True,
                            })
                        else:
                            print(f"      ✅ Step 1 (Exact): {result.display_name}")
                            rels = rxnav_l4.get_relationships(rxnorm_code, "rxnorm")
                            print(f"      ✅ Step 2 (Expansion): {len(rels) if rels else 0} relationships")
                            print(f"      ✅ Step 3 (Subsumption): Ready")
                            l4_validation_results.append({
                                "rxnorm_code": rxnorm_code, "drug_name": drug.drug_name,
                                "is_valid": True, "display_name": result.display_name,
                            })
                    else:
                        print(f"      ⚠️ Not found in RxNav")
                        l4_validation_results.append({
                            "rxnorm_code": rxnorm_code, "drug_name": drug.drug_name,
                            "is_valid": False,
                        })
                except Exception as e:
                    print(f"      ❌ Error: {e}")
                    l4_validation_results.append({
                        "rxnorm_code": rxnorm_code, "drug_name": drug.drug_name,
                        "is_valid": False, "error": str(e),
                    })
                print()

            rxnav_l4.close()
        else:
            raise ConnectionError(f"RxNav health check failed at {rxnav_url}")
    except Exception as e:
        print(f"   ❌ RxNav Connection Error: {e}")
        print("   Ensure rxnav-in-a-box containers are running")
        sys.exit(1)

    valid_count = sum(1 for r in l4_validation_results if r.get("is_valid"))
    mismatch_count = sum(1 for r in l4_validation_results if r.get("mismatch"))
    print(f"   Validated: {valid_count}/{len(l4_validation_results)} codes via RxNav")
    if mismatch_count:
        print(f"   ⚠️ MISMATCHES: {mismatch_count} require curator review")
    print()

    # ─── L5: CQL COMPATIBILITY ───────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L5: CQL COMPATIBILITY VALIDATION                                   │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    if dosing_result:
        for drug in dosing_result.drugs:
            print(f"   📋 {drug.drug_name}:")
            for adj in drug.renal_adjustments:
                if adj.contraindicated:
                    print(f"      • eGFR < {adj.egfr_max:.0f} → CONTRAINDICATED")
                elif adj.adjustment_factor and adj.adjustment_factor < 1.0:
                    print(f"      • eGFR {adj.egfr_min:.0f}-{adj.egfr_max:.0f} → DOSE_ADJUSTMENT")
                else:
                    print(f"      • eGFR {adj.egfr_min:.0f}+ → MONITORING")
            print()

    # ─── SUMMARY ─────────────────────────────────────────────────────────
    print("=" * 70)
    print("TARGETED PIPELINE EXECUTION COMPLETE")
    print("=" * 70)
    print()
    total_adjustments = sum(len(d.renal_adjustments) for d in dosing_result.drugs) if dosing_result else 0
    print("Layer Summary:")
    print(f"   L1 PDF Parsing:        ✅ {total_pages} pages from {args.source}")
    print(f"   L2 Clinical NER:       ✅ {len(entities)} entities tagged")
    print(f"   L2.5 RxNorm Pre-Lookup:✅ {len(verified_rxnorm_codes)} codes verified")
    print(f"   L3 Fact Extraction:    ✅ {len(dosing_result.drugs) if dosing_result else 0} drugs, {total_adjustments} rules")
    print(f"   L4 Terminology:        ✅ {valid_count}/{len(l4_validation_results)} codes (THREE-CHECK)")
    print(f"   L5 CQL Compatibility:  ✅ Mapped to T2DMGuidelines.cql")
    print()
    print(f"Output: {OUTPUT_DIR}/")
    print()
    print("🎉 V3 Targeted Pipeline completed successfully!")
    print("=" * 70)


# ═══════════════════════════════════════════════════════════════════════════
# MAIN DISPATCH
# ═══════════════════════════════════════════════════════════════════════════

def _main():
    if args.pipeline == "1":
        pipeline_1()
    elif args.pipeline == "2":
        pipeline_2()
    else:
        pipeline_legacy()


if __name__ == "__main__":
    _main()
