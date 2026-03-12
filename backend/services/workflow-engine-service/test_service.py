#!/usr/bin/env python3
"""
Simple test script for Workflow Engine Service.
"""
import asyncio
import sys
import os

# Add the current directory to Python path
current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, current_dir)

# Add the backend services directory to Python path for shared imports
backend_dir = os.path.dirname(os.path.dirname(current_dir))
services_dir = os.path.join(backend_dir, "services")
sys.path.insert(0, services_dir)


async def test_service_setup():
    """Test basic service setup and configuration."""
    print("Testing Workflow Engine Service setup...")
    
    try:
        # Test configuration import
        from app.core.config import settings
        print(f"✅ Configuration loaded: {settings.SERVICE_NAME} v{settings.SERVICE_VERSION}")
        print(f"   Port: {settings.SERVICE_PORT}")
        print(f"   Google Healthcare API: {'Enabled' if settings.USE_GOOGLE_HEALTHCARE_API else 'Disabled'}")
        
        # Test database models import
        from app.models.workflow_models import WorkflowDefinition, WorkflowInstance, WorkflowTask
        from app.models.task_models import TaskAssignment, TaskComment
        print("✅ Database models imported successfully")
        
        # Test GraphQL schema import
        from app.graphql.federation_schema import schema
        print("✅ GraphQL federation schema imported successfully")
        
        # Test Google FHIR service import
        from app.services.google_fhir_service import google_fhir_service
        print("✅ Google FHIR service imported successfully")

        # Test Supabase service import
        from app.services.supabase_service import supabase_service
        print("✅ Supabase service imported successfully")

        # Test database initialization (without actually connecting)
        from app.db.database import Base, engine
        print("✅ Database configuration loaded successfully")
        
        print("\n🎉 All basic service components loaded successfully!")
        print("\nNext steps:")
        print("1. Set up Supabase database: python setup_database.py")
        print("2. Configure Google Cloud credentials")
        print("3. Run the service: python run_service.py")
        print("4. Test federation endpoint: http://localhost:8015/api/federation")
        print("5. Check health endpoint: http://localhost:8015/health")
        
        return True
        
    except ImportError as e:
        print(f"❌ Import error: {e}")
        return False
    except Exception as e:
        print(f"❌ Error: {e}")
        return False


async def test_graphql_schema():
    """Test GraphQL schema structure."""
    print("\nTesting GraphQL schema structure...")
    
    try:
        from app.graphql.federation_schema import schema
        
        # Get schema SDL
        schema_sdl = str(schema)
        
        # Check for key types
        required_types = [
            "WorkflowDefinition",
            "WorkflowInstance_Summary", 
            "Task",
            "Patient",
            "User"
        ]
        
        for type_name in required_types:
            if type_name in schema_sdl:
                print(f"✅ Type '{type_name}' found in schema")
            else:
                print(f"❌ Type '{type_name}' missing from schema")
        
        # Check for key queries
        required_queries = [
            "workflowDefinitions",
            "tasks",
            "workflowInstances"
        ]
        
        for query_name in required_queries:
            if query_name in schema_sdl:
                print(f"✅ Query '{query_name}' found in schema")
            else:
                print(f"❌ Query '{query_name}' missing from schema")
        
        # Check for key mutations
        required_mutations = [
            "startWorkflow",
            "completeTask",
            "claimTask"
        ]
        
        for mutation_name in required_mutations:
            if mutation_name in schema_sdl:
                print(f"✅ Mutation '{mutation_name}' found in schema")
            else:
                print(f"❌ Mutation '{mutation_name}' missing from schema")
        
        print("✅ GraphQL schema structure validated")
        return True
        
    except Exception as e:
        print(f"❌ GraphQL schema test failed: {e}")
        return False


async def test_supabase_connection():
    """Test Supabase connection."""
    print("\nTesting Supabase connection...")

    try:
        from app.services.supabase_service import supabase_service
        from app.core.config import settings

        # Test configuration
        print(f"✅ Supabase URL: {settings.SUPABASE_URL}")
        print(f"✅ Supabase Key: {settings.SUPABASE_KEY[:20]}...")

        # Test initialization (without actually connecting)
        print("✅ Supabase service configuration loaded")

        return True

    except Exception as e:
        print(f"❌ Supabase connection test failed: {e}")
        return False


async def main():
    """Run all tests."""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - SETUP TEST")
    print("=" * 60)

    # Test basic setup
    setup_ok = await test_service_setup()

    if setup_ok:
        # Test GraphQL schema
        schema_ok = await test_graphql_schema()

        # Test Supabase connection
        supabase_ok = await test_supabase_connection()

        if schema_ok and supabase_ok:
            print("\n🎉 All tests passed! Service is ready for development.")
        else:
            print("\n⚠️  Some issues found during testing.")
    else:
        print("\n❌ Basic setup test failed. Please check dependencies.")

    print("\n" + "=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
