#!/usr/bin/env python3
"""Re-merge raw spans using the fixed SignalMerger (V4.2.4).

Reads existing raw_spans.json + guideline_tree.json from a Pipeline 1 job
output directory, re-runs the Signal Merger with the page-mapping fix,
and overwrites merged_spans.json, section_passages.json, and
shadow_classifier_log.json.

Usage:
    python remerge_spans.py <job_dir>

Example:
    python remerge_spans.py output/v4/job_monkeyocr_f172f6a9-7733-4352-a0aa-43707fdb46c8
"""

import json
import sys
import os
from pathlib import Path
from uuid import UUID
from collections import Counter

# Add the extraction module to path
# _this_dir = data/   →  _parent = guideline-atomiser/  →  _shared = shared/
_this_dir = Path(__file__).resolve().parent
_parent_dir = _this_dir.parent                            # guideline-atomiser/
_shared_dir = _parent_dir.parent.parent                   # shared/ (contains extraction/)
sys.path.insert(0, str(_shared_dir))
sys.path.insert(0, str(_parent_dir))

from extraction.v4.models import (
    RawSpan, ChannelOutput, GuidelineTree, GuidelineSection, TableBoundary,
    MergedSpan,
)
from extraction.v4.signal_merger import SignalMerger
from extraction.v4.tiering_classifier import (
    RuleBasedTieringClassifier, ShadowTieringClassifier,
)


def load_guideline_tree(tree_path: Path) -> GuidelineTree:
    """Load GuidelineTree from JSON."""
    with open(tree_path) as f:
        data = json.load(f)

    def parse_section(d: dict) -> GuidelineSection:
        children = [parse_section(c) for c in d.get("children", [])]
        return GuidelineSection(
            section_id=d["section_id"],
            heading=d["heading"],
            start_offset=d["start_offset"],
            end_offset=d["end_offset"],
            page_number=d["page_number"],
            block_type=d.get("block_type", "heading"),
            level=d.get("level", 1),
            children=children,
        )

    sections = [parse_section(s) for s in data.get("sections", [])]

    tables = []
    for t in data.get("tables", []):
        tables.append(TableBoundary(
            table_id=t["table_id"],
            section_id=t.get("section_id", ""),
            start_offset=t.get("start_offset", -1),
            end_offset=t.get("end_offset", -1),
            headers=t.get("headers", []),
            row_count=t.get("row_count", 0),
            page_number=t.get("page_number", 0),
            source=t.get("source", "marker_pipe"),
            otsl_text=t.get("otsl_text"),
        ))

    # page_map: JSON keys are strings, convert to int→int
    raw_page_map = data.get("page_map", {})
    page_map = {int(k): int(v) for k, v in raw_page_map.items()}

    tree = GuidelineTree(
        sections=sections,
        tables=tables,
        total_pages=data.get("total_pages", 0),
        alignment_confidence=data.get("alignment_confidence", 1.0),
        structural_source=data.get("structural_source", "granite_doctags"),
        page_map=page_map,
    )
    return tree


def load_raw_spans(raw_path: Path) -> tuple[list[RawSpan], dict[str, list[RawSpan]]]:
    """Load RawSpans from JSON. Returns (all_spans, by_channel)."""
    with open(raw_path) as f:
        data = json.load(f)

    # Structure: {channel: {count, error, spans: [...]}}
    by_channel: dict[str, list[RawSpan]] = {}
    all_spans: list[RawSpan] = []

    # Get valid field names for RawSpan to filter extras like 'bbox'
    valid_fields = set(RawSpan.model_fields.keys())

    for ch, ch_data in data.items():
        ch_spans = []
        for d in ch_data.get("spans", []):
            filtered = {k: v for k, v in d.items() if k in valid_fields}
            span = RawSpan(**filtered)
            ch_spans.append(span)
        by_channel[ch] = ch_spans
        all_spans.extend(ch_spans)

    return all_spans, by_channel


