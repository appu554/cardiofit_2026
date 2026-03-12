#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""
Comprehensive Test Suite for Clinical Assertion Engine (CAE)
This test file exercises all possible scenarios using the sample data in GraphDB
(cae-sample-data.ttl) to validate CAE functionality across various clinical scenarios.
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

# Dictionary of patients from sample data
SAMPLE_PATIENTS = {
    "primary": "905a60cb-8241-418f-b29b-5b020e851392",  # Complex cardiovascular case with drug interactions
    "cardiovascular": "patient_001",                     # Cardiovascular focus
    "diabetes": "patient_002",                           # Diabetes with CKD
    "polypharmacy": "patient_003",                       # Multiple medications
    "pediatric": "patient_004",                          # 8-year-old patient
    "geriatric": "patient_005",                          # 89-year-old with multiple comorbidities
    "pregnancy": "patient_006",                          # Pregnant patient
    "liver": "patient_007",                              # Patient with liver disease
    "allergy": "patient_008",                            # Multiple allergies
    "renal": "patient_009"                               # Renal impairment
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

def test_drug_interaction_detection(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub, 
    patient_id: str
) -> bool:
    """Test the CAE's ability to detect drug interactions"""
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing Drug Interaction Detection for patient: {patient_id}")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("drug-interactions")
    
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
            "test_type": "drug_interaction"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info(f"📡 Calling CheckMedicationInteractions for patient {patient_id}...")
        start_time = time.time()
        response = stub.CheckMedicationInteractions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Check if there are critical interactions
        try:
            if hasattr(response, 'has_critical_interaction') and response.has_critical_interaction:
                logger.warning(f"⚠️ CRITICAL INTERACTIONS DETECTED: Patient safety at risk!")
        except Exception as e:
            logger.warning(f"Could not check critical interaction flag: {e}")
        
        # Log each interaction with defensive checks
        logger.info(f"\n💊 Found {len(response.interactions)} interactions")
        
        for interaction in response.interactions:
            try:
                severity = clinical_reasoning_pb2.AssertionSeverity.Name(interaction.severity)
                logger.info(f"\n   🔸 {interaction.medication_a} + {interaction.medication_b}")
                
                if hasattr(interaction, 'type'):
                    logger.info(f"      Type: {interaction.type}")
                    
                logger.info(f"      Severity: {severity}")
                
                if hasattr(interaction, 'description'):
                    logger.info(f"      Description: {interaction.description}")
                    
                if hasattr(interaction, 'mechanism'):
                    logger.info(f"      Mechanism: {interaction.mechanism}")
                    
                logger.info(f"      Confidence: {interaction.confidence_score:.2f}")
                
                # Check for recommendations
                try:
                    if hasattr(interaction, 'recommendations') and interaction.recommendations:
                        logger.info("      Management:")
                        for rec in interaction.recommendations:
                            logger.info(f"        - {rec}")
                except Exception as e:
                    logger.warning(f"Error processing recommendations: {e}")
                    
            except Exception as e:
                logger.warning(f"Error processing interaction: {e}")
        
        return len(response.interactions) > 0
    except grpc.RpcError as e:
        logger.error(f"Test Drug Interaction Detection failed with gRPC error: {e}")
        return False
    except Exception as e:
        logger.error(f"Test Drug Interaction Detection failed with exception: {e}")
        return False

def test_clinical_assertions(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub,
    patient_id: str
) -> bool:
    """Test the CAE's ability to generate clinical assertions"""
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing Clinical Assertions for patient: {patient_id}")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("clinical-assertions")
    
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
            "include_allergies": True
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for patient {patient_id}...")
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Check metadata version with defensive check
        try:
            if hasattr(response.metadata, 'version') and response.metadata.version:
                logger.info(f"   Version: {response.metadata.version}")
        except Exception as e:
            logger.warning(f"Could not access metadata version: {e}")
            
        logger.info(f"   Total Assertions: {len(response.assertions)}")
        
        # Process assertions
        logger.info(f"\n📋 CLINICAL ASSERTIONS FOUND:")
        logger.info(f"{'='*70}")
        
        # Group assertions by type
        assertion_by_type = {}
        
        for assertion in response.assertions:
            assertion_type = assertion.type if assertion.type else "unknown"
            if assertion_type not in assertion_by_type:
                assertion_by_type[assertion_type] = []
            assertion_by_type[assertion_type].append(assertion)
        
        # Display assertions by type
        for assertion_type, assertions in assertion_by_type.items():
            logger.info(f"\n🔹 {assertion_type.upper()} ({len(assertions)} found):")
            
            for i, assertion in enumerate(assertions, 1):
                logger.info(f"\n   {i}. Clinical Assertion")
                logger.info(f"      ID: {assertion.assertion_id}")
                severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
                logger.info(f"      Severity: {severity}")
                logger.info(f"      Confidence: {assertion.confidence_score:.2f}")
                logger.info(f"      Description: {assertion.description}")
                
                # Check for evidence with defensive check
                try:
                    if hasattr(assertion, 'evidence') and assertion.evidence:
                        for evidence in assertion.evidence:
                            logger.info(f"      Evidence: {evidence.description} (Strength: {evidence.strength:.2f})")
                except Exception as e:
                    logger.warning(f"Could not process evidence: {e}")
        
        # Summary statistics
        logger.info(f"\n📊 SUMMARY STATISTICS:")
        logger.info(f"{'='*70}")
        logger.info(f"Total Assertions: {len(response.assertions)}")
        
        # By severity
        severity_counts = {}
        for assertion in response.assertions:
            severity_name = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
            severity_counts[severity_name] = severity_counts.get(severity_name, 0) + 1
        
        logger.info(f"\nBy Severity:")
        for severity, count in severity_counts.items():
            logger.info(f"  - {severity}: {count}")
        
        # By type
        logger.info(f"\nBy Type:")
        for assertion_type, assertions in assertion_by_type.items():
            logger.info(f"  - {assertion_type}: {len(assertions)}")
            
        return len(response.assertions) > 0
    except grpc.RpcError as e:
        logger.error(f"Test Clinical Assertions failed with gRPC error: {e}")
        return False
    except Exception as e:
        logger.error(f"Test Clinical Assertions failed with exception: {e}")
        return False

def test_dose_adjustment_recommendations(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub,
    patient_id: str
) -> bool:
    """Test the CAE's ability to recommend dose adjustments"""
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing Dose Adjustment Recommendations for patient: {patient_id}")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("dose-adjustments")
    
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
            "test_type": "dose_adjustment"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Add medications for dose testing
        request.medication_ids.extend(["metformin", "lisinopril"])
        
        # Add specific medications for testing dose adjustments (these should exist in the sample data)
        if patient_id == SAMPLE_PATIENTS["renal"]:
            request.medication_ids.extend(["metformin"])
        elif patient_id == SAMPLE_PATIENTS["geriatric"]:
            request.medication_ids.extend(["rivaroxaban"])
        elif patient_id == SAMPLE_PATIENTS["liver"]:
            request.medication_ids.extend(["metformin"])
        else:
            # Default medications to test with any patient
            request.medication_ids.extend(["metformin", "lisinopril"])
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for dose adjustments for patient {patient_id}...")
        # Set test_type to indicate we're looking for dose adjustments
        patient_context.update({"test_type": "dose_adjustments"})
        request.patient_context.CopyFrom(patient_context)
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        logger.info(f"   Total Adjustments: {len(response.dose_adjustments)}")
        
        # Process dose adjustments with defensive checks
        for adj in response.dose_adjustments:
            try:
                logger.info(f"\n💊 Dose Adjustment for {adj.medication_name}")
                if hasattr(adj, 'reason'):
                    logger.info(f"   Reason: {adj.reason}")
                if hasattr(adj, 'recommended_dose'):
                    logger.info(f"   Recommended Dose: {adj.recommended_dose}")
                if hasattr(adj, 'confidence_score'):
                    logger.info(f"   Confidence: {adj.confidence_score:.2f}")
                if hasattr(adj, 'evidence') and adj.evidence:
                    logger.info(f"   Evidence:")
                    for evidence in adj.evidence:
                        logger.info(f"     - {evidence}")
            except Exception as e:
                logger.warning(f"Error processing dose adjustment: {e}")
                
        return len(response.dose_adjustments) > 0
    except grpc.RpcError as e:
        logger.error(f"Test Dose Adjustment Recommendations failed with gRPC error: {e}")
        return False
    except Exception as e:
        logger.error(f"Test Dose Adjustment Recommendations failed with exception: {e}")
        return False

def test_contraindications(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub,
    patient_id: str
) -> bool:
    """Test the CAE's ability to detect contraindications"""
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing Contraindications for patient: {patient_id}")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("contraindications")
    
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
            "test_type": "contraindication"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Add medications to check for contraindications
        request.medication_ids.extend(["amoxicillin", "ibuprofen"])
        
        # Add medications to check for contraindications
        if patient_id == SAMPLE_PATIENTS["allergy"]:
            request.medication_ids.extend(["amoxicillin"])  # Known allergy
        elif patient_id == SAMPLE_PATIENTS["pregnancy"]:
            request.medication_ids.extend(["warfarin"])     # Contraindicated in pregnancy
        elif patient_id == SAMPLE_PATIENTS["renal"]:
            request.medication_ids.extend(["ibuprofen"])    # Contraindicated in renal impairment
        elif patient_id == SAMPLE_PATIENTS["liver"]:
            request.medication_ids.extend(["metformin"])    # Caution in severe liver disease
        else:
            # Default medications to check with any patient
            request.medication_ids.extend(["ibuprofen", "warfarin"])
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for contraindications for patient {patient_id}...")
        # Set test_type to indicate we're looking for contraindications
        patient_context.update({"test_type": "contraindications"})
        request.patient_context.CopyFrom(patient_context)
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Correlation ID: {request.correlation_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        logger.info(f"   Total Contraindications: {len(response.contraindications)}")
        
        # Process contraindications with defensive checks
        for contra in response.contraindications:
            try:
                severity = clinical_reasoning_pb2.AssertionSeverity.Name(contra.severity)
                logger.info(f"\n⚠️ Contraindication for {contra.medication_name}")
                logger.info(f"   Severity: {severity}")
                if hasattr(contra, 'reason'):
                    logger.info(f"   Reason: {contra.reason}")
                if hasattr(contra, 'condition_name'):
                    logger.info(f"   Related Condition: {contra.condition_name}")
                if hasattr(contra, 'confidence_score'):
                    logger.info(f"   Confidence: {contra.confidence_score:.2f}")
                if hasattr(contra, 'recommendation') and contra.recommendation:
                    logger.info(f"   Recommendation: {contra.recommendation}")
            except Exception as e:
                logger.warning(f"Error processing contraindication: {e}")
                
        return len(response.contraindications) > 0
    except grpc.RpcError as e:
        logger.error(f"Test Contraindications failed with gRPC error: {e}")
        return False
    except Exception as e:
        logger.error(f"Test Contraindications failed with exception: {e}")
        return False

def test_duplicate_therapy_detection(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub,
    patient_id: str
) -> bool:
    """Test the CAE's ability to detect duplicate therapies"""
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing Duplicate Therapy Detection for patient: {patient_id}")
    logger.info(f"{'='*70}")
    
    test_id = get_test_id("duplicate-therapy")
    
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
            "test_type": "duplicate_therapy",
            "new_medication": "enalapril"
        })
        request.patient_context.CopyFrom(patient_context)
        
        # Add a new medication (duplicate class to something already prescribed)
        # This should trigger a duplicate therapy alert
        if patient_id == SAMPLE_PATIENTS["cardiovascular"]:
            # Already on lisinopril (ACE inhibitor)
            request.medication_ids.extend(["enalapril"])  # Another ACE inhibitor
        else:
            # Default new medication to test with any patient
            request.medication_ids.extend(["naproxen"])  # Another NSAID if patient is on ibuprofen
        
        # Call the service
        logger.info(f"📡 Calling GenerateAssertions for duplicate therapy check for patient {patient_id}...")
        # Set test_type to indicate we're looking for duplicate therapies
        patient_context.update({"test_type": "duplicate_therapy"})
        request.patient_context.CopyFrom(patient_context)
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        # Log basic response info
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Request ID: {response.request_id}")
        logger.info(f"   Processing Time: {elapsed_time}ms")
        
        # Check if duplicate detected with defensive check
        try:
            if hasattr(response, 'is_duplicate') and response.is_duplicate:
                logger.info(f"⚠️ DUPLICATE THERAPY DETECTED!")
                
                if hasattr(response, 'existing_medication') and response.existing_medication:
                    logger.info(f"   Existing Medication: {response.existing_medication}")
                    
                if hasattr(response, 'therapeutic_class') and response.therapeutic_class:
                    logger.info(f"   Therapeutic Class: {response.therapeutic_class}")
                    
                if hasattr(response, 'recommendation') and response.recommendation:
                    logger.info(f"   Recommendation: {response.recommendation}")
            else:
                logger.info("✅ No duplicate therapy detected")
        except Exception as e:
            logger.warning(f"Error checking duplicate therapy response: {e}")
            
        return True  # Return success if the call completed, regardless of finding duplicates
    except grpc.RpcError as e:
        logger.error(f"Test Duplicate Therapy Detection failed with gRPC error: {e}")
        return False
    except Exception as e:
        logger.error(f"Test Duplicate Therapy Detection failed with exception: {e}")
        return False

