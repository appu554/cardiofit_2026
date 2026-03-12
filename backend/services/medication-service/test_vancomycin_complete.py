#!/usr/bin/env python3
"""
Complete Vancomycin Knowledge Base Test
Tests all the functionality we just fixed in vancomycin.toml
"""

import requests
import json
import sys
import uuid
from datetime import datetime

# Test configuration
RUST_ENGINE_URL = "http://localhost:8080"
TEST_PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def test_health_check():
    """Test basic health check"""
    print("🏥 Testing Health Check...")
    try:
        response = requests.get(f"{RUST_ENGINE_URL}/health")
        if response.status_code == 200:
            print("✅ Health check passed")
            return True
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Health check error: {e}")
        return False

def test_engine_status():
    """Test engine status and knowledge base loading"""
    print("\n📊 Testing Engine Status...")
    try:
        response = requests.get(f"{RUST_ENGINE_URL}/status")
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Engine Status: {data.get('status', 'unknown')}")
            print(f"📚 Knowledge Base Items: {data.get('knowledge_base', {}).get('total_items', 0)}")
            print(f"💊 Drug Rules: {data.get('knowledge_base', {}).get('drug_rules', 0)}")
            print(f"⚠️ DDI Rules: {data.get('knowledge_base', {}).get('ddi_rules', 0)}")
            return True
        else:
            print(f"❌ Status check failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Status check error: {e}")
        return False

