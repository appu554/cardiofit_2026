#!/usr/bin/env python3
"""
Phase 1 Medication Service Test Suite
Tests the complete Phase 1 functionality including:
- Clinical Recipe Catalog
- Clinical Recipe Execution  
- Dose Calculation
- Drug Interaction Checking
- Patient Medication Retrieval
"""

import asyncio
import httpx
import json
from datetime import datetime

BASE_URL = "http://localhost:8009"

async def test_health_check():
    """Test basic health check"""
    print("🏥 Testing Health Check...")
    async with httpx.AsyncClient() as client:
        response = await client.get(f"{BASE_URL}/health")
        print(f"Status: {response.status_code}")
        print(f"Response: {response.json()}")
        return response.status_code == 200

async def test_clinical_recipe_catalog():
    """Test clinical recipe catalog retrieval"""
    print("\n📚 Testing Clinical Recipe Catalog...")
    async with httpx.AsyncClient() as client:
        response = await client.get(f"{BASE_URL}/api/clinical-recipes/catalog")
        data = response.json()
        print(f"Status: {response.status_code}")
        print(f"Total Recipes: {data.get('total_recipes', 0)}")
        
        if data.get('recipes'):
            print("Sample Recipes:")
            for i, (recipe_id, recipe_info) in enumerate(list(data['recipes'].items())[:3]):
                print(f"  {i+1}. {recipe_id}: {recipe_info.get('name', 'N/A')}")
        
        return response.status_code == 200 and data.get('total_recipes', 0) > 0

async def test_dose_calculation():
    """Test dose calculation functionality"""
    print("\n💊 Testing Dose Calculation...")
    
    test_request = {
        "patient_id": "test-patient-123",
        "medication_code": "vancomycin",
        "indication": "pneumonia",
        "calculation_type": "weight_based",
        "patient_context": {
            "weight_kg": 70,
            "age_years": 45,
            "creatinine_clearance": 80
        },
        "dosing_parameters": {
            "dose_per_kg": 15
        }
    }
    
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{BASE_URL}/api/dose-calculation/calculate",
            json=test_request
        )
        data = response.json()
        print(f"Status: {response.status_code}")
        print(f"Calculation Status: {data.get('status', 'unknown')}")

        if data.get('error'):
            print(f"Error: {data.get('error')}")

        if data.get('calculated_dose'):
            dose = data['calculated_dose']
            print(f"Calculated Dose: {dose.get('display_string', 'N/A')}")
            print(f"Method: {dose.get('calculation_method', 'N/A')}")

        return response.status_code == 200 and data.get('status') == 'success'

async def test_drug_interactions():
    """Test drug interaction checking"""
    print("\n⚠️  Testing Drug Interaction Checking...")
    
    test_request = {
        "patient_id": "test-patient-456",
        "new_medication_code": "1049502"  # Acetaminophen
    }
    
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{BASE_URL}/api/drug-interactions/check",
            json=test_request
        )
        data = response.json()
        print(f"Status: {response.status_code}")
        print(f"Check Status: {data.get('status', 'unknown')}")
        print(f"Has Interactions: {data.get('has_interactions', False)}")
        print(f"Total Interactions: {data.get('total_interactions', 0)}")
        
        return response.status_code == 200 and data.get('status') == 'success'

async def test_clinical_recipe_execution():
    """Test clinical recipe execution"""
    print("\n🧠 Testing Clinical Recipe Execution...")
    
    test_request = {
        "patient_id": "test-patient-789",
        "action_type": "medication_prescribing",
        "medication_data": {
            "code": "vancomycin",
            "name": "Vancomycin",
            "dose": 1000,
            "unit": "mg",
            "route": "IV"
        },
        "patient_data": {
            "age_years": 65,
            "weight_kg": 80,
            "conditions": ["pneumonia", "chronic_kidney_disease"]
        },
        "clinical_data": {
            "creatinine_clearance": 45,
            "indication": "hospital_acquired_pneumonia"
        }
    }
    
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{BASE_URL}/api/clinical-recipes/execute",
            json=test_request
        )
        data = response.json()
        print(f"Status: {response.status_code}")
        print(f"Execution Status: {data.get('status', 'unknown')}")
        print(f"Overall Safety Status: {data.get('overall_safety_status', 'unknown')}")
        print(f"Recipes Executed: {data.get('total_recipes_executed', 0)}")
        print(f"Critical Issues: {data.get('critical_issues', 0)}")
        print(f"Warnings: {data.get('warnings', 0)}")
        
        if data.get('execution_summary'):
            summary = data['execution_summary']
            print(f"Total Execution Time: {summary.get('total_time_ms', 0)}ms")
        
        return response.status_code == 200 and data.get('status') == 'success'

async def test_patient_medications():
    """Test patient medication retrieval"""
    print("\n👤 Testing Patient Medication Retrieval...")
    
    patient_id = "test-patient-public"
    
    async with httpx.AsyncClient() as client:
        response = await client.get(f"{BASE_URL}/api/public/medication-requests/patient/{patient_id}")
        data = response.json()
        print(f"Status: {response.status_code}")
        print(f"Patient ID: {data.get('patient_id', 'N/A')}")
        print(f"Medication Count: {data.get('count', 0)}")
        
        if data.get('error'):
            print(f"Note: {data['error']} (Expected for test patient)")
        
        return response.status_code == 200

async def run_comprehensive_test():
    """Run all Phase 1 tests"""
    print("🚀 Starting Phase 1 Medication Service Comprehensive Test Suite")
    print("=" * 70)
    
    tests = [
        ("Health Check", test_health_check),
        ("Clinical Recipe Catalog", test_clinical_recipe_catalog),
        ("Dose Calculation", test_dose_calculation),
        ("Drug Interactions", test_drug_interactions),
        ("Clinical Recipe Execution", test_clinical_recipe_execution),
        ("Patient Medications", test_patient_medications),
    ]
    
    results = {}
    
    for test_name, test_func in tests:
        try:
            result = await test_func()
            results[test_name] = "✅ PASS" if result else "❌ FAIL"
        except Exception as e:
            print(f"Error in {test_name}: {e}")
            results[test_name] = f"❌ ERROR: {str(e)}"
    
    print("\n" + "=" * 70)
    print("📊 PHASE 1 TEST RESULTS SUMMARY")
    print("=" * 70)
    
    for test_name, result in results.items():
        print(f"{test_name:.<40} {result}")
    
    passed = sum(1 for r in results.values() if "✅" in r)
    total = len(results)
    
    print(f"\nOverall: {passed}/{total} tests passed")
    
    if passed == total:
        print("🎉 ALL PHASE 1 TESTS PASSED! Medication Service is fully functional.")
    else:
        print("⚠️  Some tests failed. Check the output above for details.")
    
    return passed == total

if __name__ == "__main__":
    asyncio.run(run_comprehensive_test())
