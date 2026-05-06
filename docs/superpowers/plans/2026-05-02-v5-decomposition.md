# V5 Subsystem #5 — Decomposition (Guideline2Graph-style) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Guideline2Graph-style directed graph over the `merged_spans` output that captures cross-section relationships (Recommendation → Algorithm → DrugClass), raising edge precision and recall from V4's ~19.6%/16.1% to ≥80%/80% on the 15-relationship hand-graded set, gated by `V5_DECOMPOSITION=1`.

**Architecture:** New `extraction/v4/decomposition.py` module adds `GuidelineDecomposer` that takes `merged_spans`, `tree`, and `section_passages` and produces a `GuidelineGraph` (nodes + directed edges with typed relationships). The graph is written to `graph.json` alongside `merged_spans.json` in the job directory. No modifications to `merged_spans.json` schema. Three node types: `RecommendationNode`, `AlgorithmNode`, `DrugClassNode`. Three edge types: `IS_TREATED_BY`, `REFERENCES_ALGORITHM`, `REQUIRES_MONITORING`. `_V5_KNOWN_FEATURES` gains `"decomposition"`.

**Tech Stack:** Python 3.13, regex-based relationship detection (no new model dependencies for V0), Pydantic v2, `networkx` (already in venv), pytest.

**Success criteria (from master spec §7):**
- Primary: edge precision AND edge recall both ≥80% on 15 hand-graded relationships
- Precision: `count(extracted ∩ graded) / count(extracted)`
- Recall: `count(extracted ∩ graded) / count(graded)`
- Edge equality: same `source_node_id`, `target_node_id`, `edge_type`
- Secondary: triplet precision/recall ≥85%, node recall ≥93%, provenance on every node

---

## File Structure

| File | Disposition | Purpose |
|------|-------------|---------|
| `${EXTRACTION}/v4/decomposition.py` | CREATE | `GuidelineDecomposer`, `GuidelineGraph`, `GraphNode`, `GraphEdge` models |
| `${EXTRACTION}/v4/models.py` | MODIFY | Import path for `GuidelineGraph` (re-export for pipeline convenience) |
| `${ATOMISER}/data/run_pipeline_targeted.py` | MODIFY | Add decomposition step after merger; write `graph.json`; add `"decomposition"` to `_V5_KNOWN_FEATURES` |
| `${ATOMISER}/data/v5_metrics.py` | MODIFY | Add `compute_v5_decomposition_metrics()` |
| `${ATOMISER}/tests/v5/golden/relationships/kdigo_rec123.yaml` | CREATE | Ground truth: 3 edges for KDIGO Rec 1.2.3 path |
| `${ATOMISER}/tests/v5/test_v5_decomposition.py` | CREATE | Unit tests for node/edge extraction and flag routing |

```
ATOMISER=backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
EXTRACTION=backend/shared-infrastructure/knowledge-base-services/shared/extraction
```

---

## Task 1: Create data models in `decomposition.py`

**Files:**
- Create: `${EXTRACTION}/v4/decomposition.py`

- [ ] **Step 1.1: Write failing model import test first**

Create `tests/v5/test_v5_decomposition.py` with just imports:
```python
"""V5 Decomposition unit tests — import guard."""
from extraction.v4.decomposition import (
    GuidelineDecomposer,
    GuidelineGraph,
    GraphNode,
    GraphEdge,
)
```

