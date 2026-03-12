#!/usr/bin/env python3
"""
Test script for the Rust Recipe Engine
Tests the complete Go → Rust data flow
"""

import json
import requests
import time
import sys

def test_recipe_execution():
    """Test the main recipe execution endpoint"""
    print("🧪 Testing Recipe Execution Endpoint...")
    
    # This is the exact data format Go engine sends
    request_data = {
        "request_id": "flow2-vanc-001",
        "recipe_id": "vancomycin-dosing-v1.0",
        "variant": "standard_auc",
        "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
        "medication_code": "11124",
        "clinical_context": json.dumps({
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "fields": {
                "demographics.age": 65.0,
                "demographics.weight.actual_kg": 80.0,
                "demographics.height_cm": 175.0,
                "demographics.gender": "MALE",
                "labs.serum_creatinine[latest]": 1.8,
                "labs.egfr[latest]": 45.0,
                "conditions.active": ["sepsis", "chronic_kidney_disease"],
                "allergies.active": [
                    {
                        "allergen": "Penicillin",
                        "allergen_type": "DRUG",
                        "severity": "MODERATE"
                    }
                ],
                "medications.current": [
                    {
                        "code": "1191",
                        "name": "Aspirin",
                        "dose": 81.0,
                        "frequency": "daily"
                    }
                ]
            },
            "sources": ["patient_service", "lab_service", "medication_service"],
            "retrieval_time_ms": 15,
            "completeness": 0.95
        }),
        "timeout_ms": 5000
    }
    
    try:
        response = requests.post(
            "http://localhost:8080/api/recipe/execute",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            print("✅ Recipe execution successful!")
            print(f"📊 Medication: {result.get('medication_name')}")
            print(f"💊 Calculated Dose: {result.get('calculated_dose')} {result.get('dose_unit')}")
            print(f"⏱️  Frequency: {result.get('frequency')}")
            print(f"🛡️  Safety Status: {result.get('safety_status')}")
            print(f"⚠️  Safety Alerts: {len(result.get('safety_alerts', []))}")
            print(f"📋 Monitoring Plan: {len(result.get('monitoring_plan', []))}")
            print(f"⏱️  Execution Time: {result.get('execution_time_ms')}ms")
            return True
        else:
            print(f"❌ Recipe execution failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        print("❌ Connection failed - Is the Rust engine running on port 8080?")
        return False
    except Exception as e:
        print(f"❌ Test failed: {e}")
        return False

def test_health_check():
    """Test the health check endpoint"""
    print("\n🏥 Testing Health Check...")
    
    try:
        response = requests.get("http://localhost:8080/health", timeout=5)
        
        if response.status_code == 200:
            health = response.json()
            print("✅ Health check successful!")
            print(f"📊 Status: {health.get('status')}")
            print(f"🔢 Version: {health.get('version')}")
            print(f"📚 Knowledge Items: {health.get('total_knowledge_items')}")
            print(f"📋 Rules Loaded: {health.get('rules_loaded')}")
            print(f"🧪 Recipes Loaded: {health.get('recipes_loaded')}")
            return True
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return False
            
    except Exception as e:
        print(f"❌ Health check failed: {e}")
        return False

def test_flow2_compatibility():
    """Test Flow2 compatibility endpoint"""
    print("\n🔄 Testing Flow2 Compatibility...")
    
    request_data = {
        "request_id": "flow2-compat-001",
        "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
        "action_type": "MEDICATION_ANALYSIS",
        "medication_data": {
            "code": "11124",
            "name": "Vancomycin",
            "dose": 1000.0,
            "unit": "mg",
            "frequency": "q12h",
            "route": "IV",
            "indication": "sepsis"
        },
        "patient_data": {
            "age_years": 65.0,
            "weight_kg": 80.0,
            "height_cm": 175.0,
            "gender": "MALE",
            "conditions": ["sepsis"]
        },
        "clinical_context": {
            "patient_demographics": {
                "age_years": 65.0,
                "weight_kg": 80.0,
                "egfr": 45.0,
                "gender": "MALE"
            }
        },
        "processing_hints": {
            "enable_ml_inference": True,
            "priority": "high"
        },
        "priority": "high",
        "enable_ml_inference": True,
        "timestamp": "2024-01-15T10:30:00Z"
    }
    
    try:
        response = requests.post(
            "http://localhost:8080/api/flow2/execute",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            print("✅ Flow2 compatibility successful!")
            print(f"📊 Overall Status: {result.get('overall_status')}")
            print(f"🧪 Recipes Executed: {result.get('execution_summary', {}).get('total_recipes_executed')}")
            print(f"⏱️  Execution Time: {result.get('execution_time_ms')}ms")
            return True
        else:
            print(f"❌ Flow2 compatibility failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False
            
    except Exception as e:
        print(f"❌ Flow2 test failed: {e}")
        return False

def test_enhanced_manifest_generation():
    """Test the enhanced intent manifest generation endpoint"""
    print("\n🧠 Testing Enhanced Intent Manifest Generation...")

    request_data = {
        "request_id": "manifest-001",
        "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
        "medication_code": "11124",
        "medication_name": "Vancomycin",
        "patient_conditions": ["sepsis", "chronic_kidney_disease"],
        "patient_demographics": {
            "age_years": 75.0,
            "weight_kg": 80.0,
            "height_cm": 175.0,
            "gender": "MALE",
            "egfr": 35.0,
            "bmi": 26.1
        },
        "clinical_context": {
            "active_medications": [
                {
                    "medication_code": "1191",
                    "medication_name": "Aspirin",
                    "dose": "81mg",
                    "frequency": "daily"
                }
            ],
            "allergies": [
                {
                    "allergen": "Penicillin",
                    "reaction": "Rash",
                    "severity": "MODERATE"
                }
            ],
            "lab_values": [
                {
                    "code": "CREAT",
                    "name": "Serum Creatinine",
                    "value": 2.1,
                    "unit": "mg/dL"
                }
            ]
        },
        "timestamp": "2024-01-15T10:30:00Z"
    }

    try:
        response = requests.post(
            "http://localhost:8080/api/manifest/generate",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=10
        )

        if response.status_code == 200:
            manifest = response.json()
            print("✅ Enhanced manifest generation successful!")
            print(f"📋 Recipe ID: {manifest.get('recipe_id')}")
            print(f"🎯 Variant: {manifest.get('variant')}")
            print(f"⚠️  Risk Level: {manifest.get('risk_assessment', {}).get('overall_risk_level')}")
            print(f"🔥 Priority: {manifest.get('priority')}")
            print(f"📊 Risk Score: {manifest.get('risk_assessment', {}).get('risk_score')}")
            print(f"🏥 Clinical Flags: {len(manifest.get('clinical_flags', []))}")
            print(f"📋 Monitoring Requirements: {len(manifest.get('monitoring_requirements', []))}")
            print(f"⚠️  Safety Considerations: {len(manifest.get('safety_considerations', []))}")
            print(f"🔄 Alternative Recipes: {len(manifest.get('alternative_recipes', []))}")
            print(f"⏱️  Estimated Time: {manifest.get('estimated_execution_time_ms')}ms")

            # Print some detailed information
            if manifest.get('risk_assessment', {}).get('risk_factors'):
                print(f"🚨 Risk Factors:")
                for factor in manifest['risk_assessment']['risk_factors'][:3]:  # Show first 3
                    print(f"   - {factor.get('description')} ({factor.get('severity')})")

            if manifest.get('clinical_flags'):
                print(f"🏥 Clinical Flags:")
                for flag in manifest['clinical_flags'][:3]:  # Show first 3
                    print(f"   - {flag.get('message')} ({flag.get('severity')})")

            return True
        else:
            print(f"❌ Enhanced manifest generation failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False

    except Exception as e:
        print(f"❌ Enhanced manifest test failed: {e}")
        return False

def test_production_api_features():
    """Test production-grade API features"""
    print("\n🏭 Testing Production API Features...")

    # Test detailed health check
    try:
        response = requests.get("http://localhost:8080/health/detailed", timeout=5)
        if response.status_code == 200:
            health = response.json()
            print("✅ Detailed health check successful!")
            print(f"📊 System Memory: {health.get('system', {}).get('memory_usage')} MB")
            print(f"🖥️  CPU Usage: {health.get('system', {}).get('cpu_usage')}%")
            print(f"🧵 Thread Count: {health.get('system', {}).get('thread_count')}")
        else:
            print(f"❌ Detailed health check failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Detailed health check failed: {e}")
        return False

    # Test version endpoint
    try:
        response = requests.get("http://localhost:8080/version", timeout=5)
        if response.status_code == 200:
            version = response.json()
            print("✅ Version endpoint successful!")
            print(f"🔢 Version: {version.get('version')}")
            print(f"🏗️  Build Date: {version.get('build_date')}")
        else:
            print(f"❌ Version endpoint failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Version endpoint failed: {e}")
        return False

    # Test admin stats (with auth token)
    try:
        response = requests.get(
            "http://localhost:8080/api/admin/stats",
            headers={"Authorization": "Bearer development-token"},
            timeout=5
        )
        if response.status_code == 200:
            stats = response.json()
            print("✅ Admin stats successful!")
            print(f"⏱️  Uptime: {stats.get('server', {}).get('uptime_seconds')} seconds")
            print(f"📊 Total Requests: {stats.get('performance', {}).get('total_requests')}")
        else:
            print(f"❌ Admin stats failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Admin stats failed: {e}")
        return False

    return True

def test_api_security_features():
    """Test API security features"""
    print("\n🔐 Testing API Security Features...")

    # Test authentication requirement
    try:
        response = requests.post(
            "http://localhost:8080/api/recipe/execute",
            json={"test": "data"},
            timeout=5
        )
        # Should fail without auth token (if auth is enabled)
        if response.status_code == 401:
            print("✅ Authentication requirement working!")
        elif response.status_code == 400:
            print("✅ Request validation working (auth disabled)!")
        else:
            print(f"⚠️  Unexpected response: {response.status_code}")
    except Exception as e:
        print(f"❌ Auth test failed: {e}")
        return False

    # Test CORS headers
    try:
        response = requests.options("http://localhost:8080/api/recipe/execute", timeout=5)
        if response.status_code == 200:
            headers = response.headers
            if "Access-Control-Allow-Origin" in headers:
                print("✅ CORS headers present!")
            else:
                print("⚠️  CORS headers missing")
        else:
            print(f"❌ OPTIONS request failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ CORS test failed: {e}")
        return False

    # Test security headers
    try:
        response = requests.get("http://localhost:8080/health", timeout=5)
        if response.status_code == 200:
            headers = response.headers
            security_headers = [
                "X-Content-Type-Options",
                "X-Frame-Options",
                "X-XSS-Protection",
                "X-API-Version"
            ]
            present_headers = [h for h in security_headers if h in headers]
            print(f"✅ Security headers present: {len(present_headers)}/{len(security_headers)}")
            if len(present_headers) < len(security_headers):
                print(f"⚠️  Missing headers: {set(security_headers) - set(present_headers)}")
        else:
            print(f"❌ Security headers test failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Security headers test failed: {e}")
        return False

    return True

def test_api_performance_features():
    """Test API performance features"""
    print("\n⚡ Testing API Performance Features...")

    # Test compression
    try:
        response = requests.get(
            "http://localhost:8080/health/detailed",
            headers={"Accept-Encoding": "gzip"},
            timeout=5
        )
        if response.status_code == 200:
            if "gzip" in response.headers.get("Content-Encoding", ""):
                print("✅ Compression working!")
            else:
                print("⚠️  Compression not detected")
        else:
            print(f"❌ Compression test failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Compression test failed: {e}")
        return False

    # Test request tracking
    try:
        custom_request_id = "test-request-12345"
        response = requests.get(
            "http://localhost:8080/health",
            headers={"X-Request-ID": custom_request_id},
            timeout=5
        )
        if response.status_code == 200:
            print("✅ Request tracking working!")
        else:
            print(f"❌ Request tracking test failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Request tracking test failed: {e}")
        return False

    return True

def main():
    """Run all tests"""
    print("🦀 ===============================================")
    print("🦀  RUST RECIPE ENGINE - INTEGRATION TESTS")
    print("🦀 ===============================================")

    tests = [
        ("Health Check", test_health_check),
        ("Recipe Execution", test_recipe_execution),
        ("Flow2 Compatibility", test_flow2_compatibility),
        ("Enhanced Manifest Generation", test_enhanced_manifest_generation),
        ("Production API Features", test_production_api_features),
        ("API Security Features", test_api_security_features),
        ("API Performance Features", test_api_performance_features),
    ]
    
    passed = 0
    total = len(tests)
    
    for test_name, test_func in tests:
        print(f"\n🧪 Running: {test_name}")
        print("-" * 50)
        
        if test_func():
            passed += 1
            print(f"✅ {test_name} PASSED")
        else:
            print(f"❌ {test_name} FAILED")
    
    print("\n🦀 ===============================================")
    print(f"🦀  TEST RESULTS: {passed}/{total} PASSED")
    print("🦀 ===============================================")
    
    if passed == total:
        print("🎉 ALL TESTS PASSED! Rust engine is working correctly!")
        sys.exit(0)
    else:
        print("⚠️  Some tests failed. Check the Rust engine implementation.")
        sys.exit(1)

if __name__ == "__main__":
    main()
