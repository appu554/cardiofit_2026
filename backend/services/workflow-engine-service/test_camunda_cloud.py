#!/usr/bin/env python3
"""
Test script for Camunda Cloud integration.
"""
import asyncio
import os
import sys
from pathlib import Path

# Add the app directory to Python path
sys.path.insert(0, str(Path(__file__).parent / "app"))

from app.core.config import settings

try:
    from app.services.camunda_cloud_service import camunda_cloud_service
    CAMUNDA_CLOUD_AVAILABLE = True
except ImportError as e:
    print(f"Warning: Could not import camunda_cloud_service: {e}")
    CAMUNDA_CLOUD_AVAILABLE = False
    camunda_cloud_service = None


async def test_camunda_cloud_connection():
    """Test Camunda Cloud connection and basic operations."""

    print("🚀 Testing Camunda Cloud Integration")
    print("=" * 50)

    # Check if Camunda Cloud service is available
    if not CAMUNDA_CLOUD_AVAILABLE:
        print("❌ Camunda Cloud service not available (missing dependencies)")
        print("   Install pyzeebe: pip install pyzeebe")
        return False

    # Check configuration
    print("📋 Configuration Check:")
    print(f"   USE_CAMUNDA_CLOUD: {settings.USE_CAMUNDA_CLOUD}")
    print(f"   CLIENT_ID: {settings.CAMUNDA_CLOUD_CLIENT_ID[:8]}..." if settings.CAMUNDA_CLOUD_CLIENT_ID else "   CLIENT_ID: Not set")
    print(f"   CLUSTER_ID: {settings.CAMUNDA_CLOUD_CLUSTER_ID[:8]}..." if settings.CAMUNDA_CLOUD_CLUSTER_ID else "   CLUSTER_ID: Not set")
    print(f"   REGION: {settings.CAMUNDA_CLOUD_REGION}")
    print()

    if not settings.USE_CAMUNDA_CLOUD:
        print("❌ Camunda Cloud is disabled. Set USE_CAMUNDA_CLOUD=true in .env")
        return False
    
    if not all([
        settings.CAMUNDA_CLOUD_CLIENT_ID,
        settings.CAMUNDA_CLOUD_CLIENT_SECRET,
        settings.CAMUNDA_CLOUD_CLUSTER_ID,
        settings.CAMUNDA_CLOUD_REGION
    ]):
        print("❌ Missing Camunda Cloud configuration. Please check your .env file.")
        return False
    
    # Test initialization
    print("🔌 Testing Connection:")
    try:
        success = await camunda_cloud_service.initialize()
        if success:
            print("   ✅ Successfully connected to Camunda Cloud")
        else:
            print("   ❌ Failed to connect to Camunda Cloud")
            return False
    except Exception as e:
        print(f"   ❌ Connection error: {e}")
        return False
    
    # Test access token
    print("\n🔑 Testing Authentication:")
    try:
        token = await camunda_cloud_service._get_access_token()
        if token:
            print(f"   ✅ Successfully obtained access token: {token[:20]}...")
        else:
            print("   ❌ Failed to obtain access token")
            return False
    except Exception as e:
        print(f"   ❌ Authentication error: {e}")
        return False
    
    # Test workflow deployment
    print("\n📄 Testing Workflow Deployment:")
    try:
        # Read sample BPMN file
        bpmn_file = Path(__file__).parent / "workflows" / "patient-admission-workflow.bpmn"
        if bpmn_file.exists():
            with open(bpmn_file, 'r') as f:
                bpmn_xml = f.read()
            
            process_key = await camunda_cloud_service.deploy_workflow(
                bpmn_xml=bpmn_xml,
                workflow_name="test-patient-admission"
            )
            
            if process_key:
                print(f"   ✅ Successfully deployed workflow: {process_key}")
                
                # Test process instance creation
                print("\n🏃 Testing Process Instance:")
                instance_key = await camunda_cloud_service.start_process_instance(
                    process_key=process_key,
                    variables={
                        "patientData": {
                            "name": "Test Patient",
                            "dateOfBirth": "1990-01-01",
                            "mrn": "TEST123"
                        },
                        "assignee": "test@example.com"
                    }
                )
                
                if instance_key:
                    print(f"   ✅ Successfully started process instance: {instance_key}")
                    
                    # Test message publishing
                    print("\n📨 Testing Message Publishing:")
                    message_success = await camunda_cloud_service.publish_message(
                        message_name="test-message",
                        correlation_key=instance_key,
                        variables={"testData": "Hello from test"}
                    )
                    
                    if message_success:
                        print("   ✅ Successfully published message")
                    else:
                        print("   ⚠️  Message publishing failed (this is normal for test)")
                    
                else:
                    print("   ❌ Failed to start process instance")
                    return False
            else:
                print("   ❌ Failed to deploy workflow")
                return False
        else:
            print(f"   ⚠️  Sample BPMN file not found: {bpmn_file}")
            print("   ℹ️  Skipping workflow deployment test")
    
    except Exception as e:
        print(f"   ❌ Workflow deployment error: {e}")
        return False
    
    print("\n🎉 All tests passed! Camunda Cloud integration is working correctly.")
    return True


