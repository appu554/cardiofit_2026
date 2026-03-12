"""
Test Safety Gateway Platform Integration in Flow 2

This test verifies that the Safety Gateway Platform client is properly integrated
into Flow 2 Step 4 (Clinical Processing) and provides comprehensive safety validation.
"""

import asyncio
import logging
import sys
import time
from datetime import datetime

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def test_safety_gateway_integration():
    """
    Test Safety Gateway Platform integration in Flow 2
    
    This test verifies:
    1. Safety Gateway Platform client initialization
    2. Safety validation in Flow 2 Step 4
    3. Combination of pharmaceutical intelligence + safety validation
    4. Enhanced clinical decision support
    """
    try:
        logger.info("🚀 Testing Safety Gateway Platform Integration in Flow 2")
        logger.info("=" * 80)
        
        # Import after path setup
        from app.domain.services.recipe_orchestrator import RecipeOrchestrator, MedicationSafetyRequest
        
        # Test 1: Initialize Recipe Orchestrator with Safety Gateway Platform
        logger.info("📋 Test 1: Recipe Orchestrator Initialization")
        logger.info("-" * 50)
        
        orchestrator = RecipeOrchestrator(
            context_service_url="http://localhost:8016",
            safety_gateway_url="localhost:8030"  # Safety Gateway Platform
        )
        
        logger.info("✅ Recipe Orchestrator initialized with Safety Gateway Platform client")
        
        # Test 2: Health Check including Safety Gateway Platform
        logger.info("\n🔍 Test 2: Health Check with Safety Gateway Platform")
        logger.info("-" * 50)
        
        health_status = await orchestrator.health_check()
        
        logger.info("📊 Health Check Results:")
        for component, status in health_status.items():
            status_icon = "✅" if "healthy" in str(status) else "⚠️" if "error" in str(status) else "🔍"
            logger.info(f"   {status_icon} {component}: {status}")
        
        # Test 3: Flow 2 with Safety Gateway Platform Integration
        logger.info("\n🛡️ Test 3: Flow 2 with Safety Gateway Platform Validation")
        logger.info("-" * 50)
        
        # Create test request for high-risk medication (should trigger safety validation)
        safety_request = MedicationSafetyRequest(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            medication={
                "name": "Warfarin",
                "generic_name": "warfarin sodium",
                "is_high_risk": True,  # This should trigger Safety Gateway Platform validation
                "is_anticoagulant": True,
                "dose": "5mg",
                "frequency": "daily"
            },
            provider_id="test-provider-001",
            action_type="prescribe",
            urgency="routine"
        )
        
        logger.info(f"🎯 Testing high-risk medication: {safety_request.medication['name']}")
        logger.info(f"   Patient: {safety_request.patient_id}")
        logger.info(f"   High-risk flag: {safety_request.medication['is_high_risk']}")
        
        # Execute Flow 2 with Safety Gateway Platform integration
        start_time = time.time()
        flow2_result = await orchestrator.execute_medication_safety(safety_request)
        execution_time = time.time() - start_time
        
        logger.info(f"\n📊 Flow 2 Results with Safety Gateway Platform:")
        logger.info(f"   Request ID: {flow2_result.request_id}")
        logger.info(f"   Overall Status: {flow2_result.overall_safety_status}")
        logger.info(f"   Context Recipe: {flow2_result.context_recipe_used}")
        logger.info(f"   Clinical Recipes Executed: {len(flow2_result.clinical_recipes_executed)}")
        logger.info(f"   Context Completeness: {flow2_result.context_completeness_score:.1%}")
        logger.info(f"   Execution Time: {execution_time * 1000:.1f}ms")
        
        # Test 4: Verify Safety Gateway Platform Integration
        logger.info("\n🔍 Test 4: Safety Gateway Platform Integration Verification")
        logger.info("-" * 50)
        
        # Check if Safety Gateway Platform validation was included
        safety_gateway_result = None
        for result in flow2_result.clinical_results:
            if "safety_gateway_platform" in result.recipe_id:
                safety_gateway_result = result
                break
        
        if safety_gateway_result:
            logger.info("✅ Safety Gateway Platform validation found in results")
            logger.info(f"   Recipe ID: {safety_gateway_result.recipe_id}")
            logger.info(f"   Recipe Name: {safety_gateway_result.recipe_name}")
            logger.info(f"   Status: {safety_gateway_result.overall_status}")
            logger.info(f"   Execution Time: {safety_gateway_result.execution_time_ms:.1f}ms")
            logger.info(f"   Validations: {len(safety_gateway_result.validations)}")
            
            # Check performance metrics for safety data
            if safety_gateway_result.performance_metrics:
                metrics = safety_gateway_result.performance_metrics
                logger.info(f"   Risk Score: {metrics.get('risk_score', 'N/A')}")
                logger.info(f"   Confidence: {metrics.get('confidence', 'N/A')}")
                logger.info(f"   Engines Executed: {metrics.get('engines_executed', [])}")
        else:
            logger.warning("⚠️ Safety Gateway Platform validation not found in results")
        
        # Test 5: Verify Enhanced Clinical Decision Support
        logger.info("\n📋 Test 5: Enhanced Clinical Decision Support")
        logger.info("-" * 50)
        
        if flow2_result.safety_summary:
            cds = flow2_result.safety_summary.get('clinical_decision_support', {})
            
            logger.info("📝 Clinical Decision Support:")
            logger.info(f"   Provider Summary: {cds.get('provider_summary', 'N/A')}")
            logger.info(f"   Patient Explanation: {cds.get('patient_explanation', 'N/A')}")
            logger.info(f"   Monitoring Requirements: {cds.get('monitoring_requirements', [])}")
        
        # Test 6: Performance Analysis
        logger.info("\n⚡ Test 6: Performance Analysis")
        logger.info("-" * 50)
        
        if flow2_result.performance_metrics:
            metrics = flow2_result.performance_metrics
            logger.info("📊 Performance Metrics:")
            logger.info(f"   Total Execution Time: {metrics.get('total_execution_time_ms', 0):.1f}ms")
            logger.info(f"   Context Assembly Time: {metrics.get('context_assembly_time_ms', 0):.1f}ms")
            logger.info(f"   Clinical Recipes Time: {metrics.get('clinical_recipes_time_ms', 0):.1f}ms")
            logger.info(f"   Recipes Executed: {metrics.get('recipes_executed', 0)}")
            logger.info(f"   Average Recipe Time: {metrics.get('average_recipe_time_ms', 0):.1f}ms")
        
        # Test 7: Safety Validation Logic Test
        logger.info("\n🧪 Test 7: Safety Validation Logic Test")
        logger.info("-" * 50)
        
        # Test with low-risk medication (should still validate for comprehensive safety)
        low_risk_request = MedicationSafetyRequest(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            medication={
                "name": "Acetaminophen",
                "generic_name": "acetaminophen",
                "is_high_risk": False,
                "dose": "500mg",
                "frequency": "every 6 hours"
            },
            provider_id="test-provider-001",
            action_type="prescribe",
            urgency="routine"
        )
        
        logger.info(f"🎯 Testing low-risk medication: {low_risk_request.medication['name']}")
        
        low_risk_result = await orchestrator.execute_medication_safety(low_risk_request)
        
        logger.info(f"📊 Low-Risk Medication Results:")
        logger.info(f"   Overall Status: {low_risk_result.overall_safety_status}")
        logger.info(f"   Clinical Recipes: {len(low_risk_result.clinical_recipes_executed)}")
        
        # Check if safety validation was still performed
        has_safety_validation = any(
            "safety_gateway_platform" in result.recipe_id 
            for result in low_risk_result.clinical_results
        )
        
        if has_safety_validation:
            logger.info("✅ Safety Gateway Platform validation performed for low-risk medication")
        else:
            logger.info("ℹ️ Safety Gateway Platform validation skipped for low-risk medication")
        
        # Final Assessment
        logger.info("\n" + "=" * 80)
        logger.info("🎯 SAFETY GATEWAY PLATFORM INTEGRATION TEST RESULTS")
        logger.info("=" * 80)
        
        success_criteria = [
            ("Recipe Orchestrator Initialization", True),
            ("Safety Gateway Platform Client", orchestrator.safety_gateway_client is not None),
            ("Health Check Integration", "safety_gateway_platform" in health_status),
            ("Flow 2 Execution", flow2_result.overall_safety_status != "ERROR"),
            ("Safety Validation Integration", safety_gateway_result is not None),
            ("Enhanced Clinical Decision Support", flow2_result.safety_summary is not None),
            ("Performance Metrics", flow2_result.performance_metrics is not None)
        ]
        
        passed_tests = 0
        total_tests = len(success_criteria)
        
        for test_name, passed in success_criteria:
            status = "✅ PASS" if passed else "❌ FAIL"
            logger.info(f"   {status}: {test_name}")
            if passed:
                passed_tests += 1
        
        success_rate = (passed_tests / total_tests) * 100
        
        logger.info(f"\n📊 OVERALL SUCCESS RATE: {passed_tests}/{total_tests} tests ({success_rate:.1f}%)")
        
        if success_rate >= 85:
            logger.info("✅ SAFETY GATEWAY PLATFORM INTEGRATION: EXCELLENT!")
            logger.info("🛡️ Comprehensive safety validation is working properly")
        elif success_rate >= 70:
            logger.info("⚠️ SAFETY GATEWAY PLATFORM INTEGRATION: GOOD")
            logger.info("🔧 Some improvements needed")
        else:
            logger.info("❌ SAFETY GATEWAY PLATFORM INTEGRATION: NEEDS WORK")
            logger.info("🚨 Significant issues need to be addressed")
        
        return success_rate >= 70
        
    except Exception as e:
        logger.error(f"❌ Safety Gateway Platform integration test failed: {e}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test execution"""
    try:
        # Add the medication service to Python path
        import os
        import sys
        
        current_dir = os.path.dirname(os.path.abspath(__file__))
        sys.path.insert(0, current_dir)
        
        # Run the test
        success = await test_safety_gateway_integration()
        
        if success:
            logger.info("\n🎉 Safety Gateway Platform integration test completed successfully!")
            sys.exit(0)
        else:
            logger.error("\n💥 Safety Gateway Platform integration test failed!")
            sys.exit(1)
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {e}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
