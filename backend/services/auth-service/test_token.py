#!/usr/bin/env python
"""
Generate a test Supabase token for testing.
This script generates a test token that can be used to test the Auth Service.
"""

import jwt
import time
import os
import json

# Supabase JWT secret - must match the one in run_service.py
JWT_SECRET = "test_jwt_secret_for_development_only_do_not_use_in_production"

# Create a payload for the token
payload = {
    "iss": "supabase",  # Issuer - must be "supabase" for Supabase tokens
    "sub": "test-user-id",  # Subject - user ID
    "aud": "authenticated",  # Audience - "authenticated" for authenticated users
    "exp": int(time.time()) + 3600,  # Expiration time - 1 hour from now
    "iat": int(time.time()),  # Issued at time - now
    "email": "test@example.com",  # User email
    "phone": "",  # User phone
    "app_metadata": {
        "provider": "email",
        "providers": ["email"]
    },
    "user_metadata": {
        "email_verified": True
    },
    "role": "authenticated",  # User role
    "aal": "aal1",  # Authentication Assurance Level
    "amr": [
        {
            "method": "password",
            "timestamp": int(time.time())
        }
    ],
    "session_id": "test-session-id",
    "is_anonymous": False
}

# Encode the token
token = jwt.encode(payload, JWT_SECRET, algorithm="HS256")

# Print the token
print("\n=== TEST SUPABASE TOKEN ===")
print(token)
print("===========================\n")

# Print the decoded token for verification
print("Decoded token payload:")
print(json.dumps(payload, indent=2))
print()

# Print instructions
print("To test this token with the Auth Service:")
print("1. Copy the token above")
print("2. Use it in a request to the Auth Service:")
print("   curl -X POST http://localhost:8001/api/auth/verify -H \"Authorization: Bearer YOUR_TOKEN\"")
print()
print("To test this token with the FHIR Service:")
print("1. Copy the token above")
print("2. Use it in a request to the FHIR Service:")
print("   curl -X GET http://localhost:8004/api/fhir/Patient -H \"Authorization: Bearer YOUR_TOKEN\"")
print()
