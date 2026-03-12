#!/usr/bin/env python3
"""
Comprehensive test runner for Workflow Engine Service.
"""
import asyncio
import os
import sys
import subprocess
import argparse
from pathlib import Path
from typing import List, Dict, Any

# Add the app directory to Python path
sys.path.insert(0, str(Path(__file__).parent / "app"))

def run_command(command: List[str], cwd: str = None) -> Dict[str, Any]:
    """Run a command and return the result."""
    try:
        result = subprocess.run(
            command,
            cwd=cwd or Path(__file__).parent,
            capture_output=True,
            text=True,
            timeout=300  # 5 minutes timeout
        )
        return {
            "success": result.returncode == 0,
            "stdout": result.stdout,
            "stderr": result.stderr,
            "returncode": result.returncode
        }
    except subprocess.TimeoutExpired:
        return {
            "success": False,
            "stdout": "",
            "stderr": "Command timed out after 5 minutes",
            "returncode": -1
        }
    except Exception as e:
        return {
            "success": False,
            "stdout": "",
            "stderr": str(e),
            "returncode": -1
        }

def install_dependencies():
    """Install test dependencies."""
    print("📦 Installing test dependencies...")
    
    # Install main dependencies
    result = run_command([sys.executable, "-m", "pip", "install", "-r", "requirements.txt"])
    if not result["success"]:
        print(f"❌ Failed to install main dependencies: {result['stderr']}")
        return False
    
    # Install additional test dependencies
    test_deps = [
        "pytest-cov==4.1.0",
        "pytest-mock==3.12.0",
        "pytest-xdist==3.5.0",
        "factory-boy==3.3.0"
    ]
    
    for dep in test_deps:
        result = run_command([sys.executable, "-m", "pip", "install", dep])
        if not result["success"]:
            print(f"⚠️  Warning: Failed to install {dep}: {result['stderr']}")
    
    print("✅ Dependencies installed successfully")
    return True

def run_unit_tests(verbose: bool = False, coverage: bool = False):
    """Run unit tests."""
    print("\n🧪 Running Unit Tests...")
    print("=" * 50)
    
    command = [sys.executable, "-m", "pytest", "tests/unit/"]
    
    if verbose:
        command.append("-v")
    
    if coverage:
        command.extend(["--cov=app", "--cov-report=term-missing", "--cov-report=html"])
    
    command.extend(["-m", "unit"])
    
    result = run_command(command)
    
    if result["success"]:
        print("✅ Unit tests passed")
        print(result["stdout"])
    else:
        print("❌ Unit tests failed")
        print(result["stderr"])
        if result["stdout"]:
            print(result["stdout"])
    
    return result["success"]

def run_integration_tests(verbose: bool = False):
    """Run integration tests."""
    print("\n🔗 Running Integration Tests...")
    print("=" * 50)
    
    command = [sys.executable, "-m", "pytest", "tests/integration/"]
    
    if verbose:
        command.append("-v")
    
    command.extend(["-m", "integration"])
    
    result = run_command(command)
    
    if result["success"]:
        print("✅ Integration tests passed")
        print(result["stdout"])
    else:
        print("❌ Integration tests failed")
        print(result["stderr"])
        if result["stdout"]:
            print(result["stdout"])
    
    return result["success"]

def run_federation_tests(verbose: bool = False):
    """Run federation tests."""
    print("\n🌐 Running Federation Tests...")
    print("=" * 50)
    
    command = [sys.executable, "-m", "pytest", "tests/integration/test_graphql_federation.py"]
    
    if verbose:
        command.append("-v")
    
    command.extend(["-m", "federation"])
    
    result = run_command(command)
    
    if result["success"]:
        print("✅ Federation tests passed")
        print(result["stdout"])
    else:
        print("❌ Federation tests failed")
        print(result["stderr"])
        if result["stdout"]:
            print(result["stdout"])
    
    return result["success"]

def run_workflow_tests(verbose: bool = False):
    """Run end-to-end workflow tests."""
    print("\n🔄 Running Workflow Tests...")
    print("=" * 50)
    
    command = [sys.executable, "-m", "pytest", "tests/integration/test_end_to_end_workflow.py"]
    
    if verbose:
        command.append("-v")
    
    command.extend(["-m", "workflow"])
    
    result = run_command(command)
    
    if result["success"]:
        print("✅ Workflow tests passed")
        print(result["stdout"])
    else:
        print("❌ Workflow tests failed")
        print(result["stderr"])
        if result["stdout"]:
            print(result["stdout"])
    
    return result["success"]

