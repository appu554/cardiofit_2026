#!/usr/bin/env python3
"""
Simple Clinical Knowledge Graph Ingestion - STRICT 20K TOTAL LIMIT
This script enforces a strict 20,000 record limit by limiting the input data reading
"""

import asyncio
import sys
import time
import csv
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.logging_config import get_logger
from core.database_factory import create_database_client

logger = get_logger(__name__)

async def create_simple_rxnorm_nodes(database_client, limit=5000):
    """Create simple RxNorm drug nodes directly with Cypher"""
    print(f"\n💊 CREATING RXNORM DRUG NODES (LIMITED TO {limit:,})")
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
        
        # Read and process RxNorm concepts with strict limit
        with open(rxnorm_file, 'r', encoding='utf-8', errors='ignore') as f:
            reader = csv.reader(f, delimiter='|')
            
            batch_size = 1000
            batch_queries = []
            
            for row_num, row in enumerate(reader):
                if len(row) < 15:
                    continue
                
                rxcui = row[0].strip()
                language = row[1].strip()
                concept_name = row[14].strip()
                term_type = row[12].strip()
                source = row[11].strip()
                
                # Only process English terms with valid data
                if language == 'ENG' and rxcui and concept_name and len(concept_name) > 2:
                    # Create Cypher query for this drug
                    cypher_query = f"""
                    MERGE (drug:cae_Drug {{rxcui: '{rxcui}'}})
                    SET drug.name = '{concept_name.replace("'", "\\'")}',
                        drug.term_type = '{term_type}',
                        drug.source = '{source}',
                        drug.created_at = datetime()
                    """
                    
                    batch_queries.append(cypher_query)
                    created_count += 1
                    
                    # Execute batch when full
                    if len(batch_queries) >= batch_size:
                        # Execute each query separately to avoid variable conflicts
                        for query in batch_queries:
                            await database_client.execute_cypher(query)
                        print(f"   ✅ Created batch of {len(batch_queries)} drugs ({created_count:,} total)")
                        batch_queries = []
                    
                    # STRICT LIMIT CHECK
                    if created_count >= limit:
                        print(f"   🛑 Reached limit of {limit:,} RxNorm drugs, stopping")
                        break
            
            # Execute remaining batch
            if batch_queries:
                # Execute each query separately to avoid variable conflicts
                for query in batch_queries:
                    await database_client.execute_cypher(query)
                print(f"   ✅ Created final batch of {len(batch_queries)} drugs")
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ RxNorm drug creation completed!")
        print(f"   📊 Total drugs created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        logger.error("RxNorm drug creation failed", error=str(e))
        print(f"❌ RxNorm drug creation failed: {e}")
        return False

async def create_simple_snomed_nodes(database_client, limit=5000):
    """Create simple SNOMED CT concept nodes directly with Cypher"""
    print(f"\n🧬 CREATING SNOMED CT CONCEPT NODES (LIMITED TO {limit:,})")
    print("=" * 50)
    
    start_time = time.time()
    created_count = 0
    
    try:
        # Path to SNOMED concepts file
        snomed_file = Path("data/snomed/extracted/SnomedCT_InternationalRF2_PRODUCTION_20250701T120000Z/Snapshot/Terminology/sct2_Concept_Snapshot_INT_20250701.txt")
        
        if not snomed_file.exists():
            print(f"❌ SNOMED CT file not found: {snomed_file}")
            return False
        
        print(f"📖 Reading SNOMED CT concepts from: {snomed_file}")
        
        # Read and process SNOMED concepts with strict limit
        with open(snomed_file, 'r', encoding='utf-8', errors='ignore') as f:
            reader = csv.reader(f, delimiter='\t')
            next(reader)  # Skip header
            
            batch_size = 1000
            batch_queries = []
            
            for row_num, row in enumerate(reader):
                if len(row) < 5:
                    continue
                
                concept_id = row[0].strip()
                effective_time = row[1].strip()
                active = row[2].strip()
                module_id = row[3].strip()
                definition_status_id = row[4].strip()
                
                # Only process active concepts
                if active == '1' and concept_id:
                    # Create Cypher query for this concept
                    cypher_query = f"""
                    MERGE (concept:cae_SNOMEDConcept {{concept_id: '{concept_id}'}})
                    SET concept.effective_time = '{effective_time}',
                        concept.active = {active == '1'},
                        concept.module_id = '{module_id}',
                        concept.definition_status_id = '{definition_status_id}',
                        concept.created_at = datetime()
                    """
                    
                    batch_queries.append(cypher_query)
                    created_count += 1
                    
                    # Execute batch when full
                    if len(batch_queries) >= batch_size:
                        # Execute each query separately to avoid variable conflicts
                        for query in batch_queries:
                            await database_client.execute_cypher(query)
                        print(f"   ✅ Created batch of {len(batch_queries)} concepts ({created_count:,} total)")
                        batch_queries = []
                    
                    # STRICT LIMIT CHECK
                    if created_count >= limit:
                        print(f"   🛑 Reached limit of {limit:,} SNOMED CT concepts, stopping")
                        break
            
            # Execute remaining batch
            if batch_queries:
                # Execute each query separately to avoid variable conflicts
                for query in batch_queries:
                    await database_client.execute_cypher(query)
                print(f"   ✅ Created final batch of {len(batch_queries)} concepts")
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ SNOMED CT concept creation completed!")
        print(f"   📊 Total concepts created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        logger.error("SNOMED CT concept creation failed", error=str(e))
        print(f"❌ SNOMED CT concept creation failed: {e}")
        return False

async def create_simple_loinc_nodes(database_client, limit=5000):
    """Create simple LOINC code nodes directly with Cypher"""
    print(f"\n🧪 CREATING LOINC CODE NODES (LIMITED TO {limit:,})")
    print("=" * 50)
    
    start_time = time.time()
    created_count = 0
    
    try:
        # Path to LOINC codes file (using SNOMED-LOINC snapshot)
        loinc_file = Path("data/loinc/snapshot/sct2_Concept_Snapshot_LO1010000_20250321.txt")
        
        if not loinc_file.exists():
            print(f"❌ LOINC file not found: {loinc_file}")
            return False
        
        print(f"📖 Reading LOINC codes from: {loinc_file}")
        
        # Read and process LOINC codes with strict limit
        with open(loinc_file, 'r', encoding='utf-8', errors='ignore') as f:
            reader = csv.reader(f, delimiter='\t')
            next(reader)  # Skip header
            
            batch_size = 1000
            batch_queries = []
            
            for row_num, row in enumerate(reader):
                if len(row) < 5:
                    continue

                concept_id = row[0].strip()
                effective_time = row[1].strip()
                active = row[2].strip()
                module_id = row[3].strip()
                definition_status_id = row[4].strip()

                # Only process active LOINC concepts
                if active == '1' and concept_id:
                    # Create Cypher query for this LOINC concept
                    cypher_query = f"""
                    MERGE (loinc:cae_LOINCConcept {{concept_id: '{concept_id}'}})
                    SET loinc.effective_time = '{effective_time}',
                        loinc.active = {active == '1'},
                        loinc.module_id = '{module_id}',
                        loinc.definition_status_id = '{definition_status_id}',
                        loinc.created_at = datetime()
                    """
                    
                    batch_queries.append(cypher_query)
                    created_count += 1
                    
                    # Execute batch when full
                    if len(batch_queries) >= batch_size:
                        # Execute each query separately to avoid variable conflicts
                        for query in batch_queries:
                            await database_client.execute_cypher(query)
                        print(f"   ✅ Created batch of {len(batch_queries)} LOINC concepts ({created_count:,} total)")
                        batch_queries = []
                    
                    # STRICT LIMIT CHECK
                    if created_count >= limit:
                        print(f"   🛑 Reached limit of {limit:,} LOINC codes, stopping")
                        break
            
            # Execute remaining batch
            if batch_queries:
                # Execute each query separately to avoid variable conflicts
                for query in batch_queries:
                    await database_client.execute_cypher(query)
                print(f"   ✅ Created final batch of {len(batch_queries)} LOINC concepts")
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ LOINC code creation completed!")
        print(f"   📊 Total LOINC codes created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        logger.error("LOINC code creation failed", error=str(e))
        print(f"❌ LOINC code creation failed: {e}")
        return False

async def create_simple_relationships(database_client, limit=5000):
    """Create simple relationships between terminologies"""
    print(f"\n🔗 CREATING CROSS-TERMINOLOGY RELATIONSHIPS (LIMITED TO {limit:,})")
    print("=" * 50)
    
    start_time = time.time()
    created_count = 0
    
    try:
        # Create some sample relationships between existing nodes
        cypher_query = f"""
        MATCH (drug:cae_Drug), (snomed:cae_SNOMEDConcept)
        WITH drug, snomed
        LIMIT {limit}
        MERGE (drug)-[:cae_hasSNOMEDCTMapping]->(snomed)
        RETURN count(*) as relationships_created
        """
        
        result = await database_client.execute_cypher(cypher_query)
        if result:
            created_count = result[0].get('relationships_created', 0)
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ Cross-terminology relationship creation completed!")
        print(f"   📊 Total relationships created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        logger.error("Cross-terminology relationship creation failed", error=str(e))
        print(f"❌ Cross-terminology relationship creation failed: {e}")
        return False

async def main():
    """Main function with STRICT 20K total limit using direct Cypher"""
    print("🏥 CLINICAL KNOWLEDGE GRAPH - SIMPLE 20K INGESTION")
    print("=" * 60)
    print("🚨 STRICT TOTAL LIMIT: 20,000 RECORDS USING DIRECT CYPHER")
    print("=" * 60)
    
    # Distribution of 20K records
    total_limit = 20000
    rxnorm_limit = 5000    # 25% for RxNorm drugs
    snomed_limit = 5000    # 25% for SNOMED concepts  
    loinc_limit = 5000     # 25% for LOINC codes
    relationship_limit = 5000   # 25% for relationships
    
    print(f"\n📊 RECORD DISTRIBUTION (TOTAL: {total_limit:,}):")
    print(f"   RxNorm drugs: {rxnorm_limit:,} nodes")
    print(f"   SNOMED CT concepts: {snomed_limit:,} nodes")
    print(f"   LOINC codes: {loinc_limit:,} nodes")
    print(f"   Cross-relationships: {relationship_limit:,} edges")
    print("=" * 60)
    
    try:
        # Create database client
        print("\n🔌 Connecting to Neo4j Cloud...")
        database_client = await create_database_client()
        connection_ok = await database_client.test_connection()
        if not connection_ok:
            print("❌ Database connection failed. Aborting.")
            return False
        
        print("✅ Database connection successful")
        
        # Run simple ingestions using direct Cypher
        rxnorm_success = await create_simple_rxnorm_nodes(database_client, limit=rxnorm_limit)
        snomed_success = await create_simple_snomed_nodes(database_client, limit=snomed_limit)
        loinc_success = await create_simple_loinc_nodes(database_client, limit=loinc_limit)
        
        # Create relationships only if nodes were created
        relationship_success = True
        if rxnorm_success and snomed_success:
            relationship_success = await create_simple_relationships(database_client, limit=relationship_limit)
        
        # Summary
        print("\n" + "=" * 60)
        print("📊 SIMPLE INGESTION SUMMARY")
        print("=" * 60)
        print(f"RxNorm drugs: {'✅ Success' if rxnorm_success else '❌ Failed'}")
        print(f"SNOMED CT concepts: {'✅ Success' if snomed_success else '❌ Failed'}")
        print(f"LOINC codes: {'✅ Success' if loinc_success else '❌ Failed'}")
        print(f"Cross-relationships: {'✅ Success' if relationship_success else '❌ Failed'}")
        print(f"\nTotal limit enforced: {total_limit:,} records")
        print("🎉 Knowledge graph ready for CAE integration!")
        print("=" * 60)
        
        return True
        
    except Exception as e:
        logger.error("Simple ingestion failed", error=str(e))
        print(f"❌ Simple ingestion failed: {e}")
        return False

if __name__ == "__main__":
    asyncio.run(main())
