"""
Vaidshala Phase 5: Constrained Atomiser

LLM-assisted extraction for genuine guideline gaps only.
All output: DRAFT status + mandatory SME review + confidence cap at 0.85

Philosophy: Deterministic First, LLM as Last Resort
"""

from .constrained_atomiser import (
    ConstrainedAtomiser,
    AtomiserConfig,
    AtomisedRecommendation
)
from .atomiser_registry import AtomiserRegistry, CQLEntry
from .kb_router import KBRouter, KnowledgeBase, RoutingResult
from .phase4_integration import Phase4Integrator, IntegrationResult

__all__ = [
    # Core Atomiser
    'ConstrainedAtomiser',
    'AtomiserConfig',
    'AtomisedRecommendation',
    # Registry
    'AtomiserRegistry',
    'CQLEntry',
    # KB Routing
    'KBRouter',
    'KnowledgeBase',
    'RoutingResult',
    # Phase 4 Integration
    'Phase4Integrator',
    'IntegrationResult',
]