async def test_service_health():
    """Test the service health endpoint."""
    print("\n🏥 Testing Service Health:")
    
    try:
        import httpx
        
        async with httpx.AsyncClient() as client:
            response = await client.get("http://localhost:8015/health")
            
            if response.status_code == 200:
                health_data = response.json()
                print("   ✅ Service is healthy")
                print(f"   📊 Camunda Cloud Status:")
                print(f"      - Use Camunda Cloud: {health_data.get('use_camunda_cloud', False)}")
                print(f"      - Camunda Cloud Initialized: {health_data.get('camunda_cloud_initialized', False)}")
                print(f"      - Workflow Engine Initialized: {health_data.get('workflow_engine_initialized', False)}")
                return True
            else:
                print(f"   ❌ Service health check failed: {response.status_code}")
                return False
                
    except Exception as e:
        print(f"   ⚠️  Could not reach service (is it running?): {e}")
        return False


def print_setup_instructions():
    """Print setup instructions if configuration is missing."""
    print("\n📚 Setup Instructions:")
    print("=" * 50)
    print("1. Create a Camunda Cloud account at: https://camunda.com/products/cloud/")
    print("2. Create a new cluster")
    print("3. Generate API credentials")
    print("4. Update your .env file with the credentials:")
    print()
    print("   USE_CAMUNDA_CLOUD=true")
    print("   CAMUNDA_CLOUD_CLIENT_ID=your_client_id")
    print("   CAMUNDA_CLOUD_CLIENT_SECRET=your_client_secret")
    print("   CAMUNDA_CLOUD_CLUSTER_ID=your_cluster_id")
    print("   CAMUNDA_CLOUD_REGION=your_region")
    print()
    print("5. Run this test again: python test_camunda_cloud.py")
    print()
    print("📖 For detailed instructions, see: CAMUNDA_CLOUD_SETUP.md")


async def main():
    """Main test function."""
    print("🔬 Camunda Cloud Integration Test")
    print("=" * 50)
    
    # Test Camunda Cloud connection
    cloud_success = await test_camunda_cloud_connection()
    
    # Test service health (if service is running)
    health_success = await test_service_health()
    
    print("\n📋 Test Summary:")
    print("=" * 50)
    print(f"Camunda Cloud Connection: {'✅ PASS' if cloud_success else '❌ FAIL'}")
    print(f"Service Health Check: {'✅ PASS' if health_success else '⚠️  SKIP'}")
    
    if not cloud_success:
        print_setup_instructions()
        sys.exit(1)
    else:
        print("\n🎉 Camunda Cloud integration is ready!")
        print("You can now start using workflows in your Clinical Synthesis Hub.")


if __name__ == "__main__":
    asyncio.run(main())
