#!/usr/bin/env python3
"""
Test script to verify that the new recipes from ContextRecipeBook.txt are loaded correctly
"""
import sys
import asyncio
from pathlib import Path

# Add the app directory to the path
sys.path.append('.')

from app.services.recipe_management_service import RecipeManagementService

async def test_recipes():
    """Test loading and validating recipes"""
    try:
        print("🔍 Initializing Recipe Management Service...")
        service = RecipeManagementService()
        
        print("\n📚 Loaded recipes:")
        for recipe_id, recipe in service.loaded_recipes.items():
            print(f"  ✅ {recipe_id} v{recipe.version} - {recipe.recipe_name}")
        
        print(f"\n📊 Total recipes loaded: {len(service.loaded_recipes)}")
        
        # Test loading specific new recipes from ContextRecipeBook.txt
        new_recipes = [
            "medication_safety_base_context_v2",
            "medication_renal_context_v2", 
            "cae_integration_context_v1",
            "safety_gateway_context_v1",
            "code_blue_context_v2",
            "workflow_engine_context_v1",
            "apollo_federation_context_v1"
        ]
        
        print("\n🧪 Testing new recipes from ContextRecipeBook.txt:")
        for recipe_id in new_recipes:
            try:
                recipe = await service.load_recipe(recipe_id)
                print(f"  ✅ {recipe.recipe_id} v{recipe.version}")
                print(f"     Clinical scenario: {recipe.clinical_scenario}")
                print(f"     Data points: {len(recipe.required_data_points)}")
                print(f"     SLA: {recipe.sla_ms}ms")
                print(f"     QoS Tier: {getattr(recipe, 'qos_tier', 'N/A')}")
                print()
            except Exception as e:
                print(f"  ❌ Error loading {recipe_id}: {e}")
        
        # Test recipe validation
        print("🔍 Testing recipe validation:")
        try:
            recipe = await service.load_recipe("medication_safety_base_context_v2")
            validation_result = await service.validate_recipe(recipe)
            print(f"  Validation result: {'✅ VALID' if validation_result['valid'] else '❌ INVALID'}")
            if validation_result['errors']:
                print(f"  Errors: {validation_result['errors']}")
            if validation_result['warnings']:
                print(f"  Warnings: {validation_result['warnings']}")
        except Exception as e:
            print(f"  ❌ Validation error: {e}")
            
    except Exception as e:
        print(f"❌ Error initializing service: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(test_recipes())
