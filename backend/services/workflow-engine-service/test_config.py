#!/usr/bin/env python3
"""
Simple configuration test for Workflow Engine Service.
"""
import os
import sys
from pathlib import Path

# Add the app directory to Python path
sys.path.insert(0, str(Path(__file__).parent / "app"))

from app.core.config import settings


def test_configuration():
    """Test the configuration settings."""
    
    print("🔧 Workflow Engine Service Configuration Test")
    print("=" * 60)
    
    # Basic service configuration
    print("📋 Basic Service Configuration:")
    print(f"   SERVICE_NAME: {settings.SERVICE_NAME}")
    print(f"   SERVICE_VERSION: {settings.SERVICE_VERSION}")
    print(f"   SERVICE_PORT: {settings.SERVICE_PORT}")
    print(f"   DEBUG: {settings.DEBUG}")
    print()
    
    # Database configuration
    print("🗄️  Database Configuration:")
    print(f"   SUPABASE_URL: {settings.SUPABASE_URL}")
    print(f"   SUPABASE_KEY: {settings.SUPABASE_KEY[:20]}..." if settings.SUPABASE_KEY else "   SUPABASE_KEY: Not set")
    print(f"   DATABASE_URL: {'Set' if settings.DATABASE_URL else 'Not set'}")
    print()
    
    # Google Healthcare API configuration
    print("🏥 Google Healthcare API Configuration:")
    print(f"   USE_GOOGLE_HEALTHCARE_API: {settings.USE_GOOGLE_HEALTHCARE_API}")
    print(f"   GOOGLE_CLOUD_PROJECT: {settings.GOOGLE_CLOUD_PROJECT}")
    print(f"   GOOGLE_CLOUD_LOCATION: {settings.GOOGLE_CLOUD_LOCATION}")
    print(f"   GOOGLE_CLOUD_DATASET: {settings.GOOGLE_CLOUD_DATASET}")
    print(f"   GOOGLE_CLOUD_FHIR_STORE: {settings.GOOGLE_CLOUD_FHIR_STORE}")
    print()
    
    # Camunda configuration
    print("⚙️  Camunda Configuration:")
    print(f"   USE_CAMUNDA_CLOUD: {settings.USE_CAMUNDA_CLOUD}")
    
    if settings.USE_CAMUNDA_CLOUD:
        print("   🌟 Camunda Cloud Settings:")
        print(f"      CLIENT_ID: {settings.CAMUNDA_CLOUD_CLIENT_ID[:8]}..." if settings.CAMUNDA_CLOUD_CLIENT_ID else "      CLIENT_ID: Not set")
        print(f"      CLIENT_SECRET: {'Set' if settings.CAMUNDA_CLOUD_CLIENT_SECRET else 'Not set'}")
        print(f"      CLUSTER_ID: {settings.CAMUNDA_CLOUD_CLUSTER_ID}")
        print(f"      REGION: {settings.CAMUNDA_CLOUD_REGION}")
        print(f"      AUTH_SERVER: {settings.CAMUNDA_CLOUD_AUTHORIZATION_SERVER_URL}")
        
        # Validate Camunda Cloud configuration
        missing_config = []
        if not settings.CAMUNDA_CLOUD_CLIENT_ID:
            missing_config.append("CLIENT_ID")
        if not settings.CAMUNDA_CLOUD_CLIENT_SECRET:
            missing_config.append("CLIENT_SECRET")
        if not settings.CAMUNDA_CLOUD_CLUSTER_ID:
            missing_config.append("CLUSTER_ID")
        if not settings.CAMUNDA_CLOUD_REGION:
            missing_config.append("REGION")
        
        if missing_config:
            print(f"      ❌ Missing: {', '.join(missing_config)}")
        else:
            print("      ✅ All Camunda Cloud settings configured")
    else:
        print("   🏠 Local Camunda Settings:")
        print(f"      ENGINE_URL: {settings.CAMUNDA_ENGINE_URL}")
    
    print()
    
    # External services configuration
    print("🔗 External Services Configuration:")
    print(f"   AUTH_SERVICE_URL: {settings.AUTH_SERVICE_URL}")
    print(f"   PATIENT_SERVICE_URL: {settings.PATIENT_SERVICE_URL}")
    print(f"   MEDICATION_SERVICE_URL: {settings.MEDICATION_SERVICE_URL}")
    print(f"   ORDER_SERVICE_URL: {settings.ORDER_SERVICE_URL}")
    print(f"   SCHEDULING_SERVICE_URL: {settings.SCHEDULING_SERVICE_URL}")
    print(f"   ENCOUNTER_SERVICE_URL: {settings.ENCOUNTER_SERVICE_URL}")
    print()
    
    # Workflow engine settings
    print("⏱️  Workflow Engine Settings:")
    print(f"   WORKFLOW_EXECUTION_TIMEOUT: {settings.WORKFLOW_EXECUTION_TIMEOUT} seconds")
    print(f"   TASK_ASSIGNMENT_TIMEOUT: {settings.TASK_ASSIGNMENT_TIMEOUT} seconds")
    print(f"   EVENT_POLLING_INTERVAL: {settings.EVENT_POLLING_INTERVAL} seconds")
    print(f"   TASK_POLLING_INTERVAL: {settings.TASK_POLLING_INTERVAL} seconds")
    print()
    
    # Check for .env file
    env_file = Path(__file__).parent / ".env"
    print("📄 Environment File:")
    print(f"   .env file exists: {'✅ Yes' if env_file.exists() else '❌ No'}")
    if env_file.exists():
        print(f"   .env file path: {env_file}")
    print()
    
    # Overall status
    print("📊 Configuration Status:")
    
    issues = []
    
    # Check critical settings
    if not settings.SUPABASE_URL:
        issues.append("Missing SUPABASE_URL")
    if not settings.DATABASE_URL:
        issues.append("Missing DATABASE_URL")
    
    if settings.USE_CAMUNDA_CLOUD:
        if not all([
            settings.CAMUNDA_CLOUD_CLIENT_ID,
            settings.CAMUNDA_CLOUD_CLIENT_SECRET,
            settings.CAMUNDA_CLOUD_CLUSTER_ID,
            settings.CAMUNDA_CLOUD_REGION
        ]):
            issues.append("Incomplete Camunda Cloud configuration")
    
    if issues:
        print("   ❌ Issues found:")
        for issue in issues:
            print(f"      - {issue}")
        return False
    else:
        print("   ✅ Configuration looks good!")
        return True


