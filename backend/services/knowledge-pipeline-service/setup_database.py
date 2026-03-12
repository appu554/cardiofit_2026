#!/usr/bin/env python3
"""
Database Setup Script for Clinical Knowledge Graph
Choose between GraphDB (local) or Neo4j Cloud (managed)
"""

import asyncio
import os
from pathlib import Path

def print_banner():
    print("=" * 70)
    print("🏥 CLINICAL KNOWLEDGE GRAPH - DATABASE SETUP")
    print("=" * 70)
    print()

def print_database_options():
    print("📊 Choose your database platform:")
    print()
    print("1. 🌐 Neo4j Cloud (AuraDB) - RECOMMENDED")
    print("   ✅ Fully managed cloud service")
    print("   ✅ Auto-scaling and high availability") 
    print("   ✅ Enterprise security and backup")
    print("   ✅ No local setup required")
    print("   ✅ Free tier available (200k nodes)")
    print("   ✅ Optimized for graph queries")
    print()
    print("2. 🗄️ GraphDB (Local)")
    print("   ✅ RDF/SPARQL support")
    print("   ✅ Semantic reasoning capabilities")
    print("   ✅ Local control and privacy")
    print("   ⚠️  Requires local installation")
    print("   ⚠️  Manual scaling and maintenance")
    print()

def get_user_choice():
    while True:
        choice = input("Enter your choice (1 for Neo4j Cloud, 2 for GraphDB): ").strip()
        if choice in ['1', '2']:
            return choice
        print("❌ Invalid choice. Please enter 1 or 2.")

def setup_neo4j_cloud():
    print("\n🌐 Setting up Neo4j Cloud (AuraDB)...")
    print()
    print("📋 Follow these steps:")
    print("1. Visit: https://neo4j.com/cloud/aura/")
    print("2. Create a free account")
    print("3. Create a new database instance")
    print("4. Note your connection details")
    print()
    
    # Get connection details
    print("🔧 Enter your Neo4j Cloud connection details:")
    
    uri = input("Neo4j URI (e.g., neo4j+s://xxxxx.databases.neo4j.io): ").strip()
    if not uri:
        uri = "neo4j+s://your-instance.databases.neo4j.io"
    
    username = input("Username [neo4j]: ").strip()
    if not username:
        username = "neo4j"
    
    password = input("Password: ").strip()
    if not password:
        print("⚠️  Password is required for Neo4j Cloud")
        password = "your-password-here"
    
    database = input("Database name [neo4j]: ").strip()
    if not database:
        database = "neo4j"
    
    return {
        'DATABASE_TYPE': 'neo4j',
        'NEO4J_URI': uri,
        'NEO4J_USERNAME': username,
        'NEO4J_PASSWORD': password,
        'NEO4J_DATABASE': database
    }

def setup_graphdb():
    print("\n🗄️ Setting up GraphDB (Local)...")
    print()
    print("📋 Prerequisites:")
    print("1. GraphDB should be running on localhost:7200")
    print("2. Repository 'cae-clinical-intelligence' should exist")
    print("3. GraphDB should be accessible without authentication")
    print()
    
    endpoint = input("GraphDB endpoint [http://localhost:7200]: ").strip()
    if not endpoint:
        endpoint = "http://localhost:7200"
    
    repository = input("Repository name [cae-clinical-intelligence]: ").strip()
    if not repository:
        repository = "cae-clinical-intelligence"
    
    username = input("Username (leave empty if no auth): ").strip()
    password = input("Password (leave empty if no auth): ").strip()
    
    return {
        'DATABASE_TYPE': 'graphdb',
        'GRAPHDB_ENDPOINT': endpoint,
        'GRAPHDB_REPOSITORY': repository,
        'GRAPHDB_USERNAME': username,
        'GRAPHDB_PASSWORD': password
    }

def create_env_file(config):
    """Create .env file with database configuration"""
    env_file = Path(".env")
    
    print(f"\n📝 Creating configuration file: {env_file}")
    
    env_content = "# Clinical Knowledge Graph Database Configuration\n"
    env_content += f"# Generated on {datetime.now().isoformat()}\n\n"
    
    for key, value in config.items():
        env_content += f"{key}={value}\n"
    
    with open(env_file, 'w') as f:
        f.write(env_content)
    
    print(f"✅ Configuration saved to {env_file}")

