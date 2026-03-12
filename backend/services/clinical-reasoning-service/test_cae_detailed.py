#!/usr/bin/env python
"""
Detailed CAE Test Script
Tests various aspects of the Clinical Assertion Engine
"""

import sys
import grpc
import logging
from datetime import datetime
from pathlib import Path
import json

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Add parent directory to path
parent_dir = str(Path(__file__).resolve().parent)
sys.path.insert(0, parent_dir)

# Import proto files
try:
    from app.proto import clinical_reasoning_pb2
    from app.proto import clinical_reasoning_pb2_grpc
except ImportError as e:
    logger.error(f"Failed to import proto files: {e}")
    sys.exit(1)

def test_health_check():
    """Test the health check endpoint"""
    logger.info("Testing Health Check...")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    request = clinical_reasoning_pb2.HealthCheckRequest(service="clinical-reasoning-service")
    
    try:
        response = stub.HealthCheck(request)
        status_name = clinical_reasoning_pb2.HealthCheckResponse.ServingStatus.Name(response.status)
        logger.info(f"✅ Health Check Response: {status_name}")
        return True
    except grpc.RpcError as e:
        logger.error(f"❌ Health Check Failed: {e.code()}: {e.details()}")
        return False

def test_medication_interactions():
    """Test medication interaction checking"""
    logger.info("\nTesting Medication Interactions...")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Create request
    request = clinical_reasoning_pb2.MedicationInteractionRequest()
    request.patient_id = "test-patient-001"
    request.medication_ids.extend(["warfarin", "aspirin"])
    request.new_medication_id = "ibuprofen"
    
    # Add patient context
    from google.protobuf.struct_pb2 import Struct
    patient_context = Struct()
    patient_context.update({
        "age": 67,
        "kidney_function": "normal",
        "liver_function": "normal"
    })
    request.patient_context.CopyFrom(patient_context)
    
    try:
        response = stub.CheckMedicationInteractions(request)
        logger.info(f"✅ Found {len(response.interactions)} interactions")
        
        for interaction in response.interactions:
            severity = clinical_reasoning_pb2.AssertionSeverity.Name(interaction.severity)
            logger.info(f"  - {interaction.medication_a} + {interaction.medication_b}: {interaction.description}")
            logger.info(f"    Severity: {severity}, Confidence: {interaction.confidence_score}")
            
        return True
    except grpc.RpcError as e:
        logger.error(f"❌ Medication Interaction Check Failed: {e.code()}: {e.details()}")
        return False

def test_dosing_calculation():
    """Test dosing calculation"""
    logger.info("\nTesting Dosing Calculation...")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Create request
    request = clinical_reasoning_pb2.DosingCalculationRequest()
    request.patient_id = "test-patient-001"
    request.medication_id = "metformin"
    request.indication = "type 2 diabetes"
    
    # Add patient parameters
    from google.protobuf.struct_pb2 import Struct
    patient_params = Struct()
    patient_params.update({
        "weight": 78.5,
        "age": 67,
        "renal_function": "normal",
        "hepatic_function": "normal"
    })
    request.patient_parameters.CopyFrom(patient_params)
    
    try:
        response = stub.CalculateDosing(request)
        logger.info(f"✅ Dosing Recommendation:")
        logger.info(f"  - Dose: {response.dosing.dose}")
        logger.info(f"  - Frequency: {response.dosing.frequency}")
        logger.info(f"  - Route: {response.dosing.route}")
        logger.info(f"  - Duration: {response.dosing.duration}")
        
        return True
    except grpc.RpcError as e:
        logger.error(f"❌ Dosing Calculation Failed: {e.code()}: {e.details()}")
        return False

def test_contraindications():
    """Test contraindication checking"""
    logger.info("\nTesting Contraindications...")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Create request
    request = clinical_reasoning_pb2.ContraindicationRequest()
    request.patient_id = "test-patient-001"
    request.medication_ids.extend(["ibuprofen"])
    request.condition_ids.extend(["chronic_kidney_disease", "hypertension"])
    
    try:
        response = stub.CheckContraindications(request)
        logger.info(f"✅ Found {len(response.contraindications)} contraindications")
        
        for contra in response.contraindications:
            severity = clinical_reasoning_pb2.AssertionSeverity.Name(contra.severity)
            logger.info(f"  - {contra.medication_id} contraindicated for {contra.condition_id}")
            logger.info(f"    Type: {contra.type}, Severity: {severity}")
            logger.info(f"    Description: {contra.description}")
            
        return True
    except grpc.RpcError as e:
        logger.error(f"❌ Contraindication Check Failed: {e.code()}: {e.details()}")
        return False

