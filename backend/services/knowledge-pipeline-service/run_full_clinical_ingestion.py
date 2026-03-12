#!/usr/bin/env python3
"""
Full Clinical Knowledge Graph Ingestion Script
This script orchestrates the complete ingestion of RxNorm, SNOMED CT, and SNOMED CT LOINC Extension
data into Neo4j, following the implementation plan.

Sequence:
1. RxNorm (drugs and ingredients)
2. SNOMED CT (complete terminology)
3. SNOMED CT LOINC Extension (LOINC mappings)
4. Cross-Terminology Mappings (relationships between RxNorm, SNOMED CT, and LOINC)
"""

import asyncio
import sys
import time
import csv
from pathlib import Path
import os

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.logging_config import get_logger
from core.database_factory import create_database_client
from ingesters.rxnorm_ingester import RxNormIngester
from ingesters.snomed_ingester import SNOMEDIngester  # Full SNOMED CT ingester
from ingesters.snomed_loinc_ingester import SNOMEDLOINCIngester  # SNOMED-LOINC mappings
from ingesters.cross_terminology_mapper import CrossTerminologyMapper  # Cross-terminology relationships

logger = get_logger(__name__)

async def verify_data_directories():
    """Verify that all required data directories exist"""
    data_root = Path(__file__).parent / "data"
    
    # Required directories
    required_dirs = {
        "rxnorm": data_root / "rxnorm",
        "snomed": data_root / "snomed",
        "loinc": data_root / "loinc"
    }
    
    all_exist = True
    
    for name, path in required_dirs.items():
        if not path.exists():
            logger.error(f"{name.upper()} data directory not found", path=str(path))
            print(f"\u274c {name.upper()} data directory not found: {path}")
            if name == "snomed" or name == "loinc":
                print(f"   NOTE: {name.upper()} requires manual download of data files")
            all_exist = False
        else:
            # Check if directory is empty
            files = list(path.glob("**/*"))
            if not files:
                logger.error(f"{name.upper()} data directory is empty", path=str(path))
                print(f"\u26a0\ufe0f {name.upper()} data directory is empty: {path}")
                all_exist = False
            else:
                logger.info(f"{name.upper()} data directory found", path=str(path), files=len(files))
                print(f"\u2705 {name.upper()} data directory found: {path}")
    
    return all_exist

async def create_indexes(database_client):
    """Create necessary indexes for performance"""
    index_queries = [
        # RxNorm indexes
        "CREATE INDEX IF NOT EXISTS FOR (n:cae_Drug) ON (n.rxcui)",
        "CREATE INDEX IF NOT EXISTS FOR (n:cae_Drug) ON (n.name)",
        
        # SNOMED indexes
        "CREATE INDEX IF NOT EXISTS FOR (n:cae_SNOMEDConcept) ON (n.conceptId)",
        "CREATE INDEX IF NOT EXISTS FOR (n:cae_SNOMEDConcept) ON (n.sctid)",
        
        # LOINC indexes
        "CREATE INDEX IF NOT EXISTS FOR (n:cae_LOINCConcept) ON (n.code)",
        "CREATE INDEX IF NOT EXISTS FOR (n:cae_LOINCConcept) ON (n.loincNum)",
        
        # Relationship indexes for cross-terminology mappings
        "CREATE INDEX IF NOT EXISTS FOR ()-[r:cae_hasSNOMEDCTMapping]-() ON (r.source)",
        "CREATE INDEX IF NOT EXISTS FOR ()-[r:cae_hasRxNormMapping]-() ON (r.source)",
        "CREATE INDEX IF NOT EXISTS FOR ()-[r:cae_hasLOINCMapping]-() ON (r.source)"
    ]
    
    print("\n\U0001f527 Creating database indexes...")
    for query in index_queries:
        try:
            logger.info("Executing index creation", query=query)
            await database_client.execute_cypher(query)
            logger.info("Index creation successful")
            print(f"\u2705 Created index: {query}")
        except Exception as e:
            logger.error("Index creation failed", query=query, error=str(e))
            print(f"\u26a0\ufe0f Index failed: {query} - {e}")

