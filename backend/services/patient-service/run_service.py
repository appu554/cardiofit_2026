#!/usr/bin/env python
"""
Run script for patient-service Service.
This script adds the backend directory to the Python path and starts the service.
"""

import sys
import os
import subprocess

# Add the backend directory to the Python path
# Need to go up two levels: patient-service -> services -> backend
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, backend_dir)

# Print the backend directory for debugging
print(f"Backend directory: {backend_dir}")
print(f"Checking if shared module exists: {os.path.exists(os.path.join(backend_dir, 'shared'))}")
if os.path.exists(os.path.join(backend_dir, 'shared')):
    print(f"Contents of shared directory:")
    for item in os.listdir(os.path.join(backend_dir, 'shared')):
        print(f"  {item}")

# Set environment variables
os.environ["AUTH_SERVICE_URL"] = "http://localhost:8001/api"
os.environ["PYTHONPATH"] = backend_dir  # Set PYTHONPATH to include the backend directory

# Set Google Cloud Healthcare API environment variables
os.environ["USE_GOOGLE_HEALTHCARE_API"] = "true"
os.environ["GOOGLE_CLOUD_PROJECT_ID"] = "cardiofit-905a8"
os.environ["GOOGLE_CLOUD_LOCATION"] = "asia-south1"
os.environ["GOOGLE_CLOUD_DATASET_ID"] = "clinical-synthesis-hub"
os.environ["GOOGLE_CLOUD_FHIR_STORE_ID"] = "fhir-store"
os.environ["GOOGLE_CLOUD_CREDENTIALS_PATH"] = "credentials/google-credentials.json"
os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = "credentials/google-credentials.json"

# Verify shared module is importable
try:
    from shared.auto_import import ensure_shared_importable
    ensure_shared_importable()
    print("[OK] Successfully imported shared module")

    # Verify HeaderAuthMiddleware is importable
    from shared.auth import HeaderAuthMiddleware
    print("✓ Successfully imported HeaderAuthMiddleware")
except ImportError as e:
    print(f"! Warning: Could not import shared module: {e}")
    print("! This might cause problems when importing HeaderAuthMiddleware")
    print("! Make sure the shared module exists at:", os.path.join(backend_dir, "shared"))
    print("! Current Python path:", sys.path)

# Print configuration
print("Starting patient-service Service with the following configuration:")
print(f"  Python Path: {sys.path[0]}")
print(f"  PYTHONPATH: {os.environ['PYTHONPATH']}")
print(f"  AUTH_SERVICE_URL: {os.environ['AUTH_SERVICE_URL']}")
print(f"  USE_GOOGLE_HEALTHCARE_API: {os.environ['USE_GOOGLE_HEALTHCARE_API']}")
print(f"  GOOGLE_CLOUD_PROJECT_ID: {os.environ['GOOGLE_CLOUD_PROJECT_ID']}")
print(f"  GOOGLE_CLOUD_LOCATION: {os.environ['GOOGLE_CLOUD_LOCATION']}")
print(f"  GOOGLE_CLOUD_DATASET_ID: {os.environ['GOOGLE_CLOUD_DATASET_ID']}")
print(f"  GOOGLE_CLOUD_FHIR_STORE_ID: {os.environ['GOOGLE_CLOUD_FHIR_STORE_ID']}")
print(f"  GOOGLE_CLOUD_CREDENTIALS_PATH: {os.environ['GOOGLE_CLOUD_CREDENTIALS_PATH']}")
print(f"  GOOGLE_APPLICATION_CREDENTIALS: {os.environ['GOOGLE_APPLICATION_CREDENTIALS']}")
print("")

# Run the service using uvicorn with the same Python executable
# Use sys.executable to ensure we use the same Python that's running this script
cmd = [sys.executable, "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8003", "--reload"]
print(f"Running command: {' '.join(cmd)}")
subprocess.run(cmd, env=os.environ)  # Pass the environment variables to the subprocess
