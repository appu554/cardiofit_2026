#!/usr/bin/env python3
"""
Direct CAE Service Test
Tests the CAE service directly via gRPC to isolate issues
"""

import asyncio
import grpc
import sys
import os
from datetime import datetime

# Add the app directory to Python path
sys.path.append(os.path.join(os.path.dirname(__file__), 'app'))

# Import the generated gRPC classes
from proto import clinical_reasoning_pb2
from proto import clinical_reasoning_pb2_grpc

async def test_cae_direct():
    """Test CAE service directly via gRPC"""
    
    print("🚀 Direct CAE Service Test")
    print("=" * 70)
    
    # Create gRPC channel
    channel = grpc.aio.insecure_channel('localhost:8027')
    stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
    
    try:
        # Test 1: Health Check
        print("\n🔍 Health Check Test")
        print("-" * 30)

        health_request = clinical_reasoning_pb2.HealthCheckRequest()
        health_response = await stub.HealthCheck(health_request)

        print(f"📊 Health Status: {health_response.status}")
        print(f"⏱️  Response Time: Available")
        
        # Test 2: Generate Clinical Assertions
        print("\n🧪 Clinical Assertions Test")
        print("-" * 30)
        
        # Create test request
        from google.protobuf.struct_pb2 import Struct

        patient_context = Struct()
        patient_context.update({
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "allergy_ids": [],
            "metadata": {},
            "context_version": "",
            "assembly_time": "0001-01-01T00:00:00Z"
        })

        request = clinical_reasoning_pb2.ClinicalAssertionRequest(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            correlation_id="test_correlation_123",
            medication_ids=["warfarin", "aspirin", "ibuprofen"],
            condition_ids=["atrial_fibrillation", "hypertension"],
            patient_context=patient_context,
            priority=clinical_reasoning_pb2.PRIORITY_STANDARD,
            reasoner_types=["interaction", "contraindication", "duplicate_therapy", "dosing"]
        )
        
        print(f"📤 Testing with:")
        print(f"   Patient ID: {request.patient_id}")
        print(f"   Medications: {list(request.medication_ids)}")
        print(f"   Conditions: {list(request.condition_ids)}")
        print(f"   Reasoners: {list(request.reasoner_types)}")
        
        # Make the request
        start_time = datetime.now()
        response = await stub.GenerateAssertions(request)
        end_time = datetime.now()
        
        processing_time = (end_time - start_time).total_seconds() * 1000
        
        print(f"\n📥 CAE Response:")
        print(f"   Processing Time: {processing_time:.1f}ms")
        print(f"   Total Assertions: {len(response.assertions)}")
        print(f"   Request ID: {response.request_id}")
        print(f"   Correlation ID: {response.correlation_id}")

        if response.assertions:
            print(f"\n📋 Assertions Generated:")
            for i, assertion in enumerate(response.assertions, 1):
                print(f"   {i}. ID: {assertion.id}")
                print(f"      Type: {assertion.type}")
                print(f"      Severity: {assertion.severity}")
                print(f"      Title: {assertion.title}")
                print(f"      Description: {assertion.description}")
                print(f"      Confidence: {assertion.confidence_score}")
                if assertion.metadata:
                    print(f"      Metadata: {dict(assertion.metadata)}")
                print()
        else:
            print("   ⚠️  No assertions generated")

        if response.metadata:
            print(f"📊 Response Metadata:")
            print(f"   Reasoner Version: {response.metadata.reasoner_version}")
            print(f"   Knowledge Version: {response.metadata.knowledge_version}")
            print(f"   Processing Time: {response.metadata.processing_time_ms}ms")
            if response.metadata.warnings:
                print(f"   Warnings: {list(response.metadata.warnings)}")
        
        # Test 3: Error scenarios
        print("\n🧪 Error Scenario Test")
        print("-" * 30)
        
        # Test with invalid patient ID
        error_context = Struct()
        error_context.update({"allergy_ids": []})

        error_request = clinical_reasoning_pb2.ClinicalAssertionRequest(
            patient_id="invalid-patient-id",
            correlation_id="error_test_123",
            medication_ids=["unknown-medication"],
            condition_ids=[],
            patient_context=error_context,
            priority=clinical_reasoning_pb2.PRIORITY_STANDARD,
            reasoner_types=["interaction"]
        )
        
        try:
            error_response = await stub.GenerateAssertions(error_request)
            print(f"📥 Error Response:")
            print(f"   Request ID: {error_response.request_id}")
            print(f"   Assertions: {len(error_response.assertions)}")
            if error_response.metadata and error_response.metadata.warnings:
                print(f"   Warnings: {list(error_response.metadata.warnings)}")
        except grpc.RpcError as e:
            print(f"❌ gRPC Error: {e.code()} - {e.details()}")
        except Exception as e:
            print(f"❌ Unexpected Error: {e}")
            
    except grpc.RpcError as e:
        print(f"❌ gRPC Connection Error: {e.code()} - {e.details()}")
        print("   Make sure CAE service is running on localhost:8027")
    except Exception as e:
        print(f"❌ Unexpected Error: {e}")
        import traceback
        traceback.print_exc()
    finally:
        await channel.close()
    
    print("\n" + "=" * 70)
    print("🎉 DIRECT CAE TEST COMPLETED!")
    print("=" * 70)

if __name__ == "__main__":
    asyncio.run(test_cae_direct())