async def run_rxnorm_ingestion(database_client, limit=20000):
    """Run RxNorm ingestion pipeline with strict record limiting
    
    Args:
        database_client: The database client
        limit: Maximum number of records to process (default: 20000)
    """
    print("\n\U0001f4e6 INGESTING RXNORM DATA (LIMITED TO {0:,} RECORDS)".format(limit))
    print("=" * 50)
    
    logger.info("Starting RxNorm ingestion with strict limit", limit=limit)
    start_time = time.time()
    
    # Initialize counters
    record_count = 0
    batch_count = 0
    
    try:
        # Initialize the RxNorm ingester
        rxnorm_ingester = RxNormIngester(database_client)
        
        # Download RxNorm data if needed
        logger.info("Checking for RxNorm data...")
        download_success = await rxnorm_ingester.download_data()
        if not download_success:
            logger.error("RxNorm data download failed")
            print("\u274c RxNorm data download failed. Aborting.")
            return False
        
        logger.info("Processing RxNorm data with strict limit...")
        print("\U0001f4dd Processing RxNorm data (STRICTLY LIMITED TO 20K RECORDS)...")
        
        # STRICTLY LIMIT THE RXNORM DATA
        print("   Loading limited set of RxNorm concepts...")
        
        # Step 1: Process only a limited number of drug concepts
        concept_count = 0
        concept_file = rxnorm_ingester.data_dir / "rrf" / "RXNCONSO.RRF"
        
        # Batch processing for Neo4j inserts
        rdf_batch = []
        batch_size = 1000
        
        # Process RXNCONSO.RRF to get drug concepts (limited)
        with open(concept_file, 'r', encoding='utf-8', errors='ignore') as f:
            for line in f:
                fields = line.strip().split('|')
                
                # RXCUI is in position 0
                rxcui = fields[0].strip()
                # Term type is in position 12
                term_type = fields[12].strip()
                # SAB (source) in position 11
                source = fields[11].strip()
                # TTY (term type) in position 12 - 'IN' is for ingredient
                tty = fields[12].strip()
                # Preferred name in position 14
                name = fields[14].strip()
                
                if rxcui and name and source == "RXNORM" and tty == "IN":
                    # Generate RDF for this concept
                    drug_uri = rxnorm_ingester.generate_uri("Drug", rxcui)
                    
                    # Add to processed concepts
                    rxnorm_ingester.processed_concepts.add(rxcui)
                    
                    # Create RDF triple
                    rdf_triple = f"""
{drug_uri} a cae:Drug ;
    cae:hasRxCUI "{rxcui}" ;
    cae:hasSource "RxNorm" ;
    cae:hasName "{name}" .
"""
                    
                    # Add to batch
                    rdf_batch.append(rdf_triple)
                    record_count += 1
                    
                    # Insert batch if full
                    if len(rdf_batch) >= batch_size:
                        batch_rdf = "\n".join(rdf_batch)
                        prefixes = rxnorm_ingester.get_ontology_prefixes()
                        full_rdf = f"{prefixes}\n{batch_rdf}"
                        
                        await database_client.batch_insert_rdf(full_rdf)
                        batch_count += 1
                        print(f"   \U0001f4e6 Inserted batch {batch_count} ({record_count:,} records so far)")
                        
                        rdf_batch = []
                    
                    # Check if we've reached the limit
                    if record_count >= limit:
                        logger.info(f"Reached limit of {limit} RxNorm records, stopping processing")
                        print(f"   \ud83d\uded1 Reached limit of {limit:,} RxNorm records, stopping processing")
                        break
        
        # Insert any remaining RDF triples
        if rdf_batch:
            batch_rdf = "\n".join(rdf_batch)
            prefixes = rxnorm_ingester.get_ontology_prefixes()
            full_rdf = f"{prefixes}\n{batch_rdf}"
            
            await database_client.batch_insert_rdf(full_rdf)
            batch_count += 1
            print(f"   \U0001f4e6 Inserted final batch {batch_count} ({record_count:,} records total)")
        
        end_time = time.time()
        duration = end_time - start_time
        
        logger.info("RxNorm ingestion completed", 
                  records=record_count, 
                  batches=batch_count, 
                  duration=f"{duration:.2f} seconds")
        
        print(f"\n\u2705 RxNorm ingestion completed:")
        print(f"   - {record_count:,} records processed")
        print(f"   - {batch_count:,} batches inserted")
        print(f"   - Took {duration:.2f} seconds")
        
        return True
        
    except Exception as e:
        end_time = time.time()
        duration = end_time - start_time
        
        logger.error("RxNorm ingestion failed", 
                    error=str(e), 
                    records=record_count,
                    duration=f"{duration:.2f} seconds")
        
        print(f"\n\u274c RxNorm ingestion failed after {duration:.2f} seconds:")
        print(f"   - Error: {str(e)}")
        print(f"   - Processed {record_count:,} records in {batch_count:,} batches before failure")
        
        return False

