#!/usr/bin/env python3
"""
Simple test script for Workflow Engine Service.
Tests only core functionality without optional dependencies.
"""
import os
import sys

# Load environment variables from .env file
try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    print("⚠️  python-dotenv not available, .env file won't be loaded")

# Add the current directory to Python path
current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, current_dir)

# Add the backend services directory to Python path for shared imports
backend_dir = os.path.dirname(os.path.dirname(current_dir))
services_dir = os.path.join(backend_dir, "services")
sys.path.insert(0, services_dir)


def test_basic_imports():
    """Test basic imports."""
    print("Testing basic imports...")
    
    try:
        # Test configuration import
        from app.core.config import settings
        print(f"✅ Configuration loaded: {settings.SERVICE_NAME} v{settings.SERVICE_VERSION}")
        print(f"   Port: {settings.SERVICE_PORT}")
        print(f"   Supabase URL: {settings.SUPABASE_URL}")
        
        return True
        
    except ImportError as e:
        print(f"❌ Import error: {e}")
        return False
    except Exception as e:
        print(f"❌ Error: {e}")
        return False


def test_database_models():
    """Test database models import."""
    print("\nTesting database models...")
    
    try:
        from app.models.workflow_models import WorkflowDefinition, WorkflowInstance, WorkflowTask
        from app.models.task_models import TaskAssignment, TaskComment
        print("✅ Database models imported successfully")
        
        # Test model attributes
        print(f"✅ WorkflowDefinition table: {WorkflowDefinition.__tablename__}")
        print(f"✅ WorkflowInstance table: {WorkflowInstance.__tablename__}")
        print(f"✅ WorkflowTask table: {WorkflowTask.__tablename__}")
        
        return True
        
    except ImportError as e:
        print(f"❌ Database models import error: {e}")
        return False
    except Exception as e:
        print(f"❌ Database models error: {e}")
        return False


def test_database_connection():
    """Test database connection setup."""
    print("\nTesting database connection setup...")
    
    try:
        from app.db.database import engine, Base
        print("✅ Database engine created successfully")
        print(f"✅ Database URL configured: {str(engine.url)[:50]}...")
        
        return True
        
    except ImportError as e:
        print(f"❌ Database connection import error: {e}")
        print("   Please install: pip install sqlalchemy psycopg2-binary")
        return False
    except Exception as e:
        print(f"❌ Database connection error: {e}")
        return False


def test_supabase_config():
    """Test Supabase configuration."""
    print("\nTesting Supabase configuration...")
    
    try:
        from app.core.config import settings
        
        print(f"✅ Supabase URL: {settings.SUPABASE_URL}")
        print(f"✅ Supabase Key: {settings.SUPABASE_KEY[:20]}...")
        print(f"✅ Database URL: {settings.DATABASE_URL[:50]}...")
        
        return True
        
    except Exception as e:
        print(f"❌ Supabase configuration error: {e}")
        return False


def test_optional_services():
    """Test optional services."""
    print("\nTesting optional services...")
    
    # Test Google FHIR service
    try:
        from app.services.google_fhir_service import google_fhir_service
        print("✅ Google FHIR service imported successfully")
    except ImportError as e:
        print(f"⚠️  Google FHIR service not available: {e}")
    
    # Test Supabase service
    try:
        from app.services.supabase_service import supabase_service
        print("✅ Supabase service imported successfully")
    except ImportError as e:
        print(f"⚠️  Supabase service not available: {e}")
    
    return True


def main():
    """Run all tests."""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - SIMPLE SETUP TEST")
    print("=" * 60)
    
    tests = [
        ("Basic Imports", test_basic_imports),
        ("Database Models", test_database_models),
        ("Database Connection", test_database_connection),
        ("Supabase Configuration", test_supabase_config),
        ("Optional Services", test_optional_services),
    ]
    
    passed = 0
    total = len(tests)
    
    for test_name, test_func in tests:
        print(f"\n--- {test_name} ---")
        try:
            if test_func():
                passed += 1
                print(f"✅ {test_name} PASSED")
            else:
                print(f"❌ {test_name} FAILED")
        except Exception as e:
            print(f"❌ {test_name} ERROR: {e}")
    
    print("\n" + "=" * 60)
    print(f"TEST RESULTS: {passed}/{total} tests passed")
    
    if passed == total:
        print("🎉 All tests passed! Ready for database setup.")
        print("\nNext steps:")
        print("1. Run: python setup_database_simple.py")
        print("2. Run: python run_service.py")
    elif passed >= 3:
        print("⚠️  Core functionality working. Some optional features may not be available.")
        print("\nNext steps:")
        print("1. Install missing dependencies")
        print("2. Run: python setup_database_simple.py")
    else:
        print("❌ Critical issues found. Please install dependencies:")
        print("   pip install fastapi uvicorn sqlalchemy psycopg2-binary python-dotenv")
    
    print("=" * 60)


if __name__ == "__main__":
    main()
