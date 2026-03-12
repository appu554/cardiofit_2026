#!/usr/bin/env python3
"""
Clear RxNorm drugs and restart ingestion
"""

import asyncio
import sys
import time
import csv
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.database_factory import create_database_client

async def clear_rxnorm_drugs(database_client):
    """Clear all RxNorm drug nodes and their relationships"""
    print("🗑️ CLEARING RXNORM DRUGS")
    print("=" * 40)
    
    try:
        # Count existing drugs
        result = await database_client.execute_cypher("MATCH (d:cae_Drug) RETURN count(d) as count")
        existing_count = result[0]['count'] if result else 0
        print(f"📊 Found {existing_count:,} existing RxNorm drugs")
        
        if existing_count > 0:
            # Delete all relationships involving RxNorm drugs first
            print("🔗 Deleting relationships involving RxNorm drugs...")
            await database_client.execute_cypher("""
                MATCH (d:cae_Drug)-[r]-()
                DELETE r
            """)
            
            # Delete all RxNorm drug nodes
            print("💊 Deleting RxNorm drug nodes...")
            await database_client.execute_cypher("""
                MATCH (d:cae_Drug)
                DELETE d
            """)
            
            print("✅ RxNorm drugs cleared successfully")
        else:
            print("ℹ️ No RxNorm drugs found to clear")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to clear RxNorm drugs: {e}")
        return False

async def restart_rxnorm_ingestion(database_client, limit=5000):
    """Restart RxNorm ingestion with better filtering"""
    print(f"\n💊 RESTARTING RXNORM INGESTION (LIMITED TO {limit:,})")
    print("=" * 50)
    
    start_time = time.time()
    created_count = 0
    
    try:
        # Path to RxNorm concepts file
        rxnorm_file = Path("data/rxnorm/rrf/RXNCONSO.RRF")
        
        if not rxnorm_file.exists():
            print(f"❌ RxNorm file not found: {rxnorm_file}")
            return False
        
        print(f"📖 Reading RxNorm concepts from: {rxnorm_file}")
        
        # Read and process RxNorm concepts with better filtering
        with open(rxnorm_file, 'r', encoding='utf-8', errors='ignore') as f:
            reader = csv.reader(f, delimiter='|')
            
            batch_size = 100  # Smaller batches for better performance
            processed_rxcuis = set()  # Track unique RxCUIs
            
            for row_num, row in enumerate(reader):
                if len(row) < 15:
                    continue
                
                rxcui = row[0].strip()
                language = row[1].strip()
                concept_name = row[14].strip()
                term_type = row[12].strip()
                source = row[11].strip()
                
                # Better filtering criteria
                if (language == 'ENG' and 
                    rxcui and 
                    concept_name and 
                    len(concept_name) > 2 and
                    rxcui not in processed_rxcuis and  # Avoid duplicates
                    term_type in ['IN', 'PIN', 'BN', 'SBD', 'SCD']):  # Focus on important term types
                    
                    # Clean concept name for Cypher
                    clean_name = concept_name.replace("'", "\\'").replace('"', '\\"')
                    
                    # Create Cypher query for this drug
                    cypher_query = f"""
                    MERGE (drug:cae_Drug {{rxcui: '{rxcui}'}})
                    SET drug.name = '{clean_name}',
                        drug.term_type = '{term_type}',
                        drug.source = '{source}',
                        drug.created_at = datetime()
                    """
                    
                    try:
                        await database_client.execute_cypher(cypher_query)
                        processed_rxcuis.add(rxcui)
                        created_count += 1
                        
                        # Progress update
                        if created_count % batch_size == 0:
                            print(f"   ✅ Created {created_count:,} drugs so far...")
                        
                        # STRICT LIMIT CHECK
                        if created_count >= limit:
                            print(f"   🛑 Reached limit of {limit:,} RxNorm drugs, stopping")
                            break
                            
                    except Exception as e:
                        print(f"   ⚠️ Failed to create drug {rxcui}: {e}")
                        continue
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ RxNorm drug ingestion completed!")
        print(f"   📊 Total drugs created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        print(f"❌ RxNorm drug ingestion failed: {e}")
        return False

async def recreate_relationships(database_client, limit=5000):
    """Recreate relationships between RxNorm drugs and other terminologies"""
    print(f"\n🔗 RECREATING CROSS-TERMINOLOGY RELATIONSHIPS (LIMITED TO {limit:,})")
    print("=" * 60)
    
    start_time = time.time()
    
    try:
        # Create relationships between RxNorm drugs and SNOMED concepts
        cypher_query = f"""
        MATCH (drug:cae_Drug), (snomed:cae_SNOMEDConcept)
        WITH drug, snomed
        LIMIT {limit}
        MERGE (drug)-[:cae_hasSNOMEDCTMapping]->(snomed)
        RETURN count(*) as relationships_created
        """
        
        result = await database_client.execute_cypher(cypher_query)
        created_count = result[0].get('relationships_created', 0) if result else 0
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ Cross-terminology relationship creation completed!")
        print(f"   📊 Total relationships created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        print(f"❌ Cross-terminology relationship creation failed: {e}")
        return False

async def main():
    """Main function to clear and restart RxNorm ingestion"""
    print("🔄 CLEAR AND RESTART RXNORM INGESTION")
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
        
        # Step 1: Clear existing RxNorm drugs
        clear_success = await clear_rxnorm_drugs(database_client)
        if not clear_success:
            return False
        
        # Step 2: Restart RxNorm ingestion
        rxnorm_success = await restart_rxnorm_ingestion(database_client, limit=5000)
        if not rxnorm_success:
            return False
        
        # Step 3: Recreate relationships
        relationship_success = await recreate_relationships(database_client, limit=5000)
        
        # Final summary
        print("\n" + "=" * 60)
        print("📊 RESTART SUMMARY")
        print("=" * 60)
        print(f"Clear RxNorm: {'✅ Success' if clear_success else '❌ Failed'}")
        print(f"Restart RxNorm: {'✅ Success' if rxnorm_success else '❌ Failed'}")
        print(f"Recreate relationships: {'✅ Success' if relationship_success else '❌ Failed'}")
        print("🎉 RxNorm restart completed!")
        print("=" * 60)
        
        return True
        
    except Exception as e:
        print(f"❌ Restart failed: {e}")
        return False

if __name__ == "__main__":
    asyncio.run(main())
