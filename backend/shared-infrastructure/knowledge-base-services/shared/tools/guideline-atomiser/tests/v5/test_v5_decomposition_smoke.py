"""Smoke acceptance tests for V5 Decomposition.

Requires a completed job run with V5_DECOMPOSITION=1.

Run:
    PYTHONPATH=. V5_DECOMPOSITION=1 V5_BBOX_PROVENANCE=1 \\
        pytest tests/v5/test_v5_decomposition_smoke.py -v -m smoke
"""
from __future__ import annotations

import json
import os
from pathlib import Path

import pytest

pytestmark = pytest.mark.smoke

_DECOMP_ENABLED = os.getenv("V5_DECOMPOSITION", "") not in ("", "0")

SMOKE_JOB_DIR = Path("data/output/v4")


def _latest(pattern: str) -> Path | None:
    """Return the most recently modified file matching glob pattern, or None."""
    candidates = list(SMOKE_JOB_DIR.glob(pattern))
    return max(candidates, key=lambda p: p.stat().st_mtime) if candidates else None


@pytest.mark.skipif(
    _latest("**/graph.json") is None,
    reason="No graph.json found — run pipeline first",
)
def test_graph_json_written():
    """graph.json is written and contains top-level nodes and edges keys."""
    latest_graph = _latest("**/graph.json")
    with open(latest_graph) as f:
        data = json.load(f)
    assert "nodes" in data, "graph.json missing 'nodes' key"
    assert "edges" in data, "graph.json missing 'edges' key"
    assert data.get("node_count", 0) >= 0, "node_count must be non-negative"
    assert data.get("edge_count", 0) >= 0, "edge_count must be non-negative"


@pytest.mark.skipif(
    _latest("**/graph.json") is None,
    reason="No graph.json found — run pipeline first",
)
def test_every_node_has_provenance():
    """Every node in graph.json carries at least one source_span_id."""
    latest_graph = _latest("**/graph.json")
    with open(latest_graph) as f:
        data = json.load(f)
    for node in data.get("nodes", []):
        assert len(node.get("source_span_ids", [])) >= 1, (
            f"Node {node.get('id')!r} has no source_span_ids"
        )


@pytest.mark.skipif(
    _latest("**/job_metadata.json") is None or not _DECOMP_ENABLED,
    reason="No job_metadata.json found or V5_DECOMPOSITION not set — run pipeline with V5_DECOMPOSITION=1 first",
)
def test_decomposition_active_in_metadata():
    """job_metadata.json reports decomposition in v5_features_enabled.

    Only asserts when V5_DECOMPOSITION=1 because pipeline output from a run
    without the flag legitimately excludes decomposition from the features list.
    """
    latest_meta = _latest("**/job_metadata.json")
    with open(latest_meta) as f:
        meta = json.load(f)
    assert "decomposition" in meta.get("v5_features_enabled", []), (
        f"Expected 'decomposition' in v5_features_enabled, got {meta.get('v5_features_enabled')}"
    )


@pytest.mark.skipif(
    _latest("**/metrics.json") is None or not _DECOMP_ENABLED,
    reason="No metrics.json found or V5_DECOMPOSITION not set — run pipeline with V5_DECOMPOSITION=1 first",
)
def test_decomposition_metrics_written():
    """metrics.json contains v5_decomposition key with node_count and edge_count."""
    latest_metrics = _latest("**/metrics.json")
    with open(latest_metrics) as f:
        m = json.load(f)
    assert "v5_decomposition" in m, "metrics.json missing 'v5_decomposition' key"
    assert m["v5_decomposition"]["node_count"] >= 0, "node_count must be non-negative"
    assert m["v5_decomposition"]["edge_count"] >= 0, "edge_count must be non-negative"
