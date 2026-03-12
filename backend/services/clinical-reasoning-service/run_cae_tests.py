"""
Test Runner for CAE Engine Neo4j Integration

Provides multiple test options:
1. Simple validation test
2. Comprehensive pytest suite
3. Performance benchmarks
4. Individual component tests
"""

import asyncio
import subprocess
import sys
import os
from pathlib import Path

# Load environment variables
try:
    from dotenv import load_dotenv
    load_dotenv()
    print("✅ Loaded environment variables from .env file")
except ImportError:
    print("⚠️  python-dotenv not installed")

def print_header():
    """Print test runner header"""
    print("🧪 CAE Engine Neo4j Integration Test Runner")
    print("=" * 60)
    print("Available test options:")
    print("1. 🚀 Quick Validation Test (Recommended)")
    print("2. 🔬 Comprehensive Test Suite (pytest)")
    print("3. ⚡ Performance Benchmarks")
    print("4. 🏥 Health Check Only")
    print("5. 📊 All Tests")
    print("=" * 60)

def run_command(command: str, description: str):
    """Run a command and return success status"""
    print(f"\n🔄 Running: {description}")
    print(f"Command: {command}")
    print("-" * 40)
    
    try:
        result = subprocess.run(command, shell=True, capture_output=False, text=True)
        success = result.returncode == 0
        
        if success:
            print(f"✅ {description}: PASSED")
        else:
            print(f"❌ {description}: FAILED (Exit code: {result.returncode})")
        
        return success
    
    except Exception as e:
        print(f"❌ {description}: ERROR - {e}")
        return False

async def run_health_check():
    """Run quick health check"""
    print("\n🏥 Running Health Check...")
    
    # Add app to path
    app_dir = Path(__file__).parent / "app"
    sys.path.insert(0, str(app_dir))
    
    try:
        from app.cae_engine_neo4j import CAEEngine
        
        cae_engine = CAEEngine()
        initialized = await cae_engine.initialize()
        
        if not initialized:
            print("❌ CAE Engine initialization failed")
            return False
        
        health = await cae_engine.get_health_status()
        
        print(f"Status: {health['status']}")
        print(f"Neo4j Connected: {health['neo4j_connection']}")
        print(f"Active Checkers: {health['checkers']}")
        
        await cae_engine.close()
        
        if health['status'] == 'HEALTHY' and health['neo4j_connection']:
            print("✅ Health Check: PASSED")
            return True
        else:
            print("❌ Health Check: FAILED")
            return False
    
    except Exception as e:
        print(f"❌ Health Check: ERROR - {e}")
        return False

def main():
    """Main test runner"""
    print_header()
    
    # Get user choice
    try:
        choice = input("\nEnter your choice (1-5): ").strip()
    except KeyboardInterrupt:
        print("\n👋 Test runner cancelled")
        return
    
    python_exe = sys.executable
    results = []
    
    if choice == "1":
        # Quick validation test
        success = run_command(
            f'"{python_exe}" test_cae_simple.py',
            "Quick Validation Test"
        )
        results.append(("Quick Validation", success))
    
    elif choice == "2":
        # Comprehensive pytest suite
        success = run_command(
            f'"{python_exe}" -m pytest test_neo4j_cae_integration.py -v',
            "Comprehensive Test Suite"
        )
        results.append(("Comprehensive Tests", success))
    
    elif choice == "3":
        # Performance benchmarks
        success = run_command(
            f'"{python_exe}" -m pytest test_neo4j_cae_integration.py::TestNeo4jCAEIntegration::test_performance_benchmarks -v',
            "Performance Benchmarks"
        )
        results.append(("Performance Tests", success))
    
    elif choice == "4":
        # Health check only
        success = asyncio.run(run_health_check())
        results.append(("Health Check", success))
    
    elif choice == "5":
        # All tests
        print("\n🚀 Running All Tests...")
        
        # Health check
        health_success = asyncio.run(run_health_check())
        results.append(("Health Check", health_success))
        
        # Quick validation
        quick_success = run_command(
            f'"{python_exe}" test_cae_simple.py',
            "Quick Validation Test"
        )
        results.append(("Quick Validation", quick_success))
        
        # Comprehensive tests
        comprehensive_success = run_command(
            f'"{python_exe}" -m pytest test_neo4j_cae_integration.py -v',
            "Comprehensive Test Suite"
        )
        results.append(("Comprehensive Tests", comprehensive_success))
    
    else:
        print("❌ Invalid choice. Please select 1-5.")
        return
    
    # Print summary
    print("\n" + "=" * 60)
    print("📊 TEST EXECUTION SUMMARY")
    print("=" * 60)
    
    total_tests = len(results)
    passed_tests = sum(1 for _, success in results if success)
    
    for test_name, success in results:
        status = "✅ PASSED" if success else "❌ FAILED"
        print(f"{test_name}: {status}")
    
    success_rate = (passed_tests / total_tests * 100) if total_tests > 0 else 0
    
    print(f"\n📈 Overall Success Rate: {success_rate:.1f}%")
    print(f"✅ Passed: {passed_tests}")
    print(f"❌ Failed: {total_tests - passed_tests}")
    
    if success_rate >= 80:
        print("\n🎉 CAE Engine Neo4j Integration: SUCCESSFUL!")
        print("✅ Ready for production deployment")
    else:
        print("\n⚠️  CAE Engine Neo4j Integration: NEEDS ATTENTION")
        print("❌ Some tests failed - review logs for details")
    
    print("\n📝 Next Steps:")
    if success_rate >= 80:
        print("1. Deploy CAE Engine to production")
        print("2. Integrate with Safety Gateway Platform")
        print("3. Monitor performance metrics")
        print("4. Enhance knowledge graph data")
    else:
        print("1. Review failed test logs")
        print("2. Check Neo4j connection and data")
        print("3. Verify environment configuration")
        print("4. Re-run tests after fixes")

if __name__ == "__main__":
    main()
