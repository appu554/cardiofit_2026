"""V5 Bbox Provenance metrics sidecar.

Reads MergedSpan records (raw dicts) from a job output directory and computes
V5 bbox provenance coverage metrics. Designed to run standalone with no
dependencies on the V4 extraction modules — operates purely on JSON dicts.

Usage:
    Imported:
        from v5_metrics import compute_v5_bbox_metrics, write_v5_metrics

    Script:
        python3 v5_metrics.py <job_dir>
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
            existing = {}
    else:
        existing = {}
    existing.update(metrics)
    out_path.write_text(json.dumps(existing, indent=2, sort_keys=True))


def _main(argv: list[str]) -> int:
    if len(argv) != 2:
        print("usage: python3 v5_metrics.py <job_dir>", file=sys.stderr)
        return 2
    job_dir = Path(argv[1])
    spans_path = job_dir / "merged_spans.json"
    if not spans_path.exists():
        print(f"error: {spans_path} not found", file=sys.stderr)
        return 1
    spans = json.loads(spans_path.read_text())
    metrics = compute_v5_bbox_metrics(spans)
    write_v5_metrics(job_dir, metrics)
    bp = metrics["v5_bbox_provenance"]
    print(
        f"v5_bbox_provenance: total={bp['total_spans']} "
        f"with_prov={bp['spans_with_provenance']} "
        f"coverage={bp['bbox_coverage_pct']:.2f}% "
        f"channels={bp['channels_seen']} "
        f"multi={bp['spans_multi_channel']}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(_main(sys.argv))
