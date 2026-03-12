"""
Neo4j Data Verification Script

This script checks what data actually exists in your Neo4j database
and identifies why the CAE queries are failing.
"""

import asyncio
import logging
from dotenv import load_dotenv
load_dotenv()

import sys
from pathlib import Path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

from app.knowledge.neo4j_client import Neo4jCloudClient

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def check_neo4j_data():
    """Check what data exists in Neo4j"""
    print("🔍 Checking Neo4j Database Contents")
    print("=" * 50)
    
    client = Neo4jCloudClient()
    
    try:
        # Connect to Neo4j
        connected = await client.connect()
        if not connected:
            print("❌ Failed to connect to Neo4j")
            return
        
        print("✅ Connected to Neo4j Cloud")
        print()
        
        # Check 1: What node labels exist?
        print("📊 1. Checking Node Labels...")
        labels_query = "CALL db.labels()"
        labels_result = await client.execute_cypher(labels_query)
        
        if labels_result:
            print("Found labels:")
            for record in labels_result:
                label = record.get('label', 'Unknown')
                print(f"  - {label}")
        else:
            print("❌ No labels found in database")
        
        print()
        
        # Check 2: What relationship types exist?
        print("🔗 2. Checking Relationship Types...")
        rel_query = "CALL db.relationshipTypes()"
        rel_result = await client.execute_cypher(rel_query)
        
        if rel_result:
            print("Found relationship types:")
            for record in rel_result:
                rel_type = record.get('relationshipType', 'Unknown')
                print(f"  - {rel_type}")
        else:
            print("❌ No relationship types found in database")
        
        print()
        
        # Check 3: Look for CAE-specific nodes
        print("🏥 3. Checking for CAE-specific Data...")
        
        cae_labels = ['cae_Drug', 'cae_AdverseEvent', 'cae_SNOMEDConcept', 'cae_DosingAdjustment']
        
        for label in cae_labels:
            count_query = f"MATCH (n:{label}) RETURN count(n) as count"
            try:
                count_result = await client.execute_cypher(count_query)
                if count_result and len(count_result) > 0:
                    count = count_result[0].get('count', 0)
                    status = "✅" if count > 0 else "❌"
                    print(f"  {status} {label}: {count} nodes")
                else:
                    print(f"  ❌ {label}: 0 nodes")
            except Exception as e:
                print(f"  ❌ {label}: Label doesn't exist")
        
        print()
        
        # Check 4: Look for any drug-related data
        print("💊 4. Checking for Any Drug Data...")
        
        # Try different possible drug labels
        possible_drug_labels = ['Drug', 'Medication', 'RxNormConcept', 'cae_Drug']
        
        for label in possible_drug_labels:
            try:
                sample_query = f"MATCH (n:{label}) RETURN n LIMIT 3"
                sample_result = await client.execute_cypher(sample_query)
                if sample_result and len(sample_result) > 0:
                    print(f"  ✅ Found {label} nodes:")
                    for i, record in enumerate(sample_result[:3]):
                        node = record.get('n', {})
                        name = node.get('name', node.get('preferred_name', 'Unknown'))
                        print(f"    {i+1}. {name}")
                    break
            except:
                continue
        else:
            print("  ❌ No drug-related nodes found with common labels")
        
        print()
        
        # Check 5: Total node count
        print("📈 5. Database Statistics...")
        total_query = "MATCH (n) RETURN count(n) as total_nodes"
        total_result = await client.execute_cypher(total_query)
        
        if total_result and len(total_result) > 0:
            total = total_result[0].get('total_nodes', 0)
            print(f"  Total nodes in database: {total:,}")
        
        total_rel_query = "MATCH ()-[r]->() RETURN count(r) as total_relationships"
        total_rel_result = await client.execute_cypher(total_rel_query)
        
        if total_rel_result and len(total_rel_result) > 0:
            total_rels = total_rel_result[0].get('total_relationships', 0)
            print(f"  Total relationships: {total_rels:,}")
        
        print()
        
        # Check 6: Sample some actual data
        print("🔬 6. Sample Data...")
        sample_query = "MATCH (n) RETURN labels(n) as labels, keys(n) as properties LIMIT 5"
        sample_result = await client.execute_cypher(sample_query)
        
        if sample_result:
            print("Sample nodes:")
            for i, record in enumerate(sample_result):
                labels = record.get('labels', [])
                props = record.get('properties', [])
                print(f"  {i+1}. Labels: {labels}, Properties: {props}")
        
    except Exception as e:
        print(f"❌ Error checking Neo4j data: {e}")
    
    finally:
        await client.disconnect()
    
    print("\n" + "=" * 50)
    print("🎯 DIAGNOSIS:")
    print("If you see mostly empty results above, it means:")
    print("1. Your Neo4j database doesn't have CAE-specific clinical data")
    print("2. The data might be in a different format/schema")
    print("3. You need to run the knowledge pipeline to populate clinical data")
    print("\n💡 SOLUTION:")
    print("Run the knowledge pipeline service to ingest clinical data into Neo4j")

if __name__ == "__main__":
    asyncio.run(check_neo4j_data())
