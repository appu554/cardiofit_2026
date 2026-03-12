"""
Self-Improving Intelligence Layer for Clinical Assertion Engine

This module provides adaptive intelligence capabilities including:
- Dynamic rule learning and adaptation
- Performance optimization systems
- Confidence evolution algorithms
- Pattern-based rule generation
"""

from .rule_engine import SelfImprovingRuleEngine
from .performance_optimizer import PerformanceOptimizer
from .confidence_evolver import ConfidenceEvolver
from .pattern_learner import PatternLearner

__all__ = [
    'SelfImprovingRuleEngine',
    'PerformanceOptimizer', 
    'ConfidenceEvolver',
    'PatternLearner'
]
