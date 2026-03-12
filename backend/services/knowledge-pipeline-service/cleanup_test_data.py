"""
Clean up test data from Neo4j
"""

import asyncio
import sys
from pathlib import Path

# Add the src directory to the Python path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from core.database_factory import create_database_client
from core.neo4j_client import Neo4jCloudClient
import structlog

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.dev.ConsoleRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)


async def cleanup_test_data():
    """Remove test data from Neo4j"""
    
    print("🧹 CLEANING UP TEST DATA FROM NEO4J")
    print("=" * 60)
    
    try:
        # Create database client
        client = await create_database_client()
        
        # Check current data
        print("\n📊 Current database state:")
        result = await client.execute_cypher("MATCH (n) RETURN count(n) as count")
        node_count = result[0]['count'] if result else 0
        print(f"   Total nodes: {node_count}")
        
        # Delete all nodes and relationships
        print("\n🗑️  Deleting all nodes and relationships...")
        await client.execute_cypher("MATCH (n) DETACH DELETE n")
        
        # Verify deletion
        result = await client.execute_cypher("MATCH (n) RETURN count(n) as count")
        new_count = result[0]['count'] if result else 0
        print(f"\n✅ Cleanup complete!")
        print(f"   Nodes remaining: {new_count}")
        
        await client.disconnect()
        
    except Exception as e:
        logger.error("Cleanup failed", error=str(e))
        print(f"\n❌ Error: {e}")


if __name__ == "__main__":
    asyncio.run(cleanup_test_data())
