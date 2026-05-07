"""Tests for v5_metrics.py sidecar — V5 Bbox Provenance metrics computation."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest

# Add data/ dir to path so we can import v5_metrics standalone.
DATA_DIR = Path(__file__).resolve().parents[2] / "data"
sys.path.insert(0, str(DATA_DIR))

from v5_metrics import compute_v5_bbox_metrics, write_v5_metrics  # noqa: E402


def _span(provenance=None):
    s = {"span_id": "s1", "text": "x"}
    if provenance is not None:
        s["channel_provenance"] = provenance
    return s


def test_compute_metrics_all_provenance():
    spans = [
        _span([{"channel_id": "A", "bbox": [0, 0, 1, 1]}]),
        _span([{"channel_id": "A", "bbox": [0, 0, 1, 1]}]),
        _span([{"channel_id": "B", "bbox": [0, 0, 1, 1]}]),
    ]
    m = compute_v5_bbox_metrics(spans)["v5_bbox_provenance"]
    assert m["total_spans"] == 3
    assert m["spans_with_provenance"] == 3
    assert m["channel_provenance_pct"] == 100.0


def test_compute_metrics_partial_provenance():
    spans = [
        _span([{"channel_id": "A", "bbox": [0, 0, 1, 1]}]),
        _span([{"channel_id": "A", "bbox": [0, 0, 1, 1]}]),
        _span([]),
        _span(None),
    ]
    m = compute_v5_bbox_metrics(spans)["v5_bbox_provenance"]
    assert m["total_spans"] == 4
    assert m["spans_with_provenance"] == 2
    assert m["channel_provenance_pct"] == 50.0


def test_compute_metrics_no_provenance():
    spans = [_span([]), _span(None), _span([])]
    m = compute_v5_bbox_metrics(spans)["v5_bbox_provenance"]
    assert m["bbox_coverage_pct"] == 0.0
    assert m["channels_seen"] == []
    assert m["spans_with_provenance"] == 0


def test_compute_metrics_zero_spans():
    m = compute_v5_bbox_metrics([])["v5_bbox_provenance"]
    assert m["total_spans"] == 0
    assert m["bbox_coverage_pct"] == 0.0
    assert m["channels_seen"] == []
    assert m["spans_multi_channel"] == 0


def test_compute_metrics_channels_seen():
    spans = [
        _span([{"channel_id": "A", "bbox": [0, 0, 1, 1]}]),
        _span([{"channel_id": "B", "bbox": [0, 0, 1, 1]}]),
        _span([{"channel_id": "A", "bbox": [0, 0, 1, 1]}]),
    ]
    m = compute_v5_bbox_metrics(spans)["v5_bbox_provenance"]
    assert m["channels_seen"] == ["A", "B"]


def test_compute_metrics_multi_channel():
    spans = [
        _span([
            {"channel_id": "A", "bbox": [0, 0, 1, 1]},
            {"channel_id": "B", "bbox": [0, 0, 1, 1]},
        ]),
        _span([{"channel_id": "A", "bbox": [0, 0, 1, 1]}]),
    ]
    m = compute_v5_bbox_metrics(spans)["v5_bbox_provenance"]
    assert m["spans_multi_channel"] == 1


def test_write_v5_metrics_creates_file(tmp_path: Path):
    metrics = {"v5_bbox_provenance": {"total_spans": 0, "bbox_coverage_pct": 0.0}}
    write_v5_metrics(tmp_path, metrics)
    out = tmp_path / "metrics.json"
    assert out.exists()
    data = json.loads(out.read_text())
    assert "v5_bbox_provenance" in data


def test_write_v5_metrics_merges_existing(tmp_path: Path):
    existing = tmp_path / "metrics.json"
    existing.write_text(json.dumps({"existing_key": 1}))
    write_v5_metrics(tmp_path, {"v5_bbox_provenance": {"total_spans": 5}})
    data = json.loads(existing.read_text())
    assert data["existing_key"] == 1
    assert "v5_bbox_provenance" in data
    assert data["v5_bbox_provenance"]["total_spans"] == 5
