#!/usr/bin/env python3
"""
Script to fix the Postman collection by removing 'display' fields from ReferenceInput objects
since the supergraph schema doesn't include the display field in ReferenceInput.
"""

import json
import re

def fix_postman_collection():
    # Read the Postman collection
    with open('postman/Comprehensive-Order-Management-GraphQL.postman_collection.json', 'r', encoding='utf-8') as f:
        collection = json.load(f)
    
    # Function to recursively remove display fields from reference objects
    def remove_display_from_references(obj):
        if isinstance(obj, dict):
            # Check if this looks like a reference object with display field
            if 'reference' in obj and 'display' in obj:
                # Remove the display field
                obj_copy = obj.copy()
                del obj_copy['display']
                return obj_copy
            else:
                # Recursively process all values
                return {k: remove_display_from_references(v) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [remove_display_from_references(item) for item in obj]
        else:
            return obj
    
    # Process each item in the collection
    def process_item(item):
        if 'request' in item and 'body' in item['request'] and 'raw' in item['request']['body']:
            try:
                # Parse the JSON from the raw field
                raw_data = json.loads(item['request']['body']['raw'])
                
                # Remove display fields from variables
                if 'variables' in raw_data:
                    raw_data['variables'] = remove_display_from_references(raw_data['variables'])
                
                # Convert back to JSON string
                item['request']['body']['raw'] = json.dumps(raw_data, indent=2)
                
                print(f"Fixed request: {item.get('name', 'Unknown')}")
                
            except json.JSONDecodeError:
                print(f"Skipping non-JSON request: {item.get('name', 'Unknown')}")
        
        # Process nested items
        if 'item' in item:
            for nested_item in item['item']:
                process_item(nested_item)
    
    # Process all items in the collection
    for item in collection.get('item', []):
        process_item(item)
    
    # Write the fixed collection back
    with open('postman/Comprehensive-Order-Management-GraphQL.postman_collection.json', 'w', encoding='utf-8') as f:
        json.dump(collection, f, indent=2)
    
    print("✅ Postman collection fixed! All 'display' fields removed from ReferenceInput objects.")

if __name__ == "__main__":
    fix_postman_collection()