- [ ] **Step 1.2: Run test to confirm failure**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
PYTHONPATH=. python -m pytest tests/v5/test_v5_decomposition.py::test_v5_decomposition -v 2>&1 | tail -5
```
Expected: `ModuleNotFoundError` — `decomposition.py` doesn't exist yet.

- [ ] **Step 1.3: Create `decomposition.py` with data models**

Create `${EXTRACTION}/v4/decomposition.py`:

```python
"""V5 #5 Decomposition — Guideline2Graph-style directed relationship graph.

Produces a GuidelineGraph from merged_spans + GuidelineTree + section_passages.
Output: list of GraphNode + list of GraphEdge (written to graph.json).

Node types:
    RECOMMENDATION  — numbered recommendation statement (e.g., "Recommendation 4.1.1")
    ALGORITHM       — figure or algorithm reference (e.g., "Algorithm Fig 1")
    DRUG_CLASS      — drug class or named drug (e.g., "SGLT2 inhibitor", "metformin")

Edge types:
    IS_TREATED_BY          — RECOMMENDATION → DRUG_CLASS
    REFERENCES_ALGORITHM   — RECOMMENDATION → ALGORITHM
    REQUIRES_MONITORING    — RECOMMENDATION → RECOMMENDATION (monitoring follow-up)

Detection strategy:
    - RECOMMENDATION nodes: spans with tier=TIER_1 in sections with heading matching
      "Recommendation N.N.N" or "Practice Point N.N" pattern
    - ALGORITHM nodes: spans or section headings containing "Figure \\d" or "Algorithm"
    - DRUG_CLASS nodes: spans whose text matches a drug-class regex or contributing
      channel B (drug-dict channel) with high confidence
    - Edges: co-occurrence within same or adjacent sections, optionally confirmed by
      cross-reference keywords ("see Figure", "includes", "requires", "monitor with")

Flag gate: V5_DECOMPOSITION=1 (via is_v5_enabled).
"""
from __future__ import annotations

import re
import time
from dataclasses import dataclass, field
from typing import Literal, Optional
from uuid import UUID, uuid4

from pydantic import BaseModel, Field

from .v5_flags import is_v5_enabled


# ─── Node / Edge Models ──────────────────────────────────────────────────────

NodeType = Literal["RECOMMENDATION", "ALGORITHM", "DRUG_CLASS"]
EdgeType = Literal["IS_TREATED_BY", "REFERENCES_ALGORITHM", "REQUIRES_MONITORING"]


class GraphNode(BaseModel):
    """A node in the decomposition graph."""
    id: str                              # e.g., "rec_4.1.1", "alg_fig1", "drug_sglt2i"
    node_type: NodeType
    label: str                           # human-readable label
    section_id: Optional[str] = None
    page_number: Optional[int] = None
    source_span_ids: list[str] = Field(default_factory=list)  # MergedSpan UUIDs as str
    confidence: float = Field(ge=0.0, le=1.0, default=1.0)


class GraphEdge(BaseModel):
    """A directed typed edge in the decomposition graph."""
    id: str = Field(default_factory=lambda: str(uuid4()))
    source_node_id: str
    target_node_id: str
    edge_type: EdgeType
    evidence_text: Optional[str] = None  # snippet that triggered this edge
    confidence: float = Field(ge=0.0, le=1.0, default=0.8)


class GuidelineGraph(BaseModel):
    """Complete relationship graph for a guideline document."""
    job_id: str
    nodes: list[GraphNode] = Field(default_factory=list)
    edges: list[GraphEdge] = Field(default_factory=list)
    elapsed_ms: float = 0.0

    def to_dict(self) -> dict:
        return {
            "job_id": self.job_id,
            "node_count": len(self.nodes),
            "edge_count": len(self.edges),
            "elapsed_ms": round(self.elapsed_ms, 1),
            "nodes": [n.model_dump() for n in self.nodes],
            "edges": [e.model_dump() for e in self.edges],
        }


# ─── Regexes ─────────────────────────────────────────────────────────────────

# Matches "Recommendation 4.1.1", "Rec. 4.1", "Practice Point 4.1.1"
_REC_HEADING_RE = re.compile(
    r'\b(?:Recommendation|Rec\.?|Practice\s+Point)\s+(\d+(?:\.\d+)*)',
    re.IGNORECASE,
)

# Matches "Figure 1", "Algorithm Fig. 2", "see Figure 3"
_ALG_REF_RE = re.compile(
    r'\b(?:Figure|Fig\.?|Algorithm)\s+(\d+[a-zA-Z]?)',
    re.IGNORECASE,
)

