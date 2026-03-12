#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CAE Minimal Test Script

This script tests the CAE service with a minimal request to verify basic functionality.
It attempts to use the simplest possible format that might work.
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

def test_minimal_assertion(stub):
    """
    Test a minimal clinical assertion request with only required fields
    """
    logger.info("\n" + "="*70)
    logger.info("Testing Minimal Clinical Assertion Request")
    logger.info("="*70)
    
    test_id = f"minimal-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    
    try:
        # Create the most minimal request possible
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.correlation_id = f"req_{test_id}"
        
        # Try the patient ID from real patient test
        request.patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        
        # Minimal patient context
        patient_context = Struct()
        patient_context.update({
            "patient_id": request.patient_id,
            "test_type": "generate_assertions"
        })
        request.patient_context.CopyFrom(patient_context)
        
        logger.info(f"📡 Calling GenerateAssertions with minimal request...")
        logger.info(f"Request: {request}")
        
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        logger.info("\n✅ Response received!")
        logger.info(f"Processing Time: {elapsed_time}ms")
        
        # Log full response for analysis
        logger.info("\n📋 Full response content:")
        logger.info(f"{str(response)}")
        
        # Check for assertions
        if hasattr(response, 'assertions') and response.assertions:
            logger.info(f"\n🔎 Found {len(response.assertions)} assertions")
            for i, assertion in enumerate(response.assertions, 1):
                try:
                    logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled'}")
                    if hasattr(assertion, 'description'):
                        logger.info(f"      Description: {assertion.description}")
                except Exception as e:
                    logger.warning(f"Error accessing assertion fields: {e}")
            return True
        else:
            if hasattr(response, 'metadata') and hasattr(response.metadata, 'warnings'):
                logger.warning(f"\n⚠️ Warning: {response.metadata.warnings}")
            logger.warning("\n❌ No assertions generated")
            return False
    except Exception as e:
        logger.error(f"❌ Test failed with exception: {e}")
        return False

def test_real_patient_variant(stub):
    """
    Test a variant of the real patient test that worked previously
    """
    logger.info("\n" + "="*70)
    logger.info("Testing Real Patient Variant")
    logger.info("="*70)
    
    test_id = f"real-patient-variant-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    
    try:
        # Create a request based on the working real patient test
        request = clinical_reasoning_pb2.ClinicalAssertionRequest()
        request.correlation_id = f"req_{test_id}"
        request.patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        
        # Try to match the format from the working test
        patient_context = Struct()
        patient_context.update({
            "patient_id": request.patient_id,
            "source": "graphdb",
            "include_medications": True,
            "include_conditions": True,
            "include_allergies": True,
            "include_labs": True
        })
        request.patient_context.CopyFrom(patient_context)
        
        logger.info(f"📡 Calling GenerateAssertions with real patient request...")
        
        start_time = time.time()
        response = stub.GenerateAssertions(request)
        elapsed_time = int((time.time() - start_time) * 1000)
        
        logger.info("\n✅ Response received!")
        logger.info(f"Processing Time: {elapsed_time}ms")
        
        # Log response metadata
        if hasattr(response, 'metadata'):
            if hasattr(response.metadata, 'reasoner_version'):
                logger.info(f"Reasoner Version: {response.metadata.reasoner_version}")
            if hasattr(response.metadata, 'knowledge_version'):
                logger.info(f"Knowledge Version: {response.metadata.knowledge_version}")
            if hasattr(response.metadata, 'warnings'):
                logger.warning(f"Warnings: {response.metadata.warnings}")
        
        # Check for assertions
        if hasattr(response, 'assertions') and response.assertions:
            logger.info(f"\n🔎 Found {len(response.assertions)} assertions")
            for i, assertion in enumerate(response.assertions[:5], 1):  # Show first 5
                try:
                    logger.info(f"\n   {i}. {assertion.title if hasattr(assertion, 'title') else 'Untitled'}")
                    if hasattr(assertion, 'description'):
                        logger.info(f"      Description: {assertion.description}")
                except Exception as e:
                    logger.warning(f"Error accessing assertion fields: {e}")
            return True
        else:
            logger.warning("\n❌ No assertions generated")
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
    logger.info("🏥 CAE MINIMAL TEST SUITE")
    logger.info("="*70)
    
    # Test health check first
    health_result = test_health_check(stub)
    
    if health_result:
        # Test minimal assertion
        minimal_result = test_minimal_assertion(stub)
        
        # Test real patient variant
        real_patient_result = test_real_patient_variant(stub)
        
        # Display final test summary
        logger.info("\n" + "="*70)
        logger.info("📊 FINAL TEST SUMMARY")
        logger.info("="*70)
        
        logger.info(f"Health Check: {'✅ PASSED' if health_result else '❌ FAILED'}")
        logger.info(f"Minimal Assertion: {'✅ PASSED' if minimal_result else '❌ FAILED'}")
        logger.info(f"Real Patient Variant: {'✅ PASSED' if real_patient_result else '❌ FAILED'}")
        
        if not minimal_result and not real_patient_result:
            logger.warning("""
            No assertions were generated in any test. This suggests:
            
            1. The CAE reasoning engine may not be configured correctly
            2. The required data for clinical assertions may not be in the GraphDB
            3. The test_type or patient context format may be incorrect
            
            Check with the CAE development team to confirm:
            - That the CAE service is fully set up with its reasoning engine
            - The expected format for patient context to trigger assertions
            - If any specific reasoners need to be enabled or configured
            """)
    else:
        logger.error("Health check failed. Please ensure the CAE service is running.")
