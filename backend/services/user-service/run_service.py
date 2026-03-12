#!/usr/bin/env python
# Add the backend directory to the Python path to make shared modules importable
import sys
import os
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Verify shared module is importable
try:
    from shared.auto_import import ensure_shared_importable
    ensure_shared_importable()
    print("[OK] Successfully imported shared module")
except ImportError as e:
    print(f"! Warning: Could not import shared module: {e}")
    print("! This might cause problems when importing HeaderAuthMiddleware")


"""
Run script for user-service Service.
This script adds the backend directory to the Python path and starts the service.
"""

import sys
import os
import subprocess

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, backend_dir)

# Set environment variables
os.environ["AUTH_SERVICE_URL"] = "http://localhost:8001/api"

# Print configuration
print("Starting user-service Service with the following configuration:")
print(f"  Python Path: {sys.path[0]}")
print(f"  AUTH_SERVICE_URL: {os.environ['AUTH_SERVICE_URL']}")
print("")

# Run the service using uvicorn
cmd = ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--reload"]
subprocess.run(cmd)
