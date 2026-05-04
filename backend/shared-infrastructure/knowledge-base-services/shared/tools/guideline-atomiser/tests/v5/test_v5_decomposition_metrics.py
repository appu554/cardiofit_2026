"""Unit tests for compute_v5_decomposition_metrics()."""
from __future__ import annotations
import sys
from pathlib import Path

import pytest

# Add data/ dir to path so we can import v5_metrics standalone.
DATA_DIR = Path(__file__).resolve().parents[2] / "data"
sys.path.insert(0, str(DATA_DIR))

from v5_metrics import compute_v5_decomposition_metrics  # noqa: E402


def _make_graph(nodes=2, edges=3):
    return {
        "job_id": "test",
        "node_count": nodes,
        "edge_count": edges,
        "nodes": [{"id": f"n{i}", "node_type": "RECOMMENDATION", "label": f"N{i}", "source_span_ids": []} for i in range(nodes)],
        "edges": [{"id": f"e{i}", "source_node_id": "n0", "target_node_id": "n1", "edge_type": "IS_TREATED_BY"} for i in range(edges)],
    }


def test_structural_only_no_gt():
    """Without ground truth, returns node_count and edge_count only."""
    result = compute_v5_decomposition_metrics(_make_graph(2, 3))
    assert "v5_decomposition" in result
    d = result["v5_decomposition"]
    assert d["node_count"] == 2
    assert d["edge_count"] == 3
    assert "edge_precision_pct" not in d


def test_empty_graph():
    """Empty graph (0 nodes, 0 edges) returns zeros without error."""
    result = compute_v5_decomposition_metrics({"node_count": 0, "edge_count": 0, "nodes": [], "edges": []})
    assert result["v5_decomposition"]["node_count"] == 0
    assert result["v5_decomposition"]["edge_count"] == 0


def test_with_ground_truth_perfect_match(tmp_path):
    """Perfect match -> precision=100.0, recall=100.0, verdict=PASS."""
    import yaml
    gt = tmp_path / "gt.yaml"
    gt.write_text(yaml.dump({"relationships": [
        {"source_node_id": "n0", "target_node_id": "n1", "edge_type": "IS_TREATED_BY"},
    ]}))
    graph = {
        "node_count": 2, "edge_count": 1, "nodes": [],
        "edges": [{"source_node_id": "n0", "target_node_id": "n1", "edge_type": "IS_TREATED_BY"}],
    }
    result = compute_v5_decomposition_metrics(graph, ground_truth_yaml=str(gt))
    d = result["v5_decomposition"]
    assert d["edge_precision_pct"] == 100.0
    assert d["edge_recall_pct"] == 100.0
    assert result.get("verdict") == "PASS"


def test_with_ground_truth_no_overlap(tmp_path):
    """No overlap -> precision=0.0, recall=0.0, verdict=FAIL."""
    import yaml
    gt = tmp_path / "gt.yaml"
    gt.write_text(yaml.dump({"relationships": [
        {"source_node_id": "a", "target_node_id": "b", "edge_type": "IS_TREATED_BY"},
    ]}))
    graph = {
        "node_count": 2, "edge_count": 1, "nodes": [],
        "edges": [{"source_node_id": "x", "target_node_id": "y", "edge_type": "IS_TREATED_BY"}],
    }
    result = compute_v5_decomposition_metrics(graph, ground_truth_yaml=str(gt))
    assert result["v5_decomposition"]["edge_precision_pct"] == 0.0
    assert result["v5_decomposition"]["edge_recall_pct"] == 0.0
    assert result.get("verdict") == "FAIL"


def test_missing_gt_file():
    """Missing YAML path -> graceful degradation, no crash."""
    result = compute_v5_decomposition_metrics(_make_graph(), ground_truth_yaml="/nonexistent/gt.yaml")
    assert "v5_decomposition" in result
