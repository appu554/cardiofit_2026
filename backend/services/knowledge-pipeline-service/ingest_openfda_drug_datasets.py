#!/usr/bin/env python3
"""
Ingest OpenFDA Drug Datasets: Drug Labeling, NDC Directory, and Drugs@FDA
"""

import asyncio
import sys
import time
import json
import requests
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.database_factory import create_database_client

# OpenFDA API Configuration
OPENFDA_API_KEY = "Fd4NqfzTO03RYq4KINOZwg8lYz7sgkDriTeGYMnB"

# OpenFDA Drug API Endpoints
DRUG_LABEL_URL = "https://api.fda.gov/drug/label.json"
DRUG_NDC_URL = "https://api.fda.gov/drug/ndc.json"
DRUGS_FDA_URL = "https://api.fda.gov/drug/drugsfda.json"

def fetch_openfda_drug_data(url, skip=0, limit=100, search_query=None):
    """Fetch OpenFDA drug data from any endpoint"""
    params = {
        'api_key': OPENFDA_API_KEY,
        'skip': skip,
        'limit': limit
    }
    
    if search_query:
        params['search'] = search_query
    
    try:
        print(f"   📡 Requesting data from {url.split('/')[-1]} (skip={skip}, limit={limit})")
        response = requests.get(url, params=params, timeout=30)
        
        if response.status_code == 200:
            data = response.json()
            results = data.get('results', [])
            total = data.get('meta', {}).get('results', {}).get('total', 0)
            print(f"   ✅ Received {len(results)} records (total available: {total:,})")
            return results
        elif response.status_code == 404:
            print(f"   ℹ️ No more data available (404)")
            return []
        else:
            print(f"   ⚠️ API request failed: {response.status_code}")
            print(f"   Response: {response.text[:200]}...")
            return []
    except Exception as e:
        print(f"   ⚠️ API request error: {e}")
        return []

async def create_drug_label_node(database_client, label_data, label_id):
    """Create a drug label node in Neo4j"""
    try:
        # Extract key information from drug label
        openfda = label_data.get('openfda', {})
        brand_name = openfda.get('brand_name', ['Unknown'])[0] if openfda.get('brand_name') else 'Unknown'
        generic_name = openfda.get('generic_name', ['Unknown'])[0] if openfda.get('generic_name') else 'Unknown'
        manufacturer_name = openfda.get('manufacturer_name', ['Unknown'])[0] if openfda.get('manufacturer_name') else 'Unknown'
        
        # Get dosage and administration info
        dosage_and_administration = label_data.get('dosage_and_administration', [''])[0][:200] if label_data.get('dosage_and_administration') else ''
        
        # Get warnings
        warnings = label_data.get('warnings', [''])[0][:200] if label_data.get('warnings') else ''
        
        # Clean strings for Cypher
        brand_name_clean = brand_name.replace("'", "\\'").replace('"', '\\"')[:100]
        generic_name_clean = generic_name.replace("'", "\\'").replace('"', '\\"')[:100]
        manufacturer_clean = manufacturer_name.replace("'", "\\'").replace('"', '\\"')[:100]
        dosage_clean = dosage_and_administration.replace("'", "\\'").replace('"', '\\"')
        warnings_clean = warnings.replace("'", "\\'").replace('"', '\\"')
        
        # Create Cypher query
        cypher_query = f"""
        MERGE (label:cae_DrugLabel {{label_id: '{label_id}'}})
        SET label.brand_name = '{brand_name_clean}',
            label.generic_name = '{generic_name_clean}',
            label.manufacturer_name = '{manufacturer_clean}',
            label.dosage_and_administration = '{dosage_clean}',
            label.warnings = '{warnings_clean}',
            label.created_at = datetime()
        """
        
        await database_client.execute_cypher(cypher_query)
        return True
        
    except Exception as e:
        print(f"   ⚠️ Failed to create drug label {label_id}: {e}")
        return False

