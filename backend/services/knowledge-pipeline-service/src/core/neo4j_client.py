"""
Neo4j Cloud (AuraDB) Client for Clinical Knowledge Graph
Provides connection and operations for Neo4j Cloud database
"""

import asyncio
import structlog
from typing import Dict, List, Optional, Any, AsyncGenerator
from datetime import datetime
import json
from pathlib import Path

from neo4j import AsyncGraphDatabase, AsyncDriver, AsyncSession
from neo4j.exceptions import ServiceUnavailable, AuthError, ClientError

from core.config import settings


logger = structlog.get_logger(__name__)


class Neo4jCloudClient:
    """Neo4j Cloud (AuraDB) client for clinical knowledge graph operations"""
    
    def __init__(self):
        self.driver: Optional[AsyncDriver] = None
        self.connected = False
        
        # Neo4j Cloud configuration
        self.uri = getattr(settings, 'NEO4J_URI', 'neo4j+s://52721fa5.databases.neo4j.io')
        self.username = getattr(settings, 'NEO4J_USERNAME', 'neo4j')
        self.password = getattr(settings, 'NEO4J_PASSWORD', '')
        self.database = getattr(settings, 'NEO4J_DATABASE', 'neo4j')

        # Connection settings optimized for cloud
        self.max_connection_lifetime = 3600  # 1 hour
        self.max_connection_pool_size = 50
        self.connection_acquisition_timeout = 60

        logger.info("Neo4j Cloud client initialized",
                   uri=self.uri,
                   database=self.database,
                   username=self.username)
    
    async def connect(self) -> bool:
        """Connect to Neo4j Cloud (AuraDB)"""
        try:
            if self.connected and self.driver:
                return True

            logger.info("Connecting to Neo4j Cloud (AuraDB)...")

            # Create driver with cloud-optimized settings
            # For neo4j+s:// URIs, encryption is handled by the URI scheme
            self.driver = AsyncGraphDatabase.driver(
                self.uri,
                auth=(self.username, self.password),
                max_connection_lifetime=self.max_connection_lifetime,
                max_connection_pool_size=self.max_connection_pool_size,
                connection_acquisition_timeout=self.connection_acquisition_timeout
                # Note: encrypted=True is not needed for neo4j+s:// URIs
            )

            # Test connection
            if await self.test_connection():
                self.connected = True
                logger.info("✅ Neo4j Cloud connection established successfully")
                return True
            else:
                logger.error("❌ Neo4j Cloud connection test failed")
                return False
                
        except AuthError as e:
            logger.error("❌ Neo4j Cloud authentication failed", error=str(e))
            return False
        except ServiceUnavailable as e:
            logger.error("❌ Neo4j Cloud service unavailable", error=str(e))
            return False
        except Exception as e:
            logger.error("❌ Neo4j Cloud connection failed", error=str(e))
            return False
    
    async def test_connection(self) -> bool:
        """Test Neo4j Cloud connection"""
        try:
            if not self.driver:
                return False
            
            async with self.driver.session(database=self.database) as session:
                result = await session.run("RETURN 1 as test")
                record = await result.single()
                
                if record and record["test"] == 1:
                    logger.info("Neo4j Cloud connection test successful")
                    return True
                else:
                    logger.error("Neo4j Cloud connection test returned unexpected result")
                    return False
                    
        except Exception as e:
            logger.error("Neo4j Cloud connection test failed", error=str(e))
            return False
    
    async def disconnect(self):
        """Disconnect from Neo4j Cloud"""
        try:
            if self.driver:
                await self.driver.close()
                self.driver = None
                self.connected = False
                logger.info("Neo4j Cloud connection closed")
        except Exception as e:
            logger.error("Error closing Neo4j Cloud connection", error=str(e))
    
    async def execute_cypher(self, cypher_query: str, parameters: Dict = None) -> List[Dict]:
        """Execute Cypher query on Neo4j Cloud"""
        try:
            if not self.connected or not self.driver:
                if not await self.connect():
                    raise Exception("Cannot connect to Neo4j Cloud")
            
            async with self.driver.session(database=self.database) as session:
                result = await session.run(cypher_query, parameters or {})
                records = await result.data()
                return records
                
        except ClientError as e:
            logger.error("Neo4j Cloud Cypher query error", 
                        query=cypher_query[:100] + "..." if len(cypher_query) > 100 else cypher_query,
                        error=str(e))
            raise
        except Exception as e:
            logger.error("Neo4j Cloud query execution failed", error=str(e))
            raise
    
    async def create_indexes(self):
        """Create indexes for clinical knowledge graph"""
        indexes = [
            # Drug-related indexes
            "CREATE INDEX drug_rxcui IF NOT EXISTS FOR (d:Drug) ON (d.rxcui)",
            "CREATE INDEX drug_name IF NOT EXISTS FOR (d:Drug) ON (d.name)",
            
            # SNOMED CT indexes
            "CREATE INDEX snomed_concept_id IF NOT EXISTS FOR (s:SNOMEDConcept) ON (s.conceptId)",
            "CREATE INDEX snomed_fsn IF NOT EXISTS FOR (s:SNOMEDConcept) ON (s.fullySpecifiedName)",
            
            # LOINC indexes
            "CREATE INDEX loinc_code IF NOT EXISTS FOR (l:LOINCCode) ON (l.loincNumber)",
            "CREATE INDEX loinc_name IF NOT EXISTS FOR (l:LOINCCode) ON (l.longCommonName)",
            
            # Clinical entity indexes
            "CREATE INDEX condition_code IF NOT EXISTS FOR (c:Condition) ON (c.code)",
            "CREATE INDEX observation_code IF NOT EXISTS FOR (o:Observation) ON (o.code)",
            
            # Relationship indexes for performance
            "CREATE INDEX interaction_severity IF NOT EXISTS FOR ()-[r:INTERACTS_WITH]-() ON (r.severity)",
            "CREATE INDEX mapping_confidence IF NOT EXISTS FOR ()-[r:MAPS_TO]-() ON (r.confidence)"
        ]
        
        logger.info("Creating Neo4j Cloud indexes for clinical knowledge graph...")
        
        for index_query in indexes:
            try:
                await self.execute_cypher(index_query)
                logger.debug("Index created", query=index_query)
            except Exception as e:
                logger.warning("Index creation failed", query=index_query, error=str(e))
        
        logger.info("✅ Neo4j Cloud indexes creation completed")
    
    async def clear_database(self):
        """Clear all data from Neo4j Cloud database (use with caution!)"""
        try:
            logger.warning("🚨 Clearing Neo4j Cloud database - this will delete ALL data!")
            
            # Delete all relationships first
            await self.execute_cypher("MATCH ()-[r]-() DELETE r")
            
            # Delete all nodes
            await self.execute_cypher("MATCH (n) DELETE n")
            
            logger.info("✅ Neo4j Cloud database cleared")
            
        except Exception as e:
            logger.error("Failed to clear Neo4j Cloud database", error=str(e))
            raise
    
    async def get_database_stats(self) -> Dict[str, Any]:
        """Get Neo4j Cloud database statistics"""
        try:
            stats_queries = {
                "total_nodes": "MATCH (n) RETURN count(n) as count",
                "total_relationships": "MATCH ()-[r]-() RETURN count(r) as count",
                "node_labels": "CALL db.labels() YIELD label RETURN collect(label) as labels",
                "relationship_types": "CALL db.relationshipTypes() YIELD relationshipType RETURN collect(relationshipType) as types",
                "database_info": "CALL dbms.components() YIELD name, versions, edition RETURN name, versions, edition"
            }
            
            stats = {}
            for stat_name, query in stats_queries.items():
                try:
                    result = await self.execute_cypher(query)
                    if result:
                        if stat_name in ["total_nodes", "total_relationships"]:
                            stats[stat_name] = result[0]["count"]
                        elif stat_name in ["node_labels", "relationship_types"]:
                            stats[stat_name] = result[0][list(result[0].keys())[0]]
                        else:
                            stats[stat_name] = result
                except Exception as e:
                    logger.warning(f"Failed to get {stat_name}", error=str(e))
                    stats[stat_name] = "unavailable"
            
            return stats
            
        except Exception as e:
            logger.error("Failed to get Neo4j Cloud database stats", error=str(e))
            return {}
    
    async def batch_create_nodes(self, nodes: List[Dict], batch_size: int = 1000):
        """Create nodes in batches for better performance"""
        try:
            total_nodes = len(nodes)
            logger.info(f"Creating {total_nodes} nodes in Neo4j Cloud in batches of {batch_size}")
            
            for i in range(0, total_nodes, batch_size):
                batch = nodes[i:i + batch_size]
                
                # Group nodes by label for efficient creation
                nodes_by_label = {}
                for node in batch:
                    label = node.get('label', 'Node')
                    if label not in nodes_by_label:
                        nodes_by_label[label] = []
                    nodes_by_label[label].append(node)
                
                # Create nodes for each label
                for label, label_nodes in nodes_by_label.items():
                    cypher = f"""
                    UNWIND $nodes as node
                    CREATE (n:{label})
                    SET n = node.properties
                    """
                    
                    await self.execute_cypher(cypher, {"nodes": label_nodes})
                
                logger.debug(f"Created batch {i//batch_size + 1}/{(total_nodes-1)//batch_size + 1}")
            
            logger.info(f"✅ Successfully created {total_nodes} nodes in Neo4j Cloud")
            
        except Exception as e:
            logger.error("Failed to batch create nodes in Neo4j Cloud", error=str(e))
            raise
    
    async def batch_create_relationships(self, relationships: List[Dict], batch_size: int = 1000):
        """Create relationships in batches for better performance"""
        try:
            total_rels = len(relationships)
            logger.info(f"Creating {total_rels} relationships in Neo4j Cloud in batches of {batch_size}")
            
            for i in range(0, total_rels, batch_size):
                batch = relationships[i:i + batch_size]
                
                cypher = """
                UNWIND $relationships as rel
                MATCH (a {id: rel.from_id}), (b {id: rel.to_id})
                CALL apoc.create.relationship(a, rel.type, rel.properties, b) YIELD rel as r
                RETURN count(r)
                """
                
                await self.execute_cypher(cypher, {"relationships": batch})
                logger.debug(f"Created relationship batch {i//batch_size + 1}/{(total_rels-1)//batch_size + 1}")
            
            logger.info(f"✅ Successfully created {total_rels} relationships in Neo4j Cloud")
            
        except Exception as e:
            logger.error("Failed to batch create relationships in Neo4j Cloud", error=str(e))
            raise
