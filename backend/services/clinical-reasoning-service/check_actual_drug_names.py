"""
Check Actual Drug Names in Neo4j

This script checks what drug names actually exist in your Neo4j database
and finds the correct names to use in CAE queries.
"""

import asyncio
from dotenv import load_dotenv
load_dotenv()

import sys
from pathlib import Path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

from app.knowledge.neo4j_client import Neo4jCloudClient

async def check_actual_drug_names():
    """Check what drug names actually exist in Neo4j"""
    print("🔍 Checking Actual Drug Names in Neo4j")
    print("=" * 50)
    
    client = Neo4jCloudClient()
    
    try:
        await client.connect()
        print("✅ Connected to Neo4j")
        print()
        
        # Check 1: Sample drug names
        print("💊 1. Sample Drug Names (first 20):")
        sample_query = "MATCH (d:cae_Drug) RETURN d.name as name LIMIT 20"
        sample_result = await client.execute_cypher(sample_query)
        
        if sample_result:
            for i, record in enumerate(sample_result, 1):
                name = record.get('name', 'Unknown')
                print(f"  {i:2d}. {name}")
        
        print()
        
        # Check 2: Look for specific drugs we're testing
        test_drugs = ['warfarin', 'ciprofloxacin', 'penicillin', 'digoxin', 'metformin']
        
        print("🎯 2. Looking for Test Drugs:")
        for drug in test_drugs:
            # Exact match
            exact_query = "MATCH (d:cae_Drug) WHERE toLower(d.name) = toLower($drug_name) RETURN d.name as name"
            exact_result = await client.execute_cypher(exact_query, {'drug_name': drug})
            
            if exact_result:
                actual_name = exact_result[0].get('name', 'Unknown')
                print(f"  ✅ {drug} -> Found as: {actual_name}")
            else:
                # Partial match
                partial_query = "MATCH (d:cae_Drug) WHERE toLower(d.name) CONTAINS toLower($drug_name) RETURN d.name as name LIMIT 5"
                partial_result = await client.execute_cypher(partial_query, {'drug_name': drug})
                
                if partial_result:
                    print(f"  🔍 {drug} -> Similar names found:")
                    for record in partial_result:
                        similar_name = record.get('name', 'Unknown')
                        print(f"      - {similar_name}")
                else:
                    print(f"  ❌ {drug} -> Not found")
        
        print()
        
        # Check 3: Drug interaction relationships
        print("🔗 3. Checking Drug Interactions:")
        interaction_query = """
        MATCH (d1:cae_Drug)-[r:cae_interactsWith]->(d2:cae_Drug)
        RETURN d1.name as drug1, d2.name as drug2, type(r) as relationship
        LIMIT 10
        """
        interaction_result = await client.execute_cypher(interaction_query)
        
        if interaction_result:
            print("  Found interactions:")
            for record in interaction_result:
                drug1 = record.get('drug1', 'Unknown')
                drug2 = record.get('drug2', 'Unknown')
                rel = record.get('relationship', 'Unknown')
                print(f"    {drug1} --{rel}--> {drug2}")
        else:
            print("  ❌ No drug interactions found")
        
        print()
        
        # Check 4: Adverse event relationships
        print("🚨 4. Checking Adverse Events:")
        ae_query = """
        MATCH (d:cae_Drug)-[r:cae_hasAdverseEvent]->(ae:cae_AdverseEvent)
        RETURN d.name as drug_name, ae.reaction as reaction, keys(ae) as ae_properties
        LIMIT 10
        """
        ae_result = await client.execute_cypher(ae_query)
        
        if ae_result:
            print("  Found adverse events:")
            for record in ae_result:
                drug_name = record.get('drug_name', 'Unknown')
                reaction = record.get('reaction', 'Unknown')
                properties = record.get('ae_properties', [])
                print(f"    {drug_name} -> {reaction}")
                print(f"      Properties: {properties}")
        else:
            print("  ❌ No adverse events found")
        
        print()
        
        # Check 5: What properties do AdverseEvent nodes have?
        print("📋 5. AdverseEvent Properties:")
        ae_props_query = """
        MATCH (ae:cae_AdverseEvent)
        RETURN keys(ae) as properties, ae
        LIMIT 5
        """
        ae_props_result = await client.execute_cypher(ae_props_query)
        
        if ae_props_result:
            for i, record in enumerate(ae_props_result, 1):
                properties = record.get('properties', [])
                ae_node = record.get('ae', {})
                print(f"  {i}. Properties: {properties}")
                
                # Show sample values
                for prop in properties[:5]:  # Show first 5 properties
                    value = ae_node.get(prop, 'N/A')
                    print(f"     {prop}: {value}")
                print()
        
    except Exception as e:
        print(f"❌ Error: {e}")
    
    finally:
        await client.disconnect()

if __name__ == "__main__":
    asyncio.run(check_actual_drug_names())
