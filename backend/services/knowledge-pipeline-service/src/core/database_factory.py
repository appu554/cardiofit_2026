"""
Database Factory for Clinical Knowledge Graph
Provides unified interface for GraphDB and Neo4j Cloud
"""

import structlog
from typing import Union, Protocol, runtime_checkable
from abc import ABC, abstractmethod

from core.config import settings
from core.graphdb_client import GraphDBClient
from core.neo4j_client import Neo4jCloudClient
from core.neo4j_ingester_adapter import Neo4jCompatibleClient


logger = structlog.get_logger(__name__)


@runtime_checkable
class DatabaseClient(Protocol):
    """Protocol for database clients"""
    
    async def connect(self) -> bool:
        """Connect to database"""
        ...
    
    async def disconnect(self):
        """Disconnect from database"""
        ...
    
    async def test_connection(self) -> bool:
        """Test database connection"""
        ...


class DatabaseFactory:
    """Factory for creating database clients based on configuration"""
    
    @staticmethod
    def create_client() -> Union[GraphDBClient, Neo4jCloudClient]:
        """Create appropriate database client based on configuration"""
        
        database_type = settings.DATABASE_TYPE.lower()
        
        if database_type == "neo4j":
            logger.info("🌐 Creating Neo4j Cloud client with GraphDB compatibility")
            neo4j_client = Neo4jCloudClient()
            return Neo4jCompatibleClient(neo4j_client)
        
        elif database_type == "graphdb":
            logger.info("🗄️ Creating GraphDB client")
            return GraphDBClient()
        
        else:
            logger.warning(f"Unknown database type: {database_type}, defaulting to Neo4j Cloud")
            return Neo4jCloudClient()
    
    @staticmethod
    def get_database_info() -> dict:
        """Get information about the configured database"""
        database_type = settings.DATABASE_TYPE.lower()
        
        if database_type == "neo4j":
            return {
                "type": "Neo4j Cloud (AuraDB)",
                "uri": settings.NEO4J_URI,
                "database": settings.NEO4J_DATABASE,
                "username": settings.NEO4J_USERNAME,
                "features": [
                    "Managed cloud service",
                    "Auto-scaling",
                    "High availability",
                    "Enterprise security",
                    "Cypher query language",
                    "Graph algorithms"
                ]
            }
        
        elif database_type == "graphdb":
            return {
                "type": "GraphDB",
                "endpoint": settings.GRAPHDB_ENDPOINT,
                "repository": settings.GRAPHDB_REPOSITORY,
                "username": settings.GRAPHDB_USERNAME,
                "features": [
                    "RDF/SPARQL support",
                    "Semantic reasoning",
                    "Ontology management",
                    "SHACL validation",
                    "Full-text search",
                    "Geospatial queries"
                ]
            }
        
        else:
            return {
                "type": "Unknown",
                "error": f"Unsupported database type: {database_type}"
            }


class UnifiedDatabaseAdapter:
    """Unified adapter that provides common interface for both database types"""
    
    def __init__(self):
        self.client = DatabaseFactory.create_client()
        self.database_type = settings.DATABASE_TYPE.lower()
        
        logger.info("Unified database adapter initialized",
                   database_type=self.database_type,
                   client_type=type(self.client).__name__)
    
    async def connect(self) -> bool:
        """Connect to the configured database"""
        return await self.client.connect()
    
    async def disconnect(self):
        """Disconnect from the database"""
        await self.client.disconnect()
    
    async def test_connection(self) -> bool:
        """Test database connection"""
        return await self.client.test_connection()
    
    async def initialize_schema(self):
        """Initialize database schema/indexes"""
        if self.database_type == "neo4j":
            await self.client.create_indexes()
        elif self.database_type == "graphdb":
            # GraphDB schema initialization if needed
            pass
    
    async def clear_database(self):
        """Clear all data from database"""
        if hasattr(self.client, 'clear_database'):
            await self.client.clear_database()
        else:
            logger.warning("Clear database not supported for this client type")
    
    async def get_stats(self) -> dict:
        """Get database statistics"""
        if hasattr(self.client, 'get_database_stats'):
            return await self.client.get_database_stats()
        elif hasattr(self.client, 'get_repository_stats'):
            return await self.client.get_repository_stats()
        else:
            return {"error": "Statistics not available for this database type"}
    
    async def execute_query(self, query: str, parameters: dict = None):
        """Execute database query (unified interface)"""
        if self.database_type == "neo4j":
            return await self.client.execute_cypher(query, parameters)
        elif self.database_type == "graphdb":
            return await self.client.execute_sparql_query(query)
        else:
            raise NotImplementedError(f"Query execution not implemented for {self.database_type}")
    
    async def batch_insert_triples(self, triples: list, batch_size: int = 1000):
        """Insert RDF triples in batches (for GraphDB)"""
        if self.database_type == "graphdb":
            await self.client.batch_insert_triples(triples, batch_size)
        else:
            logger.warning("RDF triple insertion not supported for Neo4j - use Cypher instead")
    
    async def batch_create_nodes(self, nodes: list, batch_size: int = 1000):
        """Create nodes in batches (for Neo4j)"""
        if self.database_type == "neo4j":
            await self.client.batch_create_nodes(nodes, batch_size)
        else:
            logger.warning("Node creation not supported for GraphDB - use RDF triples instead")
    
    async def batch_create_relationships(self, relationships: list, batch_size: int = 1000):
        """Create relationships in batches (for Neo4j)"""
        if self.database_type == "neo4j":
            await self.client.batch_create_relationships(relationships, batch_size)
        else:
            logger.warning("Relationship creation not supported for GraphDB - use RDF triples instead")
    
    def get_client(self):
        """Get the underlying database client"""
        return self.client
    
    def get_database_info(self) -> dict:
        """Get database configuration information"""
        return DatabaseFactory.get_database_info()


# Convenience functions
async def create_database_client():
    """Create and connect to database client"""
    # Use the factory to create the appropriate client
    client = DatabaseFactory.create_client()

    # Connect to the database
    if hasattr(client, 'connect'):
        if await client.connect():
            logger.info("✅ Database connection established successfully")
            return client
        else:
            logger.error("❌ Failed to connect to database")
            raise Exception("Database connection failed")
    else:
        # For clients that don't have async connect, assume they're ready
        logger.info("✅ Database client created successfully")
        return client


async def validate_database_connection():
    """Validate database connection and return status"""
    try:
        adapter = UnifiedDatabaseAdapter()
        
        if await adapter.connect():
            stats = await adapter.get_stats()
            await adapter.disconnect()
            
            return {
                "status": "connected",
                "database_info": adapter.get_database_info(),
                "stats": stats
            }
        else:
            return {
                "status": "failed",
                "database_info": adapter.get_database_info(),
                "error": "Connection test failed"
            }
    
    except Exception as e:
        return {
            "status": "error",
            "error": str(e)
        }


# Export main classes and functions
__all__ = [
    'DatabaseFactory',
    'UnifiedDatabaseAdapter', 
    'DatabaseClient',
    'create_database_client',
    'validate_database_connection'
]
