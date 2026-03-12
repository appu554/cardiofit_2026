#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CAE Improved Clinical Scenario Tests

This script tests the CAE service with direct medication data using the format
from the working test_cae_real_patient.py.
"""

import os
import sys
import time
import logging
import grpc
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

def test_health_check(stub):
    """Test basic health check to ensure service is responding"""
    logger.info("\n" + "="*70)
    logger.info("Testing CAE Health Check")
    logger.info("="*70)

    try:
        # Create health check request
        request = clinical_reasoning_pb2.HealthCheckRequest()
        
        # Call the service
        logger.info("📡 Calling HealthCheck...")
        response = stub.HealthCheck(request)
        
        # Log response
        logger.info(f"✅ Server Health: {response.status}")
        return True
    except Exception as e:
        logger.error(f"❌ Health check failed with exception: {e}")
        return False

def test_warfarin_drug_interactions(stub):
    """
    Test warfarin drug interactions with direct medication IDs
    """
    logger.info("\n" + "="*70)
    logger.info("SCENARIO: Warfarin Drug Interactions Test")
    logger.info("="*70)
    logger.info("Medications: warfarin + aspirin + ibuprofen")
    logger.info("Expected: Bleeding risk interaction alerts")
    logger.info("="*70)
    
    test_id = f"warfarin-interaction-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    
    try:
        # Create request with test patient ID
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = "test-patient-warfarin"  # Placeholder ID
        request.correlation_id = f"req_{test_id}"
        
        # Add medications directly using medication_ids
        request.medication_ids.extend([
            "warfarin",  # Anticoagulant 
            "aspirin",   # Antiplatelet - should interact with warfarin
            "ibuprofen"  # NSAID - should interact with both warfarin and aspirin
        ])
        
        # Add relevant conditions
        request.condition_ids.extend([
            "atrial_fibrillation"  # Common indication for warfarin
        ])
        
        # Create patient context
        patient_context = Struct()
        patient_context.update({
            "request_source": "test_suite",
            "test_type": "drug_interaction_test",
            "include_graphdb_data": True  # This appears to be important
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Set priority - used in working test
        request.priority = clinical_reasoning_pb2.AssertionPriority.PRIORITY_URGENT
        
        # Request specific reasoner types - this was key in working test
        request.reasoner_types.extend([
            "interaction",  # Critical for drug interaction tests
            "dosing", 
            "contraindication", 
            "duplicate_therapy"
        ])
        
        logger.info(f"📡 Calling GenerateAssertions for warfarin interactions...")
        
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        logger.info(f"   Total Assertions: {len(response.assertions)}")
        
        # Check for interactions
        interaction_count = 0
        
        # Check assertions first
        if hasattr(response, 'assertions') and response.assertions:
            logger.info(f"\n🔎 Found {len(response.assertions)} assertions")
            
            for i, assertion in enumerate(response.assertions, 1):
                try:
                    severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
                    logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled'}")
                    logger.info(f"      Severity: {severity}")
                    logger.info(f"      Type: {assertion.type}")
                    logger.info(f"      Description: {assertion.description}")
                    
                    # Count interaction type assertions
                    if assertion.type.lower() == "interaction":
                        interaction_count += 1
                except Exception as e:
                    logger.warning(f"Error accessing assertion fields: {e}")
        
        # Check specific interaction field
        if hasattr(response, 'interactions') and response.interactions:
            interaction_count += len(response.interactions)
            logger.info(f"\n🔎 Found {len(response.interactions)} direct interactions")
            
            for i, interaction in enumerate(response.interactions, 1):
                try:
                    severity = clinical_reasoning_pb2.AssertionSeverity.Name(interaction.severity)
                    logger.info(f"\n   {i}. {interaction.medication_a} + {interaction.medication_b}")
                    logger.info(f"      Severity: {severity}")
                    logger.info(f"      Description: {interaction.description if hasattr(interaction, 'description') else 'No description'}")
                except Exception as e:
                    logger.warning(f"Error accessing interaction fields: {e}")
        
        # Check if any assertion mentions bleeding risk or anticoagulant effects
        bleeding_risk_found = False
        if hasattr(response, 'assertions') and response.assertions:
            for assertion in response.assertions:
                if hasattr(assertion, 'description'):
                    desc = assertion.description.lower()
                    if ('bleeding' in desc or 'anticoagulant' in desc):
                        bleeding_risk_found = True
                        logger.info(f"\n✅ Found bleeding risk assertion: {assertion.description}")
        
        if interaction_count > 0 or bleeding_risk_found:
            logger.info(f"\n✅ TEST PASSED: Found drug interaction evidence (direct interactions: {interaction_count}, bleeding risk assertions: {bleeding_risk_found})")
            return True
        else:
            logger.warning("\n❌ TEST FAILED: No medication interactions detected")
            return False
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False

def test_nsaid_duplication(stub):
    """
    Test duplicate NSAID therapy alert
    """
    logger.info("\n" + "="*70)
    logger.info("SCENARIO: NSAID Therapeutic Duplication Test")
    logger.info("="*70)
    logger.info("Medications: ibuprofen + naproxen (both NSAIDs)")
    logger.info("Expected: Duplicate therapeutic class alert")
    logger.info("="*70)
    
    test_id = f"nsaid-duplication-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    
    try:
        # Create request
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = "test-patient-nsaid"
        request.correlation_id = f"req_{test_id}"
        
        # Add medications directly using medication_ids
        request.medication_ids.extend([
            "ibuprofen",  # NSAID
            "naproxen"    # NSAID - should trigger duplication alert
        ])
        
        # Add some conditions
        request.condition_ids.extend([
            "osteoarthritis",  # Common indication for NSAIDs
            "joint_pain" 
        ])
        
        # Create patient context
        patient_context = Struct()
        patient_context.update({
            "request_source": "test_suite",
            "test_type": "duplicate_therapy_test",
            "include_graphdb_data": True
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Set priority
        request.priority = clinical_reasoning_pb2.AssertionPriority.PRIORITY_URGENT
        
        # Request specific reasoner types - especially duplicate_therapy
        request.reasoner_types.extend([
            "duplicate_therapy",  # Critical for this test
            "interaction",
            "dosing",
            "contraindication"
        ])
        
        logger.info(f"📡 Calling GenerateAssertions for duplicate therapy...")
        
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        logger.info(f"   Total Assertions: {len(response.assertions)}")
        
        # Check for duplicate therapy assertions
        duplication_found = False
        nsaid_found = False
        
        if hasattr(response, 'assertions') and response.assertions:
            logger.info(f"\n🔎 Found {len(response.assertions)} assertions")
            
            for i, assertion in enumerate(response.assertions, 1):
                try:
                    severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
                    logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled'}")
                    logger.info(f"      Type: {assertion.type}")
                    logger.info(f"      Description: {assertion.description}")
                    
                    # Check if this is a duplicate therapy alert
                    if hasattr(assertion, 'description'):
                        desc = assertion.description.lower()
                        if (assertion.type.lower() == "duplicate_therapy" or 
                            "duplicate" in desc or 
                            ("nsaid" in desc and ("ibuprofen" in desc or "naproxen" in desc))):
                            duplication_found = True
                            logger.info("      ✅ NSAID duplication detected!")
                        elif "ibuprofen" in desc and "naproxen" in desc:
                            nsaid_found = True
                            logger.info("      ✅ Multiple NSAIDs mentioned")
                except Exception as e:
                    logger.warning(f"Error accessing assertion fields: {e}")
        
        if duplication_found or nsaid_found:
            logger.info("\n✅ TEST PASSED: Multiple NSAIDs detected")
            return True
        else:
            logger.warning("\n❌ TEST FAILED: No duplicate NSAID therapy alert detected")
            return False
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False

def test_pregnancy_contraindications(stub):
    """
    Test pregnancy contraindications
    """
    logger.info("\n" + "="*70)
    logger.info("SCENARIO: Pregnancy Contraindications Test")
    logger.info("="*70)
    logger.info("Patient: 28-year-old pregnant female")
    logger.info("Medications: warfarin (contraindicated in pregnancy)")
    logger.info("Expected: Pregnancy contraindication alert")
    logger.info("="*70)
    
    test_id = f"pregnancy-contraindication-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    
    try:
        # Create request
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = "test-patient-pregnant"
        request.correlation_id = f"req_{test_id}"
        
        # Add medications directly using medication_ids
        request.medication_ids.extend([
            "warfarin"  # Contraindicated in pregnancy
        ])
        
        # Add pregnancy condition directly
        request.condition_ids.extend([
            "pregnancy",
            "first_trimester_pregnancy"
        ])
        
        # Create patient context with demographic information
        patient_context = Struct()
        patient_context.update({
            "request_source": "test_suite",
            "test_type": "contraindication_test",
            "include_graphdb_data": True,
            "patient": {
                "gender": "female",
                "age": 28,
                "pregnant": True
            }
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Set priority
        request.priority = clinical_reasoning_pb2.AssertionPriority.PRIORITY_URGENT
        
        # Request specific reasoner types - especially contraindication
        request.reasoner_types.extend([
            "contraindication",  # Critical for this test
            "interaction",
            "dosing",
            "duplicate_therapy"
        ])
        
        logger.info(f"📡 Calling GenerateAssertions for pregnancy contraindications...")
        
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        logger.info("\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        logger.info(f"   Total Assertions: {len(response.assertions)}")
        
        # Check for contraindication assertions
        contraindication_found = False
        
        if hasattr(response, 'assertions') and response.assertions:
            logger.info(f"\n🔎 Found {len(response.assertions)} assertions")
            
            for i, assertion in enumerate(response.assertions, 1):
                try:
                    if hasattr(assertion, 'severity'):
                        severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled'}")
                        logger.info(f"      Type: {assertion.type}")
                        logger.info(f"      Description: {assertion.description}")
                    
                    # Check if this is a pregnancy contraindication alert - more flexible check
                    if hasattr(assertion, 'description'):
                        desc = assertion.description.lower()
                        if ((assertion.type.lower() == "contraindication" or assertion.type.lower() == "absolute") and 
                            ("pregnancy" in desc or "pregnant" in desc or "teratogenic" in desc)):
                            contraindication_found = True
                            logger.info("      ✅ Pregnancy contraindication detected!")
                except Exception as e:
                    logger.warning(f"Error accessing assertion fields: {e}")
        
        if contraindication_found:
            logger.info("\n✅ TEST PASSED: Pregnancy contraindication alert detected")
            return True
        else:
            logger.warning("\n❌ TEST FAILED: No pregnancy contraindication alerts detected")
            return False
            
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False

if __name__ == "__main__":
    # Create a gRPC channel
    channel = grpc.insecure_channel(CAE_SERVER)
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Run tests
    logger.info("\n" + "="*70)
    logger.info("🏥 CAE IMPROVED SCENARIO TEST SUITE")
    logger.info("="*70)
    
    # Test health check first
    health_result = test_health_check(stub)
    
    if health_result:
        # Run all test scenarios
        results = {}
        
        # Run warfarin interactions test
        results["Warfarin Interactions"] = test_warfarin_drug_interactions(stub)
        
        # Run NSAID duplication test
        results["NSAID Duplication"] = test_nsaid_duplication(stub)
        
        # Run pregnancy contraindications test
        results["Pregnancy Contraindications"] = test_pregnancy_contraindications(stub)
        
        # Display final test summary
        logger.info("\n" + "="*70)
        logger.info("📊 FINAL IMPROVED SCENARIO TEST SUMMARY")
        logger.info("="*70)
        
        passed = 0
        for test_name, result in results.items():
            status = "✅ PASSED" if result else "❌ FAILED"
            logger.info(f"{test_name}: {status}")
            if result:
                passed += 1
        
        logger.info(f"\nTotal: {passed}/{len(results)} scenarios passed")
        
        if passed == 0:
            logger.warning("""
            ⚠️ All improved tests still failed. This suggests one of the following issues:
            
            1. The CAE knowledge base may not include rules for these specific clinical scenarios
            2. The medication and condition IDs used may not match what CAE expects
            3. The CAE service may require additional configuration
            
            Consider trying the tests with the exact same medications used in test_cae_real_patient.py
            and comparing the request structures more carefully.
            """)
        elif passed < len(results):
            logger.warning(f"""
            ⚠️ {len(results) - passed} improved tests failed.
            
            This suggests that the CAE service is working but may not have rules implemented
            for all clinical scenarios we're testing.
            """)
        else:
            logger.info("""
            🎉 All improved tests passed!
            
            The key differences that made these tests work were:
            1. Using medication_ids and condition_ids directly (not in Struct)
            2. Explicitly specifying reasoner_types
            3. Setting priority to PRIORITY_URGENT
            4. Including include_graphdb_data=True in patient_context
            """)
    else:
        logger.error("Health check failed. Please ensure the CAE service is running.")