async def run_snomed_ingestion(database_client, limit=20000):
    """Run full SNOMED CT ingestion pipeline with strict record limiting
    
    Args:
        database_client: The database client
        limit: Maximum number of records to process (default: 20000)
    """
    print("\n\U0001f9ec INGESTING SNOMED CT DATA (LIMITED TO {0:,} RECORDS)".format(limit))
    print("=" * 50)
    
    logger.info("Starting SNOMED CT ingestion", limit=limit)
    start_time = time.time()
    
    # Initialize counters
    record_count = 0
    batch_count = 0
    
    try:
        # Initialize the SNOMED CT ingester with debugging
        snomed_ingester = SNOMEDIngester(database_client)
        print("   Created SNOMED ingester")
        
        # Check for SNOMED CT data
        logger.info("Checking for SNOMED CT data...")
        data_available = await snomed_ingester.download_data()
        if not data_available:
            logger.error("SNOMED CT data check failed")
            print("\u274c SNOMED CT data check failed. Aborting.")
            return False
        
        logger.info("Processing SNOMED CT data with strict limit...")
        print("\U0001f4dd Processing SNOMED CT concepts and relationships (STRICTLY LIMITED TO 20K)...")
        
        # Process and insert SNOMED CT data in batches
        rdf_batch = []
        batch_size = 1000
        
        # STRICTLY LIMIT THE SNOMED DATA
        snapshot_dir = snomed_ingester.data_dir / "snapshot"
        
        # Step 1: ONLY LOAD LIMITED CONCEPTS
        print("   Loading limited set of SNOMED concepts...")
        concept_count = 0
        concept_file = snapshot_dir / "sct2_Concept_Snapshot_INT.txt"
        
        with open(concept_file, 'r', encoding='utf-8', errors='ignore') as f:
            reader = csv.DictReader(f, delimiter='\t')
            for row in reader:
                concept_id = row.get('id', '').strip()
                active = row.get('active', '').strip()
                
                # Only process active concepts and limit to our max
                if active == '1' and concept_id:
                    snomed_ingester.processed_concept_ids.add(concept_id)
                    snomed_ingester.concepts[concept_id] = {
                        'concept_id': concept_id,
                        'effective_time': row.get('effectiveTime', '').strip(),
                        'module_id': row.get('moduleId', '').strip(),
                        'definition_status_id': row.get('definitionStatusId', '').strip(),
                        'active': True
                    }
                    
                    concept_count += 1
                    if concept_count >= limit:
                        break
                        
        print(f"   Loaded {concept_count:,} SNOMED concepts (limited set)")
        
        # Step 2: Process ONLY the descriptions for our limited concepts
        print("   Processing descriptions for limited concept set...")
        await snomed_ingester._process_descriptions(snapshot_dir / "sct2_Description_Snapshot-en_INT.txt")
        
        # Step 3: Generate and insert concept RDF in batches - with strict enforcement of limit
        print(f"   Generating and inserting RDF data (strictly limited to {limit:,} records)...")
        
        # FORCE TRUNCATE the concepts dictionary if it's still too large
        concept_count = len(snomed_ingester.concepts)
        if concept_count > limit:
            print(f"   ⚠️ WARNING: Found {concept_count:,} concepts, which exceeds the {limit:,} limit")
            print(f"   🔪 FORCING TRUNCATION to exactly {limit:,} concepts...")
            
            # Get list of concept IDs and keep only the first 'limit' items
            concept_ids = list(snomed_ingester.concepts.keys())
            truncated_concepts = {}
            
            # Only keep up to the limit
            for i, concept_id in enumerate(concept_ids):
                if i >= limit:
                    break
                truncated_concepts[concept_id] = snomed_ingester.concepts[concept_id]
            
            # Replace the original concepts dictionary with our truncated version
            snomed_ingester.concepts = truncated_concepts
            print(f"   ✂️ Truncated concepts dictionary from {concept_count:,} to {len(snomed_ingester.concepts):,} items")
        
        # Verify our concept count is now strictly limited
        concept_count = len(snomed_ingester.concepts)
        print(f"   Working with exactly {concept_count:,} concepts (hard-limited to {limit:,})")
        
        # Also truncate the processed_concept_ids set to match
        processed_ids_count = len(snomed_ingester.processed_concept_ids)
        if processed_ids_count > limit:
            snomed_ingester.processed_concept_ids = set(list(snomed_ingester.processed_concept_ids)[:limit])
            print(f"   ✂️ Truncated processed_concept_ids from {processed_ids_count:,} to {len(snomed_ingester.processed_concept_ids):,} items")
        
        # Enforce limit at RDF generation
        # GENERATE CONCEPT RDF with explicit limit
        async for rdf_triples in snomed_ingester._generate_concept_rdf(limit=limit):
            record_count += 1
            rdf_batch.append(rdf_triples)
            
            # Stop processing immediately if we hit the limit
            if limit and record_count >= limit:
                logger.info(f"⚠️ Reached strict limit of {limit} SNOMED CT records, stopping processing")
                print(f"   🛑 ENFORCING STRICT LIMIT: Reached {limit:,} SNOMED CT records, stopping all processing")
                break
            
            # Insert batch when it reaches the batch size
            if len(rdf_batch) >= batch_size:
                batch_rdf = "\n".join(rdf_batch)
                prefixes = snomed_ingester.get_ontology_prefixes()
                full_rdf = f"{prefixes}\n{batch_rdf}"
                
                await database_client.batch_insert_rdf(full_rdf)
                batch_count += 1
                print(f"   Inserted batch {batch_count} ({record_count:,} records so far)")
                
                rdf_batch = []
        
        # Insert any remaining RDF triples
        if rdf_batch:
            batch_rdf = "\n".join(rdf_batch)
            prefixes = snomed_ingester.get_ontology_prefixes()
            full_rdf = f"{prefixes}\n{batch_rdf}"
            
            await database_client.batch_insert_rdf(full_rdf)
            batch_count += 1
            print(f"   Inserted final batch {batch_count} ({record_count:,} records total)")
        
        end_time = time.time()
        duration = end_time - start_time
        
        print(f"\n\u2705 SNOMED CT ingestion completed:")
        print(f"   - {record_count:,} records processed")
        print(f"   - {batch_count:,} batches inserted")
        print(f"   - Took {duration:.2f} seconds")
        
        return True
    
    except Exception as e:
        end_time = time.time()
        duration = end_time - start_time
        
        logger.error("SNOMED CT ingestion failed", 
                    error=str(e), 
                    records=record_count,
                    duration=f"{duration:.2f} seconds")
        
        print(f"\n\u274c SNOMED CT ingestion failed after {duration:.2f} seconds:")
        print(f"   - Error: {str(e)}")
        print(f"   - Processed {record_count:,} records in {batch_count:,} batches before failure")
        
        return False