def test_comprehensive_assertions():
    """Test comprehensive clinical assertions"""
    logger.info("\nTesting Comprehensive Clinical Assertions...")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Create a comprehensive request
    request = clinical_reasoning_pb2.ClinicalAssertionRequest()
    request.patient_id = "test-patient-comprehensive"
    request.correlation_id = f"test-comprehensive-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    
    # Add multiple medications (known interactions)
    request.medication_ids.extend([
        "warfarin",      # Anticoagulant
        "aspirin",       # Antiplatelet
        "ibuprofen",     # NSAID
        "lisinopril",    # ACE inhibitor
        "metoprolol"     # Beta blocker
    ])
    
    # Add conditions
    request.condition_ids.extend([
        "atrial_fibrillation",
        "hypertension",
        "coronary_artery_disease",
        "diabetes_type_2"
    ])
    
    # Create detailed patient context
    from google.protobuf.struct_pb2 import Struct
    patient_context = Struct()
    patient_context.update({
        "demographics": {
            "age": 67,
            "gender": "male",
            "weight": 78.5,
            "height": 175
        },
        "kidney_function": "mild_impairment",
        "liver_function": "normal",
        "allergies": ["penicillin"],
        "laboratory_values": {
            "creatinine": 1.4,
            "egfr": 55,
            "alt": 25,
            "ast": 22
        }
    })
    request.patient_context.CopyFrom(patient_context)
    
    # Set high priority
    request.priority = clinical_reasoning_pb2.AssertionPriority.PRIORITY_URGENT
    
    # Request all reasoner types
    request.reasoner_types.extend([
        "interaction", 
        "dosing", 
        "contraindication", 
        "duplicate_therapy", 
        "clinical_context"
    ])
    
    try:
        response = stub.GenerateAssertions(request)
        logger.info(f"✅ Received {len(response.assertions)} assertions")
        logger.info(f"   Request ID: {response.request_id}")
        logger.info(f"   Processing Time: {response.metadata.processing_time_ms}ms")
        
        # Categorize assertions by type
        assertions_by_type = {}
        for assertion in response.assertions:
            assertion_type = assertion.type
            if assertion_type not in assertions_by_type:
                assertions_by_type[assertion_type] = []
            assertions_by_type[assertion_type].append(assertion)
        
        # Display assertions by category
        for assertion_type, assertions in assertions_by_type.items():
            logger.info(f"\n   {assertion_type.upper()} ({len(assertions)} found):")
            for assertion in assertions:
                severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
                logger.info(f"   - {assertion.title}")
                logger.info(f"     Severity: {severity}, Confidence: {assertion.confidence_score:.2f}")
                logger.info(f"     {assertion.description}")
                
                # Show recommendations
                if assertion.recommendations:
                    logger.info("     Recommendations:")
                    for rec in assertion.recommendations:
                        priority = clinical_reasoning_pb2.RecommendationPriority.Name(rec.priority)
                        logger.info(f"     • {rec.description} (Priority: {priority})")
        
        return True
    except grpc.RpcError as e:
        logger.error(f"❌ Comprehensive Assertion Generation Failed: {e.code()}: {e.details()}")
        return False

def main():
    """Run all tests"""
    logger.info("=" * 80)
    logger.info("Clinical Assertion Engine (CAE) Comprehensive Test Suite")
    logger.info("=" * 80)
    
    tests = [
        ("Health Check", test_health_check),
        ("Medication Interactions", test_medication_interactions),
        ("Dosing Calculations", test_dosing_calculation),
        ("Contraindications", test_contraindications),
        ("Comprehensive Assertions", test_comprehensive_assertions)
    ]
    
    results = []
    for test_name, test_func in tests:
        try:
            success = test_func()
            results.append((test_name, success))
        except Exception as e:
            logger.error(f"Test {test_name} failed with exception: {e}")
            results.append((test_name, False))
    
    # Summary
    logger.info("\n" + "=" * 80)
    logger.info("TEST SUMMARY")
    logger.info("=" * 80)
    
    passed = sum(1 for _, success in results if success)
    total = len(results)
    
    for test_name, success in results:
        status = "✅ PASSED" if success else "❌ FAILED"
        logger.info(f"{test_name}: {status}")
    
    logger.info(f"\nTotal: {passed}/{total} tests passed")
    
    if passed == total:
        logger.info("\n🎉 All tests passed! CAE is working correctly with gRPC.")
    else:
        logger.warning(f"\n⚠️  {total - passed} tests failed. Please check the implementation.")

if __name__ == "__main__":
    main()
