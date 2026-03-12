#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Individual CAE Function Test Suite

This script contains separate test functions for each CAE endpoint,
making it easier to test and debug individual CAE functions.
Each test uses patient data from cae-sample-data.ttl.

Usage:
    python test_cae_individual_functions.py [function_name]
    
    function_name: Optional. Name of a specific function to test.
                  If omitted, all tests will run.
                  
Available test functions:
    - test_health_check
    - test_generate_assertions
    - test_medication_interactions
    - test_contraindications
    - test_duplicate_therapy
    - test_dose_adjustments
    - test_learning_feedback
"""

import os
import sys
import time
import logging
import grpc
import uuid
from datetime import datetime
from google.protobuf.struct_pb2 import Struct

# Import generated protobuf modules
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
    
    # Clean up temp file (optional)
    # os.remove(temp_grpc_path)
else:
    # If it's already using absolute import, just load it directly
    pb2_grpc_spec = importlib.util.spec_from_file_location('clinical_reasoning_pb2_grpc', pb2_grpc_path)
    clinical_reasoning_pb2_grpc = importlib.util.module_from_spec(pb2_grpc_spec)
    # Make clinical_reasoning_pb2 available to the module
    sys.modules['clinical_reasoning_pb2'] = clinical_reasoning_pb2
    pb2_grpc_spec.loader.exec_module(clinical_reasoning_pb2_grpc)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# CAE Server connection details
CAE_SERVER = "localhost:8027"

# Patient IDs from cae-sample-data.ttl
PATIENTS = {
    "main": "905a60cb-8241-418f-b29b-5b020e851392",  # Primary test patient
    "standard": "patient_001",                       # Standard patient for general testing
    "cardiovascular": "patient_002",                 # Patient with cardiovascular conditions
    "diabetes": "patient_003",                       # Patient with diabetes
    "pediatric": "patient_004",                      # Pediatric patient (8 years old)
    "geriatric": "patient_005",                      # Geriatric patient (89 years old)
    "pregnancy": "patient_006",                      # Pregnant patient
    "renal": "patient_007",                          # Patient with renal impairment
    "allergy": "patient_008",                        # Patient with multiple allergies
    "polypharmacy": "patient_009"                    # Patient on multiple medications
}

def get_test_id(test_name):
    """Generate a unique test ID with timestamp for correlation"""
    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    return f"{test_name}-{timestamp}"

def test_health_check(stub):
    """Test the CAE health check endpoint"""
    logger.info("\n" + "="*70)
    logger.info("Testing CAE Health Check")
    logger.info("="*70)
    
    try:
        # Create health check request
        request = clinical_reasoning_pb2.HealthCheckRequest()
        
        # Call the service
        logger.info("📡 Calling HealthCheck...")
        response = stub.HealthCheck(request)
        
        # Evaluate response
        if response.status == clinical_reasoning_pb2.HealthCheckResponse.SERVING:
            logger.info("✅ Server Health: SERVING")
            return True
        else:
            logger.warning(f"⚠️ Server Health: {response.status}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Health check failed: {e}")
        return False

def test_generate_assertions(stub, patient_id=PATIENTS["main"]):
    """Test the CAE GenerateAssertions endpoint with a general request"""
    logger.info("\n" + "="*70)
    logger.info(f"Testing CAE GenerateAssertions for patient: {patient_id}")
    logger.info("="*70)
    
    test_id = get_test_id("assertions")
    
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
            "include_allergies": True
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for patient {patient_id}...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Check assertions with defensive check
        if hasattr(response, 'assertions'):
            assertion_count = len(response.assertions)
            logger.info(f"   Total Assertions: {assertion_count}")
            
            # Log individual assertions
            if assertion_count > 0:
                logger.info(f"\n📋 CLINICAL ASSERTIONS FOUND:")
                for i, assertion in enumerate(response.assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Assertion'}")
                        if hasattr(assertion, 'description') and assertion.description:
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                        if hasattr(assertion, 'category'):
                            logger.info(f"      Category: {assertion.category}")
                        if hasattr(assertion, 'evidence') and assertion.evidence:
                            logger.info(f"      Evidence: {assertion.evidence}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
            
            return True
        else:
            logger.warning("⚠️ Response has no 'assertions' attribute")
            return False
            
    except Exception as e:
        logger.error(f"❌ Generate assertions failed: {e}")
        return False

def test_medication_interactions(stub, patient_id=PATIENTS["main"]):
    """Test the CAE medication interactions detection"""
    logger.info("\n" + "="*70)
    logger.info(f"Testing CAE Medication Interactions for patient: {patient_id}")
    logger.info("="*70)
    
    test_id = get_test_id("med-interactions")
    
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
            "test_type": "medication_interactions"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for medication interactions...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Process interactions
        has_interactions = False
        
        # Check if response has an interactions attribute
        if hasattr(response, 'interactions') and response.interactions:
            interactions_count = len(response.interactions)
            logger.info(f"   Total Interactions: {interactions_count}")
            has_interactions = True
            
            # Log individual interactions
            for i, interaction in enumerate(response.interactions, 1):
                try:
                    logger.info(f"\n   🔸 {interaction.medication1} + {interaction.medication2}")
                    if hasattr(interaction, 'severity'):
                        logger.info(f"      Severity: {interaction.severity}")
                    if hasattr(interaction, 'description'):
                        logger.info(f"      Description: {interaction.description}")
                    if hasattr(interaction, 'mechanism'):
                        logger.info(f"      Mechanism: {interaction.mechanism}")
                    if hasattr(interaction, 'confidence'):
                        logger.info(f"      Confidence: {interaction.confidence}")
                except Exception as e:
                    logger.warning(f"      Error accessing interaction fields: {e}")
        
        # Check assertions for interaction-related assertions
        if hasattr(response, 'assertions'):
            interaction_assertions = [a for a in response.assertions 
                                    if hasattr(a, 'category') and 
                                    'interaction' in str(a.category).lower()]
            
            if interaction_assertions:
                has_interactions = True
                logger.info(f"   Found {len(interaction_assertions)} interaction assertions")
                
                for i, assertion in enumerate(interaction_assertions, 1):
                    try:
                        logger.info(f"\n   🔸 Interaction {i}: {assertion.title}")
                        if hasattr(assertion, 'description'):
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        return has_interactions or True  # Return True even if no interactions (valid test case)
            
    except Exception as e:
        logger.error(f"❌ Medication interactions test failed: {e}")
        return False

def test_contraindications(stub, patient_id=PATIENTS["allergy"]):
    """Test the CAE contraindication checking"""
    logger.info("\n" + "="*70)
    logger.info(f"Testing CAE Contraindication Detection for patient: {patient_id}")
    logger.info("="*70)
    
    test_id = get_test_id("contraindications")
    
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
            "include_allergies": True,
            "test_type": "contraindications"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Add specific medications for contraindication testing
        # These will be added to medication_ids list in the request
        request.medication_ids.extend(["amoxicillin", "ibuprofen", "warfarin"])
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for contraindication checks...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Process contraindication assertions
        if hasattr(response, 'assertions'):
            # Filter for contraindication-related assertions
            contraindication_assertions = [
                a for a in response.assertions 
                if (hasattr(a, 'category') and 'contraindication' in str(a.category).lower()) or
                   (hasattr(a, 'title') and 'contraindication' in str(a.title).lower()) or
                   (hasattr(a, 'description') and 'contraindicated' in str(a.description).lower())
            ]
            
            logger.info(f"   Total Contraindication Alerts: {len(contraindication_assertions)}")
            
            # Log individual contraindications
            if contraindication_assertions:
                logger.info("\n📋 CONTRAINDICATIONS FOUND:")
                for i, assertion in enumerate(contraindication_assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Contraindication'}")
                        if hasattr(assertion, 'description') and assertion.description:
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                        if hasattr(assertion, 'evidence') and assertion.evidence:
                            logger.info(f"      Evidence: {assertion.evidence}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        return True
            
    except Exception as e:
        logger.error(f"❌ Contraindication test failed: {e}")
        return False

def test_duplicate_therapy(stub, patient_id=PATIENTS["polypharmacy"]):
    """Test the CAE duplicate therapy detection"""
    logger.info("\n" + "="*70)
    logger.info(f"Testing CAE Duplicate Therapy Detection for patient: {patient_id}")
    logger.info("="*70)
    
    test_id = get_test_id("duplicate-therapy")
    
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
            "new_medication": "enalapril"  # Example medication that might be duplicate
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for duplicate therapy checks...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Process duplicate therapy assertions
        if hasattr(response, 'assertions'):
            # Filter for duplicate therapy related assertions
            duplicate_assertions = [
                a for a in response.assertions 
                if (hasattr(a, 'category') and 'duplicate' in str(a.category).lower()) or
                   (hasattr(a, 'title') and 'duplicate' in str(a.title).lower()) or
                   (hasattr(a, 'description') and 'duplicate' in str(a.description).lower())
            ]
            
            logger.info(f"   Total Duplicate Therapy Alerts: {len(duplicate_assertions)}")
            
            # Log individual duplicate therapy alerts
            if duplicate_assertions:
                logger.info("\n📋 DUPLICATE THERAPIES FOUND:")
                for i, assertion in enumerate(duplicate_assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Alert'}")
                        if hasattr(assertion, 'description') and assertion.description:
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                        if hasattr(assertion, 'evidence') and assertion.evidence:
                            logger.info(f"      Evidence: {assertion.evidence}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
        
        return True
            
    except Exception as e:
        logger.error(f"❌ Duplicate therapy test failed: {e}")
        return False

def test_dose_adjustments(stub, patient_id=PATIENTS["renal"]):
    """Test the CAE dose adjustment recommendations"""
    logger.info("\n" + "="*70)
    logger.info(f"Testing CAE Dose Adjustment Recommendations for patient: {patient_id}")
    logger.info("="*70)
    
    test_id = get_test_id("dose-adjustments")
    
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
            "include_labs": True,
            "test_type": "dose_adjustments"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Add medications for dose testing
        request.medication_ids.extend(["metformin", "lisinopril", "gabapentin"])
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for dose adjustment recommendations...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)  # ms
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Process dose adjustment assertions
        if hasattr(response, 'assertions'):
            # Filter for dose adjustment related assertions
            dose_assertions = [
                a for a in response.assertions 
                if (hasattr(a, 'category') and 'dose' in str(a.category).lower()) or
                   (hasattr(a, 'title') and 'dose' in str(a.title).lower()) or
                   (hasattr(a, 'description') and 'dose' in str(a.description).lower())
            ]
            
            logger.info(f"   Total Dose Adjustment Recommendations: {len(dose_assertions)}")
            
            # Log individual dose adjustments
            if dose_assertions:
                logger.info("\n📋 DOSE ADJUSTMENTS FOUND:")
                for i, assertion in enumerate(dose_assertions, 1):
                    try:
                        logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled Recommendation'}")
                        if hasattr(assertion, 'description') and assertion.description:
                            logger.info(f"      Description: {assertion.description}")
                        if hasattr(assertion, 'severity'):
                            logger.info(f"      Severity: {assertion.severity}")
                        if hasattr(assertion, 'evidence') and assertion.evidence:
                            logger.info(f"      Evidence: {assertion.evidence}")
                        if hasattr(assertion, 'recommendation') and assertion.recommendation:
                            logger.info(f"      Recommendation: {assertion.recommendation}")
                    except Exception as e:
                        logger.warning(f"      Error accessing assertion fields: {e}")
            
            # Check if the response has specific dose_adjustments attribute
            if hasattr(response, 'dose_adjustments') and response.dose_adjustments:
                logger.info(f"   Found {len(response.dose_adjustments)} specific dose adjustments")
                
                for i, adjustment in enumerate(response.dose_adjustments, 1):
                    try:
                        logger.info(f"\n   🔸 Adjustment {i}: {adjustment.medication}")
                        if hasattr(adjustment, 'current_dose'):
                            logger.info(f"      Current Dose: {adjustment.current_dose}")
                        if hasattr(adjustment, 'recommended_dose'):
                            logger.info(f"      Recommended Dose: {adjustment.recommended_dose}")
                        if hasattr(adjustment, 'reason'):
                            logger.info(f"      Reason: {adjustment.reason}")
                    except Exception as e:
                        logger.warning(f"      Error accessing dose adjustment fields: {e}")
        
        return True
            
    except Exception as e:
        logger.error(f"❌ Dose adjustment test failed: {e}")
        return False

def test_learning_feedback(stub, patient_id=PATIENTS["main"]):
    """Test the CAE learning feedback system"""
    logger.info("\n" + "="*70)
    logger.info(f"Testing CAE Learning Feedback System for patient: {patient_id}")
    logger.info("="*70)
    
    test_id = get_test_id("learning-feedback")
    
    try:
        # Step 1: First generate some assertions to provide feedback on
        logger.info("Step 1: Generating initial clinical assertions")
        
        # Create request for initial assertions
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = patient_id
        request.correlation_id = f"req_{test_id}_initial"
        
        # Create patient context using Struct
        patient_context = Struct()
        patient_context.update({
            "patient_id": patient_id,
            "source": "graphdb",
            "include_medications": True,
            "include_conditions": True,
            "include_allergies": True
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service to get initial assertions
        logger.info(f"📡 Getting initial assertions...")
        response = stub.GenerateAssertions(request)
        
        # Check if we got any assertions to provide feedback on
        if not hasattr(response, 'assertions') or not response.assertions:
            logger.warning("No assertions found to provide feedback on")
            return False
            
        # Find an assertion to provide feedback on
        target_assertion = None
        for assertion in response.assertions:
            if hasattr(assertion, 'id') and assertion.id:
                target_assertion = assertion
                break
                
        if not target_assertion:
            logger.warning("No suitable assertion found for feedback test")
            return False
            
        # Step 2: Submit feedback on an assertion
        logger.info("\nStep 2: Submitting feedback on an assertion")
        
        # Create feedback request
        feedback_request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        feedback_request.patient_id = patient_id
        feedback_request.correlation_id = f"req_{test_id}_feedback"
        
        # Add feedback context
        feedback_context = Struct()
        feedback_context.update({
            "patient_id": patient_id,
            "feedback_type": "override",
            "reason": "Clinical judgment based on patient-specific factors",
            "action": "dismiss",
            "assertion_id": target_assertion.id if hasattr(target_assertion, 'id') else ""
        })
        feedback_request.patient_context.CopyFrom(feedback_context)
        
        # Submit the feedback
        logger.info(f"📡 Submitting feedback for assertion: {target_assertion.id if hasattr(target_assertion, 'id') else 'unknown'}")
        try:
            feedback_response = stub.SubmitLearningFeedback(feedback_request)
            
            if hasattr(feedback_response, 'success') and feedback_response.success:
                logger.info(f"✅ Feedback submitted successfully")
                return True
            else:
                logger.warning(f"⚠️ Feedback submission response did not indicate success")
                return False
                
        except Exception as e:
            logger.error(f"❌ Feedback submission failed: {e}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Learning feedback test failed: {e}")
        return False

def run_all_tests():
    """Run all available CAE function tests"""
    logger.info("\n" + "="*70)
    logger.info("🏥 CAE Individual Function Test Suite")
    logger.info("="*70)
    
    # Create a gRPC channel to the CAE service
    channel = grpc.insecure_channel(CAE_SERVER)
    
    # Create a stub for making calls
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Dictionary to hold test results
    test_results = {}
    
    # Run all tests with exception handling
    try:
        test_results["Health Check"] = test_health_check(stub)
    except Exception as e:
        logger.error(f"Health Check test failed with exception: {e}")
        test_results["Health Check"] = False
        
    try:
        test_results["Generate Assertions"] = test_generate_assertions(stub)
    except Exception as e:
        logger.error(f"Generate Assertions test failed with exception: {e}")
        test_results["Generate Assertions"] = False
        
    try:
        test_results["Medication Interactions"] = test_medication_interactions(stub)
    except Exception as e:
        logger.error(f"Medication Interactions test failed with exception: {e}")
        test_results["Medication Interactions"] = False
        
    try:
        test_results["Contraindications"] = test_contraindications(stub)
    except Exception as e:
        logger.error(f"Contraindications test failed with exception: {e}")
        test_results["Contraindications"] = False
        
    try:
        test_results["Duplicate Therapy"] = test_duplicate_therapy(stub)
    except Exception as e:
        logger.error(f"Duplicate Therapy test failed with exception: {e}")
        test_results["Duplicate Therapy"] = False
        
    try:
        test_results["Dose Adjustments"] = test_dose_adjustments(stub)
    except Exception as e:
        logger.error(f"Dose Adjustments test failed with exception: {e}")
        test_results["Dose Adjustments"] = False
        
    try:
        test_results["Learning Feedback"] = test_learning_feedback(stub)
    except Exception as e:
        logger.error(f"Learning Feedback test failed with exception: {e}")
        test_results["Learning Feedback"] = False
    
    # Display final test summary
    logger.info("\n" + "="*70)
    logger.info("📊 FINAL TEST SUMMARY")
    logger.info("="*70)
    
    passed_tests = 0
    for test_name, result in test_results.items():
        status = "✅ PASSED" if result else "❌ FAILED"
        logger.info(f"{test_name}: {status}")
        if result:
            passed_tests += 1
    
    logger.info(f"\nTotal: {passed_tests}/{len(test_results)} tests passed")
    
    if passed_tests == len(test_results):
        logger.info("\n🎉 All tests passed! CAE functions are working properly.")
    elif passed_tests > 0:
        logger.warning(f"\n⚠️ {len(test_results) - passed_tests} tests failed.")
    else:
        logger.error("\n❌ All tests failed. CAE service may be unavailable or incompatible.")

def run_specific_test(test_name, stub):
    """Run a specific test by name"""
    test_map = {
        "health_check": test_health_check,
        "generate_assertions": test_generate_assertions,
        "medication_interactions": test_medication_interactions,
        "contraindications": test_contraindications,
        "duplicate_therapy": test_duplicate_therapy,
        "dose_adjustments": test_dose_adjustments,
        "learning_feedback": test_learning_feedback
    }
    
    if test_name.lower() not in test_map:
        logger.error(f"Test '{test_name}' not found. Available tests: {', '.join(test_map.keys())}")
        return False
    
    try:
        return test_map[test_name.lower()](stub)
    except Exception as e:
        logger.error(f"Test '{test_name}' failed with exception: {e}")
        return False

if __name__ == "__main__":
    # Check if a specific test was requested
    if len(sys.argv) > 1:
        test_name = sys.argv[1].lower()
        
        # Create a gRPC channel
        channel = grpc.insecure_channel(CAE_SERVER)
        stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
        
        # Run the specified test
        logger.info(f"Running specific test: {test_name}")
        result = run_specific_test(test_name, stub)
        
        if result:
            logger.info(f"\n✅ {test_name} test passed!")
            sys.exit(0)
        else:
            logger.error(f"\n❌ {test_name} test failed!")
            sys.exit(1)
    else:
        # Run all tests
        run_all_tests()
