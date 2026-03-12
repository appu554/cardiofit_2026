"""
GraphDB to Neo4j Adapter - Compatibility Shim
=============================================

⚠️  This module is DEPRECATED and exists only for backward compatibility.

The Python-based SPARQL→Cypher transformation has been replaced by neosemantics (n10s),
which provides native RDF import directly into Neo4j.

New Implementation:
- Use: neo4j_sync_service.py with n10s integration
- See: n10s_rdf_importer.py for n10s-specific operations

This shim re-exports the deprecated class to maintain backward compatibility
with existing code. New code should NOT import from this module.

@deprecated Use Neo4jTerminologySyncService instead
"""

import warnings
warnings.warn(
    "Importing from graphdb_neo4j_adapter is deprecated. "
    "Use Neo4jTerminologySyncService from services.neo4j_sync_service instead. "
    "The new n10s-based approach is faster and more reliable.",
    DeprecationWarning,
    stacklevel=2
)

# Re-export for backward compatibility (with deprecation warning already issued)
from .graphdb_neo4j_adapter_deprecated import GraphDBToNeo4jAdapter

# Also provide alternative import path
GraphDBNeo4jAdapter = GraphDBToNeo4jAdapter  # Alias for compatibility

__all__ = ['GraphDBToNeo4jAdapter', 'GraphDBNeo4jAdapter']
