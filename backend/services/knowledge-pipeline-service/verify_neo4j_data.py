#!/usr/bin/env python3
"""
Verify Neo4j Cloud data after ingestion
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.database_factory import create_database_client

async def verify_data():
    """Verify the data in Neo4j Cloud"""
    print("🔍 VERIFYING NEO4J CLOUD DATA")
    print("=" * 50)
    
    try:
        # Create database client
        database_client = await create_database_client()
        connection_ok = await database_client.test_connection()
        if not connection_ok:
            print("❌ Database connection failed")
            return False
        
        print("✅ Connected to Neo4j Cloud")
        
        # Check RxNorm drugs
        result = await database_client.execute_cypher("MATCH (d:cae_Drug) RETURN count(d) as count")
        drug_count = result[0]['count'] if result else 0
        print(f"💊 RxNorm drugs: {drug_count:,} nodes")
        
        # Check SNOMED concepts
        result = await database_client.execute_cypher("MATCH (s:cae_SNOMEDConcept) RETURN count(s) as count")
        snomed_count = result[0]['count'] if result else 0
        print(f"🧬 SNOMED CT concepts: {snomed_count:,} nodes")
        
        # Check LOINC concepts
        result = await database_client.execute_cypher("MATCH (l:cae_LOINCConcept) RETURN count(l) as count")
        loinc_count = result[0]['count'] if result else 0
        print(f"🧪 LOINC concepts: {loinc_count:,} nodes")
        
        # Check relationships
        result = await database_client.execute_cypher("MATCH ()-[r:cae_hasSNOMEDCTMapping]->() RETURN count(r) as count")
        rel_count = result[0]['count'] if result else 0
        print(f"🔗 Cross-relationships: {rel_count:,} edges")
        
        # Total nodes
        result = await database_client.execute_cypher("MATCH (n) RETURN count(n) as count")
        total_nodes = result[0]['count'] if result else 0
        print(f"📊 Total nodes: {total_nodes:,}")
        
        # Total relationships
        result = await database_client.execute_cypher("MATCH ()-[r]->() RETURN count(r) as count")
        total_rels = result[0]['count'] if result else 0
        print(f"🔗 Total relationships: {total_rels:,}")
        
        print("\n" + "=" * 50)
        print(f"🎯 TOTAL RECORDS: {total_nodes + total_rels:,}")
        
        # Sample some data
        print("\n📋 SAMPLE DATA:")
        print("-" * 30)
        
        # Sample RxNorm drugs
        result = await database_client.execute_cypher("MATCH (d:cae_Drug) RETURN d.rxcui, d.name LIMIT 3")
        if result:
            print("💊 Sample RxNorm drugs:")
            for row in result:
                print(f"   - {row['d.rxcui']}: {row['d.name']}")
        
        # Sample SNOMED concepts
        result = await database_client.execute_cypher("MATCH (s:cae_SNOMEDConcept) RETURN s.concept_id LIMIT 3")
        if result:
            print("🧬 Sample SNOMED concepts:")
            for row in result:
                print(f"   - {row['s.concept_id']}")
        
        # Sample LOINC concepts
        result = await database_client.execute_cypher("MATCH (l:cae_LOINCConcept) RETURN l.concept_id LIMIT 3")
        if result:
            print("🧪 Sample LOINC concepts:")
            for row in result:
                print(f"   - {row['l.concept_id']}")
        
        return True
        
    except Exception as e:
        print(f"❌ Verification failed: {e}")
        return False

if __name__ == "__main__":
    asyncio.run(verify_data())
