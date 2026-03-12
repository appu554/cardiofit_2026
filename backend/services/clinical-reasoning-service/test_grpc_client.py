#!/usr/bin/env python
"""
Clinical Assertion Engine (CAE) gRPC Test Client
This script tests the CAE gRPC interface by sending a clinical assertion request
and displaying the response.
"""

import sys
import grpc
import logging
from datetime import datetime
from pathlib import Path

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Add parent directory to path to allow imports
parent_dir = str(Path(__file__).resolve().parent)
sys.path.insert(0, parent_dir)

# Import proto files
try:
    from app.proto import clinical_reasoning_pb2
    from app.proto import clinical_reasoning_pb2_grpc
except ImportError:
    try:
        from proto import clinical_reasoning_pb2
        from proto import clinical_reasoning_pb2_grpc
    except ImportError as e:
        logger.error(f"Failed to import proto files: {e}")
        sys.exit(1)

def create_test_request():
    """Create a test clinical assertion request with sample data"""
    request = clinical_reasoning_pb2.ClinicalAssertionRequest()
    
    # Set patient ID - using a known comprehensive test patient
    request.patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
    request.correlation_id = f"test-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    
    # Add medication IDs
    request.medication_ids.extend(["1049502", "1049221"])  # Warfarin, Aspirin
    
    # Add condition IDs
    request.condition_ids.extend(["I48.91", "I25.10"])  # Atrial Fibrillation, CAD
    
    # Create patient context with demographics and additional data
    from google.protobuf.struct_pb2 import Struct
    patient_context = Struct()
    
    # Only include medications, let CAE fetch demographics from GraphDB
    patient_context.update({
        "medications": [
            {
                "code": "1049502",
                "system": "RXNORM",
                "display": "Warfarin 5 MG Oral Tablet",
                "dosage": "5mg daily"
            },
            {
                "code": "1049221",
                "system": "RXNORM",
                "display": "Aspirin 81 MG Oral Tablet",
                "dosage": "81mg daily"
            }
        ],
        "conditions": [
            {
                "code": "I48.91",
                "system": "ICD10",
                "display": "Atrial Fibrillation"
            },
            {
                "code": "I25.10",
                "system": "ICD10",
                "display": "Coronary Artery Disease"
            }
        ]
    })
    
    request.patient_context.CopyFrom(patient_context)
    
    # Set priority
    request.priority = clinical_reasoning_pb2.AssertionPriority.PRIORITY_URGENT
    
    # Request all reasoner types
    request.reasoner_types.extend(["interaction", "dosing", "contraindication", "duplicate_therapy", "clinical_context"])
    
    return request

def run_test():
    """Run the gRPC test client"""
    logger.info("🚀 Starting CAE gRPC Test Client...")
    
    # Create a gRPC channel
    channel = grpc.insecure_channel('localhost:8027')
    
    # Create a stub (client)
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    # Create a request
    request = create_test_request()
    logger.info(f"📝 Created test request for patient: {request.patient_id}")
    
    try:
        # Make the call
        logger.info("📞 Calling GenerateAssertions...")
        response = stub.GenerateAssertions(request)
        
        # Process the response
        logger.info(f"✅ Received response with {len(response.assertions)} assertions")
        
        # Print each assertion
        for i, assertion in enumerate(response.assertions, 1):
            severity = clinical_reasoning_pb2.AssertionSeverity.Name(assertion.severity)
            logger.info(f"  Assertion {i}: {assertion.assertion_type} - {assertion.description} (Severity: {severity})")
            
            # Print recommendations if any
            for j, rec in enumerate(assertion.recommendations, 1):
                logger.info(f"    Recommendation {j}: {rec.description}")
        
        logger.info("✅ Test completed successfully")
        
    except grpc.RpcError as e:
        logger.error(f"❌ RPC failed: {e.code()}: {e.details()}")
    except Exception as e:
        logger.error(f"❌ Error: {str(e)}")

if __name__ == "__main__":
    run_test()
