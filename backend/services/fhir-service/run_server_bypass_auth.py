#!/usr/bin/env python
"""
Run the FHIR service with authentication bypassed.
This script sets BYPASS_AUTH=true and runs the service.
"""

import os
import sys
import subprocess

# Add the root directory to the Python path
root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, root_dir)

# Set environment variables
os.environ["BYPASS_AUTH"] = "true"
os.environ["ENVIRONMENT"] = "development"

# Run the service using uvicorn
cmd = ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8004", "--reload"]
subprocess.run(cmd)
