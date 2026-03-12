from typing import Dict, Any
from fastapi import Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

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
                "timeline:read",
                "timeline:write"
            ]
        }
    except Exception as e:
        # Log the error but continue with a default payload
        logger.error(f"Error extracting token payload: {str(e)}")
        return {
            "token": "default-token",
            "user_id": "default-user",
            "permissions": ["timeline:read", "timeline:write"]
        }