async def create_ndc_node(database_client, ndc_data, ndc_id):
    """Create an NDC (National Drug Code) node in Neo4j"""
    try:
        # Extract key information from NDC data
        product_ndc = ndc_data.get('product_ndc', 'Unknown')
        brand_name = ndc_data.get('brand_name', 'Unknown')
        generic_name = ndc_data.get('generic_name', 'Unknown')
        labeler_name = ndc_data.get('labeler_name', 'Unknown')
        product_type = ndc_data.get('product_type', 'Unknown')
        route = ndc_data.get('route', ['Unknown'])[0] if ndc_data.get('route') else 'Unknown'
        
        # Clean strings for Cypher
        brand_name_clean = brand_name.replace("'", "\\'").replace('"', '\\"')[:100]
        generic_name_clean = generic_name.replace("'", "\\'").replace('"', '\\"')[:100]
        labeler_clean = labeler_name.replace("'", "\\'").replace('"', '\\"')[:100]
        product_type_clean = product_type.replace("'", "\\'").replace('"', '\\"')[:50]
        route_clean = route.replace("'", "\\'").replace('"', '\\"')[:50]
        
        # Create Cypher query
        cypher_query = f"""
        MERGE (ndc:cae_NDC {{ndc_id: '{ndc_id}', product_ndc: '{product_ndc}'}})
        SET ndc.brand_name = '{brand_name_clean}',
            ndc.generic_name = '{generic_name_clean}',
            ndc.labeler_name = '{labeler_clean}',
            ndc.product_type = '{product_type_clean}',
            ndc.route = '{route_clean}',
            ndc.created_at = datetime()
        """
        
        await database_client.execute_cypher(cypher_query)
        return True
        
    except Exception as e:
        print(f"   ⚠️ Failed to create NDC {ndc_id}: {e}")
        return False

async def create_drugsfda_node(database_client, drugsfda_data, drugsfda_id):
    """Create a Drugs@FDA node in Neo4j"""
    try:
        # Extract key information from Drugs@FDA data
        application_number = drugsfda_data.get('application_number', 'Unknown')
        sponsor_name = drugsfda_data.get('sponsor_name', 'Unknown')
        
        # Get products info (first product if multiple)
        products = drugsfda_data.get('products', [])
        if products:
            product = products[0]
            brand_name = product.get('brand_name', 'Unknown')
            active_ingredients = product.get('active_ingredients', [])
            active_ingredient = active_ingredients[0].get('name', 'Unknown') if active_ingredients else 'Unknown'
            dosage_form = product.get('dosage_form', 'Unknown')
            route = product.get('route', 'Unknown')
        else:
            brand_name = 'Unknown'
            active_ingredient = 'Unknown'
            dosage_form = 'Unknown'
            route = 'Unknown'
        
        # Clean strings for Cypher
        sponsor_clean = sponsor_name.replace("'", "\\'").replace('"', '\\"')[:100]
        brand_name_clean = brand_name.replace("'", "\\'").replace('"', '\\"')[:100]
        active_ingredient_clean = active_ingredient.replace("'", "\\'").replace('"', '\\"')[:100]
        dosage_form_clean = dosage_form.replace("'", "\\'").replace('"', '\\"')[:50]
        route_clean = route.replace("'", "\\'").replace('"', '\\"')[:50]
        
        # Create Cypher query
        cypher_query = f"""
        MERGE (drugfda:cae_DrugsFDA {{drugsfda_id: '{drugsfda_id}', application_number: '{application_number}'}})
        SET drugfda.sponsor_name = '{sponsor_clean}',
            drugfda.brand_name = '{brand_name_clean}',
            drugfda.active_ingredient = '{active_ingredient_clean}',
            drugfda.dosage_form = '{dosage_form_clean}',
            drugfda.route = '{route_clean}',
            drugfda.created_at = datetime()
        """
        
        await database_client.execute_cypher(cypher_query)
        return True
        
    except Exception as e:
        print(f"   ⚠️ Failed to create Drugs@FDA {drugsfda_id}: {e}")
        return False

