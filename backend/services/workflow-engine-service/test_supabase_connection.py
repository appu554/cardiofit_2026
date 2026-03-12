#!/usr/bin/env python3
"""
Test Supabase connection and table existence for Workflow Engine Service.
"""

import asyncio
import sys
import os

# Add the app directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '.'))

from app.services.supabase_service import supabase_service


async def test_supabase_connection():
    """Test Supabase connection and table existence."""
    print("🔍 Testing Supabase Connection...")
    
    # Test initialization
    print("\n1. Testing Supabase initialization...")
    success = await supabase_service.initialize()
    if success:
        print("   ✅ Supabase client initialized successfully")
    else:
        print("   ❌ Supabase client initialization failed")
        return False
    
    # Test basic connection
    print("\n2. Testing basic connection...")
    try:
        # Try a simple query to test connection
        response = supabase_service.client.table("workflow_instances").select("*").limit(1).execute()
        print("   ✅ Basic connection test passed")
    except Exception as e:
        print(f"   ❌ Basic connection test failed: {e}")
        return False
    
    # Test required tables existence
    print("\n3. Testing required tables...")
    required_tables = [
        "service_task_logs",
        "event_store", 
        "event_processing_logs",
        "workflow_events_log"
    ]
    
    for table_name in required_tables:
        try:
            response = supabase_service.client.table(table_name).select("*").limit(1).execute()
            print(f"   ✅ Table '{table_name}' exists and accessible")
        except Exception as e:
            print(f"   ❌ Table '{table_name}' error: {e}")
            print(f"      This table may not exist or may have permission issues")
    
    # Test insert operations
    print("\n4. Testing insert operations...")
    
    # Test service_task_logs insert
    try:
        test_log = {
            "service_name": "test-service",
            "operation": "test-operation", 
            "parameters": '{"test": "data"}',
            "result": '{"success": true}',
            "status": "success",
            "executed_at": "2024-01-01T00:00:00Z",
            "source": "test"
        }
        response = supabase_service.client.table("service_task_logs").insert(test_log).execute()
        if response.data:
            print("   ✅ service_task_logs insert test passed")
            # Clean up
            supabase_service.client.table("service_task_logs").delete().eq("source", "test").execute()
        else:
            print(f"   ❌ service_task_logs insert test failed: {response}")
    except Exception as e:
        print(f"   ❌ service_task_logs insert test failed: {e}")
    
    # Test event_store insert
    try:
        test_event = {
            "event_type": "test.event",
            "event_data": '{"test": "data"}',
            "source": "test",
            "created_at": "2024-01-01T00:00:00Z"
        }
        response = supabase_service.client.table("event_store").insert(test_event).execute()
        if response.data:
            print("   ✅ event_store insert test passed")
            # Clean up
            supabase_service.client.table("event_store").delete().eq("source", "test").execute()
        else:
            print(f"   ❌ event_store insert test failed: {response}")
    except Exception as e:
        print(f"   ❌ event_store insert test failed: {e}")
    
    # Test event_processing_logs insert
    try:
        test_processing_log = {
            "event_type": "test.event",
            "status": "processed",
            "processed_at": "2024-01-01T00:00:00Z",
            "source": "test"
        }
        response = supabase_service.client.table("event_processing_logs").insert(test_processing_log).execute()
        if response.data:
            print("   ✅ event_processing_logs insert test passed")
            # Clean up
            supabase_service.client.table("event_processing_logs").delete().eq("source", "test").execute()
        else:
            print(f"   ❌ event_processing_logs insert test failed: {response}")
    except Exception as e:
        print(f"   ❌ event_processing_logs insert test failed: {e}")
    
    print("\n🎉 Supabase connection test completed!")
    return True


async def main():
    """Run the Supabase connection test."""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - SUPABASE CONNECTION TEST")
    print("=" * 60)
    
    await test_supabase_connection()
    
    print("\n" + "=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
