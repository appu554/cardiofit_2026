#!/usr/bin/env python3
"""
Test Neo4j Cloud connection for Clinical Knowledge Graph
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

async def test_neo4j_connection():
    """Test Neo4j Cloud connection"""
    print("🌐 Testing Neo4j Cloud Connection...")
    print("=" * 50)
    
    try:
        from core.database_factory import validate_database_connection
        
        result = await validate_database_connection()
        
        print(f"📊 Connection Status: {result['status']}")
        
        if result["status"] == "connected":
            print("✅ Neo4j Cloud connection successful!")
            
            db_info = result.get("database_info", {})
            print(f"🌐 Database Type: {db_info.get('type', 'Unknown')}")
            print(f"🔗 URI: {db_info.get('uri', 'Unknown')}")
            print(f"💾 Database: {db_info.get('database', 'Unknown')}")
            print(f"👤 Username: {db_info.get('username', 'Unknown')}")
            
            stats = result.get("stats", {})
            if stats and isinstance(stats, dict):
                print("\n📈 Database Statistics:")
                for key, value in stats.items():
                    if key != "database_info":
                        print(f"   {key}: {value}")
            
            print("\n🎉 Your Neo4j Cloud instance is ready for the knowledge pipeline!")
            return True
            
        else:
            print("❌ Neo4j Cloud connection failed!")
            error = result.get("error", "Unknown error")
            print(f"Error: {error}")
            
            print("\n🔧 Troubleshooting:")
            print("1. Check your Neo4j Cloud instance is running")
            print("2. Verify your connection URI and credentials")
            print("3. Ensure your IP is whitelisted in Neo4j Cloud")
            return False
    
    except Exception as e:
        print(f"❌ Connection test failed: {e}")
        print("\n🔧 Troubleshooting:")
        print("1. Make sure neo4j driver is installed: pip install neo4j")
        print("2. Check your .env file configuration")
        print("3. Verify your Neo4j Cloud credentials")
        return False

async def test_basic_operations():
    """Test basic Neo4j operations"""
    print("\n🧪 Testing Basic Neo4j Operations...")
    print("=" * 50)
    
    try:
        from core.neo4j_client import Neo4jCloudClient
        
        client = Neo4jCloudClient()
        
        if await client.connect():
            print("✅ Connected to Neo4j Cloud")
            
            # Test basic query
            result = await client.execute_cypher("RETURN 'Hello Neo4j!' as message")
            if result:
                print(f"✅ Basic query successful: {result[0]['message']}")
            
            # Test node creation
            await client.execute_cypher("""
                MERGE (test:TestNode {id: 'pipeline-test', name: 'Knowledge Pipeline Test'})
                RETURN test.name as name
            """)
            print("✅ Test node created successfully")
            
            # Test node retrieval
            result = await client.execute_cypher("""
                MATCH (test:TestNode {id: 'pipeline-test'})
                RETURN test.name as name
            """)
            if result:
                print(f"✅ Test node retrieved: {result[0]['name']}")
            
            # Clean up test node
            await client.execute_cypher("""
                MATCH (test:TestNode {id: 'pipeline-test'})
                DELETE test
            """)
            print("✅ Test node cleaned up")
            
            await client.disconnect()
            print("✅ All basic operations successful!")
            return True
        
        else:
            print("❌ Failed to connect to Neo4j Cloud")
            return False
    
    except Exception as e:
        print(f"❌ Basic operations test failed: {e}")
        return False

async def main():
    """Main test function"""
    print("🏥 CLINICAL KNOWLEDGE GRAPH - NEO4J CLOUD TEST")
    print("=" * 60)
    
    # Test connection
    connection_ok = await test_neo4j_connection()
    
    if connection_ok:
        # Test basic operations
        operations_ok = await test_basic_operations()
        
        if operations_ok:
            print("\n" + "=" * 60)
            print("🎉 ALL TESTS PASSED!")
            print("🚀 Your Neo4j Cloud is ready for the knowledge pipeline!")
            print("\nNext steps:")
            print("1. Run: python start_pipeline.py --sources rxnorm snomed loinc")
            print("2. Monitor progress as clinical data is loaded")
            print("3. Explore your knowledge graph in Neo4j Browser")
            print("=" * 60)
        else:
            print("\n❌ Basic operations test failed")
    else:
        print("\n❌ Connection test failed")
        print("Please check your configuration and try again")

if __name__ == "__main__":
    asyncio.run(main())