async def ingest_drug_labels(database_client, limit=5000):
    """Ingest OpenFDA Drug Label data"""
    print(f"\n📋 INGESTING DRUG LABELS (LIMITED TO {limit:,})")
    print("=" * 60)
    
    start_time = time.time()
    created_count = 0
    batch_size = 100
    
    try:
        skip = 0
        
        while created_count < limit:
            remaining = limit - created_count
            fetch_limit = min(batch_size, remaining)
            
            # Fetch data from OpenFDA Drug Label API
            labels = fetch_openfda_drug_data(DRUG_LABEL_URL, skip=skip, limit=fetch_limit)
            
            if not labels:
                print("   ⚠️ No more drug label data available")
                break
            
            # Process each label
            batch_success = 0
            for i, label in enumerate(labels):
                if created_count >= limit:
                    break
                
                label_id = f"label_{skip + i + 1}"
                success = await create_drug_label_node(database_client, label, label_id)
                
                if success:
                    batch_success += 1
                    created_count += 1
            
            print(f"   ✅ Created {batch_success} drug labels ({created_count:,} total)")
            
            skip += len(labels)
            time.sleep(0.5)  # Be respectful to the API
            
            if len(labels) < fetch_limit:
                break
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ Drug label ingestion completed!")
        print(f"   📊 Total drug labels created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        print(f"❌ Drug label ingestion failed: {e}")
        return False

async def ingest_ndc_directory(database_client, limit=5000):
    """Ingest OpenFDA NDC Directory data"""
    print(f"\n🏷️ INGESTING NDC DIRECTORY (LIMITED TO {limit:,})")
    print("=" * 60)
    
    start_time = time.time()
    created_count = 0
    batch_size = 100
    
    try:
        skip = 0
        
        while created_count < limit:
            remaining = limit - created_count
            fetch_limit = min(batch_size, remaining)
            
            # Fetch data from OpenFDA NDC API
            ndcs = fetch_openfda_drug_data(DRUG_NDC_URL, skip=skip, limit=fetch_limit)
            
            if not ndcs:
                print("   ⚠️ No more NDC data available")
                break
            
            # Process each NDC
            batch_success = 0
            for i, ndc in enumerate(ndcs):
                if created_count >= limit:
                    break
                
                ndc_id = f"ndc_{skip + i + 1}"
                success = await create_ndc_node(database_client, ndc, ndc_id)
                
                if success:
                    batch_success += 1
                    created_count += 1
            
            print(f"   ✅ Created {batch_success} NDC records ({created_count:,} total)")
            
            skip += len(ndcs)
            time.sleep(0.5)  # Be respectful to the API
            
            if len(ndcs) < fetch_limit:
                break
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ NDC directory ingestion completed!")
        print(f"   📊 Total NDC records created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        print(f"❌ NDC directory ingestion failed: {e}")
        return False

async def ingest_drugsfda(database_client, limit=5000):
    """Ingest OpenFDA Drugs@FDA data"""
    print(f"\n🏛️ INGESTING DRUGS@FDA (LIMITED TO {limit:,})")
    print("=" * 60)
    
    start_time = time.time()
    created_count = 0
    batch_size = 100
    
    try:
        skip = 0
        
        while created_count < limit:
            remaining = limit - created_count
            fetch_limit = min(batch_size, remaining)
            
            # Fetch data from OpenFDA Drugs@FDA API
            drugs = fetch_openfda_drug_data(DRUGS_FDA_URL, skip=skip, limit=fetch_limit)
            
            if not drugs:
                print("   ⚠️ No more Drugs@FDA data available")
                break
            
            # Process each drug
            batch_success = 0
            for i, drug in enumerate(drugs):
                if created_count >= limit:
                    break
                
                drugsfda_id = f"drugsfda_{skip + i + 1}"
                success = await create_drugsfda_node(database_client, drug, drugsfda_id)
                
                if success:
                    batch_success += 1
                    created_count += 1
            
            print(f"   ✅ Created {batch_success} Drugs@FDA records ({created_count:,} total)")
            
            skip += len(drugs)
            time.sleep(0.5)  # Be respectful to the API
            
            if len(drugs) < fetch_limit:
                break
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ Drugs@FDA ingestion completed!")
        print(f"   📊 Total Drugs@FDA records created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True

    except Exception as e:
        print(f"❌ Drugs@FDA ingestion failed: {e}")
        return False

async def create_drug_relationships(database_client, limit=2000):
    """Create relationships between different drug datasets"""
    print(f"\n🔗 CREATING DRUG DATASET RELATIONSHIPS (LIMITED TO {limit:,})")
    print("=" * 60)

    start_time = time.time()
    total_relationships = 0

    try:
        # Create relationships between RxNorm drugs and Drug Labels based on name similarity
        cypher_query = f"""
        MATCH (drug:cae_Drug), (label:cae_DrugLabel)
        WHERE toLower(drug.name) CONTAINS toLower(label.brand_name)
           OR toLower(label.brand_name) CONTAINS toLower(drug.name)
           OR toLower(drug.name) CONTAINS toLower(label.generic_name)
           OR toLower(label.generic_name) CONTAINS toLower(drug.name)
        WITH drug, label
        LIMIT {limit // 4}
        MERGE (drug)-[:cae_hasLabel]->(label)
        RETURN count(*) as relationships_created
        """

        result = await database_client.execute_cypher(cypher_query)
        drug_label_rels = result[0].get('relationships_created', 0) if result else 0
        total_relationships += drug_label_rels
        print(f"   💊➡️📋 RxNorm → Drug Labels: {drug_label_rels:,} relationships")

        # Create relationships between RxNorm drugs and NDC records
        cypher_query = f"""
        MATCH (drug:cae_Drug), (ndc:cae_NDC)
        WHERE toLower(drug.name) CONTAINS toLower(ndc.brand_name)
           OR toLower(ndc.brand_name) CONTAINS toLower(drug.name)
           OR toLower(drug.name) CONTAINS toLower(ndc.generic_name)
           OR toLower(ndc.generic_name) CONTAINS toLower(drug.name)
        WITH drug, ndc
        LIMIT {limit // 4}
        MERGE (drug)-[:cae_hasNDC]->(ndc)
        RETURN count(*) as relationships_created
        """

        result = await database_client.execute_cypher(cypher_query)
        drug_ndc_rels = result[0].get('relationships_created', 0) if result else 0
        total_relationships += drug_ndc_rels
        print(f"   💊➡️🏷️ RxNorm → NDC: {drug_ndc_rels:,} relationships")

        # Create relationships between RxNorm drugs and Drugs@FDA
        cypher_query = f"""
        MATCH (drug:cae_Drug), (drugfda:cae_DrugsFDA)
        WHERE toLower(drug.name) CONTAINS toLower(drugfda.brand_name)
           OR toLower(drugfda.brand_name) CONTAINS toLower(drug.name)
           OR toLower(drug.name) CONTAINS toLower(drugfda.active_ingredient)
           OR toLower(drugfda.active_ingredient) CONTAINS toLower(drug.name)
        WITH drug, drugfda
        LIMIT {limit // 4}
        MERGE (drug)-[:cae_hasFDAApproval]->(drugfda)
        RETURN count(*) as relationships_created
        """

        result = await database_client.execute_cypher(cypher_query)
        drug_fda_rels = result[0].get('relationships_created', 0) if result else 0
        total_relationships += drug_fda_rels
        print(f"   💊➡️🏛️ RxNorm → Drugs@FDA: {drug_fda_rels:,} relationships")

        # Create relationships between Drug Labels and NDC records
        cypher_query = f"""
        MATCH (label:cae_DrugLabel), (ndc:cae_NDC)
        WHERE toLower(label.brand_name) = toLower(ndc.brand_name)
           OR toLower(label.generic_name) = toLower(ndc.generic_name)
        WITH label, ndc
        LIMIT {limit // 4}
        MERGE (label)-[:cae_hasNDC]->(ndc)
        RETURN count(*) as relationships_created
        """

        result = await database_client.execute_cypher(cypher_query)
        label_ndc_rels = result[0].get('relationships_created', 0) if result else 0
        total_relationships += label_ndc_rels
        print(f"   📋➡️🏷️ Drug Labels → NDC: {label_ndc_rels:,} relationships")

        elapsed_time = time.time() - start_time
        print(f"\n✅ Drug dataset relationships created!")
        print(f"   📊 Total relationships: {total_relationships:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")

        return True

    except Exception as e:
        print(f"❌ Failed to create drug relationships: {e}")
        return False

async def verify_drug_datasets(database_client):
    """Verify the drug datasets were created correctly"""
    print(f"\n🔍 VERIFYING DRUG DATASETS")
    print("=" * 40)

    try:
        # Count each dataset
        datasets = [
            ('cae_DrugLabel', 'Drug Labels'),
            ('cae_NDC', 'NDC Records'),
            ('cae_DrugsFDA', 'Drugs@FDA Records')
        ]

        for label, name in datasets:
            result = await database_client.execute_cypher(f"MATCH (n:{label}) RETURN count(n) as count")
            count = result[0]['count'] if result else 0
            print(f"📋 {name}: {count:,} nodes")

        # Count relationships
        relationships = [
            ('cae_hasLabel', 'Drug → Label'),
            ('cae_hasNDC', 'Drug → NDC'),
            ('cae_hasFDAApproval', 'Drug → FDA'),
        ]

        print("\n🔗 Relationships:")
        for rel_type, description in relationships:
            result = await database_client.execute_cypher(f"""
                MATCH ()-[r:{rel_type}]->()
                RETURN count(r) as count
            """)
            count = result[0]['count'] if result else 0
            print(f"   {description}: {count:,}")

        # Sample data
        print("\n📋 Sample Drug Labels:")
        result = await database_client.execute_cypher("""
            MATCH (label:cae_DrugLabel)
            RETURN label.brand_name, label.generic_name, label.manufacturer_name
            LIMIT 3
        """)

        if result:
            for row in result:
                brand = row['label.brand_name'][:30] + "..." if len(row['label.brand_name']) > 30 else row['label.brand_name']
                generic = row['label.generic_name'][:30] + "..." if len(row['label.generic_name']) > 30 else row['label.generic_name']
                print(f"   {brand} ({generic}) - {row['label.manufacturer_name']}")

        return True

    except Exception as e:
        print(f"❌ Failed to verify drug datasets: {e}")
        return False

async def main():
    """Main function to ingest all OpenFDA drug datasets"""
    print("💊 OPENFDA DRUG DATASETS INGESTION")
    print("=" * 60)
    print("Adding Drug Labels, NDC Directory, and Drugs@FDA data")
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

        # Ingest each dataset (5K records each = 15K total)
        label_success = await ingest_drug_labels(database_client, limit=5000)
        ndc_success = await ingest_ndc_directory(database_client, limit=5000)
        drugsfda_success = await ingest_drugsfda(database_client, limit=5000)

        # Create relationships between datasets
        relationship_success = True
        if label_success or ndc_success or drugsfda_success:
            relationship_success = await create_drug_relationships(database_client, limit=2000)

        # Verify data
        verify_success = await verify_drug_datasets(database_client)

        # Final summary
        print("\n" + "=" * 60)
        print("📊 DRUG DATASETS INGESTION SUMMARY")
        print("=" * 60)
        print(f"Drug Labels: {'✅ Success' if label_success else '❌ Failed'}")
        print(f"NDC Directory: {'✅ Success' if ndc_success else '❌ Failed'}")
        print(f"Drugs@FDA: {'✅ Success' if drugsfda_success else '❌ Failed'}")
        print(f"Relationships: {'✅ Success' if relationship_success else '❌ Failed'}")
        print(f"Verification: {'✅ Success' if verify_success else '❌ Failed'}")
        print("🎉 OpenFDA drug datasets ingestion completed!")
        print("Your knowledge graph now includes comprehensive drug information!")
        print("=" * 60)

        return True

    except Exception as e:
        print(f"❌ Drug datasets ingestion failed: {e}")
        return False

if __name__ == "__main__":
    asyncio.run(main())
