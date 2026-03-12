#!/usr/bin/env python3
"""
Test Runner for Clinical Context Service
Runs comprehensive integration tests for all three pillars
"""
import sys
import subprocess
import os
from pathlib import Path

def run_tests():
    """Run all tests for the Clinical Context Service"""
    print("🧪 Clinical Context Service - Comprehensive Test Suite")
    print("=" * 60)
    print("Testing the Three Pillars of Excellence:")
    print("1. Federated GraphQL API (The 'Unified Data Graph')")
    print("2. Clinical Context Recipe System (The 'Governance Engine')")
    print("3. Multi-Layer Intelligent Cache (The 'Performance Accelerator')")
    print("=" * 60)
    
    # Change to the service directory
    service_dir = Path(__file__).parent
    os.chdir(service_dir)
    
    # Test commands to run
    test_commands = [
        {
            "name": "Recipe System Integration Tests",
            "command": ["python", "-m", "pytest", "tests/test_recipe_system_integration.py", "-v", "-s"],
            "description": "Tests Pillar 2: Clinical Context Recipe System"
        },
        {
            "name": "GraphQL API Tests",
            "command": ["python", "-m", "pytest", "tests/test_graphql_api.py", "-v", "-s"],
            "description": "Tests Pillar 1: Federated GraphQL API"
        },
        {
            "name": "All Tests with Coverage",
            "command": ["python", "-m", "pytest", "tests/", "--cov=app", "--cov-report=html", "--cov-report=term-missing", "-v"],
            "description": "Complete test suite with coverage analysis"
        }
    ]
    
    results = []
    
    for test_suite in test_commands:
        print(f"\n🔬 Running {test_suite['name']}")
        print(f"   {test_suite['description']}")
        print("-" * 50)
        
        try:
            result = subprocess.run(
                test_suite["command"],
                capture_output=False,
                text=True,
                timeout=300  # 5 minutes timeout
            )
            
            if result.returncode == 0:
                print(f"✅ {test_suite['name']} - PASSED")
                results.append((test_suite['name'], "PASSED"))
            else:
                print(f"❌ {test_suite['name']} - FAILED")
                results.append((test_suite['name'], "FAILED"))
                
        except subprocess.TimeoutExpired:
            print(f"⏰ {test_suite['name']} - TIMEOUT")
            results.append((test_suite['name'], "TIMEOUT"))
        except Exception as e:
            print(f"💥 {test_suite['name']} - ERROR: {e}")
            results.append((test_suite['name'], "ERROR"))
    
    # Print summary
    print("\n" + "=" * 60)
    print("🏁 TEST SUMMARY")
    print("=" * 60)
    
    passed = 0
    failed = 0
    
    for test_name, status in results:
        status_icon = "✅" if status == "PASSED" else "❌"
        print(f"{status_icon} {test_name}: {status}")
        
        if status == "PASSED":
            passed += 1
        else:
            failed += 1
    
    print("-" * 60)
    print(f"Total Tests: {len(results)}")
    print(f"Passed: {passed}")
    print(f"Failed: {failed}")
    
    if failed == 0:
        print("\n🎉 ALL TESTS PASSED! Clinical Context Service is ready for deployment.")
        return 0
    else:
        print(f"\n⚠️  {failed} test suite(s) failed. Please review and fix issues.")
        return 1


def install_dependencies():
    """Install test dependencies"""
    print("📦 Installing test dependencies...")
    
    try:
        subprocess.run([
            sys.executable, "-m", "pip", "install", "-r", "requirements.txt"
        ], check=True)
        
        # Install additional test dependencies
        test_deps = [
            "pytest>=7.4.3",
            "pytest-asyncio>=0.21.1",
            "pytest-mock>=3.12.0",
            "pytest-cov>=4.1.0",
            "httpx>=0.25.2"
        ]
        
        subprocess.run([
            sys.executable, "-m", "pip", "install"
        ] + test_deps, check=True)
        
        print("✅ Dependencies installed successfully")
        return True
        
    except subprocess.CalledProcessError as e:
        print(f"❌ Failed to install dependencies: {e}")
        return False


def check_environment():
    """Check if the environment is ready for testing"""
    print("🔍 Checking test environment...")
    
    # Check Python version
    if sys.version_info < (3, 8):
        print("❌ Python 3.8+ is required")
        return False
    
    print(f"✅ Python version: {sys.version}")
    
    # Check if we're in the right directory
    if not Path("app").exists():
        print("❌ Not in the correct service directory")
        return False
    
    print("✅ Service directory structure verified")
    
    # Check if requirements.txt exists
    if not Path("requirements.txt").exists():
        print("❌ requirements.txt not found")
        return False
    
    print("✅ Requirements file found")
    
    return True


def main():
    """Main test runner function"""
    print("🚀 Clinical Context Service Test Runner")
    print("Implementing the Three Pillars of Excellence")
    
    # Check environment
    if not check_environment():
        print("❌ Environment check failed")
        return 1
    
    # Install dependencies
    if not install_dependencies():
        print("❌ Dependency installation failed")
        return 1
    
    # Run tests
    return run_tests()


if __name__ == "__main__":
    exit_code = main()
    sys.exit(exit_code)
