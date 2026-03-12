"""
CardioFit Multi-KB Query Router
Intelligent routing layer for all CardioFit Knowledge Bases
"""

from .multi_kb_query_router import (
    MultiKBQueryRouter,
    MultiKBQueryRequest,
    MultiKBQueryResponse,
    QueryPattern,
    DataSource
)

from .performance_monitor import PerformanceMonitor
from .cache_coordinator import CacheCoordinator
from .fallback_handler import FallbackHandler

__version__ = "1.0.0"
__all__ = [
    "MultiKBQueryRouter",
    "MultiKBQueryRequest",
    "MultiKBQueryResponse",
    "QueryPattern",
    "DataSource",
    "PerformanceMonitor",
    "CacheCoordinator",
    "FallbackHandler"
]