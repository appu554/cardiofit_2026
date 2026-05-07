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
    """merged_spans.json contains at least one Channel-D span with provenance.

    Channel D's V5 lane chain can produce several distinct model_version values
    (the original test pinned only one of them):

      - "table@v1.0"              ← generic docling fallback path
      - "pipe-table@v1.0"         ← V4 marker pipe path
      - "docling-otsl@v1.0"       ← V4 Granite-Docling OTSL path
      - "nvidia/NVIDIA-Nemotron-Parse-v1.1-TC@sidecar"  ← V5 nemotron self-hosted
      - "nvidia/nemotron-parse-1.1@nim"                 ← V5 nemotron cloud

    What we actually care about is that *some* table-cell span made it through
    Channel D with bbox-provenance attached — that proves V5 #1 (table specialist)
    + V5 #2 (bbox provenance) cooperated end-to-end. Pinning a single model_version
    string was a fragility that broke the moment a lane priority changed.
    """
    latest_spans = _latest("**/merged_spans.json")
    with open(latest_spans) as f:
        spans = json.load(f)

    # Any merged span where Channel D contributed AND at least one
    # channel_provenance entry has channel_id == "D" (the actual invariant —
    # model_version is downstream of routing decisions and shouldn't be
    # load-bearing for a smoke test). Note: MergedSpan does NOT have a
    # ``.channel`` field — that's a RawSpan attribute. Merged spans carry
    # ``contributing_channels`` (a list of channel-id strings).
    channel_d_with_prov = [
        s for s in spans
        if "D" in (s.get("contributing_channels") or [])
        and any(
            isinstance(p, dict) and p.get("channel_id") == "D"
            for p in s.get("channel_provenance", [])
        )
    ]
    assert len(channel_d_with_prov) > 0, (
        "No Channel-D spans with channel_provenance found in merged_spans.json — "
        "table_specialist + bbox_provenance not cooperating"
    )


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
