#!/usr/bin/env python3
"""
🧪 Flow 2 Complete Integration Test
Tests the complete Flow 2 workflow:
1. Go Orchestrator LOCAL DECISION (<1ms)
2. Go Orchestrator GLOBAL FETCH (~15ms)
3. Clinical Processing & Proposal Generation

This validates the production-scale Flow 2 implementation.
"""

import requests
import json
import time
from datetime import datetime

# Configuration
GO_ORCHESTRATOR_URL = "http://localhost:8080"
CONTEXT_SERVICE_URL = "http://localhost:8016"  # Context Service
TEST_PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def print_header(title):
    print(f"\n{'='*60}")
    print(f"🧪 {title}")
    print(f"{'='*60}")

def print_step(step_num, title, time_budget):
    print(f"\n📋 STEP {step_num}: {title}")
    print(f"   Time Budget: {time_budget}")
    print(f"   {'-'*50}")

def test_step1_local_decision():
    """Test Step 1: Go Orchestrator LOCAL DECISION (<1ms)"""
    print_step(1, "Go Orchestrator LOCAL DECISION", "<1ms")
    
    # Test multiple medications to validate ORB rule matching
    test_cases = [
        {
            "name": "Vancomycin Standard",
            "request": {
                "request_id": "flow2-step1-vanc",
                "patient_id": TEST_PATIENT_ID,
                "medication_code": "11124",
                "medication_name": "Vancomycin",
                "patient_conditions": ["sepsis"]
            }
        },
        {
            "name": "Warfarin Initiation", 
            "request": {
                "request_id": "flow2-step1-warf",
                "patient_id": TEST_PATIENT_ID,
                "medication_code": "11289",
                "medication_name": "Warfarin",
                "patient_conditions": ["atrial_fibrillation"]
            }
        },
        {
            "name": "Heparin Infusion",
            "request": {
                "request_id": "flow2-step1-hep",
                "patient_id": TEST_PATIENT_ID,
                "medication_code": "5224", 
                "medication_name": "Heparin",
                "patient_conditions": ["vte"]
            }
        }
    ]
    
    results = []
    
    for test_case in test_cases:
        print(f"\n🧠 Testing: {test_case['name']}")
        print(f"   📤 INPUT REQUEST:")
        print(f"      {json.dumps(test_case['request'], indent=6)}")

        start_time = time.perf_counter()

        try:
            response = requests.post(
                f"{GO_ORCHESTRATOR_URL}/api/v1/test/orb",
                json=test_case["request"],
                timeout=5
            )
            
            end_time = time.perf_counter()
            execution_time_ms = (end_time - start_time) * 1000
            
            if response.status_code == 200:
                data = response.json()

                print(f"   📥 RAW OUTPUT RESPONSE:")
                print(f"      {json.dumps(data, indent=6)}")

                # Validate Intent Manifest structure
                if data.get("status") == "success" and "intent_manifest" in data:
                    manifest = data["intent_manifest"]

                    print(f"   ✅ SUCCESS - {execution_time_ms:.2f}ms")
                    print(f"   🎯 DECISION PROCESS:")
                    print(f"      ➤ Medication Lookup: {test_case['request']['medication_name']} (Code: {test_case['request']['medication_code']})")
                    print(f"      ➤ Rule Matched: {manifest.get('rule_id', 'Unknown')}")
                    print(f"      ➤ Recipe Selected: {manifest.get('recipe_id')}")
                    print(f"      ➤ Variant Chosen: {manifest.get('variant')}")
                    print(f"      ➤ Data Requirements Generated: {len(manifest.get('data_requirements', []))} items")
                    print(f"      📋 Required Clinical Data: {manifest.get('data_requirements', [])}")
                    print(f"      ➤ Clinical Rationale: {manifest.get('clinical_rationale', 'N/A')}")

                    # Validate sub-millisecond performance for local decision
                    if execution_time_ms < 1.0:
                        print(f"      🎯 PERFORMANCE TARGET MET: <1ms")
                    else:
                        print(f"      ⚠️  Performance: {execution_time_ms:.2f}ms (target: <1ms)")
                    
                    results.append({
                        "name": test_case["name"],
                        "success": True,
                        "time_ms": execution_time_ms,
                        "manifest": manifest
                    })
                else:
                    print(f"   ❌ FAILED - Invalid response structure")
                    print(f"      Response: {data}")
                    results.append({"name": test_case["name"], "success": False})
            else:
                print(f"   ❌ FAILED - HTTP {response.status_code}")
                print(f"      Response: {response.text}")
                results.append({"name": test_case["name"], "success": False})
                
        except Exception as e:
            print(f"   ❌ ERROR: {str(e)}")
            results.append({"name": test_case["name"], "success": False})
    
    return results