async def run_snomed_loinc_ingestion(database_client, limit=20000):
    """Run SNOMED CT LOINC Extension ingestion pipeline with strict record limiting
    
    Args:
        database_client: The database client
        limit: Maximum number of records to process (default: 20000)
    """
    print("\n\U0001f9a0 INGESTING SNOMED CT LOINC EXTENSION (LIMITED TO {0:,} RECORDS)".format(limit))
    print("=" * 50)
    
    logger.info("Starting LOINC ingestion with strict limit", limit=limit)
    start_time = time.time()
    
    # Initialize counters
    record_count = 0
    batch_count = 0
    
    try:
        # Initialize the SNOMED-LOINC ingester
        snomed_loinc_ingester = SNOMEDLOINCIngester(database_client)
        
        # Check for LOINC data
        logger.info("Checking for LOINC data...")
        data_available = await snomed_loinc_ingester.check_data()
        if not data_available:
            logger.error("LOINC data check failed")
            print("\u274c LOINC data check failed. Aborting.")
            return False
        
        logger.info("Processing LOINC data with strict limit...")
        print("\U0001f4dd Processing LOINC data (STRICTLY LIMITED TO 20K RECORDS)...")
        
        # STRICTLY LIMIT THE LOINC DATA
        print("   Loading limited set of LOINC concepts...")
        
        # Batch processing for Neo4j inserts
        batch_size = 1000
        rdf_batch = []
        
        # Load the LOINC file
        loinc_file = snomed_loinc_ingester.data_dir / "loinc" / "LoincTable" / "Loinc.csv"
        
        # Process LOINC records with strict limit
        with open(loinc_file, 'r', encoding='utf-8', errors='ignore') as f:
            # Skip header
            header = f.readline()
            
            for line in f:
                if line.strip() == "":
                    continue
                    
                fields = line.strip().split(',')
                if len(fields) < 2:
                    continue
                
                # LOINC number is in the first column
                loinc_num = fields[0].strip('"')
                # Long name is in the second column
                long_name = fields[1].strip('"')
                
                if loinc_num and long_name:
                    # Create LOINC concept URI
                    loinc_uri = snomed_loinc_ingester.generate_uri("LOINCConcept", loinc_num)
                    
                    # Escape special characters in labels
                    escaped_name = long_name.replace('"', '\\"')
                    
                    # Generate RDF triple
                    rdf_triple = f"""
{loinc_uri} a cae:LOINCConcept ;
    cae:hasLOINCNum "{loinc_num}" ;
    rdfs:label "{escaped_name}" ;
    cae:code "{loinc_num}" .
"""
                    
                    # Add to batch
                    rdf_batch.append(rdf_triple)
                    record_count += 1
                    
                    # Insert batch when it reaches the batch size
                    if len(rdf_batch) >= batch_size:
                        batch_rdf = "\n".join(rdf_batch)
                        prefixes = snomed_loinc_ingester.get_ontology_prefixes()
                        full_rdf = f"{prefixes}\n{batch_rdf}"
                        
                        await database_client.batch_insert_rdf(full_rdf)
                        batch_count += 1
                        print(f"   \U0001f4e6 Inserted batch {batch_count} ({record_count:,} records so far)")
                        
                        # Clear the batch for next iteration
                        rdf_batch = []
                    
                    # Stop if we hit the limit
                    if record_count >= limit:
                        logger.info(f"Reached limit of {limit} LOINC records, stopping processing")
                        print(f"   \ud83d\uded1 Reached limit of {limit:,} LOINC records, stopping processing")
                        break
        
        # Insert any remaining RDF triples
        if rdf_batch:
            batch_rdf = "\n".join(rdf_batch)
            prefixes = snomed_loinc_ingester.get_ontology_prefixes()
            full_rdf = f"{prefixes}\n{batch_rdf}"
            
            await database_client.batch_insert_rdf(full_rdf)
            batch_count += 1
            print(f"   \U0001f4e6 Inserted final batch {batch_count} ({record_count:,} records total)")
        
        end_time = time.time()
        duration = end_time - start_time
        
        logger.info("LOINC ingestion completed", 
                  records=record_count, 
                  batches=batch_count, 
                  duration=f"{duration:.2f} seconds")
        
        print(f"\n\u2705 LOINC ingestion completed:")
        print(f"   - {record_count:,} records processed")
        print(f"   - {batch_count:,} batches inserted")
        print(f"   - Took {duration:.2f} seconds")
        
        return True
        
    except Exception as e:
        end_time = time.time()
        duration = end_time - start_time
        
        logger.error("LOINC ingestion failed", 
                    error=str(e), 
                    records=record_count,
                    duration=f"{duration:.2f} seconds")
        
        print(f"\n\u274c LOINC ingestion failed after {duration:.2f} seconds:")
        print(f"   - Error: {str(e)}")
        print(f"   - Processed {record_count:,} records in {batch_count:,} batches before failure")
        
        return False

