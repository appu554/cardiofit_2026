"""
Simple Flow 2 Integration Test

A simple test script to validate the Flow 2 implementation works correctly.
This script tests the core components without external dependencies.
"""

import asyncio
import sys
import os
import logging
from datetime import datetime

# Add the app directory to the Python path
sys.path.append(os.path.join(os.path.dirname(__file__), 'app'))

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def test_recipe_orchestrator():
    """Test the Recipe Orchestrator functionality"""
    try:
        from app.domain.services.recipe_orchestrator import RecipeOrchestrator, MedicationSafetyRequest
        
        logger.info("🧪 Testing Recipe Orchestrator...")
        
        # Create orchestrator
        orchestrator = RecipeOrchestrator()
        
        # Test context recipe determination
        request = MedicationSafetyRequest(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            medication={
                "name": "warfarin",
                "dose": "5mg",
                "frequency": "daily",
                "is_anticoagulant": True
            },
            provider_id="provider_123",
            action_type="prescribe",
            urgency="routine"
        )
        
        # Test recipe determination
        context_recipe_id = orchestrator._determine_context_recipe(request)
        logger.info(f"✅ Context recipe determined: {context_recipe_id}")
        
        # Test minimal context fallback
        minimal_context = await orchestrator._get_minimal_context_fallback(request)
        logger.info(f"✅ Minimal context fallback works: {minimal_context.context_id}")
        
        # Test health check
        health_status = await orchestrator.health_check()
        logger.info(f"✅ Health check completed: {health_status['recipe_orchestrator']}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Recipe Orchestrator test failed: {str(e)}")
        return False


async def test_context_data_adapter():
    """Test the Context Data Adapter functionality"""
    try:
        from app.domain.services.context_data_adapter import ContextDataAdapter
        from app.infrastructure.context_service_client import ClinicalContext
        
        logger.info("🧪 Testing Context Data Adapter...")
        
        # Create adapter
        adapter = ContextDataAdapter()
        
        # Create sample context data
        sample_context = ClinicalContext(
            context_id="test_context_123",
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            recipe_used="medication_safety_base_context_v2",
            assembled_data={
                "patient": {
                    "age": 65,
                    "weight": 75,
                    "gender": "male",
                    "conditions": ["hypertension", "diabetes"]
                },
                "labs": {
                    "creatinine": 1.2,
                    "alt": 30
                },
                "medications": {
                    "current": [
                        {"name": "lisinopril", "dose": "10mg"}
                    ]
                }
            },
            completeness_score=0.85,
            data_freshness={},
            source_metadata={},
            safety_flags=[],
            governance_tags=[],
            status="COMPLETE",
            assembled_at=datetime.now(),
            assembly_duration_ms=85.0,
            connection_errors=[]
        )
        
        # Test transformation
        medication_data = {"name": "warfarin", "is_anticoagulant": True}
        recipe_context = adapter.transform_context_for_recipes(
            context_data=sample_context,
            medication_data=medication_data,
            action_type="prescribe"
        )
        
        logger.info(f"✅ Context transformation successful")
        logger.info(f"   Patient age: {recipe_context.patient_data.get('age')}")
        logger.info(f"   Lab creatinine: {recipe_context.clinical_data.get('labs', {}).get('creatinine')}")
        logger.info(f"   Current medications: {len(recipe_context.clinical_data.get('current_medications', []))}")
        
        # Test validation
        validation_results = adapter.validate_transformed_data(recipe_context)
        logger.info(f"✅ Data validation completed: quality score {validation_results['data_quality_score']:.2%}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Context Data Adapter test failed: {str(e)}")
        return False


async def test_clinical_recipe_engine():
    """Test the Clinical Recipe Engine functionality"""
    try:
        from app.domain.services.clinical_recipe_engine import ClinicalRecipeEngine, RecipeContext
        
        logger.info("🧪 Testing Clinical Recipe Engine...")
        
        # Create recipe engine
        engine = ClinicalRecipeEngine()
        
        # Check registered recipes
        catalog = engine.get_recipe_catalog()
        logger.info(f"✅ Recipe engine loaded with {len(catalog)} recipes")
        
        # Create sample context
        context = RecipeContext(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            action_type="prescribe",
            medication_data={
                "name": "warfarin",
                "is_anticoagulant": True
            },
            patient_data={
                "age": 65,
                "conditions": ["atrial_fibrillation"]
            },
            provider_data={},
            encounter_data={},
            clinical_data={
                "labs": {"inr": 2.5},
                "current_medications": []
            },
            timestamp=datetime.now()
        )
        
        # Execute applicable recipes
        results = await engine.execute_applicable_recipes(context)
        logger.info(f"✅ Executed {len(results)} clinical recipes")
        
        for result in results:
            logger.info(f"   - {result.recipe_id}: {result.overall_status}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Clinical Recipe Engine test failed: {str(e)}")
        return False


async def test_flow2_integration():
    """Test the complete Flow 2 integration"""
    try:
        logger.info("🧪 Testing Complete Flow 2 Integration...")
        
        # Test all components
        orchestrator_ok = await test_recipe_orchestrator()
        adapter_ok = await test_context_data_adapter()
        engine_ok = await test_clinical_recipe_engine()
        
        if orchestrator_ok and adapter_ok and engine_ok:
            logger.info("🎉 All Flow 2 components are working correctly!")
            return True
        else:
            logger.error("❌ Some Flow 2 components failed")
            return False
        
    except Exception as e:
        logger.error(f"❌ Flow 2 integration test failed: {str(e)}")
        return False


async def main():
    """Main test function"""
    logger.info("🚀 Starting Flow 2 Integration Tests")
    logger.info("=" * 60)
    
    try:
        # Run the complete integration test
        success = await test_flow2_integration()
        
        logger.info("=" * 60)
        if success:
            logger.info("✅ Flow 2 Implementation Test: PASSED")
            logger.info("🎯 Flow 2 is ready for integration with Context Service!")
            logger.info("")
            logger.info("Next Steps:")
            logger.info("1. Start the Context Service on port 8016")
            logger.info("2. Test with real Context Service integration")
            logger.info("3. Run end-to-end API tests")
            return 0
        else:
            logger.error("❌ Flow 2 Implementation Test: FAILED")
            logger.error("🔧 Please check the error messages above and fix the issues")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1


if __name__ == "__main__":
    # Run the test
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
