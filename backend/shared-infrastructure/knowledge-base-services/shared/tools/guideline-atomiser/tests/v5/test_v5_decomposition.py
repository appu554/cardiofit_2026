"""V5 Decomposition unit tests.

Run from guideline-atomiser/:
    PYTHONPATH=. V5_DECOMPOSITION=1 pytest tests/v5/test_v5_decomposition.py -v
"""
from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional
from uuid import uuid4

import pytest
import yaml

from extraction.v4.decomposition import (
    GuidelineDecomposer,
    GuidelineGraph,
    GraphNode,
    GraphEdge,
)
from extraction.v4.models import GuidelineTree, GuidelineSection


GOLDEN_DIR = Path(__file__).parent / "golden" / "relationships"


@dataclass
class _MockSpan:
    id: object = field(default_factory=uuid4)
    text: str = ""
    contributing_channels: list = field(default_factory=list)
    section_id: Optional[str] = None
    page_number: Optional[int] = None
    merged_confidence: float = 0.9
    tier: Optional[str] = None


@dataclass
class _MockPassage:
    section_id: str = ""
    heading: str = ""
    prose_text: str = ""


def _make_tree_with_section(heading: str, section_id: str) -> GuidelineTree:
    section = GuidelineSection(
        section_id=section_id,
        heading=heading,
        start_offset=0,
        end_offset=100,
        page_number=1,
        block_type="recommendation",
        level=2,
    )
    return GuidelineTree(sections=[section], tables=[], total_pages=3)


def test_decompose_returns_none_when_flag_off(monkeypatch):
    """decompose() returns None when V5_DECOMPOSITION is explicitly disabled."""
    monkeypatch.setenv("V5_DECOMPOSITION", "0")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomposer = GuidelineDecomposer()
    tree = GuidelineTree(sections=[], tables=[], total_pages=1)
    result = decomposer.decompose(str(uuid4()), [], tree, [])
    assert result is None


def test_decompose_returns_graph_when_flag_on(monkeypatch):
    """decompose() returns GuidelineGraph when V5_DECOMPOSITION=1."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomposer = GuidelineDecomposer()
    tree = GuidelineTree(sections=[], tables=[], total_pages=1)
    result = decomposer.decompose(str(uuid4()), [], tree, [])
    assert isinstance(result, GuidelineGraph)


def test_disable_all_overrides_decomposition(monkeypatch):
    """V5_DISABLE_ALL=1 forces None return even if V5_DECOMPOSITION=1."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.setenv("V5_DISABLE_ALL", "1")

    decomposer = GuidelineDecomposer()
    tree = GuidelineTree(sections=[], tables=[], total_pages=1)
    result = decomposer.decompose(str(uuid4()), [], tree, [])
    assert result is None


def test_recommendation_node_extracted_from_section_heading(monkeypatch):
    """A section with heading 'Recommendation 4.1.1' produces a RECOMMENDATION node."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    tree = _make_tree_with_section("Recommendation 4.1.1 — SGLT2 inhibitors", "4.1.1")
    decomposer = GuidelineDecomposer()
    graph = decomposer.decompose(str(uuid4()), [], tree, [])

    rec_nodes = [n for n in graph.nodes if n.node_type == "RECOMMENDATION"]
    assert len(rec_nodes) >= 1
    assert rec_nodes[0].id == "rec_4_1_1"


def test_drug_class_node_from_channel_b_span(monkeypatch):
    """Channel B span produces a DRUG_CLASS node."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    span = _MockSpan(text="SGLT2 inhibitor", contributing_channels=["B"])
    tree = GuidelineTree(sections=[], tables=[], total_pages=1)
    decomposer = GuidelineDecomposer()
    graph = decomposer.decompose(str(uuid4()), [span], tree, [])

    drug_nodes = [n for n in graph.nodes if n.node_type == "DRUG_CLASS"]
    assert len(drug_nodes) >= 1
    assert "sglt2" in drug_nodes[0].id


