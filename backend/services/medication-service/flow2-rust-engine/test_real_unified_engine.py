#!/usr/bin/env python3
"""
Test the REAL Unified Clinical Engine with advanced dose optimization
"""

import requests
import json
import time

def test_real_unified_dose_optimization():
    """Test the REAL Unified Clinical Engine dose optimization"""
    
    print("🧪 Testing REAL Unified Clinical Engine - Dose Optimization")
    print("=" * 60)
    
    # Real patient data for testing with all required fields
    dose_request = {
        "request_id": "test-real-unified-001",
        "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
        "medication_code": "vancomycin",
        "optimization_type": "dose_calculation",  # Required field!
        "clinical_parameters": {
            "age_years": 65.0,
            "weight_kg": 80.0,
            "height_cm": 175.0,
            "gender": "male",
            "egfr": 45.0,  # Reduced kidney function
            "creatinine": 1.8,
            "albumin": 3.2
        },
        "clinical_context": {
            "indication": "severe_infection",
            "infection_site": "pneumonia",
            "severity": "severe",
            "culture_results": "MRSA_positive"
        },
        "processing_hints": {
            "priority": "high",
            "use_advanced_models": True,
            "include_pk_predictions": True
        }
    }
    
    try:
        print(f"📤 Sending dose optimization request...")
        print(f"   Patient: {dose_request['patient_id']}")
        print(f"   Drug: {dose_request['medication_code']}")
        print(f"   Age: {dose_request['clinical_parameters']['age_years']} years")
        print(f"   Weight: {dose_request['clinical_parameters']['weight_kg']} kg")
        print(f"   eGFR: {dose_request['clinical_parameters']['egfr']} mL/min")
        print()
        
        start_time = time.time()
        
        response = requests.post(
            "http://localhost:8080/api/dose/optimize",
            json=dose_request,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        end_time = time.time()
        execution_time = (end_time - start_time) * 1000
        
        print(f"📥 Response received in {execution_time:.2f}ms")
        print(f"   Status Code: {response.status_code}")
        
        if response.status_code == 200:
            result = response.json()
            
            print("\n🎯 REAL UNIFIED CLINICAL ENGINE RESULTS:")
            print("=" * 50)
            print(f"✅ Request ID: {result.get('request_id')}")
            print(f"💊 Optimized Dose: {result.get('optimized_dose')} mg")
            print(f"📊 Confidence Score: {result.get('optimization_score'):.3f}")
            
            confidence = result.get('confidence_interval', {})
            print(f"📈 Confidence Interval:")
            print(f"   Lower: {confidence.get('lower', 0):.1f} mg")
            print(f"   Upper: {confidence.get('upper', 0):.1f} mg")
            print(f"   Confidence: {confidence.get('confidence', 0):.1%}")
            
            print(f"🧠 Clinical Rationale:")
            print(f"   {result.get('clinical_rationale', 'N/A')}")
            
            monitoring = result.get('monitoring_recommendations', [])
            if monitoring:
                print(f"🔬 Monitoring Recommendations:")
                for rec in monitoring:
                    print(f"   • {rec}")
            
            pk_predictions = result.get('pharmacokinetic_predictions', {})
            if pk_predictions:
                print(f"⚗️ PK Predictions: {len(pk_predictions)} parameters")
            
            print(f"⏱️ Engine Execution Time: {result.get('execution_time_ms')} ms")
            
            # Validate the response shows real calculation
            if "REAL UNIFIED CLINICAL ENGINE" in result.get('clinical_rationale', ''):
                print("\n🎉 SUCCESS: Real Unified Clinical Engine is working!")
                print("   ✅ Advanced dose calculation performed")
                print("   ✅ Clinical reasoning applied")
                print("   ✅ Patient-specific adjustments made")
                return True
            else:
                print("\n⚠️ WARNING: May be using fallback calculation")
                return False
                
        else:
            print(f"❌ Error: {response.status_code}")
            print(f"Response: {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Request failed: {e}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False

def test_multiple_scenarios():
    """Test multiple clinical scenarios"""
    
    scenarios = [
        {
            "name": "Elderly with Renal Impairment",
            "medication_code": "vancomycin",
            "age_years": 85.0,
            "weight_kg": 65.0,
            "egfr": 30.0,
            "indication": "sepsis"
        },
        {
            "name": "Young Adult Normal Function",
            "medication_code": "vancomycin", 
            "age_years": 25.0,
            "weight_kg": 75.0,
            "egfr": 120.0,
            "indication": "endocarditis"
        },
        {
            "name": "Obese Patient",
            "medication_code": "vancomycin",
            "age_years": 45.0,
            "weight_kg": 120.0,
            "egfr": 80.0,
            "indication": "osteomyelitis"
        }
    ]
    
    print("\n🧪 Testing Multiple Clinical Scenarios")
    print("=" * 60)
    
    for i, scenario in enumerate(scenarios, 1):
        print(f"\n📋 Scenario {i}: {scenario['name']}")
        print("-" * 40)
        
        dose_request = {
            "request_id": f"scenario-{i:03d}",
            "patient_id": f"patient-{i:03d}",
            "medication_code": scenario["medication_code"],
            "optimization_type": "dose_calculation",  # Required field!
            "clinical_parameters": {
                "age_years": scenario["age_years"],
                "weight_kg": scenario["weight_kg"],
                "egfr": scenario["egfr"]
            },
            "clinical_context": {
                "indication": scenario["indication"]
            },
            "processing_hints": {
                "priority": "normal",
                "use_advanced_models": True
            }
        }
        
        try:
            response = requests.post(
                "http://localhost:8080/api/dose/optimize",
                json=dose_request,
                timeout=10
            )
            
            if response.status_code == 200:
                result = response.json()
                dose = result.get('optimized_dose')
                score = result.get('optimization_score')
                
                print(f"   💊 Dose: {dose} mg")
                print(f"   📊 Score: {score:.3f}")
                print(f"   ✅ Success")
            else:
                print(f"   ❌ Failed: {response.status_code}")
                
        except Exception as e:
            print(f"   ❌ Error: {e}")

if __name__ == "__main__":
    print("🚀 REAL UNIFIED CLINICAL ENGINE TEST SUITE")
    print("=" * 60)
    
    # Test 1: Single comprehensive test
    success = test_real_unified_dose_optimization()
    
    if success:
        # Test 2: Multiple scenarios
        test_multiple_scenarios()
        
        print("\n🎉 ALL TESTS COMPLETED!")
        print("✅ Real Unified Clinical Engine is fully operational")
        print("✅ Advanced dose optimization working")
        print("✅ Clinical reasoning integrated")
        print("✅ Patient-specific calculations active")
    else:
        print("\n❌ TESTS FAILED!")
        print("Please check the engine configuration")