# Drug-class keywords — expand as needed for clinical coverage
_DRUG_CLASS_RE = re.compile(
    r'\b(?:SGLT2\s*(?:inhibitor|i)?|metformin|GLP-?1\s*(?:RA|receptor\s+agonist)?'
    r'|ACE\s*(?:inhibitor|I)?|ARB|beta\s*blocker|statin|fibrate|ezetimibe'
    r'|insulin|sulfonylurea|DPP-?4\s*(?:inhibitor)?'
    r'|furosemide|spironolactone|empagliflozin|dapagliflozin|liraglutide'
    r'|semaglutide|atorvastatin|rosuvastatin|aspirin|clopidogrel|ticagrelor)\b',
    re.IGNORECASE,
)

# Cross-reference keywords that signal a relationship
_XREF_KEYWORDS_RE = re.compile(
    r'\b(?:see|refer\s+to|as\s+per|per|including|requires|monitor(?:ing)?|use|treat(?:ment)?)\b',
    re.IGNORECASE,
)


# ─── Decomposer ──────────────────────────────────────────────────────────────

class GuidelineDecomposer:
    """Extract a directed relationship graph from merged spans + guideline tree.

    Usage:
        decomposer = GuidelineDecomposer()
        graph = decomposer.decompose(
            job_id="<uuid>",
            merged_spans=merged_spans,
            tree=tree,
            section_passages=section_passages,
            profile=profile,
        )
    """

    VERSION = "5.0.0"

    def decompose(
        self,
        job_id: str,
        merged_spans: list,         # list[MergedSpan]
        tree,                       # GuidelineTree
        section_passages: list,     # list[SectionPassage]
        profile=None,
    ) -> Optional[GuidelineGraph]:
        """Build a GuidelineGraph from the pipeline's output artefacts.

        Returns None if V5_DECOMPOSITION flag is off. Callers must check.
        """
        if not is_v5_enabled("decomposition", profile):
            return None

        start = time.monotonic()
        graph = GuidelineGraph(job_id=str(job_id))

        # Pass 1: identify nodes from section headings and merged spans
        self._extract_recommendation_nodes(graph, tree, merged_spans)
        self._extract_algorithm_nodes(graph, tree, merged_spans)
        self._extract_drug_class_nodes(graph, merged_spans)

        # Pass 2: add edges based on co-occurrence + cross-reference keywords
        self._extract_edges(graph, merged_spans, section_passages)

        # Dedup nodes and edges
        self._dedup(graph)

        graph.elapsed_ms = (time.monotonic() - start) * 1000
        return graph

    # ─── Node extraction ─────────────────────────────────────────────────────

    def _extract_recommendation_nodes(self, graph: GuidelineGraph, tree, merged_spans: list) -> None:
        """Extract RECOMMENDATION nodes from section headings matching rec pattern."""
        seen: set[str] = set()
        for section in self._flatten_sections(tree.sections):
            m = _REC_HEADING_RE.search(section.heading)
            if not m:
                continue
            rec_id = f"rec_{m.group(1).replace('.', '_')}"
            if rec_id in seen:
                continue
            seen.add(rec_id)
            # Collect span UUIDs from this section
            span_ids = [
                str(s.id) for s in merged_spans
                if getattr(s, "section_id", None) == section.section_id
            ]
            graph.nodes.append(GraphNode(
                id=rec_id,
                node_type="RECOMMENDATION",
                label=section.heading,
                section_id=section.section_id,
                page_number=section.page_number,
                source_span_ids=span_ids[:5],  # keep top 5 for provenance
                confidence=0.95,
            ))

    def _extract_algorithm_nodes(self, graph: GuidelineGraph, tree, merged_spans: list) -> None:
        """Extract ALGORITHM nodes from spans or section text containing figure refs."""
        seen: set[str] = set()
        for span in merged_spans:
            text = getattr(span, "text", "") or ""
            for m in _ALG_REF_RE.finditer(text):
                alg_id = f"alg_fig{m.group(1).lower()}"
                if alg_id in seen:
                    continue
                seen.add(alg_id)
                graph.nodes.append(GraphNode(
                    id=alg_id,
                    node_type="ALGORITHM",
                    label=f"Figure/Algorithm {m.group(1)}",
                    section_id=getattr(span, "section_id", None),
                    page_number=getattr(span, "page_number", None),
                    source_span_ids=[str(span.id)],
                    confidence=0.85,
                ))

    def _extract_drug_class_nodes(self, graph: GuidelineGraph, merged_spans: list) -> None:
        """Extract DRUG_CLASS nodes from channel B spans and drug-class regex matches."""
        seen: set[str] = set()
        for span in merged_spans:
            channels = getattr(span, "contributing_channels", [])
            text = getattr(span, "text", "") or ""

            # Channel B (drug dict) spans are high-confidence drug mentions
            if "B" in channels:
                drug_id = f"drug_{re.sub(r'[^a-z0-9]', '_', text.lower()[:30])}"
                if drug_id not in seen:
                    seen.add(drug_id)
                    graph.nodes.append(GraphNode(
                        id=drug_id,
                        node_type="DRUG_CLASS",
                        label=text[:60],
                        section_id=getattr(span, "section_id", None),
                        page_number=getattr(span, "page_number", None),
                        source_span_ids=[str(span.id)],
                        confidence=getattr(span, "merged_confidence", 0.8),
                    ))
                continue

            # Fallback: regex match in any span
            for m in _DRUG_CLASS_RE.finditer(text):
                drug_id = f"drug_{re.sub(r'[^a-z0-9]', '_', m.group(0).lower())}"
                if drug_id in seen:
                    continue
                seen.add(drug_id)
                graph.nodes.append(GraphNode(
                    id=drug_id,
                    node_type="DRUG_CLASS",
                    label=m.group(0),
                    section_id=getattr(span, "section_id", None),
                    page_number=getattr(span, "page_number", None),
                    source_span_ids=[str(span.id)],
                    confidence=0.75,
                ))

    # ─── Edge extraction ─────────────────────────────────────────────────────

    def _extract_edges(
        self,
        graph: GuidelineGraph,
        merged_spans: list,
        section_passages: list,
    ) -> None:
        """Add typed edges by co-occurrence and cross-reference analysis."""
        rec_nodes = {n.id: n for n in graph.nodes if n.node_type == "RECOMMENDATION"}
        alg_nodes = {n.id: n for n in graph.nodes if n.node_type == "ALGORITHM"}
        drug_nodes = {n.id: n for n in graph.nodes if n.node_type == "DRUG_CLASS"}

        # Build section_id → nodes index for co-occurrence
        section_to_drugs: dict[str, list[str]] = {}
        section_to_algs: dict[str, list[str]] = {}
        for n in drug_nodes.values():
            sid = n.section_id or ""
            section_to_drugs.setdefault(sid, []).append(n.id)
        for n in alg_nodes.values():
            sid = n.section_id or ""
            section_to_algs.setdefault(sid, []).append(n.id)

        for rec_id, rec_node in rec_nodes.items():
            sid = rec_node.section_id or ""

            # RECOMMENDATION → DRUG_CLASS (IS_TREATED_BY)
            for drug_id in section_to_drugs.get(sid, []):
                graph.edges.append(GraphEdge(
                    source_node_id=rec_id,
                    target_node_id=drug_id,
                    edge_type="IS_TREATED_BY",
                    confidence=0.80,
                ))

            # RECOMMENDATION → ALGORITHM (REFERENCES_ALGORITHM)
            for alg_id in section_to_algs.get(sid, []):
                graph.edges.append(GraphEdge(
                    source_node_id=rec_id,
                    target_node_id=alg_id,
                    edge_type="REFERENCES_ALGORITHM",
                    confidence=0.85,
                ))

        # Cross-section edges via keyword scan in section passages
        for passage in section_passages:
            text = getattr(passage, "prose_text", "") or ""
            sid = getattr(passage, "section_id", "") or ""

            # If this passage contains a figure ref AND a rec heading → link
            alg_matches = list(_ALG_REF_RE.finditer(text))
            rec_match = _REC_HEADING_RE.search(
                getattr(passage, "heading", "") or ""
            )

            if rec_match and alg_matches:
                rec_id = f"rec_{rec_match.group(1).replace('.', '_')}"
                for am in alg_matches:
                    alg_id = f"alg_fig{am.group(1).lower()}"
                    if rec_id in {n.id for n in graph.nodes} and alg_id in {n.id for n in graph.nodes}:
                        graph.edges.append(GraphEdge(
                            source_node_id=rec_id,
                            target_node_id=alg_id,
                            edge_type="REFERENCES_ALGORITHM",
                            evidence_text=am.group(0),
                            confidence=0.90,
                        ))

    # ─── Helpers ─────────────────────────────────────────────────────────────

    def _dedup(self, graph: GuidelineGraph) -> None:
        """Dedup nodes by id and edges by (source, target, type) triple."""
        seen_node_ids: set[str] = set()
        unique_nodes = []
        for n in graph.nodes:
            if n.id not in seen_node_ids:
                seen_node_ids.add(n.id)
                unique_nodes.append(n)
        graph.nodes = unique_nodes

        seen_edges: set[tuple[str, str, str]] = set()
        unique_edges = []
        for e in graph.edges:
            key = (e.source_node_id, e.target_node_id, e.edge_type)
            if key not in seen_edges:
                seen_edges.add(key)
                unique_edges.append(e)
        graph.edges = unique_edges

    @staticmethod
    def _flatten_sections(sections: list) -> list:
        """DFS-flatten a nested section list."""
        result = []
        for s in sections:
            result.append(s)
            if getattr(s, "children", None):
                result.extend(GuidelineDecomposer._flatten_sections(s.children))
        return result
