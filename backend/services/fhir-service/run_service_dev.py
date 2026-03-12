#!/usr/bin/env python
"""
Run the FHIR service in development mode.
This script:
1. Adds the root directory to the Python path
2. Sets the necessary environment variables for development
3. Runs the service using uvicorn
"""

import os
import sys
import subprocess

# Add the root directory to the Python path
root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, root_dir)

# Also add the backend directory to the Python path (in case we're in backend/backend)
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if os.path.basename(backend_dir) == "backend":
    sys.path.insert(0, backend_dir)

# Print the Python path for debugging
print("Python path:")
for path in sys.path:
    print(f"  - {path}")

# Set environment variables
os.environ["AUTH_SERVICE_URL"] = "http://localhost:8001/api"
os.environ["ENVIRONMENT"] = "development"  # Development mode - accepts invalid tokens

print("Starting FHIR Service in DEVELOPMENT MODE with the following configuration:")
print(f"  Python Path: {sys.path[0]}")
print(f"  AUTH_SERVICE_URL: {os.environ['AUTH_SERVICE_URL']}")
print(f"  ENVIRONMENT: {os.environ['ENVIRONMENT']}")
print("")
print("WARNING: In development mode, authentication is not strictly enforced!")
print("         Invalid tokens will be accepted for convenience.")
print("         Use run_service.py for production mode with strict authentication.")
print("")

# Run the service using uvicorn
cmd = ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8004", "--reload"]
subprocess.run(cmd)
