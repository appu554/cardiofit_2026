#!/usr/bin/env python3
"""
Comprehensive test suite for the Unified Dose Safety Engine
Tests all API endpoints, performance requirements, and integration scenarios
"""

import asyncio
import aiohttp
import json
import time
import sys
import subprocess
from typing import Dict, List, Any
from concurrent.futures import ThreadPoolExecutor
import uuid

class UnifiedEngineTestSuite:
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.session = None
        self.test_results = []
        
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    def log_test(self, test_name: str, success: bool, duration_ms: float, details: str = ""):
        """Log test results"""
        status = "✅ PASS" if success else "❌ FAIL"
        print(f"{status} {test_name} ({duration_ms:.1f}ms) {details}")
        self.test_results.append({
            "test": test_name,
            "success": success,
            "duration_ms": duration_ms,
            "details": details
        })
    
    async def test_health_endpoint(self):
        """Test basic health endpoint"""
        start_time = time.time()
        try:
            async with self.session.get(f"{self.base_url}/health") as response:
                duration_ms = (time.time() - start_time) * 1000
                
                if response.status == 200:
                    data = await response.json()
                    if data.get("status") == "healthy" and data.get("engine") == "unified-clinical-engine":
                        self.log_test("Health Check", True, duration_ms, f"Status: {data.get('status')}")
                        return True
                    else:
                        self.log_test("Health Check", False, duration_ms, f"Invalid response: {data}")
                        return False
                else:
                    self.log_test("Health Check", False, duration_ms, f"HTTP {response.status}")
                    return False
        except Exception as e:
            duration_ms = (time.time() - start_time) * 1000
            self.log_test("Health Check", False, duration_ms, f"Exception: {str(e)}")
            return False
    
    async def test_dose_optimization(self):
        """Test dose optimization endpoint"""
        start_time = time.time()
        try:
            request_data = {
                "request_id": f"test-dose-{uuid.uuid4()}",
                "patient_id": "test-patient-123",
                "medication_code": "metformin",
                "clinical_parameters": {},
                "optimization_type": "standard",
                "clinical_context": {},
                "processing_hints": {}
            }
            
            async with self.session.post(
                f"{self.base_url}/api/dose/optimize",
                json=request_data,
                headers={"Content-Type": "application/json"}
            ) as response:
                duration_ms = (time.time() - start_time) * 1000
                
                if response.status == 200:
                    data = await response.json()
                    if (data.get("request_id") == request_data["request_id"] and 
                        "optimized_dose" in data and 
                        data.get("execution_time_ms", 0) > 0):
                        self.log_test("Dose Optimization", True, duration_ms, 
                                    f"Dose: {data.get('optimized_dose')}mg")
                        return True
                    else:
                        self.log_test("Dose Optimization", False, duration_ms, 
                                    f"Invalid response structure: {data}")
                        return False
                else:
                    self.log_test("Dose Optimization", False, duration_ms, f"HTTP {response.status}")
                    return False
        except Exception as e:
            duration_ms = (time.time() - start_time) * 1000
            self.log_test("Dose Optimization", False, duration_ms, f"Exception: {str(e)}")
            return False
    
    async def test_medication_intelligence(self):
        """Test medication intelligence endpoint"""
        start_time = time.time()
        try:
            request_data = {
                "request_id": f"test-intel-{uuid.uuid4()}",
                "patient_id": "test-patient-123",
                "medications": [{"code": "metformin", "dose": "500mg"}],
                "patient_context": {},
                "analysis_type": "comprehensive",
                "clinical_context": {}
            }
            
            async with self.session.post(
                f"{self.base_url}/api/medication/intelligence",
                json=request_data,
                headers={"Content-Type": "application/json"}
            ) as response:
                duration_ms = (time.time() - start_time) * 1000
                
                if response.status == 200:
                    data = await response.json()
                    if (data.get("request_id") == request_data["request_id"] and 
                        "intelligence_score" in data and 
                        data.get("execution_time_ms", 0) > 0):
                        self.log_test("Medication Intelligence", True, duration_ms, 
                                    f"Score: {data.get('intelligence_score')}")
                        return True
                    else:
                        self.log_test("Medication Intelligence", False, duration_ms, 
                                    f"Invalid response: {data}")
                        return False
                else:
                    self.log_test("Medication Intelligence", False, duration_ms, f"HTTP {response.status}")
                    return False
        except Exception as e:
            duration_ms = (time.time() - start_time) * 1000
            self.log_test("Medication Intelligence", False, duration_ms, f"Exception: {str(e)}")
            return False
    
    async def test_flow2_integration(self):
        """Test Flow2 integration endpoint"""
        start_time = time.time()
        try:
            request_data = {
                "request_id": f"test-flow2-{uuid.uuid4()}",
                "patient_id": "test-patient-123",
                "medication_code": "metformin",
                "indication": "diabetes",
                "patient_context": {},
                "clinical_context": {},
                "processing_options": {}
            }
            
            async with self.session.post(
                f"{self.base_url}/api/flow2/execute",
                json=request_data,
                headers={"Content-Type": "application/json"}
            ) as response:
                duration_ms = (time.time() - start_time) * 1000
                
                if response.status == 200:
                    data = await response.json()
                    if (data.get("request_id") == request_data["request_id"] and 
                        data.get("overall_status") == "success" and 
                        data.get("processing_time_ms", 0) > 0):
                        self.log_test("Flow2 Integration", True, duration_ms, 
                                    f"Status: {data.get('overall_status')}")
                        return True
                    else:
                        self.log_test("Flow2 Integration", False, duration_ms, 
                                    f"Invalid response: {data}")
                        return False
                else:
                    self.log_test("Flow2 Integration", False, duration_ms, f"HTTP {response.status}")
                    return False
        except Exception as e:
            duration_ms = (time.time() - start_time) * 1000
            self.log_test("Flow2 Integration", False, duration_ms, f"Exception: {str(e)}")
            return False
    
    async def test_performance_requirements(self):
        """Test sub-100ms performance requirement"""
        print("\n🚀 Testing Performance Requirements (Sub-100ms)...")
        
        # Test multiple requests to get average performance
        durations = []
        success_count = 0
        
        for i in range(10):
            start_time = time.time()
            try:
                request_data = {
                    "request_id": f"perf-test-{i}",
                    "patient_id": "test-patient-123",
                    "medication_code": "metformin",
                    "clinical_parameters": {},
                    "optimization_type": "standard",
                    "clinical_context": {},
                    "processing_hints": {}
                }
                
                async with self.session.post(
                    f"{self.base_url}/api/dose/optimize",
                    json=request_data,
                    headers={"Content-Type": "application/json"}
                ) as response:
                    duration_ms = (time.time() - start_time) * 1000
                    durations.append(duration_ms)
                    
                    if response.status == 200:
                        success_count += 1
                        
            except Exception as e:
                duration_ms = (time.time() - start_time) * 1000
                durations.append(duration_ms)
        
        avg_duration = sum(durations) / len(durations)
        max_duration = max(durations)
        min_duration = min(durations)
        
        # Check if average is under 100ms
        performance_ok = avg_duration < 100
        
        self.log_test("Performance Test", performance_ok, avg_duration, 
                     f"Avg: {avg_duration:.1f}ms, Max: {max_duration:.1f}ms, Min: {min_duration:.1f}ms, Success: {success_count}/10")
        
        return performance_ok
    
    async def test_concurrent_load(self):
        """Test concurrent request handling"""
        print("\n⚡ Testing Concurrent Load (20 simultaneous requests)...")
        
        start_time = time.time()
        
        async def single_request(request_id: str):
            try:
                request_data = {
                    "request_id": request_id,
                    "patient_id": "test-patient-123",
                    "medication_code": "metformin",
                    "clinical_parameters": {},
                    "optimization_type": "standard",
                    "clinical_context": {},
                    "processing_hints": {}
                }
                
                async with self.session.post(
                    f"{self.base_url}/api/dose/optimize",
                    json=request_data,
                    headers={"Content-Type": "application/json"}
                ) as response:
                    return response.status == 200
            except:
                return False
        
        # Create 20 concurrent requests
        tasks = [single_request(f"concurrent-{i}") for i in range(20)]
        results = await asyncio.gather(*tasks)
        
        duration_ms = (time.time() - start_time) * 1000
        success_count = sum(results)
        
        concurrent_ok = success_count >= 18  # Allow 2 failures out of 20
        
        self.log_test("Concurrent Load", concurrent_ok, duration_ms, 
                     f"Success: {success_count}/20 requests")
        
        return concurrent_ok
    
    async def run_all_tests(self):
        """Run all test suites"""
        print("🧪 Starting Unified Dose Safety Engine Test Suite")
        print("=" * 60)
        
        # Basic functionality tests
        print("\n📋 Basic Functionality Tests:")
        await self.test_health_endpoint()
        await self.test_dose_optimization()
        await self.test_medication_intelligence()
        await self.test_flow2_integration()
        
        # Performance tests
        await self.test_performance_requirements()
        await self.test_concurrent_load()
        
        # Summary
        print("\n" + "=" * 60)
        print("📊 Test Summary:")
        
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results if result["success"])
        
        print(f"Total Tests: {total_tests}")
        print(f"Passed: {passed_tests}")
        print(f"Failed: {total_tests - passed_tests}")
        print(f"Success Rate: {(passed_tests/total_tests)*100:.1f}%")
        
        if passed_tests == total_tests:
            print("\n🎉 ALL TESTS PASSED! Unified Engine is ready for production!")
            return True
        else:
            print(f"\n⚠️  {total_tests - passed_tests} tests failed. Please review and fix issues.")
            return False

async def main():
    """Main test runner"""
    print("🚀 Unified Dose Safety Engine - Comprehensive Test Suite")
    print("Testing against: http://localhost:8080")
    print("Make sure the engine is running before starting tests!")
    print()
    
    # Wait a moment for user to confirm
    input("Press Enter to start tests...")
    
    async with UnifiedEngineTestSuite() as test_suite:
        success = await test_suite.run_all_tests()
        
        if success:
            sys.exit(0)
        else:
            sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())
