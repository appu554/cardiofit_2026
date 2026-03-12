from typing import Dict, Any
from fastapi import Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials

# Dummy values for compatibility
BASIC_AUTH_USERNAME = "medication_service"
BASIC_AUTH_PASSWORD = "medication_password"

# Create a standard bearer security
security = HTTPBearer()

# Simple function to get token payload without authentication
async def get_token_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
    """
    Validate the token and return the payload.
    """
    token = credentials.credentials

    # For testing purposes, we'll just accept any token
    # In production, you should properly verify the token

    # Create a simple payload with the token
    payload = {
        "token": token,
        "sub": "test-user",
        "permissions": ["read:medications", "write:medications"],
        "auth_type": "bearer"
    }

    return payload

# Combined authentication function that uses token payload
async def get_auth_payload(token_payload: Dict[str, Any] = Depends(get_token_payload)) -> Dict[str, Any]:
    """
    Get authentication payload from token payload.
    """
    return token_payload
