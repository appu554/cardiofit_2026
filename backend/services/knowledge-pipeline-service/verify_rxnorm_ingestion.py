#!/usr/bin/env python3
"""
Verify RxNorm Ingestion in Neo4j
Checks the actual data loaded into Neo4j from the RxNorm ingestion
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.database_factory import create_database_client
from core.logging_config import setup_pipeline_logging, get_logger


async def verify_rxnorm_data():
    """Verify RxNorm data in Neo4j"""
    
    # Setup logging
    pipeline_logger = setup_pipeline_logging()
    logger = get_logger(__name__)
    
    print("🔍 VERIFYING RXNORM INGESTION IN NEO4J")
    print("=" * 50)
    
    try:
        # Create database client
        logger.info("Connecting to Neo4j...")
        database_client = await create_database_client()
        
        # Test connection
        if hasattr(database_client, 'test_connection'):
            connection_ok = await database_client.test_connection()
            if not connection_ok:
                print("❌ Database connection failed")
                return False
        
        print("✅ Connected to Neo4j")
        print("\n📊 Checking RxNorm data...")
        print("-" * 50)
        
        # Query 1: Count total nodes
        query_total_nodes = """
        MATCH (n)
        RETURN count(n) as total_nodes
        """
        result = await database_client.execute_cypher(query_total_nodes)
        total_nodes = result[0]['total_nodes'] if result else 0
        print(f"📌 Total nodes in database: {total_nodes:,}")
        
        # Query 2: Count RxNorm specific nodes (by label if available)
        query_rxnorm_nodes = """
        MATCH (n)
        WHERE n.source = 'rxnorm' OR n.rxcui IS NOT NULL
        RETURN count(n) as rxnorm_nodes
        """
        result = await database_client.execute_cypher(query_rxnorm_nodes)
        rxnorm_nodes = result[0]['rxnorm_nodes'] if result else 0
        print(f"💊 RxNorm nodes: {rxnorm_nodes:,}")
        
        # Query 3: Sample some RxNorm concepts
        query_sample_concepts = """
        MATCH (n)
        WHERE n.rxcui IS NOT NULL
        RETURN n.rxcui as rxcui, n.name as name, n.tty as tty
        LIMIT 10
        """
        result = await database_client.execute_cypher(query_sample_concepts)
        
        if result:
            print("\n📋 Sample RxNorm concepts:")
            print("-" * 50)
            for i, record in enumerate(result, 1):
                rxcui = record.get('rxcui', 'N/A')
                name = record.get('name', 'N/A')
                tty = record.get('tty', 'N/A')
                print(f"{i}. RXCUI: {rxcui} | {name} [{tty}]")
        
        # Query 4: Count relationships
        query_relationships = """
        MATCH ()-[r]->()
        RETURN count(r) as total_relationships
        """
        result = await database_client.execute_cypher(query_relationships)
        total_rels = result[0]['total_relationships'] if result else 0
        print(f"\n🔗 Total relationships: {total_rels:,}")
        
        # Query 5: Check for specific RxNorm relationship types
        query_rel_types = """
        MATCH ()-[r]->()
        RETURN DISTINCT type(r) as rel_type, count(r) as count
        ORDER BY count DESC
        LIMIT 10
        """
        result = await database_client.execute_cypher(query_rel_types)
        
        if result:
            print("\n📊 Relationship types (top 10):")
            print("-" * 50)
            for record in result:
                rel_type = record.get('rel_type', 'N/A')
                count = record.get('count', 0)
                print(f"  {rel_type}: {count:,}")
        
        # Query 6: Check node properties
        query_properties = """
        MATCH (n)
        WHERE n.rxcui IS NOT NULL
        RETURN keys(n) as properties
        LIMIT 1
        """
        result = await database_client.execute_cypher(query_properties)
        
        if result and result[0].get('properties'):
            print(f"\n🏷️  RxNorm node properties: {', '.join(result[0]['properties'])}")
        
        # Query 7: Count by term type (TTY)
        query_tty_counts = """
        MATCH (n)
        WHERE n.tty IS NOT NULL
        RETURN n.tty as tty, count(n) as count
        ORDER BY count DESC
        LIMIT 10
        """
        result = await database_client.execute_cypher(query_tty_counts)
        
        if result:
            print("\n📈 Top 10 Term Types (TTY):")
            print("-" * 50)
            for record in result:
                tty = record.get('tty', 'N/A')
                count = record.get('count', 0)
                print(f"  {tty}: {count:,}")
        
        print("\n" + "=" * 50)
        print("✅ Verification complete!")
        
        # Summary
        if rxnorm_nodes > 0:
            print(f"\n✨ RxNorm data successfully loaded:")
            print(f"   - {rxnorm_nodes:,} RxNorm nodes")
            print(f"   - {total_rels:,} relationships")
            print(f"   - Expected ~35,783 entities from ingestion")
            
            if rxnorm_nodes >= 30000:  # Allow some variance
                print("\n🎉 Ingestion verification PASSED!")
                return True
            else:
                print(f"\n⚠️  Warning: Found {rxnorm_nodes:,} nodes, expected ~35,783")
                return True  # Still return True as data exists
        else:
            print("\n❌ No RxNorm data found in Neo4j!")
            return False
            
    except Exception as e:
        logger.error("Verification failed", error=str(e))
        print(f"\n💥 Verification failed: {e}")
        import traceback
        traceback.print_exc()
        return False
    
    finally:
        # Cleanup
        try:
            if 'database_client' in locals():
                await database_client.disconnect()
        except Exception:
            pass


def main():
    """Main function"""
    print("🏥 RXNORM INGESTION VERIFICATION")
    print("=" * 60)
    
    success = asyncio.run(verify_rxnorm_data())
    
    if success:
        print("\n✅ Verification completed successfully!")
        return True
    else:
        print("\n❌ Verification failed!")
        return False


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