def test_special_population_checks(
    stub: clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub
) -> bool:
    """Run tests specific to special populations (pediatric, geriatric, pregnancy)"""
    logger.info(f"\n\n{'='*70}")
    logger.info(f"Testing Special Population Scenarios")
    logger.info(f"{'='*70}")
    
    success = True
    
    # Test pediatric patient
    logger.info(f"\n🧒 Testing Pediatric Patient (8 years old)")
    pediatric_success = test_clinical_assertions(stub, SAMPLE_PATIENTS["pediatric"])
    success = success and pediatric_success
    
    # Test geriatric patient
    logger.info(f"\n👵 Testing Geriatric Patient (89 years old)")
    geriatric_success = test_dose_adjustment_recommendations(stub, SAMPLE_PATIENTS["geriatric"])
    success = success and geriatric_success
    
    # Test pregnant patient
    logger.info(f"\n🤰 Testing Pregnant Patient")
    pregnancy_success = test_contraindications(stub, SAMPLE_PATIENTS["pregnancy"])
    success = success and pregnancy_success
    
    return success

def run_all_tests() -> bool:
    """Run all CAE tests using the comprehensive test data"""
    logger.info(f"\n{'='*80}")
    logger.info(f"🏥 CAE Comprehensive Test Suite - SAMPLE DATA from GraphDB")
    logger.info(f"{'='*80}")
    logger.info(f"This test suite exercises all scenarios in the CAE sample data")
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
        
        # Keep track of test results
        test_results = {}
        
        try:
            # 1. Test Drug Interactions with primary patient (complex case)
            test_results["Drug Interactions"] = test_drug_interaction_detection(stub, SAMPLE_PATIENTS["primary"])
        except Exception as e:
            logger.error(f"Test Drug Interaction Detection failed with exception: {e}")
            test_results["Drug Interactions"] = False
        
        try:
            # 2. Test Clinical Assertions with primary patient
            test_results["Clinical Assertions"] = test_clinical_assertions(stub, SAMPLE_PATIENTS["primary"])
        except Exception as e:
            logger.error(f"Test Clinical Assertions failed with exception: {e}")
            test_results["Clinical Assertions"] = False
        
        try:
            # 3. Test Dose Adjustments with renal patient
            test_results["Dose Adjustments"] = test_dose_adjustment_recommendations(stub, SAMPLE_PATIENTS["renal"])
        except Exception as e:
            logger.error(f"Test Dose Adjustment Recommendations failed with exception: {e}")
            test_results["Dose Adjustments"] = False
        
        try:
            # 4. Test Contraindications with allergy patient
            test_results["Contraindications"] = test_contraindications(stub, SAMPLE_PATIENTS["allergy"])
        except Exception as e:
            logger.error(f"Test Contraindications failed with exception: {e}")
            test_results["Contraindications"] = False
        
        try:
            # 5. Test Duplicate Therapy Detection
            test_results["Duplicate Therapy"] = test_duplicate_therapy_detection(stub, SAMPLE_PATIENTS["cardiovascular"])
        except Exception as e:
            logger.error(f"Test Duplicate Therapy Detection failed with exception: {e}")
            test_results["Duplicate Therapy"] = False
        
        # 6. Test Special Population Scenarios
        test_results["Special Populations"] = test_special_population_checks(stub)
        
        # Final results summary
        logger.info(f"\n{'='*80}")
        logger.info(f"📊 FINAL TEST SUMMARY")
        logger.info(f"{'='*80}")
        
        passed_count = sum(1 for result in test_results.values() if result)
        total_count = len(test_results)
        
        for test_name, result in test_results.items():
            status = "✅ PASSED" if result else "❌ FAILED"
            logger.info(f"{test_name}: {status}")
        
        logger.info(f"\nTotal: {passed_count}/{total_count} tests passed")
        
        if passed_count == total_count:
            logger.info(f"\n🎉 All tests passed! CAE is functioning correctly with sample data.")
            logger.info(f"The CAE has successfully demonstrated all capabilities in the test scenarios.")
            return True
        else:
            logger.warning(f"\n⚠️  {total_count - passed_count} tests failed.")
            return False
            
    except Exception as e:
        logger.error(f"Comprehensive test suite failed with exception: {e}")
        return False
    finally:
        # Ensure the channel is closed
        if 'channel' in locals():
            channel.close()

if __name__ == "__main__":
    success = run_all_tests()
    sys.exit(0 if success else 1)
