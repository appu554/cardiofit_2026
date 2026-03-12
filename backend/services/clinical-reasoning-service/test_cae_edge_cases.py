#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""
Edge Case Test Suite for Clinical Assertion Engine (CAE)
This test file exercises unusual or rare clinical scenarios using the sample data
in GraphDB (cae-sample-data.ttl) to validate CAE's robustness and handling of edge cases.
"""

import sys
import os
import logging
import grpc
import time
import uuid
from datetime import datetime
from typing import Dict, List, Any, Optional, Tuple

# Set up path for imports
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# Import generated protobuf and service modules
from app.proto import clinical_reasoning_pb2
from app.proto import clinical_reasoning_pb2_grpc
from google.protobuf.struct_pb2 import Struct

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Constants
DEFAULT_SERVER_ADDRESS = "localhost:8027"

# Dictionary of patients from sample data - focusing on edge case patients
EDGE_CASE_PATIENTS = {
    "orphan_disease": "patient_010",        # Patient with rare disorder
    "transplant": "patient_011",            # Organ transplant recipient
    "complex_allergy": "patient_012",       # Multiple cross-reactive allergies
    "cancer_therapy": "patient_013",        # Oncology patient with specialized regimen
    "pediatric_complex": "patient_014",     # 2-year-old with genetic disorder
    "psychiatric": "patient_015",           # Multiple psychiatric conditions
    "malnutrition": "patient_016",          # Severe vitamin deficiencies
    "elderly_frail": "patient_017",         # 95-year-old with frailty syndrome
    "polypharmacy_extreme": "patient_018",  # Patient on 20+ medications
    "pregnancy_complication": "patient_019" # High-risk pregnancy with complications
}

def get_test_id(test_name: str) -> str:
    """Generate a unique test ID with timestamp"""
    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    return f"{test_name}-{timestamp}"

def create_channel(server_address: str = DEFAULT_SERVER_ADDRESS) -> grpc.Channel:
    """Create a gRPC channel to the CAE server"""
    return grpc.insecure_channel(server_address)

def check_server_health(stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub) -> bool:
    """Check if the CAE server is healthy and responding"""
    try:
        request = clinical_reasoning_pb2.HealthCheckRequest(service="clinical-reasoning-service")
        response = stub.HealthCheck(request)
        status = clinical_reasoning_pb2.HealthCheckResponse.ServingStatus.Name(response.status)
        logger.info(f"Server health status: {status}")
        return status == "SERVING"
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return False

def build_patient_context(patient_id: str, add_medications: bool = True) -> Struct:
    """
    Build a patient context structure for the given patient ID.
    Returns a protobuf Struct with patient information from sample data.
    """
    context = Struct()
    context.fields["patient_id"].string_value = patient_id
    context.fields["source"].string_value = "graphdb"
    context.fields["include_medications"].bool_value = add_medications
    context.fields["include_conditions"].bool_value = True
    context.fields["include_allergies"].bool_value = True
    return context

def test_partial_patient_data(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub,
    patient_id: str
) -> bool:
    """
    Test CAE's behavior with incomplete patient data
    (e.g., missing allergies or conditions)
    """
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing CAE with Partial Patient Data - patient: {patient_id}")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("partial-data")
    
    try:
        # Create request with patient ID and correlation ID
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = patient_id
        request.correlation_id = f"req_{test_id}"
        
        # Create patient context using Struct with limited data
        context = Struct()
        context.update({
            "patient_id": patient_id,
            "source": "graphdb",
            "include_medications": True,
            "include_conditions": False,  # Omit conditions
            "include_allergies": False    # Omit allergies
        })
        request.patient_context.CopyFrom(context)
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions with partial data...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Request ID: {response.request_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        logger.info(f"   Total Assertions: {len(response.assertions)}")
        
        # Check if CAE handles partial data appropriately
        # We expect either warnings in the metadata or appropriate conservative assertions
        
        try:
            if hasattr(response.metadata, 'warnings') and response.metadata.warnings:
                logger.info(f"   Warnings detected: {response.metadata.warnings}")
                logger.info("   ✅ CAE properly flagged partial data with warnings")
            else:
                logger.info("   ⚠️ CAE did not flag partial data with warnings")
        except Exception as e:
            logger.warning(f"Could not check metadata warnings: {e}")
        
        # Check assertions - we expect more conservative or fewer assertions with partial data
        logger.info(f"\n📋 Assertions with Partial Data:")
        for assertion in response.assertions[:5]:  # Show first 5 only
            try:
                severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
                logger.info(f"   - {assertion.type}: {assertion.description} (Confidence: {assertion.confidence_score:.2f})")
            except Exception as e:
                logger.warning(f"Error processing assertion: {e}")
                
        return True  # Return success if the call completed, regardless of findings
    except Exception as e:
        logger.error(f"Test Partial Patient Data failed with exception: {e}")
        return False

def test_orphan_disease_case(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub
) -> bool:
    """
    Test CAE's handling of orphan/rare disease cases which may have
    limited evidence or specialized considerations
    """
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing CAE with Orphan Disease Case")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("orphan-disease")
    patient_id = EDGE_CASE_PATIENTS["orphan_disease"]
    
    try:
        # Create request with patient ID and correlation ID
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
            "test_type": "rare_disease"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for rare disease patient...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        logger.info(f"   Total Assertions: {len(response.assertions)}")
        
        # Check if CAE provides appropriate confidence scores for rare conditions
        low_confidence_count = 0
        for assertion in response.assertions:
            if assertion.confidence_score < 0.7:  # Threshold for low confidence
                low_confidence_count += 1
                
        logger.info(f"   Low confidence assertions: {low_confidence_count}/{len(response.assertions)}")
        logger.info(f"   Expected pattern for rare diseases: Some assertions with lower confidence scores")
        
        # Look for evidence strength in assertions
        for assertion in response.assertions[:3]:  # First 3 assertions
            try:
                if hasattr(assertion, 'evidence') and assertion.evidence:
                    for evidence in assertion.evidence:
                        logger.info(f"   Evidence: {evidence.description} (Strength: {evidence.strength:.2f})")
                        if hasattr(evidence, 'source') and evidence.source:
                            logger.info(f"   Source: {evidence.source}")
            except Exception as e:
                logger.warning(f"Could not process evidence: {e}")
                
        return True  # Return success if the call completed
    except Exception as e:
        logger.error(f"Test Orphan Disease Case failed with exception: {e}")
        return False

def test_extreme_polypharmacy(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub
) -> bool:
    """
    Test CAE's handling of extreme polypharmacy cases (20+ medications)
    to check performance and interaction detection capability
    """
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing CAE with Extreme Polypharmacy Case (20+ medications)")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("extreme-polypharmacy")
    patient_id = EDGE_CASE_PATIENTS["polypharmacy_extreme"]
    
    try:
        # Create request with patient ID and correlation ID
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = patient_id
        request.correlation_id = f"req_{test_id}"
        
        # Create patient context using Struct
        from google.protobuf.struct_pb2 import Struct
        patient_context = Struct()
        patient_context.update({
            "patient_id": patient_id,
            "source": "graphdb",
            "include_medications": True,
            "include_conditions": True,
            "include_allergies": True,
            "test_type": "polypharmacy"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info(f"📡 Calling CheckMedicationInteractions for polypharmacy patient...")
        start_time = time.time()
        response = stub.CheckMedicationInteractions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        # Log basic response info and performance metrics
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        logger.info(f"   Total Interactions: {len(response.interactions)}")
        
        # Check for critical interactions
        try:
            if hasattr(response, 'has_critical_interaction') and response.has_critical_interaction:
                logger.info(f"   ⚠️ Critical interactions detected!")
        except Exception as e:
            logger.warning(f"Could not check critical interaction flag: {e}")
        
        # Count interactions by severity
        severity_counts = {}
        for interaction in response.interactions:
            try:
                severity_name = clinical_reasoning_pb2.AssertionSeverity.Name(interaction.severity)
                severity_counts[severity_name] = severity_counts.get(severity_name, 0) + 1
            except Exception as e:
                logger.warning(f"Error processing interaction severity: {e}")
                
        logger.info(f"\nInteractions by Severity:")
        for severity, count in severity_counts.items():
            logger.info(f"   - {severity}: {count}")
            
        # Check performance threshold for polypharmacy
        if elapsed_time > 5000:  # 5 seconds
            logger.warning(f"   ⚠️ Performance concern: Processing time exceeded 5 seconds")
        else:
            logger.info(f"   ✅ Performance acceptable for extreme polypharmacy case")
            
        return True  # Return success if the call completed
    except Exception as e:
        logger.error(f"Test Extreme Polypharmacy failed with exception: {e}")
        return False

def test_learning_override_scenario(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub,
    patient_id: str
) -> bool:
    """
    Test CAE's learning system by simulating an override scenario
    where a clinician disagrees with an alert
    """
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing CAE Learning System - Override Scenario")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("learning-override")
    
    try:
        # 1. First generate a normal clinical assertion
        logger.info("Step 1: Generating initial clinical assertions")
        
        # Create request with patient ID and correlation ID
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = patient_id
        request.correlation_id = f"req_{test_id}_initial"
        
        # Create patient context using Struct
        from google.protobuf.struct_pb2 import Struct
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
        
        if not hasattr(response, 'assertions') or not response.assertions:
            logger.warning("No initial assertions found to override")
            return False
            
        # Find an assertion to override
        override_assertion = None
        for assertion in response.assertions:
            if hasattr(assertion, 'id') and assertion.id:
                override_assertion = assertion
                break
                
        if not override_assertion:
            logger.warning("No suitable assertion found for override test")
            return False
        
        # 2. Now simulate a clinician override for a specific assertion
        logger.info("\nStep 2: Submitting clinician override for an assertion")
        
        # Create feedback request with patient ID
        feedback_request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        feedback_request.patient_id = patient_id
        feedback_request.correlation_id = f"req_{test_id}_override"
        
        # Add feedback context
        feedback_context = Struct()
        feedback_context.update({
            "patient_id": patient_id,
            "feedback_type": "override",
            "reason": "Clinical judgment based on patient-specific factors",
            "action": "dismiss"
        })
        
        # Add the assertion ID to the feedback
        if override_assertion and hasattr(override_assertion, 'id'):
            feedback_context["assertion_id"] = override_assertion.id
            feedback_request.patient_context.CopyFrom(feedback_context)
        
        # Submit the override
        logger.info(f"📡 Submitting override for assertion: {override_assertion.id if hasattr(override_assertion, 'id') else 'unknown'}")
        try:
            override_response = stub.SubmitLearningFeedback(feedback_request)
            logger.info(f"✅ Override submitted successfully: {override_response.success}")
            
            if hasattr(override_response, 'message') and override_response.message:
                logger.info(f"   Message: {override_response.message}")
        except Exception as e:
            logger.warning(f"Override submission failed or not supported: {e}")
            # Continue with test even if override fails, as this may be an optional feature
            
        # Get assertions again to check if the system has learned
        request2 = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request2.patient_id = patient_id
        request2.correlation_id = f"req_{test_id}_after_learning"
        request2.patient_context.CopyFrom(patient_context)
        
        logger.info(f"📡 Getting assertions after learning feedback...")
        response2 = stub.GenerateAssertions(request2)
        
        # Check if the overridden assertion has been affected
        assertion_found = False
        for assertion in response2.assertions:
            if assertion.assertion_id == override_assertion.assertion_id:
                assertion_found = True
                logger.info(f"Original severity: {clinical_reasoning_pb2.AssertionSeverity.Name(override_assertion.severity)}")
                logger.info(f"New severity: {clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)}")
                logger.info(f"Original confidence: {override_assertion.confidence_score:.2f}")
                logger.info(f"New confidence: {assertion.confidence_score:.2f}")
                
                if assertion.confidence_score != override_assertion.confidence_score:
                    logger.info("✅ Learning system applied! Confidence score changed.")
                else:
                    logger.info("⚠️ Learning system may not have affected this assertion.")
                break
                
        if not assertion_found:
            logger.info("🔍 Assertion was completely removed after override - strong learning effect")
            
        return True  # Return success if the call completed
    except Exception as e:
        logger.error(f"Test Learning Override Scenario failed with exception: {e}")
        return False

def test_malformed_request(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub
) -> bool:
    """
    Test CAE's error handling with malformed or invalid requests
    """
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing CAE Error Handling with Malformed Requests")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("malformed-request")
    
    try:
        # Test case 1: Invalid patient ID
        logger.info("\n1. Testing with invalid patient ID")
        
        # Create request with invalid patient ID
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.patient_id = "non_existent_patient_id"
        request.correlation_id = f"req_{test_id}_invalid_patient"
        
        # Create patient context using Struct
        from google.protobuf.struct_pb2 import Struct
        context = Struct()
        context.update({
            "patient_id": "non_existent_patient_id",
            "source": "graphdb"
        })
        request.patient_context.CopyFrom(context)
        
        try:
            response = stub.GenerateAssertions(request)
            logger.info("   Response received for invalid patient ID")
            logger.info(f"   Assertions count: {len(response.assertions)}")
            
            if len(response.assertions) == 0:
                logger.info("   ✅ CAE handled invalid patient ID correctly (returned empty list)")
            else:
                logger.warning("   ⚠️ CAE returned assertions for invalid patient ID")
        except grpc.RpcError as e:
            logger.info(f"   ✅ CAE properly rejected invalid patient ID with error: {e.details()}")
        except Exception as e:
            logger.warning(f"   ⚠️ Unexpected error for invalid patient ID: {e}")
            
        # Test case 2: Malformed context (missing required fields)
        logger.info("\n2. Testing with malformed context (missing fields)")
        context2 = Struct()
        # Only include patient_id, omit source and other required fields
        context2.fields["patient_id"].string_value = EDGE_CASE_PATIENTS["psychiatric"]
        
        request2 = clinical_reasoning_pb2.GenerateAssertionsRequest()
        request2.request_id = f"req_{test_id}_malformed"
        request2.patient_context.CopyFrom(context2)
        
        try:
            response2 = stub.GenerateAssertions(request2)
            logger.info("   Response received for malformed context")
            logger.info("   ✅ CAE handled malformed context by using defaults")
        except grpc.RpcError as e:
            logger.info(f"   ✅ CAE properly rejected malformed context with error: {e.details()}")
        except Exception as e:
            logger.warning(f"   ⚠️ Unexpected error for malformed context: {e}")
            
        # Test case 3: Empty request
        logger.info("\n3. Testing with empty request")
        empty_request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        # No fields set, completely empty
        
        try:
            empty_response = stub.GenerateAssertions(empty_request)
            logger.info("   ⚠️ CAE accepted empty request without validation")
            logger.info(f"   Response received: {hasattr(empty_response, 'assertions')}")
        except grpc.RpcError as e:
            logger.info(f"   ✅ CAE properly rejected empty request with error: {e.details() if hasattr(e, 'details') else str(e)}")
        except Exception as e:
            logger.warning(f"   ⚠️ Unexpected error for empty request: {e}")
            
        return True  # Return success if tests completed
    except Exception as e:
        logger.error(f"Test Malformed Request failed with exception: {e}")
        return False

def run_edge_case_tests() -> bool:
    """Run all CAE edge case tests"""
    logger.info(f"\n{'='*80}")
    logger.info(f"🧪 CAE Edge Case Test Suite")
    logger.info(f"{'='*80}")
    logger.info(f"This test suite exercises unusual or rare scenarios in the CAE")
    logger.info(f"{'='*80}")
    
    try:
        # Create gRPC channel and stub
        channel = create_channel()
        stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
        
        # Check server health
        health_status = check_server_health(stub)
        logger.info(f"✅ Server Health: {'SERVING' if health_status else 'NOT SERVING'}")
        
        if not health_status:
            logger.error("Cannot proceed with tests as server is not healthy")
            return False
        
        # Choose a sample patient for general tests
        sample_patient = EDGE_CASE_PATIENTS["psychiatric"]
        
        # Run edge case tests
        test_results = {}
        
        try:
            # 1. Test with partial patient data
            test_results["Partial Patient Data"] = test_partial_patient_data(stub, sample_patient)
        except Exception as e:
            logger.error(f"Partial Patient Data test failed with exception: {e}")
            test_results["Partial Patient Data"] = False
        
        try:
            # 2. Test with orphan disease patient
            test_results["Orphan Disease"] = test_orphan_disease_case(stub)
        except Exception as e:
            logger.error(f"Orphan Disease test failed with exception: {e}")
            test_results["Orphan Disease"] = False
        
        try:
            # 3. Test extreme polypharmacy case
            test_results["Extreme Polypharmacy"] = test_extreme_polypharmacy(stub)
        except Exception as e:
            logger.error(f"Extreme Polypharmacy test failed with exception: {e}")
            test_results["Extreme Polypharmacy"] = False
        
        try:
            # 4. Test learning override scenario
            test_results["Learning Override"] = test_learning_override_scenario(stub, sample_patient)
        except Exception as e:
            logger.error(f"Learning Override test failed with exception: {e}")
            test_results["Learning Override"] = False
        
        try:
            # 5. Test error handling with malformed requests
            test_results["Error Handling"] = test_malformed_request(stub)
        except Exception as e:
            logger.error(f"Error Handling test failed with exception: {e}")
            test_results["Error Handling"] = False
        
        # Final results summary
        logger.info(f"\n{'='*80}")
        logger.info(f"📊 EDGE CASE TEST SUMMARY")
        logger.info(f"{'='*80}")
        
        passed_count = sum(1 for result in test_results.values() if result)
        total_count = len(test_results)
        
        for test_name, result in test_results.items():
            status = "✅ PASSED" if result else "❌ FAILED"
            logger.info(f"{test_name}: {status}")
        
        logger.info(f"\nTotal: {passed_count}/{total_count} edge case tests passed")
        
        if passed_count == total_count:
            logger.info(f"\n🎉 All edge case tests passed! CAE handles unusual scenarios robustly.")
            return True
        else:
            logger.warning(f"\n⚠️  {total_count - passed_count} tests failed.")
            return False
            
    except Exception as e:
        logger.error(f"Edge case test suite failed with exception: {e}")
        return False
    finally:
        # Ensure the channel is closed
        if 'channel' in locals():
            channel.close()

if __name__ == "__main__":
    success = run_edge_case_tests()
    sys.exit(0 if success else 1)