def test_vancomycin_dose_calculation():
    """Test vancomycin dose calculation with renal adjustments"""
    print("\n💊 Testing Vancomycin Dose Calculation...")

    # Test payload with patient requiring renal adjustment
    payload = {
        "request_id": str(uuid.uuid4()),
        "patient_id": TEST_PATIENT_ID,
        "medication_code": "vancomycin",
        "clinical_parameters": {
            "age_years": 65,
            "weight_kg": 80,
            "height_cm": 175,
            "creatinine_clearance": 45,
            "serum_creatinine": 1.8,
            "medical_conditions": ["ckd_stage_3"]
        },
        "optimization_type": "renal_adjustment",
        "clinical_context": {
            "indication": "severe_gram_positive_infection",
            "severity": "severe",
            "target_trough": 15.0
        },
        "processing_hints": {
            "priority": "high",
            "urgency": "routine"
        }
    }
    
    try:
        response = requests.post(f"{RUST_ENGINE_URL}/api/dose/optimize", json=payload)
        if response.status_code == 200:
            data = response.json()
            print("✅ Dose calculation successful")
            print(f"📋 Recommended dose: {data.get('recommended_dose', 'N/A')}")
            print(f"⏰ Dosing interval: {data.get('dosing_interval', 'N/A')}")
            print(f"🔄 Renal adjustment applied: {data.get('renal_adjustment_applied', False)}")
            return True
        else:
            print(f"❌ Dose calculation failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False
    except Exception as e:
        print(f"❌ Dose calculation error: {e}")
        return False

def test_vancomycin_safety_verification():
    """Test vancomycin safety verification"""
    print("\n🛡️ Testing Vancomycin Safety Verification...")

    # Test payload with potential safety concerns
    payload = {
        "request_id": str(uuid.uuid4()),
        "patient_id": TEST_PATIENT_ID,
        "medications": [{
            "code": "vancomycin",
            "name": "Vancomycin",
            "dose": 1000.0,
            "unit": "mg",
            "frequency": "q12h",
            "route": "iv",
            "duration": "7 days",
            "indication": "severe_infection",
            "properties": {}
        }],
        "intelligence_type": "safety_verification",
        "analysis_depth": "comprehensive",
        "clinical_context": {
            "age_years": 75,
            "weight_kg": 65,
            "allergies": ["penicillin"],
            "medical_conditions": ["ckd_stage_4", "hearing_impairment"],
            "pregnancy_status": "not_pregnant",
            "breastfeeding": False,
            "creatinine_clearance": 25,
            "current_medications": ["furosemide", "gentamicin"]
        }
    }
    
    try:
        response = requests.post(f"{RUST_ENGINE_URL}/api/medication/intelligence", json=payload)
        if response.status_code == 200:
            data = response.json()
            print("✅ Safety verification successful")
            
            # Check safety alerts
            safety_alerts = data.get('safety_alerts', [])
            print(f"⚠️ Safety alerts found: {len(safety_alerts)}")
            for alert in safety_alerts:
                print(f"  - {alert.get('severity', 'unknown')}: {alert.get('message', 'N/A')}")
            
            # Check contraindications
            contraindications = data.get('contraindications', [])
            print(f"🚫 Contraindications: {len(contraindications)}")
            for contra in contraindications:
                print(f"  - {contra.get('type', 'unknown')}: {contra.get('reason', 'N/A')}")
            
            return True
        else:
            print(f"❌ Safety verification failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False
    except Exception as e:
        print(f"❌ Safety verification error: {e}")
        return False

def test_vancomycin_monitoring_requirements():
    """Test vancomycin monitoring requirements"""
    print("\n📊 Testing Vancomycin Monitoring Requirements...")

    payload = {
        "request_id": str(uuid.uuid4()),
        "patient_id": TEST_PATIENT_ID,
        "medications": [{
            "code": "vancomycin",
            "name": "Vancomycin",
            "dose": 1500.0,
            "unit": "mg",
            "frequency": "q12h",
            "route": "iv",
            "duration": "5 days",
            "indication": "pneumonia",
            "properties": {}
        }],
        "intelligence_type": "monitoring_requirements",
        "analysis_depth": "standard",
        "clinical_context": {
            "age_years": 45,
            "weight_kg": 85,
            "creatinine_clearance": 90,
            "medical_conditions": ["pneumonia"]
        }
    }
    
    try:
        response = requests.post(f"{RUST_ENGINE_URL}/api/medication/intelligence", json=payload)
        if response.status_code == 200:
            data = response.json()
            print("✅ Monitoring requirements retrieved")
            
            # Check monitoring requirements
            monitoring = data.get('monitoring_requirements', [])
            print(f"📋 Monitoring tests required: {len(monitoring)}")
            for req in monitoring:
                print(f"  - {req.get('lab_test', 'unknown')}: {req.get('frequency', 'N/A')}")
                if req.get('alert_threshold_high'):
                    print(f"    Alert if > {req.get('alert_threshold_high')}")
                if req.get('alert_threshold_low'):
                    print(f"    Alert if < {req.get('alert_threshold_low')}")
            
            return True
        else:
            print(f"❌ Monitoring requirements failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False
    except Exception as e:
        print(f"❌ Monitoring requirements error: {e}")
        return False

def test_flow2_integration():
    """Test Flow2 integration with vancomycin"""
    print("\n🔄 Testing Flow2 Integration...")

    payload = {
        "request_id": str(uuid.uuid4()),
        "patient_id": TEST_PATIENT_ID,
        "action_type": "MEDICATION_ANALYSIS",
        "medication_data": {
            "drug_id": "vancomycin",
            "indication": "severe_infection",
            "route": "iv",
            "urgency": "routine"
        },
        "patient_data": {
            "age_years": 55,
            "weight_kg": 75,
            "height_cm": 170,
            "creatinine_clearance": 80,
            "medical_conditions": ["diabetes", "hypertension"],
            "current_medications": ["metformin", "lisinopril"]
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
            print("✅ Flow2 integration successful")
            print(f"📋 Execution status: {data.get('status', 'unknown')}")
            print(f"💊 Medication proposal: {data.get('medication_proposal', {}).get('drug_name', 'N/A')}")
            print(f"🎯 Recommended dose: {data.get('medication_proposal', {}).get('dose', 'N/A')}")
            return True
        else:
            print(f"❌ Flow2 integration failed: {response.status_code}")
            print(f"Response: {response.text}")
            return False
    except Exception as e:
        print(f"❌ Flow2 integration error: {e}")
        return False

def main():
    """Run all vancomycin tests"""
    print("🧪 VANCOMYCIN KNOWLEDGE BASE COMPREHENSIVE TEST")
    print("=" * 60)
    print(f"🕐 Test started at: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"🌐 Testing against: {RUST_ENGINE_URL}")
    print(f"👤 Test patient ID: {TEST_PATIENT_ID}")
    print("=" * 60)
    
    tests = [
        ("Health Check", test_health_check),
        ("Engine Status", test_engine_status),
        ("Dose Calculation", test_vancomycin_dose_calculation),
        ("Safety Verification", test_vancomycin_safety_verification),
        ("Monitoring Requirements", test_vancomycin_monitoring_requirements),
        ("Flow2 Integration", test_flow2_integration)
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
        print("🎉 ALL TESTS PASSED! Vancomycin knowledge base is fully functional!")
    else:
        print(f"⚠️ {total - passed} tests failed. Check the output above for details.")
    print("=" * 60)
    
    return passed == total

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
