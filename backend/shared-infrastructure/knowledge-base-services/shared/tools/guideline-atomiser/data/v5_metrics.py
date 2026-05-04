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


_BBOX_COVERAGE_THRESHOLD = 99.0


def compute_v5_bbox_metrics(merged_spans: list[dict[str, Any]]) -> dict[str, Any]:
    """Compute V5 bbox provenance metrics from a list of MergedSpan dicts.

    Pure function. No I/O. Treats missing or empty `channel_provenance`
    identically (both count as "no provenance").

    Returns a dict with three top-level keys matching the spec §7 sidecar
    metrics.json shape:
      - ``v5_bbox_provenance``: raw counts and coverage
      - ``primary``: per-metric status dicts with threshold and verdict
      - ``verdict``: "PASS" or "FAIL" aggregate
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
        coverage_pct = round((spans_with_provenance / total_spans) * 100.0, 2)
    else:
        coverage_pct = 0.0

    primary_status = "PASS" if coverage_pct >= _BBOX_COVERAGE_THRESHOLD else "FAIL"

    return {
        "v5_bbox_provenance": {
            "total_spans": total_spans,
            "spans_with_provenance": spans_with_provenance,
            "bbox_coverage_pct": coverage_pct,
            "channels_seen": sorted(channels),
            "spans_multi_channel": spans_multi_channel,
        },
        "primary": {
            "bbox_coverage_pct": {
                "v5": coverage_pct,
                "threshold": _BBOX_COVERAGE_THRESHOLD,
                "status": primary_status,
            },
        },
        "verdict": primary_status,
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


_TABLE_CELL_ACCURACY_THRESHOLD = 85.0


def compute_v5_table_metrics(
    merged_spans: list[dict[str, Any]],
    ground_truth_csvs: list[str] | None = None,
) -> dict[str, Any]:
    """Compute V5 Table Specialist metrics.

    Without ground_truth_csvs: reports only structural metadata (docling_table_cell_spans,
    total table-cell spans, % from docling path).

    With ground_truth_csvs: also computes cell accuracy against golden CSVs.
    Each CSV must have columns: row_idx, col_idx, expected_text.
    """
    import csv as _csv

    def _has_provenance_version(span: dict, version: str) -> bool:
        return any(
            isinstance(p, dict) and p.get("model_version") == version
            for p in span.get("channel_provenance", [])
        )

    # MergedSpan stores origin in channel_provenance.model_version, not channel_metadata.
    # table@v1.0  → Docling TableBlock path (V5 Table Specialist)
    # pipe-table@v1.0 → V4 pipe/OTSL path
    total_table_spans = len([
        s for s in merged_spans
        if isinstance(s, dict)
        and any(
            isinstance(p, dict) and "table" in p.get("model_version", "")
            for p in s.get("channel_provenance", [])
        )
    ])

    docling_table_spans = len([
        s for s in merged_spans
        if isinstance(s, dict) and _has_provenance_version(s, "table@v1.0")
    ])

    accuracy_pct: float | None = None
    if ground_truth_csvs:
        per_table_accuracy: list[float] = []

        for csv_path in ground_truth_csvs:
            golden: dict[tuple[int, int], str] = {}
            try:
                with open(csv_path) as f:
                    for row in _csv.DictReader(f):
                        golden[(int(row["row_idx"]), int(row["col_idx"]))] = row["expected_text"]
            except (OSError, KeyError, ValueError):
                continue

            extracted: dict[tuple[int, int], str] = {}
            for s in merged_spans:
                if not isinstance(s, dict):
                    continue
                cm = s.get("channel_metadata", {})
                if not isinstance(cm, dict):
                    continue
                if cm.get("table_source") == "docling_tableblock":
                    r = cm.get("row_index", -1)
                    c = cm.get("col_index", -1)
                    extracted[(r, c)] = (s.get("text") or "").strip()

            if golden:
                correct = sum(
                    1 for (r, c), expected in golden.items()
                    if extracted.get((r, c), "").lower() == expected.lower()
                )
                per_table_accuracy.append(correct / len(golden) * 100.0)

        accuracy_pct = (
            round(sum(per_table_accuracy) / len(per_table_accuracy), 2)
            if per_table_accuracy else 0.0
        )

    primary_status: str | None = None
    if accuracy_pct is not None:
        primary_status = "PASS" if accuracy_pct >= _TABLE_CELL_ACCURACY_THRESHOLD else "FAIL"

    result: dict[str, Any] = {
        "v5_table_specialist": {
            "total_table_cell_spans": total_table_spans,
            "docling_table_cell_spans": docling_table_spans,
            "docling_coverage_pct": round(
                docling_table_spans / total_table_spans * 100.0, 2
            ) if total_table_spans > 0 else 0.0,
        },
    }
    if accuracy_pct is not None:
        result["v5_table_specialist"]["cell_accuracy_pct"] = accuracy_pct
        result.setdefault("primary", {})["table_cell_accuracy_pct"] = {
            "v5": accuracy_pct,
            "threshold": _TABLE_CELL_ACCURACY_THRESHOLD,
            "status": primary_status,
        }
        result["verdict_table"] = primary_status

    return result


def compute_v5_ce_metrics(merged_spans: list[dict[str, Any]]) -> dict[str, Any]:
    """Compute V5 Consensus Entropy gate metrics.

    Pure function. No I/O. Reads ce_flagged from MergedSpan dicts.
    """
    total_spans = len(merged_spans)
    flagged = sum(1 for s in merged_spans if s.get("ce_flagged") is True)
    suppression_pct = round(flagged / total_spans * 100.0, 2) if total_spans > 0 else 0.0

    return {
        "v5_ce_gate": {
            "total_spans": total_spans,
            "ce_flagged_spans": flagged,
            "suppression_pct": suppression_pct,
        },
    }


_DECOMP_EDGE_THRESHOLD = 80.0


def compute_v5_decomposition_metrics(
    graph_dict: dict[str, Any],
    ground_truth_yaml: str | None = None,
) -> dict[str, Any]:
    """Compute V5 Decomposition metrics (edge precision + recall).

    graph_dict: dict from GuidelineGraph.to_dict() (loaded from graph.json).
    ground_truth_yaml: optional path to YAML with a 'relationships' list, each
        entry having source_node_id, target_node_id, edge_type.

    Without ground_truth_yaml: reports structural metadata only.
    With ground_truth_yaml: computes precision/recall against graded edges.
    """
    import yaml as _yaml

    extracted_edges = graph_dict.get("edges", [])
    extracted_set = {
        (e["source_node_id"], e["target_node_id"], e["edge_type"])
        for e in extracted_edges
        if isinstance(e, dict)
    }

    node_count = graph_dict.get("node_count", len(graph_dict.get("nodes", [])))
    edge_count = graph_dict.get("edge_count", len(extracted_edges))

    if ground_truth_yaml is None:
        return {
            "v5_decomposition": {
                "node_count": node_count,
                "edge_count": edge_count,
            }
        }

    try:
        with open(ground_truth_yaml) as f:
            data = _yaml.safe_load(f)
        graded = data.get("relationships", []) if isinstance(data, dict) else []
    except (OSError, Exception):
        graded = []

    graded_set = {
        (r["source_node_id"], r["target_node_id"], r["edge_type"])
        for r in graded
        if isinstance(r, dict)
    }

    if not graded_set:
        return {
            "v5_decomposition": {
                "node_count": node_count,
                "edge_count": edge_count,
                "error": "empty or unreadable ground truth",
            }
        }

    true_positives = extracted_set & graded_set
    precision = round(len(true_positives) / len(extracted_set) * 100.0, 2) if extracted_set else 0.0
    recall = round(len(true_positives) / len(graded_set) * 100.0, 2)

    p_status = "PASS" if precision >= _DECOMP_EDGE_THRESHOLD else "FAIL"
    r_status = "PASS" if recall >= _DECOMP_EDGE_THRESHOLD else "FAIL"
    overall = "PASS" if p_status == "PASS" and r_status == "PASS" else "FAIL"

    return {
        "v5_decomposition": {
            "node_count": node_count,
            "edge_count": edge_count,
            "graded_edge_count": len(graded_set),
            "true_positive_edges": len(true_positives),
            "edge_precision_pct": precision,
            "edge_recall_pct": recall,
        },
        "primary": {
            "edge_precision_pct": {
                "v5": precision,
                "threshold": _DECOMP_EDGE_THRESHOLD,
                "status": p_status,
            },
            "edge_recall_pct": {
                "v5": recall,
                "threshold": _DECOMP_EDGE_THRESHOLD,
                "status": r_status,
            },
        },
        "verdict": overall,
    }


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
    args = parser.parse_args(argv[1:] if argv is not None else None)

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
        table_metrics = compute_v5_table_metrics(spans)
        if table_metrics["v5_table_specialist"]["docling_table_cell_spans"] > 0:
            write_v5_metrics(job_dir, table_metrics)
            ts = table_metrics["v5_table_specialist"]
            print(
                f"{job_dir.name}: docling_table_spans={ts['docling_table_cell_spans']} "
                f"({ts['docling_coverage_pct']:.1f}% of table cells)"
            )
        # CE gate metrics are always written (zero flagged is still a valid audit record).
        ce_metrics = compute_v5_ce_metrics(spans)
        write_v5_metrics(job_dir, ce_metrics)
        ceg = ce_metrics["v5_ce_gate"]
        print(
            f"{job_dir.name}: ce_flagged={ceg['ce_flagged_spans']} "
            f"({ceg['suppression_pct']:.1f}% suppressed)"
        )
        graph_path = job_dir / "graph.json"
        if graph_path.exists():
            graph_dict = json.loads(graph_path.read_text(encoding="utf-8"))
            decomp_metrics = compute_v5_decomposition_metrics(graph_dict)
            write_v5_metrics(job_dir, decomp_metrics)
            dm = decomp_metrics["v5_decomposition"]
            print(f"{job_dir.name}: decomp nodes={dm['node_count']}, edges={dm['edge_count']}")
    return rc


if __name__ == "__main__":
    sys.exit(_main(sys.argv))
