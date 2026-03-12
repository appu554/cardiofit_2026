#!/usr/bin/env python
"""
Run the Auth Service with Supabase JWT validation.
This script:
1. Sets the necessary environment variables for Supabase
2. Configures the JWT validation to accept Supabase tokens
3. Runs the service using uvicorn
"""

import os
import subprocess

# Set environment variables
os.environ["SUPABASE_URL"] = "https://auugxeqzgrnknklgwqrh.supabase.co"
os.environ["SUPABASE_KEY"] = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8"

# Set your Supabase JWT Secret here
# You can find this in your Supabase dashboard: Project Settings > API > JWT Settings
os.environ["SUPABASE_JWT_SECRET"] = "nXwqv86rPXO5HqJ1R1xeQnhy9JbeLeLypwUZmMoJ1prMGG6io5lU88nD6lG8MmvpN7Z2pZJvfuF33Z1x2PwCoA=="

# Configure JWT validation options
# These settings make the validation more permissive to accept Supabase tokens
os.environ["SUPABASE_JWT_AUDIENCE"] = "authenticated"  # Accept "authenticated" as the audience
os.environ["SUPABASE_JWT_ISSUER"] = "https://auugxeqzgrnknklgwqrh.supabase.co/auth/v1"  # Accept Supabase as the issuer
os.environ["SUPABASE_JWT_VERIFY_AUDIENCE"] = "false"  # Skip audience verification
os.environ["SUPABASE_JWT_VERIFY_ISSUER"] = "false"  # Skip issuer verification

# Set DEBUG mode
os.environ["DEBUG"] = "true"

# Set FORCE_AUTH_SUCCESS to force successful authentication in development
# Set to "true" to always return success, "false" to enforce proper validation
os.environ["FORCE_AUTH_SUCCESS"] = "false"

print("\nStarting Auth Service with the following configuration:")
print(f"  SUPABASE_URL: {os.environ['SUPABASE_URL']}")
print(f"  SUPABASE_JWT_SECRET: {'*' * len(os.environ['SUPABASE_JWT_SECRET'])}")
print(f"  SUPABASE_JWT_AUDIENCE: {os.environ['SUPABASE_JWT_AUDIENCE']}")
print(f"  SUPABASE_JWT_VERIFY_AUDIENCE: {os.environ['SUPABASE_JWT_VERIFY_AUDIENCE']}")
print(f"  SUPABASE_JWT_VERIFY_ISSUER: {os.environ['SUPABASE_JWT_VERIFY_ISSUER']}")
print(f"  DEBUG: {os.environ['DEBUG']}")
print(f"  FORCE_AUTH_SUCCESS: {os.environ['FORCE_AUTH_SUCCESS']}")
print("")

if os.environ["SUPABASE_JWT_SECRET"] == "your-supabase-jwt-secret":
    print("IMPORTANT: You need to replace 'your-supabase-jwt-secret' with your actual Supabase JWT secret!")
    print("           You can find this in your Supabase dashboard: Project Settings > API > JWT Settings")
    print("           Without the correct JWT secret, tokens from Supabase will NOT be validated correctly.")
    print("")
    
    # Ask if the user wants to continue
    response = input("Do you want to continue anyway? (y/n): ")
    if response.lower() != "y":
        print("Exiting...")
        exit(1)

# Run the service using uvicorn
cmd = ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8001", "--reload"]
subprocess.run(cmd)
