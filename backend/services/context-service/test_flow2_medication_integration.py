"""
Test Flow 2 Integration: Context Service → Medication Service

This test validates that the Context Service can now call the Medication Service
to get clinical recipes during context assembly, implementing the correct Flow 2 architecture.
"""

import asyncio
import sys
import os
import logging

# Add the app directory to the Python path
sys.path.append(os.path.join(os.path.dirname(__file__), 'app'))

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def test_context_service_medication_integration():
    """
    Test that Context Service can call Medication Service for clinical recipes
    """
    try:
        logger.info("🧪 Testing Context Service → Medication Service Integration")
        logger.info("=" * 70)
        
        from app.services.context_assembly_service import ContextAssemblyService
        
        # Create context assembly service
        assembly_service = ContextAssemblyService()
        
        # Test 1: Get clinical recipes from Medication Service
        logger.info("📋 Test 1: Getting clinical recipes from Medication Service...")
        clinical_recipes = await assembly_service.get_clinical_recipes_from_medication_service()
        
        if clinical_recipes:
            logger.info(f"✅ Retrieved {len(clinical_recipes)} clinical recipes")
            
            # Show first few recipes
            for recipe in clinical_recipes[:3]:
                logger.info(f"   - {recipe.get('recipe_id', 'Unknown')}: {recipe.get('recipe_name', 'Unknown')}")
            
            if len(clinical_recipes) > 3:
                logger.info(f"   ... and {len(clinical_recipes) - 3} more recipes")
        else:
            logger.warning("⚠️ No clinical recipes retrieved - Medication Service may not be running")
            return False
        
        # Test 2: Analyze clinical recipe requirements
        logger.info("🧠 Test 2: Analyzing clinical recipe requirements...")
        
        sample_medication = {
            "name": "warfarin",
            "is_anticoagulant": True,
            "dose": "5mg"
        }
        
        requirements = await assembly_service.analyze_clinical_recipe_requirements(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            medication_data=sample_medication,
            clinical_recipes=clinical_recipes[:5]  # Test with first 5 recipes
        )
        
        logger.info("✅ Clinical Recipe Requirements Analysis:")
        logger.info(f"   Triggered Recipes: {len(requirements.get('triggered_recipes', []))}")
        
        for recipe_id in requirements.get('triggered_recipes', []):
            logger.info(f"     - {recipe_id}")
        
        logger.info(f"   Required Patient Data: {requirements.get('required_patient_data', [])}")
        logger.info(f"   Required Clinical Data: {requirements.get('required_clinical_data', [])}")
        logger.info(f"   Total Analyzed: {requirements.get('total_analyzed', 0)}")
        
        # Test 3: Verify Flow 2 integration is working
        logger.info("🎯 Test 3: Verifying Flow 2 integration...")
        
        if (len(clinical_recipes) > 0 and 
            len(requirements.get('triggered_recipes', [])) > 0 and
            len(requirements.get('required_patient_data', [])) > 0):
            
            logger.info("✅ Flow 2 Integration: WORKING")
            logger.info("   Context Service can successfully:")
            logger.info("   ✓ Call Medication Service for clinical recipes")
            logger.info("   ✓ Analyze recipe requirements")
            logger.info("   ✓ Determine data needs based on clinical logic")
            
            return True
        else:
            logger.warning("⚠️ Flow 2 Integration: PARTIAL")
            logger.warning("   Some components may not be fully functional")
            return False
        
    except Exception as e:
        logger.error(f"❌ Context Service → Medication Service integration test failed: {str(e)}")
        return False


async def test_graphql_available_recipes():
    """
    Test that the GraphQL getAvailableRecipes query now works
    """
    try:
        logger.info("🧪 Testing GraphQL getAvailableRecipes Query")
        
        from app.api.graphql.resolvers import context_resolver
        
        # Test the resolver directly
        recipes = await context_resolver.get_available_recipes()
        
        if recipes:
            logger.info(f"✅ GraphQL getAvailableRecipes returned {len(recipes)} recipes")
            
            for recipe in recipes[:3]:
                logger.info(f"   - {recipe.recipe_id}: {recipe.recipe_name}")
            
            return True
        else:
            logger.warning("⚠️ GraphQL getAvailableRecipes returned no recipes")
            return False
            
    except Exception as e:
        logger.error(f"❌ GraphQL getAvailableRecipes test failed: {str(e)}")
        return False


async def main():
    """Main test function"""
    logger.info("🚀 Starting Flow 2 Context Service Integration Test")
    logger.info("🎯 This validates that Context Service can call Medication Service")
    logger.info("")
    
    try:
        # Test Context Service → Medication Service integration
        integration_ok = await test_context_service_medication_integration()
        
        # Test GraphQL resolver
        graphql_ok = await test_graphql_available_recipes()
        
        logger.info("=" * 70)
        if integration_ok and graphql_ok:
            logger.info("🎉 Flow 2 Context Service Integration: PASSED")
            logger.info("✅ Context Service can now call Medication Service!")
            logger.info("")
            logger.info("🔄 Complete Flow 2 Architecture Now Working:")
            logger.info("1. Medication Service → Context Service (get context)")
            logger.info("2. Context Service → Medication Service (get clinical recipes) ✅ NEW!")
            logger.info("3. Context Service → Optimized context assembly")
            logger.info("4. Medication Service → Execute with real context")
            logger.info("")
            logger.info("🎯 Next Steps:")
            logger.info("1. Test complete end-to-end Flow 2 workflow")
            logger.info("2. Verify Context Service uses clinical recipe requirements")
            logger.info("3. Test with real medication safety scenarios")
            
            return 0
        else:
            logger.error("❌ Flow 2 Context Service Integration: FAILED")
            logger.error("🔧 Check that both services are running:")
            logger.error("   - Context Service on port 8016")
            logger.error("   - Medication Service on port 8009")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1


if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