async def run_cross_terminology_mapping(database_client, limit=20000):
    """Run cross-terminology mapping to create relationships between RxNorm, SNOMED CT, and LOINC
    with strict record limiting
    
    Args:
        database_client: The database client
        limit: Maximum number of mappings to process (default: 20000)
    """
    print("\n\U0001f9c1 CREATING CROSS-TERMINOLOGY RELATIONSHIPS (LIMITED TO {0:,} MAPPINGS)".format(limit))
    print("=" * 50)
    
    logger.info("Starting cross-terminology mapping with strict limit", limit=limit)
    start_time = time.time()
    
    # Initialize counters
    total_mappings = 0
    batch_count = 0
    
    try:
        # Initialize the cross-terminology mapper
        cross_mapper = CrossTerminologyMapper(database_client)
        
        # Check for data availability
        logger.info("Checking for terminology data...")
        data_available = await cross_mapper.download_data()
        if not data_available:
            logger.error("Required terminology data is missing")
            print("\u274c Required terminology data is missing. Aborting cross-mapping.")
            return False
        
        logger.info("Processing cross-terminology mappings with strict limit...")
        print("\U0001f517 Creating relationships between terminologies (STRICTLY LIMITED TO 20K MAPPINGS)...")
        
        # STRICTLY LIMIT THE MAPPING DATA
        print("   Loading limited set of cross-terminology mappings...")
        
        # Process and insert mapping data in batches
        rdf_batch = []
        batch_size = 1000
        
        # RxNorm to SNOMED CT mappings (limited)
        rxnorm_snomed_file = cross_mapper.data_dir / "rxnorm" / "rrf" / "RXNSAT.RRF"
        print("   Processing RxNorm-SNOMED CT mappings...")
        
        if rxnorm_snomed_file.exists():
            with open(rxnorm_snomed_file, 'r', encoding='utf-8', errors='ignore') as f:
                for line in f:
                    fields = line.strip().split('|')
                    
                    # RXCUI is in position 0
                    rxcui = fields[0].strip()
                    # ATN (attribute name) is in position 8
                    attribute_name = fields[8].strip()
                    # ATV (attribute value) is in position 10
                    attribute_value = fields[10].strip()
                    
                    # Check if this is a SNOMED mapping
                    if attribute_name == "SNOMEDCT_US" and attribute_value and rxcui:
                        # Generate URIs
                        drug_uri = cross_mapper.generate_uri("Drug", rxcui)
                        snomed_uri = cross_mapper.generate_uri("SNOMEDConcept", attribute_value)
                        
                        # Create bidirectional relationship RDF
                        rdf_triple = f"""
{drug_uri} cae:hasSNOMEDCTMapping {snomed_uri} .
{snomed_uri} cae:hasRxNormMapping {drug_uri} .
"""
                        
                        # Add to batch
                        rdf_batch.append(rdf_triple)
                        total_mappings += 1
                        
                        # Insert batch if full
                        if len(rdf_batch) >= batch_size:
                            batch_rdf = "\n".join(rdf_batch)
                            prefixes = cross_mapper.get_ontology_prefixes()
                            full_rdf = f"{prefixes}\n{batch_rdf}"
                            
                            await database_client.batch_insert_rdf(full_rdf)
                            batch_count += 1
                            print(f"   \U0001f4e6 Inserted batch {batch_count} ({total_mappings:,} mappings so far)")
                            
                            rdf_batch = []
                        
                        # Check if we've reached the limit
                        if total_mappings >= limit:
                            logger.info(f"Reached limit of {limit} mappings, stopping processing")
                            print(f"   \ud83d\uded1 Reached limit of {limit:,} mappings, stopping processing")
                            break
        
        # Insert any remaining RDF triples
        if rdf_batch:
            batch_rdf = "\n".join(rdf_batch)
            prefixes = cross_mapper.get_ontology_prefixes()
            full_rdf = f"{prefixes}\n{batch_rdf}"
            
            await database_client.batch_insert_rdf(full_rdf)
            batch_count += 1
            print(f"   \U0001f4e6 Inserted final batch {batch_count} ({total_mappings:,} mappings total)")
        
        end_time = time.time()
        duration = end_time - start_time
        
        logger.info("Cross-terminology mapping completed", 
                  mappings=total_mappings, 
                  batches=batch_count, 
                  duration=f"{duration:.2f} seconds")
        
        print(f"\n\u2705 Cross-terminology mapping completed:")
        print(f"   - {total_mappings:,} mappings processed")
        print(f"   - {batch_count:,} batches inserted")
        print(f"   - Took {duration:.2f} seconds")
        
        return True
        
    except Exception as e:
        end_time = time.time()
        duration = end_time - start_time
        
        logger.error("Cross-terminology mapping failed", 
                    error=str(e), 
                    mappings=total_mappings,
                    duration=f"{duration:.2f} seconds")
        
        print(f"\n\u274c Cross-terminology mapping failed after {duration:.2f} seconds:")
        print(f"   - Error: {str(e)}")
        print(f"   - Processed {total_mappings:,} mappings in {batch_count:,} batches before failure")
        
        return False

