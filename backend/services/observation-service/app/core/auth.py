from typing import Dict, Any, Optional
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import httpx
import logging
from app.core.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

security = HTTPBearer()

async def _validate_and_get_payload(token: str) -> Dict[str, Any]:
    """
    Validate the token with the Auth Service and return the payload.
    This is a helper function containing the core validation logic.
    """
    # Call the Auth Service to verify the token
    async with httpx.AsyncClient() as client:
        try:
            # Use the Supabase-specific endpoint
            response = await client.post(
                f"{settings.AUTH_SERVICE_URL}/auth/verify",
                headers={"Authorization": f"Bearer {token}"}
            )

            if response.status_code != 200:
                logger.error(f"Token validation failed: {response.status_code} - {response.text}")
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Invalid authentication credentials",
                    headers={"WWW-Authenticate": "Bearer"},
                )

            result = response.json()

            if not result.get("valid", False):
                logger.error(f"Token validation failed: {result.get('error', 'Unknown error')}")
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail=result.get("error", "Invalid token"),
                    headers={"WWW-Authenticate": "Bearer"},
                )

            # Get the user info from the response
            user_info = result.get("user", {})

            # Create a payload with the token and user info
            payload = {
                "token": token,
                "sub": user_info.get("sub") or user_info.get("id", "unknown"),
                "email": user_info.get("email", ""),
                "role": user_info.get("role", "user"),
                "permissions": user_info.get("permissions", ["read:observations", "write:observations"])
            }

            return payload

        except httpx.RequestError as e:
            logger.error(f"Error calling Auth Service: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                detail="Authentication service unavailable",
            )

async def get_token_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
    """
    FastAPI dependency to validate the token and return the payload.
    Uses the _validate_and_get_payload helper function.
    """
    token = credentials.credentials
    return await _validate_and_get_payload(token)
