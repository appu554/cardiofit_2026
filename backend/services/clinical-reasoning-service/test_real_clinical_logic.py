#!/usr/bin/env python3
"""
Test script for real clinical logic in CAE gRPC service
"""

import asyncio
import sys
from pathlib import Path

# Add the shared directory to the path
sys.path.insert(0, str(Path(__file__).parent.parent / 'shared'))

from cae_grpc_client import CAEgRPCClient

async def test_real_clinical_logic():
    """Test the CAE gRPC service with real clinical scenarios"""
    print("🧪 Testing Real Clinical Logic in CAE")
    print("=" * 50)
    
    try:
        async with CAEgRPCClient(service_name="clinical-test") as client:
            print("✅ Connected to CAE service")
            
            # Test 1: Real Warfarin + Aspirin Interaction
            print("\n💊 Test 1: Warfarin + Aspirin Interaction")
            print("-" * 40)
            result = await client.check_medication_interactions(
                patient_id="test-patient-001",
                medication_ids=["warfarin", "aspirin"],
                patient_context={
                    "age": 75,
                    "weight": 70,
                    "kidney_function": "mild_impairment"
                }
            )
            
            print(f"Found {len(result['interactions'])} interactions:")
            for interaction in result['interactions']:
                print(f"  🔴 {interaction['medication_a']} + {interaction['medication_b']}")
                print(f"     Severity: {interaction['severity']}")
                print(f"     Mechanism: {interaction['mechanism']}")
                print(f"     Clinical Effect: {interaction['clinical_effect']}")
                print(f"     Confidence: {interaction['confidence_score']}")
                print(f"     Evidence: {', '.join(interaction['evidence_sources'][:2])}")
                print()
            
            # Test 2: Real Warfarin Dosing Calculation
            print("📏 Test 2: Warfarin Dosing for Elderly Patient")
            print("-" * 40)
            dosing_result = await client.calculate_dosing(
                patient_id="test-patient-001",
                medication_id="warfarin",
                patient_parameters={
                    "age": 78,
                    "weight": 65,
                    "liver_function": "mild_impairment",
                    "indication": "atrial_fibrillation"
                }
            )
            
            print(f"Medication: {dosing_result['dosing']['medication_id']}")
            print(f"Dose: {dosing_result['dosing']['dose']}")
            print(f"Frequency: {dosing_result['dosing']['frequency']}")
            print(f"Route: {dosing_result['dosing']['route']}")
            print(f"Rationale: {dosing_result['dosing']['rationale']}")
            print(f"Adjustments: {len(dosing_result['adjustments'])}")
            for adj in dosing_result['adjustments']:
                print(f"  - {adj['type']}: {adj['adjustment']} ({adj['rationale']})")
            print()
            
            # Test 3: Real Contraindication Check
            print("⚠️  Test 3: Contraindication Check - Warfarin in Pregnancy")
            print("-" * 40)
            contraindication_result = await client.check_contraindications(
                patient_id="test-patient-002",
                medication_ids=["warfarin"],
                condition_ids=["pregnancy"],
                patient_context={
                    "age": 28,
                    "pregnancy_status": "pregnant",
                    "gestational_age": 12
                }
            )
            
            print(f"Found {len(contraindication_result['contraindications'])} contraindications:")
            for contraindication in contraindication_result['contraindications']:
                print(f"  🚫 {contraindication['medication_id']}")
                print(f"     Type: {contraindication['type']}")
                print(f"     Severity: {contraindication['severity']}")
                print(f"     Description: {contraindication['description']}")
                print(f"     Override Possible: {contraindication['override_possible']}")
                print(f"     Evidence: {', '.join(contraindication['evidence_sources'][:2])}")
                print()
            
            # Test 4: Complex Drug Interaction - Multiple Medications
            print("🔬 Test 4: Complex Multi-Drug Interaction Check")
            print("-" * 40)
            complex_result = await client.check_medication_interactions(
                patient_id="test-patient-003",
                medication_ids=["warfarin", "amiodarone", "digoxin"],
                patient_context={
                    "age": 82,
                    "weight": 60,
                    "kidney_function": "moderate_impairment",
                    "liver_function": "normal"
                }
            )
            
            print(f"Found {len(complex_result['interactions'])} interactions:")
            for interaction in complex_result['interactions']:
                print(f"  🔴 {interaction['medication_a']} + {interaction['medication_b']}")
                print(f"     Severity: {interaction['severity']}")
                print(f"     Description: {interaction['description']}")
                print()
            
            # Test 5: Dosing with Kidney Impairment
            print("🫘 Test 5: Dosing Adjustment for Kidney Impairment")
            print("-" * 40)
            kidney_dosing = await client.calculate_dosing(
                patient_id="test-patient-004",
                medication_id="lisinopril",
                patient_parameters={
                    "age": 65,
                    "weight": 80,
                    "kidney_function": "moderate_impairment",
                    "creatinine_clearance": 45,
                    "indication": "hypertension"
                }
            )
            
            print(f"Medication: {kidney_dosing['dosing']['medication_id']}")
            print(f"Dose: {kidney_dosing['dosing']['dose']}")
            print(f"Rationale: {kidney_dosing['dosing']['rationale']}")
            print(f"Warnings: {', '.join(kidney_dosing['dosing']['warnings'])}")
            print()
            
            # Test 6: Allergy Cross-Sensitivity
            print("🤧 Test 6: Allergy Cross-Sensitivity Check")
            print("-" * 40)
            allergy_result = await client.check_contraindications(
                patient_id="test-patient-005",
                medication_ids=["amoxicillin"],
                allergy_ids=["penicillin"],
                patient_context={
                    "age": 45,
                    "allergy_severity": "severe"
                }
            )
            
            print(f"Found {len(allergy_result['contraindications'])} contraindications:")
            for contraindication in allergy_result['contraindications']:
                print(f"  🚫 {contraindication['medication_id']}")
                print(f"     Type: {contraindication['type']}")
                print(f"     Description: {contraindication['description']}")
                print(f"     Override: {contraindication['override_rationale']}")
                print()
            
            print("🎉 All real clinical logic tests completed successfully!")
            
    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(test_real_clinical_logic())
