from typing import Dict, Any
from fastapi import Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials

# Create a standard bearer security
security = HTTPBearer()

# Simple function to get token payload without authentication
async def get_token_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
    """
    Validate the token and return the payload.
    
    For development purposes, this function accepts any token.
    In production, you should properly verify the token with the Auth service.
    """
    token = credentials.credentials

    # For testing purposes, we'll just accept any token
    # In production, you should properly verify the token

    # Create a simple payload with the token
    payload = {
        "token": token,
        "sub": "test-user",
        "permissions": ["read:conditions", "write:conditions"],
        "auth_type": "bearer"
    }

    return payload