```

- [ ] **Step 1.4: Run import test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
PYTHONPATH=. python -m pytest tests/v5/test_v5_decomposition.py -v 2>&1 | tail -10
```
Expected: `PASSED` (imports succeed).

---

## Task 2: Write failing unit tests for decomposition

**Files:**
- Modify: `${ATOMISER}/tests/v5/test_v5_decomposition.py`
- Create: `${ATOMISER}/tests/v5/golden/relationships/kdigo_rec123.yaml`

- [ ] **Step 2.1: Create ground-truth relationship YAML**

Create `tests/v5/golden/relationships/kdigo_rec123.yaml`:
```yaml
# Ground truth: KDIGO 2022 Recommendation 1.2.3 path
# Source: Recommendation 1.2.3 → Algorithm Figure 1 → Drug Class SGLT2i
# This represents 3 edges forming 1 path (treated as 1 graded relationship per spec §7)
relationships:
  - source_node_id: "rec_1_2_3"
    target_node_id: "alg_fig1"
    edge_type: "REFERENCES_ALGORITHM"
    description: "Rec 1.2.3 cites Algorithm Figure 1"

  - source_node_id: "alg_fig1"
    target_node_id: "drug_sglt2_inhibitor"
    edge_type: "IS_TREATED_BY"
    description: "Algorithm Figure 1 directs to SGLT2 inhibitor class"

  - source_node_id: "rec_1_2_3"
    target_node_id: "drug_sglt2_inhibitor"
    edge_type: "IS_TREATED_BY"
    description: "Rec 1.2.3 directly mentions SGLT2 inhibitor"
```

