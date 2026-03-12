#!/usr/bin/env python
"""
Run script for observation-service Service.
This script adds the backend directory to the Python path and starts the service.
"""

import sys
import os
import subprocess
import site

# Add the necessary directories to the Python path before any other imports
# Current: observation-service -> services -> backend
current_dir = os.path.dirname(__file__)
services_dir = os.path.abspath(os.path.join(current_dir, ".."))
backend_dir = os.path.abspath(os.path.join(services_dir, ".."))
shared_dir = os.path.join(backend_dir, 'shared')  # shared is in backend/shared

# Add to Python path at the very beginning
sys.path.insert(0, services_dir)  # Add services directory to path
sys.path.insert(0, backend_dir)   # Add backend directory to path

# Add the shared directory to site-packages
site.addsitedir(shared_dir)

# Print directory information for debugging
print(f"Current directory: {current_dir}")
print(f"Backend directory: {backend_dir}")
print(f"Services directory: {services_dir}")
print(f"Shared directory: {shared_dir}")
print(f"Checking if shared module exists: {os.path.exists(shared_dir)}")

# Print directory structure
print("\nDirectory structure:")
print(f"  {backend_dir}:")
for item in os.listdir(backend_dir):
    print(f"    {item}")

if os.path.exists(shared_dir):
    print(f"\n  {shared_dir}:")
    for item in os.listdir(shared_dir):
        print(f"    {item}")
        
        # Print contents of google_healthcare if it exists
        if item == 'google_healthcare':
            google_healthcare_dir = os.path.join(shared_dir, item)
            print(f"      {google_healthcare_dir}:")
            if os.path.exists(google_healthcare_dir):
                for subitem in os.listdir(google_healthcare_dir):
                    print(f"        {subitem}")

# Set environment variables
os.environ["AUTH_SERVICE_URL"] = "http://localhost:8001/api"
os.environ["PYTHONPATH"] = os.pathsep.join([backend_dir, shared_dir] + sys.path)

# Set Google Cloud Healthcare API environment variables
os.environ["USE_GOOGLE_HEALTHCARE_API"] = "true"
os.environ["GOOGLE_CLOUD_PROJECT_ID"] = "cardiofit-905a8"
os.environ["GOOGLE_CLOUD_LOCATION"] = "asia-south1"
os.environ["GOOGLE_CLOUD_DATASET_ID"] = "clinical-synthesis-hub"
os.environ["GOOGLE_CLOUD_FHIR_STORE_ID"] = "fhir-store"
os.environ["GOOGLE_CLOUD_CREDENTIALS_PATH"] = "credentials/google-credentials.json"
os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = "credentials/google-credentials.json"

# Print Python path for debugging
print("\nCurrent Python path:")
for path in sys.path:
    print(f"  {path}")

# Verify shared module is importable
try:
    # Try to import the google_healthcare module directly
    try:
        from shared.google_healthcare.client import GoogleHealthcareClient
        print("✓ Successfully imported GoogleHealthcareClient from shared.google_healthcare.client")
    except ImportError as e:
        print(f"! Error importing GoogleHealthcareClient: {e}")
        # Print the contents of the google_healthcare directory if it exists
        google_healthcare_dir = os.path.join(shared_dir, 'google_healthcare')
        if os.path.exists(google_healthcare_dir):
            print(f"Contents of {google_healthcare_dir}:")
            for item in os.listdir(google_healthcare_dir):
                print(f"  {item}")
    
    # Try to import the auto_import utility
    try:
        from shared.auto_import import ensure_shared_importable
        ensure_shared_importable()
        print("✓ Successfully imported shared.auto_import")
    except ImportError as e:
        print(f"! Warning: Could not import shared.auto_import: {e}")
    
    # Verify HeaderAuthMiddleware is importable
    try:
        from shared.auth import HeaderAuthMiddleware
        print("✓ Successfully imported HeaderAuthMiddleware")
    except ImportError as e:
        print(f"! Warning: Could not import HeaderAuthMiddleware: {e}")
        
except ImportError as e:
    print(f"! Error: Could not import shared module: {e}")
    print(f"! Make sure the shared module exists at: {shared_dir}")
    print("! Current Python path:", sys.path)
    sys.exit(1)

# Print configuration
print("Starting observation-service Service with the following configuration:")
print(f"  Python Path: {sys.path[0]}")
print(f"  PYTHONPATH: {os.environ['PYTHONPATH']}")
print(f"  AUTH_SERVICE_URL: {os.environ['AUTH_SERVICE_URL']}")
print("")

# Run the service using uvicorn with the current Python executable
cmd = [sys.executable, "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8007", "--reload"]
subprocess.run(cmd, env=os.environ)  # Pass the environment variables to the subprocess
