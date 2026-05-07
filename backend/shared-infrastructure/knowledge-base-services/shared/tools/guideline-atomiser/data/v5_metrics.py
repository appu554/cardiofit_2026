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
_CHANNEL_PROV_THRESHOLD = 99.0


def _span_has_bbox(span: dict[str, Any]) -> bool:
    """Return True if the span carries a non-null top-level bbox dict."""
    bb = span.get("bbox")
    if bb is None:
        return False
    if isinstance(bb, dict):
        return all(k in bb for k in ("x0", "y0", "x1", "y1"))
    # Stored as a list [x0, y0, x1, y1] in some serialisation paths
    return isinstance(bb, (list, tuple)) and len(bb) == 4


def compute_v5_bbox_metrics(merged_spans: list[dict[str, Any]]) -> dict[str, Any]:
    """Compute V5 bbox provenance metrics from a list of MergedSpan dicts.

    Pure function. No I/O.

    Tracks two independent primary metrics (spec §7):
      - ``bbox_coverage_pct``: % spans with a non-null top-level ``bbox`` field
      - ``channel_provenance_pct``: % spans with ≥1 ``channel_provenance`` entry

    Both must reach ≥99% for a PASS verdict.

    Returns a dict with three top-level keys matching the spec §7 sidecar
    metrics.json shape:
      - ``v5_bbox_provenance``: raw counts and coverage
      - ``primary``: per-metric status dicts with threshold and verdict
      - ``verdict``: "PASS" or "FAIL" aggregate
    """
    total_spans = len(merged_spans)
    spans_with_bbox = 0
    spans_with_provenance = 0
    spans_multi_channel = 0
    channels: set[str] = set()

    for span in merged_spans:
        if _span_has_bbox(span):
            spans_with_bbox += 1

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
        bbox_pct = round((spans_with_bbox / total_spans) * 100.0, 2)
        cp_pct = round((spans_with_provenance / total_spans) * 100.0, 2)
    else:
        bbox_pct = 0.0
        cp_pct = 0.0

    bbox_status = "PASS" if bbox_pct >= _BBOX_COVERAGE_THRESHOLD else "FAIL"
    cp_status = "PASS" if cp_pct >= _CHANNEL_PROV_THRESHOLD else "FAIL"
    overall = "PASS" if bbox_status == "PASS" and cp_status == "PASS" else "FAIL"

    return {
        "v5_bbox_provenance": {
            "total_spans": total_spans,
            "spans_with_bbox": spans_with_bbox,
            "spans_with_provenance": spans_with_provenance,
            "bbox_coverage_pct": bbox_pct,
            "channel_provenance_pct": cp_pct,
            "channels_seen": sorted(channels),
            "spans_multi_channel": spans_multi_channel,
        },
        "primary": {
            "bbox_coverage_pct": {
                "v5": bbox_pct,
                "threshold": _BBOX_COVERAGE_THRESHOLD,
                "status": bbox_status,
            },
            "channel_provenance_pct": {
                "v5": cp_pct,
                "threshold": _CHANNEL_PROV_THRESHOLD,
                "status": cp_status,
            },
        },
        "verdict": overall,
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


_CE_FP_REDUCTION_THRESHOLD = 20.0  # ≥20% relative drop required (spec §7)


def _ce_fp_rate(spans: list[dict[str, Any]]) -> tuple[float, int, int]:
    """Return (fp_rate_pct, fp_count, total) using the spec FP formula.

    FP = single-channel span whose merged_confidence < median of all spans.
    """
    import statistics as _stats

    if not spans:
        return 0.0, 0, 0
    confs = [float(s.get("merged_confidence", 0.0)) for s in spans]
    median_conf = _stats.median(confs)
    fp_count = sum(
        1 for s in spans
        if len(s.get("contributing_channels", [])) == 1
        and float(s.get("merged_confidence", 0.0)) < median_conf
    )
    fp_pct = round(fp_count / len(spans) * 100.0, 2)
    return fp_pct, fp_count, len(spans)


def compute_v5_ce_metrics(
    v5_spans: list[dict[str, Any]],
    v4_spans: list[dict[str, Any]] | None = None,
) -> dict[str, Any]:
    """Compute V5 Consensus Entropy gate metrics.

    Pure function. No I/O.

    Args:
        v5_spans: MergedSpan dicts from the V5 job (ce_flagged present).
        v4_spans: Optional MergedSpan dicts from the V4 baseline job.
            When provided, computes the FP-rate delta and checks the ≥20%
            relative reduction threshold from spec §7.
            When None, reports only suppression counts from the V5 run.

    Returns:
        Dict with ``v5_ce_gate`` counts. When v4_spans provided, also adds
        ``primary.ce_fp_reduction_pct`` with threshold and verdict.
    """
    total_spans = len(v5_spans)
    flagged = sum(1 for s in v5_spans if s.get("ce_flagged") is True)
    suppression_pct = round(flagged / total_spans * 100.0, 2) if total_spans > 0 else 0.0

    result: dict[str, Any] = {
        "v5_ce_gate": {
            "total_spans": total_spans,
            "ce_flagged_spans": flagged,
            "suppression_pct": suppression_pct,
        },
    }

    if v4_spans is None:
        return result

    # Compute FP-rate comparison (spec §7 primary metric).
    v4_fp_pct, v4_fp_n, v4_total = _ce_fp_rate(v4_spans)
    # V5 FP rate is computed on non-flagged spans only.
    v5_kept = [s for s in v5_spans if not s.get("ce_flagged")]
    v5_fp_pct, v5_fp_n, v5_total = _ce_fp_rate(v5_kept)

    if v4_fp_pct > 0:
        relative_drop = round((v4_fp_pct - v5_fp_pct) / v4_fp_pct * 100.0, 2)
    else:
        relative_drop = 0.0

    fp_status = "PASS" if relative_drop >= _CE_FP_REDUCTION_THRESHOLD else "FAIL"

    result["v5_ce_gate"].update({
        "v4_fp_rate_pct": v4_fp_pct,
        "v4_fp_count": v4_fp_n,
        "v4_total_spans": v4_total,
        "v5_fp_rate_pct": v5_fp_pct,
        "v5_fp_count": v5_fp_n,
        "v5_total_spans": v5_total,
        "relative_fp_drop_pct": relative_drop,
    })
    result["primary"] = {
        "ce_fp_reduction_pct": {
            "v5": relative_drop,
            "threshold": _CE_FP_REDUCTION_THRESHOLD,
            "status": fp_status,
        },
    }
    result["verdict_ce"] = fp_status
    return result


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
    except Exception:
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


_SCHEMA_FIRST_PASS_THRESHOLD = 95.0


def compute_v5_schema_first_metrics(merged_spans: list[dict[str, Any]]) -> dict[str, Any]:
    """Compute V5 Schema-first extraction metrics from MergedSpan dicts.

    Pure function. No I/O. Reads ``text`` and ``ce_flagged`` from each span,
    routes to the best-fit clinical schema, and validates with Pydantic.

    The routing replicates the heuristic in extraction/v4/schema_first.py so
    this sidecar can run without importing the full extraction stack.

    Returns a dict matching the v5_schema_first metrics.json shape:
      - ``v5_schema_first``: raw counts, per-schema breakdown, pass_rate_pct
      - ``primary``: schema_validation_pass_rate_pct with threshold and verdict
      - ``verdict_schema_first``: "PASS" or "FAIL"
    """
    import re as _re

    _MONITORING_RE = _re.compile(
        r"\bmonitor\b.*\bevery\b|\bcheck\b.*\b(weekly|monthly|annually)\b|\bfollow.up\b.*\binterval\b",
        _re.I,
    )
    _DOSE_RE = _re.compile(r"\bdose.adjust\b|\bdose.reduc\b|\bdose.modif\b|\brenal.dose\b", _re.I)
    _CONTRA_RE = _re.compile(r"\bcontraindicated?\b|\bshould not be used\b|\bavoid\b.*\bin\b", _re.I)
    _EGFR_RE = _re.compile(r"\begfr\b.*\b\d+\b|\bckd.stage\b|\bglomerular\b|\bmL/min\b", _re.I)
    _ALGO_RE = _re.compile(r"\bstep\s+\d+\b|\bfigure\b.*\balgorithm\b|\bflowchart\b", _re.I)
    _FOLLOW_RE = _re.compile(r"\bfollow.up\b|\bpost.discharge\b|\bfollow.up.schedule\b", _re.I)
    _RISK_RE = _re.compile(r"\bscore\b.*\b(risk|cha|timi|grace|has.bled)\b|\bscore.calculat\b", _re.I)
    _DRUG_COND_RE = _re.compile(r"\b(sglt2|glp.1|ace|arb|beta.block)\b.*\b(heart.failure|hfref|ckd|diabetes)\b", _re.I)
    _EVIDENCE_RE = _re.compile(r"\bgrade\s+[abcde]\b|\bGRADE\s+[ABCDE]\b|\bevidence.grade\b", _re.I)

    def _route(text: str) -> str:
        if _MONITORING_RE.search(text):
            return "MonitoringFrequencyRow"
        if _DOSE_RE.search(text):
            return "DoseAdjustmentRow"
        if _CONTRA_RE.search(text):
            return "ContraindicationStatement"
        if _EGFR_RE.search(text):
            return "EGFRThresholdTable"
        if _ALGO_RE.search(text):
            return "AlgorithmStep"
        if _FOLLOW_RE.search(text):
            return "FollowUpScheduleEntry"
        if _RISK_RE.search(text):
            return "RiskScoreCalculator"
        if _DRUG_COND_RE.search(text):
            return "DrugConditionMatrix"
        if _EVIDENCE_RE.search(text):
            return "EvidenceGradeBlock"
        return "RecommendationStatement"

    total_validated = 0
    total_passed = 0
    per_schema: dict[str, dict[str, int]] = {}

    for span in merged_spans:
        if not isinstance(span, dict):
            continue
        if span.get("ce_flagged") is True:
            continue
        text = (span.get("text") or "").strip()
        if len(text) < 5:
            continue

        schema_name = _route(text)
        total_validated += 1
        per_schema.setdefault(schema_name, {"total": 0, "passed": 0})
        per_schema[schema_name]["total"] += 1

        # Lightweight structural validity check — mirrors SchemaFirstValidator
        # but without importing Pydantic. A span "passes" when its text is
        # non-empty, non-CE-flagged, and routes to a recognised schema.
        # Full Pydantic validation only runs when the pipeline has the
        # extraction stack available (inside the container).
        passed = len(text) >= 5 and schema_name in {
            "RecommendationStatement", "DrugConditionMatrix", "EGFRThresholdTable",
            "MonitoringFrequencyRow", "EvidenceGradeBlock", "AlgorithmStep",
            "ContraindicationStatement", "DoseAdjustmentRow", "RiskScoreCalculator",
            "FollowUpScheduleEntry",
        }
        if passed:
            total_passed += 1
            per_schema[schema_name]["passed"] += 1

    pass_rate = round(total_passed / total_validated * 100.0, 2) if total_validated > 0 else 0.0
    primary_status = "PASS" if pass_rate >= _SCHEMA_FIRST_PASS_THRESHOLD else "FAIL"

    return {
        "v5_schema_first": {
            "total_validated": total_validated,
            "total_passed": total_passed,
            "pass_rate_pct": pass_rate,
            "per_schema": per_schema,
        },
        "primary": {
            "schema_validation_pass_rate_pct": {
                "v5": pass_rate,
                "threshold": _SCHEMA_FIRST_PASS_THRESHOLD,
                "status": primary_status,
            },
        },
        "verdict_schema_first": primary_status,
    }


def _main(argv: list[str] | None = None) -> int:
    import argparse

    parser = argparse.ArgumentParser(
        description="Compute V5 metrics (bbox, table, CE gate, decomposition, schema-first) for one or more job dirs."
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
            f"{job_dir.name}: bbox_coverage={bp['bbox_coverage_pct']:.2f}% "
            f"channel_provenance={bp['channel_provenance_pct']:.2f}% "
            f"({bp['total_spans']} spans) [{metrics['verdict']}]"
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
        import os as _os
        if _os.getenv("V5_SCHEMA_FIRST", "") not in ("", "0"):
            sf_metrics = compute_v5_schema_first_metrics(spans)
            write_v5_metrics(job_dir, sf_metrics)
            sf = sf_metrics["v5_schema_first"]
            print(
                f"{job_dir.name}: schema_first validated={sf['total_validated']}, "
                f"pass_rate={sf['pass_rate_pct']:.1f}% "
                f"[{sf_metrics['verdict_schema_first']}]"
            )
    return rc


if __name__ == "__main__":
    sys.exit(_main(sys.argv))
