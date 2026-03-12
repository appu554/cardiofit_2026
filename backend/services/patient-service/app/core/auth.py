from fastapi import Depends, HTTPException, status, Request
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import logging
from typing import Dict, Any, Optional

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Create security scheme
security = HTTPBearer()

async def get_token_payload(
    credentials: HTTPAuthorizationCredentials = Depends(security),
    request: Request = None
) -> Dict[str, Any]:
    """
    Extract the token payload from the authorization header and request headers.
    This version uses the X-User-* headers set by the API Gateway if available.
    """
    try:
        # Extract the token from the authorization header
        token = credentials.credentials

        # Check if we have user info in request headers (set by API Gateway)
        if request and request.headers.get("X-User-ID"):
            # Get user info from headers
            user_id = request.headers.get("X-User-ID")
            user_role = request.headers.get("X-User-Role", "authenticated")
            user_roles_str = request.headers.get("X-User-Roles", "")
            user_permissions_str = request.headers.get("X-User-Permissions", "")
            user_email = request.headers.get("X-User-Email", "")

            # Parse roles and permissions
            user_roles = user_roles_str.split(",") if user_roles_str else []
            user_permissions = user_permissions_str.split(",") if user_permissions_str else []

            logger.info(f"Using user info from headers: ID={user_id}, Role={user_role}")

            # Return user info from headers
            return {
                "token": token,
                "user_id": user_id,
                "email": user_email,
                "role": user_role,
                "roles": user_roles,
                "permissions": user_permissions
            }

        # Fallback to hardcoded values for testing
        logger.warning("No user info in headers, using fallback test values")
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
                "encounter:write"
            ]
        }
    except Exception as e:
        logger.error(f"Error extracting token payload: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid authentication credentials",
            headers={"WWW-Authenticate": "Bearer"}
        )
