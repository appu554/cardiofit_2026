#!/usr/bin/env python3
"""
Test Vancomycin ORB Rule Integration with Flow2
Tests the bridge between TOML drug rules and ORB rules for Flow2 integration
"""

import requests
import json
import sys
import uuid
from datetime import datetime

# Test configuration
RUST_ENGINE_URL = "http://localhost:8080"
TEST_PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def test_flow2_vancomycin_standard():
    """Test Flow2 integration with vancomycin standard dosing"""
    print("\n🔄 Testing Flow2 Vancomycin Standard Dosing...")
    
    # Payload that should match vancomycin-standard-selection-v1 ORB rule
    payload = {
        "request_id": str(uuid.uuid4()),
        "patient_id": TEST_PATIENT_ID,
        "action_type": "MEDICATION_ANALYSIS",
        "medication_data": {
            "code": "vancomycin",  # Use 'code' not 'drug_id'
            "name": "Vancomycin",  # Provide explicit name for ORB rule matching
            "indication": "severe_infection",
            "route": "iv",
            "urgency": "routine"
        },
        "patient_data": {
            "age_years": 55,
            "weight_kg": 75,
            "height_cm": 170,
            "creatinine_clearance": 80,
            "medical_conditions": ["pneumonia"],
            "current_medications": []
        },
        "clinical_context": {
            "indication": "hospital_acquired_pneumonia",
            "severity": "moderate",
            "culture_results": "pending"
        },
        "processing_hints": {
            "priority": "high",
            "enable_safety_checks": True,
            "include_monitoring": True
        },
        "priority": "high",
        "enable_ml_inference": True,
        "timeout": 30000,
        "timestamp": "2025-08-15T15:45:00Z"
    }
    
    try:
        response = requests.post(f"{RUST_ENGINE_URL}/api/flow2/execute", json=payload)
        if response.status_code == 200:
            data = response.json()
            print("✅ Flow2 vancomycin standard dosing successful")
            print(f"📋 Execution status: {data.get('overall_status', 'unknown')}")
            print(f"💊 Recipe ID: {data.get('execution_summary', {}).get('recipe_id', 'N/A')}")
            print(f"🎯 Variant: {data.get('execution_summary', {}).get('variant', 'N/A')}")
            print(f"📊 Data requirements: {len(data.get('execution_summary', {}).get('data_requirements', []))}")
            return True
        else:
            print(f"❌ Flow2 vancomycin standard failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False
    except Exception as e:
        print(f"❌ Flow2 vancomycin standard error: {e}")
        return False

def test_flow2_vancomycin_dialysis():
    """Test Flow2 integration with vancomycin dialysis dosing"""
    print("\n🔄 Testing Flow2 Vancomycin Dialysis Dosing...")
    
    # Payload that should match vancomycin-dialysis-selection-v1 ORB rule
    payload = {
        "request_id": str(uuid.uuid4()),
        "patient_id": TEST_PATIENT_ID,
        "action_type": "MEDICATION_ANALYSIS",
        "medication_data": {
            "code": "vancomycin",
            "name": "Vancomycin",  # Exact match for ORB rule
            "indication": "severe_infection",
            "route": "iv",
            "urgency": "routine"
        },
        "patient_data": {
            "age_years": 65,
            "weight_kg": 80,
            "height_cm": 175,
            "creatinine_clearance": 15,  # Severe renal impairment
            "medical_conditions": ["esrd", "hemodialysis"],  # Should trigger dialysis rule
            "dialysis_status": "hemodialysis",  # Explicit dialysis status
            "current_medications": []
        },
        "clinical_context": {
            "indication": "sepsis",
            "severity": "severe",
            "dialysis_schedule": "monday_wednesday_friday"
        },
        "processing_hints": {
            "priority": "high",
            "enable_safety_checks": True,
            "include_monitoring": True
        },
        "priority": "high",
        "enable_ml_inference": True,
        "timeout": 30000,
        "timestamp": "2025-08-15T15:45:00Z"
    }
    
    try:
        response = requests.post(f"{RUST_ENGINE_URL}/api/flow2/execute", json=payload)
        if response.status_code == 200:
            data = response.json()
            print("✅ Flow2 vancomycin dialysis dosing successful")
            print(f"📋 Execution status: {data.get('overall_status', 'unknown')}")
            print(f"💊 Recipe ID: {data.get('execution_summary', {}).get('recipe_id', 'N/A')}")
            print(f"🎯 Variant: {data.get('execution_summary', {}).get('variant', 'N/A')}")
            print(f"📊 Data requirements: {len(data.get('execution_summary', {}).get('data_requirements', []))}")
            return True
        else:
            print(f"❌ Flow2 vancomycin dialysis failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False
    except Exception as e:
        print(f"❌ Flow2 vancomycin dialysis error: {e}")
        return False

def test_orb_rule_loading():
    """Test if ORB rules are properly loaded"""
    print("\n📋 Testing ORB Rule Loading...")
    
    try:
        response = requests.get(f"{RUST_ENGINE_URL}/status")
        if response.status_code == 200:
            data = response.json()
            print("✅ Engine status retrieved")
            print(f"📊 Total knowledge items: {data.get('knowledge_base', {}).get('total_items', 0)}")
            print(f"💊 Drug rules: {data.get('knowledge_base', {}).get('drug_rules', 0)}")
            print(f"⚠️ DDI rules: {data.get('knowledge_base', {}).get('ddi_rules', 0)}")
            print(f"📋 ORB rules: {data.get('knowledge_base', {}).get('orb_rules', 0)}")
            return True
        else:
            print(f"❌ Status check failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Status check error: {e}")
        return False

def main():
    """Run vancomycin ORB rule tests"""
    print("🧪 VANCOMYCIN ORB RULE INTEGRATION TEST")
    print("=" * 60)
    print(f"🕐 Test started at: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"🌐 Testing against: {RUST_ENGINE_URL}")
    print(f"👤 Test patient ID: {TEST_PATIENT_ID}")
    print("=" * 60)
    
    tests = [
        ("ORB Rule Loading", test_orb_rule_loading),
        ("Flow2 Vancomycin Standard", test_flow2_vancomycin_standard),
        ("Flow2 Vancomycin Dialysis", test_flow2_vancomycin_dialysis),
    ]
    
    passed = 0
    total = len(tests)
    
    for test_name, test_func in tests:
        try:
            if test_func():
                passed += 1
            else:
                print(f"❌ {test_name} failed")
        except Exception as e:
            print(f"❌ {test_name} crashed: {e}")
    
    print("\n" + "=" * 60)
    print(f"🧪 TEST RESULTS: {passed}/{total} tests passed")
    if passed == total:
        print("🎉 ALL TESTS PASSED! ORB rule integration is working!")
    else:
        print(f"⚠️ {total - passed} tests failed. Check the output above for details.")
    print("=" * 60)
    
    return passed == total

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
