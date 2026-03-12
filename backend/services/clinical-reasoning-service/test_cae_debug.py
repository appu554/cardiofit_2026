#!/usr/bin/env python
"""
Debug test for CAE to understand why assertions are not being returned
"""

import sys
import grpc
import logging
from pathlib import Path

# Configure detailed logging
logging.basicConfig(
    level=logging.DEBUG,
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

def test_simple_assertion():
    """Test with minimal request to debug"""
    logger.info("Testing simple assertion generation...")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Create minimal request
    request = clinical_reasoning_pb2.ClinicalAssertionRequest()
    request.patient_id = "test-debug-patient"
    request.correlation_id = "debug-test-001"
    
    # Just two medications known to interact
    request.medication_ids.extend(["warfarin", "aspirin"])
    
    # One condition
    request.condition_ids.extend(["atrial_fibrillation"])
    
    # Simple context
    from google.protobuf.struct_pb2 import Struct
    patient_context = Struct()
    patient_context.update({
        "age": 65,
        "test": "debug"
    })
    request.patient_context.CopyFrom(patient_context)
    
    # Request only interaction reasoner
    request.reasoner_types.extend(["interaction"])
    
    try:
        logger.info("Sending request...")
        response = stub.GenerateAssertions(request)
        
        logger.info(f"Response received:")
        logger.info(f"  Request ID: {response.request_id}")
        logger.info(f"  Processing Time: {response.metadata.processing_time_ms}ms")
        logger.info(f"  Assertions Count: {len(response.assertions)}")
        
        if response.assertions:
            for assertion in response.assertions:
                logger.info(f"\nAssertion found:")
                logger.info(f"  ID: {assertion.id}")
                logger.info(f"  Type: {assertion.type}")
                logger.info(f"  Title: {assertion.title}")
                logger.info(f"  Description: {assertion.description}")
                logger.info(f"  Severity: {clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)}")
                logger.info(f"  Confidence: {assertion.confidence_score}")
        else:
            logger.warning("No assertions returned!")
            
            # Try checking metadata
            if hasattr(response, 'metadata') and response.metadata:
                logger.info(f"Metadata fields: {response.metadata}")
                
        return True
        
    except grpc.RpcError as e:
        logger.error(f"RPC Error: {e.code()}: {e.details()}")
        return False

def test_direct_interaction_check():
    """Test medication interaction directly"""
    logger.info("\nTesting direct medication interaction check...")
    
    channel = grpc.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    request = clinical_reasoning_pb2.MedicationInteractionRequest()
    request.patient_id = "test-patient"
    request.medication_ids.extend(["warfarin", "aspirin"])
    
    try:
        response = stub.CheckMedicationInteractions(request)
        logger.info(f"Direct interaction check - Found {len(response.interactions)} interactions")
        
        for interaction in response.interactions:
            logger.info(f"  {interaction.medication_a} + {interaction.medication_b}: {interaction.description}")
            
        return True
    except grpc.RpcError as e:
        logger.error(f"Direct interaction check failed: {e}")
        return False

if __name__ == "__main__":
    logger.info("=== CAE Debug Test ===")
    
    # Test both approaches
    test_simple_assertion()
    test_direct_interaction_check()
