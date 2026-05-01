"""V5 Bbox Provenance — smoke acceptance gate.

Two test scenarios:
1. Synthetic: asserts the V5 metric pipeline produces >=99.5% bbox_coverage_pct
   on a realistic in-memory span set (no GPU, no RunPod, no GCP).
2. V4 baseline: asserts that existing V4 job dirs (flag off) produce 0% bbox
   coverage — confirms the feature flag default-off contract. Skips when no
   smoke-set V4 job dirs are present locally.
"""
from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest

# Add data/ dir to path so we can import v5_metrics standalone.
DATA_DIR = Path(__file__).resolve().parents[2] / "data"
sys.path.insert(0, str(DATA_DIR))

from v5_metrics import compute_v5_bbox_metrics  # noqa: E402

_TOTAL_SPANS = 200
_V5_SPANS = 199   # one V4 fallthrough simulates text-only footnote
_SPANS_PER_PAGE = 20
_COLS = 5


def _make_v5_span(channel_id: str, page: int, x0: float, span_idx: int) -> dict:
    return {
        "span_id": f"span_{span_idx:04d}",
        "text": f"Clinical text span {span_idx}",
        "channel_provenance": [
            {
                "channel_id": channel_id,
                "bbox": {"x0": x0, "y0": 100.0, "x1": x0 + 200.0, "y1": 130.0},
                "page_number": page,
                "confidence": 0.9,
                "model_version": "test-model@v1",
                "notes": None,
            }
        ],
    }


def _make_v4_span(span_idx: int) -> dict:
    """V4 span: no channel_provenance key at all."""
    return {
        "span_id": f"span_{span_idx:04d}",
        "text": f"V4 text span {span_idx}",
    }


def test_smoke_bbox_coverage_gte_99_pct():
    """Acceptance gate: 199/200 V5 spans -> >=99.5% bbox coverage."""
    channels = ["A", "B", "C", "D"]
    spans: list[dict] = []
    for i in range(_V5_SPANS):
        ch = channels[i % len(channels)]
        page = (i // _SPANS_PER_PAGE) + 1
        x0 = 50.0 + (i % _COLS) * 220.0
        spans.append(_make_v5_span(ch, page, x0, i))
    # One V4-style span with no provenance — simulates a text-only
    # footnote that fell through without bbox.
    spans.append(_make_v4_span(_V5_SPANS))

    metrics = compute_v5_bbox_metrics(spans)
    bbox = metrics["v5_bbox_provenance"]

    assert bbox["total_spans"] == _TOTAL_SPANS, f"expected {_TOTAL_SPANS} spans, got {bbox['total_spans']}"
    assert bbox["spans_with_provenance"] == _V5_SPANS, f"expected {_V5_SPANS} with provenance, got {bbox['spans_with_provenance']}"
    # 99.5% = 199/200: one intentional V4 fallthrough (text-only footnote without bbox)
    assert bbox["bbox_coverage_pct"] >= 99.5, f"bbox_coverage_pct {bbox['bbox_coverage_pct']:.2f}% < 99.5% threshold"
    assert "A" in bbox["channels_seen"], f"channel A missing from channels_seen: {bbox['channels_seen']}"
    # Spec §7 shape: primary and verdict keys must be present
    assert metrics["verdict"] == "PASS", f"expected PASS verdict, got {metrics['verdict']}"
    assert metrics["primary"]["bbox_coverage_pct"]["status"] == "PASS"


def test_v4_baseline_has_zero_bbox_coverage(v4_baseline_jobs: dict) -> None:
    """V4 jobs (flag off) must report 0% bbox coverage — verifies default-off contract.

    Self-skips when no smoke-set V4 job dirs exist locally (CI without job
    output, or fresh checkout). Full smoke is documented in RUNPOD_SMOKE_V5.md.
    """
    if not v4_baseline_jobs:
        pytest.skip("No V4 smoke-set job dirs found in data/output/v4/ — skipping baseline check")
    for src_pdf, job_dir in v4_baseline_jobs.items():
        spans_path = Path(job_dir) / "merged_spans.json"
        spans = json.loads(spans_path.read_text(encoding="utf-8"))
        metrics = compute_v5_bbox_metrics(spans)
        bp = metrics["v5_bbox_provenance"]
        assert bp["bbox_coverage_pct"] == 0.0, (
            f"{src_pdf}: V4 baseline expected 0.0% bbox coverage "
            f"(flag was off), got {bp['bbox_coverage_pct']:.2f}%"
        )
        assert metrics["verdict"] == "FAIL", (
            f"{src_pdf}: expected FAIL verdict for 0% V4 baseline, got {metrics['verdict']}"
        )
