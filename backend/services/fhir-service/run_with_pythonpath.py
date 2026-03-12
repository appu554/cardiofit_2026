#!/usr/bin/env python
"""
Special run script for fhir-service that explicitly sets the PYTHONPATH.
This script is a workaround for the "No module named 'shared.auth'" error.
"""

import os
import sys
import subprocess

# Get the absolute path to the backend directory
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))

# Set the PYTHONPATH environment variable to include the backend directory
os.environ["PYTHONPATH"] = backend_dir

# Print configuration
print("Starting fhir-service with explicit PYTHONPATH:")
print(f"  PYTHONPATH: {os.environ['PYTHONPATH']}")
print(f"  Running on port: 8014 (changed from default 8004)")
print("")

# Run the service using uvicorn with the explicit PYTHONPATH
cmd = ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8014", "--reload"]

# Pass the environment variables to the subprocess
subprocess.run(cmd, env=os.environ)