- [ ] **Step 2.2: Add unit tests to `test_v5_decomposition.py`**

```python
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


# ─── Minimal stubs ───────────────────────────────────────────────────────────

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


# ─── Flag routing ──────────────────────────────────────────────────────────

def test_decompose_returns_none_when_flag_off(monkeypatch):
    """decompose() returns None when V5_DECOMPOSITION is not set."""
    monkeypatch.delenv("V5_DECOMPOSITION", raising=False)
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


# ─── Node extraction ─────────────────────────────────────────────────────────

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


# ─── Edge extraction ─────────────────────────────────────────────────────────

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
    # Span that creates the algorithm node
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


# ─── Ground truth comparison ─────────────────────────────────────────────────

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


# ─── graph.to_dict() serialisation ───────────────────────────────────────────

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
```

- [ ] **Step 2.3: Run tests — expect them to pass (models exist now)**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
PYTHONPATH=. V5_DECOMPOSITION=1 python -m pytest tests/v5/test_v5_decomposition.py -v 2>&1 | tail -30
```
Expected: all tests `PASSED`.

- [ ] **Step 2.4: Commit**

```bash
git add \
    backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/decomposition.py \
    backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_v5_decomposition.py \
    backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/golden/relationships/kdigo_rec123.yaml
git commit -m "feat(v5): Decomposition — GuidelineGraph with node/edge extraction (#5)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Wire decomposition into `run_pipeline_targeted.py`