def main():
    if len(sys.argv) < 2:
        print("Usage: python remerge_spans.py <job_dir>")
        sys.exit(1)

    job_dir = Path(sys.argv[1])
    if not job_dir.is_absolute():
        job_dir = _this_dir / sys.argv[1]

    if not job_dir.exists():
        print(f"ERROR: Job directory not found: {job_dir}")
        sys.exit(1)

    raw_path = job_dir / "raw_spans.json"
    tree_path = job_dir / "guideline_tree.json"
    meta_path = job_dir / "job_metadata.json"
    text_path = job_dir / "normalized_text.txt"

    for p in [raw_path, tree_path, meta_path]:
        if not p.exists():
            print(f"ERROR: Required file missing: {p}")
            sys.exit(1)

    # Load metadata
    with open(meta_path) as f:
        metadata = json.load(f)
    job_id = UUID(metadata["job_id"])

    print(f"Re-merging job: {job_id}")
    print(f"  Source: {metadata.get('source_pdf', 'unknown')}")

    # Load inputs
    print("  Loading guideline tree...")
    tree = load_guideline_tree(tree_path)
    print(f"    {len(tree.sections)} top-level sections, {len(tree.tables)} tables")

    print("  Loading raw spans...")
    all_raw, by_channel = load_raw_spans(raw_path)
    print(f"    {len(all_raw)} raw spans")

    channel_outputs = []
    for ch, spans in sorted(by_channel.items()):
        channel_outputs.append(ChannelOutput(
            channel=ch,
            spans=spans,
        ))
        print(f"    Channel {ch}: {len(spans)} spans")

    # Set up classifier (shadow mode with model if available)
    model_path = _this_dir / "models" / "tier_assigner.joblib"
    if model_path.exists():
        print(f"  Classifier: SHADOW (model: {model_path.name})")
        classifier = ShadowTieringClassifier(str(model_path))
    else:
        print("  Classifier: RULE-BASED (no model found)")
        classifier = RuleBasedTieringClassifier()

    # Run merge
    print("\n  Running Signal Merger V4.2.4...")
    merger = SignalMerger()
    merged = merger.merge(job_id, channel_outputs, tree, classifier)
    print(f"  Result: {len(merged)} merged spans")

    # Page distribution
    page_counts = Counter(s.page_number for s in merged)
    page1_count = page_counts.get(1, 0)
    none_count = page_counts.get(None, 0)
    print(f"\n  Page distribution:")
    print(f"    Page 1: {page1_count} spans ({page1_count*100/len(merged):.1f}%)")
    if none_count:
        print(f"    Page None: {none_count} spans")
    print(f"    Pages with spans: {len([p for p in page_counts if p and p > 0])}")
    neg_start = sum(1 for s in merged if s.start < 0)
    print(f"    Spans with start=-1: {neg_start}")

    # Save merged spans
    merged_path = job_dir / "merged_spans.json"
    merged_dicts = [s.model_dump(mode="json") for s in merged]
    with open(merged_path, "w") as f:
        json.dump(merged_dicts, f, indent=2, default=str)
    print(f"\n  Wrote: {merged_path.name} ({len(merged_dicts)} spans)")

    # Build section passages
    if text_path.exists():
        print("  Building section passages...")
        text = text_path.read_text()
        passages = merger.assemble_section_passages(merged, tree, text)
        passages_dicts = []
        for p in passages:
            passages_dicts.append({
                "section_id": p.section_id,
                "heading": p.heading,
                "page_number": p.page_number,
                "prose_text": p.prose_text[:200] + "..." if len(p.prose_text) > 200 else p.prose_text,
                "span_ids": [str(sid) for sid in p.span_ids],
                "span_count": p.span_count,
                "child_section_ids": p.child_section_ids,
                "start_offset": p.start_offset,
                "end_offset": p.end_offset,
            })
        passages_path = job_dir / "section_passages.json"
        with open(passages_path, "w") as f:
            json.dump(passages_dicts, f, indent=2, default=str)
        print(f"  Wrote: {passages_path.name} ({len(passages_dicts)} passages)")

    # Save shadow log if applicable
    if hasattr(classifier, 'shadow_log'):
        shadow_log = classifier.shadow_log
        if shadow_log:
            shadow_path = job_dir / "shadow_classifier_log.json"
            with open(shadow_path, "w") as f:
                json.dump(shadow_log, f, indent=2, default=str)
            print(f"  Wrote: {shadow_path.name} ({len(shadow_log)} entries)")

    # Update metadata
    metadata["pipeline_version"] = "4.2.4"
    metadata["total_merged_spans"] = len(merged)
    metadata["remerge_note"] = "Re-merged with V4.2.4 page mapping fix"
    with open(meta_path, "w") as f:
        json.dump(metadata, f, indent=2, default=str)
    print(f"  Updated: {meta_path.name}")

    print("\nDone!")


if __name__ == "__main__":
    main()
