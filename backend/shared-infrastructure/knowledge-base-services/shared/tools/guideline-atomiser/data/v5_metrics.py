"""V5 Bbox Provenance metrics sidecar.

Reads MergedSpan records (raw dicts) from a job output directory and computes
V5 bbox provenance coverage metrics. Designed to run standalone with no
dependencies on the V4 extraction modules — operates purely on JSON dicts.

Usage:
    Imported:
        from v5_metrics import compute_v5_bbox_metrics, write_v5_metrics

    Script:
        python3 v5_metrics.py <job_dir> [<job_dir> ...]
"""
from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any


def compute_v5_bbox_metrics(merged_spans: list[dict[str, Any]]) -> dict[str, Any]:
    """Compute V5 bbox provenance metrics from a list of MergedSpan dicts.

    Pure function. No I/O. Treats missing or empty `channel_provenance`
    identically (both count as "no provenance").
    """
    total_spans = len(merged_spans)
    spans_with_provenance = 0
    spans_multi_channel = 0
    channels: set[str] = set()

    for span in merged_spans:
        prov = span.get("channel_provenance") or []
        if not prov:
            continue
        spans_with_provenance += 1
        span_channels: set[str] = set()
        for entry in prov:
            cid = entry.get("channel_id")
            if cid is not None:
                channels.add(cid)
                span_channels.add(cid)
        if len(span_channels) >= 2:
            spans_multi_channel += 1

    if total_spans > 0:
        coverage_pct = (spans_with_provenance / total_spans) * 100.0
    else:
        coverage_pct = 0.0

    return {
        "v5_bbox_provenance": {
            "total_spans": total_spans,
            "spans_with_provenance": spans_with_provenance,
            "bbox_coverage_pct": coverage_pct,
            "channels_seen": sorted(channels),
            "spans_multi_channel": spans_multi_channel,
        }
    }


def write_v5_metrics(job_dir: Path, metrics: dict[str, Any]) -> None:
    """Merge metrics into job_dir/metrics.json (creates if absent).

    Existing top-level keys are preserved; provided keys overwrite matching
    top-level entries.
    """
    job_dir = Path(job_dir)
    job_dir.mkdir(parents=True, exist_ok=True)
    out_path = job_dir / "metrics.json"
    if out_path.exists():
        try:
            existing = json.loads(out_path.read_text())
            if not isinstance(existing, dict):
                existing = {}
        except (json.JSONDecodeError, OSError):
            print(f"[v5_metrics] Warning: could not read existing {out_path}, starting fresh", file=sys.stderr)
            existing = {}
    else:
        existing = {}
    existing.update(metrics)
    merged = existing
    tmp_path = out_path.with_suffix(".json.tmp")
    tmp_path.write_text(json.dumps(merged, indent=2), encoding="utf-8")
    tmp_path.replace(out_path)


def _main(argv: list[str] | None = None) -> int:
    import argparse

    parser = argparse.ArgumentParser(
        description="Compute V5 bbox provenance metrics for one or more job dirs."
    )
    parser.add_argument(
        "job_dirs",
        nargs="+",
        metavar="job_dir",
        help="Pipeline 1 output directory (one or more)",
    )
    args = parser.parse_args(argv if argv is None else argv[1:])

    rc = 0
    for job_dir_str in args.job_dirs:
        job_dir = Path(job_dir_str)
        spans_path = job_dir / "merged_spans.json"
        if not spans_path.exists():
            print(f"[v5_metrics] ERROR: {spans_path} not found", file=sys.stderr)
            rc = 1
            continue
        spans = json.loads(spans_path.read_text(encoding="utf-8"))
        metrics = compute_v5_bbox_metrics(spans)
        write_v5_metrics(job_dir, metrics)
        bp = metrics["v5_bbox_provenance"]
        print(
            f"{job_dir.name}: bbox_coverage_pct={bp['bbox_coverage_pct']:.2f}% "
            f"({bp['total_spans']} spans)"
        )
    return rc


if __name__ == "__main__":
    sys.exit(_main(sys.argv))
