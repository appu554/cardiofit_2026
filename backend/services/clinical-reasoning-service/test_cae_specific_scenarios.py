#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CAE Specific Scenario Tests

This script focuses on testing specific clinical scenarios that should trigger 
alerts based on the sample data in cae-sample-data.ttl.

Usage:
    python test_cae_specific_scenarios.py [scenario_name]
    
    scenario_name: Optional. Name of specific scenario to test.
                  If omitted, all scenarios will be run.
"""

import os
import sys
import time
import logging
import grpc
import uuid
from datetime import datetime
from google.protobuf.struct_pb2 import Struct

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# CAE Server connection details
CAE_SERVER = "localhost:8027"

# Import generated protobuf modules - using direct import to avoid relative import issues
sys.path.append(os.path.join(os.path.dirname(__file__), 'app', 'proto'))

# Fix for relative import error
import sys, importlib.util

# Load the protobuf modules directly
proto_dir = os.path.join(os.path.dirname(__file__), 'app', 'proto')

# Load clinical_reasoning_pb2
pb2_path = os.path.join(proto_dir, 'clinical_reasoning_pb2.py')
pb2_spec = importlib.util.spec_from_file_location('clinical_reasoning_pb2', pb2_path)
clinical_reasoning_pb2 = importlib.util.module_from_spec(pb2_spec)
pb2_spec.loader.exec_module(clinical_reasoning_pb2)

# Load clinical_reasoning_pb2_grpc
pb2_grpc_path = os.path.join(proto_dir, 'clinical_reasoning_pb2_grpc.py')
# Need to modify the grpc file to use absolute import
with open(pb2_grpc_path, 'r') as f:
    grpc_content = f.read()

# If the file uses relative import, create a modified version with absolute import
if 'from . import clinical_reasoning_pb2' in grpc_content:
    modified_content = grpc_content.replace(
        'from . import clinical_reasoning_pb2 as clinical__reasoning__pb2',
        'import clinical_reasoning_pb2 as clinical__reasoning__pb2')
    
    # Create a temporary modified file
    temp_grpc_path = os.path.join(proto_dir, 'temp_clinical_reasoning_pb2_grpc.py')
    with open(temp_grpc_path, 'w') as f:
        f.write(modified_content)
    
    # Load the modified grpc module
    pb2_grpc_spec = importlib.util.spec_from_file_location('clinical_reasoning_pb2_grpc', temp_grpc_path)
    clinical_reasoning_pb2_grpc = importlib.util.module_from_spec(pb2_grpc_spec)
    # Make clinical_reasoning_pb2 available to the module
    sys.modules['clinical_reasoning_pb2'] = clinical_reasoning_pb2
    pb2_grpc_spec.loader.exec_module(clinical_reasoning_pb2_grpc)
else:
    # If it's already using absolute import, just load it directly
    pb2_grpc_spec = importlib.util.spec_from_file_location('clinical_reasoning_pb2_grpc', pb2_grpc_path)
    clinical_reasoning_pb2_grpc = importlib.util.module_from_spec(pb2_grpc_spec)
    # Make clinical_reasoning_pb2 available to the module
    sys.modules['clinical_reasoning_pb2'] = clinical_reasoning_pb2
    pb2_grpc_spec.loader.exec_module(clinical_reasoning_pb2_grpc)

def get_test_id(test_name):
    """Generate a unique test ID with timestamp for correlation"""
    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    return f"{test_name}-{timestamp}"

def test_warfarin_interactions(stub):
    """
    Test specific anticoagulant interactions with warfarin
    
    Patient 905a60cb has warfarin, aspirin and ibuprofen which should trigger 
    interaction alerts due to increased bleeding risk.
    """
    patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
    logger.info("\n" + "="*70)
    logger.info(f"SCENARIO: Warfarin Drug Interactions")
    logger.info("="*70)
    logger.info(f"Patient: {patient_id}")
    logger.info(f"Medications: warfarin + aspirin + ibuprofen")
    logger.info(f"Expected: Multiple medication interaction alerts for increased bleeding risk")
    logger.info("="*70)
    
    test_id = get_test_id("warfarin-interactions")
    
    try:
        # Create a clinical assertion request
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = patient_id
        request.correlation_id = f"req_{test_id}"
        
        # Create patient context using Struct
        patient_context = Struct()
        patient_context.update({
            "patient_id": patient_id,
            "source": "graphdb",
            "include_medications": True,
            "test_type": "medication_interactions",
            # Explicitly set medications for interaction checking
            "medications": ["warfarin", "aspirin", "ibuprofen"]
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info("📡 Calling GenerateAssertions for medication interactions...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Process interactions from response.interactions or response.assertions
        found_interactions = False
        
        # Check for interactions in the response (direct interactions attribute)
        if hasattr(response, 'interactions') and response.interactions:
            found_interactions = True
            logger.info(f"\n🔎 Found {len(response.interactions)} direct interaction records")
            
            for i, interaction in enumerate(response.interactions, 1):
                try:
                    logger.info(f"\n   {i}. {interaction.medication1} + {interaction.medication2}")
                    if hasattr(interaction, 'severity'):
                        logger.info(f"      Severity: {interaction.severity}")
                    if hasattr(interaction, 'description'):
                        logger.info(f"      Description: {interaction.description}")
                    if hasattr(interaction, 'mechanism'):
                        logger.info(f"      Mechanism: {interaction.mechanism}")
                except Exception as e:
                    logger.warning(f"      Error accessing interaction fields: {e}")
        
        # Check for interaction-related assertions
        if hasattr(response, 'assertions'):
            interaction_assertions = [
                a for a in response.assertions 
                if (hasattr(a, 'category') and 'interaction' in str(a.category).lower()) or
                   (hasattr(a, 'title') and any(med in str(a.title).lower() 
                                             for med in ['warfarin', 'aspirin', 'ibuprofen']))
            ]
            
            if interaction_assertions:
                found_interactions = True
                logger.info(f"\n🔎 Found {len(interaction_assertions)} interaction assertions")
                
                for i, assertion in enumerate(interaction_assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Interaction'}")
                        if hasattr(assertion, 'description'):
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        # Evaluation criteria
        if found_interactions:
            logger.info("\n✅ TEST PASSED: Found expected warfarin interactions")
            return True
        else:
            logger.warning("\n❌ TEST FAILED: No warfarin interactions detected")
            return False
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False

def test_nsaid_duplication(stub):
    """
    Test duplicate therapy detection with multiple NSAIDs
    
    Checks if the CAE detects therapeutic duplication when both ibuprofen and naproxen 
    (both NSAIDs) are present in the medication list.
    """
    # Use a patient that doesn't already have multiple NSAIDs
    patient_id = "patient_001"
    
    logger.info("\n" + "="*70)
    logger.info(f"SCENARIO: NSAID Therapeutic Duplication")
    logger.info("="*70)
    logger.info(f"Patient: {patient_id}")
    logger.info(f"Medications: ibuprofen + naproxen (both NSAIDs)")
    logger.info(f"Expected: Therapeutic duplication alert")
    logger.info("="*70)
    
    test_id = get_test_id("nsaid-duplication")
    
    try:
        # Create a clinical assertion request
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = patient_id
        request.correlation_id = f"req_{test_id}"
        
        # Create patient context using Struct
        patient_context = Struct()
        patient_context.update({
            "patient_id": patient_id,
            "source": "graphdb",
            "include_medications": True,
            "test_type": "duplicate_therapy",
            "medications": ["ibuprofen", "naproxen"]
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info("📡 Calling GenerateAssertions for duplicate therapy check...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Look for duplicate therapy alerts in assertions
        duplicate_found = False
        
        if hasattr(response, 'assertions'):
            duplicate_assertions = [
                a for a in response.assertions 
                if (hasattr(a, 'category') and 'duplicate' in str(a.category).lower()) or
                   (hasattr(a, 'title') and 'duplicate' in str(a.title).lower()) or
                   (hasattr(a, 'description') and all(med in str(a.description).lower() 
                                                  for med in ['ibuprofen', 'naproxen']))
            ]
            
            if duplicate_assertions:
                duplicate_found = True
                logger.info(f"\n🔎 Found {len(duplicate_assertions)} duplicate therapy assertions")
                
                for i, assertion in enumerate(duplicate_assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Alert'}")
                        if hasattr(assertion, 'description'):
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
                        
        # Check for any NSAID-related assertions
        if not duplicate_found and hasattr(response, 'assertions'):
            nsaid_assertions = [
                a for a in response.assertions 
                if (hasattr(a, 'title') and any(med in str(a.title).lower() 
                                             for med in ['ibuprofen', 'naproxen', 'nsaid']))
            ]
            
            if nsaid_assertions:
                logger.info(f"\n🔍 Found {len(nsaid_assertions)} NSAID-related assertions, but not specific duplicate therapy alerts")
                
                for i, assertion in enumerate(nsaid_assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Alert'}")
                        if hasattr(assertion, 'description'):
                            logger.info(f"      Description: {assertion.description}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
                
        # Evaluation criteria
        if duplicate_found:
            logger.info("\n✅ TEST PASSED: Detected duplicate NSAID therapy")
            return True
        else:
            logger.warning("\n❌ TEST FAILED: No duplicate NSAID therapy alert detected")
            return False
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False

def test_pediatric_medication_dosing(stub):
    """
    Test pediatric medication dosing alerts
    
    Checks if the CAE provides pediatric-specific dosing alerts for
    patient_004 who is 8 years old.
    """
    patient_id = "patient_004"  # 8-year-old patient
    
    logger.info("\n" + "="*70)
    logger.info(f"SCENARIO: Pediatric Medication Dosing")
    logger.info("="*70)
    logger.info(f"Patient: {patient_id} (8 years old)")
    logger.info(f"Medications: ibuprofen, amoxicillin")
    logger.info(f"Expected: Pediatric-specific dosing recommendations")
    logger.info("="*70)
    
    test_id = get_test_id("pediatric-dosing")
    
    try:
        # Create a clinical assertion request
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = patient_id
        request.correlation_id = f"req_{test_id}"
        
        # Create patient context using Struct
        patient_context = Struct()
        patient_context.update({
            "patient_id": patient_id,
            "source": "graphdb",
            "include_medications": True,
            "include_conditions": True,
            "test_type": "dose_adjustments",
            "medications": ["ibuprofen", "amoxicillin"]
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info("📡 Calling GenerateAssertions for pediatric dosing...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Look for pediatric-specific dosing alerts
        pediatric_alerts_found = False
        
        if hasattr(response, 'assertions'):
            pediatric_assertions = [
                a for a in response.assertions 
                if ((hasattr(a, 'category') and 'pediatric' in str(a.category).lower()) or
                    (hasattr(a, 'title') and any(term in str(a.title).lower() 
                                              for term in ['pediatric', 'child', 'children'])) or
                    (hasattr(a, 'description') and any(term in str(a.description).lower() 
                                                   for term in ['pediatric', 'child', 'children', 'age', 'young'])))
            ]
            
            if pediatric_assertions:
                pediatric_alerts_found = True
                logger.info(f"\n🔎 Found {len(pediatric_assertions)} pediatric-specific assertions")
                
                for i, assertion in enumerate(pediatric_assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Alert'}")
                        if hasattr(assertion, 'description'):
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                        if hasattr(assertion, 'recommendation') and assertion.recommendation:
                            logger.info(f"      Recommendation: {assertion.recommendation}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        # Also check for any dose adjustment recommendations
        if hasattr(response, 'dose_adjustments') and response.dose_adjustments:
            pediatric_alerts_found = True
            logger.info(f"\n🔎 Found {len(response.dose_adjustments)} dose adjustment recommendations")
            
            for i, adjustment in enumerate(response.dose_adjustments, 1):
                try:
                    logger.info(f"\n   {i}. {adjustment.medication}")
                    if hasattr(adjustment, 'recommended_dose'):
                        logger.info(f"      Recommended Dose: {adjustment.recommended_dose}")
                    if hasattr(adjustment, 'reason'):
                        logger.info(f"      Reason: {adjustment.reason}")
                except Exception as e:
                    logger.warning(f"      Error accessing dose adjustment fields: {e}")
                    
        # Evaluation criteria
        if pediatric_alerts_found:
            logger.info("\n✅ TEST PASSED: Found pediatric-specific medication alerts")
            return True
        else:
            logger.warning("\n❌ TEST FAILED: No pediatric-specific medication alerts detected")
            return False
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False

def test_pregnancy_contraindications(stub):
    """
    Test pregnancy contraindications
    
    Checks if the CAE detects medication contraindications for
    patient_006 who is pregnant.
    """
    patient_id = "patient_006"  # 28-year-old pregnant patient
    
    logger.info("\n" + "="*70)
    logger.info(f"SCENARIO: Pregnancy Contraindications")
    logger.info("="*70)
    logger.info(f"Patient: {patient_id} (28 years old, pregnant)")
    logger.info(f"Medications: warfarin, ibuprofen (contraindicated in pregnancy)")
    logger.info(f"Expected: Pregnancy contraindication alerts")
    logger.info("="*70)
    
    test_id = get_test_id("pregnancy-contraindications")
    
    try:
        # Create a clinical assertion request
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = patient_id
        request.correlation_id = f"req_{test_id}"
        
        # Create patient context using Struct
        patient_context = Struct()
        patient_context.update({
            "patient_id": patient_id,
            "source": "graphdb",
            "include_medications": True,
            "include_conditions": True,
            "test_type": "contraindications",
            "medications": ["warfarin", "ibuprofen"],
            "conditions": ["pregnancy"]  # Explicitly add pregnancy as a condition
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info("📡 Calling GenerateAssertions for pregnancy contraindications...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Look for pregnancy-related contraindication alerts
        pregnancy_alerts_found = False
        
        if hasattr(response, 'assertions'):
            pregnancy_assertions = [
                a for a in response.assertions 
                if ((hasattr(a, 'category') and 'contraindication' in str(a.category).lower()) or
                    (hasattr(a, 'title') and any(term in str(a.title).lower() 
                                              for term in ['pregnancy', 'pregnant', 'contraindication'])) or
                    (hasattr(a, 'description') and any(term in str(a.description).lower() 
                                                   for term in ['pregnancy', 'pregnant', 'fetus', 'teratogenic'])))
            ]
            
            if pregnancy_assertions:
                pregnancy_alerts_found = True
                logger.info(f"\n🔎 Found {len(pregnancy_assertions)} pregnancy-related assertions")
                
                for i, assertion in enumerate(pregnancy_assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Alert'}")
                        if hasattr(assertion, 'description'):
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                        if hasattr(assertion, 'recommendation') and assertion.recommendation:
                            logger.info(f"      Recommendation: {assertion.recommendation}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        # Evaluation criteria
        if pregnancy_alerts_found:
            logger.info("\n✅ TEST PASSED: Found pregnancy contraindication alerts")
            return True
        else:
            logger.warning("\n❌ TEST FAILED: No pregnancy contraindication alerts detected")
            return False
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False

def run_all_scenarios(stub):
    """Run all test scenarios and report results"""
    logger.info("\n" + "="*70)
    logger.info("🏥 CAE SPECIFIC CLINICAL SCENARIO TEST SUITE")
    logger.info("="*70)
    
    # Dictionary to hold test results
    results = {}
    
    # Run all scenarios
    results["Warfarin Interactions"] = test_warfarin_interactions(stub)
    results["NSAID Duplication"] = test_nsaid_duplication(stub)
    results["Pediatric Dosing"] = test_pediatric_medication_dosing(stub)
    results["Pregnancy Contraindications"] = test_pregnancy_contraindications(stub)
    
    # Display final test summary
    logger.info("\n" + "="*70)
    logger.info("📊 FINAL SCENARIO TEST SUMMARY")
    logger.info("="*70)
    
    passed_tests = 0
    for scenario, result in results.items():
        status = "✅ PASSED" if result else "❌ FAILED"
        logger.info(f"{scenario}: {status}")
        if result:
            passed_tests += 1
    
    logger.info(f"\nTotal: {passed_tests}/{len(results)} scenarios passed")
    
    if passed_tests == len(results):
        logger.info("\n🎉 All scenario tests passed! CAE correctly identifies clinical issues.")
        return True
    else:
        logger.warning(f"\n⚠️ {len(results) - passed_tests} scenario tests failed.")
        return False

def run_specific_scenario(stub, scenario_name):
    """Run a specific clinical scenario test"""
    scenario_map = {
        "warfarin": test_warfarin_interactions,
        "nsaid": test_nsaid_duplication,
        "pediatric": test_pediatric_medication_dosing,
        "pregnancy": test_pregnancy_contraindications
    }
    
    # Find the closest matching scenario
    matching_scenario = None
    for key in scenario_map:
        if scenario_name.lower() in key or key in scenario_name.lower():
            matching_scenario = key
            break
    
    if not matching_scenario:
        logger.error(f"Scenario '{scenario_name}' not found. Available scenarios: {', '.join(scenario_map.keys())}")
        return False
        
    # Run the selected scenario
    logger.info(f"Running specific scenario: {matching_scenario}")
    return scenario_map[matching_scenario](stub)

if __name__ == "__main__":
    # Create a gRPC channel
    channel = grpc.insecure_channel(CAE_SERVER)
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Check if a specific scenario was requested
    if len(sys.argv) > 1:
        scenario_name = sys.argv[1].lower()
        result = run_specific_scenario(stub, scenario_name)
        
        if result:
            logger.info(f"\n✅ {scenario_name} scenario passed!")
            sys.exit(0)
        else:
            logger.error(f"\n❌ {scenario_name} scenario failed!")
            sys.exit(1)
    else:
        # Run all scenarios
        success = run_all_scenarios(stub)
        sys.exit(0 if success else 1)
