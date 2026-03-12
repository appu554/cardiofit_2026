#!/usr/bin/env python3
"""
Limited Clinical Knowledge Graph Ingestion - STRICT 20K TOTAL LIMIT
This script enforces a strict 20,000 record limit across ALL terminologies
"""

import asyncio
import sys
import time
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.logging_config import get_logger
from core.database_factory import create_database_client
from ingesters.rxnorm_ingester import RxNormIngester
from ingesters.snomed_ingester import SNOMEDIngester
from ingesters.snomed_loinc_ingester import SNOMEDLOINCIngester
from ingesters.cross_terminology_mapper import CrossTerminologyMapper

logger = get_logger(__name__)

async def run_limited_rxnorm_ingestion(database_client, limit=5000):
    """Run limited RxNorm ingestion"""
    print(f"\n📦 INGESTING RXNORM DATA (LIMITED TO {limit:,} RECORDS)")
    print("=" * 50)
    
    start_time = time.time()
    record_count = 0
    batch_count = 0
    
    try:
        rxnorm_ingester = RxNormIngester(database_client)
        
        # Check for RxNorm data
        data_available = await rxnorm_ingester.check_data()
        if not data_available:
            print("❌ RxNorm data check failed. Aborting.")
            return False
        
        print("📝 Processing RxNorm data with STRICT limit...")
        
        # Process data with strict limit
        batch_size = 1000
        rdf_batch = []
        
        async for rdf_triple in rxnorm_ingester.process_data():
            if rdf_triple and rdf_triple.strip():
                rdf_batch.append(rdf_triple)
                record_count += 1
                
                # Insert batch when full
                if len(rdf_batch) >= batch_size:
                    batch_rdf = "\n".join(rdf_batch)
                    prefixes = rxnorm_ingester.get_ontology_prefixes()
                    full_rdf = f"{prefixes}\n{batch_rdf}"
                    
                    await database_client.batch_insert_rdf(full_rdf)
                    batch_count += 1
                    print(f"   📦 Inserted batch {batch_count} ({record_count:,} records so far)")
                    
                    rdf_batch = []
                
                # STRICT LIMIT CHECK
                if record_count >= limit:
                    print(f"   🛑 Reached limit of {limit:,} RxNorm records, stopping processing")
                    break
        
        # Insert remaining batch
        if rdf_batch:
            batch_rdf = "\n".join(rdf_batch)
            prefixes = rxnorm_ingester.get_ontology_prefixes()
            full_rdf = f"{prefixes}\n{batch_rdf}"
            await database_client.batch_insert_rdf(full_rdf)
            batch_count += 1
            print(f"   📦 Inserted final batch {batch_count}")
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ RxNorm ingestion completed!")
        print(f"   📊 Total records processed: {record_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        logger.error("RxNorm ingestion failed", error=str(e))
        print(f"❌ RxNorm ingestion failed: {e}")
        return False

async def run_limited_snomed_ingestion(database_client, limit=5000):
    """Run limited SNOMED CT ingestion"""
    print(f"\n🧬 INGESTING SNOMED CT DATA (LIMITED TO {limit:,} RECORDS)")
    print("=" * 50)
    
    start_time = time.time()
    record_count = 0
    batch_count = 0
    
    try:
        snomed_ingester = SNOMEDIngester(database_client)
        
        # Check for SNOMED data
        data_available = await snomed_ingester.check_data()
        if not data_available:
            print("❌ SNOMED CT data check failed. Aborting.")
            return False
        
        print("📝 Processing SNOMED CT data with STRICT limit...")
        
        # Process data with strict limit
        batch_size = 1000
        rdf_batch = []
        
        async for rdf_triple in snomed_ingester.process_data():
            if rdf_triple and rdf_triple.strip():
                rdf_batch.append(rdf_triple)
                record_count += 1
                
                # Insert batch when full
                if len(rdf_batch) >= batch_size:
                    batch_rdf = "\n".join(rdf_batch)
                    prefixes = snomed_ingester.get_ontology_prefixes()
                    full_rdf = f"{prefixes}\n{batch_rdf}"
                    
                    await database_client.batch_insert_rdf(full_rdf)
                    batch_count += 1
                    print(f"   📦 Inserted batch {batch_count} ({record_count:,} records so far)")
                    
                    rdf_batch = []
                
                # STRICT LIMIT CHECK
                if record_count >= limit:
                    print(f"   🛑 Reached limit of {limit:,} SNOMED CT records, stopping processing")
                    break
        
        # Insert remaining batch
        if rdf_batch:
            batch_rdf = "\n".join(rdf_batch)
            prefixes = snomed_ingester.get_ontology_prefixes()
            full_rdf = f"{prefixes}\n{batch_rdf}"
            await database_client.batch_insert_rdf(full_rdf)
            batch_count += 1
            print(f"   📦 Inserted final batch {batch_count}")
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ SNOMED CT ingestion completed!")
        print(f"   📊 Total records processed: {record_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        logger.error("SNOMED CT ingestion failed", error=str(e))
        print(f"❌ SNOMED CT ingestion failed: {e}")
        return False

async def run_limited_loinc_ingestion(database_client, limit=5000):
    """Run limited LOINC ingestion"""
    print(f"\n🧪 INGESTING LOINC DATA (LIMITED TO {limit:,} RECORDS)")
    print("=" * 50)
    
    start_time = time.time()
    record_count = 0
    batch_count = 0
    
    try:
        loinc_ingester = SNOMEDLOINCIngester(database_client)
        
        # Check for LOINC data
        data_available = await loinc_ingester.check_data()
        if not data_available:
            print("❌ LOINC data check failed. Aborting.")
            return False
        
        print("📝 Processing LOINC data with STRICT limit...")
        
        # Process data with strict limit
        batch_size = 1000
        rdf_batch = []
        
        async for rdf_triple in loinc_ingester.process_data():
            if rdf_triple and rdf_triple.strip():
                rdf_batch.append(rdf_triple)
                record_count += 1
                
                # Insert batch when full
                if len(rdf_batch) >= batch_size:
                    batch_rdf = "\n".join(rdf_batch)
                    prefixes = loinc_ingester.get_ontology_prefixes()
                    full_rdf = f"{prefixes}\n{batch_rdf}"
                    
                    await database_client.batch_insert_rdf(full_rdf)
                    batch_count += 1
                    print(f"   📦 Inserted batch {batch_count} ({record_count:,} records so far)")
                    
                    rdf_batch = []
                
                # STRICT LIMIT CHECK
                if record_count >= limit:
                    print(f"   🛑 Reached limit of {limit:,} LOINC records, stopping processing")
                    break
        
        # Insert remaining batch
        if rdf_batch:
            batch_rdf = "\n".join(rdf_batch)
            prefixes = loinc_ingester.get_ontology_prefixes()
            full_rdf = f"{prefixes}\n{batch_rdf}"
            await database_client.batch_insert_rdf(full_rdf)
            batch_count += 1
            print(f"   📦 Inserted final batch {batch_count}")
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ LOINC ingestion completed!")
        print(f"   📊 Total records processed: {record_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        logger.error("LOINC ingestion failed", error=str(e))
        print(f"❌ LOINC ingestion failed: {e}")
        return False

async def run_limited_cross_mapping(database_client, limit=5000):
    """Run limited cross-terminology mapping"""
    print(f"\n🔗 CREATING CROSS-TERMINOLOGY RELATIONSHIPS (LIMITED TO {limit:,} MAPPINGS)")
    print("=" * 50)
    
    start_time = time.time()
    mapping_count = 0
    batch_count = 0
    
    try:
        cross_mapper = CrossTerminologyMapper(database_client)
        
        print("📝 Processing cross-terminology mappings with STRICT limit...")
        
        # Process mappings with strict limit
        batch_size = 1000
        rdf_batch = []
        
        async for rdf_triple in cross_mapper.process_data():
            if rdf_triple and rdf_triple.strip():
                rdf_batch.append(rdf_triple)
                mapping_count += 1
                
                # Insert batch when full
                if len(rdf_batch) >= batch_size:
                    batch_rdf = "\n".join(rdf_batch)
                    prefixes = cross_mapper.get_ontology_prefixes()
                    full_rdf = f"{prefixes}\n{batch_rdf}"
                    
                    await database_client.batch_insert_rdf(full_rdf)
                    batch_count += 1
                    print(f"   📦 Inserted batch {batch_count} ({mapping_count:,} mappings so far)")
                    
                    rdf_batch = []
                
                # STRICT LIMIT CHECK
                if mapping_count >= limit:
                    print(f"   🛑 Reached limit of {limit:,} cross-mappings, stopping processing")
                    break
        
        # Insert remaining batch
        if rdf_batch:
            batch_rdf = "\n".join(rdf_batch)
            prefixes = cross_mapper.get_ontology_prefixes()
            full_rdf = f"{prefixes}\n{batch_rdf}"
            await database_client.batch_insert_rdf(full_rdf)
            batch_count += 1
            print(f"   📦 Inserted final batch {batch_count}")
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ Cross-terminology mapping completed!")
        print(f"   📊 Total mappings processed: {mapping_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        logger.error("Cross-terminology mapping failed", error=str(e))
        print(f"❌ Cross-terminology mapping failed: {e}")
        return False

async def main():
    """Main function with STRICT 20K total limit"""
    print("🏥 CLINICAL KNOWLEDGE GRAPH - LIMITED INGESTION")
    print("=" * 60)
    print("🚨 STRICT TOTAL LIMIT: 20,000 RECORDS ACROSS ALL TERMINOLOGIES")
    print("=" * 60)
    
    # Distribution of 20K records
    total_limit = 20000
    rxnorm_limit = 5000    # 25% for RxNorm drugs
    snomed_limit = 5000    # 25% for SNOMED concepts  
    loinc_limit = 5000     # 25% for LOINC codes
    mapping_limit = 5000   # 25% for cross-terminology relationships
    
    print(f"\n📊 RECORD DISTRIBUTION (TOTAL: {total_limit:,}):")
    print(f"   RxNorm: {rxnorm_limit:,} records")
    print(f"   SNOMED CT: {snomed_limit:,} records")
    print(f"   LOINC: {loinc_limit:,} records")
    print(f"   Cross-mappings: {mapping_limit:,} records")
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
        
        # Run limited ingestions
        rxnorm_success = await run_limited_rxnorm_ingestion(database_client, limit=rxnorm_limit)
        snomed_success = await run_limited_snomed_ingestion(database_client, limit=snomed_limit)
        loinc_success = await run_limited_loinc_ingestion(database_client, limit=loinc_limit)
        
        # Run cross-mapping only if previous steps succeeded
        mapping_success = True
        if rxnorm_success and snomed_success and loinc_success:
            mapping_success = await run_limited_cross_mapping(database_client, limit=mapping_limit)
        
        # Summary
        print("\n" + "=" * 60)
        print("📊 INGESTION SUMMARY")
        print("=" * 60)
        print(f"RxNorm: {'✅ Success' if rxnorm_success else '❌ Failed'}")
        print(f"SNOMED CT: {'✅ Success' if snomed_success else '❌ Failed'}")
        print(f"LOINC: {'✅ Success' if loinc_success else '❌ Failed'}")
        print(f"Cross-mappings: {'✅ Success' if mapping_success else '❌ Failed'}")
        print(f"\nTotal limit enforced: {total_limit:,} records")
        print("=" * 60)
        
        return True
        
    except Exception as e:
        logger.error("Limited ingestion failed", error=str(e))
        print(f"❌ Limited ingestion failed: {e}")
        return False

if __name__ == "__main__":
    asyncio.run(main())