def update_config_file(config):
    """Update the config.py file with new defaults"""
    config_file = Path("src/core/config.py")
    
    if not config_file.exists():
        print(f"⚠️  Config file not found: {config_file}")
        return
    
    print(f"📝 Updating configuration defaults in {config_file}")
    
    # Read current config
    with open(config_file, 'r') as f:
        content = f.read()
    
    # Update DATABASE_TYPE default
    if 'DATABASE_TYPE' in config:
        content = content.replace(
            'DATABASE_TYPE: str = Field(\n        default="neo4j"',
            f'DATABASE_TYPE: str = Field(\n        default="{config["DATABASE_TYPE"]}"'
        )
    
    # Update Neo4j defaults if applicable
    if config.get('DATABASE_TYPE') == 'neo4j':
        if 'NEO4J_URI' in config:
            content = content.replace(
                'NEO4J_URI: str = Field(\n        default="neo4j+s://your-instance.databases.neo4j.io"',
                f'NEO4J_URI: str = Field(\n        default="{config["NEO4J_URI"]}"'
            )
    
    # Write updated config
    with open(config_file, 'w') as f:
        f.write(content)
    
    print("✅ Configuration defaults updated")

async def test_connection(config):
    """Test database connection"""
    print("\n🧪 Testing database connection...")
    
    # Set environment variables temporarily
    for key, value in config.items():
        os.environ[key] = value
    
    try:
        from core.database_factory import validate_database_connection
        
        result = await validate_database_connection()
        
        if result["status"] == "connected":
            print("✅ Database connection successful!")
            
            db_info = result.get("database_info", {})
            print(f"📊 Database Type: {db_info.get('type', 'Unknown')}")
            
            if config['DATABASE_TYPE'] == 'neo4j':
                print(f"🌐 Neo4j URI: {db_info.get('uri', 'Unknown')}")
                print(f"💾 Database: {db_info.get('database', 'Unknown')}")
            else:
                print(f"🗄️ GraphDB Endpoint: {db_info.get('endpoint', 'Unknown')}")
                print(f"📚 Repository: {db_info.get('repository', 'Unknown')}")
            
            stats = result.get("stats", {})
            if stats:
                print(f"📈 Database Stats: {stats}")
            
            return True
        else:
            print("❌ Database connection failed!")
            print(f"Error: {result.get('error', 'Unknown error')}")
            return False
    
    except Exception as e:
        print(f"❌ Connection test failed: {e}")
        return False

def print_next_steps(config):
    """Print next steps after setup"""
    print("\n🎯 Next Steps:")
    print()
    
    if config['DATABASE_TYPE'] == 'neo4j':
        print("1. 📖 Read the Neo4j Cloud setup guide:")
        print("   cat NEO4J_CLOUD_SETUP.md")
        print()
        print("2. 🧪 Install Neo4j driver:")
        print("   pip install neo4j")
        print()
    else:
        print("1. 🗄️ Ensure GraphDB is running:")
        print("   http://localhost:7200")
        print()
        print("2. 📚 Verify repository exists:")
        print("   cae-clinical-intelligence")
        print()
    
    print("3. ✅ Validate all data sources:")
    print("   python validate_data_sources.py")
    print()
    print("4. 🚀 Run the knowledge pipeline:")
    print("   python start_pipeline.py --sources rxnorm snomed loinc")
    print()
    print("5. 🔍 Monitor progress and explore your clinical knowledge graph!")

async def main():
    from datetime import datetime
    
    print_banner()
    print_database_options()
    
    choice = get_user_choice()
    
    if choice == '1':
        config = setup_neo4j_cloud()
    else:
        config = setup_graphdb()
    
    # Create configuration files
    create_env_file(config)
    update_config_file(config)
    
    # Test connection
    connection_ok = await test_connection(config)
    
    if connection_ok:
        print("\n🎉 Database setup completed successfully!")
        print_next_steps(config)
    else:
        print("\n⚠️  Database setup completed but connection test failed.")
        print("Please check your configuration and try again.")
        print()
        print("💡 Troubleshooting:")
        if config['DATABASE_TYPE'] == 'neo4j':
            print("- Verify your Neo4j Cloud instance is running")
            print("- Check your connection URI and credentials")
            print("- Ensure your IP is whitelisted in Neo4j Cloud")
        else:
            print("- Verify GraphDB is running on localhost:7200")
            print("- Check that the repository exists")
            print("- Ensure GraphDB is accessible")

if __name__ == "__main__":
    asyncio.run(main())
