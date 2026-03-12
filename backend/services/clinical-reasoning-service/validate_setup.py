#!/usr/bin/env python3
"""
Setup Validation Script

This script validates that all dependencies are properly installed
and our enhanced orchestration components can be imported.
"""

import sys
import importlib

def test_import(module_name, description=""):
    """Test if a module can be imported"""
    try:
        importlib.import_module(module_name)
        print(f"✅ {module_name} - {description}")
        return True
    except ImportError as e:
        print(f"❌ {module_name} - {description} - Error: {e}")
        return False
    except Exception as e:
        print(f"⚠️  {module_name} - {description} - Unexpected error: {e}")
        return False

def main():
    print("🔍 Python Environment Validation")
    print("=" * 50)
    print(f"Python version: {sys.version}")
    print("=" * 50)
    
    # Test core Python modules
    print("\n📦 Core Python Modules:")
    core_modules = [
        ("asyncio", "Async programming"),
        ("json", "JSON handling"),
        ("logging", "Logging system"),
        ("datetime", "Date/time handling"),
        ("typing", "Type hints"),
        ("dataclasses", "Data classes"),
        ("enum", "Enumerations"),
        ("collections", "Collections"),
        ("hashlib", "Hashing"),
        ("time", "Time utilities")
    ]
    
    core_success = 0
    for module, desc in core_modules:
        if test_import(module, desc):
            core_success += 1
    
    # Test third-party dependencies
    print("\n📚 Third-Party Dependencies:")
    third_party_modules = [
        ("numpy", "Numerical computing"),
        ("pandas", "Data analysis"),
        ("sklearn", "Machine learning"),
        ("networkx", "Graph algorithms"),
        ("redis", "Redis client"),
        ("fastapi", "Web framework"),
        ("grpcio", "gRPC framework"),
        ("pydantic", "Data validation"),
        ("httpx", "HTTP client"),
        ("aiohttp", "Async HTTP client")
    ]
    
    third_party_success = 0
    for module, desc in third_party_modules:
        if test_import(module, desc):
            third_party_success += 1
    
    # Test our application modules
    print("\n🏗️  Application Modules:")
    
    # Add current directory to path
    sys.path.insert(0, '.')
    
    app_modules = [
        ("app.orchestration.request_router", "Base request router"),
        ("app.cache.redis_client", "Redis cache client"),
        ("app.graph.graphdb_client", "GraphDB client"),
        ("app.reasoners.medication_interaction", "Medication interaction reasoner")
    ]
    
    app_success = 0
    for module, desc in app_modules:
        if test_import(module, desc):
            app_success += 1
    
    # Test enhanced orchestration modules
    print("\n🚀 Enhanced Orchestration Modules:")
    enhanced_modules = [
        ("app.orchestration.graph_request_router", "Graph-powered request router"),
        ("app.graph.query_optimizer", "Graph query optimizer"),
        ("app.cache.intelligent_cache", "Intelligent caching system"),
        ("app.orchestration.pattern_based_batching", "Pattern-based batching"),
        ("app.orchestration.intelligent_circuit_breaker", "Circuit breaker with learning")
    ]
    
    enhanced_success = 0
    for module, desc in enhanced_modules:
        if test_import(module, desc):
            enhanced_success += 1
    
    # Summary
    print("\n" + "=" * 50)
    print("📊 VALIDATION SUMMARY")
    print("=" * 50)
    print(f"Core Python modules: {core_success}/{len(core_modules)}")
    print(f"Third-party dependencies: {third_party_success}/{len(third_party_modules)}")
    print(f"Application modules: {app_success}/{len(app_modules)}")
    print(f"Enhanced orchestration: {enhanced_success}/{len(enhanced_modules)}")
    
    total_success = core_success + third_party_success + app_success + enhanced_success
    total_modules = len(core_modules) + len(third_party_modules) + len(app_modules) + len(enhanced_modules)
    
    print(f"\nOverall success rate: {total_success}/{total_modules} ({total_success/total_modules:.1%})")
    
    if total_success == total_modules:
        print("\n🎉 All modules imported successfully!")
        print("✅ Environment is ready for enhanced orchestration testing!")
    elif enhanced_success == len(enhanced_modules):
        print("\n🚀 Enhanced orchestration modules are working!")
        print("⚠️  Some optional dependencies may be missing, but core functionality is available.")
    else:
        print("\n⚠️  Some modules failed to import.")
        print("💡 Try installing missing dependencies with: py -m pip install -r requirements.txt")
    
    # Next steps
    print("\n📋 NEXT STEPS:")
    if enhanced_success == len(enhanced_modules):
        print("1. ✅ Run enhanced orchestration tests")
        print("2. ✅ Start the CAE server")
        print("3. ✅ Test with real clinical data")
    else:
        print("1. 🔧 Install missing dependencies")
        print("2. 🔍 Check import errors above")
        print("3. 🔄 Re-run this validation script")
    
    print("=" * 50)

if __name__ == "__main__":
    main()
