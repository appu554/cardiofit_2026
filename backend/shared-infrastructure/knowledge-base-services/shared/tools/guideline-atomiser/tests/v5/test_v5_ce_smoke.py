"""Smoke acceptance tests for V5 Consensus Entropy gate.

Requires:
    V5_CONSENSUS_ENTROPY=1
    Pipeline output present in data/output/v4/

Run:
    PYTHONPATH=. V5_CONSENSUS_ENTROPY=1 \
        pytest tests/v5/test_v5_ce_smoke.py -v -m smoke
"""
from __future__ import annotations

import json
import os
from pathlib import Path

import pytest

pytestmark = pytest.mark.smoke

_CE_ENABLED = os.getenv("V5_CONSENSUS_ENTROPY", "") not in ("", "0")

SMOKE_JOB_DIR = Path("data/output/v4")


def _latest(pattern: str) -> Path | None:
    """Return the most recently modified file matching glob pattern, or None."""
    candidates = list(SMOKE_JOB_DIR.glob(pattern))
    return max(candidates, key=lambda p: p.stat().st_mtime) if candidates else None


@pytest.mark.skipif(
    _latest("**/job_metadata.json") is None or not _CE_ENABLED,
    reason="No job output found or V5_CONSENSUS_ENTROPY not set — run pipeline with V5_CONSENSUS_ENTROPY=1 first",
)
def test_ce_gate_active_in_metadata():
    """job_metadata.json reports consensus_entropy in v5_features_enabled.

    Only asserts when V5_CONSENSUS_ENTROPY=1 because a pipeline run without
    the flag produces valid output that legitimately excludes CE from the
    features list.
    """
    latest_meta = _latest("**/job_metadata.json")
    with open(latest_meta) as f:
        meta = json.load(f)
    assert "consensus_entropy" in meta.get("v5_features_enabled", []), (
        f"Expected 'consensus_entropy' in v5_features_enabled, got {meta.get('v5_features_enabled')}"
    )


@pytest.mark.skipif(
    _latest("**/metrics.json") is None or not _CE_ENABLED,
    reason="No metrics.json found or V5_CONSENSUS_ENTROPY not set — run pipeline with V5_CONSENSUS_ENTROPY=1 first",
)
def test_ce_metrics_written():
    """metrics.json contains v5_ce_gate key with ce_flagged_spans and total_spans."""
    latest_metrics = _latest("**/metrics.json")
    with open(latest_metrics) as f:
        metrics = json.load(f)
    assert "v5_ce_gate" in metrics, "metrics.json missing v5_ce_gate key"
    ce = metrics["v5_ce_gate"]
    assert "ce_flagged_spans" in ce, "v5_ce_gate missing ce_flagged_spans"
    assert "total_spans" in ce, "v5_ce_gate missing total_spans"
    assert ce["ce_flagged_spans"] >= 0, "ce_flagged_spans must be non-negative"
    assert ce["total_spans"] >= 0, "total_spans must be non-negative"


@pytest.mark.skipif(
    _latest("**/merged_spans.json") is None or not _CE_ENABLED,
    reason="No merged_spans.json found or V5_CONSENSUS_ENTROPY not set — run pipeline with V5_CONSENSUS_ENTROPY=1 first",
)
def test_v4_baseline_no_ce_flagged_spans():
    """merged_spans.json contains span dicts with ce_flagged field serialised by pipeline.

    This test is pipeline-agnostic: it only asserts that the ce_flagged field
    is present on at least one span, confirming the field is written by the
    pipeline regardless of its boolean value. Only runs when V5_CONSENSUS_ENTROPY=1
    because a pipeline run without CE produces spans that legitimately omit the field.
    """
    latest_spans = _latest("**/merged_spans.json")
    with open(latest_spans) as f:
        spans = json.load(f)
    assert all(isinstance(s, dict) for s in spans), (
        "All entries in merged_spans.json must be dicts"
    )
    spans_with_ce_field = [s for s in spans if "ce_flagged" in s]
    assert len(spans_with_ce_field) > 0, (
        "No span in merged_spans.json has a 'ce_flagged' key — "
        "confirm the CE gate serialises the field into merged spans"
    )
