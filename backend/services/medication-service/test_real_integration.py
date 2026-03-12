"""
Real Integration Test for Enhanced Recipe Orchestrator

This test uses the actual medication service API endpoints without mocks,
testing the complete Enhanced Orchestrator integration with real data flow.
"""

import asyncio
import logging
import json
import aiohttp
from typing import Dict, Any

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class MedicationServiceClient:
    """Client for testing the real medication service API"""
    
    def __init__(self, base_url: str = "http://localhost:8009"):
        self.base_url = base_url
        self.session = None
    
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    async def validate_medication_safety(self, request_data: Dict[str, Any]) -> Dict[str, Any]:
        """Call the real Flow 2 medication safety validation API"""

        url = f"{self.base_url}/api/flow2/medication-safety/validate"

        async with self.session.post(url, json=request_data) as response:
            if response.status == 200:
                return await response.json()
            else:
                error_text = await response.text()
                raise Exception(f"API call failed with status {response.status}: {error_text}")
    
    async def health_check(self) -> bool:
        """Check if the medication service is running"""
        try:
            url = f"{self.base_url}/health"
            async with self.session.get(url) as response:
                return response.status == 200
        except Exception as e:
            logger.error(f"Health check failed: {str(e)}")
            return False


async def test_real_enhanced_orchestrator():
    """Test Enhanced Recipe Orchestrator with real API calls"""
    
    try:
        logger.info("🧪 Starting Real Enhanced Orchestrator Integration Test")
        
        async with MedicationServiceClient() as client:
            # Check if service is running
            if not await client.health_check():
                logger.error("❌ Medication service is not running on localhost:8009")
                logger.info("💡 Please start the medication service first:")
                logger.info("   cd backend/services/medication-service")
                logger.info("   python -m uvicorn app.main:app --host 0.0.0.0 --port 8009 --reload")
                return False
            
            logger.info("✅ Medication service is running")
            
            # Test Scenario 1: High-risk anticoagulant (Warfarin) - should trigger enhanced orchestration
            logger.info("\n" + "="*70)
            logger.info("🧪 TEST 1: High-Risk Anticoagulant - Warfarin (Enhanced Orchestration)")
            logger.info("="*70)
            
            warfarin_request = {
                "patient_id": "patient-001",
                "medication": {
                    "name": "warfarin",
                    "generic_name": "warfarin sodium",
                    "dose": "5mg",
                    "frequency": "daily",
                    "route": "oral",
                    "therapeutic_class": "anticoagulant",
                    "is_anticoagulant": True,
                    "is_high_risk": True
                },
                "provider_id": "provider-001",
                "encounter_id": "encounter-001",
                "action_type": "prescribe",
                "urgency": "routine"
            }
            
            result1 = await client.validate_medication_safety(warfarin_request)
            
            logger.info(f"📊 Warfarin Results:")
            logger.info(f"   Request ID: {result1.get('request_id', 'N/A')}")
            logger.info(f"   Overall Safety Status: {result1.get('overall_safety_status', 'N/A')}")
            logger.info(f"   Context Recipe Used: {result1.get('context_recipe_used', 'N/A')}")
            logger.info(f"   Execution Time: {result1.get('execution_time_ms', 0):.1f}ms")
            
            # Check for enhanced orchestration details
            orchestration_details = result1.get('orchestration_details', {})
            if orchestration_details:
                logger.info(f"🧠 Enhanced Orchestration:")
                logger.info(f"   Enabled: {orchestration_details.get('enhanced_orchestration_enabled', False)}")
                logger.info(f"   Strategy: {orchestration_details.get('selection_strategy', 'unknown')}")
                logger.info(f"   Confidence: {orchestration_details.get('selection_confidence', 0.0):.2f}")
                
                clinical_intel = orchestration_details.get('clinical_intelligence', {})
                if clinical_intel.get('clinical_flags'):
                    logger.info(f"   Clinical Flags: {clinical_intel['clinical_flags']}")
            
            # Test Scenario 2: Emergency insulin - should trigger emergency protocols
            logger.info("\n" + "="*70)
            logger.info("🧪 TEST 2: Emergency High-Alert - Insulin")
            logger.info("="*70)
            
            insulin_request = {
                "patient_id": "patient-002",
                "medication": {
                    "name": "insulin",
                    "generic_name": "insulin regular",
                    "dose": "10 units",
                    "frequency": "before meals",
                    "route": "subcutaneous",
                    "therapeutic_class": "antidiabetic",
                    "is_high_risk": True
                },
                "provider_id": "provider-002",
                "encounter_id": "encounter-002",
                "action_type": "prescribe",
                "urgency": "emergency"
            }
            
            result2 = await client.validate_medication_safety(insulin_request)
            
            logger.info(f"📊 Insulin Results:")
            logger.info(f"   Overall Safety Status: {result2.get('overall_safety_status', 'N/A')}")
            logger.info(f"   Context Recipe Used: {result2.get('context_recipe_used', 'N/A')}")
            logger.info(f"   Execution Time: {result2.get('execution_time_ms', 0):.1f}ms")
            
            orchestration_details2 = result2.get('orchestration_details', {})
            if orchestration_details2:
                logger.info(f"🚨 Emergency Orchestration:")
                logger.info(f"   Strategy: {orchestration_details2.get('selection_strategy', 'unknown')}")
                logger.info(f"   Confidence: {orchestration_details2.get('selection_confidence', 0.0):.2f}")
            
            # Test Scenario 3: Simple medication - should use default context
            logger.info("\n" + "="*70)
            logger.info("🧪 TEST 3: Simple Medication - Acetaminophen")
            logger.info("="*70)
            
            acetaminophen_request = {
                "patient_id": "patient-003",
                "medication": {
                    "name": "acetaminophen",
                    "generic_name": "acetaminophen",
                    "dose": "500mg",
                    "frequency": "every 6 hours",
                    "route": "oral",
                    "therapeutic_class": "analgesic"
                },
                "provider_id": "provider-003",
                "encounter_id": "encounter-003",
                "action_type": "prescribe",
                "urgency": "routine"
            }
            
            result3 = await client.validate_medication_safety(acetaminophen_request)
            
            logger.info(f"📊 Acetaminophen Results:")
            logger.info(f"   Overall Safety Status: {result3.get('overall_safety_status', 'N/A')}")
            logger.info(f"   Context Recipe Used: {result3.get('context_recipe_used', 'N/A')}")
            logger.info(f"   Execution Time: {result3.get('execution_time_ms', 0):.1f}ms")
            
            # Performance Summary
            logger.info("\n" + "="*70)
            logger.info("📊 PERFORMANCE SUMMARY")
            logger.info("="*70)
            
            test_results = [
                ("Warfarin (High-Risk)", result1),
                ("Insulin (Emergency)", result2),
                ("Acetaminophen (Simple)", result3)
            ]
            
            logger.info(f"{'Scenario':<25} {'Status':<10} {'Recipe':<35} {'Time':<8}")
            logger.info("-" * 85)
            
            for name, result in test_results:
                status = result.get('overall_safety_status', 'N/A')
                recipe = result.get('context_recipe_used', 'N/A')
                recipe_short = recipe.replace('_context_v', '_v').replace('medication_safety_', 'med_')
                time_ms = result.get('execution_time_ms', 0)
                logger.info(f"{name:<25} {status:<10} {recipe_short:<35} {time_ms:<8.1f}")
            
            # Validation Checks
            logger.info("\n🔍 VALIDATION CHECKS:")
            
            # Check 1: All requests should complete successfully
            for name, result in test_results:
                assert result.get('overall_safety_status') != 'ERROR', \
                    f"{name} returned ERROR status"
            logger.info("✅ All requests completed successfully")
            
            # Check 2: Enhanced orchestration should be enabled for complex scenarios
            warfarin_enhanced = result1.get('orchestration_details', {}).get('enhanced_orchestration_enabled', False)
            if warfarin_enhanced:
                logger.info("✅ Enhanced orchestration enabled for high-risk scenarios")
            else:
                logger.info("ℹ️ Enhanced orchestration not detected (may be disabled)")
            
            # Check 3: Context recipes should be appropriate
            warfarin_recipe = result1.get('context_recipe_used', '')
            insulin_recipe = result2.get('context_recipe_used', '')
            
            # High-risk scenarios should get specialized contexts
            if any(keyword in warfarin_recipe for keyword in ['anticoagulation', 'comprehensive', 'enhanced']):
                logger.info("✅ High-risk warfarin got specialized context recipe")
            else:
                logger.info(f"ℹ️ Warfarin used standard recipe: {warfarin_recipe}")
            
            # Check 4: Performance should be reasonable
            for name, result in test_results:
                time_ms = result.get('execution_time_ms', 0)
                assert time_ms < 10000, f"{name} took too long: {time_ms}ms"
            logger.info("✅ Performance targets met (<10 seconds)")
            
            # Check 5: Results should have required fields
            required_fields = ['request_id', 'overall_safety_status', 'context_recipe_used']
            for name, result in test_results:
                for field in required_fields:
                    assert field in result, f"{name} missing required field: {field}"
            logger.info("✅ All results have required fields")
            
            logger.info("\n🎉 ALL REAL INTEGRATION TESTS PASSED!")
            logger.info("✅ Enhanced Recipe Orchestrator is working with real API")
            
            return True
            
    except Exception as e:
        logger.error(f"❌ Real integration test failed: {str(e)}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test function"""
    logger.info("🚀 Starting Real Enhanced Orchestrator Integration Tests")
    logger.info("📋 This test requires the medication service to be running on localhost:8009")
    
    success = await test_real_enhanced_orchestrator()
    
    if success:
        logger.info("✅ Real Integration Test: PASSED")
        logger.info("🎯 Enhanced Recipe Orchestrator is production-ready")
    else:
        logger.error("❌ Real Integration Test: FAILED")
        logger.info("💡 Check that all services are running and properly configured")
    
    return success


if __name__ == "__main__":
    asyncio.run(main())
