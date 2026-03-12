"""
Knowledge Graph Service for CAE Engine

Service layer for accessing clinical knowledge from Neo4j with caching,
optimized for clinical reasoning queries.
"""

from typing import List, Dict, Any, Optional
import logging
from .neo4j_client import Neo4jCloudClient
from .query_cache import Neo4jQueryCache

logger = logging.getLogger(__name__)

class KnowledgeGraphService:
    """Service layer for accessing clinical knowledge from Neo4j"""

    def __init__(self):
        self.client = Neo4jCloudClient()
        self.cache = Neo4jQueryCache(default_ttl=300)  # 5 minutes
        self.logger = logging.getLogger(__name__)

    async def initialize(self):
        """Initialize the knowledge graph service"""
        connected = await self.client.connect()
        if connected:
            self.logger.info("Knowledge Graph Service initialized with Neo4j client")
            return True
        else:
            self.logger.error("Failed to connect to Neo4j")
            return False
    
    async def query_with_cache(self, query: str, parameters: Dict[str, Any] = None,
                              cache_ttl: int = None) -> List[Dict[str, Any]]:
        """Execute query with caching"""

        # Try cache first
        cached_result = await self.cache.get(query, parameters)
        if cached_result is not None:
            return cached_result

        # Execute query using existing Neo4j client
        result = await self.client.execute_cypher(query, parameters)

        # Cache result
        await self.cache.set(query, parameters, result, cache_ttl)

        return result
    
    async def get_drug_interactions(self, drug_names: List[str]) -> List[Dict[str, Any]]:
        """Get drug-drug interactions for given drugs"""
        if not drug_names or len(drug_names) < 2:
            return []

        # Capitalize drug names to match Neo4j data
        capitalized_names = [name.capitalize() for name in drug_names]

        query = """
        MATCH (d1:cae_Drug)-[r:cae_interactsWith]-(d2:cae_Drug)
        WHERE d1.name IN $drug_names AND d2.name IN $drug_names
        RETURN d1.name as drug1, d2.name as drug2,
               'major' as severity, 'unknown' as mechanism,
               'potential interaction' as clinical_effect, 'monitor closely' as management
        """

        return await self.query_with_cache(query, {'drug_names': capitalized_names}, cache_ttl=600)
    
    async def get_adverse_events(self, drug_names: List[str]) -> List[Dict[str, Any]]:
        """Get adverse events for given drugs"""
        if not drug_names:
            return []

        # Capitalize drug names to match Neo4j data
        capitalized_names = [name.capitalize() for name in drug_names]

        query = """
        MATCH (d:cae_Drug)-[:cae_hasAdverseEvent]->(ae:cae_AdverseEvent)
        WHERE d.name IN $drug_names AND ae.serious = 1
        RETURN d.name as drug_name, ae.reaction as reaction,
               'serious' as outcome, ae.country as country
        LIMIT 50
        """

        return await self.query_with_cache(query, {'drug_names': capitalized_names}, cache_ttl=300)
    
    async def get_contraindications(self, drug_names: List[str],
                                  conditions: List[str]) -> List[Dict[str, Any]]:
        """Get contraindications for drugs and conditions"""
        if not drug_names or not conditions:
            return []

        # This relationship doesn't exist in your Neo4j database yet
        # Return empty list for now - can be populated later
        return []

    async def get_dosing_adjustments(self, drug_names: List[str],
                                   patient_factors: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Get dosing adjustments based on patient factors"""
        if not drug_names:
            return []

        # This relationship doesn't exist in your Neo4j database yet
        # Return empty list for now - can be populated later
        return []

    async def get_cache_stats(self) -> Dict[str, Any]:
        """Get cache performance statistics"""
        return self.cache.get_stats()

    async def close(self):
        """Close the knowledge graph service"""
        await self.client.disconnect()
