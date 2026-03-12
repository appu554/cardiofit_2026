#!/usr/bin/env python3
"""
Flow 2 Go Engine Test Script

This script tests the Go Enhanced Orchestrator to ensure it's working correctly
with the mock Rust engine during parallel development.

Usage:
    python test_flow2_go_engine.py
"""

import requests
import json
import time
from datetime import datetime

# Configuration
GO_ENGINE_URL = "http://localhost:8080"
TEST_PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def test_health_check():
    """Test the health check endpoint"""
    print("🏥 Testing health check...")
    
    try:
        response = requests.get(f"{GO_ENGINE_URL}/health", timeout=5)
        if response.status_code == 200:
            print("✅ Health check passed")
            return True
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ Health check failed: {e}")
        return False

def test_flow2_execution():
    """Test the main Flow 2 execution endpoint"""
    print("\n🧪 Testing Flow 2 execution...")
    
    # Test request
    request_data = {
        "patient_id": TEST_PATIENT_ID,
        "action_type": "PROPOSE_MEDICATION",
        "medication_data": {
            "medications": [
                {
                    "code": "acetaminophen",
                    "name": "Acetaminophen",
                    "dose": 500.0,
                    "unit": "mg",
                    "frequency": "twice daily",
                    "route": "oral",
                    "indication": "pain relief"
                }
            ]
        },
        "patient_data": {
            "weight_kg": 70.0,
            "age_years": 45.0,
            "height_cm": 175.0
        },
        "clinical_context": {
            "allergies": [],
            "conditions": [],
            "current_medications": []
        },
        "processing_hints": {
            "priority": "normal",
            "enable_ml_inference": True
        },
        "enable_ml_inference": True
    }
    
    try:
        start_time = time.time()
        response = requests.post(
            f"{GO_ENGINE_URL}/api/v1/flow2/execute",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        execution_time = (time.time() - start_time) * 1000  # Convert to ms
        
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Flow 2 execution successful")
            print(f"   Execution time: {execution_time:.1f}ms")
            print(f"   Overall status: {result.get('overall_status')}")
            print(f"   Recipes executed: {len(result.get('recipe_results', []))}")
            print(f"   Engine used: {result.get('engine_used')}")
            
            # Print recipe results
            for recipe in result.get('recipe_results', []):
                print(f"   Recipe: {recipe.get('recipe_name')} - {recipe.get('overall_status')}")
            
            return True
        else:
            print(f"❌ Flow 2 execution failed: {response.status_code}")
            print(f"   Response: {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Flow 2 execution failed: {e}")
        return False

def test_medication_intelligence():
    """Test the medication intelligence endpoint"""
    print("\n🧠 Testing medication intelligence...")
    
    request_data = {
        "patient_id": TEST_PATIENT_ID,
        "medications": [
            {
                "code": "warfarin",
                "name": "Warfarin",
                "dose": 5.0,
                "unit": "mg",
                "frequency": "daily"
            },
            {
                "code": "acetaminophen",
                "name": "Acetaminophen",
                "dose": 500.0,
                "unit": "mg",
                "frequency": "twice daily"
            }
        ],
        "intelligence_type": "comprehensive",
        "analysis_depth": "detailed",
        "include_predictions": True,
        "include_alternatives": True
    }
    
    try:
        start_time = time.time()
        response = requests.post(
            f"{GO_ENGINE_URL}/api/v1/flow2/medication-intelligence",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        execution_time = (time.time() - start_time) * 1000
        
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Medication intelligence successful")
            print(f"   Execution time: {execution_time:.1f}ms")
            print(f"   Intelligence score: {result.get('intelligence_score')}")
            return True
        else:
            print(f"❌ Medication intelligence failed: {response.status_code}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Medication intelligence failed: {e}")
        return False

def test_dose_optimization():
    """Test the dose optimization endpoint"""
    print("\n💊 Testing dose optimization...")
    
    request_data = {
        "patient_id": TEST_PATIENT_ID,
        "medication_code": "warfarin",
        "clinical_parameters": {
            "weight_kg": 70.0,
            "age_years": 65.0,
            "creatinine_clearance": 80.0,
            "target_inr": 2.5
        },
        "optimization_goals": [
            "minimize_bleeding_risk",
            "achieve_target_inr",
            "minimize_dose_changes"
        ]
    }
    
    try:
        start_time = time.time()
        response = requests.post(
            f"{GO_ENGINE_URL}/api/v1/flow2/dose-optimization",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        execution_time = (time.time() - start_time) * 1000
        
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Dose optimization successful")
            print(f"   Execution time: {execution_time:.1f}ms")
            print(f"   Optimized dose: {result.get('optimized_dose')}mg")
            print(f"   Optimization score: {result.get('optimization_score')}")
            return True
        else:
            print(f"❌ Dose optimization failed: {response.status_code}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Dose optimization failed: {e}")
        return False

def test_safety_validation():
    """Test the safety validation endpoint"""
    print("\n🛡️  Testing safety validation...")
    
    request_data = {
        "patient_id": TEST_PATIENT_ID,
        "medications": [
            {
                "code": "warfarin",
                "name": "Warfarin",
                "dose": 5.0,
                "unit": "mg",
                "frequency": "daily"
            },
            {
                "code": "aspirin",
                "name": "Aspirin",
                "dose": 81.0,
                "unit": "mg",
                "frequency": "daily"
            }
        ],
        "clinical_context": {
            "allergies": [],
            "conditions": ["atrial_fibrillation"],
            "lab_values": {
                "inr": 2.1,
                "creatinine": 1.0
            }
        },
        "validation_level": "comprehensive"
    }
    
    try:
        start_time = time.time()
        response = requests.post(
            f"{GO_ENGINE_URL}/api/v1/flow2/safety-validation",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        execution_time = (time.time() - start_time) * 1000
        
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Safety validation successful")
            print(f"   Execution time: {execution_time:.1f}ms")
            print(f"   Safety status: {result.get('overall_safety_status')}")
            print(f"   Safety score: {result.get('safety_score')}")
            print(f"   Drug interactions: {len(result.get('drug_interactions', []))}")
            return True
        else:
            print(f"❌ Safety validation failed: {response.status_code}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Safety validation failed: {e}")
        return False

def test_metrics_endpoint():
    """Test the metrics endpoint"""
    print("\n📊 Testing metrics endpoint...")
    
    try:
        response = requests.get(f"{GO_ENGINE_URL}/metrics", timeout=5)
        if response.status_code == 200:
            print("✅ Metrics endpoint accessible")
            # Check if it contains Prometheus metrics
            if "# HELP" in response.text:
                print("   Prometheus metrics format detected")
            return True
        else:
            print(f"❌ Metrics endpoint failed: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ Metrics endpoint failed: {e}")
        return False

def run_performance_test():
    """Run a simple performance test"""
    print("\n⚡ Running performance test...")
    
    request_data = {
        "patient_id": TEST_PATIENT_ID,
        "action_type": "PROPOSE_MEDICATION",
        "medication_data": {
            "medications": [
                {
                    "code": "acetaminophen",
                    "name": "Acetaminophen",
                    "dose": 500.0,
                    "unit": "mg",
                    "frequency": "twice daily"
                }
            ]
        },
        "patient_data": {
            "weight_kg": 70.0,
            "age_years": 45.0
        }
    }
    
    # Run 10 requests and measure performance
    execution_times = []
    successful_requests = 0
    
    for i in range(10):
        try:
            start_time = time.time()
            response = requests.post(
                f"{GO_ENGINE_URL}/api/v1/flow2/execute",
                json=request_data,
                headers={"Content-Type": "application/json"},
                timeout=10
            )
            execution_time = (time.time() - start_time) * 1000
            
            if response.status_code == 200:
                execution_times.append(execution_time)
                successful_requests += 1
            
        except requests.exceptions.RequestException:
            pass
    
    if execution_times:
        avg_time = sum(execution_times) / len(execution_times)
        min_time = min(execution_times)
        max_time = max(execution_times)
        
        print(f"✅ Performance test completed")
        print(f"   Successful requests: {successful_requests}/10")
        print(f"   Average time: {avg_time:.1f}ms")
        print(f"   Min time: {min_time:.1f}ms")
        print(f"   Max time: {max_time:.1f}ms")
        
        # Check if we meet performance targets
        if avg_time < 50:  # Target: <50ms
            print("   🎯 Performance target met (<50ms average)")
        else:
            print("   ⚠️  Performance target not met (>50ms average)")
        
        return True
    else:
        print("❌ Performance test failed - no successful requests")
        return False

def main():
    print("🧪 Flow 2 Go Engine Test Suite")
    print("=" * 50)
    print(f"Testing Go Engine at: {GO_ENGINE_URL}")
    print(f"Test Patient ID: {TEST_PATIENT_ID}")
    print(f"Test started at: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print()
    print("⚠️  IMPORTANT: This test requires REAL services:")
    print("   • Go Engine must be connected to real Rust Engine")
    print("   • Redis must be running for caching")
    print("   • All dependencies must be healthy")
    print()
    
    # Run all tests
    tests = [
        ("Health Check", test_health_check),
        ("Flow 2 Execution", test_flow2_execution),
        ("Medication Intelligence", test_medication_intelligence),
        ("Dose Optimization", test_dose_optimization),
        ("Safety Validation", test_safety_validation),
        ("Metrics Endpoint", test_metrics_endpoint),
        ("Performance Test", run_performance_test),
    ]
    
    passed_tests = 0
    total_tests = len(tests)
    
    for test_name, test_func in tests:
        try:
            if test_func():
                passed_tests += 1
        except Exception as e:
            print(f"❌ {test_name} failed with exception: {e}")
    
    # Summary
    print("\n" + "=" * 50)
    print(f"📊 Test Summary: {passed_tests}/{total_tests} tests passed")
    
    if passed_tests == total_tests:
        print("🎉 All tests passed! Go Engine is working correctly.")
        return 0
    else:
        print("⚠️  Some tests failed. Check the output above for details.")
        return 1

if __name__ == "__main__":
    exit(main())
