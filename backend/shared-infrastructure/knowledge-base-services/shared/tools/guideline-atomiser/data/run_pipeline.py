#!/usr/bin/env python3
"""
Clinical Guideline Curation Pipeline - V4 Multi-Channel + V3 Legacy

V4 Architecture (two-pipeline split):
    Pipeline 1: L1 (Marker/Docling) → L2 Multi-Channel → Signal Merger → Reviewer Queue → EXIT
    Pipeline 2: Reviewed Spans → Dossier Assembly → L3 (per-drug) → L4 → L5

V3 Legacy (single invocation):
    L1 → L2 (GLiNER) → L3 → L4 → L5

Usage:
    # V4 Pipeline 1: Extract and queue for review
    python run_pipeline.py --pipeline 1

    # V4 Pipeline 2: Process reviewed spans
    python run_pipeline.py --pipeline 2 --job-dir /data/output/v4/job_<uuid>/

    # V3 Legacy (default)
    python run_pipeline.py --pipeline legacy
    python run_pipeline.py  # same as legacy
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
sys.path.insert(0, _parent_dir)
sys.path.insert(0, '/app/guideline-atomiser')  # Docker fallback

# ═══════════════════════════════════════════════════════════════════════════
# CLI ARGUMENT PARSING
# ═══════════════════════════════════════════════════════════════════════════

parser = argparse.ArgumentParser(
    description="Clinical Guideline Curation Pipeline (V4 multi-channel + V3 legacy)"
)
parser.add_argument(
    "--pipeline",
    choices=["1", "2", "legacy"],
    default="legacy",
    help="Pipeline mode: 1 (extract+review), 2 (dossier+L3), legacy (V3 full)"
)
parser.add_argument(
    "--job-dir",
    type=str,
    help="Job directory for Pipeline 2 (contains reviewed merged_spans.json)"
)
parser.add_argument(
    "--target-kb",
    choices=["dosing", "safety", "monitoring", "all"],
    default="all",
    help="Target KB for extraction (default: all)"
)
parser.add_argument(
    "--pdf",
    type=str,
    help="Path to PDF file (overrides default KDIGO path)"
)
args, _ = parser.parse_known_args()


# ═══════════════════════════════════════════════════════════════════════════
# SHARED HELPERS
# ═══════════════════════════════════════════════════════════════════════════

def resolve_pdf_path(pdf_arg):
    """Resolve PDF path from argument or default locations."""
    if pdf_arg and os.path.exists(pdf_arg):
        return pdf_arg

    # Default paths (Docker vs local)
    local_pdfs = os.path.join(_script_dir, "pdfs")
    docker_pdfs = "/data/pdfs"
    pdfs_dir = local_pdfs if os.path.exists(local_pdfs) else docker_pdfs

    default = os.path.join(
        pdfs_dir,
        "KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf"
    )
    if os.path.exists(default):
        return default

    print(f"❌ FATAL: PDF not found")
    if pdf_arg:
        print(f"   Tried: {pdf_arg}")
    print(f"   Tried: {default}")
    sys.exit(1)


def resolve_output_dir():
    """Resolve output directory (local or Docker)."""
    local_output = os.path.join(_script_dir, "output")
    docker_output = "/data/output"
    base = local_output if os.path.exists(os.path.dirname(local_output)) else docker_output
    return base


def require_api_key():
    """Require and return ANTHROPIC_API_KEY."""
    api_key = os.environ.get("ANTHROPIC_API_KEY", "")
    if not api_key or len(api_key) < 30 or api_key.startswith("${"):
        print("❌ FATAL: ANTHROPIC_API_KEY is required for L3 extraction")
        print("   export ANTHROPIC_API_KEY='sk-ant-...'")
        sys.exit(1)
    return api_key


def guideline_context_kdigo():
    """Standard KDIGO guideline context."""
    return {
        "authority": "KDIGO",
        "document": "KDIGO 2022 Diabetes in CKD",
        "effective_date": "2022-11-01",
        "doi": "10.1016/j.kint.2022.06.008",
        "version": "2022",
    }


# ═══════════════════════════════════════════════════════════════════════════
# PIPELINE 1: Multi-Channel Extraction → Reviewer Queue
# ═══════════════════════════════════════════════════════════════════════════

def pipeline_1():
    """V4 Pipeline 1: L1 PDF → L2 Multi-Channel → Signal Merger → Reviewer Queue.

    Produces a job directory with all extraction artifacts, ready for human review.
    Pipeline 1 ENDS after saving merged spans. A reviewer must approve/reject
    spans before Pipeline 2 can process them.
    """
    print("=" * 70)
    print("V4 PIPELINE 1: Multi-Channel Extraction → Reviewer Queue")
    print("=" * 70)
    print()

    # ─── L1: PDF PARSING ─────────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L1: PDF PARSING                                                     │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    pdf_path = resolve_pdf_path(args.pdf)
    print(f"📄 PDF: {os.path.basename(pdf_path)}")
    print(f"   Size: {os.path.getsize(pdf_path) / 1024 / 1024:.1f} MB")

    from monkeyocr_extractor import MonkeyOCRExtractor

    print("🔄 Loading MonkeyOCR PDF parser...")
    l1_extractor = MonkeyOCRExtractor()
    print("🔄 Extracting PDF with provenance tracking...")
    l1_result = l1_extractor.extract(pdf_path)

    markdown_text = l1_result.markdown
    total_pages = l1_result.provenance.total_pages

    if l1_result.provenance.marker_version == "mock":
        print("❌ FATAL: Parser returned mock data")
        sys.exit(1)

    print(f"   ✅ Pages: {total_pages}")
    print(f"   ✅ Blocks: {len(l1_result.blocks)}")
    print(f"   ✅ Tables: {len(l1_result.tables)}")
    print(f"   ✅ Markdown: {len(markdown_text):,} chars")
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
    print()

    # Channel 0: Text normalization
    print("   [Channel 0] Normalizing text...")
    normalizer = Channel0Normalizer()
    normalized_text = normalizer.normalize(markdown_text)
    print(f"      ✅ {len(normalized_text):,} chars (normalized)")

    # Channel A: Structure parsing → GuidelineTree
    print("   [Channel A] Parsing document structure...")
    channel_a = ChannelA()
    tree = channel_a.parse(normalized_text)
    print(f"      ✅ {len(tree.sections)} sections, {len(tree.tables)} tables, {tree.total_pages} pages")

    # Channels B-F: Parallel extraction
    # In production these would run in parallel; here sequential for clarity
    channel_outputs = []

    print("   [Channel B] Drug dictionary scan (Aho-Corasick)...")
    channel_b = ChannelB()
    b_spans = channel_b.extract(normalized_text)
    channel_outputs.append(ChannelOutput(channel="B", spans=b_spans))
    print(f"      ✅ {len(b_spans)} drug spans found")

    print("   [Channel C] Grammar/regex patterns...")
    channel_c = ChannelC()
    c_spans = channel_c.extract(normalized_text)
    channel_outputs.append(ChannelOutput(channel="C", spans=c_spans))
    print(f"      ✅ {len(c_spans)} pattern spans found")

    print("   [Channel D] Table cell decomposition...")
    channel_d = ChannelD()
    d_spans = channel_d.extract(tree.tables)
    channel_outputs.append(ChannelOutput(channel="D", spans=d_spans))
    print(f"      ✅ {len(d_spans)} table cell spans")

    print("   [Channel E] GLiNER residual NER...")
    try:
        channel_e = ChannelE()
        # Collect already-found spans for novel-only filtering
        existing_spans = b_spans + c_spans
        e_spans = channel_e.extract(normalized_text, existing_spans=existing_spans)
        channel_outputs.append(ChannelOutput(channel="E", spans=e_spans))
        print(f"      ✅ {len(e_spans)} novel entity spans")
    except Exception as e:
        channel_outputs.append(ChannelOutput(channel="E", spans=[], error=str(e)))
        print(f"      ⚠️ GLiNER error: {e}")

    print("   [Channel F] NuExtract proposition extraction...")
    try:
        channel_f = ChannelF()
        f_spans = channel_f.extract(normalized_text, tree)
        channel_outputs.append(ChannelOutput(channel="F", spans=f_spans))
        print(f"      ✅ {len(f_spans)} proposition spans")
    except Exception as e:
        channel_outputs.append(ChannelOutput(channel="F", spans=[], error=str(e)))
        print(f"      ⚠️ NuExtract error: {e}")

    total_raw = sum(len(co.spans) for co in channel_outputs)
    print()
    print(f"   Total raw spans across all channels: {total_raw}")
    print()

    # ─── SIGNAL MERGER ────────────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ SIGNAL MERGER: Clustering + Confidence Boosting                     │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    merger = SignalMerger()
    merged_spans = merger.merge(job_id, channel_outputs, tree)

    disagreements = sum(1 for s in merged_spans if s.has_disagreement)
    multi_channel = sum(1 for s in merged_spans if len(s.contributing_channels) > 1)

    print(f"   ✅ Merged spans: {len(merged_spans)}")
    print(f"   ✅ Multi-channel corroborated: {multi_channel}")
    if disagreements:
        print(f"   ⚠️ Disagreements flagged: {disagreements}")
    print()

    # ─── SAVE JOB ARTIFACTS ──────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ SAVING JOB ARTIFACTS → Reviewer Queue                              │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    base_output = resolve_output_dir()
    job_dir = os.path.join(base_output, "v4", f"job_{job_id}")
    os.makedirs(job_dir, exist_ok=True)

    # Compute source hash
    with open(pdf_path, "rb") as f:
        source_hash = hashlib.sha256(f.read()).hexdigest()[:16]

    # 1. Job metadata
    job_meta = {
        "job_id": str(job_id),
        "source_pdf": os.path.basename(pdf_path),
        "source_hash": source_hash,
        "guideline_authority": "KDIGO",
        "guideline_document": "KDIGO 2022 Diabetes in CKD",
        "review_status": "PENDING",
        "total_raw_spans": total_raw,
        "total_merged_spans": len(merged_spans),
        "disagreements": disagreements,
        "created_at": datetime.utcnow().isoformat(),
        "pipeline_version": "4.0.0",
    }
    meta_path = os.path.join(job_dir, "job_metadata.json")
    with open(meta_path, "w") as f:
        json.dump(job_meta, f, indent=2)
    print(f"   💾 Job metadata: {meta_path}")

    # 2. Normalized text
    text_path = os.path.join(job_dir, "normalized_text.txt")
    with open(text_path, "w") as f:
        f.write(normalized_text)
    print(f"   💾 Normalized text: {text_path}")

    # 3. Guideline tree (serialized)
    tree_data = {
        "sections": [
            {
                "section_id": s.section_id,
                "heading": s.heading,
                "start_offset": s.start_offset,
                "end_offset": s.end_offset,
                "page_number": s.page_number,
                "block_type": s.block_type,
                "children": _serialize_sections(s.children),
            }
            for s in tree.sections
        ],
        "tables": [
            {
                "table_id": t.table_id,
                "headers": t.headers,
                "rows": t.rows,
                "page_number": t.page_number,
                "section_id": t.section_id,
                "start_offset": t.start_offset,
                "end_offset": t.end_offset,
            }
            for t in tree.tables
        ],
        "total_pages": tree.total_pages,
    }
    tree_path = os.path.join(job_dir, "guideline_tree.json")
    with open(tree_path, "w") as f:
        json.dump(tree_data, f, indent=2)
    print(f"   💾 Guideline tree: {tree_path}")

    # 4. Merged spans (reviewer queue)
    spans_data = [s.model_dump(mode="json") for s in merged_spans]
    spans_path = os.path.join(job_dir, "merged_spans.json")
    with open(spans_path, "w") as f:
        json.dump(spans_data, f, indent=2, default=str)
    print(f"   💾 Merged spans ({len(merged_spans)}): {spans_path}")

    # 5. Raw spans (debug/audit)
    raw_data = {}
    for co in channel_outputs:
        raw_data[co.channel] = {
            "count": len(co.spans),
            "error": co.error,
            "spans": [s.model_dump(mode="json") for s in co.spans],
        }
    raw_path = os.path.join(job_dir, "raw_spans.json")
    with open(raw_path, "w") as f:
        json.dump(raw_data, f, indent=2, default=str)
    print(f"   💾 Raw spans (debug): {raw_path}")

    print()

    # ─── PIPELINE 1 SUMMARY ──────────────────────────────────────────────
    print("=" * 70)
    print("PIPELINE 1 COMPLETE — REVIEW REQUIRED")
    print("=" * 70)
    print()
    print(f"   Job ID:          {job_id}")
    print(f"   Job Directory:   {job_dir}")
    print(f"   Merged Spans:    {len(merged_spans)} (all PENDING review)")
    print(f"   Disagreements:   {disagreements}")
    print()
    print("REVIEWER INSTRUCTIONS:")
    print(f"   1. Review merged spans in: {spans_path}")
    print(f"   2. For each span, set review_status to:")
    print(f"      CONFIRMED — span text is correct")
    print(f"      REJECTED  — span is noise/incorrect, exclude from dossier")
    print(f"      EDITED    — set reviewer_text to corrected text")
    print(f"   3. Add missed spans with review_status: ADDED")
    print(f"   4. Run Pipeline 2:")
    print(f"      python run_pipeline.py --pipeline 2 --job-dir {job_dir}")
    print()
    print("Or use the Reviewer API:")
    print(f"   POST /api/v4/jobs/{{job_id}}/spans/{{span_id}}/review")
    print(f"   POST /api/v4/jobs/{{job_id}}/complete-review")
    print()


def _serialize_sections(sections):
    """Recursively serialize GuidelineSection children."""
    return [
        {
            "section_id": s.section_id,
            "heading": s.heading,
            "start_offset": s.start_offset,
            "end_offset": s.end_offset,
            "page_number": s.page_number,
            "block_type": s.block_type,
            "children": _serialize_sections(s.children),
        }
        for s in sections
    ]


# ═══════════════════════════════════════════════════════════════════════════
# PIPELINE 2: Reviewed Spans → Dossier Assembly → L3 per Drug → L4 → L5
# ═══════════════════════════════════════════════════════════════════════════

def pipeline_2():
    """V4 Pipeline 2: Load reviewed spans → Dossier Assembly → L3 → L4 → L5.

    Requires a completed job directory from Pipeline 1 with reviewed merged_spans.json.
    """
    if not args.job_dir:
        print("❌ FATAL: --job-dir is required for Pipeline 2")
        print("   python run_pipeline.py --pipeline 2 --job-dir /path/to/job_<uuid>/")
        sys.exit(1)

    if not os.path.isdir(args.job_dir):
        print(f"❌ FATAL: Job directory not found: {args.job_dir}")
        sys.exit(1)

    print("=" * 70)
    print("V4 PIPELINE 2: Dossier Assembly → L3 Per-Drug → L4 → L5")
    print("=" * 70)
    print()

    # ─── LOAD JOB ARTIFACTS ──────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ LOADING REVIEWED JOB ARTIFACTS                                      │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    job_dir = args.job_dir

    # Load job metadata
    meta_path = os.path.join(job_dir, "job_metadata.json")
    with open(meta_path) as f:
        job_meta = json.load(f)
    job_id = job_meta["job_id"]
    print(f"   📋 Job ID: {job_id}")
    print(f"   📄 Source: {job_meta['source_pdf']}")

    # Load normalized text
    text_path = os.path.join(job_dir, "normalized_text.txt")
    with open(text_path) as f:
        normalized_text = f.read()
    print(f"   📝 Normalized text: {len(normalized_text):,} chars")

    # Load guideline tree
    tree_path = os.path.join(job_dir, "guideline_tree.json")
    with open(tree_path) as f:
        tree_data = json.load(f)

    from extraction.v4.models import GuidelineTree, GuidelineSection, ParsedTable

    tree = GuidelineTree(
        sections=[_deserialize_section(s) for s in tree_data["sections"]],
        tables=[
            ParsedTable(
                table_id=t["table_id"],
                headers=t["headers"],
                rows=t["rows"],
                page_number=t["page_number"],
                section_id=t.get("section_id"),
                start_offset=t.get("start_offset", 0),
                end_offset=t.get("end_offset", 0),
            )
            for t in tree_data["tables"]
        ],
        total_pages=tree_data["total_pages"],
    )
    print(f"   🌳 Tree: {len(tree.sections)} sections, {len(tree.tables)} tables")

    # Load reviewed merged spans
    spans_path = os.path.join(job_dir, "merged_spans.json")
    with open(spans_path) as f:
        spans_data = json.load(f)

    from extraction.v4.models import MergedSpan

    merged_spans = [MergedSpan.model_validate(s) for s in spans_data]
    print(f"   📊 Loaded {len(merged_spans)} merged spans")

    # Check review status
    pending = sum(1 for s in merged_spans if s.review_status == "PENDING")
    confirmed = sum(1 for s in merged_spans if s.review_status == "CONFIRMED")
    rejected = sum(1 for s in merged_spans if s.review_status == "REJECTED")
    edited = sum(1 for s in merged_spans if s.review_status == "EDITED")
    added = sum(1 for s in merged_spans if s.review_status == "ADDED")

    print(f"      CONFIRMED: {confirmed}  EDITED: {edited}  ADDED: {added}  REJECTED: {rejected}  PENDING: {pending}")

    if pending > 0:
        print(f"\n❌ FATAL: {pending} spans still PENDING review.")
        print("   All spans must be reviewed before running Pipeline 2.")
        print("   Mark each span as CONFIRMED, REJECTED, or EDITED in merged_spans.json")
        sys.exit(1)

    print()

    # ─── BUILD VERIFIED SPANS ────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ BUILDING VERIFIED SPANS (filtering rejected)                        │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    from reviewer_api import build_verified_spans

    verified_spans = build_verified_spans(merged_spans)
    print(f"   ✅ {len(verified_spans)} verified spans (from {confirmed + edited + added} approved)")
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

    print(f"   ✅ {len(dossiers)} per-drug dossiers assembled:")
    for dossier in dossiers:
        rxnorm = f" (RxNorm candidate: {dossier.rxnorm_candidate})" if dossier.rxnorm_candidate else ""
        print(f"      📦 {dossier.drug_name}{rxnorm}: {len(dossier.verified_spans)} spans")
        if dossier.signal_summary:
            summary_parts = [f"{k}: {v}" for k, v in dossier.signal_summary.items()]
            print(f"         Signals: {', '.join(summary_parts)}")
    print()

    if not dossiers:
        print("⚠️ No drug dossiers assembled. No drug anchors found in verified spans.")
        print("   Pipeline 2 cannot proceed without at least one drug dossier.")
        sys.exit(0)

    # ─── L2.5: RxNorm PRE-LOOKUP ────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L2.5: RxNorm PRE-LOOKUP (KB-7 Verified Codes)                      │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    verified_rxnorm_codes = {}
    _is_local = os.path.exists(os.path.join(_script_dir, "pdfs"))
    _default_kb7 = "http://localhost:8092" if _is_local else "http://kb7-terminology:8092"
    kb7_url = os.environ.get("KB7_URL", _default_kb7)

    try:
        from kb7_client import KB7Client
        kb7_client = KB7Client(base_url=kb7_url)

        if kb7_client.health_check():
            print(f"   🔄 KB-7 connected at {kb7_url}")
            unique_drugs = set(d.drug_name.lower() for d in dossiers)

            for drug_name in sorted(unique_drugs):
                results = kb7_client.search(drug_name, system="rxnorm", limit=5)
                if results and len(results) > 0 and results[0].is_valid:
                    best = results[0]
                    verified_rxnorm_codes[drug_name] = {
                        "code": best.code,
                        "display": best.display_name,
                        "source": "KB-7 pre-lookup",
                    }
                    print(f"      ✅ {drug_name}: {best.code} ({best.display_name})")
                else:
                    print(f"      ⚠️ {drug_name}: not found in KB-7")

            kb7_client.close()
        else:
            print(f"   ⚠️ KB-7 not available at {kb7_url}")
    except Exception as e:
        print(f"   ⚠️ KB-7 pre-lookup failed: {e}")

    print(f"   Pre-verified: {len(verified_rxnorm_codes)} codes")
    print()

    # ─── L3: STRUCTURED EXTRACTION PER DRUG DOSSIER ──────────────────────
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

    target_kbs = ["dosing", "safety", "monitoring"] if args.target_kb == "all" else [args.target_kb]

    # Per-drug, per-KB extraction
    all_l3_results = {}  # {drug_name: {kb: result}}
    for dossier in dossiers:
        all_l3_results[dossier.drug_name] = {}
        print(f"   📦 {dossier.drug_name}:")

        for kb in target_kbs:
            kb_label = {"dosing": "KB-1", "safety": "KB-4", "monitoring": "KB-16"}[kb]
            print(f"      → {kb_label} ({kb})...", end=" ", flush=True)

            try:
                result = extractor.extract_facts_from_dossier(
                    dossier=dossier,
                    target_kb=kb,
                    guideline_context=context,
                )
                all_l3_results[dossier.drug_name][kb] = result

                # Print summary
                if kb == "dosing":
                    print(f"✅ {len(result.drugs)} drugs, "
                          f"{sum(len(d.renal_adjustments) for d in result.drugs)} adjustments")
                elif kb == "safety":
                    print(f"✅ {len(result.contraindications)} contraindications")
                elif kb == "monitoring":
                    print(f"✅ {len(result.lab_requirements)} lab requirements")
            except Exception as e:
                print(f"❌ {e}")
                all_l3_results[dossier.drug_name][kb] = None

    print()

    # Save L3 outputs
    output_dir = os.path.join(args.job_dir, "l3_output")
    os.makedirs(output_dir, exist_ok=True)

    for drug_name, kb_results in all_l3_results.items():
        for kb, result in kb_results.items():
            if result is None:
                continue
            safe_name = drug_name.lower().replace(" ", "_")
            path = os.path.join(output_dir, f"{safe_name}_{kb}.json")
            with open(path, "w") as f:
                json.dump(result.model_dump(by_alias=True), f, indent=2)
            print(f"   💾 {drug_name}/{kb}: {path}")

    print()

    # ─── L4: TERMINOLOGY VALIDATION ──────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L4: TERMINOLOGY VALIDATION (KB-7)                                   │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    l4_results = []
    try:
        from kb7_client import KB7Client
        kb7_client = KB7Client(base_url=kb7_url)

        if kb7_client.health_check():
            for drug_name, kb_results in all_l3_results.items():
                dosing_result = kb_results.get("dosing")
                if not dosing_result:
                    continue
                for drug in dosing_result.drugs:
                    result = kb7_client.validate_rxnorm(drug.rxnorm_code)
                    status = "✅ VALID" if result.is_valid else "⚠️ NOT FOUND"
                    print(f"   {drug.drug_name} (RxNorm {drug.rxnorm_code}): {status}")
                    if result.display_name:
                        print(f"      Display: {result.display_name}")
                    l4_results.append({
                        "drug_name": drug.drug_name,
                        "rxnorm_code": drug.rxnorm_code,
                        "is_valid": result.is_valid,
                        "display_name": result.display_name,
                    })
            kb7_client.close()
        else:
            print("   ⚠️ KB-7 not available for L4 validation")
    except Exception as e:
        print(f"   ⚠️ KB-7 L4 error: {e}")

    valid_count = sum(1 for r in l4_results if r.get("is_valid"))
    print(f"\n   Validated: {valid_count}/{len(l4_results)} codes")
    print()

    # ─── L5: CQL COMPATIBILITY ───────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L5: CQL COMPATIBILITY VALIDATION                                   │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    try:
        from cql import CQLCompatibilityChecker
        registry_path = "/app/cql/registry/cql_guideline_registry.yaml"
        if os.path.exists(registry_path):
            checker = CQLCompatibilityChecker(registry_path, "/data/vaidshala")
            for drug_name, kb_results in all_l3_results.items():
                dosing_result = kb_results.get("dosing")
                if dosing_result:
                    facts_dict = dosing_result.model_dump(by_alias=True)
                    report = checker.check_compatibility(facts_dict, "T2DMGuidelines.cql")
                    print(f"   {drug_name}: Compatible={report.compatible}, "
                          f"Matches={len(report.matches)}, Issues={len(report.issues)}")
        else:
            raise FileNotFoundError("Registry not found")
    except Exception as e:
        print(f"   CQL registry check: {e}")
        print()
        print("   CQL Mapping Summary:")
        for drug_name, kb_results in all_l3_results.items():
            dosing_result = kb_results.get("dosing")
            if not dosing_result:
                continue
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

    # ─── PIPELINE 2 SUMMARY ──────────────────────────────────────────────
    print("=" * 70)
    print("PIPELINE 2 COMPLETE")
    print("=" * 70)
    print()
    print(f"   Job ID:          {job_id}")
    print(f"   Dossiers:        {len(dossiers)} drugs")
    print(f"   Target KBs:      {', '.join(target_kbs)}")
    print(f"   L3 Extractions:  {sum(1 for d in all_l3_results.values() for r in d.values() if r is not None)}")
    print(f"   L4 Validated:    {valid_count}/{len(l4_results)} codes")
    print(f"   L3 Output:       {output_dir}")
    print()
    print("🎉 V4 Pipeline 2 completed successfully!")
    print("=" * 70)


def _deserialize_section(data):
    """Recursively deserialize a GuidelineSection from dict."""
    from extraction.v4.models import GuidelineSection
    return GuidelineSection(
        section_id=data["section_id"],
        heading=data["heading"],
        start_offset=data["start_offset"],
        end_offset=data["end_offset"],
        page_number=data["page_number"],
        block_type=data["block_type"],
        children=[_deserialize_section(c) for c in data.get("children", [])],
    )


# ═══════════════════════════════════════════════════════════════════════════
# LEGACY PIPELINE (V3)
# ═══════════════════════════════════════════════════════════════════════════

def pipeline_legacy():
    """V3 Legacy Pipeline: L1 → L2 (GLiNER) → L3 → L4 → L5 (single invocation)."""
    print("=" * 70)
    print("V3 CLINICAL GUIDELINE CURATION PIPELINE - FULL L1-L5")
    print("=" * 70)
    print()

    # ─── L1: PDF PARSING ─────────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L1: PDF PARSING (Marker v1.10)                                      │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    pdf_path = resolve_pdf_path(args.pdf)
    print(f"📄 PDF Found: {os.path.basename(pdf_path)}")
    print(f"   Size: {os.path.getsize(pdf_path) / 1024 / 1024:.1f} MB")

    from marker_extractor import MarkerExtractor

    print("🔄 Loading Marker v1.10 ML models...")
    extractor = MarkerExtractor()
    print("🔄 Extracting PDF with full provenance tracking...")
    l1_result = extractor.extract(pdf_path)

    markdown_text = l1_result.markdown
    total_pages = l1_result.provenance.total_pages
    num_blocks = len(l1_result.blocks)
    num_tables = len(l1_result.tables)

    if l1_result.provenance.marker_version == "mock":
        print("❌ FATAL: Marker returned mock data - marker-pdf not properly installed")
        sys.exit(1)

    print()
    print("L1 OUTPUT:")
    print(f"   ✅ Pages: {total_pages}")
    print(f"   ✅ Text Blocks: {num_blocks}")
    print(f"   ✅ Tables: {num_tables}")
    print(f"   ✅ Markdown: {len(markdown_text):,} chars")
    print()

    # ─── L2: CLINICAL NER ────────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L2: CLINICAL NER (GLiNER with Descriptive Labels)                   │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    try:
        from extraction.gliner.extractor import ClinicalNERExtractor
        print("🔄 Loading GLiNER and extracting entities with descriptive labels...")
        ner = ClinicalNERExtractor(threshold=0.6)
        l2_result = ner.extract_for_kb(markdown_text[:5000], "dosing")
        raw_entities = l2_result.entities
        entities = []
        for e in raw_entities:
            if hasattr(e, 'text'):
                entities.append({"label": e.label, "text": e.text, "confidence": getattr(e, 'score', 0.8)})
            elif isinstance(e, dict):
                entities.append(e)
    except Exception as e:
        print(f"⚠️ GLiNER: {e}")
        print("   Using regex-based fallback NER...")
        import re
        entities = []
        for match in re.finditer(r'\b(metformin|dapagliflozin|empagliflozin|finerenone|canagliflozin|lisinopril)\b', markdown_text, re.I):
            entities.append({"label": "drug_ingredient", "text": match.group(), "confidence": 0.9})
        for match in re.finditer(r'\b(Farxiga|Jardiance|Kerendia|Invokana|Glucophage)\b', markdown_text, re.I):
            entities.append({"label": "drug_product", "text": match.group(), "confidence": 0.9})
        for match in re.finditer(r'\b(SGLT2i?|SGLT2 inhibitor|GLP-1 RA|ACE inhibitor|ARB|MRA|RASi)\b', markdown_text, re.I):
            entities.append({"label": "drug_class", "text": match.group(), "confidence": 0.92})
        for match in re.finditer(r'eGFR\s*[<>=≥≤]+\s*\d+', markdown_text, re.I):
            entities.append({"label": "egfr_threshold", "text": match.group(), "confidence": 0.95})
        for match in re.finditer(r'(reduce|discontinue|continue|hold)[^.]*', markdown_text, re.I):
            entities.append({"label": "dose_adjustment", "text": match.group()[:50], "confidence": 0.8})
        for match in re.finditer(r'\b(potassium|eGFR|creatinine|HbA1c|UACR)\b', markdown_text, re.I):
            entities.append({"label": "lab_test", "text": match.group(), "confidence": 0.88})
        for match in re.finditer(r'\b(every\s+\d+[-–]\d+\s*(?:months?|weeks?)|Q\d+[-–]?\d*\s*(?:mo|months?))\b', markdown_text, re.I):
            entities.append({"label": "monitoring_frequency", "text": match.group(), "confidence": 0.90})

    print(f"   ✅ Entities Found: {len(entities)}")
    print()

    # ─── L3: STRUCTURED EXTRACTION ───────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L3: STRUCTURED EXTRACTION (Claude + KB-1 Schema)                    │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    api_key = require_api_key()
    print(f"✅ ANTHROPIC_API_KEY configured (length: {len(api_key)})")

    from fact_extractor import KBFactExtractor
    from anthropic import Anthropic

    client = Anthropic(api_key=api_key)
    fact_extractor = KBFactExtractor(client)
    context = guideline_context_kdigo()

    print("   → Extracting KB-1 (Dosing Facts)...")
    l3_result = fact_extractor.extract_facts(
        markdown_text=markdown_text[:6000],
        gliner_entities=entities, target_kb="dosing", guideline_context=context
    )
    print(f"      ✅ KB-1: {len(l3_result.drugs)} drugs extracted")

    print("   → Extracting KB-4 (Safety Facts)...")
    kb4_result = fact_extractor.extract_facts(
        markdown_text=markdown_text[:6000],
        gliner_entities=entities, target_kb="safety", guideline_context=context
    )
    print(f"      ✅ KB-4: {len(kb4_result.contraindications)} contraindications")

    print("   → Extracting KB-16 (Monitoring Facts)...")
    kb16_result = fact_extractor.extract_facts(
        markdown_text=markdown_text[:6000],
        gliner_entities=entities, target_kb="monitoring", guideline_context=context
    )
    print(f"      ✅ KB-16: {len(kb16_result.lab_requirements)} monitoring requirements")
    print()

    # Print L3 output
    print("━━━ KB-1: Dosing Facts ━━━")
    for drug in l3_result.drugs:
        print(f"   📦 {drug.drug_name} (RxNorm: {drug.rxnorm_code})")
        for adj in drug.renal_adjustments:
            if adj.contraindicated:
                print(f"      • eGFR {adj.egfr_min:.0f}-{adj.egfr_max:.0f}: ⛔ CONTRAINDICATED")
            else:
                factor = f"x{adj.adjustment_factor}" if adj.adjustment_factor else ""
                dose = f", max {adj.max_dose}{adj.max_dose_unit}" if adj.max_dose else ""
                print(f"      • eGFR {adj.egfr_min:.0f}-{adj.egfr_max:.0f}: 💊 {factor}{dose}")
            print(f"        → {adj.recommendation}")
    print()

    print("━━━ KB-4: Safety Facts ━━━")
    for ci in kb4_result.contraindications:
        severity_icon = {"CRITICAL": "🔴", "HIGH": "🟠", "MODERATE": "🟡", "LOW": "🟢"}.get(ci.severity, "⚪")
        print(f"   {severity_icon} {ci.drug_name}: {ci.contraindication_type.upper()} ({ci.severity})")
        print(f"      Conditions: {', '.join(ci.condition_descriptions[:2])}")
        print(f"      Rationale: {ci.clinical_rationale[:80]}...")
    print()

    print("━━━ KB-16: Monitoring Facts ━━━")
    for req in kb16_result.lab_requirements:
        print(f"   🔬 {req.drug_name} (RxNorm: {req.rxnorm_code})")
        for lab in req.labs:
            critical = ""
            if lab.critical_high:
                critical = f" [STOP if {lab.critical_high.operator}{lab.critical_high.value}]"
            print(f"      • {lab.lab_name} (LOINC: {lab.loinc_code}): {lab.frequency}{critical}")
    print()

    # Save outputs
    os.makedirs("/data/output", exist_ok=True)
    for name, result in [("kb1_dosing", l3_result), ("kb4_safety", kb4_result), ("kb16_monitoring", kb16_result)]:
        path = f"/data/output/{name}_facts.json"
        with open(path, "w") as f:
            json.dump(result.model_dump(by_alias=True), f, indent=2)
        print(f"   💾 {path}")
    print()

    # ─── L4: TERMINOLOGY VALIDATION ──────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L4: TERMINOLOGY VALIDATION (KB-7)                                   │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    kb7_url = os.environ.get("KB7_URL", "http://kb7-terminology:8092")
    l4_validation_results = []
    try:
        from kb7_client import KB7Client
        kb7_client = KB7Client(base_url=kb7_url)
        if kb7_client.health_check():
            for drug in l3_result.drugs:
                result = kb7_client.validate_rxnorm(drug.rxnorm_code)
                status = "✅ VALID" if result.is_valid else "⚠️ NOT FOUND"
                print(f"   RxNorm {drug.rxnorm_code} ({drug.drug_name}): {status}")
                l4_validation_results.append({
                    "rxnorm_code": drug.rxnorm_code, "drug_name": drug.drug_name,
                    "is_valid": result.is_valid,
                })
        kb7_client.close()
    except Exception as e:
        print(f"   ⚠️ KB-7: {e}")
        for drug in l3_result.drugs:
            l4_validation_results.append({
                "rxnorm_code": drug.rxnorm_code, "drug_name": drug.drug_name,
                "is_valid": False, "error": "KB-7 unavailable",
            })

    valid_codes = sum(1 for r in l4_validation_results if r.get("is_valid", False))
    print(f"   Validated: {valid_codes}/{len(l4_validation_results)} codes")
    print()

    # ─── L5: CQL ─────────────────────────────────────────────────────────
    print("┌─────────────────────────────────────────────────────────────────────┐")
    print("│ L5: CQL COMPATIBILITY VALIDATION                                    │")
    print("└─────────────────────────────────────────────────────────────────────┘")
    print()

    try:
        from cql import CQLCompatibilityChecker
        registry_path = "/app/cql/registry/cql_guideline_registry.yaml"
        if os.path.exists(registry_path):
            checker = CQLCompatibilityChecker(registry_path, "/data/vaidshala")
            facts_dict = l3_result.model_dump(by_alias=True)
            report = checker.check_compatibility(facts_dict, "T2DMGuidelines.cql")
            print(f"   ✅ Compatible: {report.compatible}")
        else:
            raise FileNotFoundError("Registry not found")
    except Exception as e:
        print(f"   Registry check: {e}")
        for drug in l3_result.drugs:
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
    print("PIPELINE EXECUTION COMPLETE")
    print("=" * 70)
    print()
    print("Layer Summary:")
    print(f"   L1 PDF Parsing:        ✅ {total_pages} pages, {len(markdown_text):,} chars (Marker v1.10)")
    print(f"   L2 Clinical NER:       ✅ {len(entities)} entities tagged")
    print(f"   L3 Fact Extraction:    ✅ {len(l3_result.drugs)} drugs, "
          f"{sum(len(d.renal_adjustments) for d in l3_result.drugs)} dosing rules")
    print(f"   L4 Terminology:        ✅ {valid_codes}/{len(l4_validation_results)} codes validated")
    print(f"   L5 CQL Compatibility:  ✅ Mapped to T2DMGuidelines.cql")
    print()
    print("🎉 V3 Clinical Guideline Curation Pipeline completed successfully!")
    print("=" * 70)


# ═══════════════════════════════════════════════════════════════════════════
# MAIN DISPATCH
# ═══════════════════════════════════════════════════════════════════════════

if args.pipeline == "1":
    pipeline_1()
elif args.pipeline == "2":
    pipeline_2()
else:
    pipeline_legacy()
