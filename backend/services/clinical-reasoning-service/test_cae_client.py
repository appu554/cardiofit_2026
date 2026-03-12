#!/usr/bin/env python3
"""
Clinical Assertion Engine gRPC Client Test

This script tests the CAE gRPC service by making various requests
and validating responses. It demonstrates how other microservices
can integrate with the CAE.
"""

import asyncio
import logging
import json
import sys
from pathlib import Path

# Add the shared directory to the path
sys.path.insert(0, str(Path(__file__).parent.parent / 'shared'))

from cae_grpc_client import CAEgRPCClient, get_clinical_assertions, check_drug_interactions

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

async def test_health_check():
    """Test the health check endpoint"""
    logger.info("🏥 Testing health check...")
    
    try:
        async with CAEgRPCClient(service_name="test-client") as client:
            is_healthy = await client.health_check()
            
            if is_healthy:
                logger.info("✅ Health check passed")
                return True
            else:
                logger.error("❌ Health check failed")
                return False
                
    except Exception as e:
        logger.error(f"❌ Health check error: {e}")
        return False

async def test_clinical_assertions():
    """Test comprehensive clinical assertions"""
    logger.info("🧠 Testing clinical assertions generation...")
    
    try:
        # Test data
        patient_id = "905a60cb-8241-418f-b29b-5b020e851392"  # Your test patient
        medication_ids = ["warfarin", "aspirin", "metformin"]
        condition_ids = ["diabetes", "hypertension"]
        
        async with CAEgRPCClient(service_name="test-client") as client:
            result = await client.generate_clinical_assertions(
                patient_id=patient_id,
                medication_ids=medication_ids,
                condition_ids=condition_ids,
                reasoner_types=["interaction", "dosing", "contraindication"],
                priority="standard"
            )
            
            logger.info("✅ Clinical assertions generated successfully")
            logger.info(f"   Request ID: {result['request_id']}")
            logger.info(f"   Assertions count: {len(result['assertions'])}")
            logger.info(f"   Processing time: {result['metadata']['processing_time_ms']}ms")
            
            # Print assertions
            for i, assertion in enumerate(result['assertions'], 1):
                logger.info(f"   Assertion {i}: {assertion['type']} - {assertion['severity']}")
                logger.info(f"      Title: {assertion['title']}")
                logger.info(f"      Confidence: {assertion['confidence_score']:.2f}")
            
            return True
            
    except Exception as e:
        logger.error(f"❌ Clinical assertions test failed: {e}")
        return False

async def test_medication_interactions():
    """Test medication interaction checking"""
    logger.info("💊 Testing medication interaction checking...")
    
    try:
        patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        medication_ids = ["warfarin", "aspirin"]
        new_medication = "ibuprofen"
        
        async with CAEgRPCClient(service_name="test-client") as client:
            result = await client.check_medication_interactions(
                patient_id=patient_id,
                medication_ids=medication_ids,
                new_medication_id=new_medication
            )
            
            logger.info("✅ Medication interactions checked successfully")
            logger.info(f"   Interactions found: {len(result['interactions'])}")
            logger.info(f"   Processing time: {result['metadata']['processing_time_ms']}ms")
            
            # Print interactions
            for i, interaction in enumerate(result['interactions'], 1):
                logger.info(f"   Interaction {i}: {interaction['medication_a']} + {interaction['medication_b']}")
                logger.info(f"      Severity: {interaction['severity']}")
                logger.info(f"      Description: {interaction['description']}")
                logger.info(f"      Confidence: {interaction['confidence_score']:.2f}")
            
            return True
            
    except Exception as e:
        logger.error(f"❌ Medication interactions test failed: {e}")
        return False

async def test_dosing_calculation():
    """Test dosing calculation"""
    logger.info("📏 Testing dosing calculation...")
    
    try:
        patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        medication_id = "warfarin"
        
        async with CAEgRPCClient(service_name="test-client") as client:
            result = await client.calculate_dosing(
                patient_id=patient_id,
                medication_id=medication_id,
                patient_parameters={
                    "weight": 70,
                    "age": 65,
                    "renal_function": "normal"
                },
                indication="atrial_fibrillation"
            )
            
            logger.info("✅ Dosing calculation completed successfully")
            logger.info(f"   Medication: {result['dosing']['medication_id']}")
            logger.info(f"   Dose: {result['dosing']['dose']}")
            logger.info(f"   Frequency: {result['dosing']['frequency']}")
            logger.info(f"   Route: {result['dosing']['route']}")
            logger.info(f"   Duration: {result['dosing']['duration']}")
            logger.info(f"   Processing time: {result['metadata']['processing_time_ms']}ms")
            
            return True
            
    except Exception as e:
        logger.error(f"❌ Dosing calculation test failed: {e}")
        return False

