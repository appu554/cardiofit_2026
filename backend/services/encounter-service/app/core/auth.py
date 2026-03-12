from typing import Dict, Any, Optional
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import httpx
import logging
from app.core.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Set up security scheme
security = HTTPBearer()

async def get_token_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
    """
    Get the token payload from the authorization header.
    
    For development, this function returns a dummy payload with the token.
    In production, it would validate the token with Auth0 or another auth provider.
    """
    try:
        # Extract the token from the authorization header
        token = credentials.credentials
        
        # For development, just return a dummy payload with the token
        # In production, validate the token with Auth0 or another auth provider
        return {"token": token, "sub": "user123", "permissions": ["read:encounters", "write:encounters"]}
    except Exception as e:
        logger.error(f"Error getting token payload: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid authentication credentials",
            headers={"WWW-Authenticate": "Bearer"},
        )

async def get_auth_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
    """Alias for get_token_payload for compatibility with other services."""
    return await get_token_payload(credentials)
