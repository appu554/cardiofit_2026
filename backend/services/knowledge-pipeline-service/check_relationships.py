#!/usr/bin/env python3
"""
Check relationships in the clinical knowledge graph
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.database_factory import create_database_client

async def check_all_relationships(database_client):
    """Check all types of relationships in the knowledge graph"""
    print("🔗 CHECKING ALL RELATIONSHIPS")
    print("=" * 50)
    
    try:
        # Get all relationship types
        result = await database_client.execute_cypher("""
            MATCH ()-[r]->()
            RETURN DISTINCT type(r) as relationship_type, count(r) as count
            ORDER BY count DESC
        """)
        
        if result:
            print("📊 Relationship Types:")
            total_relationships = 0
            for row in result:
                rel_type = row['relationship_type']
                count = row['count']
                total_relationships += count
                print(f"   {rel_type}: {count:,} relationships")
            
            print(f"\n🎯 Total Relationships: {total_relationships:,}")
        else:
            print("⚠️ No relationships found")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to check relationships: {e}")
        return False

async def check_cross_terminology_mappings(database_client):
    """Check specific cross-terminology mappings"""
    print("\n🔍 CHECKING CROSS-TERMINOLOGY MAPPINGS")
    print("=" * 50)
    
    try:
        # Check RxNorm to SNOMED mappings
        result = await database_client.execute_cypher("""
            MATCH (drug:cae_Drug)-[r:cae_hasSNOMEDCTMapping]->(snomed:cae_SNOMEDConcept)
            RETURN count(r) as count
        """)
        rxnorm_snomed_count = result[0]['count'] if result else 0
        print(f"💊➡️🧬 RxNorm → SNOMED CT: {rxnorm_snomed_count:,} mappings")
        
        # Check SNOMED to RxNorm mappings (reverse)
        result = await database_client.execute_cypher("""
            MATCH (snomed:cae_SNOMEDConcept)-[r:cae_hasRxNormMapping]->(drug:cae_Drug)
            RETURN count(r) as count
        """)
        snomed_rxnorm_count = result[0]['count'] if result else 0
        print(f"🧬➡️💊 SNOMED CT → RxNorm: {snomed_rxnorm_count:,} mappings")
        
        # Check SNOMED to LOINC mappings
        result = await database_client.execute_cypher("""
            MATCH (snomed:cae_SNOMEDConcept)-[r:cae_hasLOINCMapping]->(loinc:cae_LOINCConcept)
            RETURN count(r) as count
        """)
        snomed_loinc_count = result[0]['count'] if result else 0
        print(f"🧬➡️🧪 SNOMED CT → LOINC: {snomed_loinc_count:,} mappings")
        
        # Check LOINC to SNOMED mappings (reverse)
        result = await database_client.execute_cypher("""
            MATCH (loinc:cae_LOINCConcept)-[r:cae_hasSNOMEDCTMapping]->(snomed:cae_SNOMEDConcept)
            RETURN count(r) as count
        """)
        loinc_snomed_count = result[0]['count'] if result else 0
        print(f"🧪➡️🧬 LOINC → SNOMED CT: {loinc_snomed_count:,} mappings")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to check cross-terminology mappings: {e}")
        return False

async def sample_relationships(database_client):
    """Show sample relationships"""
    print("\n📋 SAMPLE RELATIONSHIPS")
    print("=" * 30)
    
    try:
        # Sample RxNorm to SNOMED mappings
        result = await database_client.execute_cypher("""
            MATCH (drug:cae_Drug)-[r:cae_hasSNOMEDCTMapping]->(snomed:cae_SNOMEDConcept)
            RETURN drug.rxcui, drug.name, snomed.concept_id
            LIMIT 5
        """)
        
        if result:
            print("💊➡️🧬 Sample RxNorm → SNOMED CT mappings:")
            for row in result:
                drug_name = row['drug.name'][:30] + "..." if len(row['drug.name']) > 30 else row['drug.name']
                print(f"   {row['drug.rxcui']} ({drug_name}) → {row['snomed.concept_id']}")
        
        # Sample relationship patterns
        result = await database_client.execute_cypher("""
            MATCH (n1)-[r]->(n2)
            RETURN labels(n1)[0] as source_type, type(r) as relationship, labels(n2)[0] as target_type, count(*) as count
            ORDER BY count DESC
            LIMIT 10
        """)
        
        if result:
            print("\n🔗 Top relationship patterns:")
            for row in result:
                print(f"   {row['source_type']} --[{row['relationship']}]--> {row['target_type']}: {row['count']:,}")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to sample relationships: {e}")
        return False

async def check_node_connectivity(database_client):
    """Check how well connected the nodes are"""
    print("\n🌐 CHECKING NODE CONNECTIVITY")
    print("=" * 40)
    
    try:
        # Nodes with no relationships
        result = await database_client.execute_cypher("""
            MATCH (n)
            WHERE NOT (n)-[]-()
            RETURN labels(n)[0] as node_type, count(n) as isolated_count
        """)
        
        if result:
            print("🏝️ Isolated nodes (no relationships):")
            for row in result:
                if row['isolated_count'] > 0:
                    print(f"   {row['node_type']}: {row['isolated_count']:,} isolated nodes")
        
        # Nodes with relationships
        result = await database_client.execute_cypher("""
            MATCH (n)-[r]-()
            RETURN labels(n)[0] as node_type, count(DISTINCT n) as connected_count
        """)
        
        if result:
            print("\n🔗 Connected nodes:")
            for row in result:
                print(f"   {row['node_type']}: {row['connected_count']:,} connected nodes")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to check node connectivity: {e}")
        return False

async def main():
    """Main function to check relationships"""
    print("🔍 CLINICAL KNOWLEDGE GRAPH - RELATIONSHIP ANALYSIS")
    print("=" * 60)
    
    try:
        # Create database client
        print("🔌 Connecting to Neo4j Cloud...")
        database_client = await create_database_client()
        connection_ok = await database_client.test_connection()
        if not connection_ok:
            print("❌ Database connection failed")
            return False
        
        print("✅ Connected to Neo4j Cloud")
        
        # Run all relationship checks
        await check_all_relationships(database_client)
        await check_cross_terminology_mappings(database_client)
        await sample_relationships(database_client)
        await check_node_connectivity(database_client)
        
        print("\n" + "=" * 60)
        print("✅ RELATIONSHIP ANALYSIS COMPLETED")
        print("=" * 60)
        
        return True
        
    except Exception as e:
        print(f"❌ Relationship analysis failed: {e}")
        return False

if __name__ == "__main__":
    asyncio.run(main())
