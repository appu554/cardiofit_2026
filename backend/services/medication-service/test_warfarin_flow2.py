#!/usr/bin/env python3
"""
Test Warfarin Flow2 Integration with Unified Clinical Engine
Tests the working warfarin drug rules with Flow2 requests
"""

import requests
import json
import uuid
from datetime import datetime

# Test configuration
RUST_ENGINE_URL = "http://localhost:8080"
TEST_PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def test_warfarin_flow2_standard():
    """Test standard warfarin initiation scenario"""
    print("🧪 Testing Warfarin Flow2 - Standard Initiation")
    
    request_data = {
        "request_id": str(uuid.uuid4()),
        "patient_id": TEST_PATIENT_ID,
        "action_type": "MEDICATION_ANALYSIS",
        "medication_data": {
            "code": "warfarin",
            "name": "Warfarin",
            "indication": "atrial_fibrillation",
            "route": "oral",
            "urgency": "routine"
        },
        "patient_data": {
            "age_years": 65,
            "weight_kg": 75,
            "height_cm": 170,
            "medical_conditions": ["atrial_fibrillation"],
            "current_medications": []
        },
        "clinical_context": {
            "indication": "stroke_prevention",
            "target_inr": "2.0-3.0",
            "severity": "moderate"
        },
        "processing_hints": {
            "enable_safety_checks": True,
            "include_monitoring": True,
            "priority": "high"
        },
        "priority": "high",
        "enable_ml_inference": True,
        "timeout": 30000,
        "timestamp": "2025-08-15T15:45:00Z"
    }
    
    try:
        response = requests.post(
            f"{RUST_ENGINE_URL}/api/flow2/execute",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        print(f"📊 Status Code: {response.status_code}")
        print(f"📋 Response Headers: {dict(response.headers)}")
        
        if response.status_code == 200:
            result = response.json()
            print("✅ SUCCESS: Warfarin Flow2 request processed!")
            print(f"📄 Response: {json.dumps(result, indent=2)}")
            return True
        else:
            print(f"❌ FAILED: Status {response.status_code}")
            print(f"📄 Error Response: {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ REQUEST ERROR: {e}")
        return False

def test_warfarin_flow2_elderly():
    """Test warfarin for elderly patient scenario"""
    print("\n🧪 Testing Warfarin Flow2 - Elderly Patient")
    
    request_data = {
        "request_id": str(uuid.uuid4()),
        "patient_id": TEST_PATIENT_ID,
        "action_type": "MEDICATION_ANALYSIS",
        "medication_data": {
            "code": "warfarin",
            "name": "Warfarin",
            "indication": "atrial_fibrillation",
            "route": "oral",
            "urgency": "routine"
        },
        "patient_data": {
            "age_years": 85,
            "weight_kg": 65,
            "height_cm": 165,
            "medical_conditions": ["atrial_fibrillation", "hypertension"],
            "current_medications": ["metoprolol", "lisinopril"]
        },
        "clinical_context": {
            "indication": "stroke_prevention",
            "target_inr": "2.0-3.0",
            "severity": "moderate",
            "fall_risk": "high"
        },
        "processing_hints": {
            "enable_safety_checks": True,
            "include_monitoring": True,
            "priority": "high"
        },
        "priority": "high",
        "enable_ml_inference": True,
        "timeout": 30000,
        "timestamp": "2025-08-15T15:45:00Z"
    }
    
    try:
        response = requests.post(
            f"{RUST_ENGINE_URL}/api/flow2/execute",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        print(f"📊 Status Code: {response.status_code}")
        
        if response.status_code == 200:
            result = response.json()
            print("✅ SUCCESS: Elderly warfarin Flow2 request processed!")
            print(f"📄 Response: {json.dumps(result, indent=2)}")
            return True
        else:
            print(f"❌ FAILED: Status {response.status_code}")
            print(f"📄 Error Response: {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ REQUEST ERROR: {e}")
        return False

def test_engine_status():
    """Test engine status and health"""
    print("\n🔍 Testing Engine Status")
    
    try:
        # Test status endpoint
        status_response = requests.get(f"{RUST_ENGINE_URL}/status", timeout=10)
        print(f"📊 Status Endpoint: {status_response.status_code}")
        if status_response.status_code == 200:
            status_data = status_response.json()
            print(f"📄 Status: {json.dumps(status_data, indent=2)}")
        
        # Test health endpoint
        health_response = requests.get(f"{RUST_ENGINE_URL}/health", timeout=10)
        print(f"📊 Health Endpoint: {health_response.status_code}")
        if health_response.status_code == 200:
            health_data = health_response.json()
            print(f"📄 Health: {json.dumps(health_data, indent=2)}")
            
        return True
        
    except requests.exceptions.RequestException as e:
        print(f"❌ ENGINE STATUS ERROR: {e}")
        return False

def main():
    """Run all warfarin Flow2 tests"""
    print("🦀 WARFARIN FLOW2 INTEGRATION TESTS")
    print("=" * 50)
    
    # Test engine status first
    if not test_engine_status():
        print("❌ Engine not available - stopping tests")
        return
    
    # Run warfarin tests
    results = []
    results.append(test_warfarin_flow2_standard())
    results.append(test_warfarin_flow2_elderly())
    
    # Summary
    print("\n" + "=" * 50)
    print("📊 TEST SUMMARY")
    print(f"✅ Passed: {sum(results)}")
    print(f"❌ Failed: {len(results) - sum(results)}")
    print(f"📈 Success Rate: {sum(results)/len(results)*100:.1f}%")
    
    if all(results):
        print("🎉 ALL TESTS PASSED! Warfarin Flow2 integration working!")
    else:
        print("⚠️  Some tests failed - check logs above")

if __name__ == "__main__":
    main()
