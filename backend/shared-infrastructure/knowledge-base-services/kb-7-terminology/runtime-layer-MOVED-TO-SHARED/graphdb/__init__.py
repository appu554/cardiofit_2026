"""
GraphDB Client Module for KB7 Neo4j Dual-Stream & Service Runtime Layer

This module provides comprehensive GraphDB connectivity and SPARQL operations
for clinical knowledge extraction and reasoning.
"""

from .client import (
    GraphDBClient,
    SPARQLQueryType,
    SPARQLResult,
    DrugInteraction,
    DrugContraindication
)

__all__ = [
    'GraphDBClient',
    'SPARQLQueryType',
    'SPARQLResult',
    'DrugInteraction',
    'DrugContraindication'
]

__version__ = "1.0.0"