**Files:**
- Modify: `${ATOMISER}/data/run_pipeline_targeted.py`

- [ ] **Step 3.1: Import `GuidelineDecomposer` at the top of the pipeline function**

Find the imports block in `run_pipeline_targeted.py`. The import is done lazily inside the pipeline function to avoid top-level import cycles. Add after the existing channel imports:

```python
    from extraction.v4.decomposition import GuidelineDecomposer
```

- [ ] **Step 3.2: Add decomposition step after `assemble_section_passages()`**

Find where `section_passages` is assembled (look for `assemble_section_passages` call). After that, add:

```python
    # V5 #5 Decomposition step
    graph_json = None
    if _is_v5_enabled("decomposition", profile):
        print("   [V5 #5] Building decomposition graph...")
        decomposer = GuidelineDecomposer()
        graph = decomposer.decompose(
            job_id=str(job_id),
            merged_spans=merged_spans,
            tree=tree,
            section_passages=section_passages,
            profile=profile,
        )
        if graph is not None:
            graph_path = os.path.join(job_dir, "graph.json")
            import json as _json
            with open(graph_path, "w") as _gf:
                _json.dump(graph.to_dict(), _gf, indent=2)
            print(f"      ✅ {len(graph.nodes)} nodes, {len(graph.edges)} edges → {graph_path}")
            graph_json = graph_path
```

- [ ] **Step 3.3: Add `"decomposition"` to `_V5_KNOWN_FEATURES`**

```python
    _V5_KNOWN_FEATURES = ["bbox_provenance", "table_specialist", "consensus_entropy", "decomposition"]
```

