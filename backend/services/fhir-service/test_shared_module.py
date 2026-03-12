#!/usr/bin/env python
"""
Test the shared module.
This script tests if the shared module can be imported.
"""

import os
import sys

# Add the root directory to the Python path
root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, root_dir)

# Also add the backend directory to the Python path (in case we're in backend/backend)
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if os.path.basename(backend_dir) == "backend":
    sys.path.insert(0, backend_dir)

# Print the Python path
print("Python path:")
for path in sys.path:
    print(f"  - {path}")

# Try to import the shared module
try:
    import shared
    print("\n✅ Successfully imported shared module")
    
    import shared.auth
    print("✅ Successfully imported shared.auth module")
    
    from shared.auth.middleware import AuthenticationMiddleware
    print("✅ Successfully imported AuthenticationMiddleware")
    
    print("\nShared module is working correctly!")
except ImportError as e:
    print(f"\n❌ Failed to import shared module: {str(e)}")
    print("\nTry running this script from the root directory:")
    print("  cd backend")
    print("  python services/fhir-service/test_shared_module.py")
    
    # Print the directory structure
    print("\nDirectory structure:")
    for root, dirs, files in os.walk(root_dir):
        level = root.replace(root_dir, '').count(os.sep)
        indent = ' ' * 4 * level
        print(f"{indent}{os.path.basename(root)}/")
        sub_indent = ' ' * 4 * (level + 1)
        for file in files:
            if file.endswith('.py'):
                print(f"{sub_indent}{file}")
