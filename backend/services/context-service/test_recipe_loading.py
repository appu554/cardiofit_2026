#!/usr/bin/env python3
"""
Test script to verify recipe loading after adding new DataSourceType values
"""
import sys
import asyncio
from pathlib import Path

# Add the app directory to the path
sys.path.append('.')

from app.services.recipe_management_service import RecipeManagementService

def test_recipe_loading():
    """Test loading recipes with new data source types"""
    try:
        print("🔍 Testing recipe loading with new DataSourceType values...")
        
        # Initialize the service (this will load all recipes)
        service = RecipeManagementService()
        
        print(f"\n📚 Total recipes loaded: {len(service.loaded_recipes)}")
        
        # List all loaded recipes
        for recipe_id, recipe in service.loaded_recipes.items():
            print(f"  ✅ {recipe_id} v{recipe.version} - {recipe.recipe_name}")
            print(f"     Clinical scenario: {recipe.clinical_scenario}")
            print(f"     Data points: {len(recipe.required_data_points)}")
            print(f"     SLA: {getattr(recipe, 'sla_ms', 'N/A')}ms")
            print()
        
        # Check for our new recipes specifically
        new_recipes = [
            "medication_safety_base_context_v2",
            "medication_renal_context_v2", 
            "cae_integration_context_v1",
            "safety_gateway_context_v1",
            "code_blue_context_v2",
            "workflow_engine_context_v1",
            "apollo_federation_context_v1"
        ]
        
        print("🧪 Checking for new recipes from ContextRecipeBook.txt:")
        found_count = 0
        for recipe_id in new_recipes:
            if recipe_id in service.loaded_recipes:
                recipe = service.loaded_recipes[recipe_id]
                print(f"  ✅ {recipe_id} v{recipe.version} - LOADED")
                found_count += 1
            else:
                print(f"  ❌ {recipe_id} - NOT FOUND")
        
        print(f"\n📊 Summary: {found_count}/{len(new_recipes)} new recipes loaded successfully")
        
        if found_count == len(new_recipes):
            print("🎉 All new recipes from ContextRecipeBook.txt are loaded!")
        else:
            print("⚠️ Some recipes are missing. Check for parsing errors.")
            
    except Exception as e:
        print(f"❌ Error testing recipe loading: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    test_recipe_loading()
