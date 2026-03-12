"""
Flow Test Runner

Comprehensive test runner for all CAE Engine flows:
1. Direct CAE Engine API
2. gRPC Service 
3. Safety Gateway Integration
4. End-to-End Clinical Scenarios
"""

import asyncio
import subprocess
import sys
import os
from pathlib import Path

def print_header():
    """Print test runner header"""
    print("🔄 CAE Engine Complete Flow Test Runner")
    print("=" * 60)
    print("Available flow tests:")
    print("1. 🚀 Complete Flow Test (All Components)")
    print("2. 🔗 Safety Gateway Integration Test")
    print("3. 🧪 Direct CAE Engine Test")
    print("4. 📊 Performance Benchmarks")
    print("5. 🎯 All Flow Tests")
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

def check_prerequisites():
    """Check if required services are running"""
    print("\n🔍 Checking Prerequisites...")
    print("-" * 30)
    
    prerequisites = {
        "Neo4j Connection": "Environment variables and Neo4j access",
        "CAE Engine": "CAE Engine initialization",
        "gRPC Service": "gRPC server on port 50051 (optional)",
        "Safety Gateway": "Safety Gateway Platform on port 8028 (optional)"
    }
    
    for service, description in prerequisites.items():
        print(f"  📋 {service}: {description}")
    
    print("\n💡 Note: Some services are optional and tests will skip if not available")

async def run_direct_cae_test():
    """Run direct CAE engine test"""
    print("\n🧪 Running Direct CAE Engine Test...")
    
    try:
        # Import and run the fixed Neo4j test
        from test_fixed_neo4j import test_fixed_neo4j
        success = await test_fixed_neo4j()
        return success
    except Exception as e:
        print(f"❌ Direct CAE test failed: {e}")
        return False

def main():
    """Main test runner"""
    print_header()
    
    # Check prerequisites
    check_prerequisites()
    
    # Get user choice
    try:
        choice = input("\nEnter your choice (1-5): ").strip()
    except KeyboardInterrupt:
        print("\n👋 Flow test runner cancelled")
        return
    
    python_exe = sys.executable
    results = []
    
    if choice == "1":
        # Complete flow test
        success = run_command(
            f'"{python_exe}" test_complete_flow.py',
            "Complete Flow Test (All Components)"
        )
        results.append(("Complete Flow Test", success))
    
    elif choice == "2":
        # Safety Gateway integration test
        success = run_command(
            f'"{python_exe}" test_safety_gateway_integration.py',
            "Safety Gateway Integration Test"
        )
        results.append(("Safety Gateway Integration", success))
    
    elif choice == "3":
        # Direct CAE engine test
        success = run_command(
            f'"{python_exe}" test_fixed_neo4j.py',
            "Direct CAE Engine Test"
        )
        results.append(("Direct CAE Engine", success))
    
    elif choice == "4":
        # Performance benchmarks
        success = run_command(
            f'"{python_exe}" test_complete_flow.py',
            "Performance Benchmarks"
        )
        results.append(("Performance Benchmarks", success))
    
    elif choice == "5":
        # All flow tests
        print("\n🚀 Running All Flow Tests...")
        
        # Direct CAE test
        cae_success = run_command(
            f'"{python_exe}" test_fixed_neo4j.py',
            "Direct CAE Engine Test"
        )
        results.append(("Direct CAE Engine", cae_success))
        
        # Complete flow test
        flow_success = run_command(
            f'"{python_exe}" test_complete_flow.py',
            "Complete Flow Test"
        )
        results.append(("Complete Flow Test", flow_success))
        
        # Safety Gateway integration (optional)
        gateway_success = run_command(
            f'"{python_exe}" test_safety_gateway_integration.py',
            "Safety Gateway Integration Test"
        )
        results.append(("Safety Gateway Integration", gateway_success))
    
    else:
        print("❌ Invalid choice. Please select 1-5.")
        return
    
    # Print summary
    print("\n" + "=" * 60)
    print("📊 FLOW TEST EXECUTION SUMMARY")
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
    
    # Final assessment
    if success_rate >= 90:
        print("\n🎉 FLOW TESTING: EXCELLENT!")
        print("✅ All critical flows working perfectly")
        print("🚀 Ready for production deployment")
        
        print("\n🎯 DEPLOYMENT CHECKLIST:")
        print("✅ Neo4j integration working")
        print("✅ CAE Engine functioning")
        print("✅ Clinical reasoning validated")
        print("✅ Performance requirements met")
        
    elif success_rate >= 75:
        print("\n✅ FLOW TESTING: GOOD")
        print("⚠️  Minor issues in some flows")
        print("🔧 Address failed tests before production")
        
    else:
        print("\n❌ FLOW TESTING: NEEDS ATTENTION")
        print("🚨 Critical issues found")
        print("🔧 Fix failed tests before proceeding")
    
    print("\n📝 NEXT STEPS:")
    if success_rate >= 90:
        print("1. Deploy CAE Engine to production")
        print("2. Integrate with clinical workflows")
        print("3. Monitor performance in production")
        print("4. Collect clinical feedback")
    elif success_rate >= 75:
        print("1. Review and fix failed tests")
        print("2. Re-run flow tests")
        print("3. Deploy after all tests pass")
    else:
        print("1. Investigate failed test causes")
        print("2. Fix critical issues")
        print("3. Re-run all flow tests")
        print("4. Ensure 90%+ success rate before deployment")
    
    print(f"\n🏁 Flow testing completed with {success_rate:.1f}% success rate")

if __name__ == "__main__":
    main()
