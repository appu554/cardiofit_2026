#!/usr/bin/env python3
"""
Test script to verify governance validation for recipes
"""
import sys
import asyncio
from pathlib import Path

# Add the app directory to the path
sys.path.append('.')

from app.services.recipe_management_service import RecipeManagementService

async def test_governance():
    """Test governance validation for our new recipes"""
    try:
        print("🔍 Testing governance validation...")
        
        service = RecipeManagementService()
        
        # Test loading a specific recipe with governance validation
        recipe_id = "medication_safety_base_context_v2"
        
        print(f"\n🧪 Testing recipe: {recipe_id}")
        
        # Check if recipe is in loaded recipes
        if recipe_id in service.loaded_recipes:
            recipe = service.loaded_recipes[recipe_id]
            print(f"  ✅ Recipe found in loaded recipes")
            print(f"  Recipe ID: {recipe.recipe_id}")
            print(f"  Version: {recipe.version}")
            
            # Check governance metadata
            if recipe.governance_metadata:
                print(f"  ✅ Governance metadata present")
                print(f"  Approved by: {recipe.governance_metadata.approved_by}")
                print(f"  Approval date: {recipe.governance_metadata.approval_date}")
                print(f"  Effective date: {recipe.governance_metadata.effective_date}")
                print(f"  Expiry date: {recipe.governance_metadata.expiry_date}")
                print(f"  Is expired: {recipe.is_expired()}")
                print(f"  Validate governance: {recipe.validate_governance()}")
                
                # Test the service's governance validation
                governance_valid = await service._validate_governance_approval(recipe)
                print(f"  Service governance validation: {governance_valid}")
                
            else:
                print(f"  ❌ No governance metadata")
            
            # Try to load the recipe through the service (this includes governance validation)
            try:
                loaded_recipe = await service.load_recipe(recipe_id)
                print(f"  ✅ Recipe loaded successfully through service")
                print(f"  Loaded recipe ID: {loaded_recipe.recipe_id}")
            except Exception as e:
                print(f"  ❌ Failed to load recipe through service: {e}")
        else:
            print(f"  ❌ Recipe not found in loaded recipes")
            
    except Exception as e:
        print(f"❌ Error testing governance: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(test_governance())