def test_imports():
    """Test if all required modules can be imported."""
    
    print("📦 Testing Module Imports:")
    print("=" * 60)
    
    imports_status = {}
    
    # Test core imports
    try:
        from app.services.workflow_engine_service import workflow_engine_service
        imports_status["workflow_engine_service"] = "✅"
    except Exception as e:
        imports_status["workflow_engine_service"] = f"❌ {e}"
    
    try:
        from app.services.workflow_definition_service import workflow_definition_service
        imports_status["workflow_definition_service"] = "✅"
    except Exception as e:
        imports_status["workflow_definition_service"] = f"❌ {e}"
    
    try:
        from app.services.workflow_instance_service import workflow_instance_service
        imports_status["workflow_instance_service"] = "✅"
    except Exception as e:
        imports_status["workflow_instance_service"] = f"❌ {e}"
    
    try:
        from app.services.task_service import task_service
        imports_status["task_service"] = "✅"
    except Exception as e:
        imports_status["task_service"] = f"❌ {e}"
    
    try:
        from app.services.camunda_service import camunda_service
        imports_status["camunda_service"] = "✅"
    except Exception as e:
        imports_status["camunda_service"] = f"❌ {e}"
    
    try:
        from app.services.camunda_cloud_service import camunda_cloud_service
        imports_status["camunda_cloud_service"] = "✅"
    except Exception as e:
        imports_status["camunda_cloud_service"] = f"❌ {e}"
    
    try:
        from app.services.supabase_service import supabase_service
        imports_status["supabase_service"] = "✅"
    except Exception as e:
        imports_status["supabase_service"] = f"❌ {e}"
    
    # Print results
    for service, status in imports_status.items():
        print(f"   {service}: {status}")
    
    print()
    
    # Check for successful imports
    successful_imports = sum(1 for status in imports_status.values() if status == "✅")
    total_imports = len(imports_status)
    
    print(f"📊 Import Summary: {successful_imports}/{total_imports} successful")
    
    return successful_imports == total_imports


def main():
    """Main test function."""
    print("🧪 Workflow Engine Service - Configuration & Import Test")
    print("=" * 70)
    print()
    
    # Test configuration
    config_ok = test_configuration()
    print()
    
    # Test imports
    imports_ok = test_imports()
    print()
    
    # Final summary
    print("🎯 Test Summary:")
    print("=" * 70)
    print(f"Configuration Test: {'✅ PASS' if config_ok else '❌ FAIL'}")
    print(f"Import Test: {'✅ PASS' if imports_ok else '❌ FAIL'}")
    
    if config_ok and imports_ok:
        print("\n🎉 All tests passed! The service is ready to run.")
        print("\nNext steps:")
        print("1. Start the service: python run_service.py")
        print("2. Check health: curl http://localhost:8015/health")
        print("3. Install Camunda dependencies if needed: pip install pyzeebe")
    else:
        print("\n⚠️  Some tests failed. Please check the configuration and dependencies.")
        
        if not config_ok:
            print("\nConfiguration issues:")
            print("- Check your .env file")
            print("- Ensure all required environment variables are set")
        
        if not imports_ok:
            print("\nImport issues:")
            print("- Install missing dependencies: pip install -r requirements.txt")
            print("- Check for syntax errors in the code")


if __name__ == "__main__":
    main()
