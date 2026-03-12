"""
Neo4j Client for CAE Engine

Optimized Neo4j client for clinical knowledge graph queries with connection pooling,
async operations, and performance monitoring.
"""

import os
import logging
from typing import List, Dict, Any, Optional
from neo4j import AsyncGraphDatabase, AsyncDriver
from neo4j.exceptions import ClientError, ServiceUnavailable, AuthError

logger = logging.getLogger(__name__)

class Neo4jCloudClient:
    """Neo4j Cloud (AuraDB) client for CAE Engine clinical queries"""
    
    def __init__(self):
        self.driver: Optional[AsyncDriver] = None
        self.connected = False
        
        # Neo4j Cloud configuration from environment
        self.uri = os.getenv('NEO4J_URI', 'neo4j+s://52721fa5.databases.neo4j.io')
        self.username = os.getenv('NEO4J_USERNAME', 'neo4j')
        self.password = os.getenv('NEO4J_PASSWORD', '')
        self.database = os.getenv('NEO4J_DATABASE', 'neo4j')
        
        # Connection settings optimized for cloud
        self.max_connection_lifetime = int(os.getenv('NEO4J_MAX_CONNECTION_LIFETIME', '3600'))
        self.max_connection_pool_size = int(os.getenv('NEO4J_MAX_CONNECTION_POOL_SIZE', '50'))
        self.connection_acquisition_timeout = int(os.getenv('NEO4J_CONNECTION_ACQUISITION_TIMEOUT', '60'))
        
        logger.info("Neo4j Cloud client initialized for CAE Engine",
                   extra={'uri': self.uri, 'database': self.database, 'username': self.username})
    
    async def connect(self) -> bool:
        """Connect to Neo4j Cloud (AuraDB)"""
        try:
            if self.connected and self.driver:
                return True
            
            logger.info("Connecting to Neo4j Cloud (AuraDB) for CAE Engine...")
            
            # Create driver with cloud-optimized settings
            self.driver = AsyncGraphDatabase.driver(
                self.uri,
                auth=(self.username, self.password),
                max_connection_lifetime=self.max_connection_lifetime,
                max_connection_pool_size=self.max_connection_pool_size,
                connection_acquisition_timeout=self.connection_acquisition_timeout
            )
            
            # Test connection
            if await self.test_connection():
                self.connected = True
                logger.info("✅ Neo4j Cloud connection established successfully for CAE Engine")
                return True
            else:
                logger.error("❌ Neo4j Cloud connection test failed for CAE Engine")
                return False
                
        except AuthError as e:
            logger.error(f"❌ Neo4j Cloud authentication failed for CAE Engine: {e}")
            return False
        except ServiceUnavailable as e:
            logger.error(f"❌ Neo4j Cloud service unavailable for CAE Engine: {e}")
            return False
        except Exception as e:
            logger.error(f"❌ Neo4j Cloud connection failed for CAE Engine: {e}")
            return False
    
    async def test_connection(self) -> bool:
        """Test Neo4j connection"""
        try:
            if not self.driver:
                return False
                
            async with self.driver.session(database=self.database) as session:
                result = await session.run("RETURN 'CAE Engine Connection Test' as test")
                record = await result.single()
                return record is not None
                
        except Exception as e:
            logger.error(f"Neo4j connection test failed for CAE Engine: {e}")
            return False
    
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
            logger.error(f"Neo4j Cloud Cypher query error for CAE Engine: {e}",
                        extra={'query': cypher_query[:100] + "..." if len(cypher_query) > 100 else cypher_query})
            raise
        except Exception as e:
            logger.error(f"Neo4j Cloud query execution failed for CAE Engine: {e}")
            raise
    
    async def disconnect(self):
        """Disconnect from Neo4j Cloud"""
        try:
            if self.driver:
                await self.driver.close()
                self.connected = False
                logger.info("Neo4j Cloud connection closed for CAE Engine")
        except Exception as e:
            logger.error(f"Error closing Neo4j connection for CAE Engine: {e}")
    
    def get_connection_info(self) -> Dict[str, Any]:
        """Get connection information"""
        return {
            'uri': self.uri,
            'database': self.database,
            'username': self.username,
            'connected': self.connected,
            'max_connection_pool_size': self.max_connection_pool_size,
            'connection_acquisition_timeout': self.connection_acquisition_timeout
        }
