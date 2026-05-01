"""V5 Bbox Provenance — smoke acceptance gate.

Single-test suite that asserts the V5 metric pipeline produces
>=99% bbox_coverage_pct on a realistic synthetic span set.

No GPU, no RunPod, no GCP, no real PDF — pure in-memory test on raw
dicts shaped like serialise_provenance_list() output.
"""
from __future__ import annotations

import sys
from pathlib import Path

# Add data/ dir to path so we can import v5_metrics standalone.
DATA_DIR = Path(__file__).resolve().parents[2] / "data"
sys.path.insert(0, str(DATA_DIR))

from v5_metrics import compute_v5_bbox_metrics  # noqa: E402


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
    """Acceptance gate: 199/200 V5 spans -> >=99% bbox coverage."""
    channels = ["A", "B", "C", "D"]
    spans: list[dict] = []
    for i in range(199):
        ch = channels[i % len(channels)]
        page = (i // 20) + 1
        x0 = 50.0 + (i % 5) * 220.0
        spans.append(_make_v5_span(ch, page, x0, i))
    # One V4-style span with no provenance — simulates a text-only
    # footnote that fell through without bbox.
    spans.append(_make_v4_span(199))

    metrics = compute_v5_bbox_metrics(spans)
    bbox = metrics["v5_bbox_provenance"]

    assert bbox["total_spans"] == 200
    assert bbox["spans_with_provenance"] == 199
    assert bbox["bbox_coverage_pct"] >= 99.0
    assert "A" in bbox["channels_seen"]
