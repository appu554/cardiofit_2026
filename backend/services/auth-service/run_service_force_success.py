#!/usr/bin/env python
"""
Run the Auth Service with forced authentication success.
This script:
1. Sets the necessary environment variables
2. Forces authentication to always succeed (for testing)
3. Runs the service using uvicorn
"""

import os
import subprocess

# Set environment variables
os.environ["SUPABASE_URL"] = "https://auugxeqzgrnknklgwqrh.supabase.co"
os.environ["SUPABASE_KEY"] = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8"

# For testing purposes, we'll use a simple JWT secret
# In production, this should be a secure random string
# You can generate one with: python -c "import secrets; print(secrets.token_hex(32))"
os.environ["SUPABASE_JWT_SECRET"] = "test_jwt_secret_for_development_only_do_not_use_in_production"

# Set DEBUG mode
os.environ["DEBUG"] = "true"

# Set FORCE_AUTH_SUCCESS to force successful authentication in development
# This will make the Auth Service always return success, even for invalid tokens
os.environ["FORCE_AUTH_SUCCESS"] = "true"

print("Starting Auth Service with FORCED SUCCESS MODE:")
print(f"  SUPABASE_URL: {os.environ['SUPABASE_URL']}")
print(f"  SUPABASE_JWT_SECRET: {'*' * len(os.environ['SUPABASE_JWT_SECRET'])}")
print(f"  DEBUG: {os.environ['DEBUG']}")
print(f"  FORCE_AUTH_SUCCESS: {os.environ['FORCE_AUTH_SUCCESS']}")
print("")
print("WARNING: This mode will accept ANY token and return success!")
print("         This is for testing purposes only.")
print("         Use run_service.py for normal operation.")
print("")

# Run the service using uvicorn
cmd = ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8001", "--reload"]
subprocess.run(cmd)
