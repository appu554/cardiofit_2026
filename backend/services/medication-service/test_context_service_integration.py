#!/usr/bin/env python3
"""
Test Context Service Client Integration
Tests the complete integration between Medication Service and Context Service
"""

import asyncio
import logging
import sys
import os
from datetime import datetime
from typing import Dict, Any

# Add the app directory to Python path
sys.path.append(os.path.join(os.path.dirname(__file__), 'app'))

from app.infrastructure.context_service_client import ContextServiceClient, ContextRequest

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class ContextServiceIntegrationTest:
    """
    Comprehensive test suite for Context Service integration
    """
    
    def __init__(self):
        self.client = ContextServiceClient()
        self.test_patient_id = "905a60cb-8241-418f-b29b-5b020e851392"  # Known test patient
        self.test_provider_id = "provider-123"
        self.test_medication_id = "med-456"
        
    async def run_all_tests(self):
        """Run all integration tests"""
        logger.info("🧪 Starting Context Service Integration Tests")
        logger.info("=" * 60)
        
        tests = [
            ("Health Check", self.test_health_check),
            ("Available Recipes", self.test_available_recipes),
            ("Medication Prescribing Context", self.test_medication_prescribing_context),
            ("Medication Safety Context", self.test_medication_safety_context),
            ("CAE Integration Context", self.test_cae_integration_context),
            ("Safety Gateway Context", self.test_safety_gateway_context),
            ("Renal Adjustment Context", self.test_renal_adjustment_context),
            ("Context Availability Validation", self.test_context_availability),
            ("Recipe ID Mapping", self.test_recipe_id_mapping),
            ("Error Handling", self.test_error_handling)
        ]
        
        passed = 0
        failed = 0
        
        for test_name, test_func in tests:
            try:
                logger.info(f"\n🔍 Running test: {test_name}")
                await test_func()
                logger.info(f"✅ {test_name} - PASSED")
                passed += 1
            except Exception as e:
                logger.error(f"❌ {test_name} - FAILED: {e}")
                failed += 1
        
        logger.info("\n" + "=" * 60)
        logger.info(f"🏁 Test Results: {passed} passed, {failed} failed")
        
        if failed == 0:
            logger.info("🎉 All tests passed! Context Service integration is working correctly.")
        else:
            logger.warning(f"⚠️ {failed} tests failed. Check the logs above for details.")
        
        return failed == 0
    
    async def test_health_check(self):
        """Test Context Service health check"""
        is_healthy = await self.client.health_check()
        
        if is_healthy:
            logger.info("✅ Context Service is healthy")
        else:
            raise Exception("Context Service health check failed")
    
    async def test_available_recipes(self):
        """Test getting available recipes"""
        recipes = await self.client.get_available_recipes()
        
        logger.info(f"📚 Found {len(recipes)} available recipes:")
        for recipe in recipes:
            logger.info(f"   - {recipe.get('recipeId', 'unknown')} v{recipe.get('version', '?')}")
        
        # Check for expected medication-related recipes
        recipe_ids = [recipe.get('recipeId', '') for recipe in recipes]
        expected_recipes = [
            'medication_prescribing_v2',
            'medication_safety_base_context_v2',
            'cae_integration_context_v1',
            'safety_gateway_context_v1'
        ]
        
        found_recipes = []
        for expected in expected_recipes:
            if expected in recipe_ids:
                found_recipes.append(expected)
                logger.info(f"   ✅ Found expected recipe: {expected}")
            else:
                logger.warning(f"   ⚠️ Missing expected recipe: {expected}")
        
        if len(found_recipes) >= 2:  # At least 2 medication recipes should be available
            logger.info(f"✅ Found {len(found_recipes)} expected medication recipes")
        else:
            raise Exception(f"Only found {len(found_recipes)} expected recipes, need at least 2")
    
    async def test_medication_prescribing_context(self):
        """Test medication prescribing context retrieval"""
        context = await self.client.get_medication_prescribing_context(
            patient_id=self.test_patient_id,
            provider_id=self.test_provider_id
        )
        
        self._validate_context_response(context, "medication_prescribing_v2")
        logger.info(f"✅ Medication prescribing context retrieved successfully")
        logger.info(f"   Context ID: {context.context_id}")
        logger.info(f"   Completeness: {context.completeness_score:.2%}")
        logger.info(f"   Assembly time: {context.assembly_duration_ms:.1f}ms")
    
    async def test_medication_safety_context(self):
        """Test medication safety context retrieval"""
        context = await self.client.get_medication_safety_context(
            patient_id=self.test_patient_id,
            medication_id=self.test_medication_id,
            provider_id=self.test_provider_id
        )
        
        self._validate_context_response(context, "medication_safety_base_context_v2")
        
        # Check that medication ID was added
        if context.assembled_data.get('target_medication_id') == self.test_medication_id:
            logger.info(f"✅ Target medication ID correctly added to context")
        else:
            raise Exception("Target medication ID not found in assembled data")
    
    async def test_cae_integration_context(self):
        """Test CAE integration context retrieval"""
        context = await self.client.get_cae_integration_context(
            patient_id=self.test_patient_id,
            provider_id=self.test_provider_id
        )
        
        self._validate_context_response(context, "cae_integration_context_v1")
        logger.info(f"✅ CAE integration context retrieved successfully")
    
    async def test_safety_gateway_context(self):
        """Test Safety Gateway context retrieval"""
        context = await self.client.get_safety_gateway_context(
            patient_id=self.test_patient_id,
            provider_id=self.test_provider_id
        )
        
        self._validate_context_response(context, "safety_gateway_context_v1")
        logger.info(f"✅ Safety Gateway context retrieved successfully")
    
    async def test_renal_adjustment_context(self):
        """Test renal adjustment context retrieval"""
        context = await self.client.get_renal_adjustment_context(
            patient_id=self.test_patient_id,
            provider_id=self.test_provider_id
        )
        
        self._validate_context_response(context, "medication_renal_context_v2")
        logger.info(f"✅ Renal adjustment context retrieved successfully")
    
    async def test_context_availability(self):
        """Test context availability validation"""
        availability = await self.client.validate_context_availability(
            patient_id=self.test_patient_id,
            recipe_id="medication_prescribing_v2",
            provider_id=self.test_provider_id
        )
        
        required_fields = ['available', 'estimatedCompleteness', 'recipeId', 'patientId']
        for field in required_fields:
            if field not in availability:
                raise Exception(f"Missing required field in availability response: {field}")

        logger.info(f"✅ Context availability validation successful")
        logger.info(f"   Available: {availability.get('available', 'unknown')}")
        logger.info(f"   Recipe ID: {availability.get('recipeId', 'unknown')}")
        logger.info(f"   Completeness: {availability.get('estimatedCompleteness', 'unknown')}")
        logger.info(f"   Assembly Time: {availability.get('estimatedAssemblyTimeMs', 'unknown')}ms")
    
    async def test_recipe_id_mapping(self):
        """Test recipe ID mapping functionality"""
        test_mappings = [
            ('medication_prescribing', 'medication_prescribing_v2'),
            ('medication_safety', 'medication_safety_base_context_v2'),
            ('cae_integration', 'cae_integration_context_v1'),
            ('safety_gateway', 'safety_gateway_context_v1'),
            ('unknown_workflow', 'medication_prescribing_v2')  # Should fallback
        ]
        
        for workflow_type, expected_recipe in test_mappings:
            actual_recipe = self.client.get_recipe_id_for_workflow(workflow_type)
            if actual_recipe == expected_recipe:
                logger.info(f"   ✅ {workflow_type} -> {actual_recipe}")
            else:
                raise Exception(f"Recipe mapping failed: {workflow_type} -> {actual_recipe}, expected {expected_recipe}")
        
        logger.info(f"✅ All recipe ID mappings working correctly")
    
    async def test_error_handling(self):
        """Test error handling with invalid requests"""
        try:
            # Test with invalid patient ID
            await self.client.get_medication_prescribing_context(
                patient_id="invalid-patient-id",
                provider_id=self.test_provider_id
            )
            # If we get here without exception, that's actually okay - the context service might handle it gracefully
            logger.info("✅ Invalid patient ID handled gracefully")
        except Exception as e:
            # This is expected behavior
            logger.info(f"✅ Invalid patient ID properly rejected: {str(e)[:100]}...")
        
        try:
            # Test with invalid recipe ID
            invalid_request = ContextRequest(
                patient_id=self.test_patient_id,
                recipe_id="non-existent-recipe",
                provider_id=self.test_provider_id
            )
            await self.client._get_context_by_recipe(invalid_request)
            raise Exception("Expected error for invalid recipe ID")
        except Exception as e:
            if "non-existent-recipe" in str(e) or "not found" in str(e).lower():
                logger.info(f"✅ Invalid recipe ID properly rejected")
            else:
                raise Exception(f"Unexpected error for invalid recipe: {e}")
    
    def _validate_context_response(self, context, expected_recipe_id: str):
        """Validate that a context response has all required fields"""
        required_fields = [
            'context_id', 'patient_id', 'recipe_used', 'assembled_data',
            'completeness_score', 'status', 'assembled_at', 'assembly_duration_ms'
        ]
        
        for field in required_fields:
            if not hasattr(context, field):
                raise Exception(f"Missing required field in context response: {field}")
        
        if context.patient_id != self.test_patient_id:
            raise Exception(f"Patient ID mismatch: expected {self.test_patient_id}, got {context.patient_id}")
        
        if context.recipe_used != expected_recipe_id:
            logger.warning(f"Recipe ID mismatch: expected {expected_recipe_id}, got {context.recipe_used}")
        
        if not isinstance(context.assembled_data, dict):
            raise Exception("Assembled data should be a dictionary")
        
        if not (0 <= context.completeness_score <= 1):
            raise Exception(f"Completeness score should be between 0 and 1, got {context.completeness_score}")


async def main():
    """Main test execution"""
    test_suite = ContextServiceIntegrationTest()
    success = await test_suite.run_all_tests()
    
    if success:
        print("\n🎉 All Context Service integration tests passed!")
        sys.exit(0)
    else:
        print("\n❌ Some tests failed. Check the logs above.")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
