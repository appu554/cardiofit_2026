#!/usr/bin/env python3
"""
Ingest 5,000 OpenFDA adverse event records into Neo4j Cloud
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
OPENFDA_BASE_URL = "https://api.fda.gov/drug/event.json"

def fetch_openfda_data(skip=0, limit=100):
    """Fetch OpenFDA adverse event data"""
    params = {
        'api_key': OPENFDA_API_KEY,
        'skip': skip,
        'limit': limit,
        'search': 'serious:1'  # Focus on serious adverse events only
    }

    try:
        print(f"   📡 Requesting OpenFDA data (skip={skip}, limit={limit})")
        response = requests.get(OPENFDA_BASE_URL, params=params, timeout=30)

        if response.status_code == 200:
            data = response.json()
            results = data.get('results', [])
            print(f"   ✅ Received {len(results)} records")
            return results
        else:
            print(f"   ⚠️ API request failed: {response.status_code}")
            print(f"   Response: {response.text[:200]}...")
            return []
    except Exception as e:
        print(f"   ⚠️ API request error: {e}")
        return []

async def create_adverse_event_node(database_client, event_data, event_id):
    """Create an adverse event node in Neo4j"""
    try:
        # Extract key information
        receive_date = event_data.get('receivedate', 'Unknown')
        serious = event_data.get('serious', 0)
        country = event_data.get('occurcountry', 'Unknown')
        
        # Extract patient info
        patient = event_data.get('patient', {})
        patient_age = patient.get('patientonsetage', 'Unknown')
        patient_sex = patient.get('patientsex', 'Unknown')
        
        # Extract drug info (first drug if multiple)
        drugs = patient.get('drug', [])
        primary_drug = drugs[0] if drugs else {}
        drug_name = primary_drug.get('medicinalproduct', 'Unknown')
        
        # Extract reactions (first reaction if multiple)
        reactions = patient.get('reaction', [])
        primary_reaction = reactions[0] if reactions else {}
        reaction_term = primary_reaction.get('reactionmeddrapt', 'Unknown')
        
        # Clean strings for Cypher
        drug_name_clean = drug_name.replace("'", "\\'").replace('"', '\\"')[:100]
        reaction_clean = reaction_term.replace("'", "\\'").replace('"', '\\"')[:100]
        country_clean = country.replace("'", "\\'").replace('"', '\\"')[:50]
        
        # Create Cypher query
        cypher_query = f"""
        MERGE (event:cae_AdverseEvent {{event_id: '{event_id}'}})
        SET event.receive_date = '{receive_date}',
            event.serious = {serious},
            event.country = '{country_clean}',
            event.patient_age = '{patient_age}',
            event.patient_sex = '{patient_sex}',
            event.drug_name = '{drug_name_clean}',
            event.reaction = '{reaction_clean}',
            event.created_at = datetime()
        """
        
        await database_client.execute_cypher(cypher_query)
        return True
        
    except Exception as e:
        print(f"   ⚠️ Failed to create adverse event {event_id}: {e}")
        return False

async def create_drug_adverse_event_relationships(database_client, limit=1000):
    """Create relationships between RxNorm drugs and adverse events"""
    print(f"\n🔗 CREATING DRUG-ADVERSE EVENT RELATIONSHIPS (LIMITED TO {limit:,})")
    print("=" * 60)
    
    start_time = time.time()
    
    try:
        # Create relationships between RxNorm drugs and adverse events based on drug names
        cypher_query = f"""
        MATCH (drug:cae_Drug), (event:cae_AdverseEvent)
        WHERE toLower(event.drug_name) CONTAINS toLower(drug.name) 
           OR toLower(drug.name) CONTAINS toLower(event.drug_name)
        WITH drug, event
        LIMIT {limit}
        MERGE (drug)-[:cae_hasAdverseEvent]->(event)
        RETURN count(*) as relationships_created
        """
        
        result = await database_client.execute_cypher(cypher_query)
        created_count = result[0].get('relationships_created', 0) if result else 0
        
        elapsed_time = time.time() - start_time
        print(f"✅ Drug-adverse event relationships created!")
        print(f"   📊 Total relationships: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to create drug-adverse event relationships: {e}")
        return False

async def ingest_openfda_adverse_events(database_client, limit=5000):
    """Ingest OpenFDA adverse event data"""
    print(f"\n⚠️ INGESTING OPENFDA ADVERSE EVENTS (LIMITED TO {limit:,})")
    print("=" * 60)
    
    start_time = time.time()
    created_count = 0
    batch_size = 100
    
    try:
        skip = 0

        while created_count < limit:
            # Calculate how many records to fetch in this batch
            remaining = limit - created_count
            fetch_limit = min(batch_size, remaining)

            print(f"📡 Fetching OpenFDA data (skip={skip}, limit={fetch_limit})...")

            # Fetch data from OpenFDA API
            events = fetch_openfda_data(skip=skip, limit=fetch_limit)

            if not events:
                print("   ⚠️ No more data available from OpenFDA API")
                break

            # Process each event
            batch_success = 0
            for i, event in enumerate(events):
                if created_count >= limit:
                    break

                event_id = f"fda_{skip + i + 1}"
                success = await create_adverse_event_node(database_client, event, event_id)

                if success:
                    batch_success += 1
                    created_count += 1

            print(f"   ✅ Created {batch_success} adverse events ({created_count:,} total)")

            # Move to next batch
            skip += len(events)

            # Small delay to be respectful to the API
            time.sleep(0.5)

            # Break if we didn't get a full batch (end of data)
            if len(events) < fetch_limit:
                print("   ℹ️ Reached end of available OpenFDA data")
                break
        
        elapsed_time = time.time() - start_time
        print(f"\n✅ OpenFDA adverse event ingestion completed!")
        print(f"   📊 Total adverse events created: {created_count:,}")
        print(f"   ⏱️ Time taken: {elapsed_time:.2f} seconds")
        
        return True
        
    except Exception as e:
        print(f"❌ OpenFDA adverse event ingestion failed: {e}")
        return False

async def verify_openfda_data(database_client):
    """Verify the OpenFDA data was created correctly"""
    print(f"\n🔍 VERIFYING OPENFDA DATA")
    print("=" * 40)
    
    try:
        # Count adverse events
        result = await database_client.execute_cypher("MATCH (e:cae_AdverseEvent) RETURN count(e) as count")
        event_count = result[0]['count'] if result else 0
        print(f"⚠️ Adverse events: {event_count:,} nodes")
        
        # Count drug-adverse event relationships
        result = await database_client.execute_cypher("""
            MATCH (drug:cae_Drug)-[r:cae_hasAdverseEvent]->(event:cae_AdverseEvent)
            RETURN count(r) as count
        """)
        rel_count = result[0]['count'] if result else 0
        print(f"🔗 Drug-adverse event relationships: {rel_count:,}")
        
        # Sample adverse events
        result = await database_client.execute_cypher("""
            MATCH (e:cae_AdverseEvent)
            RETURN e.event_id, e.drug_name, e.reaction, e.country
            LIMIT 5
        """)
        
        if result:
            print("\n📋 Sample adverse events:")
            for row in result:
                drug_name = row['e.drug_name'][:30] + "..." if len(row['e.drug_name']) > 30 else row['e.drug_name']
                reaction = row['e.reaction'][:30] + "..." if len(row['e.reaction']) > 30 else row['e.reaction']
                print(f"   {row['e.event_id']}: {drug_name} → {reaction} ({row['e.country']})")
        
        return True
        
    except Exception as e:
        print(f"❌ Failed to verify OpenFDA data: {e}")
        return False

async def main():
    """Main function to ingest OpenFDA data"""
    print("⚠️ OPENFDA ADVERSE EVENTS INGESTION")
    print("=" * 60)
    print("Adding real-world adverse event data to clinical knowledge graph")
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
        
        # Step 1: Ingest adverse events
        ingest_success = await ingest_openfda_adverse_events(database_client, limit=5000)
        if not ingest_success:
            return False
        
        # Step 2: Create relationships with existing drugs
        relationship_success = await create_drug_adverse_event_relationships(database_client, limit=1000)
        
        # Step 3: Verify data
        verify_success = await verify_openfda_data(database_client)
        
        # Final summary
        print("\n" + "=" * 60)
        print("📊 OPENFDA INGESTION SUMMARY")
        print("=" * 60)
        print(f"Adverse events: {'✅ Success' if ingest_success else '❌ Failed'}")
        print(f"Drug relationships: {'✅ Success' if relationship_success else '❌ Failed'}")
        print(f"Data verification: {'✅ Success' if verify_success else '❌ Failed'}")
        print("🎉 OpenFDA ingestion completed!")
        print("Your knowledge graph now includes real-world adverse event data!")
        print("=" * 60)
        
        return True
        
    except Exception as e:
        print(f"❌ OpenFDA ingestion failed: {e}")
        return False

if __name__ == "__main__":
    asyncio.run(main())
