"""
GraphDB Semantic Manager
Multi-KB semantic operations with GraphDB integration
"""

from .multi_kb_graphdb_manager import (
    MultiKBGraphDBManager,
    SPARQLQuery,
    SPARQLQueryType,
    SPARQLResult,
    GraphDBRepository
)

__all__ = [
    'MultiKBGraphDBManager',
    'SPARQLQuery',
    'SPARQLQueryType',
    'SPARQLResult',
    'GraphDBRepository'
]