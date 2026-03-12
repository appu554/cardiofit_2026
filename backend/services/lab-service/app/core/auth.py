from typing import Dict, Any
from fastapi import Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials

# Create a standard bearer security
security = HTTPBearer()

async def get_token_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
    """
    Extract the token payload from the authorization header.
    This is a simplified version that just returns the token for forwarding.
    """
    try:
        # Extract the token from the authorization header
        token = credentials.credentials

        # Return a dictionary with the token and comprehensive permissions
        return {
            "token": token,
            "user_id": "test-user-id",
            "email": "doctor@example.com",
            "role": "doctor",
            "roles": ["doctor"],
            "permissions": [
                "patient:read",
                "patient:write",
                "observation:read",
                "observation:write",
                "condition:read",
                "condition:write",
                "medication:read",
                "medication:write",
                "encounter:read",
                "encounter:write",
                "lab:read",
                "lab:write",
                "diagnosticreport:read",
                "diagnosticreport:write"
            ]
        }
    except Exception as e:
        # Log the error but continue with a default payload
        print(f"Error extracting token payload: {str(e)}")
        return {
            "token": "default-token",
            "user_id": "default-user",
            "permissions": ["lab:read", "lab:write", "diagnosticreport:read", "diagnosticreport:write"]
        }
