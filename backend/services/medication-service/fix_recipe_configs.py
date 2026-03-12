#!/usr/bin/env python3
"""
Quick fix script to add missing 'description' fields to recipe configs
"""

import re

def fix_recipe_configs():
    """Add missing description fields to all recipe configs"""
    
    file_path = "app/domain/services/clinical_recipes_complete.py"
    
    # Read the file
    with open(file_path, 'r') as f:
        content = f.read()
    
    # Pattern to find config dictionaries missing description
    pattern = r"config = \{'id': '([^']+)', 'name': '([^']+)', 'priority': (\d+), 'qosTier': '([^']+)'\}"
    
    def replace_config(match):
        recipe_id = match.group(1)
        name = match.group(2)
        priority = match.group(3)
        qos_tier = match.group(4)
        
        # Generate description from name
        description = name.lower()
        
        return f"""config = {{
            'id': '{recipe_id}',
            'name': '{name}',
            'description': '{description}',
            'priority': {priority},
            'qosTier': '{qos_tier}'
        }}"""
    
    # Replace all matching patterns
    fixed_content = re.sub(pattern, replace_config, content)
    
    # Write back to file
    with open(file_path, 'w') as f:
        f.write(fixed_content)
    
    print("✅ Fixed all recipe configs with missing description fields")

if __name__ == "__main__":
    fix_recipe_configs()
