"""
Vaidshala Phase 4: Extraction Pipeline

Deterministic extraction of clinical guideline content.
LLM used ONLY for genuinely ambiguous cases (<5% of content).

Modules:
    - table_extractor: COR/LOE table extraction from PDFs
    - temporal_extractor: Temporal constraint extraction for KB-3
    - kb15_formatter: Evidence metadata formatting for KB-15
"""

from .table_extractor import (
    GuidelineTableExtractor,
    KB15Formatter,
    KB3TemporalFormatter,
    RecommendationRow,
    TemporalConstraint,
    ClassOfRecommendation,
    LevelOfEvidence,
    TemporalConstraintType
)

__all__ = [
    'GuidelineTableExtractor',
    'KB15Formatter',
    'KB3TemporalFormatter',
    'RecommendationRow',
    'TemporalConstraint',
    'ClassOfRecommendation',
    'LevelOfEvidence',
    'TemporalConstraintType'
]

__version__ = '1.0.0'