def run_legacy_tests():
    """Run existing legacy test files."""
    print("\n🔧 Running Legacy Tests...")
    print("=" * 50)
    
    legacy_tests = [
        "test_simple.py",
        "test_service.py",
        "test_config.py",
        "test_phase4_integration.py",
        "test_phase5_features.py"
    ]
    
    results = []
    
    for test_file in legacy_tests:
        if Path(test_file).exists():
            print(f"\n--- Running {test_file} ---")
            result = run_command([sys.executable, test_file])
            
            if result["success"]:
                print(f"✅ {test_file} passed")
            else:
                print(f"❌ {test_file} failed")
                print(result["stderr"])
            
            results.append(result["success"])
        else:
            print(f"⚠️  {test_file} not found, skipping")
    
    return all(results) if results else True

def run_linting():
    """Run code linting."""
    print("\n🔍 Running Code Linting...")
    print("=" * 50)
    
    # Check if flake8 is available
    flake8_result = run_command([sys.executable, "-m", "flake8", "--version"])
    if not flake8_result["success"]:
        print("⚠️  flake8 not available, skipping linting")
        return True
    
    # Run flake8
    result = run_command([
        sys.executable, "-m", "flake8", 
        "app/", "tests/",
        "--max-line-length=120",
        "--ignore=E501,W503"
    ])
    
    if result["success"]:
        print("✅ Code linting passed")
    else:
        print("❌ Code linting failed")
        print(result["stdout"])
        print(result["stderr"])
    
    return result["success"]

def generate_test_report(results: Dict[str, bool]):
    """Generate a test report."""
    print("\n📊 Test Report")
    print("=" * 50)
    
    total_tests = len(results)
    passed_tests = sum(1 for result in results.values() if result)
    failed_tests = total_tests - passed_tests
    
    print(f"Total Test Suites: {total_tests}")
    print(f"Passed: {passed_tests}")
    print(f"Failed: {failed_tests}")
    print(f"Success Rate: {(passed_tests/total_tests)*100:.1f}%")
    
    print("\nDetailed Results:")
    for test_name, result in results.items():
        status = "✅ PASSED" if result else "❌ FAILED"
        print(f"  {test_name}: {status}")
    
    if failed_tests == 0:
        print("\n🎉 All tests passed! The service is ready for deployment.")
    else:
        print(f"\n⚠️  {failed_tests} test suite(s) failed. Please review and fix the issues.")
    
    return failed_tests == 0

def main():
    """Main test runner function."""
    parser = argparse.ArgumentParser(description="Workflow Engine Service Test Runner")
    parser.add_argument("--unit", action="store_true", help="Run only unit tests")
    parser.add_argument("--integration", action="store_true", help="Run only integration tests")
    parser.add_argument("--federation", action="store_true", help="Run only federation tests")
    parser.add_argument("--workflow", action="store_true", help="Run only workflow tests")
    parser.add_argument("--legacy", action="store_true", help="Run only legacy tests")
    parser.add_argument("--lint", action="store_true", help="Run only linting")
    parser.add_argument("--coverage", action="store_true", help="Generate coverage report")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    parser.add_argument("--install-deps", action="store_true", help="Install dependencies before running tests")
    
    args = parser.parse_args()
    
    print("🚀 Workflow Engine Service - Phase 6 Test Suite")
    print("=" * 60)
    
    # Install dependencies if requested
    if args.install_deps:
        if not install_dependencies():
            sys.exit(1)
    
    results = {}
    
    # Run specific test types if requested
    if args.unit:
        results["Unit Tests"] = run_unit_tests(args.verbose, args.coverage)
    elif args.integration:
        results["Integration Tests"] = run_integration_tests(args.verbose)
    elif args.federation:
        results["Federation Tests"] = run_federation_tests(args.verbose)
    elif args.workflow:
        results["Workflow Tests"] = run_workflow_tests(args.verbose)
    elif args.legacy:
        results["Legacy Tests"] = run_legacy_tests()
    elif args.lint:
        results["Code Linting"] = run_linting()
    else:
        # Run all tests
        results["Unit Tests"] = run_unit_tests(args.verbose, args.coverage)
        results["Integration Tests"] = run_integration_tests(args.verbose)
        results["Federation Tests"] = run_federation_tests(args.verbose)
        results["Workflow Tests"] = run_workflow_tests(args.verbose)
        results["Legacy Tests"] = run_legacy_tests()
        results["Code Linting"] = run_linting()
    
    # Generate report
    all_passed = generate_test_report(results)
    
    # Exit with appropriate code
    sys.exit(0 if all_passed else 1)

if __name__ == "__main__":
    main()