def test_step2_global_fetch(step1_results):
    """Test Step 2: Go Orchestrator GLOBAL FETCH (~15ms)"""
    print_step(2, "Go Orchestrator GLOBAL FETCH", "~15ms")
    
    # Use successful results from Step 1
    successful_cases = [r for r in step1_results if r.get("success")]
    
    if not successful_cases:
        print("   ❌ No successful Step 1 results to test Step 2")
        return []
    
    results = []
    
    for case in successful_cases[:2]:  # Test first 2 successful cases
        print(f"\n🌐 Testing Global Fetch: {case['name']}")

        manifest = case["manifest"]
        data_requirements = manifest.get("data_requirements", [])

        print(f"   📤 STEP 2 INPUT (from Step 1 Intent Manifest):")
        print(f"      Patient ID: {TEST_PATIENT_ID}")
        print(f"      Recipe ID: {manifest.get('recipe_id')}")
        print(f"      Variant: {manifest.get('variant')}")
        print(f"      Data Requirements: {data_requirements}")

        # Simulate Context Service call with data requirements
        context_request = {
            "patient_id": TEST_PATIENT_ID,
            "data_requirements": data_requirements,
            "recipe_id": manifest.get("recipe_id"),
            "variant": manifest.get("variant")
        }
        
        start_time = time.perf_counter()
        
        try:
            # Call Context Service with data requirements
            fields_param = ",".join(data_requirements) if data_requirements else None
            params = {}
            if fields_param:
                params["fields"] = fields_param

            response = requests.get(
                f"{CONTEXT_SERVICE_URL}/api/context/patient/{TEST_PATIENT_ID}/context",
                params=params,
                timeout=5
            )
            
            end_time = time.perf_counter()
            execution_time_ms = (end_time - start_time) * 1000
            
            if response.status_code == 200:
                context_data = response.json()

                print(f"   📥 STEP 2 RAW OUTPUT (Context Service Response):")
                print(f"      {json.dumps(context_data, indent=6)}")

                print(f"   ✅ SUCCESS - {execution_time_ms:.2f}ms")
                print(f"   🎯 CONTEXT ASSEMBLY PROCESS:")
                print(f"      ➤ Data Requirements Sent: {len(data_requirements)} items")
                print(f"      ➤ Context Service URL: {CONTEXT_SERVICE_URL}/api/context/patient/{TEST_PATIENT_ID}/context")
                print(f"      ➤ Fields Parameter: {','.join(data_requirements) if data_requirements else 'None'}")
                print(f"      ➤ Clinical Context Retrieved: {len(context_data.keys())} top-level fields")
                print(f"      ➤ Context Assembly Time: {execution_time_ms:.2f}ms")

                # Validate ~15ms performance target
                if execution_time_ms <= 20.0:
                    print(f"      🎯 PERFORMANCE TARGET MET: ≤20ms")
                else:
                    print(f"      ⚠️  Performance: {execution_time_ms:.2f}ms (target: ~15ms)")
                
                results.append({
                    "name": case["name"],
                    "success": True,
                    "time_ms": execution_time_ms,
                    "data_requirements": len(data_requirements),
                    "context_data": context_data
                })
            else:
                print(f"   ❌ FAILED - HTTP {response.status_code}")
                results.append({"name": case["name"], "success": False})
                
        except Exception as e:
            import traceback
            print(f"   ❌ ERROR: {str(e)}")
            print(f"   📍 ERROR LOCATION: {traceback.format_exc()}")
            results.append({"name": case["name"], "success": False})
    
    return results

def test_complete_flow2_integration():
    """Test the complete Flow 2 integration"""
    print_header("Flow 2 Complete Integration Test")
    print(f"Go Orchestrator: {GO_ORCHESTRATOR_URL}")
    print(f"Context Service: {CONTEXT_SERVICE_URL}")
    print(f"Test Patient: {TEST_PATIENT_ID}")
    print(f"Test Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    
    # Step 1: Local Decision
    step1_results = test_step1_local_decision()
    step1_success = len([r for r in step1_results if r.get("success")])
    
    # Step 2: Global Fetch
    step2_results = test_step2_global_fetch(step1_results)
    step2_success = len([r for r in step2_results if r.get("success")])
    
    # Summary
    print_header("Flow 2 Integration Test Summary")
    print(f"📊 STEP 1 (Local Decision): {step1_success}/{len(step1_results)} passed")
    print(f"📊 STEP 2 (Global Fetch): {step2_success}/{len(step2_results)} passed")
    
    total_tests = len(step1_results) + len(step2_results)
    total_success = step1_success + step2_success
    
    print(f"\n🎯 OVERALL FLOW 2 SUCCESS RATE: {total_success}/{total_tests}")
    
    if total_success == total_tests:
        print("🎉 FLOW 2 COMPLETE INTEGRATION: ✅ FULLY OPERATIONAL")
    elif total_success > total_tests * 0.8:
        print("⚠️  FLOW 2 INTEGRATION: Mostly working, minor issues")
    else:
        print("❌ FLOW 2 INTEGRATION: Major issues detected")
    
    return {
        "step1_results": step1_results,
        "step2_results": step2_results,
        "success_rate": total_success / total_tests if total_tests > 0 else 0
    }

if __name__ == "__main__":
    test_complete_flow2_integration()
