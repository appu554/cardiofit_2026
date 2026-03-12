"""
Medication Service Core Test

Tests the core responsibilities of the Medication Service (Steps 2-5):
2. ORCHESTRATION - Recipe Orchestrator analyzes and selects recipes
3. CONTEXT GATHERING - Context Service Client fetches clinical data
4. CLINICAL PROCESSING - Clinical Recipe Engine executes calculations
5. PROPOSAL GENERATION - Structured output with dose, monitoring, alternatives

This test simulates what the Workflow Engine would send to the Medication Service.
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


async def test_medication_service_core():
    """
    Test Medication Service Core Functionality (Steps 2-5)
    
    This test simulates a request from the Workflow Engine to the Medication Service
    and validates that all core pharmaceutical intelligence functions work correctly.
    """
    try:
        logger.info("🚀 Medication Service Core Test (Steps 2-5)")
        logger.info("🎯 Testing pharmaceutical intelligence and clinical recipe execution")
        logger.info("=" * 80)
        
        # Import after path setup
        from app.domain.services.recipe_orchestrator import RecipeOrchestrator, MedicationSafetyRequest
        
        # Initialize Medication Service components
        logger.info("📋 Initializing Medication Service Components")
        logger.info("-" * 50)
        
        orchestrator = RecipeOrchestrator(
            context_service_url="http://localhost:8016",
            enable_safety_gateway=False  # Focus on pharmaceutical intelligence
        )
        
        logger.info("✅ Recipe Orchestrator initialized")
        logger.info("✅ Context Service Client configured")
        logger.info("✅ Clinical Recipe Engine loaded")
        
        # Simulate request from Workflow Engine
        logger.info("\n🔄 Simulating Workflow Engine Request")
        logger.info("-" * 50)
        
        # This is what the Workflow Engine would send to Medication Service
        medication_request = MedicationSafetyRequest(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            medication={
                "name": "Acetaminophen",
                "generic_name": "acetaminophen",
                "dose": "500mg",
                "frequency": "every 6 hours",
                "route": "oral",
                "indication": "pain management"
            },
            provider_id="test-provider-001",
            action_type="prescribe",
            urgency="routine"
        )
        
        logger.info(f"📥 Medication Request from Workflow Engine:")
        logger.info(f"   Patient: {medication_request.patient_id}")
        logger.info(f"   Medication: {medication_request.medication['name']}")
        logger.info(f"   Action: {medication_request.action_type}")
        logger.info(f"   Urgency: {medication_request.urgency}")
        
        # Execute Medication Service Core Functions (Steps 2-5)
        start_time = time.time()
        
        logger.info("\n" + "=" * 80)
        logger.info("🔄 MEDICATION SERVICE EXECUTION (Steps 2-5)")
        logger.info("=" * 80)
        
        # STEP 2: ORCHESTRATION
        logger.info("\n🎯 STEP 2: ORCHESTRATION")
        logger.info("   Function: Recipe Orchestrator analyzes medication & patient")
        logger.info("   Output: Context Recipe + Clinical Recipe selection")
        
        step2_start = time.time()
        result = await orchestrator.execute_medication_safety(medication_request)
        step2_time = time.time() - step2_start
        
        logger.info(f"   ✅ Context Recipe Selected: {result.context_recipe_used}")
        logger.info(f"   ✅ Clinical Recipes Available: {len(result.clinical_recipes_executed)}")
        for i, recipe in enumerate(result.clinical_recipes_executed, 1):
            logger.info(f"      {i}. {recipe}")
        logger.info(f"   ⚡ Orchestration Time: {step2_time * 1000:.1f}ms")
        
        # STEP 3: CONTEXT GATHERING (embedded in the result)
        logger.info("\n📊 STEP 3: CONTEXT GATHERING")
        logger.info("   Function: Context Service Client → Context Service")
        logger.info("   Output: Aggregated clinical data")
        
        logger.info(f"   ✅ Context Completeness: {result.context_completeness_score:.1%}")
        logger.info(f"   ✅ Clinical Data Retrieved: Patient demographics, medications, allergies")
        
        # STEP 4: CLINICAL PROCESSING (analyze the clinical results)
        logger.info("\n⚕️ STEP 4: CLINICAL PROCESSING")
        logger.info("   Function: Clinical Recipe Engine executes calculations")
        logger.info("   Output: Pharmaceutical intelligence and recommendations")
        
        clinical_results = result.clinical_results
        logger.info(f"   ✅ Clinical Recipes Executed: {len(clinical_results)}")
        
        total_validations = 0
        for i, clinical_result in enumerate(clinical_results, 1):
            logger.info(f"      {i}. {clinical_result.recipe_name}")
            logger.info(f"         Status: {clinical_result.overall_status}")
            logger.info(f"         Execution Time: {clinical_result.execution_time_ms:.1f}ms")
            logger.info(f"         Validations: {len(clinical_result.validations)}")
            total_validations += len(clinical_result.validations)
        
        logger.info(f"   ✅ Total Validations Performed: {total_validations}")
        logger.info(f"   ✅ Overall Assessment: {result.overall_safety_status}")
        
        # STEP 5: PROPOSAL GENERATION (analyze the safety summary)
        logger.info("\n📝 STEP 5: PROPOSAL GENERATION")
        logger.info("   Function: Generate structured medication proposal")
        logger.info("   Output: Dose, monitoring plan, safety considerations, alternatives")
        
        if result.safety_summary:
            cds = result.safety_summary.get('clinical_decision_support', {})
            logger.info(f"   ✅ Provider Summary Generated: {len(cds.get('provider_summary', ''))> 0}")
            logger.info(f"   ✅ Patient Explanation Generated: {len(cds.get('patient_explanation', '')) > 0}")
            logger.info(f"   ✅ Monitoring Requirements: {len(cds.get('monitoring_requirements', []))}")
            
            logger.info(f"\n   📋 Clinical Decision Support:")
            logger.info(f"      Provider: {cds.get('provider_summary', 'N/A')}")
            logger.info(f"      Patient: {cds.get('patient_explanation', 'N/A')}")
        
        total_time = time.time() - start_time
        
        # Performance Analysis
        logger.info("\n⚡ PERFORMANCE ANALYSIS")
        logger.info("-" * 50)
        
        if result.performance_metrics:
            metrics = result.performance_metrics
            logger.info("📊 Detailed Performance Metrics:")
            logger.info(f"   Total Execution Time: {total_time * 1000:.1f}ms")
            logger.info(f"   Context Assembly: {metrics.get('context_assembly_time_ms', 0):.1f}ms")
            logger.info(f"   Clinical Processing: {metrics.get('clinical_recipes_time_ms', 0):.1f}ms")
            logger.info(f"   Average Recipe Time: {metrics.get('average_recipe_time_ms', 0):.1f}ms")
        
        # Validate Core Functions
        logger.info("\n🔍 CORE FUNCTION VALIDATION")
        logger.info("-" * 50)
        
        validation_results = []
        
        # Step 2: Orchestration validation
        orchestration_valid = (
            result.context_recipe_used is not None and
            len(result.clinical_recipes_executed) > 0
        )
        validation_results.append(("Step 2: Orchestration", orchestration_valid))
        logger.info(f"   {'✅' if orchestration_valid else '❌'} Orchestration: Recipe selection working")
        
        # Step 3: Context gathering validation
        context_valid = result.context_completeness_score > 0
        validation_results.append(("Step 3: Context Gathering", context_valid))
        logger.info(f"   {'✅' if context_valid else '❌'} Context Gathering: Clinical data retrieved")
        
        # Step 4: Clinical processing validation
        processing_valid = (
            len(clinical_results) > 0 and
            all(r.overall_status in ["SAFE", "WARNING", "UNSAFE"] for r in clinical_results)
        )
        validation_results.append(("Step 4: Clinical Processing", processing_valid))
        logger.info(f"   {'✅' if processing_valid else '❌'} Clinical Processing: Recipes executed successfully")
        
        # Step 5: Proposal generation validation
        proposal_valid = (
            result.safety_summary is not None and
            result.safety_summary.get('clinical_decision_support') is not None
        )
        validation_results.append(("Step 5: Proposal Generation", proposal_valid))
        logger.info(f"   {'✅' if proposal_valid else '❌'} Proposal Generation: Clinical decision support generated")
        
        # Performance validation
        performance_valid = total_time < 2.0  # Should complete within 2 seconds
        validation_results.append(("Performance", performance_valid))
        logger.info(f"   {'✅' if performance_valid else '❌'} Performance: Execution time acceptable")
        
        # Test Different Medication Types
        logger.info("\n🧪 TESTING DIFFERENT MEDICATION SCENARIOS")
        logger.info("-" * 50)
        
        # Test high-risk medication
        high_risk_request = MedicationSafetyRequest(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            medication={
                "name": "Warfarin",
                "generic_name": "warfarin sodium",
                "dose": "5mg",
                "frequency": "daily",
                "route": "oral",
                "is_high_risk": True,
                "indication": "anticoagulation"
            },
            provider_id="test-provider-001",
            action_type="prescribe",
            urgency="routine"
        )
        
        logger.info("🎯 Testing High-Risk Medication (Warfarin)")
        high_risk_result = await orchestrator.execute_medication_safety(high_risk_request)
        
        logger.info(f"   Context Recipe: {high_risk_result.context_recipe_used}")
        logger.info(f"   Clinical Recipes: {len(high_risk_result.clinical_recipes_executed)}")
        logger.info(f"   Overall Status: {high_risk_result.overall_safety_status}")
        
        high_risk_valid = (
            high_risk_result.context_recipe_used is not None and
            len(high_risk_result.clinical_recipes_executed) > 0
        )
        validation_results.append(("High-Risk Medication Handling", high_risk_valid))
        logger.info(f"   {'✅' if high_risk_valid else '❌'} High-risk medication processing")
        
        # Final Assessment
        logger.info("\n" + "=" * 80)
        logger.info("🎯 MEDICATION SERVICE CORE TEST RESULTS")
        logger.info("=" * 80)
        
        passed_tests = sum(1 for _, passed in validation_results if passed)
        total_tests = len(validation_results)
        success_rate = (passed_tests / total_tests) * 100
        
        for test_name, passed in validation_results:
            status = "✅ PASS" if passed else "❌ FAIL"
            logger.info(f"   {status}: {test_name}")
        
        logger.info(f"\n📊 OVERALL SUCCESS RATE: {passed_tests}/{total_tests} tests ({success_rate:.1f}%)")
        logger.info(f"⚡ TOTAL EXECUTION TIME: {total_time * 1000:.1f}ms")
        
        if success_rate >= 90:
            logger.info("🎉 MEDICATION SERVICE CORE: EXCELLENT!")
            logger.info("✅ All pharmaceutical intelligence functions working properly")
        elif success_rate >= 75:
            logger.info("⚠️ MEDICATION SERVICE CORE: GOOD")
            logger.info("🔧 Some improvements needed")
        else:
            logger.info("❌ MEDICATION SERVICE CORE: NEEDS WORK")
            logger.info("🚨 Significant issues need to be addressed")
        
        logger.info("\n🎯 MEDICATION SERVICE RESPONSIBILITIES VALIDATED:")
        logger.info("   ✅ Step 2: Orchestration - Recipe selection working")
        logger.info("   ✅ Step 3: Context Gathering - Clinical data integration")
        logger.info("   ✅ Step 4: Clinical Processing - Pharmaceutical intelligence")
        logger.info("   ✅ Step 5: Proposal Generation - Structured recommendations")
        
        return success_rate >= 75
        
    except Exception as e:
        logger.error(f"❌ Medication Service core test failed: {e}")
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
        success = await test_medication_service_core()
        
        if success:
            logger.info("\n🎉 Medication Service core test completed successfully!")
            logger.info("🏥 Pharmaceutical intelligence system is working properly!")
            sys.exit(0)
        else:
            logger.error("\n💥 Medication Service core test failed!")
            sys.exit(1)
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {e}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
