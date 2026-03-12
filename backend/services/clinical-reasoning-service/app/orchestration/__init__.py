"""
Clinical Assertion Engine - Orchestration Layer

This module provides the orchestration layer for the CAE, including:
- Request routing and priority classification
- Parallel execution of clinical reasoners
- Decision aggregation and conflict resolution
- Performance optimization and caching
"""

from .request_router import RequestRouter
from .parallel_executor import ParallelExecutor
from .decision_aggregator import DecisionAggregator
from .priority_queue import PriorityQueue

__all__ = [
    'RequestRouter',
    'ParallelExecutor', 
    'DecisionAggregator',
    'PriorityQueue'
]
