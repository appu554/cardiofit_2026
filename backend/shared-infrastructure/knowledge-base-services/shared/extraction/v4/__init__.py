"""
V4.1 Multi-Channel Extraction Module (Hybrid Architecture).

Replaces V3's single GLiNER NER (L2) with a multi-channel extraction pipeline:
  Channel 0: Text Normalizer (ligature/OCR/symbol fix)
  Channel A: Structural Oracle (Granite-Docling 258M + Marker alignment + regex fallback)
  Channel B: Aho-Corasick drug dictionary
  Channel C: Grammar/regex patterns (drug-agnostic)
  Channel D: Table decomposer — dual-source: pipe tables + OTSL tables
  Channel E: GLiNER residual booster (full text, no truncation)
  Channel F: NuExtract 2.0-8B via Ollama sidecar (prose only, temperature=0)
  Signal Merger -> DB -> Reviewer -> Dossier Assembly -> L3

V4.1 Hybrid Architecture:
  L1: Marker (text-of-record) + Granite-Docling (structural oracle)
  Channel A aligns DocTags headings to Marker ATX headings
  Channel D routes by table.source: "marker_pipe" | "granite_otsl"
  Channel F calls Ollama on host (Metal GPU) instead of in-process model

Pipeline Position:
    L1 (Marker) -> L2 (V4.1 Multi-Channel) -> Reviewer -> Dossier -> L3 (Claude)
"""

from .models import (
    RawSpan,
    MergedSpan,
    ReviewerDecision,
    VerifiedSpan,
    DrugDossier,
    DossierResult,
    GuidelineTree,
    GuidelineSection,
    SectionPassage,
    TableBoundary,
    ChannelOutput,
    ChannelStatus,
)
from .guideline_profile import GuidelineProfile
from .tiering_classifier import (
    TieringClassifier,
    RuleBasedTieringClassifier,
    TrainedTieringClassifier,
    TieringResult,
)
from .channel_g_sentence import ChannelG, ChannelGSentence
from .channel_h_recovery import ChannelH, ChannelHRecovery
from .range_integrity_engine import RangeIntegrityEngine, RangeIntegrityReport

__all__ = [
    "RawSpan",
    "MergedSpan",
    "ReviewerDecision",
    "VerifiedSpan",
    "DrugDossier",
    "DossierResult",
    "GuidelineTree",
    "GuidelineSection",
    "SectionPassage",
    "TableBoundary",
    "ChannelOutput",
    "ChannelStatus",
    "GuidelineProfile",
    "TieringClassifier",
    "RuleBasedTieringClassifier",
    "TrainedTieringClassifier",
    "TieringResult",
    "ChannelG",
    "ChannelGSentence",
    "ChannelH",
    "ChannelHRecovery",
    "RangeIntegrityEngine",
    "RangeIntegrityReport",
]

__version__ = "4.3.0"
