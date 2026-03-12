#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CAE Direct Scenario Tests

This script tests specific clinical scenarios by directly supplying
all needed data in the request rather than relying on GraphDB relationships.

Usage:
    python test_cae_direct_scenarios.py [scenario_name]
    
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

def test_direct_medication_interactions(stub):
    """
    Test warfarin drug interactions by directly providing complete medication data
    """
    logger.info("\n" + "="*70)
    logger.info(f"SCENARIO: Direct Warfarin Drug Interactions Test")
    logger.info("="*70)
    logger.info(f"Medications: warfarin + aspirin + ibuprofen")
    logger.info(f"Expected: Bleeding risk interaction alerts")
    logger.info("="*70)
    
    test_id = get_test_id("direct-warfarin")
    
    try:
        # Create a clinical assertion request with complete data
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = f"direct-test-{uuid.uuid4()}"
        request.correlation_id = f"req_{test_id}"
        
        # Create detailed patient context using Struct
        patient_context = Struct()
        
        # Define complete patient data including medications with details
        patient_context.update({
            "test_type": "medication_interactions",
            "patient": {
                "id": f"direct-test-{uuid.uuid4()}",
                "age": 67,
                "gender": "male",
                "weight": 78.5
            },
            "medications": [
                {
                    "id": "11289",
                    "name": "warfarin",
                    "brand": "Coumadin",
                    "class": "anticoagulant",
                    "dosage": "5 mg",
                    "frequency": "daily",
                    "active": True
                },
                {
                    "id": "1191",
                    "name": "aspirin",
                    "brand": "Bayer",
                    "class": "antiplatelet",
                    "dosage": "81 mg",
                    "frequency": "daily",
                    "active": True
                },
                {
                    "id": "5640",
                    "name": "ibuprofen",
                    "brand": "Advil",
                    "class": "nsaid",
                    "dosage": "400 mg",
                    "frequency": "as needed",
                    "active": True
                }
            ],
            "include_interactions": True,
            "detailed_analysis": True
        })
        
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info("📡 Calling GenerateAssertions with direct medication data...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Process interactions from response
        found_interactions = False
        interaction_data = []
        
        # Check for interactions in the response
        if hasattr(response, 'interactions') and response.interactions:
            found_interactions = True
            logger.info(f"\n🔎 Found {len(response.interactions)} direct interaction records")
            
            for i, interaction in enumerate(response.interactions, 1):
                try:
                    interaction_info = {
                        "medications": f"{interaction.medication1} + {interaction.medication2}" if hasattr(interaction, 'medication1') and hasattr(interaction, 'medication2') else "Unnamed interaction"
                    }
                    
                    if hasattr(interaction, 'severity'):
                        interaction_info["severity"] = interaction.severity
                        
                    if hasattr(interaction, 'description'):
                        interaction_info["description"] = interaction.description
                        
                    if hasattr(interaction, 'mechanism'):
                        interaction_info["mechanism"] = interaction.mechanism
                        
                    interaction_data.append(interaction_info)
                    
                    logger.info(f"\n   {i}. {interaction_info.get('medications')}")
                    logger.info(f"      Severity: {interaction_info.get('severity', 'Not specified')}")
                    logger.info(f"      Description: {interaction_info.get('description', 'Not specified')}")
                    
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
                        assertion_info = {
                            "title": assertion.title if hasattr(assertion, 'title') else "Untitled Interaction"
                        }
                        
                        if hasattr(assertion, 'description'):
                            assertion_info["description"] = assertion.description
                            
                        if hasattr(assertion, 'severity'):
                            assertion_info["severity"] = assertion.severity
                            
                        interaction_data.append(assertion_info)
                        
                        logger.info(f"\n   {i}. {assertion_info.get('title')}")
                        logger.info(f"      Description: {assertion_info.get('description', 'Not specified')}")
                        logger.info(f"      Severity: {assertion_info.get('severity', 'Not specified')}")
                        
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        # Dump full response for analysis if no interactions found
        if not found_interactions:
            logger.info("\n📋 Full response content for debugging:")
            logger.info(f"{str(response)[:1000]}...")  # Showing first 1000 chars
            
            # Look for any assertions at all
            if hasattr(response, 'assertions') and response.assertions:
                logger.info(f"\n🔍 Found {len(response.assertions)} assertions (not interaction specific)")
                for i, assertion in enumerate(response.assertions[:3], 1):  # Show first 3 as examples
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled'}")
                        if hasattr(assertion, 'description'):
                            logger.info(f"      Description: {assertion.description}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        # Evaluation criteria
        if found_interactions:
            logger.info("\n✅ TEST PASSED: Found expected medication interactions")
            return True, interaction_data
        else:
            logger.warning("\n❌ TEST FAILED: No medication interactions detected")
            return False, []
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False, []

def test_direct_nsaid_duplication(stub):
    """
    Test duplicate therapy detection with multiple NSAIDs by directly providing medication data
    """
    logger.info("\n" + "="*70)
    logger.info(f"SCENARIO: Direct NSAID Therapeutic Duplication Test")
    logger.info("="*70)
    logger.info(f"Medications: ibuprofen + naproxen (both NSAIDs)")
    logger.info(f"Expected: Duplicate therapeutic class alert")
    logger.info("="*70)
    
    test_id = get_test_id("direct-nsaid")
    
    try:
        # Create a clinical assertion request with complete data
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = f"direct-test-{uuid.uuid4()}"
        request.correlation_id = f"req_{test_id}"
        
        # Create detailed patient context using Struct
        patient_context = Struct()
        
        # Define complete patient data with duplicate NSAID medications
        patient_context.update({
            "test_type": "duplicate_therapy",
            "patient": {
                "id": f"direct-test-{uuid.uuid4()}",
                "age": 45,
                "gender": "female",
                "weight": 65.0
            },
            "medications": [
                {
                    "id": "5640",
                    "name": "ibuprofen",
                    "brand": "Advil",
                    "class": "nsaid",
                    "dosage": "400 mg",
                    "frequency": "three times daily",
                    "active": True
                },
                {
                    "id": "7258",
                    "name": "naproxen",
                    "brand": "Aleve",
                    "class": "nsaid",
                    "dosage": "220 mg",
                    "frequency": "twice daily",
                    "active": True
                }
            ],
            "check_duplicate_therapy": True,
            "detailed_analysis": True
        })
        
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info("📡 Calling GenerateAssertions for duplicate therapy...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Look for duplicate therapy alerts in assertions
        duplicate_found = False
        duplicate_data = []
        
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
                        assertion_info = {
                            "title": assertion.title if hasattr(assertion, 'title') else "Untitled Alert"
                        }
                        
                        if hasattr(assertion, 'description'):
                            assertion_info["description"] = assertion.description
                            
                        if hasattr(assertion, 'severity'):
                            assertion_info["severity"] = assertion.severity
                            
                        duplicate_data.append(assertion_info)
                        
                        logger.info(f"\n   {i}. {assertion_info.get('title')}")
                        logger.info(f"      Description: {assertion_info.get('description', 'Not specified')}")
                        logger.info(f"      Severity: {assertion_info.get('severity', 'Not specified')}")
                        
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
                
                for i, assertion in enumerate(nsaid_assertions[:3], 1):  # Show first 3
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Alert'}")
                        if hasattr(assertion, 'description'):
                            logger.info(f"      Description: {assertion.description}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        # Dump full response for analysis if no duplicates found
        if not duplicate_found:
            logger.info("\n📋 Full response content for debugging:")
            logger.info(f"{str(response)[:1000]}...")  # Showing first 1000 chars
            
        # Evaluation criteria
        if duplicate_found:
            logger.info("\n✅ TEST PASSED: Detected duplicate NSAID therapy")
            return True, duplicate_data
        else:
            logger.warning("\n❌ TEST FAILED: No duplicate NSAID therapy alert detected")
            return False, []
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False, []

def test_direct_pregnancy_contraindications(stub):
    """
    Test pregnancy contraindications by directly providing patient and medication data
    """
    logger.info("\n" + "="*70)
    logger.info(f"SCENARIO: Direct Pregnancy Contraindications Test")
    logger.info("="*70)
    logger.info(f"Patient: 28-year-old pregnant female")
    logger.info(f"Medications: warfarin (contraindicated in pregnancy)")
    logger.info(f"Expected: Pregnancy contraindication alert")
    logger.info("="*70)
    
    test_id = get_test_id("direct-pregnancy")
    
    try:
        # Create a clinical assertion request with complete data
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = f"direct-test-{uuid.uuid4()}"
        request.correlation_id = f"req_{test_id}"
        
        # Create detailed patient context using Struct
        patient_context = Struct()
        
        # Define complete patient data with pregnancy condition and contraindicated medication
        patient_context.update({
            "test_type": "contraindications",
            "patient": {
                "id": f"direct-test-{uuid.uuid4()}",
                "age": 28,
                "gender": "female",
                "weight": 65.0
            },
            "medications": [
                {
                    "id": "11289",
                    "name": "warfarin",
                    "brand": "Coumadin",
                    "class": "anticoagulant",
                    "dosage": "5 mg",
                    "frequency": "daily",
                    "active": True
                }
            ],
            "conditions": [
                {
                    "id": "77386006",  # SNOMED CT for Pregnancy
                    "name": "Pregnant",
                    "code": "77386006",
                    "coding_system": "SNOMEDCT",
                    "active": True
                }
            ],
            "check_contraindications": True,
            "detailed_analysis": True
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
        pregnancy_data = []
        
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
                        assertion_info = {
                            "title": assertion.title if hasattr(assertion, 'title') else "Untitled Alert"
                        }
                        
                        if hasattr(assertion, 'description'):
                            assertion_info["description"] = assertion.description
                            
                        if hasattr(assertion, 'severity'):
                            assertion_info["severity"] = assertion.severity
                            
                        if hasattr(assertion, 'recommendation') and assertion.recommendation:
                            assertion_info["recommendation"] = assertion.recommendation
                            
                        pregnancy_data.append(assertion_info)
                        
                        logger.info(f"\n   {i}. {assertion_info.get('title')}")
                        logger.info(f"      Description: {assertion_info.get('description', 'Not specified')}")
                        logger.info(f"      Severity: {assertion_info.get('severity', 'Not specified')}")
                        if "recommendation" in assertion_info:
                            logger.info(f"      Recommendation: {assertion_info['recommendation']}")
                        
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        # Dump full response for analysis if no alerts found
        if not pregnancy_alerts_found:
            logger.info("\n📋 Full response content for debugging:")
            logger.info(f"{str(response)[:1000]}...")  # Showing first 1000 chars
            
        # Evaluation criteria
        if pregnancy_alerts_found:
            logger.info("\n✅ TEST PASSED: Found pregnancy contraindication alerts")
            return True, pregnancy_data
        else:
            logger.warning("\n❌ TEST FAILED: No pregnancy contraindication alerts detected")
            return False, []
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False, []

def run_all_scenarios(stub):
    """Run all test scenarios and report results"""
    logger.info("\n" + "="*70)
    logger.info("🏥 CAE DIRECT CLINICAL SCENARIO TEST SUITE")
    logger.info("="*70)
    
    # Dictionary to hold test results and data
    results = {}
    all_data = {}
    
    # Run all scenarios
    success, data = test_direct_medication_interactions(stub)
    results["Warfarin Interactions"] = success
    all_data["Warfarin Interactions"] = data
    
    success, data = test_direct_nsaid_duplication(stub)
    results["NSAID Duplication"] = success
    all_data["NSAID Duplication"] = data
    
    success, data = test_direct_pregnancy_contraindications(stub)
    results["Pregnancy Contraindications"] = success
    all_data["Pregnancy Contraindications"] = data
    
    # Display final test summary
    logger.info("\n" + "="*70)
    logger.info("📊 FINAL DIRECT SCENARIO TEST SUMMARY")
    logger.info("="*70)
    
    passed_tests = 0
    for scenario, result in results.items():
        status = "✅ PASSED" if result else "❌ FAILED"
        logger.info(f"{scenario}: {status}")
        if result:
            passed_tests += 1
    
    logger.info(f"\nTotal: {passed_tests}/{len(results)} scenarios passed")
    
    if passed_tests == len(results):
        logger.info("\n🎉 All direct scenario tests passed! CAE correctly identifies clinical issues.")
    else:
        logger.warning(f"\n⚠️ {len(results) - passed_tests} direct scenario tests failed.")
        
    # Provide analysis of test results
    logger.info("\n" + "="*70)
    logger.info("📋 ANALYSIS OF TEST RESULTS")
    logger.info("="*70)
    
    if passed_tests == 0:
        logger.warning("""
        All tests failed, which suggests one of the following issues:
        
        1. The CAE service may not be fully configured to detect these clinical issues
        2. The request structure may need adjustments to match CAE's expected input format
        3. The specific clinical rules for these scenarios may not be implemented in CAE
        
        Next steps:
        1. Review CAE documentation for correct request formatting
        2. Check if these clinical scenarios are supported in the current CAE version
        3. Consider discussing with the CAE development team to confirm feature support
        """)
    
    return passed_tests == len(results)

def run_specific_scenario(stub, scenario_name):
    """Run a specific clinical scenario test"""
    scenario_map = {
        "warfarin": test_direct_medication_interactions,
        "interaction": test_direct_medication_interactions,
        "nsaid": test_direct_nsaid_duplication,
        "duplicate": test_direct_nsaid_duplication,
        "pregnancy": test_direct_pregnancy_contraindications
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
    success, _ = scenario_map[matching_scenario](stub)
    return success

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
