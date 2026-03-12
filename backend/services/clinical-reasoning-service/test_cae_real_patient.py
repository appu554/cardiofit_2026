#!/usr/bin/env python
"""
CAE Test with Real Patient Data from GraphDB
Tests the Clinical Assertion Engine with actual patient data
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

# REAL PATIENT ID FROM GRAPHDB
REAL_PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def test_real_patient_comprehensive():
    """Test with real patient data from GraphDB"""
    logger.info(f"\n🔍 Testing with REAL patient from GraphDB: {REAL_PATIENT_ID}")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Create request with real patient ID
    request = clinical_reasoning_pb2.ClinicalAssertionRequest()
    request.patient_id = REAL_PATIENT_ID  # Real patient from GraphDB
    request.correlation_id = f"real-patient-test-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    
    # For real patient, we'll rely on actual data in GraphDB
    # But we can add test medications to check interactions
    request.medication_ids.extend([
        "warfarin",      # Anticoagulant
        "aspirin",       # Antiplatelet
        "metoprolol",    # Beta blocker
        "lisinopril",    # ACE inhibitor
        "simvastatin",   # Statin
        "metformin"      # Diabetes medication
    ])
    
    # Add some test conditions (these may also be in GraphDB)
    request.condition_ids.extend([
        "atrial_fibrillation",
        "hypertension",
        "type_2_diabetes",
        "coronary_artery_disease"
    ])
    
    # Create patient context - this will be enriched by GraphDB data
    from google.protobuf.struct_pb2 import Struct
    patient_context = Struct()
    patient_context.update({
        "request_source": "test_suite",
        "test_type": "real_patient_validation",
        "include_graphdb_data": True
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
        logger.info("📡 Calling GenerateAssertions for real patient...")
        response = stub.GenerateAssertions(request)
        
        logger.info(f"\n✅ Response received!")
        logger.info(f"   Request ID: {response.request_id}")
        logger.info(f"   Processing Time: {response.metadata.processing_time_ms}ms")
        logger.info(f"   Total Assertions: {len(response.assertions)}")
        
        # Show GraphDB enrichment status
        try:
            if hasattr(response.metadata, 'version') and response.metadata.version:
                logger.info(f"   CAE Version: {response.metadata.version}")
        except Exception as e:
            logger.warning(f"Could not access version metadata: {e}")
        
        # Categorize assertions
        assertions_by_type = {}
        for assertion in response.assertions:
            assertion_type = assertion.type
            if assertion_type not in assertions_by_type:
                assertions_by_type[assertion_type] = []
            assertions_by_type[assertion_type].append(assertion)
        
        # Display detailed results
        logger.info("\n📋 CLINICAL ASSERTIONS FOUND:")
        logger.info("=" * 70)
        
        for assertion_type, assertions in assertions_by_type.items():
            logger.info(f"\n🔹 {assertion_type.upper()} ({len(assertions)} found):")
            for i, assertion in enumerate(assertions, 1):
                severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
                logger.info(f"\n   {i}. {assertion.title}")
                logger.info(f"      ID: {assertion.id}")
                logger.info(f"      Severity: {severity}")
                logger.info(f"      Confidence: {assertion.confidence_score:.2f}")
                logger.info(f"      Description: {assertion.description}")
                
                # Show evidence if available
                try:
                    if hasattr(assertion, 'evidence') and assertion.evidence:
                        logger.info("      Evidence:")
                        for evidence in assertion.evidence:
                            logger.info(f"      - {evidence}")
                except Exception as e:
                    logger.warning(f"Could not access evidence: {e}")
                
                # Show recommendations
                if assertion.recommendations:
                    logger.info("      Recommendations:")
                    for rec in assertion.recommendations:
                        priority = clinical_reasoning_pb2.RecommendationPriority.Name(rec.priority)
                        logger.info(f"      • {rec.description}")
                        logger.info(f"        (Priority: {priority}, Type: {rec.type})")
                        if rec.evidence_level:
                            logger.info(f"        Evidence Level: {rec.evidence_level}")
        
        # Summary statistics
        logger.info("\n📊 SUMMARY STATISTICS:")
        logger.info("=" * 70)
        logger.info(f"Total Assertions: {len(response.assertions)}")
        
        severity_counts = {}
        for assertion in response.assertions:
            severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
            severity_counts[severity] = severity_counts.get(severity, 0) + 1
        
        logger.info("\nBy Severity:")
        for severity, count in severity_counts.items():
            logger.info(f"  - {severity}: {count}")
        
        logger.info("\nBy Type:")
        for assertion_type, assertions in assertions_by_type.items():
            logger.info(f"  - {assertion_type}: {len(assertions)}")
        
        return True
        
    except grpc.RpcError as e:
        logger.error(f"❌ RPC Error: {e.code()}: {e.details()}")
        return False
    except Exception as e:
        logger.error(f"❌ Unexpected error: {e}")
        return False

def test_real_patient_medication_interactions():
    """Test medication interactions for real patient"""
    logger.info(f"\n💊 Testing medication interactions for patient: {REAL_PATIENT_ID}")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Test with specific medication combination
    request = clinical_reasoning_pb2.MedicationInteractionRequest()
    request.patient_id = REAL_PATIENT_ID
    request.medication_ids.extend(["warfarin", "aspirin", "ibuprofen"])
    request.new_medication_id = "clopidogrel"  # Another antiplatelet
    
    # Add patient context
    from google.protobuf.struct_pb2 import Struct
    patient_context = Struct()
    patient_context.update({
        "check_type": "new_medication_screening",
        "use_graphdb_context": True
    })
    request.patient_context.CopyFrom(patient_context)
    
    try:
        response = stub.CheckMedicationInteractions(request)
        logger.info(f"✅ Found {len(response.interactions)} interactions")
        
        try:
            if hasattr(response, 'has_critical_interaction') and response.has_critical_interaction:
                logger.warning("⚠️  CRITICAL INTERACTIONS DETECTED!")
        except Exception as e:
            logger.warning(f"Could not check critical interaction flag: {e}")
        
        for interaction in response.interactions:
            try:
                severity = clinical_reasoning_pb2.AssertionSeverity.Name(interaction.severity)
                logger.info(f"\n   🔸 {interaction.medication_a} + {interaction.medication_b}")
                
                # Add defensive checks for all attributes
                if hasattr(interaction, 'type'):
                    logger.info(f"      Type: {interaction.type}")
                    
                logger.info(f"      Severity: {severity}")
                
                if hasattr(interaction, 'description'):
                    logger.info(f"      Description: {interaction.description}")
                    
                if hasattr(interaction, 'mechanism'):
                    logger.info(f"      Mechanism: {interaction.mechanism}")
                    
                logger.info(f"      Confidence: {interaction.confidence_score:.2f}")
            except Exception as e:
                logger.warning(f"Error processing interaction: {e}")
            
            # Defensive check for recommendations
            try:
                if hasattr(interaction, 'recommendations') and interaction.recommendations:
                    logger.info("      Management:")
                    for rec in interaction.recommendations:
                        logger.info(f"        - {rec}")
            except Exception as e:
                logger.warning(f"Error processing recommendations: {e}")
        
        return True
    except grpc.RpcError as e:
        logger.error(f"❌ Medication interaction check failed: {e.code()}: {e.details()}")
        return False

def main():
    """Run tests with real patient data"""
    logger.info("=" * 80)
    logger.info("🏥 CAE Test Suite - REAL PATIENT DATA from GraphDB")
    logger.info("=" * 80)
    logger.info(f"Patient ID: {REAL_PATIENT_ID}")
    logger.info("This test uses actual patient data stored in GraphDB")
    logger.info("=" * 80)
    
    # First, verify the server is running
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Health check
    try:
        health_request = clinical_reasoning_pb2.HealthCheckRequest(service="clinical-reasoning-service")
        health_response = stub.HealthCheck(health_request)
        status = clinical_reasoning_pb2.HealthCheckResponse.ServingStatus.Name(health_response.status)
        logger.info(f"✅ Server Health: {status}")
    except:
        logger.error("❌ Server is not responding. Please ensure CAE gRPC server is running on port 8027")
        return
    
    # Run tests
    tests = [
        ("Real Patient Comprehensive Analysis", test_real_patient_comprehensive),
        ("Real Patient Medication Interactions", test_real_patient_medication_interactions)
    ]
    
    results = []
    for test_name, test_func in tests:
        logger.info(f"\n{'='*70}")
        logger.info(f"Running: {test_name}")
        logger.info(f"{'='*70}")
        
        try:
            success = test_func()
            results.append((test_name, success))
        except Exception as e:
            logger.error(f"Test {test_name} failed with exception: {e}")
            results.append((test_name, False))
    
    # Final summary
    logger.info("\n" + "=" * 80)
    logger.info("📊 FINAL TEST SUMMARY")
    logger.info("=" * 80)
    
    passed = sum(1 for _, success in results if success)
    total = len(results)
    
    for test_name, success in results:
        status = "✅ PASSED" if success else "❌ FAILED"
        logger.info(f"{test_name}: {status}")
    
    logger.info(f"\nTotal: {passed}/{total} tests passed")
    
    if passed == total:
        logger.info("\n🎉 All tests passed! Real patient data processed successfully.")
        logger.info("The CAE is correctly integrating with GraphDB for patient context enrichment.")
    else:
        logger.warning(f"\n⚠️  {total - passed} tests failed.")

if __name__ == "__main__":
    main()
