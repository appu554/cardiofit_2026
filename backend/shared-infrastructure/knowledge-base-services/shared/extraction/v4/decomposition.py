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
from typing import Literal, Optional
from uuid import uuid4

from pydantic import BaseModel, Field

from .v5_flags import is_v5_enabled


# ─── Node / Edge Models ──────────────────────────────────────────────────────

NodeType = Literal["RECOMMENDATION", "ALGORITHM", "DRUG_CLASS"]
EdgeType = Literal["IS_TREATED_BY", "REFERENCES_ALGORITHM", "REQUIRES_MONITORING"]


class GraphNode(BaseModel):
    """A node in the decomposition graph."""
    id: str
    node_type: NodeType
    label: str
    section_id: Optional[str] = None
    page_number: Optional[int] = None
    source_span_ids: list[str] = Field(default_factory=list)
    confidence: float = Field(ge=0.0, le=1.0, default=1.0)


class GraphEdge(BaseModel):
    """A directed typed edge in the decomposition graph."""
    id: str = Field(default_factory=lambda: str(uuid4()))
    source_node_id: str
    target_node_id: str
    edge_type: EdgeType
    evidence_text: Optional[str] = None
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

_REC_HEADING_RE = re.compile(
    r'\b(?:Recommendation|Rec\.?|Practice\s+Point)\s+(\d+(?:\.\d+)*)',
    re.IGNORECASE,
)

_ALG_REF_RE = re.compile(
    r'\b(?:Figure|Fig\.?|Algorithm)\s+(\d+[a-zA-Z]?)',
    re.IGNORECASE,
)

_DRUG_CLASS_RE = re.compile(
    r'\b(?:SGLT2\s*(?:inhibitor|i)?|metformin|GLP-?1\s*(?:RA|receptor\s+agonist)?'
    r'|ACE\s*(?:inhibitor|I)?|ARB|beta\s*blocker|statin|fibrate|ezetimibe'
    r'|insulin|sulfonylurea|DPP-?4\s*(?:inhibitor)?'
    r'|furosemide|spironolactone|empagliflozin|dapagliflozin|liraglutide'
    r'|semaglutide|atorvastatin|rosuvastatin|aspirin|clopidogrel|ticagrelor)\b',
    re.IGNORECASE,
)

_XREF_KEYWORDS_RE = re.compile(
    r'\b(?:see|refer\s+to|as\s+per|per|including|requires|monitor(?:ing)?|use|treat(?:ment)?)\b',
    re.IGNORECASE,
)


# ─── Decomposer ──────────────────────────────────────────────────────────────

class GuidelineDecomposer:
    """Extract a directed relationship graph from merged spans + guideline tree."""

    VERSION = "5.0.0"

    def decompose(
        self,
        job_id: str,
        merged_spans: list,
        tree,
        section_passages: list,
        profile=None,
    ) -> Optional[GuidelineGraph]:
        """Build a GuidelineGraph. Returns None if V5_DECOMPOSITION flag is off."""
        if not is_v5_enabled("decomposition", profile):
            return None

        start = time.monotonic()
        graph = GuidelineGraph(job_id=str(job_id))

        self._extract_recommendation_nodes(graph, tree, merged_spans)
        self._extract_algorithm_nodes(graph, tree, merged_spans)
        self._extract_drug_class_nodes(graph, merged_spans)
        self._extract_edges(graph, merged_spans, section_passages)
        self._dedup(graph)

        graph.elapsed_ms = (time.monotonic() - start) * 1000
        return graph

    def _extract_recommendation_nodes(self, graph: GuidelineGraph, tree, merged_spans: list) -> None:
        seen: set[str] = set()
        for section in self._flatten_sections(tree.sections):
            m = _REC_HEADING_RE.search(section.heading)
            if not m:
                continue
            rec_id = f"rec_{m.group(1).replace('.', '_')}"
            if rec_id in seen:
                continue
            seen.add(rec_id)
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
                source_span_ids=span_ids[:5],
                confidence=0.95,
            ))

    def _extract_algorithm_nodes(self, graph: GuidelineGraph, tree, merged_spans: list) -> None:
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
        seen: set[str] = set()
        for span in merged_spans:
            channels = getattr(span, "contributing_channels", [])
            text = getattr(span, "text", "") or ""

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

    def _extract_edges(self, graph: GuidelineGraph, merged_spans: list, section_passages: list) -> None:
        # REQUIRES_MONITORING (RECOMMENDATION → RECOMMENDATION) detection requires
        # clinical signal not available in V0 (e.g., eGFR threshold cross-references).
        # The edge type is declared in EdgeType for future extension; no edges are
        # emitted here for it in V0.
        rec_nodes = {n.id: n for n in graph.nodes if n.node_type == "RECOMMENDATION"}
        alg_nodes = {n.id: n for n in graph.nodes if n.node_type == "ALGORITHM"}
        drug_nodes = {n.id: n for n in graph.nodes if n.node_type == "DRUG_CLASS"}

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
            for drug_id in section_to_drugs.get(sid, []):
                graph.edges.append(GraphEdge(
                    source_node_id=rec_id,
                    target_node_id=drug_id,
                    edge_type="IS_TREATED_BY",
                    confidence=0.80,
                ))
            for alg_id in section_to_algs.get(sid, []):
                graph.edges.append(GraphEdge(
                    source_node_id=rec_id,
                    target_node_id=alg_id,
                    edge_type="REFERENCES_ALGORITHM",
                    confidence=0.85,
                ))

        node_ids = {n.id for n in graph.nodes}
        for passage in section_passages:
            text = getattr(passage, "prose_text", "") or ""
            alg_matches = list(_ALG_REF_RE.finditer(text))
            rec_match = _REC_HEADING_RE.search(getattr(passage, "heading", "") or "")
            if rec_match and alg_matches:
                rec_id = f"rec_{rec_match.group(1).replace('.', '_')}"
                for am in alg_matches:
                    alg_id = f"alg_fig{am.group(1).lower()}"
                    if rec_id in node_ids and alg_id in node_ids:
                        graph.edges.append(GraphEdge(
                            source_node_id=rec_id,
                            target_node_id=alg_id,
                            edge_type="REFERENCES_ALGORITHM",
                            evidence_text=am.group(0),
                            confidence=0.90,
                        ))

    def _dedup(self, graph: GuidelineGraph) -> None:
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
        result = []
        for s in sections:
            result.append(s)
            if getattr(s, "children", None):
                result.extend(GuidelineDecomposer._flatten_sections(s.children))
        return result
