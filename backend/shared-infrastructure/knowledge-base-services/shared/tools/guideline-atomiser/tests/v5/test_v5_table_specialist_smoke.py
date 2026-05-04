"""Smoke acceptance test for V5 Table Specialist.

Requires:
    V5_TABLE_SPECIALIST=1
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf present (or any PDF in data/output/v4/)

Run:
    PYTHONPATH=. V5_TABLE_SPECIALIST=1 V5_BBOX_PROVENANCE=1 \
        pytest tests/v5/test_v5_table_specialist_smoke.py -v -m smoke
"""
from __future__ import annotations

import json
from pathlib import Path

import pytest

pytestmark = pytest.mark.smoke

SMOKE_JOB_DIR = Path("data/output/v4")


def _latest(pattern: str) -> Path | None:
    """Return the most recently modified file matching glob pattern, or None."""
    candidates = list(SMOKE_JOB_DIR.glob(pattern))
    return max(candidates, key=lambda p: p.stat().st_mtime) if candidates else None


@pytest.mark.skipif(
    _latest("**/job_metadata.json") is None,
    reason="No job output found — run pipeline first",
)
def test_table_specialist_active_in_metadata():
    """job_metadata.json reports table_specialist in v5_features_enabled."""
    latest_meta = _latest("**/job_metadata.json")
    with open(latest_meta) as f:
        meta = json.load(f)
    assert "table_specialist" in meta.get("v5_features_enabled", []), (
        f"Expected 'table_specialist' in v5_features_enabled, got {meta.get('v5_features_enabled')}"
    )


@pytest.mark.skipif(
    _latest("**/merged_spans.json") is None,
    reason="No merged_spans.json found — run pipeline first",
)
def test_docling_table_spans_present():
    """merged_spans.json contains at least one span from docling_tableblock."""
    latest_spans = _latest("**/merged_spans.json")
    with open(latest_spans) as f:
        spans = json.load(f)
    # MergedSpan stores origin in channel_provenance.model_version (not channel_metadata).
    # table@v1.0 is set by the bbox-provenance layer for Docling TableBlock spans.
    docling_spans = [
        s for s in spans
        if any(
            isinstance(p, dict) and p.get("model_version") == "table@v1.0"
            for p in s.get("channel_provenance", [])
        )
    ]
    assert len(docling_spans) > 0, "No table@v1.0 provenance spans found in merged_spans.json"


@pytest.mark.skipif(
    _latest("**/metrics.json") is None,
    reason="No metrics.json found — run v5_metrics.py first",
)
def test_table_specialist_metrics_written():
    """metrics.json contains v5_table_specialist key with docling stats."""
    latest_metrics = _latest("**/metrics.json")
    with open(latest_metrics) as f:
        m = json.load(f)
    assert "v5_table_specialist" in m, "metrics.json missing v5_table_specialist key"
    ts = m["v5_table_specialist"]
    assert "docling_table_cell_spans" in ts
    assert ts["docling_table_cell_spans"] >= 0