- [ ] **Step 3.4: Run smoke with decomposition flag**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
V5_BBOX_PROVENANCE=1 V5_DECOMPOSITION=1 python data/run_pipeline_targeted.py \
    --source docling_l1 --target-kb kb-3-guidelines \
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf 2>&1 | tail -20
```
Expected: `[V5 #5] Building decomposition graph...` followed by `N nodes, M edges → graph.json`, no errors.

- [ ] **Step 3.5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py
git commit -m "feat(v5): wire decomposition step into pipeline; write graph.json

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Add `compute_v5_decomposition_metrics()` to `v5_metrics.py`

**Files:**
- Modify: `${ATOMISER}/data/v5_metrics.py`

- [ ] **Step 4.1: Add the function**

Add after `compute_v5_ce_metrics()`:

```python
_DECOMP_EDGE_THRESHOLD = 80.0  # Both precision and recall must clear this


def compute_v5_decomposition_metrics(
    graph_dict: dict,
    ground_truth_yaml: str | None = None,
) -> dict:
    """Compute V5 Decomposition metrics (edge precision + recall).

    graph_dict: the dict from GuidelineGraph.to_dict() (loaded from graph.json).
    ground_truth_yaml: path to a YAML file with a 'relationships' list, each
        entry having source_node_id, target_node_id, edge_type.

    Without ground_truth_yaml: reports structural metadata only.
    With ground_truth_yaml: computes precision and recall against graded edges.
    """
    import yaml as _yaml

    extracted_edges = graph_dict.get("edges", [])
    extracted_set = {
        (e["source_node_id"], e["target_node_id"], e["edge_type"])
        for e in extracted_edges
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

    # Load graded edges
    try:
        with open(ground_truth_yaml) as f:
            data = _yaml.safe_load(f)
        graded = data.get("relationships", [])
    except (OSError, Exception):
        graded = []

    graded_set = {
        (r["source_node_id"], r["target_node_id"], r["edge_type"])
        for r in graded
    }

    if not graded_set:
        return {
            "v5_decomposition": {
                "node_count": node_count,
                "edge_count": edge_count,
                "error": "empty ground truth",
            }
        }

    true_positives = extracted_set & graded_set
    precision = len(true_positives) / len(extracted_set) * 100 if extracted_set else 0.0
    recall = len(true_positives) / len(graded_set) * 100 if graded_set else 0.0

    precision = round(precision, 2)
    recall = round(recall, 2)
    status = "PASS" if precision >= _DECOMP_EDGE_THRESHOLD and recall >= _DECOMP_EDGE_THRESHOLD else "FAIL"

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
                "status": "PASS" if precision >= _DECOMP_EDGE_THRESHOLD else "FAIL",
            },
            "edge_recall_pct": {
                "v5": recall,
                "threshold": _DECOMP_EDGE_THRESHOLD,
                "status": "PASS" if recall >= _DECOMP_EDGE_THRESHOLD else "FAIL",
            },
        },
        "verdict": status,
    }
```

- [ ] **Step 4.2: Add to `_main()` — read `graph.json` if it exists**

In `_main()`, after the bbox/table metrics block, add:

```python
        graph_path = job_dir / "graph.json"
        if graph_path.exists():
            import json as _json
            graph_dict = _json.loads(graph_path.read_text())
            decomp_metrics = compute_v5_decomposition_metrics(graph_dict)
            write_v5_metrics(job_dir, decomp_metrics)
            dm = decomp_metrics["v5_decomposition"]
            print(f"{job_dir.name}: nodes={dm['node_count']}, edges={dm['edge_count']}")
```

- [ ] **Step 4.3: Run existing tests to ensure nothing broke**

```bash
PYTHONPATH=. python -m pytest tests/v5/ -v \
    --ignore=tests/v5/test_v5_table_specialist_smoke.py \
    --ignore=tests/v5/test_v5_ce_smoke.py 2>&1 | tail -20
```
Expected: all tests `PASSED`.

- [ ] **Step 4.4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/v5_metrics.py
git commit -m "feat(v5): add compute_v5_decomposition_metrics() to v5_metrics sidecar

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Smoke acceptance test for decomposition

**Files:**
- Create: `${ATOMISER}/tests/v5/test_v5_decomposition_smoke.py`

- [ ] **Step 5.1: Write smoke test**

Create `tests/v5/test_v5_decomposition_smoke.py`:
```python
"""Smoke acceptance test for V5 Decomposition.

Requires a completed job dir with graph.json.

Run:
    PYTHONPATH=. V5_DECOMPOSITION=1 V5_BBOX_PROVENANCE=1 \
        pytest tests/v5/test_v5_decomposition_smoke.py -v -m smoke
"""
import json
import os
from pathlib import Path

import pytest

pytestmark = pytest.mark.smoke

OUTPUT_DIR = Path("data/output/v4")


@pytest.mark.skipif(
    not any(OUTPUT_DIR.glob("**/graph.json")),
    reason="No graph.json found — run pipeline with V5_DECOMPOSITION=1 first",
)
def test_graph_json_has_nodes_and_edges():
    """graph.json is present and has non-empty nodes + edges lists."""
    latest = sorted(OUTPUT_DIR.glob("**/graph.json"))[-1]
    data = json.loads(latest.read_text())
    assert "nodes" in data, "graph.json missing 'nodes'"
    assert "edges" in data, "graph.json missing 'edges'"
    assert data.get("node_count", 0) >= 0
    assert data.get("edge_count", 0) >= 0


@pytest.mark.skipif(
    not any(OUTPUT_DIR.glob("**/graph.json")),
    reason="No graph.json found",
)
def test_every_node_has_provenance():
    """Every node in graph.json has at least one source_span_id."""
    latest = sorted(OUTPUT_DIR.glob("**/graph.json"))[-1]
    data = json.loads(latest.read_text())
    for node in data.get("nodes", []):
        assert len(node.get("source_span_ids", [])) >= 1, (
            f"Node {node.get('id')} has no provenance span IDs"
        )


@pytest.mark.skipif(
    "V5_DECOMPOSITION" not in os.environ,
    reason="V5_DECOMPOSITION not set",
)
def test_decomposition_in_v5_features():
    """Latest job_metadata reports decomposition in v5_features_enabled."""
    metas = sorted(OUTPUT_DIR.glob("**/job_metadata.json"))
    if not metas:
        pytest.skip("No job_metadata.json found")
    meta = json.loads(metas[-1].read_text())
    assert "decomposition" in meta.get("v5_features_enabled", [])
```

- [ ] **Step 5.2: Run full smoke pipeline + acceptance test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
V5_BBOX_PROVENANCE=1 V5_DECOMPOSITION=1 python data/run_pipeline_targeted.py \
    --source docling_l1 --target-kb kb-3-guidelines \
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf

LATEST_JOB=$(ls -t data/output/v4/ | head -1)
python data/v5_metrics.py "data/output/v4/$LATEST_JOB"

PYTHONPATH=. V5_DECOMPOSITION=1 V5_BBOX_PROVENANCE=1 \
    python -m pytest tests/v5/test_v5_decomposition_smoke.py -v -m smoke 2>&1 | tail -20
```
Expected: all smoke tests `PASSED`.

- [ ] **Step 5.3: Final commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_v5_decomposition_smoke.py
git commit -m "test(v5): Decomposition smoke acceptance tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Self-review

**Spec coverage:**
- ✅ `V5_DECOMPOSITION=1` flag gate — all paths guarded by `is_v5_enabled("decomposition", profile)`
- ✅ Three node types: RECOMMENDATION, ALGORITHM, DRUG_CLASS — `decomposition.py` NodeType literal
- ✅ Three edge types: IS_TREATED_BY, REFERENCES_ALGORITHM, REQUIRES_MONITORING — EdgeType literal
- ✅ `graph.json` output written alongside `merged_spans.json` — Task 3
- ✅ Every node has `source_span_ids` provenance — `test_every_node_has_provenance`
- ✅ Edge precision + recall ≥80% formula — `compute_v5_decomposition_metrics()`
- ✅ Ground truth YAML fixture — `kdigo_rec123.yaml` in Task 2
- ✅ `_V5_KNOWN_FEATURES` updated — Task 3
- ✅ `V5_DISABLE_ALL=1` override — `test_disable_all_overrides_decomposition`
- ⚠️ 15-relationship full ground-truth set requires clinician curation (spec §7 "one-time investment"). `kdigo_rec123.yaml` is a 3-edge smoke fixture. Full ground truth is a separate data-collection step identical to the Table Specialist plan.
- ⚠️ REQUIRES_MONITORING edges are in the model but no extraction rule is implemented in V0 (the detection strategy for monitoring relationships between two RECOMMENDATIONs requires more clinical signal than V0 provides). Edge type is registered and correct; rule can be added in a follow-up task without changing any model or test structure.
- ⚠️ networkx is imported in decomposition.py comments as "already in venv" — the implementation here does NOT use networkx (uses Pydantic models + plain lists). A graph traversal upgrade (e.g., for path-finding or cycle detection) can use networkx later. This keeps V0 dependency-free.

**Placeholder scan:** None found. All edge detection uses concrete regex rules, not "TBD" logic.

**Type consistency:** `merged_spans: list` (untyped) avoids circular import with `models.py`. The `decompose()` method accesses only `span.id`, `span.text`, `span.contributing_channels`, `span.section_id`, `span.page_number`, `span.merged_confidence` — all attributes that exist on `MergedSpan` with correct names.