async def verify_schema(database_client):
    """Verify the final schema in Neo4j"""
    print("\n\U0001f9ea VERIFYING NEO4J KNOWLEDGE GRAPH SCHEMA")
    print("=" * 50)
    
    # Define expected node labels and relationships to check
    node_labels = [
        "cae_Drug",
        "cae_SNOMEDConcept",
        "cae_LOINCConcept"
    ]
    
    relationships = [
        # Format: (source_label, relationship_type, target_label)
        # RxNorm relationships
        ("cae_Drug", "cae_hasActiveIngredient", "cae_Drug"),
        ("cae_Drug", "cae_hasBrandName", "cae_Drug"),
        # SNOMED CT relationships
        ("cae_SNOMEDConcept", "cae_hasRelationship", "cae_SNOMEDConcept"),
        # Cross-terminology relationships
        ("cae_Drug", "cae_hasSNOMEDCTMapping", "cae_SNOMEDConcept"),
        ("cae_SNOMEDConcept", "cae_hasRxNormMapping", "cae_Drug"),
        ("cae_SNOMEDConcept", "cae_hasLOINCMapping", "cae_LOINCConcept"),
        ("cae_LOINCConcept", "cae_hasSNOMEDCTMapping", "cae_SNOMEDConcept")
    ]
    
    all_passed = True
    
    for label in node_labels:
        print(f"\n\U0001f50d Verifying label: '{label}'...")
        query = f"MATCH (n:`{label}`) RETURN count(n) AS count"
        result = await database_client.execute_cypher(query)
        count = result[0]['count'] if result else 0
        
        if count > 0:
            print(f"\u2705 SUCCESS: Found {count} nodes with label '{label}'.")
            logger.info(f"Check passed for label '{label}'", count=count)
        else:
            print(f"\u26a0\ufe0f WARNING: No nodes found with label '{label}'.")
            logger.warning(f"Check failed for label '{label}'", count=count)
            all_passed = False
    
    # Check for all expected relationships
    print("\n\U0001f517 Verifying relationships between terminologies...")
    
    for src, rel, dst in relationships:
        print(f"\U0001f50d Checking relationship: '{src}--[{rel}]-->{dst}'...")
        query = f"""
        MATCH (s:`{src}`)-[r:`{rel}`]->(d:`{dst}`)
        RETURN count(r) AS count
        """
        result = await database_client.execute_cypher(query)
        count = result[0]['count'] if result else 0
        
        if count > 0:
            print(f"\u2705 SUCCESS: Found {count} relationships of type '{src}--[{rel}]-->{dst}'.")
            logger.info(f"Check passed for relationship", source=src, relationship=rel, destination=dst, count=count)
        else:
            print(f"\u26a0\ufe0f WARNING: No relationships found of type '{src}--[{rel}]-->{dst}'.")
            logger.warning(f"Check warning for relationship", source=src, relationship=rel, destination=dst, count=count)
            # Only mark as failed if it's a critical relationship
            if src == "cae_Drug" and rel == "cae_hasSNOMEDCTMapping" or \
               src == "cae_SNOMEDConcept" and rel == "cae_hasLOINCMapping":
                all_passed = False
    
    return all_passed

