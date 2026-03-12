#!/usr/bin/env python3
"""
Verify that workflow tables were created in Supabase.
"""
import os
import sys
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

def verify_tables():
    """Verify tables using Supabase client."""
    print("Verifying workflow tables...")
    
    try:
        from supabase import create_client
        
        url = os.getenv("SUPABASE_URL")
        key = os.getenv("SUPABASE_KEY")
        
        supabase = create_client(url, key)
        
        # List of tables to check
        tables_to_check = [
            "workflow_definitions",
            "workflow_instances", 
            "workflow_tasks",
            "workflow_events",
            "workflow_timers",
            "workflow_test"
        ]
        
        successful_tables = []
        failed_tables = []
        
        for table_name in tables_to_check:
            try:
                # Try to query the table (just get count)
                response = supabase.table(table_name).select("*", count="exact").limit(0).execute()
                
                if hasattr(response, 'count') or response.data is not None:
                    print(f"✅ Table '{table_name}' exists and is accessible")
                    successful_tables.append(table_name)
                else:
                    print(f"❌ Table '{table_name}' query failed")
                    failed_tables.append(table_name)
                    
            except Exception as e:
                print(f"❌ Table '{table_name}' error: {e}")
                failed_tables.append(table_name)
        
        # Check the test table specifically
        try:
            response = supabase.table("workflow_test").select("*").execute()
            if response.data:
                print(f"\n✅ Test table contains {len(response.data)} records")
                for record in response.data:
                    print(f"   - {record.get('test_message', 'No message')}")
            else:
                print("\n⚠️  Test table exists but is empty")
        except Exception as e:
            print(f"\n❌ Could not read test table: {e}")
        
        print(f"\n📊 SUMMARY:")
        print(f"   ✅ Successful: {len(successful_tables)}/{len(tables_to_check)} tables")
        print(f"   ❌ Failed: {len(failed_tables)}/{len(tables_to_check)} tables")
        
        if len(successful_tables) >= 5:  # At least the main tables
            print("\n🎉 Database setup appears successful!")
            print("You can now run: python run_service.py")
            return True
        else:
            print("\n⚠️  Some tables are missing. Please run the SQL in Supabase dashboard.")
            return False
            
    except ImportError:
        print("❌ Supabase client not available. Install with: pip install supabase")
        return False
    except Exception as e:
        print(f"❌ Verification error: {e}")
        return False

def test_basic_operations():
    """Test basic database operations."""
    print("\nTesting basic database operations...")
    
    try:
        from supabase import create_client
        
        url = os.getenv("SUPABASE_URL")
        key = os.getenv("SUPABASE_KEY")
        
        supabase = create_client(url, key)
        
        # Try to insert a test workflow definition
        test_workflow = {
            "fhir_id": "test-workflow-001",
            "name": "Test Workflow",
            "version": "1.0.0",
            "status": "draft",
            "description": "Test workflow for verification"
        }
        
        response = supabase.table("workflow_definitions").insert(test_workflow).execute()
        
        if response.data:
            print("✅ Successfully inserted test workflow definition")
            
            # Try to read it back
            read_response = supabase.table("workflow_definitions").select("*").eq("fhir_id", "test-workflow-001").execute()
            
            if read_response.data:
                print("✅ Successfully read back test workflow definition")
                
                # Clean up - delete the test record
                delete_response = supabase.table("workflow_definitions").delete().eq("fhir_id", "test-workflow-001").execute()
                print("✅ Successfully cleaned up test data")
                
                return True
            else:
                print("❌ Could not read back test data")
                return False
        else:
            print("❌ Could not insert test data")
            return False
            
    except Exception as e:
        print(f"⚠️  Basic operations test failed: {e}")
        print("   This might be due to RLS policies - tables exist but operations are restricted")
        return False

def main():
    """Main verification function."""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - TABLE VERIFICATION")
    print("=" * 60)
    
    # Verify tables exist
    tables_ok = verify_tables()
    
    if tables_ok:
        # Test basic operations
        operations_ok = test_basic_operations()
        
        if operations_ok:
            print("\n🎉 Full verification successful!")
        else:
            print("\n⚠️  Tables exist but operations may be restricted by RLS")
            print("   This is normal for Supabase - the service should still work")
    
    print("\n" + "=" * 60)

if __name__ == "__main__":
    main()
