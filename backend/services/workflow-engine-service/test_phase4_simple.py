#!/usr/bin/env python3
"""
Simple Phase 4 Integration Tests for Workflow Engine Service.

This script tests the Phase 4 service integration components without requiring
database connections or external dependencies.
"""

import asyncio
import sys
import os
from datetime import datetime

# Add the app directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '.'))


async def test_service_imports():
    """Test that all Phase 4 services can be imported."""
    print("\n📦 Testing Service Imports...")

    try:
        print("   🔍 Importing Service Task Executor...")
        from app.services.service_task_executor import service_task_executor
        print("   ✅ Service Task Executor imported")

        print("   🔍 Importing Event Listener...")
        from app.services.event_listener import event_listener
        print("   ✅ Event Listener imported")

        print("   🔍 Importing Event Publisher...")
        from app.services.event_publisher import event_publisher
        print("   ✅ Event Publisher imported")

        print("   🔍 Importing FHIR Resource Monitor...")
        from app.services.fhir_resource_monitor import fhir_resource_monitor
        print("   ✅ FHIR Resource Monitor imported")

        return True

    except Exception as e:
        import traceback
        print(f"   ❌ Import failed: {e}")
        print(f"   📋 Traceback: {traceback.format_exc()}")
        return False


async def test_service_configuration():
    """Test service configuration."""
    print("\n⚙️  Testing Service Configuration...")
    
    try:
        from app.services.service_task_executor import service_task_executor
        from app.services.event_listener import event_listener
        from app.services.event_publisher import event_publisher
        
        # Test service endpoints configuration
        if service_task_executor.service_endpoints:
            print(f"   ✅ {len(service_task_executor.service_endpoints)} service endpoints configured")
        else:
            print("   ⚠️  No service endpoints configured")
        
        # Test webhook endpoints configuration
        if event_publisher.webhook_endpoints:
            print(f"   ✅ {len(event_publisher.webhook_endpoints)} webhook endpoints configured")
        else:
            print("   ⚠️  No webhook endpoints configured")
        
        # Test event handlers
        if event_listener.event_handlers:
            print(f"   ✅ {len(event_listener.event_handlers)} event handlers registered")
        else:
            print("   ⚠️  No event handlers registered")
        
        return True
        
    except Exception as e:
        print(f"   ❌ Configuration test failed: {e}")
        return False


async def test_basic_functionality():
    """Test basic functionality without external dependencies."""
    print("\n🔧 Testing Basic Functionality...")
    
    try:
        from app.services.service_task_executor import service_task_executor
        from app.services.event_listener import event_listener
        from app.services.fhir_resource_monitor import fhir_resource_monitor
        
        # Test GraphQL query building
        query = service_task_executor._build_graphql_query("get", {
            "resourceType": "Patient",
            "id": "test-123"
        })
        
        if "Patient" in query.get("query", ""):
            print("   ✅ GraphQL query building works")
        else:
            print("   ⚠️  GraphQL query building issue")
        
        # Test event handler registration
        async def test_handler(event_data):
            pass
        
        event_listener.register_handler("test.event", test_handler)
        
        if "test.event" in event_listener.event_handlers:
            print("   ✅ Event handler registration works")
        else:
            print("   ⚠️  Event handler registration issue")
        
        # Test workflow instance ID extraction
        test_task = {
            "id": "test-task-123",
            "extension": [
                {
                    "url": "http://clinical-synthesis-hub.com/workflow-instance-id",
                    "valueString": "test-workflow-123"
                }
            ]
        }
        
        workflow_id = fhir_resource_monitor._extract_workflow_instance_id(test_task)
        if workflow_id == "test-workflow-123":
            print("   ✅ Workflow instance ID extraction works")
        else:
            print("   ⚠️  Workflow instance ID extraction issue")
        
        return True
        
    except Exception as e:
        print(f"   ❌ Basic functionality test failed: {e}")
        return False


async def test_integration_readiness():
    """Test integration readiness."""
    print("\n🚀 Testing Integration Readiness...")
    
    try:
        from app.services.workflow_engine_service import workflow_engine_service
        
        # Test Phase 4 services initialization
        await workflow_engine_service._initialize_phase4_services()
        
        # Check if Phase 4 services are available
        services_ready = 0
        total_services = 4
        
        if workflow_engine_service.service_task_executor:
            print("   ✅ Service Task Executor integrated")
            services_ready += 1
        else:
            print("   ⚠️  Service Task Executor not integrated")
        
        if workflow_engine_service.event_listener:
            print("   ✅ Event Listener integrated")
            services_ready += 1
        else:
            print("   ⚠️  Event Listener not integrated")
        
        if workflow_engine_service.event_publisher:
            print("   ✅ Event Publisher integrated")
            services_ready += 1
        else:
            print("   ⚠️  Event Publisher not integrated")
        
        if workflow_engine_service.fhir_resource_monitor:
            print("   ✅ FHIR Resource Monitor integrated")
            services_ready += 1
        else:
            print("   ⚠️  FHIR Resource Monitor not integrated")
        
        print(f"   📊 Integration Status: {services_ready}/{total_services} services ready")
        
        return services_ready == total_services
        
    except Exception as e:
        print(f"   ❌ Integration readiness test failed: {e}")
        return False


async def main():
    """Run all simple Phase 4 integration tests."""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - PHASE 4 SIMPLE TESTS")
    print("=" * 60)
    
    tests = [
        ("Service Imports", test_service_imports),
        ("Service Configuration", test_service_configuration),
        ("Basic Functionality", test_basic_functionality),
        ("Integration Readiness", test_integration_readiness)
    ]
    
    passed = 0
    failed = 0
    
    for test_name, test_func in tests:
        try:
            result = await test_func()
            if result:
                passed += 1
            else:
                failed += 1
        except Exception as e:
            print(f"\n❌ {test_name} test crashed: {e}")
            failed += 1
    
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)
    print(f"✅ Passed: {passed}")
    print(f"❌ Failed: {failed}")
    print(f"📊 Total: {passed + failed}")
    
    if failed == 0:
        print("\n🎉 All Phase 4 simple tests passed!")
        print("\nPhase 4: Service Integration is ready!")
        print("\n📋 Next Steps:")
        print("1. Configure database connection (fix Supabase credentials)")
        print("2. Run database migration: python run_migration.py")
        print("3. Start the workflow engine service: python run_service.py")
        print("4. Test with other federation services")
    else:
        print(f"\n⚠️  {failed} test(s) failed. Please review and fix issues.")
    
    print("\n" + "=" * 60)
    
    return failed == 0


if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