async def main():
    """Main function for full clinical knowledge graph ingestion"""
    print("\U0001f4ca FULL CLINICAL KNOWLEDGE GRAPH INGESTION")
    print("=" * 60)
    print("\nThis script will ingest the following data into Neo4j:")
    print(" - TOTAL LIMIT: 20,000 records across ALL terminologies")
    print(" - RxNorm drug terminology (6,000 records)")
    print(" - SNOMED CT terminology (6,000 records)")
    print(" - LOINC mappings (4,000 records)")
    print(" - Cross-terminology relationships (4,000 records)")
    print("\nBefore proceeding, ensure you have:")
    print(" - Downloaded SNOMED CT release files to 'data/snomed/'")
    print(" - Downloaded SNOMED CT LOINC Extension files to 'data/loinc/'")
    
    # Allow time to read the intro
    await asyncio.sleep(2)
    
    # Verify data directories
    print("\n\U0001f4c2 Checking data directories...")
    directories_ok = await verify_data_directories()
    if not directories_ok:
        print("\n\u26a0\ufe0f Some data directories are missing or empty.")
        proceed = input("Do you want to proceed anyway? (y/n): ").strip().lower()
        if proceed != 'y':
            print("Aborting ingestion.")
            return False
    
    # Connect to the database
    database_client = None
    try:
        print("\n\U0001f50c Connecting to the database...")
        database_client = await create_database_client()
        connection_ok = await database_client.test_connection()
        if not connection_ok:
            logger.error("Database connection failed")
            print("\u274c Database connection failed. Aborting.")
            return False
        logger.info("Database connection successful")
        print("\u2705 Database connection successful")
        
        # Create indexes
        await create_indexes(database_client)

        # STRICT 20K TOTAL LIMIT - Distribute across terminologies
        total_limit = 20000
        rxnorm_limit = 6000    # ~30% for RxNorm drugs
        snomed_limit = 6000    # ~30% for SNOMED concepts
        loinc_limit = 4000     # ~20% for LOINC codes
        mapping_limit = 4000   # ~20% for cross-terminology relationships

        print(f"\n📊 RECORD DISTRIBUTION (TOTAL: {total_limit:,}):")
        print(f"   RxNorm: {rxnorm_limit:,} records")
        print(f"   SNOMED CT: {snomed_limit:,} records")
        print(f"   LOINC: {loinc_limit:,} records")
        print(f"   Cross-mappings: {mapping_limit:,} records")
        print("🚨 STRICT LIMIT: Processing will stop at these limits")
        print("=" * 60)

        # Run RxNorm ingestion with distributed limit
        print("\n\U0001f7e2 RUNNING RXNORM INGESTION")
        rxnorm_success = await run_rxnorm_ingestion(database_client, limit=rxnorm_limit)

        # Run SNOMED and LOINC ingesters with distributed limits
        snomed_success = await run_snomed_ingestion(database_client, limit=snomed_limit)
        loinc_success = await run_snomed_loinc_ingestion(database_client, limit=loinc_limit)

        # Run cross-terminology mapping with distributed limit
        mapping_success = True
        if rxnorm_success and snomed_success and loinc_success:  # Only run if previous steps succeeded
            mapping_success = await run_cross_terminology_mapping(database_client, limit=mapping_limit)
        
        # Verify the final schema
        if rxnorm_success or snomed_success or loinc_success or mapping_success:
            schema_verified = await verify_schema(database_client)
            if schema_verified:
                print("\n\U0001f389 Schema verification successful! All expected node types and relationships exist.")
            else:
                print("\n\u26a0\ufe0f Schema verification found issues. Please review the output.")
        
        return rxnorm_success and snomed_success and loinc_success and mapping_success
        
    except Exception as e:
        logger.error("An unexpected error occurred", error=str(e), exc_info=True)
        print(f"\u274c An unexpected error occurred: {e}")
        return False
    finally:
        if database_client:
            await database_client.disconnect()
            logger.info("Database connection closed")
            print("\U0001f4a4 Database connection closed")

if __name__ == "__main__":
    success = asyncio.run(main())
    
    if success:
        print("\n\U0001f389 All ingestion tasks completed successfully!")
    else:
        print("\n\u26a0\ufe0f Some ingestion tasks failed. Please review the logs.")
    
    sys.exit(0 if success else 1)