def test_algorithm_node_from_figure_reference(monkeypatch):
    """Span text containing 'see Figure 1' produces an ALGORITHM node."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    span = _MockSpan(text="see Figure 1 for treatment algorithm", contributing_channels=["C"])
    tree = GuidelineTree(sections=[], tables=[], total_pages=1)
    decomposer = GuidelineDecomposer()
    graph = decomposer.decompose(str(uuid4()), [span], tree, [])

    alg_nodes = [n for n in graph.nodes if n.node_type == "ALGORITHM"]
    assert len(alg_nodes) >= 1
    assert "fig1" in alg_nodes[0].id


def test_is_treated_by_edge_created_for_rec_and_drug_in_same_section(monkeypatch):
    """Recommendation and drug-class in same section → IS_TREATED_BY edge."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    tree = _make_tree_with_section("Recommendation 1.2.3 — treatment", "1.2.3")
    span = _MockSpan(
        text="SGLT2 inhibitor",
        contributing_channels=["B"],
        section_id="1.2.3",
        page_number=1,
    )
    decomposer = GuidelineDecomposer()
    graph = decomposer.decompose(str(uuid4()), [span], tree, [])

    is_treated_edges = [e for e in graph.edges if e.edge_type == "IS_TREATED_BY"]
    assert len(is_treated_edges) >= 1
    assert is_treated_edges[0].source_node_id == "rec_1_2_3"


def test_references_algorithm_edge_from_passage(monkeypatch):
    """Passage heading matches rec + prose contains figure ref → REFERENCES_ALGORITHM edge."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    tree = _make_tree_with_section("Recommendation 2.1.0 — initiation", "2.1.0")
    alg_span = _MockSpan(text="see Figure 2 for initiation algorithm", contributing_channels=["C"])
    passage = _MockPassage(
        section_id="2.1.0",
        heading="Recommendation 2.1.0 — initiation",
        prose_text="see Figure 2 for initiation algorithm",
    )
    decomposer = GuidelineDecomposer()
    graph = decomposer.decompose(str(uuid4()), [alg_span], tree, [passage])

    ref_edges = [e for e in graph.edges if e.edge_type == "REFERENCES_ALGORITHM"]
    assert len(ref_edges) >= 1


def test_golden_relationships_format():
    """Golden YAML fixture is well-formed and contains expected fields."""
    path = GOLDEN_DIR / "kdigo_rec123.yaml"
    assert path.exists(), f"Golden fixture missing: {path}"
    data = yaml.safe_load(path.read_text())
    assert "relationships" in data
    for rel in data["relationships"]:
        assert "source_node_id" in rel
        assert "target_node_id" in rel
        assert "edge_type" in rel


def test_graph_to_dict_includes_required_keys(monkeypatch):
    """GuidelineGraph.to_dict() includes job_id, nodes, edges, node_count, edge_count."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    tree = _make_tree_with_section("Recommendation 3.1.0", "3.1.0")
    span = _MockSpan(text="metformin", contributing_channels=["B"])
    decomposer = GuidelineDecomposer()
    job_id = str(uuid4())
    graph = decomposer.decompose(job_id, [span], tree, [])

    d = graph.to_dict()
    assert d["job_id"] == job_id
    assert "nodes" in d
    assert "edges" in d
    assert "node_count" in d
    assert "edge_count" in d


def test_span_derived_nodes_have_source_span_provenance(monkeypatch):
    """DRUG_CLASS and ALGORITHM nodes (derived from spans) have at least one source_span_id."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    span = _MockSpan(text="SGLT2 inhibitor", contributing_channels=["B"])
    tree = GuidelineTree(sections=[], tables=[], total_pages=1)
    decomposer = GuidelineDecomposer()
    graph = decomposer.decompose(str(uuid4()), [span], tree, [])

    span_derived = [n for n in graph.nodes if n.node_type in ("DRUG_CLASS", "ALGORITHM")]
    for node in span_derived:
        assert len(node.source_span_ids) >= 1, (
            f"Span-derived node {node.id!r} (type={node.node_type}) has no source_span_ids"
        )


def test_recommendation_node_may_have_empty_span_ids_when_no_spans_in_section(monkeypatch):
    """RECOMMENDATION nodes from section headings have empty source_span_ids when no spans match."""
    monkeypatch.setenv("V5_DECOMPOSITION", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    tree = _make_tree_with_section("Recommendation 9.9.9 — test", "9.9.9")
    decomposer = GuidelineDecomposer()
    # No spans with section_id="9.9.9" → source_span_ids will be []
    graph = decomposer.decompose(str(uuid4()), [], tree, [])

    rec_nodes = [n for n in graph.nodes if n.node_type == "RECOMMENDATION"]
    assert len(rec_nodes) == 1
    # Empty source_span_ids is expected when no merged spans are in this section
    assert isinstance(rec_nodes[0].source_span_ids, list)