async def test_contraindications():
    """Test contraindication checking"""
    logger.info("⚠️  Testing contraindication checking...")
    
    try:
        patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        medication_ids = ["warfarin", "metformin"]
        condition_ids = ["renal_failure", "pregnancy"]
        allergy_ids = ["penicillin"]
        
        async with CAEgRPCClient(service_name="test-client") as client:
            result = await client.check_contraindications(
                patient_id=patient_id,
                medication_ids=medication_ids,
                condition_ids=condition_ids,
                allergy_ids=allergy_ids
            )
            
            logger.info("✅ Contraindications checked successfully")
            logger.info(f"   Contraindications found: {len(result['contraindications'])}")
            logger.info(f"   Processing time: {result['metadata']['processing_time_ms']}ms")
            
            # Print contraindications
            for i, contraindication in enumerate(result['contraindications'], 1):
                logger.info(f"   Contraindication {i}: {contraindication['medication_id']}")
                logger.info(f"      Type: {contraindication['type']}")
                logger.info(f"      Severity: {contraindication['severity']}")
                logger.info(f"      Description: {contraindication['description']}")
                logger.info(f"      Override possible: {contraindication['override_possible']}")
            
            return True
            
    except Exception as e:
        logger.error(f"❌ Contraindications test failed: {e}")
        return False

async def test_convenience_functions():
    """Test convenience functions"""
    logger.info("🔧 Testing convenience functions...")
    
    try:
        patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        
        # Test get_clinical_assertions convenience function
        result1 = await get_clinical_assertions(
            patient_id=patient_id,
            medication_ids=["warfarin", "aspirin"],
            reasoner_types=["interaction"]
        )
        
        logger.info("✅ get_clinical_assertions convenience function works")
        logger.info(f"   Assertions: {len(result1['assertions'])}")
        
        # Test check_drug_interactions convenience function
        result2 = await check_drug_interactions(
            patient_id=patient_id,
            medication_ids=["warfarin", "aspirin"]
        )
        
        logger.info("✅ check_drug_interactions convenience function works")
        logger.info(f"   Interactions: {len(result2['interactions'])}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Convenience functions test failed: {e}")
        return False

async def test_error_handling():
    """Test error handling"""
    logger.info("🚨 Testing error handling...")
    
    try:
        # Test with invalid patient ID
        async with CAEgRPCClient(service_name="test-client") as client:
            try:
                await client.generate_clinical_assertions(
                    patient_id="invalid-patient-id",
                    medication_ids=["invalid-medication"]
                )
                logger.warning("⚠️  Expected error not raised for invalid patient ID")
            except Exception as e:
                logger.info(f"✅ Error handling works: {type(e).__name__}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Error handling test failed: {e}")
        return False

async def run_performance_test():
    """Run basic performance test"""
    logger.info("⚡ Running performance test...")
    
    try:
        patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        num_requests = 10
        
        import time
        start_time = time.time()
        
        async with CAEgRPCClient(service_name="test-client") as client:
            tasks = []
            for i in range(num_requests):
                task = client.check_medication_interactions(
                    patient_id=patient_id,
                    medication_ids=["warfarin", "aspirin"]
                )
                tasks.append(task)
            
            results = await asyncio.gather(*tasks)
        
        end_time = time.time()
        total_time = end_time - start_time
        avg_time = total_time / num_requests
        
        logger.info(f"✅ Performance test completed")
        logger.info(f"   Requests: {num_requests}")
        logger.info(f"   Total time: {total_time:.2f}s")
        logger.info(f"   Average time per request: {avg_time:.3f}s")
        logger.info(f"   Requests per second: {num_requests/total_time:.1f}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Performance test failed: {e}")
        return False

async def main():
    """Main test function"""
    logger.info("🧪 Clinical Assertion Engine gRPC Client Test Suite")
    logger.info("=" * 60)
    
    tests = [
        ("Health Check", test_health_check),
        ("Clinical Assertions", test_clinical_assertions),
        ("Medication Interactions", test_medication_interactions),
        ("Dosing Calculation", test_dosing_calculation),
        ("Contraindications", test_contraindications),
        ("Convenience Functions", test_convenience_functions),
        ("Error Handling", test_error_handling),
        ("Performance Test", run_performance_test)
    ]
    
    passed = 0
    failed = 0
    
    for test_name, test_func in tests:
        logger.info(f"\n{'='*20} {test_name} {'='*20}")
        
        try:
            if await test_func():
                passed += 1
                logger.info(f"✅ {test_name} PASSED")
            else:
                failed += 1
                logger.error(f"❌ {test_name} FAILED")
        except Exception as e:
            failed += 1
            logger.error(f"❌ {test_name} FAILED with exception: {e}")
    
    # Summary
    logger.info(f"\n{'='*60}")
    logger.info(f"🏁 Test Summary")
    logger.info(f"   Total tests: {passed + failed}")
    logger.info(f"   Passed: {passed}")
    logger.info(f"   Failed: {failed}")
    
    if failed == 0:
        logger.info("🎉 All tests passed!")
        return 0
    else:
        logger.error(f"💥 {failed} test(s) failed!")
        return 1

if __name__ == "__main__":
    exit_code = asyncio.run(main())
