#!/usr/bin/env python3
"""
Create required database tables for Workflow Engine Service in Supabase.
"""

import asyncio
import sys
import os

# Add the app directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '.'))

from app.services.supabase_service import supabase_service


async def create_workflow_tables():
    """Create the required workflow engine tables in Supabase."""
    print("🔧 Creating Workflow Engine Tables in Supabase...")
    
    # Initialize Supabase service
    success = await supabase_service.initialize()
    if not success:
        print("❌ Failed to initialize Supabase service")
        return False
    
    # Read the SQL migration file
    sql_file_path = os.path.join(os.path.dirname(__file__), 'migrations', 'create_workflow_tables.sql')
    
    try:
        with open(sql_file_path, 'r') as f:
            sql_content = f.read()
    except FileNotFoundError:
        print(f"❌ SQL migration file not found: {sql_file_path}")
        return False
    
    # Split SQL content into individual statements
    sql_statements = [stmt.strip() for stmt in sql_content.split(';') if stmt.strip()]
    
    print(f"📝 Found {len(sql_statements)} SQL statements to execute")
    
    # Execute each SQL statement
    for i, statement in enumerate(sql_statements, 1):
        if not statement or statement.startswith('--'):
            continue
            
        try:
            print(f"   {i:2d}. Executing SQL statement...")
            
            # Use Supabase RPC to execute raw SQL
            response = supabase_service.client.rpc('exec_sql', {'sql': statement}).execute()
            
            if response.data is not None:
                print(f"      ✅ Statement {i} executed successfully")
            else:
                print(f"      ⚠️  Statement {i} executed with warnings: {response}")
                
        except Exception as e:
            # Some statements might fail if tables already exist, which is okay
            if "already exists" in str(e).lower() or "duplicate" in str(e).lower():
                print(f"      ℹ️  Statement {i} skipped (already exists)")
            else:
                print(f"      ❌ Statement {i} failed: {e}")
    
    print("\n🎉 Table creation process completed!")
    return True


async def verify_tables():
    """Verify that all required tables were created successfully."""
    print("\n🔍 Verifying table creation...")
    
    required_tables = [
        "service_task_logs",
        "event_store", 
        "event_processing_logs",
        "workflow_events_log"
    ]
    
    all_tables_exist = True
    
    for table_name in required_tables:
        try:
            response = supabase_service.client.table(table_name).select("*").limit(1).execute()
            print(f"   ✅ Table '{table_name}' exists and accessible")
        except Exception as e:
            print(f"   ❌ Table '{table_name}' error: {e}")
            all_tables_exist = False
    
    if all_tables_exist:
        print("\n🎉 All required tables exist and are accessible!")
    else:
        print("\n⚠️  Some tables are missing or inaccessible.")
    
    return all_tables_exist


async def main():
    """Main function to create and verify tables."""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - TABLE CREATION")
    print("=" * 60)
    
    # Note: Supabase doesn't support direct SQL execution via the Python client
    # The tables need to be created manually in the Supabase dashboard
    print("📋 MANUAL SETUP REQUIRED:")
    print("   1. Go to your Supabase dashboard")
    print("   2. Navigate to the SQL Editor")
    print("   3. Copy and paste the contents of:")
    print("      backend/services/workflow-engine-service/migrations/create_workflow_tables.sql")
    print("   4. Execute the SQL script")
    print("   5. Run this script again to verify the tables")
    
    print("\n🔍 Checking if tables already exist...")
    await verify_tables()
    
    print("\n" + "=" * 60)
    print("NEXT STEPS:")
    print("1. If tables don't exist, create them using the SQL script in Supabase dashboard")
    print("2. Run the Phase 4 integration tests again:")
    print("   python test_phase4_integration.py")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
